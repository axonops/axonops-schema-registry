@mcp
Feature: MCP Schema Evolution Tools
  MCP tools for suggesting schema evolutions and planning migration paths.

  Background:
    Given I register an Avro schema for subject "evolve-test"

  # --- suggest_schema_evolution ---

  Scenario: Suggest adding a field to an Avro schema
    When I call MCP tool "suggest_schema_evolution" with JSON input:
      """
      {
        "subject": "evolve-test",
        "change_type": "add_field",
        "field_name": "email",
        "field_type": "string"
      }
      """
    Then the MCP result should contain "snippet"
    And the MCP result should contain "email"
    And the MCP result should contain "advice"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Suggest deprecating a field
    When I call MCP tool "suggest_schema_evolution" with JSON input:
      """
      {
        "subject": "evolve-test",
        "change_type": "deprecate_field",
        "field_name": "old_field"
      }
      """
    Then the MCP result should contain "steps"
    And the MCP result should contain "deprecate"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Suggest adding an enum symbol
    When I call MCP tool "suggest_schema_evolution" with JSON input:
      """
      {
        "subject": "evolve-test",
        "change_type": "add_enum_symbol",
        "enum_symbol": "NEW_STATUS"
      }
      """
    Then the MCP result should contain "NEW_STATUS"
    And the MCP result should contain "advice"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Suggest evolution with unsupported change type
    When I call MCP tool "suggest_schema_evolution" with JSON input:
      """
      {
        "subject": "evolve-test",
        "change_type": "unsupported_change"
      }
      """
    Then the MCP result should contain "unsupported change_type"
    And the audit log should contain event "mcp_tool_call"

  # --- plan_migration_path ---

  Scenario: Plan migration from source to target schema
    When I call MCP tool "plan_migration_path" with JSON input:
      """
      {
        "subject": "evolve-test",
        "target_schema": "{\"type\":\"record\",\"name\":\"Target\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"email\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "steps"
    And the MCP result should contain "total_steps"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Plan migration with no changes needed
    When I call MCP tool "plan_migration_path" with JSON input:
      """
      {
        "subject": "evolve-test",
        "target_schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "total_steps"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Plan migration returns compatibility level
    When I call MCP tool "plan_migration_path" with JSON input:
      """
      {
        "subject": "evolve-test",
        "target_schema": "{\"type\":\"record\",\"name\":\"V2\",\"fields\":[{\"name\":\"new_field\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "compatibility_level"
    And the MCP result should contain "current_version"
    And the audit log should contain event "mcp_tool_call"
