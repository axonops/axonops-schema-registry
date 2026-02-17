@functional @axonops-only
Feature: API Documentation
  As a developer integrating with the schema registry,
  I want to access interactive API documentation and the OpenAPI specification
  so that I can explore available endpoints and their contracts.

  Scenario: Swagger UI is accessible at /docs
    When I GET "/docs"
    Then the response status should be 200
    And the response header "Content-Type" should contain "text/html"
    And the response body should contain "swagger-ui"

  Scenario: OpenAPI spec is accessible at /openapi.yaml
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response header "Content-Type" should contain "text/yaml"
    And the response body should contain "openapi:"

  Scenario: OpenAPI spec documents schema endpoints
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "/schemas/types"
    And the response body should contain "/schemas/ids/{id}"
    And the response body should contain "/schemas/ids/{id}/schema"
    And the response body should contain "/schemas/ids/{id}/subjects"
    And the response body should contain "/schemas/ids/{id}/versions"

  Scenario: OpenAPI spec documents subject endpoints
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "/subjects"
    And the response body should contain "/subjects/{subject}/versions"
    And the response body should contain "/subjects/{subject}/versions/{version}"
    And the response body should contain "/subjects/{subject}/versions/{version}/schema"
    And the response body should contain "/subjects/{subject}/versions/{version}/referencedby"

  Scenario: OpenAPI spec documents config and mode endpoints
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "/config"
    And the response body should contain "/config/{subject}"
    And the response body should contain "/mode"
    And the response body should contain "/mode/{subject}"

  Scenario: OpenAPI spec documents compatibility endpoints
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "/compatibility/subjects/{subject}/versions/{version}"
    And the response body should contain "/compatibility/subjects/{subject}/versions"

  Scenario: OpenAPI spec documents import and metadata endpoints
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "/import/schemas"
    And the response body should contain "/contexts"
    And the response body should contain "/v1/metadata/id"
    And the response body should contain "/v1/metadata/version"

  Scenario: OpenAPI spec documents admin endpoints
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "/admin/users"
    And the response body should contain "/admin/users/{id}"
    And the response body should contain "/admin/apikeys"
    And the response body should contain "/admin/apikeys/{id}"
    And the response body should contain "/admin/apikeys/{id}/revoke"
    And the response body should contain "/admin/apikeys/{id}/rotate"
    And the response body should contain "/admin/roles"

  Scenario: OpenAPI spec documents account endpoints
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "/me"
    And the response body should contain "/me/password"

  Scenario: OpenAPI spec includes security schemes
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "basicAuth"
    And the response body should contain "apiKey"
    And the response body should contain "bearerAuth"

  Scenario: OpenAPI spec includes error code definitions
    When I GET "/openapi.yaml"
    Then the response status should be 200
    And the response body should contain "ErrorResponse"
    And the response body should contain "error_code"
