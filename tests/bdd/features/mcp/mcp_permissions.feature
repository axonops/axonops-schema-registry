@mcp @mcp-permissions
Feature: MCP Permission Scopes — Granular Tool Access Control
  Permission scopes control which MCP tools are visible to AI agents.
  Each preset (readonly, developer, operator, admin, full) exposes a
  different subset of tools, allowing fine-grained access control.

  # ==========================================================================
  # 1. READONLY PRESET
  # ==========================================================================

  Scenario: Readonly preset allows list_subjects
    Given MCP permission preset is "readonly"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error

  Scenario: Readonly preset hides register_schema
    Given MCP permission preset is "readonly"
    When I list MCP tools
    Then the MCP result should not contain "register_schema"

  # ==========================================================================
  # 2. DEVELOPER PRESET
  # ==========================================================================

  Scenario: Developer preset allows register_schema
    Given MCP permission preset is "developer"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "perm-dev-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error

  Scenario: Developer preset hides delete_subject
    Given MCP permission preset is "developer"
    When I list MCP tools
    Then the MCP result should not contain "delete_subject"

  Scenario: Developer preset hides create_user
    Given MCP permission preset is "developer"
    When I list MCP tools
    Then the MCP result should not contain "create_user"

  # ==========================================================================
  # 3. OPERATOR PRESET
  # ==========================================================================

  Scenario: Operator preset allows delete_subject
    Given MCP permission preset is "operator"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "perm-op-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "perm-op-test"}
      """
    Then the MCP result should not be an error

  Scenario: Operator preset hides create_user
    Given MCP permission preset is "operator"
    When I list MCP tools
    Then the MCP result should not contain "create_user"

  # ==========================================================================
  # 4. ADMIN PRESET
  # ==========================================================================

  Scenario: Admin preset allows create_user
    Given MCP permission preset is "admin"
    When I list MCP tools
    Then the MCP result should contain "create_user"

  # ==========================================================================
  # 5. SYSTEM TOOLS ALWAYS AVAILABLE
  # ==========================================================================

  Scenario: System tools available under readonly preset
    Given MCP permission preset is "readonly"
    When I call MCP tool "health_check" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error

  Scenario: System tools listed under readonly preset
    Given MCP permission preset is "readonly"
    When I list MCP tools
    Then the MCP result should contain "health_check"
    And the MCP result should contain "get_server_info"
    And the MCP result should contain "get_schema_types"
    And the MCP result should contain "list_contexts"

  # ==========================================================================
  # 6. CUSTOM SCOPES
  # ==========================================================================

  Scenario: Custom scopes allow schema_read but block config and writes
    Given MCP permission scopes are "schema_read"
    When I list MCP tools
    Then the MCP result should contain "get_latest_schema"
    And the MCP result should contain "list_subjects"
    And the MCP result should not contain "register_schema"
    And the MCP result should not contain "get_config"
