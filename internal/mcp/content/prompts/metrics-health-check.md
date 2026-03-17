# Metrics Health Check

Analyze the schema registry's health using metrics. Follow these steps:

1. Use `get_metrics_summary` to get an overview of all key metrics across every category.

2. Check wire-compatible counters:
   - `kafka_schema_registry_api_failure_count` vs `kafka_schema_registry_api_success_count` -- a high failure ratio indicates problems.
   - `kafka_schema_registry_master_slave_role` should be 1 (leader). 0 means the node is not serving writes.

3. Check request metrics:
   - `schema_registry_requests_in_flight` -- sustained high values indicate backpressure.
   - `schema_registry_request_duration_seconds` -- rising p99 latency indicates performance degradation.

4. Check storage health:
   - `schema_registry_storage_errors_total` -- any non-zero rate indicates storage backend issues.
   - `schema_registry_storage_latency_seconds` -- rising latency indicates database problems.

5. Check cache performance:
   - Compare `schema_registry_cache_hits_total` vs `schema_registry_cache_misses_total` -- low hit ratio means the cache is not effective.

6. Check compatibility metrics:
   - `schema_registry_compatibility_errors_total` -- internal compatibility check failures.
   - `schema_registry_compatibility_checks_total` with `result="incompatible"` -- high rates mean clients are submitting breaking schemas.

7. Check authentication:
   - `schema_registry_auth_failures_total` -- high rates may indicate brute-force attempts or misconfigured clients.

8. Check MCP operations (if MCP is enabled):
   - `schema_registry_mcp_tool_call_errors_total` -- tools returning errors.
   - `schema_registry_mcp_permission_denied_total` -- blocked tool calls.

9. Use `get_metrics_by_category` with category `storage` or `auth` to drill into specific areas.

10. Use `query_metric` with name `error` to find all error-related metrics at once.

11. Summarize findings as: **healthy**, **degraded**, or **unhealthy** with specific reasons and recommended actions.
