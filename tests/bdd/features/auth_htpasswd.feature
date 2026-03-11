@auth
Feature: htpasswd file authentication
  As an operator, I want to authenticate users via an Apache-style htpasswd
  file so that I can manage credentials in a simple, file-based format
  without requiring a database.

  The test server has auth enabled with basic method, RBAC enabled with
  default_role "readonly", and an htpasswd file containing:
    - htuser1 / htpassword1
    - htuser2 / htpassword2

  A database super_admin user "admin" / "admin-password" is also pre-seeded.

  @auth
  Scenario: htpasswd user can authenticate and read subjects
    Given I authenticate as "htuser1" with password "htpassword1"
    When I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: htpasswd user with wrong password gets 401
    Given I authenticate as "htuser1" with password "wrong-password"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type  | auth_failure         |
      | outcome     | failure              |
      | actor_type  | anonymous            |
      | reason      | no_valid_credentials |
      | method      | GET                  |
      | path        | /subjects            |
      | status_code | 401                  |

  @auth
  Scenario: htpasswd user not in file gets 401
    Given I authenticate as "nonexistent" with password "any-password"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type  | auth_failure         |
      | outcome     | failure              |
      | actor_type  | anonymous            |
      | reason      | no_valid_credentials |
      | method      | GET                  |
      | path        | /subjects            |
      | status_code | 401                  |

  @auth
  Scenario: htpasswd user gets default readonly role
    Given I authenticate as "htuser1" with password "htpassword1"
    When I register a schema under subject "test-htpasswd":
      """
      {"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type  | auth_forbidden    |
      | outcome     | failure           |
      | actor_id    | htuser1           |
      | actor_type  | user              |
      | auth_method | basic             |
      | role        | readonly          |
      | reason      | permission_denied |
      | method      | POST              |
      | path        | /subjects/test-htpasswd/versions |
      | status_code | 403               |

  @auth
  Scenario: database user takes priority over htpasswd user
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "htuser1" password "db-password" role "admin"
    Then the response status should be 201
    # Database user should authenticate with DB password, not htpasswd password
    When I authenticate as "htuser1" with password "db-password"
    And I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: second htpasswd user also works
    Given I authenticate as "htuser2" with password "htpassword2"
    When I GET "/subjects"
    Then the response status should be 200
