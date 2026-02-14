@functional
Feature: Compatibility Configuration & Testing — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive compatibility configuration and testing scenarios from the
  Confluent Schema Registry v8.1.1 test suite.

  # ==========================================================================
  # GLOBAL COMPATIBILITY CONFIGURATION
  # ==========================================================================

  Scenario: Get default global compatibility
    When I get the global config
    Then the response status should be 200
    And the response should have field "compatibilityLevel"

  Scenario: Set and get global compatibility level
    When I set the global config to "FORWARD"
    Then the response status should be 200
    When I get the global config
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    # Reset
    When I set the global config to "NONE"

  Scenario: Set subject-level compatibility independent of global
    Given I set the global config to "NONE"
    When I set the config for subject "cc-subj-level" to "FORWARD"
    Then the response status should be 200
    When I get the config for subject "cc-subj-level"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    When I get the global config
    And the response field "compatibilityLevel" should be "NONE"

  Scenario: Set compatibility for non-existent subject succeeds
    When I set the config for subject "cc-nonexist-subj" to "FULL"
    Then the response status should be 200
    When I get the config for subject "cc-nonexist-subj"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  Scenario: Get config for subject with no subject-level config returns 404
    When I get the config for subject "cc-no-config-at-all"
    Then the response status should be 404
    And the response should have error code 40408

  Scenario: Delete subject-level config reverts to global
    Given I set the global config to "FULL"
    And I set the config for subject "cc-del-config" to "BACKWARD"
    When I delete the config for subject "cc-del-config"
    Then the response status should be 200
    When I get the config for subject "cc-del-config"
    Then the response status should be 404
    And the response should have error code 40408
    # With defaultToGlobal the fallback works
    When I GET "/config/cc-del-config?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    # Reset
    When I set the global config to "NONE"

  # ==========================================================================
  # COMPATIBILITY TESTING ENDPOINTS
  # ==========================================================================

  Scenario: Test compatibility against non-existent version returns 404
    Given subject "cc-test-nover" has schema:
      """
      {"type":"record","name":"TestNV","fields":[{"name":"a","type":"string"}]}
      """
    When I check compatibility of schema against subject "cc-test-nover" version 100:
      """
      {"type":"record","name":"TestNV","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 404
    And the response should have error code 40402

  Scenario: Test compatibility against invalid version returns 422
    Given subject "cc-test-invver" has schema:
      """
      {"type":"record","name":"TestIV","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/cc-test-invver/versions/earliest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestIV\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 422

  Scenario: Backward compatibility — adding field with default is compatible
    Given subject "cc-back-compat" has compatibility level "BACKWARD"
    And subject "cc-back-compat" has schema:
      """
      {"type":"record","name":"Back","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "cc-back-compat":
      """
      {"type":"record","name":"Back","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the compatibility check should be compatible

  Scenario: Backward compatibility — adding field without default is incompatible
    Given subject "cc-back-incompat" has compatibility level "BACKWARD"
    And subject "cc-back-incompat" has schema:
      """
      {"type":"record","name":"BackI","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "cc-back-incompat":
      """
      {"type":"record","name":"BackI","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: Changing compatibility to NONE allows incompatible registration
    Given subject "cc-change-none" has compatibility level "BACKWARD"
    And subject "cc-change-none" has schema:
      """
      {"type":"record","name":"ChgNone","fields":[{"name":"f1","type":"string"}]}
      """
    # Incompatible schema rejected under BACKWARD
    When I register a schema under subject "cc-change-none":
      """
      {"type":"record","name":"ChgNone","fields":[{"name":"f1","type":"int"}]}
      """
    Then the response status should be 409
    # Change to NONE and retry
    When I set the config for subject "cc-change-none" to "NONE"
    And I register a schema under subject "cc-change-none":
      """
      {"type":"record","name":"ChgNone","fields":[{"name":"f1","type":"int"}]}
      """
    Then the response status should be 200

  Scenario: Change compatibility from FORWARD to BACKWARD enforces new rules
    Given the global compatibility level is "NONE"
    And subject "cc-fwd-to-bwd" has compatibility level "FORWARD"
    And subject "cc-fwd-to-bwd" has schema:
      """
      {"type":"record","name":"FwdBwd","fields":[{"name":"f1","type":"string"}]}
      """
    # Forward-compatible: removing a field (old can read new data)
    When I register a schema under subject "cc-fwd-to-bwd":
      """
      {"type":"record","name":"FwdBwd","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the response status should be 200
    # Switch to BACKWARD
    When I set the config for subject "cc-fwd-to-bwd" to "BACKWARD"
    # Backward-compatible: adding field WITH default succeeds
    When I register a schema under subject "cc-fwd-to-bwd":
      """
      {"type":"record","name":"FwdBwd","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"},{"name":"f3","type":"string","default":"y"}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # TRANSITIVE COMPATIBILITY VIA REST
  # ==========================================================================

  Scenario: FORWARD_TRANSITIVE — compatible with latest but not all versions
    Given the global compatibility level is "NONE"
    And subject "cc-ft-rest" has schema:
      """
      {"type":"record","name":"FTRest","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    And subject "cc-ft-rest" has schema:
      """
      {"type":"record","name":"FTRest","fields":[{"name":"f1","type":"string"}]}
      """
    When I set the config for subject "cc-ft-rest" to "FORWARD_TRANSITIVE"
    # v3 adds f3 — compatible with v2 (latest, only f1) but v1 can't read without f2 (no default)
    When I check compatibility of schema against subject "cc-ft-rest":
      """
      {"type":"record","name":"FTRest","fields":[{"name":"f1","type":"string"},{"name":"f3","type":"string"}]}
      """
    Then the compatibility check should be compatible
    When I check compatibility of schema against all versions of subject "cc-ft-rest":
      """
      {"type":"record","name":"FTRest","fields":[{"name":"f1","type":"string"},{"name":"f3","type":"string"}]}
      """
    Then the compatibility check should be incompatible

  # ==========================================================================
  # defaultToGlobal PARAMETER
  # ==========================================================================

  Scenario: defaultToGlobal returns global config when no subject config exists
    Given I set the global config to "BACKWARD_TRANSITIVE"
    When I GET "/config/cc-dtg-test?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD_TRANSITIVE"
    When I GET "/config/cc-dtg-test"
    Then the response status should be 404
    And the response should have error code 40408
    # Reset
    When I set the global config to "NONE"

  Scenario: defaultToGlobal returns subject config when it exists
    Given I set the global config to "NONE"
    And I set the config for subject "cc-dtg-subj" to "FULL_TRANSITIVE"
    When I GET "/config/cc-dtg-subj?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL_TRANSITIVE"
