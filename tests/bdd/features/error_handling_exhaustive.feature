@functional
Feature: Error Handling & Edge Cases — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive error handling tests covering invalid schemas, bad references,
  error codes, and global ID consistency.

  # ==========================================================================
  # INVALID SCHEMA ERRORS
  # ==========================================================================

  Scenario: Register unparseable Avro schema returns INVALID_SCHEMA
    When I register a schema under subject "err-ex-bad-avro":
      """
      this is not valid json or avro at all
      """
    Then the response status should be 422
    And the response should have error code 42201
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | err-ex-bad-avro                          |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/err-ex-bad-avro/versions       |
      | status_code          | 422                                      |
      | reason               | invalid_schema                           |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Register invalid JSON in schema field returns error
    When I POST "/subjects/err-ex-bad-json/versions" with body:
      """
      {"schema": "not-valid-json{{{"}
      """
    Then the response status should be 422
    And the response should have error code 42201
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | err-ex-bad-json                          |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/err-ex-bad-json/versions       |
      | status_code          | 422                                      |
      | reason               | invalid_schema                           |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Register empty schema string returns error
    When I POST "/subjects/err-ex-empty/versions" with body:
      """
      {"schema": ""}
      """
    Then the response status should be 422
    And the audit log should contain an event:
      | event_type           | schema_register                      |
      | outcome              | failure                              |
      | actor_id             |                                      |
      | actor_type           | anonymous                            |
      | auth_method          |                                      |
      | role                 |                                      |
      | target_type          | subject                              |
      | target_id            | err-ex-empty                         |
      | schema_id            |                                      |
      | version              |                                      |
      | schema_type          |                                      |
      | before_hash          |                                      |
      | after_hash           |                                      |
      | context              | .                                    |
      | transport_security   | tls                                  |
      | source_ip            | *                                    |
      | user_agent           | *                                    |
      | method               | POST                                 |
      | path                 | /subjects/err-ex-empty/versions      |
      | status_code          | 422                                  |
      | reason               | invalid_schema                       |
      | error                |                                      |
      | request_body         |                                      |
      | metadata             |                                      |
      | timestamp            | *                                    |
      | duration_ms          | *                                    |
      | request_id           | *                                    |

  # ==========================================================================
  # BAD REFERENCE ERRORS
  # ==========================================================================

  Scenario: Register with reference to non-existent subject returns error
    When I register a schema under subject "err-ex-ref-nosub" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadRef\",\"fields\":[{\"name\":\"data\",\"type\":\"com.missing.Type\"}]}",
        "references": [
          {"name": "com.missing.Type", "subject": "nonexistent-subject", "version": 1}
        ]
      }
      """
    Then the response status should be 422
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | err-ex-ref-nosub                         |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/err-ex-ref-nosub/versions      |
      | status_code          | 422                                      |
      | reason               | invalid_schema                           |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Register with reference to non-existent version returns error
    Given subject "err-ex-ref-src" has schema:
      """
      {"type":"record","name":"ErrSrc","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "err-ex-ref-badver" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadVer\",\"fields\":[{\"name\":\"src\",\"type\":\"ErrSrc\"}]}",
        "references": [
          {"name": "ErrSrc", "subject": "err-ex-ref-src", "version": 999}
        ]
      }
      """
    Then the response status should be 422
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | failure                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | err-ex-ref-badver                         |
      | schema_id            |                                           |
      | version              |                                           |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           |                                           |
      | context              | .                                         |
      | transport_security   | tls                                       |
      | source_ip            | *                                         |
      | user_agent           | *                                         |
      | method               | POST                                      |
      | path                 | /subjects/err-ex-ref-badver/versions      |
      | status_code          | 422                                       |
      | reason               | invalid_schema                            |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |

  # ==========================================================================
  # GLOBAL ID CONSISTENCY
  # ==========================================================================

  Scenario: Same schema under different subjects gets same global ID
    Given the global compatibility level is "NONE"
    When I register a schema under subject "err-ex-id-s1":
      """
      {"type":"record","name":"GlobalID","fields":[{"name":"v","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    When I register a schema under subject "err-ex-id-s2":
      """
      {"type":"record","name":"GlobalID","fields":[{"name":"v","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"
    And the audit log should contain an event:
      | event_type           | schema_register                       |
      | outcome              | success                               |
      | actor_id             |                                       |
      | actor_type           | anonymous                             |
      | auth_method          |                                       |
      | role                 |                                       |
      | target_type          | subject                               |
      | target_id            | err-ex-id-s2                          |
      | schema_id            | *                                     |
      | version              | *                                     |
      | schema_type          | AVRO                                  |
      | before_hash          |                                       |
      | after_hash           | sha256:*                              |
      | context              | .                                     |
      | transport_security   | tls                                   |
      | source_ip            | *                                     |
      | user_agent           | *                                     |
      | method               | POST                                  |
      | path                 | /subjects/err-ex-id-s2/versions       |
      | status_code          | 200                                   |
      | reason               |                                       |
      | error                |                                       |
      | request_body         |                                       |
      | metadata             |                                       |
      | timestamp            | *                                     |
      | duration_ms          | *                                     |
      | request_id           | *                                     |

  Scenario: Different schemas get different global IDs
    Given the global compatibility level is "NONE"
    When I register a schema under subject "err-ex-diffid-s1":
      """
      {"type":"record","name":"DiffID1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id1"
    When I register a schema under subject "err-ex-diffid-s2":
      """
      {"type":"record","name":"DiffID2","fields":[{"name":"b","type":"int"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id2"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | err-ex-diffid-s2                         |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/err-ex-diffid-s2/versions      |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ==========================================================================
  # SUBJECT AND VERSION ERROR CODES
  # ==========================================================================

  Scenario: Operations on non-existent subject return correct error codes
    When I get version 1 of subject "err-ex-nosub-ops"
    Then the response status should be 404
    And the response should have error code 40401
    When I list versions of subject "err-ex-nosub-ops"
    Then the response status should be 404
    And the response should have error code 40401
    When I get the latest version of subject "err-ex-nosub-ops"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Invalid version number returns 422
    Given subject "err-ex-invver" has schema:
      """
      {"type":"record","name":"InvVer","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/err-ex-invver/versions/0"
    Then the response status should be 422
    And the response should have error code 42202
    When I GET "/subjects/err-ex-invver/versions/-2"
    Then the response status should be 422
    And the response should have error code 42202

  Scenario: Non-existent schema ID returns 404
    When I GET "/schemas/ids/99999"
    Then the response status should be 404
    And the response should have error code 40403

  # ==========================================================================
  # JSON 404 for unknown routes
  # ==========================================================================

  @axonops-only
  Scenario: GET unknown path returns JSON 404
    When I GET "/this/path/does/not/exist"
    Then the response status should be 404
    And the response should have error code 404
    And the response field "message" should be "HTTP 404 Not Found"

  Scenario: POST unknown path returns JSON 404
    When I POST "/nonexistent/endpoint" with body:
      """
      {"foo": "bar"}
      """
    Then the response status should be 404
    And the response should have error code 404

  # ==========================================================================
  # Error code 40406 for individually soft-deleted versions
  # ==========================================================================

  @axonops-only
  Scenario: GET individually soft-deleted version returns 40406
    Given the global compatibility level is "NONE"
    And subject "err-40406-sub" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "err-40406-sub" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "err-40406-sub"
    Then the response status should be 200
    When I get version 1 of subject "err-40406-sub"
    Then the response status should be 404
    And the response should have error code 40406
    And the audit log should contain an event:
      | event_type           | schema_delete_soft                        |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | err-40406-sub                             |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          | sha256:*                                  |
      | after_hash           |                                           |
      | context              | .                                         |
      | transport_security   | tls                                       |
      | source_ip            | *                                         |
      | user_agent           | *                                         |
      | method               | DELETE                                    |
      | path                 | /subjects/err-40406-sub/versions/1        |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |

  @axonops-only
  Scenario: GET soft-deleted version with deleted=true still returns 200
    Given the global compatibility level is "NONE"
    And subject "err-40406-del" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "err-40406-del" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "err-40406-del"
    Then the response status should be 200
    When I GET "/subjects/err-40406-del/versions/1?deleted=true"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_delete_soft                        |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | err-40406-del                             |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          | sha256:*                                  |
      | after_hash           |                                           |
      | context              | .                                         |
      | transport_security   | tls                                       |
      | source_ip            | *                                         |
      | user_agent           | *                                         |
      | method               | DELETE                                    |
      | path                 | /subjects/err-40406-del/versions/1        |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |

  @axonops-only
  Scenario: GET raw schema of individually soft-deleted version returns 40406
    Given the global compatibility level is "NONE"
    And subject "err-40406-raw" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "err-40406-raw" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "err-40406-raw"
    Then the response status should be 200
    When I GET "/subjects/err-40406-raw/versions/1/schema"
    Then the response status should be 404
    And the response should have error code 40406
    And the audit log should contain an event:
      | event_type           | schema_delete_soft                        |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | err-40406-raw                             |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          | sha256:*                                  |
      | after_hash           |                                           |
      | context              | .                                         |
      | transport_security   | tls                                       |
      | source_ip            | *                                         |
      | user_agent           | *                                         |
      | method               | DELETE                                    |
      | path                 | /subjects/err-40406-raw/versions/1        |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
