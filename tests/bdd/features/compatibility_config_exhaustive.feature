@functional
Feature: Compatibility Configuration & Testing — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive compatibility configuration and testing scenarios from the
  Confluent Schema Registry v8.1.1 test suite.

  # ==========================================================================
  # GLOBAL COMPATIBILITY CONFIGURATION
  # ==========================================================================

  Scenario: Get default global compatibility
    When I get the global config
    Then the response status should be 200
    And the response should have field "compatibilityLevel"

  Scenario: Set and get global compatibility level
    When I set the global config to "FORWARD"
    Then the response status should be 200
    When I get the global config
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    # Reset
    When I set the global config to "NONE"
    And the audit log should contain an event:
      | event_type           | config_update                  |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | config                         |
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
      | path                 | /config                        |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |

  Scenario: Set subject-level compatibility independent of global
    Given I set the global config to "NONE"
    When I set the config for subject "cc-subj-level" to "FORWARD"
    Then the response status should be 200
    When I get the config for subject "cc-subj-level"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FORWARD"
    When I get the global config
    And the response field "compatibilityLevel" should be "NONE"
    And the audit log should contain an event:
      | event_type           | config_update                  |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | config                         |
      | target_id            | cc-subj-level                  |
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
      | path                 | /config/cc-subj-level          |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |

  Scenario: Set compatibility for non-existent subject succeeds
    When I set the config for subject "cc-nonexist-subj" to "FULL"
    Then the response status should be 200
    When I get the config for subject "cc-nonexist-subj"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    And the audit log should contain an event:
      | event_type           | config_update                  |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | config                         |
      | target_id            | cc-nonexist-subj               |
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
      | path                 | /config/cc-nonexist-subj       |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |

  Scenario: Get config for subject with no subject-level config returns 404
    When I get the config for subject "cc-no-config-at-all"
    Then the response status should be 404
    And the response should have error code 40408

  Scenario: Delete subject-level config reverts to global
    Given I set the global config to "FULL"
    And I set the config for subject "cc-del-config" to "BACKWARD"
    When I delete the config for subject "cc-del-config"
    Then the response status should be 200
    When I get the config for subject "cc-del-config"
    Then the response status should be 404
    And the response should have error code 40408
    # With defaultToGlobal the fallback works
    When I GET "/config/cc-del-config?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    # Reset
    When I set the global config to "NONE"
    And the audit log should contain an event:
      | event_type           | config_delete                  |
      | outcome              | success                        |
      | actor_id             |                                |
      | actor_type           | anonymous                      |
      | auth_method          |                                |
      | role                 |                                |
      | target_type          | config                         |
      | target_id            | cc-del-config                  |
      | schema_id            |                                |
      | version              |                                |
      | schema_type          |                                |
      | before_hash          | sha256:*                       |
      | after_hash           |                                |
      | context              | .                              |
      | transport_security   | tls                            |
      | source_ip            | *                              |
      | user_agent           | *                              |
      | method               | DELETE                         |
      | path                 | /config/cc-del-config          |
      | status_code          | 200                            |
      | reason               |                                |
      | error                |                                |
      | request_body         |                                |
      | metadata             |                                |
      | timestamp            | *                              |
      | duration_ms          | *                              |
      | request_id           | *                              |

  # ==========================================================================
  # COMPATIBILITY TESTING ENDPOINTS
  # ==========================================================================

  Scenario: Test compatibility against non-existent version returns 404
    Given subject "cc-test-nover" has schema:
      """
      {"type":"record","name":"TestNV","fields":[{"name":"a","type":"string"}]}
      """
    When I check compatibility of schema against subject "cc-test-nover" version 100:
      """
      {"type":"record","name":"TestNV","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 404
    And the response should have error code 40402

  Scenario: Test compatibility against invalid version returns 422
    Given subject "cc-test-invver" has schema:
      """
      {"type":"record","name":"TestIV","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/cc-test-invver/versions/earliest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestIV\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 422

  Scenario: Backward compatibility — adding field with default is compatible
    Given subject "cc-back-compat" has compatibility level "BACKWARD"
    And subject "cc-back-compat" has schema:
      """
      {"type":"record","name":"Back","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "cc-back-compat":
      """
      {"type":"record","name":"Back","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the compatibility check should be compatible

  Scenario: Backward compatibility — adding field without default is incompatible
    Given subject "cc-back-incompat" has compatibility level "BACKWARD"
    And subject "cc-back-incompat" has schema:
      """
      {"type":"record","name":"BackI","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "cc-back-incompat":
      """
      {"type":"record","name":"BackI","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: Changing compatibility to NONE allows incompatible registration
    Given subject "cc-change-none" has compatibility level "BACKWARD"
    And subject "cc-change-none" has schema:
      """
      {"type":"record","name":"ChgNone","fields":[{"name":"f1","type":"string"}]}
      """
    # Incompatible schema rejected under BACKWARD
    When I register a schema under subject "cc-change-none":
      """
      {"type":"record","name":"ChgNone","fields":[{"name":"f1","type":"int"}]}
      """
    Then the response status should be 409
    # Change to NONE and retry
    When I set the config for subject "cc-change-none" to "NONE"
    And I register a schema under subject "cc-change-none":
      """
      {"type":"record","name":"ChgNone","fields":[{"name":"f1","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | cc-change-none                           |
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
      | path                 | /subjects/cc-change-none/versions        |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Change compatibility from FORWARD to BACKWARD enforces new rules
    Given the global compatibility level is "NONE"
    And subject "cc-fwd-to-bwd" has compatibility level "FORWARD"
    And subject "cc-fwd-to-bwd" has schema:
      """
      {"type":"record","name":"FwdBwd","fields":[{"name":"f1","type":"string"}]}
      """
    # Forward-compatible: removing a field (old can read new data)
    When I register a schema under subject "cc-fwd-to-bwd":
      """
      {"type":"record","name":"FwdBwd","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the response status should be 200
    # Switch to BACKWARD
    When I set the config for subject "cc-fwd-to-bwd" to "BACKWARD"
    # Backward-compatible: adding field WITH default succeeds
    When I register a schema under subject "cc-fwd-to-bwd":
      """
      {"type":"record","name":"FwdBwd","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"},{"name":"f3","type":"string","default":"y"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | cc-fwd-to-bwd                            |
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
      | path                 | /subjects/cc-fwd-to-bwd/versions         |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ==========================================================================
  # TRANSITIVE COMPATIBILITY VIA REST
  # ==========================================================================

  Scenario: FORWARD_TRANSITIVE — compatible with latest but not all versions
    Given the global compatibility level is "NONE"
    And subject "cc-ft-rest" has schema:
      """
      {"type":"record","name":"FTRest","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    And subject "cc-ft-rest" has schema:
      """
      {"type":"record","name":"FTRest","fields":[{"name":"f1","type":"string"}]}
      """
    When I set the config for subject "cc-ft-rest" to "FORWARD_TRANSITIVE"
    # v3 adds f3 — compatible with v2 (latest, only f1) but v1 can't read without f2 (no default)
    When I check compatibility of schema against subject "cc-ft-rest":
      """
      {"type":"record","name":"FTRest","fields":[{"name":"f1","type":"string"},{"name":"f3","type":"string"}]}
      """
    Then the compatibility check should be compatible
    When I check compatibility of schema against all versions of subject "cc-ft-rest":
      """
      {"type":"record","name":"FTRest","fields":[{"name":"f1","type":"string"},{"name":"f3","type":"string"}]}
      """
    Then the compatibility check should be incompatible

  # ==========================================================================
  # defaultToGlobal PARAMETER
  # ==========================================================================

  Scenario: defaultToGlobal returns global config when no subject config exists
    Given I set the global config to "BACKWARD_TRANSITIVE"
    When I GET "/config/cc-dtg-test?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD_TRANSITIVE"
    When I GET "/config/cc-dtg-test"
    Then the response status should be 404
    And the response should have error code 40408
    # Reset
    When I set the global config to "NONE"

  Scenario: defaultToGlobal returns subject config when it exists
    Given I set the global config to "NONE"
    And I set the config for subject "cc-dtg-subj" to "FULL_TRANSITIVE"
    When I GET "/config/cc-dtg-subj?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL_TRANSITIVE"
