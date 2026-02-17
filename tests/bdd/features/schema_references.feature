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
