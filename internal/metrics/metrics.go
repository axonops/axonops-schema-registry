// Package metrics provides Prometheus metrics for the schema registry.
package metrics

import (
	"net/http"
	"strconv"
	"strings"
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
	SchemasTotal       *prometheus.GaugeVec
	SubjectsTotal      prometheus.Gauge
	SchemaVersions     *prometheus.GaugeVec
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

	// MCP metrics
	MCPToolCallsTotal        *prometheus.CounterVec
	MCPToolCallDuration      *prometheus.HistogramVec
	MCPToolCallErrors        *prometheus.CounterVec
	MCPToolCallsActive       prometheus.Gauge
	MCPConfirmationsTotal    *prometheus.CounterVec // labels: outcome (token_issued, confirmed, token_rejected)
	MCPPolicyDenialsTotal    *prometheus.CounterVec // labels: reason (origin_rejected, confirmation_required)
	MCPPermissionDeniedTotal *prometheus.CounterVec // labels: tool, scope

	// Audit output metrics
	AuditEventsTotal        *prometheus.CounterVec   // labels: output, status
	AuditOutputErrorsTotal  *prometheus.CounterVec   // labels: output
	AuditWebhookDroppedTotal prometheus.Counter
	AuditWebhookBatchSize   prometheus.Histogram
	AuditWebhookFlushDuration prometheus.Histogram

	// Per-principal metrics (optional, may be nil if disabled)
	PrincipalRequestsTotal *prometheus.CounterVec // labels: principal, method, path, status
	PrincipalMCPCallsTotal *prometheus.CounterVec // labels: principal, tool, status

	// Confluent-compatible metrics (kafka_schema_registry_* prefix for dashboard compatibility)
	ConfluentRegisteredCount  prometheus.Counter       // kafka_schema_registry_registered_count
	ConfluentDeletedCount     prometheus.Counter       // kafka_schema_registry_deleted_count
	ConfluentAPISuccessCount  prometheus.Counter       // kafka_schema_registry_api_success_count
	ConfluentAPIFailureCount  prometheus.Counter       // kafka_schema_registry_api_failure_count
	ConfluentSchemasCreated   *prometheus.CounterVec   // kafka_schema_registry_schemas_created{schema_type}
	ConfluentSchemasDeleted   *prometheus.CounterVec   // kafka_schema_registry_schemas_deleted{schema_type}
	ConfluentMasterSlaveRole  prometheus.Gauge         // kafka_schema_registry_master_slave_role (always 1.0 for standalone)
	ConfluentNodeCount        prometheus.Gauge         // kafka_schema_registry_node_count (always 1 for standalone)
	ConfluentEndpointRequests *prometheus.CounterVec   // kafka_schema_registry_jersey_metrics_request_total{endpoint}
	ConfluentEndpointLatency  *prometheus.HistogramVec // kafka_schema_registry_jersey_metrics_request_latency_seconds{endpoint}
	ConfluentEndpointErrors   *prometheus.CounterVec   // kafka_schema_registry_jersey_metrics_request_error_total{endpoint}

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

	// MCP metrics
	m.MCPToolCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_mcp_tool_calls_total",
			Help: "Total number of MCP tool invocations",
		},
		[]string{"tool", "status"},
	)

	m.MCPToolCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "schema_registry_mcp_tool_call_duration_seconds",
			Help:    "MCP tool call latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tool"},
	)

	m.MCPToolCallErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_mcp_tool_call_errors_total",
			Help: "Total number of MCP tool calls that returned errors",
		},
		[]string{"tool"},
	)

	m.MCPToolCallsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "schema_registry_mcp_tool_calls_active",
			Help: "Number of MCP tool calls currently being processed",
		},
	)

	m.MCPConfirmationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_mcp_confirmations_total",
			Help: "Total number of MCP two-phase confirmation events",
		},
		[]string{"outcome"},
	)

	m.MCPPolicyDenialsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_mcp_policy_denials_total",
			Help: "Total number of MCP policy denial events",
		},
		[]string{"reason"},
	)

	m.MCPPermissionDeniedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_mcp_permission_denied_total",
			Help: "Total number of MCP tool calls blocked by permission scopes",
		},
		[]string{"tool", "scope"},
	)

	// Audit output metrics
	m.AuditEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_audit_events_total",
			Help: "Total number of audit events written per output and status",
		},
		[]string{"output", "status"},
	)

	m.AuditOutputErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_audit_output_errors_total",
			Help: "Total number of audit output write errors per output",
		},
		[]string{"output"},
	)

	m.AuditWebhookDroppedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "schema_registry_audit_webhook_dropped_total",
			Help: "Total number of audit events dropped due to webhook buffer overflow",
		},
	)

	m.AuditWebhookBatchSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "schema_registry_audit_webhook_batch_size",
			Help:    "Distribution of webhook batch sizes (number of events per flush)",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
	)

	m.AuditWebhookFlushDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "schema_registry_audit_webhook_flush_duration_seconds",
			Help:    "Time taken to flush webhook batches to the HTTP endpoint",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Confluent-compatible metrics (kafka_schema_registry_* prefix)
	// These mirror Confluent Schema Registry JMX metrics so that existing
	// Grafana dashboards and Prometheus alerts continue to work.
	m.ConfluentRegisteredCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_schema_registry_registered_count",
			Help: "Total number of schemas registered (Confluent-compatible)",
		},
	)

	m.ConfluentDeletedCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_schema_registry_deleted_count",
			Help: "Total number of schemas deleted (Confluent-compatible)",
		},
	)

	m.ConfluentAPISuccessCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_schema_registry_api_success_count",
			Help: "Total number of successful API calls (Confluent-compatible)",
		},
	)

	m.ConfluentAPIFailureCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_schema_registry_api_failure_count",
			Help: "Total number of failed API calls (Confluent-compatible)",
		},
	)

	m.ConfluentSchemasCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_schema_registry_schemas_created",
			Help: "Total number of schemas created by type (Confluent-compatible)",
		},
		[]string{"schema_type"},
	)

	m.ConfluentSchemasDeleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_schema_registry_schemas_deleted",
			Help: "Total number of schemas deleted by type (Confluent-compatible)",
		},
		[]string{"schema_type"},
	)

	m.ConfluentMasterSlaveRole = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kafka_schema_registry_master_slave_role",
			Help: "1.0 if this node is the active leader, 0.0 if follower (Confluent-compatible). Always 1.0 for standalone deployments.",
		},
	)
	m.ConfluentMasterSlaveRole.Set(1.0)

	m.ConfluentNodeCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kafka_schema_registry_node_count",
			Help: "Number of schema registry nodes in the cluster (Confluent-compatible). Always 1 for standalone deployments.",
		},
	)
	m.ConfluentNodeCount.Set(1)

	// Per-endpoint Confluent-compatible metrics (jersey_metrics_*)
	m.ConfluentEndpointRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_schema_registry_jersey_metrics_request_total",
			Help: "Total number of requests per endpoint (Confluent-compatible)",
		},
		[]string{"endpoint"},
	)

	m.ConfluentEndpointLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_schema_registry_jersey_metrics_request_latency_seconds",
			Help:    "Request latency per endpoint in seconds (Confluent-compatible)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	m.ConfluentEndpointErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_schema_registry_jersey_metrics_request_error_total",
			Help: "Total number of request errors per endpoint (Confluent-compatible)",
		},
		[]string{"endpoint"},
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
		m.MCPToolCallsTotal,
		m.MCPToolCallDuration,
		m.MCPToolCallErrors,
		m.MCPToolCallsActive,
		m.MCPConfirmationsTotal,
		m.MCPPolicyDenialsTotal,
		m.MCPPermissionDeniedTotal,
		m.AuditEventsTotal,
		m.AuditOutputErrorsTotal,
		m.AuditWebhookDroppedTotal,
		m.AuditWebhookBatchSize,
		m.AuditWebhookFlushDuration,
		m.ConfluentRegisteredCount,
		m.ConfluentDeletedCount,
		m.ConfluentAPISuccessCount,
		m.ConfluentAPIFailureCount,
		m.ConfluentSchemasCreated,
		m.ConfluentSchemasDeleted,
		m.ConfluentMasterSlaveRole,
		m.ConfluentNodeCount,
		m.ConfluentEndpointRequests,
		m.ConfluentEndpointLatency,
		m.ConfluentEndpointErrors,
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

		statusStr := strconv.Itoa(wrapped.statusCode)
		m.RequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		m.RequestDuration.WithLabelValues(r.Method, path).Observe(duration)

		// Confluent-compatible API call counters
		if wrapped.statusCode >= 200 && wrapped.statusCode < 400 {
			m.ConfluentAPISuccessCount.Inc()
		} else {
			m.ConfluentAPIFailureCount.Inc()
		}

		// Per-endpoint Confluent-compatible metrics
		if endpoint := confluentEndpoint(r.Method, path); endpoint != "" {
			m.ConfluentEndpointRequests.WithLabelValues(endpoint).Inc()
			m.ConfluentEndpointLatency.WithLabelValues(endpoint).Observe(duration)
			if wrapped.statusCode >= 400 {
				m.ConfluentEndpointErrors.WithLabelValues(endpoint).Inc()
			}
		}
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
	// Strip /contexts/{context} prefix and normalize the inner path,
	// then re-add the normalized prefix to avoid metric label explosion.
	contextPrefix := ""
	innerPath := path
	if startsWith(path, "/contexts/") {
		rest := path[len("/contexts/"):]
		idx := 0
		for idx < len(rest) && rest[idx] != '/' {
			idx++
		}
		contextPrefix = "/contexts/{context}"
		if idx < len(rest) {
			innerPath = rest[idx:]
		} else {
			return contextPrefix
		}
	}

	// Handle common schema registry paths
	var normalized string
	switch {
	case startsWith(innerPath, "/subjects/") && contains(innerPath, "/versions/"):
		normalized = "/subjects/{subject}/versions/{version}"
	case startsWith(innerPath, "/subjects/") && endsWith(innerPath, "/versions"):
		normalized = "/subjects/{subject}/versions"
	case startsWith(innerPath, "/subjects/"):
		normalized = "/subjects/{subject}"
	case startsWith(innerPath, "/schemas/ids/"):
		normalized = "/schemas/ids/{id}"
	case startsWith(innerPath, "/config/"):
		normalized = "/config/{subject}"
	case startsWith(innerPath, "/mode/"):
		normalized = "/mode/{subject}"
	case startsWith(innerPath, "/compatibility/subjects/"):
		normalized = "/compatibility/subjects/{subject}/versions/{version}"
	default:
		normalized = innerPath
	}

	return contextPrefix + normalized
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

// confluentEndpoint maps a normalized HTTP method+path to Confluent's @PerformanceMetric
// endpoint names. Returns "" for paths that have no Confluent equivalent.
func confluentEndpoint(method, path string) string {
	// Strip context prefix if present
	if startsWith(path, "/contexts/{context}") {
		path = path[len("/contexts/{context}"):]
	}

	switch {
	// Schema operations
	case method == "GET" && path == "/schemas":
		return "schemas.get-schemas"
	case method == "GET" && path == "/schemas/types":
		return "schemas.get-types"
	case method == "GET" && path == "/schemas/ids/{id}" && !contains(path, "/subjects") && !contains(path, "/versions"):
		return "schemas.ids.get-schema"
	case method == "GET" && startsWith(path, "/schemas/ids/{id}"):
		return "schemas.ids.get-schema"

	// Subject operations
	case method == "GET" && path == "/subjects":
		return "subjects.list"
	case method == "POST" && path == "/subjects/{subject}" && !contains(path, "/versions"):
		return "subjects.get-schema"
	case method == "DELETE" && path == "/subjects/{subject}" && !contains(path, "/versions"):
		return "subjects.delete-subject"

	// Subject version operations
	case method == "POST" && path == "/subjects/{subject}/versions":
		return "subjects.versions.register"
	case method == "GET" && path == "/subjects/{subject}/versions" && !contains(path, "{version}"):
		return "subjects.versions.list"
	case method == "GET" && path == "/subjects/{subject}/versions/{version}":
		return "subjects.versions.get-schema"
	case method == "DELETE" && path == "/subjects/{subject}/versions/{version}":
		return "subjects.versions.deleteSchemaVersion-schema"

	// Compatibility
	case method == "POST" && startsWith(path, "/compatibility/"):
		return "compatibility.subjects.versions.verify"

	// Config
	case method == "GET" && path == "/config":
		return "config.get-global"
	case method == "PUT" && path == "/config":
		return "config.update-global"
	case method == "DELETE" && path == "/config":
		return "config.delete-global"
	case method == "GET" && path == "/config/{subject}":
		return "config.get-subject"
	case method == "PUT" && path == "/config/{subject}":
		return "config.update-subject"
	case method == "DELETE" && path == "/config/{subject}":
		return "config.delete-subject"

	// Mode
	case method == "GET" && path == "/mode":
		return "mode.get-global"
	case method == "PUT" && path == "/mode":
		return "mode.update-global"
	case method == "DELETE" && path == "/mode":
		return "mode.delete-global"
	case method == "GET" && path == "/mode/{subject}":
		return "mode.get-subject"
	case method == "PUT" && path == "/mode/{subject}":
		return "mode.update-subject"
	case method == "DELETE" && path == "/mode/{subject}":
		return "mode.delete-subject"

	// Contexts
	case method == "GET" && path == "/contexts":
		return "contexts.list"

	default:
		return ""
	}
}

// RecordSchemaRegistration records a schema registration attempt.
func (m *Metrics) RecordSchemaRegistration(schemaType string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.RegistrationsTotal.WithLabelValues(schemaType, status).Inc()

	// Confluent-compatible counters
	if success {
		m.ConfluentRegisteredCount.Inc()
		m.ConfluentSchemasCreated.WithLabelValues(confluentSchemaType(schemaType)).Inc()
	}
}

// RecordSchemaDeletion records a schema or subject deletion.
func (m *Metrics) RecordSchemaDeletion(schemaType string) {
	m.ConfluentDeletedCount.Inc()
	m.ConfluentSchemasDeleted.WithLabelValues(confluentSchemaType(schemaType)).Inc()
}

// confluentSchemaType converts our schema type strings to Confluent's lowercase format.
func confluentSchemaType(schemaType string) string {
	switch schemaType {
	case "AVRO":
		return "avro"
	case "JSON":
		return "json"
	case "PROTOBUF":
		return "protobuf"
	default:
		return strings.ToLower(schemaType)
	}
}

// RecordCompatibilityCheck records a compatibility check result.
func (m *Metrics) RecordCompatibilityCheck(schemaType, level string, compatible bool) {
	result := "compatible"
	if !compatible {
		result = "incompatible"
	}
	m.CompatibilityChecks.WithLabelValues(schemaType, level, result).Inc()
}

// RecordCompatibilityError records an internal error during a compatibility check.
// This is distinct from an "incompatible" result — it means the check itself failed.
func (m *Metrics) RecordCompatibilityError(schemaType, level string) {
	m.CompatibilityErrors.WithLabelValues(schemaType, level).Inc()
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

// EnablePrincipalMetrics registers per-principal metric collectors.
// Call this when per_principal_metrics is enabled in config.
func (m *Metrics) EnablePrincipalMetrics() {
	m.PrincipalRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_principal_requests_total",
			Help: "Total HTTP requests per authenticated principal",
		},
		[]string{"principal", "method", "path", "status"},
	)

	m.PrincipalMCPCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "schema_registry_principal_mcp_calls_total",
			Help: "Total MCP tool calls per authenticated principal",
		},
		[]string{"principal", "tool", "status"},
	)

	m.registry.MustRegister(m.PrincipalRequestsTotal, m.PrincipalMCPCallsTotal)
}

// RecordPrincipalRequest records an HTTP request for a specific principal.
func (m *Metrics) RecordPrincipalRequest(principal, method, path, status string) {
	if m.PrincipalRequestsTotal != nil {
		m.PrincipalRequestsTotal.WithLabelValues(principal, method, path, status).Inc()
	}
}

// RecordPrincipalMCPCall records an MCP tool call for a specific principal.
func (m *Metrics) RecordPrincipalMCPCall(principal, tool, status string) {
	if m.PrincipalMCPCallsTotal != nil {
		m.PrincipalMCPCallsTotal.WithLabelValues(principal, tool, status).Inc()
	}
}

// RecordMCPConfirmation records an MCP two-phase confirmation event.
// Outcome values: "token_issued", "confirmed", "token_rejected".
func (m *Metrics) RecordMCPConfirmation(outcome string) {
	m.MCPConfirmationsTotal.WithLabelValues(outcome).Inc()
}

// RecordMCPPolicyDenial records an MCP policy denial event.
// Reason values: "origin_rejected", "confirmation_required".
func (m *Metrics) RecordMCPPolicyDenial(reason string) {
	m.MCPPolicyDenialsTotal.WithLabelValues(reason).Inc()
}

// RecordMCPPermissionDenied records a tool blocked by permission scopes.
func (m *Metrics) RecordMCPPermissionDenied(tool, scope string) {
	m.MCPPermissionDeniedTotal.WithLabelValues(tool, scope).Inc()
}

// RecordMCPToolCall records an MCP tool call.
func (m *Metrics) RecordMCPToolCall(tool, status string, duration time.Duration) {
	m.MCPToolCallsTotal.WithLabelValues(tool, status).Inc()
	m.MCPToolCallDuration.WithLabelValues(tool).Observe(duration.Seconds())
	if status == "error" {
		m.MCPToolCallErrors.WithLabelValues(tool).Inc()
	}
}

// RecordAuditEvent records an audit event write for a given output.
func (m *Metrics) RecordAuditEvent(output, status string) {
	m.AuditEventsTotal.WithLabelValues(output, status).Inc()
}

// RecordAuditOutputError records a write error for a given audit output.
func (m *Metrics) RecordAuditOutputError(output string) {
	m.AuditOutputErrorsTotal.WithLabelValues(output).Inc()
}

// RecordAuditWebhookDrop records an event dropped due to webhook buffer overflow.
func (m *Metrics) RecordAuditWebhookDrop() {
	m.AuditWebhookDroppedTotal.Inc()
}

// RecordAuditWebhookFlush records a webhook batch flush.
func (m *Metrics) RecordAuditWebhookFlush(batchSize int, duration time.Duration) {
	m.AuditWebhookBatchSize.Observe(float64(batchSize))
	m.AuditWebhookFlushDuration.Observe(duration.Seconds())
}

// GaugeSource provides the data needed to periodically refresh gauge metrics.
// This avoids importing the registry or storage packages from the metrics package.
type GaugeSource interface {
	// SubjectCount returns the number of active subjects.
	SubjectCount() (int, error)
	// SchemaCountsByType returns schema counts keyed by type (e.g. "AVRO", "JSON", "PROTOBUF").
	SchemaCountsByType() (map[string]int, error)
}

// StartGaugeRefresh starts a background goroutine that periodically refreshes
// the schemas_total and subjects_total gauge metrics by querying the source.
// The goroutine stops when the stop channel is closed.
func (m *Metrics) StartGaugeRefresh(source GaugeSource, interval time.Duration, stop <-chan struct{}) {
	// Do an immediate refresh before starting the ticker
	m.refreshGauges(source)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.refreshGauges(source)
			case <-stop:
				return
			}
		}
	}()
}

func (m *Metrics) refreshGauges(source GaugeSource) {
	if count, err := source.SubjectCount(); err == nil {
		m.SubjectsTotal.Set(float64(count))
	}
	if counts, err := source.SchemaCountsByType(); err == nil {
		for _, st := range []string{"AVRO", "PROTOBUF", "JSON"} {
			m.SchemasTotal.WithLabelValues(st).Set(float64(counts[st]))
		}
	}
}
