@mcp
Feature: MCP Comparison and Search Tools
  MCP tools for comparing schemas, matching subjects, and explaining compatibility.

  # --- match_subjects ---

  Scenario: Match subjects by substring
    Given I register an Avro schema for subject "cmp-alpha-value"
    And I register an Avro schema for subject "cmp-beta-value"
    And I register an Avro schema for subject "other-gamma"
    When I call MCP tool "match_subjects" with input:
      | pattern | cmp- |
    Then the MCP result should contain "cmp-alpha-value"
    And the MCP result should contain "cmp-beta-value"
    And the MCP result should not contain "other-gamma"

  Scenario: Match subjects by regex
    Given I register an Avro schema for subject "rx-alpha-value"
    And I register an Avro schema for subject "rx-beta-key"
    When I call MCP tool "match_subjects" with input:
      | pattern | ^rx-.*-value$ |
      | regex   | true          |
    Then the MCP result should contain "rx-alpha-value"
    And the MCP result should not contain "rx-beta-key"

  Scenario: Match subjects with no matches
    When I call MCP tool "match_subjects" with input:
      | pattern | nonexistent_subject_pattern |
    Then the MCP result should contain "\"count\":0"

  # --- suggest_compatible_change ---

  Scenario: Suggest compatible change - add field
    Given I register an Avro schema for subject "suggest-cmp-test"
    When I call MCP tool "suggest_compatible_change" with input:
      | subject     | suggest-cmp-test |
      | change_type | add_field        |
    Then the MCP result should contain "advice"
    And the MCP result should contain "BACKWARD"

  Scenario: Suggest compatible change - rename field
    Given I register an Avro schema for subject "suggest-rename-test"
    When I call MCP tool "suggest_compatible_change" with input:
      | subject     | suggest-rename-test |
      | change_type | rename_field        |
    Then the MCP result should contain "advice"
    And the MCP result should contain "alias"

  # --- check_compatibility_multi ---

  Scenario: Check compatibility against multiple subjects
    Given I register an Avro schema for subject "multi-compat-1"
    When I call MCP tool "check_compatibility_multi" with JSON input:
      """
      {
        "subjects": ["multi-compat-1"],
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "all_compatible"
    And the MCP result should contain "results"

  # --- explain_compatibility_failure ---

  Scenario: Explain compatibility for a compatible schema
    Given I register an Avro schema for subject "explain-cmp-test"
    When I call MCP tool "explain_compatibility_failure" with JSON input:
      """
      {
        "subject": "explain-cmp-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "is_compatible"

  # --- diff_schemas ---

  Scenario: Diff two schema versions
    Given I register an Avro schema for subject "diff-cmp-test"
    When I call MCP tool "diff_schemas" with input:
      | subject      | diff-cmp-test |
      | version_from | 1             |
      | version_to   | 1             |
    Then the MCP result should contain "diffs"

  # --- compare_subjects ---

  Scenario: Compare two different subjects
    Given I register an Avro schema for subject "compare-a-test"
    And I register an Avro schema for subject "compare-b-test"
    When I call MCP tool "compare_subjects" with input:
      | subject_a | compare-a-test |
      | subject_b | compare-b-test |
    Then the MCP result should contain "common_fields"
    And the MCP result should contain "compare-a-test"
    And the MCP result should contain "compare-b-test"
