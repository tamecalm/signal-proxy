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
)

func main() {
	ui.PrintBanner()

	// Load and validate configuration
	cfg := config.Load()
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
		metrics.Shutdown(context.Background())
	}()

	// Start the proxy server
	srv := proxy.NewServer(cfg)
	if err := srv.Start(ctx); err != nil {
		ui.LogStatus("error", "Server failed: "+err.Error())
		log.Fatal(err)
	}
}
