@functional
Feature: Schema Lookup & Retrieval — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive schema lookup and retrieval tests covering all API patterns
  from the Confluent Schema Registry v8.1.1 test suite.

  # ==========================================================================
  # GET SCHEMA BY ID
  # ==========================================================================

  Scenario: Get schema by non-existent ID returns 404
    When I GET "/schemas/ids/99999"
    Then the response status should be 404
    And the response should have error code 40403

  Scenario: Get schema by ID returns correct schema
    When I register a schema under subject "lookup-by-id":
      """
      {"type":"record","name":"LookupByID","fields":[{"name":"n","type":"string"}]}
      """
    And I store the response field "id" as "schema_id"
    When I get schema by ID {{schema_id}}
    Then the response status should be 200
    And the response should contain "LookupByID"
    And the response should have field "schema"

  Scenario: Get schema-only by ID returns raw schema string
    When I register a schema under subject "lookup-raw":
      """
      {"type":"record","name":"LookupRaw","fields":[{"name":"n","type":"string"}]}
      """
    And I store the response field "id" as "schema_id"
    When I get the raw schema by ID {{schema_id}}
    Then the response status should be 200
    And the response should contain "LookupRaw"

  Scenario: Get schema with fetchMaxId returns maxId field
    When I register a schema under subject "lookup-maxid-1":
      """
      {"type":"record","name":"MaxID1","fields":[{"name":"a","type":"string"}]}
      """
    And I store the response field "id" as "schema_id"
    When I register a schema under subject "lookup-maxid-2":
      """
      {"type":"record","name":"MaxID2","fields":[{"name":"b","type":"string"}]}
      """
    When I GET "/schemas/ids/{{schema_id}}?fetchMaxId=true"
    Then the response status should be 200
    And the response should have field "maxId"

  Scenario: Get schema types returns supported types
    When I GET "/schemas/types"
    Then the response status should be 200
    And the response array should contain "AVRO"
    And the response array should contain "JSON"
    And the response array should contain "PROTOBUF"

  # ==========================================================================
  # GET SCHEMA BY SUBJECT AND VERSION
  # ==========================================================================

  Scenario: Get version returns full metadata
    Given the global compatibility level is "NONE"
    And subject "lookup-ver" has schema:
      """
      {"type":"record","name":"Ver1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "lookup-ver" has schema:
      """
      {"type":"record","name":"Ver2","fields":[{"name":"b","type":"string"}]}
      """
    When I get version 1 of subject "lookup-ver"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response field "subject" should be "lookup-ver"
    When I get version 2 of subject "lookup-ver"
    Then the response status should be 200
    And the response field "version" should be 2
    When I get the latest version of subject "lookup-ver"
    Then the response status should be 200
    And the response field "version" should be 2

  Scenario: Get latest version schema-only
    Given the global compatibility level is "NONE"
    And subject "lookup-latest-raw" has schema:
      """
      {"type":"record","name":"LatestV1","fields":[{"name":"v","type":"string"}]}
      """
    And subject "lookup-latest-raw" has schema:
      """
      {"type":"record","name":"LatestV2","fields":[{"name":"w","type":"string"}]}
      """
    When I GET "/subjects/lookup-latest-raw/versions/latest/schema"
    Then the response status should be 200
    And the response should contain "LatestV2"

  Scenario: Get specific version schema-only
    Given subject "lookup-specraw" has schema:
      """
      {"type":"record","name":"SpecRaw","fields":[{"name":"v","type":"string"}]}
      """
    When I GET "/subjects/lookup-specraw/versions/1/schema"
    Then the response status should be 200
    And the response should contain "SpecRaw"

  # ==========================================================================
  # SCHEMA LOOKUP (POST /subjects/{subject})
  # ==========================================================================

  Scenario: Lookup schema under non-existent subject returns 404
    When I lookup schema in subject "lookup-nonexistent":
      """
      {"type":"string"}
      """
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Lookup non-existent schema under existing subject returns 404
    Given subject "lookup-miss-subj" has schema:
      """
      {"type":"record","name":"Exists","fields":[{"name":"a","type":"string"}]}
      """
    When I lookup schema in subject "lookup-miss-subj":
      """
      {"type":"record","name":"NotHere","fields":[{"name":"b","type":"int"}]}
      """
    Then the response status should be 404
    And the response should have error code 40403

  # ==========================================================================
  # SUBJECTS AND VERSIONS BY SCHEMA ID
  # ==========================================================================

  Scenario: Get subjects associated with schema ID
    When I register a schema under subject "lookup-assoc-s1":
      """
      {"type":"record","name":"Assoc","fields":[{"name":"x","type":"string"}]}
      """
    And I store the response field "id" as "assoc_id"
    When I register a schema under subject "lookup-assoc-s2":
      """
      {"type":"record","name":"Assoc","fields":[{"name":"x","type":"string"}]}
      """
    When I get the subjects for schema ID {{assoc_id}}
    Then the response status should be 200
    And the response array should contain "lookup-assoc-s1"
    And the response array should contain "lookup-assoc-s2"

  Scenario: Get subjects by schema ID after soft-delete excludes deleted
    When I register a schema under subject "lookup-delsub-s1":
      """
      {"type":"record","name":"DelSubAssoc","fields":[{"name":"x","type":"string"}]}
      """
    And I store the response field "id" as "delsub_id"
    When I register a schema under subject "lookup-delsub-s2":
      """
      {"type":"record","name":"DelSubAssoc","fields":[{"name":"x","type":"string"}]}
      """
    When I delete subject "lookup-delsub-s2"
    Then the response status should be 200
    When I get the subjects for schema ID {{delsub_id}}
    Then the response status should be 200
    And the response array should contain "lookup-delsub-s1"
    When I GET "/schemas/ids/{{delsub_id}}/subjects?deleted=true"
    Then the response status should be 200
    And the response array should contain "lookup-delsub-s1"
    And the response array should contain "lookup-delsub-s2"

  Scenario: Get subjects for non-existent schema ID returns 404
    When I GET "/schemas/ids/99999/subjects"
    Then the response status should be 404
    And the response should have error code 40403

  Scenario: Get versions associated with schema ID
    When I register a schema under subject "lookup-ver-assoc-s1":
      """
      {"type":"record","name":"VerAssoc","fields":[{"name":"x","type":"string"}]}
      """
    And I store the response field "id" as "verassoc_id"
    When I register a schema under subject "lookup-ver-assoc-s2":
      """
      {"type":"record","name":"VerAssoc","fields":[{"name":"x","type":"string"}]}
      """
    When I get versions for schema ID {{verassoc_id}}
    Then the response status should be 200
    And the response should contain "lookup-ver-assoc-s1"
    And the response should contain "lookup-ver-assoc-s2"

  # ==========================================================================
  # SCHEMAS LIST ENDPOINT
  # ==========================================================================

  Scenario: List schemas with latestOnly returns one per subject
    Given the global compatibility level is "NONE"
    And subject "lookup-list-s1" has schema:
      """
      {"type":"record","name":"List1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "lookup-list-s1" has schema:
      """
      {"type":"record","name":"List1v2","fields":[{"name":"b","type":"string"}]}
      """
    And subject "lookup-list-s2" has schema:
      """
      {"type":"record","name":"List2","fields":[{"name":"c","type":"string"}]}
      """
    When I GET "/schemas?latestOnly=true&subjectPrefix=lookup-list-"
    Then the response status should be 200
    And the response should be an array of length 2
