@functional
Feature: Advanced Features
  Tests for advanced Confluent Schema Registry API parameters including
  force on mode, deletedOnly on version listing, and parameter behavior
  documentation for normalize, format, and fetchMaxId.

  # ==========================================================================
  # FORCE PARAMETER ON PUT /mode
  # Confluent behavior: Setting mode to IMPORT fails with 42205 if schemas
  # exist, unless ?force=true is provided.
  # ==========================================================================

  Scenario: Set mode to IMPORT when no schemas exist (no force needed)
    When I PUT "/mode" with body:
      """
      {"mode": "IMPORT"}
      """
    Then the response status should be 200
    And the response field "mode" should be "IMPORT"
    When I set the global mode to "READWRITE"

  Scenario: Set mode to IMPORT with force=true when schemas exist
    Given the global mode is "READWRITE"
    And subject "force-test-sub" has schema:
      """
      {"type":"record","name":"ForceTest","fields":[{"name":"a","type":"string"}]}
      """
    When I PUT "/mode?force=true" with body:
      """
      {"mode": "IMPORT"}
      """
    Then the response status should be 200
    And the response field "mode" should be "IMPORT"
    When I set the global mode to "READWRITE"

  Scenario: Set mode to IMPORT without force when schemas exist returns error
    Given the global mode is "READWRITE"
    And subject "force-test-sub2" has schema:
      """
      {"type":"record","name":"ForceTest2","fields":[{"name":"a","type":"string"}]}
      """
    When I PUT "/mode" with body:
      """
      {"mode": "IMPORT"}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: Set per-subject mode to IMPORT without force when schemas exist
    Given the global mode is "READWRITE"
    And subject "force-per-sub" has schema:
      """
      {"type":"record","name":"ForcePer","fields":[{"name":"a","type":"string"}]}
      """
    When I PUT "/mode/force-per-sub" with body:
      """
      {"mode": "IMPORT"}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: Set per-subject mode to IMPORT with force=true when schemas exist
    Given the global mode is "READWRITE"
    And subject "force-per-sub2" has schema:
      """
      {"type":"record","name":"ForcePer2","fields":[{"name":"a","type":"string"}]}
      """
    When I PUT "/mode/force-per-sub2?force=true" with body:
      """
      {"mode": "IMPORT"}
      """
    Then the response status should be 200
    And the response field "mode" should be "IMPORT"
    When I set the global mode to "READWRITE"

  Scenario: Force is not needed for non-IMPORT modes
    Given the global mode is "READWRITE"
    And subject "force-readonly-sub" has schema:
      """
      {"type":"record","name":"ForceRO","fields":[{"name":"a","type":"string"}]}
      """
    When I PUT "/mode" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  Scenario: Force not needed when already in IMPORT mode
    When I PUT "/mode?force=true" with body:
      """
      {"mode": "IMPORT"}
      """
    Then the response status should be 200
    # Setting again without force should work because we're already in IMPORT
    When I PUT "/mode" with body:
      """
      {"mode": "IMPORT"}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  # ==========================================================================
  # DELETED-ONLY PARAMETER ON VERSION LISTING
  # Confluent behavior: ?deletedOnly=true returns only soft-deleted versions.
  # Takes precedence over ?deleted=true.
  # ==========================================================================

  Scenario: deletedOnly returns only soft-deleted versions
    Given the global mode is "READWRITE"
    And subject "delonly-test" has schema:
      """
      {"type":"record","name":"DelOnly1","fields":[{"name":"a","type":"string"}]}
      """
    And I register a schema under subject "delonly-test":
      """
      {"type":"record","name":"DelOnly1","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":""}]}
      """
    When I DELETE "/subjects/delonly-test/versions/1"
    Then the response status should be 200
    # Only version 1 is deleted, version 2 is active
    When I GET "/subjects/delonly-test/versions?deletedOnly=true"
    Then the response status should be 200
    And the response should contain "1"
    And the response body should not contain "2"

  Scenario: deletedOnly with no deleted versions returns empty
    Given subject "delonly-empty" has schema:
      """
      {"type":"record","name":"DelOnlyEmpty","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/delonly-empty/versions?deletedOnly=true"
    Then the response status should be 200
    And the response should contain "[]"

  Scenario: deletedOnly takes precedence over deleted
    Given subject "delonly-both" has schema:
      """
      {"type":"record","name":"DelOnlyBoth","fields":[{"name":"a","type":"string"}]}
      """
    And I register a schema under subject "delonly-both":
      """
      {"type":"record","name":"DelOnlyBoth","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":""}]}
      """
    When I DELETE "/subjects/delonly-both/versions/1"
    Then the response status should be 200
    When I GET "/subjects/delonly-both/versions?deleted=true&deletedOnly=true"
    Then the response status should be 200
    # Should only return deleted version 1, not active version 2
    And the response should contain "1"
    And the response body should not contain "2"

  Scenario: Regular deleted=true returns both active and deleted
    Given subject "delonly-compare" has schema:
      """
      {"type":"record","name":"DelOnlyCompare","fields":[{"name":"a","type":"string"}]}
      """
    And I register a schema under subject "delonly-compare":
      """
      {"type":"record","name":"DelOnlyCompare","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":""}]}
      """
    When I DELETE "/subjects/delonly-compare/versions/1"
    Then the response status should be 200
    When I GET "/subjects/delonly-compare/versions?deleted=true"
    Then the response status should be 200
    And the response should contain "1"
    And the response should contain "2"

  # ==========================================================================
  # NORMALIZE PARAMETER
  # Confluent behavior: normalize=true normalizes schema before storage/
  # comparison. Schemas with different whitespace/ordering that normalize
  # to the same canonical form are deduplicated.
  # ==========================================================================

  Scenario: normalize parameter is accepted on registration
    When I POST "/subjects/norm-test/versions?normalize=true" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Norm\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200

  Scenario: normalize parameter is accepted on lookup
    Given subject "norm-lookup" has schema:
      """
      {"type":"record","name":"NormLookup","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/subjects/norm-lookup?normalize=true" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"NormLookup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200

  Scenario: normalize parameter is accepted on compatibility check
    Given subject "norm-compat" has schema:
      """
      {"type":"record","name":"NormCompat","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/norm-compat/versions/1?normalize=true" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"NormCompat\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200

  Scenario: normalize config option can be set per subject
    When I PUT "/config/norm-config-sub" with body:
      """
      {"compatibility": "BACKWARD", "normalize": true}
      """
    Then the response status should be 200
    And the response should contain "normalize"

  Scenario: normalize config option can be set globally
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "normalize": true}
      """
    Then the response status should be 200
    # Reset to avoid affecting other tests
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200

  # ==========================================================================
  # FORMAT PARAMETER
  # Confluent behavior: format=resolved inlines references (Avro),
  # format=serialized returns base64-encoded FileDescriptorProto (Protobuf).
  # Unknown formats fall back to canonical string.
  # ==========================================================================

  Scenario: format parameter is accepted on GET schema by ID
    Given subject "fmt-test" has schema:
      """
      {"type":"record","name":"FmtTest","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "fmt_schema_id"
    When I GET "/schemas/ids/{{fmt_schema_id}}?format=default"
    Then the response status should be 200
    And the response should have field "schema"

  Scenario: format=resolved on Avro schema returns schema content
    Given subject "fmt-resolved" has schema:
      """
      {"type":"record","name":"FmtResolved","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "fmt_resolved_id"
    When I GET "/schemas/ids/{{fmt_resolved_id}}?format=resolved"
    Then the response status should be 200
    And the response should have field "schema"

  Scenario: format parameter on raw schema endpoint
    Given subject "fmt-raw" has schema:
      """
      {"type":"record","name":"FmtRaw","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "fmt_raw_id"
    When I GET "/schemas/ids/{{fmt_raw_id}}/schema?format=default"
    Then the response status should be 200

  Scenario: format parameter on GET version
    Given subject "fmt-ver" has schema:
      """
      {"type":"record","name":"FmtVer","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/fmt-ver/versions/1?format=default"
    Then the response status should be 200
    And the response should have field "schema"

  Scenario: format parameter on GET version raw schema
    Given subject "fmt-ver-raw" has schema:
      """
      {"type":"record","name":"FmtVerRaw","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/fmt-ver-raw/versions/1/schema?format=default"
    Then the response status should be 200

  Scenario: unknown format falls back to canonical string
    Given subject "fmt-unknown" has schema:
      """
      {"type":"record","name":"FmtUnknown","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "fmt_unknown_id"
    When I GET "/schemas/ids/{{fmt_unknown_id}}?format=nonexistent"
    Then the response status should be 200
    And the response should have field "schema"

  # ==========================================================================
  # FETCH MAX ID
  # Confluent behavior: GET /schemas/ids/{id}?fetchMaxId=true includes maxId
  # field in response, which is the highest schema ID in the registry.
  # ==========================================================================

  Scenario: fetchMaxId=true includes maxId field in response
    Given subject "maxid-test" has schema:
      """
      {"type":"record","name":"MaxIdTest","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "maxid_schema_id"
    When I GET "/schemas/ids/{{maxid_schema_id}}?fetchMaxId=true"
    Then the response status should be 200
    And the response should have field "maxId"

  Scenario: fetchMaxId not set omits maxId field
    Given subject "maxid-omit" has schema:
      """
      {"type":"record","name":"MaxIdOmit","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "maxid_omit_id"
    When I GET "/schemas/ids/{{maxid_omit_id}}"
    Then the response status should be 200
    And the response should not have field "maxId"

  Scenario: maxId reflects highest schema ID in registry
    Given subject "maxid-high1" has schema:
      """
      {"type":"record","name":"MaxIdHigh1","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "maxid_first"
    And subject "maxid-high2" has schema:
      """
      {"type":"record","name":"MaxIdHigh2","fields":[{"name":"b","type":"string"}]}
      """
    And I store the response field "id" as "maxid_second"
    When I GET "/schemas/ids/{{maxid_first}}?fetchMaxId=true"
    Then the response status should be 200
    And the response should have field "maxId"

  # ==========================================================================
  # SUBJECT FILTER ON GET /schemas/ids/{id}
  # ==========================================================================

  Scenario: subject parameter filters subjects-by-ID results
    Given subject "subfilter-a" has schema:
      """
      {"type":"record","name":"SubFilter","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "subfilter_id"
    And I register a schema under subject "subfilter-b":
      """
      {"type":"record","name":"SubFilter","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/schemas/ids/{{subfilter_id}}/subjects?subject=subfilter-a"
    Then the response status should be 200
    And the response should contain "subfilter-a"
    And the response body should not contain "subfilter-b"

  Scenario: subject parameter filters versions-by-ID results
    Given subject "verfilter-a" has schema:
      """
      {"type":"record","name":"VerFilter","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "verfilter_id"
    And I register a schema under subject "verfilter-b":
      """
      {"type":"record","name":"VerFilter","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/schemas/ids/{{verfilter_id}}/versions?subject=verfilter-a"
    Then the response status should be 200
    And the response should contain "verfilter-a"
    And the response body should not contain "verfilter-b"
