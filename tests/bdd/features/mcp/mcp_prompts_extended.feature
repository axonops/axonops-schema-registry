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
