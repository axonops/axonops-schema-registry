@functional @audit
Feature: REST API Audit Logging
  The REST API MUST emit audit events for security-relevant operations so that
  operators can track schema changes, config updates, and deletions.
  Unauthenticated requests MUST still be audited with an empty user field.

  This test suite runs WITHOUT authentication enabled, so all requests are
  anonymous. Actor fields MUST reflect: actor_type=anonymous, empty actor_id.

  # --- Schema Events ---

  Scenario: Schema registration emits schema_register audit event
    When I register a schema under subject "audit-rest-register":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                                      |
      | duration_ms        | *                                      |
      | event_type         | schema_register                        |
      | outcome            | success                                |
      | actor_id           |                                        |
      | actor_type         | anonymous                              |
      | role               |                                        |
      | auth_method        |                                        |
      | target_type        | subject                                |
      | target_id          | audit-rest-register                    |
      | schema_id          | *                                      |
      | version            | *                                      |
      | schema_type        | AVRO                                   |
      | before_hash        |                                        |
      | after_hash         | sha256:*                               |
      | context            | .                                      |
      | request_id         | *                                      |
      | transport_security | tls                                    |
      | source_ip          | *                                      |
      | user_agent         | *                                      |
      | method             | POST                                   |
      | path               | /subjects/audit-rest-register/versions |
      | status_code        | 200                                    |
      | reason             |                                        |
      | error              |                                        |
      | request_body       |                                        |
      | metadata           |                                        |

  Scenario: Schema version deletion emits schema_delete audit event
    Given I register a schema under subject "audit-rest-verdel":
      """
      {"type":"string"}
      """
    When I delete version 1 of subject "audit-rest-verdel"
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                                      |
      | duration_ms        | *                                      |
      | event_type         | schema_delete_soft                     |
      | outcome            | success                                |
      | actor_id           |                                        |
      | actor_type         | anonymous                              |
      | role               |                                        |
      | auth_method        |                                        |
      | target_type        | subject                                |
      | target_id          | audit-rest-verdel                      |
      | schema_id          | *                                      |
      | version            | *                                      |
      | schema_type        | AVRO                                   |
      | before_hash        | sha256:*                               |
      | after_hash         |                                        |
      | context            | .                                      |
      | request_id         | *                                      |
      | transport_security | tls                                    |
      | source_ip          | *                                      |
      | user_agent         | *                                      |
      | method             | DELETE                                 |
      | path               | /subjects/audit-rest-verdel/versions/1 |
      | status_code        | 200                                    |
      | reason             |                                        |
      | error              |                                        |
      | request_body       |                                        |
      | metadata           |                                        |

  Scenario: Schema lookup emits schema_lookup audit event
    Given I register a schema under subject "audit-rest-lookup":
      """
      {"type":"string"}
      """
    When I lookup schema in subject "audit-rest-lookup":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                           |
      | duration_ms        | *                           |
      | event_type         | schema_lookup               |
      | outcome            | success                     |
      | actor_id           |                             |
      | actor_type         | anonymous                   |
      | role               |                             |
      | auth_method        |                             |
      | target_type        | subject                     |
      | target_id          | audit-rest-lookup           |
      | schema_id          | *                           |
      | version            | *                           |
      | schema_type        | AVRO                        |
      | before_hash        |                             |
      | after_hash         |                             |
      | context            | .                            |
      | request_id         | *                           |
      | transport_security | tls                         |
      | source_ip          | *                           |
      | user_agent         | *                           |
      | method             | POST                        |
      | path               | /subjects/audit-rest-lookup |
      | status_code        | 200                         |
      | reason             |                             |
      | error              |                             |
      | request_body       |                             |
      | metadata           |                             |

  # --- Subject Events ---

  Scenario: Subject deletion emits subject_delete audit event
    Given I register a schema under subject "audit-rest-delete":
      """
      {"type":"string"}
      """
    When I delete subject "audit-rest-delete"
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                           |
      | duration_ms        | *                           |
      | event_type         | subject_delete_soft         |
      | outcome            | success                     |
      | actor_id           |                             |
      | actor_type         | anonymous                   |
      | role               |                             |
      | auth_method        |                             |
      | target_type        | subject                     |
      | target_id          | audit-rest-delete           |
      | schema_id          |                             |
      | version            |                             |
      | schema_type        | AVRO                        |
      | before_hash        | sha256:*                    |
      | after_hash         |                             |
      | context            | .                           |
      | request_id         | *                           |
      | transport_security | tls                         |
      | source_ip          | *                           |
      | user_agent         | *                           |
      | method             | DELETE                      |
      | path               | /subjects/audit-rest-delete |
      | status_code        | 200                         |
      | reason             |                             |
      | error              |                             |
      | request_body       |                             |
      | metadata           |                             |

  Scenario: Permanent subject deletion emits subject_delete audit event
    Given I register a schema under subject "audit-rest-permdel":
      """
      {"type":"string"}
      """
    And I delete subject "audit-rest-permdel"
    When I permanently delete subject "audit-rest-permdel"
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                            |
      | duration_ms        | *                            |
      | event_type         | subject_delete_permanent     |
      | outcome            | success                      |
      | actor_id           |                              |
      | actor_type         | anonymous                    |
      | role               |                              |
      | auth_method        |                              |
      | target_type        | subject                      |
      | target_id          | audit-rest-permdel           |
      | schema_id          |                              |
      | version            |                              |
      | schema_type        | AVRO                         |
      | before_hash        | sha256:*                     |
      | after_hash         |                              |
      | context            | .                            |
      | request_id         | *                            |
      | transport_security | tls                          |
      | source_ip          | *                            |
      | user_agent         | *                            |
      | method             | DELETE                       |
      | path               | /subjects/audit-rest-permdel |
      | status_code        | 200                          |
      | reason             |                              |
      | error              |                              |
      | request_body       |                              |
      | metadata           |                              |

  # --- Config Events ---

  Scenario: Config update emits config_update audit event
    When I set the global compatibility level to "FULL"
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *             |
      | duration_ms        | *             |
      | event_type         | config_update |
      | outcome            | success       |
      | actor_id           |               |
      | actor_type         | anonymous     |
      | role               |               |
      | auth_method        |               |
      | target_type        | config        |
      | target_id          | _global       |
      | schema_id          |               |
      | version            |               |
      | schema_type        |               |
      | before_hash        | *             |
      | after_hash         | sha256:*      |
      | context            | .             |
      | request_id         | *             |
      | transport_security | tls           |
      | source_ip          | *             |
      | user_agent         | *             |
      | method             | PUT           |
      | path               | /config       |
      | status_code        | 200           |
      | reason             |               |
      | error              |               |
      | request_body       |               |
      | metadata           |               |

  Scenario: Subject config update emits config_update audit event
    Given I register a schema under subject "audit-rest-cfgupd":
      """
      {"type":"string"}
      """
    When I PUT "/config/audit-rest-cfgupd" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                             |
      | duration_ms        | *                             |
      | event_type         | config_update                 |
      | outcome            | success                       |
      | actor_id           |                               |
      | actor_type         | anonymous                     |
      | role               |                               |
      | auth_method        |                               |
      | target_type        | config                        |
      | target_id          | audit-rest-cfgupd             |
      | schema_id          |                               |
      | version            |                               |
      | schema_type        |                               |
      | before_hash        | *                             |
      | after_hash         | sha256:*                      |
      | context            | .                             |
      | request_id         | *                             |
      | transport_security | tls                           |
      | source_ip          | *                             |
      | user_agent         | *                             |
      | method             | PUT                           |
      | path               | /config/audit-rest-cfgupd     |
      | status_code        | 200                           |
      | reason             |                               |
      | error              |                               |
      | request_body       |                               |
      | metadata           |                               |

  Scenario: Config delete emits config_delete audit event
    Given I register a schema under subject "audit-rest-cfgdel":
      """
      {"type":"string"}
      """
    And I PUT "/config/audit-rest-cfgdel" with body:
      """
      {"compatibility": "FULL"}
      """
    When I DELETE "/config/audit-rest-cfgdel"
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                             |
      | duration_ms        | *                             |
      | event_type         | config_delete                 |
      | outcome            | success                       |
      | actor_id           |                               |
      | actor_type         | anonymous                     |
      | role               |                               |
      | auth_method        |                               |
      | target_type        | config                        |
      | target_id          | audit-rest-cfgdel             |
      | schema_id          |                               |
      | version            |                               |
      | schema_type        |                               |
      | before_hash        | sha256:*                      |
      | after_hash         |                               |
      | context            | .                             |
      | request_id         | *                             |
      | transport_security | tls                           |
      | source_ip          | *                             |
      | user_agent         | *                             |
      | method             | DELETE                        |
      | path               | /config/audit-rest-cfgdel     |
      | status_code        | 200                           |
      | reason             |                               |
      | error              |                               |
      | request_body       |                               |
      | metadata           |                               |

  # --- Mode Events ---

  Scenario: Mode update emits mode_update audit event
    When I PUT "/mode" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *             |
      | duration_ms        | *             |
      | event_type         | mode_update   |
      | outcome            | success       |
      | actor_id           |               |
      | actor_type         | anonymous     |
      | role               |               |
      | auth_method        |               |
      | target_type        | mode          |
      | target_id          | _global       |
      | schema_id          |               |
      | version            |               |
      | schema_type        |               |
      | before_hash        | *             |
      | after_hash         | sha256:*      |
      | context            | .             |
      | request_id         | *             |
      | transport_security | tls           |
      | source_ip          | *             |
      | user_agent         | *             |
      | method             | PUT           |
      | path               | /mode         |
      | status_code        | 200           |
      | reason             |               |
      | error              |               |
      | request_body       |               |
      | metadata           |               |

  Scenario: Subject mode update emits mode_update audit event
    Given I register a schema under subject "audit-rest-modeupd":
      """
      {"type":"string"}
      """
    When I PUT "/mode/audit-rest-modeupd" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                            |
      | duration_ms        | *                            |
      | event_type         | mode_update                  |
      | outcome            | success                      |
      | actor_id           |                              |
      | actor_type         | anonymous                    |
      | role               |                              |
      | auth_method        |                              |
      | target_type        | mode                         |
      | target_id          | audit-rest-modeupd           |
      | schema_id          |                              |
      | version            |                              |
      | schema_type        |                              |
      | before_hash        | *                            |
      | after_hash         | sha256:*                     |
      | context            | .                            |
      | request_id         | *                            |
      | transport_security | tls                          |
      | source_ip          | *                            |
      | user_agent         | *                            |
      | method             | PUT                          |
      | path               | /mode/audit-rest-modeupd     |
      | status_code        | 200                          |
      | reason             |                              |
      | error              |                              |
      | request_body       |                              |
      | metadata           |                              |

  Scenario: Mode delete emits mode_delete audit event
    Given I register a schema under subject "audit-rest-modedel":
      """
      {"type":"string"}
      """
    And I PUT "/mode/audit-rest-modedel" with body:
      """
      {"mode": "READONLY"}
      """
    When I DELETE "/mode/audit-rest-modedel"
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                            |
      | duration_ms        | *                            |
      | event_type         | mode_delete                  |
      | outcome            | success                      |
      | actor_id           |                              |
      | actor_type         | anonymous                    |
      | role               |                              |
      | auth_method        |                              |
      | target_type        | mode                         |
      | target_id          | audit-rest-modedel           |
      | schema_id          |                              |
      | version            |                              |
      | schema_type        |                              |
      | before_hash        | sha256:*                     |
      | after_hash         |                              |
      | context            | .                            |
      | request_id         | *                            |
      | transport_security | tls                          |
      | source_ip          | *                            |
      | user_agent         | *                            |
      | method             | DELETE                       |
      | path               | /mode/audit-rest-modedel     |
      | status_code        | 200                          |
      | reason             |                              |
      | error              |                              |
      | request_body       |                              |
      | metadata           |                              |

  # --- Import Events ---

  Scenario: Schema import emits schema_import audit event
    When I PUT "/mode" with body:
      """
      {"mode": "IMPORT"}
      """
    And I POST "/import/schemas" with body:
      """
      {
        "schemas": [
          {
            "subject": "audit-rest-import",
            "version": 1,
            "id": 99901,
            "schemaType": "AVRO",
            "schema": "{\"type\":\"string\"}"
          }
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *               |
      | duration_ms        | *               |
      | event_type         | schema_import   |
      | outcome            | success         |
      | actor_id           |                 |
      | actor_type         | anonymous       |
      | role               |                 |
      | auth_method        |                 |
      | target_type        | subject         |
      | target_id          | audit-rest-import |
      | schema_id          | 99901           |
      | version            | 1               |
      | schema_type        | AVRO            |
      | before_hash        |                 |
      | after_hash         | sha256:*        |
      | context            | .               |
      | request_id         | *               |
      | transport_security | tls             |
      | source_ip          | *               |
      | user_agent         | *               |
      | method             | POST            |
      | path               | /import/schemas |
      | status_code        | 200             |
      | reason             |                 |
      | error              |                 |
      | request_body       |                 |
      | metadata           | *               |

  # --- Exporter Events ---

  Scenario: Exporter creation emits exporter_create audit event
    When I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-create",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                |
      | duration_ms        | *                |
      | event_type         | exporter_create  |
      | outcome            | success          |
      | actor_id           |                  |
      | actor_type         | anonymous        |
      | role               |                  |
      | auth_method        |                  |
      | target_type        | exporter         |
      | target_id          | audit-exp-create |
      | schema_id          |                  |
      | version            |                  |
      | schema_type        |                  |
      | before_hash        |                  |
      | after_hash         | sha256:*         |
      | context            | .                 |
      | request_id         | *                |
      | transport_security | tls              |
      | source_ip          | *                |
      | user_agent         | *                |
      | method             | POST             |
      | path               | /exporters       |
      | status_code        | 200              |
      | reason             |                  |
      | error              |                  |
      | request_body       |                  |
      | metadata           |                  |

  Scenario: Exporter update emits exporter_update audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-update",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    When I PUT "/exporters/audit-exp-update" with body:
      """
      {
        "name": "audit-exp-update",
        "contextType": "NONE",
        "subjects": ["audit-exp-sub-new"]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                           |
      | duration_ms        | *                           |
      | event_type         | exporter_update             |
      | outcome            | success                     |
      | actor_id           |                             |
      | actor_type         | anonymous                   |
      | role               |                             |
      | auth_method        |                             |
      | target_type        | exporter                    |
      | target_id          | audit-exp-update            |
      | schema_id          |                             |
      | version            |                             |
      | schema_type        |                             |
      | before_hash        | sha256:*                    |
      | after_hash         | sha256:*                    |
      | context            | .                            |
      | request_id         | *                           |
      | transport_security | tls                         |
      | source_ip          | *                           |
      | user_agent         | *                           |
      | method             | PUT                         |
      | path               | /exporters/audit-exp-update |
      | status_code        | 200                         |
      | reason             |                             |
      | error              |                             |
      | request_body       |                             |
      | metadata           |                             |

  Scenario: Exporter deletion emits exporter_delete audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-delete",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    When I DELETE "/exporters/audit-exp-delete"
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                           |
      | duration_ms        | *                           |
      | event_type         | exporter_delete             |
      | outcome            | success                     |
      | actor_id           |                             |
      | actor_type         | anonymous                   |
      | role               |                             |
      | auth_method        |                             |
      | target_type        | exporter                    |
      | target_id          | audit-exp-delete            |
      | schema_id          |                             |
      | version            |                             |
      | schema_type        |                             |
      | before_hash        | sha256:*                    |
      | after_hash         |                             |
      | context            | .                            |
      | request_id         | *                           |
      | transport_security | tls                         |
      | source_ip          | *                           |
      | user_agent         | *                           |
      | method             | DELETE                      |
      | path               | /exporters/audit-exp-delete |
      | status_code        | 200                         |
      | reason             |                             |
      | error              |                             |
      | request_body       |                             |
      | metadata           |                             |

  Scenario: Exporter pause emits exporter_pause audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-pause",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    When I PUT "/exporters/audit-exp-pause/pause" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                                |
      | duration_ms        | *                                |
      | event_type         | exporter_pause                   |
      | outcome            | success                          |
      | actor_id           |                                  |
      | actor_type         | anonymous                        |
      | role               |                                  |
      | auth_method        |                                  |
      | target_type        | exporter                         |
      | target_id          | audit-exp-pause                  |
      | schema_id          |                                  |
      | version            |                                  |
      | schema_type        |                                  |
      | before_hash        | sha256:*                         |
      | after_hash         | sha256:*                         |
      | context            | .                                 |
      | request_id         | *                                |
      | transport_security | tls                              |
      | source_ip          | *                                |
      | user_agent         | *                                |
      | method             | PUT                              |
      | path               | /exporters/audit-exp-pause/pause |
      | status_code        | 200                              |
      | reason             |                                  |
      | error              |                                  |
      | request_body       |                                  |
      | metadata           |                                  |

  Scenario: Exporter resume emits exporter_resume audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-resume",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    And I PUT "/exporters/audit-exp-resume/pause" with body:
      """
      {}
      """
    When I PUT "/exporters/audit-exp-resume/resume" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                                    |
      | duration_ms        | *                                    |
      | event_type         | exporter_resume                      |
      | outcome            | success                              |
      | actor_id           |                                      |
      | actor_type         | anonymous                            |
      | role               |                                      |
      | auth_method        |                                      |
      | target_type        | exporter                             |
      | target_id          | audit-exp-resume                     |
      | schema_id          |                                      |
      | version            |                                      |
      | schema_type        |                                      |
      | before_hash        | sha256:*                             |
      | after_hash         | sha256:*                             |
      | context            | .                                     |
      | request_id         | *                                    |
      | transport_security | tls                                  |
      | source_ip          | *                                    |
      | user_agent         | *                                    |
      | method             | PUT                                  |
      | path               | /exporters/audit-exp-resume/resume   |
      | status_code        | 200                                  |
      | reason             |                                      |
      | error              |                                      |
      | request_body       |                                      |
      | metadata           |                                      |

  Scenario: Exporter reset emits exporter_reset audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-reset",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    When I PUT "/exporters/audit-exp-reset/reset" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                                |
      | duration_ms        | *                                |
      | event_type         | exporter_reset                   |
      | outcome            | success                          |
      | actor_id           |                                  |
      | actor_type         | anonymous                        |
      | role               |                                  |
      | auth_method        |                                  |
      | target_type        | exporter                         |
      | target_id          | audit-exp-reset                  |
      | schema_id          |                                  |
      | version            |                                  |
      | schema_type        |                                  |
      | before_hash        | sha256:*                         |
      | after_hash         | sha256:*                         |
      | context            | .                                 |
      | request_id         | *                                |
      | transport_security | tls                              |
      | source_ip          | *                                |
      | user_agent         | *                                |
      | method             | PUT                              |
      | path               | /exporters/audit-exp-reset/reset |
      | status_code        | 200                              |
      | reason             |                                  |
      | error              |                                  |
      | request_body       |                                  |
      | metadata           |                                  |

  # --- Cross-cutting Audit Properties ---

  Scenario: Request ID appears in audit entries
    When I register a schema under subject "audit-rest-reqid":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain "request_id"

  Scenario: Unauthenticated requests have empty user in audit log
    When I register a schema under subject "audit-rest-nouser":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | timestamp          | *                                     |
      | duration_ms        | *                                     |
      | event_type         | schema_register                       |
      | outcome            | success                               |
      | actor_id           |                                       |
      | actor_type         | anonymous                             |
      | role               |                                       |
      | auth_method        |                                       |
      | target_type        | subject                               |
      | target_id          | audit-rest-nouser                     |
      | schema_id          | *                                     |
      | version            | *                                     |
      | schema_type        | AVRO                                  |
      | before_hash        |                                       |
      | after_hash         | sha256:*                              |
      | context            | .                                     |
      | request_id         | *                                     |
      | transport_security | tls                                   |
      | source_ip          | *                                     |
      | user_agent         | *                                     |
      | method             | POST                                  |
      | path               | /subjects/audit-rest-nouser/versions  |
      | status_code        | 200                                   |
      | reason             |                                       |
      | error              |                                       |
      | request_body       |                                       |
      | metadata           |                                       |

  Scenario: Read-only operations are not audited by default
    Given I register a schema under subject "audit-rest-readonly":
      """
      {"type":"string"}
      """
    When I GET "/subjects"
    Then the response status should be 200
    And the audit log should not contain event "subject_list"

  Scenario: Multiple write operations produce separate audit entries
    When I register a schema under subject "audit-rest-multi-1":
      """
      {"type":"string"}
      """
    And I register a schema under subject "audit-rest-multi-2":
      """
      {"type":"int"}
      """
    Then the audit log should contain "audit-rest-multi-1"
    And the audit log should contain "audit-rest-multi-2"
