@functional @contexts
Feature: Contexts — Edge Cases and Error Conditions
  Verify error handling and boundary conditions for context operations.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # VALID CONTEXT NAMES
  # ==========================================================================

  Scenario: Context name with dash is valid
    When I POST "/subjects/:.my-context:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DashCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".my-context"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.my-context:s1                              |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .my-context                                  |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.my-context:s1/versions           |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  Scenario: Context name with underscore is valid
    When I POST "/subjects/:.my_context:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UnderCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".my_context"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.my_context:s1                              |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .my_context                                  |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.my_context:s1/versions           |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  Scenario: Context name with numbers is valid
    When I POST "/subjects/:.ctx123:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"NumCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".ctx123"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ctx123:s1                                  |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ctx123                                      |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ctx123:s1/versions               |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  Scenario: Context name with mixed case is valid
    When I POST "/subjects/:.MyContext:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MixedCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".MyContext"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.MyContext:s1                                |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .MyContext                                   |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.MyContext:s1/versions             |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # SCHEMA DEDUP WITHIN CONTEXT
  # ==========================================================================

  Scenario: Same schema registered in same subject returns existing
    When I POST "/subjects/:.dedup-ctx:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Dedup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    When I POST "/subjects/:.dedup-ctx:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Dedup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.dedup-ctx:s1                               |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .dedup-ctx                                   |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.dedup-ctx:s1/versions            |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  Scenario: Same schema in different subjects within same context shares ID
    When I POST "/subjects/:.shared-id:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SharedId\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "shared_id"
    When I POST "/subjects/:.shared-id:s2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SharedId\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "shared_id"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.shared-id:s2                               |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .shared-id                                   |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.shared-id:s2/versions            |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # OPERATIONS ON NON-EXISTENT CONTEXT DATA
  # ==========================================================================

  Scenario: Get versions for non-existent subject in context returns 404
    When I GET "/subjects/:.nonexist-ctx:no-such-subject/versions"
    Then the response status should be 404

  Scenario: Get specific version for non-existent subject in context returns 404
    When I GET "/subjects/:.nonexist-ctx2:no-such/versions/1"
    Then the response status should be 404

  Scenario: Delete non-existent subject in context returns 404
    When I DELETE "/subjects/:.nonexist-ctx3:no-such"
    Then the response status should be 404

  Scenario: Config for non-existent subject in context returns 404
    When I GET "/config/:.nonexist-ctx4:no-such"
    Then the response status should be 404

  Scenario: Multiple schemas in same context get sequential IDs
    When I POST "/subjects/:.seq-ctx:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Seq1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    When I POST "/subjects/:.seq-ctx:s2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Seq2\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 2
    When I POST "/subjects/:.seq-ctx:s3/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Seq3\",\"fields\":[{\"name\":\"c\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 3
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.seq-ctx:s3                                 |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .seq-ctx                                     |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.seq-ctx:s3/versions              |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
