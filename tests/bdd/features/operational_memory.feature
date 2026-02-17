@operational @memory
Feature: Memory Backend Operational Behaviour
  As an operator, I want to understand the behaviour of the memory backend during restarts
  and verify the registry handles stop/start/kill/pause operations correctly

  Background:
    Given a running schema registry with memory backend

  Scenario: Data is lost on registry restart (expected)
    Given I have registered 5 schemas across multiple subjects
    When I restart the schema registry
    And I wait for the registry to become healthy
    Then I list all subjects
    And the response should be an array of length 0

  Scenario: Registry assigns IDs from 1 after restart
    Given I have registered schemas under subjects "users-value" and "orders-value"
    When I restart the schema registry
    And I wait for the registry to become healthy
    And I register a schema under subject "post-restart":
      """
      {"type":"record","name":"PostRestart","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should be 1

  Scenario: Stop and start the registry
    When I stop the schema registry
    And I wait for the registry to become unhealthy
    And I start the schema registry
    And I wait for the registry to become healthy
    Then I GET "/"
    And the response status should be 200

  Scenario: Registry becomes unhealthy when stopped
    When I stop the schema registry
    And I wait for the registry to become unhealthy
    Then I start the schema registry
    And I wait for the registry to become healthy

  Scenario: Data is lost after stop and start (expected)
    Given I have registered 5 schemas across multiple subjects
    When I stop the schema registry
    And I start the schema registry
    And I wait for the registry to become healthy
    Then I list all subjects
    And the response should be an array of length 0

  Scenario: Recovery from SIGKILL
    Given I have registered schemas under subjects "users-value" and "orders-value"
    When I kill the database container
    And I start the schema registry
    And I wait for the registry to become healthy
    Then I list all subjects
    And the response should be an array of length 0

  Scenario: IDs reset from 1 after SIGKILL recovery
    Given I have registered schemas under subjects "users-value" and "orders-value"
    When I kill the database container
    And I start the schema registry
    And I wait for the registry to become healthy
    And I register a schema under subject "after-kill":
      """
      {"type":"record","name":"AfterKill","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should be 1

  Scenario: Pause freezes the registry and unpause resumes it
    Given I have registered schemas under subjects "users-value" and "orders-value"
    When I pause the database
    And I wait for the registry to become unhealthy
    And I unpause the database
    And I wait for the registry to become healthy
    Then I get version 1 of subject "users-value"
    And the response status should be 200

  Scenario: Data is preserved after pause and unpause
    Given I have registered 5 schemas across multiple subjects
    When I pause the database
    And I wait for the registry to become unhealthy
    And I unpause the database
    And I wait for the registry to become healthy
    Then I list all subjects
    And the response should be an array of length 5

  Scenario: Configuration resets to default after restart
    When I set the global config to "FULL"
    And the response status should be 200
    And I restart the schema registry
    And I wait for the registry to become healthy
    And I GET "/config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: Mode resets to default after restart
    When I set the global mode to "READONLY"
    And the response status should be 200
    And I restart the schema registry
    And I wait for the registry to become healthy
    And I GET "/mode"
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  Scenario: Register new schemas after restart
    Given I have registered 5 schemas across multiple subjects
    When I restart the schema registry
    And I wait for the registry to become healthy
    And I register a schema under subject "fresh-after-restart":
      """
      {"type":"record","name":"FreshAfterRestart","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    When I list all subjects
    Then the response should be an array of length 1

  Scenario: Multiple restart cycles work correctly
    When I register a schema under subject "cycle-1":
      """
      {"type":"record","name":"Cycle1","fields":[{"name":"f","type":"string"}]}
      """
    And the response status should be 200
    And I restart the schema registry
    And I wait for the registry to become healthy
    And I register a schema under subject "cycle-2":
      """
      {"type":"record","name":"Cycle2","fields":[{"name":"f","type":"string"}]}
      """
    And the response status should be 200
    And I restart the schema registry
    And I wait for the registry to become healthy
    Then I list all subjects
    And the response should be an array of length 0
    When I register a schema under subject "cycle-3":
      """
      {"type":"record","name":"Cycle3","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should be 1
