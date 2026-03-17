@functional @analysis
Feature: REST Schema Validation and Normalization
  REST endpoints for validating and normalizing schemas without registering them.

  # --- POST /schemas/validate ---

  Scenario: Validate valid Avro schema
    When I POST "/schemas/validate" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_valid" should be true
    And the response field "schema_type" should be "AVRO"

  Scenario: Validate invalid Avro schema
    When I POST "/schemas/validate" with body:
      """
      {"schema": "{\"type\":\"invalid_type\"}"}
      """
    Then the response status should be 200
    And the response field "is_valid" should be false

  Scenario: Validate with missing schema field returns 400
    When I POST "/schemas/validate" with body:
      """
      {"schemaType": "AVRO"}
      """
    Then the response status should be 400
    And the response should have error code 42201

  Scenario: Validate defaults to AVRO when schemaType omitted
    When I POST "/schemas/validate" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "schema_type" should be "AVRO"

  Scenario: Validate valid JSON Schema
    When I POST "/schemas/validate" with body:
      """
      {"schema": "{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"integer\"}}}", "schemaType": "JSON"}
      """
    Then the response status should be 200
    And the response field "is_valid" should be true
    And the response field "schema_type" should be "JSON"

  Scenario: Validate valid Protobuf schema
    When I POST "/schemas/validate" with body:
      """
      {"schema": "syntax = \"proto3\"; message Test { int32 id = 1; }", "schemaType": "PROTOBUF"}
      """
    Then the response status should be 200
    And the response field "is_valid" should be true
    And the response field "schema_type" should be "PROTOBUF"

  # --- POST /schemas/normalize ---

  Scenario: Normalize valid Avro schema
    When I POST "/schemas/normalize" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response should have field "canonical"
    And the response should have field "fingerprint"

  Scenario: Normalize returns non-empty fingerprint
    When I POST "/schemas/normalize" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "fingerprint" should not be empty

  Scenario: Normalize invalid schema returns 422
    When I POST "/schemas/normalize" with body:
      """
      {"schema": "{\"type\":\"invalid\"}"}
      """
    Then the response status should be 422

  Scenario: Normalize with missing schema field returns 400
    When I POST "/schemas/normalize" with body:
      """
      {"schemaType": "AVRO"}
      """
    Then the response status should be 400
    And the response should have error code 42201

  Scenario: Normalize defaults to AVRO when schemaType omitted
    When I POST "/schemas/normalize" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "schema_type" should be "AVRO"

  Scenario: Normalize JSON Schema
    When I POST "/schemas/normalize" with body:
      """
      {"schema": "{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"integer\"}}}", "schemaType": "JSON"}
      """
    Then the response status should be 200
    And the response should have field "canonical"
    And the response field "schema_type" should be "JSON"
