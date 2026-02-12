@functional
Feature: Configuration and Mode Management Advanced
  As an operator, I want to verify configuration isolation, fallback behavior,
  case handling, and edge cases for compatibility levels and modes

  # --- Config isolation ---

  Scenario: Setting subject-A config does not affect subject-B
    When I set the config for subject "subj-a" to "NONE"
    Then the response status should be 200
    When I set the config for subject "subj-b" to "FULL"
    Then the response status should be 200
    When I get the config for subject "subj-a"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    When I get the config for subject "subj-b"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  # --- Subject config overrides global during registration ---

  Scenario: Subject config overrides global during schema registration
    Given the global compatibility level is "BACKWARD"
    And subject "override-test" has compatibility level "NONE"
    And subject "override-test" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "override-test":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  # --- Delete subject config falls back to global ---

  Scenario: Delete subject config causes fallback to global
    Given the global compatibility level is "FULL"
    And subject "fallback-cfg" has compatibility level "NONE"
    When I get the config for subject "fallback-cfg"
    Then the response field "compatibilityLevel" should be "NONE"
    When I delete the config for subject "fallback-cfg"
    Then the response status should be 200
    When I get the config for subject "fallback-cfg"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  # --- Delete global config reverts to BACKWARD default ---

  Scenario: Delete global config reverts to BACKWARD default
    When I set the global config to "FULL_TRANSITIVE"
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "FULL_TRANSITIVE"
    When I delete the global config
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "BACKWARD"

  # --- Delete non-existent subject config returns 404 ---

  Scenario: Delete config for non-existent subject returns 404
    When I delete the config for subject "never-configured-subject"
    Then the response status should be 404
    And the response should have error code 40401

  # --- Invalid compatibility level returns 422 ---

  Scenario: Invalid compatibility level returns 422 with error code 42203
    When I set the global config to "INVALID_LEVEL"
    Then the response status should be 422
    And the response should have error code 42203
    And the response should have field "message"

  # --- Case insensitivity for compatibility levels ---

  Scenario: Compatibility level is case insensitive
    When I PUT "/config" with body:
      """
      {"compatibility": "backward"}
      """
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"
    When I PUT "/config" with body:
      """
      {"compatibility": "Forward_Transitive"}
      """
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD_TRANSITIVE"

  # --- Set and get all 3 valid mode values ---

  Scenario: Set and get all valid mode values
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READONLY"
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "IMPORT"

  # --- Invalid mode value returns 422 ---

  Scenario: Invalid mode value returns 422 with error code 42204
    When I PUT "/mode" with body:
      """
      {"mode": "INVALID_MODE"}
      """
    Then the response status should be 422
    And the response should have error code 42204
    And the response should have field "message"

  # --- Delete non-existent subject mode returns 404 ---

  Scenario: Delete mode for non-existent subject returns 404
    When I delete the mode for subject "never-moded-subject"
    Then the response status should be 404
    And the response should have error code 40401

  # --- Mode fallback to global ---

  Scenario: Subject mode not set falls back to global mode
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the mode for subject "no-mode-set-subject"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"

  # --- Per-subject mode set and retrieved independently from global ---

  Scenario: Per-subject mode is independent from global mode
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I set the mode for subject "independent-mode-a" to "READONLY"
    Then the response status should be 200
    When I set the mode for subject "independent-mode-b" to "IMPORT"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"
    When I get the mode for subject "independent-mode-a"
    Then the response field "mode" should be "READONLY"
    When I get the mode for subject "independent-mode-b"
    Then the response field "mode" should be "IMPORT"
