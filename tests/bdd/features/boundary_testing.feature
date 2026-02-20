@functional
Feature: Boundary Testing
  Test large payloads and boundary conditions for the schema registry

  Background:
    Given the schema registry is running
    And no subjects exist
    And the global compatibility level is "NONE"

  Scenario: Register large Avro schema with many fields
    When I POST "/subjects/large-schema/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"LargeRecord\",\"fields\":[{\"name\":\"field1\",\"type\":\"string\"},{\"name\":\"field2\",\"type\":\"int\"},{\"name\":\"field3\",\"type\":\"long\"},{\"name\":\"field4\",\"type\":\"float\"},{\"name\":\"field5\",\"type\":\"double\"},{\"name\":\"field6\",\"type\":\"boolean\"},{\"name\":\"field7\",\"type\":\"string\"},{\"name\":\"field8\",\"type\":\"int\"},{\"name\":\"field9\",\"type\":\"long\"},{\"name\":\"field10\",\"type\":\"float\"},{\"name\":\"field11\",\"type\":\"double\"},{\"name\":\"field12\",\"type\":\"boolean\"},{\"name\":\"field13\",\"type\":\"string\"},{\"name\":\"field14\",\"type\":\"int\"},{\"name\":\"field15\",\"type\":\"long\"},{\"name\":\"field16\",\"type\":\"float\"},{\"name\":\"field17\",\"type\":\"double\"},{\"name\":\"field18\",\"type\":\"boolean\"},{\"name\":\"field19\",\"type\":\"string\"},{\"name\":\"field20\",\"type\":\"int\"},{\"name\":\"field21\",\"type\":\"long\"},{\"name\":\"field22\",\"type\":\"float\"},{\"name\":\"field23\",\"type\":\"double\"},{\"name\":\"field24\",\"type\":\"boolean\"},{\"name\":\"field25\",\"type\":\"string\"},{\"name\":\"field26\",\"type\":\"int\"},{\"name\":\"field27\",\"type\":\"long\"},{\"name\":\"field28\",\"type\":\"float\"},{\"name\":\"field29\",\"type\":\"double\"},{\"name\":\"field30\",\"type\":\"boolean\"},{\"name\":\"field31\",\"type\":\"string\"},{\"name\":\"field32\",\"type\":\"int\"},{\"name\":\"field33\",\"type\":\"long\"},{\"name\":\"field34\",\"type\":\"float\"},{\"name\":\"field35\",\"type\":\"double\"},{\"name\":\"field36\",\"type\":\"boolean\"},{\"name\":\"field37\",\"type\":\"string\"},{\"name\":\"field38\",\"type\":\"int\"},{\"name\":\"field39\",\"type\":\"long\"},{\"name\":\"field40\",\"type\":\"float\"},{\"name\":\"field41\",\"type\":\"double\"},{\"name\":\"field42\",\"type\":\"boolean\"},{\"name\":\"field43\",\"type\":\"string\"},{\"name\":\"field44\",\"type\":\"int\"},{\"name\":\"field45\",\"type\":\"long\"},{\"name\":\"field46\",\"type\":\"float\"},{\"name\":\"field47\",\"type\":\"double\"},{\"name\":\"field48\",\"type\":\"boolean\"},{\"name\":\"field49\",\"type\":\"string\"},{\"name\":\"field50\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/large-schema/versions/1"
    Then the response status should be 200
    And the response should contain "field50"

  Scenario: Register deeply nested Avro schema
    When I POST "/subjects/nested-schema/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Level1\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"},{\"name\":\"nested\",\"type\":{\"type\":\"record\",\"name\":\"Level2\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"},{\"name\":\"nested\",\"type\":{\"type\":\"record\",\"name\":\"Level3\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"int\"}]}}]}}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/nested-schema/versions/1/schema"
    Then the response status should be 200
    And the response should contain "Level3"

  Scenario: Register schema with very long field names
    When I POST "/subjects/long-field-names/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"LongFieldNames\",\"fields\":[{\"name\":\"this_is_a_very_long_field_name_that_exceeds_one_hundred_characters_to_test_boundary_conditions_properly\",\"type\":\"string\"},{\"name\":\"another_extremely_long_field_name_designed_to_validate_handling_of_extended_identifiers_in_schema\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  Scenario: Register many versions under one subject
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version1\",\"fields\":[{\"name\":\"field1\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version2\",\"fields\":[{\"name\":\"field2\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version3\",\"fields\":[{\"name\":\"field3\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version4\",\"fields\":[{\"name\":\"field4\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version5\",\"fields\":[{\"name\":\"field5\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version6\",\"fields\":[{\"name\":\"field6\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version7\",\"fields\":[{\"name\":\"field7\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version8\",\"fields\":[{\"name\":\"field8\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version9\",\"fields\":[{\"name\":\"field9\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/versioned-subject/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Version10\",\"fields\":[{\"name\":\"field10\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects/versioned-subject/versions"
    Then the response status should be 200
    And the response should be an array of length 10

  Scenario: Register many subjects
    When I POST "/subjects/subject-01/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject01\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-02/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject02\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-03/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject03\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-04/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject04\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-05/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject05\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-06/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject06\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-07/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject07\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-08/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject08\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-09/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject09\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-10/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject10\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-11/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject11\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-12/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject12\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-13/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject13\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-14/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject14\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-15/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject15\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-16/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject16\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-17/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject17\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-18/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject18\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-19/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject19\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I POST "/subjects/subject-20/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Subject20\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    When I GET "/subjects"
    Then the response status should be 200
    And the response should be an array of length 20

  Scenario: Empty schema string returns error
    When I POST "/subjects/empty-schema/versions" with body:
      """
      {
        "schema": ""
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Invalid JSON as schema returns error
    When I POST "/subjects/invalid-json/versions" with body:
      """
      {
        "schema": "{not valid json at all"
      }
      """
    Then the response status should be 422
    And the response should have error code 42201

  Scenario: Schema with Unicode characters in doc fields
    When I POST "/subjects/unicode-schema/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"UnicodeRecord\",\"doc\":\"Documentation with Unicode 日本語 and émojis\",\"fields\":[{\"name\":\"username\",\"type\":\"string\",\"doc\":\"Имя пользователя\"},{\"name\":\"count\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/unicode-schema/versions/1/schema"
    Then the response status should be 200
    And the response should contain "UnicodeRecord"

  Scenario: Schema with special characters in subject name
    When I POST "/subjects/com.example.test-service.events/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"TestEvent\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/com.example.test-service.events/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  Scenario: Very long subject name
    When I POST "/subjects/this-is-an-extremely-long-subject-name-that-is-designed-to-test-the-boundary-conditions-for-subject-name-length-validation-in-the-schema-registry-implementation-to-ensure-robustness-and-proper-handling/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"LongSubject\",\"fields\":[{\"name\":\"value\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    When I GET "/subjects/this-is-an-extremely-long-subject-name-that-is-designed-to-test-the-boundary-conditions-for-subject-name-length-validation-in-the-schema-registry-implementation-to-ensure-robustness-and-proper-handling/versions/1"
    Then the response status should be 200
    And the response field "version" should be 1
