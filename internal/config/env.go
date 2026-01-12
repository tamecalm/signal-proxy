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

// TunnelProvider represents the tunnel provider type
type TunnelProvider string

const (
	TunnelNone       TunnelProvider = "none"
	TunnelNgrok      TunnelProvider = "ngrok"
	TunnelCloudflare TunnelProvider = "cloudflare"
	TunnelAuto       TunnelProvider = "auto"
)

// EnvConfig holds environment-specific configuration
type EnvConfig struct {
	// Environment name (development, production)
	Env Environment

	// Domain settings
	Domain  string
	BaseURL string

	// Feature flags
	Debug bool

	// Environment-specific values
	LogLevel string

	// Tunnel provider selection: "ngrok", "cloudflare", "none", or "auto"
	TunnelProvider TunnelProvider

	// Ngrok configuration (development only)
	NgrokEnabled bool
	NgrokDomain  string

	// Cloudflare Tunnel configuration (development only)
	CloudflareEnabled bool
	CloudflareDomain  string
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
		cfg.BaseURL = getEnvOrDefault("BASE_URL", "https://"+cfg.Domain)
		cfg.Debug = getEnvOrDefault("DEBUG", "false") == "true"
		cfg.TunnelProvider = TunnelNone // Never use tunnels in production
		cfg.NgrokEnabled = false
		cfg.CloudflareEnabled = false
		if cfg.LogLevel == "info" {
			cfg.LogLevel = "info"
		}
	default: // Development
		cfg.Env = Development // Normalize unknown envs to development

		// Load tunnel configurations
		cfg.NgrokEnabled = getEnvOrDefault("NGROK_ENABLED", "false") == "true"
		cfg.NgrokDomain = getEnvOrDefault("NGROK_DOMAIN", "")
		cfg.CloudflareEnabled = getEnvOrDefault("CLOUDFLARE_ENABLED", "false") == "true"
		cfg.CloudflareDomain = getEnvOrDefault("CLOUDFLARE_DOMAIN", "")

		// Determine tunnel provider
		providerStr := strings.ToLower(getEnvOrDefault("TUNNEL_PROVIDER", "auto"))
		switch providerStr {
		case "ngrok":
			cfg.TunnelProvider = TunnelNgrok
		case "cloudflare":
			cfg.TunnelProvider = TunnelCloudflare
		case "none":
			cfg.TunnelProvider = TunnelNone
		default:
			cfg.TunnelProvider = TunnelAuto
		}

		// Resolve domain based on tunnel provider
		cfg.Domain = cfg.resolveDomain()
		cfg.BaseURL = getEnvOrDefault("BASE_URL", "https://"+cfg.Domain)
		cfg.Debug = getEnvOrDefault("DEBUG", "true") == "true"
		if cfg.LogLevel == "info" {
			cfg.LogLevel = "debug" // Dev default
		}
	}

	return cfg
}

// resolveDomain determines the domain based on tunnel provider configuration
func (e *EnvConfig) resolveDomain() string {
	// Check for explicit DOMAIN override first
	if domain := os.Getenv("DOMAIN"); domain != "" {
		return domain
	}

	switch e.TunnelProvider {
	case TunnelCloudflare:
		if e.CloudflareDomain != "" {
			return e.CloudflareDomain
		}
	case TunnelNgrok:
		if e.NgrokDomain != "" {
			return e.NgrokDomain
		}
	case TunnelAuto:
		// Auto-detect: prefer Cloudflare if enabled and configured
		if e.CloudflareEnabled && e.CloudflareDomain != "" {
			return e.CloudflareDomain
		}
		// Fall back to ngrok if enabled and configured
		if e.NgrokEnabled && e.NgrokDomain != "" {
			return e.NgrokDomain
		}
	}

	// Default to localhost
	return "localhost:8443"
}

// GetActiveTunnelProvider returns the effective tunnel provider being used
func (e *EnvConfig) GetActiveTunnelProvider() TunnelProvider {
	if e.TunnelProvider == TunnelAuto {
		if e.CloudflareEnabled && e.CloudflareDomain != "" {
			return TunnelCloudflare
		}
		if e.NgrokEnabled && e.NgrokDomain != "" {
			return TunnelNgrok
		}
		return TunnelNone
	}
	return e.TunnelProvider
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

// String returns the tunnel provider name
func (t TunnelProvider) String() string {
	return string(t)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
