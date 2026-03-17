@functional @analysis
Feature: REST Analysis Edge Cases
  Edge cases and cross-cutting concerns for REST analysis endpoints.

  Scenario: Invalid JSON body to validate endpoint
    When I POST "/schemas/validate" with raw body "{not valid json"
    Then the response status should be 400

  Scenario: Invalid regex in field search
    Given subject "edge-regex-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/schemas/search/field" with body:
      """
      {"field": "[invalid", "mode": "regex"}
      """
    Then the response status should be 400

  Scenario: Quality works with JSON Schema
    When I POST "/schemas/quality" with body:
      """
      {"schema": "{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"integer\"},\"name\":{\"type\":\"string\"}}}", "schemaType": "JSON"}
      """
    Then the response status should be 200
    And the response should have field "grade"

  Scenario: Compare with nonexistent subject returns 404
    Given subject "edge-compare-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/compatibility/compare" with body:
      """
      {"subject1": "edge-compare-value", "subject2": "nonexistent-compare-value"}
      """
    Then the response status should be 404

  Scenario: Diff defaults version1 to 1
    Given the global compatibility level is "NONE"
    And subject "edge-diff-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "edge-diff-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    When I POST "/subjects/edge-diff-value/diff" with body:
      """
      {"version2": 2}
      """
    Then the response status should be 200
    And the response field "version1" should be 1
    And the response field "version2" should be 2

  Scenario: Count subjects empty registry
    When I GET "/subjects/count"
    Then the response status should be 200
    And the response field "count" should be 0
