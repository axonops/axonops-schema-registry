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
      | context              | .                     |
      | transport_security   | tls                  |
      | method               | GET                  |
      | path                 | /me                  |
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

  Scenario: Change password successfully
    When I POST "/me/password" with body:
      """
      {"old_password": "old-pass-123", "new_password": "new-pass-456"}
      """
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type           | password_change    |
      | outcome              | success            |
      | actor_id             | selfservice-user   |
      | actor_type           | user               |
      | auth_method          | basic              |
      | role                 | developer          |
      | target_type          | user               |
      | target_id            | selfservice-user   |
      | schema_id            |                    |
      | version              |                    |
      | schema_type          |                    |
      | before_hash          |                    |
      | after_hash           |                    |
      | context              | .                   |
      | transport_security   | tls                |
      | method               | POST               |
      | path                 | /me/password       |
      | status_code          | 204                |
      | reason               |                    |
      | error                |                    |
      | request_body         |                    |
      | metadata             |                    |
      | timestamp            | *                  |
      | duration_ms          | *                  |
      | request_id           | *                  |
      | source_ip            | *                  |
      | user_agent           | *                  |
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
    And the audit log should contain an event:
      | event_type           | auth_forbidden    |
      | outcome              | failure           |
      | actor_id             | selfservice-user  |
      | actor_type           | user              |
      | auth_method          | basic             |
      | role                 | developer         |
      | target_type          |                   |
      | target_id            |                   |
      | schema_id            |                   |
      | version              |                   |
      | schema_type          |                   |
      | before_hash          |                   |
      | after_hash           |                   |
      | context              | .                  |
      | transport_security   | tls               |
      | method               | POST              |
      | path                 | /me/password      |
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
      | context              | .                     |
      | transport_security   | tls                  |
      | method               | POST                 |
      | path                 | /me/password         |
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
