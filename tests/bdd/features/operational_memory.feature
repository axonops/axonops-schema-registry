@operational @memory
Feature: Memory Backend Operational Behaviour
  As an operator, I want to understand the behaviour of the memory backend during restarts

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
