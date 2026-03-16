@functional
Feature: Mode Enforcement
  Confluent Schema Registry supports READWRITE, READONLY, READONLY_OVERRIDE,
  and IMPORT modes. When a subject or global mode is READONLY or READONLY_OVERRIDE,
  write operations are blocked with error code 42205.

  # ==========================================================================
  # READONLY MODE — BLOCKS ALL WRITES
  # ==========================================================================

  Scenario: READONLY mode blocks schema registration
    Given the global mode is "READONLY"
    When I register a schema under subject "mode-ro-reg":
      """
      {"type":"record","name":"RO","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    # Reset mode
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                   |
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
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
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

  Scenario: READONLY mode blocks subject deletion
    Given the global mode is "READWRITE"
    And subject "mode-ro-del" has schema:
      """
      {"type":"record","name":"RODel","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I DELETE "/subjects/mode-ro-del"
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                   |
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
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
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

  Scenario: READONLY mode blocks version deletion
    Given the global mode is "READWRITE"
    And subject "mode-ro-delv" has schema:
      """
      {"type":"record","name":"RODelV","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I DELETE "/subjects/mode-ro-delv/versions/1"
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                   |
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
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
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

  Scenario: READONLY mode still allows GET operations
    Given the global mode is "READWRITE"
    And subject "mode-ro-get" has schema:
      """
      {"type":"record","name":"ROGet","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I get the latest version of subject "mode-ro-get"
    Then the response status should be 200
    When I list all subjects
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                   |
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
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
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

  # ==========================================================================
  # PER-SUBJECT READONLY MODE
  # ==========================================================================

  Scenario: Per-subject READONLY blocks writes only on that subject
    Given the global mode is "READWRITE"
    When I set the mode for subject "mode-per-ro" to "READONLY"
    Then the response status should be 200
    When I register a schema under subject "mode-per-ro":
      """
      {"type":"record","name":"PerRO","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    # Other subjects still work
    When I register a schema under subject "mode-per-rw":
      """
      {"type":"record","name":"PerRW","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | mode_update                   |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | mode-per-ro                   |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
      | path                 | /mode/mode-per-ro             |
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

  # ==========================================================================
  # READONLY_OVERRIDE MODE
  # ==========================================================================

  Scenario: READONLY_OVERRIDE is a valid mode
    When I set the global mode to "READONLY_OVERRIDE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READONLY_OVERRIDE"
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                   |
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
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
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

  Scenario: READONLY_OVERRIDE blocks schema registration
    Given the global mode is "READONLY_OVERRIDE"
    When I register a schema under subject "mode-override-reg":
      """
      {"type":"record","name":"Override","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                   |
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
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
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

  Scenario: READONLY_OVERRIDE allows changing mode back
    When I set the global mode to "READONLY_OVERRIDE"
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                   |
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
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
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

  # ==========================================================================
  # DELETE /mode/{subject}
  # ==========================================================================

  Scenario: DELETE /mode/{subject} removes subject mode
    When I set the mode for subject "mode-del-test" to "READONLY"
    Then the response status should be 200
    When I GET "/mode/mode-del-test"
    Then the response field "mode" should be "READONLY"
    When I delete the mode for subject "mode-del-test"
    Then the response status should be 200
    When I GET "/mode/mode-del-test"
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type           | mode_delete                   |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | mode-del-test                 |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | sha256:*                      |
      | after_hash           |                               |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | DELETE                        |
      | path                 | /mode/mode-del-test           |
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

  Scenario: DELETE /mode/{subject} when no mode returns 404
    When I delete the mode for subject "mode-del-nonexist"
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type           | mode_delete                   |
      | outcome              | failure                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | mode-del-nonexist             |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          |                               |
      | after_hash           |                               |
      | context              | .                              |
      | transport_security   | tls                           |
      | method               | DELETE                        |
      | path                 | /mode/mode-del-nonexist       |
      | status_code          | 404                           |
      | reason               | not_found                     |
      | error                |                               |
      | request_body         |                               |
      | metadata             |                               |
      | timestamp            | *                             |
      | duration_ms          | *                             |
      | request_id           | *                             |
      | source_ip            | *                             |
      | user_agent           | *                             |

  # ==========================================================================
  # IMPORT MODE
  # ==========================================================================

  Scenario: IMPORT mode allows registration with explicit ID
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-import-with-id/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportWithId\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 99990}
      """
    Then the response status should be 200
    And the response field "id" should be 99990
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | schema_register                         |
      | outcome              | success                                 |
      | actor_id             |                                         |
      | actor_type           | anonymous                               |
      | auth_method          |                                         |
      | role                 |                                         |
      | target_type          | subject                                 |
      | target_id            | mode-import-with-id                     |
      | schema_id            | *                                       |
      | version              | *                                       |
      | schema_type          | AVRO                                    |
      | before_hash          |                                         |
      | after_hash           | sha256:*                                |
      | context              | .                                       |
      | transport_security   | tls                                     |
      | method               | POST                                    |
      | path                 | /subjects/mode-import-with-id/versions  |
      | status_code          | 200                                     |
      | reason               |                                         |
      | error                |                                         |
      | request_body         |                                         |
      | metadata             |                                         |
      | timestamp            | *                                       |
      | duration_ms          | *                                       |
      | request_id           | *                                       |
      | source_ip            | *                                       |
      | user_agent           | *                                       |

  Scenario: IMPORT mode rejects different schema with same ID
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-import-dup1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportDup1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 99991}
      """
    Then the response status should be 200
    And the response field "id" should be 99991
    # Try to register a DIFFERENT schema with the SAME ID
    When I POST "/subjects/mode-import-dup2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportDup2\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}", "id": 99991}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"
    # First import succeeded
    And the audit log should contain an event:
      | event_type           | schema_register                        |
      | outcome              | success                                |
      | actor_id             |                                        |
      | actor_type           | anonymous                              |
      | auth_method          |                                        |
      | role                 |                                        |
      | target_type          | subject                                |
      | target_id            | mode-import-dup1                       |
      | schema_id            | *                                      |
      | version              | *                                      |
      | schema_type          | AVRO                                   |
      | before_hash          |                                        |
      | after_hash           | sha256:*                               |
      | context              | .                                      |
      | transport_security   | tls                                    |
      | method               | POST                                   |
      | path                 | /subjects/mode-import-dup1/versions    |
      | status_code          | 200                                    |
      | reason               |                                        |
      | error                |                                        |
      | request_body         |                                        |
      | metadata             |                                        |
      | timestamp            | *                                      |
      | duration_ms          | *                                      |
      | request_id           | *                                      |
      | source_ip            | *                                      |
      | user_agent           | *                                      |
    # Second import failed (ID conflict)
    And the audit log should contain an event:
      | event_type           | schema_register                        |
      | outcome              | failure                                |
      | actor_id             |                                        |
      | actor_type           | anonymous                              |
      | auth_method          |                                        |
      | role                 |                                        |
      | target_type          | subject                                |
      | target_id            | mode-import-dup2                       |
      | schema_id            | *                                      |
      | version              | *                                      |
      | schema_type          | AVRO                                   |
      | before_hash          |                                        |
      | after_hash           |                                        |
      | context              | .                                      |
      | transport_security   | tls                                    |
      | method               | POST                                   |
      | path                 | /subjects/mode-import-dup2/versions    |
      | status_code          | 422                                    |
      | reason               | invalid_schema                         |
      | error                |                                        |
      | request_body         |                                        |
      | metadata             |                                        |
      | timestamp            | *                                      |
      | duration_ms          | *                                      |
      | request_id           | *                                      |
      | source_ip            | *                                      |
      | user_agent           | *                                      |

  Scenario: IMPORT mode allows same schema with same ID in different subject
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-import-share1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportShare\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 99992}
      """
    Then the response status should be 200
    And the response field "id" should be 99992
    # Same schema content, same ID, different subject — should succeed
    When I POST "/subjects/mode-import-share2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportShare\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 99992}
      """
    Then the response status should be 200
    And the response field "id" should be 99992
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | mode-import-share2                       |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | method               | POST                                     |
      | path                 | /subjects/mode-import-share2/versions    |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
      | source_ip            | *                                        |
      | user_agent           | *                                        |

  Scenario: IMPORT mode rejects registration without explicit ID
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I register a schema under subject "mode-import-no-id":
      """
      {"type":"record","name":"ImportNoId","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_update                   |
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
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
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

  # ==========================================================================
  # INVALID MODE
  # ==========================================================================

  Scenario: Invalid mode value returns 42204
    When I PUT "/mode" with body:
      """
      {"mode": "INVALID_MODE"}
      """
    Then the response status should be 422
    And the response should have error code 42204
    # Invalid mode value — 422 failure
    And the audit log should contain an event:
      | event_type           | mode_update                   |
      | outcome              | failure                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | _global                       |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | *                             |
      | after_hash           |                               |
      | context              | .                              |
      | transport_security   | tls                           |
      | method               | PUT                           |
      | path                 | /mode                         |
      | status_code          | 422                           |
      | reason               | invalid_schema                |
      | error                |                               |
      | request_body         |                               |
      | metadata             |                               |
      | timestamp            | *                             |
      | duration_ms          | *                             |
      | request_id           | *                             |
      | source_ip            | *                             |
      | user_agent           | *                             |

  # ==========================================================================
  # Explicit ID enforcement — explicit schema IDs in register requests
  # are only allowed when the mode is IMPORT.
  # ==========================================================================

  Scenario: Explicit ID in READWRITE mode is rejected with 42205
    Given the global mode is "READWRITE"
    When I POST "/subjects/mode-rw-explicit/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ExplicitRW\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12345}
      """
    Then the response status should be 422
    And the response should have error code 42205
    # Explicit ID in READWRITE mode — 422 failure
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | mode-rw-explicit                         |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | method               | POST                                     |
      | path                 | /subjects/mode-rw-explicit/versions      |
      | status_code          | 422                                      |
      | reason               | invalid_schema                           |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
      | source_ip            | *                                        |
      | user_agent           | *                                        |

  Scenario: Explicit ID in IMPORT mode succeeds
    Given the global mode is "IMPORT"
    When I POST "/subjects/mode-import-explicit/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ExplicitImp\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12345}
      """
    Then the response status should be 200
    And the response field "id" should be 12345
    When I set the global mode to "READWRITE"
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | mode-import-explicit                       |
      | schema_id            | *                                          |
      | version              | *                                          |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           | sha256:*                                   |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | method               | POST                                       |
      | path                 | /subjects/mode-import-explicit/versions    |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |
      | source_ip            | *                                          |
      | user_agent           | *                                          |

  Scenario: Per-subject IMPORT mode allows explicit ID
    Given the global mode is "READWRITE"
    When I set the mode for subject "mode-subj-import" to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-subj-import/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SubjImp\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12346}
      """
    Then the response status should be 200
    And the response field "id" should be 12346
    And the audit log should contain an event:
      | event_type           | mode_update                   |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | mode                          |
      | target_id            | mode-subj-import              |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
      | path                 | /mode/mode-subj-import        |
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
