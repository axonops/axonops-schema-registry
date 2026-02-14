@functional
Feature: Avro Compatibility — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive Avro compatibility tests from the Confluent Schema Registry v8.1.1
  test suite covering backward, forward, full, and transitive modes.

  # ==========================================================================
  # BACKWARD COMPATIBILITY (Section 22)
  # ==========================================================================

  Scenario: Backward — adding field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-add-def" has compatibility level "BACKWARD"
    And subject "avro-ex-back-add-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-add-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"foo"}]}
      """
    Then the response status should be 200

  Scenario: Backward — adding field without default is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-add-nodef" has compatibility level "BACKWARD"
    And subject "avro-ex-back-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-add-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: Backward — removing field is compatible (old reader ignores extra)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-remove" has compatibility level "BACKWARD"
    And subject "avro-ex-back-remove" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I register a schema under subject "avro-ex-back-remove":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Backward — changing field name with alias is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-alias" has compatibility level "BACKWARD"
    And subject "avro-ex-back-alias" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-alias":
      """
      {"type":"record","name":"R","fields":[{"name":"f1_new","type":"string","aliases":["f1"]}]}
      """
    Then the response status should be 200

  Scenario: Backward — evolving field type to union is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-to-union" has compatibility level "BACKWARD"
    And subject "avro-ex-back-to-union" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-to-union":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":["null","string"]}]}
      """
    Then the response status should be 200

  Scenario: Backward — removing type from union is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-narrow-union" has compatibility level "BACKWARD"
    And subject "avro-ex-back-narrow-union" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":["null","string"]}]}
      """
    When I register a schema under subject "avro-ex-back-narrow-union":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: Backward — adding type to union is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-widen-union" has compatibility level "BACKWARD"
    And subject "avro-ex-back-widen-union" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":["null","string"]}]}
      """
    When I register a schema under subject "avro-ex-back-widen-union":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":["null","string","int"]}]}
      """
    Then the response status should be 200

  Scenario: Backward — int to long promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-int-long" has compatibility level "BACKWARD"
    And subject "avro-ex-back-int-long" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-back-int-long":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    Then the response status should be 200

  Scenario: Backward — int to float promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-int-float" has compatibility level "BACKWARD"
    And subject "avro-ex-back-int-float" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-back-int-float":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"float"}]}
      """
    Then the response status should be 200

  Scenario: Backward — int to double promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-int-double" has compatibility level "BACKWARD"
    And subject "avro-ex-back-int-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-back-int-double":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"double"}]}
      """
    Then the response status should be 200

  Scenario: Backward — long to float promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-long-float" has compatibility level "BACKWARD"
    And subject "avro-ex-back-long-float" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    When I register a schema under subject "avro-ex-back-long-float":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"float"}]}
      """
    Then the response status should be 200

  Scenario: Backward — long to double promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-long-double" has compatibility level "BACKWARD"
    And subject "avro-ex-back-long-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    When I register a schema under subject "avro-ex-back-long-double":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"double"}]}
      """
    Then the response status should be 200

  Scenario: Backward — float to double promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-float-double" has compatibility level "BACKWARD"
    And subject "avro-ex-back-float-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"float"}]}
      """
    When I register a schema under subject "avro-ex-back-float-double":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"double"}]}
      """
    Then the response status should be 200

  Scenario: Backward — string to bytes promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-str-bytes" has compatibility level "BACKWARD"
    And subject "avro-ex-back-str-bytes" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-str-bytes":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"bytes"}]}
      """
    Then the response status should be 200

  Scenario: Backward — bytes to string promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-bytes-str" has compatibility level "BACKWARD"
    And subject "avro-ex-back-bytes-str" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"bytes"}]}
      """
    When I register a schema under subject "avro-ex-back-bytes-str":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Backward — changing field type incompatibly is rejected
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-type-change" has compatibility level "BACKWARD"
    And subject "avro-ex-back-type-change" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-type-change":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    Then the response status should be 409

  Scenario: Backward — changing record name is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-rename" has compatibility level "BACKWARD"
    And subject "avro-ex-back-rename" has schema:
      """
      {"type":"record","name":"Original","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-back-rename":
      """
      {"type":"record","name":"Renamed","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: Backward — adding enum symbol is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-enum-add" has compatibility level "BACKWARD"
    And subject "avro-ex-back-enum-add" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B"]}}]}
      """
    When I register a schema under subject "avro-ex-back-enum-add":
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B","C"]}}]}
      """
    Then the response status should be 200

  Scenario: Backward — removing enum symbol is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-back-enum-remove" has compatibility level "BACKWARD"
    And subject "avro-ex-back-enum-remove" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B","C"]}}]}
      """
    When I register a schema under subject "avro-ex-back-enum-remove":
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B"]}}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # FORWARD COMPATIBILITY (Section 23)
  # ==========================================================================

  Scenario: Forward — adding field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-add-def" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-add-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-fwd-add-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the response status should be 200

  Scenario: Forward — adding field without default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-add-nodef" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-fwd-add-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Forward — removing field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-remove-def" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-remove-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I register a schema under subject "avro-ex-fwd-remove-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Forward — removing field without default is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-remove-nodef" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-remove-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-fwd-remove-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: Forward — int to long is incompatible (old reader can't read long)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-int-long" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-int-long" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-fwd-int-long":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    Then the response status should be 409

  Scenario: Forward — long to int is compatible (old reader promotes int to long)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fwd-long-int" has compatibility level "FORWARD"
    And subject "avro-ex-fwd-long-int" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    When I register a schema under subject "avro-ex-fwd-long-int":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # FULL COMPATIBILITY (Section 24)
  # ==========================================================================

  Scenario: Full — adding field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-add-def" has compatibility level "FULL"
    And subject "avro-ex-full-add-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-full-add-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the response status should be 200

  Scenario: Full — adding field without default is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-add-nodef" has compatibility level "FULL"
    And subject "avro-ex-full-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-full-add-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: Full — removing field with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-remove-def" has compatibility level "FULL"
    And subject "avro-ex-full-remove-def" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I register a schema under subject "avro-ex-full-remove-def":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Full — removing field without default is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-remove-nodef" has compatibility level "FULL"
    And subject "avro-ex-full-remove-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-full-remove-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: Full — string/bytes bidirectional promotion is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-str-bytes" has compatibility level "FULL"
    And subject "avro-ex-full-str-bytes" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I register a schema under subject "avro-ex-full-str-bytes":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"bytes"}]}
      """
    Then the response status should be 200

  Scenario: Full — int to long is incompatible (only forward-compatible)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-full-int-long" has compatibility level "FULL"
    And subject "avro-ex-full-int-long" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"int"}]}
      """
    When I register a schema under subject "avro-ex-full-int-long":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"long"}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # TRANSITIVE COMPATIBILITY (Section 25)
  # ==========================================================================

  Scenario: BACKWARD_TRANSITIVE — progressive field addition is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-bt-add" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-bt-add" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-bt-add" to "BACKWARD_TRANSITIVE"
    And I register a schema under subject "avro-ex-bt-add":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"},{"name":"f3","type":"string","default":"b"}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE — removing default transitively is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-bt-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-bt-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-bt-nodef" to "BACKWARD_TRANSITIVE"
    And I register a schema under subject "avro-ex-bt-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE — progressive field removal is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-ft-remove" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"},{"name":"f3","type":"string","default":"b"}]}
      """
    And subject "avro-ex-ft-remove" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-ft-remove" to "FORWARD_TRANSITIVE"
    And I register a schema under subject "avro-ex-ft-remove":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: FORWARD_TRANSITIVE — adding field without default is compatible (old readers ignore new fields)
    Given the global compatibility level is "NONE"
    And subject "avro-ex-ft-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-ft-add-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-ft-add-nodef" to "FORWARD_TRANSITIVE"
    And I register a schema under subject "avro-ex-ft-add-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f3","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: FULL_TRANSITIVE — safe evolution with defaults is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fullt-safe" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-fullt-safe" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-fullt-safe" to "FULL_TRANSITIVE"
    And I register a schema under subject "avro-ex-fullt-safe":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"},{"name":"f3","type":"string","default":"b"}]}
      """
    Then the response status should be 200

  Scenario: FULL_TRANSITIVE — field without default transitively is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-fullt-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-fullt-nodef" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"a"}]}
      """
    When I set the config for subject "avro-ex-fullt-nodef" to "FULL_TRANSITIVE"
    And I register a schema under subject "avro-ex-fullt-nodef":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # COMPATIBILITY CHECK ENDPOINT (REST API)
  # ==========================================================================

  Scenario: Compatibility check endpoint — compatible returns is_compatible true
    Given the global compatibility level is "NONE"
    And subject "avro-ex-check-compat" has compatibility level "BACKWARD"
    And subject "avro-ex-check-compat" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "avro-ex-check-compat":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    Then the compatibility check should be compatible

  Scenario: Compatibility check endpoint — incompatible returns is_compatible false
    Given the global compatibility level is "NONE"
    And subject "avro-ex-check-incompat" has compatibility level "BACKWARD"
    And subject "avro-ex-check-incompat" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "avro-ex-check-incompat":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: Compatibility check against specific version
    Given the global compatibility level is "NONE"
    And subject "avro-ex-check-ver" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-check-ver" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I set the config for subject "avro-ex-check-ver" to "BACKWARD"
    And I check compatibility of schema against subject "avro-ex-check-ver" version 1:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"},{"name":"f3","type":"string","default":"y"}]}
      """
    Then the compatibility check should be compatible

  Scenario: Compatibility check against all versions
    Given the global compatibility level is "NONE"
    And subject "avro-ex-check-all" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"}]}
      """
    And subject "avro-ex-check-all" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"}]}
      """
    When I set the config for subject "avro-ex-check-all" to "BACKWARD"
    And I check compatibility of schema against all versions of subject "avro-ex-check-all":
      """
      {"type":"record","name":"R","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":"x"},{"name":"f3","type":"string","default":"y"}]}
      """
    Then the compatibility check should be compatible

  # ==========================================================================
  # NESTED RECORD COMPATIBILITY
  # ==========================================================================

  Scenario: Backward — nested record field addition with default is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-nested-add" has compatibility level "BACKWARD"
    And subject "avro-ex-nested-add" has schema:
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"a","type":"string"}]}}]}
      """
    When I register a schema under subject "avro-ex-nested-add":
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":"x"}]}}]}
      """
    Then the response status should be 200

  Scenario: Backward — nested record type change is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-nested-type" has compatibility level "BACKWARD"
    And subject "avro-ex-nested-type" has schema:
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"a","type":"string"}]}}]}
      """
    When I register a schema under subject "avro-ex-nested-type":
      """
      {"type":"record","name":"Outer","fields":[{"name":"inner","type":{"type":"record","name":"Inner","fields":[{"name":"a","type":"int"}]}}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # MAP AND ARRAY COMPATIBILITY
  # ==========================================================================

  Scenario: Backward — map value type change is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-map-type" has compatibility level "BACKWARD"
    And subject "avro-ex-map-type" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"m","type":{"type":"map","values":"string"}}]}
      """
    When I register a schema under subject "avro-ex-map-type":
      """
      {"type":"record","name":"R","fields":[{"name":"m","type":{"type":"map","values":"int"}}]}
      """
    Then the response status should be 409

  Scenario: Backward — array item type change is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-array-type" has compatibility level "BACKWARD"
    And subject "avro-ex-array-type" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"a","type":{"type":"array","items":"string"}}]}
      """
    When I register a schema under subject "avro-ex-array-type":
      """
      {"type":"record","name":"R","fields":[{"name":"a","type":{"type":"array","items":"int"}}]}
      """
    Then the response status should be 409

  Scenario: Backward — map value promotion (int to long) is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-map-promo" has compatibility level "BACKWARD"
    And subject "avro-ex-map-promo" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"m","type":{"type":"map","values":"int"}}]}
      """
    When I register a schema under subject "avro-ex-map-promo":
      """
      {"type":"record","name":"R","fields":[{"name":"m","type":{"type":"map","values":"long"}}]}
      """
    Then the response status should be 200

  Scenario: Backward — array item promotion (int to long) is compatible
    Given the global compatibility level is "NONE"
    And subject "avro-ex-array-promo" has compatibility level "BACKWARD"
    And subject "avro-ex-array-promo" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"a","type":{"type":"array","items":"int"}}]}
      """
    When I register a schema under subject "avro-ex-array-promo":
      """
      {"type":"record","name":"R","fields":[{"name":"a","type":{"type":"array","items":"long"}}]}
      """
    Then the response status should be 200
