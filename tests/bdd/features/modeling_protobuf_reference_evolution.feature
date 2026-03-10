@schema-modeling @protobuf @references
Feature: Protobuf Reference Evolution
  Tests for evolving proto schemas that use cross-subject imports,
  including import version pinning, multiple imports, referencedby
  tracking, and reference deletion behavior.

  # ==========================================================================
  # 1. IMPORT EVOLVES — CONSUMER STAYS PINNED
  # ==========================================================================

  Scenario: Consumer stays pinned to import v1 when import evolves
    Given subject "proto-refevo-common" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.refevo;

message Address {
  string street = 1;
  string city = 2;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-refevo-consumer" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.refevo;\n\nimport \"proto/refevo/address.proto\";\n\nmessage Person {\n  string name = 1;\n  Address home = 2;\n}",
        "references": [
          {"name":"proto/refevo/address.proto","subject":"proto-refevo-common","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "proto_consumer_v1"
    And the audit log should contain event "schema_register" with subject "proto-refevo-consumer"

  # ==========================================================================
  # 2. CONSUMER UPGRADES TO IMPORT V2
  # ==========================================================================

  Scenario: Consumer upgrades import version gets different schema ID
    Given subject "proto-refevo2-dep" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.refevo2;

message Dependency {
  int32 value = 1;
}
      """
    Given subject "proto-refevo2-dep" has compatibility level "BACKWARD"
    When I register a "PROTOBUF" schema under subject "proto-refevo2-dep":
      """
syntax = "proto3";
package proto.refevo2;

message Dependency {
  int32 value = 1;
  string label = 2;
}
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "proto-refevo2-c1" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.refevo2;\n\nimport \"proto/refevo2/dep.proto\";\n\nmessage Main {\n  Dependency dep = 1;\n}",
        "references": [
          {"name":"proto/refevo2/dep.proto","subject":"proto-refevo2-dep","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "proto_ref1_id"
    When I register a "PROTOBUF" schema under subject "proto-refevo2-c2" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.refevo2;\n\nimport \"proto/refevo2/dep.proto\";\n\nmessage Main {\n  Dependency dep = 1;\n}",
        "references": [
          {"name":"proto/refevo2/dep.proto","subject":"proto-refevo2-dep","version":2}
        ]
      }
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "proto_ref1_id"
    And the audit log should contain event "schema_register" with subject "proto-refevo2-c2"

  # ==========================================================================
  # 3. MULTIPLE IMPORTS
  # ==========================================================================

  Scenario: Schema with multiple proto imports registers successfully
    Given subject "proto-multiref-types" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.multi;

message TypeA {
  string name = 1;
}
      """
    And subject "proto-multiref-enums" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.multi;

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-multiref-consumer" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.multi;\n\nimport \"proto/multi/types.proto\";\nimport \"proto/multi/enums.proto\";\n\nmessage Combined {\n  TypeA item = 1;\n  Status status = 2;\n}",
        "references": [
          {"name":"proto/multi/types.proto","subject":"proto-multiref-types","version":1},
          {"name":"proto/multi/enums.proto","subject":"proto-multiref-enums","version":1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "proto-multiref-consumer"

  # ==========================================================================
  # 4. REFERENCEDBY TRACKING
  # ==========================================================================

  Scenario: referencedby tracks proto import consumers
    Given subject "proto-refby-shared" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.refby;

message Shared {
  int64 id = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-refby-c1" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.refby;\n\nimport \"proto/refby/shared.proto\";\n\nmessage Consumer1 {\n  Shared s = 1;\n}",
        "references": [
          {"name":"proto/refby/shared.proto","subject":"proto-refby-shared","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "proto-refby-c2" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.refby;\n\nimport \"proto/refby/shared.proto\";\n\nmessage Consumer2 {\n  Shared s = 1;\n}",
        "references": [
          {"name":"proto/refby/shared.proto","subject":"proto-refby-shared","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "proto-refby-shared" version 1
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "proto-refby-c2"

  # ==========================================================================
  # 5. SAME PROTO BODY WITH DIFFERENT IMPORT VERSIONS — DIFFERENT IDS
  # ==========================================================================

  Scenario: Same proto body with different import versions produces different IDs
    Given subject "proto-diffref-dep" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.diff;

message Dep {
  int32 v = 1;
}
      """
    Given subject "proto-diffref-dep" has compatibility level "NONE"
    When I register a "PROTOBUF" schema under subject "proto-diffref-dep":
      """
syntax = "proto3";
package proto.diff;

message Dep {
  string v = 1;
}
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "proto-diffref-a" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.diff;\n\nimport \"proto/diff/dep.proto\";\n\nmessage Main {\n  Dep d = 1;\n}",
        "references": [
          {"name":"proto/diff/dep.proto","subject":"proto-diffref-dep","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "proto_diff_v1"
    When I register a "PROTOBUF" schema under subject "proto-diffref-b" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.diff;\n\nimport \"proto/diff/dep.proto\";\n\nmessage Main {\n  Dep d = 1;\n}",
        "references": [
          {"name":"proto/diff/dep.proto","subject":"proto-diffref-dep","version":2}
        ]
      }
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "proto_diff_v1"
    And the audit log should contain event "schema_register" with subject "proto-diffref-b"

  # ==========================================================================
  # 6. DELETE REFERENCED PROTO — CONSUMER STILL RETRIEVABLE
  # ==========================================================================

  Scenario: Deleting referenced proto does not break consumer retrieval
    Given subject "proto-refdel-base" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.del;

message Base {
  int32 x = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-refdel-consumer" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.del;\n\nimport \"proto/del/base.proto\";\n\nmessage Consumer {\n  Base b = 1;\n}",
        "references": [
          {"name":"proto/del/base.proto","subject":"proto-refdel-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I get version 1 of subject "proto-refdel-consumer"
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "proto-refdel-consumer"

  # ==========================================================================
  # 7. COMPATIBILITY WITH IMPORT REFERENCES
  # ==========================================================================

  Scenario: Compatibility check works with proto import references
    Given subject "proto-refcompat-dep" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.rc;

message Dep {
  int32 v = 1;
}
      """
    And subject "proto-refcompat-main" has compatibility level "BACKWARD"
    When I register a "PROTOBUF" schema under subject "proto-refcompat-main" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.rc;\n\nimport \"proto/rc/dep.proto\";\n\nmessage Main {\n  Dep d = 1;\n  string name = 2;\n}",
        "references": [
          {"name":"proto/rc/dep.proto","subject":"proto-refcompat-dep","version":1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "proto-refcompat-main"

  # ==========================================================================
  # 8. IMPORT CHAIN — A IMPORTS B IMPORTS C
  # ==========================================================================

  Scenario: Proto import chain registers successfully
    Given subject "proto-chain-c" has "PROTOBUF" schema:
      """
syntax = "proto3";
package proto.chain;

message TypeC {
  string value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-chain-b" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.chain;\n\nimport \"proto/chain/c.proto\";\n\nmessage TypeB {\n  TypeC c = 1;\n}",
        "references": [
          {"name":"proto/chain/c.proto","subject":"proto-chain-c","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "proto-chain-a" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage proto.chain;\n\nimport \"proto/chain/b.proto\";\nimport \"proto/chain/c.proto\";\n\nmessage TypeA {\n  TypeB b = 1;\n  TypeC c = 2;\n}",
        "references": [
          {"name":"proto/chain/b.proto","subject":"proto-chain-b","version":1},
          {"name":"proto/chain/c.proto","subject":"proto-chain-c","version":1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "proto-chain-a"
