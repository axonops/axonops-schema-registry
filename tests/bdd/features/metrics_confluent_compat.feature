@functional @metrics
Feature: Confluent-Compatible Prometheus Metrics
  As a schema registry operator migrating from Confluent
  I want AxonOps Schema Registry to expose Confluent-compatible Prometheus metrics
  So that existing Grafana dashboards and Prometheus alerts continue to work

  Background:
    Given the schema registry is running
    And the global compatibility level is "NONE"

  # ---------------------------------------------------------------------------
  # Static gauges: master-slave role and node count
  # ---------------------------------------------------------------------------

  Scenario: master_slave_role gauge is always 1 (standalone leader)
    Then the Prometheus metric "kafka_schema_registry_master_slave_role" should exist
    And the Prometheus metric "kafka_schema_registry_master_slave_role" should have value 1

  Scenario: node_count gauge is always 1 (standalone)
    Then the Prometheus metric "kafka_schema_registry_node_count" should exist
    And the Prometheus metric "kafka_schema_registry_node_count" should have value 1

  # ---------------------------------------------------------------------------
  # API call counters
  # ---------------------------------------------------------------------------

  Scenario: api_success_count increments on successful requests
    Given I store the current value of metric "kafka_schema_registry_api_success_count" as "before"
    When I GET "/"
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_api_success_count" should have increased from "before"

  Scenario: api_failure_count increments on failed requests
    Given I store the current value of metric "kafka_schema_registry_api_failure_count" as "before"
    When I GET "/subjects/nonexistent-subject-xxxxx/versions/999"
    Then the response status should be 404
    And the Prometheus metric "kafka_schema_registry_api_failure_count" should have increased from "before"

  # ---------------------------------------------------------------------------
  # Schema registration counters
  # ---------------------------------------------------------------------------

  Scenario: registered_count increments when a schema is registered
    Given I store the current value of metric "kafka_schema_registry_registered_count" as "before"
    When I register a schema under subject "metrics-reg-test":
      """
      {"type":"record","name":"MetricsReg","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_registered_count" should have increased from "before"

  Scenario: schemas_created counter tracks Avro registrations
    Given I store the current value of metric "kafka_schema_registry_registered_count" as "before_total"
    When I register a schema under subject "metrics-avro-test":
      """
      {"type":"record","name":"MetricsAvro","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_schemas_created" with labels "schema_type=\"avro\"" should exist
    And the Prometheus metric "kafka_schema_registry_registered_count" should have increased from "before_total"

  Scenario: schemas_created counter tracks JSON Schema registrations
    When I register a "JSON" schema under subject "metrics-json-test":
      """
      {"type":"object","properties":{"id":{"type":"integer"}}}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_schemas_created" with labels "schema_type=\"json\"" should exist

  # ---------------------------------------------------------------------------
  # Schema deletion counters
  # ---------------------------------------------------------------------------

  Scenario: deleted_count increments when a subject is deleted
    Given subject "metrics-del-test" has schema:
      """
      {"type":"record","name":"MetricsDel","fields":[{"name":"id","type":"int"}]}
      """
    And I store the current value of metric "kafka_schema_registry_deleted_count" as "before"
    When I delete subject "metrics-del-test"
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_deleted_count" should have increased from "before"

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

  # ---------------------------------------------------------------------------
  # All Confluent-compatible metrics exist in /metrics output
  # ---------------------------------------------------------------------------

  Scenario: All Confluent-compatible metrics appear in Prometheus output
    # Trigger at least one registration so counters are initialized
    When I register a schema under subject "metrics-existence-test":
      """
      {"type":"record","name":"MetricsExist","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the Prometheus metric "kafka_schema_registry_registered_count" should exist
    And the Prometheus metric "kafka_schema_registry_deleted_count" should exist
    And the Prometheus metric "kafka_schema_registry_api_success_count" should exist
    And the Prometheus metric "kafka_schema_registry_api_failure_count" should exist
    And the Prometheus metric "kafka_schema_registry_schemas_created" should exist
    And the Prometheus metric "kafka_schema_registry_master_slave_role" should exist
    And the Prometheus metric "kafka_schema_registry_node_count" should exist

  # ---------------------------------------------------------------------------
  # AxonOps-native metrics still work
  # ---------------------------------------------------------------------------

  Scenario: AxonOps-native metrics coexist with Confluent metrics
    When I GET "/"
    Then the response status should be 200
    And the Prometheus metric "schema_registry_requests_total" should exist
    And the Prometheus metric "schema_registry_request_duration_seconds" should exist
    And the Prometheus metric "schema_registry_requests_in_flight" should exist
    And the Prometheus metric "kafka_schema_registry_api_success_count" should exist
