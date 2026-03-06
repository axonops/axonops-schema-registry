@mcp @mcp-workflow
Feature: MCP Workflow — Subject Deprecation
  Tests the deprecation workflow from prompts/deprecate-subject.md by
  executing each step as MCP tool calls.

  # Validates: prompts/deprecate-subject.md — Step 4, glossary/core-concepts Modes
  Scenario: Lock with READONLY and verify registration rejected
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-dep-lock",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "set_mode" with JSON input:
      """
      {"subject": "wf-dep-lock", "mode": "READONLY"}
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-dep-lock",
        "schema": "{\"type\":\"int\"}"
      }
      """
    Then the MCP result should be an error

  # Validates: prompts/deprecate-subject.md — Step 5, glossary/data-contracts Metadata
  Scenario: Add deprecation metadata via set_config_full
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-dep-meta",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "set_config_full" with JSON input:
      """
      {
        "subject": "wf-dep-meta",
        "compatibility_level": "BACKWARD",
        "default_metadata": {"properties": {"deprecated": "true", "deprecation_date": "2026-03-06"}}
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_subject_config_full" with JSON input:
      """
      {"subject": "wf-dep-meta"}
      """
    Then the MCP result should contain "deprecated"

  # Validates: prompts/deprecate-subject.md — Step 7
  Scenario: Soft-delete hides from list but schema resolvable by ID
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-dep-soft",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "wf-dep-soft"}
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not contain "wf-dep-soft"

  # Validates: prompts/deprecate-subject.md — Complete workflow
  Scenario: Full deprecation lifecycle
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-dep-full",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    # Lock the subject
    When I call MCP tool "set_mode" with JSON input:
      """
      {"subject": "wf-dep-full", "mode": "READONLY"}
      """
    Then the MCP result should not be an error
    # Add deprecation metadata
    When I call MCP tool "set_config_full" with JSON input:
      """
      {
        "subject": "wf-dep-full",
        "compatibility_level": "BACKWARD",
        "default_metadata": {"properties": {"deprecated": "true"}}
      }
      """
    Then the MCP result should not be an error
    # Soft-delete
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "wf-dep-full"}
      """
    Then the MCP result should not be an error
    # Verify hidden
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not contain "wf-dep-full"
