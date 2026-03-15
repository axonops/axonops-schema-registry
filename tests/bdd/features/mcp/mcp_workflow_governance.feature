@mcp @mcp-workflow
Feature: MCP Workflow — Governance Setup
  Tests the governance workflow from prompts/governance-setup.md
  by executing each step as MCP tool calls.

  # Validates: prompts/governance-setup.md — Step 2, glossary/compatibility Configuration Resolution
  Scenario: Set global compatibility and verify resolution
    When I call MCP tool "set_config" with JSON input:
      """
      {"compatibility_level": "FULL"}
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_config" with JSON input:
      """
      {}
      """
    Then the MCP result should contain "FULL"
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
      | path                 | get_config             |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # Validates: prompts/governance-setup.md — Step 3
  Scenario: Score quality across subjects
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-gov-quality",
        "schema": "{\"type\":\"record\",\"name\":\"GovTest\",\"namespace\":\"com.example\",\"doc\":\"Test schema\",\"fields\":[{\"name\":\"id\",\"type\":\"string\",\"doc\":\"ID\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "score_schema_quality" with JSON input:
      """
      {"subject": "wf-gov-quality"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "score"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | wf-gov-quality         |
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
      | path                 | score_schema_quality   |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # Validates: prompts/governance-setup.md — Step 3
  Scenario: Check field consistency for type drift
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-gov-cons-a",
        "schema": "{\"type\":\"record\",\"name\":\"A\",\"fields\":[{\"name\":\"customer_id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-gov-cons-b",
        "schema": "{\"type\":\"record\",\"name\":\"B\",\"fields\":[{\"name\":\"customer_id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_field_consistency" with JSON input:
      """
      {
        "field": "customer_id"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "customer_id"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          |                            |
      | target_id            |                            |
      | schema_id            |                            |
      | version              |                            |
      | schema_type          |                            |
      | before_hash          |                            |
      | after_hash           |                            |
      | context              |                            |
      | transport_security   |                            |
      | source_ip            |                            |
      | user_agent           |                            |
      | method               | MCP                        |
      | path                 | check_field_consistency    |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

  # Validates: prompts/governance-setup.md — Step 4, glossary/data-contracts 3-Layer Merge
  Scenario: Data contract with override metadata for PII
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-gov-contract",
        "schema": "{\"type\":\"record\",\"name\":\"Customer\",\"fields\":[{\"name\":\"email\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "set_config_full" with JSON input:
      """
      {
        "subject": "wf-gov-contract",
        "compatibility_level": "BACKWARD",
        "default_metadata": {
          "properties": {"pii": "true", "owner": "data-team"}
        }
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_subject_config_full" with JSON input:
      """
      {"subject": "wf-gov-contract"}
      """
    Then the MCP result should contain "pii"
    And the MCP result should contain "data-team"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | wf-gov-contract            |
      | schema_id            |                            |
      | version              |                            |
      | schema_type          |                            |
      | before_hash          |                            |
      | after_hash           |                            |
      | context              |                            |
      | transport_security   |                            |
      | source_ip            |                            |
      | user_agent           |                            |
      | method               | MCP                        |
      | path                 | get_subject_config_full    |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |
