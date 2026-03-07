# Metrics Health Check

Analyze the schema registry's health using Prometheus metrics. Follow these steps:

1. Use `get_metrics_summary` to get an overview of all key metrics.
2. Check the Confluent-compatible counters:
   - `kafka_schema_registry_api_failure_count` vs `api_success_count` -- a high failure ratio indicates problems.
   - `kafka_schema_registry_master_slave_role` should be 1 (leader). 0 means the node is not serving requests.
3. Check AxonOps-native metrics:
   - `schema_registry_requests_in_flight` -- sustained high values indicate backpressure.
   - `schema_registry_storage_errors_total` -- any non-zero rate indicates storage backend issues.
   - `schema_registry_auth_failures_total` -- high rates may indicate brute-force attempts.
4. Use `query_metric` with name `error` to find all error-related metrics at once.
5. Summarize findings as: healthy, degraded, or unhealthy with specific reasons.
