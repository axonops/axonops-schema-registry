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

  Scenario: Create and get a user
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "alice", "password": "secret123", "role": "developer"}
      """
    Then the MCP result should contain "alice"
    And the MCP result should contain "developer"
    When I call MCP tool "list_users"
    Then the MCP result should contain "alice"

  Scenario: Update a user role
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "bob", "password": "secret123", "role": "developer"}
      """
    Then the MCP result should contain "bob"
    When I call MCP tool "update_user" with JSON input:
      """
      {"id": 1, "role": "admin"}
      """
    Then the MCP result should contain "admin"

  Scenario: Delete a user
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "charlie", "password": "secret123", "role": "readonly"}
      """
    Then the MCP result should contain "charlie"
    When I call MCP tool "delete_user" with input:
      | id | 1 |
    Then the MCP result should contain "true"
    When I call MCP tool "list_users"
    Then the MCP result should not contain "charlie"

  Scenario: Create and list API keys
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "keyowner", "password": "secret123", "role": "developer"}
      """
    Then the MCP result should contain "keyowner"
    When I call MCP tool "create_apikey" with JSON input:
      """
      {"user_id": 1, "name": "my-key", "role": "developer", "expires_in": 3600}
      """
    Then the MCP result should contain "my-key"
    When I call MCP tool "list_apikeys"
    Then the MCP result should contain "my-key"

  Scenario: Revoke an API key
    When I call MCP tool "create_user" with JSON input:
      """
      {"username": "revokeowner", "password": "secret123", "role": "developer"}
      """
    When I call MCP tool "create_apikey" with JSON input:
      """
      {"user_id": 1, "name": "revoke-key", "role": "developer", "expires_in": 3600}
      """
    Then the MCP result should contain "revoke-key"
    When I call MCP tool "revoke_apikey" with input:
      | id | 1 |
    Then the MCP result should contain "true"
    When I call MCP tool "get_apikey" with input:
      | id | 1 |
    Then the MCP result should not contain "\"enabled\":true"
