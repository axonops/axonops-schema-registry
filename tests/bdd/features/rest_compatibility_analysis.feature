@functional @analysis
Feature: REST Compatibility Analysis
  REST endpoints for multi-subject compatibility checking, suggestions, explanations, and comparisons.

  # --- POST /compatibility/check ---

  Scenario: Compatible schema against one subject
    Given the global compatibility level is "BACKWARD"
    And subject "compat-user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/compatibility/check" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}", "subjects": ["compat-user-value"]}
      """
    Then the response status should be 200
    And the response should contain "is_compatible"

  Scenario: Incompatible schema
    Given the global compatibility level is "BACKWARD"
    And subject "incompat-user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/compatibility/check" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}", "subjects": ["incompat-user-value"]}
      """
    Then the response status should be 200
    And the response should contain "is_compatible"

  Scenario: Check against multiple subjects
    Given the global compatibility level is "NONE"
    And subject "multi-compat-a-value" has schema:
      """
      {"type":"record","name":"A","fields":[{"name":"id","type":"int"}]}
      """
    And subject "multi-compat-b-value" has schema:
      """
      {"type":"record","name":"B","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/compatibility/check" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"A\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}", "subjects": ["multi-compat-a-value", "multi-compat-b-value"]}
      """
    Then the response status should be 200
    And the response field "results" should be an array of length 2

  Scenario: Missing schema returns 400
    When I POST "/compatibility/check" with body:
      """
      {"subjects": ["some-value"]}
      """
    Then the response status should be 400

  Scenario: Empty subjects array returns empty results
    When I POST "/compatibility/check" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}", "subjects": []}
      """
    Then the response status should be 200
    And the response field "results" should be an array of length 0

  # --- POST /compatibility/subjects/{subject}/suggest ---

  Scenario: Suggest for BACKWARD compat
    Given the global compatibility level is "BACKWARD"
    And subject "suggest-bw-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/compatibility/subjects/suggest-bw-value/suggest" with body:
      """
      {}
      """
    Then the response status should be 200
    And the response should contain "Add new fields with default values"
    And the response field "compatibility_level" should be "BACKWARD"

  Scenario: Suggest for FORWARD compat
    Given subject "suggest-fw-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    And subject "suggest-fw-value" has compatibility level "FORWARD"
    When I POST "/compatibility/subjects/suggest-fw-value/suggest" with body:
      """
      {}
      """
    Then the response status should be 200
    And the response should contain "Remove fields"

  Scenario: Suggest for NONE compat
    And subject "suggest-none-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    And subject "suggest-none-value" has compatibility level "NONE"
    When I POST "/compatibility/subjects/suggest-none-value/suggest" with body:
      """
      {}
      """
    Then the response status should be 200
    And the response should contain "Any change is allowed"

  Scenario: Suggest for nonexistent subject falls back to BACKWARD
    When I POST "/compatibility/subjects/nonexistent-suggest-value/suggest" with body:
      """
      {}
      """
    Then the response status should be 200
    And the response should contain "Add new fields with default values"

  # --- POST /compatibility/subjects/{subject}/explain ---

  Scenario: Explain compatible schema
    Given the global compatibility level is "BACKWARD"
    And subject "explain-ok-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/compatibility/subjects/explain-ok-value/explain" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: Explain incompatible schema
    Given the global compatibility level is "BACKWARD"
    And subject "explain-fail-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/compatibility/subjects/explain-fail-value/explain" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false

  Scenario: Explain missing schema returns 400
    Given subject "explain-noschema-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/compatibility/subjects/explain-noschema-value/explain" with body:
      """
      {"schemaType": "AVRO"}
      """
    Then the response status should be 400

  # --- POST /compatibility/compare ---

  Scenario: Compare two subjects with shared fields
    Given subject "compare-a-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "compare-b-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"double"}]}
      """
    When I POST "/compatibility/compare" with body:
      """
      {"subject1": "compare-a-value", "subject2": "compare-b-value"}
      """
    Then the response status should be 200
    And the response should have field "shared"
    And the response should have field "only_in_sub1"
    And the response should have field "only_in_sub2"

  Scenario: Compare with missing subject returns 400
    When I POST "/compatibility/compare" with body:
      """
      {"subject1": "only-one-value"}
      """
    Then the response status should be 400
