@mcp @data-contracts @ai
Feature: MCP AI Data Modeling — Data Contracts (Metadata & Rules)
  An AI agent uses MCP tools to manage data contracts: attaching metadata
  to schemas for data governance, defining domain rules for validation,
  encoding rules for encryption, and migration rules for schema evolution.

  # ==========================================================================
  # 1. AI REGISTERS SCHEMAS WITH METADATA
  # ==========================================================================

  Scenario: AI registers a schema with PII metadata tags
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-user-pii-value",
        "schema": "{\"type\":\"record\",\"name\":\"UserPII\",\"namespace\":\"com.gdpr\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"},{\"name\":\"phone\",\"type\":[\"null\",\"string\"],\"default\":null}]}",
        "metadata": {
          "properties": {
            "owner": "team-identity",
            "domain": "user-management",
            "classification": "restricted"
          },
          "tags": {
            "email": ["PII", "GDPR"],
            "phone": ["PII"]
          },
          "sensitive": ["email", "phone"]
        }
      }
      """
    Then the MCP result should contain "dc-user-pii-value"
    And the MCP result should contain "\"version\":1"
    # AI retrieves and verifies metadata via get_subject_metadata
    When I call MCP tool "get_subject_metadata" with input:
      | subject | dc-user-pii-value |
    Then the MCP result should contain "team-identity"
    And the MCP result should contain "restricted"
    And the MCP result should contain "PII"
    And the MCP result should contain "GDPR"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-user-pii-value      |
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
      | path                 | get_subject_metadata   |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: AI registers a schema with domain ownership metadata
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-order-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"OrderPlaced\",\"namespace\":\"com.orders\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"total\",\"type\":\"double\"}]}",
        "metadata": {
          "properties": {
            "owner": "team-commerce",
            "domain": "orders",
            "tier": "critical",
            "sla": "99.99"
          }
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I call MCP tool "get_subject_metadata" with input:
      | subject | dc-order-events-value |
    Then the MCP result should contain "team-commerce"
    And the MCP result should contain "critical"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-order-events-value  |
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
      | path                 | get_subject_metadata   |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 2. AI REGISTERS SCHEMAS WITH DOMAIN RULES
  # ==========================================================================

  Scenario: AI registers a schema with CEL domain validation rule
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-validated-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"Transaction\",\"namespace\":\"com.finance\",\"fields\":[{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":\"long\"}]}",
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
              "expr": "message.currency in ['USD', 'EUR', 'GBP']",
              "onFailure": "DLQ"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "validateAmount"
    And the MCP result should contain "validateCurrency"
    # AI retrieves the schema and verifies rules are stored
    When I call MCP tool "get_latest_schema" with input:
      | subject | dc-validated-events-value |
    Then the MCP result should contain "validateAmount"
    And the MCP result should contain "CONDITION"
    And the MCP result should contain "CEL"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-validated-events-value |
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

  Scenario: AI registers a schema with TRANSFORM domain rule
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-transformed-value",
        "schema": "{\"type\":\"record\",\"name\":\"LogEntry\",\"fields\":[{\"name\":\"message\",\"type\":\"string\"},{\"name\":\"level\",\"type\":\"string\"}]}",
        "rule_set": {
          "domainRules": [
            {
              "name": "normalizeLevel",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.level.upperCase()"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "normalizeLevel"
    And the MCP result should contain "TRANSFORM"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-transformed-value   |
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
  # 3. AI REGISTERS SCHEMAS WITH ENCODING RULES FOR ENCRYPTION
  # ==========================================================================

  Scenario: AI configures field-level encryption via encoding rules
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-encrypted-payment-value",
        "schema": "{\"type\":\"record\",\"name\":\"PaymentInfo\",\"namespace\":\"com.payments\",\"fields\":[{\"name\":\"card_number\",\"type\":\"string\"},{\"name\":\"cardholder\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"}]}",
        "metadata": {
          "properties": {"classification": "PCI-DSS"},
          "sensitive": ["card_number", "cardholder"]
        },
        "rule_set": {
          "encodingRules": [
            {
              "name": "encryptCardNumber",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "tags": ["PCI"],
              "params": {
                "encrypt.kek.name": "payment-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              },
              "onFailure": "ERROR"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "encryptCardNumber"
    And the MCP result should contain "ENCRYPT"
    And the MCP result should contain "AES256_GCM"
    # AI verifies the full schema with rules
    When I call MCP tool "get_latest_schema" with input:
      | subject | dc-encrypted-payment-value |
    Then the MCP result should contain "encryptCardNumber"
    And the MCP result should contain "payment-kek"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-encrypted-payment-value |
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
  # 4. AI REGISTERS SCHEMAS WITH MIGRATION RULES
  # ==========================================================================

  Scenario: AI sets up migration rules for schema evolution
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-migration-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"UserProfile\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}",
        "rule_set": {
          "migrationRules": [
            {
              "name": "upgradeAddTimestamp",
              "kind": "TRANSFORM",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "message.set('created_at', 0)"
            },
            {
              "name": "downgradeRemoveTimestamp",
              "kind": "TRANSFORM",
              "mode": "DOWNGRADE",
              "type": "CEL",
              "expr": "message.remove('created_at')"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "upgradeAddTimestamp"
    And the MCP result should contain "UPGRADE"
    And the MCP result should contain "downgradeRemoveTimestamp"
    And the MCP result should contain "DOWNGRADE"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-migration-events-value |
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
  # 5. AI USES COMBINED METADATA + RULES
  # ==========================================================================

  Scenario: AI registers a schema with all three rule types and metadata
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-full-contract-value",
        "schema": "{\"type\":\"record\",\"name\":\"CustomerEvent\",\"namespace\":\"com.crm\",\"fields\":[{\"name\":\"customer_id\",\"type\":\"string\"},{\"name\":\"action\",\"type\":\"string\"},{\"name\":\"ssn\",\"type\":[\"null\",\"string\"],\"default\":null}]}",
        "metadata": {
          "properties": {
            "owner": "team-crm",
            "domain": "customers",
            "classification": "confidential"
          },
          "tags": {
            "ssn": ["PII", "SENSITIVE", "GDPR"]
          },
          "sensitive": ["ssn"]
        },
        "rule_set": {
          "domainRules": [
            {
              "name": "validateAction",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.action in ['CREATE', 'UPDATE', 'DELETE']",
              "onFailure": "ERROR"
            }
          ],
          "migrationRules": [
            {
              "name": "upgradeV1toV2",
              "kind": "TRANSFORM",
              "mode": "UPDOWN",
              "type": "CEL",
              "expr": "message"
            }
          ],
          "encodingRules": [
            {
              "name": "encryptSSN",
              "kind": "TRANSFORM",
              "mode": "WRITEREAD",
              "type": "ENCRYPT",
              "tags": ["PII"],
              "params": {
                "encrypt.kek.name": "pii-kek",
                "encrypt.dek.algorithm": "AES256_GCM"
              }
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Verify metadata
    When I call MCP tool "get_subject_metadata" with input:
      | subject | dc-full-contract-value |
    Then the MCP result should contain "team-crm"
    And the MCP result should contain "confidential"
    And the MCP result should contain "PII"
    # Verify rules
    When I call MCP tool "get_latest_schema" with input:
      | subject | dc-full-contract-value |
    Then the MCP result should contain "validateAction"
    And the MCP result should contain "upgradeV1toV2"
    And the MCP result should contain "encryptSSN"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-full-contract-value |
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
  # 6. AI SETS CONFIG-LEVEL DEFAULT RULES
  # ==========================================================================

  Scenario: AI sets default metadata and rules at config level
    # AI sets default metadata for all schemas in a subject
    When I call MCP tool "set_config_full" with JSON input:
      """
      {
        "subject": "dc-config-rules-value",
        "compatibility_level": "BACKWARD",
        "default_metadata": {
          "properties": {
            "platform": "kafka",
            "environment": "production"
          }
        },
        "default_rule_set": {
          "domainRules": [
            {
              "name": "defaultValidation",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true",
              "onFailure": "ERROR"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "BACKWARD"
    # AI retrieves the full config to verify defaults
    When I call MCP tool "get_config_full" with input:
      | subject | dc-config-rules-value |
    Then the MCP result should contain "production"
    And the MCP result should contain "defaultValidation"
    # AI registers a schema — defaults should be applied
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-config-rules-value",
        "schema": "{\"type\":\"record\",\"name\":\"ConfigRuleTest\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI verifies the registered schema has default metadata
    When I call MCP tool "get_subject_metadata" with input:
      | subject | dc-config-rules-value |
    Then the MCP result should contain "production"
    And the MCP result should contain "kafka"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-config-rules-value  |
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
      | path                 | get_subject_metadata   |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  Scenario: AI sets override rules at config level
    When I call MCP tool "set_config_full" with JSON input:
      """
      {
        "subject": "dc-override-rules-value",
        "compatibility_level": "NONE",
        "override_metadata": {
          "properties": {
            "security": "internal",
            "approved": "true"
          }
        },
        "override_rule_set": {
          "domainRules": [
            {
              "name": "securityCheck",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "NONE"
    When I call MCP tool "get_config_full" with input:
      | subject | dc-override-rules-value |
    Then the MCP result should contain "internal"
    And the MCP result should contain "securityCheck"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-override-rules-value |
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
      | path                 | get_config_full        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 7. AI EVOLVES SCHEMA WITH DIFFERENT RULES PER VERSION
  # ==========================================================================

  Scenario: AI evolves a schema with updated rules in each version
    When I call MCP tool "set_config" with input:
      | subject             | dc-evolving-rules-value |
      | compatibility_level | NONE                    |
    # v1: Schema with basic validation rule
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-evolving-rules-value",
        "schema": "{\"type\":\"record\",\"name\":\"Audit\",\"fields\":[{\"name\":\"action\",\"type\":\"string\"},{\"name\":\"actor\",\"type\":\"string\"}]}",
        "rule_set": {
          "domainRules": [
            {
              "name": "checkAction",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.action) > 0"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And the MCP result should contain "checkAction"
    # v2: Schema with updated rules (add timestamp check)
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-evolving-rules-value",
        "schema": "{\"type\":\"record\",\"name\":\"Audit\",\"fields\":[{\"name\":\"action\",\"type\":\"string\"},{\"name\":\"actor\",\"type\":\"string\"},{\"name\":\"ts\",\"type\":\"long\"}]}",
        "rule_set": {
          "domainRules": [
            {
              "name": "checkAction",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.action) > 0"
            },
            {
              "name": "checkTimestamp",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.ts > 0"
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":2"
    And the MCP result should contain "checkTimestamp"
    # AI retrieves v1 to verify rules are version-specific
    When I call MCP tool "get_schema_version" with input:
      | subject | dc-evolving-rules-value |
      | version | 1                       |
    Then the MCP result should contain "checkAction"
    And the MCP result should not contain "checkTimestamp"
    # AI retrieves v2 to verify updated rules
    When I call MCP tool "get_schema_version" with input:
      | subject | dc-evolving-rules-value |
      | version | 2                       |
    Then the MCP result should contain "checkAction"
    And the MCP result should contain "checkTimestamp"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-evolving-rules-value |
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
  # 8. AI USES DISABLED RULES
  # ==========================================================================

  Scenario: AI registers a schema with a disabled rule
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-disabled-rule-value",
        "schema": "{\"type\":\"record\",\"name\":\"Beta\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "rule_set": {
          "domainRules": [
            {
              "name": "activeRule",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.id) > 0"
            },
            {
              "name": "disabledRule",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "false",
              "disabled": true
            }
          ]
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I call MCP tool "get_latest_schema" with input:
      | subject | dc-disabled-rule-value |
    Then the MCP result should contain "activeRule"
    And the MCP result should contain "disabledRule"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | dc-disabled-rule-value |
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
  # 9. METADATA DOES NOT AFFECT SCHEMA IDENTITY
  # ==========================================================================

  Scenario: AI registers same schema with different metadata — same ID
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-identity-test-a",
        "schema": "{\"type\":\"record\",\"name\":\"IdentityTest\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}",
        "metadata": {
          "properties": {"owner": "team-a"}
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    And I store the MCP result field "id" as "schema_id"
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "dc-identity-test-b",
        "schema": "{\"type\":\"record\",\"name\":\"IdentityTest\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}",
        "metadata": {
          "properties": {"owner": "team-b"}
        }
      }
      """
    Then the MCP result should contain "\"version\":1"
    # Both subjects should share the same global schema ID
    When I call MCP tool "get_subjects_for_schema" with input:
      | id | $schema_id |
    Then the MCP result should contain "dc-identity-test-a"
    And the MCP result should contain "dc-identity-test-b"
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
