@functional
Feature: Compatibility Check Verbose Mode
  Confluent Schema Registry supports a ?verbose=true query parameter on
  compatibility check endpoints. When verbose is true, the response includes
  a "messages" array describing incompatibilities. When not verbose (default),
  the messages field is omitted.

  # ==========================================================================
  # VERBOSE = FALSE (default) — messages should be omitted
  # ==========================================================================

  Scenario: Compatible schema without verbose omits messages field
    Given the global compatibility level is "BACKWARD"
    And subject "compat-verbose-1" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/compat-verbose-1/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"V1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"x\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true
    And the response should not have field "messages"

  Scenario: Incompatible schema without verbose omits messages field
    Given the global compatibility level is "BACKWARD"
    And subject "compat-verbose-2" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/compat-verbose-2/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"V2\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
    And the response should not have field "messages"

  # ==========================================================================
  # VERBOSE = TRUE — messages should be included
  # ==========================================================================

  Scenario: Compatible schema with verbose=true has empty or no messages
    Given the global compatibility level is "BACKWARD"
    And subject "compat-verbose-3" has schema:
      """
      {"type":"record","name":"V3","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/compat-verbose-3/versions/latest?verbose=true" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"V3\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"x\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: Incompatible Avro schema with verbose=true includes messages
    Given the global compatibility level is "BACKWARD"
    And subject "compat-verbose-4" has schema:
      """
      {"type":"record","name":"V4","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/compat-verbose-4/versions/latest?verbose=true" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"V4\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
    And the response should have field "messages"

  Scenario: Incompatible Avro schema with verbose against specific version
    Given the global compatibility level is "BACKWARD"
    And subject "compat-verbose-5" has schema:
      """
      {"type":"record","name":"V5","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/compat-verbose-5/versions/1?verbose=true" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"V5\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
    And the response should have field "messages"

  # ==========================================================================
  # VERBOSE WITH DIFFERENT SCHEMA TYPES
  # ==========================================================================

  Scenario: Incompatible Protobuf schema with verbose=true includes messages
    Given the global compatibility level is "BACKWARD"
    And subject "compat-verbose-proto" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message ProtoV1 {
        string name = 1;
      }
      """
    When I POST "/compatibility/subjects/compat-verbose-proto/versions/latest?verbose=true" with body:
      """
      {"schema": "syntax = \"proto3\";\nmessage ProtoV1 {\n  int32 name = 1;\n}", "schemaType": "PROTOBUF"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
    And the response should have field "messages"

  Scenario: Incompatible JSON Schema with verbose=true includes messages
    Given the global compatibility level is "BACKWARD"
    And subject "compat-verbose-json" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I POST "/compatibility/subjects/compat-verbose-json/versions/latest?verbose=true" with body:
      """
      {"schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"integer\"}},\"required\":[\"name\"]}", "schemaType": "JSON"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
    And the response should have field "messages"

  # ==========================================================================
  # VERBOSE AGAINST ALL VERSIONS
  # ==========================================================================

  Scenario: Verbose check against all versions of subject
    Given the global compatibility level is "BACKWARD"
    And subject "compat-verbose-all" has schema:
      """
      {"type":"record","name":"All1","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/compat-verbose-all/versions?verbose=true" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"All1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
    And the response should have field "messages"
