@mcp @audit
Feature: MCP Audit Logging
  The MCP server MUST emit audit events for tool calls, errors, and confirmation
  flows so that operators can track all operations performed via AI assistants.

  Scenario: Tool call emits mcp_tool_call audit event
    When I call MCP tool "health_check"
    Then the MCP result should contain "healthy"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Tool error emits mcp_tool_error audit event
    When I call MCP tool "get_schema_version" with input:
      | subject | nonexistent-subject |
      | version | 1                   |
    Then the MCP result should contain "error"
    And the audit log should contain event "mcp_tool_error"

  Scenario: Schema registration audit includes subject name
    Given I register an Avro schema for subject "audit-subject-test"
    When I call MCP tool "get_latest_schema" with input:
      | subject | audit-subject-test |
    Then the audit log should contain "audit-subject-test"

  Scenario: Confirmation token issuance emits mcp_confirm_issued
    Given MCP confirmations are enabled
    And I register an Avro schema for subject "audit-confirm-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "audit-confirm-sub", "permanent": true, "dry_run": true}
      """
    Then the MCP result should contain "confirm_token"
    And the audit log should contain event "mcp_confirm_issued"

  Scenario: Confirmation rejection emits mcp_confirm_rejected
    Given MCP confirmations are enabled
    And I register an Avro schema for subject "audit-reject-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "audit-reject-sub", "permanent": true, "confirm_token": "invalid-token"}
      """
    Then the MCP result should contain "error"
    And the audit log should contain event "mcp_confirm_rejected"

  Scenario: Event filtering hides disabled events
    When I call MCP tool "health_check"
    Then the audit log should contain event "mcp_tool_call"
    And the audit log should not contain event "schema_get"

  Scenario: Multiple tool calls produce multiple audit entries
    When I call MCP tool "health_check"
    And I call MCP tool "get_server_info"
    Then the audit log should contain "health_check"
    And the audit log should contain "get_server_info"
