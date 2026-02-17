@functional
Feature: JSON Schema Combined Compatibility Draft 2020-12 — Data-Driven (Confluent v8.1.1)
  Data-driven JSON Schema compatibility tests from the Confluent Schema Registry v8.1.1
  test suite (18 test cases).

  Scenario: JSON Schema combined Draft-2020 001 — Detect compatible change to combined schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-001" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"enum": ["one", "two", "three"]}, {"type": "string"}, {"maxLength": 5}]}}}
      """
    When I set the config for subject "jsc20-001" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-001":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"maxLength": 5}, {"enum": ["one", "two", "three"]}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 002 — Detect incompatible change to combined schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-002" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"enum": ["one", "two", "three"]}, {"type": "string"}, {"maxLength": 5}]}}}
      """
    When I set the config for subject "jsc20-002" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-002":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"maxLength": 5}, {"enum": ["one", "two", "three"]}, {"type": "number"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-2020 003 — Detect combined schema with duplicates in original
    Given the global compatibility level is "NONE"
    And subject "jsc20-003" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-003" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-003":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 004 — Detect combined schema with duplicates in update
    Given the global compatibility level is "NONE"
    And subject "jsc20-004" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-004" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-004":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 005 — Detect compatible change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-005" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-005" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-005":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 006 — Detect compatible change to oneOf schema with more types
    Given the global compatibility level is "NONE"
    And subject "jsc20-006" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-006" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-006":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "boolean"}, {"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 007 — Detect compatible change to oneOf schema with more properties
    Given the global compatibility level is "NONE"
    And subject "jsc20-007" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "string", "maxLength": 5}]}}}
      """
    When I set the config for subject "jsc20-007" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-007":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "boolean"}, {"type": "number"}, {"title": "my string", "type": "string", "maxLength": 8}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 008 — Detect incompatible change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-008" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-008" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-008":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-2020 009 — Detect compatible change to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-009" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"enum": ["one", "two", "three"]}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-009" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-009":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 010 — Detect incompatible change to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-010" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-010" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-010":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"enum": ["one", "two", "three"]}, {"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-2020 011 — Detect compatible change from oneOf to anyOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-011" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-011" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-011":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 012 — Detect compatible change from oneOf to anyOf schema with more properties
    Given the global compatibility level is "NONE"
    And subject "jsc20-012" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-012" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-012":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"anyOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 013 — Detect compatible change from allOf to anyOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-013" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-013" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-013":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 014 — Detect compatible change from allOf to anyOf schema with more properties
    Given the global compatibility level is "NONE"
    And subject "jsc20-014" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc20-014" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-014":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"anyOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 015 — Detect compatible change from non-combined to anyOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-015" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"type": "string"}}}
      """
    When I set the config for subject "jsc20-015" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-015":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"anyOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 016 — Detect compatible change from non-combined to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-016" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"type": "string"}}}
      """
    When I set the config for subject "jsc20-016" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-016":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 017 — Detect compatible change from non-combined to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-017" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"type": "string"}}}
      """
    When I set the config for subject "jsc20-017" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-017":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-2020 018 — Detect incompatible change from non-combined to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc20-018" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"type": "string"}}}
      """
    When I set the config for subject "jsc20-018" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc20-018":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "properties": {"prop1": {"allOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible
