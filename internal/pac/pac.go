package pac

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"signal-proxy/internal/auth"
	"signal-proxy/internal/ui"
)

// Config holds PAC-related configuration
type Config struct {
	Enabled        bool
	ProxyHost      string // e.g., "private.zignal.site"
	HTTPPort       string // e.g., "8080"
	SOCKS5Port     string // e.g., "1080"
	Token          string // Optional secret token for access control
	DefaultUser    string // Default username if no user param provided
	RateLimitRPM   int    // Rate limit for PAC endpoint
}

// Handler creates an HTTP handler for the PAC endpoint
type Handler struct {
	config    *Config
	userStore *auth.UserStore

	// Rate limiting
	rateMu      sync.Mutex
	rateTokens  map[string]int
	rateWindow  map[string]time.Time
}

// NewHandler creates a new PAC handler
func NewHandler(cfg *Config, userStore *auth.UserStore) *Handler {
	return &Handler{
		config:     cfg,
		userStore:  userStore,
		rateTokens: make(map[string]int),
		rateWindow: make(map[string]time.Time),
	}
}

// ServeHTTP handles PAC file requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)

	// Rate limiting
	if h.config.RateLimitRPM > 0 && !h.checkRateLimit(clientIP) {
		ui.LogStatus("warn", "PAC rate limited: "+clientIP)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	// Token-based access control (if configured)
	if h.config.Token != "" {
		token := r.URL.Query().Get("token")
		if token != h.config.Token {
			ui.LogStatus("warn", "PAC invalid token from: "+clientIP)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Get username from query parameter or use default
	username := r.URL.Query().Get("user")
	if username == "" {
		username = h.config.DefaultUser
	}

	if username == "" {
		// No user specified and no default configured
		h.sendErrorPAC(w, "No user specified. Use ?user=USERNAME")
		return
	}

	// Look up user's password (we need the plain password for PAC embedding)
	// Note: For security, we store a separate "pac_password" or use a token
	// Since we're using bcrypt hashes, we can't retrieve the original password
	// Instead, we'll embed a placeholder that the user must configure
	password := r.URL.Query().Get("pass")
	if password == "" {
		// Generate PAC with placeholder - user must provide password via query param
		// This is the safest approach since we can't reverse bcrypt hashes
		h.sendPACWithPlaceholder(w, username)
		return
	}

	// Validate credentials before embedding
	_, valid := h.userStore.ValidateCredentials(username, password)
	if !valid {
		ui.LogStatus("warn", "PAC invalid credentials for user: "+username)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate PAC with embedded credentials
	pac := h.generatePAC(username, password)
	h.sendPAC(w, pac)

	ui.LogStatus("info", "PAC served for user: "+username+" from "+clientIP)
}

// generatePAC creates the PAC file content with embedded credentials
func (h *Handler) generatePAC(username, password string) string {
	proxyURL := fmt.Sprintf("%s:%s@%s:%s",
		username, password, h.config.ProxyHost, h.config.HTTPPort)

	// SOCKS5 proxy URL (for SOCKS-capable clients)
	socks5URL := fmt.Sprintf("%s:%s@%s:%s",
		username, password, h.config.ProxyHost, h.config.SOCKS5Port)

	return fmt.Sprintf(`function FindProxyForURL(url, host) {
    // Don't proxy local addresses
    if (isPlainHostName(host) ||
        shExpMatch(host, "*.local") ||
        isInNet(host, "192.168.0.0", "255.255.0.0") ||
        isInNet(host, "10.0.0.0", "255.0.0.0") ||
        isInNet(host, "172.16.0.0", "255.240.0.0") ||
        host == "localhost" ||
        host == "127.0.0.1") {
        return "DIRECT";
    }
    
    // Route everything else through proxy
    // Primary: HTTP/HTTPS proxy, Fallback: SOCKS5
    return "PROXY %s; SOCKS5 %s; DIRECT";
}
`, proxyURL, socks5URL)
}

// sendPACWithPlaceholder sends a PAC file with placeholders for credentials
func (h *Handler) sendPACWithPlaceholder(w http.ResponseWriter, username string) {
	pac := fmt.Sprintf(`function FindProxyForURL(url, host) {
    // PAC file for user: %s
    // Note: This PAC requires authentication. Your browser/system will prompt for password.
    
    // Don't proxy local addresses
    if (isPlainHostName(host) ||
        shExpMatch(host, "*.local") ||
        isInNet(host, "192.168.0.0", "255.255.0.0") ||
        isInNet(host, "10.0.0.0", "255.0.0.0") ||
        isInNet(host, "172.16.0.0", "255.240.0.0") ||
        host == "localhost" ||
        host == "127.0.0.1") {
        return "DIRECT";
    }
    
    // Route everything else through proxy (credentials required separately)
    return "PROXY %s:%s; SOCKS5 %s:%s; DIRECT";
}
`, username, h.config.ProxyHost, h.config.HTTPPort, h.config.ProxyHost, h.config.SOCKS5Port)

	h.sendPAC(w, pac)
}

// sendErrorPAC sends a PAC file that returns DIRECT with an error comment
func (h *Handler) sendErrorPAC(w http.ResponseWriter, message string) {
	pac := fmt.Sprintf(`// Error: %s
function FindProxyForURL(url, host) {
    return "DIRECT";
}
`, message)

	h.sendPAC(w, pac)
}

// sendPAC sends the PAC content with proper headers
func (h *Handler) sendPAC(w http.ResponseWriter, content string) {
	// Set proper content type for PAC files
	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")

	// Allow short-term caching (5 min) - Android refetches PAC on every connection,
	// causing 50-200ms latency per connection setup. Caching fixes this.
	w.Header().Set("Cache-Control", "public, max-age=300")

	// CORS headers for browser compatibility
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

// checkRateLimit implements simple rate limiting for the PAC endpoint
func (h *Handler) checkRateLimit(clientIP string) bool {
	h.rateMu.Lock()
	defer h.rateMu.Unlock()

	now := time.Now()
	windowStart, exists := h.rateWindow[clientIP]

	// Reset window if expired (1 minute window)
	if !exists || now.Sub(windowStart) > time.Minute {
		h.rateWindow[clientIP] = now
		h.rateTokens[clientIP] = 1
		return true
	}

	// Check if under limit
	if h.rateTokens[clientIP] < h.config.RateLimitRPM {
		h.rateTokens[clientIP]++
		return true
	}

	return false
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxied requests)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the chain
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

// GenerateToken creates a random access token (utility function)
func GenerateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Return first 7 characters as requested
	return hex.EncodeToString(bytes)[:7], nil
}
