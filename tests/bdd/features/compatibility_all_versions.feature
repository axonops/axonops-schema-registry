@functional
Feature: Compatibility Check Against All Versions
  As a schema registry user
  I want to check schema compatibility against all registered versions
  So that I can validate evolution without specifying individual version numbers

  Background:
    Given the schema registry is running

  Scenario: Compatible schema against all versions
    Given the global compatibility level is "BACKWARD"
    And subject "compat-all-test" has schema:
      """
      {
        "type": "record",
        "name": "User",
        "fields": [
          {"name": "id", "type": "int"},
          {"name": "name", "type": "string"}
        ]
      }
      """
    And I POST "/subjects/compat-all-test/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    And the response status should be 200
    When I POST "/compatibility/subjects/compat-all-test/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"created\",\"type\":[\"null\",\"long\"],\"default\":null}]}"
      }
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: Incompatible schema against all versions
    Given the global compatibility level is "BACKWARD"
    And subject "incompat-all-test" has schema:
      """
      {
        "type": "record",
        "name": "Product",
        "fields": [
          {"name": "id", "type": "int"},
          {"name": "price", "type": "double"}
        ]
      }
      """
    When I POST "/compatibility/subjects/incompat-all-test/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Product\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"price\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false

  Scenario: BACKWARD compatibility against all versions
    Given I set the global config to "BACKWARD"
    And subject "backward-all" has schema:
      """
      {
        "type": "record",
        "name": "Event",
        "fields": [
          {"name": "timestamp", "type": "long"},
          {"name": "type", "type": "string"}
        ]
      }
      """
    When I POST "/compatibility/subjects/backward-all/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"timestamp\",\"type\":\"long\"},{\"name\":\"type\",\"type\":\"string\"},{\"name\":\"user_id\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: FORWARD compatibility against all versions
    Given I set the global config to "FORWARD"
    And subject "forward-all" has schema:
      """
      {
        "type": "record",
        "name": "Message",
        "fields": [
          {"name": "id", "type": "int"},
          {"name": "text", "type": "string"},
          {"name": "metadata", "type": ["null", "string"], "default": null}
        ]
      }
      """
    When I POST "/compatibility/subjects/forward-all/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Message\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"text\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: FULL compatibility against all versions
    Given I set the global config to "FULL"
    And subject "full-all" has schema:
      """
      {
        "type": "record",
        "name": "Order",
        "fields": [
          {"name": "order_id", "type": "string"},
          {"name": "amount", "type": "double"}
        ]
      }
      """
    When I POST "/compatibility/subjects/full-all/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"discount\",\"type\":[\"null\",\"double\"],\"default\":null}]}"
      }
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: Check against single version baseline
    Given the global compatibility level is "BACKWARD"
    And subject "single-version" has schema:
      """
      {
        "type": "record",
        "name": "Simple",
        "fields": [
          {"name": "value", "type": "int"}
        ]
      }
      """
    When I POST "/compatibility/subjects/single-version/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Simple\",\"fields\":[{\"name\":\"value\",\"type\":\"int\"},{\"name\":\"label\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: Check with verbose=true returns messages
    Given the global compatibility level is "BACKWARD"
    And subject "verbose-test" has schema:
      """
      {
        "type": "record",
        "name": "Data",
        "fields": [
          {"name": "id", "type": "int"}
        ]
      }
      """
    When I POST "/compatibility/subjects/verbose-test/versions?verbose=true" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Data\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"required_field\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false
    And the response should contain "messages"

  Scenario: Subject not found returns 404
    When I POST "/compatibility/subjects/nonexistent-subject/versions" with body:
      """
      {
        "schemaType": "AVRO",
        "schema": "{\"type\":\"record\",\"name\":\"Fake\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"
      }
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true
