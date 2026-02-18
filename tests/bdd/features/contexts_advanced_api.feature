@functional @contexts
Feature: Contexts — Advanced API Surface Coverage
  Verify that ALL schema registry API endpoints work correctly with context-scoped
  subjects using the :.contextname:subject qualified format. Each scenario targets
  a specific API endpoint or behavior to ensure complete API surface coverage.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # RAW SCHEMA ENDPOINT
  # ==========================================================================

  Scenario: Raw schema endpoint returns schema string for context-scoped subject
    When I POST "/subjects/:.api1:raw-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi1\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.api1:raw-test/versions/1/schema"
    Then the response status should be 200
    And the response body should contain "record"
    And the response body should contain "TestApi1"

  # ==========================================================================
  # SCHEMA LOOKUP IN CONTEXT
  # ==========================================================================

  Scenario: Schema lookup returns correct result in context
    When I POST "/subjects/:.api2:lookup-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi2\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.api2:lookup-subj" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi2\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And the response field "subject" should be "lookup-subj"
    And the response field "version" should be 1
    And I store the response field "id" as "api2_id"

  # ==========================================================================
  # CROSS-CONTEXT LOOKUP ISOLATION
  # ==========================================================================

  Scenario: Schema lookup in wrong context returns 404
    When I POST "/subjects/:.api3a:cross-lookup/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi3\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    # Lookup the same schema in a different context — should NOT find it
    When I POST "/subjects/:.api3b:cross-lookup" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi3\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 404

  # ==========================================================================
  # SCHEMA ID SCOPING
  # ==========================================================================

  Scenario: Schema ID in context is not accessible from default context
    When I POST "/subjects/:.api4:id-subjects/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi4\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "api4_schema_id"
    # Verify schema exists in its context
    When I GET "/subjects/:.api4:id-subjects/versions/1"
    Then the response status should be 200
    And the response body should contain "TestApi4"
    # Schema ID 1 from context .api4 should NOT be found in default context
    When I GET "/schemas/ids/1"
    Then the response status should be 404

  # ==========================================================================
  # DELETED SUBJECTS LISTING
  # ==========================================================================

  Scenario: Deleted subjects listing with context-qualified and default subjects
    # Register and delete a schema in a context
    When I POST "/subjects/:.api5:del-list/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi5\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/:.api5:del-list"
    Then the response status should be 200
    # Root /subjects should NOT list context-scoped subjects
    When I GET "/subjects"
    Then the response status should be 200
    And the response body should not contain "del-list"
    # Register and delete a default-context subject
    When I POST "/subjects/default-api5-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi5Default\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/default-api5-subj"
    Then the response status should be 200
    # Deleted default subject should appear in ?deleted=true listing
    When I GET "/subjects?deleted=true"
    Then the response status should be 200
    And the response array should contain "default-api5-subj"

  # ==========================================================================
  # SOFT DELETE — VERSION ACCESS
  # ==========================================================================

  Scenario: Get schema version after soft delete returns 404
    When I POST "/subjects/:.api6:soft-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi6\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/:.api6:soft-del"
    Then the response status should be 200
    # After soft delete, versions are not accessible
    When I GET "/subjects/:.api6:soft-del/versions/1"
    Then the response status should be 404

  # ==========================================================================
  # PERMANENT DELETE — FULL LIFECYCLE
  # ==========================================================================

  Scenario: Permanent delete then verify subject is completely gone
    When I POST "/subjects/:.api7:perm-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi7\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    # Step 1: Soft delete
    When I DELETE "/subjects/:.api7:perm-del"
    Then the response status should be 200
    # Step 2: Permanent delete
    When I DELETE "/subjects/:.api7:perm-del?permanent=true"
    Then the response status should be 200
    # Subject should be completely gone
    When I GET "/subjects/:.api7:perm-del/versions"
    Then the response status should be 404

  # ==========================================================================
  # SCHEMA FINGERPRINT DEDUP WITHIN CONTEXT
  # ==========================================================================

  Scenario: Same schema under different subjects in same context shares schema ID
    When I POST "/subjects/:.api8:subj-a/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi8\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_a"
    When I POST "/subjects/:.api8:subj-b/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi8\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_b"
    # Same schema content should be deduplicated — same ID within context
    Then the response field "id" should equal stored "id_a"

  # ==========================================================================
  # LATEST VERSION TRACKING
  # ==========================================================================

  Scenario: Get latest version tracks correctly after multiple registrations
    # Register v1
    When I POST "/subjects/:.api9:latest-track/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi9\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    # Register v2 (add optional field with default)
    When I POST "/subjects/:.api9:latest-track/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi9\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Latest should be version 2
    When I GET "/subjects/:.api9:latest-track/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 2

  # ==========================================================================
  # LIST VERSIONS — MULTI-VERSION
  # ==========================================================================

  Scenario: List versions returns all versions registered in context
    # Register v1
    When I POST "/subjects/:.api10:multi-ver/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi10v1\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    # Register v2 (same record name, add field with default — backward compatible)
    When I POST "/subjects/:.api10:multi-ver/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi10v1\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"a\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Register v3 (same record name, add another field with default — backward compatible)
    When I POST "/subjects/:.api10:multi-ver/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi10v1\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"a\",\"type\":\"string\",\"default\":\"\"},{\"name\":\"b\",\"type\":\"int\",\"default\":0}]}"}
      """
    Then the response status should be 200
    # List all versions
    When I GET "/subjects/:.api10:multi-ver/versions"
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain integer 1
    And the response array should contain integer 2
    And the response array should contain integer 3

  # ==========================================================================
  # SUBJECT LISTING — DEFAULT CONTEXT SCOPING
  # ==========================================================================

  Scenario: Subject listing is scoped to default context only
    # Register in default context
    When I POST "/subjects/default-api11/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi11Default\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    # Register in named context
    When I POST "/subjects/:.api11:ctx-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi11Ctx\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    # Default subject listing should contain default-context subject
    When I GET "/subjects"
    Then the response status should be 200
    And the response array should contain "default-api11"
    # Context-scoped subjects should NOT appear in default listing
    And the response body should not contain "ctx-subj"
    And the response body should not contain ":.api11:"

  # ==========================================================================
  # SCHEMA TYPES — GLOBAL ENDPOINT
  # ==========================================================================

  Scenario: Schema types endpoint is global and not context-scoped
    When I GET "/schemas/types"
    Then the response status should be 200
    And the response array should contain "AVRO"
    And the response array should contain "PROTOBUF"
    And the response array should contain "JSON"

  # ==========================================================================
  # COMPATIBILITY CHECK — SPECIFIC VERSION IN CONTEXT
  # ==========================================================================

  Scenario: Compatibility check against specific version in context
    # Register v1
    When I POST "/subjects/:.api13:compat-ver/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi13\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    # Set BACKWARD compatibility
    When I PUT "/config/:.api13:compat-ver" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Register v2 (compatible change)
    When I POST "/subjects/:.api13:compat-ver/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi13\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Check compatibility against version 1 specifically
    When I POST "/compatibility/subjects/:.api13:compat-ver/versions/1" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi13\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"status\",\"type\":\"string\",\"default\":\"active\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # COMPATIBILITY CHECK — LATEST VERSION IN CONTEXT
  # ==========================================================================

  Scenario: Compatibility check against latest version in context
    # Register v1
    When I POST "/subjects/:.api14:compat-latest/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi14\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    # Set BACKWARD compatibility
    When I PUT "/config/:.api14:compat-latest" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    # Check compatibility against latest (which is v1)
    When I POST "/compatibility/subjects/:.api14:compat-latest/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi14\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"email\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # NON-EXISTENT VERSION — 404
  # ==========================================================================

  Scenario: Get non-existent version in context returns 404
    When I POST "/subjects/:.api15:ver-404/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi15\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.api15:ver-404/versions/99"
    Then the response status should be 404

  # ==========================================================================
  # DELETE NON-EXISTENT SUBJECT — 404
  # ==========================================================================

  Scenario: Delete non-existent subject in context returns 404
    When I DELETE "/subjects/:.api16:nonexistent"
    Then the response status should be 404

  # ==========================================================================
  # VERSIONS ENDPOINT AFTER REGISTRATION
  # ==========================================================================

  Scenario: Register schema in context then verify versions endpoint
    When I POST "/subjects/:.api17:versions-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TestApi17\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.api17:versions-test/versions"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain integer 1
