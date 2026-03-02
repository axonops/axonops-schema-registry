@mcp
Feature: MCP Validation, Export, and Statistics Tools
  MCP tools for validating schemas, exporting data, and getting registry statistics.

  # --- validate_schema ---

  Scenario: Validate a valid Avro schema
    When I call MCP tool "validate_schema" with JSON input:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the MCP result should contain "\"valid\":true"
    And the MCP result should contain "fingerprint"

  Scenario: Validate an invalid Avro schema
    When I call MCP tool "validate_schema" with JSON input:
      """
      {"schema": "{\"type\":\"invalid_type\"}"}
      """
    Then the MCP result should contain "\"valid\":false"
    And the MCP result should contain "error"

  Scenario: Validate a valid JSON Schema
    When I call MCP tool "validate_schema" with JSON input:
      """
      {"schema": "{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"integer\"}}}", "schema_type": "JSON"}
      """
    Then the MCP result should contain "\"valid\":true"

  # --- normalize_schema ---

  Scenario: Normalize an Avro schema
    When I call MCP tool "normalize_schema" with JSON input:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the MCP result should contain "normalized"
    And the MCP result should contain "fingerprint"

  # --- validate_subject_name ---

  Scenario: Validate subject name with topic_name strategy - valid
    When I call MCP tool "validate_subject_name" with input:
      | subject  | my-topic-value |
      | strategy | topic_name     |
    Then the MCP result should contain "\"valid\":true"

  Scenario: Validate subject name with topic_name strategy - invalid
    When I call MCP tool "validate_subject_name" with input:
      | subject  | my-topic       |
      | strategy | topic_name     |
    Then the MCP result should contain "\"valid\":false"

  Scenario: Validate subject name with record_name strategy
    When I call MCP tool "validate_subject_name" with input:
      | subject  | com.example.User |
      | strategy | record_name      |
    Then the MCP result should contain "\"valid\":true"

  # --- search_schemas ---

  Scenario: Search schemas by content
    Given I register an Avro schema for subject "search-val-test"
    When I call MCP tool "search_schemas" with input:
      | pattern | string |
    Then the MCP result should contain "search-val-test"

  Scenario: Search schemas with no matches
    When I call MCP tool "search_schemas" with input:
      | pattern | nonexistent_pattern_xyz |
    Then the MCP result should contain "\"count\":0"

  # --- get_schema_history ---

  Scenario: Get schema history for a subject
    Given I register an Avro schema for subject "history-val-test"
    When I call MCP tool "get_schema_history" with input:
      | subject | history-val-test |
    Then the MCP result should contain "history-val-test"
    And the MCP result should contain "\"count\":1"

  # --- export_schema ---

  Scenario: Export a single schema version
    Given I register an Avro schema for subject "export-val-test"
    When I call MCP tool "export_schema" with input:
      | subject | export-val-test |
      | version | 1               |
    Then the MCP result should contain "export-val-test"
    And the MCP result should contain "schema_type"

  # --- export_subject ---

  Scenario: Export all versions for a subject
    Given I register an Avro schema for subject "export-subj-val-test"
    When I call MCP tool "export_subject" with input:
      | subject | export-subj-val-test |
    Then the MCP result should contain "export-subj-val-test"
    And the MCP result should contain "\"count\":1"

  # --- get_registry_statistics ---

  Scenario: Get registry statistics
    Given I register an Avro schema for subject "stats-val-test"
    When I call MCP tool "get_registry_statistics"
    Then the MCP result should contain "total_subjects"
    And the MCP result should contain "total_versions"

  # --- count_versions ---

  Scenario: Count versions for a subject
    Given I register an Avro schema for subject "count-val-test"
    When I call MCP tool "count_versions" with input:
      | subject | count-val-test |
    Then the MCP result should contain "\"count\":1"

  # --- count_subjects ---

  Scenario: Count subjects in the registry
    Given I register an Avro schema for subject "count-subj-val-test"
    When I call MCP tool "count_subjects"
    Then the MCP result should contain "count"
