@functional @concurrency
Feature: Concurrent Operations Safety
  As a schema registry operator
  I want the registry to handle truly concurrent goroutine operations safely
  So that parallel clients do not corrupt data or produce inconsistent results

  Background:
    Given the schema registry is running
    And no subjects exist
    And the global compatibility level is "NONE"

  Scenario: Concurrent schema registration produces unique IDs
    When 10 goroutines each register a unique Avro schema to separate subjects
    Then all concurrent results should succeed
    And all returned schema IDs should be unique

  Scenario: Concurrent identical registration is idempotent
    When 10 goroutines register the same Avro schema to subject "idempotent-subject"
    Then all concurrent results should succeed
    And all returned schema IDs should be identical
    And subject "idempotent-subject" should have exactly 1 version

  Scenario: Concurrent deletes do not corrupt state
    Given 10 subjects each with one Avro schema
    When 10 goroutines each soft-delete their own subject
    Then all concurrent results should succeed
    And GET /subjects should return an empty array

  Scenario: Concurrent reads during writes return consistent data
    Given subject "rw-subject" has schema:
      """
      {"type":"record","name":"Seed","fields":[{"name":"x","type":"int"}]}
      """
    When 5 writer goroutines add versions and 5 reader goroutines read latest from subject "rw-subject"
    Then no concurrent results should have a 500 status
    And all reader responses should contain a valid schema

  Scenario: Mode switch blocks concurrent writes
    When I set the global mode to "READONLY"
    And 5 goroutines attempt to register schemas to subject "blocked-subject"
    Then all concurrent results should have status 422
