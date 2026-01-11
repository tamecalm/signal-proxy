package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"time"

	"signal-proxy/internal/config"
	"signal-proxy/internal/ui"
)

// Server handles TLS connections and proxies them to Signal servers.
type Server struct {
	Config   *config.Config
	ln       net.Listener
	connSem  chan struct{}   // Semaphore for connection limiting
	wg       sync.WaitGroup  // Tracks active connections for graceful shutdown
	shutdown chan struct{}   // Signals shutdown to accept loop
}

// NewServer creates a new proxy server with the given configuration.
func NewServer(cfg *config.Config) *Server {
	return &Server{
		Config:   cfg,
		connSem:  make(chan struct{}, cfg.MaxConns),
		shutdown: make(chan struct{}),
	}
}

// Start begins accepting connections. It blocks until shutdown or error.
// The context is used for graceful shutdown - cancel it to initiate shutdown.
func (s *Server) Start(ctx context.Context) error {
	// 1. Configure TLS
	cert, err := tls.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"h2", "http/1.1"},
	}

	// 2. Start Listener
	s.ln, err = tls.Listen("tcp", s.Config.Listen, tlsConfig)
	if err != nil {
		return err
	}

	ui.LogStatus("success", "Proxy active on "+s.Config.Listen)
	ui.LogStatus("info", "Max connections: "+itoa(s.Config.MaxConns))

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

		// Set accept deadline to check shutdown periodically
		if tcpLn, ok := s.ln.(*net.TCPListener); ok {
			tcpLn.SetDeadline(time.Now().Add(1 * time.Second))
		}

		conn, err := s.ln.Accept()
		if err != nil {
			// Check if it's a timeout (normal during shutdown checks)
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// Check if listener was closed
			select {
			case <-s.shutdown:
				return s.drainConnections()
			default:
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
	activeConns := int(MetricActiveConns.Get())
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
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
