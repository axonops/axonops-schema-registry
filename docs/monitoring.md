# Monitoring

This guide covers the observability features of AxonOps Schema Registry, including health checks, Prometheus metrics, structured logging, and Grafana dashboard setup.

## Contents

- [Overview](#overview)
- [Health Check](#health-check)
- [Prometheus Metrics](#prometheus-metrics)
  - [Request Metrics](#request-metrics)
  - [Schema Metrics](#schema-metrics)
  - [Compatibility Metrics](#compatibility-metrics)
  - [Storage Metrics](#storage-metrics)
  - [Cache Metrics](#cache-metrics)
  - [Auth Metrics](#auth-metrics)
  - [Rate Limit Metrics](#rate-limit-metrics)
  - [Runtime Metrics](#runtime-metrics)
  - [Path Normalization](#path-normalization)
- [Prometheus Scrape Configuration](#prometheus-scrape-configuration)
- [Recommended Alerts](#recommended-alerts)
- [Logging](#logging)
  - [Log Format](#log-format)
  - [Log Levels](#log-levels)
  - [Configuration](#configuration)
  - [Request Logging](#request-logging)
- [Grafana Dashboard](#grafana-dashboard)
- [Server Metadata](#server-metadata)

---

## Overview

The registry exposes Prometheus metrics, structured logging, and health check endpoints for comprehensive observability. All monitoring endpoints are unauthenticated, making them suitable for external probes and scrape targets without credential management.

## Health Check

`GET /` returns an empty JSON object with HTTP 200 when the service is healthy:

```bash
curl -s http://localhost:8081/ | jq .
```

```json
{}
```

This endpoint verifies database connectivity and is suitable for load balancer health probes and container orchestrator liveness checks (Kubernetes `livenessProbe`, ECS health check, etc.).

Example Kubernetes probe configuration:

```yaml
livenessProbe:
  httpGet:
    path: /
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Prometheus Metrics

Metrics are exposed at `GET /metrics` in Prometheus/OpenMetrics format. This endpoint is unauthenticated and excluded from request metric recording to avoid self-referential inflation.

```bash
curl -s http://localhost:8081/metrics
```

### Request Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `schema_registry_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests |
| `schema_registry_request_duration_seconds` | Histogram | `method`, `path` | Request latency in seconds |
| `schema_registry_requests_in_flight` | Gauge | -- | Number of requests currently being processed |

### Schema Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `schema_registry_schemas_total` | Gauge | `type` | Total schemas by type (AVRO, PROTOBUF, JSON) |
| `schema_registry_subjects_total` | Gauge | -- | Total number of subjects |
| `schema_registry_schema_versions` | Gauge | `subject` | Number of versions per subject |
| `schema_registry_registrations_total` | Counter | `type`, `status` | Schema registration attempts (`success` or `failure`) |

### Compatibility Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `schema_registry_compatibility_checks_total` | Counter | `type`, `level`, `result` | Compatibility checks performed (`compatible` or `incompatible`) |
| `schema_registry_compatibility_errors_total` | Counter | `type`, `level` | Compatibility check errors |

### Storage Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `schema_registry_storage_operations_total` | Counter | `backend`, `operation` | Total storage operations |
| `schema_registry_storage_latency_seconds` | Histogram | `backend`, `operation` | Storage operation latency in seconds |
| `schema_registry_storage_errors_total` | Counter | `backend`, `operation` | Storage operation errors |

### Cache Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `schema_registry_cache_hits_total` | Counter | `cache` | Cache hits |
| `schema_registry_cache_misses_total` | Counter | `cache` | Cache misses |
| `schema_registry_cache_size` | Gauge | `cache` | Current number of entries in cache |

### Auth Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `schema_registry_auth_attempts_total` | Counter | `method` | Authentication attempts |
| `schema_registry_auth_failures_total` | Counter | `method`, `reason` | Authentication failures |
| `schema_registry_auth_latency_seconds` | Histogram | `method` | Authentication latency in seconds |

### Rate Limit Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `schema_registry_rate_limit_hits_total` | Counter | `client` | Requests rejected by rate limiting |

### Runtime Metrics

Go runtime and process metrics are automatically included via the standard Prometheus collectors:

- `go_goroutines` -- number of active goroutines
- `go_gc_duration_seconds` -- GC pause duration
- `go_memstats_alloc_bytes` -- allocated heap memory
- `process_cpu_seconds_total` -- CPU time consumed
- `process_open_fds` -- open file descriptors
- `process_resident_memory_bytes` -- resident set size

### Path Normalization

Path labels in request metrics are normalized to prevent cardinality explosion. Dynamic path segments are replaced with placeholders:

| Actual Path | Normalized Label |
|-------------|-----------------|
| `/subjects/users-value/versions/1` | `/subjects/{subject}/versions/{version}` |
| `/subjects/users-value/versions` | `/subjects/{subject}/versions` |
| `/subjects/users-value` | `/subjects/{subject}` |
| `/schemas/ids/42` | `/schemas/ids/{id}` |
| `/config/users-value` | `/config/{subject}` |
| `/mode/users-value` | `/mode/{subject}` |
| `/compatibility/subjects/users-value/versions/latest` | `/compatibility/subjects/{subject}/versions/{version}` |

Static paths (`/`, `/metrics`, `/schemas/types`) are recorded as-is.

## Prometheus Scrape Configuration

Add the registry as a scrape target in your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'schema-registry'
    static_configs:
      - targets: ['schema-registry:8081']
    metrics_path: /metrics
    scrape_interval: 15s
```

For multiple instances behind a load balancer, target each instance directly so that per-instance metrics are preserved:

```yaml
scrape_configs:
  - job_name: 'schema-registry'
    static_configs:
      - targets:
          - 'schema-registry-1:8081'
          - 'schema-registry-2:8081'
          - 'schema-registry-3:8081'
    metrics_path: /metrics
    scrape_interval: 15s
```

If you use Kubernetes service discovery:

```yaml
scrape_configs:
  - job_name: 'schema-registry'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        regex: schema-registry
        action: keep
      - source_labels: [__meta_kubernetes_pod_ip]
        target_label: __address__
        replacement: '$1:8081'
```

## Recommended Alerts

The following Prometheus alerting rules cover the most critical failure modes. Adjust thresholds to match your traffic patterns and SLOs.

```yaml
groups:
  - name: schema-registry
    rules:
      - alert: SchemaRegistryHighErrorRate
        expr: rate(schema_registry_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Schema registry error rate is elevated"
          description: "More than 0.1 server errors per second over the last 5 minutes."

      - alert: SchemaRegistryHighLatency
        expr: histogram_quantile(0.99, rate(schema_registry_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Schema registry p99 latency exceeds 1 second"
          description: "The 99th percentile request latency has been above 1s for 5 minutes."

      - alert: SchemaRegistryStorageErrors
        expr: rate(schema_registry_storage_errors_total[5m]) > 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Schema registry storage backend is producing errors"
          description: "Storage errors detected for backend {{ $labels.backend }}, operation {{ $labels.operation }}."

      - alert: SchemaRegistryAuthFailures
        expr: rate(schema_registry_auth_failures_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High rate of authentication failures"
          description: "More than 10 auth failures per second, possibly indicating a brute-force attempt."

      - alert: SchemaRegistryRateLimiting
        expr: rate(schema_registry_rate_limit_hits_total[5m]) > 0
        for: 5m
        labels:
          severity: info
        annotations:
          summary: "Rate limiting is actively rejecting requests"
          description: "Client {{ $labels.client }} is being rate limited."

      - alert: SchemaRegistryDown
        expr: up{job="schema-registry"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Schema registry instance is down"
          description: "Prometheus cannot reach the schema registry metrics endpoint."
```

## Logging

The registry uses Go's `log/slog` package for structured logging. All log output is written to stdout, making it compatible with container log aggregation systems (Fluentd, Fluent Bit, Loki, CloudWatch Logs, etc.).

### Log Format

In JSON format (the default), each log entry is a single line:

```json
{"time":"2024-01-15T10:30:00.000Z","level":"INFO","msg":"schema registered","subject":"users-value","version":1,"id":42}
```

In text format:

```
time=2024-01-15T10:30:00.000Z level=INFO msg="schema registered" subject=users-value version=1 id=42
```

### Log Levels

| Level | Usage |
|-------|-------|
| `debug` | Detailed internal operations, useful for development and troubleshooting |
| `info` | Normal operational events: startup, schema registration, configuration changes |
| `warn` | Recoverable issues: deprecated usage, fallback behavior |
| `error` | Failures: storage errors, invalid requests, unrecoverable conditions |

### Configuration

Set the log level and format in the configuration file:

```yaml
logging:
  level: info
  format: json
```

Or override via environment variable:

```bash
export SCHEMA_REGISTRY_LOG_LEVEL=debug
```

### Request Logging

Every HTTP request is logged at `INFO` level with the following fields:

| Field | Description |
|-------|-------------|
| `method` | HTTP method (GET, POST, PUT, DELETE) |
| `path` | Request path |
| `status` | HTTP response status code |
| `duration` | Request processing time |
| `remote` | Client IP address |

Example:

```json
{"time":"2024-01-15T10:30:01.123Z","level":"INFO","msg":"request","method":"POST","path":"/subjects/users-value/versions","status":200,"duration":"2.45ms","remote":"10.0.0.5:43210"}
```

## Grafana Dashboard

A Grafana dashboard for the schema registry should include the following panels. All queries assume the Prometheus job name is `schema-registry`.

**Request Rate and Latency**

- Request rate by endpoint: `sum(rate(schema_registry_requests_total[5m])) by (method, path)`
- Request latency heatmap: `schema_registry_request_duration_seconds_bucket`
- p50/p95/p99 latency: `histogram_quantile(0.99, sum(rate(schema_registry_request_duration_seconds_bucket[5m])) by (le))`
- Requests in flight: `schema_registry_requests_in_flight`

**Error Rate**

- Error rate by status: `sum(rate(schema_registry_requests_total{status=~"[45].."}[5m])) by (status)`
- Error ratio: `sum(rate(schema_registry_requests_total{status=~"5.."}[5m])) / sum(rate(schema_registry_requests_total[5m]))`

**Schema Registration**

- Registration rate: `sum(rate(schema_registry_registrations_total[5m])) by (type, status)`
- Failure ratio: `sum(rate(schema_registry_registrations_total{status="failure"}[5m])) / sum(rate(schema_registry_registrations_total[5m]))`

**Active Subjects and Schemas**

- Total subjects: `schema_registry_subjects_total`
- Total schemas by type: `schema_registry_schemas_total`
- Versions per subject (top 10): `topk(10, schema_registry_schema_versions)`

**Compatibility Checks**

- Check rate: `sum(rate(schema_registry_compatibility_checks_total[5m])) by (result)`
- Incompatible ratio: `sum(rate(schema_registry_compatibility_checks_total{result="incompatible"}[5m])) / sum(rate(schema_registry_compatibility_checks_total[5m]))`

**Storage Latency**

- Operation latency by backend: `histogram_quantile(0.99, sum(rate(schema_registry_storage_latency_seconds_bucket[5m])) by (backend, operation, le))`
- Operation rate: `sum(rate(schema_registry_storage_operations_total[5m])) by (backend, operation)`
- Storage error rate: `sum(rate(schema_registry_storage_errors_total[5m])) by (backend, operation)`

**Cache Hit Rate**

- Hit rate: `sum(rate(schema_registry_cache_hits_total[5m])) / (sum(rate(schema_registry_cache_hits_total[5m])) + sum(rate(schema_registry_cache_misses_total[5m])))`
- Cache size: `schema_registry_cache_size`

**Authentication**

- Auth attempt rate: `sum(rate(schema_registry_auth_attempts_total[5m])) by (method)`
- Auth failure rate: `sum(rate(schema_registry_auth_failures_total[5m])) by (method, reason)`
- Auth latency: `histogram_quantile(0.99, sum(rate(schema_registry_auth_latency_seconds_bucket[5m])) by (method, le))`

**Runtime**

- Goroutines: `go_goroutines{job="schema-registry"}`
- Memory usage: `process_resident_memory_bytes{job="schema-registry"}`
- GC pause duration: `go_gc_duration_seconds{job="schema-registry"}`

## Server Metadata

Two additional endpoints provide server identification and version information. These are useful for verifying deployments and debugging multi-instance environments.

**Cluster ID**

```bash
curl -s http://localhost:8081/v1/metadata/id | jq .
```

```json
{
  "id": "default-cluster"
}
```

**Server Version**

```bash
curl -s http://localhost:8081/v1/metadata/version | jq .
```

```json
{
  "version": "1.0.0",
  "commit": "abc123",
  "build_time": "2024-01-15T10:00:00Z"
}
```

The version and commit values are set at build time via linker flags. See [Configuration](configuration.md) for build details.

---

See also: [Deployment](deployment.md) | [Configuration](configuration.md) | [Troubleshooting](troubleshooting.md)
