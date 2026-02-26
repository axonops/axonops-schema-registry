@functional @axonops-only
Feature: HTTP 405 Method Not Allowed
  The registry MUST return HTTP 405 with a JSON error body for unsupported HTTP methods.
  This ensures Confluent wire compatibility for API clients.

  Background:
    Given the schema registry is running

  Scenario: POST to GET-only /schemas/types endpoint returns 405
    When I POST "/schemas/types" with body:
      """
      {"type": "AVRO"}
      """
    Then the response status should be 405
    And the response should have error code 405

  Scenario: DELETE to GET-only /schemas/types endpoint returns 405
    When I DELETE "/schemas/types"
    Then the response status should be 405
    And the response should have error code 405

  Scenario: PUT to GET-only /schemas/types endpoint returns 405
    When I PUT "/schemas/types" with body:
      """
      {}
      """
    Then the response status should be 405
    And the response should have error code 405

  Scenario: POST to GET-only /schemas endpoint returns 405
    When I POST "/schemas" with body:
      """
      {"schema": "{}"}
      """
    Then the response status should be 405
    And the response should have error code 405

  Scenario: PUT to GET-only /subjects endpoint returns 405
    When I PUT "/subjects" with body:
      """
      {"subject": "test"}
      """
    Then the response status should be 405
    And the response should have error code 405

  Scenario: DELETE to GET-only /schemas/ids/{id} endpoint returns 405
    When I DELETE "/schemas/ids/1"
    Then the response status should be 405
    And the response should have error code 405

  Scenario: PUT to GET/DELETE-only /subjects/{subject}/versions/{version} returns 405
    Given subject "test-value" has schema:
      """
      {"type":"record","name":"Test","fields":[{"name":"f","type":"string"}]}
      """
    When I PUT "/subjects/test-value/versions/1" with body:
      """
      {"schema": "{}"}
      """
    Then the response status should be 405
    And the response should have error code 405

  Scenario: 405 response contains Method Not Allowed message
    When I DELETE "/schemas/types"
    Then the response status should be 405
    And the response should contain "Method Not Allowed"

  Scenario: 405 response has correct Confluent content type
    When I POST "/schemas/types" with body:
      """
      {"type": "AVRO"}
      """
    Then the response status should be 405
    And the response header "Content-Type" should contain "application/vnd.schemaregistry"

  Scenario: DELETE to POST-only /subjects/{subject}/versions endpoint returns 405
    Given subject "test-value" has schema:
      """
      {"type":"record","name":"Test","fields":[{"name":"f","type":"string"}]}
      """
    When I DELETE "/subjects/test-value/versions"
    Then the response status should be 405
    And the response should have error code 405

  Scenario: PUT to DELETE-only /subjects/{subject} endpoint returns 405
    Given subject "test-value" has schema:
      """
      {"type":"record","name":"Test","fields":[{"name":"f","type":"string"}]}
      """
    When I PUT "/subjects/test-value" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 405
    And the response should have error code 405
