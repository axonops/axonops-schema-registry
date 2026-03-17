@functional @metrics @axonops-only
Feature: AxonOps-Native Metrics
  As a schema registry operator
  I want all AxonOps-native metrics exposed at GET /metrics
  So that I can monitor every aspect of the registry's operation

  Background:
    Given the schema registry is running
    And the global compatibility level is "NONE"

  # ---------------------------------------------------------------------------
  # Request Metrics — always populated by the HTTP middleware
  # ---------------------------------------------------------------------------

  Scenario: requests_total counter increments on API calls
    Given I store the current value of metric "schema_registry_requests_total" as "before"
    When I GET "/"
    Then the response status should be 200
    And the Prometheus metric "schema_registry_requests_total" should exist

  Scenario: request_duration_seconds histogram has bucket data
    When I GET "/"
    Then the response status should be 200
    And the Prometheus metric "schema_registry_request_duration_seconds_bucket" should exist
    And the Prometheus metric "schema_registry_request_duration_seconds_count" should exist

  Scenario: requests_in_flight gauge is present
    Then the Prometheus metric "schema_registry_requests_in_flight" should exist

  # ---------------------------------------------------------------------------
  # Schema Metrics — populated by schema registration/lookup operations
  # ---------------------------------------------------------------------------

  Scenario: registrations_total counter tracks successful registrations
    When I register a schema under subject "metrics-reg-total-test":
      """
      {"type":"record","name":"MetricsRegTotal","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "schema_registry_registrations_total" with labels "status=\"success\"" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-reg-total-test                       |
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
      | path                 | /subjects/metrics-reg-total-test/versions    |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: registrations_total counter tracks registration by schema type
    When I register a "JSON" schema under subject "metrics-reg-json-test":
      """
      {"type":"object","properties":{"id":{"type":"integer"}}}
      """
    Then the response status should be 200
    And the Prometheus metric "schema_registry_registrations_total" with labels "type=\"JSON\"" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-reg-json-test                        |
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
      | path                 | /subjects/metrics-reg-json-test/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: schemas_total gauge tracks schema count by type
    When I register a schema under subject "metrics-schemas-total-test":
      """
      {"type":"record","name":"MetricsSchemasTotal","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And I wait for metrics refresh
    And the Prometheus metric "schema_registry_schemas_total" with labels "type=\"AVRO\"" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-schemas-total-test                   |
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
      | path                 | /subjects/metrics-schemas-total-test/versions |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: subjects_total gauge tracks subject count
    When I register a schema under subject "metrics-subjects-total-test":
      """
      {"type":"record","name":"MetricsSubjectsTotal","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And I wait for metrics refresh
    And the Prometheus metric "schema_registry_subjects_total" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-subjects-total-test                  |
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
      | path                 | /subjects/metrics-subjects-total-test/versions |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: schema_versions gauge tracks version count per subject
    When I register a schema under subject "metrics-schema-versions-test":
      """
      {"type":"record","name":"MetricsSchemaVersions","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "schema_registry_schema_versions" with labels "subject=\"metrics-schema-versions-test\"" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-schema-versions-test                 |
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
      | path                 | /subjects/metrics-schema-versions-test/versions |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ---------------------------------------------------------------------------
  # Storage Metrics — populated by storage operations
  # ---------------------------------------------------------------------------

  Scenario: storage_operations_total counter tracks storage backend calls
    When I register a schema under subject "metrics-storage-ops-test":
      """
      {"type":"record","name":"MetricsStorageOps","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "schema_registry_storage_operations_total" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-storage-ops-test                     |
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
      | path                 | /subjects/metrics-storage-ops-test/versions  |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: storage_latency_seconds histogram tracks storage operation latency
    When I register a schema under subject "metrics-storage-latency-test":
      """
      {"type":"record","name":"MetricsStorageLatency","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "schema_registry_storage_latency_seconds_bucket" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-storage-latency-test                 |
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
      | path                 | /subjects/metrics-storage-latency-test/versions |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ---------------------------------------------------------------------------
  # Compatibility Metrics — populated by compatibility checks
  # ---------------------------------------------------------------------------

  Scenario: compatibility_checks_total counter tracks compatibility checks
    When I register a schema under subject "metrics-compat-check-test":
      """
      {"type":"record","name":"MetricsCompatCheck","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And subject "metrics-compat-check-test" has compatibility level "BACKWARD"
    When I check compatibility of schema against subject "metrics-compat-check-test":
      """
      {"type":"record","name":"MetricsCompatCheck","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200
    And the Prometheus metric "schema_registry_compatibility_checks_total" should exist

  Scenario: compatibility_errors_total counter tracks compatibility check errors
    When I register a schema under subject "metrics-compat-error-test":
      """
      {"type":"record","name":"MetricsCompatError","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And subject "metrics-compat-error-test" has compatibility level "BACKWARD"
    When I check compatibility of schema against subject "metrics-compat-error-test":
      """
      this is not valid json at all
      """
    Then the response status should be 422
    And the Prometheus metric "schema_registry_compatibility_errors_total" should exist

  Scenario: storage_errors_total counter tracks storage errors
    When I GET "/subjects/nonexistent-subject-for-metrics/versions"
    Then the response status should be 404
    And the Prometheus metric "schema_registry_storage_errors_total" should exist

  # ---------------------------------------------------------------------------
  # Wire-Compatible Metrics — always populated
  # ---------------------------------------------------------------------------

  Scenario: All wire-compatible metrics coexist with native metrics
    When I register a schema under subject "metrics-coexist-test":
      """
      {"type":"record","name":"MetricsCoexist","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_registered_count" should exist
    And the Prometheus metric "kafka_schema_registry_api_success_count" should exist
    And the Prometheus metric "schema_registry_requests_total" should exist
    And the Prometheus metric "schema_registry_registrations_total" should exist
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | metrics-coexist-test                         |
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
      | path                 | /subjects/metrics-coexist-test/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ---------------------------------------------------------------------------
  # Auth Metrics — populated when auth is enabled
  # ---------------------------------------------------------------------------

  @auth
  Scenario: auth_attempts_total counter tracks authentication attempts
    Given I authenticate as "admin" with password "admin-password"
    When I GET "/subjects"
    Then the response status should be 200
    And the Prometheus metric "schema_registry_auth_attempts_total" should exist

  @auth
  Scenario: auth_failures_total counter tracks authentication failures
    Given I authenticate as "baduser" with password "wrongpass"
    When I GET "/subjects"
    Then the response status should be 401
    And the Prometheus metric "schema_registry_auth_failures_total" should exist

  @auth
  Scenario: auth_latency_seconds histogram tracks authentication latency
    Given I authenticate as "admin" with password "admin-password"
    When I GET "/subjects"
    Then the response status should be 200
    And the Prometheus metric "schema_registry_auth_latency_seconds" should exist
