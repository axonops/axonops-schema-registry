@functional @contexts
Feature: Contexts — Per-Context Config and Mode
  Compatibility configuration and registry modes are per-context.
  Setting config/mode in one context MUST NOT affect another context.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # PER-SUBJECT CONFIG IN CONTEXT
  # ==========================================================================

  Scenario: Set per-subject config in context
    When I POST "/subjects/:.cfg-ctx:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.cfg-ctx:s1" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    When I GET "/config/:.cfg-ctx:s1"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  Scenario: Delete per-subject config in context
    When I POST "/subjects/:.cfg-del:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgDelTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.cfg-del:s1" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    When I DELETE "/config/:.cfg-del:s1"
    Then the response status should be 200

  Scenario: Config in one context does not affect another context
    # Set FULL in context A
    When I POST "/subjects/:.cfg-iso-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgIsoA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.cfg-iso-a:s1" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Set NONE in context B
    When I POST "/subjects/:.cfg-iso-b:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgIsoB\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.cfg-iso-b:s1" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Verify context A still has FULL
    When I GET "/config/:.cfg-iso-a:s1"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    # Verify context B has NONE
    When I GET "/config/:.cfg-iso-b:s1"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"

  # ==========================================================================
  # PER-SUBJECT MODE IN CONTEXT
  # ==========================================================================

  Scenario: Set per-subject mode in context
    When I POST "/subjects/:.mode-ctx:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/mode/:.mode-ctx:s1" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    When I GET "/mode/:.mode-ctx:s1"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"

  Scenario: Delete per-subject mode in context
    When I POST "/subjects/:.mode-del:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeDelTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/mode/:.mode-del:s1" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    When I DELETE "/mode/:.mode-del:s1"
    Then the response status should be 200

  Scenario: Mode in one context does not affect another context
    When I POST "/subjects/:.mode-iso-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeIsoA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/mode/:.mode-iso-a:s1" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.mode-iso-b:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ModeIsoB\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/mode/:.mode-iso-a:s1"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"
    # Context B has no explicit mode set, so it falls back to global READWRITE
    When I GET "/mode/:.mode-iso-b:s1"
    Then the response status should be 404

  # ==========================================================================
  # COMPATIBILITY ENFORCEMENT WITH CONTEXT CONFIG
  # ==========================================================================

  Scenario: Backward compatibility enforced within context
    When I POST "/subjects/:.cfg-enforce:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgEnforce\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.cfg-enforce:s1" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Compatible change (add optional field with default) — same record name
    When I POST "/subjects/:.cfg-enforce:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgEnforce\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200

  Scenario: Incompatible change rejected in context with BACKWARD config
    When I POST "/subjects/:.cfg-reject:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgReject\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.cfg-reject:s1" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Incompatible change (change field type)
    When I POST "/subjects/:.cfg-reject:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgReject\",\"fields\":[{\"name\":\"a\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 409

  Scenario: NONE compatibility allows any change in context
    When I POST "/subjects/:.cfg-none:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgNone\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.cfg-none:s1" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Totally different schema — allowed under NONE
    When I POST "/subjects/:.cfg-none:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CfgNone\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
