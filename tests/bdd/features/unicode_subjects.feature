@functional @edge-case
Feature: Unicode and Special Character Subject Names
  As a schema registry user
  I want to use Unicode and special characters in subject names
  So that I can support international naming conventions and diverse use cases

  Background:
    Given the schema registry is running
    And the global compatibility level is "NONE"

  # ---------------------------------------------------------------------------
  # Basic Unicode subject names
  # ---------------------------------------------------------------------------

  Scenario: Register schema with Unicode subject name containing CJK characters
    When I register a schema under subject "test-subject":
      """
      {"type":"record","name":"CjkTest","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "test-subject"
    Then the response status should be 200
    And the response field "subject" should be "test-subject"

  # ---------------------------------------------------------------------------
  # Subject names with dots and hyphens
  # ---------------------------------------------------------------------------

  Scenario: Subject name with dots is valid
    When I register a schema under subject "com.example.events.user-created":
      """
      {"type":"record","name":"DotSubject","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "com.example.events.user-created"
    Then the response status should be 200
    And the response field "subject" should be "com.example.events.user-created"

  Scenario: Subject name with underscores and numbers
    When I register a schema under subject "my_subject_123_v2":
      """
      {"type":"record","name":"UnderscoreSubj","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    When I list all subjects
    Then the response status should be 200
    And the response array should contain "my_subject_123_v2"

  # ---------------------------------------------------------------------------
  # Subject names with Kafka topic conventions
  # ---------------------------------------------------------------------------

  Scenario: Subject name following Kafka TopicNameStrategy (topic-value)
    When I register a schema under subject "orders.events-value":
      """
      {"type":"record","name":"OrderEvent","fields":[{"name":"order_id","type":"string"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "orders.events-key":
      """
      {"type":"record","name":"OrderKey","fields":[{"name":"key","type":"string"}]}
      """
    Then the response status should be 200
    When I list all subjects
    Then the response status should be 200
    And the response array should contain "orders.events-value"
    And the response array should contain "orders.events-key"

  # ---------------------------------------------------------------------------
  # Special character edge cases
  # ---------------------------------------------------------------------------

  Scenario: Subject name with colons is valid
    When I register a schema under subject "ns:my-subject:v1":
      """
      {"type":"record","name":"ColonSubj","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    When I list all subjects
    Then the response status should be 200
    And the response array should contain "ns:my-subject:v1"

  Scenario: Subject name with tilde is valid
    When I register a schema under subject "test~subject":
      """
      {"type":"record","name":"TildeSubj","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "test~subject"
    Then the response status should be 200
    And the response field "subject" should be "test~subject"

  # ---------------------------------------------------------------------------
  # Long subject names
  # ---------------------------------------------------------------------------

  Scenario: Long subject name (200 characters) is valid
    When I register a schema under subject "very-long-subject-name-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":
      """
      {"type":"record","name":"LongSubj","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # Delete and re-register with special characters
  # ---------------------------------------------------------------------------

  Scenario: Delete and re-register subject with dots in name
    When I register a schema under subject "com.example.delete-test":
      """
      {"type":"record","name":"DelDot","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    When I delete subject "com.example.delete-test"
    Then the response status should be 200
    When I register a schema under subject "com.example.delete-test":
      """
      {"type":"record","name":"DelDotV2","fields":[{"name":"id","type":"string"},{"name":"name","type":"string","default":""}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "com.example.delete-test"
    Then the response status should be 200
    And the response should contain "DelDotV2"

  # ---------------------------------------------------------------------------
  # Subject names used in compatibility checks
  # ---------------------------------------------------------------------------

  Scenario: Compatibility check works with dotted subject names
    Given I set the global compatibility level to "BACKWARD"
    When I register a schema under subject "com.example.compat-test":
      """
      {"type":"record","name":"CompatDot","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    # Backward-compatible change (add optional field)
    When I check compatibility of schema against subject "com.example.compat-test":
      """
      {"type":"record","name":"CompatDot","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}
      """
    Then the compatibility check should be compatible
