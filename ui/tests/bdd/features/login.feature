Feature: User authentication
  As a UI user
  I want to log in and out of the Schema Registry UI
  So that I can securely access the system

  Background:
    Given the UI server is running
    And the default admin user exists

  Scenario: Successful login with valid credentials
    When I login as "admin" with password "admin"
    Then the response status should be 200
    And the response should contain username "admin"
    And a session cookie should be set

  Scenario: Login fails with wrong password
    When I login as "admin" with password "wrongpassword"
    Then the response status should be 401

  Scenario: Login fails with empty username
    When I login as "" with password "admin"
    Then the response status should be 400

  Scenario: Login fails with empty password
    When I login as "admin" with password ""
    Then the response status should be 400

  Scenario: Login fails for non-existent user
    When I login as "nobody" with password "secret"
    Then the response status should be 401

  Scenario: Login fails for disabled user
    Given user "bob" exists with password "bob12345"
    And user "bob" is disabled
    When I login as "bob" with password "bob12345"
    Then the response status should be 401

  Scenario: Successful logout
    Given I am logged in as "admin"
    When I logout
    Then the response status should be 204
    And the session cookie should be cleared
