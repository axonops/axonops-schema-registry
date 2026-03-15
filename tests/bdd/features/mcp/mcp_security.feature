@mcp @mcp-security
Feature: MCP Security — Tool Policy and Access Control
  The MCP server enforces security policies including tool visibility,
  read-only mode, and access control. Tools that are denied by policy
  are invisible to MCP clients (not listed in tools/list).

  # ==========================================================================
  # 1. DEFAULT MODE — ALL TOOLS AVAILABLE
  # ==========================================================================

  Scenario: Default config allows all tools
    When I list MCP tools
    Then the MCP result should contain "health_check"
    And the MCP result should contain "register_schema"
    And the MCP result should contain "delete_subject"
    And the MCP result should contain "list_subjects"
    And the MCP result should contain "get_config"
    And the MCP result should contain "set_config"

  Scenario: Tool listing includes read and write tools
    When I list MCP tools
    Then the MCP result should contain "get_schema_by_id"
    And the MCP result should contain "get_latest_schema"
    And the MCP result should contain "check_compatibility"
    And the MCP result should contain "list_versions"

  # ==========================================================================
  # 2. TOOL OPERATIONS WORK UNDER DEFAULT POLICY
  # ==========================================================================

  Scenario: Write tools work under default policy
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "security-test-write",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I call MCP tool "list_subjects"
    Then the MCP result should contain "security-test-write"
    When I call MCP tool "delete_subject" with input:
      | subject | security-test-write |
    Then the MCP result should contain "1"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call           |
      | outcome              | success                 |
      | actor_id             | mcp-anonymous           |
      | actor_type           | anonymous               |
      | auth_method          |                         |
      | role                 |                         |
      | target_type          | subject                 |
      | target_id            | security-test-write     |
      | schema_id            |                         |
      | version              |                         |
      | schema_type          |                         |
      | before_hash          |                         |
      | after_hash           |                         |
      | context              |                         |
      | transport_security   |                         |
      | source_ip            |                         |
      | user_agent           |                         |
      | method               | MCP                     |
      | path                 | delete_subject          |
      | status_code          | 0                       |
      | reason               |                         |
      | error                |                         |
      | request_body         |                         |
      | metadata             |                         |
      | timestamp            | *                       |
      | duration_ms          | *                       |
      | request_id           |                         |

  Scenario: Config tools work under default policy
    When I call MCP tool "set_config" with input:
      | subject             | security-config-test |
      | compatibility_level | FULL                 |
    When I call MCP tool "get_config" with input:
      | subject | security-config-test |
    Then the MCP result should contain "FULL"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call            |
      | outcome              | success                  |
      | actor_id             | mcp-anonymous            |
      | actor_type           | anonymous                |
      | auth_method          |                          |
      | role                 |                          |
      | target_type          | subject                  |
      | target_id            | security-config-test     |
      | schema_id            |                          |
      | version              |                          |
      | schema_type          |                          |
      | before_hash          |                          |
      | after_hash           |                          |
      | context              |                          |
      | transport_security   |                          |
      | source_ip            |                          |
      | user_agent           |                          |
      | method               | MCP                      |
      | path                 | get_config               |
      | status_code          | 0                        |
      | reason               |                          |
      | error                |                          |
      | request_body         |                          |
      | metadata             |                          |
      | timestamp            | *                        |
      | duration_ms          | *                        |
      | request_id           |                          |
