@functional @data-contracts
Feature: EncodingRules Support in RuleSet
  The RuleSet struct supports three types of rules: migrationRules, domainRules,
  and encodingRules. EncodingRules allow defining data encoding transformations
  such as encryption, compression, and serialization format conversions.
  These rules are stored per schema registration and can also be configured
  at the subject level via defaultRuleSet and overrideRuleSet.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # SCHEMA REGISTRATION WITH ENCODING RULES
  # ==========================================================================

  Scenario: Register schema with encodingRules — stored and returned
    When I POST "/subjects/enc-rules-basic/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EncBasic\",\"fields\":[{\"name\":\"ssn\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "encryptSSN",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "tags": ["PII"],
              "params": {"encrypt.kek.name": "ssn-key"},
              "expr": "message.ssn"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/enc-rules-basic/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "encodingRules"
    And the response body should contain "encryptSSN"
    And the response body should contain "TRANSFORM"
    And the response body should contain "WRITE"
    And the response body should contain "ENCRYPT"
    And the response body should contain "PII"
    And the response body should contain "ssn-key"
    And the response body should contain "message.ssn"

  # ==========================================================================
  # ALL THREE RULE TYPES TOGETHER
  # ==========================================================================

  Scenario: Register schema with all three rule types — migrationRules, domainRules, and encodingRules
    When I POST "/subjects/enc-rules-all-three/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"AllThree\",\"fields\":[{\"name\":\"email\",\"type\":\"string\"},{\"name\":\"payload\",\"type\":\"bytes\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "migrationRules": [
            {
              "name": "upgradeV1",
              "kind": "TRANSFORM",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "message"
            }
          ],
          "domainRules": [
            {
              "name": "validateEmail",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "message.email.matches('^[a-zA-Z0-9+_.-]+@[a-zA-Z0-9.-]+$')"
            }
          ],
          "encodingRules": [
            {
              "name": "compressPayload",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "COMPRESS",
              "expr": "message.payload"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/enc-rules-all-three/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    # Verify migrationRules present
    And the response body should contain "migrationRules"
    And the response body should contain "upgradeV1"
    And the response body should contain "UPGRADE"
    # Verify domainRules present
    And the response body should contain "domainRules"
    And the response body should contain "validateEmail"
    And the response body should contain "CONDITION"
    # Verify encodingRules present
    And the response body should contain "encodingRules"
    And the response body should contain "compressPayload"
    And the response body should contain "COMPRESS"

  # ==========================================================================
  # CONFIG WITH defaultRuleSet INCLUDING ENCODING RULES
  # ==========================================================================

  Scenario: Config with defaultRuleSet including encodingRules
    When I PUT "/config/enc-rules-default-cfg" with body:
      """
      {
        "compatibility": "BACKWARD",
        "defaultRuleSet": {
          "encodingRules": [
            {
              "name": "defaultEncrypt",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "tags": ["SENSITIVE"],
              "params": {"encrypt.kek.name": "default-key"}
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/config/enc-rules-default-cfg"
    Then the response status should be 200
    And the response should have field "defaultRuleSet"
    And the response body should contain "encodingRules"
    And the response body should contain "defaultEncrypt"
    And the response body should contain "ENCRYPT"
    And the response body should contain "SENSITIVE"
    And the response body should contain "default-key"

  # ==========================================================================
  # CONFIG WITH overrideRuleSet INCLUDING ENCODING RULES
  # ==========================================================================

  Scenario: Config with overrideRuleSet including encodingRules
    When I PUT "/config/enc-rules-override-cfg" with body:
      """
      {
        "compatibility": "NONE",
        "overrideRuleSet": {
          "encodingRules": [
            {
              "name": "overrideCompress",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "COMPRESS",
              "params": {"compress.type": "gzip"}
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/config/enc-rules-override-cfg"
    Then the response status should be 200
    And the response should have field "overrideRuleSet"
    And the response body should contain "encodingRules"
    And the response body should contain "overrideCompress"
    And the response body should contain "COMPRESS"
    And the response body should contain "gzip"

  # ==========================================================================
  # SCHEMA WITHOUT RULES — ruleSet OMITTED
  # ==========================================================================

  Scenario: Schema without any rules — ruleSet omitted in response
    When I POST "/subjects/enc-rules-no-rules/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"NoRules\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/enc-rules-no-rules/versions/1"
    Then the response status should be 200
    And the response body should not contain "ruleSet"
    And the response body should not contain "encodingRules"
    And the response body should not contain "migrationRules"
    And the response body should not contain "domainRules"

  # ==========================================================================
  # ENCODING RULES WITH ALL RULE FIELDS
  # ==========================================================================

  Scenario: EncodingRules with all rule fields populated
    When I POST "/subjects/enc-rules-all-fields/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"AllFields\",\"fields\":[{\"name\":\"ssn\",\"type\":\"string\"},{\"name\":\"dob\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "encryptPII",
              "doc": "Encrypts all PII fields using AES-256-GCM envelope encryption",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "tags": ["PII", "GDPR"],
              "params": {
                "encrypt.kek.name": "pii-master-key",
                "encrypt.algorithm": "AES256_GCM"
              },
              "expr": "message.ssn",
              "onSuccess": "NONE",
              "onFailure": "ERROR",
              "disabled": false
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/enc-rules-all-fields/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "encodingRules"
    # Verify all fields present
    And the response body should contain "encryptPII"
    And the response body should contain "Encrypts all PII fields using AES-256-GCM envelope encryption"
    And the response body should contain "TRANSFORM"
    And the response body should contain "WRITE"
    And the response body should contain "ENCRYPT"
    And the response body should contain "PII"
    And the response body should contain "GDPR"
    And the response body should contain "pii-master-key"
    And the response body should contain "AES256_GCM"
    And the response body should contain "message.ssn"
    And the response body should contain "NONE"
    And the response body should contain "ERROR"

  # ==========================================================================
  # MULTIPLE ENCODING RULES IN A SINGLE RULESET
  # ==========================================================================

  Scenario: Multiple encodingRules in a single ruleSet
    When I POST "/subjects/enc-rules-multiple/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"MultiEnc\",\"fields\":[{\"name\":\"ssn\",\"type\":\"string\"},{\"name\":\"creditCard\",\"type\":\"string\"},{\"name\":\"data\",\"type\":\"bytes\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "encryptSSN",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "tags": ["PII"],
              "params": {"encrypt.kek.name": "ssn-key"},
              "expr": "message.ssn"
            },
            {
              "name": "encryptCC",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "tags": ["PCI"],
              "params": {"encrypt.kek.name": "cc-key"},
              "expr": "message.creditCard"
            },
            {
              "name": "compressData",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "COMPRESS",
              "expr": "message.data"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/enc-rules-multiple/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "encodingRules"
    And the response body should contain "encryptSSN"
    And the response body should contain "encryptCC"
    And the response body should contain "compressData"
    And the response body should contain "PII"
    And the response body should contain "PCI"
    And the response body should contain "ssn-key"
    And the response body should contain "cc-key"
    And the response body should contain "COMPRESS"

  # ==========================================================================
  # ENCODING RULES RETRIEVED VIA SCHEMA ID ENDPOINT
  # ==========================================================================

  Scenario: EncodingRules returned when fetching schema by global ID
    When I POST "/subjects/enc-rules-by-id/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EncById\",\"fields\":[{\"name\":\"token\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "tokenEncrypt",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "tags": ["SECRET"],
              "params": {"encrypt.kek.name": "token-key"},
              "expr": "message.token"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "enc_schema_id"
    When I GET "/schemas/ids/{{enc_schema_id}}"
    Then the response status should be 200
    And the response should have field "schema"

  # ==========================================================================
  # ENCODING RULES WITH DISABLED FLAG
  # ==========================================================================

  Scenario: EncodingRules with disabled flag set to true
    When I POST "/subjects/enc-rules-disabled/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EncDisabled\",\"fields\":[{\"name\":\"secret\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "disabledEncrypt",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "expr": "message.secret",
              "disabled": true
            }
          ]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/enc-rules-disabled/versions/1"
    Then the response status should be 200
    And the response should have field "ruleSet"
    And the response body should contain "encodingRules"
    And the response body should contain "disabledEncrypt"
    And the response body should contain "disabled"

  # ==========================================================================
  # CONFIG defaultRuleSet WITH ALL THREE RULE TYPES
  # ==========================================================================

  Scenario: Config defaultRuleSet with migrationRules, domainRules, and encodingRules
    When I PUT "/config/enc-rules-cfg-all-three" with body:
      """
      {
        "compatibility": "BACKWARD",
        "defaultRuleSet": {
          "migrationRules": [
            {
              "name": "cfgMigration",
              "kind": "TRANSFORM",
              "mode": "UPGRADE",
              "type": "CEL",
              "expr": "message"
            }
          ],
          "domainRules": [
            {
              "name": "cfgDomainValidate",
              "kind": "CONDITION",
              "mode": "WRITE",
              "type": "CEL",
              "expr": "size(message.name) > 0"
            }
          ],
          "encodingRules": [
            {
              "name": "cfgEncodingEncrypt",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "tags": ["CONFIDENTIAL"],
              "params": {"encrypt.kek.name": "cfg-key"}
            }
          ]
        }
      }
      """
    Then the response status should be 200
    When I GET "/config/enc-rules-cfg-all-three"
    Then the response status should be 200
    And the response should have field "defaultRuleSet"
    And the response body should contain "migrationRules"
    And the response body should contain "cfgMigration"
    And the response body should contain "domainRules"
    And the response body should contain "cfgDomainValidate"
    And the response body should contain "encodingRules"
    And the response body should contain "cfgEncodingEncrypt"
    And the response body should contain "CONFIDENTIAL"
    And the response body should contain "cfg-key"

  # ==========================================================================
  # ENCODING RULES PRESERVED ACROSS SCHEMA VERSIONS
  # ==========================================================================

  Scenario: EncodingRules are independent per version
    # Register v1 with encoding rules
    When I POST "/subjects/enc-rules-versioned/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EncVersioned\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "v1Encrypt",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "ENCRYPT",
              "expr": "message.name"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    # Register v2 with different encoding rules
    When I PUT "/config/enc-rules-versioned" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    When I POST "/subjects/enc-rules-versioned/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"EncVersioned\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\"}]}",
        "schemaType": "AVRO",
        "ruleSet": {
          "encodingRules": [
            {
              "name": "v2Compress",
              "kind": "TRANSFORM",
              "mode": "WRITE",
              "type": "COMPRESS",
              "expr": "message.name"
            }
          ]
        }
      }
      """
    Then the response status should be 200
    # Verify v1 still has its original encoding rules
    When I GET "/subjects/enc-rules-versioned/versions/1"
    Then the response status should be 200
    And the response body should contain "v1Encrypt"
    And the response body should contain "ENCRYPT"
    And the response body should not contain "v2Compress"
    # Verify v2 has its own encoding rules
    When I GET "/subjects/enc-rules-versioned/versions/2"
    Then the response status should be 200
    And the response body should contain "v2Compress"
    And the response body should contain "COMPRESS"
    And the response body should not contain "v1Encrypt"
