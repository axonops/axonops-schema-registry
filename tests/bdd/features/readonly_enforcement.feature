@functional
Feature: READONLY Enforcement and Permanent Delete Restrictions
  READONLY and READONLY_OVERRIDE modes block data mutations (schema registration,
  subject deletion, version deletion) but do NOT block config or mode changes.
  Note: Confluent Schema Registry blocks config writes in READONLY mode but
  allows mode changes. AxonOps intentionally allows both config and mode
  changes in READONLY mode (marked @axonops-only where behavior diverges).

  Additionally, permanent delete of version "latest" should not be allowed —
  only explicit numeric version numbers can be permanently deleted.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # READONLY MODE — CONFIG AND MODE CHANGES STILL ALLOWED
  # ==========================================================================

  @axonops-only
  Scenario: READONLY mode allows setting subject config
    Given subject "ro-cfg-allowed" has schema:
      """
      {"type":"record","name":"ROCfg","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I set the config for subject "ro-cfg-allowed" to "NONE"
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  @axonops-only
  Scenario: READONLY mode allows deleting subject config
    Given subject "ro-cfgdel-allowed" has schema:
      """
      {"type":"record","name":"ROCfgDel","fields":[{"name":"a","type":"string"}]}
      """
    And I set the config for subject "ro-cfgdel-allowed" to "NONE"
    When I set the global mode to "READONLY"
    And I delete the config for subject "ro-cfgdel-allowed"
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  Scenario: READONLY mode allows setting mode
    When I set the global mode to "READONLY"
    # Mode changes are always allowed (otherwise you'd be stuck)
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"

  Scenario: READONLY mode allows deleting subject mode
    When I set the mode for subject "ro-modedel-allowed" to "IMPORT"
    And I set the global mode to "READONLY"
    And I delete the mode for subject "ro-modedel-allowed"
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  # ==========================================================================
  # READONLY_OVERRIDE — ALSO ALLOWS CONFIG AND MODE CHANGES
  # ==========================================================================

  @axonops-only
  Scenario: READONLY_OVERRIDE allows setting subject config
    Given subject "override-cfg-allowed" has schema:
      """
      {"type":"record","name":"OverrideCfg","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY_OVERRIDE"
    And I set the config for subject "override-cfg-allowed" to "NONE"
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  Scenario: READONLY_OVERRIDE allows changing mode back to READWRITE
    When I set the global mode to "READONLY_OVERRIDE"
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"

  # ==========================================================================
  # PERMANENT DELETE OF "LATEST" BLOCKED
  # ==========================================================================

  @axonops-only
  Scenario: Permanent delete of version "latest" is not allowed
    Given subject "perm-del-latest" has schema:
      """
      {"type":"record","name":"PermDelLatest","fields":[{"name":"a","type":"string"}]}
      """
    # Soft-delete first
    When I DELETE "/subjects/perm-del-latest/versions/1"
    Then the response status should be 200
    # Try permanent delete of "latest" — should fail
    When I DELETE "/subjects/perm-del-latest/versions/latest?permanent=true"
    Then the response status should be 422

  Scenario: Permanent delete with explicit version number works
    Given subject "perm-del-explicit" has schema:
      """
      {"type":"record","name":"PermDelExplicit","fields":[{"name":"a","type":"string"}]}
      """
    # Soft-delete first
    When I DELETE "/subjects/perm-del-explicit/versions/1"
    Then the response status should be 200
    # Permanent delete with explicit version — should succeed
    When I DELETE "/subjects/perm-del-explicit/versions/1?permanent=true"
    Then the response status should be 200
