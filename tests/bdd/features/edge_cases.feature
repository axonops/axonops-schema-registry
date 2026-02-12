@functional
Feature: Edge Cases and Boundary Conditions
  Test special characters in subject names, malformed inputs, pagination boundaries,
  idempotent registration, and compatibility check edge cases.

  # ==========================================================================
  # SPECIAL CHARACTERS IN SUBJECT NAMES
  # ==========================================================================

  Scenario: Subject name with dots works
    When I register a schema under subject "com.example.Topic-value":
      """
      {"type":"record","name":"Dotted","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "com.example.Topic-value"
    Then the response status should be 200
    And the response field "subject" should be "com.example.Topic-value"

  Scenario: Subject name with underscores works
    When I register a schema under subject "my_topic_value":
      """
      {"type":"record","name":"Underscore","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "my_topic_value"
    Then the response status should be 200
    And the response field "subject" should be "my_topic_value"

  Scenario: Subject name with dashes works
    When I register a schema under subject "my-topic-value":
      """
      {"type":"record","name":"Dashed","fields":[{"name":"x","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "my-topic-value"
    Then the response status should be 200
    And the response field "subject" should be "my-topic-value"

  # ==========================================================================
  # MALFORMED REQUEST BODIES
  # ==========================================================================

  Scenario: Register with empty JSON body returns error
    When I POST "/subjects/edge-empty-body/versions" with body:
      """
      {}
      """
    Then the response status should be 422

  Scenario: PUT /config with empty body returns error
    When I PUT "/config" with body:
      """
      {}
      """
    Then the response status should be 422

  Scenario: PUT /config with invalid compatibility level returns error
    When I PUT "/config" with body:
      """
      {"compatibility": "INVALID_LEVEL"}
      """
    Then the response status should be 422
    And the response should have error code 42203

  Scenario: PUT /mode with empty body returns error
    When I PUT "/mode" with body:
      """
      {}
      """
    Then the response status should be 422

  Scenario: PUT /mode with invalid mode value returns error
    When I PUT "/mode" with body:
      """
      {"mode": "INVALID_MODE"}
      """
    Then the response status should be 422
    And the response should have error code 42204

  # ==========================================================================
  # PAGINATION EDGE CASES (GET /schemas)
  # ==========================================================================

  Scenario: GET /schemas with large offset returns empty array
    Given subject "edge-page-a" has schema:
      """
      {"type":"record","name":"PageA","fields":[{"name":"x","type":"string"}]}
      """
    When I GET "/schemas?offset=9999"
    Then the response status should be 200
    And the response should be an array of length 0

  Scenario: GET /schemas with offset=0 and limit=1 returns exactly 1 result
    Given subject "edge-page-b1" has schema:
      """
      {"type":"record","name":"PageB1","fields":[{"name":"x","type":"string"}]}
      """
    And subject "edge-page-b2" has schema:
      """
      {"type":"record","name":"PageB2","fields":[{"name":"y","type":"string"}]}
      """
    When I GET "/schemas?offset=0&limit=1"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # IDEMPOTENT REGISTRATION
  # ==========================================================================

  Scenario: Registering same schema twice returns same ID and version
    When I register a schema under subject "edge-idempotent":
      """
      {"type":"record","name":"Idem","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    When I register a schema under subject "edge-idempotent":
      """
      {"type":"record","name":"Idem","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "second_id"
    When I list versions of subject "edge-idempotent"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # COMPATIBILITY CHECK AGAINST SPECIFIC VERSION NUMBERS
  # ==========================================================================

  Scenario: Compatibility check against specific version 1
    Given the global compatibility level is "BACKWARD"
    And subject "edge-compat-ver" has schema:
      """
      {"type":"record","name":"Compat","fields":[{"name":"a","type":"string"}]}
      """
    When I check compatibility of schema against subject "edge-compat-ver" version 1:
      """
      {"type":"record","name":"Compat","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":"x"}]}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: Compatibility check against non-existent version returns 404
    Given subject "edge-compat-404" has schema:
      """
      {"type":"record","name":"Compat404","fields":[{"name":"a","type":"string"}]}
      """
    When I check compatibility of schema against subject "edge-compat-404" version 99:
      """
      {"type":"record","name":"Compat404","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 404
    And the response should have error code 40402

  Scenario: Compatibility check against non-existent subject returns 404
    When I POST "/compatibility/subjects/edge-no-such-subject/versions/1" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Ghost\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 404
    And the response should have error code 40401

  # ==========================================================================
  # VERSION BOUNDARY TESTS
  # ==========================================================================

  Scenario: GET version 0 returns error
    Given subject "edge-ver-zero" has schema:
      """
      {"type":"record","name":"VerZero","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/edge-ver-zero/versions/0"
    Then the response status should be 422
    And the response should have error code 42202

  Scenario: GET version -2 returns error (only -1 is valid)
    Given subject "edge-ver-neg" has schema:
      """
      {"type":"record","name":"VerNeg","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/edge-ver-neg/versions/-2"
    Then the response status should be 422
    And the response should have error code 42202

  Scenario: GET version with non-numeric string returns error
    Given subject "edge-ver-abc" has schema:
      """
      {"type":"record","name":"VerAbc","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/edge-ver-abc/versions/abc"
    Then the response status should be 422
    And the response should have error code 42202

  Scenario: GET version -1 works like latest
    Given the global compatibility level is "NONE"
    And subject "edge-ver-minus1" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "edge-ver-minus1" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I GET "/subjects/edge-ver-minus1/versions/-1"
    Then the response status should be 200
    And the response field "version" should be 2
