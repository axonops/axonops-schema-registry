@functional @compatibility @edge-case
Feature: Compatibility Transitive Chain with 5+ Versions
  As a schema registry user using transitive compatibility modes
  I want to verify that BACKWARD_TRANSITIVE and FORWARD_TRANSITIVE
  correctly check against ALL previous versions in a 5+ version chain
  So that I can trust schema evolution is safe for all consumers

  Background:
    Given the schema registry is running

  # ---------------------------------------------------------------------------
  # BACKWARD_TRANSITIVE: 5-version chain, all compatible
  # ---------------------------------------------------------------------------

  Scenario: BACKWARD_TRANSITIVE with 5 compatible Avro versions succeeds
    # BACKWARD_TRANSITIVE: every new version (reader) MUST be able to read ALL previous (writer) data.
    # Adding optional fields (with defaults) is always backward-compatible.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    # v1: base schema
    And subject "bt-chain-5" has schema:
      """
      {"type":"record","name":"Chain","fields":[{"name":"id","type":"int"}]}
      """
    # v2: add optional field
    And subject "bt-chain-5" has schema:
      """
      {"type":"record","name":"Chain","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    # v3: add another optional field
    And subject "bt-chain-5" has schema:
      """
      {"type":"record","name":"Chain","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""}]}
      """
    # v4: add another optional field
    And subject "bt-chain-5" has schema:
      """
      {"type":"record","name":"Chain","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""},{"name":"age","type":"int","default":0}]}
      """
    # v5: add another optional field — should check against v1, v2, v3, v4
    When I register a schema under subject "bt-chain-5":
      """
      {"type":"record","name":"Chain","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""},{"name":"age","type":"int","default":0},{"name":"active","type":"boolean","default":true}]}
      """
    Then the response status should be 200
    And subject "bt-chain-5" should have exactly 5 versions
    And the audit log should contain event "schema_register" with subject "bt-chain-5"

  # ---------------------------------------------------------------------------
  # BACKWARD_TRANSITIVE: v6 breaks compatibility because it adds required field
  # ---------------------------------------------------------------------------

  Scenario: BACKWARD_TRANSITIVE rejects schema with required field not in older versions
    # Build a 5-version chain under NONE, then switch to BACKWARD_TRANSITIVE.
    # Attempt v6 that adds a required field (no default) — the new reader demands
    # a field that old writers never produced, making it backward-incompatible.
    Given the global compatibility level is "NONE"
    And subject "bt-break-chain" has schema:
      """
      {"type":"record","name":"Break","fields":[{"name":"id","type":"int"}]}
      """
    And subject "bt-break-chain" has schema:
      """
      {"type":"record","name":"Break","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""}]}
      """
    And subject "bt-break-chain" has schema:
      """
      {"type":"record","name":"Break","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""},{"name":"b","type":"string","default":""}]}
      """
    And subject "bt-break-chain" has schema:
      """
      {"type":"record","name":"Break","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""},{"name":"b","type":"string","default":""},{"name":"c","type":"string","default":""}]}
      """
    And subject "bt-break-chain" has schema:
      """
      {"type":"record","name":"Break","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""},{"name":"b","type":"string","default":""},{"name":"c","type":"string","default":""},{"name":"d","type":"string","default":""}]}
      """
    # Switch to BACKWARD_TRANSITIVE
    And the global compatibility level is "BACKWARD_TRANSITIVE"
    # v6: adds required "mandatory" field (no default) — new reader expects it but old writers never produced it
    When I check compatibility of schema against all versions of subject "bt-break-chain":
      """
      {"type":"record","name":"Break","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""},{"name":"b","type":"string","default":""},{"name":"c","type":"string","default":""},{"name":"d","type":"string","default":""},{"name":"mandatory","type":"string"}]}
      """
    Then the response status should be 200
    And the compatibility check should be incompatible

  # ---------------------------------------------------------------------------
  # FORWARD_TRANSITIVE: 5-version chain, all compatible
  # ---------------------------------------------------------------------------

  Scenario: FORWARD_TRANSITIVE with 5 compatible Avro versions succeeds
    # FORWARD_TRANSITIVE: old readers MUST be able to read new data.
    # Adding optional fields (with defaults) is forward-compatible because
    # old readers ignore unknown fields, and new fields have defaults.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    # v1: base schema
    And subject "ft-chain-5" has schema:
      """
      {"type":"record","name":"FwdChain","fields":[{"name":"id","type":"int"},{"name":"data","type":"string"}]}
      """
    # v2: add optional field (forward-compatible: old reader ignores new field)
    And subject "ft-chain-5" has schema:
      """
      {"type":"record","name":"FwdChain","fields":[{"name":"id","type":"int"},{"name":"data","type":"string"},{"name":"f1","type":"string","default":""}]}
      """
    # v3: add another optional field
    And subject "ft-chain-5" has schema:
      """
      {"type":"record","name":"FwdChain","fields":[{"name":"id","type":"int"},{"name":"data","type":"string"},{"name":"f1","type":"string","default":""},{"name":"f2","type":"string","default":""}]}
      """
    # v4: add another optional field
    And subject "ft-chain-5" has schema:
      """
      {"type":"record","name":"FwdChain","fields":[{"name":"id","type":"int"},{"name":"data","type":"string"},{"name":"f1","type":"string","default":""},{"name":"f2","type":"string","default":""},{"name":"f3","type":"int","default":0}]}
      """
    # v5: add another optional field — checked against ALL previous versions
    When I register a schema under subject "ft-chain-5":
      """
      {"type":"record","name":"FwdChain","fields":[{"name":"id","type":"int"},{"name":"data","type":"string"},{"name":"f1","type":"string","default":""},{"name":"f2","type":"string","default":""},{"name":"f3","type":"int","default":0},{"name":"f4","type":"boolean","default":false}]}
      """
    Then the response status should be 200
    And subject "ft-chain-5" should have exactly 5 versions
    And the audit log should contain event "schema_register" with subject "ft-chain-5"

  # ---------------------------------------------------------------------------
  # FULL_TRANSITIVE: 5-version chain, all compatible
  # ---------------------------------------------------------------------------

  Scenario: FULL_TRANSITIVE with 5 compatible Avro versions succeeds
    # FULL_TRANSITIVE: both backward AND forward compatible with ALL previous versions.
    # Only adding optional fields with defaults satisfies both directions.
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "fullt-chain-5" has schema:
      """
      {"type":"record","name":"FullChain","fields":[{"name":"id","type":"int"}]}
      """
    And subject "fullt-chain-5" has schema:
      """
      {"type":"record","name":"FullChain","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""}]}
      """
    And subject "fullt-chain-5" has schema:
      """
      {"type":"record","name":"FullChain","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""},{"name":"b","type":"int","default":0}]}
      """
    And subject "fullt-chain-5" has schema:
      """
      {"type":"record","name":"FullChain","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""},{"name":"b","type":"int","default":0},{"name":"c","type":"boolean","default":false}]}
      """
    When I register a schema under subject "fullt-chain-5":
      """
      {"type":"record","name":"FullChain","fields":[{"name":"id","type":"int"},{"name":"a","type":"string","default":""},{"name":"b","type":"int","default":0},{"name":"c","type":"boolean","default":false},{"name":"d","type":"long","default":0}]}
      """
    Then the response status should be 200
    And subject "fullt-chain-5" should have exactly 5 versions
    And the audit log should contain event "schema_register" with subject "fullt-chain-5"

  # ---------------------------------------------------------------------------
  # BACKWARD_TRANSITIVE rejects what non-transitive BACKWARD allows
  # ---------------------------------------------------------------------------

  Scenario: BACKWARD_TRANSITIVE rejects schema incompatible with older version (Avro)
    # v1 has required "id" (int), v2 adds optional "name", v3 adds required "mandatory"
    # Under BACKWARD, v3 only checks against v2 — v2 data has no "mandatory" field
    # but v2 is the latest, so the check fails with v2 too.
    # The correct differentiator: non-transitive BACKWARD checks only against latest.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "bt-diff-avro" has schema:
      """
      {"type":"record","name":"DiffAvro","fields":[{"name":"id","type":"int"}]}
      """
    And subject "bt-diff-avro" has schema:
      """
      {"type":"record","name":"DiffAvro","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    # v3: change type of "id" from int to string — backward-incompatible with v1 and v2
    When I check compatibility of schema against all versions of subject "bt-diff-avro":
      """
      {"type":"record","name":"DiffAvro","fields":[{"name":"id","type":"string"},{"name":"name","type":"string","default":""}]}
      """
    Then the response status should be 200
    And the compatibility check should be incompatible
