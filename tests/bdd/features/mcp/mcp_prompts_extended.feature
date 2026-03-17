@mcp @mcp-prompts
Feature: MCP Extended Prompts — Guided Workflows and Domain Knowledge
  An AI agent uses extended MCP prompts for glossary lookups, migration
  workflows, RBAC setup, schema references, encryption lifecycle, data
  contract rules, registry health audits, and schema evolution recipes.

  # ==========================================================================
  # 1. GLOSSARY LOOKUP
  # ==========================================================================

  Scenario: Glossary lookup routes to compatibility resource
    When I get MCP prompt "glossary-lookup" with arguments:
      | topic | compatibility modes |
    Then the MCP prompt result should contain "schema://glossary/compatibility"
    And the MCP prompt description should contain "compatibility"

  # ==========================================================================
  # 2. CONFLUENT MIGRATION
  # ==========================================================================

  Scenario: Import from Confluent prompt
    When I get MCP prompt "import-from-confluent"
    Then the MCP prompt result should contain "IMPORT"
    And the MCP prompt result should contain "READWRITE"
    And the MCP prompt result should contain "wire-compatible"
    And the MCP prompt description should contain "Confluent"

  # ==========================================================================
  # 3. RBAC SETUP
  # ==========================================================================

  Scenario: Setup RBAC prompt
    When I get MCP prompt "setup-rbac"
    Then the MCP prompt result should contain "admin"
    And the MCP prompt result should contain "read"
    And the MCP prompt result should contain "write"
    And the MCP prompt result should contain "create_user"

  # ==========================================================================
  # 4. SCHEMA REFERENCES
  # ==========================================================================

  Scenario: Schema references guide prompt
    When I get MCP prompt "schema-references-guide"
    Then the MCP prompt result should contain "Avro"
    And the MCP prompt result should contain "Protobuf"
    And the MCP prompt result should contain "JSON Schema"
    And the MCP prompt result should contain "reference"

  # ==========================================================================
  # 5. FULL ENCRYPTION LIFECYCLE
  # ==========================================================================

  Scenario: Full encryption lifecycle prompt
    When I get MCP prompt "full-encryption-lifecycle"
    Then the MCP prompt result should contain "KEK"
    And the MCP prompt result should contain "DEK"
    And the MCP prompt result should contain "Key Rotation"
    And the MCP prompt result should contain "Rewrap"

  # ==========================================================================
  # 6. DATA RULES DEEP DIVE
  # ==========================================================================

  Scenario: Data rules deep dive prompt
    When I get MCP prompt "data-rules-deep-dive"
    Then the MCP prompt result should contain "Domain Rules"
    And the MCP prompt result should contain "Migration Rules"
    And the MCP prompt result should contain "Encoding Rules"
    And the MCP prompt result should contain "3-Layer Merge"

  # ==========================================================================
  # 7. REGISTRY HEALTH AUDIT
  # ==========================================================================

  Scenario: Registry health audit prompt
    When I get MCP prompt "registry-health-audit"
    Then the MCP prompt result should contain "health_check"
    And the MCP prompt result should contain "get_registry_statistics"
    And the MCP prompt result should contain "score_schema_quality"

  # ==========================================================================
  # 8. SCHEMA EVOLUTION COOKBOOK
  # ==========================================================================

  Scenario: Schema evolution cookbook prompt
    When I get MCP prompt "schema-evolution-cookbook"
    Then the MCP prompt result should contain "Recipe"
    And the MCP prompt result should contain "Add an Optional Field"
    And the MCP prompt result should contain "Rename a Field"
    And the MCP prompt result should contain "three-phase"

  # ==========================================================================
  # 9. NEW KAFKA TOPIC
  # ==========================================================================

  Scenario: New Kafka topic prompt with topic name
    When I get MCP prompt "new-kafka-topic" with arguments:
      | topic_name | user-events |
    Then the MCP prompt result should contain "user-events-key"
    And the MCP prompt result should contain "user-events-value"
    And the MCP prompt result should contain "TopicNameStrategy"
    And the MCP prompt result should contain "validate_schema"
    And the MCP prompt description should contain "user-events"

  Scenario: New Kafka topic prompt with explicit format
    When I get MCP prompt "new-kafka-topic" with arguments:
      | topic_name | orders   |
      | format     | PROTOBUF |
    Then the MCP prompt result should contain "orders-key"
    And the MCP prompt result should contain "PROTOBUF"
    And the MCP prompt description should contain "PROTOBUF"

  # ==========================================================================
  # 10. DEBUG DESERIALIZATION
  # ==========================================================================

  Scenario: Debug deserialization prompt
    When I get MCP prompt "debug-deserialization"
    Then the MCP prompt result should contain "Wire Format"
    And the MCP prompt result should contain "0x00"
    And the MCP prompt result should contain "get_schema_by_id"
    And the MCP prompt result should contain "Unknown magic byte"
    And the MCP prompt result should contain "diff_schemas"

  # ==========================================================================
  # 11. DEPRECATE SUBJECT
  # ==========================================================================

  Scenario: Deprecate subject prompt with existing subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-deprecate-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I get MCP prompt "deprecate-subject" with arguments:
      | subject | prompt-deprecate-test |
    Then the MCP prompt result should contain "prompt-deprecate-test"
    And the MCP prompt result should contain "READONLY"
    And the MCP prompt result should contain "delete_subject"
    And the MCP prompt result should contain "version: 1"

  # ==========================================================================
  # 12. CI/CD INTEGRATION
  # ==========================================================================

  Scenario: CI/CD integration prompt
    When I get MCP prompt "cicd-integration"
    Then the MCP prompt result should contain "validate_schema"
    And the MCP prompt result should contain "check_compatibility"
    And the MCP prompt result should contain "score_schema_quality"
    And the MCP prompt result should contain "register_schema"
    And the MCP prompt result should contain "Pipeline"

  # ==========================================================================
  # 13. TEAM ONBOARDING
  # ==========================================================================

  Scenario: Team onboarding prompt
    When I get MCP prompt "team-onboarding" with arguments:
      | team_name | payments |
    Then the MCP prompt result should contain "payments"
    And the MCP prompt result should contain "create_user"
    And the MCP prompt result should contain "create_apikey"
    And the MCP prompt result should contain "list_subjects"
    And the MCP prompt description should contain "payments"

  # ==========================================================================
  # 14. GOVERNANCE SETUP
  # ==========================================================================

  Scenario: Governance setup prompt
    When I get MCP prompt "governance-setup"
    Then the MCP prompt result should contain "validate_subject_name"
    And the MCP prompt result should contain "score_schema_quality"
    And the MCP prompt result should contain "Data Contracts"
    And the MCP prompt result should contain "RBAC"

  # ==========================================================================
  # 15. CROSS-CUTTING CHANGE
  # ==========================================================================

  Scenario: Cross-cutting change prompt
    When I get MCP prompt "cross-cutting-change" with arguments:
      | field_name | customer_id |
    Then the MCP prompt result should contain "customer_id"
    And the MCP prompt result should contain "find_schemas_by_field"
    And the MCP prompt result should contain "check_field_consistency"
    And the MCP prompt result should contain "check_compatibility_multi"
    And the MCP prompt description should contain "customer_id"

  # ==========================================================================
  # 16. SCHEMA REVIEW CHECKLIST
  # ==========================================================================

  Scenario: Schema review checklist prompt with existing subject
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "prompt-review-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should not be an error
    When I get MCP prompt "schema-review-checklist" with arguments:
      | subject | prompt-review-test |
    Then the MCP prompt result should contain "prompt-review-test"
    And the MCP prompt result should contain "validate_schema"
    And the MCP prompt result should contain "score_schema_quality"
    And the MCP prompt result should contain "check_compatibility"
    And the MCP prompt result should contain "version: 1"
