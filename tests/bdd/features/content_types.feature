@functional
Feature: Content-Type Header Handling
  Verify the schema registry uses the correct Content-Type in responses
  and accepts both standard JSON and schema registry content types.

  Scenario: Response Content-Type contains schema registry type
    When I get the global config
    Then the response status should be 200
    And the response header "Content-Type" should contain "application/vnd.schemaregistry.v1+json"

  Scenario: Schema registration response has correct Content-Type
    When I register a schema under subject "ct-reg":
      """
      {"type":"record","name":"CT","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    And the response header "Content-Type" should contain "application/vnd.schemaregistry.v1+json"

  Scenario: Error response has correct Content-Type
    When I GET "/subjects/ct-no-such-subject/versions/1"
    Then the response status should be 404
    And the response header "Content-Type" should contain "application/vnd.schemaregistry.v1+json"
