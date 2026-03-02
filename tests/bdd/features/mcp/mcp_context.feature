@mcp
Feature: MCP Context & Import Tools
  MCP tools for multi-tenant contexts and schema import.

  Scenario: List contexts
    When I call MCP tool "list_contexts"
    Then the MCP result should contain "."

  Scenario: Import a schema with preserved ID
    When I call MCP tool "set_mode" with input:
      | mode  | IMPORT |
      | force | true   |
    When I call MCP tool "import_schemas" with JSON input:
      """
      {"schemas":[{"id":100,"subject":"mcp-import-test","version":1,"schema":"{\"type\":\"string\"}"}]}
      """
    Then the MCP result should contain "1"
