package config

import (
	"os"
	"strings"
)

// Environment represents the application environment
type Environment string

const (
	// Development environment - localhost, debug enabled
	Development Environment = "development"
	// Production environment - real domain, production settings
	Production Environment = "production"
)

// EnvConfig holds environment-specific configuration
type EnvConfig struct {
	// Environment name (development, production)
	Env Environment

	// Domain settings
	Domain        string
	APIDomain     string
	BaseURL       string
	AllowedOrigin string

	// Feature flags
	Debug bool

	// Environment-specific values
	LogLevel string

	// Ngrok configuration (development only)
	NgrokEnabled bool
	NgrokDomain  string

	// Proxy mode configuration
	ProxyMode       string // "signal" (default) or "https"
	HTTPProxyPort   string // HTTP proxy port (default :8080)
	HTTPProxyTLS    bool   // Enable TLS for HTTP proxy
	HTTPProxyTLSPort string // HTTPS proxy port (default :8443)
	SOCKS5Port      string // SOCKS5 proxy port (default :1080)
	UsersFile       string // Path to users.json

	// PAC (Proxy Auto-Config) configuration
	PACEnabled      bool   // Enable PAC endpoint (/proxy.pac)
	PACToken        string // Optional secret token for PAC access control
	PACDefaultUser  string // Default username if no user param provided
	PACRateLimitRPM int    // Rate limit for PAC endpoint (requests per minute)
}

// LoadEnv loads environment configuration from environment variables
func LoadEnv() *EnvConfig {
	env := getEnvOrDefault("APP_ENV", "development")

	cfg := &EnvConfig{
		Env:      Environment(strings.ToLower(env)),
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),
	}

	// Set environment-specific defaults
	switch cfg.Env {
	case Production:
		cfg.Domain = getEnvOrDefault("DOMAIN", "proxy.yourdomain.com")
		cfg.APIDomain = getEnvOrDefault("API_DOMAIN", "api."+cfg.Domain)
		cfg.BaseURL = getEnvOrDefault("BASE_URL", "https://"+cfg.Domain)
		cfg.AllowedOrigin = getEnvOrDefault("ALLOWED_ORIGIN", "*")
		cfg.Debug = getEnvOrDefault("DEBUG", "false") == "true"
		cfg.NgrokEnabled = false // Never use ngrok in production
		if cfg.LogLevel == "info" {
			cfg.LogLevel = "info"
		}
	default: // Development
		cfg.Env = Development // Normalize unknown envs to development

		// Ngrok configuration (development only)
		cfg.NgrokEnabled = getEnvOrDefault("NGROK_ENABLED", "true") == "true"
		cfg.NgrokDomain = getEnvOrDefault("NGROK_DOMAIN", "")

		// Use ngrok domain if enabled and provided, otherwise fall back to localhost
		if cfg.NgrokEnabled && cfg.NgrokDomain != "" {
			cfg.Domain = getEnvOrDefault("DOMAIN", cfg.NgrokDomain)
		} else {
			cfg.Domain = getEnvOrDefault("DOMAIN", "localhost:8443")
		}

		cfg.BaseURL = getEnvOrDefault("BASE_URL", "https://"+cfg.Domain)
		cfg.APIDomain = cfg.Domain
		cfg.AllowedOrigin = "*"
		cfg.Debug = getEnvOrDefault("DEBUG", "true") == "true"
		if cfg.LogLevel == "info" {
			cfg.LogLevel = "debug" // Dev default
		}
	}

	// Load proxy mode configuration (applies to both dev and prod)
	cfg.ProxyMode = strings.ToLower(getEnvOrDefault("PROXY_MODE", "signal"))
	cfg.HTTPProxyPort = getEnvOrDefault("HTTP_PROXY_PORT", ":8080")
	cfg.HTTPProxyTLS = getEnvOrDefault("HTTP_PROXY_TLS", "true") == "true"
	cfg.HTTPProxyTLSPort = getEnvOrDefault("HTTP_PROXY_TLS_PORT", ":8443")
	cfg.SOCKS5Port = getEnvOrDefault("SOCKS5_PORT", ":1080")
	cfg.UsersFile = getEnvOrDefault("USERS_FILE", "users.json")

	// Load PAC configuration
	cfg.PACEnabled = getEnvOrDefault("PAC_ENABLED", "true") == "true"
	cfg.PACToken = getEnvOrDefault("PAC_TOKEN", "") // Empty = no token required
	cfg.PACDefaultUser = getEnvOrDefault("PAC_DEFAULT_USER", "")
	cfg.PACRateLimitRPM = parseIntOrDefault(getEnvOrDefault("PAC_RATE_LIMIT_RPM", "60"), 60)

	return cfg
}

// IsDevelopment returns true if running in development mode
func (e *EnvConfig) IsDevelopment() bool {
	return e.Env == Development
}

// IsProduction returns true if running in production mode
func (e *EnvConfig) IsProduction() bool {
	return e.Env == Production
}

// String returns the environment name
func (e Environment) String() string {
	return string(e)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseIntOrDefault parses a string as int, returning default on error
func parseIntOrDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	// Simple integer parsing without importing strconv
	result := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			return defaultValue
		}
	}
	return result
}
