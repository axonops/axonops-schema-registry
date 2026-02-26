@functional @data-contracts
Feature: RuleSet Validation
  Server-side validation of ruleSet structure in schema registration and config operations.
  Invalid rule definitions are rejected at registration time with error code 42201.

  Background:
    Given the schema registry is running
    And the global compatibility level is "NONE"

  # ==========================================================================
  # VALID RULESETS
  # ==========================================================================

  Scenario: Valid ruleSet on schema registration succeeds
    When I POST "/subjects/valid-ruleset/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"ValidRuleSet\",\"fields\":[{\"name\":\"email\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "validateEmail",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.email.contains('@')"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/valid-ruleset/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "validateEmail"
    And the response body should contain "CONDITION"

  Scenario: Valid migration rule with UPGRADE mode succeeds
    When I POST "/subjects/migration-upgrade/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MigrationUp\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "migrationRules": [
            {
              "name": "upgradeRule",
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
    And the response should have field "id"
    When I GET "/subjects/migration-upgrade/versions/1"
    Then the response status should be 200
    And the response body should contain "migrationRules"
    And the response body should contain "UPGRADE"

  Scenario: Valid onSuccess and onFailure actions succeed
    When I POST "/subjects/on-actions/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"OnActions\",\"fields\":[{\"name\":\"val\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "ruleWithActions",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.val) > 0",
              "onSuccess": "DLQ",
              "onFailure": "ERROR"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/subjects/on-actions/versions/1"
    Then the response status should be 200
    And the response body should contain "DLQ"
    And the response body should contain "ERROR"

  Scenario: Empty ruleSet (no rules) succeeds
    When I POST "/subjects/empty-ruleset/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EmptyRuleSet\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": []
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  Scenario: Multiple valid rules in domainRules succeeds
    When I POST "/subjects/multi-rules/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MultiRules\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"email\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "rule1",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.id > 0"
            },
            {
              "name": "rule2",
              "kind": "TRANSFORM",
              "mode": "READ",
              "type": "CEL",
              "expr": "message"
            },
            {
              "name": "rule3",
              "kind": "CONDITION",
              "mode": "WRITEREAD",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/subjects/multi-rules/versions/1"
    Then the response status should be 200
    And the response body should contain "rule1"
    And the response body should contain "rule2"
    And the response body should contain "rule3"

  Scenario: Valid ruleSet in config PUT /config succeeds
    When I PUT "/config" with body:
      """
      {
        "compatibility": "BACKWARD",
        "defaultRuleSet": {
          "domainRules": [
            {
              "name": "globalRule",
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
    When I GET "/config"
    Then the response status should be 200
    And the response should have field "defaultRuleSet"
    And the response body should contain "globalRule"

  Scenario: Valid ruleSet in config PUT /config/{subject} succeeds
    When I PUT "/config/subject-rules" with body:
      """
      {
        "compatibility": "NONE",
        "overrideRuleSet": {
          "domainRules": [
            {
              "name": "subjectRule",
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
    When I GET "/config/subject-rules"
    Then the response status should be 200
    And the response should have field "overrideRuleSet"
    And the response body should contain "subjectRule"

  # ==========================================================================
  # INVALID RULESETS — MISSING FIELDS
  # ==========================================================================

  Scenario: Missing rule name returns 422 with error 42201
    When I POST "/subjects/missing-name/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MissingName\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Empty rule name returns 422 with error 42201
    When I POST "/subjects/empty-name/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EmptyName\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  # ==========================================================================
  # INVALID RULESETS — INVALID KIND
  # ==========================================================================

  Scenario: Invalid rule kind returns 422 with error 42201
    When I POST "/subjects/invalid-kind/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"InvalidKind\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "badKind",
              "kind": "INVALID",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Rule kind accepts only CONDITION or TRANSFORM
    When I POST "/subjects/bad-kind-verify/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadKind\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "badKindRule",
              "kind": "FILTER",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  # ==========================================================================
  # INVALID RULESETS — INVALID MODE FOR DOMAIN RULES
  # ==========================================================================

  Scenario: Invalid mode for domain rule returns 422
    When I POST "/subjects/invalid-domain-mode/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"InvalidDomainMode\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "badMode",
              "kind": "CONDITION",
              "mode": "INVALID",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Domain rule mode must be WRITE, READ, or WRITEREAD
    When I POST "/subjects/bad-domain-mode/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadDomainMode\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "upgradeToDomain",
              "kind": "CONDITION",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Domain rule mode WRITE succeeds
    When I POST "/subjects/domain-write/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"DomainWrite\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "writeRule",
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

  Scenario: Domain rule mode READ succeeds
    When I POST "/subjects/domain-read/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"DomainRead\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "readRule",
              "kind": "CONDITION",
              "mode": "READ",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200

  Scenario: Domain rule mode WRITEREAD succeeds
    When I POST "/subjects/domain-writeread/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"DomainWriteRead\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "writeReadRule",
              "kind": "CONDITION",
              "mode": "WRITEREAD",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # INVALID RULESETS — INVALID MODE FOR MIGRATION RULES
  # ==========================================================================

  Scenario: Invalid mode for migration rule returns 422
    When I POST "/subjects/invalid-migration-mode/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"InvalidMigrationMode\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "migrationRules": [
            {
              "name": "badMode",
              "kind": "TRANSFORM",
              "mode": "INVALID",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Migration rule mode must be UPGRADE, DOWNGRADE, or UPDOWN
    When I POST "/subjects/bad-migration-mode/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadMigrationMode\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "migrationRules": [
            {
              "name": "readInMigration",
              "kind": "TRANSFORM",
              "mode": "READ",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Migration rule mode UPGRADE succeeds
    When I POST "/subjects/mig-upgrade/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MigUpgrade\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "migrationRules": [
            {
              "name": "upgradeRule",
              "kind": "TRANSFORM",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200

  Scenario: Migration rule mode DOWNGRADE succeeds
    When I POST "/subjects/mig-downgrade/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MigDowngrade\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "migrationRules": [
            {
              "name": "downgradeRule",
              "kind": "TRANSFORM",
              "mode": "DOWNGRADE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200

  Scenario: Migration rule mode UPDOWN succeeds
    When I POST "/subjects/mig-updown/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MigUpDown\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "migrationRules": [
            {
              "name": "updownRule",
              "kind": "TRANSFORM",
              "mode": "UPDOWN",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # INVALID RULESETS — INVALID ON_SUCCESS / ON_FAILURE ACTIONS
  # ==========================================================================

  Scenario: Invalid onSuccess action returns 422
    When I POST "/subjects/invalid-onsuccess/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"InvalidOnSuccess\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "badSuccess",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true",
              "onSuccess": "INVALID_ACTION"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Invalid onFailure action returns 422
    When I POST "/subjects/invalid-onfailure/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"InvalidOnFailure\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "badFailure",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true",
              "onFailure": "INVALID_ACTION"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: onSuccess and onFailure accept NONE, DLQ, ERROR, or empty
    When I POST "/subjects/all-actions/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"AllActions\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "domainRules": [
            {
              "name": "rule1",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true",
              "onSuccess": "NONE",
              "onFailure": "DLQ"
            },
            {
              "name": "rule2",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true",
              "onSuccess": "ERROR",
              "onFailure": "NONE"
            },
            {
              "name": "rule3",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true",
              "onSuccess": "",
              "onFailure": ""
            }
          ]
        }
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # INVALID RULESETS — CONFIG OPERATIONS
  # ==========================================================================

  Scenario: Invalid defaultRuleSet in config PUT /config returns 422
    When I PUT "/config" with body:
      """
      {
        "compatibility": "BACKWARD",
        "defaultRuleSet": {
          "domainRules": [
            {
              "name": "",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Invalid overrideRuleSet in config PUT /config/{subject} returns 422
    When I PUT "/config/bad-override" with body:
      """
      {
        "compatibility": "NONE",
        "overrideRuleSet": {
          "domainRules": [
            {
              "name": "badRule",
              "kind": "INVALID_KIND",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: CONDITION kind is valid for migration rules
    When I PUT "/config/mig-condition-rules" with body:
      """
      {
        "compatibility": "FULL",
        "defaultRuleSet": {
          "migrationRules": [
            {
              "name": "conditionMigKind",
              "kind": "CONDITION",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200

  Scenario: Invalid rule kind in migration rules of config returns 422
    When I PUT "/config/bad-mig-rules" with body:
      """
      {
        "compatibility": "FULL",
        "defaultRuleSet": {
          "migrationRules": [
            {
              "name": "badMigKind",
              "kind": "FILTER",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  # ==========================================================================
  # ENCODING RULES (OPTIONAL VALIDATION)
  # ==========================================================================

  Scenario: Valid encoding rule with WRITE mode succeeds
    When I POST "/subjects/encoding-write/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EncodingWrite\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "encodeRule",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 200

  Scenario: Encoding rule mode must be WRITE, READ, or WRITEREAD
    When I POST "/subjects/encoding-invalid/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EncodingInvalid\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "encodeRule",
              "kind": "TRANSFORM",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "true"
            }
          ]
        }
      }
      """
    Then the response status should be 422
    And the response should have error code 42201
