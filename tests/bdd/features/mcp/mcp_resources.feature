@mcp @mcp-resources
Feature: MCP Resources — Read-Only Registry State
  An AI agent reads MCP resources to get context about the schema registry
  state without using tools. Resources provide read-only snapshots of
  server info, subjects, schemas, configurations, and more.

  # ==========================================================================
  # 1. STATIC RESOURCES
  # ==========================================================================

  Scenario: Read server info resource
    When I read MCP resource "schema://server/info"
    Then the MCP resource result should contain "AVRO"
    And the MCP resource result should contain "PROTOBUF"
    And the MCP resource result should contain "JSON"

  Scenario: Read schema types resource
    When I read MCP resource "schema://types"
    Then the MCP resource result should contain "AVRO"
    And the MCP resource result should contain "PROTOBUF"
    And the MCP resource result should contain "JSON"

  Scenario: Read server config resource
    When I read MCP resource "schema://server/config"
    Then the MCP resource result should contain "compatibility"
    And the MCP resource result should contain "mode"

  Scenario: Read contexts resource
    When I read MCP resource "schema://contexts"
    Then the MCP resource result should contain "."

  Scenario: Read subjects resource when empty
    When I read MCP resource "schema://subjects"
    Then the MCP resource result should contain "[]"

  # ==========================================================================
  # 2. SUBJECT RESOURCES (TEMPLATED)
  # ==========================================================================

  Scenario: Read subjects resource after registration
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-test-subject",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I read MCP resource "schema://subjects"
    Then the MCP resource result should contain "res-test-subject"

  Scenario: Read subject detail resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-detail-test",
        "schema": "{\"type\":\"record\",\"name\":\"Detail\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I read MCP resource "schema://subjects/res-detail-test"
    Then the MCP resource result should contain "res-detail-test"
    And the MCP resource result should contain "latest"

  Scenario: Read subject versions resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-versions-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I read MCP resource "schema://subjects/res-versions-test/versions"
    Then the MCP resource result should contain "1"

  Scenario: Read subject version detail resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-version-detail",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I read MCP resource "schema://subjects/res-version-detail/versions/1"
    Then the MCP resource result should contain "res-version-detail"
    And the MCP resource result should contain "AVRO"

  Scenario: Read subject config resource
    When I call MCP tool "set_config" with input:
      | subject             | res-config-test |
      | compatibility_level | FULL            |
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-config-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I read MCP resource "schema://subjects/res-config-test/config"
    Then the MCP resource result should contain "FULL"

  Scenario: Read subject mode resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-mode-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I read MCP resource "schema://subjects/res-mode-test/mode"
    Then the MCP resource result should contain "mode"

  # ==========================================================================
  # 3. SCHEMA RESOURCES (TEMPLATED)
  # ==========================================================================

  Scenario: Read schema by ID resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-schema-id-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    When I read MCP resource "schema://schemas/1"
    Then the MCP resource result should contain "AVRO"

  Scenario: Read schema subjects resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-schema-subjects-a",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-schema-subjects-b",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I read MCP resource "schema://schemas/1/subjects"
    Then the MCP resource result should contain "res-schema-subjects-a"
    And the MCP resource result should contain "res-schema-subjects-b"

  Scenario: Read schema versions resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-schema-versions-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I read MCP resource "schema://schemas/1/versions"
    Then the MCP resource result should contain "res-schema-versions-test"

  # ==========================================================================
  # 4. ADDITIONAL STATIC RESOURCES
  # ==========================================================================

  Scenario: Read global mode resource
    When I read MCP resource "schema://mode"
    Then the MCP resource result should contain "mode"

  Scenario: Read KEKs list resource
    When I read MCP resource "schema://keks"
    Then the MCP resource result should contain "["

  Scenario: Read exporters list resource
    When I read MCP resource "schema://exporters"
    Then the MCP resource result should contain "[]"

  Scenario: Read server status resource
    When I read MCP resource "schema://status"
    Then the MCP resource result should contain "healthy"

  # ==========================================================================
  # 5. ADDITIONAL TEMPLATED RESOURCES
  # ==========================================================================

  Scenario: Read context subjects resource
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "res-ctx-subjects-test",
        "schema": "{\"type\":\"string\"}"
      }
      """
    When I read MCP resource "schema://contexts/./subjects"
    Then the MCP resource result should contain "res-ctx-subjects-test"
