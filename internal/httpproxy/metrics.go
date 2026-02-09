package httpproxy

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricRequests counts total HTTP proxy requests by user and method
	MetricRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "httpproxy_requests_total",
		Help: "Total HTTP proxy requests by user and method",
	}, []string{"user", "method"})

	// MetricBytes counts total bytes transferred by user and direction
	MetricBytes = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "httpproxy_bytes_total",
		Help: "Total bytes transferred by user and direction",
	}, []string{"user", "direction"})

	// MetricActiveConns tracks current active proxy connections
	MetricActiveConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "httpproxy_active_connections",
		Help: "Current active HTTP proxy connections",
	})

	// MetricAuthFailures counts authentication failures by type
	MetricAuthFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "httpproxy_auth_failures_total",
		Help: "Total authentication failures by type",
	}, []string{"type"})

	// MetricRateLimited counts rate limited requests by user
	MetricRateLimited = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "httpproxy_rate_limited_total",
		Help: "Total rate limited requests by user",
	}, []string{"user"})

	// MetricErrors counts proxy errors by type
	MetricErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "httpproxy_errors_total",
		Help: "Total proxy errors by type",
	}, []string{"type"})

	// MetricDuration tracks request duration
	MetricDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "httpproxy_request_duration_seconds",
		Help:    "HTTP proxy request duration in seconds",
		Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300},
	})
)
