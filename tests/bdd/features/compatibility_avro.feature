@functional @compatibility
Feature: Avro Schema Compatibility
  Exhaustive Avro compatibility checks across all seven compatibility modes,
  covering type promotions, field additions/removals, enum evolution, unions,
  nested records, maps, arrays, fixed types, and error validation.

  # ---------------------------------------------------------------------------
  # BACKWARD mode (8 scenarios)
  #   New schema (reader) must be able to read data written by old schema (writer).
  #   reader=new, writer=old
  # ---------------------------------------------------------------------------

  Scenario: BACKWARD - add field with default is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-back-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-back-1":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string","default":"unknown"}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - add field without default is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-back-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-back-2":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - remove field is compatible (reader ignores extra writer fields)
    Given the global compatibility level is "BACKWARD"
    And subject "avro-back-3" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    When I register a schema under subject "avro-back-3":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - change field type string to int is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-back-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-back-4":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"int"}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - type promotion int to long is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-back-5" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    When I register a schema under subject "avro-back-5":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"long"}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - reverse type promotion long to int is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-back-6" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"long"}]}
      """
    When I register a schema under subject "avro-back-6":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - add enum symbol is compatible (reader has superset)
    Given the global compatibility level is "BACKWARD"
    And subject "avro-back-7" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["ACTIVE","INACTIVE"]}}]}
      """
    When I register a schema under subject "avro-back-7":
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["ACTIVE","INACTIVE","PENDING"]}}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - rename field is incompatible (old field missing, no default)
    Given the global compatibility level is "BACKWARD"
    And subject "avro-back-8" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-back-8":
      """
      {"type":"record","name":"User","fields":[{"name":"full_name","type":"string"}]}
      """
    Then the response status should be 409

  # ---------------------------------------------------------------------------
  # BACKWARD_TRANSITIVE mode (6 scenarios)
  #   New schema must be backward compatible with ALL previous versions.
  # ---------------------------------------------------------------------------

  Scenario: BACKWARD_TRANSITIVE - 3-version chain all backward compatible
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And subject "avro-bt-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-bt-1":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null},{"name":"age","type":["null","int"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - v3 incompatible with v1 but compatible with v2 is rejected
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"}]}
      """
    And subject "avro-bt-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-bt-2":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"int","default":0}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD_TRANSITIVE - type promotion chain int to long to float
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-3" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    And subject "avro-bt-3" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"long"}]}
      """
    When I register a schema under subject "avro-bt-3":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"float"}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - each version adds field with default
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And subject "avro-bt-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string","default":"none"}]}
      """
    When I register a schema under subject "avro-bt-4":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string","default":"none"},{"name":"phone","type":"string","default":"none"}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - v3 changes type of field present in v1
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-5" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"}]}
      """
    And subject "avro-bt-5" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"},{"name":"extra","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-bt-5":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"int","default":0},{"name":"extra","type":["null","string"],"default":null}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD_TRANSITIVE - enum grows each version
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-6" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["A"]}}]}
      """
    And subject "avro-bt-6" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["A","B"]}}]}
      """
    When I register a schema under subject "avro-bt-6":
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["A","B","C"]}}]}
      """
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # FORWARD mode (8 scenarios)
  #   Old schema (reader) must be able to read data written by new schema (writer).
  #   reader=old, writer=new
  # ---------------------------------------------------------------------------

  Scenario: FORWARD - add field is compatible (old reader ignores new field)
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-fwd-1":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: FORWARD - remove required field that old reader uses is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    When I register a schema under subject "avro-fwd-2":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: FORWARD - add field without default is compatible (forward only checks old reads new)
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-3" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-fwd-3":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    Then the response status should be 200

  Scenario: FORWARD - change field type is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-fwd-4":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"int"}]}
      """
    Then the response status should be 409

  Scenario: FORWARD - string to bytes promotion is compatible
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-5" has schema:
      """
      {"type":"record","name":"Data","fields":[{"name":"payload","type":"string"}]}
      """
    When I register a schema under subject "avro-fwd-5":
      """
      {"type":"record","name":"Data","fields":[{"name":"payload","type":"bytes"}]}
      """
    Then the response status should be 200

  Scenario: FORWARD - type demotion long to int is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-6" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    When I register a schema under subject "avro-fwd-6":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"long"}]}
      """
    Then the response status should be 409

  Scenario: FORWARD - add enum symbol is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-7" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["ACTIVE","INACTIVE"]}}]}
      """
    When I register a schema under subject "avro-fwd-7":
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["ACTIVE","INACTIVE","PENDING"]}}]}
      """
    Then the response status should be 409

  Scenario: FORWARD - remove optional field with default is compatible
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-8" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"nickname","type":"string","default":"none"}]}
      """
    When I register a schema under subject "avro-fwd-8":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # FORWARD_TRANSITIVE mode (5 scenarios)
  #   New schema must be forward compatible with ALL previous versions.
  # ---------------------------------------------------------------------------

  Scenario: FORWARD_TRANSITIVE - 3-version compatible chain
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "avro-ft-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And subject "avro-ft-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    When I register a schema under subject "avro-ft-1":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"},{"name":"phone","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: FORWARD_TRANSITIVE - v3 breaks v1 forward compat
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "avro-ft-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"}]}
      """
    And subject "avro-ft-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-ft-2":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"int","default":0},{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE - enum addition in chain is incompatible
    Given the global compatibility level is "NONE"
    And subject "avro-ft-3" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["A","B","C"]}}]}
      """
    And subject "avro-ft-3" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["A","B","C","D"]}}]}
      """
    Given subject "avro-ft-3" has compatibility level "FORWARD_TRANSITIVE"
    When I register a schema under subject "avro-ft-3":
      """
      {"type":"record","name":"Event","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["A","B","C","D","E"]}}]}
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE - progressive field addition
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "avro-ft-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And subject "avro-ft-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    When I register a schema under subject "avro-ft-4":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"},{"name":"score","type":"long"}]}
      """
    Then the response status should be 200

  Scenario: FORWARD_TRANSITIVE - field type change breaks chain
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "avro-ft-5" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    And subject "avro-ft-5" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"},{"name":"label","type":"string"}]}
      """
    When I register a schema under subject "avro-ft-5":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"string"},{"name":"label","type":"string"}]}
      """
    Then the response status should be 409

  # ---------------------------------------------------------------------------
  # FULL mode (7 scenarios)
  #   New schema must be both backward AND forward compatible with latest.
  # ---------------------------------------------------------------------------

  Scenario: FULL - add field with default is compatible (safe both ways)
    Given the global compatibility level is "FULL"
    And subject "avro-full-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-full-1":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: FULL - add field without default is incompatible (fails backward)
    Given the global compatibility level is "FULL"
    And subject "avro-full-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-full-2":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: FULL - remove field without default is incompatible (fails forward)
    Given the global compatibility level is "FULL"
    And subject "avro-full-3" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    When I register a schema under subject "avro-full-3":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: FULL - string to bytes bidirectional promotion is compatible
    Given the global compatibility level is "FULL"
    And subject "avro-full-4" has schema:
      """
      {"type":"record","name":"Data","fields":[{"name":"payload","type":"string"}]}
      """
    When I register a schema under subject "avro-full-4":
      """
      {"type":"record","name":"Data","fields":[{"name":"payload","type":"bytes"}]}
      """
    Then the response status should be 200

  Scenario: FULL - int to long promotion is incompatible (one-way only)
    Given the global compatibility level is "FULL"
    And subject "avro-full-5" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    When I register a schema under subject "avro-full-5":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"long"}]}
      """
    Then the response status should be 409

  Scenario: FULL - add nullable union field with default null is compatible
    Given the global compatibility level is "FULL"
    And subject "avro-full-6" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-full-6":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"phone","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: FULL - identical schema is compatible
    Given the global compatibility level is "FULL"
    And subject "avro-full-7" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    When I register a schema under subject "avro-full-7":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # FULL_TRANSITIVE mode (4 scenarios)
  #   New schema must be both backward AND forward compatible with ALL versions.
  # ---------------------------------------------------------------------------

  Scenario: FULL_TRANSITIVE - safe 3-version evolution with nullable fields
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "avro-flt-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And subject "avro-flt-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-flt-1":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null},{"name":"phone","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: FULL_TRANSITIVE - one-way promotion in chain is incompatible
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "avro-flt-2" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    And subject "avro-flt-2" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"},{"name":"label","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-flt-2":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"long"},{"name":"label","type":["null","string"],"default":null}]}
      """
    Then the response status should be 409

  Scenario: FULL_TRANSITIVE - complex 3-version with nullable fields
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "avro-flt-3" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}
      """
    And subject "avro-flt-3" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"tag","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-flt-3":
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"tag","type":["null","string"],"default":null},{"name":"score","type":["null","int"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: FULL_TRANSITIVE - remove field in chain is incompatible
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "avro-flt-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    And subject "avro-flt-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"},{"name":"phone","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-flt-4":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"phone","type":["null","string"],"default":null}]}
      """
    Then the response status should be 409

  # ---------------------------------------------------------------------------
  # NONE mode (2 scenarios)
  #   No compatibility checks — any schema change is allowed.
  # ---------------------------------------------------------------------------

  Scenario: NONE - completely different schema is accepted
    Given the global compatibility level is "NONE"
    And subject "avro-none-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-none-1":
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"long"},{"name":"total","type":"double"}]}
      """
    Then the response status should be 200

  Scenario: NONE - incompatible type change is accepted
    Given the global compatibility level is "NONE"
    And subject "avro-none-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    When I register a schema under subject "avro-none-2":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"int"},{"name":"age","type":"string"}]}
      """
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # Edge Cases (8 scenarios)
  #   Complex Avro types: unions, nested records, maps, arrays, fixed.
  # ---------------------------------------------------------------------------

  Scenario: BACKWARD - union expansion (add type to union) is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-edge-1" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"payload","type":["null","string"]}]}
      """
    When I register a schema under subject "avro-edge-1":
      """
      {"type":"record","name":"Event","fields":[{"name":"payload","type":["null","string","int"]}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - union contraction (remove type from union) is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-edge-2" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"payload","type":["null","string","int"]}]}
      """
    When I register a schema under subject "avro-edge-2":
      """
      {"type":"record","name":"Event","fields":[{"name":"payload","type":["null","string"]}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - nested record field type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-edge-3" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"address","type":{"type":"record","name":"Address","fields":[{"name":"city","type":"string"}]}}]}
      """
    When I register a schema under subject "avro-edge-3":
      """
      {"type":"record","name":"User","fields":[{"name":"address","type":{"type":"record","name":"Address","fields":[{"name":"city","type":"int"}]}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - map value type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-edge-4" has schema:
      """
      {"type":"record","name":"Config","fields":[{"name":"props","type":{"type":"map","values":"string"}}]}
      """
    When I register a schema under subject "avro-edge-4":
      """
      {"type":"record","name":"Config","fields":[{"name":"props","type":{"type":"map","values":"int"}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - array item type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-edge-5" has schema:
      """
      {"type":"record","name":"Collection","fields":[{"name":"items","type":{"type":"array","items":"string"}}]}
      """
    When I register a schema under subject "avro-edge-5":
      """
      {"type":"record","name":"Collection","fields":[{"name":"items","type":{"type":"array","items":"int"}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - fixed type size change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-edge-6" has schema:
      """
      {"type":"record","name":"Token","fields":[{"name":"hash","type":{"type":"fixed","name":"Hash","size":16}}]}
      """
    When I register a schema under subject "avro-edge-6":
      """
      {"type":"record","name":"Token","fields":[{"name":"hash","type":{"type":"fixed","name":"Hash","size":32}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - fixed type name change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-edge-7" has schema:
      """
      {"type":"record","name":"Token","fields":[{"name":"hash","type":{"type":"fixed","name":"MD5Hash","size":16}}]}
      """
    When I register a schema under subject "avro-edge-7":
      """
      {"type":"record","name":"Token","fields":[{"name":"hash","type":{"type":"fixed","name":"SHA256Hash","size":16}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - record name change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-edge-8" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-edge-8":
      """
      {"type":"record","name":"Person","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 409

  # ---------------------------------------------------------------------------
  # Error Validation (6 scenarios)
  #   Verify error response structure, compatibility check endpoint, config
  #   overrides, and mode enforcement.
  # ---------------------------------------------------------------------------

  Scenario: Error validation - 409 response has error_code field
    Given the global compatibility level is "BACKWARD"
    And subject "avro-err-1" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-err-1":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the response status should be 409
    And the response should have error code 409

  Scenario: Error validation - compatibility check endpoint returns is_compatible false
    Given the global compatibility level is "BACKWARD"
    And subject "avro-err-2" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I check compatibility of schema against subject "avro-err-2":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the compatibility check should be incompatible

  Scenario: Error validation - per-subject NONE override bypasses BACKWARD global
    Given the global compatibility level is "BACKWARD"
    And subject "avro-err-3" has compatibility level "NONE"
    And subject "avro-err-3" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-err-3":
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 200

  Scenario: Error validation - delete per-subject config falls back to global
    Given the global compatibility level is "BACKWARD"
    And subject "avro-err-4" has compatibility level "NONE"
    And subject "avro-err-4" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I delete the config for subject "avro-err-4"
    And I register a schema under subject "avro-err-4":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: Error validation - READONLY mode can be set and retrieved
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the global mode
    Then the response status should be 200
    And the response field "mode" should be "READONLY"
    # Reset mode so cleanup can proceed
    When I set the global mode to "READWRITE"
    Then the response status should be 200

  Scenario: Error validation - compatibility check endpoint returns is_compatible true for compatible schema
    Given the global compatibility level is "BACKWARD"
    And subject "avro-err-6" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I check compatibility of schema against subject "avro-err-6":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the compatibility check should be compatible

  # --- Gap-filling: Avro-specific compatibility rules ---

  Scenario: BACKWARD - fixed type size mismatch is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-1" has schema:
      """
      {"type":"record","name":"Data","fields":[{"name":"hash","type":{"type":"fixed","name":"Hash","size":16}}]}
      """
    When I register a schema under subject "avro-gap-1":
      """
      {"type":"record","name":"Data","fields":[{"name":"hash","type":{"type":"fixed","name":"Hash","size":32}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - fixed type name mismatch is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-2" has schema:
      """
      {"type":"record","name":"Data","fields":[{"name":"hash","type":{"type":"fixed","name":"MD5Hash","size":16}}]}
      """
    When I register a schema under subject "avro-gap-2":
      """
      {"type":"record","name":"Data","fields":[{"name":"hash","type":{"type":"fixed","name":"SHA1Hash","size":16}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - record name mismatch is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-3" has schema:
      """
      {"type":"record","name":"UserV1","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-gap-3":
      """
      {"type":"record","name":"UserV2","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - union widening (add type to union) is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-4" has schema:
      """
      {"type":"record","name":"Evt","fields":[{"name":"val","type":["null","string"]}]}
      """
    When I register a schema under subject "avro-gap-4":
      """
      {"type":"record","name":"Evt","fields":[{"name":"val","type":["null","string","int"]}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - union narrowing (remove type from union) is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-5" has schema:
      """
      {"type":"record","name":"Evt","fields":[{"name":"val","type":["null","string","int"]}]}
      """
    When I register a schema under subject "avro-gap-5":
      """
      {"type":"record","name":"Evt","fields":[{"name":"val","type":["null","string"]}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - non-union to union widening is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-6" has schema:
      """
      {"type":"record","name":"Evt","fields":[{"name":"val","type":"string"}]}
      """
    When I register a schema under subject "avro-gap-6":
      """
      {"type":"record","name":"Evt","fields":[{"name":"val","type":["null","string"]}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - map value type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-7" has schema:
      """
      {"type":"record","name":"Data","fields":[{"name":"meta","type":{"type":"map","values":"string"}}]}
      """
    When I register a schema under subject "avro-gap-7":
      """
      {"type":"record","name":"Data","fields":[{"name":"meta","type":{"type":"map","values":"int"}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - array item type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-8" has schema:
      """
      {"type":"record","name":"Data","fields":[{"name":"items","type":{"type":"array","items":"string"}}]}
      """
    When I register a schema under subject "avro-gap-8":
      """
      {"type":"record","name":"Data","fields":[{"name":"items","type":{"type":"array","items":"int"}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - deeply nested record field change detected
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-9" has schema:
      """
      {"type":"record","name":"Root","fields":[{"name":"child","type":{"type":"record","name":"Child","fields":[{"name":"grandchild","type":{"type":"record","name":"GrandChild","fields":[{"name":"value","type":"string"}]}}]}}]}
      """
    When I register a schema under subject "avro-gap-9":
      """
      {"type":"record","name":"Root","fields":[{"name":"child","type":{"type":"record","name":"Child","fields":[{"name":"grandchild","type":{"type":"record","name":"GrandChild","fields":[{"name":"value","type":"int"}]}}]}}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - string to bytes promotion is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-10" has schema:
      """
      {"type":"record","name":"Data","fields":[{"name":"payload","type":"string"}]}
      """
    When I register a schema under subject "avro-gap-10":
      """
      {"type":"record","name":"Data","fields":[{"name":"payload","type":"bytes"}]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - bytes to string promotion is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "avro-gap-11" has schema:
      """
      {"type":"record","name":"Data","fields":[{"name":"payload","type":"bytes"}]}
      """
    When I register a schema under subject "avro-gap-11":
      """
      {"type":"record","name":"Data","fields":[{"name":"payload","type":"string"}]}
      """
    Then the response status should be 200
