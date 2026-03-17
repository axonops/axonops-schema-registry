@auth @rate-limiting
Feature: Rate limiting
  The schema registry supports configurable rate limiting to protect
  against excessive request volume.

  Background:
    Given I authenticate as "admin" with password "admin-password"

  Scenario: Requests within rate limit succeed
    When I GET "/subjects"
    Then the response status should be 200
    When I GET "/subjects"
    Then the response status should be 200
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Rate limit exceeded returns 429
    When I send 20 rapid requests to "/subjects"
    Then at least one response should have status 429
