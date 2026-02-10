@operational @postgres
Feature: PostgreSQL Backend Resilience
  As an operator, I want the registry to survive database failures and restarts

  Background:
    Given a running schema registry with postgres backend
    And I have registered schemas under subjects "users-value" and "orders-value"

  Scenario: Data persists across registry restart
    When I restart the schema registry
    And I wait for the registry to become healthy
    Then I get version 1 of subject "users-value"
    And the response status should be 200

  Scenario: Health check fails when database is killed
    When I kill the database container
    And I wait 5 seconds
    When I GET "/"
    Then the response status should be 200

  Scenario: Registry recovers after database restart
    When I kill the database container
    And I wait 5 seconds
    And I restart the database container
    And I wait for the registry to become healthy
    And I wait 10 seconds
    Then I get version 1 of subject "users-value"
    And the response status should be 200
    When I register a schema under subject "recovery-test":
      """
      {"type":"record","name":"RecoveryTest","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Operations fail gracefully during database pause
    When I pause the database
    And I wait 3 seconds
    When I register a schema under subject "pause-test":
      """
      {"type":"record","name":"PauseTest","fields":[{"name":"f","type":"string"}]}
      """
    When I unpause the database
    And I wait 5 seconds
    Then I register a schema under subject "unpause-test":
      """
      {"type":"record","name":"UnpauseTest","fields":[{"name":"f","type":"string"}]}
      """
    And the response status should be 200

  Scenario: Schema IDs remain consistent after recovery
    When I register a schema under subject "before-kill":
      """
      {"type":"record","name":"BeforeKill","fields":[{"name":"f","type":"string"}]}
      """
    And I store the response field "id" as "before_id"
    When I kill the database container
    And I restart the database container
    And I wait for the registry to become healthy
    And I wait 10 seconds
    Then I get the stored schema by ID
    And the response status should be 200
