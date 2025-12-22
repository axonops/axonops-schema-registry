// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	config     config.RateLimitConfig
	mu         sync.Mutex
	global     *tokenBucket
	clients    map[string]*tokenBucket
	endpoints  map[string]*tokenBucket
}

// tokenBucket implements the token bucket algorithm.
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(cfg config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:    cfg,
		clients:   make(map[string]*tokenBucket),
		endpoints: make(map[string]*tokenBucket),
	}

	if cfg.Enabled {
		rl.global = newTokenBucket(float64(cfg.BurstSize), float64(cfg.RequestsPerSecond))
	}

	return rl
}

// newTokenBucket creates a new token bucket.
func newTokenBucket(maxTokens, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// allow checks if a request is allowed and consumes a token if so.
func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	// Check and consume token
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

// remaining returns the number of remaining tokens.
func (tb *tokenBucket) remaining() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return int(tb.tokens)
}

// Middleware returns HTTP middleware for rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		var bucket *tokenBucket
		var key string

		// Determine which bucket to use
		if rl.config.PerClient {
			key = getClientIP(r)
			bucket = rl.getClientBucket(key)
		} else if rl.config.PerEndpoint {
			key = r.Method + ":" + r.URL.Path
			bucket = rl.getEndpointBucket(key)
		} else {
			bucket = rl.global
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.config.RequestsPerSecond))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(bucket.remaining()))

		if !bucket.allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientBucket returns the token bucket for a client.
func (rl *RateLimiter) getClientBucket(clientIP string) *tokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.clients[clientIP]
	if !ok {
		bucket = newTokenBucket(float64(rl.config.BurstSize), float64(rl.config.RequestsPerSecond))
		rl.clients[clientIP] = bucket
	}

	return bucket
}

// getEndpointBucket returns the token bucket for an endpoint.
func (rl *RateLimiter) getEndpointBucket(endpoint string) *tokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.endpoints[endpoint]
	if !ok {
		bucket = newTokenBucket(float64(rl.config.BurstSize), float64(rl.config.RequestsPerSecond))
		rl.endpoints[endpoint] = bucket
	}

	return bucket
}

// getClientIP extracts the client IP from a request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP
		ips := splitAndTrim(xff, ",")
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// splitAndTrim splits a string and trims whitespace from each part.
func splitAndTrim(s, sep string) []string {
	var result []string
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && string(s[j]) != sep {
			j++
		}
		part := s[i:j]
		// Trim whitespace
		start, end := 0, len(part)
		for start < end && (part[start] == ' ' || part[start] == '\t') {
			start++
		}
		for end > start && (part[end-1] == ' ' || part[end-1] == '\t') {
			end--
		}
		if start < end {
			result = append(result, part[start:end])
		}
		i = j + 1
	}
	return result
}

// CleanupStaleClients removes client buckets that haven't been used recently.
func (rl *RateLimiter) CleanupStaleClients(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, bucket := range rl.clients {
		bucket.mu.Lock()
		if now.Sub(bucket.lastRefill) > maxAge {
			delete(rl.clients, key)
		}
		bucket.mu.Unlock()
	}
}
