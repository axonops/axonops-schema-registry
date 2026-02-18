@functional @contexts
Feature: Contexts — Cross-Context Isolation
  Verify that schema registry contexts provide full isolation.
  Operations in one context MUST NOT affect data in another context.
  Schema IDs, subjects, versions, config, and modes are all per-context.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # SCHEMA ID ISOLATION
  # ==========================================================================

  Scenario: Schema IDs are per-context — same schema gets different IDs in different contexts
    When I POST "/subjects/:.ctx-id-a:test-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IdTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "ctx_a_id"
    When I POST "/subjects/:.ctx-id-b:test-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IdTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "ctx_b_id"
    # Both contexts should assign ID 1 independently
    Then the response field "id" should be 1

  Scenario: Schema IDs start at 1 in each new context
    # Register in context A
    When I POST "/subjects/:.fresh-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"FreshA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    # Register a different schema in context A (gets ID 2)
    When I POST "/subjects/:.fresh-a:s2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"FreshA2\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 2
    # Register in context B — should still start at 1
    When I POST "/subjects/:.fresh-b:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"FreshB\",\"fields\":[{\"name\":\"c\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1

  # ==========================================================================
  # VERSION ISOLATION
  # ==========================================================================

  Scenario: Version numbering is independent across contexts
    When I POST "/subjects/:.ver-a:versioned/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Versioned\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ver-a:versioned/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Versioned\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Context A should now have 2 versions
    When I GET "/subjects/:.ver-a:versioned/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    # Same subject in context B starts at version 1
    When I POST "/subjects/:.ver-b:versioned/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"VerB\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.ver-b:versioned/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # SUBJECT ISOLATION
  # ==========================================================================

  Scenario: Subject listing is context-scoped via URL prefix
    When I POST "/subjects/:.subj-a:only-in-a/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"OnlyA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.subj-b:only-in-b/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"OnlyB\",\"fields\":[{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Default context listing should NOT contain subjects from other contexts
    When I GET "/subjects"
    Then the response status should be 200
    And the response body should not contain "only-in-a"
    And the response body should not contain "only-in-b"

  Scenario: Schema by ID is context-scoped
    # Register in context A
    When I POST "/subjects/:.byid-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ByIdA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_a"
    # Register in context B
    When I POST "/subjects/:.byid-b:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ByIdB\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "id_b"
    # Retrieve from context A returns context A schema
    When I GET "/subjects/:.byid-a:s1/versions/1"
    Then the response status should be 200
    And the response body should contain "ByIdA"
    # Retrieve from context B returns context B schema
    When I GET "/subjects/:.byid-b:s1/versions/1"
    Then the response status should be 200
    And the response body should contain "ByIdB"

  # ==========================================================================
  # DELETE ISOLATION
  # ==========================================================================

  Scenario: Delete in one context does not affect another
    When I POST "/subjects/:.del-iso-a:shared/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DelIsoA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.del-iso-b:shared/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DelIsoB\",\"fields\":[{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Delete from context A
    When I DELETE "/subjects/:.del-iso-a:shared"
    Then the response status should be 200
    # Context B is unaffected
    When I GET "/subjects/:.del-iso-b:shared/versions/1"
    Then the response status should be 200
    And the response body should contain "DelIsoB"
    # Context A is deleted
    When I GET "/subjects/:.del-iso-a:shared/versions"
    Then the response status should be 404

  Scenario: Permanent delete in one context does not affect another
    When I POST "/subjects/:.pdel-a:shared/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"PDelA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.pdel-b:shared/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"PDelB\",\"fields\":[{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Soft delete then permanent delete from context A
    When I DELETE "/subjects/:.pdel-a:shared"
    Then the response status should be 200
    When I DELETE "/subjects/:.pdel-a:shared?permanent=true"
    Then the response status should be 200
    # Context B is unaffected
    When I GET "/subjects/:.pdel-b:shared/versions/1"
    Then the response status should be 200
    And the response body should contain "PDelB"

  # ==========================================================================
  # LOOKUP ISOLATION
  # ==========================================================================

  Scenario: Schema lookup is context-scoped
    When I POST "/subjects/:.lookup-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LookupA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Lookup in context A — should find it
    When I POST "/subjects/:.lookup-a:s1" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LookupA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Lookup in context B — should NOT find it
    When I POST "/subjects/:.lookup-b:s1" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LookupA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 404

  Scenario: Soft-delete isolation between contexts
    When I POST "/subjects/:.soft-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SoftA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.soft-b:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SoftB\",\"fields\":[{\"name\":\"b\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Soft-delete context A
    When I DELETE "/subjects/:.soft-a:s1"
    Then the response status should be 200
    # Context A versions are gone
    When I GET "/subjects/:.soft-a:s1/versions"
    Then the response status should be 404
    # Context B still has its version
    When I GET "/subjects/:.soft-b:s1/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  Scenario: Schema fingerprint dedup is per-context
    # Same schema content in two contexts should each get their own ID
    When I POST "/subjects/:.fp-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"FpTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "fp_id_a"
    When I POST "/subjects/:.fp-b:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"FpTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "fp_id_b"
    # Both should have ID 1 (per-context)
    Then the response field "id" should be 1

  Scenario: Default context and named context are isolated
    # Register in default context (no prefix)
    When I POST "/subjects/shared-name/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DefaultS\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "default_id"
    # Register in named context
    When I POST "/subjects/:.named:shared-name/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"NamedS\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Default context version contains DefaultS
    When I GET "/subjects/shared-name/versions/1"
    Then the response status should be 200
    And the response body should contain "DefaultS"
    # Named context version contains NamedS
    When I GET "/subjects/:.named:shared-name/versions/1"
    Then the response status should be 200
    And the response body should contain "NamedS"

  Scenario: List versions is context-scoped
    When I POST "/subjects/:.lv-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LvTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.lv-a:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LvTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.lv-b:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LvB\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Context A has 2 versions
    When I GET "/subjects/:.lv-a:s1/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    # Context B has 1 version
    When I GET "/subjects/:.lv-b:s1/versions"
    Then the response status should be 200
    And the response should be an array of length 1
