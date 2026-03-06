@mcp @mcp-workflow
Feature: MCP Workflow — Cross-Cutting Change
  Tests the cross-cutting change workflow from prompts/cross-cutting-change.md
  by executing each step as MCP tool calls.

  # Validates: prompts/cross-cutting-change.md — Step 2
  Scenario: Find schemas by field across multiple registered schemas
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-xcut-orders",
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"customer_id\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-xcut-payments",
        "schema": "{\"type\":\"record\",\"name\":\"Payment\",\"fields\":[{\"name\":\"customer_id\",\"type\":\"string\"},{\"name\":\"method\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "find_schemas_by_field" with JSON input:
      """
      {"field_name": "customer_id"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "wf-xcut-orders"
    And the MCP result should contain "wf-xcut-payments"

  # Validates: prompts/cross-cutting-change.md — Step 3
  Scenario: Check field consistency for shared field
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-xcut-cons-a",
        "schema": "{\"type\":\"record\",\"name\":\"A\",\"fields\":[{\"name\":\"tenant_id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-xcut-cons-b",
        "schema": "{\"type\":\"record\",\"name\":\"B\",\"fields\":[{\"name\":\"tenant_id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_field_consistency" with JSON input:
      """
      {
        "field_name": "tenant_id",
        "subjects": ["wf-xcut-cons-a", "wf-xcut-cons-b"]
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "tenant_id"

  # Validates: prompts/cross-cutting-change.md — Step 6
  Scenario: Check compatibility multi for change across subjects
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-xcut-multi-a",
        "schema": "{\"type\":\"record\",\"name\":\"EventA\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-xcut-multi-b",
        "schema": "{\"type\":\"record\",\"name\":\"EventB\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_compatibility_multi" with JSON input:
      """
      {
        "checks": [
          {
            "subject": "wf-xcut-multi-a",
            "schema": "{\"type\":\"record\",\"name\":\"EventA\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"ts\",\"type\":[\"null\",\"long\"],\"default\":null}]}"
          },
          {
            "subject": "wf-xcut-multi-b",
            "schema": "{\"type\":\"record\",\"name\":\"EventB\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"ts\",\"type\":[\"null\",\"long\"],\"default\":null}]}"
          }
        ]
      }
      """
    Then the MCP result should not be an error
