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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | evolve-test                |
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
      | path                 | suggest_schema_evolution   |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | evolve-test                |
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
      | path                 | suggest_schema_evolution   |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | evolve-test                |
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
      | path                 | suggest_schema_evolution   |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

  Scenario: Suggest evolution with unsupported change type
    When I call MCP tool "suggest_schema_evolution" with JSON input:
      """
      {
        "subject": "evolve-test",
        "change_type": "unsupported_change"
      }
      """
    Then the MCP result should contain "unsupported change_type"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | evolve-test                |
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
      | path                 | suggest_schema_evolution   |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | evolve-test                |
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
      | path                 | plan_migration_path        |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

  Scenario: Plan migration with no changes needed
    When I call MCP tool "plan_migration_path" with JSON input:
      """
      {
        "subject": "evolve-test",
        "target_schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "total_steps"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | evolve-test                |
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
      | path                 | plan_migration_path        |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | evolve-test                |
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
      | path                 | plan_migration_path        |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |
