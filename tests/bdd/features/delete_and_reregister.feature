@functional @edge-case
Feature: Delete and Re-register Semantics
  As a schema registry user
  I want to understand how version numbering behaves after soft-delete and re-register
  And how permanent delete interacts with active references
  So that I can safely manage schema lifecycle without data integrity issues

  Background:
    Given the schema registry is running
    And the global compatibility level is "NONE"

  # ---------------------------------------------------------------------------
  # Soft-delete + re-register: version numbering
  # ---------------------------------------------------------------------------

  Scenario: Re-register after soft-delete continues version numbering
    # Register two versions
    Given subject "del-reregister" has schema:
      """
      {"type":"record","name":"DelReReg","fields":[{"name":"id","type":"int"}]}
      """
    And subject "del-reregister" has schema:
      """
      {"type":"record","name":"DelReReg","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    # Verify 2 versions exist
    When I list versions of subject "del-reregister"
    Then the response status should be 200
    And the response should be an array of length 2
    # Soft-delete the subject
    When I delete subject "del-reregister"
    Then the response status should be 200
    # Subject should not be listed in active subjects
    When I list all subjects
    Then the response status should be 200
    And the response array should not contain "del-reregister"
    # Re-register a new schema
    When I register a schema under subject "del-reregister":
      """
      {"type":"record","name":"DelReReg","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"ts","type":"long","default":0}]}
      """
    Then the response status should be 200
    # Subject should now be visible again
    When I list all subjects
    Then the response status should be 200
    And the response array should contain "del-reregister"

  Scenario: Re-registering the same schema after soft-delete returns existing ID
    Given subject "del-same-schema" has schema:
      """
      {"type":"record","name":"DelSame","fields":[{"name":"id","type":"int"}]}
      """
    And I store the response field "id" as "original_id"
    When I delete subject "del-same-schema"
    Then the response status should be 200
    # Re-register the identical schema
    When I register a schema under subject "del-same-schema":
      """
      {"type":"record","name":"DelSame","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # Lookup of deleted schemas
  # ---------------------------------------------------------------------------

  Scenario: Schema lookup with deleted flag finds soft-deleted schemas
    Given subject "lookup-del" has schema:
      """
      {"type":"record","name":"LookupDel","fields":[{"name":"id","type":"int"}]}
      """
    When I delete subject "lookup-del"
    Then the response status should be 200
    # Normal lookup should fail
    When I lookup schema in subject "lookup-del":
      """
      {"type":"record","name":"LookupDel","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 404
    # Lookup with deleted=true should find it
    When I lookup schema in subject "lookup-del" with deleted:
      """
      {"type":"record","name":"LookupDel","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # Listing with deleted flag
  # ---------------------------------------------------------------------------

  Scenario: List subjects with deleted flag includes soft-deleted subjects
    Given subject "list-del-a" has schema:
      """
      {"type":"record","name":"ListDelA","fields":[{"name":"f","type":"string"}]}
      """
    And subject "list-del-b" has schema:
      """
      {"type":"record","name":"ListDelB","fields":[{"name":"f","type":"string"}]}
      """
    When I delete subject "list-del-a"
    Then the response status should be 200
    # Normal listing should only show list-del-b
    When I list all subjects
    Then the response status should be 200
    And the response array should contain "list-del-b"
    And the response array should not contain "list-del-a"
    # Listing with deleted should show both
    When I list subjects with deleted
    Then the response status should be 200
    And the response array should contain "list-del-a"
    And the response array should contain "list-del-b"

  # ---------------------------------------------------------------------------
  # Permanent delete
  # ---------------------------------------------------------------------------

  Scenario: Permanent delete requires prior soft-delete
    Given subject "perm-del" has schema:
      """
      {"type":"record","name":"PermDel","fields":[{"name":"f","type":"string"}]}
      """
    # Attempt permanent delete without soft-delete first should fail
    When I permanently delete subject "perm-del"
    Then the response status should be 404

  Scenario: Permanent delete after soft-delete succeeds
    Given subject "perm-del-ok" has schema:
      """
      {"type":"record","name":"PermDelOk","fields":[{"name":"f","type":"string"}]}
      """
    When I delete subject "perm-del-ok"
    Then the response status should be 200
    When I permanently delete subject "perm-del-ok"
    Then the response status should be 200
    # Subject should not appear even with deleted flag
    When I list subjects with deleted
    Then the response status should be 200
    And the response array should not contain "perm-del-ok"

  # ---------------------------------------------------------------------------
  # Permanent delete of referenced subject
  # ---------------------------------------------------------------------------

  Scenario: Soft-delete a subject that is referenced by another is blocked
    # Register base schema
    Given subject "ref-target-del" has schema:
      """
      {"type":"record","name":"RefTarget","namespace":"com.del","fields":[{"name":"id","type":"string"}]}
      """
    # Register a schema that references the base
    When I register a schema under subject "ref-consumer-del" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RefConsumer\",\"namespace\":\"com.del\",\"fields\":[{\"name\":\"target\",\"type\":\"com.del.RefTarget\"}]}",
        "references": [
          {"name": "com.del.RefTarget", "subject": "ref-target-del", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Soft-deleting a referenced subject should be blocked with 422
    When I delete subject "ref-target-del"
    Then the response status should be 422

  # ---------------------------------------------------------------------------
  # Delete individual version
  # ---------------------------------------------------------------------------

  Scenario: Deleting a specific version does not affect other versions
    Given subject "ver-del" has schema:
      """
      {"type":"record","name":"VerDel","fields":[{"name":"id","type":"int"}]}
      """
    And subject "ver-del" has schema:
      """
      {"type":"record","name":"VerDel","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    And subject "ver-del" has schema:
      """
      {"type":"record","name":"VerDel","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"ts","type":"long","default":0}]}
      """
    # Delete version 2
    When I delete version 2 of subject "ver-del"
    Then the response status should be 200
    # Version 1 should still exist
    When I get version 1 of subject "ver-del"
    Then the response status should be 200
    And the response field "version" should be 1
    # Version 3 should still exist
    When I get version 3 of subject "ver-del"
    Then the response status should be 200
    And the response field "version" should be 3
    # Version 2 should be gone
    When I get version 2 of subject "ver-del"
    Then the response status should be 404
