@functional
Feature: Schema References — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive schema reference tests covering delete protection, dangling
  references, multi-level references with resolved format, and cross-type references.

  # ==========================================================================
  # DELETE PROTECTION — Referenced schemas cannot be deleted
  # ==========================================================================

  Scenario: Delete referenced schema version fails with REFERENCE_EXISTS
    Given the global compatibility level is "NONE"
    And subject "ref-ex-base" has schema:
      """
      {"type":"record","name":"Base","namespace":"com.refex","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-ex-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.refex\",\"fields\":[{\"name\":\"base\",\"type\":\"com.refex.Base\"}]}",
        "references": [
          {"name": "com.refex.Base", "subject": "ref-ex-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I DELETE "/subjects/ref-ex-base/versions/1"
    Then the response status should be 422
    And the response should have error code 42206

  Scenario: Delete referenced subject fails with REFERENCE_EXISTS
    Given the global compatibility level is "NONE"
    And subject "ref-ex-subj-base" has schema:
      """
      {"type":"record","name":"SubjBase","namespace":"com.refex","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-ex-subj-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"SubjConsumer\",\"namespace\":\"com.refex\",\"fields\":[{\"name\":\"base\",\"type\":\"com.refex.SubjBase\"}]}",
        "references": [
          {"name": "com.refex.SubjBase", "subject": "ref-ex-subj-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I delete subject "ref-ex-subj-base"
    Then the response status should be 422
    And the response should have error code 42206

  # ==========================================================================
  # DELETE REFERRER THEN REFERENCE — Correct deletion order
  # ==========================================================================

  Scenario: Delete referrer then delete reference succeeds
    Given the global compatibility level is "NONE"
    And subject "ref-ex-del-base" has schema:
      """
      {"type":"record","name":"DelBase","namespace":"com.refdel","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-ex-del-referrer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"DelReferrer\",\"namespace\":\"com.refdel\",\"fields\":[{\"name\":\"base\",\"type\":\"com.refdel.DelBase\"}]}",
        "references": [
          {"name": "com.refdel.DelBase", "subject": "ref-ex-del-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "referrer_id"
    # Delete the referrer first
    When I delete subject "ref-ex-del-referrer"
    Then the response status should be 200
    # Now referencedby should be empty
    When I get the referenced by for subject "ref-ex-del-base" version 1
    Then the response status should be 200
    # Now deleting the reference should succeed
    When I delete version 1 of subject "ref-ex-del-base"
    Then the response status should be 200

  # ==========================================================================
  # MULTI-LEVEL REFERENCES WITH SCHEMA RETRIEVAL
  # ==========================================================================

  Scenario: Multi-level references — all references returned by schema ID
    Given the global compatibility level is "NONE"
    And subject "ref-ex-ml-base" has schema:
      """
      {"type":"record","name":"MLBase","namespace":"com.ml","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-ex-ml-mid" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MLMid\",\"namespace\":\"com.ml\",\"fields\":[{\"name\":\"base\",\"type\":\"com.ml.MLBase\"}]}",
        "references": [
          {"name": "com.ml.MLBase", "subject": "ref-ex-ml-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I register a schema under subject "ref-ex-ml-top" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MLTop\",\"namespace\":\"com.ml\",\"fields\":[{\"name\":\"mid\",\"type\":\"com.ml.MLMid\"}]}",
        "references": [
          {"name": "com.ml.MLBase", "subject": "ref-ex-ml-base", "version": 1},
          {"name": "com.ml.MLMid", "subject": "ref-ex-ml-mid", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "top_id"
    When I get schema by ID {{top_id}}
    Then the response status should be 200
    And the response should contain "MLTop"

  Scenario: Referencedby shows multiple referrers
    Given the global compatibility level is "NONE"
    And subject "ref-ex-shared" has schema:
      """
      {"type":"record","name":"Shared","namespace":"com.shared2","fields":[{"name":"val","type":"string"}]}
      """
    When I register a schema under subject "ref-ex-use1" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Use1\",\"namespace\":\"com.shared2\",\"fields\":[{\"name\":\"s\",\"type\":\"com.shared2.Shared\"}]}",
        "references": [
          {"name": "com.shared2.Shared", "subject": "ref-ex-shared", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "use1_id"
    When I register a schema under subject "ref-ex-use2" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Use2\",\"namespace\":\"com.shared2\",\"fields\":[{\"name\":\"s\",\"type\":\"com.shared2.Shared\"}]}",
        "references": [
          {"name": "com.shared2.Shared", "subject": "ref-ex-shared", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "use2_id"
    When I get the referenced by for subject "ref-ex-shared" version 1
    Then the response status should be 200
    And the response array should contain stored integer "use1_id"
    And the response array should contain stored integer "use2_id"

  # ==========================================================================
  # DANGLING REFERENCES — Delete and recreate
  # ==========================================================================

  Scenario: Hard-delete reference then register referrer with dangling ref fails
    Given the global compatibility level is "NONE"
    And subject "ref-ex-dangle-base" has schema:
      """
      {"type":"record","name":"DBase","namespace":"com.dangle","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-ex-dangle-user" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"DUser\",\"namespace\":\"com.dangle\",\"fields\":[{\"name\":\"base\",\"type\":\"com.dangle.DBase\"}]}",
        "references": [
          {"name": "com.dangle.DBase", "subject": "ref-ex-dangle-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Delete the referrer to unblock reference deletion
    When I delete subject "ref-ex-dangle-user"
    Then the response status should be 200
    # Soft-delete then hard-delete the reference
    When I delete subject "ref-ex-dangle-base"
    Then the response status should be 200
    When I permanently delete subject "ref-ex-dangle-base"
    Then the response status should be 200
    # Now try to register a new schema with dangling reference — should fail
    When I register a schema under subject "ref-ex-dangle-new" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"DNew\",\"namespace\":\"com.dangle\",\"fields\":[{\"name\":\"base\",\"type\":\"com.dangle.DBase\"}]}",
        "references": [
          {"name": "com.dangle.DBase", "subject": "ref-ex-dangle-base", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  # ==========================================================================
  # CROSS-TYPE REFERENCES
  # ==========================================================================

  Scenario: JSON Schema with cross-subject reference is retrievable
    Given the global compatibility level is "NONE"
    And subject "ref-ex-json-addr" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}},"required":["street"]}
      """
    When I register a "JSON" schema under subject "ref-ex-json-person" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"address\":{\"$ref\":\"address.json\"}},\"required\":[\"name\"]}",
        "schemaType": "JSON",
        "references": [
          {"name": "address.json", "subject": "ref-ex-json-addr", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "json_ref_id"
    When I get schema by ID {{json_ref_id}}
    Then the response status should be 200
    And the response should have field "references"

  Scenario: Protobuf with cross-subject reference is retrievable
    Given the global compatibility level is "NONE"
    And subject "ref-ex-proto-common" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package common;
      message Timestamp {
        int64 seconds = 1;
        int32 nanos = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "ref-ex-proto-event" with references:
      """
      {
        "schema": "syntax = \"proto3\";\nimport \"common.proto\";\npackage events;\nmessage Event {\n  string id = 1;\n  common.Timestamp created = 2;\n}",
        "schemaType": "PROTOBUF",
        "references": [
          {"name": "common.proto", "subject": "ref-ex-proto-common", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "proto_ref_id"
    When I get schema by ID {{proto_ref_id}}
    Then the response status should be 200
    And the response should have field "references"

  # ==========================================================================
  # MISSING/EMPTY REFERENCES
  # ==========================================================================

  Scenario: Register schema with empty references array succeeds
    Given the global compatibility level is "NONE"
    When I POST "/subjects/ref-ex-empty-refs/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"EmptyRefs\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "references": []}
      """
    Then the response status should be 200

  Scenario: Referencedby for non-referenced schema returns empty array
    Given the global compatibility level is "NONE"
    And subject "ref-ex-no-refs" has schema:
      """
      {"type":"record","name":"NoRefs","fields":[{"name":"a","type":"string"}]}
      """
    When I get the referenced by for subject "ref-ex-no-refs" version 1
    Then the response status should be 200
    And the response should be an array of length 0
