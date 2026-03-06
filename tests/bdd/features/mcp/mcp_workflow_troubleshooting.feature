@mcp @mcp-workflow
Feature: MCP Workflow — Troubleshooting
  Tests troubleshooting workflows from prompts/troubleshooting.md and
  prompts/debug-registration-error.md by executing MCP tool call sequences.

  # Validates: glossary/error-reference — 42201
  Scenario: Invalid schema returns error code 42201
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-trouble-invalid",
        "schema": "not valid json"
      }
      """
    Then the MCP result should be an error
    And the MCP result should contain "42201"

  # Validates: glossary/error-reference — 40401
  Scenario: Non-existent subject returns 40401
    When I call MCP tool "get_latest_schema" with JSON input:
      """
      {"subject": "wf-trouble-nonexistent-subject-xyz"}
      """
    Then the MCP result should be an error
    And the MCP result should contain "40401"

  # Validates: glossary/error-reference — 409, troubleshooting Registration failures
  Scenario: Incompatible schema returns 409 then explain failure
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-trouble-incompat",
        "schema": "{\"type\":\"record\",\"name\":\"Rec\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-trouble-incompat",
        "schema": "{\"type\":\"record\",\"name\":\"Rec\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should be an error
    And the MCP result should contain "409"
    When I call MCP tool "explain_compatibility_failure" with JSON input:
      """
      {
        "subject": "wf-trouble-incompat",
        "schema": "{\"type\":\"record\",\"name\":\"Rec\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should not be an error

  # Validates: troubleshooting — Subject not found, match_subjects fuzzy match
  Scenario: Match subjects fuzzy to find misspelled subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "orders-value",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "match_subjects" with JSON input:
      """
      {"pattern": "order*"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "orders-value"

  # Validates: glossary/error-reference — 40405, best-practices Soft-delete confusion
  Scenario: Permanent delete without soft-delete first returns error
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "wf-trouble-no-soft-delete-xyz", "permanent": true}
      """
    Then the MCP result should be an error
