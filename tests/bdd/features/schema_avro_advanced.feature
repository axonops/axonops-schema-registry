@functional @avro
Feature: Advanced Avro Schema Parsing
  As a developer, I want to register and retrieve advanced Avro schemas
  covering unions, logical types, complex nesting, real-world patterns,
  and edge cases to ensure robust Avro support

  # ---------- 1. Union containing record types ----------
  Scenario: Union containing named record types
    When I register a schema under subject "avro-adv-1":
      """
      {"type":"record","name":"Event","fields":[
        {"name":"payload","type":[
          {"type":"record","name":"ClickEvent","fields":[
            {"name":"url","type":"string"},
            {"name":"button","type":"string"}
          ]},
          {"type":"record","name":"ViewEvent","fields":[
            {"name":"page","type":"string"},
            {"name":"duration_ms","type":"long"}
          ]}
        ]}
      ]}
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"

  # ---------- 2. Union with null and complex type (nullable record) ----------
  Scenario: Nullable record via union with null
    When I register a schema under subject "avro-adv-2":
      """
      {"type":"record","name":"Order","fields":[
        {"name":"id","type":"string"},
        {"name":"shipping_address","type":["null",{"type":"record","name":"Address","fields":[
          {"name":"street","type":"string"},
          {"name":"city","type":"string"},
          {"name":"zip","type":"string"},
          {"name":"country","type":"string"}
        ]}],"default":null}
      ]}
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"

  # ---------- 3. Deeply nested records (4 levels deep) ----------
  Scenario: Four levels of nested records
    When I register a schema under subject "avro-adv-3":
      """
      {"type":"record","name":"Level1","fields":[
        {"name":"name","type":"string"},
        {"name":"child","type":{"type":"record","name":"Level2","fields":[
          {"name":"name","type":"string"},
          {"name":"child","type":{"type":"record","name":"Level3","fields":[
            {"name":"name","type":"string"},
            {"name":"child","type":{"type":"record","name":"Level4","fields":[
              {"name":"name","type":"string"},
              {"name":"value","type":"int"}
            ]}}
          ]}}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Level4"

  # ---------- 4. Record with all logical types ----------
  Scenario: Record with all Avro logical types
    When I register a schema under subject "avro-adv-4":
      """
      {"type":"record","name":"AllLogicalTypes","namespace":"com.example","fields":[
        {"name":"birth_date","type":{"type":"int","logicalType":"date"}},
        {"name":"start_time_ms","type":{"type":"int","logicalType":"time-millis"}},
        {"name":"start_time_us","type":{"type":"long","logicalType":"time-micros"}},
        {"name":"created_at_ms","type":{"type":"long","logicalType":"timestamp-millis"}},
        {"name":"created_at_us","type":{"type":"long","logicalType":"timestamp-micros"}},
        {"name":"price","type":{"type":"bytes","logicalType":"decimal","precision":20,"scale":5}},
        {"name":"event_id","type":{"type":"string","logicalType":"uuid"}},
        {"name":"fixed_decimal","type":{"type":"bytes","logicalType":"decimal","precision":38,"scale":10}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "AllLogicalTypes"

  # ---------- 5. Fixed type inside record field ----------
  Scenario: Fixed type as a record field
    When I register a schema under subject "avro-adv-5":
      """
      {"type":"record","name":"WithFixedField","fields":[
        {"name":"id","type":"string"},
        {"name":"checksum","type":{"type":"fixed","name":"MD5Hash","size":16}},
        {"name":"trace_id","type":{"type":"fixed","name":"TraceID","size":32}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "MD5Hash"

  # ---------- 6. Fixed type inside array items ----------
  Scenario: Array of fixed type values
    When I register a schema under subject "avro-adv-6":
      """
      {"type":"record","name":"FixedArray","fields":[
        {"name":"hashes","type":{"type":"array","items":{"type":"fixed","name":"SHA256","size":32}}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "SHA256"

  # ---------- 7. Enum with doc string ----------
  Scenario: Enum type with doc annotation
    When I register a schema under subject "avro-adv-7":
      """
      {"type":"record","name":"TaskRecord","fields":[
        {"name":"status","type":{
          "type":"enum",
          "name":"TaskStatus",
          "doc":"Represents the lifecycle state of a task",
          "symbols":["PENDING","RUNNING","COMPLETED","FAILED","CANCELLED"]
        }}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "TaskStatus"

  # ---------- 8. Enum inside union ----------
  Scenario: Nullable enum via union
    When I register a schema under subject "avro-adv-8":
      """
      {"type":"record","name":"EmployeeRecord","fields":[
        {"name":"name","type":"string"},
        {"name":"department","type":["null",{
          "type":"enum",
          "name":"Department",
          "symbols":["ENGINEERING","MARKETING","SALES","HR","FINANCE","OPERATIONS"]
        }],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Department"

  # ---------- 9. Map with record values containing unions ----------
  Scenario: Map of records with union fields
    When I register a schema under subject "avro-adv-9":
      """
      {"type":"record","name":"Registry","fields":[
        {"name":"services","type":{"type":"map","values":{
          "type":"record","name":"ServiceInfo","fields":[
            {"name":"host","type":"string"},
            {"name":"port","type":"int"},
            {"name":"metadata","type":["null","string"],"default":null},
            {"name":"health","type":["null",{
              "type":"enum","name":"HealthStatus","symbols":["HEALTHY","DEGRADED","UNHEALTHY"]
            }],"default":null}
          ]
        }}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "ServiceInfo"

  # ---------- 10. Array of maps of records ----------
  Scenario: Array of maps containing record values
    When I register a schema under subject "avro-adv-10":
      """
      {"type":"record","name":"Dashboard","fields":[
        {"name":"panels","type":{"type":"array","items":{"type":"map","values":{
          "type":"record","name":"Widget","fields":[
            {"name":"title","type":"string"},
            {"name":"width","type":"int"},
            {"name":"height","type":"int"}
          ]
        }}}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Widget"

  # ---------- 11. Self-referencing record (linked list) ----------
  Scenario: Self-referencing record as linked list
    When I register a schema under subject "avro-adv-11":
      """
      {"type":"record","name":"Node","fields":[
        {"name":"value","type":"int"},
        {"name":"next","type":["null","Node"],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Node"

  # ---------- 12. Record with bytes field and default ----------
  Scenario: Record with bytes field and empty default
    When I register a schema under subject "avro-adv-12":
      """
      {"type":"record","name":"BinaryData","fields":[
        {"name":"header","type":"bytes","default":""},
        {"name":"payload","type":"bytes"},
        {"name":"optional_trailer","type":["null","bytes"],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "BinaryData"

  # ---------- 13. Record with 30+ fields (stress test) ----------
  Scenario: Record with more than 30 fields
    When I register a schema under subject "avro-adv-13":
      """
      {"type":"record","name":"WideRecord","namespace":"com.example.stress","fields":[
        {"name":"f01","type":"string"},
        {"name":"f02","type":"int"},
        {"name":"f03","type":"long"},
        {"name":"f04","type":"float"},
        {"name":"f05","type":"double"},
        {"name":"f06","type":"boolean"},
        {"name":"f07","type":"bytes"},
        {"name":"f08","type":"string"},
        {"name":"f09","type":"int"},
        {"name":"f10","type":"long"},
        {"name":"f11","type":"float"},
        {"name":"f12","type":"double"},
        {"name":"f13","type":"boolean"},
        {"name":"f14","type":"string"},
        {"name":"f15","type":"int"},
        {"name":"f16","type":"long"},
        {"name":"f17","type":"string"},
        {"name":"f18","type":"int"},
        {"name":"f19","type":"long"},
        {"name":"f20","type":"float"},
        {"name":"f21","type":"double"},
        {"name":"f22","type":"boolean"},
        {"name":"f23","type":"string"},
        {"name":"f24","type":"int"},
        {"name":"f25","type":"long"},
        {"name":"f26","type":"float"},
        {"name":"f27","type":"double"},
        {"name":"f28","type":"boolean"},
        {"name":"f29","type":"string"},
        {"name":"f30","type":"int"},
        {"name":"f31","type":"long"},
        {"name":"f32","type":"string"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "WideRecord"

  # ---------- 14. Namespace inheritance in nested records ----------
  Scenario: Namespace inheritance with explicit and inherited namespaces
    When I register a schema under subject "avro-adv-14":
      """
      {"type":"record","name":"Outer","namespace":"com.example.outer","fields":[
        {"name":"inherited_ns","type":{"type":"record","name":"InheritedChild","fields":[
          {"name":"value","type":"string"}
        ]}},
        {"name":"explicit_ns","type":{"type":"record","name":"ExplicitChild","namespace":"com.example.inner","fields":[
          {"name":"value","type":"string"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "com.example.outer"
    And the response should contain "com.example.inner"

  # ---------- 15. Multiple named types in same schema ----------
  Scenario: Schema with multiple named record types
    When I register a schema under subject "avro-adv-15":
      """
      {"type":"record","name":"Invoice","fields":[
        {"name":"id","type":"string"},
        {"name":"seller","type":{"type":"record","name":"Party","fields":[
          {"name":"name","type":"string"},
          {"name":"tax_id","type":"string"}
        ]}},
        {"name":"buyer","type":"Party"},
        {"name":"line_items","type":{"type":"array","items":{"type":"record","name":"InvoiceLine","fields":[
          {"name":"description","type":"string"},
          {"name":"quantity","type":"int"},
          {"name":"unit_price","type":"double"}
        ]}}},
        {"name":"status","type":{"type":"enum","name":"InvoiceStatus","symbols":["DRAFT","SENT","PAID","OVERDUE","CANCELLED"]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Invoice"
    And the response should contain "InvoiceLine"

  # ---------- 16. Record with all 8 primitive field types ----------
  Scenario: Record covering every Avro primitive type
    When I register a schema under subject "avro-adv-16":
      """
      {"type":"record","name":"AllPrimitivesAdvanced","fields":[
        {"name":"null_field","type":"null"},
        {"name":"bool_field","type":"boolean"},
        {"name":"int_field","type":"int"},
        {"name":"long_field","type":"long"},
        {"name":"float_field","type":"float"},
        {"name":"double_field","type":"double"},
        {"name":"bytes_field","type":"bytes"},
        {"name":"string_field","type":"string"}
      ]}
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "AllPrimitivesAdvanced"

  # ---------- 17. Doc annotations at record and field level ----------
  Scenario: Schema with doc annotations on record and fields
    When I register a schema under subject "avro-adv-17":
      """
      {"type":"record","name":"DocumentedRecord","doc":"A well-documented record for testing","fields":[
        {"name":"id","type":"string","doc":"Unique identifier for this record"},
        {"name":"created_at","type":"long","doc":"Unix timestamp of creation time"},
        {"name":"description","type":["null","string"],"default":null,"doc":"Optional human-readable description"},
        {"name":"tags","type":{"type":"array","items":"string"},"doc":"Searchable tags"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "DocumentedRecord"

  # ---------- 18. Field ordering (order attribute) ----------
  Scenario: Schema with field ordering annotations
    When I register a schema under subject "avro-adv-18":
      """
      {"type":"record","name":"SortableRecord","fields":[
        {"name":"primary_key","type":"string","order":"ascending"},
        {"name":"timestamp","type":"long","order":"descending"},
        {"name":"payload","type":"bytes","order":"ignore"},
        {"name":"secondary_key","type":"int","order":"ascending"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "SortableRecord"

  # ---------- 19. Aliases on records ----------
  Scenario: Schema with aliases on the record
    When I register a schema under subject "avro-adv-19":
      """
      {"type":"record","name":"UserProfile","aliases":["UserAccount","PersonProfile"],"fields":[
        {"name":"username","type":"string"},
        {"name":"email","type":"string"},
        {"name":"age","type":"int"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "UserProfile"

  # ---------- 20. Aliases on fields ----------
  Scenario: Schema with aliases on fields
    When I register a schema under subject "avro-adv-20":
      """
      {"type":"record","name":"LegacyCompatible","fields":[
        {"name":"user_name","type":"string","aliases":["username","login"]},
        {"name":"email_address","type":"string","aliases":["email","mail"]},
        {"name":"phone_number","type":["null","string"],"default":null,"aliases":["phone","tel"]}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "LegacyCompatible"

  # ---------- 21. Decimal logical type with specific precision and scale ----------
  Scenario: Decimal logical type with high precision
    When I register a schema under subject "avro-adv-21":
      """
      {"type":"record","name":"FinancialAmount","fields":[
        {"name":"amount_bytes","type":{"type":"bytes","logicalType":"decimal","precision":38,"scale":18}},
        {"name":"amount_fixed","type":{"type":"bytes","logicalType":"decimal","precision":30,"scale":8}},
        {"name":"small_amount","type":{"type":"bytes","logicalType":"decimal","precision":5,"scale":2}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "FinancialAmount"

  # ---------- 22. Complex real-world: Kafka Connect source record ----------
  Scenario: Kafka Connect source record schema
    When I register a schema under subject "avro-adv-22":
      """
      {"type":"record","name":"ConnectRecord","namespace":"io.confluent.connect","fields":[
        {"name":"source","type":{"type":"record","name":"Source","fields":[
          {"name":"connector","type":"string"},
          {"name":"version","type":"string"},
          {"name":"ts_ms","type":"long"},
          {"name":"snapshot","type":["null","boolean"],"default":null},
          {"name":"db","type":"string"},
          {"name":"schema","type":["null","string"],"default":null},
          {"name":"table","type":"string"}
        ]}},
        {"name":"op","type":{"type":"enum","name":"Operation","symbols":["c","u","d","r"]}},
        {"name":"ts_ms","type":{"type":"long","logicalType":"timestamp-millis"}},
        {"name":"before","type":["null",{"type":"record","name":"Row","fields":[
          {"name":"id","type":"long"},
          {"name":"name","type":["null","string"],"default":null},
          {"name":"value","type":["null","double"],"default":null}
        ]}],"default":null},
        {"name":"after","type":["null","Row"],"default":null},
        {"name":"transaction","type":["null",{"type":"record","name":"Transaction","fields":[
          {"name":"id","type":"string"},
          {"name":"total_order","type":"long"},
          {"name":"data_collection_order","type":"long"}
        ]}],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "ConnectRecord"
    And the response should contain "Operation"

  # ---------- 23. Complex real-world: CDC change event ----------
  Scenario: CDC change event schema
    When I register a schema under subject "avro-adv-23":
      """
      {"type":"record","name":"ChangeEvent","namespace":"com.example.cdc","fields":[
        {"name":"event_id","type":{"type":"string","logicalType":"uuid"}},
        {"name":"event_type","type":{"type":"enum","name":"ChangeType","symbols":["INSERT","UPDATE","DELETE","TRUNCATE"]}},
        {"name":"table_name","type":"string"},
        {"name":"schema_name","type":"string"},
        {"name":"timestamp","type":{"type":"long","logicalType":"timestamp-millis"}},
        {"name":"primary_key","type":{"type":"map","values":"string"}},
        {"name":"old_values","type":["null",{"type":"map","values":["null","string","int","long","double","boolean"]}],"default":null},
        {"name":"new_values","type":["null",{"type":"map","values":["null","string","int","long","double","boolean"]}],"default":null},
        {"name":"headers","type":{"type":"map","values":"string"}},
        {"name":"sequence_number","type":"long"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "ChangeEvent"
    And the response should contain "ChangeType"

  # ---------- 24. Complex real-world: Financial transaction ----------
  Scenario: Financial transaction schema with multiple record types
    When I register a schema under subject "avro-adv-24":
      """
      {"type":"record","name":"Transaction","namespace":"com.example.finance","fields":[
        {"name":"txn_id","type":{"type":"string","logicalType":"uuid"}},
        {"name":"timestamp","type":{"type":"long","logicalType":"timestamp-millis"}},
        {"name":"amount","type":{"type":"record","name":"Money","fields":[
          {"name":"value","type":{"type":"bytes","logicalType":"decimal","precision":18,"scale":4}},
          {"name":"currency","type":{"type":"enum","name":"Currency","symbols":["USD","EUR","GBP","JPY","CHF","CAD","AUD","CNY"]}}
        ]}},
        {"name":"debit_account","type":{"type":"record","name":"Account","fields":[
          {"name":"account_id","type":"string"},
          {"name":"account_type","type":{"type":"enum","name":"AccountType","symbols":["CHECKING","SAVINGS","CREDIT","INVESTMENT"]}},
          {"name":"holder_name","type":"string"}
        ]}},
        {"name":"credit_account","type":"Account"},
        {"name":"status","type":{"type":"enum","name":"TxnStatus","symbols":["PENDING","PROCESSING","COMPLETED","FAILED","REVERSED"]}},
        {"name":"fees","type":{"type":"array","items":{"type":"record","name":"Fee","fields":[
          {"name":"type","type":"string"},
          {"name":"amount","type":"Money"}
        ]}}},
        {"name":"metadata","type":{"type":"map","values":"string"}},
        {"name":"memo","type":["null","string"],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Transaction"
    And the response should contain "Currency"
    And the response should contain "AccountType"

  # ---------- 25. Round-trip: register, get by ID, verify schema field ----------
  Scenario: Round-trip registration and retrieval verifies schema field
    When I register a schema under subject "avro-adv-25":
      """
      {"type":"record","name":"RoundTripTest","namespace":"com.example.roundtrip","fields":[
        {"name":"id","type":"string"},
        {"name":"value","type":"double"},
        {"name":"tags","type":{"type":"array","items":"string"}}
      ]}
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "RoundTripTest"
    And the response should contain "com.example.roundtrip"
    When I get version 1 of subject "avro-adv-25"
    Then the response status should be 200
    And the response field "subject" should be "avro-adv-25"
    And the response field "version" should be 1
    And the response should contain "RoundTripTest"

  # ---------- 26. Fingerprint stability: same schema under 2 subjects ----------
  Scenario: Same schema registered under two subjects gets same schema ID
    When I register a schema under subject "avro-adv-26a":
      """
      {"type":"record","name":"SharedSchema","namespace":"com.example.shared","fields":[
        {"name":"key","type":"string"},
        {"name":"value","type":"long"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I register a schema under subject "avro-adv-26b":
      """
      {"type":"record","name":"SharedSchema","namespace":"com.example.shared","fields":[
        {"name":"key","type":"string"},
        {"name":"value","type":"long"}
      ]}
      """
    Then the response status should be 200
    And the response should have field "id"
    When I get the subjects for the stored schema ID
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "avro-adv-26a"
    And the response array should contain "avro-adv-26b"

  # ---------- 27. Field defaults of every primitive type ----------
  Scenario: Schema with default values for every primitive type
    When I register a schema under subject "avro-adv-27":
      """
      {"type":"record","name":"AllDefaults","fields":[
        {"name":"null_field","type":"null","default":null},
        {"name":"bool_field","type":"boolean","default":false},
        {"name":"int_field","type":"int","default":42},
        {"name":"long_field","type":"long","default":9876543210},
        {"name":"float_field","type":"float","default":3.14},
        {"name":"double_field","type":"double","default":2.718281828},
        {"name":"string_field","type":"string","default":"hello"},
        {"name":"bytes_field","type":"bytes","default":""}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "AllDefaults"

  # ---------- 28. Enum with many symbols (10+) ----------
  Scenario: Enum with more than 10 symbols
    When I register a schema under subject "avro-adv-28":
      """
      {"type":"record","name":"HttpRequest","fields":[
        {"name":"method","type":{"type":"enum","name":"HttpMethod","symbols":[
          "GET","POST","PUT","DELETE","PATCH","HEAD","OPTIONS","TRACE","CONNECT","PURGE","LINK","UNLINK"
        ]}},
        {"name":"url","type":"string"},
        {"name":"status_code","type":"int"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "HttpMethod"
    And the response should contain "PURGE"

  # ---------- 29. Map of arrays of strings ----------
  Scenario: Map whose values are arrays of strings
    When I register a schema under subject "avro-adv-29":
      """
      {"type":"record","name":"TagIndex","fields":[
        {"name":"index","type":{"type":"map","values":{"type":"array","items":"string"}}},
        {"name":"updated_at","type":"long"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "TagIndex"

  # ---------- 30. Array of arrays (nested collection) ----------
  Scenario: Array of arrays for matrix-like data
    When I register a schema under subject "avro-adv-30":
      """
      {"type":"record","name":"Matrix","fields":[
        {"name":"rows","type":{"type":"array","items":{"type":"array","items":"double"}}},
        {"name":"dimensions","type":{"type":"record","name":"Dimensions","fields":[
          {"name":"num_rows","type":"int"},
          {"name":"num_cols","type":"int"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Matrix"

  # ---------- 31. Record with nullable array field ----------
  Scenario: Nullable array field in a record
    When I register a schema under subject "avro-adv-31":
      """
      {"type":"record","name":"UserWithTags","fields":[
        {"name":"name","type":"string"},
        {"name":"tags","type":["null",{"type":"array","items":"string"}],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "UserWithTags"

  # ---------- 32. Record with nullable map field ----------
  Scenario: Nullable map field in a record
    When I register a schema under subject "avro-adv-32":
      """
      {"type":"record","name":"ConfigEntry","fields":[
        {"name":"key","type":"string"},
        {"name":"properties","type":["null",{"type":"map","values":"string"}],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "ConfigEntry"

  # ---------- 33. Record with nullable enum field ----------
  Scenario: Nullable enum field in a record
    When I register a schema under subject "avro-adv-33":
      """
      {"type":"record","name":"Ticket","fields":[
        {"name":"id","type":"string"},
        {"name":"title","type":"string"},
        {"name":"priority","type":["null",{
          "type":"enum","name":"TicketPriority","symbols":["P1","P2","P3","P4","P5"]
        }],"default":null},
        {"name":"severity","type":["null",{
          "type":"enum","name":"Severity","symbols":["CRITICAL","HIGH","MEDIUM","LOW","INFORMATIONAL"]
        }],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "TicketPriority"
    And the response should contain "Severity"

  # ---------- 34. Schema with very long record name ----------
  Scenario: Schema with a very long record name
    When I register a schema under subject "avro-adv-34":
      """
      {"type":"record","name":"ThisIsAnExtremelyLongRecordNameThatTestsTheLimitsOfNameHandlingInTheSchemaRegistryAndShouldStillBeAcceptedWithoutErrors","fields":[
        {"name":"id","type":"string"},
        {"name":"value","type":"int"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "ThisIsAnExtremelyLongRecordName"

  # ---------- 35. Schema with nested namespace ----------
  Scenario: Schema with deeply nested namespace hierarchy
    When I register a schema under subject "avro-adv-35":
      """
      {"type":"record","name":"DeepEvent","namespace":"com.example.division.team.project.module","fields":[
        {"name":"id","type":"string"},
        {"name":"detail","type":{"type":"record","name":"DetailRecord","namespace":"com.example.division.team.project.module.detail","fields":[
          {"name":"code","type":"int"},
          {"name":"message","type":"string"},
          {"name":"sub_detail","type":{"type":"record","name":"SubDetail","namespace":"com.example.division.team.project.module.detail.sub","fields":[
            {"name":"trace","type":"string"}
          ]}}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "com.example.division.team.project.module"

  # ---------- 36. Self-referencing binary tree ----------
  Scenario: Self-referencing record as binary tree
    When I register a schema under subject "avro-adv-36":
      """
      {"type":"record","name":"BinaryTreeNode","fields":[
        {"name":"value","type":"string"},
        {"name":"left","type":["null","BinaryTreeNode"],"default":null},
        {"name":"right","type":["null","BinaryTreeNode"],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "BinaryTreeNode"

  # ---------- 37. Record reusing named types across fields ----------
  Scenario: Reusing a named enum type across multiple fields
    When I register a schema under subject "avro-adv-37":
      """
      {"type":"record","name":"FlightBooking","fields":[
        {"name":"booking_id","type":"string"},
        {"name":"departure_airport","type":{"type":"record","name":"Airport","fields":[
          {"name":"code","type":"string"},
          {"name":"name","type":"string"},
          {"name":"country","type":"string"}
        ]}},
        {"name":"arrival_airport","type":"Airport"},
        {"name":"outbound_class","type":{"type":"enum","name":"CabinClass","symbols":["ECONOMY","PREMIUM_ECONOMY","BUSINESS","FIRST"]}},
        {"name":"return_class","type":["null","CabinClass"],"default":null},
        {"name":"passenger_count","type":"int"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "FlightBooking"
    And the response should contain "CabinClass"

  # ---------- 38. Complex union with multiple record and enum types ----------
  Scenario: Union with multiple named record and enum types
    When I register a schema under subject "avro-adv-38":
      """
      {"type":"record","name":"Notification","namespace":"com.example.notifications","fields":[
        {"name":"id","type":"string"},
        {"name":"channel","type":{"type":"enum","name":"Channel","symbols":["EMAIL","SMS","PUSH","IN_APP","WEBHOOK"]}},
        {"name":"content","type":[
          {"type":"record","name":"EmailContent","fields":[
            {"name":"subject","type":"string"},
            {"name":"body_html","type":"string"},
            {"name":"from_address","type":"string"}
          ]},
          {"type":"record","name":"SmsContent","fields":[
            {"name":"message","type":"string"},
            {"name":"from_number","type":"string"}
          ]},
          {"type":"record","name":"PushContent","fields":[
            {"name":"title","type":"string"},
            {"name":"body","type":"string"},
            {"name":"icon_url","type":["null","string"],"default":null}
          ]}
        ]},
        {"name":"recipient","type":"string"},
        {"name":"sent_at","type":["null",{"type":"long","logicalType":"timestamp-millis"}],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Notification"
    And the response should contain "EmailContent"
    And the response should contain "PushContent"

  # ---------- 39. Map with complex nested values ----------
  Scenario: Map with values being arrays of records
    When I register a schema under subject "avro-adv-39":
      """
      {"type":"record","name":"Catalog","fields":[
        {"name":"categories","type":{"type":"map","values":{"type":"array","items":{"type":"record","name":"Product","fields":[
          {"name":"sku","type":"string"},
          {"name":"name","type":"string"},
          {"name":"price","type":"double"},
          {"name":"in_stock","type":"boolean"}
        ]}}}},
        {"name":"last_updated","type":{"type":"long","logicalType":"timestamp-millis"}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "Catalog"
    And the response should contain "Product"

  # ---------- 40. IoT sensor event with complex structure ----------
  Scenario: IoT sensor event with complex nested structure
    When I register a schema under subject "avro-adv-40":
      """
      {"type":"record","name":"SensorReading","namespace":"com.example.iot","fields":[
        {"name":"device_id","type":{"type":"string","logicalType":"uuid"}},
        {"name":"timestamp","type":{"type":"long","logicalType":"timestamp-micros"}},
        {"name":"sensor_type","type":{"type":"enum","name":"SensorType","symbols":[
          "TEMPERATURE","HUMIDITY","PRESSURE","LIGHT","MOTION","CO2","NOISE","VIBRATION"
        ]}},
        {"name":"reading","type":[
          {"type":"record","name":"ScalarReading","fields":[
            {"name":"value","type":"double"},
            {"name":"unit","type":"string"}
          ]},
          {"type":"record","name":"VectorReading","fields":[
            {"name":"x","type":"double"},
            {"name":"y","type":"double"},
            {"name":"z","type":"double"},
            {"name":"unit","type":"string"}
          ]}
        ]},
        {"name":"location","type":["null",{"type":"record","name":"GeoLocation","fields":[
          {"name":"latitude","type":"double"},
          {"name":"longitude","type":"double"},
          {"name":"altitude","type":["null","double"],"default":null}
        ]}],"default":null},
        {"name":"battery_pct","type":["null","float"],"default":null},
        {"name":"firmware_version","type":"string"},
        {"name":"tags","type":{"type":"map","values":"string"}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "SensorReading"
    And the response should contain "SensorType"
    And the response should contain "GeoLocation"
