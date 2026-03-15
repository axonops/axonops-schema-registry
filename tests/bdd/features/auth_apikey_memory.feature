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
    And the audit log should contain an event:
      | event_type           | auth_forbidden                        |
      | outcome              | failure                               |
      | actor_id             | *                                     |
      | actor_type           | api_key                               |
      | auth_method          | api_key                               |
      | role                 | readonly                              |
      | target_type          | subject                               |
      | target_id            | test-memory-apikey                    |
      | schema_id            |                                       |
      | version              |                                       |
      | schema_type          |                                       |
      | before_hash          |                                       |
      | after_hash           |                                       |
      | context              | .                                     |
      | transport_security   | tls                                   |
      | method               | POST                                  |
      | path                 | /subjects/test-memory-apikey/versions |
      | status_code          | 403                                   |
      | reason               | permission_denied                     |
      | error                |                                       |
      | request_body         |                                       |
      | metadata             |                                       |
      | timestamp            | *                                     |
      | duration_ms          | *                                     |
      | request_id           | *                                     |
      | source_ip            | *                                     |
      | user_agent           | *                                     |

  @auth
  Scenario: config-defined admin API key can write
    Given I authenticate with API key "test-apikey-admin"
    When I register a schema under subject "test-memory-apikey-admin":
      """
      {"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register              |
      | outcome              | success                      |
      | actor_id             | *                            |
      | actor_type           | api_key                      |
      | auth_method          | api_key                      |
      | role                 | admin                        |
      | target_type          | subject                      |
      | target_id            | test-memory-apikey-admin     |
      | schema_id            | *                            |
      | version              | *                            |
      | schema_type          | AVRO                         |
      | before_hash          |                              |
      | after_hash           | sha256:*                     |
      | context              | .                            |
      | transport_security   | tls                          |
      | method               | POST                         |
      | path                 | /subjects/test-memory-apikey-admin/versions |
      | status_code          | 200                          |
      | reason               |                              |
      | error                |                              |
      | request_body         |                              |
      | metadata             |                              |
      | timestamp            | *                            |
      | duration_ms          | *                            |
      | request_id           | *                            |
      | source_ip            | *                            |
      | user_agent           | *                            |

  @auth
  Scenario: invalid API key gets 401
    Given I authenticate with API key "wrong-key"
    When I GET "/subjects"
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
      | context              | .                    |
      | transport_security   | tls                  |
      | method               | GET                  |
      | path                 | /subjects            |
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
  Scenario: second config-defined API key also works
    Given I authenticate with API key "test-apikey-admin"
    When I GET "/subjects"
    Then the response status should be 200
