@mcp
Feature: MCP Schema Read Tools
  MCP tools for reading schemas and schema metadata.

  Background:
    Given I register an Avro schema for subject "mcp-read-test"
    And I store the response field "id" as "schema_id"

  Scenario: Get schema by ID
    When I call MCP tool "get_schema_by_id" with input:
      | id | $schema_id |
    Then the MCP result should contain "AVRO"
    And the MCP result should contain "string"

  Scenario: Get raw schema by ID
    When I call MCP tool "get_raw_schema_by_id" with input:
      | id | $schema_id |
    Then the MCP result should contain "string"

  Scenario: Get schema by subject and version
    When I call MCP tool "get_schema_version" with input:
      | subject | mcp-read-test |
      | version | 1             |
    Then the MCP result should contain "mcp-read-test"

  Scenario: Get raw schema by subject and version
    When I call MCP tool "get_raw_schema_version" with input:
      | subject | mcp-read-test |
      | version | 1             |
    Then the MCP result should contain "string"

  Scenario: Get latest schema
    When I call MCP tool "get_latest_schema" with input:
      | subject | mcp-read-test |
    Then the MCP result should contain "mcp-read-test"

  Scenario: List versions
    When I call MCP tool "list_versions" with input:
      | subject | mcp-read-test |
    Then the MCP result should contain "1"

  Scenario: Get subjects for schema
    When I call MCP tool "get_subjects_for_schema" with input:
      | id | $schema_id |
    Then the MCP result should contain "mcp-read-test"

  Scenario: Get versions for schema
    When I call MCP tool "get_versions_for_schema" with input:
      | id | $schema_id |
    Then the MCP result should contain "mcp-read-test"

  Scenario: Lookup schema by content
    When I call MCP tool "lookup_schema" with input:
      | subject | mcp-read-test     |
      | schema  | {"type":"string"} |
    Then the MCP result should contain "mcp-read-test"

  Scenario: Get schema types
    When I call MCP tool "get_schema_types"
    Then the MCP result should contain "AVRO"
    And the MCP result should contain "PROTOBUF"
    And the MCP result should contain "JSON"

  Scenario: List schemas
    When I call MCP tool "list_schemas"
    Then the MCP result should contain "mcp-read-test"

  Scenario: Get max schema ID
    When I call MCP tool "get_max_schema_id"
    Then the MCP result should contain "$schema_id"

  Scenario: Get schema by ID not found
    When I call MCP tool "get_schema_by_id" with input:
      | id | 99999 |
    Then the MCP result should contain "error"
