@auth
Feature: Config-defined API key authentication (memory storage)
  As an operator, I want to define API keys in the config file so that
  I can use API key authentication without requiring a database for
  simple single-server deployments.

  The test server has auth enabled with api_key method, RBAC enabled with
  default_role "readonly", and two config-defined API keys:
    - "test-apikey-readonly" with role "readonly"
    - "test-apikey-admin" with role "admin"

  A database super_admin user "admin" / "admin-password" is also pre-seeded.

  @auth
  Scenario: config-defined readonly API key can read subjects
    Given I authenticate with API key "test-apikey-readonly"
    When I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: config-defined readonly API key cannot write
    Given I authenticate with API key "test-apikey-readonly"
    When I register a schema under subject "test-memory-apikey":
      """
      {"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403

  @auth
  Scenario: config-defined admin API key can write
    Given I authenticate with API key "test-apikey-admin"
    When I register a schema under subject "test-memory-apikey-admin":
      """
      {"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  @auth
  Scenario: invalid API key gets 401
    Given I authenticate with API key "wrong-key"
    When I GET "/subjects"
    Then the response status should be 401

  @auth
  Scenario: second config-defined API key also works
    Given I authenticate with API key "test-apikey-admin"
    When I GET "/subjects"
    Then the response status should be 200
