@functional @metrics
Feature: Wire-Compatible Metrics
  As a schema registry operator migrating from an existing deployment
  I want AxonOps Schema Registry to expose wire-compatible metrics
  So that existing Grafana dashboards and alerting rules continue to work

  Background:
    Given the schema registry is running
    And the global compatibility level is "NONE"

  # ---------------------------------------------------------------------------
  # Static gauges
  # ---------------------------------------------------------------------------

  Scenario: master_slave_role gauge reports leader status
    Then the Prometheus metric "kafka_schema_registry_master_slave_role" should exist
    And the Prometheus metric "kafka_schema_registry_master_slave_role" should have value 1

  Scenario: node_count gauge reports cluster size
    Then the Prometheus metric "kafka_schema_registry_node_count" should exist
    And the Prometheus metric "kafka_schema_registry_node_count" should have value 1

  # ---------------------------------------------------------------------------
  # Wire-compatible metric existence (after triggering at least one operation)
  # ---------------------------------------------------------------------------

  Scenario: All wire-compatible metrics appear in metrics output
    # Trigger at least one registration so counters are initialized
    When I register a schema under subject "metrics-existence-test":
      """
      {"type":"record","name":"MetricsExist","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_registered_count" should exist
    And the Prometheus metric "kafka_schema_registry_api_success_count" should exist
    And the Prometheus metric "kafka_schema_registry_api_failure_count" should exist
    And the Prometheus metric "kafka_schema_registry_master_slave_role" should exist
    And the Prometheus metric "kafka_schema_registry_node_count" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-existence-test                       |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/metrics-existence-test/versions    |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ---------------------------------------------------------------------------
  # API call counters (AxonOps-only — counter semantics differ across JMX bridge)
  # ---------------------------------------------------------------------------

  @axonops-only
  Scenario: api_success_count increments on successful requests
    Given I store the current value of metric "kafka_schema_registry_api_success_count" as "before"
    When I GET "/"
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_api_success_count" should have increased from "before"

  @axonops-only
  Scenario: api_failure_count increments on failed requests
    Given I store the current value of metric "kafka_schema_registry_api_failure_count" as "before"
    When I GET "/subjects/nonexistent-subject-xxxxx/versions/999"
    Then the response status should be 404
    And the Prometheus metric "kafka_schema_registry_api_failure_count" should have increased from "before"

  # ---------------------------------------------------------------------------
  # Schema registration counters (AxonOps-only — increment verification)
  # ---------------------------------------------------------------------------

  @axonops-only
  Scenario: registered_count increments when a schema is registered
    Given I store the current value of metric "kafka_schema_registry_registered_count" as "before"
    When I register a schema under subject "metrics-reg-test":
      """
      {"type":"record","name":"MetricsReg","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_registered_count" should have increased from "before"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-reg-test                             |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/metrics-reg-test/versions          |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  @axonops-only
  Scenario: schemas_created counter tracks Avro registrations
    Given I store the current value of metric "kafka_schema_registry_registered_count" as "before_total"
    When I register a schema under subject "metrics-avro-test":
      """
      {"type":"record","name":"MetricsAvro","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_schemas_created" with labels "schema_type=\"avro\"" should exist
    And the Prometheus metric "kafka_schema_registry_registered_count" should have increased from "before_total"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-avro-test                            |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/metrics-avro-test/versions         |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  @axonops-only
  Scenario: schemas_created counter tracks JSON Schema registrations
    When I register a "JSON" schema under subject "metrics-json-test":
      """
      {"type":"object","properties":{"id":{"type":"integer"}}}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_schemas_created" with labels "schema_type=\"json\"" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-json-test                            |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/metrics-json-test/versions         |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ---------------------------------------------------------------------------
  # Schema deletion counters (AxonOps-only — increment verification)
  # ---------------------------------------------------------------------------

  @axonops-only
  Scenario: deleted_count increments when a subject is deleted
    Given subject "metrics-del-test" has schema:
      """
      {"type":"record","name":"MetricsDel","fields":[{"name":"id","type":"int"}]}
      """
    And I store the current value of metric "kafka_schema_registry_deleted_count" as "before"
    When I delete subject "metrics-del-test"
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_deleted_count" should have increased from "before"
    And the audit log should contain an event:
      | event_type           | subject_delete_soft                          |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-del-test                             |
      | schema_id            |                                              |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          | sha256:*                                     |
      | after_hash           |                                              |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | DELETE                                       |
      | path                 | /subjects/metrics-del-test                   |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  @axonops-only
  Scenario: schemas_deleted counter tracks deletions by type
    Given subject "metrics-del-type-test" has schema:
      """
      {"type":"record","name":"MetricsDelType","fields":[{"name":"id","type":"int"}]}
      """
    When I delete subject "metrics-del-type-test"
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_schemas_deleted" with labels "schema_type=\"avro\"" should exist
    And the audit log should contain an event:
      | event_type           | subject_delete_soft                          |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-del-type-test                        |
      | schema_id            |                                              |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          | sha256:*                                     |
      | after_hash           |                                              |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | DELETE                                       |
      | path                 | /subjects/metrics-del-type-test              |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  @axonops-only
  Scenario: deleted_count increments when a version is deleted
    Given subject "metrics-ver-del" has schema:
      """
      {"type":"record","name":"MetricsVerDel","fields":[{"name":"id","type":"int"}]}
      """
    And subject "metrics-ver-del" has schema:
      """
      {"type":"record","name":"MetricsVerDel","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    And I store the current value of metric "kafka_schema_registry_deleted_count" as "before"
    When I delete version 1 of subject "metrics-ver-del"
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_deleted_count" should have increased from "before"
    And the Prometheus metric "kafka_schema_registry_schemas_deleted" with labels "schema_type=\"avro\"" should exist
    And the audit log should contain an event:
      | event_type           | schema_delete_soft                           |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-ver-del                              |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          | sha256:*                                     |
      | after_hash           |                                              |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | DELETE                                       |
      | path                 | /subjects/metrics-ver-del/versions/1         |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ---------------------------------------------------------------------------
  # Per-endpoint Confluent-compatible metrics
  # ---------------------------------------------------------------------------

  @axonops-only
  Scenario: Per-endpoint request metrics track schema registration
    When I register a schema under subject "metrics-endpoint-test":
      """
      {"type":"record","name":"MetricsEndpoint","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_jersey_metrics_request_total" with labels "endpoint=\"subjects.versions.register\"" should exist
    And the Prometheus metric "kafka_schema_registry_jersey_metrics_request_latency_seconds_count" with labels "endpoint=\"subjects.versions.register\"" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-endpoint-test                        |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/metrics-endpoint-test/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  @axonops-only
  Scenario: Per-endpoint request metrics track subject listing
    When I GET "/subjects"
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_jersey_metrics_request_total" with labels "endpoint=\"subjects.list\"" should exist

  @axonops-only
  Scenario: Per-endpoint error metrics track failures
    When I GET "/subjects/nonexistent-endpoint-metrics-test/versions"
    Then the response status should be 404
    And the Prometheus metric "kafka_schema_registry_jersey_metrics_request_error_total" with labels "endpoint=\"subjects.versions.list\"" should exist

  # ---------------------------------------------------------------------------
  # AxonOps-native metrics coexist with wire-compatible metrics
  # ---------------------------------------------------------------------------

  @axonops-only
  Scenario: AxonOps-native request metrics are present
    When I GET "/"
    Then the response status should be 200
    And the Prometheus metric "schema_registry_requests_total" should exist
    And the Prometheus metric "schema_registry_request_duration_seconds" should exist
    And the Prometheus metric "schema_registry_requests_in_flight" should exist
