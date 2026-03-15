@mcp
Feature: MCP Dependency Graph
  MCP tool for building schema dependency graphs.

  Scenario: Get dependency graph with no references
    Given subject "depgraph-user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I call MCP tool "get_dependency_graph" with JSON input:
      """
      {"subject": "depgraph-user-value", "version": 1}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "depgraph-user-value"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | depgraph-user-value        |
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
      | path                 | get_dependency_graph       |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

  Scenario: Get dependency graph nonexistent subject
    When I call MCP tool "get_dependency_graph" with JSON input:
      """
      {"subject": "nonexistent-depgraph-value", "version": 1}
      """
    Then the MCP result should contain "nonexistent-depgraph-value"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call                  |
      | outcome              | success                        |
      | actor_id             | mcp-anonymous                  |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | subject                        |
      | target_id            | nonexistent-depgraph-value     |
      | schema_id            |                                |
      | version              |                                |
      | schema_type          |                                |
      | before_hash          |                                |
      | after_hash           |                                |
      | context              |                                |
      | transport_security   |                                |
      | source_ip            |                                |
      | user_agent           |                                |
      | method               | MCP                            |
      | path                 | get_dependency_graph           |
      | status_code          | 0                              |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           |                                |
