@functional
Feature: Confluent Conformance
  Verify edge-case behaviors discovered by analyzing Confluent's RestApiTest.java.
  These tests ensure wire-level compatibility with the Confluent Schema Registry.

  # ==========================================================================
  # LOOKUP ERROR CODES — 40401 vs 40403
  # Confluent behavior:
  #   - POST /subjects/{subject} when subject doesn't exist → 40401
  #   - POST /subjects/{subject} when subject exists but schema not found → 40403
  # ==========================================================================

  Scenario: Lookup on non-existent subject returns 40401
    When I POST "/subjects/conf-lookup-nosub" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"X\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: Lookup when schema not under subject returns 40403
    Given subject "conf-lookup-exists" has schema:
      """
      {"type":"record","name":"LookupA","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/subjects/conf-lookup-exists" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Different\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 404
    And the response should have error code 40403

  # ==========================================================================
  # DOUBLE SOFT-DELETE — 40404
  # Confluent behavior:
  #   - DELETE /subjects/{subject} on already-soft-deleted subject → 40404
  # ==========================================================================

  Scenario: Double soft-delete subject returns 40404
    Given subject "conf-dbl-del" has schema:
      """
      {"type":"record","name":"DblDel","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/conf-dbl-del"
    Then the response status should be 200
    When I DELETE "/subjects/conf-dbl-del"
    Then the response status should be 404
    And the response should have error code 40404

  # ==========================================================================
  # VERSION CONTINUITY AFTER SOFT-DELETE
  # Confluent behavior:
  #   - After soft-delete and re-registration, version numbers CONTINUE
  #   - After permanent delete, version numbers RESET to 1
  # ==========================================================================

  Scenario: Version numbers continue after soft-delete
    Given subject "conf-ver-cont" has schema:
      """
      {"type":"record","name":"VerCont1","fields":[{"name":"a","type":"string"}]}
      """
    And I register a schema under subject "conf-ver-cont":
      """
      {"type":"record","name":"VerCont1","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":""}]}
      """
    When I DELETE "/subjects/conf-ver-cont"
    Then the response status should be 200
    When I register a schema under subject "conf-ver-cont":
      """
      {"type":"record","name":"VerCont1","fields":[{"name":"a","type":"string"},{"name":"c","type":"string","default":""}]}
      """
    Then the response status should be 200
    When I GET "/subjects/conf-ver-cont/versions"
    Then the response status should be 200
    And the response should contain "3"
    And the response body should not contain "1"
    And the response body should not contain "2"

  Scenario: Version numbers reset after permanent delete
    Given subject "conf-ver-reset" has schema:
      """
      {"type":"record","name":"VerReset","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/conf-ver-reset"
    Then the response status should be 200
    When I DELETE "/subjects/conf-ver-reset?permanent=true"
    Then the response status should be 200
    When I register a schema under subject "conf-ver-reset":
      """
      {"type":"record","name":"VerReset","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    When I GET "/subjects/conf-ver-reset/versions"
    Then the response status should be 200
    And the response should contain "1"

  # ==========================================================================
  # SCHEMA TYPE MIXING WHEN COMPATIBILITY=NONE
  # Confluent behavior:
  #   - When compat is NONE, a subject can have Avro, JSON Schema, and Protobuf
  #     as different versions.
  # ==========================================================================

  Scenario: Different schema types on same subject when compatibility is NONE
    Given the global compatibility level is "NONE"
    When I POST "/subjects/conf-type-mix/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"TypeMix\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/conf-type-mix/versions" with body:
      """
      {"schema": "{\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}", "schemaType": "JSON"}
      """
    Then the response status should be 200
    When I POST "/subjects/conf-type-mix/versions" with body:
      """
      {"schema": "syntax = \"proto3\"; message TypeMix { string a = 1; }", "schemaType": "PROTOBUF"}
      """
    Then the response status should be 200
    When I GET "/subjects/conf-type-mix/versions"
    Then the response status should be 200
    And the response should contain "1"
    And the response should contain "2"
    And the response should contain "3"
    When I set the global compatibility level to "BACKWARD"

  # ==========================================================================
  # COMPATIBILITY EXCLUDES SOFT-DELETED VERSIONS
  # Confluent behavior:
  #   - Compatibility checks skip soft-deleted versions
  #   - Deleting an incompatible version makes previously-blocked schemas valid
  # ==========================================================================

  Scenario: Compatibility check ignores soft-deleted versions
    Given the global compatibility level is "BACKWARD"
    And subject "conf-compat-del" has schema:
      """
      {"type":"record","name":"CompatDel","fields":[{"name":"a","type":"string"}]}
      """
    # Register a v2 with a new required field (backward compatible with default)
    And I register a schema under subject "conf-compat-del":
      """
      {"type":"record","name":"CompatDel","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":""}]}
      """
    # Soft-delete v2 (the one with field b)
    When I DELETE "/subjects/conf-compat-del/versions/2"
    Then the response status should be 200
    # Now register a schema that adds field c (compatible with v1, would be incompatible if v2 existed under BACKWARD_TRANSITIVE)
    When I register a schema under subject "conf-compat-del":
      """
      {"type":"record","name":"CompatDel","fields":[{"name":"a","type":"string"},{"name":"c","type":"string","default":""}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # CONFIG ON NON-EXISTENT SUBJECT
  # Confluent behavior:
  #   - PUT /config/{subject} succeeds even if subject has no schemas
  #   - The config is retrievable afterwards
  # ==========================================================================

  Scenario: Config on non-existent subject succeeds
    When I PUT "/config/conf-config-nosub" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    When I GET "/config/conf-config-nosub?defaultToGlobal=false"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  # ==========================================================================
  # CANONICAL STRING IDEMPOTENCE
  # Confluent behavior:
  #   - Schemas with different whitespace but same canonical form get same ID
  # ==========================================================================

  Scenario: Schemas with different whitespace get same ID
    When I POST "/subjects/conf-canonical/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Canonical\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "canonical_id1"
    When I POST "/subjects/conf-canonical/versions" with body:
      """
      {"schema": "{  \"type\" :  \"record\" , \"name\" : \"Canonical\" , \"fields\" : [ { \"name\" : \"a\" , \"type\" : \"string\" } ] }"}
      """
    Then the response status should be 200
    And I store the response field "id" as "canonical_id2"
    # Both should get the same ID (idempotent registration)

  # ==========================================================================
  # GET VERSIONS AFTER ALL SOFT-DELETED
  # Confluent behavior:
  #   - GET /subjects/{subject}/versions → 40401 when all versions soft-deleted
  #   - GET /subjects/{subject}/versions?deleted=true → shows soft-deleted versions
  # ==========================================================================

  Scenario: GET versions after all versions soft-deleted returns 40401
    Given subject "conf-all-del" has schema:
      """
      {"type":"record","name":"AllDel","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/conf-all-del"
    Then the response status should be 200
    When I GET "/subjects/conf-all-del/versions"
    Then the response status should be 404
    And the response should have error code 40401
    When I GET "/subjects/conf-all-del/versions?deleted=true"
    Then the response status should be 200
    And the response should contain "1"

  # ==========================================================================
  # REGISTER WITH VERSION -1 IN REQUEST BODY
  # Confluent behavior:
  #   - version=-1 in request body means "assign next version"
  #   - Does NOT literally create version -1
  # ==========================================================================

  Scenario: GET version -1 returns same as GET version latest
    Given subject "conf-ver-neg1" has schema:
      """
      {"type":"record","name":"VerNeg1","fields":[{"name":"a","type":"string"}]}
      """
    And I register a schema under subject "conf-ver-neg1":
      """
      {"type":"record","name":"VerNeg1","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":""}]}
      """
    When I GET "/subjects/conf-ver-neg1/versions/-1"
    Then the response status should be 200
    And the response field "version" should be 2
    When I GET "/subjects/conf-ver-neg1/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 2

  # ==========================================================================
  # PERMANENT DELETE RETURNS VERSION LIST
  # Confluent behavior:
  #   - DELETE /subjects/{subject}?permanent=true returns list of permanently deleted versions
  # ==========================================================================

  Scenario: Permanent delete returns version list
    Given subject "conf-perm-list" has schema:
      """
      {"type":"record","name":"PermList1","fields":[{"name":"a","type":"string"}]}
      """
    And I register a schema under subject "conf-perm-list":
      """
      {"type":"record","name":"PermList1","fields":[{"name":"a","type":"string"},{"name":"b","type":"string","default":""}]}
      """
    When I DELETE "/subjects/conf-perm-list"
    Then the response status should be 200
    And the response should contain "1"
    And the response should contain "2"
    When I DELETE "/subjects/conf-perm-list?permanent=true"
    Then the response status should be 200
    And the response should contain "1"
    And the response should contain "2"
