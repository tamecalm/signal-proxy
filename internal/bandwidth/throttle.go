package bandwidth

import (
	"net"
	"sync"
	"time"
)

// ThrottledConn wraps a net.Conn with per-user bandwidth speed limiting.
// Uses a token bucket algorithm — tokens represent bytes.
// Only activated when speedMbps > 0.
type ThrottledConn struct {
	net.Conn
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // bytes per second
	lastRefill time.Time
}

// NewThrottledConn wraps a connection with an optional speed limit.
// speedMbps is the max speed in megabits per second. 0 = no throttle (returns conn as-is).
func NewThrottledConn(conn net.Conn, speedMbps int) net.Conn {
	if speedMbps <= 0 {
		return conn // no throttle
	}

	bytesPerSec := float64(speedMbps) * 1024 * 1024 / 8 // Mbps → bytes/sec

	// Allow burst up to 1 second of bandwidth
	maxTokens := bytesPerSec

	return &ThrottledConn{
		Conn:       conn,
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: bytesPerSec,
		lastRefill: time.Now(),
	}
}

// Read implements io.Reader with throttling
func (tc *ThrottledConn) Read(b []byte) (int, error) {
	tc.waitForTokens(len(b))
	n, err := tc.Conn.Read(b)
	if n > 0 {
		tc.consumeTokens(n)
	}
	return n, err
}

// Write implements io.Writer with throttling
func (tc *ThrottledConn) Write(b []byte) (int, error) {
	tc.waitForTokens(len(b))
	n, err := tc.Conn.Write(b)
	if n > 0 {
		tc.consumeTokens(n)
	}
	return n, err
}

func (tc *ThrottledConn) waitForTokens(needed int) {
	for {
		tc.mu.Lock()
		tc.refill()
		if tc.tokens >= 1 {
			tc.mu.Unlock()
			return
		}
		// Calculate how long to wait for at least some tokens
		deficit := float64(needed) - tc.tokens
		if deficit < 1 {
			deficit = 1
		}
		waitDuration := time.Duration(deficit / tc.refillRate * float64(time.Second))
		if waitDuration < time.Millisecond {
			waitDuration = time.Millisecond
		}
		if waitDuration > 100*time.Millisecond {
			waitDuration = 100 * time.Millisecond
		}
		tc.mu.Unlock()
		time.Sleep(waitDuration)
	}
}

func (tc *ThrottledConn) consumeTokens(n int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.tokens -= float64(n)
}

func (tc *ThrottledConn) refill() {
	now := time.Now()
	elapsed := now.Sub(tc.lastRefill).Seconds()
	tc.tokens += elapsed * tc.refillRate
	if tc.tokens > tc.maxTokens {
		tc.tokens = tc.maxTokens
	}
	tc.lastRefill = now
}
