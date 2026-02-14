@functional
Feature: Protobuf Compatibility Diff — Data-Driven (Confluent v8.1.1 Compatibility)
  Data-driven Protobuf compatibility tests from the Confluent Schema Registry v8.1.1
  diff-schema-examples.json test suite (43 test cases).

  Scenario: Protobuf diff 01 — Allow addition of un-reserved property
    Given the global compatibility level is "NONE"
    And subject "proto-diff-01" has "PROTOBUF" schema:
      """
      syntax = "proto3";
package foo;
message TestMessage {
    string test_string = 1;
    .foo.TestMessage.Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-01" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-01":
      """
      syntax = "proto3";
package foo;

message TestMessage {
  string test_string = 1;
  .foo.TestMessage.Status test_int = 2;
  float test_float = 3;

  message Status {
    string test_string = 1;
  }
}

      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 02 — Detect package change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-02" has "PROTOBUF" schema:
      """
      syntax = "proto3";
package foo;
message TestMessage {
    string test_string = 1;
    .foo.TestMessage.Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-02" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-02":
      """
      syntax = "proto3";
package bar;
message TestMessage {
    string test_string = 1;
    .bar.TestMessage.Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 03 — Detect change to name qualification
    Given the global compatibility level is "NONE"
    And subject "proto-diff-03" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-03" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-03":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    .TestMessage.Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 04 — Detect change to name qualification with package
    Given the global compatibility level is "NONE"
    And subject "proto-diff-04" has "PROTOBUF" schema:
      """
      syntax = "proto3";
package foo;
message TestMessage {
    string test_string = 1;
    Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-04" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-04":
      """
      syntax = "proto3";
package foo;
message TestMessage {
    string test_string = 1;
    .foo.TestMessage.Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 05 — Detect added field
    Given the global compatibility level is "NONE"
    And subject "proto-diff-05" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
}
      """
    When I set the config for subject "proto-diff-05" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-05":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    string test_string2 = 2;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 06 — Detect added required field
    Given the global compatibility level is "NONE"
    And subject "proto-diff-06" has "PROTOBUF" schema:
      """
      syntax = "proto2";
message TestMessage {
    optional string test_string = 1;
}
      """
    When I set the config for subject "proto-diff-06" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-06":
      """
      syntax = "proto2";
message TestMessage {
    optional string test_string = 1;
    required string test_string2 = 2;
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 07 — Detect added field to nested type
    Given the global compatibility level is "NONE"
    And subject "proto-diff-07" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message Wrapper {
    int32 wrapperInt = 1;
    message TestMessage {
    string test_string = 1;
}
}
      """
    When I set the config for subject "proto-diff-07" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-07":
      """
      syntax = "proto3";
message Wrapper {
    int32 wrapperInt = 1;
    message TestMessage {
    string test_string = 1;
    string test_string2 = 2;
}
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 08 — Detect removed field
    Given the global compatibility level is "NONE"
    And subject "proto-diff-08" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    string test_string2 = 2;
}
      """
    When I set the config for subject "proto-diff-08" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-08":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
}

      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 09 — Detect removed required field
    Given the global compatibility level is "NONE"
    And subject "proto-diff-09" has "PROTOBUF" schema:
      """
      syntax = "proto2";
message TestMessage {
    optional string test_string = 1;
    required string test_string2 = 2;
}
      """
    When I set the config for subject "proto-diff-09" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-09":
      """
      syntax = "proto2";
message TestMessage {
    optional string test_string = 1;
}

      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 10 — Detect changed field number
    Given the global compatibility level is "NONE"
    And subject "proto-diff-10" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
}
      """
    When I set the config for subject "proto-diff-10" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-10":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 2;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 11 — Detect compatible field number type change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-11" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    int32 test_int = 2;
}
      """
    When I set the config for subject "proto-diff-11" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-11":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    int64 test_int = 2;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 12 — Detect compatible field string type change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-12" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    string test_string2 = 2;
}
      """
    When I set the config for subject "proto-diff-12" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-12":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    bytes test_string2 = 2;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 13 — Detect field number type to enum change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-13" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    int32 test_int = 2;
}
      """
    When I set the config for subject "proto-diff-13" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-13":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    Status test_int = 2;
}
enum Status {
    ACTIVE = 0;
    INACTIVE = 1;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 14 — Detect enum type to field number change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-14" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    Status test_int = 2;
}
enum Status {
    ACTIVE = 0;
    INACTIVE = 1;
}
      """
    When I set the config for subject "proto-diff-14" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-14":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    int32 test_int = 2;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 15 — Detect field number type to message with same name as enum
    Given the global compatibility level is "NONE"
    And subject "proto-diff-15" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    int32 test_int = 2;
}
      """
    When I set the config for subject "proto-diff-15" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-15":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    .TestMessage.Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
enum Status {
    ACTIVE = 0;
    INACTIVE = 1;
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 16 — Detect message with same name as enum to field number change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-16" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    .TestMessage.Status test_int = 2;
    message Status {
    string test_string = 1;
    }
}
enum Status {
    ACTIVE = 0;
    INACTIVE = 1;
}
      """
    When I set the config for subject "proto-diff-16" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-16":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    int32 test_int = 2;
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 17 — Detect enum const change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-17" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    Status test_int = 2;
}
enum Status {
    ACTIVE = 0;
    INACTIVE = 1;
}
      """
    When I set the config for subject "proto-diff-17" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-17":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    Status test_int = 2;
}
enum Status {
    ACTIVE = 0;
    NOT_ACTIVE = 1;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 18 — Detect compatible field to oneof change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-18" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    string test_string2 = 2;
}
      """
    When I set the config for subject "proto-diff-18" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-18":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    oneof new_oneof {
        string test_string2 = 2;
        int32 other_id = 3;
}
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 19 — Detect incompatible field to oneof change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-19" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    string test_string2 = 2;
}
      """
    When I set the config for subject "proto-diff-19" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-19":
      """
      syntax = "proto3";
message TestMessage {
    oneof new_oneof {
    string test_string = 1;
    string test_string2 = 2;
        int32 other_id = 3;
}
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 20 — Detect compatible add field to oneof
    Given the global compatibility level is "NONE"
    And subject "proto-diff-20" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    oneof new_oneof {
    string test_string = 1;
    string test_string2 = 2;
}
}
      """
    When I set the config for subject "proto-diff-20" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-20":
      """
      syntax = "proto3";
message TestMessage {
    oneof new_oneof {
    string test_string = 1;
    string test_string2 = 2;
        int32 other_id = 3;
}
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 21 — Detect incompatible remove field from oneof
    Given the global compatibility level is "NONE"
    And subject "proto-diff-21" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    oneof new_oneof {
    string test_string = 1;
    string test_string2 = 2;
        int32 other_id = 3;
}
}
      """
    When I set the config for subject "proto-diff-21" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-21":
      """
      syntax = "proto3";
message TestMessage {
    oneof new_oneof {
    string test_string = 1;
    string test_string2 = 2;
}
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 22 — Detect compatible move field to oneof
    Given the global compatibility level is "NONE"
    And subject "proto-diff-22" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    string test_string2 = 2;
}
      """
    When I set the config for subject "proto-diff-22" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-22":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    oneof new_oneof {
    string test_string2 = 2;
}
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 23 — Detect incompatible move field to oneof
    Given the global compatibility level is "NONE"
    And subject "proto-diff-23" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    oneof new_oneof {
    string test_string2 = 2;
    string test_string3 = 3;
}
}
      """
    When I set the config for subject "proto-diff-23" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-23":
      """
      syntax = "proto3";
message TestMessage {
    oneof new_oneof {
    string test_string = 1;
    string test_string2 = 2;
    string test_string3 = 3;
}
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 24 — Detect incompatible move field to renamed oneof
    Given the global compatibility level is "NONE"
    And subject "proto-diff-24" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    oneof new_oneof {
    string test_string2 = 2;
    string test_string3 = 3;
}
}
      """
    When I set the config for subject "proto-diff-24" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-24":
      """
      syntax = "proto3";
message TestMessage {
    oneof new_oneof_2 {
    string test_string = 1;
    string test_string2 = 2;
    string test_string3 = 3;
}
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 25 — Detect incompatible field type change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-25" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    int32 test_int = 2;
}
      """
    When I set the config for subject "proto-diff-25" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-25":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    TestMessage2 test_int = 2;
}
message TestMessage2 {
    string s1 = 1;
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 26 — Detect incompatible field type change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-26" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    int32 test_int = 2;
}
      """
    When I set the config for subject "proto-diff-26" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-26":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    string test_string2 = 2;
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 27 — Detect incompatible field type change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-27" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    TestMessage2 test_int = 2;
}
message TestMessage2 {
    string s1 = 1;
}
      """
    When I set the config for subject "proto-diff-27" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-27":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    TestMessage3 test_int = 2;
}
message TestMessage3 {
    string s1 = 1;
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 28 — Detect incompatible field type change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-28" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    map<string, int32> test_map_int = 2;
}
      """
    When I set the config for subject "proto-diff-28" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-28":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    map<string, string> test_map_string = 2;
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 29 — Detect compatible name qualification in map
    Given the global compatibility level is "NONE"
    And subject "proto-diff-29" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    map<string, Status> test_map = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-29" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-29":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    map<string, TestMessage.Status> test_map = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 30 — Detect compatible field label change for message
    Given the global compatibility level is "NONE"
    And subject "proto-diff-30" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    optional Status test_status = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-30" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-30":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
    repeated Status test_status = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 31 — Detect compatible field label change for string
    Given the global compatibility level is "NONE"
    And subject "proto-diff-31" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    optional string test_string = 1;
    Status test_status = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-31" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-31":
      """
      syntax = "proto3";
message TestMessage {
    repeated string test_string = 1;
    Status test_status = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 32 — Detect compatible field label change for bytes
    Given the global compatibility level is "NONE"
    And subject "proto-diff-32" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    optional bytes test_bytes = 1;
    Status test_status = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-32" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-32":
      """
      syntax = "proto3";
message TestMessage {
    repeated bytes test_bytes = 1;
    Status test_status = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 33 — Detect incompatible field label change for number
    Given the global compatibility level is "NONE"
    And subject "proto-diff-33" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    optional int32 test_int = 1;
    Status test_status = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    When I set the config for subject "proto-diff-33" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-33":
      """
      syntax = "proto3";
message TestMessage {
    repeated int32 test_int = 1;
    Status test_status = 2;
    message Status {
    string test_string = 1;
    }
}
      """
    Then the compatibility check should be incompatible

  Scenario: Protobuf diff 34 — Detect compatible message addition
    Given the global compatibility level is "NONE"
    And subject "proto-diff-34" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
}
      """
    When I set the config for subject "proto-diff-34" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-34":
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
}
message TestMessage2 {
    string test_string = 1;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 35 — Detect compatible message index change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-35" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
}
      """
    When I set the config for subject "proto-diff-35" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-35":
      """
      syntax = "proto3";
message TestMessage2 {
    string test_string = 1;
}
message TestMessage {
    string test_string = 1;
}
      """
    Then the compatibility check should be compatible

  Scenario: Protobuf diff 36 — Detect compatible message index change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-36" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    string test_string = 1;
}
message TestMessage2 {
    string test_string = 1;
}
      """
    When I set the config for subject "proto-diff-36" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-36":
      """
      syntax = "proto3";
message TestMessage2 {
    string test_string = 1;
}
message TestMessage {
    string test_string = 1;
}
      """
    Then the compatibility check should be compatible

  @pending-impl
  Scenario: Protobuf diff 37 — Detect incompatible import change
    Given the global compatibility level is "NONE"
    And subject "proto-diff-37" has "PROTOBUF" schema:
      """
      syntax = "proto3";
import "google.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain.GoogleHome test_string = 1;
    }
      """
    When I set the config for subject "proto-diff-37" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-37":
      """
      syntax = "proto3";
import "google2.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain.GoogleHome test_string = 1;
    }
      """
    Then the compatibility check should be incompatible

  @pending-impl
  Scenario: Protobuf diff 38 — Detect moving to import
    Given the global compatibility level is "NONE"
    And subject "proto-diff-38" has "PROTOBUF" schema:
      """
      syntax = "proto3";
message TestMessage {
    GoogleHome test_string = 1;
    }
message GoogleHome {
  int32 deviceID = 1;
  bool enabled = 2;
}
      """
    When I set the config for subject "proto-diff-38" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-38":
      """
      syntax = "proto3";
import "google.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain.GoogleHome test_string = 1;
    }
      """
    Then the compatibility check should be incompatible

  @pending-impl
  Scenario: Protobuf diff 39 — Detect moving from import
    Given the global compatibility level is "NONE"
    And subject "proto-diff-39" has "PROTOBUF" schema:
      """
      syntax = "proto3";
import "google.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain.GoogleHome test_string = 1;
    }
      """
    When I set the config for subject "proto-diff-39" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-39":
      """
      syntax = "proto3";
message TestMessage {
    GoogleHome test_string = 1;
    }
message GoogleHome {
  int32 deviceID = 1;
  bool enabled = 2;
}
      """
    Then the compatibility check should be compatible

  @pending-impl
  Scenario: Protobuf diff 40 — Detect import change with different remote package name
    Given the global compatibility level is "NONE"
    And subject "proto-diff-40" has "PROTOBUF" schema:
      """
      syntax = "proto3";
import "google.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain.GoogleHome test_string = 1;
    }
      """
    When I set the config for subject "proto-diff-40" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-40":
      """
      syntax = "proto3";
import "google2.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain2.GoogleHome test_string = 1;
    }
      """
    Then the compatibility check should be compatible

  @pending-impl
  Scenario: Protobuf diff 41 — Detect import change with different remote package nested name
    Given the global compatibility level is "NONE"
    And subject "proto-diff-41" has "PROTOBUF" schema:
      """
      syntax = "proto3";
import "google.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain.outer.GoogleHome test_string = 1;
    }
      """
    When I set the config for subject "proto-diff-41" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-41":
      """
      syntax = "proto3";
import "google2.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain2.outer.GoogleHome test_string = 1;
    }
      """
    Then the compatibility check should be compatible

  @pending-impl
  Scenario: Protobuf diff 42 — Detect incompatible import change with different remote package nested name
    Given the global compatibility level is "NONE"
    And subject "proto-diff-42" has "PROTOBUF" schema:
      """
      syntax = "proto3";
import "google.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain.outer.GoogleHome test_string = 1;
    }
      """
    When I set the config for subject "proto-diff-42" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-42":
      """
      syntax = "proto3";
import "google2.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain2.outer.GoogleHome test_string = 1;
    }
      """
    Then the compatibility check should be incompatible

  @pending-impl
  Scenario: Protobuf diff 43 — Detect incompatible import change with same subject, different version
    Given the global compatibility level is "NONE"
    And subject "proto-diff-43" has "PROTOBUF" schema:
      """
      syntax = "proto3";
import "google.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain.outer.GoogleHome test_string = 1;
    }
      """
    When I set the config for subject "proto-diff-43" to "BACKWARD"
    And I check compatibility of "PROTOBUF" schema against subject "proto-diff-43":
      """
      syntax = "proto3";
import "google.proto";
message TestMessage {
    .io.confluent.cloud.demo.domain2.outer.GoogleHome test_string = 1;
    }
      """
    Then the compatibility check should be incompatible
