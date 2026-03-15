@mcp @encryption @ai
Feature: MCP AI Data Modeling — Encryption Lifecycle (KEK/DEK)
  An AI agent uses MCP tools to manage the full encryption lifecycle:
  creating KEKs, generating DEKs for field-level encryption, managing
  key versions, soft-delete/restore, and setting up schemas with
  encryption encoding rules linked to KEKs.

  # ==========================================================================
  # 1. AI CREATES KEK AND DEK FOR FIELD-LEVEL ENCRYPTION
  # ==========================================================================

  Scenario: AI sets up field-level encryption with KEK and DEK
    # AI creates a KEK for PII data
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-pii-kek",
        "kms_type": "unknown",
        "kms_key_id": "projects/my-project/locations/global/keyRings/pii/cryptoKeys/pii-key",
        "doc": "KEK for PII field-level encryption"
      }
      """
    Then the MCP result should contain "enc-pii-kek"
    And the MCP result should contain "unknown"
    # AI verifies the KEK was created
    When I call MCP tool "get_kek" with input:
      | name | enc-pii-kek |
    Then the MCP result should contain "enc-pii-kek"
    And the MCP result should contain "PII field-level encryption"
    # AI creates a DEK for the user-data subject
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-pii-kek",
        "subject": "user-data",
        "algorithm": "AES256_GCM",
        "encrypted_key_material": "dGVzdC1lbmNyeXB0ZWQta2V5LW1hdGVyaWFs"
      }
      """
    Then the MCP result should contain "enc-pii-kek"
    And the MCP result should contain "user-data"
    And the MCP result should contain "AES256_GCM"
    # AI retrieves the DEK
    When I call MCP tool "get_dek" with JSON input:
      """
      {
        "kek_name": "enc-pii-kek",
        "subject": "user-data",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "user-data"
    And the MCP result should contain "AES256_GCM"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | user-data              |
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

  # ==========================================================================
  # 2. AI MANAGES MULTIPLE KEKS FOR DIFFERENT DOMAINS
  # ==========================================================================

  Scenario: AI creates separate KEKs for different data domains
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-finance-kek",
        "kms_type": "unknown",
        "kms_key_id": "finance-key-001",
        "doc": "KEK for financial data encryption"
      }
      """
    Then the MCP result should contain "enc-finance-kek"
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-health-kek",
        "kms_type": "unknown",
        "kms_key_id": "health-key-001",
        "doc": "KEK for healthcare data (HIPAA)"
      }
      """
    Then the MCP result should contain "enc-health-kek"
    # AI lists all KEKs
    When I call MCP tool "list_keks"
    Then the MCP result should contain "enc-finance-kek"
    And the MCP result should contain "enc-health-kek"
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

  # ==========================================================================
  # 3. AI MANAGES DEK VERSIONS
  # ==========================================================================

  Scenario: AI creates multiple DEK versions for key rotation
    # Create KEK
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-version-kek",
        "kms_type": "unknown",
        "kms_key_id": "version-test-key"
      }
      """
    Then the MCP result should contain "enc-version-kek"
    # Create DEK v1
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-version-kek",
        "subject": "versioned-data",
        "algorithm": "AES256_GCM",
        "encrypted_key_material": "djEta2V5LW1hdGVyaWFs"
      }
      """
    Then the MCP result should contain "versioned-data"
    # Create DEK v2
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-version-kek",
        "subject": "versioned-data",
        "version": 2,
        "algorithm": "AES256_GCM",
        "encrypted_key_material": "djIta2V5LW1hdGVyaWFs"
      }
      """
    Then the MCP result should contain "versioned-data"
    # AI lists DEK versions
    When I call MCP tool "list_dek_versions" with JSON input:
      """
      {
        "kek_name": "enc-version-kek",
        "subject": "versioned-data"
      }
      """
    Then the MCP result should contain "1"
    And the MCP result should contain "2"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | versioned-data         |
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

  # ==========================================================================
  # 4. AI MANAGES DEK WITH DIFFERENT ALGORITHMS
  # ==========================================================================

  Scenario: AI creates DEKs with different encryption algorithms
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-algo-kek",
        "kms_type": "unknown",
        "kms_key_id": "algo-test-key"
      }
      """
    Then the MCP result should contain "enc-algo-kek"
    # AES256_GCM
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-algo-kek",
        "subject": "algo-gcm256",
        "algorithm": "AES256_GCM",
        "encrypted_key_material": "Z2NtMjU2LWtleQ=="
      }
      """
    Then the MCP result should contain "AES256_GCM"
    # AES128_GCM
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-algo-kek",
        "subject": "algo-gcm128",
        "algorithm": "AES128_GCM",
        "encrypted_key_material": "Z2NtMTI4LWtleQ=="
      }
      """
    Then the MCP result should contain "AES128_GCM"
    # AES256_SIV
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-algo-kek",
        "subject": "algo-siv256",
        "algorithm": "AES256_SIV",
        "encrypted_key_material": "c2l2MjU2LWtleQ=="
      }
      """
    Then the MCP result should contain "AES256_SIV"
    # AI lists all subjects under the KEK
    When I call MCP tool "list_deks" with input:
      | kek_name | enc-algo-kek |
    Then the MCP result should contain "algo-gcm256"
    And the MCP result should contain "algo-gcm128"
    And the MCP result should contain "algo-siv256"
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
      | path                 | list_deks              |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 5. AI MANAGES KEK SOFT-DELETE AND RESTORE
  # ==========================================================================

  Scenario: AI soft-deletes and restores a KEK
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-delete-kek",
        "kms_type": "unknown",
        "kms_key_id": "delete-test-key",
        "doc": "KEK to test deletion"
      }
      """
    Then the MCP result should contain "enc-delete-kek"
    # AI soft-deletes the KEK
    When I call MCP tool "delete_kek" with input:
      | name | enc-delete-kek |
    Then the MCP result should contain "true"
    # AI verifies it's no longer in the default list
    When I call MCP tool "list_keks"
    Then the MCP result should not contain "enc-delete-kek"
    # AI can still find it with deleted=true
    When I call MCP tool "get_kek" with JSON input:
      """
      {
        "name": "enc-delete-kek",
        "deleted": true
      }
      """
    Then the MCP result should contain "enc-delete-kek"
    # AI restores the KEK
    When I call MCP tool "undelete_kek" with input:
      | name | enc-delete-kek |
    Then the MCP result should contain "true"
    # AI verifies it's back in the list
    When I call MCP tool "list_keks"
    Then the MCP result should contain "enc-delete-kek"
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

  # ==========================================================================
  # 6. AI MANAGES DEK SOFT-DELETE
  # ==========================================================================

  Scenario: AI soft-deletes and restores a DEK
    # Setup KEK and DEK
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-dekdel-kek",
        "kms_type": "unknown",
        "kms_key_id": "dekdel-test-key"
      }
      """
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-dekdel-kek",
        "subject": "deletable-data",
        "algorithm": "AES256_GCM",
        "encrypted_key_material": "ZGVsZXRhYmxlLWtleQ=="
      }
      """
    Then the MCP result should contain "deletable-data"
    # AI soft-deletes the DEK (must specify version)
    When I call MCP tool "delete_dek" with JSON input:
      """
      {
        "kek_name": "enc-dekdel-kek",
        "subject": "deletable-data",
        "version": 1,
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "true"
    # AI verifies it's gone from default list
    When I call MCP tool "list_deks" with input:
      | kek_name | enc-dekdel-kek |
    Then the MCP result should not contain "deletable-data"
    # AI restores the DEK
    When I call MCP tool "undelete_dek" with JSON input:
      """
      {
        "kek_name": "enc-dekdel-kek",
        "subject": "deletable-data",
        "version": 1,
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result should contain "true"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | deletable-data         |
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

  # ==========================================================================
  # 7. AI UPDATES KEK PROPERTIES
  # ==========================================================================

  Scenario: AI updates KEK documentation and properties
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-updatable-kek",
        "kms_type": "unknown",
        "kms_key_id": "updatable-key"
      }
      """
    Then the MCP result should contain "enc-updatable-kek"
    # AI updates the KEK
    When I call MCP tool "update_kek" with JSON input:
      """
      {
        "name": "enc-updatable-kek",
        "doc": "Updated documentation for rotation",
        "kms_props": {
          "rotation_period": "90d",
          "environment": "production"
        }
      }
      """
    Then the MCP result should contain "enc-updatable-kek"
    And the MCP result should contain "Updated documentation"
    # AI verifies the update
    When I call MCP tool "get_kek" with input:
      | name | enc-updatable-kek |
    Then the MCP result should contain "Updated documentation for rotation"
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
      | path                 | get_kek                |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 8. AI COMBINES SCHEMA REGISTRATION WITH ENCRYPTION SETUP
  # ==========================================================================

  Scenario: AI sets up a complete encryption pipeline — KEK, schema with encoding rules, and DEK
    # Step 1: AI creates the KEK
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-pipeline-kek",
        "kms_type": "unknown",
        "kms_key_id": "pipeline-master-key",
        "doc": "Master key for healthcare data pipeline"
      }
      """
    Then the MCP result should contain "enc-pipeline-kek"
    # Step 2: AI registers the schema with encryption encoding rules
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "health-records-value",
        "schema": "{\"type\":\"record\",\"name\":\"PatientRecord\",\"namespace\":\"com.health\",\"fields\":[{\"name\":\"patient_id\",\"type\":\"string\"},{\"name\":\"diagnosis\",\"type\":\"string\"},{\"name\":\"ssn\",\"type\":\"string\"},{\"name\":\"notes\",\"type\":[\"null\",\"string\"],\"default\":null}]}",
        "metadata": {
          "properties": {
            "classification": "HIPAA",
            "owner": "team-health-data"
          },
          "sensitive": ["ssn", "diagnosis"]
        },
        "rule_set": {
          "encodingRules": [
            {
              "name": "encryptSSN",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "tags": ["PII", "HIPAA"],
              "params": {
                "encrypt.kek.name": "enc-pipeline-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              },
              "onFailure": "ERROR"
            },
            {
              "name": "encryptDiagnosis",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "tags": ["PHI", "HIPAA"],
              "params": {
                "encrypt.kek.name": "enc-pipeline-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              },
              "onFailure": "ERROR"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "encryptSSN"
    And the MCP result should contain "encryptDiagnosis"
    # Step 3: AI creates a DEK for the subject
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-pipeline-kek",
        "subject": "health-records-value",
        "algorithm": "AES256_GCM",
        "encrypted_key_material": "aGVhbHRoLXJlY29yZHMtZGVr"
      }
      """
    Then the MCP result should contain "health-records-value"
    # Step 4: AI verifies the complete setup
    When I call MCP tool "get_latest_schema" with input:
      | subject | health-records-value |
    Then the MCP result should contain "PatientRecord"
    And the MCP result should contain "encryptSSN"
    And the MCP result should contain "enc-pipeline-kek"
    When I call MCP tool "get_kek" with input:
      | name | enc-pipeline-kek |
    Then the MCP result should contain "healthcare data pipeline"
    When I call MCP tool "list_deks" with input:
      | kek_name | enc-pipeline-kek |
    Then the MCP result should contain "health-records-value"
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
      | path                 | list_deks              |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 9. AI HANDLES ENCRYPTION ERRORS GRACEFULLY
  # ==========================================================================

  Scenario: AI handles duplicate KEK name error
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-dup-kek",
        "kms_type": "unknown",
        "kms_key_id": "dup-key"
      }
      """
    Then the MCP result should contain "enc-dup-kek"
    # AI tries to create another KEK with the same name
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "enc-dup-kek",
        "kms_type": "unknown",
        "kms_key_id": "another-key"
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
      | path                 | create_kek             |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: AI handles non-existent KEK lookup
    When I call MCP tool "get_kek" with input:
      | name | enc-nonexistent-kek |
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
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | get_kek                |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: AI handles DEK creation for non-existent KEK
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "enc-missing-kek",
        "subject": "some-data",
        "algorithm": "AES256_GCM"
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
      | target_id            | some-data              |
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
      | path                 | create_dek             |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
