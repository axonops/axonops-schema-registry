@functional
Feature: Schema Listing
  As a developer, I want to list and query schemas across subjects

  Scenario: List all schemas
    Given subject "sub-a" has schema:
      """
      {"type":"record","name":"A","fields":[{"name":"x","type":"string"}]}
      """
    And subject "sub-b" has schema:
      """
      {"type":"record","name":"B","fields":[{"name":"y","type":"long"}]}
      """
    When I list all schemas
    Then the response status should be 200
    And the response should be valid JSON

  Scenario: Get subjects for a schema ID
    When I register a schema under subject "shared-value":
      """
      {"type":"record","name":"Shared","fields":[{"name":"id","type":"long"}]}
      """
    And I store the response field "id" as "schema_id"
    And I get the subjects for the stored schema ID
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain "shared-value"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | shared-value                             |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/shared-value/versions          |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
