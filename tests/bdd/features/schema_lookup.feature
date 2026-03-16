@functional
Feature: Schema Lookup
  As a developer, I want to check if a schema already exists under a subject
  without registering it, using the POST /subjects/{subject} endpoint

  Scenario: Lookup existing Avro schema returns subject, id, version, and schema
    Given subject "order-value" has schema:
      """
      {"type":"record","name":"Order","namespace":"com.example","fields":[{"name":"order_id","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"}]}
      """
    When I lookup schema in subject "order-value":
      """
      {"type":"record","name":"Order","namespace":"com.example","fields":[{"name":"order_id","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "subject" should be "order-value"
    And the response should have field "id"
    And the response field "version" should be 1
    And the response should have field "schema"
    And the audit log should contain an event:
      | event_type           | schema_lookup                    |
      | outcome              | success                          |
      | actor_id             |                                  |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | order-value                      |
      | schema_id            | *                                |
      | version              | *                                |
      | schema_type          | AVRO                             |
      | before_hash          |                                  |
      | after_hash           |                                  |
      | context              | .                                |
      | transport_security   | tls                              |
      | source_ip            | *                                |
      | user_agent           | *                                |
      | method               | POST                             |
      | path                 | /subjects/order-value            |
      | status_code          | 200                              |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           | *                                |

  Scenario: Lookup non-existent schema in an existing subject returns 404
    Given subject "customer-value" has schema:
      """
      {"type":"record","name":"Customer","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    When I lookup schema in subject "customer-value":
      """
      {"type":"record","name":"Address","fields":[{"name":"street","type":"string"},{"name":"city","type":"string"},{"name":"zip","type":"string"}]}
      """
    Then the response status should be 404
    And the response should have error code 40403

  Scenario: Lookup schema in a non-existent subject returns 404
    When I lookup schema in subject "no-such-subject":
      """
      {"type":"record","name":"Ghost","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Lookup existing Protobuf schema with schemaType specified
    Given subject "proto-lookup" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package inventory;
      message Product {
        string sku = 1;
        string name = 2;
        int32 quantity = 3;
      }
      """
    When I lookup a "PROTOBUF" schema in subject "proto-lookup":
      """
      syntax = "proto3";
      package inventory;
      message Product {
        string sku = 1;
        string name = 2;
        int32 quantity = 3;
      }
      """
    Then the response status should be 200
    And the response field "subject" should be "proto-lookup"
    And the response should have field "id"
    And the response field "version" should be 1
    And the audit log should contain an event:
      | event_type           | schema_lookup                    |
      | outcome              | success                          |
      | actor_id             |                                  |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | proto-lookup                     |
      | schema_id            | *                                |
      | version              | *                                |
      | schema_type          | PROTOBUF                         |
      | before_hash          |                                  |
      | after_hash           |                                  |
      | context              | .                                |
      | transport_security   | tls                              |
      | source_ip            | *                                |
      | user_agent           | *                                |
      | method               | POST                             |
      | path                 | /subjects/proto-lookup           |
      | status_code          | 200                              |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           | *                                |

  Scenario: Lookup existing JSON Schema with schemaType specified
    Given subject "json-lookup" has "JSON" schema:
      """
      {"type":"object","properties":{"event_type":{"type":"string"},"timestamp":{"type":"integer"},"payload":{"type":"object"}},"required":["event_type","timestamp"]}
      """
    When I lookup a "JSON" schema in subject "json-lookup":
      """
      {"type":"object","properties":{"event_type":{"type":"string"},"timestamp":{"type":"integer"},"payload":{"type":"object"}},"required":["event_type","timestamp"]}
      """
    Then the response status should be 200
    And the response field "subject" should be "json-lookup"
    And the response should have field "id"
    And the response field "version" should be 1
    And the audit log should contain an event:
      | event_type           | schema_lookup                    |
      | outcome              | success                          |
      | actor_id             |                                  |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | json-lookup                      |
      | schema_id            | *                                |
      | version              | *                                |
      | schema_type          | JSON                             |
      | before_hash          |                                  |
      | after_hash           |                                  |
      | context              | .                                |
      | transport_security   | tls                              |
      | source_ip            | *                                |
      | user_agent           | *                                |
      | method               | POST                             |
      | path                 | /subjects/json-lookup            |
      | status_code          | 200                              |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           | *                                |

  Scenario: Lookup returns correct version when multiple versions exist
    Given the global compatibility level is "NONE"
    And subject "multi-ver-lookup" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"name","type":"string"},{"name":"value","type":"double"}]}
      """
    And subject "multi-ver-lookup" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"name","type":"string"},{"name":"value","type":"double"},{"name":"unit","type":"string"}]}
      """
    And subject "multi-ver-lookup" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"name","type":"string"},{"name":"value","type":"double"},{"name":"unit","type":"string"},{"name":"tags","type":{"type":"map","values":"string"}}]}
      """
    When I lookup schema in subject "multi-ver-lookup":
      """
      {"type":"record","name":"Metric","fields":[{"name":"name","type":"string"},{"name":"value","type":"double"},{"name":"unit","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "version" should be 2
    And the response field "subject" should be "multi-ver-lookup"
    And the audit log should contain an event:
      | event_type           | schema_lookup                    |
      | outcome              | success                          |
      | actor_id             |                                  |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | multi-ver-lookup                 |
      | schema_id            | *                                |
      | version              | *                                |
      | schema_type          | AVRO                             |
      | before_hash          |                                  |
      | after_hash           |                                  |
      | context              | .                                |
      | transport_security   | tls                              |
      | source_ip            | *                                |
      | user_agent           | *                                |
      | method               | POST                             |
      | path                 | /subjects/multi-ver-lookup       |
      | status_code          | 200                              |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           | *                                |

  Scenario: Lookup after soft-delete returns 404
    Given subject "del-lookup" has schema:
      """
      {"type":"record","name":"Session","fields":[{"name":"session_id","type":"string"},{"name":"user_id","type":"string"},{"name":"started_at","type":"long"}]}
      """
    When I delete subject "del-lookup"
    Then the response status should be 200
    When I lookup schema in subject "del-lookup":
      """
      {"type":"record","name":"Session","fields":[{"name":"session_id","type":"string"},{"name":"user_id","type":"string"},{"name":"started_at","type":"long"}]}
      """
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type           | subject_delete_soft              |
      | outcome              | success                          |
      | actor_id             |                                  |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | del-lookup                       |
      | schema_id            |                                  |
      | version              |                                  |
      | schema_type          | AVRO                             |
      | before_hash          | sha256:*                         |
      | after_hash           |                                  |
      | context              | .                                |
      | transport_security   | tls                              |
      | source_ip            | *                                |
      | user_agent           | *                                |
      | method               | DELETE                           |
      | path                 | /subjects/del-lookup             |
      | status_code          | 200                              |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           | *                                |

  Scenario: Lookup after soft-delete with deleted flag returns the schema
    Given subject "del-lookup-recover" has schema:
      """
      {"type":"record","name":"AuditLog","fields":[{"name":"action","type":"string"},{"name":"actor","type":"string"},{"name":"timestamp","type":"long"}]}
      """
    When I delete subject "del-lookup-recover"
    Then the response status should be 200
    When I lookup schema in subject "del-lookup-recover" with deleted:
      """
      {"type":"record","name":"AuditLog","fields":[{"name":"action","type":"string"},{"name":"actor","type":"string"},{"name":"timestamp","type":"long"}]}
      """
    Then the response status should be 200
    And the response field "subject" should be "del-lookup-recover"
    And the response field "version" should be 1
    And the response should have field "id"
    And the audit log should contain an event:
      | event_type           | subject_delete_soft              |
      | outcome              | success                          |
      | actor_id             |                                  |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | del-lookup-recover               |
      | schema_id            |                                  |
      | version              |                                  |
      | schema_type          | AVRO                             |
      | before_hash          | sha256:*                         |
      | after_hash           |                                  |
      | context              | .                                |
      | transport_security   | tls                              |
      | source_ip            | *                                |
      | user_agent           | *                                |
      | method               | DELETE                           |
      | path                 | /subjects/del-lookup-recover     |
      | status_code          | 200                              |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           | *                                |
    And the audit log should contain an event:
      | event_type           | schema_lookup                              |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | del-lookup-recover                         |
      | schema_id            | *                                          |
      | version              | *                                          |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           |                                            |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/del-lookup-recover               |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  Scenario: Lookup does not create a new version
    Given the global compatibility level is "NONE"
    And subject "no-side-effect" has schema:
      """
      {"type":"record","name":"Sensor","fields":[{"name":"sensor_id","type":"string"},{"name":"reading","type":"float"}]}
      """
    When I lookup schema in subject "no-side-effect":
      """
      {"type":"record","name":"Sensor","fields":[{"name":"sensor_id","type":"string"},{"name":"reading","type":"float"}]}
      """
    Then the response status should be 200
    And the response field "version" should be 1
    When I list versions of subject "no-side-effect"
    Then the response status should be 200
    And the response should be an array of length 1
    And the audit log should contain an event:
      | event_type           | schema_lookup                    |
      | outcome              | success                          |
      | actor_id             |                                  |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | no-side-effect                   |
      | schema_id            | *                                |
      | version              | *                                |
      | schema_type          | AVRO                             |
      | before_hash          |                                  |
      | after_hash           |                                  |
      | context              | .                                |
      | transport_security   | tls                              |
      | source_ip            | *                                |
      | user_agent           | *                                |
      | method               | POST                             |
      | path                 | /subjects/no-side-effect         |
      | status_code          | 200                              |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           | *                                |

  Scenario: Lookup with empty schema returns error
    Given subject "empty-lookup" has schema:
      """
      {"type":"record","name":"Ping","fields":[{"name":"ts","type":"long"}]}
      """
    When I lookup schema in subject "empty-lookup":
      """
      """
    Then the response status should be 404
    And the response should have error code 40403
