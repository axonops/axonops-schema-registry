@mcp @mcp-workflow
Feature: MCP Workflow — New Kafka Topic Setup
  Tests the multi-step workflow from prompts/new-kafka-topic.md by executing
  each step as MCP tool calls.

  # Validates: prompts/new-kafka-topic.md — Steps 2-4, glossary/core-concepts Subject Naming
  Scenario: Register key and value schemas following TopicNameStrategy
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "orders-key",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "orders-value",
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should contain "orders-key"
    And the MCP result should contain "orders-value"

  # Validates: prompts/new-kafka-topic.md — Step 6, glossary/best-practices
  Scenario: Validate schema syntax before registration
    When I call MCP tool "validate_schema" with JSON input:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "schema_type": "AVRO"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "valid"

  # Validates: prompts/new-kafka-topic.md — Step 6, invalid schema
  Scenario: Validate catches invalid schema syntax
    When I call MCP tool "validate_schema" with JSON input:
      """
      {
        "schema": "{\"type\":\"not-a-type\"}",
        "schema_type": "AVRO"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "\"valid\":false"

  # Validates: prompts/new-kafka-topic.md — Step 7, glossary/compatibility
  Scenario: Check compatibility before registration
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-compat-test",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "wf-compat-test",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "true"

  # Validates: prompts/new-kafka-topic.md — Step 5
  Scenario: Set compatibility level for new subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-config-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "set_config" with JSON input:
      """
      {
        "subject": "wf-config-test",
        "compatibility_level": "FULL"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_config" with JSON input:
      """
      {
        "subject": "wf-config-test"
      }
      """
    Then the MCP result should contain "FULL"

  # Validates: prompts/new-kafka-topic.md — Step 12, glossary/contexts
  Scenario: Register schema with context parameter
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-ctx-orders-value",
        "schema": "{\"type\":\"string\"}",
        "context": ".staging"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_subjects" with JSON input:
      """
      {"context": ".staging"}
      """
    Then the MCP result should contain "wf-ctx-orders-value"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not contain "wf-ctx-orders-value"

  # Validates: prompts/new-kafka-topic.md — Step 9, glossary/core-concepts Schema IDs
  Scenario: Retrieve registered schemas by ID and by subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "wf-retrieve-test",
        "schema": "{\"type\":\"record\",\"name\":\"TestEvent\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "\"id\":"
    When I call MCP tool "get_latest_schema" with JSON input:
      """
      {"subject": "wf-retrieve-test"}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "TestEvent"
