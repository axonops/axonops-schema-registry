@functional
Feature: Schema & Subject Deletion — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive deletion tests covering soft/hard delete for both versions and
  subjects, including re-registration behavior and edge cases.

  # ==========================================================================
  # VERSION SOFT DELETE
  # ==========================================================================

  Scenario: Soft-delete a version returns the deleted version number
    Given the global compatibility level is "NONE"
    And subject "del-ex-soft" has schema:
      """
      {"type":"record","name":"DelV1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ex-soft" has schema:
      """
      {"type":"record","name":"DelV2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "del-ex-soft"
    Then the response status should be 200
    When I list versions of subject "del-ex-soft"
    Then the response should be an array of length 1

  Scenario: Get soft-deleted version without deleted flag returns 404
    Given subject "del-ex-flag" has schema:
      """
      {"type":"record","name":"Flag","fields":[{"name":"a","type":"string"}]}
      """
    When I delete version 1 of subject "del-ex-flag"
    Then the response status should be 200
    When I get version 1 of subject "del-ex-flag"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Get soft-deleted version with deleted=true succeeds
    Given subject "del-ex-getdel" has schema:
      """
      {"type":"record","name":"GetDel","fields":[{"name":"a","type":"string"}]}
      """
    When I delete version 1 of subject "del-ex-getdel"
    Then the response status should be 200
    When I GET "/subjects/del-ex-getdel/versions/1?deleted=true"
    Then the response status should be 200
    And the response should contain "GetDel"

  Scenario: Lookup soft-deleted schema fails without deleted flag
    Given subject "del-ex-lookup" has schema:
      """
      {"type":"record","name":"LookupDel","fields":[{"name":"a","type":"string"}]}
      """
    When I delete version 1 of subject "del-ex-lookup"
    Then the response status should be 200
    When I lookup schema in subject "del-ex-lookup":
      """
      {"type":"record","name":"LookupDel","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 404

  @axonops-only
  Scenario: Lookup soft-deleted schema succeeds with deleted=true
    Given subject "del-ex-lookup2" has schema:
      """
      {"type":"record","name":"LookupDel2","fields":[{"name":"a","type":"string"}]}
      """
    When I delete version 1 of subject "del-ex-lookup2"
    Then the response status should be 200
    When I lookup schema in subject "del-ex-lookup2" with deleted:
      """
      {"type":"record","name":"LookupDel2","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # VERSION HARD DELETE
  # ==========================================================================

  Scenario: Hard-delete a version after soft-delete succeeds
    Given subject "del-ex-hard" has schema:
      """
      {"type":"record","name":"HardDel","fields":[{"name":"a","type":"string"}]}
      """
    When I delete version 1 of subject "del-ex-hard"
    Then the response status should be 200
    When I permanently delete version 1 of subject "del-ex-hard"
    Then the response status should be 200
    When I GET "/subjects/del-ex-hard/versions/1?deleted=true"
    Then the response status should be 404

  Scenario: Delete latest version falls back to previous version
    Given the global compatibility level is "NONE"
    And subject "del-ex-latest" has schema:
      """
      {"type":"record","name":"Latest1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ex-latest" has schema:
      """
      {"type":"record","name":"Latest2","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-ex-latest/versions/latest"
    Then the response status should be 200
    When I get the latest version of subject "del-ex-latest"
    Then the response status should be 200
    And the response field "version" should be 1

  Scenario: Delete non-existent subject version returns 404
    When I DELETE "/subjects/del-ex-nosub/versions/1"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Delete non-existent version under existing subject returns 404
    Given subject "del-ex-nover" has schema:
      """
      {"type":"record","name":"NoVer","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-ex-nover/versions/99"
    Then the response status should be 404
    And the response should have error code 40402

  # ==========================================================================
  # SUBJECT SOFT DELETE
  # ==========================================================================

  Scenario: Soft-delete subject returns list of deleted version numbers
    Given the global compatibility level is "NONE"
    And subject "del-ex-subj" has schema:
      """
      {"type":"record","name":"SubjDel1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ex-subj" has schema:
      """
      {"type":"record","name":"SubjDel2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete subject "del-ex-subj"
    Then the response status should be 200

  # ==========================================================================
  # SUBJECT HARD DELETE
  # ==========================================================================

  Scenario: Hard-delete subject requires prior soft-delete
    Given the global compatibility level is "NONE"
    And subject "del-ex-subj-hard" has schema:
      """
      {"type":"record","name":"SubjHard1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ex-subj-hard" has schema:
      """
      {"type":"record","name":"SubjHard2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete subject "del-ex-subj-hard"
    Then the response status should be 200
    When I permanently delete subject "del-ex-subj-hard"
    Then the response status should be 200
    When I GET "/subjects?deleted=true"
    Then the response status should be 200

  # ==========================================================================
  # RE-REGISTRATION AFTER DELETE
  # ==========================================================================

  Scenario: Re-register after soft-delete creates new version numbers
    Given the global compatibility level is "NONE"
    And subject "del-ex-rereg" has schema:
      """
      {"type":"record","name":"ReReg1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ex-rereg" has schema:
      """
      {"type":"record","name":"ReReg2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete subject "del-ex-rereg"
    Then the response status should be 200
    When I register a schema under subject "del-ex-rereg":
      """
      {"type":"record","name":"ReReg1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    When I list versions of subject "del-ex-rereg"
    Then the response status should be 200

  Scenario: Lookup after delete and re-register finds new version
    Given the global compatibility level is "NONE"
    And subject "del-ex-lookup-rereg" has schema:
      """
      {"type":"record","name":"LRR1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ex-lookup-rereg" has schema:
      """
      {"type":"record","name":"LRR2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "del-ex-lookup-rereg"
    Then the response status should be 200
    # Lookup deleted schema without flag fails
    When I lookup schema in subject "del-ex-lookup-rereg":
      """
      {"type":"record","name":"LRR1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 404
    # Re-register creates new version
    When I register a schema under subject "del-ex-lookup-rereg":
      """
      {"type":"record","name":"LRR1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    # Now lookup finds the new version
    When I lookup schema in subject "del-ex-lookup-rereg":
      """
      {"type":"record","name":"LRR1","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # COMPATIBILITY AFTER DELETE
  # ==========================================================================

  Scenario: Compatibility check works correctly after deleting incompatible version
    Given the global compatibility level is "NONE"
    And subject "del-ex-compat" has schema:
      """
      {"type":"record","name":"Compat","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "del-ex-compat" has schema:
      """
      {"type":"record","name":"Compat","fields":[{"name":"f1","type":"int"}]}
      """
    When I delete version 2 of subject "del-ex-compat"
    Then the response status should be 200
    When I set the config for subject "del-ex-compat" to "BACKWARD"
    And I register a schema under subject "del-ex-compat":
      """
      {"type":"record","name":"Compat","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # SUBJECT CONFIG AFTER DELETE
  # ==========================================================================

  Scenario: Subject compatibility config preserved on soft-delete, removed on permanent delete
    Given I set the global config to "FULL"
    And subject "del-ex-cfg" has compatibility level "BACKWARD"
    And subject "del-ex-cfg" has schema:
      """
      {"type":"record","name":"Cfg","fields":[{"name":"a","type":"string"}]}
      """
    When I delete subject "del-ex-cfg"
    Then the response status should be 200
    # Config is preserved after soft-delete (matches Confluent behavior)
    When I get the config for subject "del-ex-cfg"
    Then the response status should be 200
    # Permanent delete removes config
    When I permanently delete subject "del-ex-cfg"
    Then the response status should be 200
    When I get the config for subject "del-ex-cfg"
    Then the response status should be 404
    # Reset global config
    When I set the global config to "NONE"

  Scenario: Subject config persists when individual versions are deleted
    Given the global compatibility level is "NONE"
    And subject "del-ex-cfg-ver" has compatibility level "BACKWARD"
    And subject "del-ex-cfg-ver" has schema:
      """
      {"type":"record","name":"CfgVer","fields":[{"name":"a","type":"string"}]}
      """
    And I set the config for subject "del-ex-cfg-ver" to "NONE"
    And subject "del-ex-cfg-ver" has schema:
      """
      {"type":"record","name":"CfgVer","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":""}]}
      """
    When I delete version 1 of subject "del-ex-cfg-ver"
    And I delete version 2 of subject "del-ex-cfg-ver"
    Then the response status should be 200
    When I get the config for subject "del-ex-cfg-ver"
    Then the response status should be 200

  # ==========================================================================
  # GET latest?deleted=true — when all versions of a subject are soft-deleted,
  # GET versions/latest returns 404. With deleted=true, the highest-versioned
  # soft-deleted schema should be returned.
  # ==========================================================================

  @axonops-only
  Scenario: GET latest returns 404 after all versions soft-deleted
    Given the global compatibility level is "NONE"
    And subject "del-ex-latest-del" has schema:
      """
      {"type":"record","name":"LatDel1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ex-latest-del" has schema:
      """
      {"type":"record","name":"LatDel2","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-ex-latest-del"
    Then the response status should be 200
    When I GET "/subjects/del-ex-latest-del/versions/latest"
    Then the response status should be 404
    When I GET "/subjects/del-ex-latest-del/versions/latest?deleted=true"
    Then the response status should be 200
    And the response field "version" should be 2
    And the response should contain "LatDel2"

  Scenario: GET specific version with deleted=true after soft-delete
    Given subject "del-ex-specific-del" has schema:
      """
      {"type":"record","name":"SpecDel","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/del-ex-specific-del"
    Then the response status should be 200
    When I GET "/subjects/del-ex-specific-del/versions/1"
    Then the response status should be 404
    When I GET "/subjects/del-ex-specific-del/versions/1?deleted=true"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response should contain "SpecDel"

  Scenario: GET specific deleted version with deleted=true while active versions exist
    Given the global compatibility level is "NONE"
    And subject "del-ex-partial-del" has schema:
      """
      {"type":"record","name":"PD1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-ex-partial-del" has schema:
      """
      {"type":"record","name":"PD2","fields":[{"name":"b","type":"string"}]}
      """
    When I DELETE "/subjects/del-ex-partial-del/versions/2"
    Then the response status should be 200
    When I get the latest version of subject "del-ex-partial-del"
    Then the response status should be 200
    And the response field "version" should be 1
    When I GET "/subjects/del-ex-partial-del/versions/2?deleted=true"
    Then the response status should be 200
    And the response field "version" should be 2
    And the response should contain "PD2"
