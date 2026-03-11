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
    And the audit log should contain event "mcp_tool_call"

  Scenario: Get dependency graph nonexistent subject
    When I call MCP tool "get_dependency_graph" with JSON input:
      """
      {"subject": "nonexistent-depgraph-value", "version": 1}
      """
    Then the MCP result should contain "nonexistent-depgraph-value"
    And the audit log should contain event "mcp_tool_call"
