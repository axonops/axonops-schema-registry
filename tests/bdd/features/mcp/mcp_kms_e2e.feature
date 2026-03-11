@mcp @kms @ai
Feature: MCP E2E — Field-Level Encryption with Real KMS
  An AI agent uses MCP tools to manage field-level encryption with real
  HashiCorp Vault and OpenBao Transit engines.
  Run via: make test-bdd-kms BACKEND=memory|postgres|mysql|cassandra These tests verify that
  server-side key generation, key wrapping/unwrapping, multi-algorithm
  DEKs, key versioning, rewrap, and KMS connectivity testing all work
  end-to-end through the MCP transport layer.

  # ==========================================================================
  # 1. VAULT TRANSIT — SERVER-SIDE DEK GENERATION VIA MCP
  # ==========================================================================

  Scenario: AI creates a shared Vault KEK and generates a DEK via MCP
    # AI creates a shared KEK backed by Vault Transit
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-vault-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "doc": "Vault Transit KEK for MCP E2E",
        "shared": true
      }
      """
    Then the MCP result should contain "mcp-vault-kek"
    And the MCP result should contain "hcvault"
    And the MCP result field "shared" should be non-empty
    # AI creates a DEK — server generates key material via Vault Transit
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-vault-kek",
        "subject": "vault.user.email",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "vault.user.email"
    And the MCP result should contain "AES256_GCM"
    And the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    # AI verifies the encrypted key material can be unwrapped via Vault Transit
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    And the audit log should contain event "mcp_tool_call"

  Scenario: AI creates Vault Transit DEK with AES128_GCM algorithm via MCP
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-vault-aes128-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    Then the MCP result should contain "mcp-vault-aes128-kek"
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-vault-aes128-kek",
        "subject": "vault.payment.card",
        "algorithm": "AES128_GCM"
      }
      """
    Then the MCP result should contain "AES128_GCM"
    And the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    And the audit log should contain event "mcp_tool_call"

  Scenario: AI creates Vault Transit DEK with AES256_SIV algorithm via MCP
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-vault-siv-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    Then the MCP result should contain "mcp-vault-siv-kek"
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-vault-siv-kek",
        "subject": "vault.ssn.field",
        "algorithm": "AES256_SIV"
      }
      """
    Then the MCP result should contain "AES256_SIV"
    And the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    And the audit log should contain event "mcp_tool_call"

  # ==========================================================================
  # 2. OPENBAO TRANSIT — SERVER-SIDE DEK GENERATION VIA MCP
  # ==========================================================================

  Scenario: AI creates a shared OpenBao KEK and generates a DEK via MCP
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-bao-kek",
        "kms_type": "openbao",
        "kms_key_id": "test-key",
        "doc": "OpenBao Transit KEK for MCP E2E",
        "shared": true
      }
      """
    Then the MCP result should contain "mcp-bao-kek"
    And the MCP result should contain "openbao"
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-bao-kek",
        "subject": "bao.user.email",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "bao.user.email"
    And the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "openbao" and key ID "test-key"
    And the audit log should contain event "mcp_tool_call"

  Scenario: AI creates OpenBao Transit DEK with AES128_GCM via MCP
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-bao-aes128-kek",
        "kms_type": "openbao",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-bao-aes128-kek",
        "subject": "bao.payment.card",
        "algorithm": "AES128_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "openbao" and key ID "test-key"
    And the audit log should contain event "mcp_tool_call"

  # ==========================================================================
  # 3. MULTI-VERSION DEKS — UNIQUE KEY MATERIAL PER VERSION
  # ==========================================================================

  Scenario: AI creates multi-version DEKs with unique key material via MCP
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-vault-multiversion-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    # DEK v1
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-vault-multiversion-kek",
        "subject": "vault.versioned.field",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And I store the MCP result field "keyMaterial" as "v1_key"
    And I store the MCP result field "encryptedKeyMaterial" as "v1_enc"
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # DEK v2
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-vault-multiversion-kek",
        "subject": "vault.versioned.field",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "keyMaterial" should not equal stored "v1_key"
    And the MCP result field "encryptedKeyMaterial" should not equal stored "v1_enc"
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # AI lists DEK versions
    When I call MCP tool "list_dek_versions" with JSON input:
      """
      {
        "kek_name": "mcp-vault-multiversion-kek",
        "subject": "vault.versioned.field"
      }
      """
    Then the MCP result should contain "1"
    And the MCP result should contain "2"
    And the audit log should contain event "mcp_tool_call"

  # ==========================================================================
  # 4. CROSS-KMS ISOLATION
  # ==========================================================================

  Scenario: AI verifies Vault and OpenBao produce independent keys via MCP
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-cross-vault-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-cross-bao-kek",
        "kms_type": "openbao",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    # Generate DEK via Vault
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-cross-vault-kek",
        "subject": "cross.vault.data",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And I store the MCP result field "keyMaterial" as "vault_key"
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # Generate DEK via OpenBao
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-cross-bao-kek",
        "subject": "cross.bao.data",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "keyMaterial" should not equal stored "vault_key"
    And I can unwrap the MCP result encrypted key material using KMS type "openbao" and key ID "test-key"
    And the audit log should contain event "mcp_tool_call"

  # ==========================================================================
  # 5. KMS CONNECTIVITY TESTING VIA MCP
  # ==========================================================================

  Scenario: AI tests Vault KEK connectivity via test_kek tool
    When I call MCP tool "test_kek" with JSON input:
      """
      {
        "name": "mcp-test-vault-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key"
      }
      """
    Then the MCP result should contain "true"
    And the audit log should contain event "mcp_tool_call"

  Scenario: AI tests OpenBao KEK connectivity via test_kek tool
    When I call MCP tool "test_kek" with JSON input:
      """
      {
        "name": "mcp-test-bao-kek",
        "kms_type": "openbao",
        "kms_key_id": "test-key"
      }
      """
    Then the MCP result should contain "true"
    And the audit log should contain event "mcp_tool_call"

  Scenario: AI handles test_kek with unregistered KMS provider
    When I call MCP tool "test_kek" with JSON input:
      """
      {
        "name": "mcp-test-bad-kek",
        "kms_type": "nonexistent-kms-provider",
        "kms_key_id": "some-key"
      }
      """
    Then the MCP result should contain "error"
    And the audit log should contain event "mcp_tool_error"

  # ==========================================================================
  # 6. REWRAP DEK AFTER KEK ROTATION VIA MCP
  # ==========================================================================

  Scenario: AI rewraps a DEK under the current KEK version via MCP
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-rewrap-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    # Create DEK
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-rewrap-kek",
        "subject": "rewrap.sensitive.field",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "encryptedKeyMaterial" should be non-empty
    And I store the MCP result field "encryptedKeyMaterial" as "original_enc"
    # Rewrap the DEK
    When I call MCP tool "rewrap_dek" with JSON input:
      """
      {
        "kek_name": "mcp-rewrap-kek",
        "subject": "rewrap.sensitive.field",
        "version": 1,
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "rewrap.sensitive.field"
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    And the audit log should contain event "mcp_tool_call"

  # ==========================================================================
  # 7. SECURITY — GET DEK NEVER RETURNS PLAINTEXT KEY MATERIAL
  # ==========================================================================

  Scenario: AI verifies get_dek strips plaintext keyMaterial
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-security-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    # Create DEK — plaintext keyMaterial returned on creation
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-security-kek",
        "subject": "security.test.field",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    And I store the MCP result field "encryptedKeyMaterial" as "stored_enc"
    # Retrieve DEK — plaintext should NOT be returned
    When I call MCP tool "get_dek" with JSON input:
      """
      {
        "kek_name": "mcp-security-kek",
        "subject": "security.test.field",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "security.test.field"
    And the MCP result field "encryptedKeyMaterial" should equal stored "stored_enc"
    And the MCP result should not contain "keyMaterial"
    And the audit log should contain event "mcp_tool_call"

  # ==========================================================================
  # 8. NON-SHARED KEK — NO SERVER-SIDE GENERATION
  # ==========================================================================

  Scenario: AI creates DEK under non-shared KEK — no key material generated
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-nonshared-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": false
      }
      """
    Then the MCP result should contain "mcp-nonshared-kek"
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-nonshared-kek",
        "subject": "nonshared.data",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "nonshared.data"
    And the MCP result field "keyMaterial" should be empty or absent
    And the MCP result field "encryptedKeyMaterial" should be empty or absent
    And the audit log should contain event "mcp_tool_call"

  # ==========================================================================
  # 9. MULTIPLE SUBJECTS — INDEPENDENT KEYS PER SUBJECT
  # ==========================================================================

  Scenario: AI generates independent keys for multiple subjects under same KEK
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-multi-subject-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    # DEK for email field
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-multi-subject-kek",
        "subject": "multi.user.email",
        "algorithm": "AES256_GCM"
      }
      """
    And I store the MCP result field "keyMaterial" as "email_key"
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # DEK for phone field
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-multi-subject-kek",
        "subject": "multi.user.phone",
        "algorithm": "AES256_GCM"
      }
      """
    And the MCP result field "keyMaterial" should not equal stored "email_key"
    And I store the MCP result field "keyMaterial" as "phone_key"
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # DEK for SSN field
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-multi-subject-kek",
        "subject": "multi.user.ssn",
        "algorithm": "AES256_GCM"
      }
      """
    And the MCP result field "keyMaterial" should not equal stored "email_key"
    And the MCP result field "keyMaterial" should not equal stored "phone_key"
    # AI lists all subjects under the KEK
    When I call MCP tool "list_deks" with input:
      | kek_name | mcp-multi-subject-kek |
    Then the MCP result should contain "multi.user.email"
    And the MCP result should contain "multi.user.phone"
    And the MCP result should contain "multi.user.ssn"
    And the audit log should contain event "mcp_tool_call"

  # ==========================================================================
  # 10. ENCRYPTED DEK LIFECYCLE — SOFT-DELETE AND RESTORE
  # ==========================================================================

  Scenario: AI manages encrypted DEK lifecycle via MCP
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mcp-lifecycle-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mcp-lifecycle-kek",
        "subject": "lifecycle.encrypted.field",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "encryptedKeyMaterial" should be non-empty
    And I store the MCP result field "encryptedKeyMaterial" as "original_enc"
    # Soft-delete the DEK
    When I call MCP tool "delete_dek" with JSON input:
      """
      {
        "kek_name": "mcp-lifecycle-kek",
        "subject": "lifecycle.encrypted.field",
        "version": 1,
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "true"
    # Verify it's gone from default list
    When I call MCP tool "list_deks" with input:
      | kek_name | mcp-lifecycle-kek |
    Then the MCP result should not contain "lifecycle.encrypted.field"
    # Restore the DEK
    When I call MCP tool "undelete_dek" with JSON input:
      """
      {
        "kek_name": "mcp-lifecycle-kek",
        "subject": "lifecycle.encrypted.field",
        "version": 1,
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "true"
    # Verify encrypted material persists
    When I call MCP tool "get_dek" with JSON input:
      """
      {
        "kek_name": "mcp-lifecycle-kek",
        "subject": "lifecycle.encrypted.field",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "encryptedKeyMaterial" should equal stored "original_enc"
    And the audit log should contain event "mcp_tool_call"
