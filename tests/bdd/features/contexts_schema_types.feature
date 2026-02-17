@functional
Feature: Contexts — Schema Types
  Verify that Avro, Protobuf, and JSON Schema types all work correctly
  within context-prefixed subjects.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # AVRO IN CONTEXT
  # ==========================================================================

  Scenario: Register Avro schema in context
    When I POST "/subjects/:.type-ctx:avro-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AvroCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    When I GET "/subjects/:.type-ctx:avro-test/versions/1"
    Then the response status should be 200
    And the response field "schemaType" should be "AVRO"

  Scenario: Avro compatibility check in context
    When I POST "/subjects/:.type-ctx2:avro-compat/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AvroCompat\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I set the config for subject ":.type-ctx2:avro-compat" to "BACKWARD"
    When I POST "/compatibility/subjects/:.type-ctx2:avro-compat/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AvroCompat\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # PROTOBUF IN CONTEXT
  # ==========================================================================

  Scenario: Register Protobuf schema in context
    When I POST "/subjects/:.type-ctx3:proto-test/versions" with body:
      """
      {"schemaType": "PROTOBUF", "schema": "syntax = \"proto3\";\nmessage ProtoCtx {\n  string name = 1;\n}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.type-ctx3:proto-test/versions/1"
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"

  Scenario: Protobuf compatibility check in context
    When I POST "/subjects/:.type-ctx4:proto-compat/versions" with body:
      """
      {"schemaType": "PROTOBUF", "schema": "syntax = \"proto3\";\nmessage ProtoCompat {\n  string name = 1;\n}"}
      """
    Then the response status should be 200
    When I set the config for subject ":.type-ctx4:proto-compat" to "BACKWARD"
    When I POST "/compatibility/subjects/:.type-ctx4:proto-compat/versions/latest" with body:
      """
      {"schemaType": "PROTOBUF", "schema": "syntax = \"proto3\";\nmessage ProtoCompat {\n  string name = 1;\n  int32 age = 2;\n}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # JSON SCHEMA IN CONTEXT
  # ==========================================================================

  Scenario: Register JSON Schema in context
    When I POST "/subjects/:.type-ctx5:json-test/versions" with body:
      """
      {"schemaType": "JSON", "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"}}}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.type-ctx5:json-test/versions/1"
    Then the response status should be 200
    And the response field "schemaType" should be "JSON"

  Scenario: JSON Schema compatibility check in context
    When I POST "/subjects/:.type-ctx6:json-compat/versions" with body:
      """
      {"schemaType": "JSON", "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"age\":{\"type\":\"integer\"}},\"required\":[\"name\"]}"}
      """
    Then the response status should be 200
    When I set the config for subject ":.type-ctx6:json-compat" to "BACKWARD"
    # BACKWARD for JSON Schema: removing a property is compatible
    When I POST "/compatibility/subjects/:.type-ctx6:json-compat/versions/latest" with body:
      """
      {"schemaType": "JSON", "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"}},\"required\":[\"name\"]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # MIXED TYPES
  # ==========================================================================

  Scenario: Different schema types in same context
    When I POST "/subjects/:.mixed-ctx:avro-s/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Mixed\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.mixed-ctx:proto-s/versions" with body:
      """
      {"schemaType": "PROTOBUF", "schema": "syntax = \"proto3\";\nmessage Mixed {\n  string a = 1;\n}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.mixed-ctx:json-s/versions" with body:
      """
      {"schemaType": "JSON", "schema": "{\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"}
      """
    Then the response status should be 200
    # All three subjects exist in the same context
    When I GET "/subjects/:.mixed-ctx:avro-s/versions/1"
    Then the response status should be 200
    And the response field "schemaType" should be "AVRO"
    When I GET "/subjects/:.mixed-ctx:proto-s/versions/1"
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"
    When I GET "/subjects/:.mixed-ctx:json-s/versions/1"
    Then the response status should be 200
    And the response field "schemaType" should be "JSON"

  Scenario: Same schema type across different contexts
    When I POST "/subjects/:.type-a:avro-cross/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CrossA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.type-b:avro-cross/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CrossB\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.type-a:avro-cross/versions/1"
    Then the response status should be 200
    And the response body should contain "CrossA"
    When I GET "/subjects/:.type-b:avro-cross/versions/1"
    Then the response status should be 200
    And the response body should contain "CrossB"
