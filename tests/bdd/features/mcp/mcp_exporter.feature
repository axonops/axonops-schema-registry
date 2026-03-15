@mcp
Feature: MCP Exporter Tools
  MCP tools for managing schema exporters (Schema Linking) that replicate
  schemas to a destination schema registry.

  Scenario: Create and get exporter
    When I call MCP tool "create_exporter" with JSON input:
      """
      {"name":"mcp-exp-test","context_type":"AUTO","subjects":["orders-value"],"config":{"schema.registry.url":"http://dest:8081"}}
      """
    Then the MCP result should contain "mcp-exp-test"
    When I call MCP tool "get_exporter" with input:
      | name | mcp-exp-test |
    Then the MCP result should contain "mcp-exp-test"
    Then the MCP result should contain "AUTO"
    Then the MCP result should contain "orders-value"
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
      | path                 | get_exporter           |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: List exporters
    When I call MCP tool "create_exporter" with JSON input:
      """
      {"name":"mcp-exp-a","context_type":"AUTO"}
      """
    When I call MCP tool "create_exporter" with JSON input:
      """
      {"name":"mcp-exp-b","context_type":"AUTO"}
      """
    When I call MCP tool "list_exporters"
    Then the MCP result should contain "mcp-exp-a"
    Then the MCP result should contain "mcp-exp-b"
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
      | path                 | list_exporters         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Delete exporter
    When I call MCP tool "create_exporter" with JSON input:
      """
      {"name":"mcp-exp-del","context_type":"AUTO"}
      """
    When I call MCP tool "delete_exporter" with input:
      | name | mcp-exp-del |
    Then the MCP result should contain "true"
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
      | path                 | delete_exporter        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Pause and resume exporter
    When I call MCP tool "create_exporter" with JSON input:
      """
      {"name":"mcp-exp-status","context_type":"AUTO"}
      """
    When I call MCP tool "get_exporter_status" with input:
      | name | mcp-exp-status |
    Then the MCP result should contain "PAUSED"
    When I call MCP tool "resume_exporter" with input:
      | name | mcp-exp-status |
    Then the MCP result should contain "RUNNING"
    When I call MCP tool "pause_exporter" with input:
      | name | mcp-exp-status |
    Then the MCP result should contain "PAUSED"
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
      | path                 | pause_exporter         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Reset exporter
    When I call MCP tool "create_exporter" with JSON input:
      """
      {"name":"mcp-exp-reset","context_type":"AUTO"}
      """
    When I call MCP tool "reset_exporter" with input:
      | name | mcp-exp-reset |
    Then the MCP result should contain "reset"
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
      | path                 | reset_exporter         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Get and update exporter config
    When I call MCP tool "create_exporter" with JSON input:
      """
      {"name":"mcp-exp-cfg","context_type":"AUTO","config":{"schema.registry.url":"http://original:8081"}}
      """
    When I call MCP tool "get_exporter_config" with input:
      | name | mcp-exp-cfg |
    Then the MCP result should contain "original"
    When I call MCP tool "update_exporter_config" with JSON input:
      """
      {"name":"mcp-exp-cfg","config":{"schema.registry.url":"http://updated:8081"}}
      """
    When I call MCP tool "get_exporter_config" with input:
      | name | mcp-exp-cfg |
    Then the MCP result should contain "updated"
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
      | path                 | get_exporter_config    |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
