@functional @contexts
Feature: Contexts — Core Behavior
  Contexts allow multi-tenant schema isolation. Subjects are assigned to
  contexts using the :.contextname:subject prefix format (Confluent-compatible).
  The default context is "." which is used for subjects without a prefix.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # GET /contexts — DEFAULT
  # ==========================================================================

  Scenario: Default context returned when no context-prefixed subjects exist
    When I GET "/contexts"
    Then the response status should be 200
    And the response body should contain "."

  Scenario: Default context "." is always present in fresh registry
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."

  # ==========================================================================
  # CONTEXT-PREFIXED SUBJECT REGISTRATION
  # ==========================================================================

  Scenario: Register schema with context prefix — context appears in GET /contexts
    When I POST "/subjects/:.testctx:ctx-subject1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CtxTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".testctx"
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | :.testctx:ctx-subject1                    |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           | sha256:*                                  |
      | context              | .testctx                                  |
      | transport_security   | tls                                       |
      | method               | POST                                      |
      | path                 | /subjects/:.testctx:ctx-subject1/versions |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
      | source_ip            | *                                         |
      | user_agent           | *                                         |

  Scenario: Register schemas in multiple contexts — all contexts listed
    When I POST "/subjects/:.ctx-alpha:multi-ctx/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AlphaCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx-beta:multi-ctx/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"BetaCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".ctx-alpha"
    And the response array should contain ".ctx-beta"
    And the response array should contain "."
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | :.ctx-beta:multi-ctx                      |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           | sha256:*                                  |
      | context              | .ctx-beta                                 |
      | transport_security   | tls                                       |
      | method               | POST                                      |
      | path                 | /subjects/:.ctx-beta:multi-ctx/versions   |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
      | source_ip            | *                                         |
      | user_agent           | *                                         |

  Scenario: Contexts are created implicitly on first schema registration
    # No explicit context creation API — contexts appear when schemas are registered
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should not contain ".newctx"
    When I POST "/subjects/:.newctx:first-schema/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"NewCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".newctx"
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | :.newctx:first-schema                     |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           | sha256:*                                  |
      | context              | .newctx                                   |
      | transport_security   | tls                                       |
      | method               | POST                                      |
      | path                 | /subjects/:.newctx:first-schema/versions  |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
      | source_ip            | *                                         |
      | user_agent           | *                                         |

  Scenario: GET /contexts returns sorted list
    When I POST "/subjects/:.zeta:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Zeta\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.alpha:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Alpha\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should contain ".alpha"
    And the response array should contain ".zeta"
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | :.alpha:s1                                |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           | sha256:*                                  |
      | context              | .alpha                                    |
      | transport_security   | tls                                       |
      | method               | POST                                      |
      | path                 | /subjects/:.alpha:s1/versions             |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
      | source_ip            | *                                         |
      | user_agent           | *                                         |

  Scenario: Registering schema in default context via qualified subject
    # :.: prefix maps to the default context
    When I POST "/subjects/mysubject/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DefaultCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects"
    Then the response status should be 200
    And the response array should contain "mysubject"
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | mysubject                                 |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           | sha256:*                                  |
      | context              | .                                         |
      | transport_security   | tls                                       |
      | method               | POST                                      |
      | path                 | /subjects/mysubject/versions              |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
      | source_ip            | *                                         |
      | user_agent           | *                                         |

  Scenario: Context names are case-sensitive
    When I POST "/subjects/:.CaseSensitive:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Upper\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.casesensitive:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Lower\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".CaseSensitive"
    And the response array should contain ".casesensitive"
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | :.casesensitive:s1                        |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           | sha256:*                                  |
      | context              | .casesensitive                            |
      | transport_security   | tls                                       |
      | method               | POST                                      |
      | path                 | /subjects/:.casesensitive:s1/versions     |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
      | source_ip            | *                                         |
      | user_agent           | *                                         |

  # ==========================================================================
  # CONTEXT ISOLATION — BASIC
  # ==========================================================================

  Scenario: Same subject name in different contexts are independent
    When I POST "/subjects/:.iso-a:shared-name/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IsoA\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.iso-b:shared-name/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IsoB\",\"fields\":[{\"name\":\"y\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.iso-a:shared-name/versions/1"
    Then the response status should be 200
    And the response body should contain "IsoA"
    When I GET "/subjects/:.iso-b:shared-name/versions/1"
    Then the response status should be 200
    And the response body should contain "IsoB"
    And the audit log should contain an event:
      | event_type           | schema_register                           |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | :.iso-b:shared-name                       |
      | schema_id            | *                                         |
      | version              | *                                         |
      | schema_type          | AVRO                                      |
      | before_hash          |                                           |
      | after_hash           | sha256:*                                  |
      | context              | .iso-b                                    |
      | transport_security   | tls                                       |
      | method               | POST                                      |
      | path                 | /subjects/:.iso-b:shared-name/versions    |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
      | source_ip            | *                                         |
      | user_agent           | *                                         |

  Scenario: Delete context-prefixed subject
    When I POST "/subjects/:.del-ctx:to-delete/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DelCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/:.del-ctx:to-delete"
    Then the response status should be 200
    When I GET "/subjects/:.del-ctx:to-delete/versions"
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type           | subject_delete_soft                       |
      | outcome              | success                                   |
      | actor_id             |                                           |
      | actor_type           | anonymous                                 |
      | auth_method          |                                           |
      | role                 |                                           |
      | target_type          | subject                                   |
      | target_id            | :.del-ctx:to-delete                       |
      | schema_id            |                                           |
      | version              |                                           |
      | schema_type          | AVRO                                      |
      | before_hash          | sha256:*                                  |
      | after_hash           |                                           |
      | context              | .del-ctx                                  |
      | transport_security   | tls                                       |
      | method               | DELETE                                    |
      | path                 | /subjects/:.del-ctx:to-delete             |
      | status_code          | 200                                       |
      | reason               |                                           |
      | error                |                                           |
      | request_body         |                                           |
      | metadata             |                                           |
      | timestamp            | *                                         |
      | duration_ms          | *                                         |
      | request_id           | *                                         |
      | source_ip            | *                                         |
      | user_agent           | *                                         |
