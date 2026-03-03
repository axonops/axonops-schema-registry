@mcp
Feature: MCP Server
  The schema registry exposes an MCP server for AI assistant integration.
  MCP tools provide programmatic access to registry operations.

  Scenario: Health check via MCP
    When I call MCP tool "health_check"
    Then the MCP result should contain "healthy"

  Scenario: Get server info via MCP
    When I call MCP tool "get_server_info"
    Then the MCP result should contain "AVRO"
    And the MCP result should contain "PROTOBUF"
    And the MCP result should contain "JSON"

  Scenario: List subjects when empty
    When I call MCP tool "list_subjects"
    Then the MCP result should be "[]"

  Scenario: List subjects after registration
    Given I register an Avro schema for subject "mcp-test-subject"
    When I call MCP tool "list_subjects"
    Then the MCP result should contain "mcp-test-subject"

  Scenario: List subjects with prefix filter
    Given I register an Avro schema for subject "orders-value"
    And I register an Avro schema for subject "users-value"
    When I call MCP tool "list_subjects" with input:
      | prefix | orders |
    Then the MCP result should contain "orders-value"
    And the MCP result should not contain "users-value"

  Scenario: List subjects with regex pattern filter
    Given I register an Avro schema for subject "orders-value"
    And I register an Avro schema for subject "users-value"
    And I register an Avro schema for subject "orders-key"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {"pattern": ".*-value$"}
      """
    Then the MCP result should contain "orders-value"
    And the MCP result should contain "users-value"
    And the MCP result should not contain "orders-key"

  Scenario: List subjects with prefix and pattern combined
    Given I register an Avro schema for subject "orders-value"
    And I register an Avro schema for subject "users-value"
    And I register an Avro schema for subject "orders-key"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {"prefix": "orders", "pattern": ".*-value$"}
      """
    Then the MCP result should contain "orders-value"
    And the MCP result should not contain "users-value"
    And the MCP result should not contain "orders-key"
