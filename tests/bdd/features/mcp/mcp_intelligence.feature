@mcp
Feature: MCP AI Intelligence Tools
  MCP tools for schema analysis, field search, similarity, quality scoring, and pattern detection.

  Background:
    Given I register an Avro schema for subject "intel-users-value"

  # --- find_schemas_by_field ---

  Scenario: Find schemas by field name - exact match
    When I call MCP tool "find_schemas_by_field" with input:
      | field | type |
    Then the MCP result should contain "matches"

  Scenario: Find schemas by field name - fuzzy match
    When I call MCP tool "find_schemas_by_field" with JSON input:
      """
      {"field": "typ", "match_type": "fuzzy", "threshold": 0.5}
      """
    Then the MCP result should contain "matches"

  Scenario: Find schemas by field name - regex
    When I call MCP tool "find_schemas_by_field" with JSON input:
      """
      {"field": "^ty.*", "match_type": "regex"}
      """
    Then the MCP result should contain "matches"

  Scenario: Find schemas by field name - no match
    When I call MCP tool "find_schemas_by_field" with input:
      | field | nonexistent_field_xyz |
    Then the MCP result should contain "\"count\":0"

  # --- find_schemas_by_type ---

  Scenario: Find schemas by field type
    When I call MCP tool "find_schemas_by_type" with input:
      | type_pattern | string |
    Then the MCP result should contain "matches"

  Scenario: Find schemas by type with regex
    When I call MCP tool "find_schemas_by_type" with JSON input:
      """
      {"type_pattern": "str.*", "regex": true}
      """
    Then the MCP result should contain "matches"

  # --- find_similar_schemas ---

  Scenario: Find similar schemas
    Given I register an Avro schema for subject "intel-orders-value"
    When I call MCP tool "find_similar_schemas" with JSON input:
      """
      {"subject": "intel-users-value", "threshold": 0.0}
      """
    Then the MCP result should contain "matches"
    And the MCP result should contain "intel-users-value"

  # --- score_schema_quality ---

  Scenario: Score schema quality by subject
    When I call MCP tool "score_schema_quality" with input:
      | subject | intel-users-value |
    Then the MCP result should contain "overall_score"
    And the MCP result should contain "grade"
    And the MCP result should contain "categories"

  Scenario: Score schema quality by inline schema
    When I call MCP tool "score_schema_quality" with JSON input:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Test\",\"namespace\":\"com.example\",\"doc\":\"A test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\",\"doc\":\"ID\"}]}",
        "schema_type": "AVRO"
      }
      """
    Then the MCP result should contain "overall_score"
    And the MCP result should contain "grade"

  # --- check_field_consistency ---

  Scenario: Check field consistency across schemas
    Given I register an Avro schema for subject "intel-products-value"
    When I call MCP tool "check_field_consistency" with input:
      | field | type |
    Then the MCP result should contain "consistent"
    And the MCP result should contain "usages"

  # --- get_schema_complexity ---

  Scenario: Get schema complexity by subject
    When I call MCP tool "get_schema_complexity" with input:
      | subject | intel-users-value |
    Then the MCP result should contain "field_count"
    And the MCP result should contain "grade"

  Scenario: Get schema complexity by inline schema
    When I call MCP tool "get_schema_complexity" with JSON input:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Nested\",\"fields\":[{\"name\":\"addr\",\"type\":{\"type\":\"record\",\"name\":\"Addr\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"}]}}]}",
        "schema_type": "AVRO"
      }
      """
    Then the MCP result should contain "field_count"
    And the MCP result should contain "max_depth"

  # --- detect_schema_patterns ---

  Scenario: Detect schema patterns in registry
    Given I register an Avro schema for subject "intel-events-value"
    When I call MCP tool "detect_schema_patterns"
    Then the MCP result should contain "total_subjects"
    And the MCP result should contain "schema_types"
    And the MCP result should contain "naming_suffixes"
