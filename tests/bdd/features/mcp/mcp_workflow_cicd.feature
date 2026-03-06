@mcp @mcp-workflow
Feature: MCP Workflow — CI/CD Integration
  Tests the CI/CD pipeline workflow from prompts/cicd-integration.md
  by executing each step as MCP tool calls.

  # Validates: prompts/cicd-integration.md — Pre-commit checks (validate)
  Scenario: Validate schema pass and fail cases
    When I call MCP tool "validate_schema" with JSON input:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "schema_type": "AVRO"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "valid"
    When I call MCP tool "validate_schema" with JSON input:
      """
      {
        "schema": "not valid json at all",
        "schema_type": "AVRO"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "\"valid\":false"

  # Validates: prompts/cicd-integration.md — Pre-commit checks (compatibility)
  Scenario: Check compatibility as PR gate with pass and fail
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-cicd-compat",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    # Compatible change — pass
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-cicd-compat",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"ts\",\"type\":[\"null\",\"long\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "true"
    # Incompatible change — fail
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-cicd-compat",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should contain "false"

  # Validates: prompts/cicd-integration.md — Quality gate
  Scenario: Score schema quality threshold check
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-cicd-quality",
        "schema": "{\"type\":\"record\",\"name\":\"WellDocumented\",\"namespace\":\"com.example\",\"doc\":\"A well-documented record\",\"fields\":[{\"name\":\"id\",\"type\":\"string\",\"doc\":\"Unique identifier\"},{\"name\":\"name\",\"type\":\"string\",\"doc\":\"Display name\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "score_schema_quality" with JSON input:
      """
      {"subject": "wf-cicd-quality"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "score"

  # Validates: prompts/cicd-integration.md — Deployment step
  Scenario: Register and verify with get_latest_schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-cicd-deploy",
        "schema": "{\"type\":\"record\",\"name\":\"Deployed\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_latest_schema" with JSON input:
      """
      {"subject": "wf-cicd-deploy"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "Deployed"
