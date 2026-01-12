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
	Domain   string 
	BaseURL  string 
	
	// Feature flags
	Debug    bool   
	
	// Environment-specific values
	LogLevel string
	
	// Ngrok configuration (development only)
	NgrokEnabled bool
	NgrokDomain  string
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
		cfg.Debug = getEnvOrDefault("DEBUG", "true") == "true"
		if cfg.LogLevel == "info" {
			cfg.LogLevel = "debug" // Dev default
		}
	}
	
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
