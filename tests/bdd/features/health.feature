@functional @smoke
Feature: Health Check
  The schema registry should respond to health checks

  Scenario: Health endpoint returns 200
    Given the schema registry is running
    Then the response status should be 200

  Scenario: Schema types endpoint returns supported types
    When I get the schema types
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain "AVRO"
    And the response array should contain "PROTOBUF"
    And the response array should contain "JSON"
