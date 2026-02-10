@functional
Feature: Configuration
  As an operator, I want to manage compatibility levels and modes

  Scenario: Get default global compatibility
    When I get the global config
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: Set global compatibility
    When I set the global config to "FULL"
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "FULL"

  Scenario: Set per-subject compatibility
    When I set the config for subject "my-subject" to "NONE"
    Then the response status should be 200
    When I get the config for subject "my-subject"
    Then the response field "compatibilityLevel" should be "NONE"

  Scenario: Delete per-subject compatibility falls back to global
    Given subject "my-subject" has compatibility level "FORWARD"
    When I delete the config for subject "my-subject"
    Then the response status should be 200
    When I get the config for subject "my-subject"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: Delete global config reverts to default
    When I set the global config to "FULL_TRANSITIVE"
    Then the response status should be 200
    When I delete the global config
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: Invalid compatibility level returns 422
    When I set the global config to "INVALID_LEVEL"
    Then the response status should be 422
    And the response should have error code 42203

  Scenario: Set all valid compatibility levels
    When I set the global config to "NONE"
    Then the response status should be 200
    When I set the global config to "BACKWARD"
    Then the response status should be 200
    When I set the global config to "BACKWARD_TRANSITIVE"
    Then the response status should be 200
    When I set the global config to "FORWARD"
    Then the response status should be 200
    When I set the global config to "FORWARD_TRANSITIVE"
    Then the response status should be 200
    When I set the global config to "FULL"
    Then the response status should be 200
    When I set the global config to "FULL_TRANSITIVE"
    Then the response status should be 200

  Scenario: Get default global mode
    When I get the global mode
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  Scenario: Set global mode
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READONLY"
