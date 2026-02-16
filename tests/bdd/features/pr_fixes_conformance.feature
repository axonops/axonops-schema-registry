@functional
Feature: PR Fixes Conformance
  Tests covering fixes from the Confluent compatibility review, including
  IMPORT mode enforcement, schema ID stability, sequence rewind protection,
  deletion edge cases, and JSON Schema external reference compatibility.

  # ==========================================================================
  # ISSUE 3: Explicit ID rejected outside IMPORT mode
  # Confluent behavior:
  #   - Registering a schema with an explicit "id" field in the request body
  #     is only allowed when the global or subject mode is IMPORT.
  #   - In READWRITE mode, an explicit ID must be rejected with error 42205.
  # ==========================================================================

  Scenario: Explicit ID in READWRITE mode is rejected with 42205
    Given the global mode is "READWRITE"
    When I POST "/subjects/fix3-rw-explicit/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ExplicitRW\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12345}
      """
    Then the response status should be 422
    And the response should have error code 42205

  Scenario: Explicit ID in IMPORT mode succeeds
    Given the global mode is "IMPORT"
    When I POST "/subjects/fix3-import-explicit/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ExplicitImp\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12345}
      """
    Then the response status should be 200
    And the response field "id" should be 12345
    When I set the global mode to "READWRITE"

  Scenario: Per-subject IMPORT mode allows explicit ID
    Given the global mode is "READWRITE"
    When I set the mode for subject "fix3-subj-import" to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/fix3-subj-import/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SubjImp\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12346}
      """
    Then the response status should be 200
    And the response field "id" should be 12346

  # ==========================================================================
  # ISSUE 4: Bulk import requires IMPORT mode
  # Confluent behavior:
  #   - The /import/schemas endpoint (bulk import) should only work when
  #     the global mode is IMPORT. Otherwise return 42205.
  # ==========================================================================

  @axonops-only
  Scenario: Bulk import rejected outside IMPORT mode
    Given the global mode is "READWRITE"
    When I import a schema with ID 20000 under subject "fix4-bulk-rw" version 1:
      """
      {"type":"record","name":"BulkRW","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205

  @axonops-only
  Scenario: Bulk import succeeds in IMPORT mode
    Given the global mode is "IMPORT"
    When I import a schema with ID 20000 under subject "fix4-bulk-import" version 1:
      """
      {"type":"record","name":"BulkImp","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I get schema by ID 20000
    Then the response status should be 200
    And the response should contain "BulkImp"
    When I set the global mode to "READWRITE"

  # ==========================================================================
  # ISSUE 7: ID sequence rewind protection
  # After importing schemas with high IDs, new auto-assigned IDs must be
  # strictly greater than the highest imported ID. Otherwise Kafka wire
  # format messages could collide with future registrations.
  # ==========================================================================

  Scenario: Auto-assigned IDs after import are strictly greater than imported IDs
    Given the global mode is "IMPORT"
    When I POST "/subjects/fix7-seq-import/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SeqImport\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 50000}
      """
    Then the response status should be 200
    And the response field "id" should be 50000
    When I set the global mode to "READWRITE"
    When I register a schema under subject "fix7-seq-new":
      """
      {"type":"record","name":"SeqNew","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "new_id"
    Then the stored "new_id" should be greater than 50000

  # ==========================================================================
  # ISSUE 8: Version-level soft-delete + re-register
  # After soft-deleting a specific version, re-registering the same content
  # should create a new version (not reuse the deleted slot) and the schema
  # ID should remain the same (content-addressed deduplication).
  # ==========================================================================

  Scenario: Re-register after version soft-delete creates new version with same ID
    Given the global compatibility level is "NONE"
    When I register a schema under subject "fix8-ver-reregister":
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "v1_id"
    When I register a schema under subject "fix8-ver-reregister":
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "v2_id"
    # Soft-delete version 2
    When I delete version 2 of subject "fix8-ver-reregister"
    Then the response status should be 200
    # Re-register same content as v2 — should get version 3 with same schema ID
    When I register a schema under subject "fix8-ver-reregister":
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "v2_id"
    When I get the latest version of subject "fix8-ver-reregister"
    Then the response status should be 200
    And the response field "version" should be 3

  Scenario: Re-register different content after version soft-delete
    Given the global compatibility level is "NONE"
    When I register a schema under subject "fix8-diff-reregister":
      """
      {"type":"record","name":"D1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "fix8-diff-reregister":
      """
      {"type":"record","name":"D2","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 200
    # Soft-delete version 2
    When I delete version 2 of subject "fix8-diff-reregister"
    Then the response status should be 200
    # Register completely different content
    When I register a schema under subject "fix8-diff-reregister":
      """
      {"type":"record","name":"D3","fields":[{"name":"c","type":"int"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "fix8-diff-reregister"
    Then the response status should be 200
    And the response field "version" should be 3
    And the response should contain "D3"

  # ==========================================================================
  # ISSUE 9: GET latest?deleted=true
  # When all versions of a subject are soft-deleted, GET versions/latest
  # returns 404. But GET versions/latest?deleted=true should return the
  # highest-versioned soft-deleted schema.
  # ==========================================================================

  @axonops-only
  Scenario: GET latest returns 404 after all versions soft-deleted
    Given the global compatibility level is "NONE"
    And subject "fix9-latest-del" has schema:
      """
      {"type":"record","name":"LatDel1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "fix9-latest-del" has schema:
      """
      {"type":"record","name":"LatDel2","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/fix9-latest-del"
    Then the response status should be 200
    # Without deleted=true → 404
    When I GET "/subjects/fix9-latest-del/versions/latest"
    Then the response status should be 404
    # With deleted=true → returns the highest-versioned soft-deleted schema
    When I GET "/subjects/fix9-latest-del/versions/latest?deleted=true"
    Then the response status should be 200
    And the response field "version" should be 2
    And the response should contain "LatDel2"

  Scenario: GET specific version with deleted=true after soft-delete
    Given subject "fix9-specific-del" has schema:
      """
      {"type":"record","name":"SpecDel","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/fix9-specific-del"
    Then the response status should be 200
    When I GET "/subjects/fix9-specific-del/versions/1"
    Then the response status should be 404
    When I GET "/subjects/fix9-specific-del/versions/1?deleted=true"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response should contain "SpecDel"

  Scenario: GET specific deleted version with deleted=true while active versions exist
    Given the global compatibility level is "NONE"
    And subject "fix9-partial-del" has schema:
      """
      {"type":"record","name":"PD1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "fix9-partial-del" has schema:
      """
      {"type":"record","name":"PD2","fields":[{"name":"b","type":"string"}]}
      """
    # Soft-delete only version 2
    When I DELETE "/subjects/fix9-partial-del/versions/2"
    Then the response status should be 200
    # latest without deleted=true → version 1 (only active version)
    When I get the latest version of subject "fix9-partial-del"
    Then the response status should be 200
    And the response field "version" should be 1
    # Specific deleted version with deleted=true → still accessible
    When I GET "/subjects/fix9-partial-del/versions/2?deleted=true"
    Then the response status should be 200
    And the response field "version" should be 2
    And the response should contain "PD2"

  # ==========================================================================
  # ISSUES 1 & 2: Schema ID stability after permanent delete
  # When the same schema is registered under multiple subjects, permanently
  # deleting one subject must NOT change the global schema ID. This is
  # critical for Kafka wire format compatibility (5-byte prefix encodes ID).
  # ==========================================================================

  Scenario: Schema ID stable across subjects after permanent delete of first registration
    Given the global compatibility level is "NONE"
    When I register a schema under subject "fix12-stab-a":
      """
      {"type":"record","name":"StableSchema","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "stable_id"
    # Register same schema in second subject — should get same ID
    When I register a schema under subject "fix12-stab-b":
      """
      {"type":"record","name":"StableSchema","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "stable_id"
    # Permanently delete first subject
    When I DELETE "/subjects/fix12-stab-a"
    Then the response status should be 200
    When I DELETE "/subjects/fix12-stab-a?permanent=true"
    Then the response status should be 200
    # Schema should still be accessible by the same global ID
    When I get schema by ID {{stable_id}}
    Then the response status should be 200
    And the response should contain "StableSchema"
    # ID in second subject should still match
    When I get version 1 of subject "fix12-stab-b"
    Then the response status should be 200
    And the response field "id" should equal stored "stable_id"

  Scenario: Schema ID returned by subjects endpoint after permanent delete
    Given the global compatibility level is "NONE"
    When I register a schema under subject "fix12-subj-a":
      """
      {"type":"record","name":"SubjSchema","fields":[{"name":"val","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "subj_id"
    When I register a schema under subject "fix12-subj-b":
      """
      {"type":"record","name":"SubjSchema","fields":[{"name":"val","type":"string"}]}
      """
    Then the response status should be 200
    # Permanently delete first subject
    When I DELETE "/subjects/fix12-subj-a"
    Then the response status should be 200
    When I DELETE "/subjects/fix12-subj-a?permanent=true"
    Then the response status should be 200
    # GET /schemas/ids/{id}/subjects should still return the remaining subject
    When I get the subjects for schema ID {{subj_id}}
    Then the response status should be 200
    And the response array should contain "fix12-subj-b"

  Scenario: References survive permanent delete of one registration
    Given the global compatibility level is "NONE"
    # Register base schema in two subjects
    When I register a schema under subject "fix12-ref-base-a":
      """
      {"type":"record","name":"RefBase","namespace":"com.fix12","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "base_id"
    When I register a schema under subject "fix12-ref-base-b":
      """
      {"type":"record","name":"RefBase","namespace":"com.fix12","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "base_id"
    # Register a schema that references the base
    When I register a schema under subject "fix12-ref-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.fix12\",\"fields\":[{\"name\":\"base\",\"type\":\"com.fix12.RefBase\"}]}",
        "references": [
          {"name": "com.fix12.RefBase", "subject": "fix12-ref-base-a", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Permanently delete one of the base registrations
    When I DELETE "/subjects/fix12-ref-base-b"
    Then the response status should be 200
    When I DELETE "/subjects/fix12-ref-base-b?permanent=true"
    Then the response status should be 200
    # The consumer schema should still be retrievable and the base schema accessible by ID
    When I get schema by ID {{base_id}}
    Then the response status should be 200
    And the response should contain "RefBase"

  # ==========================================================================
  # ISSUE 10: JSON Schema compatibility with external $ref
  # The compatibility checker must resolve external $ref references when
  # checking BACKWARD/FORWARD/FULL compatibility of JSON Schemas that
  # reference other schemas via the references mechanism.
  # ==========================================================================

  Scenario: JSON Schema BACKWARD compatibility with external reference (closed model)
    Given the global compatibility level is "BACKWARD"
    # Register the referenced schema
    And subject "fix10-ref-address" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}},"required":["street","city"],"additionalProperties":false}
      """
    # Register v1 with a reference to the address schema (closed content model)
    When I register a "JSON" schema under subject "fix10-ref-person" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"address\":{\"$ref\":\"address.json\"}},\"required\":[\"name\"],\"additionalProperties\":false}",
        "references": [
          {"name": "address.json", "subject": "fix10-ref-address", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Check compatibility of a BACKWARD-compatible evolution (adding optional field to closed model)
    When I check compatibility of "JSON" schema with reference "address.json" from subject "fix10-ref-address" version 1 against subject "fix10-ref-person":
      """
      {"type":"object","properties":{"name":{"type":"string"},"address":{"$ref":"address.json"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema compatibility check resolves external $ref correctly
    Given the global compatibility level is "BACKWARD"
    # Register the referenced schema
    And subject "fix10-compat-addr" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"}},"required":["street"],"additionalProperties":false}
      """
    # Register v1 that uses the external reference
    When I register a "JSON" schema under subject "fix10-compat-person" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"addr\":{\"$ref\":\"addr.json\"}},\"required\":[\"name\"],\"additionalProperties\":false}",
        "references": [
          {"name": "addr.json", "subject": "fix10-compat-addr", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Identical schema with same reference → must be compatible
    When I check compatibility of "JSON" schema with reference "addr.json" from subject "fix10-compat-addr" version 1 against subject "fix10-compat-person":
      """
      {"type":"object","properties":{"name":{"type":"string"},"addr":{"$ref":"addr.json"}},"required":["name"],"additionalProperties":false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema BACKWARD incompatible change with external reference
    Given the global compatibility level is "BACKWARD"
    And subject "fix10-incompat-addr" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"}},"required":["street"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "fix10-incompat-person" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"addr\":{\"$ref\":\"addr.json\"}},\"required\":[\"name\"],\"additionalProperties\":false}",
        "references": [
          {"name": "addr.json", "subject": "fix10-incompat-addr", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Try to register v2 that makes addr required (BACKWARD incompatible - new required field)
    When I check compatibility of "JSON" schema with reference "addr.json" from subject "fix10-incompat-addr" version 1 against subject "fix10-incompat-person":
      """
      {"type":"object","properties":{"name":{"type":"string"},"addr":{"$ref":"addr.json"}},"required":["name","addr"],"additionalProperties":false}
      """
    Then the compatibility check should be incompatible
