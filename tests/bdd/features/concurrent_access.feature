@functional
Feature: Concurrent Access and Race Conditions
  As a schema registry operator
  I want the registry to handle parallel operations safely
  So that concurrent clients don't corrupt data or violate invariants

  Background:
    Given the schema registry is running
    And no subjects exist
    And the global compatibility level is "NONE"

  Scenario: Sequential ID allocation across multiple subjects
    When I POST "/subjects/subject-a/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RecordA\",\"fields\":[{\"name\":\"field1\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "id_a"
    When I POST "/subjects/subject-b/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RecordB\",\"fields\":[{\"name\":\"field2\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "id_b"
    When I POST "/subjects/subject-c/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"RecordC\",\"fields\":[{\"name\":\"field3\",\"type\":\"boolean\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "id_c"
    Then the response field "id" should not equal stored "id_a"
    And the response field "id" should not equal stored "id_b"
    And the stored "id_a" should be greater than 0
    And the stored "id_b" should be greater than 0
    And the stored "id_c" should be greater than 0

  Scenario: Schema deduplication across subjects returns same ID
    When I POST "/subjects/subject-alpha/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"username\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    When I POST "/subjects/subject-beta/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"username\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"

  Scenario: Rapid version creation under single subject
    When I POST "/subjects/rapid-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"V1\",\"fields\":[{\"name\":\"f1\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/rapid-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"V2\",\"fields\":[{\"name\":\"f2\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/rapid-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"V3\",\"fields\":[{\"name\":\"f3\",\"type\":\"boolean\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/rapid-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"V4\",\"fields\":[{\"name\":\"f4\",\"type\":\"long\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/rapid-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"V5\",\"fields\":[{\"name\":\"f5\",\"type\":\"double\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/rapid-subject/versions"
    Then the response status should be 200
    And the response should be an array of length 5

  Scenario: Register schema during soft-delete succeeds
    When I POST "/subjects/delete-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Before\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    When I DELETE "/subjects/delete-test"
    Then the response status should be 200
    When I POST "/subjects/delete-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"After\",\"fields\":[{\"name\":\"y\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/delete-test/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  Scenario: Register after permanent delete creates new version series
    When I POST "/subjects/permanent-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Original\",\"fields\":[{\"name\":\"a\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "original_id"
    When I DELETE "/subjects/permanent-test"
    Then the response status should be 200
    When I DELETE "/subjects/permanent-test?permanent=true"
    Then the response status should be 200
    When I POST "/subjects/permanent-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"New\",\"fields\":[{\"name\":\"b\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "original_id"

  Scenario: Config change does not affect already-registered schema
    When I POST "/subjects/config-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"First\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "config_id"
    When I PUT "/config/config-test" with body:
      """
      {
        "compatibility": "BACKWARD"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/config-test/versions/1"
    Then the response status should be 200
    And the response field "id" should equal stored "config_id"

  Scenario: Mode switch to READONLY blocks new registrations
    When I PUT "/mode" with body:
      """
      {
        "mode": "READONLY"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/readonly-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Blocked\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 422
    And the response field "error_code" should be 42205

  Scenario: Mode switch to READWRITE allows registrations again
    When I PUT "/mode" with body:
      """
      {
        "mode": "READONLY"
      }
      """
    Then the response status should be 200
    When I PUT "/mode" with body:
      """
      {
        "mode": "READWRITE"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/readwrite-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Allowed\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  Scenario: Multiple subjects with identical schema receive same ID
    When I POST "/subjects/multi-a/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Shared\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "shared_id"
    When I POST "/subjects/multi-b/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Shared\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "shared_id"
    When I POST "/subjects/multi-c/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Shared\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "shared_id"

  Scenario: Sequential version numbering under rapid registration
    When I POST "/subjects/seq-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"One\",\"fields\":[{\"name\":\"a\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/seq-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Two\",\"fields\":[{\"name\":\"b\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/seq-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Three\",\"fields\":[{\"name\":\"c\",\"type\":\"boolean\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/seq-test/versions"
    Then the response status should be 200
    And the response should be an array of length 3

  Scenario: Delete and re-register preserves ID stability
    When I POST "/subjects/stable-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Stable\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "stable_id"
    When I DELETE "/subjects/stable-test"
    Then the response status should be 200
    When I POST "/subjects/stable-test/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Stable\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "stable_id"

  Scenario: Interleaved operations across subjects maintain correct versions
    When I POST "/subjects/interleave-a/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"A1\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/interleave-b/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"B1\",\"fields\":[{\"name\":\"y\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/interleave-a/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"A2\",\"fields\":[{\"name\":\"x\",\"type\":\"long\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/interleave-b/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"B2\",\"fields\":[{\"name\":\"y\",\"type\":\"boolean\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/interleave-a/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"A3\",\"fields\":[{\"name\":\"x\",\"type\":\"double\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/interleave-a/versions"
    Then the response status should be 200
    And the response should be an array of length 3
    When I GET "/subjects/interleave-b/versions"
    Then the response status should be 200
    And the response should be an array of length 2
