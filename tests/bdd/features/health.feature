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

  @axonops-only
  Scenario: Liveness endpoint returns 200
    When I GET "/health/live"
    Then the response status should be 200
    And the response field "status" should be "UP"

  @axonops-only
  Scenario: Readiness endpoint returns 200 when healthy
    When I GET "/health/ready"
    Then the response status should be 200
    And the response field "status" should be "UP"

  @axonops-only
  Scenario: Startup endpoint returns 200 when healthy
    When I GET "/health/startup"
    Then the response status should be 200
    And the response field "status" should be "UP"

  Scenario: Legacy health check still works
    When I GET "/"
    Then the response status should be 200
