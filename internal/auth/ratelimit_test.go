package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestRateLimiter_GlobalLimit(t *testing.T) {
	rl := NewRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 2,
		BurstSize:         2,
		PerClient:         false,
		PerEndpoint:       false,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 requests should succeed (burst size)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rr.Code)
	}

	// Check Retry-After header
	if rr.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestRateLimiter_PerClientLimit(t *testing.T) {
	rl := NewRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstSize:         1,
		PerClient:         true,
		PerEndpoint:       false,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First client exhausts their limit
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("first client request 1: expected 200, got %d", rr1.Code)
	}

	// First client should be rate limited
	rr1_2 := httptest.NewRecorder()
	handler.ServeHTTP(rr1_2, req1)
	if rr1_2.Code != http.StatusTooManyRequests {
		t.Errorf("first client request 2: expected 429, got %d", rr1_2.Code)
	}

	// Second client should still be allowed (separate bucket)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("second client request: expected 200, got %d", rr2.Code)
	}
}

func TestRateLimiter_PerEndpointLimit(t *testing.T) {
	rl := NewRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		BurstSize:         1,
		PerClient:         false,
		PerEndpoint:       true,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request to /test exhausts that endpoint's limit
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("endpoint /test request 1: expected 200, got %d", rr1.Code)
	}

	// Second request to /test should be rate limited
	rr1_2 := httptest.NewRecorder()
	handler.ServeHTTP(rr1_2, req1)
	if rr1_2.Code != http.StatusTooManyRequests {
		t.Errorf("endpoint /test request 2: expected 429, got %d", rr1_2.Code)
	}

	// Request to /other should still succeed (separate bucket)
	req2 := httptest.NewRequest(http.MethodGet, "/other", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("endpoint /other request: expected 200, got %d", rr2.Code)
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	rl := NewRateLimiter(config.RateLimitConfig{
		Enabled: false,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// All requests should succeed when rate limiting is disabled
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}
}

func TestRateLimiter_RateLimitHeaders(t *testing.T) {
	rl := NewRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         5,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-RateLimit-Limit") != "10" {
		t.Errorf("expected X-RateLimit-Limit 10, got %s", rr.Header().Get("X-RateLimit-Limit"))
	}

	remaining := rr.Header().Get("X-RateLimit-Remaining")
	if remaining == "" {
		t.Error("expected X-RateLimit-Remaining header to be set")
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := getClientIP(req)
	if ip != "203.0.113.195" {
		t.Errorf("expected IP 203.0.113.195, got %s", ip)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "203.0.113.195")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := getClientIP(req)
	if ip != "203.0.113.195" {
		t.Errorf("expected IP 203.0.113.195, got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr_IPv4(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	ip := getClientIP(req)
	if ip != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr_IPv6(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[::1]:54321"

	ip := getClientIP(req)
	if ip != "::1" {
		t.Errorf("expected IP ::1, got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr_IPv6Full(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "[2001:db8:85a3::8a2e:370:7334]:54321"

	ip := getClientIP(req)
	if ip != "2001:db8:85a3::8a2e:370:7334" {
		t.Errorf("expected IP 2001:db8:85a3::8a2e:370:7334, got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr_NoPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100"

	ip := getClientIP(req)
	if ip != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", ip)
	}
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	// Create a bucket that refills at 10 tokens/second with max 5 tokens
	bucket := newTokenBucket(5, 10)

	// Exhaust all tokens
	for i := 0; i < 5; i++ {
		if !bucket.allow() {
			t.Errorf("expected token %d to be available", i+1)
		}
	}

	// Next request should fail
	if bucket.allow() {
		t.Error("expected bucket to be empty")
	}

	// Remaining should be 0 or close to 0
	if bucket.remaining() > 1 {
		t.Errorf("expected remaining ~0, got %d", bucket.remaining())
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want []string
	}{
		{
			name: "simple split",
			s:    "a,b,c",
			sep:  ",",
			want: []string{"a", "b", "c"},
		},
		{
			name: "with spaces",
			s:    "a , b , c",
			sep:  ",",
			want: []string{"a", "b", "c"},
		},
		{
			name: "with tabs",
			s:    "a\t,\tb\t,\tc",
			sep:  ",",
			want: []string{"a", "b", "c"},
		},
		{
			name: "empty parts",
			s:    "a,,b",
			sep:  ",",
			want: []string{"a", "b"},
		},
		{
			name: "single value",
			s:    "only",
			sep:  ",",
			want: []string{"only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.s, tt.sep)
			if len(got) != len(tt.want) {
				t.Errorf("len mismatch: got %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("element %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
