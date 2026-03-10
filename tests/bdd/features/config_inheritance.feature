@functional @edge-case
Feature: Config Inheritance and Fallback
  As a schema registry administrator
  I want subject-level config to fall back to global config when deleted
  And I want to verify the interaction between subject and global compatibility levels
  So that configuration inheritance works predictably

  Background:
    Given the schema registry is running

  # ---------------------------------------------------------------------------
  # Subject config falls back to global when deleted
  # ---------------------------------------------------------------------------

  Scenario: Subject config falls back to global after subject config deletion
    # Set global to FULL
    Given I set the global config to "FULL"
    # Set subject-level to NONE (overrides global)
    And I set the config for subject "inherit-test" to "NONE"
    # Verify subject config is NONE
    When I get the config for subject "inherit-test"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    # Delete subject config — should fall back to global FULL
    When I delete the config for subject "inherit-test"
    Then the response status should be 200
    # Query subject config returns 404 (no subject-level config)
    When I get the config for subject "inherit-test"
    Then the response status should be 404
    And the audit log should contain event "config_delete" with subject "inherit-test"

  Scenario: Subject config deletion causes registration to use global level
    # Set global to BACKWARD, subject to NONE
    Given I set the global config to "BACKWARD"
    And I set the config for subject "inherit-func" to "NONE"
    # Register v1 under NONE — any schema is accepted
    And subject "inherit-func" has schema:
      """
      {"type":"record","name":"InheritFunc","fields":[{"name":"id","type":"int"}]}
      """
    # Register v2 with backward-incompatible change — NONE allows it
    And subject "inherit-func" has schema:
      """
      {"type":"record","name":"InheritFunc","fields":[{"name":"name","type":"string"}]}
      """
    # Delete subject config — now registrations use global BACKWARD
    When I delete the config for subject "inherit-func"
    Then the response status should be 200
    # Attempt v3 with backward-incompatible change — should be rejected by global BACKWARD
    When I register a schema under subject "inherit-func":
      """
      {"type":"record","name":"InheritFunc","fields":[{"name":"totally_new","type":"long"}]}
      """
    Then the response status should be 409
    And the audit log should contain event "config_delete" with subject "inherit-func"

  # ---------------------------------------------------------------------------
  # Subject config overrides global
  # ---------------------------------------------------------------------------

  Scenario: Subject config overrides global config
    Given I set the global config to "BACKWARD_TRANSITIVE"
    And I set the config for subject "override-test" to "NONE"
    When I get the config for subject "override-test"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    # Changing global does not affect subject
    When I set the global config to "FULL"
    And I get the config for subject "override-test"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    And the audit log should contain event "config_update"

  # ---------------------------------------------------------------------------
  # Multiple subjects with independent configs
  # ---------------------------------------------------------------------------

  Scenario: Multiple subjects have independent compatibility levels
    Given I set the global config to "BACKWARD"
    And I set the config for subject "subj-a" to "NONE"
    And I set the config for subject "subj-b" to "FORWARD"
    # Verify each subject has its own config
    When I get the config for subject "subj-a"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    When I get the config for subject "subj-b"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    # Delete subj-a config — now returns 404 (no subject-level config)
    When I delete the config for subject "subj-a"
    Then the response status should be 200
    When I get the config for subject "subj-a"
    Then the response status should be 404
    # subj-b is still FORWARD
    When I get the config for subject "subj-b"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    And the audit log should contain event "config_delete" with subject "subj-a"

  # ---------------------------------------------------------------------------
  # Global config set / get / delete cycle
  # ---------------------------------------------------------------------------

  Scenario: Global config cycle — set, verify, delete, verify default
    # Set global to FULL_TRANSITIVE
    When I set the global config to "FULL_TRANSITIVE"
    And I get the global config
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL_TRANSITIVE"
    # Delete global config
    When I delete the global config
    Then the response status should be 200
    # Global should revert to BACKWARD
    When I get the global config
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"
    And the audit log should contain event "config_delete"

  # ---------------------------------------------------------------------------
  # All 7 compatibility levels can be set per-subject
  # ---------------------------------------------------------------------------

  Scenario Outline: Set and verify each compatibility level at subject scope
    Given I set the config for subject "compat-levels" to "<level>"
    When I get the config for subject "compat-levels"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "<level>"

    Examples:
      | level                 |
      | NONE                  |
      | BACKWARD              |
      | BACKWARD_TRANSITIVE   |
      | FORWARD               |
      | FORWARD_TRANSITIVE    |
      | FULL                  |
      | FULL_TRANSITIVE       |
