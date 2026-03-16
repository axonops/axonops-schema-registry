@functional
Feature: Mode Management
  As an operator, I want to manage read/write modes globally and per subject

  Scenario: Get default global mode
    When I get the global mode
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  Scenario: Set global mode to READONLY
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
      | before_hash          | *                      |
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

  Scenario: Set global mode to IMPORT
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "IMPORT"
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
      | before_hash          | *                      |
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

  Scenario: Set per-subject mode
    When I set the mode for subject "my-subject" to "READONLY"
    Then the response status should be 200
    When I get the mode for subject "my-subject"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"
    And the audit log should contain an event:
      | event_type           | mode_update            |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | mode                   |
      | target_id            | my-subject             |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          | *                      |
      | after_hash           | sha256:*               |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | PUT                    |
      | path                 | /mode/my-subject       |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |

  Scenario: Delete per-subject mode falls back to global with defaultToGlobal
    Given subject "mode-subj" has mode "READONLY"
    When I delete the mode for subject "mode-subj"
    Then the response status should be 200
    When I get the mode for subject "mode-subj"
    Then the response status should be 404
    When I GET "/mode/mode-subj?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"
    And the audit log should contain an event:
      | event_type           | mode_delete            |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | mode                   |
      | target_id            | mode-subj              |
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
      | path                 | /mode/mode-subj        |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |

  Scenario: Per-subject mode isolation
    When I set the mode for subject "subj-a" to "READONLY"
    And I set the mode for subject "subj-b" to "IMPORT"
    When I get the mode for subject "subj-a"
    Then the response field "mode" should be "READONLY"
    When I get the mode for subject "subj-b"
    Then the response field "mode" should be "IMPORT"
    And the audit log should contain an event:
      | event_type           | mode_update            |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | mode                   |
      | target_id            | subj-b                 |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          | *                      |
      | after_hash           | sha256:*               |
      | context              | .                      |
      | transport_security   | tls                    |
      | source_ip            | *                      |
      | user_agent           | *                      |
      | method               | PUT                    |
      | path                 | /mode/subj-b           |
      | status_code          | 200                    |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |
