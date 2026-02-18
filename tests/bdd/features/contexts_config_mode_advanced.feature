@functional
Feature: Contexts — Advanced Config and Mode Behavior
  Tests advanced configuration and mode behavior within contexts including
  global config fallback, mode enforcement (READONLY blocking writes),
  per-subject overrides, config lifecycle, and cross-context isolation
  of config and mode settings.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # SCENARIO 1: GLOBAL CONFIG FALLBACK INTO CONTEXTS
  # ==========================================================================

  Scenario: Global config serves as default for context subjects
    # Set global compatibility to FULL (requires both backward and forward compat)
    When I PUT "/config" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Register v1 in .cfgm1 context
    When I POST "/subjects/:.cfgm1:global-fb/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GlobalFb\",\"namespace\":\"com.cfgm.s1\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Try to register v2 with incompatible field type change (string -> int)
    # FULL checks both directions — changing a field type breaks both backward and forward
    When I POST "/subjects/:.cfgm1:global-fb/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GlobalFb\",\"namespace\":\"com.cfgm.s1\",\"fields\":[{\"name\":\"name\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 409
    # This proves the global FULL config applies within the named context

  # ==========================================================================
  # SCENARIO 2: PER-SUBJECT CONFIG OVERRIDES GLOBAL
  # ==========================================================================

  Scenario: Per-subject config in context overrides global
    # Set global compatibility to BACKWARD
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Override per-subject config to NONE for a subject in .cfgm2
    When I PUT "/config/:.cfgm2:flexible" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Verify the per-subject config is set
    When I GET "/config/:.cfgm2:flexible"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    # Register v1
    When I POST "/subjects/:.cfgm2:flexible/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Flexible\",\"namespace\":\"com.cfgm.s2\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register a completely different v2 — should succeed because NONE allows anything
    When I POST "/subjects/:.cfgm2:flexible/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Flexible\",\"namespace\":\"com.cfgm.s2\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"active\",\"type\":\"boolean\"}]}"}
      """
    Then the response status should be 200
    # Verify 2 versions exist
    When I GET "/subjects/:.cfgm2:flexible/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # SCENARIO 3: DELETE PER-SUBJECT CONFIG FALLS BACK TO GLOBAL
  # ==========================================================================

  Scenario: Delete per-subject config falls back to global
    # Set global compatibility to BACKWARD
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Set per-subject config to NONE
    When I PUT "/config/:.cfgm3:fallback" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Register v1
    When I POST "/subjects/:.cfgm3:fallback/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Fallback\",\"namespace\":\"com.cfgm.s3\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register incompatible v2 — succeeds because NONE is set
    When I POST "/subjects/:.cfgm3:fallback/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Fallback\",\"namespace\":\"com.cfgm.s3\",\"fields\":[{\"name\":\"name\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Delete per-subject config — should fall back to global BACKWARD
    When I DELETE "/config/:.cfgm3:fallback"
    Then the response status should be 200
    # Check compatibility of another incompatible schema — should now be incompatible (BACKWARD)
    # Changing name from int to boolean is a type change, not backward compatible
    When I POST "/compatibility/subjects/:.cfgm3:fallback/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Fallback\",\"namespace\":\"com.cfgm.s3\",\"fields\":[{\"name\":\"name\",\"type\":\"boolean\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false

  # ==========================================================================
  # SCENARIO 4: READONLY MODE BLOCKS REGISTRATION IN CONTEXT
  # ==========================================================================

  Scenario: READONLY mode blocks schema registration in context
    # Register v1 first while mode is READWRITE
    When I POST "/subjects/:.cfgm4:readonly-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ReadonlyTest\",\"namespace\":\"com.cfgm.s4\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set mode for this subject to READONLY
    When I PUT "/mode/:.cfgm4:readonly-test" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Verify mode is set
    When I GET "/mode/:.cfgm4:readonly-test"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"
    # Try to register v2 — should fail with 422
    When I POST "/subjects/:.cfgm4:readonly-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ReadonlyTest\",\"namespace\":\"com.cfgm.s4\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":[\"null\",\"int\"],\"default\":null}]}"}
      """
    Then the response status should be 422
    And the response field "error_code" should be 42205

  # ==========================================================================
  # SCENARIO 5: READWRITE MODE ALLOWS REGISTRATION AFTER MODE CHANGE
  # ==========================================================================

  Scenario: READWRITE mode allows registration after mode change
    # Register v1 while READWRITE
    When I POST "/subjects/:.cfgm5:rw-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"RwTest\",\"namespace\":\"com.cfgm.s5\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set mode to READONLY
    When I PUT "/mode/:.cfgm5:rw-test" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Try to register v2 — should fail
    When I POST "/subjects/:.cfgm5:rw-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"RwTest\",\"namespace\":\"com.cfgm.s5\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 422
    # Set mode back to READWRITE
    When I PUT "/mode/:.cfgm5:rw-test" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    # Register v2 — should now succeed
    When I POST "/subjects/:.cfgm5:rw-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"RwTest\",\"namespace\":\"com.cfgm.s5\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Verify 2 versions exist
    When I GET "/subjects/:.cfgm5:rw-test/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # SCENARIO 6: MODE IN ONE CONTEXT DOES NOT AFFECT ANOTHER
  # ==========================================================================

  Scenario: Mode in one context does not affect another
    # Register v1 in context .cfgm6a
    When I POST "/subjects/:.cfgm6a:mode-iso/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeIsoA\",\"namespace\":\"com.cfgm.s6a\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register v1 in context .cfgm6b
    When I POST "/subjects/:.cfgm6b:mode-iso/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeIsoB\",\"namespace\":\"com.cfgm.s6b\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set READONLY only for subject in context .cfgm6a
    When I PUT "/mode/:.cfgm6a:mode-iso" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Try register v2 in .cfgm6a — should fail
    When I POST "/subjects/:.cfgm6a:mode-iso/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeIsoA\",\"namespace\":\"com.cfgm.s6a\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"extra\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 422
    And the response field "error_code" should be 42205
    # Register v2 in .cfgm6b — should succeed (no READONLY set here)
    When I POST "/subjects/:.cfgm6b:mode-iso/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeIsoB\",\"namespace\":\"com.cfgm.s6b\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"extra\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200

  # ==========================================================================
  # SCENARIO 7: CONFIG IN ONE CONTEXT DOES NOT LEAK TO ANOTHER
  # ==========================================================================

  Scenario: Config set in one context does not leak to another
    # Set global to BACKWARD so .cfgm7b will use it
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Set NONE for subject in .cfgm7a
    When I PUT "/config/:.cfgm7a:cfg-leak" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Register v1 in .cfgm7a
    When I POST "/subjects/:.cfgm7a:cfg-leak/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgLeakA\",\"namespace\":\"com.cfgm.s7a\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register v1 in .cfgm7b (no per-subject config — uses global BACKWARD)
    When I POST "/subjects/:.cfgm7b:cfg-leak/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgLeakB\",\"namespace\":\"com.cfgm.s7b\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register incompatible v2 in .cfgm7a — succeeds (NONE)
    When I POST "/subjects/:.cfgm7a:cfg-leak/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgLeakA\",\"namespace\":\"com.cfgm.s7a\",\"fields\":[{\"name\":\"name\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Check compat of incompatible schema in .cfgm7b — should be false (BACKWARD)
    When I POST "/compatibility/subjects/:.cfgm7b:cfg-leak/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgLeakB\",\"namespace\":\"com.cfgm.s7b\",\"fields\":[{\"name\":\"name\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false

  # ==========================================================================
  # SCENARIO 8: GLOBAL MODE APPLIES TO CONTEXT SUBJECTS
  # ==========================================================================

  Scenario: Global mode applies to default context subjects
    # Register v1 in default context BEFORE setting global mode to READONLY
    When I POST "/subjects/global-mode-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GlobalMode\",\"namespace\":\"com.cfgm.s8\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set global mode to READONLY
    When I PUT "/mode" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Try to register v2 — should fail because global mode is READONLY
    When I POST "/subjects/global-mode-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GlobalMode\",\"namespace\":\"com.cfgm.s8\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"extra\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 422
    And the response field "error_code" should be 42205
    # Reset global mode back to READWRITE for other scenarios
    When I PUT "/mode" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200

  # ==========================================================================
  # SCENARIO 9: PER-SUBJECT MODE OVERRIDES GLOBAL MODE IN CONTEXT
  # ==========================================================================

  Scenario: Per-subject mode overrides global mode in context
    # Ensure global mode is READWRITE
    When I PUT "/mode" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    # Register v1 in .cfgm9
    When I POST "/subjects/:.cfgm9:override/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeOverride\",\"namespace\":\"com.cfgm.s9\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set per-subject mode to READONLY (global is READWRITE)
    When I PUT "/mode/:.cfgm9:override" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Verify per-subject mode is READONLY
    When I GET "/mode/:.cfgm9:override"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"
    # Try to register v2 — should fail because per-subject READONLY wins over global READWRITE
    When I POST "/subjects/:.cfgm9:override/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeOverride\",\"namespace\":\"com.cfgm.s9\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 422
    And the response field "error_code" should be 42205

  # ==========================================================================
  # SCENARIO 10: DELETE PER-SUBJECT MODE RESTORES DEFAULT BEHAVIOR
  # ==========================================================================

  Scenario: Delete per-subject mode restores default behavior
    # Register v1 in .cfgm10
    When I POST "/subjects/:.cfgm10:mode-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeDel\",\"namespace\":\"com.cfgm.s10\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set per-subject mode to READONLY
    When I PUT "/mode/:.cfgm10:mode-del" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Try to register v2 — should fail
    When I POST "/subjects/:.cfgm10:mode-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeDel\",\"namespace\":\"com.cfgm.s10\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":[\"null\",\"int\"],\"default\":null}]}"}
      """
    Then the response status should be 422
    And the response field "error_code" should be 42205
    # Delete per-subject mode — falls back to global READWRITE
    When I DELETE "/mode/:.cfgm10:mode-del"
    Then the response status should be 200
    # Register v2 — should now succeed
    When I POST "/subjects/:.cfgm10:mode-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeDel\",\"namespace\":\"com.cfgm.s10\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":[\"null\",\"int\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Verify 2 versions exist
    When I GET "/subjects/:.cfgm10:mode-del/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # SCENARIO 11: COMPATIBILITY CHECK RESPECTS CONTEXT CONFIG
  # ==========================================================================

  Scenario: Compatibility check respects context config
    # Set NONE for subject in .cfgm11a
    When I PUT "/config/:.cfgm11a:compat-check" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Set FULL for subject in .cfgm11b
    When I PUT "/config/:.cfgm11b:compat-check" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Register same schema structure in both contexts (different namespaces for clarity)
    When I POST "/subjects/:.cfgm11a:compat-check/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CompatCheck\",\"namespace\":\"com.cfgm.s11a\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.cfgm11b:compat-check/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CompatCheck\",\"namespace\":\"com.cfgm.s11b\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Check compat of type-changing schema in .cfgm11a (NONE) — is_compatible true
    When I POST "/compatibility/subjects/:.cfgm11a:compat-check/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CompatCheck\",\"namespace\":\"com.cfgm.s11a\",\"fields\":[{\"name\":\"name\",\"type\":\"int\"},{\"name\":\"value\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true
    # Check compat of same type-changing schema in .cfgm11b (FULL) — is_compatible false
    When I POST "/compatibility/subjects/:.cfgm11b:compat-check/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CompatCheck\",\"namespace\":\"com.cfgm.s11b\",\"fields\":[{\"name\":\"name\",\"type\":\"int\"},{\"name\":\"value\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false

  # ==========================================================================
  # SCENARIO 12: BACKWARD COMPAT ENFORCED PER-CONTEXT FOR AVRO EVOLUTION
  # ==========================================================================

  Scenario: BACKWARD compatibility enforced per-context for Avro evolution
    # Set BACKWARD for subject in .cfgm12
    When I PUT "/config/:.cfgm12:avro-evo" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Register v1: {name: string, age: int}
    When I POST "/subjects/:.cfgm12:avro-evo/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AvroEvo\",\"namespace\":\"com.cfgm.s12\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Register v2: add optional email field — backward compatible (new field has default)
    When I POST "/subjects/:.cfgm12:avro-evo/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AvroEvo\",\"namespace\":\"com.cfgm.s12\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    # Verify 2 versions exist
    When I GET "/subjects/:.cfgm12:avro-evo/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    # Check compat of schema that changes age from int to string — should be incompatible
    When I POST "/compatibility/subjects/:.cfgm12:avro-evo/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AvroEvo\",\"namespace\":\"com.cfgm.s12\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
