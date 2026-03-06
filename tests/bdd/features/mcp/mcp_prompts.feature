@mcp @mcp-prompts
Feature: MCP Prompts — Pre-Built Conversation Templates
  An AI agent uses MCP prompts to get context-aware guidance for
  schema design, evolution, compatibility troubleshooting, encryption
  setup, and format comparison.

  # ==========================================================================
  # 1. SCHEMA DESIGN PROMPTS
  # ==========================================================================

  Scenario: Get Avro schema design prompt
    When I get MCP prompt "design-schema" with arguments:
      | format | AVRO        |
      | domain | user-events |
    Then the MCP prompt result should contain "Avro"
    And the MCP prompt result should contain "user-events"
    And the MCP prompt result should contain "register_schema"
    And the MCP prompt result should contain "Design Workflow"
    And the MCP prompt result should contain "OrderCreated"
    And the MCP prompt result should contain "Common Mistakes"
    And the MCP prompt result should contain "Starter Template"
    And the MCP prompt description should contain "AVRO"

  Scenario: Get Protobuf schema design prompt
    When I get MCP prompt "design-schema" with arguments:
      | format | PROTOBUF |
    Then the MCP prompt result should contain "Protobuf"
    And the MCP prompt result should contain "proto3"
    And the MCP prompt result should contain "Design Workflow"
    And the MCP prompt result should contain "OrderCreated"
    And the MCP prompt result should contain "Common Mistakes"
    And the MCP prompt result should contain "UNSPECIFIED"
    And the MCP prompt description should contain "PROTOBUF"

  Scenario: Get JSON Schema design prompt
    When I get MCP prompt "design-schema" with arguments:
      | format | JSON |
    Then the MCP prompt result should contain "JSON Schema"
    And the MCP prompt result should contain "additionalProperties"
    And the MCP prompt result should contain "Design Workflow"
    And the MCP prompt result should contain "OrderCreated"
    And the MCP prompt result should contain "Common Mistakes"
    And the MCP prompt result should contain "Starter Template"

  # ==========================================================================
  # 2. SCHEMA EVOLUTION PROMPT
  # ==========================================================================

  Scenario: Get schema evolution prompt with existing subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-evolve-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I get MCP prompt "evolve-schema" with arguments:
      | subject | prompt-evolve-test |
    Then the MCP prompt result should contain "prompt-evolve-test"
    And the MCP prompt result should contain "version: 1"
    And the MCP prompt result should contain "check_compatibility"

  # ==========================================================================
  # 3. COMPATIBILITY TROUBLESHOOTING PROMPT
  # ==========================================================================

  Scenario: Get compatibility troubleshooting prompt
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-compat-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I get MCP prompt "check-compatibility" with arguments:
      | subject | prompt-compat-test |
    Then the MCP prompt result should contain "prompt-compat-test"
    And the MCP prompt result should contain "BACKWARD"

  # ==========================================================================
  # 4. FORMAT COMPARISON PROMPT
  # ==========================================================================

  Scenario: Get format comparison prompt for event streaming
    When I get MCP prompt "compare-formats" with arguments:
      | use_case | event streaming |
    Then the MCP prompt result should contain "Avro"
    And the MCP prompt result should contain "Protobuf"
    And the MCP prompt result should contain "JSON Schema"
    And the MCP prompt result should contain "event streaming"

  # ==========================================================================
  # 5. ERROR DEBUGGING PROMPTS
  # ==========================================================================

  Scenario: Get debug prompt for invalid schema error
    When I get MCP prompt "debug-registration-error" with arguments:
      | error_code | 42201 |
    Then the MCP prompt result should contain "Invalid schema"
    And the MCP prompt result should contain "42201"
    And the MCP prompt result should contain "validate_schema"

  Scenario: Get debug prompt for incompatible schema error
    When I get MCP prompt "debug-registration-error" with arguments:
      | error_code | 409 |
    Then the MCP prompt result should contain "Incompatible schema"
    And the MCP prompt result should contain "check_compatibility"
    And the MCP prompt result should contain "explain_compatibility_failure"

  Scenario: Get debug prompt for subject not found error
    When I get MCP prompt "debug-registration-error" with arguments:
      | error_code | 40401 |
    Then the MCP prompt result should contain "Subject"
    And the MCP prompt result should contain "list_subjects"
    And the MCP prompt result should contain "match_subjects"

  Scenario: Get debug prompt for unknown error code falls back to general guide
    When I get MCP prompt "debug-registration-error" with arguments:
      | error_code | 99999 |
    Then the MCP prompt result should contain "Decision Tree"
    And the MCP prompt result should contain "health_check"

  # ==========================================================================
  # 6. ENCRYPTION AND EXPORT PROMPTS
  # ==========================================================================

  Scenario: Get encryption setup prompt for Vault
    When I get MCP prompt "setup-encryption" with arguments:
      | kms_type | hcvault |
    Then the MCP prompt result should contain "KEK"
    And the MCP prompt result should contain "DEK"
    And the MCP prompt result should contain "hcvault"
    And the MCP prompt result should contain "test_kek"
    And the MCP prompt result should contain "vault.address"
    And the MCP prompt result should contain "Transit"

  Scenario: Get encryption setup prompt for AWS KMS
    When I get MCP prompt "setup-encryption" with arguments:
      | kms_type | aws-kms |
    Then the MCP prompt result should contain "aws-kms"
    And the MCP prompt result should contain "aws.region"
    And the MCP prompt result should contain "ARN"

  Scenario: Get exporter configuration prompt
    When I get MCP prompt "configure-exporter" with arguments:
      | exporter_type | AUTO |
    Then the MCP prompt result should contain "AUTO"
    And the MCP prompt result should contain "create_exporter"

  # ==========================================================================
  # 7. DATA CONTRACTS AND HISTORY PROMPTS
  # ==========================================================================

  Scenario: Get data contracts setup prompt
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-contracts-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I get MCP prompt "setup-data-contracts" with arguments:
      | subject | prompt-contracts-test |
    Then the MCP prompt result should contain "prompt-contracts-test"
    And the MCP prompt result should contain "metadata"
    And the MCP prompt result should contain "ruleSet"

  Scenario: Get subject audit history prompt
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-audit-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I get MCP prompt "audit-subject-history" with arguments:
      | subject | prompt-audit-test |
    Then the MCP prompt result should contain "prompt-audit-test"
    And the MCP prompt result should contain "list_versions"

  # ==========================================================================
  # 8. BREAKING CHANGE AND MIGRATION PROMPTS
  # ==========================================================================

  Scenario: Get breaking change planning prompt
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-breaking-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I get MCP prompt "plan-breaking-change" with arguments:
      | subject | prompt-breaking-test |
    Then the MCP prompt result should contain "prompt-breaking-test"
    And the MCP prompt result should contain "READONLY"

  Scenario: Get schema migration prompt Avro to Protobuf
    When I get MCP prompt "migrate-schemas" with arguments:
      | source_format | AVRO     |
      | target_format | PROTOBUF |
    Then the MCP prompt result should contain "AVRO"
    And the MCP prompt result should contain "PROTOBUF"
    And the MCP prompt result should contain "message"
    And the MCP prompt result should contain "Type Mapping"
    And the MCP prompt result should contain "validate_schema"
    And the MCP prompt result should contain "UNSPECIFIED"

  Scenario: Get schema migration prompt Protobuf to Avro
    When I get MCP prompt "migrate-schemas" with arguments:
      | source_format | PROTOBUF |
      | target_format | AVRO     |
    Then the MCP prompt result should contain "PROTOBUF"
    And the MCP prompt result should contain "AVRO"
    And the MCP prompt result should contain "record"
    And the MCP prompt result should contain "Field numbers"

  Scenario: Get schema migration prompt Avro to JSON Schema
    When I get MCP prompt "migrate-schemas" with arguments:
      | source_format | AVRO |
      | target_format | JSON |
    Then the MCP prompt result should contain "AVRO"
    And the MCP prompt result should contain "JSON"
    And the MCP prompt result should contain "additionalProperties"

  Scenario: Get schema quality review prompt
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-quality-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I get MCP prompt "review-schema-quality" with arguments:
      | subject | prompt-quality-test |
    Then the MCP prompt result should contain "prompt-quality-test"
    And the MCP prompt result should contain "Naming conventions"

  # ==========================================================================
  # 9. GETTING STARTED AND TROUBLESHOOTING PROMPTS
  # ==========================================================================

  Scenario: Get getting-started prompt
    When I get MCP prompt "schema-getting-started"
    Then the MCP prompt result should contain "list_subjects"
    And the MCP prompt result should contain "register_schema"
    And the MCP prompt result should contain "check_compatibility"

  Scenario: Get troubleshooting prompt
    When I get MCP prompt "troubleshooting"
    Then the MCP prompt result should contain "health_check"
    And the MCP prompt result should contain "42201"
    And the MCP prompt result should contain "409"

  Scenario: Get impact analysis prompt
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-impact-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I get MCP prompt "schema-impact-analysis" with arguments:
      | subject | prompt-impact-test |
    Then the MCP prompt result should contain "prompt-impact-test"
    And the MCP prompt result should contain "get_dependency_graph"
    And the MCP prompt result should contain "check_compatibility"

  Scenario: Get naming conventions prompt
    When I get MCP prompt "schema-naming-conventions"
    Then the MCP prompt result should contain "topic_name"
    And the MCP prompt result should contain "record_name"
    And the MCP prompt result should contain "topic_record_name"

  Scenario: Get context management prompt
    When I get MCP prompt "context-management"
    Then the MCP prompt result should contain "list_contexts"
    And the MCP prompt result should contain "READWRITE"
    And the MCP prompt result should contain "highest precedence"
    And the MCP prompt result should contain "lowest precedence"
