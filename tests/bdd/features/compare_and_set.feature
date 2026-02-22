@functional
Feature: Compare-and-Set (confluent:version)
  The confluent:version metadata property is a soft hint for optimistic concurrency
  control. Confluent treats mismatches as hints, NOT hard errors — the schema is
  always registered normally. When confluent:version is specified during dedup,
  it must match the existing version for dedup to fire; otherwise a new version
  is created. After registration, confluent:version is auto-populated in the
  response with the actual assigned version number.

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
  # EXPLICIT VERSION — MISMATCH CASES
  # Confluent treats confluent:version as a soft hint, not a hard constraint.
  # Mismatches do NOT produce errors — the schema is registered normally.
  # ==========================================================================

  Scenario: confluent:version mismatch is treated as soft hint — schema registered normally
    # Register v1
    When I POST "/subjects/cas-conflict/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasConflict\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # confluent:version=1 but next expected is 2 — Confluent registers normally
    When I POST "/subjects/cas-conflict/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasConflict\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}",
        "metadata": {
          "properties": {"confluent:version": "1"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  Scenario: confluent:version with gap is treated as soft hint — schema registered normally
    # Register v1
    When I POST "/subjects/cas-gap/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasGap\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # confluent:version=5 but next expected is 2 — Confluent registers normally
    When I POST "/subjects/cas-gap/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasGap\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}",
        "metadata": {
          "properties": {"confluent:version": "5"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  # ==========================================================================
  # EXPLICIT VERSION ON EMPTY SUBJECT — SOFT HINT
  # ==========================================================================

  Scenario: confluent:version=2 on empty subject is treated as soft hint — registered as v1
    # No previous versions, confluent:version=2 — Confluent registers as v1 normally
    When I POST "/subjects/cas-empty-v2/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasEmptyV2\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:version": "2"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  # ==========================================================================
  # NON-NUMERIC confluent:version — TREATED AS AUTO-INCREMENT
  # ==========================================================================

  Scenario: confluent:version with non-numeric value is ignored
    # "abc" is not a valid integer, so it should be treated as auto-increment
    When I POST "/subjects/cas-nonnumeric/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasNonNumeric\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:version": "abc"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  # ==========================================================================
  # SEQUENTIAL CAS REGISTRATION (v1, v2, v3)
  # ==========================================================================

  Scenario: Sequential CAS registration (v1, v2, v3)
    # Register v1 with confluent:version=1
    When I POST "/subjects/cas-sequential/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasSeq\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:version": "1"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    # Register v2 with confluent:version=2
    When I POST "/subjects/cas-sequential/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasSeq\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}",
        "metadata": {
          "properties": {"confluent:version": "2"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    # Register v3 with confluent:version=3
    When I POST "/subjects/cas-sequential/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasSeq\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"},{\"name\":\"c\",\"type\":\"string\",\"default\":\"\"}]}",
        "metadata": {
          "properties": {"confluent:version": "3"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    # Verify all three versions are registered
    When I GET "/subjects/cas-sequential/versions"
    Then the response status should be 200
    Then the response body should contain "1"
    Then the response body should contain "2"
    Then the response body should contain "3"

  # ==========================================================================
  # CAS AFTER SOFT-DELETE
  # ==========================================================================

  Scenario: CAS after soft-delete succeeds with version=2
    # Register v1
    When I POST "/subjects/cas-softdel/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasSoftDel\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # Soft-delete the subject
    When I DELETE "/subjects/cas-softdel"
    Then the response status should be 200
    # Register v2 with confluent:version=2 — soft-deleted versions count
    When I POST "/subjects/cas-softdel/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasSoftDel\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}",
        "metadata": {
          "properties": {"confluent:version": "2"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"

  # ==========================================================================
  # METADATA PROPERTIES PRESERVED ALONGSIDE confluent:version
  # ==========================================================================

  Scenario: confluent:version in metadata with other properties preserved
    # Register with confluent:version=1 and additional custom properties
    When I POST "/subjects/cas-meta-props/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasMetaProps\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {
            "confluent:version": "1",
            "owner": "team-data",
            "env": "test"
          }
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    # Retrieve the version and verify other properties are preserved
    When I GET "/subjects/cas-meta-props/versions/1"
    Then the response status should be 200
    Then the response body should contain "owner"
    Then the response body should contain "team-data"
    Then the response body should contain "env"
    Then the response body should contain "test"

  # ==========================================================================
  # confluent:version AUTO-POPULATED IN RESPONSE
  # ==========================================================================

  @axonops-only
  Scenario: confluent:version auto-populated in response
    # Register schema without explicit confluent:version
    When I POST "/subjects/cas-auto-pop/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasAutoPop\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"
      }
      """
    Then the response status should be 200
    # GET the version and verify confluent:version is set
    When I GET "/subjects/cas-auto-pop/versions/1"
    Then the response status should be 200
    Then the response body should contain "confluent:version"

  # ==========================================================================
  # DUPLICATE REGISTRATION WITH confluent:version RETURNS SAME ID (DEDUP)
  # ==========================================================================

  Scenario: Duplicate registration with confluent:version returns same ID
    # Register v1 with confluent:version=1
    When I POST "/subjects/cas-dedup/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasDedup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:version": "1"}
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    # Register the exact same schema and metadata again
    When I POST "/subjects/cas-dedup/versions" with body:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"CasDedup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}",
        "metadata": {
          "properties": {"confluent:version": "1"}
        }
      }
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"
