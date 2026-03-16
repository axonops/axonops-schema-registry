@schema-modeling @negative
Feature: Negative Validation
  Tests that invalid schemas are correctly rejected with appropriate error
  codes. Covers Avro, Protobuf, and JSON Schema negative cases plus
  cross-type behaviors like idempotent re-registration and deduplication.

  # ==========================================================================
  # AVRO NEGATIVE CASES
  # ==========================================================================

  Scenario: Avro — invalid JSON is rejected
    When I register a schema under subject "neg-avro-invalid-json":
      """
      {this is not valid json
      """
    Then the response status should be 422

  Scenario: Avro — missing type field is rejected
    When I register a schema under subject "neg-avro-no-type":
      """
      {"name":"Oops","fields":[{"name":"x","type":"int"}]}
      """
    Then the response status should be 422

  Scenario: Avro — duplicate field names is rejected
    When I register a schema under subject "neg-avro-dup-fields":
      """
      {"type":"record","name":"Bad","fields":[
        {"name":"x","type":"int"},
        {"name":"x","type":"string"}
      ]}
      """
    Then the response status should be 422

  @axonops-only
  Scenario: Avro — invalid default type is rejected
    When I register a schema under subject "neg-avro-bad-default":
      """
      {"type":"record","name":"Bad","fields":[
        {"name":"count","type":"int","default":"not_a_number"}
      ]}
      """
    Then the response status should be 422

  Scenario: Avro — unknown type reference without declaration is rejected
    When I register a schema under subject "neg-avro-unknown-ref":
      """
      {"type":"record","name":"Bad","fields":[
        {"name":"item","type":"com.unknown.NonexistentType"}
      ]}
      """
    Then the response status should be 422

  @axonops-only
  Scenario: Avro — enum with empty symbols is rejected
    When I register a schema under subject "neg-avro-empty-enum":
      """
      {"type":"enum","name":"Empty","symbols":[]}
      """
    Then the response status should be 422

  @axonops-only
  Scenario: Avro — fixed with size 0 is rejected
    When I register a schema under subject "neg-avro-fixed-zero":
      """
      {"type":"fixed","name":"Zero","size":0}
      """
    Then the response status should be 422

  # ==========================================================================
  # PROTOBUF NEGATIVE CASES
  # ==========================================================================

  Scenario: Protobuf — invalid syntax is rejected
    When I register a "PROTOBUF" schema under subject "neg-proto-invalid":
      """
this is not valid protobuf
      """
    Then the response status should be 422

  @axonops-only
  Scenario: Protobuf — duplicate field number is rejected
    When I register a "PROTOBUF" schema under subject "neg-proto-dup-num":
      """
syntax = "proto3";
package test.neg;

message Bad {
  string a = 1;
  int32 b = 1;
}
      """
    Then the response status should be 422

  @axonops-only
  Scenario: Protobuf — import without reference declaration is rejected
    When I register a "PROTOBUF" schema under subject "neg-proto-missing-import":
      """
syntax = "proto3";
package test.neg;

import "nonexistent/file.proto";

message Bad {
  string name = 1;
}
      """
    Then the response status should be 422

  # ==========================================================================
  # JSON SCHEMA NEGATIVE CASES
  # ==========================================================================

  Scenario: JSON Schema — invalid JSON is rejected
    When I register a "JSON" schema under subject "neg-json-invalid":
      """
      {not valid json at all
      """
    Then the response status should be 422

  # ==========================================================================
  # CROSS-TYPE MISMATCH
  # ==========================================================================

  Scenario: Schema type mismatch — register Avro then Protobuf in same subject is rejected
    When I register a schema under subject "neg-type-mismatch":
      """
      {"type":"record","name":"First","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    # Protobuf schema is valid but incompatible with existing Avro — 409
    When I register a "PROTOBUF" schema under subject "neg-type-mismatch":
      """
syntax = "proto3";
package test.neg;

message Second {
  int64 id = 1;
}
      """
    Then the response status should be 409

  # ==========================================================================
  # CROSS-TYPE BEHAVIORS
  # ==========================================================================

  Scenario: Re-register identical schema returns 200 with same version
    When I register a schema under subject "neg-idempotent":
      """
      {"type":"record","name":"Idem","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "idem_id"
    When I register a schema under subject "neg-idempotent":
      """
      {"type":"record","name":"Idem","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "idem_id"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | neg-idempotent                           |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/neg-idempotent/versions        |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Register same Avro under two subjects returns same global ID
    When I register a schema under subject "neg-dedup-a":
      """
      {"type":"record","name":"Dedup","fields":[{"name":"x","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "dedup_global_id"
    When I register a schema under subject "neg-dedup-b":
      """
      {"type":"record","name":"Dedup","fields":[{"name":"x","type":"int"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "dedup_global_id"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | neg-dedup-b                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/neg-dedup-b/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
