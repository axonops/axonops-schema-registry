# Metrics Reference

Complete reference for all application metrics exposed by AxonOps Schema Registry at `GET /metrics` in OpenMetrics format.

## Request Metrics (3)

- **`schema_registry_requests_total`** (counter, labels: `method`, `path`, `status`) -- Total HTTP requests. Path labels are normalized to prevent cardinality explosion (e.g. `/subjects/{subject}/versions/{version}`). Use `rate()` for requests per second. Filter by `status=~"5.."` for server errors.
- **`schema_registry_request_duration_seconds`** (histogram, labels: `method`, `path`) -- HTTP request latency. Use `histogram_quantile(0.99, rate(..._bucket[5m]))` for p99 latency. Sustained high values indicate backpressure or slow storage.
- **`schema_registry_requests_in_flight`** (gauge) -- Number of requests currently being processed. A sustained high value relative to your thread/goroutine budget indicates the server is at capacity.

## Schema Metrics (4)

- **`schema_registry_schemas_total`** (gauge, labels: `type`) -- Current number of schemas by type (`AVRO`, `PROTOBUF`, `JSON`). Useful for capacity planning and tracking schema growth over time.
- **`schema_registry_subjects_total`** (gauge) -- Current number of subjects. Compare with `schema_registry_schemas_total` to understand the average schema-per-subject ratio.
- **`schema_registry_schema_versions`** (gauge, labels: `subject`) -- Number of versions per subject. High version counts on a single subject may indicate rapid schema evolution or a missing compatibility strategy.
- **`schema_registry_registrations_total`** (counter, labels: `type`, `status`) -- Schema registration attempts by type and outcome (`success` or `failure`). A high failure rate indicates clients are submitting invalid or incompatible schemas.

## Compatibility Metrics (2)

- **`schema_registry_compatibility_checks_total`** (counter, labels: `type`, `level`, `result`) -- Compatibility checks performed. The `result` label is `compatible` or `incompatible`. The `level` label shows which compatibility mode was used (e.g. `BACKWARD`, `FULL_TRANSITIVE`). High incompatible rates may indicate schemas are not following evolution rules.
- **`schema_registry_compatibility_errors_total`** (counter, labels: `type`, `level`) -- Compatibility check errors (distinct from incompatible results). These indicate internal failures during compatibility evaluation, such as unparseable schemas or reference resolution errors.

## Storage Metrics (3)

- **`schema_registry_storage_operations_total`** (counter, labels: `backend`, `operation`) -- Total storage operations by backend (`memory`, `postgres`, `mysql`, `cassandra`) and operation type. Use to understand storage call patterns and detect hotspots.
- **`schema_registry_storage_latency_seconds`** (histogram, labels: `backend`, `operation`) -- Storage operation latency. Use `histogram_quantile()` for percentiles. Rising latency often indicates database contention, network issues, or missing indexes.
- **`schema_registry_storage_errors_total`** (counter, labels: `backend`, `operation`) -- Storage operation errors. Any non-zero rate indicates a storage backend issue that needs investigation. Check database connectivity, disk space, and query logs.

## Cache Metrics (3)

- **`schema_registry_cache_hits_total`** (counter, labels: `cache`) -- Cache hits. Compute hit ratio as `hits / (hits + misses)`. A low hit ratio (below 80%) suggests the cache is too small or the access pattern has poor locality.
- **`schema_registry_cache_misses_total`** (counter, labels: `cache`) -- Cache misses. Each miss results in a storage backend call. High miss rates directly increase storage latency.
- **`schema_registry_cache_size`** (gauge, labels: `cache`) -- Current number of entries in cache. Track over time to understand cache utilization relative to configured limits.

## Auth Metrics (3)

- **`schema_registry_auth_attempts_total`** (counter, labels: `method`) -- Authentication attempts by method (`basic`, `api_key`, `jwt`, `oidc`, `ldap`, `mtls`). Use to understand which auth methods are active and their relative usage.
- **`schema_registry_auth_failures_total`** (counter, labels: `method`, `reason`) -- Authentication failures by method and reason (e.g. `invalid_credentials`, `expired_token`, `disabled_user`). A sudden spike may indicate a brute-force attempt or a misconfigured client.
- **`schema_registry_auth_latency_seconds`** (histogram, labels: `method`) -- Authentication latency by method. External auth methods (LDAP, OIDC) typically have higher latency than local methods (basic, API key). Rising latency may indicate upstream IdP issues.

## Rate Limit Metrics (1)

- **`schema_registry_rate_limit_hits_total`** (counter, labels: `client`) -- Requests rejected by rate limiting. A non-zero value means at least one client is exceeding configured rate limits. Check which client IP is affected via the `client` label.

## MCP Metrics (7)

- **`schema_registry_mcp_tool_calls_total`** (counter, labels: `tool`, `status`) -- Total MCP tool invocations by tool name and outcome (`success` or `error`). Use to understand which tools are most used and which have high error rates.
- **`schema_registry_mcp_tool_call_duration_seconds`** (histogram, labels: `tool`) -- MCP tool call latency by tool name. Tools that call the registry (e.g. `register_schema`) will be slower than read-only tools.
- **`schema_registry_mcp_tool_call_errors_total`** (counter, labels: `tool`) -- MCP tool calls that returned errors. Subset of `mcp_tool_calls_total` where `status="error"`. Use to identify tools that fail frequently.
- **`schema_registry_mcp_tool_calls_active`** (gauge) -- Number of MCP tool calls currently in progress. Sustained high values indicate slow tool execution or high concurrency.
- **`schema_registry_mcp_confirmations_total`** (counter, labels: `outcome`) -- Two-phase confirmation events. Outcomes: `token_issued` (write operation requested), `confirmed` (user approved), `token_rejected` (user denied or token expired).
- **`schema_registry_mcp_policy_denials_total`** (counter, labels: `reason`) -- Policy denial events. Reasons: `origin_rejected` (untrusted origin), `confirmation_required` (write without confirmation token).
- **`schema_registry_mcp_permission_denied_total`** (counter, labels: `tool`, `scope`) -- Tool calls blocked by permission scopes. The `scope` label shows which permission was missing (e.g. `schema_write`, `config_read`).

## Per-Principal Metrics (2, optional)

These metrics are registered only when `security.per_principal_metrics: true` is enabled.

- **`schema_registry_principal_requests_total`** (counter, labels: `principal`, `method`, `path`, `status`) -- HTTP requests per authenticated principal. Use to audit per-user activity and detect anomalous usage patterns. **Cardinality warning**: creates a time series per unique principal.
- **`schema_registry_principal_mcp_calls_total`** (counter, labels: `principal`, `tool`, `status`) -- MCP tool calls per authenticated principal. Use to audit which users are calling which tools. **Cardinality warning**: creates a time series per unique principal-tool combination.

## Wire-Compatible Metrics (8)

These metrics use the `kafka_schema_registry_` prefix, matching the metric names produced by the JMX exporter after conversion. Existing Grafana dashboards and alerting rules that query these metric names work without modification.

- **`kafka_schema_registry_registered_count`** (counter) -- Total schemas registered. Equivalent to the `registered-count` JMX MBean.
- **`kafka_schema_registry_deleted_count`** (counter) -- Total schemas deleted. Equivalent to the `deleted-count` JMX MBean.
- **`kafka_schema_registry_api_success_count`** (counter) -- Successful API calls (HTTP 2xx/3xx). Equivalent to the `api-success-count` JMX MBean.
- **`kafka_schema_registry_api_failure_count`** (counter) -- Failed API calls (HTTP 4xx/5xx). Equivalent to the `api-failure-count` JMX MBean.
- **`kafka_schema_registry_schemas_created`** (counter, labels: `schema_type`) -- Schemas created by type (`avro`, `json`, `protobuf`). Equivalent to the per-type `*-schemas-created` JMX MBeans.
- **`kafka_schema_registry_schemas_deleted`** (counter, labels: `schema_type`) -- Schemas deleted by type. Equivalent to the per-type `*-schemas-deleted` JMX MBeans.
- **`kafka_schema_registry_master_slave_role`** (gauge) -- 1.0 = leader, 0.0 = follower. Always 1.0 for standalone deployments. Equivalent to the `master-slave-role` JMX MBean.
- **`kafka_schema_registry_node_count`** (gauge) -- Cluster node count. Always 1 for standalone deployments. Equivalent to the `node-count` JMX MBean.

## Metric Types

- **Counter** -- monotonically increasing value (e.g. total requests). Use `rate()` in PromQL for per-second rates.
- **Gauge** -- value that goes up and down (e.g. in-flight requests, schema counts).
- **Histogram** -- samples observations into configurable buckets (e.g. latency). Use `histogram_quantile()` for percentiles.

## Label Cardinality

Path labels in request metrics are **normalized** to prevent cardinality explosion. Dynamic segments like subject names and schema IDs are replaced with `{subject}`, `{id}`, etc.

## MCP Tools for Metrics

- **`get_metrics_summary`** -- health-oriented overview of all categories
- **`get_metrics_by_category`** -- query a specific category (request, schema, compatibility, storage, cache, auth, rate_limit, mcp, principal, wire_compatible, runtime)
- **`query_metric`** -- search for a specific metric by name or partial match
- **`list_metrics`** -- list all available metric names grouped by category
