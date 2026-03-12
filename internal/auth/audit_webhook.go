// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// WebhookOutput delivers audit events to an HTTP endpoint with batching,
// configurable flush interval, exponential backoff retry on 5xx, and
// overflow drop when the buffer is full.
type WebhookOutput struct {
	url        string
	headers    map[string]string
	batchSize  int
	flushIvl   time.Duration
	timeout    time.Duration
	maxRetries int
	client     *http.Client
	metrics    AuditMetrics // optional

	ch     chan []byte
	wg     sync.WaitGroup
	stopCh chan struct{}
}

// NewWebhookOutput creates a webhook audit output from config.
// Defaults: batchSize=100, flushInterval=5s, timeout=10s, maxRetries=3, bufferSize=10000.
func NewWebhookOutput(cfg config.AuditWebhookConfig) (*WebhookOutput, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("audit webhook URL is required")
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	flushIvl, err := parseDurationDefault(cfg.FlushInterval, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid flush_interval: %w", err)
	}

	timeout, err := parseDurationDefault(cfg.Timeout, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	bufferSize := cfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 10000
	}

	w := &WebhookOutput{
		url:        cfg.URL,
		headers:    cfg.Headers,
		batchSize:  batchSize,
		flushIvl:   flushIvl,
		timeout:    timeout,
		maxRetries: maxRetries,
		client:     &http.Client{Timeout: timeout},
		ch:         make(chan []byte, bufferSize),
		stopCh:     make(chan struct{}),
	}

	w.wg.Add(1)
	go w.flushLoop()

	return w, nil
}

// SetMetrics sets the optional metrics recorder for webhook telemetry.
func (w *WebhookOutput) SetMetrics(m AuditMetrics) {
	w.metrics = m
}

// Write enqueues audit event data for delivery.
// If the buffer is full, the event is silently dropped.
func (w *WebhookOutput) Write(data []byte) error {
	// Make a copy since the caller may reuse the slice.
	cp := make([]byte, len(data))
	copy(cp, data)

	select {
	case w.ch <- cp:
	default:
		// Buffer full — drop event
		slog.Warn("audit webhook buffer full, dropping event")
		if w.metrics != nil {
			w.metrics.RecordAuditWebhookDrop()
		}
	}
	return nil
}

// Close signals the flush loop to drain remaining events and waits
// up to 5 seconds for completion.
func (w *WebhookOutput) Close() error {
	close(w.stopCh)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		slog.Warn("audit webhook shutdown timed out after 5s")
	}
	return nil
}

// Name returns "webhook".
func (w *WebhookOutput) Name() string { return "webhook" }

// flushLoop batches events and sends them on size or interval.
func (w *WebhookOutput) flushLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.flushIvl)
	defer ticker.Stop()

	batch := make([][]byte, 0, w.batchSize)

	for {
		select {
		case data := <-w.ch:
			batch = append(batch, data)
			if len(batch) >= w.batchSize {
				w.sendBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				w.sendBatch(batch)
				batch = batch[:0]
			}
		case <-w.stopCh:
			// Drain remaining events
			for {
				select {
				case data := <-w.ch:
					batch = append(batch, data)
					if len(batch) >= w.batchSize {
						w.sendBatch(batch)
						batch = batch[:0]
					}
				default:
					if len(batch) > 0 {
						w.sendBatch(batch)
					}
					return
				}
			}
		}
	}
}

// sendBatch sends a batch of events to the webhook endpoint.
// Events are concatenated (each already has a trailing newline).
func (w *WebhookOutput) sendBatch(batch [][]byte) {
	start := time.Now()
	var buf bytes.Buffer
	for _, data := range batch {
		buf.Write(data)
	}

	defer func() {
		if w.metrics != nil {
			w.metrics.RecordAuditWebhookFlush(len(batch), time.Since(start))
		}
	}()

	for attempt := range w.maxRetries {
		req, err := http.NewRequest("POST", w.url, bytes.NewReader(buf.Bytes()))
		if err != nil {
			slog.Warn("audit webhook: failed to create request", slog.String("error", err.Error()))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range w.headers {
			req.Header.Set(k, v)
		}

		resp, err := w.client.Do(req)
		if err != nil {
			slog.Warn("audit webhook: request failed",
				slog.String("error", err.Error()),
				slog.Int("attempt", attempt+1),
			)
			backoff(attempt)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return // Success
		}

		if resp.StatusCode >= 500 {
			slog.Warn("audit webhook: server error",
				slog.Int("status", resp.StatusCode),
				slog.Int("attempt", attempt+1),
			)
			backoff(attempt)
			continue
		}

		// 4xx — don't retry client errors
		slog.Warn("audit webhook: client error",
			slog.Int("status", resp.StatusCode),
			slog.Int("events", len(batch)),
		)
		return
	}

	slog.Warn("audit webhook: exhausted retries",
		slog.Int("max_retries", w.maxRetries),
		slog.Int("events_dropped", len(batch)),
	)
}

// backoff sleeps for exponential backoff duration.
func backoff(attempt int) {
	d := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	time.Sleep(d)
}

// parseDurationDefault parses a duration string, returning defaultVal for empty strings.
func parseDurationDefault(s string, defaultVal time.Duration) (time.Duration, error) {
	if s == "" {
		return defaultVal, nil
	}
	return time.ParseDuration(s)
}
