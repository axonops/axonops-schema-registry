@functional @analysis
Feature: REST Schema Search
  REST endpoints for searching schemas by content, field name, and field type.

  # --- POST /schemas/search ---

  Scenario: Search by substring finds matching schema
    Given subject "search-user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/schemas/search" with body:
      """
      {"query": "User"}
      """
    Then the response status should be 200
    And the response field "count" should be 1
    And the response should contain "search-user-value"

  Scenario: Search with no matches returns empty
    Given subject "search-empty-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/schemas/search" with body:
      """
      {"query": "NonexistentSchema"}
      """
    Then the response status should be 200
    And the response field "count" should be 0

  Scenario: Search with regex
    Given subject "search-regex-value" has schema:
      """
      {"type":"record","name":"UserEvent","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/schemas/search" with body:
      """
      {"query": "User.*", "regex": true}
      """
    Then the response status should be 200
    And the response field "count" should be 1

  Scenario: Search with invalid regex returns 400
    When I POST "/schemas/search" with body:
      """
      {"query": "[invalid", "regex": true}
      """
    Then the response status should be 400

  Scenario: Search with missing query returns 400
    When I POST "/schemas/search" with body:
      """
      {"regex": false}
      """
    Then the response status should be 400

  Scenario: Search respects limit
    Given subject "search-limit1-value" has schema:
      """
      {"type":"record","name":"Alpha","fields":[{"name":"id","type":"int"}]}
      """
    And subject "search-limit2-value" has schema:
      """
      {"type":"record","name":"Beta","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/schemas/search" with body:
      """
      {"query": "id", "limit": 1}
      """
    Then the response status should be 200
    And the response field "count" should be 1

  # --- POST /schemas/search/field ---

  Scenario: Find field by exact match
    Given subject "field-search-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/schemas/search/field" with body:
      """
      {"field": "name"}
      """
    Then the response status should be 200
    And the response field "count" should be 1
    And the response field "mode" should be "exact"

  Scenario: Find field by fuzzy match
    Given subject "field-fuzzy-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"username","type":"string"}]}
      """
    When I POST "/schemas/search/field" with body:
      """
      {"field": "user_name", "mode": "fuzzy", "threshold": 0.3}
      """
    Then the response status should be 200
    And the response field "mode" should be "fuzzy"

  Scenario: Find field by regex
    Given subject "field-regex-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"user_id","type":"int"},{"name":"user_name","type":"string"}]}
      """
    When I POST "/schemas/search/field" with body:
      """
      {"field": "^user_.*", "mode": "regex"}
      """
    Then the response status should be 200
    And the response field "mode" should be "regex"

  Scenario: Find field with no matches
    Given subject "field-nomatch-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/schemas/search/field" with body:
      """
      {"field": "nonexistent_field_xyz"}
      """
    Then the response status should be 200
    And the response field "count" should be 0

  Scenario: Find field with missing field returns 400
    When I POST "/schemas/search/field" with body:
      """
      {"mode": "exact"}
      """
    Then the response status should be 400

  Scenario: Find field defaults to exact mode
    Given subject "field-default-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/schemas/search/field" with body:
      """
      {"field": "id"}
      """
    Then the response status should be 200
    And the response field "mode" should be "exact"

  # --- POST /schemas/search/type ---

  Scenario: Find fields by type
    Given subject "type-search-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/schemas/search/type" with body:
      """
      {"type_pattern": "string"}
      """
    Then the response status should be 200
    And the response should contain "name"

  Scenario: Find by type with regex
    Given subject "type-regex-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/schemas/search/type" with body:
      """
      {"type_pattern": "str.*", "regex": true}
      """
    Then the response status should be 200
    And the response should contain "name"

  Scenario: Find by type with no matches
    Given subject "type-nomatch-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/schemas/search/type" with body:
      """
      {"type_pattern": "boolean"}
      """
    Then the response status should be 200
    And the response field "count" should be 0

  Scenario: Find by type with missing type_pattern returns 400
    When I POST "/schemas/search/type" with body:
      """
      {"regex": false}
      """
    Then the response status should be 400
