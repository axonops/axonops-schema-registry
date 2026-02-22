@auth
Feature: Authentication flows and RBAC
  As an operator, I want the schema registry to enforce authentication and
  role-based access control so that only authorized users can perform
  operations matching their role.

  The test server has auth enabled with basic + api_key methods, RBAC enabled,
  and a pre-seeded super_admin user: admin / admin-password (user ID=1).

  # ---------------------------------------------------------------------------
  # Authentication flows
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Valid basic auth succeeds on protected endpoint
    Given I authenticate as "admin" with password "admin-password"
    When I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: Invalid password returns 401
    Given I authenticate as "admin" with password "wrong-password"
    When I GET "/subjects"
    Then the response status should be 401

  @auth
  Scenario: Non-existent user returns 401
    Given I authenticate as "nonexistent" with password "any-password"
    When I GET "/subjects"
    Then the response status should be 401

  @auth
  Scenario: No auth header on protected endpoint returns 401
    Given I clear authentication
    When I GET "/subjects"
    Then the response status should be 401

  @auth
  Scenario: Public health endpoint works without auth
    Given I clear authentication
    When I GET "/"
    Then the response status should be 200

  @auth
  Scenario: API key auth via Basic format works
    Given I authenticate as "admin" with password "admin-password"
    And I create an API key with name "test-key" role "developer" expires_in 3600
    Then the response status should be 201
    And I store the response field "key" as "my_api_key"
    When I authenticate with stored API key "my_api_key"
    And I GET "/subjects"
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # RBAC - readonly role
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Readonly user can GET /subjects
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer" password "viewer-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "viewer" with password "viewer-pass"
    And I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: Readonly user cannot POST schema
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer2" password "viewer-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "viewer2" with password "viewer-pass"
    And I register a schema under subject "test-value":
      """
      {"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403

  @auth
  Scenario: Readonly user cannot DELETE subject
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer3" password "viewer-pass" role "readonly"
    Then the response status should be 201
    # First register a schema as admin so there is something to delete
    When I authenticate as "admin" with password "admin-password"
    And I register a schema under subject "to-delete-value":
      """
      {"type":"record","name":"ToDelete","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    When I authenticate as "viewer3" with password "viewer-pass"
    And I DELETE "/subjects/to-delete-value"
    Then the response status should be 403

  @auth
  Scenario: Readonly user can GET /config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer4" password "viewer-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "viewer4" with password "viewer-pass"
    And I GET "/config"
    Then the response status should be 200

  @auth
  Scenario: Readonly user cannot PUT /config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer5" password "viewer-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "viewer5" with password "viewer-pass"
    And I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  # ---------------------------------------------------------------------------
  # RBAC - developer role
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Developer can GET /subjects
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "dev1" password "dev-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "dev1" with password "dev-pass"
    And I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: Developer can POST schema
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "dev2" password "dev-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "dev2" with password "dev-pass"
    And I register a schema under subject "dev-test-value":
      """
      {"type":"record","name":"DevTest","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200

  @auth
  Scenario: Developer cannot DELETE subject
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "dev3" password "dev-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "admin" with password "admin-password"
    And I register a schema under subject "dev-nodelete-value":
      """
      {"type":"record","name":"DevNoDelete","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    When I authenticate as "dev3" with password "dev-pass"
    And I DELETE "/subjects/dev-nodelete-value"
    Then the response status should be 403

  @auth
  Scenario: Developer can GET /config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "dev4" password "dev-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "dev4" with password "dev-pass"
    And I GET "/config"
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # RBAC - admin role
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Admin can manage schemas (register and delete)
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "mgr1" password "mgr-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "mgr1" with password "mgr-pass"
    And I register a schema under subject "admin-test-value":
      """
      {"type":"record","name":"AdminTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    When I DELETE "/subjects/admin-test-value"
    Then the response status should be 200

  @auth
  Scenario: Admin can read admin endpoints
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "mgr2" password "mgr-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "mgr2" with password "mgr-pass"
    And I GET "/admin/users"
    Then the response status should be 200

  @auth
  Scenario: Admin cannot write admin endpoints
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "mgr3" password "mgr-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "mgr3" with password "mgr-pass"
    And I create a user with username "should-fail" password "nope" role "readonly"
    Then the response status should be 403

  # ---------------------------------------------------------------------------
  # Metrics (public endpoint, no auth needed)
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Metrics endpoint returns Prometheus format without auth
    Given I clear authentication
    When I get the metrics
    Then the response status should be 200
    And the response should contain "# HELP"

  @auth
  Scenario: Metrics contain schema_registry_requests_total
    Given I clear authentication
    # Make a request first to ensure the request counter is initialized
    When I GET "/"
    Then the response status should be 200
    When I get the metrics
    Then the response status should be 200
    And the response should contain Prometheus metric "schema_registry_requests_total"
