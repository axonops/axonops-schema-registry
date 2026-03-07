# Prometheus Metrics

## Overview

AxonOps Schema Registry exposes Prometheus metrics at `GET /metrics` in OpenMetrics format. Metrics fall into two categories:

1. **Confluent-compatible metrics** (`kafka_schema_registry_*`) -- match the metric names produced by Confluent Schema Registry's JMX exporter, so existing Grafana dashboards and alerts work without changes.
2. **AxonOps-native metrics** (`schema_registry_*`) -- provide deeper observability into storage, cache, auth, compatibility, and MCP operations.

## Confluent-Compatible Metrics

These metrics use the `kafka_schema_registry_` prefix, matching Confluent's JMX MBean names after Prometheus JMX exporter conversion:

- **`kafka_schema_registry_registered_count`** (counter) -- total schemas registered. Equivalent to Confluent's `registered-count` JMX MBean.
- **`kafka_schema_registry_deleted_count`** (counter) -- total schemas deleted. Equivalent to Confluent's `deleted-count` JMX MBean.
- **`kafka_schema_registry_api_success_count`** (counter) -- successful API calls (HTTP 2xx/3xx). Equivalent to Confluent's `api-success-count`.
- **`kafka_schema_registry_api_failure_count`** (counter) -- failed API calls (HTTP 4xx/5xx). Equivalent to Confluent's `api-failure-count`.
- **`kafka_schema_registry_schemas_created{schema_type}`** (counter) -- schemas created by type (avro, json, protobuf). Equivalent to Confluent's per-type `*-schemas-created` MBeans.
- **`kafka_schema_registry_schemas_deleted{schema_type}`** (counter) -- schemas deleted by type. Equivalent to Confluent's per-type `*-schemas-deleted` MBeans.
- **`kafka_schema_registry_master_slave_role`** (gauge) -- 1.0 = leader, 0.0 = follower. Always 1.0 for standalone. Equivalent to Confluent's `master-slave-role`.
- **`kafka_schema_registry_node_count`** (gauge) -- cluster size. Always 1 for standalone. Equivalent to Confluent's `node-count`.

## AxonOps-Native Metrics

### Request Metrics
- **`schema_registry_requests_total{method, path, status}`** -- total HTTP requests by method, normalized path, and status code.
- **`schema_registry_request_duration_seconds{method, path}`** -- request latency histogram.
- **`schema_registry_requests_in_flight`** -- concurrent requests being processed.

### Schema Metrics
- **`schema_registry_schemas_total{type}`** -- total schemas by type (gauge).
- **`schema_registry_subjects_total`** -- total subjects (gauge).
- **`schema_registry_schema_versions{subject}`** -- version count per subject (gauge).
- **`schema_registry_registrations_total{type, status}`** -- registration attempts (success/failure).

### Storage, Cache, Auth, and MCP Metrics
Use the `query_metric` or `list_metrics` tools to explore these categories interactively.

## Metric Types

- **Counter** -- monotonically increasing value (e.g., total requests). Use `rate()` in PromQL for per-second rates.
- **Gauge** -- value that goes up and down (e.g., in-flight requests, schema counts).
- **Histogram** -- samples observations into configurable buckets (e.g., latency). Use `histogram_quantile()` for percentiles.

## Label Cardinality

Path labels in request metrics are **normalized** to prevent cardinality explosion. Dynamic segments like subject names and schema IDs are replaced with `{subject}`, `{id}`, etc.
