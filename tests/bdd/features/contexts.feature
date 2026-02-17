@functional
Feature: Contexts (Multi-Tenancy)
  Contexts allow multi-tenant schema isolation. Subjects are assigned to
  contexts using the :.contextname.:subject prefix format. The default
  context is "." which is used for subjects without a prefix.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # GET /contexts — DEFAULT
  # ==========================================================================

  Scenario: Default context returned when no context-prefixed subjects exist
    When I GET "/contexts"
    Then the response status should be 200
    And the response body should contain "."

  # ==========================================================================
  # CONTEXT-PREFIXED SUBJECT REGISTRATION
  # ==========================================================================

  Scenario: Register schema with context prefix — context appears in GET /contexts
    When I POST "/subjects/:.testctx.:ctx-subject1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CtxTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response body should contain ".testctx."

  Scenario: Register schemas in multiple contexts — all contexts listed
    When I POST "/subjects/:.ctx-alpha.:multi-ctx/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AlphaCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx-beta.:multi-ctx/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"BetaCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response body should contain ".ctx-alpha."
    And the response body should contain ".ctx-beta."

  # ==========================================================================
  # CONTEXT ISOLATION — SUBJECTS IN DIFFERENT CONTEXTS ARE INDEPENDENT
  # ==========================================================================

  Scenario: Same subject name in different contexts are independent
    # Register in context "iso-a"
    When I POST "/subjects/:.iso-a.:shared-name/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IsoA\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Register in context "iso-b"
    When I POST "/subjects/:.iso-b.:shared-name/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IsoB\",\"fields\":[{\"name\":\"y\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Each context has its own version 1
    When I GET "/subjects/:.iso-a.:shared-name/versions/1"
    Then the response status should be 200
    And the response body should contain "IsoA"
    When I GET "/subjects/:.iso-b.:shared-name/versions/1"
    Then the response status should be 200
    And the response body should contain "IsoB"

  # ==========================================================================
  # GET/LIST OPERATIONS WITH CONTEXT PREFIX
  # ==========================================================================

  Scenario: List versions for context-prefixed subject
    When I POST "/subjects/:.list-ctx.:versioned/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ListCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.list-ctx.:versioned/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  Scenario: List subjects includes context-prefixed subjects
    When I POST "/subjects/:.subjlist-ctx.:findme/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SubjList\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects"
    Then the response status should be 200
    And the response body should contain ":.subjlist-ctx.:findme"

  # ==========================================================================
  # COMPATIBILITY WITHIN CONTEXT
  # ==========================================================================

  Scenario: Compatibility check works within context-prefixed subject
    When I POST "/subjects/:.compat-ctx.:compat-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CompatCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I set the config for subject ":.compat-ctx.:compat-test" to "BACKWARD"
    When I POST "/compatibility/subjects/:.compat-ctx.:compat-test/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CompatCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # DELETE CONTEXT-PREFIXED SUBJECT
  # ==========================================================================

  Scenario: Delete context-prefixed subject
    When I POST "/subjects/:.del-ctx.:to-delete/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DelCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/:.del-ctx.:to-delete"
    Then the response status should be 200
    When I GET "/subjects/:.del-ctx.:to-delete/versions"
    Then the response status should be 404
