@mcp @kms @data-contracts @ai
Feature: MCP E2E — Data Rules & Encryption Pipeline
  An AI agent uses MCP tools to set up complete data governance pipelines
  combining schema registration, metadata tagging, domain validation rules,
  encoding rules for field-level encryption, and migration rules — all
  backed by real Vault/OpenBao Transit KMS for actual key generation.

  # ==========================================================================
  # 1. FULL PIPELINE: SCHEMA + ENCODING RULES + REAL KEK/DEK
  # ==========================================================================

  Scenario: AI sets up a complete HIPAA encryption pipeline with real KMS
    # Step 1: Create shared KEK backed by Vault Transit
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "hipaa-vault-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "doc": "HIPAA field-level encryption KEK",
        "shared": true
      }
      """
    Then the MCP result should contain "hipaa-vault-kek"
    # Step 2: Register schema with encoding rules referencing the KEK
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-patient-records-value",
        "schema": "{\"type\":\"record\",\"name\":\"PatientRecord\",\"namespace\":\"com.health\",\"fields\":[{\"name\":\"patient_id\",\"type\":\"string\"},{\"name\":\"diagnosis\",\"type\":\"string\"},{\"name\":\"ssn\",\"type\":\"string\"},{\"name\":\"notes\",\"type\":[\"null\",\"string\"],\"default\":null}]}",
        "metadata": {
          "properties": {
            "classification": "HIPAA",
            "owner": "team-health-data",
            "retention": "7y"
          },
          "tags": {
            "ssn": ["PII", "HIPAA"],
            "diagnosis": ["PHI", "HIPAA"]
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
                "encrypt.kek.name": "hipaa-vault-kek",
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
                "encrypt.kek.name": "hipaa-vault-kek",
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
    # Step 3: Generate real DEK via Vault Transit
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "hipaa-vault-kek",
        "subject": "e2e-patient-records-value",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # Step 4: Verify the complete setup
    When I call MCP tool "get_latest_schema" with input:
      | subject | e2e-patient-records-value |
    Then the MCP result should contain "PatientRecord"
    And the MCP result should contain "encryptSSN"
    And the MCP result should contain "hipaa-vault-kek"
    When I call MCP tool "get_subject_metadata" with input:
      | subject | e2e-patient-records-value |
    Then the MCP result should contain "HIPAA"
    And the MCP result should contain "team-health-data"
    When I call MCP tool "list_deks" with input:
      | kek_name | hipaa-vault-kek |
    Then the MCP result should contain "e2e-patient-records-value"
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
  # 2. DOMAIN RULES + ENCODING RULES + KMS — FULL DATA CONTRACT
  # ==========================================================================

  Scenario: AI creates a PCI-DSS data contract with validation and encryption
    # Create KEK for payment data
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "pci-vault-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "doc": "PCI-DSS payment encryption KEK",
        "shared": true
      }
      """
    Then the MCP result should contain "pci-vault-kek"
    # Register schema with domain rules AND encoding rules
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-payment-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"PaymentEvent\",\"namespace\":\"com.payments\",\"fields\":[{\"name\":\"transaction_id\",\"type\":\"string\"},{\"name\":\"card_number\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {
            "classification": "PCI-DSS",
            "owner": "team-payments",
            "compliance": "PCI-DSS-v4.0"
          },
          "tags": {
            "card_number": ["PCI", "PAN", "SENSITIVE"]
          },
          "sensitive": ["card_number"]
        },
        "rule_set": {
          "domainRules": [
            {
              "name": "validateAmount",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.amount > 0",
              "onFailure": "ERROR"
            },
            {
              "name": "validateCurrency",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.currency in ['USD', 'EUR', 'GBP', 'JPY']",
              "onFailure": "DLQ"
            }
          ],
          "encodingRules": [
            {
              "name": "encryptCardNumber",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "tags": ["PCI"],
              "params": {
                "encrypt.kek.name": "pci-vault-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              },
              "onFailure": "ERROR"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "validateAmount"
    And the MCP result should contain "validateCurrency"
    And the MCP result should contain "encryptCardNumber"
    # Generate real DEK
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "pci-vault-kek",
        "subject": "e2e-payment-events-value",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # Verify full setup
    When I call MCP tool "get_latest_schema" with input:
      | subject | e2e-payment-events-value |
    Then the MCP result should contain "validateAmount"
    And the MCP result should contain "encryptCardNumber"
    And the MCP result should contain "pci-vault-kek"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | e2e-payment-events-value |
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
  # 3. SCHEMA EVOLUTION WITH RULES AND ENCRYPTION
  # ==========================================================================

  Scenario: AI evolves a schema with different encryption rules per version
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "evolve-vault-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    # Set compatibility to NONE for easy evolution
    When I call MCP tool "set_config" with input:
      | subject             | e2e-evolving-contract-value |
      | compatibility_level | NONE                        |
    # v1: Schema with SSN encryption only
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-evolving-contract-value",
        "schema": "{\"type\":\"record\",\"name\":\"UserData\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"ssn\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"owner": "team-identity", "version_note": "v1-ssn-only"},
          "sensitive": ["ssn"]
        },
        "rule_set": {
          "encodingRules": [
            {
              "name": "encryptSSN",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "evolve-vault-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "encryptSSN"
    # v2: Schema adds email field with its own encryption rule
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-evolving-contract-value",
        "schema": "{\"type\":\"record\",\"name\":\"UserData\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"ssn\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"owner": "team-identity", "version_note": "v2-add-email-encryption"},
          "sensitive": ["ssn", "email"]
        },
        "rule_set": {
          "encodingRules": [
            {
              "name": "encryptSSN",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "evolve-vault-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            },
            {
              "name": "encryptEmail",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "evolve-vault-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":2"
    And the MCP result should contain "encryptSSN"
    And the MCP result should contain "encryptEmail"
    # Generate DEK for subject
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "evolve-vault-kek",
        "subject": "e2e-evolving-contract-value",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # Verify v1 has only SSN encryption
    When I call MCP tool "get_schema_version" with input:
      | subject | e2e-evolving-contract-value |
      | version | 1                           |
    Then the MCP result should contain "encryptSSN"
    And the MCP result should not contain "encryptEmail"
    # Verify v2 has both
    When I call MCP tool "get_schema_version" with input:
      | subject | e2e-evolving-contract-value |
      | version | 2                           |
    Then the MCP result should contain "encryptSSN"
    And the MCP result should contain "encryptEmail"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | e2e-evolving-contract-value |
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
      | path                 | get_schema_version     |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 4. CONFIG-LEVEL DEFAULT RULES WITH KMS ENCRYPTION
  # ==========================================================================

  Scenario: AI sets default encryption rules at config level with real KMS
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "config-default-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    # Set default encoding rules at config level
    When I call MCP tool "set_config_full" with JSON input:
      """
      {
        "subject": "e2e-config-enc-value",
        "compatibility_level": "BACKWARD",
        "default_metadata": {
          "properties": {
            "platform": "kafka",
            "encryption": "enabled"
          }
        },
        "default_rule_set": {
          "encodingRules": [
            {
              "name": "defaultEncrypt",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "config-default-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "BACKWARD"
    # Register a schema — defaults should apply
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-config-enc-value",
        "schema": "{\"type\":\"record\",\"name\":\"ConfigTest\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"data\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Verify defaults were applied
    When I call MCP tool "get_config_full" with input:
      | subject | e2e-config-enc-value |
    Then the MCP result should contain "defaultEncrypt"
    And the MCP result should contain "config-default-kek"
    # Generate DEK
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "config-default-kek",
        "subject": "e2e-config-enc-value",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | e2e-config-enc-value   |
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

  # ==========================================================================
  # 5. MIGRATION RULES WITH ENCRYPTION PIPELINE
  # ==========================================================================

  Scenario: AI combines migration and encoding rules in a versioned pipeline
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "migration-vault-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    When I call MCP tool "set_config" with input:
      | subject             | e2e-migration-pipeline-value |
      | compatibility_level | NONE                         |
    # v1: Basic schema with encryption
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-migration-pipeline-value",
        "schema": "{\"type\":\"record\",\"name\":\"AuditEvent\",\"fields\":[{\"name\":\"actor\",\"type\":\"string\"},{\"name\":\"action\",\"type\":\"string\"}]}",
        "rule_set": {
          "encodingRules": [
            {
              "name": "encryptActor",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "migration-vault-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    # v2: Add field with migration rule + keep encryption
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-migration-pipeline-value",
        "schema": "{\"type\":\"record\",\"name\":\"AuditEvent\",\"fields\":[{\"name\":\"actor\",\"type\":\"string\"},{\"name\":\"action\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":\"long\"}]}",
        "rule_set": {
          "migrationRules": [
            {
              "name": "upgradeAddTimestamp",
              "kind": "TRANSFORM",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "message.set('timestamp', 0)"
            },
            {
              "name": "downgradeRemoveTimestamp",
              "kind": "TRANSFORM",
              "mode": "DOWNGRADE",
              "type": "CEL",
              "expr": "message.remove('timestamp')"
            }
          ],
          "encodingRules": [
            {
              "name": "encryptActor",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "migration-vault-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":2"
    And the MCP result should contain "upgradeAddTimestamp"
    And the MCP result should contain "downgradeRemoveTimestamp"
    And the MCP result should contain "encryptActor"
    # Generate DEK with real KMS
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "migration-vault-kek",
        "subject": "e2e-migration-pipeline-value",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | e2e-migration-pipeline-value |
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

  # ==========================================================================
  # 6. MULTI-DOMAIN ENCRYPTION — SEPARATE KEKs PER DATA DOMAIN
  # ==========================================================================

  Scenario: AI sets up separate encryption domains with independent KEKs
    # Finance domain — Vault KEK
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "domain-finance-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "doc": "Finance domain encryption",
        "shared": true
      }
      """
    # Healthcare domain — OpenBao KEK
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "domain-health-kek",
        "kms_type": "openbao",
        "kms_key_id": "test-key",
        "doc": "Healthcare domain encryption",
        "shared": true
      }
      """
    # Finance schema with encoding rules
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-finance-txn-value",
        "schema": "{\"type\":\"record\",\"name\":\"Transaction\",\"namespace\":\"com.finance\",\"fields\":[{\"name\":\"txn_id\",\"type\":\"string\"},{\"name\":\"account_number\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"}]}",
        "metadata": {
          "properties": {"domain": "finance", "classification": "confidential"},
          "sensitive": ["account_number"]
        },
        "rule_set": {
          "encodingRules": [
            {
              "name": "encryptAccount",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "domain-finance-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Healthcare schema with encoding rules
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-health-records-value",
        "schema": "{\"type\":\"record\",\"name\":\"HealthRecord\",\"namespace\":\"com.health\",\"fields\":[{\"name\":\"patient_id\",\"type\":\"string\"},{\"name\":\"diagnosis\",\"type\":\"string\"},{\"name\":\"lab_results\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"domain": "healthcare", "classification": "HIPAA"},
          "sensitive": ["diagnosis", "lab_results"]
        },
        "rule_set": {
          "encodingRules": [
            {
              "name": "encryptDiagnosis",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "domain-health-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Generate DEKs via different KMS backends
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "domain-finance-kek",
        "subject": "e2e-finance-txn-value",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And I store the MCP result field "keyMaterial" as "finance_key"
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "domain-health-kek",
        "subject": "e2e-health-records-value",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "keyMaterial" should not equal stored "finance_key"
    And I can unwrap the MCP result encrypted key material using KMS type "openbao" and key ID "test-key"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | e2e-health-records-value |
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

  # ==========================================================================
  # 7. DOMAIN RULES + DISABLED RULES + ENCRYPTION — MIXED PIPELINE
  # ==========================================================================

  Scenario: AI uses domain rules with disabled flags alongside encryption
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "mixed-rules-kek",
        "kms_type": "hcvault",
        "kms_key_id": "test-key",
        "shared": true
      }
      """
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-mixed-rules-value",
        "schema": "{\"type\":\"record\",\"name\":\"MixedRules\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"secret\",\"type\":\"string\"},{\"name\":\"status\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"owner": "team-platform"},
          "sensitive": ["secret"]
        },
        "rule_set": {
          "domainRules": [
            {
              "name": "activeValidation",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.id) > 0",
              "onFailure": "ERROR"
            },
            {
              "name": "betaValidation",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.status in ['active', 'pending']",
              "disabled": true
            }
          ],
          "encodingRules": [
            {
              "name": "encryptSecret",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "mixed-rules-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "activeValidation"
    And the MCP result should contain "betaValidation"
    And the MCP result should contain "encryptSecret"
    # Generate real DEK
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "mixed-rules-kek",
        "subject": "e2e-mixed-rules-value",
        "algorithm": "AES256_GCM"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "hcvault" and key ID "test-key"
    # Verify all rules persisted
    When I call MCP tool "get_latest_schema" with input:
      | subject | e2e-mixed-rules-value |
    Then the MCP result should contain "activeValidation"
    And the MCP result should contain "betaValidation"
    And the MCP result should contain "encryptSecret"
    And the MCP result should contain "mixed-rules-kek"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | e2e-mixed-rules-value  |
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
  # 8. OPENBAO-BACKED FULL PIPELINE
  # ==========================================================================

  Scenario: AI sets up a complete encryption pipeline using OpenBao Transit
    When I call MCP tool "create_kek" with JSON input:
      """
      {
        "name": "bao-pipeline-kek",
        "kms_type": "openbao",
        "kms_key_id": "test-key",
        "doc": "OpenBao-backed pipeline KEK",
        "shared": true
      }
      """
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "e2e-bao-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"SecureEvent\",\"namespace\":\"com.security\",\"fields\":[{\"name\":\"event_id\",\"type\":\"string\"},{\"name\":\"payload\",\"type\":\"string\"},{\"name\":\"source_ip\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"classification": "internal", "owner": "team-security"},
          "sensitive": ["payload", "source_ip"]
        },
        "rule_set": {
          "domainRules": [
            {
              "name": "validateEventId",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.event_id) > 0",
              "onFailure": "ERROR"
            }
          ],
          "encodingRules": [
            {
              "name": "encryptPayload",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "params": {
                "encrypt.kek.name": "bao-pipeline-kek",
                "encrypt.dek.algorithm": "AES256_SIV"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "encryptPayload"
    And the MCP result should contain "AES256_SIV"
    # Generate DEK via OpenBao
    When I call MCP tool "create_dek" with JSON input:
      """
      {
        "kek_name": "bao-pipeline-kek",
        "subject": "e2e-bao-events-value",
        "algorithm": "AES256_SIV"
      }
      """
    Then the MCP result field "keyMaterial" should be non-empty
    And the MCP result field "encryptedKeyMaterial" should be non-empty
    And I can unwrap the MCP result encrypted key material using KMS type "openbao" and key ID "test-key"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | e2e-bao-events-value   |
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
