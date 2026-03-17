@mcp @schema-modeling @ai
Feature: MCP AI Data Modeling — Error Handling and Edge Cases
  An AI agent encounters errors during schema modeling and handles them
  gracefully. Tests cover invalid schemas, missing subjects, type mismatches,
  and boundary conditions that an AI must handle in real-world usage.

  # ==========================================================================
  # 1. AI HANDLES INVALID SCHEMA GRACEFULLY
  # ==========================================================================

  Scenario: AI handles invalid Avro schema and retries with corrected version
    # AI submits a malformed Avro schema (missing closing brace)
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "error-invalid-avro",
        "schema": "{\"type\":\"record\",\"name\":\"Bad\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}"
      }
      """
    Then the MCP result should contain "error"
    # AI corrects the schema and retries
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "error-invalid-avro",
        "schema": "{\"type\":\"record\",\"name\":\"Good\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "error-invalid-avro"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | error-invalid-avro     |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 2. AI HANDLES READING NON-EXISTENT SCHEMAS
  # ==========================================================================

  Scenario: AI handles non-existent schema ID
    When I call MCP tool "get_schema_by_id" with input:
      | id | 999999 |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
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
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_schema_by_id       |
      | status_code          | 0                      |
      | reason               | *                      |
      | error                | *                      |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: AI handles non-existent subject
    When I call MCP tool "get_latest_schema" with input:
      | subject | totally-nonexistent-subject |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | totally-nonexistent-subject |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_latest_schema      |
      | status_code          | 0                      |
      | reason               | *                      |
      | error                | *                      |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: AI handles non-existent version
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "error-noversion-test",
        "schema": "{\"type\":\"record\",\"name\":\"X\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I call MCP tool "get_schema_version" with input:
      | subject | error-noversion-test |
      | version | 99                   |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | error-noversion-test   |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_schema_version     |
      | status_code          | 0                      |
      | reason               | *                      |
      | error                | *                      |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 3. AI HANDLES SCHEMA TYPE MISMATCH
  # ==========================================================================

  Scenario: AI handles registering wrong schema type for existing subject
    # Register as Avro first
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "error-type-mismatch",
        "schema": "{\"type\":\"record\",\"name\":\"T\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Try to register Protobuf under the same subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "error-type-mismatch",
        "schema": "syntax = \"proto3\";\nmessage T { int64 id = 1; }",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | error-type-mismatch    |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               | *                      |
      | error                | *                      |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 4. AI HANDLES COMPATIBILITY REJECTION
  # ==========================================================================

  Scenario: AI handles multiple incompatible changes and finds a compatible path
    When I call MCP tool "set_config" with input:
      | subject             | error-compat-path |
      | compatibility_level | BACKWARD          |
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "error-compat-path",
        "schema": "{\"type\":\"record\",\"name\":\"Profile\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI tries adding a required field without default — fails under BACKWARD
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "error-compat-path",
        "schema": "{\"type\":\"record\",\"name\":\"Profile\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\"},{\"name\":\"address\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "false"
    # AI tries changing a field type — fails under BACKWARD
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "error-compat-path",
        "schema": "{\"type\":\"record\",\"name\":\"Profile\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "false"
    # AI finds a compatible path — add optional field with default
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "error-compat-path",
        "schema": "{\"type\":\"record\",\"name\":\"Profile\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\"},{\"name\":\"bio\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "true"
    # AI registers the compatible change
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "error-compat-path",
        "schema": "{\"type\":\"record\",\"name\":\"Profile\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\"},{\"name\":\"bio\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":2"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | error-compat-path      |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 5. AI HANDLES EMPTY AND MINIMAL SCHEMAS
  # ==========================================================================

  Scenario: AI registers minimal valid schemas for all three types
    # Minimal Avro — a primitive type
    When I call MCP tool "register_schema" with input:
      | subject | minimal-avro-value    |
      | schema  | {"type":"string"}     |
    Then the MCP result should contain "\"version\":1"
    # Minimal JSON Schema — empty object
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "minimal-json-value",
        "schema": "{\"type\":\"object\"}",
        "schema_type": "JSON"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Minimal Protobuf — single message
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "minimal-proto-value",
        "schema": "syntax = \"proto3\";\nmessage Empty {}",
        "schema_type": "PROTOBUF"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | minimal-proto-value    |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 6. AI HANDLES OPERATIONS ON DELETED SUBJECTS
  # ==========================================================================

  Scenario: AI handles lookup on soft-deleted subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "error-deleted-lookup",
        "schema": "{\"type\":\"record\",\"name\":\"Temp\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I call MCP tool "delete_subject" with input:
      | subject | error-deleted-lookup |
    Then the MCP result should contain "1"
    # AI tries to get the deleted subject
    When I call MCP tool "get_latest_schema" with input:
      | subject | error-deleted-lookup |
    Then the MCP result should contain "error"
    And the audit log should contain an event:
      | event_type           | mcp_tool_error         |
      | outcome              | failure                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | error-deleted-lookup   |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_latest_schema      |
      | status_code          | 0                      |
      | reason               | *                      |
      | error                | *                      |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 7. AI VALIDATES COMPLEX NESTED AVRO SCHEMA
  # ==========================================================================

  Scenario: AI registers a deeply nested Avro schema with maps and arrays
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "complex-nested-value",
        "schema": "{\"type\":\"record\",\"name\":\"AnalyticsEvent\",\"namespace\":\"com.analytics\",\"fields\":[{\"name\":\"event_id\",\"type\":\"string\"},{\"name\":\"properties\",\"type\":{\"type\":\"map\",\"values\":\"string\"}},{\"name\":\"tags\",\"type\":{\"type\":\"array\",\"items\":\"string\"}},{\"name\":\"nested_data\",\"type\":[\"null\",{\"type\":\"record\",\"name\":\"EventData\",\"fields\":[{\"name\":\"key\",\"type\":\"string\"},{\"name\":\"values\",\"type\":{\"type\":\"array\",\"items\":\"double\"}}]}],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "complex-nested-value"
    # AI retrieves and verifies the complex schema
    When I call MCP tool "get_latest_schema" with input:
      | subject | complex-nested-value |
    Then the MCP result should contain "AnalyticsEvent"
    And the MCP result should contain "properties"
    And the MCP result should contain "EventData"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | complex-nested-value   |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_latest_schema      |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
