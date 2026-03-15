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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | match_subjects         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Match subjects by regex
    Given I register an Avro schema for subject "rx-alpha-value"
    And I register an Avro schema for subject "rx-beta-key"
    When I call MCP tool "match_subjects" with input:
      | pattern | ^rx-.*-value$ |
      | regex   | true          |
    Then the MCP result should contain "rx-alpha-value"
    And the MCP result should not contain "rx-beta-key"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | match_subjects         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: Match subjects with no matches
    When I call MCP tool "match_subjects" with input:
      | pattern | nonexistent_subject_pattern |
    Then the MCP result should contain "\"count\":0"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | match_subjects         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # --- suggest_compatible_change ---

  Scenario: Suggest compatible change - add field
    Given I register an Avro schema for subject "suggest-cmp-test"
    When I call MCP tool "suggest_compatible_change" with input:
      | subject     | suggest-cmp-test |
      | change_type | add_field        |
    Then the MCP result should contain "advice"
    And the MCP result should contain "BACKWARD"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call               |
      | outcome              | success                     |
      | actor_id             | mcp-anonymous               |
      | actor_type           | anonymous                   |
      | auth_method          |                             |
      | role                 |                             |
      | target_type          | subject                     |
      | target_id            | suggest-cmp-test            |
      | schema_id            |                             |
      | version              |                             |
      | schema_type          |                             |
      | before_hash          |                             |
      | after_hash           |                             |
      | context              |                             |
      | transport_security   |                             |
      | source_ip            |                             |
      | user_agent           |                             |
      | method               | MCP                         |
      | path                 | suggest_compatible_change   |
      | status_code          | 0                           |
      | reason               |                             |
      | error                |                             |
      | request_body         |                             |
      | metadata             |                             |
      | timestamp            | *                           |
      | duration_ms          | *                           |
      | request_id           |                             |

  Scenario: Suggest compatible change - rename field
    Given I register an Avro schema for subject "suggest-rename-test"
    When I call MCP tool "suggest_compatible_change" with input:
      | subject     | suggest-rename-test |
      | change_type | rename_field        |
    Then the MCP result should contain "advice"
    And the MCP result should contain "alias"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call               |
      | outcome              | success                     |
      | actor_id             | mcp-anonymous               |
      | actor_type           | anonymous                   |
      | auth_method          |                             |
      | role                 |                             |
      | target_type          | subject                     |
      | target_id            | suggest-rename-test         |
      | schema_id            |                             |
      | version              |                             |
      | schema_type          |                             |
      | before_hash          |                             |
      | after_hash           |                             |
      | context              |                             |
      | transport_security   |                             |
      | source_ip            |                             |
      | user_agent           |                             |
      | method               | MCP                         |
      | path                 | suggest_compatible_change   |
      | status_code          | 0                           |
      | reason               |                             |
      | error                |                             |
      | request_body         |                             |
      | metadata             |                             |
      | timestamp            | *                           |
      | duration_ms          | *                           |
      | request_id           |                             |

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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          |                            |
      | target_id            |                            |
      | schema_id            |                            |
      | version              |                            |
      | schema_type          |                            |
      | before_hash          |                            |
      | after_hash           |                            |
      | context              |                            |
      | transport_security   |                            |
      | source_ip            |                            |
      | user_agent           |                            |
      | method               | MCP                        |
      | path                 | check_compatibility_multi  |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call                    |
      | outcome              | success                          |
      | actor_id             | mcp-anonymous                    |
      | actor_type           | anonymous                        |
      | auth_method          |                                  |
      | role                 |                                  |
      | target_type          | subject                          |
      | target_id            | explain-cmp-test                 |
      | schema_id            |                                  |
      | version              |                                  |
      | schema_type          |                                  |
      | before_hash          |                                  |
      | after_hash           |                                  |
      | context              |                                  |
      | transport_security   |                                  |
      | source_ip            |                                  |
      | user_agent           |                                  |
      | method               | MCP                              |
      | path                 | explain_compatibility_failure    |
      | status_code          | 0                                |
      | reason               |                                  |
      | error                |                                  |
      | request_body         |                                  |
      | metadata             |                                  |
      | timestamp            | *                                |
      | duration_ms          | *                                |
      | request_id           |                                  |

  # --- diff_schemas ---

  Scenario: Diff two schema versions
    Given I register an Avro schema for subject "diff-cmp-test"
    When I call MCP tool "diff_schemas" with input:
      | subject      | diff-cmp-test |
      | version_from | 1             |
      | version_to   | 1             |
    Then the MCP result should contain "diffs"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | diff-cmp-test          |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | diff_schemas           |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

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
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          |                        |
      | target_id            |                        |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | compare_subjects       |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
