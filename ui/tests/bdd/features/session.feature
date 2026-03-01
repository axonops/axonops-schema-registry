Feature: Session management
  As a logged-in user
  I want my session to be validated and refreshed
  So that I can use the system without re-authenticating

  Background:
    Given the UI server is running
    And the default admin user exists

  Scenario: Session endpoint returns user info
    Given I am logged in as "admin"
    When I check my session
    Then the response status should be 200
    And the response should contain username "admin"

  Scenario: Session endpoint refreshes the cookie
    Given I am logged in as "admin"
    When I check my session
    Then a session cookie should be set

  Scenario: Session endpoint requires authentication
    When I check my session without a cookie
    Then the response status should be 401

  Scenario: Multiple users can have concurrent sessions
    Given user "alice" exists with password "alice123"
    And I am logged in as "admin"
    And another user is logged in as "alice"
    When I check my session
    Then the response should contain username "admin"
    When the other user checks their session
    Then the response should contain username "alice"

  Scenario: Auth config endpoint is publicly accessible
    When I request the auth config
    Then the response status should be 200
    And the response should indicate auth is enabled
