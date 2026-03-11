@mcp
Feature: MCP Schema Write Tools
  MCP tools for writing and managing schemas.

  Scenario: Register a schema
    When I call MCP tool "register_schema" with input:
      | subject | mcp-write-test    |
      | schema  | {"type":"string"} |
    Then the MCP result should contain "mcp-write-test"
    And the MCP result should contain "\"version\":1"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Delete a subject
    Given I register an Avro schema for subject "mcp-delete-subj"
    When I call MCP tool "delete_subject" with input:
      | subject | mcp-delete-subj |
    Then the MCP result should contain "1"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Delete a version
    Given I register an Avro schema for subject "mcp-delete-ver"
    When I call MCP tool "delete_version" with input:
      | subject | mcp-delete-ver |
      | version | 1              |
    Then the MCP result should contain "1"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Check compatibility passes
    Given I register an Avro schema for subject "mcp-compat-test"
    When I call MCP tool "check_compatibility" with input:
      | subject | mcp-compat-test   |
      | schema  | {"type":"string"} |
    Then the MCP result should contain "true"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Check compatibility fails
    Given I register an Avro schema for subject "mcp-incompat-test"
    When I call MCP tool "check_compatibility" with input:
      | subject | mcp-incompat-test |
      | schema  | {"type":"int"}    |
    Then the MCP result should contain "false"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Register schema with explicit type
    When I call MCP tool "register_schema" with input:
      | subject     | mcp-json-test                     |
      | schema      | {"type":"object","properties":{}} |
      | schema_type | JSON                              |
    Then the MCP result should contain "mcp-json-test"
    And the MCP result should contain "JSON"
    And the audit log should contain event "mcp_tool_call"
