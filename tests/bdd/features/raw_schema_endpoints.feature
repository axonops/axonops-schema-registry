@functional
Feature: Raw Schema Endpoints
  Test the endpoints that return raw schema strings without JSON metadata wrapping:
  GET /schemas/ids/{id}/schema and GET /subjects/{subject}/versions/{version}/schema
  Also tests GET /subjects/{subject}/versions/{version}/referencedby edge cases.

  # ==========================================================================
  # GET /schemas/ids/{id}/schema — RAW SCHEMA BY GLOBAL ID
  # ==========================================================================

  Scenario: GET raw Avro schema by ID returns valid JSON string
    Given subject "raw-avro" has schema:
      """
      {"type":"record","name":"RawAvro","fields":[{"name":"id","type":"string"}]}
      """
    When I get the raw schema by ID 1
    Then the response status should be 200
    And the response should contain "RawAvro"

  Scenario: GET raw Protobuf schema by ID returns proto text
    Given subject "raw-proto" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message RawProto {
        string id = 1;
      }
      """
    When I get the latest version of subject "raw-proto"
    And I store the response field "id" as "proto_id"
    When I GET "/schemas/ids/1/schema"
    Then the response status should be 200

  Scenario: GET raw JSON Schema by ID returns valid JSON string
    Given subject "raw-json" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}
      """
    When I get the latest version of subject "raw-json"
    And I store the response field "id" as "json_id"
    When I GET "/schemas/ids/1/schema"
    Then the response status should be 200

  Scenario: GET raw schema for non-existent ID returns 404
    When I GET "/schemas/ids/99999/schema"
    Then the response status should be 404
    And the response should have error code 40403

  # ==========================================================================
  # GET /subjects/{subject}/versions/{version}/schema — RAW SCHEMA BY VERSION
  # ==========================================================================

  Scenario: GET raw schema by subject version returns schema string
    Given subject "raw-ver-avro" has schema:
      """
      {"type":"record","name":"RawVerAvro","fields":[{"name":"x","type":"string"}]}
      """
    When I get the raw schema for subject "raw-ver-avro" version 1
    Then the response status should be 200
    And the response should contain "RawVerAvro"

  Scenario: GET raw schema for latest version works
    Given the global compatibility level is "NONE"
    And subject "raw-ver-latest" has schema:
      """
      {"type":"record","name":"Old","fields":[{"name":"a","type":"string"}]}
      """
    And subject "raw-ver-latest" has schema:
      """
      {"type":"record","name":"New","fields":[{"name":"b","type":"string"}]}
      """
    When I GET "/subjects/raw-ver-latest/versions/latest/schema"
    Then the response status should be 200
    And the response should contain "New"

  Scenario: GET raw schema for non-existent version returns 404
    Given subject "raw-ver-404" has schema:
      """
      {"type":"record","name":"Exists","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/raw-ver-404/versions/99/schema"
    Then the response status should be 404

  Scenario: GET raw schema for non-existent subject returns 404
    When I GET "/subjects/totally-missing-subject/versions/1/schema"
    Then the response status should be 404

  # ==========================================================================
  # GET /subjects/{subject}/versions/{version}/referencedby — EDGE CASES
  # ==========================================================================

  Scenario: referencedby with no references returns empty array
    Given subject "refby-empty" has schema:
      """
      {"type":"record","name":"NoRefs","fields":[{"name":"a","type":"string"}]}
      """
    When I get the referenced by for subject "refby-empty" version 1
    Then the response status should be 200
    And the response should be an array of length 0

  Scenario: referencedby with one reference returns array with one ID
    Given the global compatibility level is "NONE"
    And subject "refby-base" has schema:
      """
      {"type":"record","name":"Base","namespace":"com.refby","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "refby-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.refby\",\"fields\":[{\"name\":\"base\",\"type\":\"com.refby.Base\"}]}",
        "references": [
          {"name": "com.refby.Base", "subject": "refby-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "refby-base" version 1
    Then the response status should be 200
    And the response should be an array of length 1

  Scenario: referencedby for non-existent subject returns 404
    When I GET "/subjects/no-such-refby-subject/versions/1/referencedby"
    Then the response status should be 404

  Scenario: referencedby for non-existent version returns 404
    Given subject "refby-exists" has schema:
      """
      {"type":"record","name":"RefbyExists","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/refby-exists/versions/99/referencedby"
    Then the response status should be 404
