@mcp
Feature: MCP Metadata, Alias & Advanced Tools
  MCP tools for full configuration with data contracts, subject alias
  resolution, bulk schema retrieval, write mode checks, and schema formatting.

  Scenario: Get full config returns complete record
    When I call MCP tool "get_config_full"
    Then the MCP result should contain "compatibilityLevel"
    Then the MCP result should contain "BACKWARD"

  Scenario: Set config with alias and retrieve via get_config_full
    When I call MCP tool "set_config_full" with JSON input:
      """
      {"subject":"mcp-alias-src","compatibility_level":"BACKWARD","alias":"mcp-alias-target"}
      """
    Then the MCP result should contain "BACKWARD"
    When I call MCP tool "get_config_full" with input:
      | subject | mcp-alias-src |
    Then the MCP result should contain "mcp-alias-target"

  Scenario: Set config with metadata and retrieve via get_subject_config_full
    When I call MCP tool "set_config_full" with JSON input:
      """
      {"subject":"mcp-meta-subj","compatibility_level":"FULL","default_metadata":{"properties":{"owner":"team-data"}}}
      """
    When I call MCP tool "get_subject_config_full" with input:
      | subject | mcp-meta-subj |
    Then the MCP result should contain "team-data"
    Then the MCP result should contain "FULL"

  Scenario: Resolve alias
    When I call MCP tool "set_config_full" with JSON input:
      """
      {"subject":"mcp-resolve-src","compatibility_level":"BACKWARD","alias":"mcp-resolve-dest"}
      """
    When I call MCP tool "resolve_alias" with input:
      | subject | mcp-resolve-src |
    Then the MCP result should contain "mcp-resolve-dest"

  Scenario: Resolve alias with no alias returns self
    When I call MCP tool "resolve_alias" with input:
      | subject | mcp-no-alias |
    Then the MCP result should contain "mcp-no-alias"

  Scenario: Get schemas by subject returns all versions
    Given I register an Avro schema for subject "mcp-multi-ver"
    When I call MCP tool "get_schemas_by_subject" with input:
      | subject | mcp-multi-ver |
    Then the MCP result should contain "mcp-multi-ver"

  Scenario: Check write mode when writable
    When I call MCP tool "check_write_mode"
    Then the MCP result should contain "true"

  Scenario: Check write mode when readonly
    When I call MCP tool "set_mode" with input:
      | subject | mcp-ro-check |
      | mode    | READONLY     |
    When I call MCP tool "check_write_mode" with input:
      | subject | mcp-ro-check |
    Then the MCP result should contain "READONLY"

  Scenario: Format schema
    Given I register an Avro schema for subject "mcp-format-test"
    When I call MCP tool "format_schema" with input:
      | subject | mcp-format-test |
      | version | 1               |
    Then the MCP result should contain "mcp-format-test"
    Then the MCP result should contain "string"

  Scenario: Get global config direct
    When I call MCP tool "get_global_config_direct"
    Then the MCP result should contain "BACKWARD"
