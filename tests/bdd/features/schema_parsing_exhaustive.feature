@functional
Feature: Schema Parsing & Validation — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive schema parsing and validation tests covering Avro, JSON Schema,
  and Protobuf edge cases from the Confluent Schema Registry v8.1.1 test suite.

  # ==========================================================================
  # AVRO PARSING EDGE CASES (Section 32)
  # ==========================================================================

  Scenario: Parse valid Avro record schema
    When I register a schema under subject "parse-avro-valid":
      """
      {"type":"record","name":"Valid","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"int"}]}
      """
    Then the response status should be 200

  Scenario: Invalid Avro field type returns INVALID_SCHEMA
    When I register a schema under subject "parse-avro-badtype":
      """
      {"type":"record","name":"BadType","fields":[{"name":"f1","type":"str"}]}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Invalid Avro JSON returns INVALID_SCHEMA
    When I register a schema under subject "parse-avro-badjson":
      """
      {not valid json at all}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Avro schema with union default null
    When I register a schema under subject "parse-avro-union-null":
      """
      {"type":"record","name":"UnionNull","fields":[{"name":"f1","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: Avro schema with all primitive types
    When I register a schema under subject "parse-avro-prims":
      """
      {"type":"record","name":"AllPrims","fields":[{"name":"a","type":"null"},{"name":"b","type":"boolean"},{"name":"c","type":"int"},{"name":"d","type":"long"},{"name":"e","type":"float"},{"name":"f","type":"double"},{"name":"g","type":"bytes"},{"name":"h","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Avro schema with logical types
    When I register a schema under subject "parse-avro-logical":
      """
      {"type":"record","name":"Logical","fields":[{"name":"d","type":{"type":"int","logicalType":"date"}},{"name":"ts","type":{"type":"long","logicalType":"timestamp-millis"}},{"name":"uuid","type":{"type":"string","logicalType":"uuid"}}]}
      """
    Then the response status should be 200

  Scenario: Avro schema with enum type
    When I register a schema under subject "parse-avro-enum":
      """
      {"type":"record","name":"WithEnum","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["ACTIVE","INACTIVE","DELETED"]}}]}
      """
    Then the response status should be 200

  Scenario: Avro schema with fixed type
    When I register a schema under subject "parse-avro-fixed":
      """
      {"type":"record","name":"WithFixed","fields":[{"name":"hash","type":{"type":"fixed","name":"MD5","size":16}}]}
      """
    Then the response status should be 200

  Scenario: Avro schema with map and array
    When I register a schema under subject "parse-avro-collections":
      """
      {"type":"record","name":"Collections","fields":[{"name":"tags","type":{"type":"map","values":"string"}},{"name":"scores","type":{"type":"array","items":"int"}}]}
      """
    Then the response status should be 200

  Scenario: Avro schema with nested records
    When I register a schema under subject "parse-avro-nested":
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"value","type":"string"}]}}]}
      """
    Then the response status should be 200

  Scenario: Avro schema with recursive reference
    When I register a schema under subject "parse-avro-recursive":
      """
      {"type":"record","name":"Node","fields":[{"name":"value","type":"string"},{"name":"next","type":["null","Node"],"default":null}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # AVRO COMPATIBILITY — FIELD NAME ALIAS
  # ==========================================================================

  Scenario: Backward compatible — field rename with alias
    Given the global compatibility level is "NONE"
    And subject "parse-avro-alias" has compatibility level "BACKWARD"
    And subject "parse-avro-alias" has schema:
      """
      {"type":"record","name":"AliasTest","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "parse-avro-alias":
      """
      {"type":"record","name":"AliasTest","fields":[{"name":"f1_renamed","type":"string","aliases":["f1"]}]}
      """
    Then the response status should be 200

  Scenario: Backward compatible — evolving field type to union
    Given the global compatibility level is "NONE"
    And subject "parse-avro-to-union" has compatibility level "BACKWARD"
    And subject "parse-avro-to-union" has schema:
      """
      {"type":"record","name":"ToUnion","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "parse-avro-to-union":
      """
      {"type":"record","name":"ToUnion","fields":[{"name":"f1","type":["null","string"]}]}
      """
    Then the response status should be 200

  Scenario: Backward incompatible — removing type from union
    Given the global compatibility level is "NONE"
    And subject "parse-avro-narrow-union" has compatibility level "BACKWARD"
    And subject "parse-avro-narrow-union" has schema:
      """
      {"type":"record","name":"NarrowUnion","fields":[{"name":"f1","type":["null","string"]}]}
      """
    When I register a schema under subject "parse-avro-narrow-union":
      """
      {"type":"record","name":"NarrowUnion","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: Backward compatible — adding type to union
    Given the global compatibility level is "NONE"
    And subject "parse-avro-widen-union" has compatibility level "BACKWARD"
    And subject "parse-avro-widen-union" has schema:
      """
      {"type":"record","name":"WidenUnion","fields":[{"name":"f1","type":["null","string"]}]}
      """
    When I register a schema under subject "parse-avro-widen-union":
      """
      {"type":"record","name":"WidenUnion","fields":[{"name":"f1","type":["null","string","int"]}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # JSON SCHEMA PARSING (Section 33)
  # ==========================================================================

  Scenario: Parse valid JSON Schema
    When I register a "JSON" schema under subject "parse-json-valid":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200

  Scenario: Invalid JSON Schema returns INVALID_SCHEMA
    When I POST "/subjects/parse-json-invalid/versions" with body:
      """
      {"schema": "{invalid json{", "schemaType": "JSON"}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: JSON Schema with $defs references
    When I register a "JSON" schema under subject "parse-json-defs":
      """
      {"type":"object","$defs":{"Address":{"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}}}},"properties":{"home":{"$ref":"#/$defs/Address"},"work":{"$ref":"#/$defs/Address"}}}
      """
    Then the response status should be 200

  Scenario: JSON Schema with recursive $ref
    When I register a "JSON" schema under subject "parse-json-recursive":
      """
      {"type":"object","properties":{"name":{"type":"string"},"children":{"type":"array","items":{"$ref":"#"}}},"required":["name"]}
      """
    Then the response status should be 200

  Scenario: JSON Schema with enum
    When I register a "JSON" schema under subject "parse-json-enum":
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive","deleted"]},"priority":{"type":"integer","enum":[1,2,3,4,5]}}}
      """
    Then the response status should be 200

  Scenario: JSON Schema with oneOf/anyOf/allOf
    When I register a "JSON" schema under subject "parse-json-composition":
      """
      {"type":"object","properties":{"value":{"oneOf":[{"type":"string"},{"type":"integer"}]},"data":{"anyOf":[{"type":"object","properties":{"a":{"type":"string"}}},{"type":"object","properties":{"b":{"type":"integer"}}}]}}}
      """
    Then the response status should be 200

  Scenario: JSON Schema with string constraints
    When I register a "JSON" schema under subject "parse-json-string-constraints":
      """
      {"type":"object","properties":{"email":{"type":"string","format":"email","minLength":5,"maxLength":255},"code":{"type":"string","pattern":"^[A-Z]{3}$"}}}
      """
    Then the response status should be 200

  Scenario: JSON Schema with numeric constraints
    When I register a "JSON" schema under subject "parse-json-numeric-constraints":
      """
      {"type":"object","properties":{"age":{"type":"integer","minimum":0,"maximum":150},"score":{"type":"number","exclusiveMinimum":0,"exclusiveMaximum":100,"multipleOf":0.5}}}
      """
    Then the response status should be 200

  Scenario: JSON Schema with array constraints
    When I register a "JSON" schema under subject "parse-json-array-constraints":
      """
      {"type":"object","properties":{"tags":{"type":"array","items":{"type":"string"},"minItems":1,"maxItems":10,"uniqueItems":true}}}
      """
    Then the response status should be 200

  Scenario: JSON Schema with additionalProperties false
    When I register a "JSON" schema under subject "parse-json-closed":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 200

  Scenario: JSON Schema with nested objects
    When I register a "JSON" schema under subject "parse-json-nested":
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"},"geo":{"type":"object","properties":{"lat":{"type":"number"},"lng":{"type":"number"}},"required":["lat","lng"]}},"required":["street"]}}}
      """
    Then the response status should be 200

  Scenario: JSON Schema dedup — same schema gets same ID
    When I register a "JSON" schema under subject "parse-json-dedup1":
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "json_dedup_id"
    When I register a "JSON" schema under subject "parse-json-dedup2":
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "json_dedup_id"

  # ==========================================================================
  # PROTOBUF PARSING (Section 34)
  # ==========================================================================

  Scenario: Parse valid Protobuf schema
    When I register a "PROTOBUF" schema under subject "parse-proto-valid":
      """
      syntax = "proto3";
      message Valid {
        string name = 1;
        int32 age = 2;
      }
      """
    Then the response status should be 200

  Scenario: Invalid Protobuf schema returns INVALID_SCHEMA
    When I POST "/subjects/parse-proto-invalid/versions" with body:
      """
      {"schema": "this is not valid protobuf at all", "schemaType": "PROTOBUF"}
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Protobuf with nested messages
    When I register a "PROTOBUF" schema under subject "parse-proto-nested":
      """
      syntax = "proto3";
      message Outer {
        string id = 1;
        message Inner {
          string value = 1;
        }
        Inner data = 2;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf with enum
    When I register a "PROTOBUF" schema under subject "parse-proto-enum":
      """
      syntax = "proto3";
      message WithEnum {
        enum Status {
          UNKNOWN = 0;
          ACTIVE = 1;
          INACTIVE = 2;
        }
        Status status = 1;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf with oneof
    When I register a "PROTOBUF" schema under subject "parse-proto-oneof":
      """
      syntax = "proto3";
      message WithOneof {
        string id = 1;
        oneof value {
          string str_val = 2;
          int32 int_val = 3;
          bool bool_val = 4;
        }
      }
      """
    Then the response status should be 200

  Scenario: Protobuf with map
    When I register a "PROTOBUF" schema under subject "parse-proto-map":
      """
      syntax = "proto3";
      message WithMap {
        map<string, int32> counts = 1;
        map<string, string> labels = 2;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf with repeated field
    When I register a "PROTOBUF" schema under subject "parse-proto-repeated":
      """
      syntax = "proto3";
      message WithRepeated {
        repeated string tags = 1;
        repeated int32 scores = 2;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf with well-known type imports
    When I register a "PROTOBUF" schema under subject "parse-proto-wkt":
      """
      syntax = "proto3";
      import "google/protobuf/timestamp.proto";
      import "google/protobuf/wrappers.proto";
      message WithWKT {
        google.protobuf.Timestamp created = 1;
        google.protobuf.StringValue optional_name = 2;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf with package declaration
    When I register a "PROTOBUF" schema under subject "parse-proto-package":
      """
      syntax = "proto3";
      package com.example.events;
      message Event {
        string id = 1;
        string type = 2;
        bytes payload = 3;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf with optional field (proto3)
    When I register a "PROTOBUF" schema under subject "parse-proto-optional":
      """
      syntax = "proto3";
      message WithOptional {
        string required_name = 1;
        optional string optional_desc = 2;
        optional int32 optional_count = 3;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf dedup — same schema gets same ID
    When I register a "PROTOBUF" schema under subject "parse-proto-dedup1":
      """
      syntax = "proto3";
      message DedupProto {
        string id = 1;
        int32 value = 2;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "proto_dedup_id"
    When I register a "PROTOBUF" schema under subject "parse-proto-dedup2":
      """
      syntax = "proto3";
      message DedupProto {
        string id = 1;
        int32 value = 2;
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "proto_dedup_id"

  Scenario: Same message name in different packages are different schemas
    When I register a "PROTOBUF" schema under subject "parse-proto-pkg1":
      """
      syntax = "proto3";
      package com.example.v1;
      message Event {
        string id = 1;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "pkg1_id"
    When I register a "PROTOBUF" schema under subject "parse-proto-pkg2":
      """
      syntax = "proto3";
      package com.example.v2;
      message Event {
        string id = 1;
        string type = 2;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "pkg2_id"
