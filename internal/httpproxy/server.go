package httpproxy

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"signal-proxy/internal/auth"
	"signal-proxy/internal/bandwidth"
	"signal-proxy/internal/config"
	"signal-proxy/internal/pac"
	"signal-proxy/internal/ui"
)

// Server is an HTTP/HTTPS forward proxy with authentication
type Server struct {
	Config    *config.Config
	UserStore *auth.UserStore
	Bandwidth *bandwidth.Tracker

	httpServer  *http.Server
	httpsServer *http.Server
	ln          net.Listener
	tlsLn       net.Listener
	wg          sync.WaitGroup
	shutdown    chan struct{}

	// Connection tracking
	connCount   int
	connCountMu sync.Mutex

	// Transport for outgoing HTTP requests (with connection pooling)
	transport *http.Transport

	// PAC handler
	pacHandler *pac.Handler
}

// NewServer creates a new HTTP/HTTPS proxy server
func NewServer(cfg *config.Config, userStore *auth.UserStore, bw *bandwidth.Tracker) *Server {
	srv := &Server{
		Config:    cfg,
		UserStore: userStore,
		Bandwidth: bw,
		shutdown:  make(chan struct{}),
		transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	// Initialize PAC handler if enabled
	if cfg.Env.PACEnabled {
		pacConfig := &pac.Config{
			Enabled:      cfg.Env.PACEnabled,
			ProxyHost:    cfg.Env.Domain,
			HTTPPort:     strings.TrimPrefix(cfg.Env.HTTPProxyPort, ":"),
			SOCKS5Port:   strings.TrimPrefix(cfg.Env.SOCKS5Port, ":"),
			Token:        cfg.Env.PACToken,
			DefaultUser:  cfg.Env.PACDefaultUser,
			RateLimitRPM: cfg.Env.PACRateLimitRPM,
		}
		srv.pacHandler = pac.NewHandler(pacConfig, userStore)
		ui.LogStatus("info", "PAC endpoint enabled at /proxy.pac")
	}

	return srv
}

// Start begins accepting HTTP proxy connections
func (s *Server) Start(ctx context.Context) error {
	// Create HTTP handler
	handler := http.HandlerFunc(s.handleRequest)

	// Start plain HTTP proxy listener
	httpAddr := s.Config.Env.HTTPProxyPort
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	var err error
	s.ln, err = net.Listen("tcp", httpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", httpAddr, err)
	}

	s.httpServer = &http.Server{
		Handler:      handler,
		ReadTimeout:  0, // Disabled: CONNECT tunnels are long-lived, managed per-handler
		WriteTimeout: 0, // Disabled: CONNECT tunnels are long-lived, managed per-handler
		IdleTimeout:  120 * time.Second,
	}

	ui.LogStatus("info", "HTTP Proxy listening on "+httpAddr)

	// Start HTTPS proxy listener if TLS is configured
	if s.Config.Env.HTTPProxyTLS && s.Config.CertFile != "" && s.Config.KeyFile != "" {
		httpsAddr := s.Config.Env.HTTPProxyTLSPort
		if httpsAddr == "" {
			httpsAddr = ":8443"
		}

		cert, err := tls.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS cert: %w", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		s.tlsLn, err = tls.Listen("tcp", httpsAddr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to listen TLS on %s: %w", httpsAddr, err)
		}

		s.httpsServer = &http.Server{
			Handler:      handler,
			ReadTimeout:  0, // Disabled: CONNECT tunnels are long-lived, managed per-handler
			WriteTimeout: 0, // Disabled: CONNECT tunnels are long-lived, managed per-handler
			IdleTimeout:  120 * time.Second,
		}

		ui.LogStatus("info", "HTTPS Proxy listening on "+httpsAddr+" (TLS)")

		// Start HTTPS server
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.httpsServer.Serve(s.tlsLn); err != nil && err != http.ErrServerClosed {
				ui.LogStatus("error", "HTTPS proxy error: "+err.Error())
			}
		}()
	}

	// Monitor for shutdown
	go s.watchShutdown(ctx)

	// Start HTTP server (blocking)
	if err := s.httpServer.Serve(s.ln); err != nil && err != http.ErrServerClosed {
		return err
	}

	s.wg.Wait()
	return nil
}

// watchShutdown monitors context for cancellation
func (s *Server) watchShutdown(ctx context.Context) {
	<-ctx.Done()
	close(s.shutdown)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.httpServer != nil {
		s.httpServer.Shutdown(shutdownCtx)
	}
	if s.httpsServer != nil {
		s.httpsServer.Shutdown(shutdownCtx)
	}
}

// handleRequest processes incoming proxy requests
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Handle PAC file requests before proxy logic
	if s.pacHandler != nil && (r.URL.Path == "/proxy.pac" || r.RequestURI == "/proxy.pac") {
		s.pacHandler.ServeHTTP(w, r)
		return
	}

	startTime := time.Now()
	clientIP := r.RemoteAddr

	// Check IP whitelist
	if !s.UserStore.CheckIPAllowed(clientIP) {
		MetricAuthFailures.WithLabelValues("ip_blocked").Inc()
		ui.LogStatus("warn", "IP blocked: "+clientIP)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Always require authentication
	var user *auth.User
	username, password, ok := parseProxyAuth(r)
	if !ok {
		MetricAuthFailures.WithLabelValues("no_credentials").Inc()
		w.Header().Set("Proxy-Authenticate", `Basic realm="Proxy Authentication Required"`)
		http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
		return
	}

	var valid bool
	user, valid = s.UserStore.ValidateCredentials(username, password)
	if !valid {
		MetricAuthFailures.WithLabelValues("invalid_credentials").Inc()
		ui.LogStatus("warn", "Auth failed for user: "+username+" from "+clientIP)
		w.Header().Set("Proxy-Authenticate", `Basic realm="Proxy Authentication Required"`)
		http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
		return
	}

	// Determine if this user is a super_admin connecting from a trusted IP
	isSuperAdmin := false
	if user.Role == "super_admin" {
		if _, ok := s.UserStore.IsSuperAdminIP(clientIP); ok {
			isSuperAdmin = true
			ui.LogStatus("info", "HTTP super_admin verified: "+username+" from "+clientIP)
		}
	}

	if !isSuperAdmin {
		// Check rate limit
		if !s.UserStore.CheckRateLimit(username) {
			MetricRateLimited.WithLabelValues(username).Inc()
			ui.LogStatus("warn", "Rate limited: "+username)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// Check account expiry
		if !s.UserStore.CheckExpiry(username) {
			ui.LogStatus("warn", "Account expired: "+username)
			http.Error(w, "Account Expired", http.StatusForbidden)
			return
		}

		// Check bandwidth allowance
		if s.Bandwidth != nil && !s.Bandwidth.CheckAllowance(username, user.BandwidthLimitGB) {
			ui.LogStatus("warn", "Bandwidth exceeded: "+username)
			http.Error(w, "Bandwidth Limit Exceeded", http.StatusForbidden)
			return
		}

		// Check concurrent connection limit
		if s.Bandwidth != nil && !s.Bandwidth.CheckConnLimit(username, user.MaxConnections) {
			ui.LogStatus("warn", "Connection limit reached: "+username)
			http.Error(w, "Connection Limit Reached", http.StatusTooManyRequests)
			return
		}
	}

	// Track connection
	MetricActiveConns.Inc()
	defer MetricActiveConns.Dec()

	// Track per-user connection count
	if s.Bandwidth != nil && user != nil {
		s.Bandwidth.IncrementConns(user.Username)
		defer s.Bandwidth.DecrementConns(user.Username)
	}

	// Handle the request based on method
	if r.Method == http.MethodConnect {
		s.handleConnect(w, r, user, startTime)
	} else {
		s.handleHTTP(w, r, user, startTime)
	}
}

// handleConnect handles HTTPS tunneling via CONNECT method
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request, user *auth.User, startTime time.Time) {
	MetricRequests.WithLabelValues(user.Username, "CONNECT").Inc()

	// Get the target host
	targetHost := r.Host
	if !strings.Contains(targetHost, ":") {
		targetHost = targetHost + ":443"
	}

	// Connect to target with TCP keep-alive to prevent mobile NAT drops
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	targetConn, err := dialer.Dial("tcp", targetHost)
	if err != nil {
		MetricErrors.WithLabelValues("dial_failed").Inc()
		http.Error(w, "Failed to connect to target", http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	// Hijack the client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		MetricErrors.WithLabelValues("hijack_failed").Inc()
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		MetricErrors.WithLabelValues("hijack_failed").Inc()
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Enable TCP keep-alive on client side too (if underlying conn is TCP)
	if tcpConn, ok := clientConn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	// Send 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Apply optional speed throttle
	var relayClient, relayTarget net.Conn
	relayClient = clientConn
	relayTarget = targetConn
	if user.BandwidthSpeedMbps > 0 {
		relayClient = bandwidth.NewThrottledConn(clientConn, user.BandwidthSpeedMbps).(*bandwidth.ThrottledConn)
		relayTarget = bandwidth.NewThrottledConn(targetConn, user.BandwidthSpeedMbps).(*bandwidth.ThrottledConn)
	}

	// Relay data bidirectionally with buffered I/O
	var upBytes, downBytes int64
	done := make(chan struct{}, 2)

	copyBuf := func(dst, src net.Conn, bytes *int64) {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024) // 32KB buffer for efficient relay
		n, _ := io.CopyBuffer(dst, src, buf)
		*bytes = n
		// Half-close to signal the other side gracefully
		if tc, ok := dst.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}

	go copyBuf(relayTarget, relayClient, &upBytes)
	go copyBuf(relayClient, relayTarget, &downBytes)

	// Wait for both directions to finish for clean shutdown
	<-done
	<-done

	// Record metrics
	duration := time.Since(startTime).Seconds()
	MetricBytes.WithLabelValues(user.Username, "upstream").Add(float64(upBytes))
	MetricBytes.WithLabelValues(user.Username, "downstream").Add(float64(downBytes))
	MetricDuration.Observe(duration)

	// Record bandwidth usage for tracking
	if s.Bandwidth != nil {
		s.Bandwidth.RecordBytes(user.Username, upBytes, downBytes)
	}
}

// handleHTTP handles plain HTTP proxy requests
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request, user *auth.User, startTime time.Time) {
	MetricRequests.WithLabelValues(user.Username, r.Method).Inc()

	// Ensure absolute URL
	if !r.URL.IsAbs() {
		http.Error(w, "Bad Request: absolute URL required", http.StatusBadRequest)
		return
	}

	// Create outgoing request
	outReq := r.Clone(r.Context())

	// Remove hop-by-hop headers
	removeHopByHopHeaders(outReq.Header)

	// Remove Proxy-Authorization header
	outReq.Header.Del("Proxy-Authorization")

	// Perform the request
	resp, err := s.transport.RoundTrip(outReq)
	if err != nil {
		MetricErrors.WithLabelValues("request_failed").Inc()
		http.Error(w, "Failed to reach target", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	written, _ := io.Copy(w, resp.Body)

	// Record metrics
	duration := time.Since(startTime).Seconds()
	MetricBytes.WithLabelValues(user.Username, "downstream").Add(float64(written))
	MetricDuration.Observe(duration)

	// Record bandwidth usage for tracking
	if s.Bandwidth != nil {
		s.Bandwidth.RecordBytes(user.Username, 0, written)
	}
}

// parseProxyAuth extracts username and password from Proxy-Authorization header
func parseProxyAuth(r *http.Request) (username, password string, ok bool) {
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		return "", "", false
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}

	credentials := string(decoded)
	idx := strings.IndexByte(credentials, ':')
	if idx < 0 {
		return "", "", false
	}

	return credentials[:idx], credentials[idx+1:], true
}

// removeHopByHopHeaders removes headers that should not be forwarded
func removeHopByHopHeaders(h http.Header) {
	hopByHop := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Proxy-Connection",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, header := range hopByHop {
		h.Del(header)
	}
}

// Shutdown gracefully stops the proxy server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		s.httpServer.Shutdown(ctx)
	}
	if s.httpsServer != nil {
		s.httpsServer.Shutdown(ctx)
	}
	s.wg.Wait()
	return nil
}
