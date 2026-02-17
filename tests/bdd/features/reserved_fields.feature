@functional
Feature: Reserved Fields Validation
  When validateFields is enabled, schemas are checked against reserved field
  names listed in the "confluent:reserved" metadata property. Two rules apply:
  1. Reserved fields must not conflict with actual schema fields.
  2. Reserved fields from a previous version must not be removed.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # ENABLE / DISABLE validateFields
  # ==========================================================================

  Scenario: Enable validateFields via global config
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": true}
      """
    Then the response status should be 200
    When I GET "/config"
    Then the response status should be 200
    And the response body should contain "validateFields"
    # Reset
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200

  Scenario: Enable validateFields via subject config
    When I PUT "/config/reserved-cfg" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    When I GET "/config/reserved-cfg"
    Then the response status should be 200
    And the response body should contain "validateFields"
    # Cleanup
    When I DELETE "/config/reserved-cfg"
    Then the response status should be 200

  # ==========================================================================
  # RULE 1: Reserved fields must not conflict with actual fields
  # ==========================================================================

  Scenario: Registration fails when schema field conflicts with reserved field
    # Enable validateFields globally
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    # Try to register a schema with field "email" where "email" is reserved
    When I POST "/subjects/reserved-conflict/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Conflict\",\"fields\":[{\"name\":\"email\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "email"}
        }
      }
      """
    Then the response status should be 409
    And the response body should contain "conflicts with the reserved field"
    # Reset
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200

  Scenario: Registration succeeds when reserved field is not in schema
    # Enable validateFields globally
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    # Register schema with field "name" but reserve "future_field" (no conflict)
    When I POST "/subjects/reserved-noconflict/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"NoConflict\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "future_field"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    # Reset
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200

  Scenario: Multiple reserved fields — one conflicts
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    When I POST "/subjects/reserved-multi/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Multi\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"int\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "x, b"}
        }
      }
      """
    Then the response status should be 409
    And the response body should contain "conflicts with the reserved field b"
    # Reset
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200

  # ==========================================================================
  # RULE 2: Reserved fields from previous version must not be removed
  # ==========================================================================

  Scenario: Registration fails when reserved field from previous version is removed
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    # Register v1 with "future" reserved
    When I POST "/subjects/reserved-removal/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Removal\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "future"}
        }
      }
      """
    Then the response status should be 200
    # Try v2 without "future" reserved — should fail
    When I POST "/subjects/reserved-removal/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Removal\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"int\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "other"}
        }
      }
      """
    Then the response status should be 409
    And the response body should contain "reserved field future removed"
    # Reset
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200

  Scenario: Registration succeeds when reserved field is preserved across versions
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    # Register v1 with "future" reserved
    When I POST "/subjects/reserved-preserved/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Preserved\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "future"}
        }
      }
      """
    Then the response status should be 200
    # Register v2 keeping "future" reserved (adding more is OK)
    When I POST "/subjects/reserved-preserved/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Preserved\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"int\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "future,extra"}
        }
      }
      """
    Then the response status should be 200
    # Reset
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200

  # ==========================================================================
  # DISABLED BY DEFAULT
  # ==========================================================================

  Scenario: Reserved field conflicts allowed when validateFields is disabled (default)
    # validateFields defaults to false — reserved field conflicts should be allowed
    When I POST "/subjects/reserved-disabled/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Disabled\",\"fields\":[{\"name\":\"email\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "email"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  # ==========================================================================
  # SUBJECT-LEVEL OVERRIDE
  # ==========================================================================

  Scenario: Subject-level validateFields overrides global setting
    # Global: validateFields disabled
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE", "validateFields": false}
      """
    Then the response status should be 200
    # Subject: validateFields enabled
    When I PUT "/config/reserved-override" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    # This should fail because subject-level config overrides global
    When I POST "/subjects/reserved-override/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Override\",\"fields\":[{\"name\":\"x\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:reserved": "x"}
        }
      }
      """
    Then the response status should be 409
    And the response body should contain "conflicts with the reserved field"
    # Cleanup
    When I DELETE "/config/reserved-override"
    Then the response status should be 200
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200

  # ==========================================================================
  # PROTOBUF RESERVED FIELDS
  # ==========================================================================

  Scenario: Protobuf reserved field conflict detected
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    When I POST "/subjects/reserved-proto/versions" with body:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\nmessage Test {\n  string email = 1;\n}",
        "metadata": {
          "properties": {"confluent:reserved": "email"}
        }
      }
      """
    Then the response status should be 409
    And the response body should contain "conflicts with the reserved field"
    # Reset
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200

  # ==========================================================================
  # JSON SCHEMA RESERVED FIELDS
  # ==========================================================================

  Scenario: JSON Schema reserved field conflict detected
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE", "validateFields": true}
      """
    Then the response status should be 200
    When I POST "/subjects/reserved-json/versions" with body:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"email\":{\"type\":\"string\"}}}",
        "metadata": {
          "properties": {"confluent:reserved": "email"}
        }
      }
      """
    Then the response status should be 409
    And the response body should contain "conflicts with the reserved field"
    # Reset
    When I PUT "/config" with body:
      """
      {"compatibility": "BACKWARD", "validateFields": null}
      """
    Then the response status should be 200
