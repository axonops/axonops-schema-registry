@functional
Feature: JSON Schema Compatibility Diff Draft 2020-12 — Data-Driven (Confluent v8.1.1)
  Data-driven JSON Schema compatibility tests from the Confluent Schema Registry v8.1.1
  test suite (101 test cases).

  Scenario: JSON Schema diff Draft-2020 001 — Anything can change to empty schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-001" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {}}
      """
    When I set the config for subject "jsd20-001" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-001":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 002 — Detect changes to id
    Given the global compatibility level is "NONE"
    And subject "jsd20-002" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "$id": "something"}
      """
    When I set the config for subject "jsd20-002" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-002":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "$id": "something_else"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 003 — Detect changes to title
    Given the global compatibility level is "NONE"
    And subject "jsd20-003" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "title": "something"}
      """
    When I set the config for subject "jsd20-003" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-003":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "title": "something_else"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 004 — Detect changes to description
    Given the global compatibility level is "NONE"
    And subject "jsd20-004" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "description": "something"}
      """
    When I set the config for subject "jsd20-004" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-004":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "description": "something_else"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 005 — Detect changes to simple schema type
    Given the global compatibility level is "NONE"
    And subject "jsd20-005" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object"}
      """
    When I set the config for subject "jsd20-005" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-005":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array"}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 006 — Detect increased minLength string schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-006" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "minLength": 10}
      """
    When I set the config for subject "jsd20-006" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-006":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "minLength": 11}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 007 — Detect decreased minLength string schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-007" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "minLength": 11}
      """
    When I set the config for subject "jsd20-007" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-007":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "minLength": 10}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 008 — Detect increased maxLength string schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-008" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "maxLength": 10}
      """
    When I set the config for subject "jsd20-008" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-008":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "maxLength": 12}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 009 — Detect decreased maxLength string schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-009" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "maxLength": 12}
      """
    When I set the config for subject "jsd20-009" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-009":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "maxLength": 10}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 010 — Detect removed pattern from string schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-010" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "pattern": "uuid"}
      """
    When I set the config for subject "jsd20-010" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-010":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 011 — Detect changes to pattern string schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-011" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "pattern": "date-time"}
      """
    When I set the config for subject "jsd20-011" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-011":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "pattern": "uuid"}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 012 — Detect changes to pattern
    Given the global compatibility level is "NONE"
    And subject "jsd20-012" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "pattern": "date-time"}
      """
    When I set the config for subject "jsd20-012" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-012":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "pattern": "date-time"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 013 — Detect removed maximum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-013" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "maximum": 10}
      """
    When I set the config for subject "jsd20-013" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-013":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 014 — Detect increased maximum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-014" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "maximum": 10}
      """
    When I set the config for subject "jsd20-014" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-014":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "maximum": 11}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 015 — Detect decreased maximum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-015" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "maximum": 11}
      """
    When I set the config for subject "jsd20-015" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-015":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "maximum": 10}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 016 — Detect removed minimum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-016" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "minimum": 10}
      """
    When I set the config for subject "jsd20-016" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-016":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 017 — Detect increased minimum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-017" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "minimum": 10}
      """
    When I set the config for subject "jsd20-017" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-017":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "minimum": 11}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 018 — Detect decreased minimum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-018" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "minimum": 11}
      """
    When I set the config for subject "jsd20-018" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-018":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "minimum": 10}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 019 — Detect removed multipleOf number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-019" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "multipleOf": 10}
      """
    When I set the config for subject "jsd20-019" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-019":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 020 — Detect reduced multipleOf number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-020" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "multipleOf": 10}
      """
    When I set the config for subject "jsd20-020" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-020":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "multipleOf": 2}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 021 — Detect changes to multipleOf number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-021" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "multipleOf": 10}
      """
    When I set the config for subject "jsd20-021" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-021":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number", "multipleOf": 11}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 022 — Detect narrowed change to number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-022" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number"}
      """
    When I set the config for subject "jsd20-022" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-022":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "integer"}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 023 — Detect extended change to number schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-023" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "integer"}
      """
    When I set the config for subject "jsd20-023" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-023":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "number"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 024 — Detect compatible change to not schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-024" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "not": {"type": "number"}}
      """
    When I set the config for subject "jsd20-024" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-024":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "not": {"type": "integer"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 025 — Detect incompatible change to not schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-025" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "not": {"type": "integer"}}
      """
    When I set the config for subject "jsd20-025" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-025":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "not": {"type": "number"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 026 — Detect incompatible changes to enum schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-026" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "allOf": [{"type": "string"}, {"enum": ["red"]}]}
      """
    When I set the config for subject "jsd20-026" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-026":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "allOf": [{"type": "string"}, {"enum": ["blue"]}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 027 — Detect compatible changes to enum schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-027" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "allOf": [{"type": "string"}, {"enum": ["red"]}]}
      """
    When I set the config for subject "jsd20-027" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-027":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "allOf": [{"enum": ["red", "blue"]}, {"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 028 — Detect changes to type array
    Given the global compatibility level is "NONE"
    And subject "jsd20-028" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": ["object"]}
      """
    When I set the config for subject "jsd20-028" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-028":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": ["object", "array"]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 029 — Detect changes to sub schemas
    Given the global compatibility level is "NONE"
    And subject "jsd20-029" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "string"}]}
      """
    When I set the config for subject "jsd20-029" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-029":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "number"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 030 — Detect changes to number of sub schemas
    Given the global compatibility level is "NONE"
    And subject "jsd20-030" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "string"}]}
      """
    When I set the config for subject "jsd20-030" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-030":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "number"}, {"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 031 — Detect changes to remove properties
    Given the global compatibility level is "NONE"
    And subject "jsd20-031" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-031" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-031":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"bar": {"type": "string"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 032 — Detect change type of properties
    Given the global compatibility level is "NONE"
    And subject "jsd20-032" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-032" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-032":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "number"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 033 — Detect optional property added to closed content model
    Given the global compatibility level is "NONE"
    And subject "jsd20-033" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd20-033" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-033":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"type": "number"}}, "additionalProperties": false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 034 — Detect property added to partially open content model 1
    Given the global compatibility level is "NONE"
    And subject "jsd20-034" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": false, "patternProperties": {"^S_": {"type": "string"}, "^I_": {"type": "integer"}}}
      """
    When I set the config for subject "jsd20-034" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-034":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "I_123": {"type": "number"}}, "additionalProperties": false, "patternProperties": {"^S_": {"type": "string"}, "^I_": {"type": "integer"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 035 — Detect property added to partially open content model 2
    Given the global compatibility level is "NONE"
    And subject "jsd20-035" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    When I set the config for subject "jsd20-035" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-035":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 036 — Detect property added to partially open content model 3
    Given the global compatibility level is "NONE"
    And subject "jsd20-036" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    When I set the config for subject "jsd20-036" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-036":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"type": "number"}}, "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 037 — Detect property added to partially open content model 4
    Given the global compatibility level is "NONE"
    And subject "jsd20-037" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    When I set the config for subject "jsd20-037" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-037":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "required": ["bar"], "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 038 — Detect property removed from partially open content model 1
    Given the global compatibility level is "NONE"
    And subject "jsd20-038" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "I_123": {"type": "integer"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd20-038" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-038":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": false, "patternProperties": {"^S_": {"type": "string"}, "^I_": {"type": "number"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 039 — Detect property removed from partially open content model 2
    Given the global compatibility level is "NONE"
    And subject "jsd20-039" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd20-039" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-039":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 040 — Detect property removed from partially open content model 3
    Given the global compatibility level is "NONE"
    And subject "jsd20-040" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"type": "number"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd20-040" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-040":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 041 — Detect property removed from partially open content model 4
    Given the global compatibility level is "NONE"
    And subject "jsd20-041" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"oneOf": [{"type": "null"}, {"type": "string"}]}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd20-041" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-041":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": {"oneOf": [{"type": "null"}, {"type": "string"}]}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 042 — Detect property removed from partially open content model 5
    Given the global compatibility level is "NONE"
    And subject "jsd20-042" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd20-042" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-042":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "additionalProperties": {"oneOf": [{"type": "null"}, {"type": "string"}]}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 043 — Detect property added to open content model
    Given the global compatibility level is "NONE"
    And subject "jsd20-043" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-043" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-043":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}, "bar": {"type": "number"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 044 — Detect changes to composed schema type
    Given the global compatibility level is "NONE"
    And subject "jsd20-044" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"properties": {"foo": {"type": "string"}}}]}
      """
    When I set the config for subject "jsd20-044" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-044":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"properties": {"foo": {"type": "number"}}}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 045 — Detect changes to validation criteria in composed schema type for singleton
    Given the global compatibility level is "NONE"
    And subject "jsd20-045" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "string"}]}
      """
    When I set the config for subject "jsd20-045" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-045":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "allOf": [{"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 046 — Detect changes to validation criteria in composed schema type
    Given the global compatibility level is "NONE"
    And subject "jsd20-046" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "string"}, {"type": "integer"}]}
      """
    When I set the config for subject "jsd20-046" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-046":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "allOf": [{"type": "string"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 047 — Detect changes to dependencies as array
    Given the global compatibility level is "NONE"
    And subject "jsd20-047" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "number"}, "bar": {"type": "string"}}, "dependentRequired": {"foo": ["bar"]}}
      """
    When I set the config for subject "jsd20-047" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-047":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "number"}, "bar": {"type": "string"}}, "dependentRequired": {"bar": ["foo"]}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 048 — Detect compatible changes to dependencies schemas
    Given the global compatibility level is "NONE"
    And subject "jsd20-048" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "dependentSchemas": {"foo": {"type": "string"}, "bar": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-048" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-048":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "dependentSchemas": {"foo": {"type": "string"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 049 — Detect compatible changes to dependencies schemas
    Given the global compatibility level is "NONE"
    And subject "jsd20-049" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "dependentSchemas": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-049" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-049":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "dependentSchemas": {}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 050 — Detect incompatible changes to dependencies schemas
    Given the global compatibility level is "NONE"
    And subject "jsd20-050" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "dependentSchemas": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-050" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-050":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "dependentSchemas": {"foo": {"type": "number"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 051 — Detect changes to dependencies properties
    Given the global compatibility level is "NONE"
    And subject "jsd20-051" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "dependentSchemas": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-051" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-051":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "dependentSchemas": {"bar": {"type": "string"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 052 — Detect required array to be removed
    Given the global compatibility level is "NONE"
    And subject "jsd20-052" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "required": ["foo"]}
      """
    When I set the config for subject "jsd20-052" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-052":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 053 — Detect required array to be added
    Given the global compatibility level is "NONE"
    And subject "jsd20-053" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-053" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-053":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "required": ["foo"]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 054 — Detect required array to be changed
    Given the global compatibility level is "NONE"
    And subject "jsd20-054" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "required": ["foo"]}
      """
    When I set the config for subject "jsd20-054" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-054":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {"foo": {"type": "string"}}, "required": ["bar"]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 055 — Detect removed maxProperties
    Given the global compatibility level is "NONE"
    And subject "jsd20-055" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {}, "maxProperties": 1}
      """
    When I set the config for subject "jsd20-055" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-055":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 056 — Detect increased maxProperties
    Given the global compatibility level is "NONE"
    And subject "jsd20-056" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "maxProperties": 1}
      """
    When I set the config for subject "jsd20-056" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-056":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "maxProperties": 2}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 057 — Detect decreased maxProperties
    Given the global compatibility level is "NONE"
    And subject "jsd20-057" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "maxProperties": 2}
      """
    When I set the config for subject "jsd20-057" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-057":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "maxProperties": 1}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 058 — Detect removed minProperties
    Given the global compatibility level is "NONE"
    And subject "jsd20-058" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {}, "minProperties": 2}
      """
    When I set the config for subject "jsd20-058" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-058":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 059 — Detect increased minProperties
    Given the global compatibility level is "NONE"
    And subject "jsd20-059" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "minProperties": 1}
      """
    When I set the config for subject "jsd20-059" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-059":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "minProperties": 2}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 060 — Detect decreased minProperties
    Given the global compatibility level is "NONE"
    And subject "jsd20-060" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "minProperties": 2}
      """
    When I set the config for subject "jsd20-060" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-060":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "minProperties": 1}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 061 — Detect changes to all items schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-061" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array"}
      """
    When I set the config for subject "jsd20-061" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-061":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 062 — Detect changes to all items schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-062" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": {"type": "string"}}
      """
    When I set the config for subject "jsd20-062" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-062":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 063 — Detect changes to items schema list
    Given the global compatibility level is "NONE"
    And subject "jsd20-063" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}]}
      """
    When I set the config for subject "jsd20-063" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-063":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 064 — Detect item added to partially open content model 1
    Given the global compatibility level is "NONE"
    And subject "jsd20-064" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": {"type": "string"}}
      """
    When I set the config for subject "jsd20-064" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-064":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}, {"type": "string"}], "items": {"type": "string"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 065 — Detect item added to partially open content model 2
    Given the global compatibility level is "NONE"
    And subject "jsd20-065" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": {"type": "string"}}
      """
    When I set the config for subject "jsd20-065" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-065":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}, {"type": "number"}], "items": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 066 — Detect item removed from partially open content model 1
    Given the global compatibility level is "NONE"
    And subject "jsd20-066" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}, {"type": "string"}], "items": {"type": "string"}}
      """
    When I set the config for subject "jsd20-066" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-066":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": {"type": "string"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 067 — Detect item removed from partially open content model 2
    Given the global compatibility level is "NONE"
    And subject "jsd20-067" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}, {"oneOf": [{"type": "null"}, {"type": "string"}]}], "items": {"oneOf": [{"type": "null"}, {"type": "string"}]}}
      """
    When I set the config for subject "jsd20-067" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-067":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": {"oneOf": [{"type": "null"}, {"type": "string"}]}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 068 — Detect item removed from partially open content model 3
    Given the global compatibility level is "NONE"
    And subject "jsd20-068" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}, {"type": "number"}], "items": {"type": "string"}}
      """
    When I set the config for subject "jsd20-068" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-068":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 069 — Detect changes to items in the schema list
    Given the global compatibility level is "NONE"
    And subject "jsd20-069" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}]}
      """
    When I set the config for subject "jsd20-069" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-069":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "number"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 070 — Detect decreased maxItems
    Given the global compatibility level is "NONE"
    And subject "jsd20-070" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "maxItems": 2}
      """
    When I set the config for subject "jsd20-070" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-070":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "maxItems": 1}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 071 — Detect removed minItems
    Given the global compatibility level is "NONE"
    And subject "jsd20-071" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "minItems": 1}
      """
    When I set the config for subject "jsd20-071" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-071":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 072 — Detect increased minItems
    Given the global compatibility level is "NONE"
    And subject "jsd20-072" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "minItems": 1}
      """
    When I set the config for subject "jsd20-072" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-072":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "minItems": 2}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 073 — Detect decreased minItems
    Given the global compatibility level is "NONE"
    And subject "jsd20-073" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "minItems": 2}
      """
    When I set the config for subject "jsd20-073" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-073":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "minItems": 1}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 074 — Detect unique items removed
    Given the global compatibility level is "NONE"
    And subject "jsd20-074" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "uniqueItems": true}
      """
    When I set the config for subject "jsd20-074" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-074":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 075 — Detect incompatible changes to reference schemas
    Given the global compatibility level is "NONE"
    And subject "jsd20-075" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": {"$ref": "#/definitions/someRef"}, "definitions": {"someRef": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-075" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-075":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": {"$ref": "#/definitions/someRef"}, "definitions": {"someRef": {"type": "number"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 076 — Detect compatible change of moving to reference schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-076" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string"}
      """
    When I set the config for subject "jsd20-076" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-076":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "$ref": "#/definitions/someRef", "definitions": {"someRef": {"type": "string"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 077 — Detect compatible change of moving from reference schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-077" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "$ref": "#/definitions/someRef", "definitions": {"someRef": {"type": "string"}}}
      """
    When I set the config for subject "jsd20-077" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-077":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 078 — Detect added boolean additional properties
    Given the global compatibility level is "NONE"
    And subject "jsd20-078" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {}, "additionalProperties": false}
      """
    When I set the config for subject "jsd20-078" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-078":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "properties": {}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 079 — Detect removed boolean additional properties
    Given the global compatibility level is "NONE"
    And subject "jsd20-079" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": true}
      """
    When I set the config for subject "jsd20-079" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-079":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": false}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 080 — Detect narrowing changes to additional properties schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-080" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": true}
      """
    When I set the config for subject "jsd20-080" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-080":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": {"type": "number"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 081 — Detect adding empty schema to open content model
    Given the global compatibility level is "NONE"
    And subject "jsd20-081" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": true}
      """
    When I set the config for subject "jsd20-081" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-081":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": true, "properties": {"foo": true}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 082 — Detect removing empty schema from open content model
    Given the global compatibility level is "NONE"
    And subject "jsd20-082" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": true, "properties": {"foo": true}}
      """
    When I set the config for subject "jsd20-082" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-082":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 083 — Detect adding false to closed content model
    Given the global compatibility level is "NONE"
    And subject "jsd20-083" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": false}
      """
    When I set the config for subject "jsd20-083" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-083":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": false, "properties": {"foo": false}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 084 — Detect removing false from closed content model
    Given the global compatibility level is "NONE"
    And subject "jsd20-084" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": false, "properties": {"foo": false}}
      """
    When I set the config for subject "jsd20-084" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-084":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 085 — Detect changes to additional properties schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-085" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": {"type": "string"}}
      """
    When I set the config for subject "jsd20-085" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-085":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "additionalProperties": {"type": "number"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 086 — Detect removed boolean additional items
    Given the global compatibility level is "NONE"
    And subject "jsd20-086" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "items": true}
      """
    When I set the config for subject "jsd20-086" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-086":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "items": false}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 087 — Detect changes to additional items schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-087" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "items": {"type": "string"}}
      """
    When I set the config for subject "jsd20-087" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-087":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "items": {"type": "number"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-2020 088 — Detect adding empty schema to open content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-088" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": true, "prefixItems": [{"type": "number"}]}
      """
    When I set the config for subject "jsd20-088" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-088":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": true, "prefixItems": [{"type": "number"}, true]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 089 — Detect removing empty schema from open content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-089" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": true, "prefixItems": [{"type": "number"}, true]}
      """
    When I set the config for subject "jsd20-089" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-089":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": true, "prefixItems": [{"type": "number"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 090 — Detect adding false to closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-090" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": false, "prefixItems": [{"type": "number"}]}
      """
    When I set the config for subject "jsd20-090" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-090":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": false, "prefixItems": [{"type": "number"}, false]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 091 — Detect removing false from closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-091" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": false, "prefixItems": [{"type": "number"}, false]}
      """
    When I set the config for subject "jsd20-091" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-091":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "items": false, "prefixItems": [{"type": "number"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 092 — Detect adding to closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-092" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": false}
      """
    When I set the config for subject "jsd20-092" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-092":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}, {"type": "boolean"}], "items": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 093 — Detect removing from closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-093" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}, {"type": "boolean"}], "items": false}
      """
    When I set the config for subject "jsd20-093" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-093":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 094 — Detect removing from open content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-094" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": true}
      """
    When I set the config for subject "jsd20-094" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-094":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}], "items": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 095 — Detect adding to closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-095" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}], "items": false}
      """
    When I set the config for subject "jsd20-095" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-095":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 096 — Detect removing number from closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-096" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": false}
      """
    When I set the config for subject "jsd20-096" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-096":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}], "items": {"type": "number"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 097 — Detect adding number to partially open content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd20-097" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}], "items": {"type": "number"}}
      """
    When I set the config for subject "jsd20-097" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-097":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": [{"type": "string"}, {"type": "number"}], "items": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 098 — Detect change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-098" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string"}
      """
    When I set the config for subject "jsd20-098" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-098":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "number"}, {"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 099 — Detect change to optional oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-099" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "null"}
      """
    When I set the config for subject "jsd20-099" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-099":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "number"}, {"type": "null"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 100 — Detect change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-100" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "number"}]}
      """
    When I set the config for subject "jsd20-100" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-100":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "string"}, {"type": "number"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-2020 101 — Detect change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsd20-101" has "JSON" schema:
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "string", "maxLength": 3}
      """
    When I set the config for subject "jsd20-101" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd20-101":
      """
      {"$schema": "https://json-schema.org/draft/2020-12/schema", "oneOf": [{"type": "number"}, {"type": "string"}]}
      """
    Then the compatibility check should be compatible
