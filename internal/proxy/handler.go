package proxy

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"strings"
	"time"

	"signal-proxy/internal/config"
	"signal-proxy/internal/ui"
)

// HandleConnection proxies traffic between client and Signal servers.
// It respects the context for cancellation during graceful shutdown.
func HandleConnection(ctx context.Context, clientConn net.Conn, cfg *config.Config) {
	defer clientConn.Close()

	// Track connection metrics
	MetricActiveConns.Inc()
	defer MetricActiveConns.Dec()

	startTime := time.Now()
	timeout := time.Duration(cfg.TimeoutSec) * time.Second

	// Set initial deadline
	clientConn.SetDeadline(time.Now().Add(timeout))

	// Extract SNI from the TLS connection
	tlsConn, ok := clientConn.(*tls.Conn)
	if !ok {
		MetricErrorsTotal.WithLabelValues("not_tls").Inc()
		return
	}

	// Perform handshake to get SNI
	if err := tlsConn.Handshake(); err != nil {
		MetricErrorsTotal.WithLabelValues("handshake_failed").Inc()
		return
	}

	sni := strings.ToLower(tlsConn.ConnectionState().ServerName)
	target, allowed := cfg.Hosts[sni]

	if !allowed || sni == "" {
		MetricErrorsTotal.WithLabelValues("unauthorized_sni").Inc()
		ui.LogStatus("error", "Unauthorized SNI: "+sni)
		return
	}

	// Connect to Signal servers with context awareness
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	upConn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		MetricErrorsTotal.WithLabelValues("dial_failed").Inc()
		ui.LogStatus("error", "Target unreachable: "+target)
		return
	}
	defer upConn.Close()
	upConn.SetDeadline(time.Now().Add(timeout))

	MetricRelayTotal.WithLabelValues(sni).Inc()

	// Channels for relay completion
	done := make(chan struct{}, 2)
	var upBytes, downBytes int64

	// Create a context-aware copy function
	copyWithContext := func(dst, src net.Conn, bytes *int64) {
		defer func() { done <- struct{}{} }()

		// Create a done channel that closes when context is cancelled
		go func() {
			<-ctx.Done()
			src.SetDeadline(time.Now()) // Force read to return
			dst.SetDeadline(time.Now())
		}()

		n, _ := io.Copy(dst, src)
		*bytes = n
	}

	// Relay traffic bidirectionally
	go copyWithContext(upConn, clientConn, &upBytes)
	go copyWithContext(clientConn, upConn, &downBytes)

	// Wait for either side to close or context cancellation
	select {
	case <-done:
	case <-ctx.Done():
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	MetricConnectionDuration.Observe(duration)
	MetricBytesTotal.WithLabelValues(sni, "upstream").Add(float64(upBytes))
	MetricBytesTotal.WithLabelValues(sni, "downstream").Add(float64(downBytes))

	ui.LogRelay(sni, clientConn.RemoteAddr().String(), upBytes, downBytes)
}
