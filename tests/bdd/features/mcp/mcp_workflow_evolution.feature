@mcp @mcp-workflow
Feature: MCP Workflow — Schema Evolution
  Tests schema evolution recipes from prompts/schema-evolution-cookbook.md
  and evolve-schema prompt by executing MCP tool call sequences.

  # Validates: schema-evolution-cookbook.md — Recipe 1: Add Optional Field
  Scenario: Add optional field with default is BACKWARD safe
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-optional",
        "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-evo-optional",
        "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "true"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-optional",
        "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should not be an error
    And the audit log should contain event "mcp_tool_call"

  # Validates: schema-evolution-cookbook.md — Recipe 2: Add Required Field with Default
  Scenario: Add required field with default
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-required",
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-evo-required",
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"status\",\"type\":\"string\",\"default\":\"pending\"}]}"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "true"
    And the audit log should contain event "mcp_tool_call"

  # Validates: schema-evolution-cookbook.md — Recipe 3, glossary/design-patterns Three-Phase Rename
  Scenario: Three-phase rename is BACKWARD compatible
    # Phase 1: Original schema with old field
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-rename",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"user_name\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    # Phase 2: Both old and new fields
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-evo-rename",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"user_name\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"display_name\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "true"
    And the audit log should contain event "mcp_tool_call"

  # Validates: schema-evolution-cookbook.md — Recipe 4, glossary/compatibility Type Promotions
  Scenario: Compatible type widening int to long
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-widen",
        "schema": "{\"type\":\"record\",\"name\":\"Metric\",\"fields\":[{\"name\":\"value\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-evo-widen",
        "schema": "{\"type\":\"record\",\"name\":\"Metric\",\"fields\":[{\"name\":\"value\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "true"
    And the audit log should contain event "mcp_tool_call"

  # Validates: schema-evolution-cookbook.md — Recipe 5
  Scenario: Remove field under BACKWARD compatibility
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-remove",
        "schema": "{\"type\":\"record\",\"name\":\"Profile\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"legacy\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-evo-remove",
        "schema": "{\"type\":\"record\",\"name\":\"Profile\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "true"
    And the audit log should contain event "mcp_tool_call"

  # Validates: schema-evolution-cookbook.md — Recipe 6 Option A
  Scenario: Break compatibility via new subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-break-v1",
        "schema": "{\"type\":\"record\",\"name\":\"Config\",\"fields\":[{\"name\":\"val\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    # New, incompatible schema on a new subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-break-v2",
        "schema": "{\"type\":\"record\",\"name\":\"Config\",\"fields\":[{\"name\":\"values\",\"type\":{\"type\":\"array\",\"items\":\"string\"}}]}"
      }
      """
    Then the MCP result should not be an error
    And the audit log should contain event "mcp_tool_call"

  # Validates: schema-evolution-cookbook.md — Recipe 7, glossary/schema-types References
  Scenario: Add schema reference for shared type
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-address",
        "schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-customer",
        "schema": "{\"type\":\"record\",\"name\":\"Customer\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}",
        "references": [{"name": "com.example.Address", "subject": "wf-evo-address", "version": 1}]
      }
      """
    Then the MCP result should not be an error
    And the audit log should contain event "mcp_tool_call"

  # Validates: evolve-schema prompt — Step 4
  Scenario: Explain compatibility failure when evolution fails
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-evo-fail",
        "schema": "{\"type\":\"record\",\"name\":\"Item\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-evo-fail",
        "schema": "{\"type\":\"record\",\"name\":\"Item\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "false"
    When I call MCP tool "explain_compatibility_failure" with JSON input:
      """
      {
        "subject": "wf-evo-fail",
        "schema": "{\"type\":\"record\",\"name\":\"Item\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should not be an error
    And the audit log should contain event "mcp_tool_call"
