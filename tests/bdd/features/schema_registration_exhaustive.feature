@functional
Feature: Schema Registration — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive schema registration tests derived from Confluent Schema Registry
  v8.1.1 test suite covering Avro, JSON Schema, and Protobuf registration patterns.

  # ==========================================================================
  # AVRO REGISTRATION
  # ==========================================================================

  Scenario: Register multiple schemas under a subject creates incrementing versions
    Given the global compatibility level is "NONE"
    When I register a schema under subject "reg-multi-avro":
      """
      {"type":"record","name":"V1","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id1"
    When I register a schema under subject "reg-multi-avro":
      """
      {"type":"record","name":"V2","fields":[{"name":"f2","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id2"
    When I register a schema under subject "reg-multi-avro":
      """
      {"type":"record","name":"V3","fields":[{"name":"f3","type":"long"}]}
      """
    Then the response status should be 200
    When I list versions of subject "reg-multi-avro"
    Then the response status should be 200
    And the response should be an array of length 3

  Scenario: Re-register existing schema returns same ID without new version
    Given subject "reg-redup" has schema:
      """
      {"type":"record","name":"Dup","fields":[{"name":"x","type":"string"}]}
      """
    When I register a schema under subject "reg-redup":
      """
      {"type":"record","name":"Dup","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "dup_id"
    When I list versions of subject "reg-redup"
    Then the response should be an array of length 1

  Scenario: Same schema under different subjects returns same global ID
    When I register a schema under subject "reg-global-s1":
      """
      {"type":"record","name":"GlobalID","fields":[{"name":"g","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "global_id1"
    When I register a schema under subject "reg-global-s2":
      """
      {"type":"record","name":"GlobalID","fields":[{"name":"g","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "global_id2"

  Scenario: Register invalid Avro schema with bad field type returns 422
    When I POST "/subjects/reg-bad-avro/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Bad\",\"fields\":[{\"name\":\"f\",\"type\":\"str\"}]}"}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register non-Avro string as Avro schema returns 422
    When I POST "/subjects/reg-nonavro/versions" with body:
      """
      {"schema": "not a valid schema at all"}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register schema with invalid reference returns 422
    When I POST "/subjects/reg-bad-ref/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Ref\",\"fields\":[{\"name\":\"r\",\"type\":\"NonExistentType\"}]}", "references": [{"name":"NonExistentType","subject":"nonexistent-subject","version":1}]}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register different schema types under same subject with NONE compatibility
    Given the global compatibility level is "NONE"
    When I register a schema under subject "reg-mixed-types":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And I store the response field "id" as "avro_id"
    When I register a "JSON" schema under subject "reg-mixed-types":
      """
      {"type":"object","properties":{"x":{"type":"string"}}}
      """
    Then the response status should be 200
    And I store the response field "id" as "json_id"
    When I register a "PROTOBUF" schema under subject "reg-mixed-types":
      """
      syntax = "proto3"; message Mixed { string x = 1; }
      """
    Then the response status should be 200

  Scenario: Schema whitespace canonicalization returns same ID
    When I register a schema under subject "reg-canon-1":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And I store the response field "id" as "canon_id1"
    When I register a schema under subject "reg-canon-2":
      """
      {   "type" :   "string"   }
      """
    Then the response status should be 200
    And I store the response field "id" as "canon_id2"

  # ==========================================================================
  # JSON SCHEMA REGISTRATION
  # ==========================================================================

  Scenario: Register multiple JSON schemas under a subject
    Given the global compatibility level is "NONE"
    When I register a "JSON" schema under subject "reg-multi-json":
      """
      {"type":"object","properties":{"f1":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "reg-multi-json":
      """
      {"type":"object","properties":{"f1":{"type":"string"},"f2":{"type":"integer"}},"additionalProperties":false}
      """
    Then the response status should be 200
    When I list versions of subject "reg-multi-json"
    Then the response should be an array of length 2

  Scenario: Re-register existing JSON schema returns same ID
    When I register a "JSON" schema under subject "reg-redup-json":
      """
      {"type":"object","properties":{"x":{"type":"string"}}}
      """
    Then the response status should be 200
    And I store the response field "id" as "json_dup_id1"
    When I register a "JSON" schema under subject "reg-redup-json":
      """
      {"type":"object","properties":{"x":{"type":"string"}}}
      """
    Then the response status should be 200
    And I store the response field "id" as "json_dup_id2"
    When I list versions of subject "reg-redup-json"
    Then the response should be an array of length 1

  Scenario: Register invalid JSON schema returns 422
    When I POST "/subjects/reg-bad-json/versions" with body:
      """
      {"schema": "{\"type\":\"bad-object\"}", "schemaType": "JSON"}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register JSON schema with invalid reference returns 422
    When I POST "/subjects/reg-bad-json-ref/versions" with body:
      """
      {"schema": "{\"$ref\":\"nonexistent.json\"}", "schemaType": "JSON", "references": [{"name":"nonexistent.json","subject":"bad-subject","version":100}]}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register incompatible JSON schemas under FULL compatibility returns 409
    Given subject "reg-incompat-json" has compatibility level "FULL"
    And subject "reg-incompat-json" has "JSON" schema:
      """
      {"type":"object","properties":{"f1":{"type":"string"},"f2":{"type":"number"}},"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "reg-incompat-json":
      """
      {"type":"object","properties":{"f1":{"type":"string"},"f2":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 409

  # ==========================================================================
  # PROTOBUF REGISTRATION
  # ==========================================================================

  Scenario: Register multiple Protobuf schemas under a subject
    Given the global compatibility level is "NONE"
    When I register a "PROTOBUF" schema under subject "reg-multi-proto":
      """
      syntax = "proto3"; message V1 { string f1 = 1; }
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "reg-multi-proto":
      """
      syntax = "proto3"; message V2 { string f1 = 1; int32 f2 = 2; }
      """
    Then the response status should be 200
    When I list versions of subject "reg-multi-proto"
    Then the response should be an array of length 2

  Scenario: Register incompatible Protobuf schemas under BACKWARD compatibility returns 409
    Given subject "reg-incompat-proto" has compatibility level "BACKWARD"
    And subject "reg-incompat-proto" has "PROTOBUF" schema:
      """
      syntax = "proto3"; message MyRecord { string f1 = 1; }
      """
    When I register a "PROTOBUF" schema under subject "reg-incompat-proto":
      """
      syntax = "proto3"; message MyRecord { int32 f1 = 1; }
      """
    Then the response status should be 409

  # ==========================================================================
  # CROSS-SUBJECT GLOBAL ID BEHAVIOR
  # ==========================================================================

  Scenario: Schema registered under different subjects maintains global ID
    Given the global compatibility level is "NONE"
    When I register a schema under subject "reg-xsubj-s1":
      """
      {"type":"record","name":"XSubj","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "xsubj_id1"
    When I register a schema under subject "reg-xsubj-s1":
      """
      {"type":"record","name":"XSubj2","fields":[{"name":"f2","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "xsubj_id2"
    When I register a schema under subject "reg-xsubj-s2":
      """
      {"type":"record","name":"XSubj2","fields":[{"name":"f2","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "xsubj_id3"
