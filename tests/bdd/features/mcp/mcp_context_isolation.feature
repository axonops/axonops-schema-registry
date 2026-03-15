@mcp
Feature: MCP Context Isolation
  MCP tools support an optional context parameter for multi-tenant isolation.

  Background:
    Given I register an Avro schema for subject "ctx-default-value"

  Scenario: Default context returns schemas registered without context
    When I call MCP tool "get_latest_schema" with input:
      | subject | ctx-default-value |
    Then the MCP result should not be an error
    And the MCP result should contain "ctx-default-value"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | ctx-default-value      |
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
      | path                 | get_latest_schema      |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Register and retrieve schema in a named context
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "ctx-staging-value",
        "schema": "{\"type\":\"record\",\"name\":\"Staging\",\"fields\":[{\"name\":\"env\",\"type\":\"string\"}]}",
        "schema_type": "AVRO",
        "context": ".staging"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_latest_schema" with JSON input:
      """
      {"subject": "ctx-staging-value", "context": ".staging"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "ctx-staging-value"
    And the MCP result should contain "env"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | ctx-staging-value      |
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
      | path                 | get_latest_schema      |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Named context does not leak into default context
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "ctx-isolated-value",
        "schema": "{\"type\":\"record\",\"name\":\"Isolated\",\"fields\":[{\"name\":\"secret\",\"type\":\"string\"}]}",
        "schema_type": "AVRO",
        "context": ".isolated"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_subjects"
    Then the MCP result should not be an error
    And the MCP result should not contain "ctx-isolated-value"
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
      | path                 | list_subjects          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: List subjects with context parameter
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "ctx-team-value",
        "schema": "{\"type\":\"record\",\"name\":\"TeamData\",\"fields\":[{\"name\":\"team\",\"type\":\"string\"}]}",
        "schema_type": "AVRO",
        "context": ".teamctx"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_subjects" with JSON input:
      """
      {"context": ".teamctx"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "ctx-team-value"
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
      | path                 | list_subjects          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
