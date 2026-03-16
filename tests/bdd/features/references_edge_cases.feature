@functional @edge-case
Feature: Schema Reference Edge Cases
  As a developer registering schemas with cross-subject references
  I want the registry to enforce reference integrity and type constraints
  So that invalid or deeply nested references are handled correctly

  Background:
    Given the schema registry is running
    And the global compatibility level is "NONE"

  # ---------------------------------------------------------------------------
  # Cross-format reference rejection
  # ---------------------------------------------------------------------------

  Scenario: Avro schema referencing a Protobuf subject should fail
    # Register a Protobuf schema as the reference target
    Given subject "proto-ref-target" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message ProtoTarget {
        string name = 1;
      }
      """
    # Attempt to register an Avro schema with a reference to the Protobuf subject
    When I register a schema under subject "avro-cross-ref" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"AvroCross\",\"fields\":[{\"name\":\"target\",\"type\":\"ProtoTarget\"}]}",
        "references": [
          {"name": "ProtoTarget", "subject": "proto-ref-target", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  Scenario: Protobuf schema referencing an Avro subject should fail
    # Register an Avro schema as the reference target
    Given subject "avro-ref-target" has schema:
      """
      {"type":"record","name":"AvroTarget","fields":[{"name":"id","type":"int"}]}
      """
    # Attempt to register a Protobuf schema importing from the Avro subject
    When I register a "PROTOBUF" schema under subject "proto-cross-ref" with references:
      """
      {
        "schema": "syntax = \"proto3\";\nimport \"avro-ref-target.proto\";\nmessage ProtoCross {\n  AvroTarget target = 1;\n}",
        "schemaType": "PROTOBUF",
        "references": [
          {"name": "avro-ref-target.proto", "subject": "avro-ref-target", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  Scenario: JSON Schema referencing an Avro subject should fail
    Given subject "avro-json-target" has schema:
      """
      {"type":"record","name":"AvroForJson","fields":[{"name":"id","type":"string"}]}
      """
    When I register a "JSON" schema under subject "json-cross-ref" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"ref\":{\"$ref\":\"avro-json-target\"}}}",
        "schemaType": "JSON",
        "references": [
          {"name": "avro-json-target", "subject": "avro-json-target", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  # ---------------------------------------------------------------------------
  # Deeply nested reference chains (5+ levels)
  # ---------------------------------------------------------------------------

  Scenario: Avro schema with 5 levels of transitive references succeeds
    # Level 1: base type
    Given subject "deep-level-1" has schema:
      """
      {"type":"record","name":"Level1","namespace":"com.deep","fields":[{"name":"id","type":"string"}]}
      """
    # Level 2 references Level 1
    And subject "deep-level-2" has "AVRO" schema with reference "com.deep.Level1" from subject "deep-level-1" version 1:
      """
      {"type":"record","name":"Level2","namespace":"com.deep","fields":[{"name":"l1","type":"com.deep.Level1"},{"name":"val","type":"int"}]}
      """
    # Level 3 references Level 2 (which transitively requires Level 1)
    When I register a schema under subject "deep-level-3" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Level3\",\"namespace\":\"com.deep\",\"fields\":[{\"name\":\"l2\",\"type\":\"com.deep.Level2\"},{\"name\":\"data\",\"type\":\"string\"}]}",
        "references": [
          {"name": "com.deep.Level2", "subject": "deep-level-2", "version": 1},
          {"name": "com.deep.Level1", "subject": "deep-level-1", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "level3_id"
    # Level 4 references Level 3
    When I register a schema under subject "deep-level-4" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Level4\",\"namespace\":\"com.deep\",\"fields\":[{\"name\":\"l3\",\"type\":\"com.deep.Level3\"},{\"name\":\"flag\",\"type\":\"boolean\"}]}",
        "references": [
          {"name": "com.deep.Level3", "subject": "deep-level-3", "version": 1},
          {"name": "com.deep.Level2", "subject": "deep-level-2", "version": 1},
          {"name": "com.deep.Level1", "subject": "deep-level-1", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "level4_id"
    # Level 5 references Level 4
    When I register a schema under subject "deep-level-5" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Level5\",\"namespace\":\"com.deep\",\"fields\":[{\"name\":\"l4\",\"type\":\"com.deep.Level4\"},{\"name\":\"ts\",\"type\":\"long\"}]}",
        "references": [
          {"name": "com.deep.Level4", "subject": "deep-level-4", "version": 1},
          {"name": "com.deep.Level3", "subject": "deep-level-3", "version": 1},
          {"name": "com.deep.Level2", "subject": "deep-level-2", "version": 1},
          {"name": "com.deep.Level1", "subject": "deep-level-1", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Verify the final schema is retrievable
    When I get the latest version of subject "deep-level-5"
    Then the response status should be 200
    And the response should contain "Level5"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | deep-level-5                             |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/deep-level-5/versions          |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ---------------------------------------------------------------------------
  # Reference to non-existent subject
  # ---------------------------------------------------------------------------

  Scenario: Reference to non-existent subject returns 422
    When I register a schema under subject "bad-ref-test" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadRefTest\",\"fields\":[{\"name\":\"ref\",\"type\":\"com.missing.Missing\"}]}",
        "references": [
          {"name": "com.missing.Missing", "subject": "does-not-exist", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  # ---------------------------------------------------------------------------
  # Reference to non-existent version
  # ---------------------------------------------------------------------------

  Scenario: Reference to non-existent version returns 422
    Given subject "exists-but-v1" has schema:
      """
      {"type":"record","name":"ExistsV1","namespace":"com.ref","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "bad-version-ref" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BadVerRef\",\"fields\":[{\"name\":\"ref\",\"type\":\"com.ref.ExistsV1\"}]}",
        "references": [
          {"name": "com.ref.ExistsV1", "subject": "exists-but-v1", "version": 99}
        ]
      }
      """
    Then the response status should be 422

  # ---------------------------------------------------------------------------
  # ReferencedBy tracking
  # ---------------------------------------------------------------------------

  Scenario: ReferencedBy returns IDs of schemas that reference a given subject version
    Given subject "ref-base" has schema:
      """
      {"type":"record","name":"RefBase","namespace":"com.track","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-consumer-1" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer1\",\"namespace\":\"com.track\",\"fields\":[{\"name\":\"base\",\"type\":\"com.track.RefBase\"}]}",
        "references": [
          {"name": "com.track.RefBase", "subject": "ref-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "ref-base" version 1
    Then the response status should be 200
    And the response should be valid JSON
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | ref-consumer-1                           |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/ref-consumer-1/versions        |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
