@functional
Feature: JSON Schema Compatibility Diff Draft-07 — Data-Driven (Confluent v8.1.1)
  Data-driven JSON Schema compatibility tests from the Confluent Schema Registry v8.1.1
  diff-schema-examples.json test suite (104 Draft-07 test cases).

  Scenario: JSON Schema diff Draft-07 001 — Anything can change to empty schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-001" has "JSON" schema:
      """
      {"properties": {}}
      """
    When I set the config for subject "jsd07-001" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-001":
      """
      {}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 002 — Detect changes to id
    Given the global compatibility level is "NONE"
    And subject "jsd07-002" has "JSON" schema:
      """
      {"$id": "something"}
      """
    When I set the config for subject "jsd07-002" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-002":
      """
      {"$id": "something_else"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 003 — Detect changes to title
    Given the global compatibility level is "NONE"
    And subject "jsd07-003" has "JSON" schema:
      """
      {"title": "something"}
      """
    When I set the config for subject "jsd07-003" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-003":
      """
      {"title": "something_else"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 004 — Detect changes to description
    Given the global compatibility level is "NONE"
    And subject "jsd07-004" has "JSON" schema:
      """
      {"description": "something"}
      """
    When I set the config for subject "jsd07-004" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-004":
      """
      {"description": "something_else"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 005 — Detect changes to simple schema type
    Given the global compatibility level is "NONE"
    And subject "jsd07-005" has "JSON" schema:
      """
      {"type": "object"}
      """
    When I set the config for subject "jsd07-005" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-005":
      """
      {"type": "array"}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 006 — Detect increased minLength string schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-006" has "JSON" schema:
      """
      {"type": "string", "minLength": 10}
      """
    When I set the config for subject "jsd07-006" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-006":
      """
      {"type": "string", "minLength": 11}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 007 — Detect decreased minLength string schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-007" has "JSON" schema:
      """
      {"type": "string", "minLength": 11}
      """
    When I set the config for subject "jsd07-007" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-007":
      """
      {"type": "string", "minLength": 10}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 008 — Detect increased maxLength string schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-008" has "JSON" schema:
      """
      {"type": "string", "maxLength": 10}
      """
    When I set the config for subject "jsd07-008" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-008":
      """
      {"type": "string", "maxLength": 12}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 009 — Detect decreased maxLength string schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-009" has "JSON" schema:
      """
      {"type": "string", "maxLength": 12}
      """
    When I set the config for subject "jsd07-009" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-009":
      """
      {"type": "string", "maxLength": 10}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 010 — Detect removed pattern from string schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-010" has "JSON" schema:
      """
      {"type": "string", "pattern": "uuid"}
      """
    When I set the config for subject "jsd07-010" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-010":
      """
      {"type": "string"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 011 — Detect changes to pattern string schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-011" has "JSON" schema:
      """
      {"type": "string", "pattern": "date-time"}
      """
    When I set the config for subject "jsd07-011" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-011":
      """
      {"type": "string", "pattern": "uuid"}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 012 — Detect changes to pattern
    Given the global compatibility level is "NONE"
    And subject "jsd07-012" has "JSON" schema:
      """
      {"type": "string", "pattern": "date-time"}
      """
    When I set the config for subject "jsd07-012" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-012":
      """
      {"type": "string", "pattern": "date-time"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 013 — Detect removed maximum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-013" has "JSON" schema:
      """
      {"type": "number", "maximum": 10}
      """
    When I set the config for subject "jsd07-013" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-013":
      """
      {"type": "number"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 014 — Detect increased maximum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-014" has "JSON" schema:
      """
      {"type": "number", "maximum": 10}
      """
    When I set the config for subject "jsd07-014" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-014":
      """
      {"type": "number", "maximum": 11}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 015 — Detect decreased maximum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-015" has "JSON" schema:
      """
      {"type": "number", "maximum": 11}
      """
    When I set the config for subject "jsd07-015" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-015":
      """
      {"type": "number", "maximum": 10}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 016 — Detect removed minimum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-016" has "JSON" schema:
      """
      {"type": "number", "minimum": 10}
      """
    When I set the config for subject "jsd07-016" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-016":
      """
      {"type": "number"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 017 — Detect increased minimum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-017" has "JSON" schema:
      """
      {"type": "number", "minimum": 10}
      """
    When I set the config for subject "jsd07-017" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-017":
      """
      {"type": "number", "minimum": 11}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 018 — Detect decreased minimum number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-018" has "JSON" schema:
      """
      {"type": "number", "minimum": 11}
      """
    When I set the config for subject "jsd07-018" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-018":
      """
      {"type": "number", "minimum": 10}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 019 — Detect removed multipleOf number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-019" has "JSON" schema:
      """
      {"type": "number", "multipleOf": 10}
      """
    When I set the config for subject "jsd07-019" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-019":
      """
      {"type": "number"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 020 — Detect reduced multipleOf number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-020" has "JSON" schema:
      """
      {"type": "number", "multipleOf": 10}
      """
    When I set the config for subject "jsd07-020" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-020":
      """
      {"type": "number", "multipleOf": 2}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 021 — Detect changes to multipleOf number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-021" has "JSON" schema:
      """
      {"type": "number", "multipleOf": 10}
      """
    When I set the config for subject "jsd07-021" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-021":
      """
      {"type": "number", "multipleOf": 11}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 022 — Detect narrowed change to number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-022" has "JSON" schema:
      """
      {"type": "number"}
      """
    When I set the config for subject "jsd07-022" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-022":
      """
      {"type": "integer"}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 023 — Detect extended change to number schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-023" has "JSON" schema:
      """
      {"type": "integer"}
      """
    When I set the config for subject "jsd07-023" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-023":
      """
      {"type": "number"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 024 — Detect compatible change to not schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-024" has "JSON" schema:
      """
      {"not": {"type": "number"}}
      """
    When I set the config for subject "jsd07-024" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-024":
      """
      {"not": {"type": "integer"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 025 — Detect incompatible change to not schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-025" has "JSON" schema:
      """
      {"not": {"type": "integer"}}
      """
    When I set the config for subject "jsd07-025" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-025":
      """
      {"not": {"type": "number"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 026 — Detect removed enum 
    Given the global compatibility level is "NONE"
    And subject "jsd07-026" has "JSON" schema:
      """
      {"type": "string", "enum": ["red"]}
      """
    When I set the config for subject "jsd07-026" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-026":
      """
      {"type": "string"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 027 — Detect incompatible changes to const schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-027" has "JSON" schema:
      """
      {"$schema": "http://json-schema.org/draft-07/schema#", "properties": {"foo": {"const": "red"}}}
      """
    When I set the config for subject "jsd07-027" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-027":
      """
      {"$schema": "http://json-schema.org/draft-07/schema#", "properties": {"foo": {"const": "blue"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 028 — Detect incompatible changes to enum schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-028" has "JSON" schema:
      """
      {"allOf": [{"type": "string"}, {"enum": ["red"]}]}
      """
    When I set the config for subject "jsd07-028" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-028":
      """
      {"allOf": [{"type": "string"}, {"enum": ["blue"]}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 029 — Detect compatible changes to enum schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-029" has "JSON" schema:
      """
      {"allOf": [{"type": "string"}, {"enum": ["red"]}]}
      """
    When I set the config for subject "jsd07-029" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-029":
      """
      {"allOf": [{"enum": ["red", "blue"]}, {"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 030 — Detect changes to type array
    Given the global compatibility level is "NONE"
    And subject "jsd07-030" has "JSON" schema:
      """
      {"type": ["object"]}
      """
    When I set the config for subject "jsd07-030" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-030":
      """
      {"type": ["object", "array"]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 031 — Detect changes to sub schemas
    Given the global compatibility level is "NONE"
    And subject "jsd07-031" has "JSON" schema:
      """
      {"oneOf": [{"type": "string"}]}
      """
    When I set the config for subject "jsd07-031" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-031":
      """
      {"oneOf": [{"type": "number"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 032 — Detect changes to number of sub schemas
    Given the global compatibility level is "NONE"
    And subject "jsd07-032" has "JSON" schema:
      """
      {"oneOf": [{"type": "string"}]}
      """
    When I set the config for subject "jsd07-032" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-032":
      """
      {"oneOf": [{"type": "number"}, {"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 033 — Detect changes to remove properties
    Given the global compatibility level is "NONE"
    And subject "jsd07-033" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-033" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-033":
      """
      {"properties": {"bar": {"type": "string"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 034 — Detect change type of properties
    Given the global compatibility level is "NONE"
    And subject "jsd07-034" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-034" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-034":
      """
      {"properties": {"foo": {"type": "number"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 035 — Detect optional property added to closed content model
    Given the global compatibility level is "NONE"
    And subject "jsd07-035" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd07-035" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-035":
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "number"}}, "additionalProperties": false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 036 — Detect property added to partially open content model 1
    Given the global compatibility level is "NONE"
    And subject "jsd07-036" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": false, "patternProperties": {"^S_": {"type": "string"}, "^I_": {"type": "integer"}}}
      """
    When I set the config for subject "jsd07-036" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-036":
      """
      {"properties": {"foo": {"type": "string"}, "I_123": {"type": "number"}}, "additionalProperties": false, "patternProperties": {"^S_": {"type": "string"}, "^I_": {"type": "integer"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 037 — Detect property added to partially open content model 2
    Given the global compatibility level is "NONE"
    And subject "jsd07-037" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    When I set the config for subject "jsd07-037" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-037":
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 038 — Detect property added to partially open content model 3
    Given the global compatibility level is "NONE"
    And subject "jsd07-038" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    When I set the config for subject "jsd07-038" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-038":
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "number"}}, "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 039 — Detect property added to partially open content model 4
    Given the global compatibility level is "NONE"
    And subject "jsd07-039" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    When I set the config for subject "jsd07-039" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-039":
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "required": ["bar"], "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 040 — Detect property removed from partially open content model 1
    Given the global compatibility level is "NONE"
    And subject "jsd07-040" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}, "I_123": {"type": "integer"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd07-040" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-040":
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": false, "patternProperties": {"^S_": {"type": "string"}, "^I_": {"type": "number"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 041 — Detect property removed from partially open content model 2
    Given the global compatibility level is "NONE"
    And subject "jsd07-041" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd07-041" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-041":
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 042 — Detect property removed from partially open content model 3
    Given the global compatibility level is "NONE"
    And subject "jsd07-042" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "number"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd07-042" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-042":
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 043 — Detect property removed from partially open content model 4
    Given the global compatibility level is "NONE"
    And subject "jsd07-043" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"oneOf": [{"type": "null"}, {"type": "string"}]}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd07-043" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-043":
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": {"oneOf": [{"type": "null"}, {"type": "string"}]}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 044 — Detect property removed from partially open content model 5
    Given the global compatibility level is "NONE"
    And subject "jsd07-044" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd07-044" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-044":
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": {"oneOf": [{"type": "null"}, {"type": "string"}]}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 045 — Detect property added to open content model
    Given the global compatibility level is "NONE"
    And subject "jsd07-045" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-045" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-045":
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "number"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 046 — Detect changes to composed schema type
    Given the global compatibility level is "NONE"
    And subject "jsd07-046" has "JSON" schema:
      """
      {"oneOf": [{"properties": {"foo": {"type": "string"}}}]}
      """
    When I set the config for subject "jsd07-046" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-046":
      """
      {"oneOf": [{"properties": {"foo": {"type": "number"}}}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 047 — Detect changes to validation criteria in composed schema type for singleton
    Given the global compatibility level is "NONE"
    And subject "jsd07-047" has "JSON" schema:
      """
      {"oneOf": [{"type": "string"}]}
      """
    When I set the config for subject "jsd07-047" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-047":
      """
      {"allOf": [{"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 048 — Detect changes to validation criteria in composed schema type
    Given the global compatibility level is "NONE"
    And subject "jsd07-048" has "JSON" schema:
      """
      {"oneOf": [{"type": "string"}, {"type": "integer"}]}
      """
    When I set the config for subject "jsd07-048" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-048":
      """
      {"allOf": [{"type": "string"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 049 — Detect changes to dependencies as array
    Given the global compatibility level is "NONE"
    And subject "jsd07-049" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "number"}, "bar": {"type": "string"}}, "dependencies": {"foo": ["bar"]}}
      """
    When I set the config for subject "jsd07-049" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-049":
      """
      {"properties": {"foo": {"type": "number"}, "bar": {"type": "string"}}, "dependencies": {"bar": ["foo"]}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 050 — Detect compatible changes to dependencies schemas
    Given the global compatibility level is "NONE"
    And subject "jsd07-050" has "JSON" schema:
      """
      {"dependencies": {"foo": {"type": "string"}, "bar": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-050" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-050":
      """
      {"dependencies": {"foo": {"type": "string"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 051 — Detect compatible changes to dependencies schemas
    Given the global compatibility level is "NONE"
    And subject "jsd07-051" has "JSON" schema:
      """
      {"dependencies": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-051" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-051":
      """
      {"dependencies": {}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 052 — Detect incompatible changes to dependencies schemas
    Given the global compatibility level is "NONE"
    And subject "jsd07-052" has "JSON" schema:
      """
      {"dependencies": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-052" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-052":
      """
      {"dependencies": {"foo": {"type": "number"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 053 — Detect changes to dependencies properties
    Given the global compatibility level is "NONE"
    And subject "jsd07-053" has "JSON" schema:
      """
      {"dependencies": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-053" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-053":
      """
      {"dependencies": {"bar": {"type": "string"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 054 — Detect required array to be removed
    Given the global compatibility level is "NONE"
    And subject "jsd07-054" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}, "required": ["foo"]}
      """
    When I set the config for subject "jsd07-054" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-054":
      """
      {"properties": {"foo": {"type": "string"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 055 — Detect required array to be added
    Given the global compatibility level is "NONE"
    And subject "jsd07-055" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-055" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-055":
      """
      {"properties": {"foo": {"type": "string"}}, "required": ["foo"]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 056 — Detect required array to be changed
    Given the global compatibility level is "NONE"
    And subject "jsd07-056" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}, "required": ["foo"]}
      """
    When I set the config for subject "jsd07-056" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-056":
      """
      {"properties": {"foo": {"type": "string"}}, "required": ["bar"]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 057 — Detect removed maxProperties
    Given the global compatibility level is "NONE"
    And subject "jsd07-057" has "JSON" schema:
      """
      {"properties": {}, "maxProperties": 1}
      """
    When I set the config for subject "jsd07-057" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-057":
      """
      {"properties": {}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 058 — Detect increased maxProperties
    Given the global compatibility level is "NONE"
    And subject "jsd07-058" has "JSON" schema:
      """
      {"maxProperties": 1}
      """
    When I set the config for subject "jsd07-058" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-058":
      """
      {"maxProperties": 2}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 059 — Detect decreased maxProperties
    Given the global compatibility level is "NONE"
    And subject "jsd07-059" has "JSON" schema:
      """
      {"maxProperties": 2}
      """
    When I set the config for subject "jsd07-059" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-059":
      """
      {"maxProperties": 1}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 060 — Detect removed minProperties
    Given the global compatibility level is "NONE"
    And subject "jsd07-060" has "JSON" schema:
      """
      {"properties": {}, "minProperties": 2}
      """
    When I set the config for subject "jsd07-060" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-060":
      """
      {"properties": {}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 061 — Detect increased minProperties
    Given the global compatibility level is "NONE"
    And subject "jsd07-061" has "JSON" schema:
      """
      {"minProperties": 1}
      """
    When I set the config for subject "jsd07-061" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-061":
      """
      {"minProperties": 2}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 062 — Detect decreased minProperties
    Given the global compatibility level is "NONE"
    And subject "jsd07-062" has "JSON" schema:
      """
      {"minProperties": 2}
      """
    When I set the config for subject "jsd07-062" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-062":
      """
      {"minProperties": 1}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 063 — Detect changes to all items schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-063" has "JSON" schema:
      """
      {"type": "array"}
      """
    When I set the config for subject "jsd07-063" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-063":
      """
      {"type": "array", "items": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 064 — Detect changes to all items schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-064" has "JSON" schema:
      """
      {"type": "array", "items": {"type": "string"}}
      """
    When I set the config for subject "jsd07-064" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-064":
      """
      {"type": "array"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 065 — Detect changes to items schema list
    Given the global compatibility level is "NONE"
    And subject "jsd07-065" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}]}
      """
    When I set the config for subject "jsd07-065" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-065":
      """
      {"type": "array"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 066 — Detect item added to partially open content model 1
    Given the global compatibility level is "NONE"
    And subject "jsd07-066" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": {"type": "string"}}
      """
    When I set the config for subject "jsd07-066" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-066":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}, {"type": "string"}], "additionalItems": {"type": "string"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 067 — Detect item added to partially open content model 2
    Given the global compatibility level is "NONE"
    And subject "jsd07-067" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": {"type": "string"}}
      """
    When I set the config for subject "jsd07-067" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-067":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}, {"type": "number"}], "additionalItems": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 068 — Detect item removed from partially open content model 1
    Given the global compatibility level is "NONE"
    And subject "jsd07-068" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}, {"type": "string"}], "additionalItems": {"type": "string"}}
      """
    When I set the config for subject "jsd07-068" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-068":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": {"type": "string"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 069 — Detect item removed from partially open content model 2
    Given the global compatibility level is "NONE"
    And subject "jsd07-069" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}, {"oneOf": [{"type": "null"}, {"type": "string"}]}], "additionalItems": {"oneOf": [{"type": "null"}, {"type": "string"}]}}
      """
    When I set the config for subject "jsd07-069" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-069":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": {"oneOf": [{"type": "null"}, {"type": "string"}]}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 070 — Detect item removed from partially open content model 3
    Given the global compatibility level is "NONE"
    And subject "jsd07-070" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}, {"type": "number"}], "additionalItems": {"type": "string"}}
      """
    When I set the config for subject "jsd07-070" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-070":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": {"type": "string"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 071 — Detect changes to items in the schema list
    Given the global compatibility level is "NONE"
    And subject "jsd07-071" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}]}
      """
    When I set the config for subject "jsd07-071" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-071":
      """
      {"type": "array", "items": [{"type": "number"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 072 — Detect decreased maxItems
    Given the global compatibility level is "NONE"
    And subject "jsd07-072" has "JSON" schema:
      """
      {"type": "array", "maxItems": 2}
      """
    When I set the config for subject "jsd07-072" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-072":
      """
      {"type": "array", "maxItems": 1}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 073 — Detect removed minItems
    Given the global compatibility level is "NONE"
    And subject "jsd07-073" has "JSON" schema:
      """
      {"type": "array", "minItems": 1}
      """
    When I set the config for subject "jsd07-073" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-073":
      """
      {"type": "array"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 074 — Detect increased minItems
    Given the global compatibility level is "NONE"
    And subject "jsd07-074" has "JSON" schema:
      """
      {"type": "array", "minItems": 1}
      """
    When I set the config for subject "jsd07-074" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-074":
      """
      {"type": "array", "minItems": 2}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 075 — Detect decreased minItems
    Given the global compatibility level is "NONE"
    And subject "jsd07-075" has "JSON" schema:
      """
      {"type": "array", "minItems": 2}
      """
    When I set the config for subject "jsd07-075" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-075":
      """
      {"type": "array", "minItems": 1}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 076 — Detect unique items removed
    Given the global compatibility level is "NONE"
    And subject "jsd07-076" has "JSON" schema:
      """
      {"type": "array", "uniqueItems": true}
      """
    When I set the config for subject "jsd07-076" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-076":
      """
      {"type": "array"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 077 — Detect incompatible changes to reference schemas
    Given the global compatibility level is "NONE"
    And subject "jsd07-077" has "JSON" schema:
      """
      {"type": "array", "items": {"$ref": "#/definitions/someRef"}, "definitions": {"someRef": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-077" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-077":
      """
      {"type": "array", "items": {"$ref": "#/definitions/someRef"}, "definitions": {"someRef": {"type": "number"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 078 — Detect compatible change of moving to reference schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-078" has "JSON" schema:
      """
      {"type": "string"}
      """
    When I set the config for subject "jsd07-078" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-078":
      """
      {"$ref": "#/definitions/someRef", "definitions": {"someRef": {"type": "string"}}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 079 — Detect compatible change of moving from reference schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-079" has "JSON" schema:
      """
      {"$ref": "#/definitions/someRef", "definitions": {"someRef": {"type": "string"}}}
      """
    When I set the config for subject "jsd07-079" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-079":
      """
      {"type": "string"}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 080 — Detect added boolean additional properties
    Given the global compatibility level is "NONE"
    And subject "jsd07-080" has "JSON" schema:
      """
      {"properties": {}, "additionalProperties": false}
      """
    When I set the config for subject "jsd07-080" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-080":
      """
      {"properties": {}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 081 — Detect removed boolean additional properties
    Given the global compatibility level is "NONE"
    And subject "jsd07-081" has "JSON" schema:
      """
      {"additionalProperties": true}
      """
    When I set the config for subject "jsd07-081" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-081":
      """
      {"additionalProperties": false}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 082 — Detect narrowing changes to additional properties schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-082" has "JSON" schema:
      """
      {"additionalProperties": true}
      """
    When I set the config for subject "jsd07-082" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-082":
      """
      {"additionalProperties": {"type": "number"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 083 — Detect adding empty schema to open content model
    Given the global compatibility level is "NONE"
    And subject "jsd07-083" has "JSON" schema:
      """
      {"additionalProperties": true}
      """
    When I set the config for subject "jsd07-083" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-083":
      """
      {"additionalProperties": true, "properties": {"foo": true}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 084 — Detect removing empty schema from open content model
    Given the global compatibility level is "NONE"
    And subject "jsd07-084" has "JSON" schema:
      """
      {"additionalProperties": true, "properties": {"foo": true}}
      """
    When I set the config for subject "jsd07-084" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-084":
      """
      {"additionalProperties": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 085 — Detect adding false to closed content model
    Given the global compatibility level is "NONE"
    And subject "jsd07-085" has "JSON" schema:
      """
      {"additionalProperties": false}
      """
    When I set the config for subject "jsd07-085" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-085":
      """
      {"additionalProperties": false, "properties": {"foo": false}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 086 — Detect removing false from closed content model
    Given the global compatibility level is "NONE"
    And subject "jsd07-086" has "JSON" schema:
      """
      {"additionalProperties": false, "properties": {"foo": false}}
      """
    When I set the config for subject "jsd07-086" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-086":
      """
      {"additionalProperties": false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 087 — Detect changes to additional properties schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-087" has "JSON" schema:
      """
      {"additionalProperties": {"type": "string"}}
      """
    When I set the config for subject "jsd07-087" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-087":
      """
      {"additionalProperties": {"type": "number"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 088 — Detect removed boolean additional items
    Given the global compatibility level is "NONE"
    And subject "jsd07-088" has "JSON" schema:
      """
      {"additionalItems": true}
      """
    When I set the config for subject "jsd07-088" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-088":
      """
      {"additionalItems": false}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 089 — Detect changes to additional items schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-089" has "JSON" schema:
      """
      {"additionalItems": {"type": "string"}}
      """
    When I set the config for subject "jsd07-089" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-089":
      """
      {"additionalItems": {"type": "number"}}
      """
    Then the compatibility check should be incompatible

  Scenario: JSON Schema diff Draft-07 090 — Detect adding empty schema to open content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-090" has "JSON" schema:
      """
      {"type": "array", "additionalItems": true, "items": [{"type": "number"}]}
      """
    When I set the config for subject "jsd07-090" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-090":
      """
      {"type": "array", "additionalItems": true, "items": [{"type": "number"}, true]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 091 — Detect removing empty schema from open content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-091" has "JSON" schema:
      """
      {"type": "array", "additionalItems": true, "items": [{"type": "number"}, true]}
      """
    When I set the config for subject "jsd07-091" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-091":
      """
      {"type": "array", "additionalItems": true, "items": [{"type": "number"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 092 — Detect adding false to closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-092" has "JSON" schema:
      """
      {"type": "array", "additionalItems": false, "items": [{"type": "number"}]}
      """
    When I set the config for subject "jsd07-092" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-092":
      """
      {"type": "array", "additionalItems": false, "items": [{"type": "number"}, false]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 093 — Detect removing false from closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-093" has "JSON" schema:
      """
      {"type": "array", "additionalItems": false, "items": [{"type": "number"}, false]}
      """
    When I set the config for subject "jsd07-093" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-093":
      """
      {"type": "array", "additionalItems": false, "items": [{"type": "number"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 094 — Detect adding to closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-094" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": false}
      """
    When I set the config for subject "jsd07-094" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-094":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}, {"type": "boolean"}], "additionalItems": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 095 — Detect removing from closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-095" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}, {"type": "boolean"}], "additionalItems": false}
      """
    When I set the config for subject "jsd07-095" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-095":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 096 — Detect removing from open content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-096" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": true}
      """
    When I set the config for subject "jsd07-096" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-096":
      """
      {"type": "array", "items": [{"type": "string"}], "additionalItems": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 097 — Detect adding to closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-097" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}], "additionalItems": false}
      """
    When I set the config for subject "jsd07-097" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-097":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 098 — Detect removing number from closed content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-098" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": false}
      """
    When I set the config for subject "jsd07-098" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-098":
      """
      {"type": "array", "items": [{"type": "string"}], "additionalItems": {"type": "number"}}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 099 — Detect adding number to partially open content model for array
    Given the global compatibility level is "NONE"
    And subject "jsd07-099" has "JSON" schema:
      """
      {"type": "array", "items": [{"type": "string"}], "additionalItems": {"type": "number"}}
      """
    When I set the config for subject "jsd07-099" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-099":
      """
      {"type": "array", "items": [{"type": "string"}, {"type": "number"}], "additionalItems": true}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 100 — Detect change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-100" has "JSON" schema:
      """
      {"type": "string"}
      """
    When I set the config for subject "jsd07-100" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-100":
      """
      {"oneOf": [{"type": "number"}, {"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 101 — Detect change to optional oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-101" has "JSON" schema:
      """
      {"type": "null"}
      """
    When I set the config for subject "jsd07-101" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-101":
      """
      {"oneOf": [{"type": "number"}, {"type": "null"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 102 — Detect change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-102" has "JSON" schema:
      """
      {"oneOf": [{"type": "number"}]}
      """
    When I set the config for subject "jsd07-102" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-102":
      """
      {"oneOf": [{"type": "string"}, {"type": "number"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 103 — Detect change to oneOf schema
    Given the global compatibility level is "NONE"
    And subject "jsd07-103" has "JSON" schema:
      """
      {"type": "string", "maxLength": 3}
      """
    When I set the config for subject "jsd07-103" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-103":
      """
      {"oneOf": [{"type": "number"}, {"type": "string"}]}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema diff Draft-07 104 — Allow addition of un-reserved property
    Given the global compatibility level is "NONE"
    And subject "jsd07-104" has "JSON" schema:
      """
      {"properties": {"foo": {"type": "string"}}, "additionalProperties": false}
      """
    When I set the config for subject "jsd07-104" to "BACKWARD"
    And I check compatibility of "JSON" schema against subject "jsd07-104":
      """
      {"properties": {"foo": {"type": "string"}, "bar": {"type": "string"}}, "additionalProperties": false}
      """
    Then the compatibility check should be compatible
