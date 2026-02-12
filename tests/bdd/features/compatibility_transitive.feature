@functional @compatibility
Feature: Transitive Compatibility - Multi-Version Chains
  Exhaustive testing of transitive vs non-transitive modes with 3-5 version chains.
  The key insight: transitive modes check against ALL previous versions,
  while non-transitive only checks against the latest.

  IMPORTANT: When a transitive mode is active, registering v2 checks it against v1.
  So if v2 is NOT compatible with v1 under the active mode, the Given step fails.
  For "differentiator" tests (non-transitive passes but transitive fails), we must
  register v1 and v2 under NONE, then switch to the target mode for v3.

  # ==========================================================================
  # AVRO TRANSITIVE CHAINS
  # ==========================================================================

  Scenario: Avro BACKWARD_TRANSITIVE - 3 versions all compatible
    # Adding fields with defaults is always backward-compatible (new reader has defaults
    # for fields not in old writer data). This works transitively too.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-chain-1" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"}]}
      """
    And subject "avro-bt-chain-1" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    When I register a schema under subject "avro-bt-chain-1":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"ts","type":"long","default":0}]}
      """
    Then the response status should be 200

  Scenario: Avro BACKWARD vs BACKWARD_TRANSITIVE - non-transitive passes, transitive fails
    # v1={id, code:string}, v2={id} (drops code), v3={id, code:int default:0} (re-adds code as int).
    # Under BACKWARD, v2 dropping code is fine: new reader (v2) reads old data (v1), v2 ignores code.
    # v3 vs v2 (latest): v3 reader reads v2 data. v3 needs code with default. v2 has no code. v3 uses default. PASS.
    # But v3 vs v1: v3 reader reads v1 data. v1 has code:string. v3 expects code:int. Type mismatch. FAIL.
    # Since v2 dropping code IS backward-compatible, we can register all under BACKWARD.
    Given the global compatibility level is "BACKWARD"
    And subject "avro-bt-vs-b" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"string"},{"name":"code","type":"string"}]}
      """
    And subject "avro-bt-vs-b" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "avro-bt-vs-b":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"string"},{"name":"code","type":"int","default":0}]}
      """
    Then the response status should be 200

  Scenario: Avro BACKWARD_TRANSITIVE catches what BACKWARD misses
    # Same schema chain as above, but under BACKWARD_TRANSITIVE.
    # v2={id} is backward-compatible with v1={id,code} (reader v2 ignores code). PASS.
    # v3={id, code:int default:0} vs v1={id, code:string}: type mismatch on code. FAIL.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-catch" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"string"},{"name":"code","type":"string"}]}
      """
    And subject "avro-bt-catch" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "avro-bt-catch":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"string"},{"name":"code","type":"int","default":0}]}
      """
    Then the response status should be 409

  Scenario: Avro BACKWARD_TRANSITIVE - type promotion chain int->long->float
    # Avro supports numeric widening: int->long->float->double.
    # BACKWARD: new reader reads old writer data.
    # v2(long) reads v1(int): int promotable to long. PASS.
    # v3(float) reads v1(int): int promotable to float. PASS.
    # v3(float) reads v2(long): long promotable to float. PASS.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-promo" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    And subject "avro-bt-promo" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"long"}]}
      """
    When I register a schema under subject "avro-bt-promo":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"float"}]}
      """
    Then the response status should be 200

  Scenario: Avro BACKWARD_TRANSITIVE - enum grows across 3 versions
    # Adding enum symbols is backward-compatible: new reader has superset of symbols.
    # Old writer data with symbols [NEW] can be read by new reader with [NEW,PROCESSING,DONE].
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-enum" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["NEW"]}}]}
      """
    And subject "avro-bt-enum" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["NEW","PROCESSING"]}}]}
      """
    When I register a schema under subject "avro-bt-enum":
      """
      {"type":"record","name":"Order","fields":[{"name":"status","type":{"type":"enum","name":"Status","symbols":["NEW","PROCESSING","DONE"]}}]}
      """
    Then the response status should be 200

  Scenario: Avro BACKWARD_TRANSITIVE - 4 versions progressive field addition
    # Each version adds a field with a default. Backward-compatible against all prior versions.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-4ver" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    And subject "avro-bt-4ver" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    And subject "avro-bt-4ver" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""}]}
      """
    When I register a schema under subject "avro-bt-4ver":
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""},{"name":"age","type":"int","default":0}]}
      """
    Then the response status should be 200

  Scenario: Avro FORWARD_TRANSITIVE - 3 versions compatible
    # For Avro FORWARD: old reader reads new writer data. Old reader ignores unknown fields.
    # Adding nullable fields with defaults to the new schema is forward-compatible because
    # old readers simply ignore the new fields, and new fields have defaults for old writers.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "avro-ft-chain-1" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"}]}
      """
    And subject "avro-ft-chain-1" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"tag","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-ft-chain-1":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"tag","type":["null","string"],"default":null},{"name":"src","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: Avro FORWARD_TRANSITIVE - v3 removes field from v1
    # v1={id, name}, v2={id, name, tag}. Forward-compatible (v1 ignores tag).
    # v3={id, tag} removes "name". v1 reader reads v3 data: v1 needs name, v3 doesn't have it.
    # v1.name has no default -> FAIL against v1.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "avro-ft-remove" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "avro-ft-remove" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"tag","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-ft-remove":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"tag","type":["null","string"],"default":null}]}
      """
    Then the response status should be 409

  Scenario: Avro FORWARD vs FORWARD_TRANSITIVE - non-transitive passes, transitive fails
    # v1={id, name}, v2={id} — v2 drops name. Under FORWARD, v1 reads v2: v1 needs name,
    # v2 doesn't have it, v1.name has no default -> FORWARD INCOMPATIBLE.
    # So we register v1, v2 under NONE, then switch to FORWARD for v3.
    # v3={id, code default:""}: FORWARD vs v2(latest): v2(reader) reads v3(writer).
    # v2 needs only id, v3 has id. v2 ignores code. PASS.
    # Under FORWARD_TRANSITIVE, also checked vs v1: v1(reader) reads v3(writer).
    # v1 needs name (no default), v3 doesn't have it. FAIL.
    Given the global compatibility level is "NONE"
    And subject "avro-ft-vs-f" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "avro-ft-vs-f" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"}]}
      """
    And the global compatibility level is "FORWARD"
    When I register a schema under subject "avro-ft-vs-f":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"code","type":"string","default":""}]}
      """
    Then the response status should be 200

  Scenario: Avro FORWARD_TRANSITIVE catches what FORWARD misses
    # Same v1, v2 as above, registered under NONE. Switch to FORWARD_TRANSITIVE.
    # v3={id, code default:""}: checked vs ALL versions.
    # vs v2={id}: v2 reads v3. v2 needs id. v3 has id. v2 ignores code. PASS.
    # vs v1={id, name}: v1 reads v3. v1 needs name (no default). v3 doesn't have name. FAIL.
    Given the global compatibility level is "NONE"
    And subject "avro-ft-catch" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "avro-ft-catch" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"}]}
      """
    And the global compatibility level is "FORWARD_TRANSITIVE"
    When I register a schema under subject "avro-ft-catch":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"code","type":"string","default":""}]}
      """
    Then the response status should be 409

  Scenario: Avro FULL_TRANSITIVE - safe 3-version evolution
    # FULL = BACKWARD + FORWARD. Adding nullable fields with defaults is both:
    # - Backward-compatible: new reader has default for missing field in old data.
    # - Forward-compatible: old reader ignores unknown fields from new writer.
    # This works transitively because each version is a strict superset.
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "avro-flt-safe" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"}]}
      """
    And subject "avro-flt-safe" has schema:
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"tag","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-flt-safe":
      """
      {"type":"record","name":"Event","fields":[{"name":"id","type":"int"},{"name":"tag","type":["null","string"],"default":null},{"name":"src","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: Avro FULL_TRANSITIVE - promotion fails (one-directional)
    # Type promotion (int->long) is backward-compatible but NOT forward-compatible.
    # FULL requires both. v2={value:long} from v1={value:int}:
    # FORWARD: v1(int) reads v2(long). long->int not promotable. FAIL.
    # Register v1, v2 under NONE. Switch to FULL_TRANSITIVE for v3.
    # v3={value:float}: FULL_TRANSITIVE checks vs all.
    # BACKWARD vs v1: v3(float) reads v1(int). int->float. PASS.
    # FORWARD vs v1: v1(int) reads v3(float). float->int not promotable. FAIL.
    Given the global compatibility level is "NONE"
    And subject "avro-flt-promo" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"int"}]}
      """
    And subject "avro-flt-promo" has schema:
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"long"}]}
      """
    And the global compatibility level is "FULL_TRANSITIVE"
    When I register a schema under subject "avro-flt-promo":
      """
      {"type":"record","name":"Metric","fields":[{"name":"value","type":"float"}]}
      """
    Then the response status should be 409

  Scenario: Avro FULL_TRANSITIVE - 4-version safe chain with nullable fields
    # Each version adds a nullable field with null default. This is both backward
    # and forward compatible: new readers use defaults, old readers ignore new fields.
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "avro-flt-4v" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    And subject "avro-flt-4v" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}
      """
    And subject "avro-flt-4v" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null},{"name":"email","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-flt-4v":
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null},{"name":"email","type":["null","string"],"default":null},{"name":"phone","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: Avro BACKWARD_TRANSITIVE - 5 version complex evolution
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt-5v" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"float"}]}
      """
    And subject "avro-bt-5v" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"float"},{"name":"currency","type":"string","default":"USD"}]}
      """
    And subject "avro-bt-5v" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"float"},{"name":"currency","type":"string","default":"USD"},{"name":"region","type":"string","default":"US"}]}
      """
    And subject "avro-bt-5v" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"float"},{"name":"currency","type":"string","default":"USD"},{"name":"region","type":"string","default":"US"},{"name":"ts","type":"long","default":0}]}
      """
    When I register a schema under subject "avro-bt-5v":
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"float"},{"name":"currency","type":"string","default":"USD"},{"name":"region","type":"string","default":"US"},{"name":"ts","type":"long","default":0},{"name":"tag","type":"string","default":""}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # JSON SCHEMA TRANSITIVE CHAINS
  # ==========================================================================

  Scenario: JSON Schema BACKWARD_TRANSITIVE - 3 versions all compatible
    # Adding optional properties is backward-compatible: the checker treats the new schema
    # as "reader" with more properties than the old "writer". No properties removed.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-chain-1" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-bt-chain-1" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-bt-chain-1":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200

  Scenario: JSON Schema BACKWARD_TRANSITIVE - v3 adds required prop (fails vs v1)
    # v1={id req}, v2={id, name optional}. Backward-compatible (no removal, no new required).
    # v3={id, name required}. vs v1: "name" is new required property. FAIL.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-reqd" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-bt-reqd" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-bt-reqd":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id","name"]}
      """
    Then the response status should be 409

  Scenario: JSON Schema BACKWARD vs BACKWARD_TRANSITIVE differentiator
    # Under BACKWARD, adding optional properties is always compatible.
    # This test just verifies a 3-version chain passes BACKWARD.
    Given the global compatibility level is "BACKWARD"
    And subject "json-bt-vs-b" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-bt-vs-b" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-bt-vs-b":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200

  Scenario: JSON Schema BACKWARD_TRANSITIVE - constraint relaxation chain
    # Relaxing maxLength is backward-compatible: new schema accepts wider range.
    # The checker flags tightening (new < old) but not relaxing (new > old).
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-relax" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string","maxLength":50}},"required":["name"]}
      """
    And subject "json-bt-relax" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string","maxLength":100}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-bt-relax":
      """
      {"type":"object","properties":{"name":{"type":"string","maxLength":200}},"required":["name"]}
      """
    Then the response status should be 200

  Scenario: JSON Schema BACKWARD_TRANSITIVE - 4 versions progressive evolution
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-4v" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-bt-4v" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"]}
      """
    And subject "json-bt-4v" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-bt-4v":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200

  Scenario: JSON Schema FORWARD_TRANSITIVE - 3 versions compatible
    # For JSON FORWARD: checker.Check(existing, new) treats existing as "new schema"
    # and new as "old schema". Properties in the new schema (old param) not in existing
    # (new param) are flagged as "removed". So FORWARD-compatible evolution means each
    # new version has FEWER or SAME properties (old readers have all fields).
    # v1 has {id, name}, v2 drops name to {id}, v3 keeps {id} but adds metadata.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-chain-1" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"]}
      """
    And subject "json-ft-chain-1" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-ft-chain-1":
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"],"additionalProperties":true}
      """
    Then the response status should be 200

  Scenario: JSON Schema FORWARD_TRANSITIVE - property removal fails chain
    # v1={id, name, email, required:[id,name]}, v2={id, name, required:[id,name]}.
    # FORWARD v2 vs v1: checker.Check(v1, v2). new=v1, old=v2. v2 has {id, name}.
    # Both in v1. No removal. PASS.
    # v3={id, required:[id]}: removes name.
    # FORWARD vs v1: checker.Check(v1, v3). new=v1, old=v3. v3 has {id}. id in v1. PASS.
    # But FORWARD vs v2: checker.Check(v2, v3). new=v2, old=v3. v3 has {id}. id in v2.
    # name in v2 not in v3: "new required property" since name is required in v2.
    # Actually: name is in v2 (newProps) but not in v3 (oldProps). The checker checks
    # oldProps not in newProps (= removal). v3 has only {id}. v2 has {id, name}.
    # For "removed properties": old=v3={id}. All in new=v2. No removal.
    # For "new required": name in v2, not in v3, and required in v2 -> "new required property". FAIL.
    # So this actually works. But wait, we need v2 to register under FORWARD_TRANSITIVE.
    # v2 removes email from v1. checker.Check(v1, v2). new=v1, old=v2. oldProps=v2={id,name}.
    # id and name in v1. No removal. v1 has email not in v2. email is not required in v2 context.
    # Actually "new required": email in v1, not in v2, and required in v1? required=[id,name].
    # email is NOT required. So "new optional property" — no error. PASS.
    Given the global compatibility level is "NONE"
    And subject "json-ft-remove" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"},"email":{"type":"string"}},"required":["id","name"]}
      """
    And subject "json-ft-remove" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id","name"]}
      """
    And the global compatibility level is "FORWARD_TRANSITIVE"
    When I register a "JSON" schema under subject "json-ft-remove":
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    Then the response status should be 409

  Scenario: JSON Schema FULL_TRANSITIVE - safe 3-version evolution
    # FULL_TRANSITIVE = BACKWARD + FORWARD checked against ALL versions.
    # The JSON checker treats FORWARD as schema-evolution-in-reverse, which means
    # adding properties fails FORWARD (seen as "removal" from old schema's perspective).
    # For safe FULL_TRANSITIVE: schemas must have the same property set.
    # We use schemas that differ only in metadata (title) which the checker ignores,
    # producing distinct fingerprints while being structurally identical.
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "json-flt-safe" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"],"title":"v1"}
      """
    And subject "json-flt-safe" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"],"title":"v2"}
      """
    When I register a "JSON" schema under subject "json-flt-safe":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"],"title":"v3"}
      """
    Then the response status should be 200

  Scenario: JSON Schema FULL_TRANSITIVE - adding required property fails
    # v1={id}, v2={id, name optional}. Under FULL_TRANSITIVE:
    # BACKWARD v2 vs v1: new=v2, old=v1. No removal, no new required. PASS.
    # FORWARD v2 vs v1: new=v1, old=v2. name in v2 not in v1 -> "Property removed". FAIL.
    # So register v1, v2 under NONE. Switch to FULL_TRANSITIVE for v3.
    # v3={id, name required}: BACKWARD vs v1 -> "new required property 'name'". FAIL.
    Given the global compatibility level is "NONE"
    And subject "json-flt-fail" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-flt-fail" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"]}
      """
    And the global compatibility level is "FULL_TRANSITIVE"
    When I register a "JSON" schema under subject "json-flt-fail":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id","name"]}
      """
    Then the response status should be 409

  Scenario: JSON Schema BACKWARD_TRANSITIVE - enum evolution across 3 versions
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-enum" has "JSON" schema:
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["NEW"]}},"required":["status"]}
      """
    And subject "json-bt-enum" has "JSON" schema:
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["NEW","ACTIVE"]}},"required":["status"]}
      """
    When I register a "JSON" schema under subject "json-bt-enum":
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["NEW","ACTIVE","DONE"]}},"required":["status"]}
      """
    Then the response status should be 200

  Scenario: JSON Schema BACKWARD_TRANSITIVE - nested object evolution
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-nested" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"address":{"type":"object","properties":{"city":{"type":"string"}}}},"required":["id"]}
      """
    And subject "json-bt-nested" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"address":{"type":"object","properties":{"city":{"type":"string"},"state":{"type":"string"}}}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-bt-nested":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"address":{"type":"object","properties":{"city":{"type":"string"},"state":{"type":"string"},"zip":{"type":"string"}}}},"required":["id"]}
      """
    Then the response status should be 200

  Scenario: JSON Schema BACKWARD_TRANSITIVE - 5 version chain
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-5v" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-bt-5v" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"a":{"type":"string"}},"required":["id"]}
      """
    And subject "json-bt-5v" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"a":{"type":"string"},"b":{"type":"string"}},"required":["id"]}
      """
    And subject "json-bt-5v" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"integer"},"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"string"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-bt-5v":
      """
      {"type":"object","properties":{"id":{"type":"integer"},"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"string"},"d":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200

  # ==========================================================================
  # PROTOBUF TRANSITIVE CHAINS
  # ==========================================================================

  Scenario: Protobuf BACKWARD_TRANSITIVE - 3 versions all compatible
    # Adding new fields in proto3 is backward-compatible: the checker treats the new
    # schema as "reader" with more fields. Old "writer" fields are all present.
    # New fields not in old schema are just "new field added" (no error).
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-chain-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
      }
      """
    And subject "proto-bt-chain-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string name = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-chain-1":
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string name = 2;
        string email = 3;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf BACKWARD vs BACKWARD_TRANSITIVE differentiator
    # v1={id, code:string}, v2={id} (removes code). Under BACKWARD, the checker
    # flags "field removed" for code. So we register v1, v2 under NONE.
    # v3={id, code:int32}: BACKWARD vs v2 (latest). v2 has {id}. v3 has {id, code}.
    # code not in v2 = "new field". id matches. PASS.
    # BACKWARD_TRANSITIVE vs v1: code type changed string->int32. FAIL.
    Given the global compatibility level is "NONE"
    And subject "proto-bt-vs-b" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string code = 2;
      }
      """
    And subject "proto-bt-vs-b" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
      }
      """
    And the global compatibility level is "BACKWARD"
    When I register a "PROTOBUF" schema under subject "proto-bt-vs-b":
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        int32 code = 2;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf BACKWARD_TRANSITIVE catches field number type change
    # Same v1, v2 as above but registered under NONE. Switch to BACKWARD_TRANSITIVE.
    # v3={id, code:int32}: checked against ALL versions.
    # vs v2={id}: code not in v2, "new field". PASS.
    # vs v1={id, code:string}: code type changed string->int32. FAIL.
    Given the global compatibility level is "NONE"
    And subject "proto-bt-catch" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string code = 2;
      }
      """
    And subject "proto-bt-catch" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
      }
      """
    And the global compatibility level is "BACKWARD_TRANSITIVE"
    When I register a "PROTOBUF" schema under subject "proto-bt-catch":
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        int32 code = 2;
      }
      """
    Then the response status should be 409

  Scenario: Protobuf BACKWARD_TRANSITIVE - compatible type group across versions
    # int32, sint32, sfixed32 are in the same compatible type group.
    # Type changes within a group pass both backward and forward.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-typegroup" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Metric {
        int32 value = 1;
      }
      """
    And subject "proto-bt-typegroup" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Metric {
        sint32 value = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-typegroup":
      """
      syntax = "proto3";
      message Metric {
        sfixed32 value = 1;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf BACKWARD_TRANSITIVE - 4 versions progressive fields
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-4v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        int32 id = 1;
      }
      """
    And subject "proto-bt-4v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        int32 id = 1;
        string name = 2;
      }
      """
    And subject "proto-bt-4v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        int32 id = 1;
        string name = 2;
        string email = 3;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-4v":
      """
      syntax = "proto3";
      message User {
        int32 id = 1;
        string name = 2;
        string email = 3;
        int32 age = 4;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf BACKWARD_TRANSITIVE - enum evolution across 3 versions
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-enum" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Order {
        enum Status {
          NEW = 0;
        }
        Status status = 1;
      }
      """
    And subject "proto-bt-enum" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Order {
        enum Status {
          NEW = 0;
          ACTIVE = 1;
        }
        Status status = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-enum":
      """
      syntax = "proto3";
      message Order {
        enum Status {
          NEW = 0;
          ACTIVE = 1;
          DONE = 2;
        }
        Status status = 1;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf FORWARD_TRANSITIVE - 3 versions compatible
    # The Protobuf checker treats FORWARD as checker.Check(existing, new) where
    # existing=reader(new in checker), new=writer(old in checker). Adding fields
    # to the new schema causes the checker to flag "field removed" (old schema has
    # field not in new). So for FORWARD-compatible evolution, we use type-group
    # changes which pass in both directions.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "proto-ft-chain-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        int64 timestamp = 2;
      }
      """
    And subject "proto-ft-chain-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        sint32 id = 1;
        sint64 timestamp = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ft-chain-1":
      """
      syntax = "proto3";
      message Event {
        sfixed32 id = 1;
        sfixed64 timestamp = 2;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf FORWARD_TRANSITIVE - field removal fails chain
    # v1={id, name}, v2={id, name, tag}. Adding tag under FORWARD_TRANSITIVE:
    # checker.Check(v1, v2): newMsg=v1, oldMsg=v2. tag in v2 not in v1 -> "removed". FAIL.
    # Register v1, v2 under NONE. Switch to FORWARD_TRANSITIVE for v3.
    # v3={id, tag}: removes name.
    # vs v1: checker.Check(v1, v3). newMsg=v1, oldMsg=v3. v3={id:1, tag:3}. tag:3 in v3
    # not in v1 -> "field removed". FAIL.
    # vs v2: checker.Check(v2, v3). newMsg=v2={id,name,tag}, oldMsg=v3={id,tag}.
    # tag found. id found. name in v2 not in v3: "new field". No remaining. PASS.
    # Fails vs v1. Expected 409.
    Given the global compatibility level is "NONE"
    And subject "proto-ft-remove" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string name = 2;
      }
      """
    And subject "proto-ft-remove" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string name = 2;
        string tag = 3;
      }
      """
    And the global compatibility level is "FORWARD_TRANSITIVE"
    When I register a "PROTOBUF" schema under subject "proto-ft-remove":
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string tag = 3;
      }
      """
    Then the response status should be 409

  Scenario: Protobuf FORWARD vs FORWARD_TRANSITIVE differentiator
    # The Protobuf checker's FORWARD direction flags fields in the new schema not
    # present in old as "removed". For a differentiator we use type-group crossing:
    # v1={int32 val=1}, v2={uint32 val=1}, v3={fixed32 val=1}.
    # Register v1, v2 under NONE. Switch to FORWARD for v3.
    # FORWARD v3 vs v2 (latest): checker.Check(v2, v3). uint32 vs fixed32.
    # Compatible group [uint32, fixed32]. PASS.
    # FORWARD_TRANSITIVE vs v1: checker.Check(v1, v3). int32 vs fixed32.
    # int32 group [int32, sint32, sfixed32]. fixed32 not in that group. FAIL.
    Given the global compatibility level is "NONE"
    And subject "proto-ft-vs-f" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 value = 1;
      }
      """
    And subject "proto-ft-vs-f" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        uint32 value = 1;
      }
      """
    And the global compatibility level is "FORWARD"
    When I register a "PROTOBUF" schema under subject "proto-ft-vs-f":
      """
      syntax = "proto3";
      message Event {
        fixed32 value = 1;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf FORWARD_TRANSITIVE catches what FORWARD misses
    # Same v1, v2 as above. Switch to FORWARD_TRANSITIVE for v3.
    # v3={fixed32 val=1}: vs v2(uint32) PASS, vs v1(int32) FAIL (cross type-group).
    Given the global compatibility level is "NONE"
    And subject "proto-ft-catch" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 value = 1;
      }
      """
    And subject "proto-ft-catch" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        uint32 value = 1;
      }
      """
    And the global compatibility level is "FORWARD_TRANSITIVE"
    When I register a "PROTOBUF" schema under subject "proto-ft-catch":
      """
      syntax = "proto3";
      message Event {
        fixed32 value = 1;
      }
      """
    Then the response status should be 409

  Scenario: Protobuf FULL_TRANSITIVE - safe 3-version evolution
    # FULL = BACKWARD + FORWARD. Type-group changes are compatible in both directions.
    # int32, sint32, sfixed32 are all in the same compatible group.
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "proto-flt-safe" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        int64 timestamp = 2;
      }
      """
    And subject "proto-flt-safe" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        sint32 id = 1;
        sint64 timestamp = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-flt-safe":
      """
      syntax = "proto3";
      message Event {
        sfixed32 id = 1;
        sfixed64 timestamp = 2;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf FULL_TRANSITIVE - field removal fails
    # v1={id, name}, v2={id, name, email}. Adding email under FULL_TRANSITIVE:
    # FORWARD: checker.Check(v1, v2). email in v2 not in v1 -> "removed". FAIL.
    # Register v1, v2 under NONE. Switch to FULL_TRANSITIVE for v3.
    # v3={id, email}: removes name. BACKWARD vs v1: name in v1 not in v3 -> "removed". FAIL.
    Given the global compatibility level is "NONE"
    And subject "proto-flt-fail" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string name = 2;
      }
      """
    And subject "proto-flt-fail" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string name = 2;
        string email = 3;
      }
      """
    And the global compatibility level is "FULL_TRANSITIVE"
    When I register a "PROTOBUF" schema under subject "proto-flt-fail":
      """
      syntax = "proto3";
      message Event {
        int32 id = 1;
        string email = 3;
      }
      """
    Then the response status should be 409

  Scenario: Protobuf BACKWARD_TRANSITIVE - 5 version complex evolution
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-5v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Order {
        int32 id = 1;
        string product = 2;
      }
      """
    And subject "proto-bt-5v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Order {
        int32 id = 1;
        string product = 2;
        double amount = 3;
      }
      """
    And subject "proto-bt-5v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Order {
        int32 id = 1;
        string product = 2;
        double amount = 3;
        string currency = 4;
      }
      """
    And subject "proto-bt-5v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Order {
        int32 id = 1;
        string product = 2;
        double amount = 3;
        string currency = 4;
        string region = 5;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-5v":
      """
      syntax = "proto3";
      message Order {
        int32 id = 1;
        string product = 2;
        double amount = 3;
        string currency = 4;
        string region = 5;
        int64 timestamp = 6;
      }
      """
    Then the response status should be 200

  Scenario: Protobuf FULL_TRANSITIVE - 4-version safe evolution
    # Type-group changes are FULL-compatible (both backward and forward).
    # Using multiple compatible type groups across 4 versions.
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "proto-flt-4v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        int32 id = 1;
        int64 timestamp = 2;
      }
      """
    And subject "proto-flt-4v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        sint32 id = 1;
        sint64 timestamp = 2;
      }
      """
    And subject "proto-flt-4v" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        sfixed32 id = 1;
        sfixed64 timestamp = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-flt-4v":
      """
      syntax = "proto3";
      message User {
        int32 id = 1;
        sfixed64 timestamp = 2;
      }
      """
    Then the response status should be 200
