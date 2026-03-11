@mcp @audit
Feature: MCP Audit Logging
  The MCP server MUST emit audit events for tool calls, errors, and confirmation
  flows so that operators can track all operations performed via AI assistants.
  The audit log MUST identify the principal (authenticated or anonymous).

  This test suite runs with MCP enabled but WITHOUT an MCP auth_token, so all
  MCP calls are anonymous: actor_id=mcp-anonymous, actor_type=anonymous.

  # --- Basic Tool Call Events ---

  Scenario: Tool call emits mcp_tool_call audit event
    When I call MCP tool "health_check"
    Then the MCP result should contain "healthy"
    And the audit log should contain an event:
      | event_type  | mcp_tool_call  |
      | outcome     | success        |
      | actor_id    | mcp-anonymous  |
      | actor_type  | anonymous      |
      | method      | MCP            |
      | path        | health_check   |

  Scenario: Tool error emits mcp_tool_error audit event
    When I call MCP tool "get_schema_version" with input:
      | subject | nonexistent-subject |
      | version | 1                   |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type  | mcp_tool_error     |
      | outcome     | failure            |
      | actor_id    | mcp-anonymous      |
      | actor_type  | anonymous          |
      | method      | MCP                |
      | path        | get_schema_version |

  Scenario: Schema registration audit includes subject name
    Given I register an Avro schema for subject "audit-subject-test"
    When I call MCP tool "get_latest_schema" with input:
      | subject | audit-subject-test |
    Then the audit log should contain "audit-subject-test"

  # --- Confirmation Flow Events ---

  Scenario: Confirmation token issuance emits mcp_confirm_issued
    Given MCP confirmations are enabled
    And I register an Avro schema for subject "audit-confirm-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "audit-confirm-sub", "permanent": true, "dry_run": true}
      """
    Then the MCP result should contain "confirm_token"
    And the audit log should contain an event:
      | event_type  | mcp_confirm_issued |
      | outcome     | success            |
      | actor_id    | mcp-anonymous      |
      | actor_type  | anonymous          |
      | method      | MCP                |
      | path        | delete_subject     |

  Scenario: Confirmation rejection emits mcp_confirm_rejected
    Given MCP confirmations are enabled
    And I register an Avro schema for subject "audit-reject-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "audit-reject-sub", "permanent": true, "confirm_token": "invalid-token"}
      """
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type  | mcp_confirm_rejected |
      | outcome     | success              |
      | actor_id    | mcp-anonymous        |
      | actor_type  | anonymous            |
      | method      | MCP                  |
      | path        | delete_subject       |

  # --- Principal Identification ---

  Scenario: Anonymous MCP tool call records mcp-anonymous as user
    When I call MCP tool "health_check"
    Then the MCP result should contain "healthy"
    And the audit log should contain an event:
      | event_type  | mcp_tool_call  |
      | actor_id    | mcp-anonymous  |
      | actor_type  | anonymous      |
      | outcome     | success        |

  Scenario: Tool error records mcp-anonymous as user
    When I call MCP tool "get_schema_version" with input:
      | subject | nonexistent-user-sub |
      | version | 1                    |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type  | mcp_tool_error |
      | actor_id    | mcp-anonymous  |
      | actor_type  | anonymous      |
      | outcome     | failure        |

  # --- Event Filtering ---

  Scenario: Event filtering hides disabled events
    When I call MCP tool "health_check"
    Then the audit log should contain event "mcp_tool_call"
    And the audit log should not contain event "schema_get"

  # --- Multiple Operations ---

  Scenario: Multiple tool calls produce multiple audit entries
    When I call MCP tool "health_check"
    And I call MCP tool "get_server_info"
    Then the audit log should contain "health_check"
    And the audit log should contain "get_server_info"

  # --- Schema Write Tools ---

  Scenario: Register schema via MCP emits mcp_tool_call with subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {"subject": "audit-mcp-register", "schema": "{\"type\":\"string\"}", "schema_type": "AVRO"}
      """
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type  | mcp_tool_call      |
      | outcome     | success            |
      | actor_id    | mcp-anonymous      |
      | actor_type  | anonymous          |
      | method      | MCP                |
      | path        | register_schema    |
      | target_id   | audit-mcp-register |

  Scenario: Delete subject via MCP emits mcp_tool_call
    Given I register an Avro schema for subject "audit-mcp-delsub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "audit-mcp-delsub"}
      """
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type  | mcp_tool_call    |
      | outcome     | success          |
      | actor_id    | mcp-anonymous    |
      | actor_type  | anonymous        |
      | method      | MCP              |
      | path        | delete_subject   |

  # --- Config Tools ---

  Scenario: Update config via MCP emits mcp_tool_call
    When I call MCP tool "set_config" with JSON input:
      """
      {"compatibility_level": "FULL"}
      """
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type  | mcp_tool_call  |
      | outcome     | success        |
      | actor_id    | mcp-anonymous  |
      | actor_type  | anonymous      |
      | method      | MCP            |
      | path        | set_config     |

  # --- Read Tools ---

  Scenario: Read-only tool still emits mcp_tool_call audit event
    When I call MCP tool "list_subjects"
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type  | mcp_tool_call  |
      | outcome     | success        |
      | actor_id    | mcp-anonymous  |
      | actor_type  | anonymous      |
      | method      | MCP            |
      | path        | list_subjects  |
