@functional
Feature: Server Info & Misc Endpoints — Exhaustive (Confluent v8.1.1 Compatibility)
  Server information, metadata endpoints, and miscellaneous API tests.

  # ==========================================================================
  # METADATA ENDPOINTS
  # ==========================================================================

  Scenario: Get cluster ID returns valid response
    When I get the cluster ID
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Get server version returns valid response
    When I get the server version
    Then the response status should be 200
    And the response should be valid JSON

  # ==========================================================================
  # SCHEMA TYPES
  # ==========================================================================

  Scenario: Get schema types returns AVRO, JSON, and PROTOBUF
    When I get the schema types
    Then the response status should be 200
    And the response array should contain "AVRO"
    And the response array should contain "JSON"
    And the response array should contain "PROTOBUF"

  # ==========================================================================
  # HEALTH CHECK
  # ==========================================================================

  Scenario: Root endpoint returns 200
    When I GET "/"
    Then the response status should be 200
