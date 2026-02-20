@functional
Feature: DEK Registry API (Client-Side Field Level Encryption)
  The DEK Registry manages Key Encryption Keys (KEKs) and Data Encryption Keys (DEKs)
  for client-side field-level encryption. KEKs are backed by cloud KMS providers
  (AWS KMS, Azure Key Vault, GCP KMS) and DEKs are encrypted by KEKs.

  Background:
    Given the schema registry is running

  # ============================================================================
  # KEK CRUD Operations (15 scenarios)
  # ============================================================================

  Scenario: List KEKs when none exist
    When I GET "/dek-registry/v1/keks"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 0

  Scenario: Create KEK with all fields
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "test-kek-full",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/abcd1234-5678-90ab-cdef-1234567890ab",
        "kmsProps": {
          "region": "us-east-1",
          "endpoint": "https://kms.us-east-1.amazonaws.com"
        },
        "doc": "Test KEK with all fields",
        "shared": true
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "test-kek-full"
    And the response field "kmsType" should be "aws-kms"
    And the response field "kmsKeyId" should be "arn:aws:kms:us-east-1:123456789012:key/abcd1234-5678-90ab-cdef-1234567890ab"
    And the response field "doc" should be "Test KEK with all fields"
    And the response field "shared" should be true


  Scenario: Create KEK with minimal required fields
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "test-kek-minimal",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-west-2:123456789012:key/minimal-key-id"
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "test-kek-minimal"
    And the response field "kmsType" should be "aws-kms"
    And the response field "shared" should be false


  Scenario: Get KEK by name
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "get-test-kek",
        "kmsType": "azure-kms",
        "kmsKeyId": "https://myvault.vault.azure.net/keys/mykey/version",
        "doc": "Azure KEK for testing"
      }
      """
    When I GET "/dek-registry/v1/keks/get-test-kek"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "get-test-kek"
    And the response field "kmsType" should be "azure-kms"
    And the response field "doc" should be "Azure KEK for testing"

  Scenario: Get non-existent KEK returns 404
    When I GET "/dek-registry/v1/keks/does-not-exist"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40470
    And the response should contain "not found"

  Scenario: Create duplicate KEK returns conflict
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "duplicate-kek",
        "kmsType": "gcp-kms",
        "kmsKeyId": "projects/my-project/locations/us/keyRings/ring/cryptoKeys/key"
      }
      """
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "duplicate-kek",
        "kmsType": "gcp-kms",
        "kmsKeyId": "projects/my-project/locations/us/keyRings/ring/cryptoKeys/key"
      }
      """
    Then the response status should be 409
    And the response should be valid JSON
    And the response should have error code 40970
    And the response should contain "already exists"

  Scenario: Update KEK
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "update-test-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/original",
        "doc": "Original doc",
        "shared": false
      }
      """
    When I PUT "/dek-registry/v1/keks/update-test-kek" with body:
      """
      {
        "kmsProps": {
          "region": "us-east-1",
          "tag": "production"
        },
        "doc": "Updated documentation",
        "shared": true
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "doc" should be "Updated documentation"
    And the response field "shared" should be true

  Scenario: Update non-existent KEK returns 404
    When I PUT "/dek-registry/v1/keks/non-existent-kek" with body:
      """
      {
        "doc": "This should fail"
      }
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40470

  Scenario: Soft-delete KEK
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "soft-delete-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/delete-me"
      }
      """
    When I DELETE "/dek-registry/v1/keks/soft-delete-kek"
    Then the response status should be 200

  Scenario: Soft-deleted KEK not visible in default list
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "invisible-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/invisible"
      }
      """
    And I DELETE "/dek-registry/v1/keks/invisible-kek"
    When I GET "/dek-registry/v1/keks"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 0

  Scenario: Soft-deleted KEK visible with deleted parameter
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "visible-deleted-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/visible"
      }
      """
    And I DELETE "/dek-registry/v1/keks/visible-deleted-kek"
    When I GET "/dek-registry/v1/keks?deleted=true"
    Then the response status should be 200
    And the response should be valid JSON
    And the response array should contain "visible-deleted-kek"

  Scenario: Undelete KEK
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "undelete-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/undelete"
      }
      """
    And I DELETE "/dek-registry/v1/keks/undelete-kek"
    When I PUT "/dek-registry/v1/keks/undelete-kek/undelete" with body:
      """
      {}
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "name" should be "undelete-kek"
    # Verify KEK is accessible again
    When I GET "/dek-registry/v1/keks/undelete-kek"
    Then the response status should be 200

  Scenario: Permanent delete KEK
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "permanent-delete-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/permanent"
      }
      """
    And I DELETE "/dek-registry/v1/keks/permanent-delete-kek"
    When I DELETE "/dek-registry/v1/keks/permanent-delete-kek?permanent=true"
    Then the response status should be 200
    And I GET "/dek-registry/v1/keks/permanent-delete-kek?deleted=true"
    And the response status should be 404

  Scenario: List multiple KEKs
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"list-kek-1","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/key1"}
      """
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"list-kek-2","kmsType":"azure-kms","kmsKeyId":"https://vault.azure.net/keys/key2"}
      """
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"list-kek-3","kmsType":"gcp-kms","kmsKeyId":"projects/proj/locations/us/keyRings/ring/cryptoKeys/key3"}
      """
    When I GET "/dek-registry/v1/keks"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 3
    And the response array should contain "list-kek-1"
    And the response array should contain "list-kek-2"
    And the response array should contain "list-kek-3"

  Scenario: Create KEK with missing required fields returns 422
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "incomplete-kek",
        "kmsType": "aws-kms"
      }
      """
    Then the response status should be 422
    And the response should be valid JSON
    And the response should contain "kmsKeyId"

  # ============================================================================
  # DEK CRUD Operations (15 scenarios)
  # ============================================================================

  Scenario: Create DEK under existing KEK
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "dek-test-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/dek-test"
      }
      """
    When I POST "/dek-registry/v1/keks/dek-test-kek/deks" with body:
      """
      {
        "subject": "user.email",
        "algorithm": "AES256_GCM",
        "encryptedKeyMaterial": "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="
      }
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "kekName" should be "dek-test-kek"
    And the response field "subject" should be "user.email"
    And the response field "version" should be 1
    And the response field "algorithm" should be "AES256_GCM"


  Scenario: Create DEK with default algorithm
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"default-algo-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/default"}
      """
    When I POST "/dek-registry/v1/keks/default-algo-kek/deks" with body:
      """
      {
        "subject": "order.total",
        "encryptedKeyMaterial": "ZGVmYXVsdGFsZ29yaXRobQ=="
      }
      """
    Then the response status should be 200
    And the response field "algorithm" should be "AES256_GCM"

  Scenario: Create DEK with AES128_GCM algorithm
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"aes128-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/aes128"}
      """
    When I POST "/dek-registry/v1/keks/aes128-kek/deks" with body:
      """
      {
        "subject": "payment.card",
        "algorithm": "AES128_GCM",
        "encryptedKeyMaterial": "YWVzMTI4Z2Nt"
      }
      """
    Then the response status should be 200
    And the response field "algorithm" should be "AES128_GCM"

  Scenario: Create DEK with AES256_SIV algorithm
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"aes256siv-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/aes256siv"}
      """
    When I POST "/dek-registry/v1/keks/aes256siv-kek/deks" with body:
      """
      {
        "subject": "ssn",
        "algorithm": "AES256_SIV",
        "encryptedKeyMaterial": "YWVzMjU2c2l2"
      }
      """
    Then the response status should be 200
    And the response field "algorithm" should be "AES256_SIV"

  Scenario: Create DEK with invalid algorithm returns 422
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"invalid-algo-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/invalid"}
      """
    When I POST "/dek-registry/v1/keks/invalid-algo-kek/deks" with body:
      """
      {
        "subject": "test.subject",
        "algorithm": "INVALID_ALGO",
        "encryptedKeyMaterial": "aW52YWxpZA=="
      }
      """
    Then the response status should be 422
    And the response should be valid JSON
    And the response should contain "algorithm"

  Scenario: Create DEK under non-existent KEK returns 404
    When I POST "/dek-registry/v1/keks/non-existent-kek/deks" with body:
      """
      {
        "subject": "test.subject",
        "algorithm": "AES256_GCM",
        "encryptedKeyMaterial": "bm9uZXhpc3RlbnQ="
      }
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40470
    And the response should contain "not found"

  Scenario: Create DEK with same subject creates new version
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"dup-dek-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/dup"}
      """
    And I POST "/dek-registry/v1/keks/dup-dek-kek/deks" with body:
      """
      {"subject":"duplicate.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZHVwbGljYXRl"}
      """
    When I POST "/dek-registry/v1/keks/dup-dek-kek/deks" with body:
      """
      {"subject":"duplicate.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZHVwbGljYXRl"}
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "version" should be 2

  Scenario: Get DEK for subject
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"get-dek-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/get"}
      """
    And I POST "/dek-registry/v1/keks/get-dek-kek/deks" with body:
      """
      {"subject":"get.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"Z2V0ZGVr"}
      """
    When I GET "/dek-registry/v1/keks/get-dek-kek/deks/get.subject"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "subject" should be "get.subject"
    And the response field "version" should be 1
    And the response field "algorithm" should be "AES256_GCM"

  Scenario: Get DEK with algorithm query parameter
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"algo-query-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/algo"}
      """
    And I POST "/dek-registry/v1/keks/algo-query-kek/deks" with body:
      """
      {"subject":"algo.subject","algorithm":"AES128_GCM","encryptedKeyMaterial":"YWxnb3F1ZXJ5"}
      """
    When I GET "/dek-registry/v1/keks/algo-query-kek/deks/algo.subject?algorithm=AES128_GCM"
    Then the response status should be 200
    And the response field "algorithm" should be "AES128_GCM"

  Scenario: Get non-existent DEK returns 404
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"missing-dek-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/missing"}
      """
    When I GET "/dek-registry/v1/keks/missing-dek-kek/deks/non.existent"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40471
    And the response should contain "not found"

  Scenario: List DEK subjects under KEK
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"list-deks-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/list"}
      """
    And I POST "/dek-registry/v1/keks/list-deks-kek/deks" with body:
      """
      {"subject":"subject1","algorithm":"AES256_GCM","encryptedKeyMaterial":"c3ViamVjdDE="}
      """
    And I POST "/dek-registry/v1/keks/list-deks-kek/deks" with body:
      """
      {"subject":"subject2","algorithm":"AES256_GCM","encryptedKeyMaterial":"c3ViamVjdDI="}
      """
    When I GET "/dek-registry/v1/keks/list-deks-kek/deks"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 2
    And the response array should contain "subject1"
    And the response array should contain "subject2"

  Scenario: List DEK versions
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"versions-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/versions"}
      """
    And I POST "/dek-registry/v1/keks/versions-kek/deks" with body:
      """
      {"subject":"versioned.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"dmVyc2lvbjE="}
      """
    When I GET "/dek-registry/v1/keks/versions-kek/deks/versioned.subject/versions"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 1
    And the response array should contain integer 1

  Scenario: Get DEK by specific version
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"specific-version-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/specific"}
      """
    And I POST "/dek-registry/v1/keks/specific-version-kek/deks" with body:
      """
      {"subject":"specific.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"c3BlY2lmaWM="}
      """
    When I GET "/dek-registry/v1/keks/specific-version-kek/deks/specific.subject/versions/1"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "version" should be 1
    And the response field "subject" should be "specific.subject"

  Scenario: Delete DEK
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"delete-dek-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/delete"}
      """
    And I POST "/dek-registry/v1/keks/delete-dek-kek/deks" with body:
      """
      {"subject":"delete.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZGVsZXRl"}
      """
    When I DELETE "/dek-registry/v1/keks/delete-dek-kek/deks/delete.subject"
    Then the response status should be 200

  Scenario: Undelete DEK
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"undelete-dek-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/undelete"}
      """
    And I POST "/dek-registry/v1/keks/undelete-dek-kek/deks" with body:
      """
      {"subject":"undelete.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"dW5kZWxldGU="}
      """
    And I DELETE "/dek-registry/v1/keks/undelete-dek-kek/deks/undelete.subject"
    When I PUT "/dek-registry/v1/keks/undelete-dek-kek/deks/undelete.subject/undelete" with body:
      """
      {}
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "subject" should be "undelete.subject"

  # ============================================================================
  # Advanced Scenarios (10 scenarios)
  # ============================================================================

  Scenario: DEK versioning with multiple versions
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"multi-version-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/multi"}
      """
    And I POST "/dek-registry/v1/keks/multi-version-kek/deks" with body:
      """
      {"subject":"versioned.data","algorithm":"AES256_GCM","encryptedKeyMaterial":"djE="}
      """
    And I POST "/dek-registry/v1/keks/multi-version-kek/deks" with body:
      """
      {"subject":"versioned.data","algorithm":"AES128_GCM","encryptedKeyMaterial":"djI="}
      """
    And I POST "/dek-registry/v1/keks/multi-version-kek/deks" with body:
      """
      {"subject":"versioned.data","algorithm":"AES256_SIV","encryptedKeyMaterial":"djM="}
      """
    When I GET "/dek-registry/v1/keks/multi-version-kek/deks/versioned.data/versions"
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain integer 1
    And the response array should contain integer 2
    And the response array should contain integer 3

  Scenario: KEK with AWS KMS type
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "aws-kms-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/aws-test",
        "kmsProps": {"region": "us-east-1"}
      }
      """
    Then the response status should be 200
    And the response field "kmsType" should be "aws-kms"

  Scenario: KEK with Azure KMS type
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "azure-kms-kek",
        "kmsType": "azure-kms",
        "kmsKeyId": "https://mykeyvault.vault.azure.net/keys/mykey/abc123",
        "kmsProps": {"tenantId": "tenant-123"}
      }
      """
    Then the response status should be 200
    And the response field "kmsType" should be "azure-kms"

  Scenario: KEK with GCP KMS type
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "gcp-kms-kek",
        "kmsType": "gcp-kms",
        "kmsKeyId": "projects/my-project/locations/global/keyRings/my-ring/cryptoKeys/my-key",
        "kmsProps": {"projectId": "my-project"}
      }
      """
    Then the response status should be 200
    And the response field "kmsType" should be "gcp-kms"

  Scenario: Multiple DEKs under same KEK with different subjects
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"multi-subject-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/multi"}
      """
    And I POST "/dek-registry/v1/keks/multi-subject-kek/deks" with body:
      """
      {"subject":"customer.email","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZW1haWw="}
      """
    And I POST "/dek-registry/v1/keks/multi-subject-kek/deks" with body:
      """
      {"subject":"customer.phone","algorithm":"AES256_GCM","encryptedKeyMaterial":"cGhvbmU="}
      """
    And I POST "/dek-registry/v1/keks/multi-subject-kek/deks" with body:
      """
      {"subject":"customer.ssn","algorithm":"AES256_SIV","encryptedKeyMaterial":"c3Nu"}
      """
    When I GET "/dek-registry/v1/keks/multi-subject-kek/deks"
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain "customer.email"
    And the response array should contain "customer.phone"
    And the response array should contain "customer.ssn"

  Scenario: Full lifecycle - create KEK, create DEKs, soft-delete, undelete, hard-delete
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"lifecycle-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/lifecycle"}
      """
    And I POST "/dek-registry/v1/keks/lifecycle-kek/deks" with body:
      """
      {"subject":"lifecycle.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"bGlmZWN5Y2xl"}
      """
    # Soft-delete and verify hidden
    When I DELETE "/dek-registry/v1/keks/lifecycle-kek"
    Then the response status should be 200
    # Undelete and verify accessible
    When I PUT "/dek-registry/v1/keks/lifecycle-kek/undelete" with body:
      """
      {}
      """
    Then the response status should be 200
    When I GET "/dek-registry/v1/keks/lifecycle-kek"
    Then the response status should be 200
    # Soft-delete again, then permanent delete
    When I DELETE "/dek-registry/v1/keks/lifecycle-kek"
    Then the response status should be 200
    When I DELETE "/dek-registry/v1/keks/lifecycle-kek?permanent=true"
    Then the response status should be 200
    When I GET "/dek-registry/v1/keks/lifecycle-kek?deleted=true"
    Then the response status should be 404

  Scenario: DEK missing required subject returns 422
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"missing-subject-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/missing"}
      """
    When I POST "/dek-registry/v1/keks/missing-subject-kek/deks" with body:
      """
      {
        "algorithm": "AES256_GCM",
        "encryptedKeyMaterial": "bWlzc2luZw=="
      }
      """
    Then the response status should be 422
    And the response should be valid JSON
    And the response should contain "subject"

  Scenario: KEK with kmsProps as key-value map
    When I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "complex-props-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/complex",
        "kmsProps": {
          "region": "us-east-1",
          "endpoint": "https://kms.us-east-1.amazonaws.com",
          "maxRetries": "3",
          "timeout": "30s",
          "environment": "production",
          "team": "security"
        }
      }
      """
    Then the response status should be 200
    And the response field "name" should be "complex-props-kek"

  Scenario: Delete DEK for non-existent KEK returns 404
    When I DELETE "/dek-registry/v1/keks/non-existent-kek/deks/some.subject"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40471

  Scenario: List DEK versions for non-existent KEK returns 404
    When I GET "/dek-registry/v1/keks/non-existent-kek/deks/some.subject/versions"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40470

  Scenario: DEK creation under soft-deleted KEK still succeeds
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"deleted-kek-dek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/deleted"}
      """
    And I DELETE "/dek-registry/v1/keks/deleted-kek-dek"
    When I POST "/dek-registry/v1/keks/deleted-kek-dek/deks" with body:
      """
      {"subject":"blocked.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"YmxvY2tlZA=="}
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "kekName" should be "deleted-kek-dek"

  Scenario: KEK shared flag reflected in response
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {
        "name": "shared-kek",
        "kmsType": "aws-kms",
        "kmsKeyId": "arn:aws:kms:us-east-1:123456789012:key/shared",
        "shared": true
      }
      """
    When I GET "/dek-registry/v1/keks/shared-kek"
    Then the response status should be 200
    And the response field "shared" should be true
