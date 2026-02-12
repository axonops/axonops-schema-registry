@functional
Feature: Schema Listing, Querying, and Raw Schema Endpoints
  As a developer, I want to list schemas with filtering, pagination, and query them
  by ID to retrieve raw schema text, subject-version mappings, and subject lists

  Background:
    Given the global compatibility level is "NONE"

  # -----------------------------------------------------------------------
  # Scenario 1: GET /schemas returns all registered schemas
  # -----------------------------------------------------------------------
  Scenario: List all registered schemas across multiple subjects
    Given subject "orders-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"order_id","type":"long"}]}
      """
    And subject "payments-value" has schema:
      """
      {"type":"record","name":"Payment","fields":[{"name":"amount","type":"double"}]}
      """
    And subject "users-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I list all schemas
    Then the response status should be 200
    And the response should be an array of length 3
    And the response body should contain "Order"
    And the response body should contain "Payment"
    And the response body should contain "User"

  # -----------------------------------------------------------------------
  # Scenario 2: GET /schemas?subjectPrefix=X filters by subject prefix
  # -----------------------------------------------------------------------
  Scenario: Filter schemas by subject prefix
    Given subject "team-alpha-orders" has schema:
      """
      {"type":"record","name":"AlphaOrder","fields":[{"name":"id","type":"long"}]}
      """
    And subject "team-alpha-users" has schema:
      """
      {"type":"record","name":"AlphaUser","fields":[{"name":"name","type":"string"}]}
      """
    And subject "team-beta-events" has schema:
      """
      {"type":"record","name":"BetaEvent","fields":[{"name":"ts","type":"long"}]}
      """
    When I GET "/schemas?subjectPrefix=team-alpha"
    Then the response status should be 200
    And the response should be an array of length 2
    And the response body should contain "AlphaOrder"
    And the response body should contain "AlphaUser"
    And the response body should not contain "BetaEvent"

  # -----------------------------------------------------------------------
  # Scenario 3: GET /schemas?latestOnly=true returns only latest version
  # -----------------------------------------------------------------------
  Scenario: List schemas with latestOnly returns only the latest version per subject
    Given subject "evolving-value" has schema:
      """
      {"type":"record","name":"Evolving","fields":[{"name":"v1_field","type":"string"}]}
      """
    And subject "evolving-value" has schema:
      """
      {"type":"record","name":"Evolving","fields":[{"name":"v1_field","type":"string"},{"name":"v2_field","type":"int","default":0}]}
      """
    And subject "evolving-value" has schema:
      """
      {"type":"record","name":"Evolving","fields":[{"name":"v1_field","type":"string"},{"name":"v2_field","type":"int","default":0},{"name":"v3_field","type":"long","default":0}]}
      """
    When I GET "/schemas?latestOnly=true"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response body should contain "v3_field"
    And the response body should not contain "\"version\":1"

  # -----------------------------------------------------------------------
  # Scenario 4: GET /schemas?deleted=true includes soft-deleted schemas
  # -----------------------------------------------------------------------
  Scenario: List schemas with deleted flag includes soft-deleted schemas
    Given subject "active-subj" has schema:
      """
      {"type":"record","name":"Active","fields":[{"name":"a","type":"string"}]}
      """
    And subject "deleted-subj" has schema:
      """
      {"type":"record","name":"Deleted","fields":[{"name":"d","type":"string"}]}
      """
    When I delete subject "deleted-subj"
    Then the response status should be 200
    When I list all schemas
    Then the response status should be 200
    And the response should be an array of length 1
    And the response body should contain "Active"
    And the response body should not contain "Deleted"
    When I GET "/schemas?deleted=true"
    Then the response status should be 200
    And the response should be an array of length 2
    And the response body should contain "Active"
    And the response body should contain "Deleted"

  # -----------------------------------------------------------------------
  # Scenario 5: GET /schemas?offset=N&limit=M paginates results
  # -----------------------------------------------------------------------
  Scenario: Paginate schema listing with offset and limit
    Given subject "page-a" has schema:
      """
      {"type":"record","name":"PageA","fields":[{"name":"a","type":"string"}]}
      """
    And subject "page-b" has schema:
      """
      {"type":"record","name":"PageB","fields":[{"name":"b","type":"string"}]}
      """
    And subject "page-c" has schema:
      """
      {"type":"record","name":"PageC","fields":[{"name":"c","type":"string"}]}
      """
    And subject "page-d" has schema:
      """
      {"type":"record","name":"PageD","fields":[{"name":"d","type":"string"}]}
      """
    When I GET "/schemas?limit=2"
    Then the response status should be 200
    And the response should be an array of length 2
    When I GET "/schemas?offset=2&limit=2"
    Then the response status should be 200
    And the response should be an array of length 2
    When I GET "/schemas?offset=3&limit=10"
    Then the response status should be 200
    And the response should be an array of length 1
    When I GET "/schemas?offset=10"
    Then the response status should be 200
    And the response should be an array of length 0

  # -----------------------------------------------------------------------
  # Scenario 6: GET /schemas?subjectPrefix=X&latestOnly=true combined
  # -----------------------------------------------------------------------
  Scenario: Combined subject prefix and latestOnly filters
    Given subject "svc-orders" has schema:
      """
      {"type":"record","name":"OrderV1","fields":[{"name":"id","type":"long"}]}
      """
    And subject "svc-orders" has schema:
      """
      {"type":"record","name":"OrderV2","fields":[{"name":"id","type":"long"},{"name":"ts","type":"long","default":0}]}
      """
    And subject "svc-payments" has schema:
      """
      {"type":"record","name":"PaymentV1","fields":[{"name":"amount","type":"double"}]}
      """
    And subject "other-events" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"type","type":"string"}]}
      """
    When I GET "/schemas?subjectPrefix=svc-&latestOnly=true"
    Then the response status should be 200
    And the response should be an array of length 2
    And the response body should contain "OrderV2"
    And the response body should contain "PaymentV1"
    And the response body should not contain "OrderV1"
    And the response body should not contain "Event"

  # -----------------------------------------------------------------------
  # Scenario 7: GET /schemas/ids/{id}/schema returns raw schema text
  # -----------------------------------------------------------------------
  Scenario: Get raw schema text by global schema ID
    When I register a schema under subject "raw-test-value":
      """
      {"type":"record","name":"RawTest","fields":[{"name":"data","type":"bytes"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the raw schema by ID {{schema_id}}
    Then the response status should be 200
    And the response body should contain "RawTest"
    And the response body should contain "bytes"

  # -----------------------------------------------------------------------
  # Scenario 8: GET /subjects/{subject}/versions/{version}/schema returns raw
  # -----------------------------------------------------------------------
  Scenario: Get raw schema text by subject and version
    Given subject "versioned-value" has schema:
      """
      {"type":"record","name":"Versioned","fields":[{"name":"v1","type":"string"}]}
      """
    And subject "versioned-value" has schema:
      """
      {"type":"record","name":"Versioned","fields":[{"name":"v1","type":"string"},{"name":"v2","type":"int","default":0}]}
      """
    When I get the raw schema for subject "versioned-value" version 1
    Then the response status should be 200
    And the response body should contain "Versioned"
    And the response body should not contain "v2"
    When I get the raw schema for subject "versioned-value" version 2
    Then the response status should be 200
    And the response body should contain "v2"

  # -----------------------------------------------------------------------
  # Scenario 9: GET /schemas/ids/{id}/versions returns subject-version pairs
  # -----------------------------------------------------------------------
  Scenario: Get subject-version pairs for a schema ID
    When I register a schema under subject "svpair-value":
      """
      {"type":"record","name":"SVPair","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get versions for schema ID {{schema_id}}
    Then the response status should be 200
    And the response should be an array of length 1
    And the response body should contain "svpair-value"

  # -----------------------------------------------------------------------
  # Scenario 10: GET /schemas/ids/{id}/subjects?deleted=true includes deleted
  # -----------------------------------------------------------------------
  Scenario: Get subjects for schema ID includes deleted subjects when requested
    When I register a schema under subject "keep-subj":
      """
      {"type":"record","name":"Shared","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I register a schema under subject "drop-subj":
      """
      {"type":"record","name":"Shared","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200
    When I delete subject "drop-subj"
    Then the response status should be 200
    When I get the subjects for schema ID {{schema_id}}
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain "keep-subj"
    And the response body should not contain "drop-subj"
    When I GET "/schemas/ids/{{schema_id}}/subjects?deleted=true"
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "keep-subj"
    And the response array should contain "drop-subj"

  # -----------------------------------------------------------------------
  # Scenario 11: GET /schemas/ids/{id}/versions with multiple subjects
  # -----------------------------------------------------------------------
  Scenario: Get versions for a schema used by multiple subjects
    When I register a schema under subject "multi-a":
      """
      {"type":"record","name":"Common","fields":[{"name":"val","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I register a schema under subject "multi-b":
      """
      {"type":"record","name":"Common","fields":[{"name":"val","type":"string"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "multi-c":
      """
      {"type":"record","name":"Common","fields":[{"name":"val","type":"string"}]}
      """
    Then the response status should be 200
    When I get versions for schema ID {{schema_id}}
    Then the response status should be 200
    And the response should be an array of length 3
    And the response body should contain "multi-a"
    And the response body should contain "multi-b"
    And the response body should contain "multi-c"

  # -----------------------------------------------------------------------
  # Scenario 12: GET /schemas/ids/{id}/subjects with multiple subjects
  # -----------------------------------------------------------------------
  Scenario: Get subjects for a schema used by multiple subjects
    When I register a schema under subject "shared-alpha":
      """
      {"type":"record","name":"SharedSchema","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I register a schema under subject "shared-beta":
      """
      {"type":"record","name":"SharedSchema","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "shared-gamma":
      """
      {"type":"record","name":"SharedSchema","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    When I get the subjects for schema ID {{schema_id}}
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain "shared-alpha"
    And the response array should contain "shared-beta"
    And the response array should contain "shared-gamma"
