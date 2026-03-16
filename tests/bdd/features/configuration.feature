@functional
Feature: Configuration
  As an operator, I want to manage compatibility levels and modes

  Scenario: Get default global compatibility
    When I get the global config
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  Scenario: Set global compatibility
    When I set the global config to "FULL"
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "FULL"
    And the audit log should contain an event:
      | event_type           | config_update          |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | config                 |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          | sha256:*               |
      | after_hash           | sha256:*               |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | PUT                    |
      | path                 | /config                |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |

  Scenario: Set per-subject compatibility
    When I set the config for subject "my-subject" to "NONE"
    Then the response status should be 200
    When I get the config for subject "my-subject"
    Then the response field "compatibilityLevel" should be "NONE"
    And the audit log should contain an event:
      | event_type           | config_update          |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | config                 |
      | target_id            | my-subject             |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           | sha256:*               |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | PUT                    |
      | path                 | /config/my-subject     |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |

  Scenario: Delete per-subject compatibility falls back to global with defaultToGlobal
    Given subject "my-subject" has compatibility level "FORWARD"
    When I delete the config for subject "my-subject"
    Then the response status should be 200
    When I get the config for subject "my-subject"
    Then the response status should be 404
    When I GET "/config/my-subject?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"
    And the audit log should contain an event:
      | event_type           | config_delete          |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | config                 |
      | target_id            | my-subject             |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          | sha256:*               |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | DELETE                 |
      | path                 | /config/my-subject     |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |

  Scenario: Delete global config reverts to default
    When I set the global config to "FULL_TRANSITIVE"
    Then the response status should be 200
    When I delete the global config
    Then the response status should be 200
    When I get the global config
    Then the response field "compatibilityLevel" should be "BACKWARD"
    And the audit log should contain an event:
      | event_type           | config_delete          |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | config                 |
      | target_id            | _global                |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          | sha256:*               |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | DELETE                 |
      | path                 | /config                |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |

  Scenario: Invalid compatibility level returns 422
    When I set the global config to "INVALID_LEVEL"
    Then the response status should be 422
    And the response should have error code 42203
    And the audit log should contain an event:
      | event_type           | config_update          |
      | outcome              | failure                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | config                 |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | PUT                    |
      | path                 | /config                |
      | status_code          | 422                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |

  Scenario: Set all valid compatibility levels
    When I set the global config to "NONE"
    Then the response status should be 200
    When I set the global config to "BACKWARD"
    Then the response status should be 200
    When I set the global config to "BACKWARD_TRANSITIVE"
    Then the response status should be 200
    When I set the global config to "FORWARD"
    Then the response status should be 200
    When I set the global config to "FORWARD_TRANSITIVE"
    Then the response status should be 200
    When I set the global config to "FULL"
    Then the response status should be 200
    When I set the global config to "FULL_TRANSITIVE"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | config_update          |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | config                 |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          | sha256:*               |
      | after_hash           | sha256:*               |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | PUT                    |
      | path                 | /config                |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |

  Scenario: Get default global mode
    When I get the global mode
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  Scenario: Set global mode
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READONLY"
    And the audit log should contain an event:
      | event_type           | mode_update            |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | mode                   |
      | target_id            | _global                |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           | sha256:*               |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | PUT                    |
      | path                 | /mode                  |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |
