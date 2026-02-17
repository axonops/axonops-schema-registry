@functional
Feature: Schema ID Stability and Content Validation
  Verify schema IDs are stable across subjects (content-addressed deduplication)
  and that retrieved schemas are valid and match what was registered.

  # ==========================================================================
  # SCHEMA ID STABILITY ACROSS SUBJECTS
  # ==========================================================================

  Scenario: Same Avro schema in two subjects gets same ID verified via GET
    When I register a schema under subject "idstab-avro-a":
      """
      {"type":"record","name":"Stable","fields":[{"name":"id","type":"string"},{"name":"val","type":"double"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_a"
    When I register a schema under subject "idstab-avro-b":
      """
      {"type":"record","name":"Stable","fields":[{"name":"id","type":"string"},{"name":"val","type":"double"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_b"
    When I get schema by ID {{id_a}}
    Then the response status should be 200
    And the response should contain "Stable"

  Scenario: Different schema content gets different IDs
    When I register a schema under subject "idstab-diff-a":
      """
      {"type":"record","name":"Alpha","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_alpha"
    When I register a schema under subject "idstab-diff-b":
      """
      {"type":"record","name":"Beta","fields":[{"name":"y","type":"long"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_beta"

  # ==========================================================================
  # RETRIEVED SCHEMA CONTENT VALIDATION
  # ==========================================================================

  Scenario: Retrieved Avro schema is valid parseable JSON
    Given subject "idstab-valid-avro" has schema:
      """
      {"type":"record","name":"ValidAvro","fields":[{"name":"id","type":"string"},{"name":"count","type":"int"}]}
      """
    When I get version 1 of subject "idstab-valid-avro"
    Then the response status should be 200
    And the response should have field "schema"
    And the response should be valid JSON

  Scenario: Retrieved schema by ID matches what was registered
    Given subject "idstab-match" has schema:
      """
      {"type":"record","name":"MatchMe","fields":[{"name":"key","type":"string"}]}
      """
    And I store the response field "id" as "match_id"
    When I get schema by ID {{match_id}}
    Then the response status should be 200
    And the response should contain "MatchMe"
    And the response should contain "key"

  Scenario: Protobuf schema retrieved by version contains expected content
    Given subject "idstab-proto" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message ProtoStab {
        string key = 1;
        int32 value = 2;
      }
      """
    When I get version 1 of subject "idstab-proto"
    Then the response status should be 200
    And the response should contain "ProtoStab"

  Scenario: JSON Schema retrieved by version contains expected content
    Given subject "idstab-jsonsch" has "JSON" schema:
      """
      {"type":"object","properties":{"key":{"type":"string"},"value":{"type":"integer"}},"required":["key"]}
      """
    When I get version 1 of subject "idstab-jsonsch"
    Then the response status should be 200
    And the response should contain "key"

  # ==========================================================================
  # Auto-assigned IDs after import must be strictly greater than the highest
  # imported ID. Otherwise Kafka wire format messages could collide.
  # ==========================================================================

  Scenario: Auto-assigned IDs after import are strictly greater than imported IDs
    Given the global mode is "IMPORT"
    When I POST "/subjects/idstab-seq-import/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SeqImport\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 50000}
      """
    Then the response status should be 200
    And the response field "id" should be 50000
    When I set the global mode to "READWRITE"
    When I register a schema under subject "idstab-seq-new":
      """
      {"type":"record","name":"SeqNew","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "new_id"
    Then the stored "new_id" should be greater than 50000

  # ==========================================================================
  # Schema ID stability after permanent delete — when the same schema is
  # registered under multiple subjects, permanently deleting one subject
  # must NOT change the global schema ID.
  # ==========================================================================

  Scenario: Schema ID stable across subjects after permanent delete of first registration
    Given the global compatibility level is "NONE"
    When I register a schema under subject "idstab-perm-a":
      """
      {"type":"record","name":"StableSchema","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "stable_id"
    When I register a schema under subject "idstab-perm-b":
      """
      {"type":"record","name":"StableSchema","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "stable_id"
    When I DELETE "/subjects/idstab-perm-a"
    Then the response status should be 200
    When I DELETE "/subjects/idstab-perm-a?permanent=true"
    Then the response status should be 200
    When I get schema by ID {{stable_id}}
    Then the response status should be 200
    And the response should contain "StableSchema"
    When I get version 1 of subject "idstab-perm-b"
    Then the response status should be 200
    And the response field "id" should equal stored "stable_id"

  Scenario: Schema ID returned by subjects endpoint after permanent delete
    Given the global compatibility level is "NONE"
    When I register a schema under subject "idstab-subj-a":
      """
      {"type":"record","name":"SubjSchema","fields":[{"name":"val","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "subj_id"
    When I register a schema under subject "idstab-subj-b":
      """
      {"type":"record","name":"SubjSchema","fields":[{"name":"val","type":"string"}]}
      """
    Then the response status should be 200
    When I DELETE "/subjects/idstab-subj-a"
    Then the response status should be 200
    When I DELETE "/subjects/idstab-subj-a?permanent=true"
    Then the response status should be 200
    When I get the subjects for schema ID {{subj_id}}
    Then the response status should be 200
    And the response array should contain "idstab-subj-b"

  Scenario: References survive permanent delete of non-referenced registration
    Given the global compatibility level is "NONE"
    When I register a schema under subject "idstab-ref-base-a":
      """
      {"type":"record","name":"RefBase","namespace":"com.idstab","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "base_id"
    When I register a schema under subject "idstab-ref-base-b":
      """
      {"type":"record","name":"RefBase","namespace":"com.idstab","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "base_id"
    When I register a schema under subject "idstab-ref-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.idstab\",\"fields\":[{\"name\":\"base\",\"type\":\"com.idstab.RefBase\"}]}",
        "references": [
          {"name": "com.idstab.RefBase", "subject": "idstab-ref-base-a", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I DELETE "/subjects/idstab-ref-base-b"
    Then the response status should be 200
    When I DELETE "/subjects/idstab-ref-base-b?permanent=true"
    Then the response status should be 200
    When I get schema by ID {{base_id}}
    Then the response status should be 200
    And the response should contain "RefBase"

  Scenario: References survive permanent delete of canonical first-registered subject
    # This exercises the exact regression where a SQL JOIN against the
    # first-registered subject row would break after its permanent deletion.
    Given the global compatibility level is "NONE"
    When I register a schema under subject "idstab-ref-canon-a":
      """
      {"type":"record","name":"CanonBase","namespace":"com.idstab","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "canon_id"
    When I register a schema under subject "idstab-ref-canon-b":
      """
      {"type":"record","name":"CanonBase","namespace":"com.idstab","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "canon_id"
    When I register a schema under subject "idstab-ref-canon-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CanonConsumer\",\"namespace\":\"com.idstab\",\"fields\":[{\"name\":\"base\",\"type\":\"com.idstab.CanonBase\"}]}",
        "references": [
          {"name": "com.idstab.CanonBase", "subject": "idstab-ref-canon-a", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "consumer_id"
    # Permanently delete the canonical first-registered subject (the one the reference points to)
    When I DELETE "/subjects/idstab-ref-canon-a"
    Then the response status should be 200
    When I DELETE "/subjects/idstab-ref-canon-a?permanent=true"
    Then the response status should be 200
    # Schema content must still be retrievable by global ID
    When I get schema by ID {{canon_id}}
    Then the response status should be 200
    And the response should contain "CanonBase"
    # The surviving subject must still hold the schema
    When I get version 1 of subject "idstab-ref-canon-b"
    Then the response status should be 200
    And the response field "id" should equal stored "canon_id"
    # Referencedby must still report the consumer via the surviving registration
    When I get the referenced by for subject "idstab-ref-canon-b" version 1
    Then the response status should be 200
    And the response array should contain stored integer "consumer_id"
    # The surviving referenced subject must still be delete-protected
    When I DELETE "/subjects/idstab-ref-canon-b"
    Then the response status should be 422
