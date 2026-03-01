Feature: Schema Registry proxy
  As a UI user
  I want the UI server to proxy requests to the Schema Registry
  So that I can browse schemas without direct SR access

  Background:
    Given the UI server is running
    And the mock Schema Registry is running
    And I am logged in as "admin"

  Scenario: Proxy lists subjects
    When I request "/api/v1/subjects" via the proxy
    Then the response status should be 200
    And the response should contain "test-topic"

  Scenario: Proxy fetches schema by ID
    When I request "/api/v1/schemas/ids/1" via the proxy
    Then the response status should be 200

  Scenario: Proxy fetches subject versions
    When I request "/api/v1/subjects/test-topic/versions" via the proxy
    Then the response status should be 200

  Scenario: Proxy fetches global config
    When I request "/api/v1/config" via the proxy
    Then the response status should be 200

  Scenario: Proxy rejects unauthenticated requests
    Given I am not authenticated
    When I request "/api/v1/subjects" without authentication
    Then the response status should be 401

  Scenario: Proxy returns 502 when SR is down
    Given the Schema Registry is stopped
    When I request "/api/v1/subjects" via the proxy
    Then the response status should be 502
    And the response should contain "unavailable"

  Scenario: Proxy injects API token
    Given the UI is configured with API token "my-secret-token"
    When I request "/api/v1/subjects" via the proxy
    Then the Schema Registry should receive the authorization header

  Scenario: Proxy strips UI cookies before forwarding
    When I request "/api/v1/subjects" via the proxy
    Then the Schema Registry should not receive any cookies
