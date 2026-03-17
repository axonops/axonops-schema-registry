@functional @axonops-only
Feature: Delete Global Mode Configuration
  As a schema registry administrator
  I want to reset the global mode to the default
  So that I can restore normal read-write operations after maintenance or import operations

  Background:
    Given the schema registry is running

  Scenario: DELETE /mode resets global mode to READWRITE
    Given I set the global mode to "READONLY"
    And I get the global mode
    And the response field "mode" should be "READONLY"
    When I DELETE "/mode?force=true"
    Then the response status should be 200
    When I get the global mode
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_delete                   |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | _global                       |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | sha256:*                      |
      | after_hash           |                               |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | DELETE                        |
      | path                 | /mode                         |
      | status_code          | 200                           |
      | reason               |                               |
      | error                |                               |
      | request_body         |                               |
      | metadata             |                               |
      | timestamp            | *                             |
      | duration_ms          | *                             |
      | request_id           | *                             |
      | source_ip            | *                             |
      | user_agent           | *                             |

  Scenario: DELETE /mode response contains previous mode
    Given I set the global mode to "IMPORT"
    When I DELETE "/mode?force=true"
    Then the response status should be 200
    And the response field "mode" should be "IMPORT"
    And the audit log should contain an event:
      | event_type           | mode_delete                   |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | _global                       |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | sha256:*                      |
      | after_hash           |                               |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | DELETE                        |
      | path                 | /mode                         |
      | status_code          | 200                           |
      | reason               |                               |
      | error                |                               |
      | request_body         |                               |
      | metadata             |                               |
      | timestamp            | *                             |
      | duration_ms          | *                             |
      | request_id           | *                             |
      | source_ip            | *                             |
      | user_agent           | *                             |

  Scenario: Subject-level mode NOT affected by global reset
    Given I set the global mode to "READONLY"
    And I PUT "/mode/test-subject?force=true" with body:
      """
      {
        "mode": "IMPORT"
      }
      """
    And the response status should be 200
    And I GET "/mode/test-subject"
    And the response field "mode" should be "IMPORT"
    When I DELETE "/mode?force=true"
    Then the response status should be 200
    When I GET "/mode/test-subject"
    Then the response status should be 200
    And the response field "mode" should be "IMPORT"
    And the audit log should contain an event:
      | event_type           | mode_delete                   |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | _global                       |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | sha256:*                      |
      | after_hash           |                               |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | DELETE                        |
      | path                 | /mode                         |
      | status_code          | 200                           |
      | reason               |                               |
      | error                |                               |
      | request_body         |                               |
      | metadata             |                               |
      | timestamp            | *                             |
      | duration_ms          | *                             |
      | request_id           | *                             |
      | source_ip            | *                             |
      | user_agent           | *                             |

  Scenario: DELETE /mode when already READWRITE is idempotent
    Given I get the global mode
    And the response field "mode" should be "READWRITE"
    When I DELETE "/mode?force=true"
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"
    When I DELETE "/mode?force=true"
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_delete                   |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | _global                       |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | sha256:*                      |
      | after_hash           |                               |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | DELETE                        |
      | path                 | /mode                         |
      | status_code          | 200                           |
      | reason               |                               |
      | error                |                               |
      | request_body         |                               |
      | metadata             |                               |
      | timestamp            | *                             |
      | duration_ms          | *                             |
      | request_id           | *                             |
      | source_ip            | *                             |
      | user_agent           | *                             |

  Scenario: DELETE /mode allows writes after READONLY was set
    Given I set the global mode to "READONLY"
    And I POST "/subjects/test-writes/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    And the response status should be 422
    When I DELETE "/mode?force=true"
    Then the response status should be 200
    When I POST "/subjects/test-writes/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And the audit log should contain an event:
      | event_type           | schema_register                  |
      | outcome              | success                          |
      | actor_id             |                                  |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | test-writes                      |
      | schema_id            | *                                |
      | version              | *                                |
      | schema_type          | AVRO                             |
      | before_hash          |                                  |
      | after_hash           | sha256:*                         |
      | context              | .                                |
      | transport_security   | tls                              |
      | method               | POST                             |
      | path                 | /subjects/test-writes/versions   |
      | status_code          | 200                              |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           | *                                |
      | source_ip            | *                                |
      | user_agent           | *                                |
