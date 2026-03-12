package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestNewWebhookOutput_EmptyURL(t *testing.T) {
	_, err := NewWebhookOutput(config.AuditWebhookConfig{Enabled: true})
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestNewWebhookOutput_InvalidFlushInterval(t *testing.T) {
	_, err := NewWebhookOutput(config.AuditWebhookConfig{
		Enabled:       true,
		URL:           "http://localhost/events",
		FlushInterval: "not-a-duration",
	})
	if err == nil {
		t.Error("expected error for invalid flush_interval")
	}
}

func TestWebhookOutput_Name(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wo, err := NewWebhookOutput(config.AuditWebhookConfig{
		URL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wo.Close()

	if wo.Name() != "webhook" {
		t.Errorf("expected name webhook, got %s", wo.Name())
	}
}

func TestWebhookOutput_Delivery(t *testing.T) {
	var mu sync.Mutex
	var received []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
			if line != "" {
				received = append(received, line)
			}
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wo, err := NewWebhookOutput(config.AuditWebhookConfig{
		URL:           srv.URL,
		BatchSize:     5,
		FlushInterval: "100ms",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write exactly batch size to trigger immediate flush
	for i := range 5 {
		event := map[string]interface{}{"event_type": "test", "index": i}
		data, _ := json.Marshal(event)
		data = append(data, '\n')
		wo.Write(data)
	}

	// Wait for delivery
	time.Sleep(500 * time.Millisecond)
	wo.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 5 {
		t.Errorf("expected 5 events, got %d", len(received))
	}
}

func TestWebhookOutput_FlushInterval(t *testing.T) {
	var mu sync.Mutex
	var received []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
			if line != "" {
				received = append(received, line)
			}
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wo, err := NewWebhookOutput(config.AuditWebhookConfig{
		URL:           srv.URL,
		BatchSize:     1000, // Won't reach batch size
		FlushInterval: "100ms",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write fewer than batch size
	event := map[string]string{"event_type": "test"}
	data, _ := json.Marshal(event)
	data = append(data, '\n')
	wo.Write(data)

	// Wait for flush interval to trigger
	time.Sleep(300 * time.Millisecond)
	wo.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Errorf("expected 1 event via flush interval, got %d", len(received))
	}
}

func TestWebhookOutput_CustomHeaders(t *testing.T) {
	var headerReceived string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerReceived = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wo, err := NewWebhookOutput(config.AuditWebhookConfig{
		URL:           srv.URL,
		BatchSize:     1,
		FlushInterval: "50ms",
		Headers:       map[string]string{"Authorization": "Splunk test-token"},
	})
	if err != nil {
		t.Fatal(err)
	}

	wo.Write([]byte(`{"event_type":"test"}` + "\n"))
	time.Sleep(200 * time.Millisecond)
	wo.Close()

	if headerReceived != "Splunk test-token" {
		t.Errorf("expected Authorization header, got %q", headerReceived)
	}
}

func TestWebhookOutput_RetryOn503(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wo, err := NewWebhookOutput(config.AuditWebhookConfig{
		URL:        srv.URL,
		BatchSize:  1,
		MaxRetries: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	wo.Write([]byte(`{"event_type":"test"}` + "\n"))
	time.Sleep(2 * time.Second) // Allow time for retries
	wo.Close()

	got := attempts.Load()
	if got < 3 {
		t.Errorf("expected at least 3 attempts (2 failures + 1 success), got %d", got)
	}
}

func TestWebhookOutput_NoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	wo, err := NewWebhookOutput(config.AuditWebhookConfig{
		URL:        srv.URL,
		BatchSize:  1,
		MaxRetries: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	wo.Write([]byte(`{"event_type":"test"}` + "\n"))
	time.Sleep(500 * time.Millisecond)
	wo.Close()

	got := attempts.Load()
	if got != 1 {
		t.Errorf("expected exactly 1 attempt for 4xx, got %d", got)
	}
}

func TestWebhookOutput_DropOnOverflow(t *testing.T) {
	// Server that blocks forever
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Block
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wo, err := NewWebhookOutput(config.AuditWebhookConfig{
		URL:        srv.URL,
		BatchSize:  100000, // Won't flush by size
		BufferSize: 5,      // Very small buffer
	})
	if err != nil {
		t.Fatal(err)
	}

	// Fill buffer plus overflow
	for range 20 {
		wo.Write([]byte(`{"event_type":"test"}` + "\n"))
	}

	// Should not block — overflow is silently dropped
	wo.Close()
}

func TestWebhookOutput_GracefulShutdownFlushes(t *testing.T) {
	var mu sync.Mutex
	var received int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		for _, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
			if line != "" {
				received++
			}
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wo, err := NewWebhookOutput(config.AuditWebhookConfig{
		URL:           srv.URL,
		BatchSize:     1000,  // Won't trigger by size
		FlushInterval: "10s", // Won't trigger by time
	})
	if err != nil {
		t.Fatal(err)
	}

	// Write events that won't be flushed by size or interval
	for range 3 {
		wo.Write([]byte(`{"event_type":"test"}` + "\n"))
	}

	// Close should drain remaining events
	wo.Close()

	mu.Lock()
	defer mu.Unlock()
	if received != 3 {
		t.Errorf("expected 3 events flushed on shutdown, got %d", received)
	}
}
