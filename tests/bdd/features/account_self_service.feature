@auth @account
Feature: Self-service account management
  As an authenticated user, I want to view my account info and change my password
  without requiring admin privileges.

  Background:
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "selfservice-user" password "old-pass-123" role "developer"
    And the response status should be 201
    And I authenticate as "selfservice-user" with password "old-pass-123"

  Scenario: GET /me returns current user info
    When I GET "/me"
    Then the response status should be 200
    And the response field "username" should be "selfservice-user"
    And the response field "role" should be "developer"

  Scenario: GET /me without authentication returns 401
    Given I clear authentication
    When I GET "/me"
    Then the response status should be 401

  Scenario: Change password successfully
    When I POST "/me/password" with body:
      """
      {"old_password": "old-pass-123", "new_password": "new-pass-456"}
      """
    Then the response status should be 204
    # Verify new password works
    Given I authenticate as "selfservice-user" with password "new-pass-456"
    When I GET "/me"
    Then the response status should be 200
    And the response field "username" should be "selfservice-user"

  Scenario: Change password with wrong old password returns 403
    When I POST "/me/password" with body:
      """
      {"old_password": "wrong-password", "new_password": "new-pass-456"}
      """
    Then the response status should be 403

  Scenario: Change password with empty new password returns 400
    When I POST "/me/password" with body:
      """
      {"old_password": "old-pass-123", "new_password": ""}
      """
    Then the response status should be 400

  Scenario: Change password without authentication returns 401
    Given I clear authentication
    When I POST "/me/password" with body:
      """
      {"old_password": "old-pass-123", "new_password": "new-pass-456"}
      """
    Then the response status should be 401
