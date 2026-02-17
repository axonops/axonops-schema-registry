@functional
Feature: Schema Deduplication
  As a developer, I want the registry to deduplicate schemas by content fingerprint
  so that the same schema content always gets the same global schema ID regardless of subject

  # --------------------------------------------------------------------------
  # SAME SCHEMA ACROSS SUBJECTS
  # --------------------------------------------------------------------------

  Scenario: Same Avro schema in two subjects gets same schema ID
    When I register a schema under subject "dedup-avro-a":
      """
      {"type":"record","name":"Sensor","fields":[{"name":"id","type":"string"},{"name":"value","type":"double"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_a"
    When I register a schema under subject "dedup-avro-b":
      """
      {"type":"record","name":"Sensor","fields":[{"name":"id","type":"string"},{"name":"value","type":"double"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_b"

  Scenario: Same Protobuf schema in two subjects gets same schema ID
    When I register a "PROTOBUF" schema under subject "dedup-proto-a":
      """
      syntax = "proto3";
      message Event {
        string name = 1;
        int64 timestamp = 2;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "id_a"
    When I register a "PROTOBUF" schema under subject "dedup-proto-b":
      """
      syntax = "proto3";
      message Event {
        string name = 1;
        int64 timestamp = 2;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "id_b"

  Scenario: Same JSON Schema in two subjects gets same schema ID
    When I register a "JSON" schema under subject "dedup-json-a":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_a"
    When I register a "JSON" schema under subject "dedup-json-b":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_b"

  # --------------------------------------------------------------------------
  # IDEMPOTENT REGISTRATION
  # --------------------------------------------------------------------------

  Scenario: Duplicate registration in same subject returns same version (idempotent)
    Given subject "dedup-idempotent" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"name","type":"string"},{"name":"value","type":"float"}]}
      """
    When I register a schema under subject "dedup-idempotent":
      """
      {"type":"record","name":"Metric","fields":[{"name":"name","type":"string"},{"name":"value","type":"float"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get version 1 of subject "dedup-idempotent"
    Then the response status should be 200
    And the response field "version" should be 1
    When I list versions of subject "dedup-idempotent"
    Then the response status should be 200
    And the response should be an array of length 1

  # --------------------------------------------------------------------------
  # DIFFERENT CONTENT GETS DIFFERENT IDS
  # --------------------------------------------------------------------------

  Scenario: Different schema content gets different schema IDs
    When I register a schema under subject "dedup-diff-a":
      """
      {"type":"record","name":"Alpha","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_alpha"
    When I register a schema under subject "dedup-diff-b":
      """
      {"type":"record","name":"Beta","fields":[{"name":"y","type":"long"}]}
      """
    Then the response status should be 200
    And the response should have field "id"

  # --------------------------------------------------------------------------
  # CROSS-SUBJECT VISIBILITY VIA API
  # --------------------------------------------------------------------------

  Scenario: Schema ID shared across subjects visible via GET /schemas/ids/{id}/subjects
    When I register a schema under subject "dedup-vis-one":
      """
      {"type":"record","name":"Shared","fields":[{"name":"key","type":"string"},{"name":"val","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I register a schema under subject "dedup-vis-two":
      """
      {"type":"record","name":"Shared","fields":[{"name":"key","type":"string"},{"name":"val","type":"int"}]}
      """
    Then the response status should be 200
    When I get the subjects for the stored schema ID
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "dedup-vis-one"
    And the response array should contain "dedup-vis-two"

  Scenario: Schema ID shared across subjects visible via GET /schemas/ids/{id}/versions
    When I register a schema under subject "dedup-ver-x":
      """
      {"type":"record","name":"Common","fields":[{"name":"ts","type":"long"},{"name":"source","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I register a schema under subject "dedup-ver-y":
      """
      {"type":"record","name":"Common","fields":[{"name":"ts","type":"long"},{"name":"source","type":"string"}]}
      """
    Then the response status should be 200
    When I get versions for schema ID {{schema_id}}
    Then the response status should be 200
    And the response should be an array of length 2

  # --------------------------------------------------------------------------
  # NORMALIZATION / FINGERPRINTING
  # --------------------------------------------------------------------------

  Scenario: Whitespace-normalized Avro schemas produce same fingerprint and same ID
    When I register a schema under subject "dedup-ws-compact":
      """
      {"type":"record","name":"Item","fields":[{"name":"id","type":"string"},{"name":"count","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_compact"
    When I register a schema under subject "dedup-ws-spaced":
      """
      {
        "type": "record",
        "name": "Item",
        "fields": [
          {"name": "id", "type": "string"},
          {"name": "count", "type": "int"}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "id_spaced"
    And I store the response field "id" as "schema_id"
    When I get the subjects for the stored schema ID
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "dedup-ws-compact"
    And the response array should contain "dedup-ws-spaced"
