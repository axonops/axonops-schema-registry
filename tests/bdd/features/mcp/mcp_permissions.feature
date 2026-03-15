@mcp @mcp-permissions
Feature: MCP Permission Scopes — Granular Tool Access Control
  Permission scopes control which MCP tools are visible to AI agents.
  Each preset (readonly, developer, operator, admin, full) exposes a
  different subset of tools, allowing fine-grained access control.

  # ==========================================================================
  # 1. READONLY PRESET
  # ==========================================================================

  @preset-readonly
  Scenario: Readonly preset allows list_subjects
    Given MCP permission preset is "readonly"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
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
      | path                 | list_subjects          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  @preset-readonly
  Scenario: Readonly preset hides register_schema
    Given MCP permission preset is "readonly"
    When I list MCP tools
    Then the MCP result should not contain "register_schema"

  # ==========================================================================
  # 2. DEVELOPER PRESET — includes all readonly tools plus write tools
  # ==========================================================================

  @preset-developer
  Scenario: Developer preset allows register_schema
    Given MCP permission preset is "developer"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "perm-dev-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | perm-dev-test          |
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
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  @preset-developer
  Scenario: Developer preset includes readonly tools
    Given MCP permission preset is "developer"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
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
      | path                 | list_subjects          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  @preset-developer
  Scenario: Developer preset hides delete_subject
    Given MCP permission preset is "developer"
    When I list MCP tools
    Then the MCP result should not contain "delete_subject"

  @preset-developer
  Scenario: Developer preset hides create_user
    Given MCP permission preset is "developer"
    When I list MCP tools
    Then the MCP result should not contain "create_user"

  # ==========================================================================
  # 3. OPERATOR PRESET — includes all developer tools plus delete tools
  # ==========================================================================

  @preset-operator
  Scenario: Operator preset allows delete_subject
    Given MCP permission preset is "operator"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "perm-op-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "perm-op-test"}
      """
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | perm-op-test           |
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
      | path                 | delete_subject         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  @preset-operator
  Scenario: Operator preset includes readonly tools
    Given MCP permission preset is "operator"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
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
      | path                 | list_subjects          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  @preset-operator
  Scenario: Operator preset hides create_user
    Given MCP permission preset is "operator"
    When I list MCP tools
    Then the MCP result should not contain "create_user"

  # ==========================================================================
  # 4. ADMIN PRESET — includes all operator tools plus admin tools
  # ==========================================================================

  @preset-admin
  Scenario: Admin preset allows create_user
    Given MCP permission preset is "admin"
    When I list MCP tools
    Then the MCP result should contain "create_user"

  @preset-admin
  Scenario: Admin preset includes readonly tools
    Given MCP permission preset is "admin"
    When I call MCP tool "list_subjects" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
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
      | path                 | list_subjects          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  @preset-admin
  Scenario: Admin preset includes developer tools
    Given MCP permission preset is "admin"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "perm-admin-dev-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | perm-admin-dev-test    |
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
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  @preset-admin
  Scenario: Admin preset includes operator tools
    Given MCP permission preset is "admin"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "perm-admin-op-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "delete_subject" with JSON input:
      """
      {"subject": "perm-admin-op-test"}
      """
    Then the MCP result should not be an error
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | perm-admin-op-test     |
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
      | path                 | delete_subject         |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 5. SYSTEM TOOLS ALWAYS AVAILABLE
  # ==========================================================================

  @preset-readonly
  Scenario: System tools available under readonly preset
    Given MCP permission preset is "readonly"
    When I call MCP tool "health_check" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
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
      | path                 | health_check           |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  @preset-readonly
  Scenario: System tools listed under readonly preset
    Given MCP permission preset is "readonly"
    When I list MCP tools
    Then the MCP result should contain "health_check"
    And the MCP result should contain "get_server_info"
    And the MCP result should contain "get_schema_types"
    And the MCP result should contain "list_contexts"

  # ==========================================================================
  # 6. CUSTOM SCOPES
  # ==========================================================================

  @preset-custom
  Scenario: Custom scopes allow schema_read but block config and writes
    Given MCP permission scopes are "schema_read"
    When I list MCP tools
    Then the MCP result should contain "get_latest_schema"
    And the MCP result should contain "list_subjects"
    And the MCP result should not contain "register_schema"
    And the MCP result should not contain "get_config"
