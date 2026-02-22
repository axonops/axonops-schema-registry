@functional @contexts
Feature: Contexts — __GLOBAL Config/Mode Inheritance (3-Tier)
  Verify Confluent-compatible 3-tier config/mode inheritance:
    Step 1: Per-subject config/mode
    Step 2: Context-level global config/mode
    Step 3: __GLOBAL context config/mode (cross-context default)
    Step 4: Server hardcoded default (BACKWARD for config, READWRITE for mode)

  The __GLOBAL context (".__GLOBAL") is a special virtual context that only
  holds config/mode settings. Schemas and subjects CANNOT be registered there.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # CONFIG INHERITANCE — 4-TIER CHAIN
  # ==========================================================================

  Scenario: Default config without any overrides returns server default
    # No config set anywhere — should return server default (BACKWARD)
    When I GET "/config?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: __GLOBAL config acts as cross-context default
    # Set config on __GLOBAL context
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Named context with no config should inherit from __GLOBAL
    When I POST "/contexts/.gc-inherit/subjects/test-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcInherit\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts/.gc-inherit/config/test-subj?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    # Clean up
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200

  Scenario: Context-level config overrides __GLOBAL
    # Set __GLOBAL to FULL
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Set context-level config for .gc-override to NONE
    When I PUT "/config/:.gc-override:" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Register a schema so context exists
    When I POST "/contexts/.gc-override/subjects/test-s/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcOverride\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Subject in this context should get NONE (context-level), not FULL (__GLOBAL)
    When I GET "/contexts/.gc-override/config/test-s?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    # Clean up
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200

  Scenario: Per-subject config overrides both context-level and __GLOBAL
    # Set __GLOBAL to FULL
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Set context-level config for .gc-subj to NONE
    When I PUT "/config/:.gc-subj:" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Register a schema and set per-subject config
    When I POST "/contexts/.gc-subj/subjects/my-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcSubj\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/contexts/.gc-subj/config/my-topic" with body:
      """
      {"compatibility": "FORWARD"}
      """
    Then the response status should be 200
    # Subject should get FORWARD (per-subject), not NONE or FULL
    When I GET "/contexts/.gc-subj/config/my-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    # Clean up
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200

  Scenario: Delete per-subject config falls back to context-level
    # Set context-level config
    When I PUT "/config/:.gc-fallback:" with body:
      """
      {"compatibility": "FULL_TRANSITIVE"}
      """
    Then the response status should be 200
    # Register schema and set per-subject
    When I POST "/contexts/.gc-fallback/subjects/fb-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcFb\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/contexts/.gc-fallback/config/fb-topic" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Verify per-subject is NONE
    When I GET "/contexts/.gc-fallback/config/fb-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    # Delete per-subject config
    When I DELETE "/contexts/.gc-fallback/config/fb-topic"
    Then the response status should be 200
    # Should now fall back to context-level FULL_TRANSITIVE
    When I GET "/contexts/.gc-fallback/config/fb-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL_TRANSITIVE"

  Scenario: Delete context-level config falls back to __GLOBAL
    # Set __GLOBAL to FORWARD_TRANSITIVE
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FORWARD_TRANSITIVE"}
      """
    Then the response status should be 200
    # Set context-level config
    When I PUT "/config/:.gc-del:" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Register schema
    When I POST "/contexts/.gc-del/subjects/del-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcDel\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Verify context-level
    When I GET "/contexts/.gc-del/config/del-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    # Delete context-level config
    When I DELETE "/config/:.gc-del:"
    Then the response status should be 200
    # Should now fall back to __GLOBAL FORWARD_TRANSITIVE
    When I GET "/contexts/.gc-del/config/del-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD_TRANSITIVE"
    # Clean up
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200

  Scenario: Delete __GLOBAL config falls back to server default
    # Set __GLOBAL
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Register schema in a named context
    When I POST "/contexts/.gc-srvdef/subjects/def-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcSrvDef\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Verify __GLOBAL applies
    When I GET "/contexts/.gc-srvdef/config/def-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    # Delete __GLOBAL config
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200
    # Should fall back to server default BACKWARD
    When I GET "/contexts/.gc-srvdef/config/def-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: PUT /config at root does NOT affect named contexts
    # Set root global config (default context only)
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Register schema in a named context
    When I POST "/contexts/.gc-noroot/subjects/nr-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcNoRoot\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Named context should NOT see the root config — should get server default
    When I GET "/contexts/.gc-noroot/config/nr-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  # ==========================================================================
  # defaultToGlobal PARAMETER
  # ==========================================================================

  Scenario: GET /config without defaultToGlobal returns direct config only
    # Don't set any config, just check default behavior
    When I GET "/config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: GET /config with defaultToGlobal=true walks full chain
    # Set __GLOBAL config
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FULL_TRANSITIVE"}
      """
    Then the response status should be 200
    # GET /config?defaultToGlobal=true for default context should walk to __GLOBAL
    When I GET "/config?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL_TRANSITIVE"
    # GET /config without defaultToGlobal should return direct default context config (server default)
    When I GET "/config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"
    # Clean up
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200

  Scenario: GET /config/{subject}?defaultToGlobal=true walks 4-tier chain
    # Set __GLOBAL
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FORWARD"}
      """
    Then the response status should be 200
    # Register schema in default context
    When I POST "/subjects/dtg-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DtgTopic\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Without defaultToGlobal, subject with no config returns 404
    When I GET "/config/dtg-topic"
    Then the response status should be 404
    # With defaultToGlobal=true, walks chain to __GLOBAL
    When I GET "/config/dtg-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    # Clean up
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200

  # ==========================================================================
  # MODE INHERITANCE — 4-TIER CHAIN
  # ==========================================================================

  Scenario: Named context mode falls back to __GLOBAL when no context mode set
    # Set __GLOBAL mode to READONLY
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Named context with no mode should inherit READONLY from __GLOBAL
    When I POST "/contexts/.gm-inherit/subjects/mode-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GmInherit\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    # Should be blocked by READONLY from __GLOBAL
    Then the response status should be 422
    # Clean up
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200

  Scenario: READONLY on __GLOBAL blocks writes in named contexts
    # Register first in a clean context
    When I POST "/contexts/.gm-block/subjects/block-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GmBlock\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Now set __GLOBAL to READONLY
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Try to register another version — should be blocked
    When I POST "/contexts/.gm-block/subjects/block-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GmBlock\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 422
    # Clean up
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200

  Scenario: Per-context READWRITE overrides __GLOBAL READONLY
    # Set __GLOBAL to READONLY
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Set context-level mode to READWRITE (overrides __GLOBAL)
    When I PUT "/mode/:.gm-ctx-rw:" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    # Register should succeed — context READWRITE overrides __GLOBAL READONLY
    When I POST "/contexts/.gm-ctx-rw/subjects/rw-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GmCtxRw\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Clean up
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200

  Scenario: Per-subject READWRITE overrides __GLOBAL READONLY
    # Register a schema first
    When I POST "/contexts/.gm-subj-rw/subjects/srw-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GmSubjRw\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set __GLOBAL to READONLY
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Set per-subject mode to READWRITE
    When I PUT "/contexts/.gm-subj-rw/mode/srw-topic" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    # Register another version — should succeed (per-subject overrides __GLOBAL)
    When I POST "/contexts/.gm-subj-rw/subjects/srw-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GmSubjRw\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Clean up
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200

  Scenario: READONLY_OVERRIDE on default context overrides everything
    # Register a schema first
    When I POST "/contexts/.gm-ro-override/subjects/ro-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GmRoOverride\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set per-subject mode to READWRITE
    When I PUT "/contexts/.gm-ro-override/mode/ro-topic" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    # Set default context global mode to READONLY_OVERRIDE (kill switch)
    When I PUT "/mode" with body:
      """
      {"mode": "READONLY_OVERRIDE"}
      """
    Then the response status should be 200
    # Try to register — should be blocked even with per-subject READWRITE
    When I POST "/contexts/.gm-ro-override/subjects/ro-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GmRoOverride\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 422
    # Clean up
    When I PUT "/mode" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200

  # ==========================================================================
  # __GLOBAL CONTEXT — SCHEMA BLOCKING
  # ==========================================================================

  Scenario: Cannot register schema under __GLOBAL via qualified subject
    When I POST "/subjects/:.__GLOBAL:blocked-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Blocked\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 400

  Scenario: Cannot register schema under __GLOBAL via URL prefix
    When I POST "/contexts/.__GLOBAL/subjects/blocked-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Blocked\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 400

  Scenario: Cannot list subjects under __GLOBAL context
    When I GET "/contexts/.__GLOBAL/subjects"
    Then the response status should be 400

  Scenario: Config operations on __GLOBAL work
    # Set config
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    And the response field "compatibility" should be "FULL"
    # Get config
    When I GET "/config/:.__GLOBAL:"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    # Delete config
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200

  Scenario: Mode operations on __GLOBAL work
    # Set mode
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    And the response field "mode" should be "READONLY"
    # Get mode
    When I GET "/mode/:.__GLOBAL:"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"
    # Delete mode
    When I DELETE "/mode/:.__GLOBAL:"
    Then the response status should be 200
    # Restore
    When I PUT "/mode/:.__GLOBAL:" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200

  # ==========================================================================
  # CONTEXT-LEVEL CONFIG VIA QUALIFIED SUBJECT
  # ==========================================================================

  Scenario: Set context-level config via qualified subject with empty subject
    # :.myctx: is the qualified form for "context-level config of .myctx"
    When I PUT "/config/:.gc-qualified:" with body:
      """
      {"compatibility": "FORWARD"}
      """
    Then the response status should be 200
    # Register a schema so the context exists
    When I POST "/contexts/.gc-qualified/subjects/qual-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcQual\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Subject should inherit context-level config
    When I GET "/contexts/.gc-qualified/config/qual-topic?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"

  Scenario: Delete context-level config via qualified subject
    # Set context-level config
    When I PUT "/config/:.gc-dq:" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Delete it
    When I DELETE "/config/:.gc-dq:"
    Then the response status should be 200

  Scenario: GET /contexts does NOT include __GLOBAL
    # Set some config on __GLOBAL to ensure it exists in storage
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Register a schema in a real context
    When I POST "/contexts/.gc-listed/subjects/listed-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcListed\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # List contexts — should include real contexts but NOT __GLOBAL
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should not contain ".__GLOBAL"
    # Clean up
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200

  # ==========================================================================
  # COMPAT ENFORCEMENT WITH __GLOBAL CONFIG
  # ==========================================================================

  Scenario: Compatibility check enforces __GLOBAL config in named context
    # Set __GLOBAL to BACKWARD
    When I PUT "/config/:.__GLOBAL:" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Register v1
    When I POST "/contexts/.gc-compat/subjects/compat-topic/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcCompat\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # v2 backward-compatible (additive field with default)
    When I POST "/compatibility/subjects/:.gc-compat:compat-topic/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcCompat\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true
    # v2 backward-incompatible (removed required field)
    When I POST "/compatibility/subjects/:.gc-compat:compat-topic/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"GcCompat\",\"fields\":[{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
    # Clean up
    When I DELETE "/config/:.__GLOBAL:"
    Then the response status should be 200
