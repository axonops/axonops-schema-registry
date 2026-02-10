@operational @cassandra
Feature: Cassandra Backend Resilience
  As an operator, I want the registry to survive Cassandra failures and restarts

  Background:
    Given a running schema registry with cassandra backend
    And I have registered schemas under subjects "users-value" and "orders-value"

  Scenario: Data persists across registry restart
    When I restart the schema registry
    And I wait for the registry to become healthy
    Then I get version 1 of subject "users-value"
    And the response status should be 200

  Scenario: Registry recovers after database restart
    When I kill the database container
    And I wait 10 seconds
    And I restart the database container
    And I wait 30 seconds
    And I wait for the registry to become healthy
    Then I get version 1 of subject "users-value"
    And the response status should be 200
    When I register a schema under subject "recovery-test":
      """
      {"type":"record","name":"RecoveryTest","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Operations fail gracefully during database pause
    When I pause the database
    And I wait 5 seconds
    When I register a schema under subject "pause-test":
      """
      {"type":"record","name":"PauseTest","fields":[{"name":"f","type":"string"}]}
      """
    When I unpause the database
    And I wait 10 seconds
    Then I register a schema under subject "unpause-test":
      """
      {"type":"record","name":"UnpauseTest","fields":[{"name":"f","type":"string"}]}
      """
    And the response status should be 200
