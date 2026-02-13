@functional
Feature: Deletion Lifecycle (Two-Step Delete)
  Confluent Schema Registry requires a two-step deletion process:
  Step 1: Soft-delete (marks resource as deleted but keeps it)
  Step 2: Permanent delete (only works if already soft-deleted)
  Attempting to permanently delete without soft-deleting first returns 40405.

  # ==========================================================================
  # TWO-STEP DELETION — SUBJECT LEVEL
  # ==========================================================================

  Scenario: Soft-delete subject hides it from list
    Given subject "del-lifecycle-1" has schema:
      """
      {"type":"record","name":"D1","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-lifecycle-1"
    Then the response status should be 200
    When I list all subjects
    Then the response should be an array of length 0

  Scenario: Soft-deleted subject visible with deleted=true
    Given subject "del-lifecycle-2" has schema:
      """
      {"type":"record","name":"D2","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-lifecycle-2"
    Then the response status should be 200
    When I list subjects with deleted
    Then the response status should be 200
    And the response should contain "del-lifecycle-2"

  Scenario: Permanent delete without soft-delete first returns 40405
    Given subject "del-lifecycle-3" has schema:
      """
      {"type":"record","name":"D3","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-lifecycle-3?permanent=true"
    Then the response status should be 404
    And the response should have error code 40405

  Scenario: Soft-delete then permanent delete succeeds
    Given subject "del-lifecycle-4" has schema:
      """
      {"type":"record","name":"D4","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-lifecycle-4"
    Then the response status should be 200
    When I DELETE "/subjects/del-lifecycle-4?permanent=true"
    Then the response status should be 200
    When I list subjects with deleted
    Then the response body should not contain "del-lifecycle-4"

  # ==========================================================================
  # TWO-STEP DELETION — VERSION LEVEL
  # ==========================================================================

  Scenario: Permanent delete version without soft-delete first returns 40405
    Given the global compatibility level is "NONE"
    And subject "del-ver-lifecycle-1" has schema:
      """
      {"type":"record","name":"DV1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ver-lifecycle-1" has schema:
      """
      {"type":"record","name":"DV2","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-ver-lifecycle-1/versions/1?permanent=true"
    Then the response status should be 404
    And the response should have error code 40407

  Scenario: Soft-delete version then permanent delete succeeds
    Given the global compatibility level is "NONE"
    And subject "del-ver-lifecycle-2" has schema:
      """
      {"type":"record","name":"DV3","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ver-lifecycle-2" has schema:
      """
      {"type":"record","name":"DV4","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-ver-lifecycle-2/versions/1"
    Then the response status should be 200
    When I DELETE "/subjects/del-ver-lifecycle-2/versions/1?permanent=true"
    Then the response status should be 200

  # ==========================================================================
  # SOFT-DELETED VERSION VISIBILITY
  # ==========================================================================

  Scenario: Soft-deleted version hidden from version list
    Given the global compatibility level is "NONE"
    And subject "del-vis-1" has schema:
      """
      {"type":"record","name":"Vis1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-vis-1" has schema:
      """
      {"type":"record","name":"Vis2","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-vis-1/versions/1"
    Then the response status should be 200
    When I list versions of subject "del-vis-1"
    Then the response status should be 200
    And the response should be an array of length 1

  Scenario: Soft-deleted version visible with deleted=true param
    Given the global compatibility level is "NONE"
    And subject "del-vis-2" has schema:
      """
      {"type":"record","name":"Vis3","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-vis-2" has schema:
      """
      {"type":"record","name":"Vis4","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-vis-2/versions/1"
    Then the response status should be 200
    When I GET "/subjects/del-vis-2/versions?deleted=true"
    Then the response status should be 200
    And the response should be an array of length 2

  Scenario: GET soft-deleted version without deleted param returns 404
    Given subject "del-vis-3" has schema:
      """
      {"type":"record","name":"Vis5","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-vis-3/versions/1"
    Then the response status should be 200
    When I GET "/subjects/del-vis-3/versions/1"
    Then the response status should be 404

  # ==========================================================================
  # RE-REGISTRATION AFTER DELETE
  # ==========================================================================

  Scenario: Re-register after soft-delete continues version numbering
    Given the global compatibility level is "NONE"
    And subject "del-rereg-1" has schema:
      """
      {"type":"record","name":"RR1","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-rereg-1"
    Then the response status should be 200
    When I register a schema under subject "del-rereg-1":
      """
      {"type":"record","name":"RR2","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "del-rereg-1"
    Then the response status should be 200
    And the response field "version" should be 2

  Scenario: Re-register after permanent delete starts at version 1
    Given subject "del-rereg-2" has schema:
      """
      {"type":"record","name":"RR3","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-rereg-2"
    Then the response status should be 200
    When I DELETE "/subjects/del-rereg-2?permanent=true"
    Then the response status should be 200
    When I register a schema under subject "del-rereg-2":
      """
      {"type":"record","name":"RR4","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "del-rereg-2"
    Then the response status should be 200
    And the response field "version" should be 1

  # ==========================================================================
  # DELETE VERSION "latest"
  # ==========================================================================

  Scenario: DELETE version latest soft-deletes the latest version
    Given the global compatibility level is "NONE"
    And subject "del-latest-1" has schema:
      """
      {"type":"record","name":"DL1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-latest-1" has schema:
      """
      {"type":"record","name":"DL2","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-latest-1/versions/latest"
    Then the response status should be 200
    When I get the latest version of subject "del-latest-1"
    Then the response status should be 200
    And the response field "version" should be 1

  Scenario: DELETE version -1 works like latest
    Given the global compatibility level is "NONE"
    And subject "del-minus1" has schema:
      """
      {"type":"record","name":"DM1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-minus1" has schema:
      """
      {"type":"record","name":"DM2","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-minus1/versions/-1"
    Then the response status should be 200
    When I get the latest version of subject "del-minus1"
    Then the response status should be 200
    And the response field "version" should be 1

  # ==========================================================================
  # LOOKUP AFTER DELETION
  # ==========================================================================

  Scenario: Lookup schema after soft-delete returns 404
    Given subject "del-lookup-1" has schema:
      """
      {"type":"record","name":"LK1","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-lookup-1"
    Then the response status should be 200
    When I lookup schema in subject "del-lookup-1":
      """
      {"type":"record","name":"LK1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 404
