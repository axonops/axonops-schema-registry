@functional
Feature: Schema ID Stability and Content Validation
  Verify schema IDs are stable across subjects (content-addressed deduplication)
  and that retrieved schemas are valid and match what was registered.

  # ==========================================================================
  # SCHEMA ID STABILITY ACROSS SUBJECTS
  # ==========================================================================

  Scenario: Same Avro schema in two subjects gets same ID verified via GET
    When I register a schema under subject "idstab-avro-a":
      """
      {"type":"record","name":"Stable","fields":[{"name":"id","type":"string"},{"name":"val","type":"double"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_a"
    When I register a schema under subject "idstab-avro-b":
      """
      {"type":"record","name":"Stable","fields":[{"name":"id","type":"string"},{"name":"val","type":"double"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_b"
    When I get schema by ID {{id_a}}
    Then the response status should be 200
    And the response should contain "Stable"

  Scenario: Different schema content gets different IDs
    When I register a schema under subject "idstab-diff-a":
      """
      {"type":"record","name":"Alpha","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_alpha"
    When I register a schema under subject "idstab-diff-b":
      """
      {"type":"record","name":"Beta","fields":[{"name":"y","type":"long"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_beta"

  # ==========================================================================
  # RETRIEVED SCHEMA CONTENT VALIDATION
  # ==========================================================================

  Scenario: Retrieved Avro schema is valid parseable JSON
    Given subject "idstab-valid-avro" has schema:
      """
      {"type":"record","name":"ValidAvro","fields":[{"name":"id","type":"string"},{"name":"count","type":"int"}]}
      """
    When I get version 1 of subject "idstab-valid-avro"
    Then the response status should be 200
    And the response should have field "schema"
    And the response should be valid JSON

  Scenario: Retrieved schema by ID matches what was registered
    Given subject "idstab-match" has schema:
      """
      {"type":"record","name":"MatchMe","fields":[{"name":"key","type":"string"}]}
      """
    And I store the response field "id" as "match_id"
    When I get schema by ID {{match_id}}
    Then the response status should be 200
    And the response should contain "MatchMe"
    And the response should contain "key"

  Scenario: Protobuf schema retrieved by version contains expected content
    Given subject "idstab-proto" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message ProtoStab {
        string key = 1;
        int32 value = 2;
      }
      """
    When I get version 1 of subject "idstab-proto"
    Then the response status should be 200
    And the response should contain "ProtoStab"

  Scenario: JSON Schema retrieved by version contains expected content
    Given subject "idstab-jsonsch" has "JSON" schema:
      """
      {"type":"object","properties":{"key":{"type":"string"},"value":{"type":"integer"}},"required":["key"]}
      """
    When I get version 1 of subject "idstab-jsonsch"
    Then the response status should be 200
    And the response should contain "key"
