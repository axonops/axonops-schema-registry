@functional @data-contracts
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
    And the audit log should contain an event:
      | event_type          | kek_create             |
      | outcome             | success                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | method              | POST                   |
      | path                | /dek-registry/v1/keks  |
      | status_code         | 200                    |
      | target_type         | kek                    |
      | target_id           | test-kek-full          |
      | schema_id           |                        |
      | version             |                        |
      | schema_type         |                        |
      | context             | .                       |
      | before_hash         |                        |
      | after_hash          | sha256:*               |
      | transport_security  | tls                    |
      | reason              |                        |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |

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
    And the audit log should contain an event:
      | event_type          | kek_create             |
      | outcome             | success                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | method              | POST                   |
      | path                | /dek-registry/v1/keks  |
      | status_code         | 200                    |
      | target_type         | kek                    |
      | target_id           | test-kek-minimal       |
      | schema_id           |                        |
      | version             |                        |
      | schema_type         |                        |
      | context             | .                       |
      | before_hash         |                        |
      | after_hash          | sha256:*               |
      | transport_security  | tls                    |
      | reason              |                        |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |

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
    And the audit log should contain an event:
      | event_type          | kek_update                            |
      | outcome             | success                               |
      | actor_id            |                                       |
      | actor_type          | anonymous                             |
      | auth_method         |                                       |
      | role                |                                       |
      | method              | PUT                                   |
      | path                | /dek-registry/v1/keks/update-test-kek |
      | status_code         | 200                                   |
      | target_type         | kek                                   |
      | target_id           | update-test-kek                       |
      | schema_id           |                                       |
      | version             |                                       |
      | schema_type         |                                       |
      | context             | .                                      |
      | before_hash         | sha256:*                              |
      | after_hash          | sha256:*                              |
      | transport_security  | tls                                   |
      | reason              |                                       |
      | error               |                                       |
      | request_body        |                                       |
      | metadata            |                                       |
      | timestamp           | *                                     |
      | duration_ms         | *                                     |
      | request_id          | *                                     |
      | source_ip           | *                                     |
      | user_agent          | *                                     |

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
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type          | kek_delete_soft                       |
      | outcome             | success                               |
      | actor_id            |                                       |
      | actor_type          | anonymous                             |
      | auth_method         |                                       |
      | role                |                                       |
      | method              | DELETE                                |
      | path                | /dek-registry/v1/keks/soft-delete-kek |
      | status_code         | 204                                   |
      | target_type         | kek                                   |
      | target_id           | soft-delete-kek                       |
      | schema_id           |                                       |
      | version             |                                       |
      | schema_type         |                                       |
      | context             | .                                      |
      | before_hash         | sha256:*                              |
      | after_hash          |                                       |
      | transport_security  | tls                                   |
      | reason              |                                       |
      | error               |                                       |
      | request_body        |                                       |
      | metadata            |                                       |
      | timestamp           | *                                     |
      | duration_ms         | *                                     |
      | request_id          | *                                     |
      | source_ip           | *                                     |
      | user_agent          | *                                     |

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
    And the response array should not contain "invisible-kek"

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
    When I POST "/dek-registry/v1/keks/undelete-kek/undelete" with body:
      """
      {}
      """
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type           | kek_undelete                                  |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | kek                                           |
      | target_id            | undelete-kek                                  |
      | schema_id            |                                               |
      | version              |                                               |
      | schema_type          |                                               |
      | before_hash          | sha256:*                                      |
      | after_hash           | sha256:*                                      |
      | context              | .                                              |
      | transport_security   | tls                                           |
      | method               | POST                                          |
      | path                 | /dek-registry/v1/keks/undelete-kek/undelete   |
      | status_code          | 204                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
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
    Then the response status should be 204
    And I GET "/dek-registry/v1/keks/permanent-delete-kek?deleted=true"
    And the response status should be 404
    And the audit log should contain an event:
      | event_type          | kek_delete_permanent                       |
      | outcome             | success                                    |
      | actor_id            |                                            |
      | actor_type          | anonymous                                  |
      | auth_method         |                                            |
      | role                |                                            |
      | method              | DELETE                                     |
      | path                | /dek-registry/v1/keks/permanent-delete-kek |
      | status_code         | 204                                        |
      | target_type         | kek                                        |
      | target_id           | permanent-delete-kek                       |
      | schema_id           |                                            |
      | version             |                                            |
      | schema_type         |                                            |
      | context             | .                                           |
      | before_hash         | sha256:*                                   |
      | after_hash          |                                            |
      | transport_security  | tls                                        |
      | reason              |                                            |
      | error               |                                            |
      | request_body        |                                            |
      | metadata            |                                            |
      | timestamp           | *                                          |
      | duration_ms         | *                                          |
      | request_id          | *                                          |
      | source_ip           | *                                          |
      | user_agent          | *                                          |

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
    And the audit log should contain an event:
      | event_type          | dek_create                              |
      | outcome             | success                                 |
      | actor_id            |                                         |
      | actor_type          | anonymous                               |
      | auth_method         |                                         |
      | role                |                                         |
      | method              | POST                                    |
      | path                | /dek-registry/v1/keks/dek-test-kek/deks |
      | status_code         | 200                                     |
      | target_type         | dek                                     |
      | target_id           | dek-test-kek                            |
      | schema_id           |                                         |
      | version             |                                         |
      | schema_type         |                                         |
      | context             | .                                        |
      | before_hash         |                                         |
      | after_hash          | sha256:*                                |
      | transport_security  | tls                                     |
      | reason              |                                         |
      | error               |                                         |
      | request_body        |                                         |
      | metadata            |                                         |
      | timestamp           | *                                       |
      | duration_ms         | *                                       |
      | request_id          | *                                       |
      | source_ip           | *                                       |
      | user_agent          | *                                       |

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
    And the audit log should contain an event:
      | event_type          | dek_create                                  |
      | outcome             | success                                     |
      | actor_id            |                                             |
      | actor_type          | anonymous                                   |
      | auth_method         |                                             |
      | role                |                                             |
      | method              | POST                                        |
      | path                | /dek-registry/v1/keks/default-algo-kek/deks |
      | status_code         | 200                                         |
      | target_type         | dek                                         |
      | target_id           | default-algo-kek                            |
      | schema_id           |                                             |
      | version             |                                             |
      | schema_type         |                                             |
      | context             | .                                            |
      | before_hash         |                                             |
      | after_hash          | sha256:*                                    |
      | transport_security  | tls                                         |
      | reason              |                                             |
      | error               |                                             |
      | request_body        |                                             |
      | metadata            |                                             |
      | timestamp           | *                                           |
      | duration_ms         | *                                           |
      | request_id          | *                                           |
      | source_ip           | *                                           |
      | user_agent          | *                                           |

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
    And the audit log should contain an event:
      | event_type          | dek_create                            |
      | outcome             | success                               |
      | actor_id            |                                       |
      | actor_type          | anonymous                             |
      | auth_method         |                                       |
      | role                |                                       |
      | method              | POST                                  |
      | path                | /dek-registry/v1/keks/aes128-kek/deks |
      | status_code         | 200                                   |
      | target_type         | dek                                   |
      | target_id           | aes128-kek                            |
      | schema_id           |                                       |
      | version             |                                       |
      | schema_type         |                                       |
      | context             | .                                      |
      | before_hash         |                                       |
      | after_hash          | sha256:*                              |
      | transport_security  | tls                                   |
      | reason              |                                       |
      | error               |                                       |
      | request_body        |                                       |
      | metadata            |                                       |
      | timestamp           | *                                     |
      | duration_ms         | *                                     |
      | request_id          | *                                     |
      | source_ip           | *                                     |
      | user_agent          | *                                     |

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
    And the audit log should contain an event:
      | event_type          | dek_create                               |
      | outcome             | success                                  |
      | actor_id            |                                          |
      | actor_type          | anonymous                                |
      | auth_method         |                                          |
      | role                |                                          |
      | method              | POST                                     |
      | path                | /dek-registry/v1/keks/aes256siv-kek/deks |
      | status_code         | 200                                      |
      | target_type         | dek                                      |
      | target_id           | aes256siv-kek                            |
      | schema_id           |                                          |
      | version             |                                          |
      | schema_type         |                                          |
      | context             | .                                         |
      | before_hash         |                                          |
      | after_hash          | sha256:*                                 |
      | transport_security  | tls                                      |
      | reason              |                                          |
      | error               |                                          |
      | request_body        |                                          |
      | metadata            |                                          |
      | timestamp           | *                                        |
      | duration_ms         | *                                        |
      | request_id          | *                                        |
      | source_ip           | *                                        |
      | user_agent          | *                                        |

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
    And the audit log should contain an event:
      | event_type          | dek_create                             |
      | outcome             | success                                |
      | actor_id            |                                        |
      | actor_type          | anonymous                              |
      | auth_method         |                                        |
      | role                |                                        |
      | method              | POST                                   |
      | path                | /dek-registry/v1/keks/dup-dek-kek/deks |
      | status_code         | 200                                    |
      | target_type         | dek                                    |
      | target_id           | dup-dek-kek                            |
      | schema_id           |                                        |
      | version             |                                        |
      | schema_type         |                                        |
      | context             | .                                       |
      | before_hash         |                                        |
      | after_hash          | sha256:*                               |
      | transport_security  | tls                                    |
      | reason              |                                        |
      | error               |                                        |
      | request_body        |                                        |
      | metadata            |                                        |
      | timestamp           | *                                      |
      | duration_ms         | *                                      |
      | request_id          | *                                      |
      | source_ip           | *                                      |
      | user_agent          | *                                      |

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
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type          | dek_delete_soft                                          |
      | outcome             | success                                                  |
      | actor_id            |                                                          |
      | actor_type          | anonymous                                                |
      | auth_method         |                                                          |
      | role                |                                                          |
      | method              | DELETE                                                   |
      | path                | /dek-registry/v1/keks/delete-dek-kek/deks/delete.subject |
      | status_code         | 204                                                      |
      | target_type         | dek                                                      |
      | target_id           | delete-dek-kek                                           |
      | schema_id           |                                                          |
      | version             |                                                          |
      | schema_type         |                                                          |
      | context             | .                                                         |
      | before_hash         | sha256:*                                                 |
      | after_hash          |                                                          |
      | transport_security  | tls                                                      |
      | reason              |                                                          |
      | error               |                                                          |
      | request_body        |                                                          |
      | metadata            |                                                          |
      | timestamp           | *                                                        |
      | duration_ms         | *                                                        |
      | request_id          | *                                                        |
      | source_ip           | *                                                        |
      | user_agent          | *                                                        |

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
    When I POST "/dek-registry/v1/keks/undelete-dek-kek/deks/undelete.subject/undelete" with body:
      """
      {}
      """
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type           | dek_undelete                                                        |
      | outcome              | success                                                             |
      | actor_id             |                                                                     |
      | actor_type           | anonymous                                                           |
      | auth_method          |                                                                     |
      | role                 |                                                                     |
      | target_type          | dek                                                                 |
      | target_id            | undelete-dek-kek                                                    |
      | schema_id            |                                                                     |
      | version              |                                                                     |
      | schema_type          |                                                                     |
      | before_hash          | sha256:*                                                            |
      | after_hash           | sha256:*                                                            |
      | context              | .                                                                    |
      | transport_security   | tls                                                                 |
      | method               | POST                                                                |
      | path                 | /dek-registry/v1/keks/undelete-dek-kek/deks/undelete.subject/undelete |
      | status_code          | 204                                                                 |
      | reason               |                                                                     |
      | error                |                                                                     |
      | request_body         |                                                                     |
      | metadata             |                                                                     |
      | timestamp            | *                                                                   |
      | duration_ms          | *                                                                   |
      | request_id           | *                                                                   |
      | source_ip            | *                                                                   |
      | user_agent           | *                                                                   |

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
    And the audit log should contain an event:
      | event_type          | kek_create             |
      | outcome             | success                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | method              | POST                   |
      | path                | /dek-registry/v1/keks  |
      | status_code         | 200                    |
      | target_type         | kek                    |
      | target_id           | aws-kms-kek            |
      | schema_id           |                        |
      | version             |                        |
      | schema_type         |                        |
      | context             | .                       |
      | before_hash         |                        |
      | after_hash          | sha256:*               |
      | transport_security  | tls                    |
      | reason              |                        |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |

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
    And the audit log should contain an event:
      | event_type          | kek_create             |
      | outcome             | success                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | method              | POST                   |
      | path                | /dek-registry/v1/keks  |
      | status_code         | 200                    |
      | target_type         | kek                    |
      | target_id           | azure-kms-kek          |
      | schema_id           |                        |
      | version             |                        |
      | schema_type         |                        |
      | context             | .                       |
      | before_hash         |                        |
      | after_hash          | sha256:*               |
      | transport_security  | tls                    |
      | reason              |                        |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |

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
    And the audit log should contain an event:
      | event_type          | kek_create             |
      | outcome             | success                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | method              | POST                   |
      | path                | /dek-registry/v1/keks  |
      | status_code         | 200                    |
      | target_type         | kek                    |
      | target_id           | gcp-kms-kek            |
      | schema_id           |                        |
      | version             |                        |
      | schema_type         |                        |
      | context             | .                       |
      | before_hash         |                        |
      | after_hash          | sha256:*               |
      | transport_security  | tls                    |
      | reason              |                        |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |

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
    Then the response status should be 204
    # Undelete and verify accessible
    When I POST "/dek-registry/v1/keks/lifecycle-kek/undelete" with body:
      """
      {}
      """
    Then the response status should be 204
    When I GET "/dek-registry/v1/keks/lifecycle-kek"
    Then the response status should be 200
    # Soft-delete again, then permanent delete
    When I DELETE "/dek-registry/v1/keks/lifecycle-kek"
    Then the response status should be 204
    When I DELETE "/dek-registry/v1/keks/lifecycle-kek?permanent=true"
    Then the response status should be 204
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
    And the audit log should contain an event:
      | event_type          | kek_create             |
      | outcome             | success                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | method              | POST                   |
      | path                | /dek-registry/v1/keks  |
      | status_code         | 200                    |
      | target_type         | kek                    |
      | target_id           | complex-props-kek      |
      | schema_id           |                        |
      | version             |                        |
      | schema_type         |                        |
      | context             | .                       |
      | before_hash         |                        |
      | after_hash          | sha256:*               |
      | transport_security  | tls                    |
      | reason              |                        |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |

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
    And the audit log should contain an event:
      | event_type          | dek_create                                 |
      | outcome             | success                                    |
      | actor_id            |                                            |
      | actor_type          | anonymous                                  |
      | auth_method         |                                            |
      | role                |                                            |
      | method              | POST                                       |
      | path                | /dek-registry/v1/keks/deleted-kek-dek/deks |
      | status_code         | 200                                        |
      | target_type         | dek                                        |
      | target_id           | deleted-kek-dek                            |
      | schema_id           |                                            |
      | version             |                                            |
      | schema_type         |                                            |
      | context             | .                                           |
      | before_hash         |                                            |
      | after_hash          | sha256:*                                   |
      | transport_security  | tls                                        |
      | reason              |                                            |
      | error               |                                            |
      | request_body        |                                            |
      | metadata            |                                            |
      | timestamp           | *                                          |
      | duration_ms         | *                                          |
      | request_id          | *                                          |
      | source_ip           | *                                          |
      | user_agent          | *                                          |

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

  # ============================================================================
  # DEK Version Validation (4 scenarios)
  # ============================================================================

  Scenario: Get DEK version 0 returns 422 invalid version
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"ver0-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/ver0"}
      """
    And I POST "/dek-registry/v1/keks/ver0-kek/deks" with body:
      """
      {"subject":"ver0.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"djA="}
      """
    When I GET "/dek-registry/v1/keks/ver0-kek/deks/ver0.subject/versions/0"
    Then the response status should be 422
    And the response should be valid JSON
    And the response should have error code 42202
    And the response should contain "positive integer"

  Scenario: Get DEK negative version returns 422 invalid version
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"verneg-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/verneg"}
      """
    And I POST "/dek-registry/v1/keks/verneg-kek/deks" with body:
      """
      {"subject":"verneg.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"bmVn"}
      """
    When I GET "/dek-registry/v1/keks/verneg-kek/deks/verneg.subject/versions/-1"
    Then the response status should be 422
    And the response should be valid JSON
    And the response should have error code 42202
    And the response should contain "positive integer"

  Scenario: Get DEK non-numeric version returns 422 invalid version
    When I GET "/dek-registry/v1/keks/any-kek/deks/any.subject/versions/abc"
    Then the response status should be 422
    And the response should be valid JSON
    And the response should have error code 42202
    And the response should contain "positive integer"

  Scenario: Get DEK version -999 returns 422 invalid version
    When I GET "/dek-registry/v1/keks/any-kek/deks/any.subject/versions/-999"
    Then the response status should be 422
    And the response should be valid JSON
    And the response should have error code 42202
    And the response should contain "positive integer"

  Scenario: GET DEK does not return plaintext key material
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"keymaterial-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/km"}
      """
    And I POST "/dek-registry/v1/keks/keymaterial-kek/deks" with body:
      """
      {"subject":"km.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"a2V5bWF0ZXJpYWw="}
      """
    When I GET "/dek-registry/v1/keks/keymaterial-kek/deks/km.subject"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "encryptedKeyMaterial" should be "a2V5bWF0ZXJpYWw="
    And the response field "keyMaterial" should be empty or absent

  # ============================================================================
  # Confluent Wire-Compatibility: New Endpoints (5 scenarios)
  # ============================================================================

  Scenario: Create DEK with subject in path
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"path-create-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/pathcreate"}
      """
    When I POST "/dek-registry/v1/keks/path-create-kek/deks/path.subject" with body:
      """
      {"algorithm":"AES256_GCM","encryptedKeyMaterial":"cGF0aGNyZWF0ZQ=="}
      """
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "kekName" should be "path-create-kek"
    And the response field "subject" should be "path.subject"
    And the response field "version" should be 1
    And the audit log should contain an event:
      | event_type          | dek_create                                              |
      | outcome             | success                                                 |
      | actor_id            |                                                         |
      | actor_type          | anonymous                                               |
      | auth_method         |                                                         |
      | role                |                                                         |
      | method              | POST                                                    |
      | path                | /dek-registry/v1/keks/path-create-kek/deks/path.subject |
      | status_code         | 200                                                     |
      | target_type         | dek                                                     |
      | target_id           | path-create-kek                                         |
      | schema_id           |                                                         |
      | version             |                                                         |
      | schema_type         |                                                         |
      | context             | .                                                        |
      | before_hash         |                                                         |
      | after_hash          | sha256:*                                                |
      | transport_security  | tls                                                     |
      | reason              |                                                         |
      | error               |                                                         |
      | request_body        |                                                         |
      | metadata            |                                                         |
      | timestamp           | *                                                       |
      | duration_ms         | *                                                       |
      | request_id          | *                                                       |
      | source_ip           | *                                                       |
      | user_agent          | *                                                       |

  Scenario: Create DEK with subject in path and empty body
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"empty-body-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/emptybody"}
      """
    When I POST "/dek-registry/v1/keks/empty-body-kek/deks/empty.subject"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "subject" should be "empty.subject"
    And the response field "algorithm" should be "AES256_GCM"
    And the audit log should contain an event:
      | event_type          | dek_create                                              |
      | outcome             | success                                                 |
      | actor_id            |                                                         |
      | actor_type          | anonymous                                               |
      | auth_method         |                                                         |
      | role                |                                                         |
      | method              | POST                                                    |
      | path                | /dek-registry/v1/keks/empty-body-kek/deks/empty.subject |
      | status_code         | 200                                                     |
      | target_type         | dek                                                     |
      | target_id           | empty-body-kek                                          |
      | schema_id           |                                                         |
      | version             |                                                         |
      | schema_type         |                                                         |
      | context             | .                                                        |
      | before_hash         |                                                         |
      | after_hash          | sha256:*                                                |
      | transport_security  | tls                                                     |
      | reason              |                                                         |
      | error               |                                                         |
      | request_body        |                                                         |
      | metadata            |                                                         |
      | timestamp           | *                                                       |
      | duration_ms         | *                                                       |
      | request_id          | *                                                       |
      | source_ip           | *                                                       |
      | user_agent          | *                                                       |

  Scenario: Delete DEK by specific version
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"delver-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/delver"}
      """
    And I POST "/dek-registry/v1/keks/delver-kek/deks" with body:
      """
      {"subject":"delver.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZGVsdmVy"}
      """
    When I DELETE "/dek-registry/v1/keks/delver-kek/deks/delver.subject/versions/1"
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type          | dek_delete_soft                                                 |
      | outcome             | success                                                         |
      | actor_id            |                                                                 |
      | actor_type          | anonymous                                                       |
      | auth_method         |                                                                 |
      | role                |                                                                 |
      | method              | DELETE                                                          |
      | path                | /dek-registry/v1/keks/delver-kek/deks/delver.subject/versions/1 |
      | status_code         | 204                                                             |
      | target_type         | dek                                                             |
      | target_id           | delver-kek                                                      |
      | schema_id           |                                                                 |
      | version             | 1                                                               |
      | schema_type         |                                                                 |
      | context             | .                                                                |
      | before_hash         | sha256:*                                                        |
      | after_hash          |                                                                 |
      | transport_security  | tls                                                             |
      | reason              |                                                                 |
      | error               |                                                                 |
      | request_body        |                                                                 |
      | metadata            |                                                                 |
      | timestamp           | *                                                               |
      | duration_ms         | *                                                               |
      | request_id          | *                                                               |
      | source_ip           | *                                                               |
      | user_agent          | *                                                               |

  Scenario: Undelete DEK by specific version
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"undelver-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/undelver"}
      """
    And I POST "/dek-registry/v1/keks/undelver-kek/deks" with body:
      """
      {"subject":"undelver.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"dW5kZWx2ZXI="}
      """
    And I DELETE "/dek-registry/v1/keks/undelver-kek/deks/undelver.subject/versions/1"
    When I POST "/dek-registry/v1/keks/undelver-kek/deks/undelver.subject/versions/1/undelete" with body:
      """
      {}
      """
    Then the response status should be 204
    # Verify DEK version is accessible again
    When I GET "/dek-registry/v1/keks/undelver-kek/deks/undelver.subject/versions/1"
    Then the response status should be 200

  Scenario: Test KEK returns 422 without KMS configured
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"test-kms-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/testkms"}
      """
    When I POST "/dek-registry/v1/keks/test-kms-kek/test"
    Then the response status should be 422
    And the response should be valid JSON

  # ============================================================================
  # Rewrap DEK Scenarios (3 scenarios)
  # ============================================================================

  Scenario: Rewrap DEK returns 422 without KMS configured
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"rewrap-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/rewrap"}
      """
    And I POST "/dek-registry/v1/keks/rewrap-kek/deks" with body:
      """
      {"subject":"rewrap.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"cmV3cmFw"}
      """
    When I POST "/dek-registry/v1/keks/rewrap-kek/deks/rewrap.subject?rewrap=true" with body:
      """
      {}
      """
    Then the response status should be 422
    And the response should be valid JSON

  Scenario: Rewrap DEK for non-existent KEK returns 422 without KMS
    # Without KMS configured, rewrap returns 422 before checking KEK existence
    When I POST "/dek-registry/v1/keks/rewrap-missing-kek/deks/any.subject?rewrap=true" with body:
      """
      {}
      """
    Then the response status should be 422
    And the response should be valid JSON

  Scenario: Rewrap DEK for non-existent DEK returns 422 without KMS
    # Without KMS configured, rewrap returns 422 before checking DEK existence
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"rewrap-nodek-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/rewnodek"}
      """
    When I POST "/dek-registry/v1/keks/rewrap-nodek-kek/deks/missing.subject?rewrap=true" with body:
      """
      {}
      """
    Then the response status should be 422
    And the response should be valid JSON

  # ============================================================================
  # Error Path Scenarios (10 scenarios)
  # ============================================================================

  Scenario: Delete non-existent KEK returns 404
    When I DELETE "/dek-registry/v1/keks/nonexistent-delete-kek"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40470
    And the response should contain "not found"

  Scenario: Undelete non-existent KEK returns 404
    When I POST "/dek-registry/v1/keks/nonexistent-undelete-kek/undelete" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40470

  Scenario: List DEKs for non-existent KEK returns 404
    When I GET "/dek-registry/v1/keks/nonexistent-list-dek-kek/deks"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40470

  Scenario: Delete DEK version for non-existent KEK returns 404
    When I DELETE "/dek-registry/v1/keks/nonexistent-delver-kek/deks/some.subject/versions/1"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40471

  Scenario: Delete DEK version that does not exist returns 404
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"delver-miss-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/delvermiss"}
      """
    When I DELETE "/dek-registry/v1/keks/delver-miss-kek/deks/nonexistent.subject/versions/1"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40471

  Scenario: Undelete DEK version for non-existent KEK returns 404
    When I POST "/dek-registry/v1/keks/nonexistent-undver-kek/deks/some.subject/versions/1/undelete" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40471

  Scenario: Undelete DEK version that does not exist returns 404
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"undver-miss-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/undvermiss"}
      """
    When I POST "/dek-registry/v1/keks/undver-miss-kek/deks/nonexistent.subject/versions/1/undelete" with body:
      """
      {}
      """
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40471

  Scenario: Delete DEK version with invalid version returns 422
    When I DELETE "/dek-registry/v1/keks/any-kek/deks/any.subject/versions/abc"
    Then the response status should be 422
    And the response should be valid JSON
    And the response should have error code 42202
    And the response should contain "positive integer"

  Scenario: Delete DEK version with zero returns 422
    When I DELETE "/dek-registry/v1/keks/any-kek/deks/any.subject/versions/0"
    Then the response status should be 422
    And the response should be valid JSON
    And the response should have error code 42202

  Scenario: Undelete DEK version with invalid version returns 422
    When I POST "/dek-registry/v1/keks/any-kek/deks/any.subject/versions/abc/undelete" with body:
      """
      {}
      """
    Then the response status should be 422
    And the response should be valid JSON
    And the response should have error code 42202
    And the response should contain "positive integer"

  # ============================================================================
  # Permanent Delete Scenarios (3 scenarios)
  # ============================================================================

  Scenario: Permanent delete DEK after soft-delete
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"permdel-dek-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/permdel"}
      """
    And I POST "/dek-registry/v1/keks/permdel-dek-kek/deks" with body:
      """
      {"subject":"permdel.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"cGVybWRlbA=="}
      """
    And I DELETE "/dek-registry/v1/keks/permdel-dek-kek/deks/permdel.subject"
    When I DELETE "/dek-registry/v1/keks/permdel-dek-kek/deks/permdel.subject?permanent=true"
    Then the response status should be 204
    # Verify DEK is gone even with ?deleted=true
    When I GET "/dek-registry/v1/keks/permdel-dek-kek/deks/permdel.subject?deleted=true"
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type          | dek_delete_permanent                                       |
      | outcome             | success                                                    |
      | actor_id            |                                                            |
      | actor_type          | anonymous                                                  |
      | auth_method         |                                                            |
      | role                |                                                            |
      | method              | DELETE                                                     |
      | path                | /dek-registry/v1/keks/permdel-dek-kek/deks/permdel.subject |
      | status_code         | 204                                                        |
      | target_type         | dek                                                        |
      | target_id           | permdel-dek-kek                                            |
      | schema_id           |                                                            |
      | version             |                                                            |
      | schema_type         |                                                            |
      | context             | .                                                           |
      | before_hash         | sha256:*                                                   |
      | after_hash          |                                                            |
      | transport_security  | tls                                                        |
      | reason              |                                                            |
      | error               |                                                            |
      | request_body        |                                                            |
      | metadata            |                                                            |
      | timestamp           | *                                                          |
      | duration_ms         | *                                                          |
      | request_id          | *                                                          |
      | source_ip           | *                                                          |
      | user_agent          | *                                                          |

  Scenario: Permanent delete DEK version after soft-delete
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"permdel-ver-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/permdelver"}
      """
    And I POST "/dek-registry/v1/keks/permdel-ver-kek/deks" with body:
      """
      {"subject":"permdel.ver.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"cGVybWRlbHZlcg=="}
      """
    And I DELETE "/dek-registry/v1/keks/permdel-ver-kek/deks/permdel.ver.subject/versions/1"
    When I DELETE "/dek-registry/v1/keks/permdel-ver-kek/deks/permdel.ver.subject/versions/1?permanent=true"
    Then the response status should be 204
    # Verify version is gone even with ?deleted=true
    When I GET "/dek-registry/v1/keks/permdel-ver-kek/deks/permdel.ver.subject/versions/1?deleted=true"
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type          | dek_delete_permanent                                                      |
      | outcome             | success                                                                   |
      | actor_id            |                                                                           |
      | actor_type          | anonymous                                                                 |
      | auth_method         |                                                                           |
      | role                |                                                                           |
      | method              | DELETE                                                                    |
      | path                | /dek-registry/v1/keks/permdel-ver-kek/deks/permdel.ver.subject/versions/1 |
      | status_code         | 204                                                                       |
      | target_type         | dek                                                                       |
      | target_id           | permdel-ver-kek                                                           |
      | schema_id           |                                                                           |
      | version             | 1                                                                         |
      | schema_type         |                                                                           |
      | context             | .                                                                          |
      | before_hash         | sha256:*                                                                  |
      | after_hash          |                                                                           |
      | transport_security  | tls                                                                       |
      | reason              |                                                                           |
      | error               |                                                                           |
      | request_body        |                                                                           |
      | metadata            |                                                                           |
      | timestamp           | *                                                                         |
      | duration_ms         | *                                                                         |
      | request_id          | *                                                                         |
      | source_ip           | *                                                                         |
      | user_agent          | *                                                                         |

  Scenario: Permanent delete DEK without soft-delete first
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"permdel-direct-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/permdeldirect"}
      """
    And I POST "/dek-registry/v1/keks/permdel-direct-kek/deks" with body:
      """
      {"subject":"permdel.direct.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZGlyZWN0"}
      """
    When I DELETE "/dek-registry/v1/keks/permdel-direct-kek/deks/permdel.direct.subject?permanent=true"
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type          | dek_delete_permanent                                                  |
      | outcome             | success                                                               |
      | actor_id            |                                                                       |
      | actor_type          | anonymous                                                             |
      | auth_method         |                                                                       |
      | role                |                                                                       |
      | method              | DELETE                                                                |
      | path                | /dek-registry/v1/keks/permdel-direct-kek/deks/permdel.direct.subject  |
      | status_code         | 204                                                                   |
      | target_type         | dek                                                                   |
      | target_id           | permdel-direct-kek                                                    |
      | schema_id           |                                                                       |
      | version             |                                                                       |
      | schema_type         |                                                                       |
      | context             | .                                                                      |
      | before_hash         | sha256:*                                                              |
      | after_hash          |                                                                       |
      | transport_security  | tls                                                                   |
      | reason              |                                                                       |
      | error               |                                                                       |
      | request_body        |                                                                       |
      | metadata            |                                                                       |
      | timestamp           | *                                                                     |
      | duration_ms         | *                                                                     |
      | request_id          | *                                                                     |
      | source_ip           | *                                                                     |
      | user_agent          | *                                                                     |

  # ============================================================================
  # Algorithm Filter Scenarios (3 scenarios)
  # ============================================================================

  Scenario: Get DEK with wrong algorithm returns 404
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"algo-filter-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/algofilter"}
      """
    And I POST "/dek-registry/v1/keks/algo-filter-kek/deks" with body:
      """
      {"subject":"algo.filter.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"YWxnb2ZpbHRlcg=="}
      """
    When I GET "/dek-registry/v1/keks/algo-filter-kek/deks/algo.filter.subject?algorithm=AES128_GCM"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40471

  Scenario: Delete DEK filtered by algorithm
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"algo-del-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/algodel"}
      """
    And I POST "/dek-registry/v1/keks/algo-del-kek/deks" with body:
      """
      {"subject":"algo.del.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"YWxnb2RlbA=="}
      """
    When I DELETE "/dek-registry/v1/keks/algo-del-kek/deks/algo.del.subject?algorithm=AES256_GCM"
    Then the response status should be 204
    And the audit log should contain an event:
      | event_type          | dek_delete_soft                                          |
      | outcome             | success                                                  |
      | actor_id            |                                                          |
      | actor_type          | anonymous                                                |
      | auth_method         |                                                          |
      | role                |                                                          |
      | method              | DELETE                                                   |
      | path                | /dek-registry/v1/keks/algo-del-kek/deks/algo.del.subject |
      | status_code         | 204                                                      |
      | target_type         | dek                                                      |
      | target_id           | algo-del-kek                                             |
      | schema_id           |                                                          |
      | version             |                                                          |
      | schema_type         |                                                          |
      | context             | .                                                         |
      | before_hash         | sha256:*                                                 |
      | after_hash          |                                                          |
      | transport_security  | tls                                                      |
      | reason              |                                                          |
      | error               |                                                          |
      | request_body        |                                                          |
      | metadata            |                                                          |
      | timestamp           | *                                                        |
      | duration_ms         | *                                                        |
      | request_id          | *                                                        |
      | source_ip           | *                                                        |
      | user_agent          | *                                                        |

  Scenario: List DEK versions filtered by algorithm
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"algo-list-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/algolist"}
      """
    And I POST "/dek-registry/v1/keks/algo-list-kek/deks" with body:
      """
      {"subject":"algo.list.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"YWxnb2xpc3Q="}
      """
    And I POST "/dek-registry/v1/keks/algo-list-kek/deks" with body:
      """
      {"subject":"algo.list.subject","algorithm":"AES128_GCM","encryptedKeyMaterial":"YWxnb2xpc3Qy"}
      """
    When I GET "/dek-registry/v1/keks/algo-list-kek/deks/algo.list.subject/versions?algorithm=AES256_GCM"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 1

  # ============================================================================
  # Deleted Flag Scenarios (4 scenarios)
  # ============================================================================

  Scenario: Soft-deleted DEK not visible in default get
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"delflag-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/delflag"}
      """
    And I POST "/dek-registry/v1/keks/delflag-kek/deks" with body:
      """
      {"subject":"delflag.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZGVsZmxhZw=="}
      """
    And I DELETE "/dek-registry/v1/keks/delflag-kek/deks/delflag.subject"
    When I GET "/dek-registry/v1/keks/delflag-kek/deks/delflag.subject"
    Then the response status should be 404
    And the response should have error code 40471

  Scenario: Soft-deleted DEK visible with deleted parameter
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"delflag-vis-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/delflagvis"}
      """
    And I POST "/dek-registry/v1/keks/delflag-vis-kek/deks" with body:
      """
      {"subject":"delflag.vis.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZGVsZmxhZ3Zpcw=="}
      """
    And I DELETE "/dek-registry/v1/keks/delflag-vis-kek/deks/delflag.vis.subject"
    When I GET "/dek-registry/v1/keks/delflag-vis-kek/deks/delflag.vis.subject?deleted=true"
    Then the response status should be 200
    And the response should be valid JSON
    And the response field "subject" should be "delflag.vis.subject"
    And the response field "deleted" should be true

  Scenario: List DEK subjects excludes soft-deleted by default
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"delflag-list-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/delflaglist"}
      """
    And I POST "/dek-registry/v1/keks/delflag-list-kek/deks" with body:
      """
      {"subject":"delflag.list.active","algorithm":"AES256_GCM","encryptedKeyMaterial":"YWN0aXZl"}
      """
    And I POST "/dek-registry/v1/keks/delflag-list-kek/deks" with body:
      """
      {"subject":"delflag.list.deleted","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZGVsZXRlZA=="}
      """
    And I DELETE "/dek-registry/v1/keks/delflag-list-kek/deks/delflag.list.deleted"
    When I GET "/dek-registry/v1/keks/delflag-list-kek/deks"
    Then the response status should be 200
    And the response should be valid JSON
    And the response array should contain "delflag.list.active"
    And the response array should not contain "delflag.list.deleted"

  Scenario: List DEK subjects includes soft-deleted with deleted parameter
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"delflag-listdel-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/delflaglistdel"}
      """
    And I POST "/dek-registry/v1/keks/delflag-listdel-kek/deks" with body:
      """
      {"subject":"delflag.listdel.active","algorithm":"AES256_GCM","encryptedKeyMaterial":"YWN0aXZl"}
      """
    And I POST "/dek-registry/v1/keks/delflag-listdel-kek/deks" with body:
      """
      {"subject":"delflag.listdel.deleted","algorithm":"AES256_GCM","encryptedKeyMaterial":"ZGVsZXRlZA=="}
      """
    And I DELETE "/dek-registry/v1/keks/delflag-listdel-kek/deks/delflag.listdel.deleted"
    When I GET "/dek-registry/v1/keks/delflag-listdel-kek/deks?deleted=true"
    Then the response status should be 200
    And the response should be valid JSON
    And the response array should contain "delflag.listdel.active"
    And the response array should contain "delflag.listdel.deleted"

  # ============================================================================
  # Pagination Scenarios (3 scenarios)
  # ============================================================================

  Scenario: List KEKs with offset and limit
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"page-kek-1","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/page1"}
      """
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"page-kek-2","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/page2"}
      """
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"page-kek-3","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/page3"}
      """
    When I GET "/dek-registry/v1/keks?offset=0&limit=2"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 2

  Scenario: List DEK subjects with offset and limit
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"page-dek-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/pagedek"}
      """
    And I POST "/dek-registry/v1/keks/page-dek-kek/deks" with body:
      """
      {"subject":"page.subject.1","algorithm":"AES256_GCM","encryptedKeyMaterial":"cGFnZTE="}
      """
    And I POST "/dek-registry/v1/keks/page-dek-kek/deks" with body:
      """
      {"subject":"page.subject.2","algorithm":"AES256_GCM","encryptedKeyMaterial":"cGFnZTI="}
      """
    And I POST "/dek-registry/v1/keks/page-dek-kek/deks" with body:
      """
      {"subject":"page.subject.3","algorithm":"AES256_GCM","encryptedKeyMaterial":"cGFnZTM="}
      """
    When I GET "/dek-registry/v1/keks/page-dek-kek/deks?offset=1&limit=1"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 1

  Scenario: List DEK versions with offset and limit
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"page-ver-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123456789012:key/pagever"}
      """
    And I POST "/dek-registry/v1/keks/page-ver-kek/deks" with body:
      """
      {"subject":"page.ver.subject","algorithm":"AES256_GCM","encryptedKeyMaterial":"djE="}
      """
    And I POST "/dek-registry/v1/keks/page-ver-kek/deks" with body:
      """
      {"subject":"page.ver.subject","algorithm":"AES128_GCM","encryptedKeyMaterial":"djI="}
      """
    And I POST "/dek-registry/v1/keks/page-ver-kek/deks" with body:
      """
      {"subject":"page.ver.subject","algorithm":"AES256_SIV","encryptedKeyMaterial":"djM="}
      """
    When I GET "/dek-registry/v1/keks/page-ver-kek/deks/page.ver.subject/versions?offset=0&limit=2"
    Then the response status should be 200
    And the response should be valid JSON
    And the response should be an array of length 2
