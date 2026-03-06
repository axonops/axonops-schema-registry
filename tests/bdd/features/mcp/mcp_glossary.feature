@mcp @mcp-glossary
Feature: MCP Glossary Resources — Domain Knowledge for AI Assistants
  An AI agent reads glossary resources to get comprehensive domain knowledge
  about schema registry concepts. Each glossary resource is a static markdown
  document covering a specific topic area.

  # ==========================================================================
  # GLOSSARY RESOURCES
  # ==========================================================================

  Scenario: Read core-concepts glossary
    When I read MCP resource "schema://glossary/core-concepts"
    Then the MCP resource result should contain "schema registry"
    And the MCP resource result should contain "subject"
    And the MCP resource result should contain "Schema IDs"
    And the MCP resource result should contain "Wire Format"
    And the MCP resource result should contain "Deduplication"
    And the MCP resource result should contain "context"
    And the MCP resource result should contain "schema://glossary/contexts"

  Scenario: Read compatibility glossary
    When I read MCP resource "schema://glossary/compatibility"
    Then the MCP resource result should contain "BACKWARD"
    And the MCP resource result should contain "FORWARD"
    And the MCP resource result should contain "FULL_TRANSITIVE"
    And the MCP resource result should contain "Type Promotions"

  Scenario: Read data-contracts glossary
    When I read MCP resource "schema://glossary/data-contracts"
    Then the MCP resource result should contain "metadata"
    And the MCP resource result should contain "ruleSet"
    And the MCP resource result should contain "3-Layer Merge"
    And the MCP resource result should contain "Optimistic Concurrency"

  Scenario: Read encryption glossary
    When I read MCP resource "schema://glossary/encryption"
    Then the MCP resource result should contain "KEK"
    And the MCP resource result should contain "DEK"
    And the MCP resource result should contain "Envelope Encryption"
    And the MCP resource result should contain "AES256_GCM"

  Scenario: Read contexts glossary
    When I read MCP resource "schema://glossary/contexts"
    Then the MCP resource result should contain "multi-tenant"
    And the MCP resource result should contain "__GLOBAL"
    And the MCP resource result should contain "4-Tier"

  Scenario: Read contexts glossary has MCP support section
    When I read MCP resource "schema://glossary/contexts"
    Then the MCP resource result should contain "Context Support in MCP"
    And the MCP resource result should contain "78+"
    And the MCP resource result should contain "schema://contexts/{context}/subjects"
    And the MCP resource result should contain "schema://contexts/{context}/schemas/{id}"
    And the MCP resource result should contain "evolve-schema"
    And the MCP resource result should contain "check-compatibility"

  Scenario: Read exporters glossary
    When I read MCP resource "schema://glossary/exporters"
    Then the MCP resource result should contain "schema linking"
    And the MCP resource result should contain "RUNNING"
    And the MCP resource result should contain "PAUSED"

  Scenario: Read schema-types glossary
    When I read MCP resource "schema://glossary/schema-types"
    Then the MCP resource result should contain "Avro"
    And the MCP resource result should contain "Protobuf"
    And the MCP resource result should contain "JSON Schema"
    And the MCP resource result should contain "Logical Type"

  Scenario: Read design-patterns glossary
    When I read MCP resource "schema://glossary/design-patterns"
    Then the MCP resource result should contain "Event Envelope"
    And the MCP resource result should contain "Snapshot"
    And the MCP resource result should contain "Three-Phase Rename"

  Scenario: Read best-practices glossary
    When I read MCP resource "schema://glossary/best-practices"
    Then the MCP resource result should contain "Avro Best Practices"
    And the MCP resource result should contain "Protobuf Best Practices"
    And the MCP resource result should contain "Anti-Patterns"
    And the MCP resource result should contain "String Everything"
    And the MCP resource result should contain "Context Best Practices"

  Scenario: Read design-patterns glossary with schema examples
    When I read MCP resource "schema://glossary/design-patterns"
    Then the MCP resource result should contain "Event Envelope"
    And the MCP resource result should contain "Three-Phase Rename"
    And the MCP resource result should contain "EventEnvelope"
    And the MCP resource result should contain "UserCreated"
    And the MCP resource result should contain "com.company.common.Address"

  Scenario: Read mcp-configuration glossary
    When I read MCP resource "schema://glossary/mcp-configuration"
    Then the MCP resource result should contain "read_only"
    And the MCP resource result should contain "tool_policy"
    And the MCP resource result should contain "Permission Scopes"
    And the MCP resource result should contain "Two-Phase Confirmations"
    And the MCP resource result should contain "SCHEMA_REGISTRY_MCP_"

  Scenario: Read migration glossary
    When I read MCP resource "schema://glossary/migration"
    Then the MCP resource result should contain "Confluent"
    And the MCP resource result should contain "IMPORT"
    And the MCP resource result should contain "ID Preservation"
    And the MCP resource result should contain "Rollback"

  # ==========================================================================
  # NEW GLOSSARY RESOURCES
  # ==========================================================================

  Scenario: Read error-reference glossary
    When I read MCP resource "schema://glossary/error-reference"
    Then the MCP resource result should contain "42201"
    And the MCP resource result should contain "40401"
    And the MCP resource result should contain "409"
    And the MCP resource result should contain "Decision Tree"
    And the MCP resource result should contain "explain_compatibility_failure"

  Scenario: Read auth-and-security glossary
    When I read MCP resource "schema://glossary/auth-and-security"
    Then the MCP resource result should contain "super_admin"
    And the MCP resource result should contain "readonly"
    And the MCP resource result should contain "deny-by-default"
    And the MCP resource result should contain "Rate Limiting"
    And the MCP resource result should contain "Audit Logging"

  Scenario: Read storage-backends glossary
    When I read MCP resource "schema://glossary/storage-backends"
    Then the MCP resource result should contain "PostgreSQL"
    And the MCP resource result should contain "MySQL"
    And the MCP resource result should contain "Cassandra"
    And the MCP resource result should contain "Stateless"
    And the MCP resource result should contain "FOR UPDATE"

  Scenario: Read normalization-and-fingerprinting glossary
    When I read MCP resource "schema://glossary/normalization-and-fingerprinting"
    Then the MCP resource result should contain "SHA-256"
    And the MCP resource result should contain "Canonical"
    And the MCP resource result should contain "Deduplication"
    And the MCP resource result should contain "normalize"
    And the MCP resource result should contain "metadata"

  Scenario: Read tool-selection-guide glossary
    When I read MCP resource "schema://glossary/tool-selection-guide"
    Then the MCP resource result should contain "Finding Schemas"
    And the MCP resource result should contain "list_subjects"
    And the MCP resource result should contain "register_schema"
    And the MCP resource result should contain "Working with Contexts"
    And the MCP resource result should contain "Encryption"
