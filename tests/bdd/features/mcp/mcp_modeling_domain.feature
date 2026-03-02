@mcp @schema-modeling @ai
Feature: MCP AI Data Modeling — Domain Schema Design
  An AI agent uses MCP tools to design interconnected domain schemas with
  cross-subject references, model shared types, verify schema relationships,
  and build a cohesive data model across multiple Kafka topics.

  # ==========================================================================
  # 1. AI BUILDS A DOMAIN WITH AVRO CROSS-SUBJECT REFERENCES
  # ==========================================================================

  Scenario: AI models an e-commerce domain with cross-subject references
    # AI first registers the shared Address type
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "ecom.address-value",
        "schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.ecom\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"},{\"name\":\"state\",\"type\":\"string\"},{\"name\":\"zip\",\"type\":\"string\"},{\"name\":\"country\",\"type\":\"string\",\"default\":\"US\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI registers the Customer type
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "ecom.customer-value",
        "schema": "{\"type\":\"record\",\"name\":\"Customer\",\"namespace\":\"com.ecom\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"},{\"name\":\"shipping_address\",\"type\":\"com.ecom.Address\"}]}",
        "references": [
          {"name": "com.ecom.Address", "subject": "ecom.address-value", "version": 1}
        ]
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "ecom.customer-value"
    # AI registers the Product type
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "ecom.product-value",
        "schema": "{\"type\":\"record\",\"name\":\"Product\",\"namespace\":\"com.ecom\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"price\",\"type\":{\"type\":\"bytes\",\"logicalType\":\"decimal\",\"precision\":10,\"scale\":2}},{\"name\":\"category\",\"type\":{\"type\":\"enum\",\"name\":\"Category\",\"symbols\":[\"ELECTRONICS\",\"CLOTHING\",\"FOOD\",\"HOME\",\"OTHER\"]}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI checks which schemas reference the Address type
    When I call MCP tool "get_referenced_by" with input:
      | subject | ecom.address-value |
      | version | 1                  |
    Then the MCP result should not contain "error"
    # AI lists all subjects to verify the domain model
    When I call MCP tool "list_subjects"
    Then the MCP result should contain "ecom.address-value"
    And the MCP result should contain "ecom.customer-value"
    And the MCP result should contain "ecom.product-value"

  # ==========================================================================
  # 2. AI DISCOVERS AND UNDERSTANDS AN EXISTING DOMAIN
  # ==========================================================================

  Scenario: AI explores existing schemas to understand a domain
    # Setup: Pre-register some schemas to discover
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "discovery.user-value",
        "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.discover\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"role\",\"type\":{\"type\":\"enum\",\"name\":\"Role\",\"symbols\":[\"ADMIN\",\"USER\",\"VIEWER\"]}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "discovery.audit-value",
        "schema": "{\"type\":\"record\",\"name\":\"AuditEntry\",\"namespace\":\"com.discover\",\"fields\":[{\"name\":\"action\",\"type\":\"string\"},{\"name\":\"actor_id\",\"type\":\"long\"},{\"name\":\"resource\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI discovers all subjects with a prefix filter
    When I call MCP tool "list_subjects" with input:
      | prefix | discovery. |
    Then the MCP result should contain "discovery.user-value"
    And the MCP result should contain "discovery.audit-value"
    # AI retrieves server capabilities
    When I call MCP tool "get_schema_types"
    Then the MCP result should contain "AVRO"
    And the MCP result should contain "PROTOBUF"
    And the MCP result should contain "JSON"
    # AI inspects a specific schema to understand its structure
    When I call MCP tool "get_latest_schema" with input:
      | subject | discovery.user-value |
    Then the MCP result should contain "User"
    And the MCP result should contain "username"
    And the MCP result should contain "Role"
    # AI gets raw schema for programmatic analysis
    When I call MCP tool "get_raw_schema_version" with input:
      | subject | discovery.user-value |
      | version | 1                    |
    Then the MCP result should contain "com.discover"
    # AI checks the global config to understand compatibility policy
    When I call MCP tool "get_config"
    Then the MCP result should contain "BACKWARD"

  # ==========================================================================
  # 3. AI MODELS SAME DOMAIN IN MULTIPLE FORMATS
  # ==========================================================================

  Scenario: AI models a sensor domain in Avro and Protobuf for different consumers
    # AI registers an Avro schema for analytics consumers (Kafka)
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "sensor-reading-avro-value",
        "schema": "{\"type\":\"record\",\"name\":\"SensorReading\",\"namespace\":\"com.iot\",\"fields\":[{\"name\":\"device_id\",\"type\":\"string\"},{\"name\":\"temperature\",\"type\":\"double\"},{\"name\":\"humidity\",\"type\":\"double\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "AVRO"
    # AI registers an equivalent Protobuf schema for gRPC consumers
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "sensor-reading-proto-value",
        "schema": "syntax = \"proto3\";\npackage iot;\n\nmessage SensorReading {\n  string device_id = 1;\n  double temperature = 2;\n  double humidity = 3;\n  int64 timestamp = 4;\n}",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "PROTOBUF"
    # AI registers a JSON Schema for REST API validation
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "sensor-reading-json-value",
        "schema": "{\"type\":\"object\",\"properties\":{\"device_id\":{\"type\":\"string\"},\"temperature\":{\"type\":\"number\"},\"humidity\":{\"type\":\"number\"},\"timestamp\":{\"type\":\"integer\"}},\"required\":[\"device_id\",\"temperature\",\"humidity\",\"timestamp\"],\"additionalProperties\":false}",
        "schema_type": "JSON"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "JSON"
    # AI lists schemas and verifies all three representations exist
    When I call MCP tool "list_subjects" with input:
      | prefix | sensor-reading- |
    Then the MCP result should contain "sensor-reading-avro-value"
    And the MCP result should contain "sensor-reading-proto-value"
    And the MCP result should contain "sensor-reading-json-value"

  # ==========================================================================
  # 4. AI MODELS A PROTOBUF SERVICE DEFINITION
  # ==========================================================================

  Scenario: AI designs a Protobuf service with multi-message schema
    When I call MCP tool "set_config" with input:
      | subject             | grpc-user-service-value |
      | compatibility_level | BACKWARD                |
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "grpc-user-service-value",
        "schema": "syntax = \"proto3\";\npackage user.v1;\n\nenum UserStatus {\n  USER_STATUS_UNSPECIFIED = 0;\n  USER_STATUS_ACTIVE = 1;\n  USER_STATUS_SUSPENDED = 2;\n  USER_STATUS_DELETED = 3;\n}\n\nmessage User {\n  string id = 1;\n  string email = 2;\n  string display_name = 3;\n  UserStatus status = 4;\n  int64 created_at = 5;\n}",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI evolves the Protobuf with an optional phone field
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "grpc-user-service-value",
        "schema": "syntax = \"proto3\";\npackage user.v1;\n\nenum UserStatus {\n  USER_STATUS_UNSPECIFIED = 0;\n  USER_STATUS_ACTIVE = 1;\n  USER_STATUS_SUSPENDED = 2;\n  USER_STATUS_DELETED = 3;\n}\n\nmessage User {\n  string id = 1;\n  string email = 2;\n  string display_name = 3;\n  UserStatus status = 4;\n  int64 created_at = 5;\n  string phone = 6;\n}",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "\"version\":2"
    # AI verifies the evolution
    When I call MCP tool "get_latest_schema" with input:
      | subject | grpc-user-service-value |
    Then the MCP result should contain "phone"
    And the MCP result should contain "UserStatus"

  # ==========================================================================
  # 5. AI MODELS JSON SCHEMA API CONTRACTS
  # ==========================================================================

  Scenario: AI designs a REST API contract with JSON Schema and evolves it
    When I call MCP tool "set_config" with input:
      | subject             | api-product-request |
      | compatibility_level | BACKWARD            |
    When I call MCP tool "set_config" with input:
      | subject             | api-product-response |
      | compatibility_level | FULL                 |
    # AI registers the request schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "api-product-request",
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\",\"minLength\":1,\"maxLength\":255},\"price\":{\"type\":\"number\",\"minimum\":0},\"category\":{\"type\":\"string\",\"enum\":[\"electronics\",\"clothing\",\"food\",\"home\"]}},\"required\":[\"name\",\"price\",\"category\"],\"additionalProperties\":false}",
        "schema_type": "JSON"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI registers the response schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "api-product-response",
        "schema": "{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"string\",\"format\":\"uuid\"},\"name\":{\"type\":\"string\"},\"price\":{\"type\":\"number\"},\"category\":{\"type\":\"string\"},\"created_at\":{\"type\":\"string\",\"format\":\"date-time\"}},\"required\":[\"id\",\"name\",\"price\",\"category\",\"created_at\"]}",
        "schema_type": "JSON"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI evolves request — adds optional description
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "api-product-request",
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\",\"minLength\":1,\"maxLength\":255},\"price\":{\"type\":\"number\",\"minimum\":0},\"category\":{\"type\":\"string\",\"enum\":[\"electronics\",\"clothing\",\"food\",\"home\"]},\"description\":{\"type\":\"string\",\"maxLength\":2000}},\"required\":[\"name\",\"price\",\"category\"],\"additionalProperties\":false}",
        "schema_type": "JSON"
      }
      """
    Then the MCP result should contain "\"version\":2"
    # AI verifies both contracts
    When I call MCP tool "get_schemas_by_subject" with input:
      | subject | api-product-request |
    Then the MCP result should contain "api-product-request"
