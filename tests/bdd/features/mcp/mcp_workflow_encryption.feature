@mcp @mcp-workflow @kms
Feature: MCP Workflow — Encryption Lifecycle
  Tests the encryption workflow from prompts/full-encryption-lifecycle.md
  by executing each step as MCP tool calls.

  # Validates: full-encryption-lifecycle.md — Phase 1
  Scenario: Create KEK and test it
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
    When I call MCP tool "test_kek" with JSON input:
      """
      {"name": "wf-enc-kek", "kms_type": "test-kms", "kms_key_id": "test-key-1"}
      """
    Then the MCP result should not be an error

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
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should not be an error
    When I call MCP tool "undelete_dek" with JSON input:
      """
      {
        "kek_name": "wf-enc-kek4",
        "subject": "wf-enc-del-test",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should not be an error

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
    When I call MCP tool "test_kek" with JSON input:
      """
      {"name": "wf-enc-full-kek", "kms_type": "test-kms", "kms_key_id": "test-key-full"}
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
