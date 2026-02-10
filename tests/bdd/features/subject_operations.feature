@functional
Feature: Subject Operations
  As a developer, I want to manage subjects and their versions

  Scenario: List subjects when empty
    When I list all subjects
    Then the response status should be 200
    And the response should be an array of length 0

  Scenario: List subjects with schemas
    Given subject "orders-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"long"}]}
      """
    And subject "users-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I list all subjects
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "orders-value"
    And the response array should contain "users-value"

  Scenario: List versions of a subject
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And the global compatibility level is "NONE"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    When I list versions of subject "user-value"
    Then the response status should be 200
    And the response should be an array of length 2

  Scenario: Soft-delete a subject
    Given subject "temp-value" has schema:
      """
      {"type":"record","name":"Temp","fields":[{"name":"v","type":"string"}]}
      """
    When I delete subject "temp-value"
    Then the response status should be 200
    When I list all subjects
    Then the response should be an array of length 0

  Scenario: Delete a specific version
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And the global compatibility level is "NONE"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    When I delete version 1 of subject "user-value"
    Then the response status should be 200
    When I get the latest version of subject "user-value"
    Then the response field "version" should be 2

  Scenario: Delete non-existent subject returns 404
    When I delete subject "ghost"
    Then the response status should be 404
    And the response should have error code 40401
