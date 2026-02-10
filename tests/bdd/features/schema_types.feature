@functional
Feature: Schema Types
  The registry supports Avro, Protobuf, and JSON Schema types

  Scenario: Register Avro schema
    When I register a schema under subject "avro-test":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Register Protobuf schema
    When I register a "PROTOBUF" schema under subject "proto-test":
      """
      syntax = "proto3";
      message Event {
        string id = 1;
      }
      """
    Then the response status should be 200

  Scenario: Register JSON Schema
    When I register a "JSON" schema under subject "json-test":
      """
      {"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200

  Scenario: Get schema by ID shows schemaType for Protobuf
    When I register a "PROTOBUF" schema under subject "proto-type-test":
      """
      syntax = "proto3";
      message Msg {
        string name = 1;
      }
      """
    And I store the response field "id" as "schema_id"
    And I get the stored schema by ID
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"
