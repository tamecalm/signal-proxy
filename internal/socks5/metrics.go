package socks5

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricConnections counts total SOCKS5 connections by user
	MetricConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "socks5_connections_total",
		Help: "Total SOCKS5 connections by user",
	}, []string{"user"})

	// MetricBytes counts total bytes transferred by user and direction
	MetricBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "socks5_bytes_total",
		Help: "Total bytes transferred by user and direction",
	}, []string{"user", "direction"})

	// MetricActiveConns tracks current active SOCKS5 connections
	MetricActiveConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "socks5_active_connections",
		Help: "Current active SOCKS5 connections",
	})

	// MetricAuthFailures counts authentication failures by type
	MetricAuthFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "socks5_auth_failures_total",
		Help: "Total SOCKS5 authentication failures by type",
	}, []string{"type"})

	// MetricRateLimited counts rate limited requests by user
	MetricRateLimited = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "socks5_rate_limited_total",
		Help: "Total SOCKS5 rate limited requests by user",
	}, []string{"user"})

	// MetricErrors counts SOCKS5 errors by type
	MetricErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "socks5_errors_total",
		Help: "Total SOCKS5 errors by type",
	}, []string{"type"})

	// MetricDuration tracks connection duration
	MetricDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "socks5_connection_duration_seconds",
		Help:    "SOCKS5 connection duration in seconds",
		Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
	})
)
