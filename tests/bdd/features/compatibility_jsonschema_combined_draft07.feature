@functional
Feature: JSON Schema Combined Compatibility Draft-07 — Data-Driven (Confluent v8.1.1)
  Data-driven JSON Schema compatibility tests from the Confluent Schema Registry v8.1.1
  test suite (28 test cases).

  Scenario: JSON Schema combined Draft-07 001 — Detect compatible change to combined schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-001" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"enum": ["one", "two", "three"]}, {"type": "string"}, {"maxLength": 5}]}}}
      """
    When I set the config for subject "jsc07-001" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-001":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"maxLength": 5}, {"enum": ["one", "two", "three"]}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 002 — Detect incompatible change to combined schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-002" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"enum": ["one", "two", "three"]}, {"type": "string"}, {"maxLength": 5}]}}}
      """
    When I set the config for subject "jsc07-002" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-002":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"maxLength": 5}, {"enum": ["one", "two", "three"]}, {"type": "number"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-07 003 — Detect combined schema with duplicates in original
    Given the global compatibility level is "NONE"
    And subject "jsc07-003" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-003" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-003":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 004 — Detect combined schema with duplicates in update
    Given the global compatibility level is "NONE"
    And subject "jsc07-004" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-004" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-004":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 005 — Detect compatible change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-005" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-005" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-005":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 006 — Detect compatible change to oneOf schema with more types
    Given the global compatibility level is "NONE"
    And subject "jsc07-006" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-006" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-006":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "boolean"}, {"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 007 — Detect compatible change to oneOf schema with more properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-007" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string", "maxLength": 5}]}}}
      """
    When I set the config for subject "jsc07-007" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-007":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "boolean"}, {"type": "number"}, {"title": "my string", "type": "string", "maxLength": 8}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 008 — Detect incompatible change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-008" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-008" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-008":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-07 009 — Detect compatible change to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-009" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"enum": ["one", "two", "three"]}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-009" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-009":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 010 — Detect incompatible change to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-010" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-010" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-010":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"enum": ["one", "two", "three"]}, {"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-07 011 — Detect compatible change from oneOf to anyOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-011" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-011" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-011":
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 012 — Detect compatible change from oneOf to anyOf schema with more properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-012" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-012" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-012":
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 013 — Detect compatible change from allOf to anyOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-013" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-013" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-013":
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 014 — Detect incompatible change from oneOf to anyOf schema with fewer properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-014" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-014" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-014":
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-07 015 — Detect compatible change from allOf to anyOf schema with more properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-015" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-015" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-015":
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 016 — Detect compatible change from allOf to anyOf schema with fewer properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-016" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-016" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-016":
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 017 — Detect compatible change from anyOf to oneOf schema with more properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-017" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-017" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-017":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 018 — Detect incompatible change from anyOf to oneOf schema with fewer properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-018" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-018" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-018":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-07 019 — Detect compatible change from allOf to oneOf schema with more properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-019" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "number"}]}}}
      """
    When I set the config for subject "jsc07-019" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-019":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 020 — Detect compatible change from allOf to oneOf schema with fewer properties
    Given the global compatibility level is "NONE"
    And subject "jsc07-020" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-020" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-020":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 021 — Detect compatible change from anyOf to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-021" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-021" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-021":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 022 — Detect compatible change from oneOf to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-022" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-022" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-022":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 023 — Detect incompatible change from anyOf to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-023" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-023" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-023":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-07 024 — Detect incompatible change from oneOf to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-024" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "string"}]}}}
      """
    When I set the config for subject "jsc07-024" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-024":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema combined Draft-07 025 — Detect compatible change from non-combined to anyOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-025" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"type": "string"}}}
      """
    When I set the config for subject "jsc07-025" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-025":
      """
      {"type": "object", "properties": {"prop1": {"anyOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 026 — Detect compatible change from non-combined to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-026" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"type": "string"}}}
      """
    When I set the config for subject "jsc07-026" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-026":
      """
      {"type": "object", "properties": {"prop1": {"oneOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 027 — Detect compatible change from non-combined to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-027" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"type": "string"}}}
      """
    When I set the config for subject "jsc07-027" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-027":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "string"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema combined Draft-07 028 — Detect incompatible change from non-combined to allOf schema
    Given the global compatibility level is "NONE"
    And subject "jsc07-028" has "JSON" schema:
      """
      {"type": "object", "properties": {"prop1": {"type": "string"}}}
      """
    When I set the config for subject "jsc07-028" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsc07-028":
      """
      {"type": "object", "properties": {"prop1": {"allOf": [{"type": "number"}, {"type": "string"}]}}}
      """
    Then the compatibility check should be incompatible
