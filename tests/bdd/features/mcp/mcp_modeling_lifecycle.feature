@mcp @schema-modeling @ai
Feature: MCP AI Data Modeling — Full Lifecycle Management
  An AI agent manages the complete lifecycle of schemas through MCP tools:
  schema registration, version inspection, compatibility configuration,
  soft-delete and recovery, mode management, and idempotent operations.

  # ==========================================================================
  # 1. AI MANAGES SCHEMA LIFECYCLE — REGISTER, EVOLVE, DELETE, RE-REGISTER
  # ==========================================================================

  Scenario: AI manages a complete schema lifecycle via MCP
    # AI sets up the subject with BACKWARD compatibility
    When I call MCP tool "set_config" with input:
      | subject             | lifecycle.session-value |
      | compatibility_level | BACKWARD                |
    # AI registers v1
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "lifecycle.session-value",
        "schema": "{\"type\":\"record\",\"name\":\"Session\",\"namespace\":\"com.auth\",\"fields\":[{\"name\":\"session_id\",\"type\":\"string\"},{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"created_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI evolves to v2
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "lifecycle.session-value",
        "schema": "{\"type\":\"record\",\"name\":\"Session\",\"namespace\":\"com.auth\",\"fields\":[{\"name\":\"session_id\",\"type\":\"string\"},{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"created_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"ip_address\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":2"
    # AI evolves to v3
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "lifecycle.session-value",
        "schema": "{\"type\":\"record\",\"name\":\"Session\",\"namespace\":\"com.auth\",\"fields\":[{\"name\":\"session_id\",\"type\":\"string\"},{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"created_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"ip_address\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"user_agent\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":3"
    # AI soft-deletes the subject
    When I call MCP tool "delete_subject" with input:
      | subject | lifecycle.session-value |
    Then the MCP result should contain "1"
    And the MCP result should contain "2"
    And the MCP result should contain "3"
    # AI re-registers after delete (v4)
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "lifecycle.session-value",
        "schema": "{\"type\":\"record\",\"name\":\"Session\",\"namespace\":\"com.auth\",\"fields\":[{\"name\":\"session_id\",\"type\":\"string\"},{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"created_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"ip_address\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"user_agent\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"device_type\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":4"
    # AI verifies the latest version
    When I call MCP tool "get_latest_schema" with input:
      | subject | lifecycle.session-value |
    Then the MCP result should contain "device_type"
    And the MCP result should contain "Session"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | lifecycle.session-value |
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

  # ==========================================================================
  # 2. AI PERFORMS IDEMPOTENT RE-REGISTRATION
  # ==========================================================================

  Scenario: AI re-registers identical schema and gets same ID back
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "idempotent-test-value",
        "schema": "{\"type\":\"record\",\"name\":\"Notification\",\"namespace\":\"com.msg\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"channel\",\"type\":\"string\"},{\"name\":\"body\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI re-registers the exact same schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "idempotent-test-value",
        "schema": "{\"type\":\"record\",\"name\":\"Notification\",\"namespace\":\"com.msg\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"channel\",\"type\":\"string\"},{\"name\":\"body\",\"type\":\"string\"}]}"
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
      | target_id            | idempotent-test-value  |
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
  # 3. AI DELETES A SPECIFIC VERSION
  # ==========================================================================

  Scenario: AI deletes a specific bad version and continues evolution
    When I call MCP tool "set_config" with input:
      | subject             | version-mgmt-value |
      | compatibility_level | NONE               |
    # Register v1
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "version-mgmt-value",
        "schema": "{\"type\":\"record\",\"name\":\"Metric\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"double\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Register v2 (a bad version the AI wants to remove)
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "version-mgmt-value",
        "schema": "{\"type\":\"record\",\"name\":\"Metric\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"double\"},{\"name\":\"bad_field\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":2"
    # AI soft-deletes the bad version
    When I call MCP tool "delete_version" with input:
      | subject | version-mgmt-value |
      | version | 2                  |
    Then the MCP result should contain "2"
    # AI registers a correct v3
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "version-mgmt-value",
        "schema": "{\"type\":\"record\",\"name\":\"Metric\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"double\"},{\"name\":\"unit\",\"type\":\"string\",\"default\":\"\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":3"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | version-mgmt-value     |
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
  # 4. AI USES MODE TO PROTECT A PRODUCTION SUBJECT
  # ==========================================================================

  Scenario: AI locks a subject to READONLY after finalizing schema design
    # AI registers the finalized schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "finalized.config-value",
        "schema": "{\"type\":\"record\",\"name\":\"AppConfig\",\"fields\":[{\"name\":\"key\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"string\"},{\"name\":\"version\",\"type\":\"int\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI locks the subject
    When I call MCP tool "set_mode" with input:
      | subject | finalized.config-value |
      | mode    | READONLY               |
    # AI verifies the lock via check_write_mode
    When I call MCP tool "check_write_mode" with input:
      | subject | finalized.config-value |
    Then the MCP result should contain "READONLY"
    # AI unlocks for further evolution
    When I call MCP tool "set_mode" with input:
      | subject | finalized.config-value |
      | mode    | READWRITE              |
    When I call MCP tool "get_mode" with input:
      | subject | finalized.config-value |
    Then the MCP result should contain "READWRITE"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | finalized.config-value |
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
      | path                 | get_mode               |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 5. AI COMPARES COMPATIBILITY LEVELS TO CHOOSE THE RIGHT ONE
  # ==========================================================================

  Scenario: AI tests different compatibility levels to find the right policy
    # AI registers a base schema under NONE to start
    When I call MCP tool "set_config" with input:
      | subject             | compat-explore-value |
      | compatibility_level | NONE                 |
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "compat-explore-value",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"data\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI switches to BACKWARD and checks if adding a required field is allowed
    When I call MCP tool "set_config" with input:
      | subject             | compat-explore-value |
      | compatibility_level | BACKWARD             |
    # Adding required field without default — incompatible under BACKWARD
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "compat-explore-value",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"data\",\"type\":\"string\"},{\"name\":\"source\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "false"
    # AI tests with FORWARD — adding a required field IS allowed
    When I call MCP tool "set_config" with input:
      | subject             | compat-explore-value |
      | compatibility_level | FORWARD              |
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "compat-explore-value",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"data\",\"type\":\"string\"},{\"name\":\"source\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "true"
    # AI settles on FULL compatibility for production
    When I call MCP tool "set_config" with input:
      | subject             | compat-explore-value |
      | compatibility_level | FULL                 |
    When I call MCP tool "get_config" with input:
      | subject | compat-explore-value |
    Then the MCP result should contain "FULL"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | compat-explore-value   |
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
      | path                 | get_config             |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 6. AI USES SCHEMA LOOKUP TO AVOID DUPLICATES
  # ==========================================================================

  Scenario: AI uses lookup to check if a schema already exists before registering
    # AI registers a schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dedup-check-value",
        "schema": "{\"type\":\"record\",\"name\":\"LogEntry\",\"fields\":[{\"name\":\"level\",\"type\":\"string\"},{\"name\":\"message\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI checks if the schema already exists via lookup
    When I call MCP tool "lookup_schema" with JSON input:
      """
      {
        "subject": "dedup-check-value",
        "schema": "{\"type\":\"record\",\"name\":\"LogEntry\",\"fields\":[{\"name\":\"level\",\"type\":\"string\"},{\"name\":\"message\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should contain "dedup-check-value"
    And the MCP result should contain "\"version\":1"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dedup-check-value      |
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
      | path                 | lookup_schema          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 7. AI DISCOVERS SCHEMA REUSE ACROSS SUBJECTS
  # ==========================================================================

  Scenario: AI detects schema reuse via global ID across multiple subjects
    # AI registers the same schema under two subjects
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "reuse-topic-a-value",
        "schema": "{\"type\":\"record\",\"name\":\"SharedEvent\",\"namespace\":\"com.shared\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"payload\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    And I store the MCP result field "id" as "schema_id"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "reuse-topic-b-value",
        "schema": "{\"type\":\"record\",\"name\":\"SharedEvent\",\"namespace\":\"com.shared\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"payload\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI finds all subjects using the stored schema ID
    When I call MCP tool "get_subjects_for_schema" with input:
      | id | $schema_id |
    Then the MCP result should contain "reuse-topic-a-value"
    And the MCP result should contain "reuse-topic-b-value"
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
      | context              | .                      |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_subjects_for_schema |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
