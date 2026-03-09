@mcp @mcp-metrics @metrics
Feature: MCP Metrics
  MCP tool invocations MUST be tracked by Prometheus metrics so that
  operators can monitor MCP usage alongside REST API usage.

  # ---------------------------------------------------------------------------
  # Basic tool call metrics
  # ---------------------------------------------------------------------------

  Scenario: MCP tool calls are counted in Prometheus metrics
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
    And the Prometheus metric "schema_registry_mcp_tool_calls_total" with labels "tool=\"list_subjects\"" should exist

  Scenario: MCP tool call duration is tracked
    When I call MCP tool "get_schema_types" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
    And the Prometheus metric "schema_registry_mcp_tool_call_duration_seconds_count" with labels "tool=\"get_schema_types\"" should exist

  # ---------------------------------------------------------------------------
  # Confirmation metrics (two-phase flow)
  # ---------------------------------------------------------------------------

  @mcp-confirmation
  Scenario: MCP confirmation token_issued metric fires on dry_run
    Given MCP confirmations are enabled
    And I register an Avro schema for subject "metrics-confirm-test"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "metrics-confirm-test", "permanent": false}
      """
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "metrics-confirm-test", "permanent": true, "dry_run": true}
      """
    Then the MCP result should contain "confirm_token"
    And the Prometheus metric "schema_registry_mcp_confirmations_total" with labels "outcome=\"token_issued\"" should exist

  @mcp-confirmation
  Scenario: MCP confirmation confirmed metric fires on token use
    Given MCP confirmations are enabled
    And I register an Avro schema for subject "metrics-confirm-exec"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "metrics-confirm-exec", "permanent": false}
      """
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "metrics-confirm-exec", "permanent": true, "dry_run": true}
      """
    And I store the MCP result field "confirm_token" as "token"
    When I call MCP tool "delete_subject" with JSON input using stored "token":
      """
      {"subject": "metrics-confirm-exec", "permanent": true}
      """
    Then the MCP result should not be an error
    And the Prometheus metric "schema_registry_mcp_confirmations_total" with labels "outcome=\"confirmed\"" should exist

  @mcp-confirmation
  Scenario: MCP policy denial metric fires when confirmation missing
    Given MCP confirmations are enabled
    And I register an Avro schema for subject "metrics-policy-deny"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "metrics-policy-deny", "permanent": false}
      """
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "metrics-policy-deny", "permanent": true}
      """
    Then the MCP result should contain "confirmation_required"
    And the Prometheus metric "schema_registry_mcp_policy_denials_total" with labels "reason=\"confirmation_required\"" should exist

  # ---------------------------------------------------------------------------
  # Permission denied metrics
  # ---------------------------------------------------------------------------

  @mcp-permissions @preset-readonly
  Scenario: MCP permission denied metric fires for blocked tools
    Given MCP permission preset is "readonly"
    When I list MCP tools
    Then the MCP result should not contain "register_schema"
    And the Prometheus metric "schema_registry_mcp_permission_denied_total" with labels "tool=\"register_schema\"" should exist
