package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"signal-proxy/internal/config"
	"signal-proxy/internal/proxy"
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
	
	if err := cfg.Validate(); err != nil {
		ui.LogStatus("error", err.Error())
		os.Exit(1)
	}

	// Create shutdown context
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start metrics server with graceful shutdown
	metrics := proxy.NewMetricsServer(cfg.MetricsListen)
	metrics.Start()
	ui.LogStatus("info", "Metrics: http://localhost"+cfg.MetricsListen+"/metrics")

	// Shutdown metrics on exit
	go func() {
		<-ctx.Done()
		ui.LogGracefulShutdown()
		metrics.Shutdown(context.Background())
	}()

	// Start the proxy server
	srv := proxy.NewServer(cfg)
	if err := srv.Start(ctx); err != nil {
		ui.LogStatus("error", "Server failed: "+err.Error())
		log.Fatal(err)
	}
}
