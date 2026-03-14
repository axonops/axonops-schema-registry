@auth @admin
Feature: Admin User Management
  As an administrator, I want to manage users via the admin API
  so that I can control access to the schema registry.

  The test server is pre-seeded with a super_admin user:
    username="admin", password="admin-password"

  @auth
  Scenario: Create a user as admin
    Given I authenticate as "admin" with password "admin-password"
    When I create a user with username "alice" password "alice-secret" role "developer"
    Then the response status should be 201
    And the response should be valid JSON
    And the response should have field "id"
    And the response field "username" should be "alice"
    And the response field "role" should be "developer"
    And the response field "enabled" should be true
    And the audit log should contain an event:
      | event_type           | user_create  |
      | outcome              | success      |
      | actor_id             | admin        |
      | actor_type           | user         |
      | auth_method          | basic        |
      | role                 | super_admin  |
      | target_type          | user         |
      | target_id            | alice        |
      | schema_id            |              |
      | version              |              |
      | schema_type          |              |
      | before_hash          |              |
      | after_hash           | sha256:*     |
      | context              |              |
      | transport_security   | tls          |
      | method               | POST         |
      | path                 | /admin/users |
      | status_code          | 201          |
      | reason               |              |
      | error                |              |
      | request_body         |              |
      | metadata             |              |
      | timestamp            | *            |
      | duration_ms          | *            |
      | request_id           | *            |
      | source_ip            | *            |
      | user_agent           | *            |

  @auth
  Scenario: Create a user with email
    Given I authenticate as "admin" with password "admin-password"
    When I create a user with username "bob" password "bob-secret" role "readonly" email "bob@example.com"
    Then the response status should be 201
    And the response should be valid JSON
    And the response field "username" should be "bob"
    And the response field "role" should be "readonly"
    And the response field "email" should be "bob@example.com"
    And the audit log should contain an event:
      | event_type           | user_create  |
      | outcome              | success      |
      | actor_id             | admin        |
      | actor_type           | user         |
      | auth_method          | basic        |
      | role                 | super_admin  |
      | target_type          | user         |
      | target_id            | bob          |
      | schema_id            |              |
      | version              |              |
      | schema_type          |              |
      | before_hash          |              |
      | after_hash           | sha256:*     |
      | context              |              |
      | transport_security   | tls          |
      | method               | POST         |
      | path                 | /admin/users |
      | status_code          | 201          |
      | reason               |              |
      | error                |              |
      | request_body         |              |
      | metadata             |              |
      | timestamp            | *            |
      | duration_ms          | *            |
      | request_id           | *            |
      | source_ip            | *            |
      | user_agent           | *            |

  @auth
  Scenario: List users includes created users
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "carol" password "carol-secret" role "developer"
    And I create a user with username "dave" password "dave-secret" role "readonly"
    When I list all users
    Then the response status should be 200
    And the response should be valid JSON
    And the response users array should have length 3

  @auth
  Scenario: Get user by ID
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "eve" password "eve-secret" role "developer"
    And I store the response field "id" as "user_id"
    When I get user by stored ID "user_id"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "username" should be "eve"
    And the response field "role" should be "developer"
    And the response field "id" should equal stored "user_id"

  @auth
  Scenario: Update user role
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "frank" password "frank-secret" role "readonly"
    And I store the response field "id" as "user_id"
    When I update user with stored ID "user_id" with:
      """
      {"role": "developer"}
      """
    Then the response status should be 200
    And the response field "role" should be "developer"
    And the response field "username" should be "frank"
    And the audit log should contain an event:
      | event_type           | user_update  |
      | outcome              | success      |
      | actor_id             | admin        |
      | actor_type           | user         |
      | auth_method          | basic        |
      | role                 | super_admin  |
      | target_type          | user         |
      | target_id            | *            |
      | schema_id            |              |
      | version              |              |
      | schema_type          |              |
      | before_hash          | sha256:*     |
      | after_hash           | sha256:*     |
      | context              |              |
      | transport_security   | tls          |
      | method               | PUT          |
      | path                 | /admin/users |
      | status_code          | 200          |
      | reason               |              |
      | error                |              |
      | request_body         |              |
      | metadata             |              |
      | timestamp            | *            |
      | duration_ms          | *            |
      | request_id           | *            |
      | source_ip            | *            |
      | user_agent           | *            |

  @auth
  Scenario: Update user to disable account
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "grace" password "grace-secret" role "developer"
    And I store the response field "id" as "user_id"
    When I update user with stored ID "user_id" with:
      """
      {"enabled": false}
      """
    Then the response status should be 200
    And the response field "enabled" should be false
    And the response field "username" should be "grace"
    And the audit log should contain an event:
      | event_type           | user_update  |
      | outcome              | success      |
      | actor_id             | admin        |
      | actor_type           | user         |
      | auth_method          | basic        |
      | role                 | super_admin  |
      | target_type          | user         |
      | target_id            | *            |
      | schema_id            |              |
      | version              |              |
      | schema_type          |              |
      | before_hash          | sha256:*     |
      | after_hash           | sha256:*     |
      | context              |              |
      | transport_security   | tls          |
      | method               | PUT          |
      | path                 | /admin/users |
      | status_code          | 200          |
      | reason               |              |
      | error                |              |
      | request_body         |              |
      | metadata             |              |
      | timestamp            | *            |
      | duration_ms          | *            |
      | request_id           | *            |
      | source_ip            | *            |
      | user_agent           | *            |

  @auth
  Scenario: Delete user
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "heidi" password "heidi-secret" role "readonly"
    And I store the response field "id" as "user_id"
    When I delete user with stored ID "user_id"
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type           | user_delete  |
      | outcome              | success      |
      | actor_id             | admin        |
      | actor_type           | user         |
      | auth_method          | basic        |
      | role                 | super_admin  |
      | target_type          | user         |
      | target_id            | *            |
      | schema_id            |              |
      | version              |              |
      | schema_type          |              |
      | before_hash          | sha256:*     |
      | after_hash           |              |
      | context              |              |
      | transport_security   | tls          |
      | method               | DELETE       |
      | path                 | /admin/users |
      | status_code          | 204          |
      | reason               |              |
      | error                |              |
      | request_body         |              |
      | metadata             |              |
      | timestamp            | *            |
      | duration_ms          | *            |
      | request_id           | *            |
      | source_ip            | *            |
      | user_agent           | *            |

  @auth
  Scenario: Duplicate username returns 409
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ivan" password "ivan-secret" role "developer"
    When I create a user with username "ivan" password "other-password" role "readonly"
    Then the response status should be 409

  @auth
  Scenario: Create user without authentication returns 401
    Given I clear authentication
    When I create a user with username "judy" password "judy-secret" role "readonly"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type           | auth_failure         |
      | outcome              | failure              |
      | actor_id             |                      |
      | actor_type           | anonymous            |
      | auth_method          |                      |
      | role                 |                      |
      | target_type          |                      |
      | target_id            |                      |
      | schema_id            |                      |
      | version              |                      |
      | schema_type          |                      |
      | before_hash          |                      |
      | after_hash           |                      |
      | context              |                      |
      | transport_security   | tls                  |
      | method               | POST                 |
      | path                 | /admin/users         |
      | status_code          | 401                  |
      | reason               | no_valid_credentials |
      | error                |                      |
      | request_body         |                      |
      | metadata             |                      |
      | timestamp            | *                    |
      | duration_ms          | *                    |
      | request_id           | *                    |
      | source_ip            | *                    |
      | user_agent           | *                    |

  @auth
  Scenario: Create user as readonly user returns 403
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "karl" password "karl-secret" role "readonly"
    When I authenticate as "karl" with password "karl-secret"
    And I create a user with username "liam" password "liam-secret" role "readonly"
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden    |
      | outcome              | failure           |
      | actor_id             | karl              |
      | actor_type           | user              |
      | auth_method          | basic             |
      | role                 | readonly          |
      | target_type          |                   |
      | target_id            |                   |
      | schema_id            |                   |
      | version              |                   |
      | schema_type          |                   |
      | before_hash          |                   |
      | after_hash           |                   |
      | context              |                   |
      | transport_security   | tls               |
      | method               | POST              |
      | path                 | /admin/users      |
      | status_code          | 403               |
      | reason               | permission_denied |
      | error                |                   |
      | request_body         |                   |
      | metadata             |                   |
      | timestamp            | *                 |
      | duration_ms          | *                 |
      | request_id           | *                 |
      | source_ip            | *                 |
      | user_agent           | *                 |

  @auth
  Scenario: Create user with missing username returns 400
    Given I authenticate as "admin" with password "admin-password"
    When I POST "/admin/users" with body:
      """
      {"password": "some-pass", "role": "readonly"}
      """
    Then the response status should be 400

  @auth
  Scenario: Create user with missing password returns 400
    Given I authenticate as "admin" with password "admin-password"
    When I POST "/admin/users" with body:
      """
      {"username": "no-pass-user", "role": "readonly"}
      """
    Then the response status should be 400

  @auth
  Scenario: Create user with invalid role returns 400
    Given I authenticate as "admin" with password "admin-password"
    When I create a user with username "mallory" password "mallory-secret" role "superduper"
    Then the response status should be 400

  @auth
  Scenario: Get non-existent user returns 404
    Given I authenticate as "admin" with password "admin-password"
    When I get user by ID "999999"
    Then the response status should be 404

  @auth
  Scenario: List roles returns the available roles
    Given I authenticate as "admin" with password "admin-password"
    When I list roles
    Then the response status should be 200
    And the response should be valid JSON
    And the response roles array should have length 4
