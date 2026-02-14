@functional
Feature: JSON Schema Validation Compatibility — Exhaustive (Confluent v8.1.1 Compatibility)
  JSON Schema reader/writer compatibility validation tests from the Confluent
  Schema Registry v8.1.1 test suite. Tests compatible and incompatible
  reader/writer pairs across types, enums, unions, and records.

  # ==========================================================================
  # SELF-COMPATIBILITY
  # ==========================================================================

  Scenario: JSON Schema is compatible with itself
    Given the global compatibility level is "NONE"
    And subject "jsv-self" has compatibility level "BACKWARD"
    And subject "jsv-self" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"},"score":{"type":"number"},"active":{"type":"boolean"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-self":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"},"score":{"type":"number"},"active":{"type":"boolean"}}}
      """
    Then the compatibility check should be compatible

  # ==========================================================================
  # COMPATIBLE READER/WRITER PAIRS (14 cases from Confluent)
  # ==========================================================================

  Scenario: Compatible — number reader, integer writer
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-01" has compatibility level "BACKWARD"
    And subject "jsv-compat-01" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-01":
      """
      {"type":"object","properties":{"v":{"type":"number"}}}
      """
    Then the compatibility check should be compatible

  Scenario: Compatible — number array reader, integer array writer
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-02" has compatibility level "BACKWARD"
    And subject "jsv-compat-02" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"array","items":{"type":"integer"}}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-02":
      """
      {"type":"object","properties":{"v":{"type":"array","items":{"type":"number"}}}}
      """
    Then the compatibility check should be compatible

  Scenario: Compatible — enum superset reader, enum subset writer
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-03" has compatibility level "BACKWARD"
    And subject "jsv-compat-03" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"string","enum":["A","B"]}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-03":
      """
      {"type":"object","properties":{"v":{"type":"string","enum":["A","B","C"]}}}
      """
    Then the compatibility check should be compatible

  Scenario: Compatible — oneOf superset reader, oneOf subset writer
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-04" has compatibility level "BACKWARD"
    And subject "jsv-compat-04" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"oneOf":[{"type":"string"}]}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-04":
      """
      {"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: Compatible — oneOf reader, single type writer
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-05" has compatibility level "BACKWARD"
    And subject "jsv-compat-05" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"string"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-05":
      """
      {"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer"}]}}}
      """
    Then the compatibility check should be compatible

  Scenario: Incompatible — adding property to open content model (Confluent PROPERTY_ADDED_TO_OPEN_CONTENT_MODEL)
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-06" has compatibility level "BACKWARD"
    And subject "jsv-compat-06" has "JSON" schema:
      """
      {"type":"object","properties":{}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-06":
      """
      {"type":"object","properties":{"a":{"type":"integer"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — adding optional field with default to open content model
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-07" has compatibility level "BACKWARD"
    And subject "jsv-compat-07" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"integer"}},"required":["a"]}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-07":
      """
      {"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer","default":0}},"required":["a"]}
      """
    Then the compatibility check should be incompatible

  Scenario: Compatible — open content model with extra properties
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-08" has compatibility level "BACKWARD"
    And subject "jsv-compat-08" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-08":
      """
      {"type":"object","properties":{"a":{"type":"integer"}}}
      """
    Then the compatibility check should be compatible

  Scenario: Incompatible — adding non-required property to open content model
    Given the global compatibility level is "NONE"
    And subject "jsv-compat-09" has compatibility level "BACKWARD"
    And subject "jsv-compat-09" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"integer"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-compat-09":
      """
      {"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer"}}}
      """
    Then the compatibility check should be incompatible

  # ==========================================================================
  # INCOMPATIBLE READER/WRITER PAIRS (15 cases from Confluent)
  # ==========================================================================

  Scenario: Incompatible — boolean reader, integer writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-01" has compatibility level "BACKWARD"
    And subject "jsv-incompat-01" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-01":
      """
      {"type":"object","properties":{"v":{"type":"boolean"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — integer reader, boolean writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-02" has compatibility level "BACKWARD"
    And subject "jsv-incompat-02" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"boolean"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-02":
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — integer reader, number writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-03" has compatibility level "BACKWARD"
    And subject "jsv-incompat-03" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"number"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-03":
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — string reader, boolean writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-04" has compatibility level "BACKWARD"
    And subject "jsv-incompat-04" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"boolean"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-04":
      """
      {"type":"object","properties":{"v":{"type":"string"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — string reader, integer writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-05" has compatibility level "BACKWARD"
    And subject "jsv-incompat-05" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-05":
      """
      {"type":"object","properties":{"v":{"type":"string"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — integer array reader, number array writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-06" has compatibility level "BACKWARD"
    And subject "jsv-incompat-06" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"array","items":{"type":"number"}}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-06":
      """
      {"type":"object","properties":{"v":{"type":"array","items":{"type":"integer"}}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — integer array reader, string array writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-07" has compatibility level "BACKWARD"
    And subject "jsv-incompat-07" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"array","items":{"type":"string"}}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-07":
      """
      {"type":"object","properties":{"v":{"type":"array","items":{"type":"integer"}}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — enum subset reader, enum superset writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-08" has compatibility level "BACKWARD"
    And subject "jsv-incompat-08" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"string","enum":["A","B","C"]}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-08":
      """
      {"type":"object","properties":{"v":{"type":"string","enum":["A","B"]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — enum disjoint reader, enum superset writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-09" has compatibility level "BACKWARD"
    And subject "jsv-incompat-09" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"string","enum":["A","B","C"]}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-09":
      """
      {"type":"object","properties":{"v":{"type":"string","enum":["B","C"]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — integer reader, enum writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-10" has compatibility level "BACKWARD"
    And subject "jsv-incompat-10" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"string","enum":["A","B"]}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-10":
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — enum reader, integer writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-11" has compatibility level "BACKWARD"
    And subject "jsv-incompat-11" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-11":
      """
      {"type":"object","properties":{"v":{"type":"string","enum":["A","B"]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — oneOf subset reader, oneOf superset writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-12" has compatibility level "BACKWARD"
    And subject "jsv-incompat-12" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer"}]}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-12":
      """
      {"type":"object","properties":{"v":{"oneOf":[{"type":"string"}]}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — integer reader, oneOf writer
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-13" has compatibility level "BACKWARD"
    And subject "jsv-incompat-13" has "JSON" schema:
      """
      {"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"integer"}]}}}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-13":
      """
      {"type":"object","properties":{"v":{"type":"integer"}}}
      """
    Then the compatibility check should be incompatible

  Scenario: Incompatible — record adding required field without default
    Given the global compatibility level is "NONE"
    And subject "jsv-incompat-14" has compatibility level "BACKWARD"
    And subject "jsv-incompat-14" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"integer"}},"required":["a"]}
      """
    When I check compatibility of "JSON" schema against subject "jsv-incompat-14":
      """
      {"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer"}},"required":["a","b"]}
      """
    Then the compatibility check should be incompatible

  # ==========================================================================
  # TRANSITIVE COMPATIBILITY CHAINS
  # ==========================================================================

  Scenario: JSON Schema backward transitive — open content model rejects new properties
    Given the global compatibility level is "NONE"
    And subject "jsv-trans-ok" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"integer"}},"required":["a"]}
      """
    And subject "jsv-trans-ok" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"string","default":""}},"required":["a"]}
      """
    When I set the config for subject "jsv-trans-ok" to "BACKWARD_TRANSITIVE"
    And I register a "JSON" schema under subject "jsv-trans-ok":
      """
      {"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"string","default":""},"c":{"type":"number","default":0}},"required":["a"]}
      """
    Then the response status should be 409

  Scenario: JSON Schema backward transitive — closed content model allows new properties
    Given the global compatibility level is "NONE"
    And subject "jsv-trans-fail" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"integer"}},"required":["a"],"additionalProperties":false}
      """
    And subject "jsv-trans-fail" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"string"}},"required":["a"],"additionalProperties":false}
      """
    When I set the config for subject "jsv-trans-fail" to "BACKWARD_TRANSITIVE"
    And I register a "JSON" schema under subject "jsv-trans-fail":
      """
      {"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"string"},"c":{"type":"number"}},"required":["a"],"additionalProperties":false}
      """
    Then the response status should be 200

  # ==========================================================================
  # UNION / ONEOF COMPATIBILITY
  # ==========================================================================

  Scenario: Compatible — oneOf with compatible element types
    Given the global compatibility level is "NONE"
    And subject "jsv-union-compat" has compatibility level "BACKWARD"
    And subject "jsv-union-compat" has "JSON" schema:
      """
      {"oneOf":[{"type":"string"},{"type":"integer"}]}
      """
    When I check compatibility of "JSON" schema against subject "jsv-union-compat":
      """
      {"oneOf":[{"type":"string"},{"type":"integer"},{"type":"number"}]}
      """
    Then the compatibility check should be compatible

  Scenario: Incompatible — oneOf with narrowed types
    Given the global compatibility level is "NONE"
    And subject "jsv-union-incompat" has compatibility level "BACKWARD"
    And subject "jsv-union-incompat" has "JSON" schema:
      """
      {"oneOf":[{"type":"string"},{"type":"integer"},{"type":"number"}]}
      """
    When I check compatibility of "JSON" schema against subject "jsv-union-incompat":
      """
      {"oneOf":[{"type":"string"},{"type":"integer"}]}
      """
    Then the compatibility check should be incompatible
