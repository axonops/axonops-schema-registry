@functional
Feature: 3-layer metadata/ruleSet merge during schema registration
  The schema registry implements a 3-layer merge for metadata and ruleSets
  during schema registration:
    final = merge(merge(config.default, specific_request), config.override)

  When a registration request does not specify metadata or ruleSet, the values
  are inherited from the previous version of that subject.

  Override config always wins on conflicting keys.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # DEFAULT METADATA MERGE
  # ==========================================================================

  Scenario: Default metadata applied when registration has no metadata
    # Set subject config with defaultMetadata
    When I PUT "/config/cfg-merge-default" with body:
      """
      {
        "compatibility": "NONE",
        "defaultMetadata": {
          "properties": {"team": "platform", "env": "prod"}
        }
      }
      """
    Then the response status should be 200
    # Register schema without metadata
    When I POST "/subjects/cfg-merge-default/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CfgMerge1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # Verify default metadata was applied
    When I GET "/subjects/cfg-merge-default/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "platform"
    And the response body should contain "prod"

  # ==========================================================================
  # OVERRIDE METADATA MERGE
  # ==========================================================================

  Scenario: Override metadata merged with request metadata — override wins on conflict
    # Set subject config with overrideMetadata
    When I PUT "/config/cfg-merge-override" with body:
      """
      {
        "compatibility": "NONE",
        "overrideMetadata": {
          "properties": {"classification": "internal", "team": "security"}
        }
      }
      """
    Then the response status should be 200
    # Register schema WITH metadata that has a conflicting key ("team")
    When I POST "/subjects/cfg-merge-override/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CfgMerge2\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"team": "data-eng", "domain": "analytics"}
        }
      }
      """
    Then the response status should be 200
    # Verify override won on "team" and both non-conflicting keys present
    When I GET "/subjects/cfg-merge-override/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "security"
    And the response body should contain "internal"
    And the response body should contain "analytics"
    # The override "team":"security" should have replaced the request "team":"data-eng"
    And the response body should not contain "data-eng"

  # ==========================================================================
  # FULL 3-LAYER MERGE
  # ==========================================================================

  Scenario: 3-layer merge — default + specific + override all merged correctly
    # Set subject config with both default and override metadata
    When I PUT "/config/cfg-merge-3layer" with body:
      """
      {
        "compatibility": "NONE",
        "defaultMetadata": {
          "properties": {"source": "default-layer", "tier": "bronze"}
        },
        "overrideMetadata": {
          "properties": {"tier": "gold", "approved": "yes"}
        }
      }
      """
    Then the response status should be 200
    # Register schema with specific metadata
    When I POST "/subjects/cfg-merge-3layer/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CfgMerge3\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"owner": "team-alpha", "tier": "silver"}
        }
      }
      """
    Then the response status should be 200
    # Verify: default "source" present, specific "owner" present,
    # override "approved" present, override "tier":"gold" wins over both default and specific
    When I GET "/subjects/cfg-merge-3layer/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "default-layer"
    And the response body should contain "team-alpha"
    And the response body should contain "approved"
    And the response body should contain "gold"
    # "bronze" and "silver" should be overridden by "gold"
    And the response body should not contain "bronze"
    And the response body should not contain "silver"

  # ==========================================================================
  # DEFAULT RULESET MERGE
  # ==========================================================================

  Scenario: Default ruleSet applied when registration has no ruleSet
    # Set subject config with defaultRuleSet
    When I PUT "/config/cfg-merge-defrule" with body:
      """
      {
        "compatibility": "NONE",
        "defaultRuleSet": {
          "domainRules": [
            {
              "name": "defaultCheck",
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
    # Register schema without ruleSet
    When I POST "/subjects/cfg-merge-defrule/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CfgMerge4\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # Verify default ruleSet was applied
    When I GET "/subjects/cfg-merge-defrule/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "defaultCheck"
    And the response body should contain "CONDITION"

  # ==========================================================================
  # OVERRIDE RULESET MERGE
  # ==========================================================================

  Scenario: Override ruleSet merged with request ruleSet
    # Set subject config with overrideRuleSet
    When I PUT "/config/cfg-merge-ovrrule" with body:
      """
      {
        "compatibility": "NONE",
        "overrideRuleSet": {
          "domainRules": [
            {
              "name": "enforcePolicy",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.a != ''"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    # Register schema WITH its own ruleSet
    When I POST "/subjects/cfg-merge-ovrrule/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CfgMerge5\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "ruleSet": {
          "domainRules": [
            {
              "name": "userValidation",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.a) > 0"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    # Verify both rules present in merged result
    When I GET "/subjects/cfg-merge-ovrrule/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "enforcePolicy"
    And the response body should contain "userValidation"

  # ==========================================================================
  # METADATA INHERITANCE FROM PREVIOUS VERSION
  # ==========================================================================

  Scenario: Metadata inherited from previous version when not specified in request
    # Register v1 with metadata
    When I PUT "/config/cfg-merge-inherit" with body:
      """
      {
        "compatibility": "NONE"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/cfg-merge-inherit/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Inherit1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"owner": "team-data", "env": "staging"}
        }
      }
      """
    Then the response status should be 200
    # Register v2 without metadata — should inherit from v1
    When I POST "/subjects/cfg-merge-inherit/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Inherit2\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"
      }
      """
    Then the response status should be 200
    # Verify v2 inherited v1's metadata
    When I GET "/subjects/cfg-merge-inherit/versions/2"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "team-data"
    And the response body should contain "staging"

  # ==========================================================================
  # RULESET INHERITANCE FROM PREVIOUS VERSION
  # ==========================================================================

  Scenario: RuleSet inherited from previous version when not specified in request
    # Register v1 with ruleSet
    When I PUT "/config/cfg-merge-ruleinh" with body:
      """
      {
        "compatibility": "NONE"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/cfg-merge-ruleinh/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RuleInh1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "ruleSet": {
          "domainRules": [
            {
              "name": "inheritedRule",
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
    # Register v2 without ruleSet — should inherit from v1
    When I POST "/subjects/cfg-merge-ruleinh/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RuleInh2\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"
      }
      """
    Then the response status should be 200
    # Verify v2 inherited v1's ruleSet
    When I GET "/subjects/cfg-merge-ruleinh/versions/2"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "inheritedRule"

  # ==========================================================================
  # OVERRIDE WINS OVER SPECIFIC
  # ==========================================================================

  Scenario: Override metadata wins over request-specific metadata on same key
    # Set subject config with override that has key "owner"
    When I PUT "/config/cfg-merge-ovrwins" with body:
      """
      {
        "compatibility": "NONE",
        "overrideMetadata": {
          "properties": {"owner": "governance-team"}
        }
      }
      """
    Then the response status should be 200
    # Register schema with specific metadata that also has key "owner"
    When I POST "/subjects/cfg-merge-ovrwins/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"OvrWins\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"owner": "dev-team", "project": "alpha"}
        }
      }
      """
    Then the response status should be 200
    # Verify override "owner" won over specific "owner", and "project" is still present
    When I GET "/subjects/cfg-merge-ovrwins/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "governance-team"
    And the response body should contain "alpha"
    And the response body should not contain "dev-team"

  # ==========================================================================
  # DEFAULT + SPECIFIC MERGE (NO CONFLICT)
  # ==========================================================================

  Scenario: Default and specific metadata merged when keys do not conflict
    # Set default with key A
    When I PUT "/config/cfg-merge-defspec" with body:
      """
      {
        "compatibility": "NONE",
        "defaultMetadata": {
          "properties": {"region": "us-east-1"}
        }
      }
      """
    Then the response status should be 200
    # Register with specific key B
    When I POST "/subjects/cfg-merge-defspec/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"DefSpec\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"service": "payments"}
        }
      }
      """
    Then the response status should be 200
    # Verify both present
    When I GET "/subjects/cfg-merge-defspec/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "us-east-1"
    And the response body should contain "payments"

  # ==========================================================================
  # OVERRIDE + SPECIFIC MERGE (NO CONFLICT)
  # ==========================================================================

  Scenario: Override and specific metadata merged when keys do not conflict
    # Set override with key A
    When I PUT "/config/cfg-merge-ovrspec" with body:
      """
      {
        "compatibility": "NONE",
        "overrideMetadata": {
          "properties": {"compliance": "sox"}
        }
      }
      """
    Then the response status should be 200
    # Register with specific key B
    When I POST "/subjects/cfg-merge-ovrspec/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"OvrSpec\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"department": "engineering"}
        }
      }
      """
    Then the response status should be 200
    # Verify both present
    When I GET "/subjects/cfg-merge-ovrspec/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "sox"
    And the response body should contain "engineering"

  # ==========================================================================
  # NO CONFIG, NO METADATA — SCHEMA HAS NO METADATA
  # ==========================================================================

  Scenario: No config and no metadata — response has only confluent:version in metadata
    # Register schema with no metadata, no config set for subject
    # Confluent behavior: confluent:version is always auto-populated
    When I POST "/subjects/cfg-merge-nometa/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"NoMeta\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/cfg-merge-nometa/versions/1"
    Then the response status should be 200
    And the response body should contain "confluent:version"
    And the response body should not contain "ruleSet"

  # ==========================================================================
  # GLOBAL CONFIG DEFAULTS APPLIED
  # ==========================================================================

  Scenario: Global config defaultMetadata applied to any subject
    # Set global config with defaultMetadata
    When I PUT "/config" with body:
      """
      {
        "compatibility": "NONE",
        "defaultMetadata": {
          "properties": {"org": "acme-corp", "global-tag": "yes"}
        }
      }
      """
    Then the response status should be 200
    # Register schema under a subject with no subject-level config
    When I POST "/subjects/cfg-merge-global/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"GlobalDef\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # Verify global default metadata was applied
    When I GET "/subjects/cfg-merge-global/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "acme-corp"
    And the response body should contain "global-tag"
    # Clean up global config to not affect other tests
    When I PUT "/config" with body:
      """
      {
        "compatibility": "BACKWARD",
        "defaultMetadata": null
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # METADATA TAGS AND SENSITIVE MERGE
  # ==========================================================================

  Scenario: Default and specific metadata merge tags and sensitive lists
    # Set default with tags and sensitive
    When I PUT "/config/cfg-merge-tags" with body:
      """
      {
        "compatibility": "NONE",
        "defaultMetadata": {
          "tags": {"a": ["PII"]},
          "sensitive": ["ssn"]
        }
      }
      """
    Then the response status should be 200
    # Register with additional tags and sensitive
    When I POST "/subjects/cfg-merge-tags/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"TagMerge\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}",
        "metadata": {
          "tags": {"email": ["CONTACT"]},
          "sensitive": ["email"]
        }
      }
      """
    Then the response status should be 200
    # Verify both tags and both sensitive fields present
    When I GET "/subjects/cfg-merge-tags/versions/1"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "PII"
    And the response body should contain "CONTACT"
    And the response body should contain "ssn"
    And the response body should contain "email"

  # ==========================================================================
  # INHERITED METADATA ALSO GETS MERGED WITH CONFIG
  # ==========================================================================

  Scenario: Inherited metadata from previous version is merged with config override
    # Set subject config with override
    When I PUT "/config/cfg-merge-inhcfg" with body:
      """
      {
        "compatibility": "NONE",
        "overrideMetadata": {
          "properties": {"status": "approved"}
        }
      }
      """
    Then the response status should be 200
    # Register v1 with metadata
    When I POST "/subjects/cfg-merge-inhcfg/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"InhCfg1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"owner": "team-x"}
        }
      }
      """
    Then the response status should be 200
    # Verify v1 has both inherited default and override
    When I GET "/subjects/cfg-merge-inhcfg/versions/1"
    Then the response status should be 200
    And the response body should contain "team-x"
    And the response body should contain "approved"
    # Register v2 without metadata — inherits v1 metadata, then override re-applied
    When I POST "/subjects/cfg-merge-inhcfg/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"InhCfg2\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"
      }
      """
    Then the response status should be 200
    # Verify v2 has inherited "owner" and override "status"
    When I GET "/subjects/cfg-merge-inhcfg/versions/2"
    Then the response status should be 200
    And the response should have field "metadata"
    And the response body should contain "team-x"
    And the response body should contain "approved"

  # ==========================================================================
  # OVERRIDE RULESET REPLACES SAME-NAME RULES
  # ==========================================================================

  Scenario: Override ruleSet replaces rules with same name from request
    # Set override ruleSet with a rule named "validate"
    When I PUT "/config/cfg-merge-ruleovr" with body:
      """
      {
        "compatibility": "NONE",
        "overrideRuleSet": {
          "domainRules": [
            {
              "name": "validate",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.a != 'BLOCKED'"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    # Register with a rule also named "validate" but different expr
    When I POST "/subjects/cfg-merge-ruleovr/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RuleOvr\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "ruleSet": {
          "domainRules": [
            {
              "name": "validate",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.a) > 0"
            },
            {
              "name": "extraCheck",
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
    # Verify override replaced the same-name rule and extra rule is still present
    When I GET "/subjects/cfg-merge-ruleovr/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "BLOCKED"
    And the response body should contain "extraCheck"
    # The specific expr should be replaced by override expr
    And the response body should not contain "size(message.a)"

  # ==========================================================================
  # 3-LAYER MERGE WITH RULESET
  # ==========================================================================

  Scenario: 3-layer ruleSet merge — default + specific + override
    # Set default and override ruleSets
    When I PUT "/config/cfg-merge-3rule" with body:
      """
      {
        "compatibility": "NONE",
        "defaultRuleSet": {
          "domainRules": [
            {
              "name": "baseRule",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        },
        "overrideRuleSet": {
          "domainRules": [
            {
              "name": "enforcedRule",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.a != 'FORBIDDEN'"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    # Register with specific ruleSet
    When I POST "/subjects/cfg-merge-3rule/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Rule3Layer\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "ruleSet": {
          "domainRules": [
            {
              "name": "specificRule",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.a) < 100"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    # Verify all three rules present after merge
    When I GET "/subjects/cfg-merge-3rule/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "baseRule"
    And the response body should contain "specificRule"
    And the response body should contain "enforcedRule"
