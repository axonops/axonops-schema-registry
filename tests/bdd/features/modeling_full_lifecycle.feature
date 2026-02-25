@schema-modeling @lifecycle
Feature: Full Schema Lifecycle
  End-to-end lifecycle tests covering register, retrieve, verify content,
  evolve, check compatibility, idempotent re-register, soft-delete,
  re-register after delete, mode changes, and config changes.

  # ==========================================================================
  # 1. AVRO FULL LIFECYCLE
  # ==========================================================================

  Scenario: Avro full lifecycle — register evolve delete re-register
    Given subject "lifecycle-avro" has compatibility level "BACKWARD"
    # Register v1
    When I register a schema under subject "lifecycle-avro":
      """
      {"type":"record","name":"Event","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "avro_v1_id"
    # Retrieve and verify content
    When I get version 1 of subject "lifecycle-avro"
    Then the response status should be 200
    And the response body should contain "Event"
    # Evolve to v2
    When I register a schema under subject "lifecycle-avro":
      """
      {"type":"record","name":"Event","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"source","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    # Idempotent re-register v2
    When I register a schema under subject "lifecycle-avro":
      """
      {"type":"record","name":"Event","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"source","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    # Soft-delete
    When I delete subject "lifecycle-avro"
    Then the response status should be 200
    # Register v3 after delete
    When I register a schema under subject "lifecycle-avro":
      """
      {"type":"record","name":"Event","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"source","type":"string","default":""},
        {"name":"ts","type":"long","default":0}
      ]}
      """
    Then the response status should be 200
    # Verify latest
    When I get the latest version of subject "lifecycle-avro"
    Then the response status should be 200
    And the response body should contain "ts"

  # ==========================================================================
  # 2. PROTOBUF FULL LIFECYCLE
  # ==========================================================================

  Scenario: Protobuf full lifecycle — register evolve delete re-register
    Given subject "lifecycle-proto" has compatibility level "BACKWARD"
    # Register v1
    When I register a "PROTOBUF" schema under subject "lifecycle-proto":
      """
syntax = "proto3";
package lifecycle;

message Event {
  string id = 1;
  string name = 2;
}
      """
    Then the response status should be 200
    # Retrieve and verify
    When I get version 1 of subject "lifecycle-proto"
    Then the response status should be 200
    And the response body should contain "Event"
    # Evolve to v2
    When I register a "PROTOBUF" schema under subject "lifecycle-proto":
      """
syntax = "proto3";
package lifecycle;

message Event {
  string id = 1;
  string name = 2;
  string source = 3;
}
      """
    Then the response status should be 200
    # Soft-delete
    When I delete subject "lifecycle-proto"
    Then the response status should be 200
    # Register v3 after delete
    When I register a "PROTOBUF" schema under subject "lifecycle-proto":
      """
syntax = "proto3";
package lifecycle;

message Event {
  string id = 1;
  string name = 2;
  string source = 3;
  int64 timestamp = 4;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 3. JSON SCHEMA FULL LIFECYCLE
  # ==========================================================================

  Scenario: JSON Schema full lifecycle — register evolve delete re-register
    Given subject "lifecycle-json" has compatibility level "BACKWARD"
    # Register v1
    When I register a "JSON" schema under subject "lifecycle-json":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id","name"],"additionalProperties":false}
      """
    Then the response status should be 200
    # Retrieve and verify
    When I get version 1 of subject "lifecycle-json"
    Then the response status should be 200
    And the response body should contain "properties"
    # Evolve to v2 (add optional property)
    When I register a "JSON" schema under subject "lifecycle-json":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}},"required":["id","name"],"additionalProperties":false}
      """
    Then the response status should be 200
    # Soft-delete
    When I delete subject "lifecycle-json"
    Then the response status should be 200
    # Register v3 after delete
    When I register a "JSON" schema under subject "lifecycle-json":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["id","name"],"additionalProperties":false}
      """
    Then the response status should be 200

  # ==========================================================================
  # 4. CROSS-TYPE LIFECYCLE
  # ==========================================================================

  Scenario: Cross-type subjects evolve independently
    # Register Avro subject
    When I register a schema under subject "lifecycle-cross-avro":
      """
      {"type":"record","name":"A","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    # Register Proto subject
    When I register a "PROTOBUF" schema under subject "lifecycle-cross-proto":
      """
syntax = "proto3";
package cross;

message B {
  int64 id = 1;
}
      """
    Then the response status should be 200
    # Register JSON subject
    When I register a "JSON" schema under subject "lifecycle-cross-json":
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    Then the response status should be 200
    # Delete all three
    When I delete subject "lifecycle-cross-avro"
    Then the response status should be 200
    When I delete subject "lifecycle-cross-proto"
    Then the response status should be 200
    When I delete subject "lifecycle-cross-json"
    Then the response status should be 200

  # ==========================================================================
  # 5. SCHEMA WITH REFERENCES LIFECYCLE
  # ==========================================================================

  Scenario: Schema with references lifecycle — register ref then consumer then evolve
    # Register reference schema
    Given subject "lifecycle-ref-base" has schema:
      """
      {"type":"record","name":"Base","namespace":"com.lc","fields":[{"name":"id","type":"long"}]}
      """
    # Register consumer
    When I register a schema under subject "lifecycle-ref-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.lc\",\"fields\":[{\"name\":\"base\",\"type\":\"com.lc.Base\"},{\"name\":\"name\",\"type\":\"string\"}]}",
        "references": [
          {"name":"com.lc.Base","subject":"lifecycle-ref-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    # Verify referencedby
    When I get the referenced by for subject "lifecycle-ref-base" version 1
    Then the response status should be 200
    # Evolve reference
    Given subject "lifecycle-ref-base" has compatibility level "BACKWARD"
    When I register a schema under subject "lifecycle-ref-base":
      """
      {"type":"record","name":"Base","namespace":"com.lc","fields":[
        {"name":"id","type":"long"},
        {"name":"label","type":"string","default":""}
      ]}
      """
    Then the response status should be 200

  # ==========================================================================
  # 6. IMPORT AND EVOLVE
  # ==========================================================================

  Scenario: Import schema with specific ID then evolve via normal registration
    Given subject "lifecycle-import" has compatibility level "BACKWARD"
    When I set the global mode to "IMPORT"
    And I import a schema with ID 99000 under subject "lifecycle-import" version 1:
      """
      {"type":"record","name":"Imported","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    And I register a schema under subject "lifecycle-import":
      """
      {"type":"record","name":"Imported","fields":[{"name":"id","type":"long"},{"name":"data","type":"string","default":""}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # 7. MODE CHANGES
  # ==========================================================================

  Scenario: Mode changes — READWRITE to READONLY blocks writes then READWRITE resumes
    # Register a schema in READWRITE mode
    When I register a schema under subject "lifecycle-mode":
      """
      {"type":"record","name":"M","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    # Switch to READONLY
    When I set the mode for subject "lifecycle-mode" to "READONLY"
    Then the response status should be 200
    # Attempt to register should fail
    When I register a schema under subject "lifecycle-mode":
      """
      {"type":"record","name":"M","fields":[{"name":"id","type":"long"},{"name":"x","type":"string","default":""}]}
      """
    Then the response status should be 422
    # Switch back to READWRITE
    When I set the mode for subject "lifecycle-mode" to "READWRITE"
    Then the response status should be 200
    # Now registration should succeed
    When I register a schema under subject "lifecycle-mode":
      """
      {"type":"record","name":"M","fields":[{"name":"id","type":"long"},{"name":"x","type":"string","default":""}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # 8. CONFIG CHANGES
  # ==========================================================================

  Scenario: Config changes — BACKWARD to NONE allows incompatible then BACKWARD_TRANSITIVE blocks
    Given subject "lifecycle-config" has compatibility level "BACKWARD"
    And subject "lifecycle-config" has schema:
      """
      {"type":"record","name":"C","fields":[
        {"name":"a","type":"string"},
        {"name":"b","type":"int"}
      ]}
      """
    # Switch to NONE — allow incompatible change
    Given subject "lifecycle-config" has compatibility level "NONE"
    When I register a schema under subject "lifecycle-config":
      """
      {"type":"record","name":"C","fields":[
        {"name":"a","type":"string"},
        {"name":"c","type":"long"}
      ]}
      """
    Then the response status should be 200
    # Switch to BACKWARD_TRANSITIVE
    Given subject "lifecycle-config" has compatibility level "BACKWARD_TRANSITIVE"
    # v3 adds field "e" WITHOUT default. Under BACKWARD_TRANSITIVE, v3 (reader)
    # must be able to read ALL prior versions. v1 and v2 data don't have "e"
    # and v3 has no default for it — incompatible.
    When I register a schema under subject "lifecycle-config":
      """
      {"type":"record","name":"C","fields":[
        {"name":"a","type":"string"},
        {"name":"e","type":"int"}
      ]}
      """
    Then the response status should be 409
