@mcp @schema-modeling @ai
Feature: MCP AI Data Modeling — Multi-Format Schema Design
  An AI agent uses MCP tools to design schemas across Avro, Protobuf, and
  JSON Schema formats, choosing the right format for each use case, and
  managing cross-format domain models.

  # ==========================================================================
  # 1. AI DESIGNS A CQRS SYSTEM WITH DIFFERENT FORMATS PER CONCERN
  # ==========================================================================

  Scenario: AI models a CQRS system — Avro for events, Protobuf for commands, JSON for queries
    # AI uses Avro for domain events (Kafka topic, schema evolution important)
    When I call MCP tool "set_config" with input:
      | subject             | cqrs.account-events-value |
      | compatibility_level | BACKWARD_TRANSITIVE       |
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "cqrs.account-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"AccountCreated\",\"namespace\":\"com.cqrs.events\",\"fields\":[{\"name\":\"account_id\",\"type\":\"string\"},{\"name\":\"owner_name\",\"type\":\"string\"},{\"name\":\"initial_balance\",\"type\":{\"type\":\"bytes\",\"logicalType\":\"decimal\",\"precision\":12,\"scale\":2}},{\"name\":\"created_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "AVRO"
    # AI uses Protobuf for commands (gRPC, strict typing)
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "cqrs.account-commands-value",
        "schema": "syntax = \"proto3\";\npackage cqrs.commands;\n\nmessage CreateAccount {\n  string request_id = 1;\n  string owner_name = 2;\n  string initial_balance = 3;\n}\n\nmessage TransferFunds {\n  string request_id = 1;\n  string from_account = 2;\n  string to_account = 3;\n  string amount = 4;\n}",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "PROTOBUF"
    # AI uses JSON Schema for query responses (REST API)
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "cqrs.account-query-response",
        "schema": "{\"type\":\"object\",\"properties\":{\"account_id\":{\"type\":\"string\"},\"owner_name\":{\"type\":\"string\"},\"balance\":{\"type\":\"string\"},\"status\":{\"type\":\"string\",\"enum\":[\"active\",\"frozen\",\"closed\"]},\"transactions\":{\"type\":\"array\",\"items\":{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"string\"},\"amount\":{\"type\":\"string\"},\"type\":{\"type\":\"string\"}},\"required\":[\"id\",\"amount\",\"type\"]}}},\"required\":[\"account_id\",\"owner_name\",\"balance\",\"status\"]}",
        "schema_type": "JSON"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "JSON"
    # AI verifies the complete CQRS model
    When I call MCP tool "list_subjects" with input:
      | prefix | cqrs. |
    Then the MCP result should contain "cqrs.account-events-value"
    And the MCP result should contain "cqrs.account-commands-value"
    And the MCP result should contain "cqrs.account-query-response"

  # ==========================================================================
  # 2. AI DESIGNS A PROTOBUF MICROSERVICE WITH NESTED MESSAGES
  # ==========================================================================

  Scenario: AI designs a Protobuf schema with enums and nested messages
    When I call MCP tool "set_config" with input:
      | subject             | notification-service-value |
      | compatibility_level | BACKWARD                   |
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "notification-service-value",
        "schema": "syntax = \"proto3\";\npackage notification.v1;\n\nenum Channel {\n  CHANNEL_UNSPECIFIED = 0;\n  CHANNEL_EMAIL = 1;\n  CHANNEL_SMS = 2;\n  CHANNEL_PUSH = 3;\n  CHANNEL_WEBHOOK = 4;\n}\n\nenum Priority {\n  PRIORITY_UNSPECIFIED = 0;\n  PRIORITY_LOW = 1;\n  PRIORITY_NORMAL = 2;\n  PRIORITY_HIGH = 3;\n  PRIORITY_URGENT = 4;\n}\n\nmessage Recipient {\n  string id = 1;\n  string address = 2;\n  Channel channel = 3;\n}\n\nmessage Notification {\n  string id = 1;\n  string template = 2;\n  Priority priority = 3;\n  repeated Recipient recipients = 4;\n  map<string, string> variables = 5;\n  int64 send_at = 6;\n}",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI evolves — adds delivery tracking
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "notification-service-value",
        "schema": "syntax = \"proto3\";\npackage notification.v1;\n\nenum Channel {\n  CHANNEL_UNSPECIFIED = 0;\n  CHANNEL_EMAIL = 1;\n  CHANNEL_SMS = 2;\n  CHANNEL_PUSH = 3;\n  CHANNEL_WEBHOOK = 4;\n}\n\nenum Priority {\n  PRIORITY_UNSPECIFIED = 0;\n  PRIORITY_LOW = 1;\n  PRIORITY_NORMAL = 2;\n  PRIORITY_HIGH = 3;\n  PRIORITY_URGENT = 4;\n}\n\nmessage Recipient {\n  string id = 1;\n  string address = 2;\n  Channel channel = 3;\n}\n\nmessage Notification {\n  string id = 1;\n  string template = 2;\n  Priority priority = 3;\n  repeated Recipient recipients = 4;\n  map<string, string> variables = 5;\n  int64 send_at = 6;\n  string callback_url = 7;\n}",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "\"version\":2"
    # AI verifies schema content
    When I call MCP tool "get_latest_schema" with input:
      | subject | notification-service-value |
    Then the MCP result should contain "Notification"
    And the MCP result should contain "Recipient"
    And the MCP result should contain "callback_url"

  # ==========================================================================
  # 3. AI MODELS A JSON SCHEMA WITH COMPLEX VALIDATION
  # ==========================================================================

  Scenario: AI designs a JSON Schema with validation constraints for API input
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "api-user-registration",
        "schema": "{\"type\":\"object\",\"properties\":{\"username\":{\"type\":\"string\",\"minLength\":3,\"maxLength\":50,\"pattern\":\"^[a-zA-Z0-9_]+$\"},\"email\":{\"type\":\"string\",\"format\":\"email\"},\"password\":{\"type\":\"string\",\"minLength\":8,\"maxLength\":128},\"age\":{\"type\":\"integer\",\"minimum\":13,\"maximum\":150},\"preferences\":{\"type\":\"object\",\"properties\":{\"theme\":{\"type\":\"string\",\"enum\":[\"light\",\"dark\",\"auto\"]},\"language\":{\"type\":\"string\",\"pattern\":\"^[a-z]{2}(-[A-Z]{2})?$\"},\"notifications\":{\"type\":\"boolean\"}},\"additionalProperties\":false}},\"required\":[\"username\",\"email\",\"password\"],\"additionalProperties\":false}",
        "schema_type": "JSON"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI inspects the registered schema
    When I call MCP tool "get_raw_schema_version" with input:
      | subject | api-user-registration |
      | version | 1                     |
    Then the MCP result should contain "minLength"
    And the MCP result should contain "pattern"
    And the MCP result should contain "preferences"

  # ==========================================================================
  # 4. AI USES FORMAT_SCHEMA TO REVIEW REGISTERED SCHEMAS
  # ==========================================================================

  Scenario: AI formats a registered schema for human review
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "format-review-value",
        "schema": "{\"type\":\"record\",\"name\":\"Review\",\"namespace\":\"com.reviews\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"rating\",\"type\":\"int\"},{\"name\":\"comment\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI uses format_schema to get a readable version
    When I call MCP tool "format_schema" with JSON input:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Review\",\"namespace\":\"com.reviews\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"rating\",\"type\":\"int\"},{\"name\":\"comment\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "Review"

  # ==========================================================================
  # 5. AI DESIGNS AN AVRO SCHEMA WITH UNION TYPES
  # ==========================================================================

  Scenario: AI models events with union types for polymorphic payloads
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "polymorphic-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"DomainEvent\",\"namespace\":\"com.events\",\"fields\":[{\"name\":\"event_id\",\"type\":\"string\"},{\"name\":\"event_type\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"metadata\",\"type\":{\"type\":\"map\",\"values\":\"string\"}},{\"name\":\"payload\",\"type\":[\"null\",\"string\",\"long\",\"double\",{\"type\":\"map\",\"values\":\"string\"}],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I call MCP tool "get_latest_schema" with input:
      | subject | polymorphic-events-value |
    Then the MCP result should contain "DomainEvent"
    And the MCP result should contain "metadata"
    And the MCP result should contain "payload"

  # ==========================================================================
  # 6. AI MODELS A PROTOBUF WITH ONEOF FOR VARIANT TYPES
  # ==========================================================================

  Scenario: AI designs a Protobuf schema with oneof variant pattern
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "variant-message-value",
        "schema": "syntax = \"proto3\";\npackage messaging.v1;\n\nmessage TextContent {\n  string body = 1;\n}\n\nmessage ImageContent {\n  string url = 1;\n  int32 width = 2;\n  int32 height = 3;\n}\n\nmessage Message {\n  string id = 1;\n  string sender = 2;\n  int64 timestamp = 3;\n  oneof content {\n    TextContent text = 4;\n    ImageContent image = 5;\n  }\n}",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I call MCP tool "get_latest_schema" with input:
      | subject | variant-message-value |
    Then the MCP result should contain "Message"
    And the MCP result should contain "TextContent"
    And the MCP result should contain "ImageContent"
    And the MCP result should contain "oneof"
