@functional
Feature: Subject & Version Management — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive subject and version management tests from the Confluent Schema Registry
  v8.1.1 test suite.

  # ==========================================================================
  # SUBJECT LISTING
  # ==========================================================================

  Scenario: List all subjects when no subjects exist returns empty array
    When I list all subjects
    Then the response status should be 200
    And the response should be an array of length 0

  Scenario: List subjects returns all registered subjects
    Given subject "sv-list-topic1" has schema:
      """
      {"type":"record","name":"T1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "sv-list-topic2" has schema:
      """
      {"type":"record","name":"T2","fields":[{"name":"b","type":"string"}]}
      """
    When I list all subjects
    Then the response status should be 200
    And the response array should contain "sv-list-topic1"
    And the response array should contain "sv-list-topic2"

  Scenario: Soft-deleted subjects excluded from listing but visible with deleted=true
    Given subject "sv-softdel" has schema:
      """
      {"type":"record","name":"SD","fields":[{"name":"a","type":"string"}]}
      """
    When I delete subject "sv-softdel"
    Then the response status should be 200
    When I list all subjects
    Then the response status should be 200
    When I GET "/subjects?deleted=true"
    Then the response status should be 200
    And the response array should contain "sv-softdel"

  # ==========================================================================
  # VERSION LISTING
  # ==========================================================================

  Scenario: List versions for non-existent subject returns 404
    When I list versions of subject "sv-nonexist"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Get version for non-existent subject returns 404
    When I get version 1 of subject "sv-nonexist-ver"
    Then the response status should be 404
    And the response should have error code 40401

  # ==========================================================================
  # VERSION RETRIEVAL
  # ==========================================================================

  Scenario: Get non-existing version returns 404
    Given subject "sv-no-ver" has schema:
      """
      {"type":"record","name":"NoVer","fields":[{"name":"a","type":"string"}]}
      """
    When I get version 200 of subject "sv-no-ver"
    Then the response status should be 404
    And the response should have error code 40402

  Scenario: Get invalid version 0 returns 422
    Given subject "sv-inv-ver" has schema:
      """
      {"type":"record","name":"InvVer","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/sv-inv-ver/versions/0"
    Then the response status should be 422
    And the response should have error code 42202

  Scenario: Get latest version after deleting older version
    Given the global compatibility level is "NONE"
    And subject "sv-del-older" has schema:
      """
      {"type":"record","name":"Older1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "sv-del-older" has schema:
      """
      {"type":"record","name":"Older2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "sv-del-older"
    Then the response status should be 200
    When I get the latest version of subject "sv-del-older"
    Then the response status should be 200
    And the response field "version" should be 2

  Scenario: Get latest version for non-existent subject returns 404
    When I get the latest version of subject "sv-nonexist-latest"
    Then the response status should be 404
    And the response should have error code 40401
