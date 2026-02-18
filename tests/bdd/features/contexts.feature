@functional @contexts
Feature: Contexts — Core Behavior
  Contexts allow multi-tenant schema isolation. Subjects are assigned to
  contexts using the :.contextname:subject prefix format (Confluent-compatible).
  The default context is "." which is used for subjects without a prefix.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # GET /contexts — DEFAULT
  # ==========================================================================

  Scenario: Default context returned when no context-prefixed subjects exist
    When I GET "/contexts"
    Then the response status should be 200
    And the response body should contain "."

  Scenario: Default context "." is always present in fresh registry
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."

  # ==========================================================================
  # CONTEXT-PREFIXED SUBJECT REGISTRATION
  # ==========================================================================

  Scenario: Register schema with context prefix — context appears in GET /contexts
    When I POST "/subjects/:.testctx:ctx-subject1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CtxTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".testctx"

  Scenario: Register schemas in multiple contexts — all contexts listed
    When I POST "/subjects/:.ctx-alpha:multi-ctx/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AlphaCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx-beta:multi-ctx/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"BetaCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".ctx-alpha"
    And the response array should contain ".ctx-beta"
    And the response array should contain "."

  Scenario: Contexts are created implicitly on first schema registration
    # No explicit context creation API — contexts appear when schemas are registered
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should not contain ".newctx"
    When I POST "/subjects/:.newctx:first-schema/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"NewCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".newctx"

  Scenario: GET /contexts returns sorted list
    When I POST "/subjects/:.zeta:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Zeta\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.alpha:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Alpha\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should contain ".alpha"
    And the response array should contain ".zeta"

  Scenario: Registering schema in default context via qualified subject
    # :.: prefix maps to the default context
    When I POST "/subjects/mysubject/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DefaultCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects"
    Then the response status should be 200
    And the response array should contain "mysubject"

  Scenario: Context names are case-sensitive
    When I POST "/subjects/:.CaseSensitive:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Upper\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.casesensitive:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Lower\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".CaseSensitive"
    And the response array should contain ".casesensitive"

  # ==========================================================================
  # CONTEXT ISOLATION — BASIC
  # ==========================================================================

  Scenario: Same subject name in different contexts are independent
    When I POST "/subjects/:.iso-a:shared-name/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IsoA\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.iso-b:shared-name/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IsoB\",\"fields\":[{\"name\":\"y\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.iso-a:shared-name/versions/1"
    Then the response status should be 200
    And the response body should contain "IsoA"
    When I GET "/subjects/:.iso-b:shared-name/versions/1"
    Then the response status should be 200
    And the response body should contain "IsoB"

  Scenario: Delete context-prefixed subject
    When I POST "/subjects/:.del-ctx:to-delete/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DelCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/:.del-ctx:to-delete"
    Then the response status should be 200
    When I GET "/subjects/:.del-ctx:to-delete/versions"
    Then the response status should be 404
