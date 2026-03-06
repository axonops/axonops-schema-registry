@mcp @mcp-workflow
Feature: MCP Workflow — Team Onboarding
  Tests the multi-step workflow from prompts/team-onboarding.md by executing
  each step as MCP tool calls.

  # Validates: prompts/team-onboarding.md — Steps 1-2, glossary/contexts What Contexts Isolate
  Scenario: Create context, register in it, verify default context isolation
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-team-test",
        "schema": "{\"type\":\"string\"}",
        "context": ".team-alpha"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_subjects" with JSON input:
      """
      {"context": ".team-alpha"}
      """
    Then the MCP result should contain "wf-team-test"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not contain "wf-team-test"

  # Validates: prompts/team-onboarding.md — Step 3, glossary/contexts 4-Tier Inheritance
  Scenario: Set context-level compatibility default
    When I call MCP tool "set_config" with JSON input:
      """
      {
        "compatibility": "FULL_TRANSITIVE",
        "context": ".team-beta"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_config" with JSON input:
      """
      {"context": ".team-beta"}
      """
    Then the MCP result should contain "FULL_TRANSITIVE"

  # Validates: prompts/team-onboarding.md — Step 4, glossary/auth-and-security
  Scenario: Create user with write role and API key
    When I call MCP tool "create_user" with JSON input:
      """
      {
        "username": "wf-team-user",
        "password": "SecureP@ss1234!",
        "role": "write"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "wf-team-user"
    When I call MCP tool "create_apikey" with JSON input:
      """
      {
        "user_id": "wf-team-user",
        "name": "wf-team-key"
      }
      """
    Then the MCP result should not be an error

  # Validates: prompts/team-onboarding.md — Step 8
  Scenario: Context-scoped resources return only team data
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-team-gamma-schema",
        "schema": "{\"type\":\"string\"}",
        "context": ".team-gamma"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-default-schema",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I read MCP resource "schema://contexts/.team-gamma/subjects"
    Then the MCP resource result should contain "wf-team-gamma-schema"
    And the MCP resource result should not contain "wf-default-schema"

  # Validates: prompts/team-onboarding.md — Complete workflow
  Scenario: Full onboarding end-to-end
    When I call MCP tool "set_config" with JSON input:
      """
      {"compatibility": "BACKWARD_TRANSITIVE", "context": ".team-delta"}
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-team-delta-events",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "context": ".team-delta"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_subjects" with JSON input:
      """
      {"context": ".team-delta"}
      """
    Then the MCP result should contain "wf-team-delta-events"
    When I call MCP tool "get_config" with JSON input:
      """
      {"context": ".team-delta"}
      """
    Then the MCP result should contain "BACKWARD_TRANSITIVE"
    When I read MCP resource "schema://contexts/.team-delta/subjects"
    Then the MCP resource result should contain "wf-team-delta-events"
