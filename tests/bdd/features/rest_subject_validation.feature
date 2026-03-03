@functional @analysis
Feature: REST Subject Validation and Matching
  REST endpoints for validating subject names, matching subjects by pattern, and counting subjects.

  # --- POST /subjects/validate ---

  Scenario: Valid topic_name with -value suffix
    When I POST "/subjects/validate" with body:
      """
      {"subject": "my-topic-value", "strategy": "topic_name"}
      """
    Then the response status should be 200
    And the response field "valid" should be true
    And the response field "strategy" should be "topic_name"

  Scenario: Invalid topic_name missing suffix
    When I POST "/subjects/validate" with body:
      """
      {"subject": "my-topic", "strategy": "topic_name"}
      """
    Then the response status should be 200
    And the response field "valid" should be false
    And the response should contain "my-topic-value"

  Scenario: Valid topic_name with -key suffix
    When I POST "/subjects/validate" with body:
      """
      {"subject": "my-topic-key", "strategy": "topic_name"}
      """
    Then the response status should be 200
    And the response field "valid" should be true

  Scenario: Valid record_name strategy
    When I POST "/subjects/validate" with body:
      """
      {"subject": "com.example.User", "strategy": "record_name"}
      """
    Then the response status should be 200
    And the response field "valid" should be true

  Scenario: Invalid record_name strategy
    When I POST "/subjects/validate" with body:
      """
      {"subject": "123.bad", "strategy": "record_name"}
      """
    Then the response status should be 200
    And the response field "valid" should be false

  Scenario: Topic_record_name strategy valid
    When I POST "/subjects/validate" with body:
      """
      {"subject": "topic-com.example.User", "strategy": "topic_record_name"}
      """
    Then the response status should be 200
    And the response field "valid" should be true

  Scenario: Missing subject returns 400
    When I POST "/subjects/validate" with body:
      """
      {"strategy": "topic_name"}
      """
    Then the response status should be 400

  Scenario: Defaults to topic_name strategy
    When I POST "/subjects/validate" with body:
      """
      {"subject": "my-topic-value"}
      """
    Then the response status should be 200
    And the response field "strategy" should be "topic_name"
    And the response field "valid" should be true

  # --- POST /subjects/match ---

  Scenario: Match subjects by regex
    Given subject "match-users-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    And subject "match-orders-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/subjects/match" with body:
      """
      {"pattern": "match-users.*", "mode": "regex"}
      """
    Then the response status should be 200
    And the response field "count" should be 1
    And the response should contain "match-users-value"

  Scenario: Match subjects by glob
    Given subject "glob-alpha-value" has schema:
      """
      {"type":"record","name":"Alpha","fields":[{"name":"id","type":"int"}]}
      """
    And subject "glob-beta-value" has schema:
      """
      {"type":"record","name":"Beta","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/subjects/match" with body:
      """
      {"pattern": "glob-*-value", "mode": "glob"}
      """
    Then the response status should be 200
    And the response field "count" should be 2

  Scenario: Match subjects by fuzzy
    Given subject "fuzzy-users-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/subjects/match" with body:
      """
      {"pattern": "fuzzy-user-value", "mode": "fuzzy", "threshold": 0.5}
      """
    Then the response status should be 200
    And the response should contain "fuzzy-users-value"

  Scenario: Match with no results
    Given subject "nomatch-abc-value" has schema:
      """
      {"type":"record","name":"Abc","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/subjects/match" with body:
      """
      {"pattern": "^zzz.*", "mode": "regex"}
      """
    Then the response status should be 200
    And the response field "count" should be 0

  Scenario: Match with missing pattern returns 400
    When I POST "/subjects/match" with body:
      """
      {"mode": "regex"}
      """
    Then the response status should be 400

  # --- GET /subjects/count ---

  Scenario: Count subjects
    Given subject "count-a-value" has schema:
      """
      {"type":"record","name":"A","fields":[{"name":"id","type":"int"}]}
      """
    And subject "count-b-value" has schema:
      """
      {"type":"record","name":"B","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/subjects/count"
    Then the response status should be 200
    And the response field "count" should be 2
