@functional @compatibility
Feature: Compatibility Checking
  The registry enforces compatibility rules when registering new schema versions

  Scenario: Backward compatible schema is accepted
    Given the global compatibility level is "BACKWARD"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
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
      | target_id            | user-value                               |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/user-value/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Backward incompatible schema is rejected
    Given the global compatibility level is "BACKWARD"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: NONE compatibility allows any change
    Given the global compatibility level is "NONE"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "user-value":
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"long"}]}
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
      | target_id            | user-value                               |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/user-value/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Check compatibility endpoint - compatible
    Given the global compatibility level is "BACKWARD"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I check compatibility of schema against subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the compatibility check should be compatible

  Scenario: Check compatibility endpoint - incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I check compatibility of schema against subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: Per-subject compatibility overrides global
    Given the global compatibility level is "BACKWARD"
    And subject "flexible-value" has compatibility level "NONE"
    And subject "flexible-value" has schema:
      """
      {"type":"record","name":"A","fields":[{"name":"x","type":"string"}]}
      """
    When I register a schema under subject "flexible-value":
      """
      {"type":"record","name":"B","fields":[{"name":"y","type":"long"}]}
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
      | target_id            | flexible-value                           |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/flexible-value/versions        |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
