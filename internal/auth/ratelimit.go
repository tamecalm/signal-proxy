package auth

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter per user
type RateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
	limits  map[string]int // RPM limit per user
}

type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		limits:  make(map[string]int),
	}
}

// SetLimit sets the rate limit for a user in requests per minute
func (r *RateLimiter) SetLimit(username string, rpm int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limits[username] = rpm

	// Initialize or update bucket
	// Allow burst up to 10% of RPM or minimum 10 requests
	maxTokens := float64(rpm) / 6 // ~10 seconds worth
	if maxTokens < 10 {
		maxTokens = 10
	}

	r.buckets[username] = &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: float64(rpm) / 60.0, // tokens per second
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed for the user
// Returns true if allowed, false if rate limited
func (r *RateLimiter) Allow(username string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, exists := r.buckets[username]
	if !exists {
		// No limit configured, allow
		return true
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * bucket.refillRate
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastRefill = now

	// Try to consume a token
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}

	return false
}

// GetRemainingTokens returns the current token count for a user (for metrics)
func (r *RateLimiter) GetRemainingTokens(username string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bucket, exists := r.buckets[username]
	if !exists {
		return -1 // No limit
	}

	return bucket.tokens
}
