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
    And the MCP resource result should contain "Common Mistakes"

  Scenario: Read migration glossary
    When I read MCP resource "schema://glossary/migration"
    Then the MCP resource result should contain "Confluent"
    And the MCP resource result should contain "IMPORT"
    And the MCP resource result should contain "ID Preservation"
    And the MCP resource result should contain "Rollback"
