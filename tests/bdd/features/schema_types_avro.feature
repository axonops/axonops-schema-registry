@functional @avro
Feature: Avro Schema Types
  As a developer, I want to register and retrieve every valid Avro schema shape

  Scenario: Record with all primitive field types
    When I register a schema under subject "avro-primitives":
      """
      {"type":"record","name":"AllPrimitives","fields":[
        {"name":"f_null","type":"null"},
        {"name":"f_boolean","type":"boolean"},
        {"name":"f_int","type":"int"},
        {"name":"f_long","type":"long"},
        {"name":"f_float","type":"float"},
        {"name":"f_double","type":"double"},
        {"name":"f_bytes","type":"bytes"},
        {"name":"f_string","type":"string"}
      ]}
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "AllPrimitives"

  Scenario: Optional/nullable fields (union with null)
    When I register a schema under subject "avro-nullable":
      """
      {"type":"record","name":"NullableFields","fields":[
        {"name":"required_name","type":"string"},
        {"name":"optional_email","type":["null","string"]},
        {"name":"optional_age","type":["null","int"]},
        {"name":"optional_score","type":["null","double"]}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "optional_email"

  Scenario: Nested records (2 levels)
    When I register a schema under subject "avro-nested-2":
      """
      {"type":"record","name":"Order","fields":[
        {"name":"id","type":"string"},
        {"name":"customer","type":{"type":"record","name":"Customer","fields":[
          {"name":"name","type":"string"},
          {"name":"email","type":"string"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Customer"

  Scenario: Deeply nested records (3+ levels)
    When I register a schema under subject "avro-nested-3":
      """
      {"type":"record","name":"L1","fields":[
        {"name":"l2","type":{"type":"record","name":"L2","fields":[
          {"name":"l3","type":{"type":"record","name":"L3","fields":[
            {"name":"l4","type":{"type":"record","name":"L4","fields":[
              {"name":"value","type":"string"}
            ]}}
          ]}}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "L4"

  Scenario: Arrays and maps
    When I register a schema under subject "avro-collections":
      """
      {"type":"record","name":"WithCollections","fields":[
        {"name":"tags","type":{"type":"array","items":"string"}},
        {"name":"metadata","type":{"type":"map","values":"string"}},
        {"name":"scores","type":{"type":"array","items":"int"}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "tags"

  Scenario: Complex nested collections (map of arrays, array of maps)
    When I register a schema under subject "avro-complex-collections":
      """
      {"type":"record","name":"ComplexCollections","fields":[
        {"name":"map_of_arrays","type":{"type":"map","values":{"type":"array","items":"string"}}},
        {"name":"array_of_maps","type":{"type":"array","items":{"type":"map","values":"int"}}},
        {"name":"array_of_records","type":{"type":"array","items":{"type":"record","name":"Item","fields":[{"name":"name","type":"string"}]}}},
        {"name":"map_of_records","type":{"type":"map","values":"Item"}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "ComplexCollections"

  Scenario: Enum type
    When I register a schema under subject "avro-enum":
      """
      {"type":"record","name":"WithEnum","fields":[
        {"name":"status","type":{"type":"enum","name":"Status","symbols":["ACTIVE","INACTIVE","PENDING"]}},
        {"name":"priority","type":{"type":"enum","name":"Priority","symbols":["LOW","MEDIUM","HIGH","CRITICAL"]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "ACTIVE"

  Scenario: Fixed type
    When I register a schema under subject "avro-fixed":
      """
      {"type":"record","name":"WithFixed","fields":[
        {"name":"uuid","type":{"type":"fixed","name":"UUID","size":16}},
        {"name":"checksum","type":{"type":"fixed","name":"MD5","size":16}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "UUID"

  Scenario: Logical types (date, timestamp-millis, decimal)
    When I register a schema under subject "avro-logical":
      """
      {"type":"record","name":"WithLogicalTypes","fields":[
        {"name":"created_date","type":{"type":"int","logicalType":"date"}},
        {"name":"created_at","type":{"type":"long","logicalType":"timestamp-millis"}},
        {"name":"updated_at","type":{"type":"long","logicalType":"timestamp-micros"}},
        {"name":"price","type":{"type":"bytes","logicalType":"decimal","precision":10,"scale":2}},
        {"name":"event_id","type":{"type":"string","logicalType":"uuid"}},
        {"name":"event_time","type":{"type":"int","logicalType":"time-millis"}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "timestamp-millis"

  Scenario: Complex unions
    When I register a schema under subject "avro-unions":
      """
      {"type":"record","name":"WithUnions","fields":[
        {"name":"value","type":["null","string","int","double","boolean"]},
        {"name":"payload","type":["null","string",{"type":"record","name":"Payload","fields":[{"name":"data","type":"bytes"}]}]}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "WithUnions"

  Scenario: Self-referencing/recursive types
    When I register a schema under subject "avro-recursive":
      """
      {"type":"record","name":"TreeNode","fields":[
        {"name":"value","type":"string"},
        {"name":"children","type":{"type":"array","items":"TreeNode"}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "TreeNode"

  Scenario: Records with default values
    When I register a schema under subject "avro-defaults":
      """
      {"type":"record","name":"WithDefaults","fields":[
        {"name":"name","type":"string","default":"unknown"},
        {"name":"count","type":"int","default":0},
        {"name":"active","type":"boolean","default":true},
        {"name":"tags","type":{"type":"array","items":"string"},"default":[]},
        {"name":"email","type":["null","string"],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "WithDefaults"

  Scenario: Namespaced records
    When I register a schema under subject "avro-namespaced":
      """
      {"type":"record","name":"Event","namespace":"com.example.events","fields":[
        {"name":"id","type":"string"},
        {"name":"source","type":{"type":"record","name":"Source","namespace":"com.example.common","fields":[
          {"name":"system","type":"string"},
          {"name":"region","type":"string"}
        ]}},
        {"name":"timestamp","type":"long"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "com.example.events"

  Scenario: Complex real-world PaymentEvent schema
    When I register a schema under subject "avro-payment-event":
      """
      {"type":"record","name":"PaymentEvent","namespace":"com.example.payments","fields":[
        {"name":"event_id","type":{"type":"string","logicalType":"uuid"}},
        {"name":"timestamp","type":{"type":"long","logicalType":"timestamp-millis"}},
        {"name":"event_type","type":{"type":"enum","name":"PaymentEventType","symbols":["INITIATED","AUTHORIZED","CAPTURED","REFUNDED","FAILED"]}},
        {"name":"amount","type":{"type":"record","name":"Money","fields":[
          {"name":"value","type":{"type":"bytes","logicalType":"decimal","precision":10,"scale":2}},
          {"name":"currency","type":"string"}
        ]}},
        {"name":"customer","type":{"type":"record","name":"Customer","fields":[
          {"name":"id","type":"string"},
          {"name":"name","type":"string"},
          {"name":"email","type":["null","string"],"default":null}
        ]}},
        {"name":"items","type":{"type":"array","items":{"type":"record","name":"LineItem","fields":[
          {"name":"product_id","type":"string"},
          {"name":"name","type":"string"},
          {"name":"quantity","type":"int"},
          {"name":"unit_price","type":{"type":"bytes","logicalType":"decimal","precision":10,"scale":2}}
        ]}}},
        {"name":"metadata","type":{"type":"map","values":"string"}},
        {"name":"payment_method","type":["null",
          {"type":"record","name":"CardPayment","fields":[
            {"name":"last_four","type":"string"},
            {"name":"brand","type":"string"}
          ]},
          {"type":"record","name":"BankTransfer","fields":[
            {"name":"bank_name","type":"string"},
            {"name":"account_last_four","type":"string"}
          ]}
        ],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "PaymentEvent"
    And the response should contain "PaymentEventType"
    When I get version 1 of subject "avro-payment-event"
    Then the response status should be 200
    And the response field "version" should be 1

  Scenario: Retrieve schema round-trip by subject/version
    Given subject "avro-roundtrip" has schema:
      """
      {"type":"record","name":"RoundTrip","fields":[{"name":"id","type":"string"},{"name":"value","type":"int"}]}
      """
    When I get version 1 of subject "avro-roundtrip"
    Then the response status should be 200
    And the response field "subject" should be "avro-roundtrip"
    And the response field "version" should be 1
    And the response should contain "RoundTrip"
    When I get the latest version of subject "avro-roundtrip"
    Then the response status should be 200
    And the response field "version" should be 1
