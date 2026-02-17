@functional
Feature: Error Codes Exhaustive
  Verify all Confluent Schema Registry error codes are returned correctly.

  # ==========================================================================
  # 40401 — Subject not found
  # ==========================================================================

  Scenario: 40401 on GET versions of non-existent subject
    When I GET "/subjects/err-no-subject/versions"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: 40401 on GET specific version of non-existent subject
    When I GET "/subjects/err-no-subject2/versions/1"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: 40401 on DELETE non-existent subject
    When I DELETE "/subjects/err-no-subject3"
    Then the response status should be 404
    And the response should have error code 40401

  Scenario: 40402 on compatibility check against specific version of non-existent subject
    When I POST "/compatibility/subjects/err-no-subject4/versions/1" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"X\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 404
    And the response should have error code 40402

  # ==========================================================================
  # 40402 — Version not found
  # ==========================================================================

  Scenario: 40402 on GET non-existent version
    Given subject "err-ver-nf" has schema:
      """
      {"type":"record","name":"VNF","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/err-ver-nf/versions/99"
    Then the response status should be 404
    And the response should have error code 40402

  Scenario: 40402 on DELETE non-existent version
    Given subject "err-ver-nf2" has schema:
      """
      {"type":"record","name":"VNF2","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/err-ver-nf2/versions/99"
    Then the response status should be 404
    And the response should have error code 40402

  Scenario: 40402 on compatibility check against non-existent version
    Given subject "err-ver-nf3" has schema:
      """
      {"type":"record","name":"VNF3","fields":[{"name":"a","type":"string"}]}
      """
    When I POST "/compatibility/subjects/err-ver-nf3/versions/99" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"VNF3\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 404
    And the response should have error code 40402

  # ==========================================================================
  # 40403 — Schema not found
  # ==========================================================================

  Scenario: 40403 on GET non-existent schema ID
    When I GET "/schemas/ids/999999"
    Then the response status should be 404
    And the response should have error code 40403

  # ==========================================================================
  # 40405 — Subject not soft-deleted (permanent delete without prior soft-delete)
  # ==========================================================================

  Scenario: 40405 on permanent delete subject without soft-delete
    Given subject "err-perm-sub" has schema:
      """
      {"type":"record","name":"PS","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/err-perm-sub?permanent=true"
    Then the response status should be 404
    And the response should have error code 40405

  Scenario: 40407 on permanent delete version without soft-delete
    Given subject "err-perm-ver" has schema:
      """
      {"type":"record","name":"PV","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/err-perm-ver/versions/1?permanent=true"
    Then the response status should be 404
    And the response should have error code 40407

  # ==========================================================================
  # 409 — Incompatible schema
  # ==========================================================================

  Scenario: 409 when registering incompatible schema
    Given the global compatibility level is "BACKWARD"
    And subject "err-incompat" has schema:
      """
      {"type":"record","name":"IC","fields":[{"name":"a","type":"string"}]}
      """
    When I register a schema under subject "err-incompat":
      """
      {"type":"record","name":"IC","fields":[{"name":"a","type":"string"},{"name":"b","type":"string"}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # 42201 — Invalid schema
  # ==========================================================================

  Scenario: 42201 on invalid Avro schema
    When I POST "/subjects/err-invalid-avro/versions" with body:
      """
      {"schema": "not valid json at all"}
      """
    Then the response status should be 422
    And the response should have error code 42201

  # ==========================================================================
  # 42202 — Invalid version
  # ==========================================================================

  Scenario: 42202 on GET with version 0
    Given subject "err-inv-ver" has schema:
      """
      {"type":"record","name":"IV","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/err-inv-ver/versions/0"
    Then the response status should be 422
    And the response should have error code 42202

  Scenario: 42202 on GET with version abc
    Given subject "err-inv-ver2" has schema:
      """
      {"type":"record","name":"IV2","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects/err-inv-ver2/versions/abc"
    Then the response status should be 422
    And the response should have error code 42202

  # ==========================================================================
  # 42203 — Invalid compatibility level
  # ==========================================================================

  Scenario: 42203 on invalid compatibility level
    When I PUT "/config" with body:
      """
      {"compatibility": "INVALID"}
      """
    Then the response status should be 422
    And the response should have error code 42203

  # ==========================================================================
  # 42204 — Invalid mode
  # ==========================================================================

  Scenario: 42204 on invalid mode
    When I PUT "/mode" with body:
      """
      {"mode": "INVALID"}
      """
    Then the response status should be 422
    And the response should have error code 42204

  # ==========================================================================
  # 42205 — Operation not permitted (mode enforcement)
  # ==========================================================================

  Scenario: 42205 when registering in READONLY mode
    Given the global mode is "READONLY"
    When I register a schema under subject "err-readonly":
      """
      {"type":"record","name":"RO","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"
