@kms @data-contracts
Feature: KMS Server-Side Field-Level Encryption
  The DEK Registry supports server-side key generation when a KEK is configured
  as shared with a real KMS backend (HashiCorp Vault Transit or OpenBao Transit).
  When a DEK is created without client-provided encryptedKeyMaterial, the server
  generates raw key material, encrypts (wraps) it via the KMS Transit engine,
  and returns both the plaintext keyMaterial and the encryptedKeyMaterial.

  Background:
    Given the schema registry is running

  # ============================================================================
  # Vault Transit Scenarios
  # ============================================================================

  Scenario: Server-side DEK generation with Vault Transit shared KEK
    Given a shared KEK "vault-kek-aes256gcm" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "vault.user.email" under KEK "vault-kek-aes256gcm"
    Then the response status should be 200
    And the response field "kekName" should be "vault-kek-aes256gcm"
    And the response field "subject" should be "vault.user.email"
    And the response field "version" should be 1
    And the response field "algorithm" should be "AES256_GCM"
    And the response field "keyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the encrypted key material using KMS type "hcvault" and key ID "test-key"

  Scenario: Vault Transit DEK with AES128_GCM algorithm
    Given a shared KEK "vault-kek-aes128" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "vault.payment.card" under KEK "vault-kek-aes128" with algorithm "AES128_GCM"
    Then the response status should be 200
    And the response field "algorithm" should be "AES128_GCM"
    And the response field "keyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the encrypted key material using KMS type "hcvault" and key ID "test-key"

  Scenario: Vault Transit DEK with AES256_SIV algorithm
    Given a shared KEK "vault-kek-aes256siv" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "vault.ssn.field" under KEK "vault-kek-aes256siv" with algorithm "AES256_SIV"
    Then the response status should be 200
    And the response field "algorithm" should be "AES256_SIV"
    And the response field "keyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the encrypted key material using KMS type "hcvault" and key ID "test-key"

  Scenario: Multi-version DEKs under Vault Transit KEK have unique key material
    Given a shared KEK "vault-kek-multiversion" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "vault.versioned.field" under KEK "vault-kek-multiversion"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response field "keyMaterial" should be non-empty
    And I store the response field "keyMaterial" as "v1_keyMaterial"
    When I create a DEK for subject "vault.versioned.field" under KEK "vault-kek-multiversion"
    Then the response status should be 200
    And the response field "version" should be 2
    And the response field "keyMaterial" should be non-empty
    And I store the response field "keyMaterial" as "v2_keyMaterial"
    And the response field "keyMaterial" should not equal stored "v1_keyMaterial"
    When I create a DEK for subject "vault.versioned.field" under KEK "vault-kek-multiversion"
    Then the response status should be 200
    And the response field "version" should be 3
    And the response field "keyMaterial" should be non-empty
    And the response field "keyMaterial" should not equal stored "v1_keyMaterial"
    And the response field "keyMaterial" should not equal stored "v2_keyMaterial"

  # ============================================================================
  # OpenBao Transit Scenarios
  # ============================================================================

  Scenario: Server-side DEK generation with OpenBao Transit shared KEK
    Given a shared KEK "bao-kek-aes256gcm" with KMS type "openbao" and key ID "test-key"
    When I create a DEK for subject "bao.user.email" under KEK "bao-kek-aes256gcm"
    Then the response status should be 200
    And the response field "kekName" should be "bao-kek-aes256gcm"
    And the response field "subject" should be "bao.user.email"
    And the response field "version" should be 1
    And the response field "algorithm" should be "AES256_GCM"
    And the response field "keyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the encrypted key material using KMS type "openbao" and key ID "test-key"

  Scenario: OpenBao Transit DEK with AES128_GCM algorithm
    Given a shared KEK "bao-kek-aes128" with KMS type "openbao" and key ID "test-key"
    When I create a DEK for subject "bao.payment.card" under KEK "bao-kek-aes128" with algorithm "AES128_GCM"
    Then the response status should be 200
    And the response field "algorithm" should be "AES128_GCM"
    And the response field "keyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the encrypted key material using KMS type "openbao" and key ID "test-key"

  Scenario: OpenBao Transit DEK with AES256_SIV algorithm
    Given a shared KEK "bao-kek-aes256siv" with KMS type "openbao" and key ID "test-key"
    When I create a DEK for subject "bao.ssn.field" under KEK "bao-kek-aes256siv" with algorithm "AES256_SIV"
    Then the response status should be 200
    And the response field "algorithm" should be "AES256_SIV"
    And the response field "keyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the encrypted key material using KMS type "openbao" and key ID "test-key"

  # ============================================================================
  # Cross-KMS Scenarios
  # ============================================================================

  Scenario: Cross-KMS isolation between Vault and OpenBao KEKs
    Given a shared KEK "cross-vault-kek" with KMS type "hcvault" and key ID "test-key"
    And a shared KEK "cross-bao-kek" with KMS type "openbao" and key ID "test-key"
    When I create a DEK for subject "cross.vault.subject" under KEK "cross-vault-kek"
    Then the response status should be 200
    And the response field "kekName" should be "cross-vault-kek"
    And the response field "keyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the encrypted key material using KMS type "hcvault" and key ID "test-key"
    And I store the response field "keyMaterial" as "vault_key"
    When I create a DEK for subject "cross.bao.subject" under KEK "cross-bao-kek"
    Then the response status should be 200
    And the response field "kekName" should be "cross-bao-kek"
    And the response field "keyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the encrypted key material using KMS type "openbao" and key ID "test-key"
    And the response field "keyMaterial" should not equal stored "vault_key"

  # ============================================================================
  # Non-Shared / Client-Provided Scenarios
  # ============================================================================

  Scenario: Non-shared KEK skips server-side key generation
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "non-shared-kek",
        "kmsType": "hcvault",
        "kmsKeyId": "test-key",
        "shared": false
      }
      """
    Then the response status should be 200
    And the response field "shared" should be false
    When I create a DEK for subject "nonshared.subject" under KEK "non-shared-kek"
    Then the response status should be 200
    And the response field "kekName" should be "non-shared-kek"
    And the response field "subject" should be "nonshared.subject"
    And the response field "keyMaterial" should be empty or absent
    And the response field "encryptedKeyMaterial" should be empty or absent

  Scenario: Client-provided encrypted key material is preserved
    Given a shared KEK "client-material-kek" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "client.provided.field" under KEK "client-material-kek" with encrypted key material "client-provided-material"
    Then the response status should be 200
    And the response field "kekName" should be "client-material-kek"
    And the response field "subject" should be "client.provided.field"
    And the response field "encryptedKeyMaterial" should be "client-provided-material"

  # ============================================================================
  # DEK Lifecycle with Encryption
  # ============================================================================

  Scenario: Encrypted DEK persists through soft-delete and undelete lifecycle
    Given a shared KEK "lifecycle-enc-kek" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "lifecycle.encrypted.field" under KEK "lifecycle-enc-kek"
    Then the response status should be 200
    And the response field "keyMaterial" should be non-empty
    And I store the response field "encryptedKeyMaterial" as "original_encrypted"
    # Soft-delete the DEK
    When I DELETE "/dek-registry/v1/keks/lifecycle-enc-kek/deks/lifecycle.encrypted.field"
    Then the response status should be 200
    # Undelete the DEK
    When I PUT "/dek-registry/v1/keks/lifecycle-enc-kek/deks/lifecycle.encrypted.field/undelete" with body:
      """
      {}
      """
    Then the response status should be 200
    # Retrieve and verify key material persists
    When I GET "/dek-registry/v1/keks/lifecycle-enc-kek/deks/lifecycle.encrypted.field"
    Then the response status should be 200
    And the response field "subject" should be "lifecycle.encrypted.field"
    And the response field "encryptedKeyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should equal stored "original_encrypted"

  Scenario: Permanent delete removes encrypted DEK
    Given a shared KEK "permdelete-enc-kek" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "permdelete.encrypted.field" under KEK "permdelete-enc-kek"
    Then the response status should be 200
    And the response field "encryptedKeyMaterial" should be non-empty
    # Soft-delete first (required before permanent delete)
    When I DELETE "/dek-registry/v1/keks/permdelete-enc-kek/deks/permdelete.encrypted.field"
    Then the response status should be 200
    # Permanent delete
    When I DELETE "/dek-registry/v1/keks/permdelete-enc-kek/deks/permdelete.encrypted.field?permanent=true"
    Then the response status should be 200
    # Verify DEK is gone even with deleted=true
    When I GET "/dek-registry/v1/keks/permdelete-enc-kek/deks/permdelete.encrypted.field?deleted=true"
    Then the response status should be 404

  # ============================================================================
  # Error Scenarios
  # ============================================================================

  Scenario: DEK creation with unknown KMS type shared KEK succeeds without key material
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "unknown-kms-kek",
        "kmsType": "unknown-kms",
        "kmsKeyId": "some-key-id",
        "shared": true
      }
      """
    Then the response status should be 200
    And the response field "shared" should be true
    When I create a DEK for subject "unknown.kms.field" under KEK "unknown-kms-kek"
    Then the response status should be 200
    And the response field "kekName" should be "unknown-kms-kek"
    And the response field "subject" should be "unknown.kms.field"
    And the response field "keyMaterial" should be empty or absent
    And the response field "encryptedKeyMaterial" should be empty or absent

  Scenario: Vault Transit DEK version retrieval returns encrypted key material
    Given a shared KEK "vault-kek-retrieve" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "vault.retrieve.field" under KEK "vault-kek-retrieve"
    Then the response status should be 200
    And I store the response field "encryptedKeyMaterial" as "created_encrypted"
    When I GET "/dek-registry/v1/keks/vault-kek-retrieve/deks/vault.retrieve.field/versions/1"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response field "encryptedKeyMaterial" should be non-empty
    And the response field "encryptedKeyMaterial" should equal stored "created_encrypted"

  Scenario: Multiple subjects under same Vault Transit KEK have independent keys
    Given a shared KEK "vault-kek-multi-subject" with KMS type "hcvault" and key ID "test-key"
    When I create a DEK for subject "vault.multi.email" under KEK "vault-kek-multi-subject"
    Then the response status should be 200
    And the response field "keyMaterial" should be non-empty
    And I store the response field "keyMaterial" as "email_key"
    When I create a DEK for subject "vault.multi.phone" under KEK "vault-kek-multi-subject"
    Then the response status should be 200
    And the response field "keyMaterial" should be non-empty
    And the response field "keyMaterial" should not equal stored "email_key"
    And I store the response field "keyMaterial" as "phone_key"
    When I create a DEK for subject "vault.multi.ssn" under KEK "vault-kek-multi-subject"
    Then the response status should be 200
    And the response field "keyMaterial" should be non-empty
    And the response field "keyMaterial" should not equal stored "email_key"
    And the response field "keyMaterial" should not equal stored "phone_key"
