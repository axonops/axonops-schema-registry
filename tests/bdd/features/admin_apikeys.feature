@auth @admin
Feature: Admin API Key Management
  As an administrator, I want to manage API keys so that services can authenticate programmatically

  Background:
    Given the schema registry is running
    And I authenticate as "admin" with password "admin-password"

  @auth
  Scenario: Create an API key
    When I create an API key with name "my-service-key" role "admin" expires_in 86400
    Then the response status should be 201
    And the response should be valid JSON
    And the response should have field "id"
    And the response should have field "key"
    And the response should have field "key_prefix"
    And the response field "name" should be "my-service-key"
    And the response field "role" should be "admin"
    And the response field "enabled" should be true
    And the response field "key" should not be empty
    And the response field "key_prefix" should not be empty

  @auth
  Scenario: List API keys
    Given I create an API key with name "list-key-1" role "admin" expires_in 86400
    And the response status should be 201
    And I create an API key with name "list-key-2" role "readonly" expires_in 86400
    And the response status should be 201
    When I list all API keys
    Then the response status should be 200
    And the response should be valid JSON
    And the response apikeys array should have length 2

  @auth
  Scenario: Get API key by ID
    Given I create an API key with name "get-me-key" role "developer" expires_in 86400
    And the response status should be 201
    And I store the response field "id" as "key_id"
    When I get API key by stored ID "key_id"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "get-me-key"
    And the response field "role" should be "developer"
    And the response field "enabled" should be true

  @auth
  Scenario: Update API key name
    Given I create an API key with name "old-name" role "admin" expires_in 86400
    And the response status should be 201
    And I store the response field "id" as "key_id"
    When I update API key with stored ID "key_id" with:
      """
      {"name": "new-name"}
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "new-name"
    And the response field "role" should be "admin"

  @auth
  Scenario: Disable API key via update
    Given I create an API key with name "disable-me" role "admin" expires_in 86400
    And the response status should be 201
    And I store the response field "id" as "key_id"
    When I update API key with stored ID "key_id" with:
      """
      {"enabled": false}
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "enabled" should be false
    And the response field "name" should be "disable-me"

  @auth
  Scenario: Delete API key
    Given I create an API key with name "delete-me" role "admin" expires_in 86400
    And the response status should be 201
    And I store the response field "id" as "key_id"
    When I delete API key with stored ID "key_id"
    Then the response status should be 204

  @auth
  Scenario: Revoke API key sets enabled to false
    Given I create an API key with name "revoke-me" role "admin" expires_in 86400
    And the response status should be 201
    And I store the response field "id" as "key_id"
    When I revoke API key with stored ID "key_id"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "enabled" should be false
    And the response field "name" should be "revoke-me"

  @auth
  Scenario: Rotate API key returns new key and revokes old
    Given I create an API key with name "rotate-me" role "admin" expires_in 86400
    And the response status should be 201
    And I store the response field "id" as "old_key_id"
    And I store the response field "key" as "old_key_value"
    When I rotate API key with stored ID "old_key_id" expires_in 86400
    Then the response status should be 200
    And the response should be valid JSON
    And the response should have field "new_key"
    And the response should have field "revoked_id"

  @auth
  Scenario: Duplicate API key name returns 409
    Given I create an API key with name "unique-name" role "admin" expires_in 86400
    And the response status should be 201
    When I create an API key with name "unique-name" role "admin" expires_in 86400
    Then the response status should be 409

  @auth
  Scenario: Create API key without authentication returns 401
    Given I clear authentication
    When I create an API key with name "no-auth-key" role "admin" expires_in 86400
    Then the response status should be 401

  @auth
  Scenario: Create API key as readonly user returns 403
    Given I create a user with username "viewer" password "viewer-password" role "readonly"
    And the response status should be 201
    And I authenticate as "viewer" with password "viewer-password"
    When I create an API key with name "forbidden-key" role "readonly" expires_in 86400
    Then the response status should be 403

  @auth
  Scenario: Create API key with missing name returns 400
    When I POST "/admin/apikeys" with body:
      """
      {"role": "admin", "expires_in": 86400}
      """
    Then the response status should be 400
    And the response should contain "Name is required"

  @auth
  Scenario: Create API key with missing role returns 400
    When I POST "/admin/apikeys" with body:
      """
      {"name": "no-role-key", "expires_in": 86400}
      """
    Then the response status should be 400
    And the response should contain "Role is required"

  @auth
  Scenario: Get non-existent API key returns 404
    When I GET "/admin/apikeys/99999"
    Then the response status should be 404

  @auth
  Scenario: Authenticate with a created API key to access schemas
    Given I create an API key with name "schema-access-key" role "admin" expires_in 86400
    And the response status should be 201
    And I store the response field "key" as "api_key"
    When I authenticate with stored API key "api_key"
    And I list all subjects
    Then the response status should be 200
