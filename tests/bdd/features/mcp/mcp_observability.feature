@mcp @observability
Feature: MCP Observability — Logging & Error Tracking
  The MCP server instruments every tool call with structured logging and
  error tracking. These scenarios verify that tool calls complete successfully
  and that errors are surfaced correctly through the MCP protocol.

  # ==========================================================================
  # 1. SUCCESSFUL TOOL CALLS
  # ==========================================================================

  Scenario: Successful health check is tracked
    When I call MCP tool "health_check"
    Then the MCP result should contain "healthy"
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
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | health_check           |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Successful schema registration is tracked
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "obs-track-test",
        "schema": "{\"type\":\"record\",\"name\":\"Tracked\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should contain "obs-track-test"
    And the MCP result should contain "\"version\":1"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | obs-track-test         |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Successful read operations are tracked
    Given I register an Avro schema for subject "obs-read-tracked"
    And I store the response field "id" as "schema_id"
    When I call MCP tool "get_schema_by_id" with input:
      | id | $schema_id |
    Then the MCP result should contain "string"
    When I call MCP tool "list_subjects"
    Then the MCP result should contain "obs-read-tracked"
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
      | context              | .                      |
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

  # ==========================================================================
  # 2. ERROR TOOL CALLS ARE TRACKED
  # ==========================================================================

  Scenario: Error on non-existent schema is tracked
    When I call MCP tool "get_schema_by_id" with input:
      | id | 888888 |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
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
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_schema_by_id       |
      | status_code          | 0                      |
      | reason               | *                      |
      | error                | *                      |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Error on non-existent subject is tracked
    When I call MCP tool "get_latest_schema" with input:
      | subject | obs-nonexistent-subject |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | obs-nonexistent-subject |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_latest_schema      |
      | status_code          | 0                      |
      | reason               | *                      |
      | error                | *                      |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Error on invalid schema registration is tracked
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "obs-invalid-schema",
        "schema": "this is not valid json or avro"
      }
      """
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | obs-invalid-schema     |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
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

  # ==========================================================================
  # 3. MULTIPLE OPERATIONS IN SEQUENCE
  # ==========================================================================

  Scenario: Multiple tool calls in sequence are all tracked
    # Call 1: health check
    When I call MCP tool "health_check"
    Then the MCP result should contain "healthy"
    # Call 2: get server info
    When I call MCP tool "get_server_info"
    Then the MCP result should contain "AVRO"
    # Call 3: register a schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "obs-sequence-test",
        "schema": "{\"type\":\"record\",\"name\":\"SeqTest\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Call 4: read back the schema
    When I call MCP tool "get_latest_schema" with input:
      | subject | obs-sequence-test |
    Then the MCP result should contain "SeqTest"
    # Call 5: config operation
    When I call MCP tool "get_config"
    Then the MCP result should contain "BACKWARD"
    # Call 6: list subjects
    When I call MCP tool "list_subjects"
    Then the MCP result should contain "obs-sequence-test"
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
      | context              | .                      |
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

  # ==========================================================================
  # 4. MIXED SUCCESS AND ERROR CALLS
  # ==========================================================================

  Scenario: Mix of successful and failed calls are tracked correctly
    # Success
    When I call MCP tool "health_check"
    Then the MCP result should contain "healthy"
    # Error
    When I call MCP tool "get_schema_by_id" with input:
      | id | 777777 |
    Then the MCP result should contain "error"
    # Success
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "obs-mixed-test",
        "schema": "{\"type\":\"record\",\"name\":\"Mixed\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Error — bad schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "obs-mixed-bad",
        "schema": "invalid{{"
      }
      """
    Then the MCP result should contain "error"
    # Success — read back
    When I call MCP tool "get_latest_schema" with input:
      | subject | obs-mixed-test |
    Then the MCP result should contain "Mixed"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | obs-mixed-test         |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
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

  # ==========================================================================
  # 5. ADMIN OPERATIONS ARE TRACKED
  # ==========================================================================

  Scenario: Admin tool calls are tracked
    When I call MCP tool "list_roles"
    Then the MCP result should contain "admin"
    When I call MCP tool "get_cluster_id"
    Then the MCP result should not contain "error"
    When I call MCP tool "get_server_version"
    Then the MCP result should not contain "error"
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
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_server_version     |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
