@functional @smoke
Feature: Health and Metadata
  As an operator, I want to verify the registry is running and inspect metadata

  Scenario: Health check returns 200
    When I GET "/"
    Then the response status should be 200

  @axonops-only
  Scenario: Liveness endpoint returns UP
    When I GET "/health/live"
    Then the response status should be 200
    And the response field "status" should be "UP"

  @axonops-only
  Scenario: Readiness endpoint returns UP when healthy
    When I GET "/health/ready"
    Then the response status should be 200
    And the response field "status" should be "UP"

  @axonops-only
  Scenario: Startup endpoint returns UP when healthy
    When I GET "/health/startup"
    Then the response status should be 200
    And the response field "status" should be "UP"

  Scenario: Schema types endpoint returns supported types
    When I get the schema types
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain "AVRO"
    And the response array should contain "PROTOBUF"
    And the response array should contain "JSON"

  Scenario: Cluster ID endpoint
    When I get the cluster ID
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Server version endpoint
    When I get the server version
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Contexts endpoint
    When I get the contexts
    Then the response status should be 200
