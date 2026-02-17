@functional
Feature: Contexts — Edge Cases and Error Conditions
  Verify error handling and boundary conditions for context operations.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # VALID CONTEXT NAMES
  # ==========================================================================

  Scenario: Context name with dash is valid
    When I POST "/subjects/:.my-context:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DashCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".my-context"

  Scenario: Context name with underscore is valid
    When I POST "/subjects/:.my_context:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UnderCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".my_context"

  Scenario: Context name with numbers is valid
    When I POST "/subjects/:.ctx123:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"NumCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".ctx123"

  Scenario: Context name with mixed case is valid
    When I POST "/subjects/:.MyContext:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MixedCtx\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".MyContext"

  # ==========================================================================
  # SCHEMA DEDUP WITHIN CONTEXT
  # ==========================================================================

  Scenario: Same schema registered in same subject returns existing
    When I POST "/subjects/:.dedup-ctx:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Dedup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    When I POST "/subjects/:.dedup-ctx:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Dedup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"

  Scenario: Same schema in different subjects within same context shares ID
    When I POST "/subjects/:.shared-id:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SharedId\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "shared_id"
    When I POST "/subjects/:.shared-id:s2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SharedId\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "shared_id"

  # ==========================================================================
  # OPERATIONS ON NON-EXISTENT CONTEXT DATA
  # ==========================================================================

  Scenario: Get versions for non-existent subject in context returns 404
    When I GET "/subjects/:.nonexist-ctx:no-such-subject/versions"
    Then the response status should be 404

  Scenario: Get specific version for non-existent subject in context returns 404
    When I GET "/subjects/:.nonexist-ctx2:no-such/versions/1"
    Then the response status should be 404

  Scenario: Delete non-existent subject in context returns 404
    When I DELETE "/subjects/:.nonexist-ctx3:no-such"
    Then the response status should be 404

  Scenario: Config for non-existent subject in context returns 404
    When I GET "/config/:.nonexist-ctx4:no-such"
    Then the response status should be 404

  Scenario: Multiple schemas in same context get sequential IDs
    When I POST "/subjects/:.seq-ctx:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Seq1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    When I POST "/subjects/:.seq-ctx:s2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Seq2\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 2
    When I POST "/subjects/:.seq-ctx:s3/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Seq3\",\"fields\":[{\"name\":\"c\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 3
