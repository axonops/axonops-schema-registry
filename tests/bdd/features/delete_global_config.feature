@functional
Feature: Delete Global Compatibility Configuration
  As a schema registry administrator
  I want to reset the global compatibility level to the default
  So that I can revert to standard compatibility checks after testing or configuration changes

  Background:
    Given the schema registry is running

  Scenario: DELETE /config resets global compatibility to BACKWARD
    Given I set the global config to "FULL"
    And I get the global config
    And the response field "compatibilityLevel" should be "FULL"
    When I DELETE "/config"
    Then the response status should be 200
    When I get the global config
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: DELETE /config response contains previous compatibility level
    Given I set the global config to "FORWARD"
    When I DELETE "/config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"

  Scenario: Subject-level config NOT affected by global reset
    Given I set the global config to "FULL"
    And I set the config for subject "test-subject" to "NONE"
    And I get the config for subject "test-subject"
    And the response field "compatibilityLevel" should be "NONE"
    When I DELETE "/config"
    Then the response status should be 200
    When I get the config for subject "test-subject"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"

  Scenario: DELETE /config when already BACKWARD is idempotent
    Given I get the global config
    And the response field "compatibilityLevel" should be "BACKWARD"
    When I DELETE "/config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"
    When I DELETE "/config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: DELETE /config after setting advanced configuration
    Given I PUT "/config" with body:
      """
      {
        "compatibility": "FULL_TRANSITIVE",
        "normalize": true
      }
      """
    And the response status should be 200
    When I DELETE "/config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL_TRANSITIVE"
    When I get the global config
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"
