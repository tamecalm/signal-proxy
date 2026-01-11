package proxy

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"signal-proxy/internal/ui"
)

var (
	// MetricRelayTotal counts total relayed connections by SNI
	MetricRelayTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "signalproxy_relay_total",
		Help: "Total relayed connections by SNI",
	}, []string{"sni"})

	// MetricActiveConns tracks current active connections
	MetricActiveConns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "signalproxy_active_conns",
		Help: "Current active connections",
	})

	// MetricBytesTotal counts bytes transferred by direction
	MetricBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "signalproxy_bytes_total",
		Help: "Total bytes transferred",
	}, []string{"sni", "direction"})

	// MetricErrorsTotal counts errors by type
	MetricErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "signalproxy_errors_total",
		Help: "Total errors by type",
	}, []string{"type"})

	// MetricConnectionDuration tracks connection duration
	MetricConnectionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "signalproxy_connection_duration_seconds",
		Help:    "Connection duration in seconds",
		Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600},
	})

	// MetricConnectionsRejected counts rejected connections due to capacity
	MetricConnectionsRejected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "signalproxy_connections_rejected_total",
		Help: "Total connections rejected due to capacity",
	})
)

// activeConnsValue is used internally to get the current gauge value for logging
var activeConnsMu sync.Mutex
var activeConnsCount int

func init() {
	// Wrap the gauge to track count for logging purposes
	origInc := MetricActiveConns.Inc
	origDec := MetricActiveConns.Dec
	MetricActiveConns = &gaugeWrapper{
		Gauge:  MetricActiveConns.(prometheus.Gauge),
		inc:    origInc,
		dec:    origDec,
		count:  &activeConnsCount,
		mu:     &activeConnsMu,
	}
}

type gaugeWrapper struct {
	prometheus.Gauge
	inc   func()
	dec   func()
	count *int
	mu    *sync.Mutex
}

func (g *gaugeWrapper) Inc() {
	g.mu.Lock()
	*g.count++
	g.mu.Unlock()
	g.Gauge.Inc()
}

func (g *gaugeWrapper) Dec() {
	g.mu.Lock()
	*g.count--
	g.mu.Unlock()
	g.Gauge.Dec()
}

func (g *gaugeWrapper) Get() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return *g.count
}

// GetActiveConns returns the current active connection count
func GetActiveConns() int {
	activeConnsMu.Lock()
	defer activeConnsMu.Unlock()
	return activeConnsCount
}

// MetricsServer wraps the HTTP server for prometheus metrics
type MetricsServer struct {
	server *http.Server
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(addr string) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &MetricsServer{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Start begins serving metrics (non-blocking)
func (m *MetricsServer) Start() {
	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ui.LogStatus("error", "Metrics server error: "+err.Error())
		}
	}()
}

// Shutdown gracefully stops the metrics server
func (m *MetricsServer) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return m.server.Shutdown(shutdownCtx)
}
