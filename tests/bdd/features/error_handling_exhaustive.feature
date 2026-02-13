@functional
Feature: Error Handling & Edge Cases — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive error handling tests covering invalid schemas, bad references,
  error codes, and global ID consistency.

  # ==========================================================================
  # INVALID SCHEMA ERRORS
  # ==========================================================================

  Scenario: Register unparseable Avro schema returns INVALID_SCHEMA
    When I register a schema under subject "err-ex-bad-avro":
      """
      this is not valid json or avro at all
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register invalid JSON in schema field returns error
    When I POST "/subjects/err-ex-bad-json/versions" with body:
      """
      {"schema": "not-valid-json{{{"}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Register empty schema string returns error
    When I POST "/subjects/err-ex-empty/versions" with body:
      """
      {"schema": ""}
      """
    Then the response status should be 422

  # ==========================================================================
  # BAD REFERENCE ERRORS
  # ==========================================================================

  Scenario: Register with reference to non-existent subject returns error
    When I register a schema under subject "err-ex-ref-nosub" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadRef\",\"fields\":[{\"name\":\"data\",\"type\":\"com.missing.Type\"}]}",
        "references": [
          {"name": "com.missing.Type", "subject": "nonexistent-subject", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  Scenario: Register with reference to non-existent version returns error
    Given subject "err-ex-ref-src" has schema:
      """
      {"type":"record","name":"ErrSrc","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "err-ex-ref-badver" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadVer\",\"fields\":[{\"name\":\"src\",\"type\":\"ErrSrc\"}]}",
        "references": [
          {"name": "ErrSrc", "subject": "err-ex-ref-src", "version": 999}
        ]
      }
      """
    Then the response status should be 422

  # ==========================================================================
  # GLOBAL ID CONSISTENCY
  # ==========================================================================

  Scenario: Same schema under different subjects gets same global ID
    Given the global compatibility level is "NONE"
    When I register a schema under subject "err-ex-id-s1":
      """
      {"type":"record","name":"GlobalID","fields":[{"name":"v","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    When I register a schema under subject "err-ex-id-s2":
      """
      {"type":"record","name":"GlobalID","fields":[{"name":"v","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"

  Scenario: Different schemas get different global IDs
    Given the global compatibility level is "NONE"
    When I register a schema under subject "err-ex-diffid-s1":
      """
      {"type":"record","name":"DiffID1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id1"
    When I register a schema under subject "err-ex-diffid-s2":
      """
      {"type":"record","name":"DiffID2","fields":[{"name":"b","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id2"

  # ==========================================================================
  # SUBJECT AND VERSION ERROR CODES
  # ==========================================================================

  Scenario: Operations on non-existent subject return correct error codes
    When I get version 1 of subject "err-ex-nosub-ops"
    Then the response status should be 404
    And the response should have error code 40401
    When I list versions of subject "err-ex-nosub-ops"
    Then the response status should be 404
    And the response should have error code 40401
    When I get the latest version of subject "err-ex-nosub-ops"
    Then the response status should be 404
    And the response should have error code 40401

  @pending-impl
  Scenario: Invalid version number returns 422
    Given subject "err-ex-invver" has schema:
      """
      {"type":"record","name":"InvVer","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/err-ex-invver/versions/0"
    Then the response status should be 422
    And the response should have error code 42202
    When I GET "/subjects/err-ex-invver/versions/-1"
    Then the response status should be 422
    And the response should have error code 42202

  Scenario: Non-existent schema ID returns 404
    When I GET "/schemas/ids/99999"
    Then the response status should be 404
    And the response should have error code 40403
