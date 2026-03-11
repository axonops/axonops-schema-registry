@mcp
Feature: MCP Config & Mode Tools
  MCP tools for managing compatibility configuration and registry mode.

  Scenario: Get default config
    When I call MCP tool "get_config"
    Then the MCP result should contain "BACKWARD"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Set and get subject config
    When I call MCP tool "set_config" with input:
      | subject             | mcp-cfg-test |
      | compatibility_level | FULL         |
    Then the MCP result should contain "FULL"
    When I call MCP tool "get_config" with input:
      | subject | mcp-cfg-test |
    Then the MCP result should contain "FULL"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Delete subject config
    When I call MCP tool "set_config" with input:
      | subject             | mcp-cfg-del |
      | compatibility_level | NONE        |
    When I call MCP tool "delete_config" with input:
      | subject | mcp-cfg-del |
    Then the MCP result should contain "NONE"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Get default mode
    When I call MCP tool "get_mode"
    Then the MCP result should contain "READWRITE"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Set and get subject mode
    When I call MCP tool "set_mode" with input:
      | subject | mcp-mode-test |
      | mode    | READONLY      |
    When I call MCP tool "get_mode" with input:
      | subject | mcp-mode-test |
    Then the MCP result should contain "READONLY"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Delete subject mode
    When I call MCP tool "set_mode" with input:
      | subject | mcp-mode-del |
      | mode    | READONLY     |
    When I call MCP tool "delete_mode" with input:
      | subject | mcp-mode-del |
    Then the MCP result should contain "READONLY"
    And the audit log should contain event "mcp_tool_call"
