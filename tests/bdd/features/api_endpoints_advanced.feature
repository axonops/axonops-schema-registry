@functional
Feature: API Endpoint Behaviors Advanced
  As a developer, I want to verify API response shapes, defaults, and edge cases
  for schema registration, retrieval, and metadata endpoints

  # --- Schema type defaults to AVRO when not specified ---

  Scenario: Schema type defaults to AVRO when not specified
    When I register a schema under subject "default-type-test":
      """
      {"type":"record","name":"DefaultType","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get version 1 of subject "default-type-test"
    Then the response status should be 200
    And the response should not have field "schemaType"

  # --- Schema type accepted in any case ---

  Scenario: Schema type is accepted in any case
    When I POST "/subjects/case-type-lower/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CaseLower\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}", "schemaType": "avro"}
      """
    Then the response status should be 200
    When I get version 1 of subject "case-type-lower"
    Then the response status should be 200
    And the response should not have field "schemaType"

  # --- GET /schemas/types returns all three types ---

  Scenario: GET /schemas/types returns all three schema types
    When I get the schema types
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain "AVRO"
    And the response array should contain "PROTOBUF"
    And the response array should contain "JSON"

  # --- GET version "latest" returns actual version number ---

  Scenario: GET version latest returns actual version number in response
    Given the global compatibility level is "NONE"
    And subject "latest-ver-test" has schema:
      """
      {"type":"record","name":"LatestV1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "latest-ver-test" has schema:
      """
      {"type":"record","name":"LatestV2","fields":[{"name":"b","type":"string"}]}
      """
    And subject "latest-ver-test" has schema:
      """
      {"type":"record","name":"LatestV3","fields":[{"name":"c","type":"string"}]}
      """
    When I get the latest version of subject "latest-ver-test"
    Then the response status should be 200
    And the response field "version" should be 3
    And the response field "subject" should be "latest-ver-test"
    And the response should have field "id"

  # --- References field omitted when no references ---

  Scenario: References field omitted from response when schema has no references
    When I register a schema under subject "no-refs-test":
      """
      {"type":"record","name":"NoRefs","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response body should not contain "references"

  # --- References field present when schema has references ---

  Scenario: References field present when schema has references
    Given the global compatibility level is "NONE"
    And subject "ref-base" has schema:
      """
      {"type":"record","name":"Address","namespace":"com.test","fields":[{"name":"street","type":"string"},{"name":"city","type":"string"}]}
      """
    When I register a schema under subject "ref-parent" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"address\",\"type\":\"com.test.Address\"}]}",
        "references": [
          {"name": "com.test.Address", "subject": "ref-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response body should contain "references"

  # --- Raw schema by ID returns schema text, not JSON wrapper ---

  Scenario: Raw schema by ID returns schema text not JSON wrapper
    When I register a schema under subject "raw-by-id-test":
      """
      {"type":"record","name":"RawTest","fields":[{"name":"x","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the raw schema by ID {{schema_id}}
    Then the response status should be 200
    And the response body should contain "RawTest"
    And the response body should not contain "schemaType"

  # --- Raw schema by subject/version returns schema text ---

  Scenario: Raw schema by subject and version returns schema text
    Given subject "raw-ver-test" has schema:
      """
      {"type":"record","name":"RawVer","fields":[{"name":"y","type":"long"}]}
      """
    When I get the raw schema for subject "raw-ver-test" version 1
    Then the response status should be 200
    And the response body should contain "RawVer"
    And the response body should not contain "schemaType"
    And the response body should not contain "subject"

  # --- Error response has both error_code and message fields ---

  Scenario: Error response has both error_code and message fields
    When I get schema by ID 999999
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have field "error_code"
    And the response should have field "message"
    And the response should have error code 40403

  # --- Invalid schema type returns 422 ---

  Scenario: Invalid schema type returns 422
    When I POST "/subjects/invalid-type-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "schemaType": "XML"}
      """
    Then the response status should be 422
    And the response should have error code 42202
    And the response should have field "message"
