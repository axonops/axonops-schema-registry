@mcp
Feature: MCP Admin Tools
  The MCP server exposes admin tools for managing users, API keys, and roles
  when the auth service is configured.

  Scenario: List available roles
    When I call MCP tool "list_roles"
    Then the MCP result should contain "super_admin"
    And the MCP result should contain "admin"
    And the MCP result should contain "developer"
    And the MCP result should contain "readonly"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Create and get a user
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "alice", "password": "secret123", "role": "developer"}
      """
    Then the MCP result should contain "alice"
    And the MCP result should contain "developer"
    When I call MCP tool "list_users"
    Then the MCP result should contain "alice"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Update a user role
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "bob", "password": "secret123", "role": "developer"}
      """
    Then the MCP result should contain "bob"
    And I store the MCP result field "id" as "user_id"
    When I call MCP tool "update_user" with JSON input:
      """
      {"id": $user_id, "role": "admin"}
      """
    Then the MCP result should contain "admin"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Delete a user
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "charlie", "password": "secret123", "role": "readonly"}
      """
    Then the MCP result should contain "charlie"
    And I store the MCP result field "id" as "user_id"
    When I call MCP tool "delete_user" with input:
      | id | $user_id |
    Then the MCP result should contain "true"
    When I call MCP tool "list_users"
    Then the MCP result should not contain "charlie"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Create and list API keys
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "keyowner", "password": "secret123", "role": "developer"}
      """
    Then the MCP result should contain "keyowner"
    And I store the MCP result field "id" as "user_id"
    When I call MCP tool "create_apikey" with JSON input:
      """
      {"user_id": $user_id, "name": "my-key", "role": "developer", "expires_in": 3600}
      """
    Then the MCP result should contain "my-key"
    When I call MCP tool "list_apikeys"
    Then the MCP result should contain "my-key"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Revoke an API key
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "revokeowner", "password": "secret123", "role": "developer"}
      """
    And I store the MCP result field "id" as "user_id"
    When I call MCP tool "create_apikey" with JSON input:
      """
      {"user_id": $user_id, "name": "revoke-key", "role": "developer", "expires_in": 3600}
      """
    Then the MCP result should contain "revoke-key"
    And I store the MCP result field "id" as "key_id"
    When I call MCP tool "revoke_apikey" with input:
      | id | $key_id |
    Then the MCP result should contain "true"
    When I call MCP tool "get_apikey" with input:
      | id | $key_id |
    Then the MCP result should not contain "\"enabled\":true"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Change user password
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "pwuser", "password": "oldpass", "role": "developer"}
      """
    Then the MCP result should contain "pwuser"
    And I store the MCP result field "id" as "user_id"
    When I call MCP tool "change_password" with JSON input:
      """
      {"id": $user_id, "old_password": "oldpass", "new_password": "newpass"}
      """
    Then the MCP result should contain "true"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Get user by username
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "findme", "password": "secret123", "role": "admin"}
      """
    Then the MCP result should contain "findme"
    When I call MCP tool "get_user_by_username" with input:
      | username | findme |
    Then the MCP result should contain "findme"
    And the MCP result should contain "admin"
    And the audit log should contain event "mcp_tool_call"

  Scenario: Rotate an API key
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "rotateowner", "password": "secret123", "role": "developer"}
      """
    And I store the MCP result field "id" as "user_id"
    When I call MCP tool "create_apikey" with JSON input:
      """
      {"user_id": $user_id, "name": "rotate-key", "role": "developer", "expires_in": 3600}
      """
    Then the MCP result should contain "rotate-key"
    And I store the MCP result field "id" as "key_id"
    When I call MCP tool "rotate_apikey" with JSON input:
      """
      {"id": $key_id, "expires_in": 7200}
      """
    Then the MCP result should contain "key"
    And the MCP result should contain "rotated"
    And the audit log should contain event "mcp_tool_call"
