@functional
Feature: Compare-and-Set (confluent:version)
  The confluent:version metadata property enables optimistic concurrency control
  for schema registration. When set, the registry verifies the version matches
  the expected next version for the subject.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # AUTO-INCREMENT (confluent:version=0 or -1 or absent)
  # ==========================================================================

  Scenario: confluent:version absent — auto-increment succeeds
    When I POST "/subjects/cas-auto/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasAuto\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  Scenario: confluent:version=0 — auto-increment succeeds
    When I POST "/subjects/cas-zero/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasZero\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:version": "0"}
        }
      }
      """
    Then the response status should be 200

  Scenario: confluent:version=-1 — auto-increment succeeds
    When I POST "/subjects/cas-neg1/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasNeg1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:version": "-1"}
        }
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # EXPLICIT VERSION — SUCCESS CASES
  # ==========================================================================

  Scenario: confluent:version=1 on new subject succeeds
    When I POST "/subjects/cas-v1-new/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasV1New\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:version": "1"}
        }
      }
      """
    Then the response status should be 200

  Scenario: confluent:version=2 after v1 exists succeeds
    # Register v1
    When I POST "/subjects/cas-v2-after-v1/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasV2\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # Register v2 with confluent:version=2
    When I POST "/subjects/cas-v2-after-v1/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasV2\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}",
        "metadata": {
          "properties": {"confluent:version": "2"}
        }
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # EXPLICIT VERSION — FAILURE CASES
  # ==========================================================================

  @axonops-only
  Scenario: confluent:version=1 when v1 already exists fails
    # Register v1
    When I POST "/subjects/cas-conflict/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasConflict\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # Try confluent:version=1 again — should fail
    When I POST "/subjects/cas-conflict/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasConflict\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}",
        "metadata": {
          "properties": {"confluent:version": "1"}
        }
      }
      """
    Then the response status should be 409

  @axonops-only
  Scenario: confluent:version=5 when latest is v1 fails (gap)
    # Register v1
    When I POST "/subjects/cas-gap/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasGap\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # Try confluent:version=5 — gap, should fail
    When I POST "/subjects/cas-gap/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasGap\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}",
        "metadata": {
          "properties": {"confluent:version": "5"}
        }
      }
      """
    Then the response status should be 409
