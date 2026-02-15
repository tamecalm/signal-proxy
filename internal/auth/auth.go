package auth

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// User represents a proxy user with credentials and settings
type User struct {
	Username     string `json:"username"`
	Role         string `json:"role"`
	PasswordHash string `json:"password_hash"`
	RateLimitRPM int    `json:"rate_limit_rpm"` // Requests per minute, 0 = unlimited
	Enabled      bool   `json:"enabled"`
}

// UsersConfig holds all user configuration
type UsersConfig struct {
	Users         []User   `json:"users"`
	IPWhitelist   []string `json:"ip_whitelist"`    // CIDR notation, empty = allow all
	SuperAdminIPs []string `json:"super_admin_ips"` // CIDR notation for super_admin bypass
}

// UserStore manages user authentication and authorization
type UserStore struct {
	mu             sync.RWMutex
	users          map[string]*User
	ipWhitelist    []*net.IPNet
	superAdminIPs  []*net.IPNet
	superAdminUser *User // cached reference to the super_admin user
	rateLimiter    *RateLimiter
}

// NewUserStore creates a new user store from a config file
func NewUserStore(configPath string) (*UserStore, error) {
	store := &UserStore{
		users:         make(map[string]*User),
		ipWhitelist:   make([]*net.IPNet, 0),
		superAdminIPs: make([]*net.IPNet, 0),
		rateLimiter:   NewRateLimiter(),
	}

	if err := store.LoadFromFile(configPath); err != nil {
		return nil, err
	}

	return store, nil
}

// LoadFromFile loads user configuration from a JSON file
func (s *UserStore) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read users file: %w", err)
	}

	var cfg UsersConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse users file: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Load users
	s.users = make(map[string]*User)
	for i := range cfg.Users {
		user := &cfg.Users[i]
		if user.Enabled {
			s.users[strings.ToLower(user.Username)] = user
			// Initialize rate limiter for user
			if user.RateLimitRPM > 0 {
				s.rateLimiter.SetLimit(user.Username, user.RateLimitRPM)
			}
		}
	}

	// Identify super_admin user
	s.superAdminUser = nil
	for _, user := range s.users {
		if strings.ToLower(user.Role) == "super_admin" {
			s.superAdminUser = user
			break
		}
	}

	// Parse IP whitelist
	s.ipWhitelist = make([]*net.IPNet, 0, len(cfg.IPWhitelist))
	for _, cidr := range cfg.IPWhitelist {
		ipNet, err := parseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid IP whitelist entry '%s': %w", cidr, err)
		}
		s.ipWhitelist = append(s.ipWhitelist, ipNet)
	}

	// Parse super_admin IPs
	s.superAdminIPs = make([]*net.IPNet, 0, len(cfg.SuperAdminIPs))
	for _, cidr := range cfg.SuperAdminIPs {
		ipNet, err := parseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid super_admin_ips entry '%s': %w", cidr, err)
		}
		s.superAdminIPs = append(s.superAdminIPs, ipNet)
	}

	return nil
}

// ValidateCredentials checks if username and password are valid
func (s *UserStore) ValidateCredentials(username, password string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[strings.ToLower(username)]
	if !exists {
		return nil, false
	}

	// Compare password with bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, false
	}

	return user, true
}

// CheckIPAllowed verifies if an IP address is in the whitelist
// Returns true if whitelist is empty (allow all) or IP is whitelisted
func (s *UserStore) CheckIPAllowed(ipStr string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Empty whitelist means allow all
	if len(s.ipWhitelist) == 0 {
		return true
	}

	// Parse the IP address (handle host:port format)
	host := ipStr
	if strings.Contains(ipStr, ":") {
		var err error
		host, _, err = net.SplitHostPort(ipStr)
		if err != nil {
			// Might be IPv6 without port
			host = ipStr
		}
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Check against whitelist
	for _, ipNet := range s.ipWhitelist {
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// CheckRateLimit checks if request is within rate limit for user
// Returns true if allowed, false if rate limited
func (s *UserStore) CheckRateLimit(username string) bool {
	s.mu.RLock()
	user, exists := s.users[strings.ToLower(username)]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	// No rate limit configured
	if user.RateLimitRPM <= 0 {
		return true
	}

	return s.rateLimiter.Allow(username)
}

// IsSuperAdminIP checks if the given IP matches any super_admin CIDR.
// Returns the super_admin User and true if matched, nil and false otherwise.
func (s *UserStore) IsSuperAdminIP(ipStr string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.superAdminUser == nil || len(s.superAdminIPs) == 0 {
		return nil, false
	}

	ip := parseIP(ipStr)
	if ip == nil {
		return nil, false
	}

	for _, ipNet := range s.superAdminIPs {
		if ipNet.Contains(ip) {
			return s.superAdminUser, true
		}
	}

	return nil, false
}

// GetUserCount returns the number of enabled users
func (s *UserStore) GetUserCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// HashPassword generates a bcrypt hash for a password
// This is a utility function for generating hashes for users.json
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// parseCIDR parses a CIDR string, handling bare IPs without mask notation.
func parseCIDR(cidr string) (*net.IPNet, error) {
	if !strings.Contains(cidr, "/") {
		if strings.Contains(cidr, ":") {
			cidr = cidr + "/128" // IPv6
		} else {
			cidr = cidr + "/32" // IPv4
		}
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	return ipNet, err
}

// parseIP extracts and parses an IP from a string that may include a port.
func parseIP(ipStr string) net.IP {
	host := ipStr
	if strings.Contains(ipStr, ":") {
		var err error
		host, _, err = net.SplitHostPort(ipStr)
		if err != nil {
			host = ipStr // Might be IPv6 without port
		}
	}
	return net.ParseIP(host)
}
