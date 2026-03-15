@functional
Feature: Configuration and Mode Management Advanced
  As an operator, I want to verify configuration isolation, fallback behavior,
  case handling, and edge cases for compatibility levels and modes

  # --- Config isolation ---

  Scenario: Setting subject-A config does not affect subject-B
    When I set the config for subject "subj-a" to "NONE"
    Then the response status should be 200
    When I set the config for subject "subj-b" to "FULL"
    Then the response status should be 200
    When I get the config for subject "subj-a"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "NONE"
    When I get the config for subject "subj-b"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    And the audit log should contain an event:
      | event_type           | config_update                 |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | config                        |
      | target_id            | subj-b                        |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
      | path                 | /config/subj-b                |
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

  # --- Subject config overrides global during registration ---

  Scenario: Subject config overrides global during schema registration
    Given the global compatibility level is "BACKWARD"
    And subject "override-test" has compatibility level "NONE"
    And subject "override-test" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "override-test":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                       |
      | outcome              | success                               |
      | actor_id             |                                       |
      | actor_type           | anonymous                             |
      | auth_method          |                                       |
      | role                 |                                       |
      | target_type          | subject                               |
      | target_id            | override-test                         |
      | schema_id            | *                                     |
      | version              | *                                     |
      | schema_type          | AVRO                                  |
      | before_hash          |                                       |
      | after_hash           | sha256:*                              |
      | context              | .                                     |
      | transport_security   | tls                                   |
      | method               | POST                                  |
      | path                 | /subjects/override-test/versions      |
      | status_code          | 200                                   |
      | reason               |                                       |
      | error                |                                       |
      | request_body         |                                       |
      | metadata             |                                       |
      | timestamp            | *                                     |
      | duration_ms          | *                                     |
      | request_id           | *                                     |
      | source_ip            | *                                     |
      | user_agent           | *                                     |

  # --- Delete subject config falls back to global ---

  Scenario: Delete subject config causes fallback to global with defaultToGlobal
    Given the global compatibility level is "FULL"
    And subject "fallback-cfg" has compatibility level "NONE"
    When I get the config for subject "fallback-cfg"
    Then the response field "compatibilityLevel" should be "NONE"
    When I delete the config for subject "fallback-cfg"
    Then the response status should be 200
    When I GET "/config/fallback-cfg"
    Then the response status should be 404
    When I GET "/config/fallback-cfg?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    And the audit log should contain an event:
      | event_type           | config_delete                 |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | config                        |
      | target_id            | fallback-cfg                  |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | sha256:*                      |
      | after_hash           |                               |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | DELETE                        |
      | path                 | /config/fallback-cfg          |
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

  # --- Delete global config reverts to BACKWARD default ---

  Scenario: Delete global config reverts to BACKWARD default
    When I set the global config to "FULL_TRANSITIVE"
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "FULL_TRANSITIVE"
    When I delete the global config
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "BACKWARD"
    And the audit log should contain an event:
      | event_type           | config_delete                 |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | config                        |
      | target_id            | _global                       |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | sha256:*                      |
      | after_hash           |                               |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | DELETE                        |
      | path                 | /config                       |
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

  # --- Delete non-existent subject config returns 404 ---

  Scenario: Delete config for non-existent subject returns 404
    When I delete the config for subject "never-configured-subject"
    Then the response status should be 404
    And the response should have error code 40401
    And the audit log should contain an event:
      | event_type           | config_delete                         |
      | outcome              | failure                               |
      | actor_id             |                                       |
      | actor_type           | anonymous                             |
      | auth_method          |                                       |
      | role                 |                                       |
      | target_type          | config                                |
      | target_id            | never-configured-subject              |
      | schema_id            |                                       |
      | version              |                                       |
      | schema_type          |                                       |
      | before_hash          |                                       |
      | after_hash           |                                       |
      | context              |                                       |
      | transport_security   | tls                                   |
      | method               | DELETE                                |
      | path                 | /config/never-configured-subject      |
      | status_code          | 404                                   |
      | reason               | not_found                             |
      | error                |                                       |
      | request_body         |                                       |
      | metadata             |                                       |
      | timestamp            | *                                     |
      | duration_ms          | *                                     |
      | request_id           | *                                     |
      | source_ip            | *                                     |
      | user_agent           | *                                     |

  # --- Invalid compatibility level returns 422 ---

  Scenario: Invalid compatibility level returns 422 with error code 42203
    When I set the global config to "INVALID_LEVEL"
    Then the response status should be 422
    And the response should have error code 42203
    And the response should have field "message"
    And the audit log should contain an event:
      | event_type           | config_update                 |
      | outcome              | failure                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | config                        |
      | target_id            | _global                       |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          |                               |
      | after_hash           |                               |
      | context              |                               |
      | transport_security   | tls                           |
      | method               | PUT                           |
      | path                 | /config                       |
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

  # --- Case insensitivity for compatibility levels ---

  Scenario: Compatibility level is case insensitive
    When I PUT "/config" with body:
      """
      {"compatibility": "backward"}
      """
    Then the response status should be 200
    And the response field "compatibility" should be "BACKWARD"
    When I PUT "/config" with body:
      """
      {"compatibility": "Forward_Transitive"}
      """
    Then the response status should be 200
    And the response field "compatibility" should be "FORWARD_TRANSITIVE"
    And the audit log should contain an event:
      | event_type           | config_update                 |
      | outcome              | success                       |
      | actor_id             |                               |
      | actor_type           | anonymous                     |
      | auth_method          |                               |
      | role                 |                               |
      | target_type          | config                        |
      | target_id            | _global                       |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          | *                             |
      | after_hash           | sha256:*                      |
      | context              | .                             |
      | transport_security   | tls                           |
      | method               | PUT                           |
      | path                 | /config                       |
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

  # --- Set and get all 3 valid mode values ---

  Scenario: Set and get all valid mode values
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READONLY"
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "IMPORT"
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

  # --- Invalid mode value returns 422 ---

  Scenario: Invalid mode value returns 422 with error code 42204
    When I PUT "/mode" with body:
      """
      {"mode": "INVALID_MODE"}
      """
    Then the response status should be 422
    And the response should have error code 42204
    And the response should have field "message"
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
      | before_hash          |                               |
      | after_hash           |                               |
      | context              |                               |
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

  # --- Delete non-existent subject mode returns 404 ---

  Scenario: Delete mode for non-existent subject returns 404
    When I delete the mode for subject "never-moded-subject"
    Then the response status should be 404
    And the response should have error code 40401
    And the audit log should contain an event:
      | event_type           | mode_delete                           |
      | outcome              | failure                               |
      | actor_id             |                                       |
      | actor_type           | anonymous                             |
      | auth_method          |                                       |
      | role                 |                                       |
      | target_type          | mode                                  |
      | target_id            | never-moded-subject                   |
      | schema_id            |                                       |
      | version              |                                       |
      | schema_type          |                                       |
      | before_hash          |                                       |
      | after_hash           |                                       |
      | context              |                                       |
      | transport_security   | tls                                   |
      | method               | DELETE                                |
      | path                 | /mode/never-moded-subject             |
      | status_code          | 404                                   |
      | reason               | not_found                             |
      | error                |                                       |
      | request_body         |                                       |
      | metadata             |                                       |
      | timestamp            | *                                     |
      | duration_ms          | *                                     |
      | request_id           | *                                     |
      | source_ip            | *                                     |
      | user_agent           | *                                     |

  # --- Mode fallback to global ---

  Scenario: Subject mode not set returns 404 without defaultToGlobal
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the mode for subject "no-mode-set-subject"
    Then the response status should be 404
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

  Scenario: Subject mode not set falls back to global with defaultToGlobal
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I GET "/mode/no-mode-set-subject2?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"
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

  # --- Per-subject mode set and retrieved independently from global ---

  Scenario: Per-subject mode is independent from global mode
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I set the mode for subject "independent-mode-a" to "READONLY"
    Then the response status should be 200
    When I set the mode for subject "independent-mode-b" to "IMPORT"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"
    When I get the mode for subject "independent-mode-a"
    Then the response field "mode" should be "READONLY"
    When I get the mode for subject "independent-mode-b"
    Then the response field "mode" should be "IMPORT"
    And the audit log should contain an event:
      | event_type           | mode_update                           |
      | outcome              | success                               |
      | actor_id             |                                       |
      | actor_type           | anonymous                             |
      | auth_method          |                                       |
      | role                 |                                       |
      | target_type          | mode                                  |
      | target_id            | independent-mode-b                    |
      | schema_id            |                                       |
      | version              |                                       |
      | schema_type          |                                       |
      | before_hash          | *                                     |
      | after_hash           | sha256:*                              |
      | context              | .                                     |
      | transport_security   | tls                                   |
      | method               | PUT                                   |
      | path                 | /mode/independent-mode-b              |
      | status_code          | 200                                   |
      | reason               |                                       |
      | error                |                                       |
      | request_body         |                                       |
      | metadata             |                                       |
      | timestamp            | *                                     |
      | duration_ms          | *                                     |
      | request_id           | *                                     |
      | source_ip            | *                                     |
      | user_agent           | *                                     |
