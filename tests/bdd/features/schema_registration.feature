@functional
Feature: Schema Registration
  As a developer, I want to register schemas so they can be used for data validation

  Scenario: Register first Avro schema
    When I register a schema under subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    And the response should have field "id"

  Scenario: Register returns same ID for duplicate schema
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Register second version increments version
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And the global compatibility level is "NONE"
    When I register a schema under subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    Then the response status should be 200
    And the response should have field "id"

  Scenario: Get schema by ID
    When I register a schema under subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And I store the response field "id" as "schema_id"
    And I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "User"

  Scenario: Get schema by subject and version
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I get version 1 of subject "user-value"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response field "subject" should be "user-value"

  Scenario: Get latest version of subject
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And the global compatibility level is "NONE"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    When I get the latest version of subject "user-value"
    Then the response status should be 200
    And the response field "version" should be 2

  Scenario: Lookup existing schema in subject
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I lookup schema in subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    And the response field "version" should be 1

  Scenario: Lookup non-existing schema returns 404
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I lookup schema in subject "user-value":
      """
      {"type":"record","name":"Other","fields":[{"name":"id","type":"long"}]}
      """
    Then the response status should be 404

  Scenario: Register invalid schema returns 422
    When I register a schema under subject "bad-value":
      """
      {invalid json
      """
    Then the response status should be 422
