package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds all proxy configuration values.
type Config struct {
	Listen        string            `json:"listen"`
	CertFile      string            `json:"cert_file"`
	KeyFile       string            `json:"key_file"`
	TimeoutSec    int               `json:"timeout_sec"`
	MaxConns      int               `json:"max_conns"`
	MetricsListen string            `json:"metrics_listen"`
	Hosts         map[string]string `json:"hosts"`
	
	// Environment configuration (loaded from env vars)
	Env *EnvConfig `json:"-"`
}

// Load reads configuration from config.json with sensible defaults.
func Load() *Config {
	cfg := &Config{
		Listen:        ":8443",
		TimeoutSec:    300,
		MaxConns:      1000,
		MetricsListen: ":9090",
		CertFile:      "server.crt",
		KeyFile:       "server.key",
		Hosts:         make(map[string]string),
		Env:           LoadEnv(), // Load environment config
	}

	if file, err := os.Open("config.json"); err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(cfg)
	}

	// Normalize SNI keys to lowercase
	cleaned := make(map[string]string)
	for k, v := range cfg.Hosts {
		cleaned[strings.ToLower(strings.TrimSpace(k))] = v
	}
	cfg.Hosts = cleaned

	return cfg
}

// Validate checks the configuration for errors and returns helpful messages.
func (c *Config) Validate() error {
	var errs []string

	// Check required fields
	if c.Listen == "" {
		errs = append(errs, "listen address is required")
	}

	// Check certificate files exist
	if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
		errs = append(errs, fmt.Sprintf("certificate file not found: %s", c.CertFile))
	}
	if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
		errs = append(errs, fmt.Sprintf("key file not found: %s", c.KeyFile))
	}

	// Validate numeric values
	if c.TimeoutSec <= 0 {
		errs = append(errs, "timeout_sec must be positive")
	}
	if c.MaxConns <= 0 {
		errs = append(errs, "max_conns must be positive")
	}

	// Check hosts
	if len(c.Hosts) == 0 {
		errs = append(errs, "at least one host mapping is required")
	}

	if len(errs) > 0 {
		return errors.New("config validation failed:\n  - " + strings.Join(errs, "\n  - "))
	}

	return nil
}
