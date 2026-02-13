@functional
Feature: Pagination — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive pagination tests covering subjects, versions, schemas by ID,
  and referencedby endpoints with offset and limit parameters.

  # ==========================================================================
  # SUBJECT LISTING PAGINATION
  # ==========================================================================

  Scenario: List subjects with offset and limit
    Given the global compatibility level is "NONE"
    And subject "page-subj-a" has schema:
      """
      {"type":"record","name":"PageA","fields":[{"name":"a","type":"string"}]}
      """
    And subject "page-subj-b" has schema:
      """
      {"type":"record","name":"PageB","fields":[{"name":"b","type":"string"}]}
      """
    And subject "page-subj-c" has schema:
      """
      {"type":"record","name":"PageC","fields":[{"name":"c","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=page-subj-&offset=0&limit=2"
    Then the response status should be 200
    And the response should be an array of length 2
    When I GET "/subjects?subjectPrefix=page-subj-&offset=2&limit=2"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # VERSION LISTING PAGINATION
  # ==========================================================================

  @pending-impl
  Scenario: List versions with offset and limit
    Given the global compatibility level is "NONE"
    And subject "page-ver" has schema:
      """
      {"type":"record","name":"PV1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "page-ver" has schema:
      """
      {"type":"record","name":"PV2","fields":[{"name":"b","type":"string"}]}
      """
    And subject "page-ver" has schema:
      """
      {"type":"record","name":"PV3","fields":[{"name":"c","type":"string"}]}
      """
    When I GET "/subjects/page-ver/versions?offset=0&limit=1"
    Then the response status should be 200
    And the response should be an array of length 1
    When I GET "/subjects/page-ver/versions?offset=1&limit=2"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # SUBJECTS BY SCHEMA ID PAGINATION
  # ==========================================================================

  @pending-impl
  Scenario: Get subjects by schema ID with offset and limit
    Given the global compatibility level is "NONE"
    When I register a schema under subject "page-byid-s1":
      """
      {"type":"record","name":"PageByID","fields":[{"name":"x","type":"string"}]}
      """
    And I store the response field "id" as "page_id"
    When I register a schema under subject "page-byid-s2":
      """
      {"type":"record","name":"PageByID","fields":[{"name":"x","type":"string"}]}
      """
    When I register a schema under subject "page-byid-s3":
      """
      {"type":"record","name":"PageByID","fields":[{"name":"x","type":"string"}]}
      """
    When I GET "/schemas/ids/{{page_id}}/subjects?offset=0&limit=2"
    Then the response status should be 200
    And the response should be an array of length 2
    When I GET "/schemas/ids/{{page_id}}/subjects?offset=2&limit=2"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # VERSIONS BY SCHEMA ID PAGINATION
  # ==========================================================================

  @pending-impl
  Scenario: Get versions by schema ID with offset and limit
    Given the global compatibility level is "NONE"
    When I register a schema under subject "page-verid-s1":
      """
      {"type":"record","name":"PageVerID","fields":[{"name":"x","type":"string"}]}
      """
    And I store the response field "id" as "pv_id"
    When I register a schema under subject "page-verid-s2":
      """
      {"type":"record","name":"PageVerID","fields":[{"name":"x","type":"string"}]}
      """
    When I GET "/schemas/ids/{{pv_id}}/versions?offset=0&limit=1"
    Then the response status should be 200
    And the response should be an array of length 1
    When I GET "/schemas/ids/{{pv_id}}/versions?offset=1&limit=1"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # SCHEMAS LIST PAGINATION
  # ==========================================================================

  Scenario: List schemas with offset and limit
    Given the global compatibility level is "NONE"
    And subject "page-schemas-1" has schema:
      """
      {"type":"record","name":"PS1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "page-schemas-2" has schema:
      """
      {"type":"record","name":"PS2","fields":[{"name":"b","type":"string"}]}
      """
    And subject "page-schemas-3" has schema:
      """
      {"type":"record","name":"PS3","fields":[{"name":"c","type":"string"}]}
      """
    When I GET "/schemas?subjectPrefix=page-schemas-&offset=0&limit=2"
    Then the response status should be 200
    And the response should be an array of length 2
