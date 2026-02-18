@functional
Feature: Contexts — Name Validation, Error Conditions, and Edge Cases
  Verify that context name validation rejects malformed input via the URL prefix
  routing, that valid names with all allowed character types are accepted, and
  that error conditions for non-existent subjects within valid contexts are
  handled correctly.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # INVALID CONTEXT NAMES VIA URL PREFIX (422)
  # ==========================================================================

  Scenario: Context name with exclamation mark via URL prefix is rejected
    When I GET "/contexts/.invalid!/subjects"
    Then the response status should be 422
    And the response should have error code 422
    And the response body should contain "Invalid context name"

  Scenario: Context name with space via URL prefix is rejected
    When I GET "/contexts/.has%20space/subjects"
    Then the response status should be 422
    And the response should have error code 422
    And the response body should contain "Invalid context name"

  Scenario: Context name with at-sign via URL prefix is rejected
    When I GET "/contexts/.at@sign/subjects"
    Then the response status should be 422
    And the response should have error code 422
    And the response body should contain "Invalid context name"

  Scenario: Context name with hash via URL prefix is rejected
    When I GET "/contexts/.hash%23char/subjects"
    Then the response status should be 422
    And the response should have error code 422
    And the response body should contain "Invalid context name"

  Scenario: Context name with percent-encoded special characters via URL prefix is rejected
    When I GET "/contexts/.pct%25enc/subjects"
    Then the response status should be 422
    And the response should have error code 422
    And the response body should contain "Invalid context name"

  # ==========================================================================
  # VALID CONTEXT NAMES
  # ==========================================================================

  Scenario: Valid context name with numbers
    When I POST "/subjects/:.ctx123:valid-num/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ValidNum\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".ctx123"

  Scenario: Valid context name with all valid character types
    When I POST "/subjects/:.My-Ctx_v2.0:all-chars/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AllChars\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".My-Ctx_v2.0"

  # ==========================================================================
  # DEFAULT CONTEXT BEHAVIOR
  # ==========================================================================

  Scenario: Default context is accessible without context prefix
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Register in default context using plain subject
    When I POST "/subjects/plain-val/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"PlainVal\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects"
    Then the response status should be 200
    And the response array should contain "plain-val"
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."

  # ==========================================================================
  # INVALID CONTEXT ATTEMPTS DO NOT CREATE CONTEXTS
  # ==========================================================================

  Scenario: Multiple invalid context attempts do not create contexts
    When I GET "/contexts/.bad!/subjects"
    Then the response status should be 422
    When I GET "/contexts/.no@good/subjects"
    Then the response status should be 422
    When I GET "/contexts/.sp%20ace/subjects"
    Then the response status should be 422
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should not contain ".bad!"
    And the response array should not contain ".no@good"
    And the response array should not contain ".sp ace"

  # ==========================================================================
  # IMPLICIT CONTEXT CREATION
  # ==========================================================================

  Scenario: Schema registration with valid context creates context implicitly
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should not contain ".new-ctx"
    When I POST "/subjects/:.new-ctx:impl-create/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImplCreate\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should contain ".new-ctx"

  # ==========================================================================
  # ERROR CONDITIONS FOR NON-EXISTENT RESOURCES IN VALID CONTEXTS
  # ==========================================================================

  Scenario: Accessing non-existent subject in valid context returns 404
    When I GET "/subjects/:.valid-ctx:nonexistent/versions"
    Then the response status should be 404

  Scenario: Accessing non-existent version in valid context returns 404
    When I POST "/subjects/:.val13:test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Val13\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.val13:test/versions/99"
    Then the response status should be 404

  Scenario: Delete non-existent subject in context returns 404
    When I DELETE "/subjects/:.val14:doesnotexist"
    Then the response status should be 404

  Scenario: Compatibility check on non-existent subject returns 404
    When I POST "/compatibility/subjects/:.val15:nosuch/versions/1" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Compat15\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 404

  Scenario: Config for non-existent subject returns 404
    When I GET "/config/:.val16:nosuch"
    Then the response status should be 404

  Scenario: Mode for non-existent subject returns 404
    When I GET "/mode/:.val17:nosuch"
    Then the response status should be 404
