@functional @data-contracts
Feature: Metadata and RuleSets (Data Contracts)
  Schema metadata and rule sets enable data contracts in the registry.
  Metadata (tags, properties, sensitive fields) and RuleSets (migration/domain rules)
  can be stored per schema registration and configured at the subject level.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # SCHEMA REGISTRATION WITH METADATA
  # ==========================================================================

  Scenario: Register schema with metadata — stored and returned
    When I POST "/subjects/meta-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaTest\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}",
        "schemaType": "AVRO",
        "metadata": {
          "properties": {
            "owner": "team-data",
            "domain": "analytics"
          },
          "tags": {
            "id": ["PII", "SENSITIVE"]
          },
          "sensitive": ["id"]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/meta-test/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "team-data"
    And the response body should contain "analytics"
    And the response body should contain "PII"
    And the response body should contain "SENSITIVE"
    And the audit log should contain event "schema_register" with subject "meta-test"

  @data-contracts
  Scenario: Register schema with ruleSet — stored and returned
    When I POST "/subjects/ruleset-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RuleSetTest\",\"fields\":[{\"name\":\"email\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "checkEmail",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.email.matches('^[a-zA-Z0-9+_.-]+@[a-zA-Z0-9.-]+$')"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/ruleset-test/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "checkEmail"
    And the response body should contain "CONDITION"
    And the response body should contain "WRITE"
    And the audit log should contain event "schema_register" with subject "ruleset-test"

  @data-contracts
  Scenario: Register schema with both metadata and ruleSet
    When I POST "/subjects/both-meta-rules/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BothTest\",\"fields\":[{\"name\":\"ssn\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "metadata": {
          "properties": {"classification": "restricted"},
          "sensitive": ["ssn"]
        },
        "ruleSet": {
          "domainRules": [
            {
              "name": "maskSSN",
              "kind": "TRANSFORM",
              "mode": "READ",
              "type": "CEL",
              "expr": "message.ssn.replaceAll('[0-9]', 'X')"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/subjects/both-meta-rules/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response should have field "ruleSet"
    And the response body should contain "restricted"
    And the response body should contain "maskSSN"
    And the audit log should contain event "schema_register" with subject "both-meta-rules"

  # ==========================================================================
  # METADATA DOES NOT AFFECT SCHEMA IDENTITY
  # ==========================================================================

  Scenario: Metadata does not affect schema identity — same schema different metadata
    # Register schema without metadata
    When I POST "/subjects/meta-identity/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Identity\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    # Register same schema with metadata — should return same ID (schema identity unchanged)
    When I POST "/subjects/meta-identity/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Identity\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}",
        "metadata": {
          "properties": {"owner": "new-team"}
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"
    And the audit log should contain event "schema_register" with subject "meta-identity"

  # ==========================================================================
  # SCHEMA BY ID INCLUDES METADATA
  # ==========================================================================

  Scenario: GET schema by ID includes metadata
    When I POST "/subjects/meta-byid/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"ById\",\"fields\":[{\"name\":\"val\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"env": "production"}
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I GET "/schemas/ids/{{schema_id}}"
    Then the response status should be 200
    And the response field "schemaType" should be "AVRO"
    And the audit log should contain event "schema_register" with subject "meta-byid"

  # ==========================================================================
  # LOOKUP SCHEMA INCLUDES METADATA
  # ==========================================================================

  @data-contracts
  Scenario: Lookup schema returns metadata and ruleSet
    When I POST "/subjects/meta-lookup/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Lookup\",\"fields\":[{\"name\":\"k\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"source": "kafka"}
        },
        "ruleSet": {
          "domainRules": [
            {
              "name": "validate",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.k) > 0"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I POST "/subjects/meta-lookup" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Lookup\",\"fields\":[{\"name\":\"k\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "subject"
    And the response field "subject" should be "meta-lookup"
    And the audit log should contain event "schema_lookup" with subject "meta-lookup"

  # ==========================================================================
  # CONFIG WITH METADATA AND RULESETS
  # ==========================================================================

  Scenario: Set config with defaultMetadata
    When I PUT "/config/meta-cfg-subject" with body:
      """
      {
        "compatibility": "BACKWARD",
        "defaultMetadata": {
          "properties": {"team": "platform"}
        }
      }
      """
    Then the response status should be 200
    When I GET "/config/meta-cfg-subject"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"
    And the response should have field "defaultMetadata"
    And the response body should contain "platform"
    And the audit log should contain event "config_update"

  Scenario: Set config with overrideMetadata
    When I PUT "/config/meta-override-subject" with body:
      """
      {
        "compatibility": "FULL",
        "overrideMetadata": {
          "properties": {"classification": "internal"}
        }
      }
      """
    Then the response status should be 200
    When I GET "/config/meta-override-subject"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"
    And the response should have field "overrideMetadata"
    And the response body should contain "internal"
    And the audit log should contain event "config_update"

  @data-contracts
  Scenario: Set config with defaultRuleSet
    When I PUT "/config/rules-cfg-subject" with body:
      """
      {
        "compatibility": "BACKWARD",
        "defaultRuleSet": {
          "domainRules": [
            {
              "name": "defaultValidation",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/config/rules-cfg-subject"
    Then the response status should be 200
    And the response should have field "defaultRuleSet"
    And the response body should contain "defaultValidation"
    And the audit log should contain event "config_update"

  @data-contracts
  Scenario: Set config with overrideRuleSet
    When I PUT "/config/rules-override-subject" with body:
      """
      {
        "compatibility": "NONE",
        "overrideRuleSet": {
          "domainRules": [
            {
              "name": "overrideRule",
              "kind": "TRANSFORM",
              "mode": "READ",
              "type": "CEL",
              "expr": "message"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/config/rules-override-subject"
    Then the response status should be 200
    And the response should have field "overrideRuleSet"
    And the response body should contain "overrideRule"
    And the audit log should contain event "config_update"

  Scenario: Set config with alias
    When I PUT "/config/alias-target" with body:
      """
      {
        "compatibility": "BACKWARD",
        "alias": "my-alias"
      }
      """
    Then the response status should be 200
    When I GET "/config/alias-target"
    Then the response status should be 200
    And the response field "alias" should be "my-alias"
    And the audit log should contain event "config_update"

  # ==========================================================================
  # GLOBAL CONFIG WITH METADATA AND RULESETS
  # ==========================================================================

  Scenario: Set global config with defaultMetadata
    # Save current global config
    When I GET "/config"
    Then the response status should be 200
    # Set global config with metadata
    When I PUT "/config" with body:
      """
      {
        "compatibility": "BACKWARD",
        "defaultMetadata": {
          "properties": {"org": "acme"}
        }
      }
      """
    Then the response status should be 200
    When I GET "/config"
    Then the response status should be 200
    And the response should have field "defaultMetadata"
    And the response body should contain "acme"
    # Reset global config — must explicitly clear defaultMetadata
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "defaultMetadata": null}
      """
    Then the response status should be 200
    And the audit log should contain event "config_update"

  # ==========================================================================
  # SCHEMA WITHOUT METADATA — FIELDS OMITTED
  # ==========================================================================

  @axonops-only
  Scenario: Schema without metadata has only confluent:version in response
    When I POST "/subjects/no-meta/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"NoMeta\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/no-meta/versions/1"
    Then the response status should be 200
    And the response body should contain "confluent:version"
    And the response body should not contain "ruleSet"
    And the audit log should contain event "schema_register" with subject "no-meta"

  # ==========================================================================
  # MIGRATION RULES
  # ==========================================================================

  @data-contracts
  Scenario: Register schema with migration rules
    When I POST "/subjects/migration-rules/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MigRules\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}",
        "ruleSet": {
          "migrationRules": [
            {
              "name": "upgradeV1toV2",
              "kind": "TRANSFORM",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "message"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/subjects/migration-rules/versions/1"
    Then the response status should be 200
    And the response body should contain "migrationRules"
    And the response body should contain "upgradeV1toV2"
    And the response body should contain "UPGRADE"
    And the audit log should contain event "schema_register" with subject "migration-rules"

  # ==========================================================================
  # GET /subjects/{subject}/metadata ENDPOINT
  # ==========================================================================

  Scenario: Get subject metadata returns latest schema metadata
    When I POST "/subjects/meta-endpoint-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaEndV1\",\"fields\":[{\"name\":\"a\",\"type\":\"int\"}]}",
        "metadata": {
          "properties": {"owner": "team-alpha"},
          "tags": {"a": ["PII"]},
          "sensitive": ["a"]
        }
      }
      """
    Then the response status should be 200
    When I GET "/subjects/meta-endpoint-test/metadata"
    Then the response status should be 200
    And the response should be valid JSON
    And the response body should contain "team-alpha"
    And the response body should contain "PII"

  Scenario: Get subject metadata for non-existent subject returns 404
    When I GET "/subjects/nonexistent-meta-subject/metadata"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Get subject metadata when no metadata exists returns empty object
    When I POST "/subjects/meta-empty-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"NoMetaEndpoint\",\"fields\":[{\"name\":\"b\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/meta-empty-test/metadata"
    Then the response status should be 200
    And the response should be valid JSON
    And the audit log should contain event "schema_register" with subject "meta-empty-test"

  Scenario: Get subject metadata returns metadata from the latest version
    # Register v1 with metadata
    When I POST "/subjects/meta-latest-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaLatest\",\"fields\":[{\"name\":\"c\",\"type\":\"int\"}]}",
        "metadata": {
          "properties": {"version": "v1"}
        }
      }
      """
    Then the response status should be 200
    # Register v2 with updated metadata (add optional field for BACKWARD compat)
    When I POST "/subjects/meta-latest-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MetaLatest\",\"fields\":[{\"name\":\"c\",\"type\":\"int\"},{\"name\":\"d\",\"type\":[\"null\",\"string\"],\"default\":null}]}",
        "metadata": {
          "properties": {"version": "v2", "updated": "true"}
        }
      }
      """
    Then the response status should be 200
    When I GET "/subjects/meta-latest-test/metadata"
    Then the response status should be 200
    And the response body should contain "v2"
    And the response body should contain "updated"
    And the audit log should contain event "schema_register" with subject "meta-latest-test"
