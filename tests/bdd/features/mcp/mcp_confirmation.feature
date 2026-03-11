@mcp @mcp-confirmation
Feature: MCP Two-Phase Confirmation for Destructive Operations
  When confirmations are enabled, destructive MCP operations MUST go through
  a dry_run/confirm_token two-phase flow to prevent accidental data loss.

  Background:
    Given MCP confirmations are enabled

  # --- delete_subject ---

  Scenario: Permanent delete subject requires confirmation
    Given I register an Avro schema for subject "confirm-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "confirm-sub", "permanent": false}
      """
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "confirm-sub", "permanent": true}
      """
    Then the MCP result should contain "confirmation_required"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Soft delete does NOT require confirmation
    Given I register an Avro schema for subject "soft-del-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "soft-del-sub", "permanent": false}
      """
    Then the MCP result should not contain "confirmation_required"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Dry run returns confirmation token
    Given I register an Avro schema for subject "dry-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "dry-sub", "permanent": true, "dry_run": true}
      """
    Then the MCP result should contain "confirmation_required"
    And the MCP result should contain "confirm_token"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Full flow - dry_run then confirm executes operation
    Given I register an Avro schema for subject "flow-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "flow-sub", "permanent": false}
      """
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "flow-sub", "permanent": true, "dry_run": true}
      """
    Then the MCP result should contain "confirm_token"
    And I store the MCP result field "confirm_token" as "token"
    When I call MCP tool "delete_subject" with JSON input using stored "token":
      """
      {"subject": "flow-sub", "permanent": true}
      """
    Then the MCP result should not contain "confirmation_required"
    And the MCP result should not contain "error"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Token cannot be reused
    Given I register an Avro schema for subject "reuse-sub"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "reuse-sub", "permanent": false}
      """
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "reuse-sub", "permanent": true, "dry_run": true}
      """
    And I store the MCP result field "confirm_token" as "token"
    When I call MCP tool "delete_subject" with JSON input using stored "token":
      """
      {"subject": "reuse-sub", "permanent": true}
      """
    Then the MCP result should not contain "confirmation_required"
    # Second use of same token should fail
    When I call MCP tool "delete_subject" with JSON input using stored "token":
      """
      {"subject": "reuse-sub", "permanent": true}
      """
    Then the MCP result should contain "error"
    And the audit log should contain event "mcp_tool_error"

  Scenario: Token scoped to exact args
    Given I register an Avro schema for subject "scope-a"
    And I register an Avro schema for subject "scope-b"
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "scope-a", "permanent": false}
      """
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "scope-b", "permanent": false}
      """
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "scope-a", "permanent": true, "dry_run": true}
      """
    And I store the MCP result field "confirm_token" as "token"
    # Use token for a different subject
    When I call MCP tool "delete_subject" with JSON input using stored "token":
      """
      {"subject": "scope-b", "permanent": true}
      """
    Then the MCP result should contain "error"
    And the MCP result should contain "does not match"
    And the audit log should contain event "mcp_tool_error"

  # --- import_schemas ---

  Scenario: Import schemas always requires confirmation
    When I call MCP tool "import_schemas" with JSON input:
      """
      {"schemas": [{"id": 1, "subject": "imp-sub", "version": 1, "schema": "{\"type\":\"string\"}", "schema_type": "AVRO"}]}
      """
    Then the MCP result should contain "confirmation_required"
    And the audit log should contain event "mcp_tool_call"

  # --- set_mode ---

  Scenario: Set mode to IMPORT requires confirmation
    When I call MCP tool "set_mode" with JSON input:
      """
      {"mode": "IMPORT"}
      """
    Then the MCP result should contain "confirmation_required"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Set mode to READWRITE does NOT require confirmation
    When I call MCP tool "set_mode" with JSON input:
      """
      {"mode": "READWRITE"}
      """
    Then the MCP result should not contain "confirmation_required"
    And the audit log should contain event "mcp_tool_call"

  # --- delete_config ---

  Scenario: Delete global config requires confirmation
    When I call MCP tool "delete_config" with JSON input:
      """
      {}
      """
    Then the MCP result should contain "confirmation_required"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Delete subject-level config does NOT require confirmation
    Given I register an Avro schema for subject "cfg-sub"
    When I call MCP tool "delete_config" with JSON input:
      """
      {"subject": "cfg-sub"}
      """
    Then the MCP result should not contain "confirmation_required"
    And the audit log should contain event "mcp_tool_call"
