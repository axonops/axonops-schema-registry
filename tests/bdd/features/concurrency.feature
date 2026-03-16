@functional @concurrency
Feature: Concurrent Operations Safety
  As a schema registry operator
  I want the registry to handle truly concurrent goroutine operations safely
  So that parallel clients do not corrupt data or produce inconsistent results

  Background:
    Given the schema registry is running
    And no subjects exist
    And the global compatibility level is "NONE"

  Scenario: Concurrent schema registration produces unique IDs
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
      | target_id            |                                          |
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

  Scenario: Concurrent identical registration is idempotent
    When 10 goroutines register the same Avro schema to subject "idempotent-subject"
    Then all concurrent results should succeed
    And all returned schema IDs should be identical
    And subject "idempotent-subject" should have exactly 1 version
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | idempotent-subject                       |
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
      | path                 | /subjects/idempotent-subject/versions    |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Concurrent deletes do not corrupt state
    Given 10 subjects each with one Avro schema
    When 10 goroutines each soft-delete their own subject
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
      | target_id            |                                |
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

  Scenario: Concurrent reads during writes return consistent data
    Given subject "rw-subject" has schema:
      """
      {"type":"record","name":"Seed","fields":[{"name":"x","type":"int"}]}
      """
    When 5 writer goroutines add versions and 5 reader goroutines read latest from subject "rw-subject"
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
      | target_id            | rw-subject                               |
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
      | path                 | /subjects/rw-subject/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Mode switch blocks concurrent writes
    When I set the global mode to "READONLY"
    And 5 goroutines attempt to register schemas to subject "blocked-subject"
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
