package socks5

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"signal-proxy/internal/auth"
	"signal-proxy/internal/bandwidth"
	"signal-proxy/internal/config"
	"signal-proxy/internal/ui"
)

// SOCKS5 protocol constants
const (
	Version5 = 0x05

	// Authentication methods
	MethodNoAuth       = 0x00
	MethodUserPass     = 0x02
	MethodNoAcceptable = 0xFF

	// User/pass auth version
	UserPassVersion = 0x01

	// Commands
	CmdConnect = 0x01
	CmdBind    = 0x02
	CmdUDP     = 0x03

	// Address types
	AddrTypeIPv4   = 0x01
	AddrTypeDomain = 0x03
	AddrTypeIPv6   = 0x04

	// Reply codes
	ReplySucceeded          = 0x00
	ReplyGeneralFailure     = 0x01
	ReplyConnectionNotAllowed = 0x02
	ReplyNetworkUnreachable = 0x03
	ReplyHostUnreachable    = 0x04
	ReplyConnectionRefused  = 0x05
	ReplyTTLExpired         = 0x06
	ReplyCmdNotSupported    = 0x07
	ReplyAddrTypeNotSupported = 0x08
)

// Server is a SOCKS5 proxy server with authentication
type Server struct {
	Config    *config.Config
	UserStore *auth.UserStore
	Bandwidth *bandwidth.Tracker

	ln       net.Listener
	wg       sync.WaitGroup
	shutdown chan struct{}
}

// NewServer creates a new SOCKS5 proxy server
func NewServer(cfg *config.Config, userStore *auth.UserStore, bw *bandwidth.Tracker) *Server {
	return &Server{
		Config:    cfg,
		UserStore: userStore,
		Bandwidth: bw,
		shutdown:  make(chan struct{}),
	}
}

// Start begins accepting SOCKS5 connections
func (s *Server) Start(ctx context.Context) error {
	addr := s.Config.Env.SOCKS5Port
	if addr == "" {
		addr = ":1080"
	}

	var err error
	s.ln, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	ui.LogStatus("info", "SOCKS5 Proxy listening on "+addr)

	// Monitor for shutdown
	go s.watchShutdown(ctx)

	// Accept loop
	for {
		select {
		case <-s.shutdown:
			return nil
		default:
		}

		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return nil
			default:
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return err
			}
		}

		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			s.handleConnection(ctx, c)
		}(conn)
	}
}

// watchShutdown monitors context for cancellation
func (s *Server) watchShutdown(ctx context.Context) {
	<-ctx.Done()
	close(s.shutdown)
	if s.ln != nil {
		s.ln.Close()
	}
}

// handleConnection processes a SOCKS5 connection
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	startTime := time.Now()
	clientIP := conn.RemoteAddr().String()

	// Check IP whitelist
	if !s.UserStore.CheckIPAllowed(clientIP) {
		MetricAuthFailures.WithLabelValues("ip_blocked").Inc()
		ui.LogStatus("warn", "SOCKS5 IP blocked: "+clientIP)
		return
	}

	MetricActiveConns.Inc()
	defer MetricActiveConns.Dec()

	// Set initial timeout
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Always require username/password authentication
	var username string
	var err error
	username, err = s.handleMethodNegotiation(conn)
	if err != nil {
		ui.LogStatus("error", "SOCKS5 method negotiation failed: "+err.Error())
		return
	}

	// Determine if this user is a super_admin connecting from a trusted IP
	isSuperAdmin := false
	user := s.UserStore.GetUser(username)
	if user != nil && user.Role == "super_admin" {
		if _, ok := s.UserStore.IsSuperAdminIP(clientIP); ok {
			isSuperAdmin = true
			ui.LogStatus("info", "SOCKS5 super_admin verified: "+username+" from "+clientIP)
		}
	}

	if !isSuperAdmin {
		// Check rate limit
		if !s.UserStore.CheckRateLimit(username) {
			MetricRateLimited.WithLabelValues(username).Inc()
			ui.LogStatus("warn", "SOCKS5 rate limited: "+username)
			return
		}
	}

	// --- Bandwidth & plan enforcement (skip for super_admin) ---
	if !isSuperAdmin && user != nil {
		// Check account expiry
		if !s.UserStore.CheckExpiry(username) {
			ui.LogStatus("warn", "SOCKS5 account expired: "+username)
			return
		}

		// Check bandwidth allowance
		if s.Bandwidth != nil && !s.Bandwidth.CheckAllowance(username, user.BandwidthLimitGB) {
			ui.LogStatus("warn", "SOCKS5 bandwidth exceeded: "+username)
			return
		}

		// Check concurrent connection limit
		if s.Bandwidth != nil && !s.Bandwidth.CheckConnLimit(username, user.MaxConnections) {
			ui.LogStatus("warn", "SOCKS5 connection limit reached: "+username)
			return
		}
	}

	// Track per-user connection count
	if s.Bandwidth != nil {
		s.Bandwidth.IncrementConns(username)
		defer s.Bandwidth.DecrementConns(username)
	}

	// Step 2: Handle request
	targetAddr, err := s.handleRequest(conn)
	if err != nil {
		ui.LogStatus("error", "SOCKS5 request failed: "+err.Error())
		return
	}

	// Step 3: Connect to target
	targetConn, err := net.DialTimeout("tcp", targetAddr, 30*time.Second)
	if err != nil {
		s.sendReply(conn, ReplyHostUnreachable, nil)
		MetricErrors.WithLabelValues("dial_failed").Inc()
		return
	}
	defer targetConn.Close()

	// Send success reply
	localAddr := targetConn.LocalAddr().(*net.TCPAddr)
	s.sendReply(conn, ReplySucceeded, localAddr)

	MetricConnections.WithLabelValues(username).Inc()

	// Clear deadlines for relay
	conn.SetDeadline(time.Time{})
	targetConn.SetDeadline(time.Time{})

	// Apply optional speed throttle
	var relayClient, relayTarget net.Conn
	relayClient = conn
	relayTarget = targetConn
	if user != nil && user.BandwidthSpeedMbps > 0 {
		relayClient = bandwidth.NewThrottledConn(conn, user.BandwidthSpeedMbps).(*bandwidth.ThrottledConn)
		relayTarget = bandwidth.NewThrottledConn(targetConn, user.BandwidthSpeedMbps).(*bandwidth.ThrottledConn)
	}

	// Relay data bidirectionally
	var upBytes, downBytes int64
	done := make(chan struct{}, 2)

	go func() {
		n, _ := io.Copy(relayTarget, relayClient)
		upBytes = n
		done <- struct{}{}
	}()

	go func() {
		n, _ := io.Copy(relayClient, relayTarget)
		downBytes = n
		done <- struct{}{}
	}()

	// Wait for either direction to finish
	<-done

	// Record metrics
	duration := time.Since(startTime).Seconds()
	MetricBytes.WithLabelValues(username, "upstream").Add(float64(upBytes))
	MetricBytes.WithLabelValues(username, "downstream").Add(float64(downBytes))
	MetricDuration.Observe(duration)

	// Record bandwidth usage for tracking
	if s.Bandwidth != nil {
		s.Bandwidth.RecordBytes(username, upBytes, downBytes)
	}
}

// handleMethodNegotiation handles SOCKS5 method selection and authentication
func (s *Server) handleMethodNegotiation(conn net.Conn) (string, error) {
	// Read version and number of methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	if buf[0] != Version5 {
		return "", errors.New("unsupported SOCKS version")
	}

	numMethods := int(buf[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", err
	}

	// We only accept username/password authentication
	hasUserPass := false
	for _, method := range methods {
		if method == MethodUserPass {
			hasUserPass = true
			break
		}
	}

	if !hasUserPass {
		conn.Write([]byte{Version5, MethodNoAcceptable})
		MetricAuthFailures.WithLabelValues("no_auth_method").Inc()
		return "", errors.New("no acceptable auth method")
	}

	// Request username/password auth
	conn.Write([]byte{Version5, MethodUserPass})

	// Authenticate user
	return s.authenticateUser(conn)
}

// handleMethodNegotiationNoAuth handles SOCKS5 method negotiation accepting no-auth
func (s *Server) handleMethodNegotiationNoAuth(conn net.Conn) (string, error) {
	// Read version and number of methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	if buf[0] != Version5 {
		return "", errors.New("unsupported SOCKS version")
	}

	numMethods := int(buf[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", err
	}

	// Accept no-auth method
	conn.Write([]byte{Version5, MethodNoAuth})
	return "", nil // username will be set from the super_admin user
}

// authenticateUser handles username/password authentication (RFC 1929)
func (s *Server) authenticateUser(conn net.Conn) (string, error) {
	// Read auth version
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	if buf[0] != UserPassVersion {
		return "", errors.New("unsupported auth version")
	}

	// Read username
	usernameLen := int(buf[1])
	username := make([]byte, usernameLen)
	if _, err := io.ReadFull(conn, username); err != nil {
		return "", err
	}

	// Read password length
	if _, err := io.ReadFull(conn, buf[:1]); err != nil {
		return "", err
	}

	// Read password
	passwordLen := int(buf[0])
	password := make([]byte, passwordLen)
	if _, err := io.ReadFull(conn, password); err != nil {
		return "", err
	}

	// Validate credentials
	_, valid := s.UserStore.ValidateCredentials(string(username), string(password))
	if !valid {
		conn.Write([]byte{UserPassVersion, 0x01}) // Auth failure
		MetricAuthFailures.WithLabelValues("invalid_credentials").Inc()
		ui.LogStatus("warn", "SOCKS5 auth failed for: "+string(username))
		return "", errors.New("authentication failed")
	}

	// Auth success
	conn.Write([]byte{UserPassVersion, 0x00})
	return string(username), nil
}

// handleRequest handles SOCKS5 request
func (s *Server) handleRequest(conn net.Conn) (string, error) {
	// Read request header: VER, CMD, RSV, ATYP
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	if buf[0] != Version5 {
		return "", errors.New("unsupported version")
	}

	cmd := buf[1]
	addrType := buf[3]

	// We only support CONNECT
	if cmd != CmdConnect {
		s.sendReply(conn, ReplyCmdNotSupported, nil)
		return "", errors.New("unsupported command")
	}

	// Parse destination address
	var host string
	switch addrType {
	case AddrTypeIPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()

	case AddrTypeDomain:
		// Read domain length
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return "", err
		}
		domainLen := int(buf[0])
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		host = string(domain)

	case AddrTypeIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()

	default:
		s.sendReply(conn, ReplyAddrTypeNotSupported, nil)
		return "", errors.New("unsupported address type")
	}

	// Read port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portBuf)

	return fmt.Sprintf("%s:%d", host, port), nil
}

// sendReply sends a SOCKS5 reply
func (s *Server) sendReply(conn net.Conn, reply byte, addr *net.TCPAddr) {
	// Build reply: VER, REP, RSV, ATYP, BND.ADDR, BND.PORT
	resp := make([]byte, 10)
	resp[0] = Version5
	resp[1] = reply
	resp[2] = 0x00 // Reserved
	resp[3] = AddrTypeIPv4

	if addr != nil {
		ip := addr.IP.To4()
		if ip != nil {
			copy(resp[4:8], ip)
		}
		binary.BigEndian.PutUint16(resp[8:10], uint16(addr.Port))
	}

	conn.Write(resp)
}

// Shutdown gracefully stops the SOCKS5 server
func (s *Server) Shutdown(ctx context.Context) error {
	close(s.shutdown)
	if s.ln != nil {
		s.ln.Close()
	}
	s.wg.Wait()
	return nil
}
