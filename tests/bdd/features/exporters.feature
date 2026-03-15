@functional @axonops-only
Feature: Exporters API
  The Exporters API provides Confluent Schema Linking compatible functionality
  for exporting schemas to remote schema registries. Exporters can be created,
  configured, paused, resumed, reset, and deleted.

  Background:
    Given the schema registry is running

  # List Operations
  Scenario: List exporters when none exist
    When I GET "/exporters"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 0

  Scenario: List exporters after creating several
    When I POST "/exporters" with body:
      """
      {
        "name": "exporter-1",
        "contextType": "AUTO",
        "subjects": ["test-1"]
      }
      """
    And I POST "/exporters" with body:
      """
      {
        "name": "exporter-2",
        "contextType": "NONE",
        "subjects": ["test-2"]
      }
      """
    And I POST "/exporters" with body:
      """
      {
        "name": "exporter-3",
        "contextType": "AUTO",
        "subjects": ["test-3"]
      }
      """
    And I GET "/exporters"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 3
    And the response array should contain "exporter-1"
    And the response array should contain "exporter-2"
    And the response array should contain "exporter-3"
    And the audit log should contain an event:
      | event_type          | exporter_create |
      | outcome             | success         |
      | actor_id            |                 |
      | actor_type          | anonymous       |
      | auth_method         |                 |
      | role                |                 |
      | method              | POST            |
      | path                | /exporters      |
      | status_code         | 200             |
      | target_type         | exporter        |
      | target_id           | exporter-1      |
      | schema_id           |                 |
      | version             |                 |
      | schema_type         |                 |
      | context             |                 |
      | before_hash         |                 |
      | after_hash          | sha256:*        |
      | transport_security  | tls             |
      | reason              |                 |
      | error               |                 |
      | request_body        |                 |
      | metadata            |                 |
      | timestamp           | *               |
      | duration_ms         | *               |
      | request_id          | *               |
      | source_ip           | *               |
      | user_agent          | *               |

  # Create Operations
  Scenario: Create exporter with all fields
    When I POST "/exporters" with body:
      """
      {
        "name": "full-exporter",
        "contextType": "CUSTOM",
        "context": "prod",
        "subjects": ["orders-value", "users-value"],
        "subjectRenameFormat": "staging.${subject}",
        "config": {
          "schema.registry.url": "http://remote:8081",
          "basic.auth.credentials.source": "USER_INFO",
          "basic.auth.user.info": "admin:secret"
        }
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "full-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create |
      | outcome             | success         |
      | actor_id            |                 |
      | actor_type          | anonymous       |
      | auth_method         |                 |
      | role                |                 |
      | method              | POST            |
      | path                | /exporters      |
      | status_code         | 200             |
      | target_type         | exporter        |
      | target_id           | full-exporter   |
      | schema_id           |                 |
      | version             |                 |
      | schema_type         |                 |
      | context             |                 |
      | before_hash         |                 |
      | after_hash          | sha256:*        |
      | transport_security  | tls             |
      | reason              |                 |
      | error               |                 |
      | request_body        |                 |
      | metadata            |                 |
      | timestamp           | *               |
      | duration_ms         | *               |
      | request_id          | *               |
      | source_ip           | *               |
      | user_agent          | *               |

  Scenario: Create exporter with minimal fields
    When I POST "/exporters" with body:
      """
      {
        "name": "minimal-exporter"
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "minimal-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create    |
      | outcome             | success            |
      | actor_id            |                    |
      | actor_type          | anonymous          |
      | auth_method         |                    |
      | role                |                    |
      | method              | POST               |
      | path                | /exporters         |
      | status_code         | 200                |
      | target_type         | exporter           |
      | target_id           | minimal-exporter   |
      | schema_id           |                    |
      | version             |                    |
      | schema_type         |                    |
      | context             |                    |
      | before_hash         |                    |
      | after_hash          | sha256:*           |
      | transport_security  | tls                |
      | reason              |                    |
      | error               |                    |
      | request_body        |                    |
      | metadata            |                    |
      | timestamp           | *                  |
      | duration_ms         | *                  |
      | request_id          | *                  |
      | source_ip           | *                  |
      | user_agent          | *                  |

  Scenario: Create exporter with contextType AUTO
    When I POST "/exporters" with body:
      """
      {
        "name": "auto-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "auto-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create |
      | outcome             | success         |
      | actor_id            |                 |
      | actor_type          | anonymous       |
      | auth_method         |                 |
      | role                |                 |
      | method              | POST            |
      | path                | /exporters      |
      | status_code         | 200             |
      | target_type         | exporter        |
      | target_id           | auto-exporter   |
      | schema_id           |                 |
      | version             |                 |
      | schema_type         |                 |
      | context             |                 |
      | before_hash         |                 |
      | after_hash          | sha256:*        |
      | transport_security  | tls             |
      | reason              |                 |
      | error               |                 |
      | request_body        |                 |
      | metadata            |                 |
      | timestamp           | *               |
      | duration_ms         | *               |
      | request_id          | *               |
      | source_ip           | *               |
      | user_agent          | *               |

  Scenario: Create exporter with contextType CUSTOM and context
    When I POST "/exporters" with body:
      """
      {
        "name": "custom-exporter",
        "contextType": "CUSTOM",
        "context": "production",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "custom-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create  |
      | outcome             | success          |
      | actor_id            |                  |
      | actor_type          | anonymous        |
      | auth_method         |                  |
      | role                |                  |
      | method              | POST             |
      | path                | /exporters       |
      | status_code         | 200              |
      | target_type         | exporter         |
      | target_id           | custom-exporter  |
      | schema_id           |                  |
      | version             |                  |
      | schema_type         |                  |
      | context             |                  |
      | before_hash         |                  |
      | after_hash          | sha256:*         |
      | transport_security  | tls              |
      | reason              |                  |
      | error               |                  |
      | request_body        |                  |
      | metadata            |                  |
      | timestamp           | *                |
      | duration_ms         | *                |
      | request_id          | *                |
      | source_ip           | *                |
      | user_agent          | *                |

  Scenario: Create exporter with contextType NONE
    When I POST "/exporters" with body:
      """
      {
        "name": "none-exporter",
        "contextType": "NONE",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "none-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create |
      | outcome             | success         |
      | actor_id            |                 |
      | actor_type          | anonymous       |
      | auth_method         |                 |
      | role                |                 |
      | method              | POST            |
      | path                | /exporters      |
      | status_code         | 200             |
      | target_type         | exporter        |
      | target_id           | none-exporter   |
      | schema_id           |                 |
      | version             |                 |
      | schema_type         |                 |
      | context             |                 |
      | before_hash         |                 |
      | after_hash          | sha256:*        |
      | transport_security  | tls             |
      | reason              |                 |
      | error               |                 |
      | request_body        |                 |
      | metadata            |                 |
      | timestamp           | *               |
      | duration_ms         | *               |
      | request_id          | *               |
      | source_ip           | *               |
      | user_agent          | *               |

  Scenario: Create exporter with subject filters
    When I POST "/exporters" with body:
      """
      {
        "name": "filtered-exporter",
        "contextType": "AUTO",
        "subjects": ["orders-*", "users-*", "events-*"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "filtered-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create    |
      | outcome             | success            |
      | actor_id            |                    |
      | actor_type          | anonymous          |
      | auth_method         |                    |
      | role                |                    |
      | method              | POST               |
      | path                | /exporters         |
      | status_code         | 200                |
      | target_type         | exporter           |
      | target_id           | filtered-exporter  |
      | schema_id           |                    |
      | version             |                    |
      | schema_type         |                    |
      | context             |                    |
      | before_hash         |                    |
      | after_hash          | sha256:*           |
      | transport_security  | tls                |
      | reason              |                    |
      | error               |                    |
      | request_body        |                    |
      | metadata            |                    |
      | timestamp           | *                  |
      | duration_ms         | *                  |
      | request_id          | *                  |
      | source_ip           | *                  |
      | user_agent          | *                  |

  Scenario: Create exporter with multiple subjects
    When I POST "/exporters" with body:
      """
      {
        "name": "multi-subject-exporter",
        "contextType": "AUTO",
        "subjects": ["subject-1", "subject-2", "subject-3", "subject-4", "subject-5"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "multi-subject-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create          |
      | outcome             | success                  |
      | actor_id            |                          |
      | actor_type          | anonymous                |
      | auth_method         |                          |
      | role                |                          |
      | method              | POST                     |
      | path                | /exporters               |
      | status_code         | 200                      |
      | target_type         | exporter                 |
      | target_id           | multi-subject-exporter   |
      | schema_id           |                          |
      | version             |                          |
      | schema_type         |                          |
      | context             |                          |
      | before_hash         |                          |
      | after_hash          | sha256:*                 |
      | transport_security  | tls                      |
      | reason              |                          |
      | error               |                          |
      | request_body        |                          |
      | metadata            |                          |
      | timestamp           | *                        |
      | duration_ms         | *                        |
      | request_id          | *                        |
      | source_ip           | *                        |
      | user_agent          | *                        |

  Scenario: Create exporter with empty config
    When I POST "/exporters" with body:
      """
      {
        "name": "empty-config-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "config": {}
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "empty-config-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create        |
      | outcome             | success                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | method              | POST                   |
      | path                | /exporters             |
      | status_code         | 200                    |
      | target_type         | exporter               |
      | target_id           | empty-config-exporter  |
      | schema_id           |                        |
      | version             |                        |
      | schema_type         |                        |
      | context             |                        |
      | before_hash         |                        |
      | after_hash          | sha256:*               |
      | transport_security  | tls                    |
      | reason              |                        |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |

  Scenario: Create exporter returns name in response
    When I POST "/exporters" with body:
      """
      {
        "name": "response-check-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "response-check-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_create            |
      | outcome             | success                    |
      | actor_id            |                            |
      | actor_type          | anonymous                  |
      | auth_method         |                            |
      | role                |                            |
      | method              | POST                       |
      | path                | /exporters                 |
      | status_code         | 200                        |
      | target_type         | exporter                   |
      | target_id           | response-check-exporter    |
      | schema_id           |                            |
      | version             |                            |
      | schema_type         |                            |
      | context             |                            |
      | before_hash         |                            |
      | after_hash          | sha256:*                   |
      | transport_security  | tls                        |
      | reason              |                            |
      | error               |                            |
      | request_body        |                            |
      | metadata            |                            |
      | timestamp           | *                          |
      | duration_ms         | *                          |
      | request_id          | *                          |
      | source_ip           | *                          |
      | user_agent          | *                          |

  Scenario: Create duplicate exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "duplicate-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I POST "/exporters" with body:
      """
      {
        "name": "duplicate-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 409
    And the response should be valid JSON
    And the response should have error code 40950
    And the audit log should contain an event:
      | event_type          | exporter_create    |
      | outcome             | failure            |
      | actor_id            |                    |
      | actor_type          | anonymous          |
      | auth_method         |                    |
      | role                |                    |
      | method              | POST               |
      | path                | /exporters         |
      | status_code         | 409                |
      | target_type         | exporter           |
      | target_id           | duplicate-exporter |
      | schema_id           |                    |
      | version             |                    |
      | schema_type         |                    |
      | context             |                    |
      | before_hash         |                    |
      | after_hash          |                    |
      | transport_security  | tls                |
      | reason              | already_exists     |
      | error               |                    |
      | request_body        |                    |
      | metadata            |                    |
      | timestamp           | *                  |
      | duration_ms         | *                  |
      | request_id          | *                  |
      | source_ip           | *                  |
      | user_agent          | *                  |

  Scenario: Create exporter with invalid contextType
    When I POST "/exporters" with body:
      """
      {
        "name": "invalid-context-exporter",
        "contextType": "INVALID",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 422
    And the response should be valid JSON

  Scenario: Create exporter with empty name
    When I POST "/exporters" with body:
      """
      {
        "name": "",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    Then the response status should be 422
    And the response should be valid JSON

  # Get Operations
  Scenario: Get exporter by name
    When I POST "/exporters" with body:
      """
      {
        "name": "get-test-exporter",
        "contextType": "AUTO",
        "context": "",
        "subjects": ["test-value"],
        "subjectRenameFormat": "${subject}",
        "config": {
          "schema.registry.url": "http://remote:8081"
        }
      }
      """
    And I GET "/exporters/get-test-exporter"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "get-test-exporter"
    And the response field "contextType" should be "AUTO"

  Scenario: Get non-existent exporter
    When I GET "/exporters/non-existent-exporter"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Update Operations
  Scenario: Update exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "update-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I PUT "/exporters/update-test-exporter" with body:
      """
      {
        "contextType": "CUSTOM",
        "context": "updated",
        "subjects": ["new-test-value"],
        "subjectRenameFormat": "updated.${subject}"
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "update-test-exporter"
    And the audit log should contain an event:
      | event_type          | exporter_update                 |
      | outcome             | success                         |
      | actor_id            |                                 |
      | actor_type          | anonymous                       |
      | auth_method         |                                 |
      | role                |                                 |
      | method              | PUT                             |
      | path                | /exporters/update-test-exporter |
      | status_code         | 200                             |
      | target_type         | exporter                        |
      | target_id           | update-test-exporter            |
      | schema_id           |                                 |
      | version             |                                 |
      | schema_type         |                                 |
      | context             |                                 |
      | before_hash         | sha256:*                        |
      | after_hash          | sha256:*                        |
      | transport_security  | tls                             |
      | reason              |                                 |
      | error               |                                 |
      | request_body        |                                 |
      | metadata            |                                 |
      | timestamp           | *                               |
      | duration_ms         | *                               |
      | request_id          | *                               |
      | source_ip           | *                               |
      | user_agent          | *                               |

  Scenario: Update exporter subjects
    When I POST "/exporters" with body:
      """
      {
        "name": "subject-update-exporter",
        "contextType": "AUTO",
        "subjects": ["old-subject"]
      }
      """
    And I PUT "/exporters/subject-update-exporter" with body:
      """
      {
        "subjects": ["new-subject-1", "new-subject-2"]
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the audit log should contain an event:
      | event_type          | exporter_update                    |
      | outcome             | success                            |
      | actor_id            |                                    |
      | actor_type          | anonymous                          |
      | auth_method         |                                    |
      | role                |                                    |
      | method              | PUT                                |
      | path                | /exporters/subject-update-exporter |
      | status_code         | 200                                |
      | target_type         | exporter                           |
      | target_id           | subject-update-exporter            |
      | schema_id           |                                    |
      | version             |                                    |
      | schema_type         |                                    |
      | context             |                                    |
      | before_hash         | sha256:*                           |
      | after_hash          | sha256:*                           |
      | transport_security  | tls                                |
      | reason              |                                    |
      | error               |                                    |
      | request_body        |                                    |
      | metadata            |                                    |
      | timestamp           | *                                  |
      | duration_ms         | *                                  |
      | request_id          | *                                  |
      | source_ip           | *                                  |
      | user_agent          | *                                  |

  Scenario: Update exporter without changing all fields
    When I POST "/exporters" with body:
      """
      {
        "name": "partial-update-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "subjectRenameFormat": "${subject}"
      }
      """
    And I PUT "/exporters/partial-update-exporter" with body:
      """
      {
        "contextType": "NONE"
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the audit log should contain an event:
      | event_type          | exporter_update                    |
      | outcome             | success                            |
      | actor_id            |                                    |
      | actor_type          | anonymous                          |
      | auth_method         |                                    |
      | role                |                                    |
      | method              | PUT                                |
      | path                | /exporters/partial-update-exporter |
      | status_code         | 200                                |
      | target_type         | exporter                           |
      | target_id           | partial-update-exporter            |
      | schema_id           |                                    |
      | version             |                                    |
      | schema_type         |                                    |
      | context             |                                    |
      | before_hash         | sha256:*                           |
      | after_hash          | sha256:*                           |
      | transport_security  | tls                                |
      | reason              |                                    |
      | error               |                                    |
      | request_body        |                                    |
      | metadata            |                                    |
      | timestamp           | *                                  |
      | duration_ms         | *                                  |
      | request_id          | *                                  |
      | source_ip           | *                                  |
      | user_agent          | *                                  |

  # Delete Operations
  Scenario: Delete exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "delete-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I DELETE "/exporters/delete-test-exporter"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type          | exporter_delete                 |
      | outcome             | success                         |
      | actor_id            |                                 |
      | actor_type          | anonymous                       |
      | auth_method         |                                 |
      | role                |                                 |
      | method              | DELETE                          |
      | path                | /exporters/delete-test-exporter |
      | status_code         | 200                             |
      | target_type         | exporter                        |
      | target_id           | delete-test-exporter            |
      | schema_id           |                                 |
      | version             |                                 |
      | schema_type         |                                 |
      | context             |                                 |
      | before_hash         | sha256:*                        |
      | after_hash          |                                 |
      | transport_security  | tls                             |
      | reason              |                                 |
      | error               |                                 |
      | request_body        |                                 |
      | metadata            |                                 |
      | timestamp           | *                               |
      | duration_ms         | *                               |
      | request_id          | *                               |
      | source_ip           | *                               |
      | user_agent          | *                               |

  Scenario: Delete exporter returns name in response
    When I POST "/exporters" with body:
      """
      {
        "name": "delete-response-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I DELETE "/exporters/delete-response-exporter"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type          | exporter_delete                     |
      | outcome             | success                             |
      | actor_id            |                                     |
      | actor_type          | anonymous                           |
      | auth_method         |                                     |
      | role                |                                     |
      | method              | DELETE                              |
      | path                | /exporters/delete-response-exporter |
      | status_code         | 200                                 |
      | target_type         | exporter                            |
      | target_id           | delete-response-exporter            |
      | schema_id           |                                     |
      | version             |                                     |
      | schema_type         |                                     |
      | context             |                                     |
      | before_hash         | sha256:*                            |
      | after_hash          |                                     |
      | transport_security  | tls                                 |
      | reason              |                                     |
      | error               |                                     |
      | request_body        |                                     |
      | metadata            |                                     |
      | timestamp           | *                                   |
      | duration_ms         | *                                   |
      | request_id          | *                                   |
      | source_ip           | *                                   |
      | user_agent          | *                                   |

  Scenario: Delete non-existent exporter
    When I DELETE "/exporters/non-existent-delete-exporter"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Pause/Resume/Reset Operations
  Scenario: Pause exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "pause-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I PUT "/exporters/pause-test-exporter/pause" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type          | exporter_pause                       |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | method              | PUT                                  |
      | path                | /exporters/pause-test-exporter/pause |
      | status_code         | 200                                  |
      | target_type         | exporter                             |
      | target_id           | pause-test-exporter                  |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | context             |                                      |
      | before_hash         | sha256:*                             |
      | after_hash          | sha256:*                             |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  Scenario: Resume exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "resume-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I PUT "/exporters/resume-test-exporter/pause" with body:
      """
      {}
      """
    And I PUT "/exporters/resume-test-exporter/resume" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type          | exporter_resume                         |
      | outcome             | success                                 |
      | actor_id            |                                         |
      | actor_type          | anonymous                               |
      | auth_method         |                                         |
      | role                |                                         |
      | method              | PUT                                     |
      | path                | /exporters/resume-test-exporter/resume  |
      | status_code         | 200                                     |
      | target_type         | exporter                                |
      | target_id           | resume-test-exporter                    |
      | schema_id           |                                         |
      | version             |                                         |
      | schema_type         |                                         |
      | context             |                                         |
      | before_hash         | sha256:*                                |
      | after_hash          | sha256:*                                |
      | transport_security  | tls                                     |
      | reason              |                                         |
      | error               |                                         |
      | request_body        |                                         |
      | metadata            |                                         |
      | timestamp           | *                                       |
      | duration_ms         | *                                       |
      | request_id          | *                                       |
      | source_ip           | *                                       |
      | user_agent          | *                                       |

  Scenario: Reset exporter
    When I POST "/exporters" with body:
      """
      {
        "name": "reset-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I PUT "/exporters/reset-test-exporter/reset" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type          | exporter_reset                       |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | method              | PUT                                  |
      | path                | /exporters/reset-test-exporter/reset |
      | status_code         | 200                                  |
      | target_type         | exporter                             |
      | target_id           | reset-test-exporter                  |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | context             |                                      |
      | before_hash         | sha256:*                             |
      | after_hash          | sha256:*                             |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  Scenario: Pause non-existent exporter
    When I PUT "/exporters/non-existent-pause/pause" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  Scenario: Resume non-existent exporter
    When I PUT "/exporters/non-existent-resume/resume" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  Scenario: Reset non-existent exporter
    When I PUT "/exporters/non-existent-reset/reset" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Status Operations
  Scenario: Get exporter status
    When I POST "/exporters" with body:
      """
      {
        "name": "status-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"]
      }
      """
    And I GET "/exporters/status-test-exporter/status"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "status-test-exporter"

  Scenario: Get status of non-existent exporter
    When I GET "/exporters/non-existent-status/status"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Config Operations
  Scenario: Get exporter config
    When I POST "/exporters" with body:
      """
      {
        "name": "config-test-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "config": {
          "schema.registry.url": "http://remote:8081",
          "timeout.ms": "30000"
        }
      }
      """
    And I GET "/exporters/config-test-exporter/config"
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Update exporter config
    When I POST "/exporters" with body:
      """
      {
        "name": "config-update-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "config": {
          "schema.registry.url": "http://old:8081"
        }
      }
      """
    And I PUT "/exporters/config-update-exporter/config" with body:
      """
      {
        "config": {
          "schema.registry.url": "http://new:8081",
          "timeout.ms": "60000"
        }
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the audit log should contain an event:
      | event_type          | exporter_config_update                   |
      | outcome             | success                                  |
      | actor_id            |                                          |
      | actor_type          | anonymous                                |
      | auth_method         |                                          |
      | role                |                                          |
      | method              | PUT                                      |
      | path                | /exporters/config-update-exporter/config |
      | status_code         | 200                                      |
      | target_type         | exporter                                 |
      | target_id           | config-update-exporter                   |
      | schema_id           |                                          |
      | version             |                                          |
      | schema_type         |                                          |
      | context             |                                          |
      | before_hash         | sha256:*                                 |
      | after_hash          | sha256:*                                 |
      | transport_security  | tls                                      |
      | reason              |                                          |
      | error               |                                          |
      | request_body        |                                          |
      | metadata            |                                          |
      | timestamp           | *                                        |
      | duration_ms         | *                                        |
      | request_id          | *                                        |
      | source_ip           | *                                        |
      | user_agent          | *                                        |

  Scenario: Get config of non-existent exporter
    When I GET "/exporters/non-existent-config/config"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  Scenario: Update config of non-existent exporter
    When I PUT "/exporters/non-existent-config-update/config" with body:
      """
      {
        "config": {
          "schema.registry.url": "http://remote:8081"
        }
      }
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40450

  # Lifecycle Scenario
  Scenario: Exporter lifecycle - create, pause, resume, reset, delete
    When I POST "/exporters" with body:
      """
      {
        "name": "lifecycle-exporter",
        "contextType": "AUTO",
        "subjects": ["test-value"],
        "config": {
          "schema.registry.url": "http://remote:8081"
        }
      }
      """
    Then the response status should be 200
    And the response field "name" should be "lifecycle-exporter"

    When I GET "/exporters/lifecycle-exporter/status"
    Then the response status should be 200
    And the response field "name" should be "lifecycle-exporter"

    When I PUT "/exporters/lifecycle-exporter/pause" with body:
      """
      {}
      """
    Then the response status should be 200

    When I PUT "/exporters/lifecycle-exporter/resume" with body:
      """
      {}
      """
    Then the response status should be 200

    When I PUT "/exporters/lifecycle-exporter/reset" with body:
      """
      {}
      """
    Then the response status should be 200

    When I DELETE "/exporters/lifecycle-exporter"
    Then the response status should be 200

    When I GET "/exporters/lifecycle-exporter"
    Then the response status should be 404
    And the response should have error code 40450
    And the audit log should contain an event:
      | event_type          | exporter_create      |
      | outcome             | success              |
      | actor_id            |                      |
      | actor_type          | anonymous            |
      | auth_method         |                      |
      | role                |                      |
      | method              | POST                 |
      | path                | /exporters           |
      | status_code         | 200                  |
      | target_type         | exporter             |
      | target_id           | lifecycle-exporter   |
      | schema_id           |                      |
      | version             |                      |
      | schema_type         |                      |
      | context             |                      |
      | before_hash         |                      |
      | after_hash          | sha256:*             |
      | transport_security  | tls                  |
      | reason              |                      |
      | error               |                      |
      | request_body        |                      |
      | metadata            |                      |
      | timestamp           | *                    |
      | duration_ms         | *                    |
      | request_id          | *                    |
      | source_ip           | *                    |
      | user_agent          | *                    |
    And the audit log should contain an event:
      | event_type          | exporter_pause                      |
      | outcome             | success                             |
      | actor_id            |                                     |
      | actor_type          | anonymous                           |
      | auth_method         |                                     |
      | role                |                                     |
      | method              | PUT                                 |
      | path                | /exporters/lifecycle-exporter/pause |
      | status_code         | 200                                 |
      | target_type         | exporter                            |
      | target_id           | lifecycle-exporter                  |
      | schema_id           |                                     |
      | version             |                                     |
      | schema_type         |                                     |
      | context             |                                     |
      | before_hash         | sha256:*                            |
      | after_hash          | sha256:*                            |
      | transport_security  | tls                                 |
      | reason              |                                     |
      | error               |                                     |
      | request_body        |                                     |
      | metadata            |                                     |
      | timestamp           | *                                   |
      | duration_ms         | *                                   |
      | request_id          | *                                   |
      | source_ip           | *                                   |
      | user_agent          | *                                   |
    And the audit log should contain an event:
      | event_type          | exporter_resume                      |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | method              | PUT                                  |
      | path                | /exporters/lifecycle-exporter/resume |
      | status_code         | 200                                  |
      | target_type         | exporter                             |
      | target_id           | lifecycle-exporter                   |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | context             |                                      |
      | before_hash         | sha256:*                             |
      | after_hash          | sha256:*                             |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |
    And the audit log should contain an event:
      | event_type          | exporter_reset                      |
      | outcome             | success                             |
      | actor_id            |                                     |
      | actor_type          | anonymous                           |
      | auth_method         |                                     |
      | role                |                                     |
      | method              | PUT                                 |
      | path                | /exporters/lifecycle-exporter/reset |
      | status_code         | 200                                 |
      | target_type         | exporter                            |
      | target_id           | lifecycle-exporter                  |
      | schema_id           |                                     |
      | version             |                                     |
      | schema_type         |                                     |
      | context             |                                     |
      | before_hash         | sha256:*                            |
      | after_hash          | sha256:*                            |
      | transport_security  | tls                                 |
      | reason              |                                     |
      | error               |                                     |
      | request_body        |                                     |
      | metadata            |                                     |
      | timestamp           | *                                   |
      | duration_ms         | *                                   |
      | request_id          | *                                   |
      | source_ip           | *                                   |
      | user_agent          | *                                   |
    And the audit log should contain an event:
      | event_type          | exporter_delete               |
      | outcome             | success                       |
      | actor_id            |                               |
      | actor_type          | anonymous                     |
      | auth_method         |                               |
      | role                |                               |
      | method              | DELETE                        |
      | path                | /exporters/lifecycle-exporter |
      | status_code         | 200                           |
      | target_type         | exporter                      |
      | target_id           | lifecycle-exporter            |
      | schema_id           |                               |
      | version             |                               |
      | schema_type         |                               |
      | context             |                               |
      | before_hash         | sha256:*                      |
      | after_hash          |                               |
      | transport_security  | tls                           |
      | reason              |                               |
      | error               |                               |
      | request_body        |                               |
      | metadata            |                               |
      | timestamp           | *                             |
      | duration_ms         | *                             |
      | request_id          | *                             |
      | source_ip           | *                             |
      | user_agent          | *                             |
