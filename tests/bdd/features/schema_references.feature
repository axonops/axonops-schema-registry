@functional
Feature: Schema References
  As a developer, I want to register schemas that reference other schemas across subjects

  Scenario: Avro schema referencing another Avro schema
    Given the global compatibility level is "NONE"
    And subject "customer-value" has schema:
      """
      {"type":"record","name":"Customer","namespace":"com.example","fields":[
        {"name":"id","type":"string"},
        {"name":"name","type":"string"},
        {"name":"email","type":["null","string"],"default":null}
      ]}
      """
    When I register a schema under subject "order-value" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"customer\",\"type\":\"com.example.Customer\"},{\"name\":\"total\",\"type\":\"double\"}]}",
        "references": [
          {"name": "com.example.Customer", "subject": "customer-value", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Order"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | order-value                              |
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
      | path                 | /subjects/order-value/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: JSON Schema with internal $ref
    Given the global compatibility level is "NONE"
    When I register a "JSON" schema under subject "json-internal-ref":
      """
      {"type":"object","properties":{"billing":{"$ref":"#/$defs/Address"},"shipping":{"$ref":"#/$defs/Address"}},"$defs":{"Address":{"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}},"required":["street"]}}}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Address"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-internal-ref                        |
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
      | path                 | /subjects/json-internal-ref/versions     |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Get subjects by schema ID with referenced schemas
    Given the global compatibility level is "NONE"
    When I register a schema under subject "shared-type":
      """
      {"type":"record","name":"SharedType","fields":[{"name":"value","type":"string"}]}
      """
    And I store the response field "id" as "schema_id"
    And subject "use-shared-a" has schema:
      """
      {"type":"record","name":"UseA","fields":[{"name":"data","type":"string"}]}
      """
    When I get the subjects for the stored schema ID
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | shared-type                              |
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
      | path                 | /subjects/shared-type/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Register schema with non-existent reference
    Given the global compatibility level is "NONE"
    When I register a schema under subject "bad-ref" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadRef\",\"fields\":[{\"name\":\"data\",\"type\":\"com.example.Missing\"}]}",
        "references": [
          {"name": "com.example.Missing", "subject": "missing-subject", "version": 1}
        ]
      }
      """
    Then the response status should be 422
