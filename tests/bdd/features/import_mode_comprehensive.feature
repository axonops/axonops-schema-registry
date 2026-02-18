@functional
Feature: IMPORT Mode — Comprehensive Corner Cases
  Exhaustive tests for IMPORT mode behavior: mode mutual-exclusion with
  READWRITE, explicit ID and version handling, ID sequencing, per-subject
  import isolation, schema type support, real-world migration workflows,
  idempotency, and edge cases.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # MODE MUTUAL EXCLUSION — IMPORT vs READWRITE
  # ==========================================================================

  Scenario: IMPORT mode rejects registration without explicit ID via POST
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-noid/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}"}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: IMPORT mode rejects schema lookup (POST without ID to subject)
    When I set the global mode to "READWRITE"
    And subject "imp-comp-lookup-prep" has schema:
      """
      {"type":"record","name":"LookupPrep","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "IMPORT"
    # Lookup is a POST to /subjects/{subject} — treated as registration without ID
    When I POST "/subjects/imp-comp-lookup-prep/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LookupPrep\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READWRITE mode rejects explicit ID with any value
    When I POST "/subjects/imp-comp-rw-id1/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 1}
      """
    Then the response status should be 422
    And the response should have error code 42205

  Scenario: READWRITE mode rejects explicit ID with large value
    When I POST "/subjects/imp-comp-rw-idlarge/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 999999}
      """
    Then the response status should be 422
    And the response should have error code 42205

  Scenario: READONLY mode rejects registration with explicit ID
    When I set the global mode to "READONLY"
    When I POST "/subjects/imp-comp-ro-id/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 50000}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READONLY mode rejects registration without explicit ID
    When I set the global mode to "READONLY"
    When I register a schema under subject "imp-comp-ro-noid":
      """
      {"type":"string"}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: IMPORT mode still allows read operations
    When I set the global mode to "READWRITE"
    And subject "imp-comp-read" has schema:
      """
      {"type":"record","name":"ReadInImport","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "IMPORT"
    When I get the latest version of subject "imp-comp-read"
    Then the response status should be 200
    When I list all subjects
    Then the response status should be 200
    When I get the global config
    Then the response status should be 200
    When I get the global mode
    Then the response status should be 200
    And the response field "mode" should be "IMPORT"
    When I set the global mode to "READWRITE"

  Scenario: IMPORT mode allows mode changes
    When I set the global mode to "IMPORT"
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I set the global mode to "IMPORT"
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  # ==========================================================================
  # PER-SUBJECT IMPORT MODE ISOLATION
  # ==========================================================================

  Scenario: Per-subject IMPORT mode allows import on that subject only
    Given the global mode is "READWRITE"
    When I set the mode for subject "imp-comp-per-imp" to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/imp-comp-per-imp/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 70000, "version": 1}
      """
    Then the response status should be 200
    And the response field "id" should be 70000

  Scenario: Per-subject IMPORT blocks normal registration on that subject
    Given the global mode is "READWRITE"
    When I set the mode for subject "imp-comp-per-block" to "IMPORT"
    When I register a schema under subject "imp-comp-per-block":
      """
      {"type":"string"}
      """
    Then the response status should be 422
    And the response should have error code 42205

  Scenario: Per-subject IMPORT doesn't affect other subjects in READWRITE
    Given the global mode is "READWRITE"
    When I set the mode for subject "imp-comp-per-isolated" to "IMPORT"
    # Other subjects still work normally
    When I register a schema under subject "imp-comp-per-other":
      """
      {"type":"record","name":"Other","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    # The IMPORT subject blocks normal registration
    When I register a schema under subject "imp-comp-per-isolated":
      """
      {"type":"record","name":"Isolated","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422

  Scenario: Per-subject READWRITE overrides global IMPORT
    When I set the global mode to "IMPORT"
    When I PUT "/mode/imp-comp-per-rw-override" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    # Normal registration works on this subject
    When I register a schema under subject "imp-comp-per-rw-override":
      """
      {"type":"record","name":"RWOverride","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  Scenario: Per-subject IMPORT overrides global READWRITE for explicit ID
    Given the global mode is "READWRITE"
    When I set the mode for subject "imp-comp-per-imp-override" to "IMPORT"
    When I POST "/subjects/imp-comp-per-imp-override/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 70010, "version": 1}
      """
    Then the response status should be 200
    And the response field "id" should be 70010
    # But explicit ID on a non-IMPORT subject fails
    When I POST "/subjects/imp-comp-per-other-subject/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 70011}
      """
    Then the response status should be 422

  # ==========================================================================
  # EXPLICIT VERSION HANDLING
  # ==========================================================================

  Scenario: Import with version 1 — basic case
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ver1/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 71000, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 1 of subject "imp-comp-ver1"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response field "id" should be 71000

  Scenario: Import with high version number
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ver-high/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 71010, "version": 999}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 999 of subject "imp-comp-ver-high"
    Then the response status should be 200
    And the response field "version" should be 999

  Scenario: Import reverse order — version 3 before version 1
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ver-rev/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Rev\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"},{\"name\":\"c\",\"type\":\"string\"}]}", "id": 71023, "version": 3}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ver-rev/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Rev\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 71021, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ver-rev/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Rev\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"}]}", "id": 71022, "version": 2}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I list versions of subject "imp-comp-ver-rev"
    Then the response status should be 200
    And the response should be an array of length 3
    When I get version 1 of subject "imp-comp-ver-rev"
    Then the response status should be 200
    And the response field "id" should be 71021
    When I get version 3 of subject "imp-comp-ver-rev"
    Then the response status should be 200
    And the response field "id" should be 71023

  Scenario: Import with large version gaps
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ver-gaps/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 71030, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ver-gaps/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 71031, "version": 50}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ver-gaps/versions" with body:
      """
      {"schema": "{\"type\":\"long\"}", "id": 71032, "version": 500}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I list versions of subject "imp-comp-ver-gaps"
    Then the response status should be 200
    And the response should be an array of length 3
    When I get version 50 of subject "imp-comp-ver-gaps"
    Then the response status should be 200
    And the response field "id" should be 71031

  Scenario: Import without version auto-assigns sequentially
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ver-auto/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 71040}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ver-auto/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 71041}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ver-auto/versions" with body:
      """
      {"schema": "{\"type\":\"long\"}", "id": 71042}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 1 of subject "imp-comp-ver-auto"
    Then the response status should be 200
    And the response field "id" should be 71040
    When I get version 2 of subject "imp-comp-ver-auto"
    Then the response status should be 200
    And the response field "id" should be 71041
    When I get version 3 of subject "imp-comp-ver-auto"
    Then the response status should be 200
    And the response field "id" should be 71042

  Scenario: Import mixed explicit and auto-assigned versions
    When I set the global mode to "IMPORT"
    # Import with explicit version 1
    When I POST "/subjects/imp-comp-ver-mixed/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 71050, "version": 1}
      """
    Then the response status should be 200
    # Import without version — should auto-assign version 2
    When I POST "/subjects/imp-comp-ver-mixed/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 71051}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 1 of subject "imp-comp-ver-mixed"
    Then the response status should be 200
    And the response field "id" should be 71050
    When I get version 2 of subject "imp-comp-ver-mixed"
    Then the response status should be 200
    And the response field "id" should be 71051

  Scenario: Duplicate version for same subject is rejected
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ver-dup/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 71060, "version": 1}
      """
    Then the response status should be 200
    # Same version, different schema, different ID — should fail
    When I POST "/subjects/imp-comp-ver-dup/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 71061, "version": 1}
      """
    Then the response status should be 422
    When I set the global mode to "READWRITE"

  Scenario: Same version number in different subjects succeeds
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ver-sub1/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 71070, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ver-sub2/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 71071, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 1 of subject "imp-comp-ver-sub1"
    Then the response status should be 200
    And the response field "id" should be 71070
    When I get version 1 of subject "imp-comp-ver-sub2"
    Then the response status should be 200
    And the response field "id" should be 71071

  # ==========================================================================
  # ID HANDLING AND SEQUENCING
  # ==========================================================================

  Scenario: Import with ID 1 — lowest valid ID
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-id1/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 1, "version": 1}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    When I set the global mode to "READWRITE"

  Scenario: Import with very large ID
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-id-large/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 100000, "version": 1}
      """
    Then the response status should be 200
    And the response field "id" should be 100000
    When I set the global mode to "READWRITE"

  Scenario: Same schema content with same ID across subjects succeeds
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-idshare-a/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Shared\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}", "id": 72000, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-idshare-b/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Shared\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}", "id": 72000, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-idshare-c/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Shared\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}", "id": 72000, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # All subjects should return the same schema ID
    When I get version 1 of subject "imp-comp-idshare-a"
    Then the response status should be 200
    And the response field "id" should be 72000
    When I get version 1 of subject "imp-comp-idshare-b"
    Then the response status should be 200
    And the response field "id" should be 72000
    When I get version 1 of subject "imp-comp-idshare-c"
    Then the response status should be 200
    And the response field "id" should be 72000

  Scenario: Different schema content with same ID is rejected
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-idconflict-a/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 72010, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-idconflict-b/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 72010, "version": 1}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: Auto-assigned IDs continue above highest imported ID
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-seq-base/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 73000, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # Normal registration should auto-assign IDs above 73000
    When I register a schema under subject "imp-comp-seq-auto1":
      """
      {"type":"record","name":"Auto1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "auto1_id"
    And the stored "auto1_id" should be greater than 73000

  Scenario: Multiple imports then normal registration — IDs continue from max
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-seq-multi1/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 74000, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-seq-multi2/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 74500, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-seq-multi3/versions" with body:
      """
      {"schema": "{\"type\":\"long\"}", "id": 74100, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # Next auto-assigned ID should be > 74500 (the max imported)
    When I register a schema under subject "imp-comp-seq-after":
      """
      {"type":"record","name":"After","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "after_id"
    And the stored "after_id" should be greater than 74500

  # ==========================================================================
  # IDEMPOTENCY
  # ==========================================================================

  Scenario: Import same schema with same ID and same subject is idempotent
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-idemp/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Idemp\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}", "id": 75000, "version": 1}
      """
    Then the response status should be 200
    And the response field "id" should be 75000
    # Re-import identical schema — should succeed (idempotent)
    When I POST "/subjects/imp-comp-idemp/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Idemp\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}", "id": 75000, "version": 1}
      """
    Then the response status should be 200
    And the response field "id" should be 75000
    When I set the global mode to "READWRITE"
    # Should still only have one version
    When I list versions of subject "imp-comp-idemp"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # COMPATIBILITY SKIPPING IN IMPORT MODE
  # ==========================================================================

  Scenario: Import mode skips backward compatibility check
    Given the global compatibility level is "BACKWARD"
    When I set the global mode to "IMPORT"
    # Register a schema with a required field
    When I POST "/subjects/imp-comp-compat-skip/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CS\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"}]}", "id": 76000, "version": 1}
      """
    Then the response status should be 200
    # Import a schema that removes a field — backward incompatible, but import skips check
    When I POST "/subjects/imp-comp-compat-skip/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CS\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 76001, "version": 2}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  Scenario: Compatibility re-enforced after exiting import mode
    Given the global compatibility level is "BACKWARD"
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-compat-re/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CE\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 76010, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # Now backward compatibility is enforced — adding a required field without default is backward-incompatible
    When I register a schema under subject "imp-comp-compat-re":
      """
      {"type":"record","name":"CE","fields":[{"name":"a","type":"string"},{"name":"b","type":"string"}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # SCHEMA TYPE SUPPORT
  # ==========================================================================

  Scenario: Import Avro schema with explicit version
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-avro/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AvroImport\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}", "schemaType": "AVRO", "id": 77000, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 1 of subject "imp-comp-avro"
    Then the response status should be 200
    And the response field "schemaType" should be "AVRO"
    And the response field "version" should be 1
    And the response field "id" should be 77000

  Scenario: Import JSON Schema with explicit version
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-json/versions" with body:
      """
      {"schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"}},\"required\":[\"name\"]}", "schemaType": "JSON", "id": 77010, "version": 3}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 3 of subject "imp-comp-json"
    Then the response status should be 200
    And the response field "schemaType" should be "JSON"
    And the response field "version" should be 3
    And the response field "id" should be 77010

  Scenario: Import Protobuf schema with explicit version
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-proto/versions" with body:
      """
      {"schema": "syntax = \"proto3\";\nmessage ProtoImport {\n  string name = 1;\n}", "schemaType": "PROTOBUF", "id": 77020, "version": 2}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 2 of subject "imp-comp-proto"
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"
    And the response field "version" should be 2
    And the response field "id" should be 77020

  Scenario: Import mixed schema types under different subjects
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-mix-avro/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MixAvro\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 77030, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-mix-json/versions" with body:
      """
      {"schema": "{\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}", "schemaType": "JSON", "id": 77031, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-mix-proto/versions" with body:
      """
      {"schema": "syntax = \"proto3\";\nmessage MixProto {\n  string a = 1;\n}", "schemaType": "PROTOBUF", "id": 77032, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get schema by ID 77030
    Then the response status should be 200
    And the response field "schemaType" should be "AVRO"
    When I get schema by ID 77031
    Then the response status should be 200
    And the response field "schemaType" should be "JSON"
    When I get schema by ID 77032
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"

  # ==========================================================================
  # REAL-WORLD MIGRATION WORKFLOWS
  # ==========================================================================

  Scenario: Full migration workflow — import, verify, switch back, continue evolving
    # Step 1: Enter IMPORT mode
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    # Step 2: Import historical schemas with preserved IDs and versions
    When I POST "/subjects/imp-comp-migrate-users/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}", "id": 78000, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-migrate-users/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":[\"null\",\"string\"],\"default\":null}]}", "id": 78001, "version": 2}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-migrate-orders/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"order_id\",\"type\":\"int\"},{\"name\":\"user_id\",\"type\":\"int\"}]}", "id": 78002, "version": 1}
      """
    Then the response status should be 200
    # Step 3: Verify imported data
    When I list all subjects
    Then the response status should be 200
    When I get version 1 of subject "imp-comp-migrate-users"
    Then the response status should be 200
    And the response field "id" should be 78000
    When I get version 2 of subject "imp-comp-migrate-users"
    Then the response status should be 200
    And the response field "id" should be 78001
    When I get version 1 of subject "imp-comp-migrate-orders"
    Then the response status should be 200
    And the response field "id" should be 78002
    # Step 4: Switch back to READWRITE
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    # Step 5: Continue evolving schemas normally
    When I register a schema under subject "imp-comp-migrate-users":
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null},{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "new_user_id"
    # New ID should be above imported IDs
    And the stored "new_user_id" should be greater than 78002
    # New version should be 3
    When I get version 3 of subject "imp-comp-migrate-users"
    Then the response status should be 200
    And the response should contain "email"

  Scenario: Import into existing subject (using force to switch to IMPORT mode)
    Given the global mode is "READWRITE"
    And subject "imp-comp-existing" has schema:
      """
      {"type":"record","name":"Existing","fields":[{"name":"a","type":"string"}]}
      """
    # Switch to IMPORT mode requires force when schemas exist
    When I PUT "/mode?force=true" with body:
      """
      {"mode": "IMPORT"}
      """
    Then the response status should be 200
    # Import additional version with explicit ID and version
    When I POST "/subjects/imp-comp-existing/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Existing\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":[\"null\",\"string\"],\"default\":null}]}", "id": 79000, "version": 2}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I list versions of subject "imp-comp-existing"
    Then the response status should be 200
    And the response should be an array of length 2

  Scenario: Import preserves subject listing after import
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-list-a/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 79100, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-list-b/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 79101, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-list-c/versions" with body:
      """
      {"schema": "{\"type\":\"long\"}", "id": 79102, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I list all subjects
    Then the response status should be 200
    And the response should contain "imp-comp-list-a"
    And the response should contain "imp-comp-list-b"
    And the response should contain "imp-comp-list-c"

  # ==========================================================================
  # RETRIEVAL AFTER IMPORT
  # ==========================================================================

  Scenario: Schema retrievable by ID after import
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ret-id/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"RetById\",\"fields\":[{\"name\":\"val\",\"type\":\"string\"}]}", "id": 80000, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get schema by ID 80000
    Then the response status should be 200
    And the response should contain "RetById"

  Scenario: Schema subjects endpoint works for imported schemas
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ret-subj1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"RetSubj\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 80010, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ret-subj2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"RetSubj\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 80010, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get the subjects for schema ID 80010
    Then the response status should be 200
    And the response should be an array of length 2

  Scenario: Latest version resolves correctly after out-of-order import
    When I set the global mode to "IMPORT"
    # Import version 5 first, then 3, then 1
    When I POST "/subjects/imp-comp-ret-latest/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Latest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"},{\"name\":\"c\",\"type\":\"string\"},{\"name\":\"d\",\"type\":\"string\"},{\"name\":\"e\",\"type\":\"string\"}]}", "id": 80022, "version": 5}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ret-latest/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Latest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"},{\"name\":\"c\",\"type\":\"string\"}]}", "id": 80021, "version": 3}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-ret-latest/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Latest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 80020, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # "latest" should resolve to version 5
    When I get the latest version of subject "imp-comp-ret-latest"
    Then the response status should be 200
    And the response field "version" should be 5
    And the response field "id" should be 80022

  Scenario: Lookup schema after import finds correct version
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-ret-lookup/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LookupFind\",\"fields\":[{\"name\":\"key\",\"type\":\"string\"}]}", "id": 80030, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I lookup schema in subject "imp-comp-ret-lookup":
      """
      {"type":"record","name":"LookupFind","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should be 80030
    And the response field "version" should be 1

  # ==========================================================================
  # DELETION AFTER IMPORT
  # ==========================================================================

  Scenario: Soft-delete imported subject
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-del-soft/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 81000, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-del-soft/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 81001, "version": 2}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I delete subject "imp-comp-del-soft"
    Then the response status should be 200
    # Subject should not appear in normal listing
    When I list all subjects
    Then the response status should be 200
    And the response body should not contain "imp-comp-del-soft"
    # But should appear with deleted=true
    When I GET "/subjects?deleted=true"
    Then the response status should be 200
    And the response should contain "imp-comp-del-soft"

  Scenario: Delete specific imported version
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-del-ver/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 81010, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-del-ver/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 81011, "version": 2}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-del-ver/versions" with body:
      """
      {"schema": "{\"type\":\"long\"}", "id": 81012, "version": 3}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # Delete version 2
    When I delete version 2 of subject "imp-comp-del-ver"
    Then the response status should be 200
    # Versions 1 and 3 remain
    When I list versions of subject "imp-comp-del-ver"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # IMPORT AFTER SOFT-DELETE
  # ==========================================================================

  Scenario: Import into soft-deleted subject re-creates it
    Given the global mode is "READWRITE"
    And subject "imp-comp-reimport" has schema:
      """
      {"type":"record","name":"Reimport","fields":[{"name":"a","type":"string"}]}
      """
    When I delete subject "imp-comp-reimport"
    Then the response status should be 200
    When I set the global mode to "IMPORT"
    # Import a new version — should create version 2 (v1 is soft-deleted)
    When I POST "/subjects/imp-comp-reimport/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Reimport\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":[\"null\",\"string\"],\"default\":null}]}", "id": 82000, "version": 2}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 2 of subject "imp-comp-reimport"
    Then the response status should be 200
    And the response field "id" should be 82000

  # ==========================================================================
  # REFERENCES DURING IMPORT
  # ==========================================================================

  Scenario: Import schema with Avro references
    When I set the global mode to "IMPORT"
    # Import the referenced schema first
    When I POST "/subjects/imp-comp-ref-address/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.imp\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}", "id": 83000, "version": 1}
      """
    Then the response status should be 200
    # Import schema with reference
    When I POST "/subjects/imp-comp-ref-person/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.imp\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"address\",\"type\":\"com.imp.Address\"}]}", "id": 83001, "version": 1, "references": [{"name": "com.imp.Address", "subject": "imp-comp-ref-address", "version": 1}]}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get schema by ID 83001
    Then the response status should be 200
    And the response should contain "Person"

  Scenario: Import referenced schema out of order — reference first, then base
    When I set the global mode to "IMPORT"
    # Import the base schema first (even though it will be referenced later)
    When I POST "/subjects/imp-comp-ref-ooo-base/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Base\",\"namespace\":\"com.ooo\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}", "id": 83010, "version": 1}
      """
    Then the response status should be 200
    # Import the referencing schema
    When I POST "/subjects/imp-comp-ref-ooo-child/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Child\",\"namespace\":\"com.ooo\",\"fields\":[{\"name\":\"base\",\"type\":\"com.ooo.Base\"}]}", "id": 83011, "version": 1, "references": [{"name": "com.ooo.Base", "subject": "imp-comp-ref-ooo-base", "version": 1}]}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 1 of subject "imp-comp-ref-ooo-child"
    Then the response status should be 200
    And the response should contain "Child"

  # ==========================================================================
  # MULTI-VERSION EVOLUTION DURING IMPORT
  # ==========================================================================

  Scenario: Import full evolution history — 5 versions of a schema
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-evolve/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Ev\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}", "id": 84000, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-evolve/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Ev\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":[\"null\",\"string\"],\"default\":null}]}", "id": 84001, "version": 2}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-evolve/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Ev\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}", "id": 84002, "version": 3}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-evolve/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Ev\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"phone\",\"type\":[\"null\",\"string\"],\"default\":null}]}", "id": 84003, "version": 4}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-evolve/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Ev\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"phone\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"active\",\"type\":\"boolean\",\"default\":true}]}", "id": 84004, "version": 5}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # Verify all 5 versions
    When I list versions of subject "imp-comp-evolve"
    Then the response status should be 200
    And the response should be an array of length 5
    When I get version 1 of subject "imp-comp-evolve"
    Then the response status should be 200
    And the response field "id" should be 84000
    When I get version 5 of subject "imp-comp-evolve"
    Then the response status should be 200
    And the response field "id" should be 84004
    And the response should contain "active"
    # Latest should be version 5
    When I get the latest version of subject "imp-comp-evolve"
    Then the response status should be 200
    And the response field "version" should be 5

  Scenario: Import then continue normal evolution — version continuity
    When I set the global mode to "IMPORT"
    When I POST "/subjects/imp-comp-cont/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Cont\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 85000, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/imp-comp-cont/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Cont\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":[\"null\",\"string\"],\"default\":null}]}", "id": 85001, "version": 2}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # Register a new version normally — should be version 3
    When I register a schema under subject "imp-comp-cont":
      """
      {"type":"record","name":"Cont","fields":[{"name":"a","type":"string"},{"name":"b","type":["null","string"],"default":null},{"name":"c","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200
    When I list versions of subject "imp-comp-cont"
    Then the response status should be 200
    And the response should be an array of length 3
    When I get version 3 of subject "imp-comp-cont"
    Then the response status should be 200
    And the response should contain "\"c\""
