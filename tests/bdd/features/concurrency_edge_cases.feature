@functional @concurrency @edge-case
Feature: Concurrency Edge Cases
  As a schema registry operator
  I want concurrent delete + register and mode change + registration to be safe
  So that race conditions do not corrupt data or produce unexpected errors

  Background:
    Given the schema registry is running
    And no subjects exist
    And the global compatibility level is "NONE"

  # ---------------------------------------------------------------------------
  # Concurrent soft-delete + re-register race on the same subject
  # ---------------------------------------------------------------------------

  Scenario: Concurrent soft-delete and re-register on the same subject does not corrupt state
    # Setup: create a subject with one schema version
    Given subject "race-subject" has schema:
      """
      {"type":"record","name":"Race","fields":[{"name":"id","type":"int"}]}
      """
    # Soft-delete the subject
    When I delete subject "race-subject"
    Then the response status should be 200
    # Re-register a new schema under the same subject — should succeed
    When I register a schema under subject "race-subject":
      """
      {"type":"record","name":"Race","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    Then the response status should be 200
    # Subject should be visible again and have exactly 1 version (new registration after soft-delete)
    When I list versions of subject "race-subject"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | race-subject                             |
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
      | path                 | /subjects/race-subject/versions          |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Concurrent identical registrations after soft-delete converge to one version
    Given subject "idempotent-del" has schema:
      """
      {"type":"record","name":"Idem","fields":[{"name":"v","type":"string"}]}
      """
    When I delete subject "idempotent-del"
    Then the response status should be 200
    # Multiple goroutines attempt to re-register the same schema
    When 10 goroutines register the same Avro schema to subject "idempotent-del"
    Then all concurrent results should succeed
    And all returned schema IDs should be identical
    And subject "idempotent-del" should have exactly 1 version
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | idempotent-del                           |
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
      | path                 | /subjects/idempotent-del/versions        |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ---------------------------------------------------------------------------
  # Mode switch during concurrent registration
  # ---------------------------------------------------------------------------

  Scenario: Setting READONLY mode blocks concurrent writes gracefully
    # Start in READWRITE mode with a seed schema
    Given subject "mode-race" has schema:
      """
      {"type":"record","name":"ModeRace","fields":[{"name":"id","type":"int"}]}
      """
    # Switch to READONLY
    When I set the global mode to "READONLY"
    # All attempts to register should fail with 422 (mode is READONLY)
    And 5 goroutines attempt to register schemas to subject "mode-race"
    Then all concurrent results should have status 422
    And the audit log should contain an event:
      | event_type           | mode_update                    |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | mode                           |
      | target_id            | _global                        |
      | schema_id            |                                |
      | version              |                                |
      | schema_type          |                                |
      | before_hash          | *                              |
      | after_hash           | sha256:*                       |
      | context              | .                              |
      | transport_security   | tls                            |
      | source_ip            | *                              |
      | user_agent           | *                              |
      | method               | PUT                            |
      | path                 | /mode                          |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |
    # Restore READWRITE for cleanup
    When I set the global mode to "READWRITE"

  # ---------------------------------------------------------------------------
  # Concurrent deletes on the same subject
  # ---------------------------------------------------------------------------

  Scenario: Multiple concurrent soft-deletes of the same subject do not cause errors
    # Create 5 subjects then concurrently soft-delete them
    Given 5 subjects each with one Avro schema
    When 5 goroutines each soft-delete their own subject
    Then all concurrent results should succeed
    And GET /subjects should return an empty array
    And the audit log should contain an event:
      | event_type           | subject_delete_soft            |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | subject                        |
      | target_id            | *                              |
      | schema_id            |                                |
      | version              |                                |
      | schema_type          | AVRO                           |
      | before_hash          | sha256:*                       |
      | after_hash           |                                |
      | context              | .                              |
      | transport_security   | tls                            |
      | source_ip            | *                              |
      | user_agent           | *                              |
      | method               | DELETE                         |
      | path                 | /subjects/                     |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |

  # ---------------------------------------------------------------------------
  # Concurrent writes produce sequential version numbers
  # ---------------------------------------------------------------------------

  Scenario: Concurrent writes to different subjects all succeed with unique IDs
    When 10 goroutines each register a unique Avro schema to separate subjects
    Then all concurrent results should succeed
    And all returned schema IDs should be unique
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | *                                        |
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
      | path                 | /subjects/                               |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Mixed readers and writers under load do not produce server errors
    Given subject "load-subject" has schema:
      """
      {"type":"record","name":"Load","fields":[{"name":"x","type":"int"}]}
      """
    When 5 writer goroutines add versions and 5 reader goroutines read latest from subject "load-subject"
    Then no concurrent results should have a 500 status
    And all reader responses should contain a valid schema
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | load-subject                             |
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
      | path                 | /subjects/load-subject/versions          |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
