@functional
Feature: API Error Codes
  As a developer, I want the registry to return proper Confluent-compatible error codes

  # --- 40401: Subject not found ---

  Scenario: Get versions of non-existent subject returns 40401
    When I list versions of subject "nonexistent"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Get specific version of non-existent subject returns 40401
    When I get version 1 of subject "nonexistent"
    Then the response status should be 404

  Scenario: Delete non-existent subject returns 40401
    When I delete subject "nonexistent"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Get config of non-existent subject falls back to global
    When I get the config for subject "nonexistent"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  # --- 40402: Version not found ---

  Scenario: Get non-existent version returns 40402
    Given subject "version-err" has schema:
      """
      {"type":"record","name":"Test","fields":[{"name":"f","type":"string"}]}
      """
    When I get version 99 of subject "version-err"
    Then the response status should be 404
    And the response should have error code 40402

  Scenario: Delete non-existent version returns 40402
    Given subject "version-err-del" has schema:
      """
      {"type":"record","name":"Test","fields":[{"name":"f","type":"string"}]}
      """
    When I delete version 99 of subject "version-err-del"
    Then the response status should be 404
    And the response should have error code 40402

  # --- 40403: Schema not found ---

  Scenario: Get non-existent schema ID returns 40403
    When I get schema by ID 999999
    Then the response status should be 404
    And the response should have error code 40403

  Scenario: Get subjects for non-existent schema ID returns 40403
    When I get the subjects for schema ID 999999
    Then the response status should be 404
    And the response should have error code 40403

  # --- 42201: Invalid schema ---

  Scenario: Register invalid Avro schema returns 42201
    When I register a schema under subject "invalid-avro":
      """
      {"this is not valid avro"}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register invalid Protobuf schema returns 42201
    When I register a "PROTOBUF" schema under subject "invalid-proto":
      """
      this is not valid protobuf
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register invalid JSON Schema returns 42201
    When I register a "JSON" schema under subject "invalid-json":
      """
      {not valid json at all
      """
    Then the response status should be 422
    And the response should have error code 42201

  # --- 42203: Invalid compatibility level ---

  Scenario: Set invalid global compatibility level returns 42203
    When I set the global config to "INVALID_LEVEL"
    Then the response status should be 422
    And the response should have error code 42203

  Scenario: Set invalid per-subject compatibility level returns 42203
    When I set the config for subject "some-subject" to "NOT_A_LEVEL"
    Then the response status should be 422
    And the response should have error code 42203

  # --- 409: Incompatible schema ---

  Scenario: Register incompatible schema returns 409
    Given subject "compat-err" has schema:
      """
      {"type":"record","name":"Test","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "compat-err":
      """
      {"type":"record","name":"Test","fields":[{"name":"name","type":"int"}]}
      """
    Then the response status should be 409

  # --- Lookup not found ---

  Scenario: Lookup non-existent schema returns 404
    Given subject "lookup-err" has schema:
      """
      {"type":"record","name":"Test","fields":[{"name":"f","type":"string"}]}
      """
    When I lookup schema in subject "lookup-err":
      """
      {"type":"record","name":"Other","fields":[{"name":"g","type":"int"}]}
      """
    Then the response status should be 404

  # --- Empty request body ---

  Scenario: Register with empty schema string returns error
    When I register a schema under subject "empty-schema":
      """

      """
    Then the response status should be 400
