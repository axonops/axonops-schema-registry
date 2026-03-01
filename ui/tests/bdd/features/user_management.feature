Feature: User management
  As an admin
  I want to manage user accounts
  So that I can control who has access to the UI

  Background:
    Given the UI server is running
    And the default admin user exists
    And I am logged in as "admin"

  Scenario: List users shows default admin
    When I list all users
    Then the response status should be 200
    And the user list should contain 1 user
    And the user list should include "admin"

  Scenario: Create a new user
    When I create user "alice" with password "alice123"
    Then the response status should be 201
    When I list all users
    Then the user list should contain 2 users

  Scenario: Create user with short username fails
    When I create user "a" with password "password"
    Then the response status should be 400

  Scenario: Create user with short password fails
    When I create user "testuser" with password "ab"
    Then the response status should be 400

  Scenario: Create duplicate user fails
    When I create user "admin" with password "password"
    Then the response status should be 409

  Scenario: Disable a user
    Given user "bob" exists with password "bob12345"
    When I disable user "bob"
    Then the response status should be 204

  Scenario: Cannot disable the last active user
    When I disable user "admin"
    Then the response status should be 409

  Scenario: Re-enable a disabled user
    Given user "bob" exists with password "bob12345"
    And user "bob" is disabled
    When I enable user "bob"
    Then the response status should be 204

  Scenario: Change user password
    Given user "bob" exists with password "bob12345"
    When I change the password for "bob" to "newpass99"
    Then the response status should be 204

  Scenario: Delete a user
    Given user "bob" exists with password "bob12345"
    When I delete user "bob"
    Then the response status should be 204
    When I list all users
    Then the user list should contain 1 user

  Scenario: Cannot delete the last user
    When I delete user "admin"
    Then the response status should be 409

  Scenario: Delete non-existent user returns 404
    When I delete user "ghost"
    Then the response status should be 404

  Scenario: Change my own password
    When I change my password from "admin" to "newadmin1"
    Then the response status should be 204

  Scenario: User management requires authentication
    Given I am not authenticated
    When I list all users without authentication
    Then the response status should be 401
