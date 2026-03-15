@mcp @mcp-workflow
Feature: MCP Workflow — Encryption Lifecycle
  Tests the encryption workflow from prompts/full-encryption-lifecycle.md
  by executing each step as MCP tool calls.

  # Validates: full-encryption-lifecycle.md — Phase 1
  # test_kek requires a real KMS provider, so we only create and verify storage
  Scenario: Create KEK and verify it exists
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "wf-enc-kek",
        "kms_type": "test-kms",
        "kms_key_id": "test-key-1",
        "shared": true
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "wf-enc-kek"
    When I call MCP tool "list_keks" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "wf-enc-kek"
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
      | path                 | list_keks              |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # Validates: full-encryption-lifecycle.md — Phase 2
  Scenario: Create DEK and verify retrieval
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "wf-enc-kek2",
        "kms_type": "test-kms",
        "kms_key_id": "test-key-2",
        "shared": true
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-kek2",
        "subject": "wf-enc-dek-test",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-kek2",
        "subject": "wf-enc-dek-test"
      }
      """
    Then the MCP result should not be an error
    And the MCP result should contain "AES256_GCM"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | wf-enc-dek-test        |
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
      | path                 | get_dek                |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # Validates: full-encryption-lifecycle.md — Phase 3
  Scenario: Multiple DEK versions for rotation
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "wf-enc-kek3",
        "kms_type": "test-kms",
        "kms_key_id": "test-key-3",
        "shared": true
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-kek3",
        "subject": "wf-enc-rotate-test",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_dek_versions" with JSON input:
      """
      {
        "kek_name": "wf-enc-kek3",
        "subject": "wf-enc-rotate-test"
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
      | target_id            | wf-enc-rotate-test     |
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
      | path                 | list_dek_versions      |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # Validates: full-encryption-lifecycle.md — Phase 5, glossary/encryption Soft-Delete
  Scenario: Soft-delete and restore DEK
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "wf-enc-kek4",
        "kms_type": "test-kms",
        "kms_key_id": "test-key-4",
        "shared": true
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-kek4",
        "subject": "wf-enc-del-test",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "delete_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-kek4",
        "subject": "wf-enc-del-test",
        "algorithm": "AES256_GCM",
        "version": 1
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "undelete_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-kek4",
        "subject": "wf-enc-del-test",
        "algorithm": "AES256_GCM",
        "version": 1
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
      | target_id            | wf-enc-del-test        |
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
      | path                 | undelete_dek           |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # Validates: full-encryption-lifecycle.md — Complete workflow
  Scenario: Full encryption lifecycle end-to-end
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "wf-enc-full-kek",
        "kms_type": "test-kms",
        "kms_key_id": "test-key-full",
        "shared": true
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-full-kek",
        "subject": "wf-enc-full-test",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "get_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-full-kek",
        "subject": "wf-enc-full-test"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "list_keks" with JSON input:
      """
      {}
      """
    Then the MCP result should not be an error
    And the MCP result should contain "wf-enc-full-kek"
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
      | path                 | list_keks              |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
