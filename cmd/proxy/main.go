package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"signal-proxy/internal/auth"
	"signal-proxy/internal/config"
	"signal-proxy/internal/httpproxy"
	"signal-proxy/internal/proxy"
	"signal-proxy/internal/socks5"
	"signal-proxy/internal/ui"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	// We ignore the error because in production/docker we might relying on system env vars
	_ = godotenv.Load()

	// Display banner with version and tagline
	ui.PrintBanner()

	// Load and validate configuration
	cfg := config.Load()

	// Display environment info
	if cfg.Env.IsDevelopment() {
		ui.LogStatus("info", "Environment: "+ui.Warn("DEVELOPMENT"))
		ui.LogStatus("info", "Domain: "+cfg.Env.Domain)
	} else {
		ui.LogStatus("info", "Environment: "+ui.Success("PRODUCTION"))
		ui.LogStatus("info", "Domain: "+cfg.Env.Domain)
	}

	// Create shutdown context
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start metrics server with graceful shutdown
	metrics := proxy.NewMetricsServer(cfg.MetricsListen)
	metrics.Start()

	// Shutdown metrics on exit
	go func() {
		<-ctx.Done()
		ui.LogGracefulShutdown()
		metrics.Shutdown(context.Background())
	}()

	// Switch behavior based on proxy mode
	switch cfg.Env.ProxyMode {
	case "https", "http", "general":
		// HTTP/HTTPS/SOCKS5 proxy mode
		runHTTPSProxyMode(ctx, cfg)
	default:
		// Signal proxy mode (default)
		runSignalProxyMode(ctx, cfg)
	}
}

// runSignalProxyMode starts the Signal TLS proxy (original behavior)
func runSignalProxyMode(ctx context.Context, cfg *config.Config) {
	ui.LogStatus("info", "Proxy Mode: "+ui.Success("SIGNAL"))

	if err := cfg.Validate(); err != nil {
		ui.LogStatus("error", err.Error())
		os.Exit(1)
	}

	// Start the proxy server
	srv := proxy.NewServer(cfg)

	// Listen for SIGHUP to reload certificates
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-sighup:
				ui.LogStatus("info", "SIGHUP received, reloading certificates...")
				if err := srv.Reload(); err != nil {
					ui.LogStatus("error", "Reload failed: "+err.Error())
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := srv.Start(ctx); err != nil {
		ui.LogStatus("error", "Server failed: "+err.Error())
		log.Fatal(err)
	}
}

// runHTTPSProxyMode starts the HTTP/HTTPS/SOCKS5 proxy
func runHTTPSProxyMode(ctx context.Context, cfg *config.Config) {
	ui.LogStatus("info", "Proxy Mode: "+ui.Success("HTTPS/SOCKS5"))

	// Load user store
	userStore, err := auth.NewUserStore(cfg.Env.UsersFile)
	if err != nil {
		ui.LogStatus("error", "Failed to load users: "+err.Error())
		os.Exit(1)
	}
	ui.LogStatus("info", "Loaded "+itoa(userStore.GetUserCount())+" users from "+cfg.Env.UsersFile)

	// Create HTTP proxy server
	httpSrv := httpproxy.NewServer(cfg, userStore)

	// Create SOCKS5 proxy server
	socks5Srv := socks5.NewServer(cfg, userStore)

	// Start SOCKS5 in background
	go func() {
		if err := socks5Srv.Start(ctx); err != nil {
			ui.LogStatus("error", "SOCKS5 server failed: "+err.Error())
		}
	}()

	// Start HTTP proxy (blocking)
	if err := httpSrv.Start(ctx); err != nil {
		ui.LogStatus("error", "HTTP proxy failed: "+err.Error())
		log.Fatal(err)
	}
}

// itoa is a simple int to string helper
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
}
