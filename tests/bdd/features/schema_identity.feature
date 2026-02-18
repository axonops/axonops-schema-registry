@functional
Feature: Schema Identity — metadata and ruleSet do not affect global ID
  Metadata and ruleSet are envelope-level properties that enrich a schema registration
  but MUST NOT affect schema identity for global ID purposes. The global schema ID is
  determined solely by the schema text (content-addressed via fingerprint). However,
  different metadata or ruleSet values DO create new versions within a subject, because
  each version is a distinct registration record.

  Key invariants:
    - Same schema text + different metadata = new version, SAME global ID
    - Same schema text + same metadata = dedup (returns existing, no new version)
    - Same schema text + different ruleSet = new version, SAME global ID
    - Different schema text = new version, potentially new global ID

  Background:
    Given the schema registry is running

  # ==========================================================================
  # 1. SAME SCHEMA, NO METADATA, REGISTERED TWICE = DEDUP (SAME VERSION)
  # ==========================================================================

  Scenario: Same schema without metadata registered twice is deduplicated
    When I POST "/subjects/identity-dedup-plain/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"PlainDedup\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    # Register the exact same schema again — should be idempotent
    When I POST "/subjects/identity-dedup-plain/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"PlainDedup\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"
    # Only one version should exist
    When I GET "/subjects/identity-dedup-plain/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # 2. SAME SCHEMA + DIFFERENT METADATA = SAME ID, NEW VERSION
  # ==========================================================================

  Scenario: Same schema with different metadata gets same global ID but new version
    # Register schema without metadata (v1)
    When I POST "/subjects/identity-meta-diff/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaDiff\",\"fields\":[{\"name\":\"key\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "base_id"
    # Register same schema with metadata (v2) — same ID, new version
    When I POST "/subjects/identity-meta-diff/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaDiff\",\"fields\":[{\"name\":\"key\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {
            "owner": "team-platform",
            "domain": "events"
          }
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "base_id"
    # Should now have two versions
    When I GET "/subjects/identity-meta-diff/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    # v1 should have no metadata
    When I GET "/subjects/identity-meta-diff/versions/1"
    Then the response status should be 200
    And the response body should not contain "owner"
    # v2 should have metadata
    When I GET "/subjects/identity-meta-diff/versions/2"
    Then the response status should be 200
    And the response body should contain "team-platform"
    And the response body should contain "events"
    And the response field "id" should equal stored "base_id"

  # ==========================================================================
  # 3. SAME SCHEMA + SAME METADATA = DEDUP (SAME ID, SAME VERSION)
  # ==========================================================================

  Scenario: Same schema with identical metadata registered twice is deduplicated
    When I POST "/subjects/identity-meta-same/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaSame\",\"fields\":[{\"name\":\"val\",\"type\":\"long\"}]}",
        "metadata": {
          "properties": {
            "env": "production"
          }
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "meta_same_id"
    # Register the exact same schema with the exact same metadata
    When I POST "/subjects/identity-meta-same/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaSame\",\"fields\":[{\"name\":\"val\",\"type\":\"long\"}]}",
        "metadata": {
          "properties": {
            "env": "production"
          }
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "meta_same_id"
    # Only one version should exist — full dedup
    When I GET "/subjects/identity-meta-same/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # 4. SAME SCHEMA + DIFFERENT RULESET = SAME ID, NEW VERSION
  # ==========================================================================

  Scenario: Same schema with different ruleSet gets same global ID but new version
    # Register schema with ruleSet A (v1)
    When I POST "/subjects/identity-rules-diff/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RulesDiff\",\"fields\":[{\"name\":\"email\",\"type\":\"string\"}]}",
        "ruleSet": {
          "domainRules": [
            {
              "name": "validateEmail",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.email) > 0"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "rules_id"
    # Register same schema with ruleSet B (v2) — same ID, new version
    When I POST "/subjects/identity-rules-diff/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RulesDiff\",\"fields\":[{\"name\":\"email\",\"type\":\"string\"}]}",
        "ruleSet": {
          "domainRules": [
            {
              "name": "checkEmailFormat",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.email.matches('^[a-z]+@[a-z]+\\\\.[a-z]+$')"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "rules_id"
    # Two versions should exist
    When I GET "/subjects/identity-rules-diff/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    # v1 should have ruleSet A
    When I GET "/subjects/identity-rules-diff/versions/1"
    Then the response status should be 200
    And the response body should contain "validateEmail"
    # v2 should have ruleSet B
    When I GET "/subjects/identity-rules-diff/versions/2"
    Then the response status should be 200
    And the response body should contain "checkEmailFormat"
    And the response field "id" should equal stored "rules_id"

  # ==========================================================================
  # 5. SCHEMA WITH METADATA, THEN SAME SCHEMA WITHOUT = NEW VERSION
  # ==========================================================================

  Scenario: Schema without metadata inherits from previous version (dedup)
    # Register with metadata (v1)
    When I POST "/subjects/identity-meta-then-none/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaThenNone\",\"fields\":[{\"name\":\"code\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {
            "classification": "internal"
          },
          "sensitive": ["code"]
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "mtn_id"
    # Register same schema without metadata — Confluent behavior: metadata inherits
    # from previous version, so this matches v1 and deduplicates (same version returned)
    When I POST "/subjects/identity-meta-then-none/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaThenNone\",\"fields\":[{\"name\":\"code\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "mtn_id"
    # Only one version should exist (deduplication happened)
    When I GET "/subjects/identity-meta-then-none/versions"
    Then the response status should be 200
    And the response should be an array of length 1
    # v1 has metadata
    When I GET "/subjects/identity-meta-then-none/versions/1"
    Then the response status should be 200
    And the response body should contain "classification"
    And the response body should contain "internal"

  Scenario: Schema with explicitly different metadata creates new version with same ID
    # Register with metadata A (v1)
    When I POST "/subjects/identity-meta-explicit-diff/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaExplDiff\",\"fields\":[{\"name\":\"code\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"classification": "internal"}
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "med_id"
    # Register same schema with different metadata B (v2)
    When I POST "/subjects/identity-meta-explicit-diff/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaExplDiff\",\"fields\":[{\"name\":\"code\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"classification": "public"}
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "med_id"
    # Two versions should exist
    When I GET "/subjects/identity-meta-explicit-diff/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    # v1 has "internal"
    When I GET "/subjects/identity-meta-explicit-diff/versions/1"
    Then the response status should be 200
    And the response body should contain "internal"
    # v2 has "public"
    When I GET "/subjects/identity-meta-explicit-diff/versions/2"
    Then the response status should be 200
    And the response body should contain "public"
    And the response field "id" should equal stored "med_id"

  # ==========================================================================
  # 6. DIFFERENT SCHEMA TEXT = DIFFERENT VERSION (AND POTENTIALLY DIFFERENT ID)
  # ==========================================================================

  Scenario: Different schema text creates a new version with a different global ID
    Given the global compatibility level is "NONE"
    When I POST "/subjects/identity-diff-text/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"VersionA\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "text_id_a"
    # Register a genuinely different schema
    When I POST "/subjects/identity-diff-text/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"VersionB\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"},{\"name\":\"y\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "text_id_b"
    And the response field "id" should not equal stored "text_id_a"
    # Two versions
    When I GET "/subjects/identity-diff-text/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # 7. MULTIPLE METADATA CHANGES CREATE SEQUENTIAL VERSIONS, SAME ID
  # ==========================================================================

  Scenario: Multiple metadata changes create sequential versions all sharing the same global ID
    # v1: no metadata
    When I POST "/subjects/identity-meta-seq/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaSeq\",\"fields\":[{\"name\":\"ts\",\"type\":\"long\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "seq_id"
    # v2: metadata A
    When I POST "/subjects/identity-meta-seq/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaSeq\",\"fields\":[{\"name\":\"ts\",\"type\":\"long\"}]}",
        "metadata": {
          "properties": {
            "owner": "alpha-team"
          }
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "seq_id"
    # v3: metadata B (different from A)
    When I POST "/subjects/identity-meta-seq/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaSeq\",\"fields\":[{\"name\":\"ts\",\"type\":\"long\"}]}",
        "metadata": {
          "properties": {
            "owner": "beta-team",
            "priority": "high"
          }
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "seq_id"
    # All three versions exist
    When I GET "/subjects/identity-meta-seq/versions"
    Then the response status should be 200
    And the response should be an array of length 3
    # Verify each version's content
    When I GET "/subjects/identity-meta-seq/versions/1"
    Then the response status should be 200
    And the response body should not contain "owner"
    When I GET "/subjects/identity-meta-seq/versions/2"
    Then the response status should be 200
    And the response body should contain "alpha-team"
    When I GET "/subjects/identity-meta-seq/versions/3"
    Then the response status should be 200
    And the response body should contain "beta-team"
    And the response body should contain "priority"

  # ==========================================================================
  # 8. VERIFY VERSION COUNT AFTER METADATA-ONLY CHANGES
  # ==========================================================================

  Scenario: Version count reflects metadata-only changes accurately
    # Register v1 with metadata
    When I POST "/subjects/identity-version-count/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"VerCount\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"revision": "1"}
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "vc_id"
    # Register v2 with different metadata
    When I POST "/subjects/identity-version-count/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"VerCount\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"revision": "2"}
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "vc_id"
    # Register v3 with yet another metadata
    When I POST "/subjects/identity-version-count/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"VerCount\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"revision": "3", "approved": "true"}
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "vc_id"
    # Register v4 with different metadata again
    When I POST "/subjects/identity-version-count/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"VerCount\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"revision": "4"}
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "vc_id"
    # List versions — should be exactly 4
    When I GET "/subjects/identity-version-count/versions"
    Then the response status should be 200
    And the response should be an array of length 4
    # Verify specific versions exist with correct numbering
    When I GET "/subjects/identity-version-count/versions/1"
    Then the response status should be 200
    And the response field "version" should be 1
    When I GET "/subjects/identity-version-count/versions/4"
    Then the response status should be 200
    And the response field "version" should be 4
    And the response field "id" should equal stored "vc_id"

  # ==========================================================================
  # 9. SCHEMA LOOKUP MATCHES BY CONTENT, NOT METADATA
  # ==========================================================================

  Scenario: Schema lookup matches by schema content regardless of metadata
    # Register schema with metadata
    When I POST "/subjects/identity-lookup/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"LookupTest\",\"fields\":[{\"name\":\"item\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {
            "source": "kafka-connect",
            "format": "debezium"
          },
          "tags": {
            "item": ["IDENTIFIER"]
          }
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "lookup_id"
    # Lookup the same schema WITHOUT metadata — should still find it
    When I POST "/subjects/identity-lookup" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"LookupTest\",\"fields\":[{\"name\":\"item\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And the response should have field "subject"
    And the response field "subject" should be "identity-lookup"
    And the response field "id" should equal stored "lookup_id"

  # ==========================================================================
  # 10. CROSS-SUBJECT SAME SCHEMA = SAME GLOBAL ID
  # ==========================================================================

  Scenario: Same schema registered in two different subjects gets the same global ID
    When I POST "/subjects/identity-cross-a/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CrossSubject\",\"fields\":[{\"name\":\"region\",\"type\":\"string\"},{\"name\":\"count\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "cross_id_a"
    When I POST "/subjects/identity-cross-b/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CrossSubject\",\"fields\":[{\"name\":\"region\",\"type\":\"string\"},{\"name\":\"count\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "cross_id_b"
    And the response field "id" should equal stored "cross_id_a"
    # Both subjects should be listed under the same schema ID
    When I get the subjects for schema ID {{cross_id_a}}
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "identity-cross-a"
    And the response array should contain "identity-cross-b"

  # ==========================================================================
  # 11. CROSS-SUBJECT SAME SCHEMA WITH DIFFERENT METADATA = SAME ID
  # ==========================================================================

  Scenario: Same schema in different subjects with different metadata shares the same global ID
    # Subject A: schema with metadata
    When I POST "/subjects/identity-cross-meta-a/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CrossMeta\",\"fields\":[{\"name\":\"sensor\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"region": "us-east-1"}
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "cross_meta_id"
    # Subject B: same schema with different metadata
    When I POST "/subjects/identity-cross-meta-b/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CrossMeta\",\"fields\":[{\"name\":\"sensor\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"region": "eu-west-1"}
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "cross_meta_id"
    # Both subjects share the same schema ID
    When I get the subjects for schema ID {{cross_meta_id}}
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "identity-cross-meta-a"
    And the response array should contain "identity-cross-meta-b"

  # ==========================================================================
  # 12. METADATA + RULESET COMBINED CHANGE = SAME ID, NEW VERSION
  # ==========================================================================

  Scenario: Same schema with combined metadata and ruleSet changes gets same ID, new version
    # v1: schema with metadata only
    When I POST "/subjects/identity-combo/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"ComboTest\",\"fields\":[{\"name\":\"payload\",\"type\":\"bytes\"}]}",
        "metadata": {
          "properties": {"tier": "standard"}
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "combo_id"
    # v2: same schema with metadata + ruleSet
    When I POST "/subjects/identity-combo/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"ComboTest\",\"fields\":[{\"name\":\"payload\",\"type\":\"bytes\"}]}",
        "metadata": {
          "properties": {"tier": "premium"}
        },
        "ruleSet": {
          "domainRules": [
            {
              "name": "sizeCheck",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.payload) < 1048576"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "combo_id"
    # Two versions
    When I GET "/subjects/identity-combo/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    # v1 has metadata but no ruleSet
    When I GET "/subjects/identity-combo/versions/1"
    Then the response status should be 200
    And the response body should contain "standard"
    And the response body should not contain "sizeCheck"
    # v2 has both metadata and ruleSet
    When I GET "/subjects/identity-combo/versions/2"
    Then the response status should be 200
    And the response body should contain "premium"
    And the response body should contain "sizeCheck"
    And the response field "id" should equal stored "combo_id"

  # ==========================================================================
  # 13. SAME RULESET REGISTERED TWICE = DEDUP
  # ==========================================================================

  Scenario: Same schema with identical ruleSet registered twice is deduplicated
    When I POST "/subjects/identity-rules-same/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RulesSame\",\"fields\":[{\"name\":\"msg\",\"type\":\"string\"}]}",
        "ruleSet": {
          "domainRules": [
            {
              "name": "nonEmpty",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.msg) > 0"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "rules_same_id"
    # Register the exact same schema with the exact same ruleSet
    When I POST "/subjects/identity-rules-same/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RulesSame\",\"fields\":[{\"name\":\"msg\",\"type\":\"string\"}]}",
        "ruleSet": {
          "domainRules": [
            {
              "name": "nonEmpty",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.msg) > 0"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "rules_same_id"
    # Only one version should exist — full dedup
    When I GET "/subjects/identity-rules-same/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # 14. LATEST VERSION REFLECTS MOST RECENT METADATA
  # ==========================================================================

  Scenario: Latest version endpoint returns most recent metadata registration
    # v1: no metadata
    When I POST "/subjects/identity-latest/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"LatestTest\",\"fields\":[{\"name\":\"flag\",\"type\":\"boolean\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "latest_id"
    # v2: with metadata
    When I POST "/subjects/identity-latest/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"LatestTest\",\"fields\":[{\"name\":\"flag\",\"type\":\"boolean\"}]}",
        "metadata": {
          "properties": {"status": "approved"}
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "latest_id"
    # Latest version should be v2 with the metadata
    When I GET "/subjects/identity-latest/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 2
    And the response body should contain "approved"
    And the response field "id" should equal stored "latest_id"
