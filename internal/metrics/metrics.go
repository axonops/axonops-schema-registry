// Package metrics provides Prometheus metrics for the schema registry.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the schema registry.
type Metrics struct {
	// Request metrics
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge

	// Schema metrics
	SchemasTotal      *prometheus.GaugeVec
	SubjectsTotal     prometheus.Gauge
	SchemaVersions    *prometheus.GaugeVec
	RegistrationsTotal *prometheus.CounterVec

	// Compatibility metrics
	CompatibilityChecks *prometheus.CounterVec
	CompatibilityErrors *prometheus.CounterVec

	// Storage metrics
	StorageOperations *prometheus.CounterVec
	StorageLatency    *prometheus.HistogramVec
	StorageErrors     *prometheus.CounterVec

	// Cache metrics
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
	CacheSize   *prometheus.GaugeVec

	// Auth metrics
	AuthAttempts *prometheus.CounterVec
	AuthFailures *prometheus.CounterVec
	AuthLatency  *prometheus.HistogramVec

	// Rate limit metrics
	RateLimitHits *prometheus.CounterVec

	registry *prometheus.Registry
}

// New creates a new Metrics instance with all collectors registered.
func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
	}

	// Request metrics
	m.RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "schema_registry_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	m.RequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "schema_registry_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// Schema metrics
	m.SchemasTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "schema_registry_schemas_total",
			Help: "Total number of schemas by type",
		},
		[]string{"type"},
	)

	m.SubjectsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "schema_registry_subjects_total",
			Help: "Total number of subjects",
		},
	)

	m.SchemaVersions = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "schema_registry_schema_versions",
			Help: "Number of versions per subject",
		},
		[]string{"subject"},
	)

	m.RegistrationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_registrations_total",
			Help: "Total number of schema registrations",
		},
		[]string{"type", "status"},
	)

	// Compatibility metrics
	m.CompatibilityChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_compatibility_checks_total",
			Help: "Total number of compatibility checks",
		},
		[]string{"type", "level", "result"},
	)

	m.CompatibilityErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_compatibility_errors_total",
			Help: "Total number of compatibility check errors",
		},
		[]string{"type", "level"},
	)

	// Storage metrics
	m.StorageOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_storage_operations_total",
			Help: "Total number of storage operations",
		},
		[]string{"backend", "operation"},
	)

	m.StorageLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "schema_registry_storage_latency_seconds",
			Help:    "Storage operation latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend", "operation"},
	)

	m.StorageErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_storage_errors_total",
			Help: "Total number of storage errors",
		},
		[]string{"backend", "operation"},
	)

	// Cache metrics
	m.CacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache"},
	)

	m.CacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache"},
	)

	m.CacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "schema_registry_cache_size",
			Help: "Current cache size",
		},
		[]string{"cache"},
	)

	// Auth metrics
	m.AuthAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"method"},
	)

	m.AuthFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_auth_failures_total",
			Help: "Total number of authentication failures",
		},
		[]string{"method", "reason"},
	)

	m.AuthLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "schema_registry_auth_latency_seconds",
			Help:    "Authentication latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Rate limit metrics
	m.RateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"client"},
	)

	// Register all collectors
	m.registry.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.RequestsInFlight,
		m.SchemasTotal,
		m.SubjectsTotal,
		m.SchemaVersions,
		m.RegistrationsTotal,
		m.CompatibilityChecks,
		m.CompatibilityErrors,
		m.StorageOperations,
		m.StorageLatency,
		m.StorageErrors,
		m.CacheHits,
		m.CacheMisses,
		m.CacheSize,
		m.AuthAttempts,
		m.AuthFailures,
		m.AuthLatency,
		m.RateLimitHits,
	)

	// Also register the default collectors (go runtime, process info)
	m.registry.MustRegister(prometheus.NewGoCollector())
	m.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	return m
}

// Handler returns an HTTP handler for the metrics endpoint.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// Middleware returns HTTP middleware that records request metrics.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		m.RequestsInFlight.Inc()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		m.RequestsInFlight.Dec()
		duration := time.Since(start).Seconds()

		// Normalize path for metrics (avoid high cardinality)
		path := normalizePath(r.URL.Path)

		m.RequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
		m.RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// normalizePath normalizes a URL path to reduce cardinality.
func normalizePath(path string) string {
	// Handle common schema registry paths
	switch {
	case startsWith(path, "/subjects/") && contains(path, "/versions/"):
		return "/subjects/{subject}/versions/{version}"
	case startsWith(path, "/subjects/") && endsWith(path, "/versions"):
		return "/subjects/{subject}/versions"
	case startsWith(path, "/subjects/"):
		return "/subjects/{subject}"
	case startsWith(path, "/schemas/ids/"):
		return "/schemas/ids/{id}"
	case startsWith(path, "/config/"):
		return "/config/{subject}"
	case startsWith(path, "/mode/"):
		return "/mode/{subject}"
	case startsWith(path, "/compatibility/subjects/"):
		return "/compatibility/subjects/{subject}/versions/{version}"
	}
	return path
}

// String helper functions to avoid importing strings package
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RecordSchemaRegistration records a schema registration attempt.
func (m *Metrics) RecordSchemaRegistration(schemaType string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.RegistrationsTotal.WithLabelValues(schemaType, status).Inc()
}

// RecordCompatibilityCheck records a compatibility check result.
func (m *Metrics) RecordCompatibilityCheck(schemaType, level string, compatible bool) {
	result := "compatible"
	if !compatible {
		result = "incompatible"
	}
	m.CompatibilityChecks.WithLabelValues(schemaType, level, result).Inc()
}

// RecordStorageOperation records a storage operation.
func (m *Metrics) RecordStorageOperation(backend, operation string, duration time.Duration, err error) {
	m.StorageOperations.WithLabelValues(backend, operation).Inc()
	m.StorageLatency.WithLabelValues(backend, operation).Observe(duration.Seconds())
	if err != nil {
		m.StorageErrors.WithLabelValues(backend, operation).Inc()
	}
}

// RecordCacheAccess records a cache access.
func (m *Metrics) RecordCacheAccess(cache string, hit bool) {
	if hit {
		m.CacheHits.WithLabelValues(cache).Inc()
	} else {
		m.CacheMisses.WithLabelValues(cache).Inc()
	}
}

// RecordAuthAttempt records an authentication attempt.
func (m *Metrics) RecordAuthAttempt(method string, success bool, reason string, duration time.Duration) {
	m.AuthAttempts.WithLabelValues(method).Inc()
	m.AuthLatency.WithLabelValues(method).Observe(duration.Seconds())
	if !success {
		m.AuthFailures.WithLabelValues(method, reason).Inc()
	}
}

// RecordRateLimitHit records a rate limit hit.
func (m *Metrics) RecordRateLimitHit(client string) {
	m.RateLimitHits.WithLabelValues(client).Inc()
}

// UpdateSchemaCount updates the schema count for a type.
func (m *Metrics) UpdateSchemaCount(schemaType string, count float64) {
	m.SchemasTotal.WithLabelValues(schemaType).Set(count)
}

// UpdateSubjectCount updates the subject count.
func (m *Metrics) UpdateSubjectCount(count float64) {
	m.SubjectsTotal.Set(count)
}

// UpdateCacheSize updates the cache size.
func (m *Metrics) UpdateCacheSize(cache string, size float64) {
	m.CacheSize.WithLabelValues(cache).Set(size)
}
