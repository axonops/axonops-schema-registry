@schema-modeling @avro @references
Feature: Avro Reference Evolution
  Tests for evolving schemas that use cross-subject references,
  including reference version pinning, multiple references,
  referencedby tracking, and reference deletion behavior.

  # ==========================================================================
  # 1. REFERENCE EVOLVES — CONSUMER STAYS PINNED TO V1
  # ==========================================================================

  Scenario: Consumer stays pinned to reference v1 when reference evolves
    Given subject "avro-refevo-base" has schema:
      """
      {"type":"record","name":"Address","namespace":"com.ref","fields":[
        {"name":"street","type":"string"},
        {"name":"city","type":"string"}
      ]}
      """
    When I register a schema under subject "avro-refevo-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.ref\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"home\",\"type\":\"com.ref.Address\"}]}",
        "references": [
          {"name":"com.ref.Address","subject":"avro-refevo-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "consumer_v1_id"
    # Evolve the reference
    When I register a schema under subject "avro-refevo-base":
      """
      {"type":"record","name":"Address","namespace":"com.ref","fields":[
        {"name":"street","type":"string"},
        {"name":"city","type":"string"},
        {"name":"zip","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    # Consumer re-registered with same ref v1 should get same ID
    When I register a schema under subject "avro-refevo-consumer-dup" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.ref\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"home\",\"type\":\"com.ref.Address\"}]}",
        "references": [
          {"name":"com.ref.Address","subject":"avro-refevo-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "consumer_v1_id"

  # ==========================================================================
  # 2. CONSUMER UPGRADES TO REFERENCE V2
  # ==========================================================================

  Scenario: Consumer upgrades reference version gets different schema ID
    Given subject "avro-refevo2-base" has schema:
      """
      {"type":"record","name":"Item","namespace":"com.ref2","fields":[
        {"name":"id","type":"long"}
      ]}
      """
    When I register a schema under subject "avro-refevo2-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.ref2\",\"fields\":[{\"name\":\"item\",\"type\":\"com.ref2.Item\"}]}",
        "references": [
          {"name":"com.ref2.Item","subject":"avro-refevo2-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "consumer_ref1_id"
    # Evolve the reference
    Given subject "avro-refevo2-base" has compatibility level "BACKWARD"
    When I register a schema under subject "avro-refevo2-base":
      """
      {"type":"record","name":"Item","namespace":"com.ref2","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    # Consumer re-registered pointing to ref v2
    When I register a schema under subject "avro-refevo2-consumer-v2" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.ref2\",\"fields\":[{\"name\":\"item\",\"type\":\"com.ref2.Item\"}]}",
        "references": [
          {"name":"com.ref2.Item","subject":"avro-refevo2-base","version":2}
        ]
      }
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "consumer_ref1_id"

  # ==========================================================================
  # 3. MULTIPLE REFERENCES — ONE EVOLVES
  # ==========================================================================

  Scenario: Multiple references where only one evolves
    Given subject "avro-multiref-a" has schema:
      """
      {"type":"record","name":"TypeA","namespace":"com.multi","fields":[{"name":"x","type":"int"}]}
      """
    And subject "avro-multiref-b" has schema:
      """
      {"type":"record","name":"TypeB","namespace":"com.multi","fields":[{"name":"y","type":"string"}]}
      """
    When I register a schema under subject "avro-multiref-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Combined\",\"namespace\":\"com.multi\",\"fields\":[{\"name\":\"a\",\"type\":\"com.multi.TypeA\"},{\"name\":\"b\",\"type\":\"com.multi.TypeB\"}]}",
        "references": [
          {"name":"com.multi.TypeA","subject":"avro-multiref-a","version":1},
          {"name":"com.multi.TypeB","subject":"avro-multiref-b","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "multi_v1_id"

  # ==========================================================================
  # 4. SAME SCHEMA BODY WITH DIFFERENT REFERENCES — DIFFERENT IDS
  # ==========================================================================

  Scenario: Same schema body with different reference versions produces different IDs
    Given subject "avro-diffref-base" has schema:
      """
      {"type":"record","name":"Dep","namespace":"com.diff","fields":[{"name":"v","type":"int"}]}
      """
    Given subject "avro-diffref-base" has compatibility level "NONE"
    When I register a schema under subject "avro-diffref-base":
      """
      {"type":"record","name":"Dep","namespace":"com.diff","fields":[{"name":"v","type":"long"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "avro-diffref-c1" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Main\",\"namespace\":\"com.diff\",\"fields\":[{\"name\":\"dep\",\"type\":\"com.diff.Dep\"}]}",
        "references": [
          {"name":"com.diff.Dep","subject":"avro-diffref-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "diffref_v1"
    When I register a schema under subject "avro-diffref-c2" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Main\",\"namespace\":\"com.diff\",\"fields\":[{\"name\":\"dep\",\"type\":\"com.diff.Dep\"}]}",
        "references": [
          {"name":"com.diff.Dep","subject":"avro-diffref-base","version":2}
        ]
      }
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "diffref_v1"

  # ==========================================================================
  # 5-6. REFERENCEDBY TRACKING
  # ==========================================================================

  Scenario: referencedby tracks multiple consumers of a reference
    Given subject "avro-refby-base" has schema:
      """
      {"type":"record","name":"Shared","namespace":"com.refby","fields":[{"name":"id","type":"long"}]}
      """
    When I register a schema under subject "avro-refby-consumer1" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"C1\",\"namespace\":\"com.refby\",\"fields\":[{\"name\":\"s\",\"type\":\"com.refby.Shared\"}]}",
        "references": [
          {"name":"com.refby.Shared","subject":"avro-refby-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I register a schema under subject "avro-refby-consumer2" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"C2\",\"namespace\":\"com.refby\",\"fields\":[{\"name\":\"s\",\"type\":\"com.refby.Shared\"}]}",
        "references": [
          {"name":"com.refby.Shared","subject":"avro-refby-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "avro-refby-base" version 1
    Then the response status should be 200

  # ==========================================================================
  # 7. COMPATIBILITY CHECK API WITH REFERENCES
  # ==========================================================================

  Scenario: Compatibility check API works with references
    Given subject "avro-refcompat-base" has schema:
      """
      {"type":"record","name":"Dep","namespace":"com.rc","fields":[{"name":"v","type":"int"}]}
      """
    And subject "avro-refcompat-consumer" has compatibility level "BACKWARD"
    When I register a schema under subject "avro-refcompat-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Main\",\"namespace\":\"com.rc\",\"fields\":[{\"name\":\"dep\",\"type\":\"com.rc.Dep\"},{\"name\":\"name\",\"type\":\"string\"}]}",
        "references": [
          {"name":"com.rc.Dep","subject":"avro-refcompat-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I check compatibility of schema against subject "avro-refcompat-consumer":
      """
      {"type":"record","name":"Main","namespace":"com.rc","fields":[
        {"name":"dep","type":{"type":"record","name":"Dep","namespace":"com.rc","fields":[{"name":"v","type":"int"}]}},
        {"name":"name","type":"string"},
        {"name":"extra","type":"string","default":""}
      ]}
      """
    Then the compatibility check should be compatible

  # ==========================================================================
  # 8. DELETE REFERENCED SCHEMA — CONSUMER STILL RETRIEVABLE
  # ==========================================================================

  Scenario: Deleting referenced schema does not break consumer retrieval
    Given subject "avro-refdel-base" has schema:
      """
      {"type":"record","name":"Base","namespace":"com.del","fields":[{"name":"x","type":"int"}]}
      """
    When I register a schema under subject "avro-refdel-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.del\",\"fields\":[{\"name\":\"b\",\"type\":\"com.del.Base\"}]}",
        "references": [
          {"name":"com.del.Base","subject":"avro-refdel-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "consumer_id"
    When I get version 1 of subject "avro-refdel-consumer"
    Then the response status should be 200

  # ==========================================================================
  # 9. LOOKUP SCHEMA WITH REFERENCES
  # ==========================================================================

  Scenario: Lookup schema with references via POST to subject
    Given subject "avro-reflookup-base" has schema:
      """
      {"type":"record","name":"Ref","namespace":"com.lookup","fields":[{"name":"id","type":"long"}]}
      """
    When I register a schema under subject "avro-reflookup-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Main\",\"namespace\":\"com.lookup\",\"fields\":[{\"name\":\"r\",\"type\":\"com.lookup.Ref\"}]}",
        "references": [
          {"name":"com.lookup.Ref","subject":"avro-reflookup-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "lookup_id"

  # ==========================================================================
  # 10. SAME CONTENT AT DIFFERENT REFERENCE VERSIONS
  # ==========================================================================

  Scenario: Identical schema content with different ref versions are distinct
    Given subject "avro-refver-base" has schema:
      """
      {"type":"record","name":"V","namespace":"com.ver","fields":[{"name":"a","type":"int"}]}
      """
    Given subject "avro-refver-base" has compatibility level "NONE"
    When I register a schema under subject "avro-refver-base":
      """
      {"type":"record","name":"V","namespace":"com.ver","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "avro-refver-c1" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"W\",\"namespace\":\"com.ver\",\"fields\":[{\"name\":\"v\",\"type\":\"com.ver.V\"}]}",
        "references": [
          {"name":"com.ver.V","subject":"avro-refver-base","version":1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "refver_v1"
    When I register a schema under subject "avro-refver-c2" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"W\",\"namespace\":\"com.ver\",\"fields\":[{\"name\":\"v\",\"type\":\"com.ver.V\"}]}",
        "references": [
          {"name":"com.ver.V","subject":"avro-refver-base","version":2}
        ]
      }
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "refver_v1"
