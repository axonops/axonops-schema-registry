@mcp @mcp-workflow
Feature: MCP Workflow — Troubleshooting
  Tests troubleshooting workflows from prompts/troubleshooting.md and
  prompts/debug-registration-error.md by executing MCP tool call sequences.

  # Validates: glossary/error-reference — 42201
  Scenario: Invalid schema returns error via MCP
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-trouble-invalid",
        "schema": "not valid json"
      }
      """
    Then the MCP result should be an error
    And the MCP result should contain "invalid schema"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | wf-trouble-invalid     |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               | *                      |
      | error                | *                      |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # Validates: glossary/error-reference — 40401
  Scenario: Non-existent subject returns error via MCP
    When I call MCP tool "get_latest_schema" with JSON input:
      """
      {"subject": "wf-trouble-nonexistent-subject-xyz"}
      """
    Then the MCP result should be an error
    And the MCP result should contain "not found"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error                           |
      | outcome              | failure                                  |
      | actor_id             | mcp-anonymous                            |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | wf-trouble-nonexistent-subject-xyz       |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          |                                          |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              |                                          |
      | transport_security   |                                          |
      | source_ip            |                                          |
      | user_agent           |                                          |
      | method               | MCP                                      |
      | path                 | get_latest_schema                        |
      | status_code          | 0                                        |
      | reason               | *                                        |
      | error                | *                                        |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           |                                          |

  # Validates: glossary/error-reference — 409, troubleshooting Registration failures
  Scenario: Incompatible schema returns error then explain failure
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-trouble-incompat",
        "schema": "{\"type\":\"record\",\"name\":\"Rec\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-trouble-incompat",
        "schema": "{\"type\":\"record\",\"name\":\"Rec\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should be an error
    And the MCP result should contain "incompatible"
    When I call MCP tool "explain_compatibility_failure" with JSON input:
      """
      {
        "subject": "wf-trouble-incompat",
        "schema": "{\"type\":\"record\",\"name\":\"Rec\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type           | mcp_tool_call                    |
      | outcome              | success                          |
      | actor_id             | mcp-anonymous                    |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | wf-trouble-incompat              |
      | schema_id            |                                  |
      | version              |                                  |
      | schema_type          |                                  |
      | before_hash          |                                  |
      | after_hash           |                                  |
      | context              |                                  |
      | transport_security   |                                  |
      | source_ip            |                                  |
      | user_agent           |                                  |
      | method               | MCP                              |
      | path                 | explain_compatibility_failure    |
      | status_code          | 0                                |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           |                                  |

  # Validates: troubleshooting — Subject not found, match_subjects fuzzy match
  Scenario: Match subjects fuzzy to find misspelled subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "orders-value",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "match_subjects" with JSON input:
      """
      {"pattern": "orders"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "orders-value"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | match_subjects         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # Validates: glossary/error-reference — 40405, best-practices Soft-delete confusion
  Scenario: Permanent delete without soft-delete first returns error
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "wf-trouble-no-soft-delete-xyz", "permanent": true}
      """
    Then the MCP result should be an error
    And the audit log should contain an event:
      | event_type           | mcp_tool_error                         |
      | outcome              | failure                                |
      | actor_id             | mcp-anonymous                          |
      | actor_type           | anonymous                              |
      | auth_method          |                                        |
      | role                 |                                        |
      | target_type          | subject                                |
      | target_id            | wf-trouble-no-soft-delete-xyz          |
      | schema_id            |                                        |
      | version              |                                        |
      | schema_type          |                                        |
      | before_hash          |                                        |
      | after_hash           |                                        |
      | context              |                                        |
      | transport_security   |                                        |
      | source_ip            |                                        |
      | user_agent           |                                        |
      | method               | MCP                                    |
      | path                 | delete_subject                         |
      | status_code          | 0                                      |
      | reason               | *                                      |
      | error                | *                                      |
      | request_body         |                                        |
      | metadata             |                                        |
      | timestamp            | *                                      |
      | duration_ms          | *                                      |
      | request_id           |                                        |
