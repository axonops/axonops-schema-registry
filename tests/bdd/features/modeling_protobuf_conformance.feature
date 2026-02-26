@schema-modeling @protobuf @conformance
Feature: Protobuf Conformance-Inspired Parsing
  Protobuf schemas derived from the official protobuf conformance test protos
  exercising parsing, registration, and deduplication of complex .proto definitions.

  # ==========================================================================
  # 1. ALL 15 SCALAR TYPES + NESTED MESSAGE + NESTED ENUM
  # ==========================================================================

  Scenario: All 15 scalar types with nested message and enum
    When I register a "PROTOBUF" schema under subject "proto-conform-scalars":
      """
syntax = "proto3";
package test.scalars;

message AllScalars {
  int32 f1 = 1;
  int64 f2 = 2;
  uint32 f3 = 3;
  uint64 f4 = 4;
  sint32 f5 = 5;
  sint64 f6 = 6;
  fixed32 f7 = 7;
  fixed64 f8 = 8;
  sfixed32 f9 = 9;
  sfixed64 f10 = 10;
  float f11 = 11;
  double f12 = 12;
  bool f13 = 13;
  string f14 = 14;
  bytes f15 = 15;
  NestedMsg nested = 16;
  NestedEnum status = 17;
  message NestedMsg {
    string value = 1;
  }
  enum NestedEnum {
    UNKNOWN = 0;
    ACTIVE = 1;
    INACTIVE = 2;
  }
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 2. RECURSIVE SELF-REFERENCING MESSAGE
  # ==========================================================================

  Scenario: Recursive self-referencing message
    When I register a "PROTOBUF" schema under subject "proto-conform-recursive":
      """
syntax = "proto3";
package test.recursive;

message TreeNode {
  string value = 1;
  repeated TreeNode children = 2;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 3. RECURSIVE MAP FIELD
  # ==========================================================================

  Scenario: Recursive map field
    When I register a "PROTOBUF" schema under subject "proto-conform-recmap":
      """
syntax = "proto3";
package test.recmap;

message RecursiveMap {
  map<string, RecursiveMap> entries = 1;
  string data = 2;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 4. NEGATIVE ENUM VALUE
  # ==========================================================================

  Scenario: Negative enum value registers successfully
    When I register a "PROTOBUF" schema under subject "proto-conform-neg-enum":
      """
syntax = "proto3";
package test.enums;

message Container {
  NegEnum val = 1;
  enum NegEnum {
    ZERO = 0;
    POSITIVE = 1;
    NEGATIVE = -1;
  }
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 5. ALIASED ENUM
  # ==========================================================================

  Scenario: Aliased enum with allow_alias option
    When I register a "PROTOBUF" schema under subject "proto-conform-alias-enum":
      """
syntax = "proto3";
package test.alias;

message Container {
  AliasedEnum val = 1;
  enum AliasedEnum {
    option allow_alias = true;
    DEFAULT = 0;
    FOO = 1;
    BAR = 1;
    baz = 1;
  }
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 6. ALL VALID MAP KEY TYPES
  # ==========================================================================

  Scenario: All valid map key types register successfully
    When I register a "PROTOBUF" schema under subject "proto-conform-map-keys":
      """
syntax = "proto3";
package test.maps;

message AllMapKeys {
  map<int32, string> m1 = 1;
  map<int64, string> m2 = 2;
  map<uint32, string> m3 = 3;
  map<uint64, string> m4 = 4;
  map<sint32, string> m5 = 5;
  map<sint64, string> m6 = 6;
  map<fixed32, string> m7 = 7;
  map<fixed64, string> m8 = 8;
  map<sfixed32, string> m9 = 9;
  map<sfixed64, string> m10 = 10;
  map<bool, string> m11 = 11;
  map<string, string> m12 = 12;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 7. MAP WITH NESTED MESSAGE VALUES
  # ==========================================================================

  Scenario: Map with nested message values
    When I register a "PROTOBUF" schema under subject "proto-conform-map-msg":
      """
syntax = "proto3";
package test.mapval;

message Registry {
  map<string, Entry> entries = 1;
  message Entry {
    string name = 1;
    int32 version = 2;
  }
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 8. MULTIPLE ONEOFS IN SAME MESSAGE
  # ==========================================================================

  Scenario: Multiple oneofs in same message
    When I register a "PROTOBUF" schema under subject "proto-conform-multi-oneof":
      """
syntax = "proto3";
package test.oneofs;

message MultiOneof {
  oneof choice_a {
    string name = 1;
    int32 id = 2;
  }
  oneof choice_b {
    bool flag = 3;
    bytes data = 4;
  }
  string common = 5;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 9. PROTO2 WITH DEFAULT VALUES
  # ==========================================================================

  Scenario: Proto2 with default values
    When I register a "PROTOBUF" schema under subject "proto-conform-proto2-defaults":
      """
syntax = "proto2";
package test.defaults;

message WithDefaults {
  optional int32 count = 1 [default = 42];
  optional string label = 2 [default = "unknown"];
  optional bool active = 3 [default = true];
  optional float ratio = 4 [default = 1.5];
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 10. RESERVED FIELD NUMBERS AND NAMES
  # ==========================================================================

  Scenario: Reserved field numbers and names
    When I register a "PROTOBUF" schema under subject "proto-conform-reserved":
      """
syntax = "proto3";
package test.reserved;

message Config {
  string name = 1;
  reserved 2, 15, 9 to 11;
  reserved "old_field", "deprecated";
  string current = 3;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 11. EMPTY MESSAGE
  # ==========================================================================

  Scenario: Empty message registers successfully
    When I register a "PROTOBUF" schema under subject "proto-conform-empty":
      """
syntax = "proto3";
package test.empty;

message Empty {}
      """
    Then the response status should be 200

  # ==========================================================================
  # 12. PACKED AND UNPACKED REPEATED
  # ==========================================================================

  Scenario: Packed and unpacked repeated fields
    When I register a "PROTOBUF" schema under subject "proto-conform-packed":
      """
syntax = "proto3";
package test.packed;

message Metrics {
  repeated int32 values = 1 [packed = true];
  repeated string labels = 2;
  repeated double samples = 3;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 13. WELL-KNOWN TYPES
  # ==========================================================================

  Scenario: Well-known types with imports
    When I register a "PROTOBUF" schema under subject "proto-conform-wkt":
      """
syntax = "proto3";
package test.wkt;

import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/wrappers.proto";

message EventRecord {
  google.protobuf.Timestamp created_at = 1;
  google.protobuf.Duration ttl = 2;
  google.protobuf.StringValue label = 3;
  google.protobuf.Int32Value priority = 4;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 14. SERVICE DEFINITION
  # ==========================================================================

  Scenario: Service definition registers successfully
    When I register a "PROTOBUF" schema under subject "proto-conform-service":
      """
syntax = "proto3";
package test.svc;

message Request {
  string query = 1;
}

message Response {
  string result = 1;
}

service SearchService {
  rpc Search(Request) returns (Response);
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 15. SAME PROTO IN TWO SUBJECTS — SAME ID (DEDUP)
  # ==========================================================================

  Scenario: Same proto in two subjects produces same global ID
    When I register a "PROTOBUF" schema under subject "proto-conform-dedup-a":
      """
syntax = "proto3";
package test.dedup;

message Event {
  string id = 1;
  int64 timestamp = 2;
  string payload = 3;
}
      """
    Then the response status should be 200
    And I store the response field "id" as "proto_dedup_id"
    When I register a "PROTOBUF" schema under subject "proto-conform-dedup-b":
      """
syntax = "proto3";
package test.dedup;

message Event {
  string id = 1;
  int64 timestamp = 2;
  string payload = 3;
}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "proto_dedup_id"
    And I store the response field "id" as "schema_id"
    When I get the subjects for the stored schema ID
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "proto-conform-dedup-a"
    And the response array should contain "proto-conform-dedup-b"

  # ==========================================================================
  # 16. PROTO WITH ENUM AS MAP VALUE
  # ==========================================================================

  Scenario: Proto with enum as map value
    When I register a "PROTOBUF" schema under subject "proto-conform-enum-map":
      """
syntax = "proto3";
package test.enummap;

message Config {
  map<string, Level> levels = 1;
  enum Level {
    LOW = 0;
    MEDIUM = 1;
    HIGH = 2;
  }
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 17. CONTENT ROUND-TRIP
  # ==========================================================================

  Scenario: Content round-trip verifies proto keywords preserved
    Given subject "proto-conform-roundtrip" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.roundtrip;

message Order {
  string id = 1;
  Status status = 2;
  oneof payment {
    string card = 3;
    string bank = 4;
  }
  enum Status {
    PENDING = 0;
    SHIPPED = 1;
    DELIVERED = 2;
  }
}
      """
    When I get version 1 of subject "proto-conform-roundtrip"
    Then the response status should be 200
    And the response body should contain "message"
    And the response body should contain "enum"
    And the response body should contain "oneof"

  # ==========================================================================
  # 18. PROTO3 OPTIONAL FIELD (EXPLICIT PRESENCE)
  # ==========================================================================

  Scenario: Proto3 optional field with explicit presence
    When I register a "PROTOBUF" schema under subject "proto-conform-optional":
      """
syntax = "proto3";
package test.optional;

message User {
  string name = 1;
  optional int32 age = 2;
  optional string email = 3;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 19. PROTO2 EXTENSIONS
  # ==========================================================================

  Scenario: Proto2 extensions with extend block
    When I register a "PROTOBUF" schema under subject "proto-conform-extensions":
      """
syntax = "proto2";
package test.extensions;

message Base {
  required string name = 1;
  extensions 100 to 199;
}

extend Base {
  optional int32 age = 100;
  optional string email = 101;
}
      """
    Then the response status should be 200

  # ==========================================================================
  # 20. PROTO2 GROUPS
  # ==========================================================================

  Scenario: Proto2 groups with deprecated group keyword
    When I register a "PROTOBUF" schema under subject "proto-conform-groups":
      """
syntax = "proto2";
package test.groups;

message SearchResponse {
  repeated group Result = 1 {
    required string url = 2;
    optional string title = 3;
  }
}
      """
    Then the response status should be 200
