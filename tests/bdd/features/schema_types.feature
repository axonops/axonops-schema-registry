@functional
Feature: Schema Types
  The registry supports Avro, Protobuf, and JSON Schema types

  Scenario: Register Avro schema
    When I register a schema under subject "avro-test":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"string"}]}
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
      | target_id            | avro-test                                |
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
      | path                 | /subjects/avro-test/versions             |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Register Protobuf schema
    When I register a "PROTOBUF" schema under subject "proto-test":
      """
      syntax = "proto3";
      message Event {
        string id = 1;
      }
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
      | target_id            | proto-test                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | PROTOBUF                                 |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/proto-test/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Register JSON Schema
    When I register a "JSON" schema under subject "json-test":
      """
      {"type":"object","properties":{"id":{"type":"string"}},"required":["id"]}
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
      | target_id            | json-test                                |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-test/versions             |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Get schema by ID shows schemaType for Protobuf
    When I register a "PROTOBUF" schema under subject "proto-type-test":
      """
      syntax = "proto3";
      message Msg {
        string name = 1;
      }
      """
    And I store the response field "id" as "schema_id"
    And I get the stored schema by ID
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"
