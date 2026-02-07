package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"signal-proxy/internal/config"
	"signal-proxy/internal/ui"
)

// Server handles TLS connections and proxies them to Signal servers.
type Server struct {
	Config   *config.Config
	ln       net.Listener
	connSem  chan struct{}  // Semaphore for connection limiting
	wg       sync.WaitGroup // Tracks active connections for graceful shutdown
	shutdown chan struct{}  // Signals shutdown to accept loop

	// Certificate management for hot-reloading
	mu   sync.RWMutex
	cert *tls.Certificate
}

// NewServer creates a new proxy server with the given configuration.
func NewServer(cfg *config.Config) *Server {
	return &Server{
		Config:   cfg,
		connSem:  make(chan struct{}, cfg.MaxConns),
		shutdown: make(chan struct{}),
	}
}

// Reload reloads the TLS certificate from disk.
func (s *Server) Reload() error {
	cert, err := tls.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.cert = &cert
	s.mu.Unlock()

	ui.LogStatus("success", "Certificates reloaded from disk")
	return nil
}

// getCertificate returns the current certificate for TLS handshakes.
func (s *Server) getCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cert, nil
}

// Start begins accepting connections. It blocks until shutdown or error.
// The context is used for graceful shutdown - cancel it to initiate shutdown.
func (s *Server) Start(ctx context.Context) error {
	// 1. Initial certificate load
	if err := s.Reload(); err != nil {
		return err
	}

	// TLS config for terminating the OUTER TLS connection from Signal app
	tlsConfig := &tls.Config{
		GetCertificate: s.getCertificate,
		MinVersion:     tls.VersionTLS12,
		NextProtos:     []string{"http/1.1"},
	}

	// 2. Start TLS Listener (we terminate the OUTER TLS here)
	var err error
	s.ln, err = tls.Listen("tcp", s.Config.Listen, tlsConfig)
	if err != nil {
		return err
	}

	metricsAddr := s.Config.MetricsListen
	if strings.HasPrefix(metricsAddr, ":") {
		metricsAddr = "localhost" + metricsAddr
	}
	ui.LogStatus("info", "Metrics: http://"+metricsAddr+"/metrics")
	ui.LogStatus("info", "Stats API: https://" + s.Config.Env.APIDomain + "/api/stats")

	// 3. Monitor for shutdown signal
	go s.watchShutdown(ctx)

	// 4. Accept Loop
	for {
		// Check if we're shutting down
		select {
		case <-s.shutdown:
			return s.drainConnections()
		default:
		}

		conn, err := s.ln.Accept()
		if err != nil {
			// Check if listener was closed
			select {
			case <-s.shutdown:
				return s.drainConnections()
			default:
				// Ignore temporary errors
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return err
			}
		}

		// Try to acquire connection slot (non-blocking)
		select {
		case s.connSem <- struct{}{}:
			// Got a slot, handle the connection
			s.wg.Add(1)
			go func(c net.Conn) {
				defer s.wg.Done()
				defer func() { <-s.connSem }() // Release slot when done
				HandleConnection(ctx, c, s.Config)
			}(conn)
		default:
			// At capacity, reject connection
			MetricConnectionsRejected.Inc()
			ui.LogStatus("warn", "Connection rejected: at max capacity ("+itoa(s.Config.MaxConns)+")")
			conn.Close()
		}
	}
}

// watchShutdown monitors the context for cancellation and initiates shutdown.
func (s *Server) watchShutdown(ctx context.Context) {
	<-ctx.Done()
	ui.LogStatus("warn", "Shutdown signal received...")
	close(s.shutdown)
	s.ln.Close()
}

// drainConnections waits for active connections to finish (with timeout).
func (s *Server) drainConnections() error {
	activeConns := GetActiveConns()
	if activeConns > 0 {
		ui.LogStatus("info", "Draining "+itoa(activeConns)+" active connections (30s timeout)...")
	}

	// Wait for connections with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		ui.LogStatus("success", "All connections drained. Goodbye.")
	case <-time.After(30 * time.Second):
		ui.LogStatus("warn", "Drain timeout reached. Forcing shutdown.")
	}

	return nil
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

// PeekSNI reads the TLS ClientHello from a raw connection to extract SNI.
// This is used AFTER outer TLS termination to read the INNER TLS ClientHello.
func PeekSNI(conn net.Conn) (string, []byte, error) {
	buf := make([]byte, 16384)
	n, err := conn.Read(buf)
	if err != nil {
		return "", nil, err
	}
	data := buf[:n]
	sni := extractSNI(data)
	return sni, data, nil
}

// extractSNI parses a TLS ClientHello message and extracts the SNI hostname
func extractSNI(data []byte) string {
	// Minimum TLS record header: 5 bytes
	if len(data) < 5 {
		return ""
	}

	// Check if this is a TLS Handshake record (0x16)
	if data[0] != 0x16 {
		return ""
	}

	// Skip record header (5 bytes)
	pos := 5

	// Handshake header: Type (1) + Length (3)
	if len(data) < pos+4 {
		return ""
	}

	// Check if this is a ClientHello (0x01)
	if data[pos] != 0x01 {
		return ""
	}

	// Skip handshake header (4 bytes)
	pos += 4

	// Skip version (2) + random (32)
	if len(data) < pos+34 {
		return ""
	}
	pos += 34

	// Skip session ID
	if len(data) < pos+1 {
		return ""
	}
	sessionIDLen := int(data[pos])
	pos += 1 + sessionIDLen

	// Skip cipher suites
	if len(data) < pos+2 {
		return ""
	}
	cipherSuitesLen := int(data[pos])<<8 | int(data[pos+1])
	pos += 2 + cipherSuitesLen

	// Skip compression methods
	if len(data) < pos+1 {
		return ""
	}
	compressionLen := int(data[pos])
	pos += 1 + compressionLen

	// Extensions
	if len(data) < pos+2 {
		return ""
	}
	extensionsLen := int(data[pos])<<8 | int(data[pos+1])
	pos += 2

	endPos := pos + extensionsLen
	if endPos > len(data) {
		endPos = len(data)
	}

	// Parse extensions to find SNI (type 0x0000)
	for pos+4 <= endPos {
		extType := int(data[pos])<<8 | int(data[pos+1])
		extLen := int(data[pos+2])<<8 | int(data[pos+3])
		pos += 4

		if extType == 0x0000 { // SNI extension
			if pos+5 > endPos {
				return ""
			}
			// Skip list length (2), check name type (should be 0)
			if data[pos+2] != 0x00 {
				return ""
			}
			nameLen := int(data[pos+3])<<8 | int(data[pos+4])
			if pos+5+nameLen > endPos {
				return ""
			}
			return string(data[pos+5 : pos+5+nameLen])
		}
		pos += extLen
	}

	return ""
}

// HandleConnection handles the TLS-in-TLS tunnel for Signal.
// The outer TLS is already terminated by the server listener.
// We read the inner TLS ClientHello to get the real destination SNI.
func HandleConnection(ctx context.Context, clientConn net.Conn, cfg *config.Config) {
	defer clientConn.Close()

	// Track metrics
	MetricActiveConns.Inc()
	defer MetricActiveConns.Dec()

	startTime := time.Now()
	timeout := time.Duration(cfg.TimeoutSec) * time.Second

	// Set deadline for reading inner ClientHello
	clientConn.SetDeadline(time.Now().Add(10 * time.Second))

	// Read the INNER TLS ClientHello (this is sent inside the outer TLS tunnel)
	sni, initialData, err := PeekSNI(clientConn)
	if err != nil {
		MetricErrorsTotal.WithLabelValues("peek_failed").Inc()
		Stats.RecordError()
		ui.LogStatus("error", "Failed to peek SNI: "+err.Error())
		return
	}

	// Lookup destination
	target, allowed := cfg.Hosts[strings.ToLower(sni)]
	if !allowed || sni == "" {
		// Differentiate between Signal traffic (Inner TLS) and Stats API traffic (HTTP)
		// Signal traffic always starts with a TLS handshake (0x16)
		if len(initialData) > 0 && initialData[0] != 0x16 {
			// This looks like an HTTP request (browser/landing page)
			// Handle the Stats API directly on this connection
			handleInternalAPI(clientConn, initialData)
			return
		}

		MetricErrorsTotal.WithLabelValues("unauthorized_sni").Inc()
		Stats.RecordError()
		ui.LogStatus("error", "Unauthorized SNI: "+sni)
		return
	}

	// Connect to Signal server
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	upConn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		MetricErrorsTotal.WithLabelValues("dial_failed").Inc()
		Stats.RecordError()
		ui.LogStatus("error", "Target unreachable: "+target+" - "+err.Error())
		return
	}
	defer upConn.Close()

	// Forward the ClientHello we already read
	if len(initialData) > 0 {
		if _, err := upConn.Write(initialData); err != nil {
			MetricErrorsTotal.WithLabelValues("write_failed").Inc()
			return
		}
	}

	MetricRelayTotal.WithLabelValues(sni).Inc()
	Stats.RecordRelay()

	// Clear deadlines for relay
	clientConn.SetDeadline(time.Time{})
	upConn.SetDeadline(time.Time{})

	// Relay bidirectionally
	done := make(chan struct{}, 2)
	var upBytes, downBytes int64

	copyData := func(dst, src net.Conn, bytes *int64) {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			src.SetDeadline(time.Now().Add(timeout))
			select {
			case <-ctx.Done():
				return
			default:
			}
			nr, er := src.Read(buf)
			if nr > 0 {
				nw, ew := dst.Write(buf[:nr])
				if nw > 0 {
					*bytes += int64(nw)
				}
				if ew != nil {
					break
				}
			}
			if er != nil {
				break
			}
		}
	}

	go copyData(upConn, clientConn, &upBytes)
	go copyData(clientConn, upConn, &downBytes)

	select {
	case <-done:
	case <-ctx.Done():
	}

	// Record metrics
	duration := time.Since(startTime).Seconds()
	MetricConnectionDuration.Observe(duration)
	MetricBytesTotal.WithLabelValues(sni, "upstream").Add(float64(upBytes))
	MetricBytesTotal.WithLabelValues(sni, "downstream").Add(float64(downBytes))
	Stats.RecordBytes(upBytes + downBytes)

	ui.LogRelay(sni, clientConn.RemoteAddr().String(), upBytes, downBytes)
}

// handleInternalAPI serves the Stats API directly on the hijacked connection.
// This allows port 443 to be shared between Signal traffic and the web API.
func handleInternalAPI(conn net.Conn, initialData []byte) {
	ui.LogStatus("info", "Handling API request from "+conn.RemoteAddr().String())
	
	// Create a combined reader that puts back the data we already read
	reader := io.MultiReader(bytes.NewReader(initialData), conn)
	br := bufio.NewReader(reader)

	// Read the HTTP request from the connection
	req, err := http.ReadRequest(br)
	if err != nil {
		if err != io.EOF {
			ui.LogStatus("error", "API ReadRequest error: "+err.Error())
		}
		return
	}

	// Create a simple response writer that writes directly to the connection
	w := &simpleResponseWriter{
		conn:   conn,
		header: make(http.Header),
	}

	// Route and handle the request
	switch req.URL.Path {
	case "/api/stats":
		StatsHandler(w, req)
	case "/api/history":
		HistoryHandler(w, req)
	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}

	// Final verification that headers were sent
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
}

// simpleResponseWriter implements http.ResponseWriter for our hijacked connection.
type simpleResponseWriter struct {
	conn        net.Conn
	header      http.Header
	wroteHeader bool
	status      int
}

func (w *simpleResponseWriter) Header() http.Header {
	return w.header
}

func (w *simpleResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(b)
}

func (w *simpleResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status

	// Write HTTP/1.1 response line
	fmt.Fprintf(w.conn, "HTTP/1.1 %d %s\r\n", status, http.StatusText(status))
	
	// Write headers
	w.header.Set("Date", time.Now().Format(http.TimeFormat))
	w.header.Set("Connection", "close") // Force close for simplicity
	
	for k, vv := range w.header {
		for _, v := range vv {
			fmt.Fprintf(w.conn, "%s: %s\r\n", k, v)
		}
	}
	
	// End of headers
	fmt.Fprintf(w.conn, "\r\n")
}
