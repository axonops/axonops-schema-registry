@functional
Feature: Schema Listing
  As a developer, I want to list and query schemas across subjects

  Scenario: List all schemas
    Given subject "sub-a" has schema:
      """
      {"type":"record","name":"A","fields":[{"name":"x","type":"string"}]}
      """
    And subject "sub-b" has schema:
      """
      {"type":"record","name":"B","fields":[{"name":"y","type":"long"}]}
      """
    When I list all schemas
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Get subjects for a schema ID
    When I register a schema under subject "shared-value":
      """
      {"type":"record","name":"Shared","fields":[{"name":"id","type":"long"}]}
      """
    And I store the response field "id" as "schema_id"
    And I get the subjects for the stored schema ID
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain "shared-value"
