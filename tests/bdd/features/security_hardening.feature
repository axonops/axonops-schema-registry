@functional @security
Feature: Security Hardening
  Verify that the schema registry does not leak internal implementation details
  in error responses, and that all error responses follow the standard format.

  Background:
    Given the schema registry is running

  # ---------------------------------------------------------------------------
  # Error response format validation
  # ---------------------------------------------------------------------------

  Scenario: 404 error returns standard JSON format without internal details
    When I GET "/subjects/non-existent-subject-12345/versions"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should have error code 40401
    And the response should not contain "panic"
    And the response should not contain "goroutine"
    And the response should not contain "runtime error"

  Scenario: 422 error returns user-facing message without stack traces
    When I POST "/subjects/security-test-value/versions" with body:
      """
      {"schema": "not valid json schema {{{"}
      """
    Then the response status should be 422
    And the response should be valid JSON
    And the response should not contain "panic"
    And the response should not contain "goroutine"
    And the response should not contain ".go:"

  Scenario: Invalid compatibility level returns clean error
    When I PUT "/config" with body:
      """
      {"compatibility": "INVALID_LEVEL"}
      """
    Then the response status should be 422
    And the response should be valid JSON
    And the response should not contain "panic"
    And the response should not contain "runtime error"

  Scenario: Method not allowed returns standard JSON error
    When I PATCH "/subjects"
    Then the response status should be 405
    And the response should be valid JSON
    And the response should not contain "panic"
    And the response should not contain "goroutine"

  Scenario: Not found returns standard JSON error without file paths
    When I GET "/nonexistent/path/that/does/not/exist"
    Then the response status should be 404
    And the response should be valid JSON
    And the response should not contain "panic"
    And the response should not contain ".go:"
    And the response should not contain "internal/"

  # ---------------------------------------------------------------------------
  # DEK key material protection
  # ---------------------------------------------------------------------------

  Scenario: DEK GET response strips plaintext key material
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"security-km-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123:key/sec"}
      """
    And I POST "/dek-registry/v1/keks/security-km-kek/deks" with body:
      """
      {"subject":"security.pii","algorithm":"AES256_GCM","encryptedKeyMaterial":"c2VjdXJpdHk="}
      """
    When I GET "/dek-registry/v1/keks/security-km-kek/deks/security.pii"
    Then the response status should be 200
    And the response field "encryptedKeyMaterial" should be "c2VjdXJpdHk="
    And the response field "keyMaterial" should be empty or absent

  Scenario: DEK version GET response strips plaintext key material
    Given I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"security-ver-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123:key/ver"}
      """
    And I POST "/dek-registry/v1/keks/security-ver-kek/deks" with body:
      """
      {"subject":"security.version","algorithm":"AES256_GCM","encryptedKeyMaterial":"dmVyc2lvbg=="}
      """
    When I GET "/dek-registry/v1/keks/security-ver-kek/deks/security.version/versions/1"
    Then the response status should be 200
    And the response field "encryptedKeyMaterial" should be "dmVyc2lvbg=="
    And the response field "keyMaterial" should be empty or absent
