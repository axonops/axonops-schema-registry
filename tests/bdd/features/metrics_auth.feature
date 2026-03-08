@auth @metrics
Feature: Auth and Rate-Limit Metrics
  Authentication, rate limiting, and credential caching events MUST be
  tracked by Prometheus metrics for operational observability.

  Background:
    Given I authenticate as "admin" with password "admin-password"

  # ---------------------------------------------------------------------------
  # Rate-limit metrics
  # ---------------------------------------------------------------------------

  Scenario: Rate limit hits metric increments when rate limited
    When I send 20 rapid requests to "/subjects"
    Then at least one response should have status 429
    And the Prometheus metric "schema_registry_rate_limit_hits_total" should exist

  # ---------------------------------------------------------------------------
  # Credential cache metrics
  # ---------------------------------------------------------------------------

  Scenario: Cache miss recorded on first authentication
    When I GET "/subjects"
    Then the response status should be 200
    And the Prometheus metric "schema_registry_cache_misses_total" should exist

  Scenario: Cache hit recorded on repeated authentication
    # First request populates the cache
    When I GET "/subjects"
    Then the response status should be 200
    # Second request should hit the cache
    When I GET "/subjects"
    Then the response status should be 200
    And the Prometheus metric "schema_registry_cache_hits_total" should exist
