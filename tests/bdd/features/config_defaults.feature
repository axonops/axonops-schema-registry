@functional
Feature: Config and Mode defaultToGlobal Parameter
  Confluent Schema Registry supports a ?defaultToGlobal parameter on
  GET /config/{subject} and GET /mode/{subject}. When defaultToGlobal=true,
  the endpoint falls back to the global config/mode if no subject-specific
  value is set. When defaultToGlobal is false (default), it returns 404.

  # ==========================================================================
  # CONFIG defaultToGlobal
  # ==========================================================================

  Scenario: GET /config/{subject} without defaultToGlobal returns 404 when no subject config
    When I GET "/config/cfg-default-test"
    Then the response status should be 404

  Scenario: GET /config/{subject}?defaultToGlobal=true returns global when no subject config
    Given the global compatibility level is "FULL"
    When I GET "/config/cfg-default-test2?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  Scenario: GET /config/{subject}?defaultToGlobal=false returns 404 when no subject config
    When I GET "/config/cfg-default-test3?defaultToGlobal=false"
    Then the response status should be 404

  Scenario: GET /config/{subject} with subject config set returns it regardless of defaultToGlobal
    Given subject "cfg-has-config" has compatibility level "FORWARD"
    When I GET "/config/cfg-has-config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    When I GET "/config/cfg-has-config?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    When I GET "/config/cfg-has-config?defaultToGlobal=false"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"

  Scenario: GET /config (global) always returns a value
    When I get the global config
    Then the response status should be 200
    And the response should have field "compatibilityLevel"

  Scenario: DELETE /config resets to BACKWARD default
    When I set the global config to "FULL"
    Then the response status should be 200
    When I delete the global config
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "BACKWARD"

  # ==========================================================================
  # MODE defaultToGlobal
  # ==========================================================================

  Scenario: GET /mode/{subject} without defaultToGlobal returns 404 when no subject mode
    When I GET "/mode/mode-default-test"
    Then the response status should be 404

  Scenario: GET /mode/{subject}?defaultToGlobal=true returns global when no subject mode
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I GET "/mode/mode-default-test2?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"

  Scenario: GET /mode/{subject}?defaultToGlobal=false returns 404 when no subject mode
    When I GET "/mode/mode-default-test3?defaultToGlobal=false"
    Then the response status should be 404

  Scenario: GET /mode/{subject} with subject mode set returns it regardless of defaultToGlobal
    When I set the mode for subject "mode-has-mode" to "IMPORT"
    Then the response status should be 200
    When I GET "/mode/mode-has-mode"
    Then the response status should be 200
    And the response field "mode" should be "IMPORT"
    When I GET "/mode/mode-has-mode?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "mode" should be "IMPORT"

  Scenario: GET /mode (global) always returns a value
    When I get the global mode
    Then the response status should be 200
    And the response should have field "mode"
