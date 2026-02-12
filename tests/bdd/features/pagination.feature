@functional
Feature: Pagination
  Confluent Schema Registry supports offset/limit pagination on GET /subjects
  and deletedOnly parameter to return only soft-deleted subjects.

  # ==========================================================================
  # OFFSET AND LIMIT ON GET /subjects
  # ==========================================================================

  Scenario: limit restricts number of subjects returned
    Given subject "pag-a" has schema:
      """
      {"type":"record","name":"PagA","fields":[{"name":"a","type":"string"}]}
      """
    And subject "pag-b" has schema:
      """
      {"type":"record","name":"PagB","fields":[{"name":"a","type":"string"}]}
      """
    And subject "pag-c" has schema:
      """
      {"type":"record","name":"PagC","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=pag-&limit=2"
    Then the response status should be 200

  Scenario: offset skips subjects
    Given subject "pago-a" has schema:
      """
      {"type":"record","name":"PagoA","fields":[{"name":"a","type":"string"}]}
      """
    And subject "pago-b" has schema:
      """
      {"type":"record","name":"PagoB","fields":[{"name":"a","type":"string"}]}
      """
    And subject "pago-c" has schema:
      """
      {"type":"record","name":"PagoC","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=pago-&offset=1&limit=1"
    Then the response status should be 200
    And the response should contain "pago-b"
    And the response body should not contain "pago-a"
    And the response body should not contain "pago-c"

  Scenario: offset beyond total returns empty array
    Given subject "pagoff-a" has schema:
      """
      {"type":"record","name":"PagoffA","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=pagoff-&offset=999"
    Then the response status should be 200
    And the response should contain "[]"

  Scenario: limit=-1 returns all subjects (unlimited)
    Given subject "pagunlim-a" has schema:
      """
      {"type":"record","name":"PagUnlimA","fields":[{"name":"a","type":"string"}]}
      """
    And subject "pagunlim-b" has schema:
      """
      {"type":"record","name":"PagUnlimB","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=pagunlim-&limit=-1"
    Then the response status should be 200
    And the response should contain "pagunlim-a"
    And the response should contain "pagunlim-b"

  Scenario: limit=0 returns empty array
    Given subject "pagzero-a" has schema:
      """
      {"type":"record","name":"PagZeroA","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=pagzero-&limit=0"
    Then the response status should be 200
    And the response should contain "[]"

  # ==========================================================================
  # DELETED-ONLY ON GET /subjects
  # ==========================================================================

  Scenario: deletedOnly returns only soft-deleted subjects
    Given subject "pagdel-active" has schema:
      """
      {"type":"record","name":"PagDelActive","fields":[{"name":"a","type":"string"}]}
      """
    And subject "pagdel-deleted" has schema:
      """
      {"type":"record","name":"PagDelDeleted","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/pagdel-deleted"
    Then the response status should be 200
    When I GET "/subjects?subjectPrefix=pagdel-&deletedOnly=true"
    Then the response status should be 200
    And the response should contain "pagdel-deleted"
    And the response body should not contain "pagdel-active"

  Scenario: deletedOnly with no deleted subjects returns empty
    Given subject "pagdeln-a" has schema:
      """
      {"type":"record","name":"PagDelnA","fields":[{"name":"a","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=pagdeln-&deletedOnly=true"
    Then the response status should be 200
    And the response should contain "[]"

  Scenario: deletedOnly takes precedence over deleted
    Given subject "pagdelp-active" has schema:
      """
      {"type":"record","name":"PagDelpActive","fields":[{"name":"a","type":"string"}]}
      """
    And subject "pagdelp-deleted" has schema:
      """
      {"type":"record","name":"PagDelpDeleted","fields":[{"name":"a","type":"string"}]}
      """
    When I DELETE "/subjects/pagdelp-deleted"
    Then the response status should be 200
    When I GET "/subjects?subjectPrefix=pagdelp-&deleted=true&deletedOnly=true"
    Then the response status should be 200
    And the response should contain "pagdelp-deleted"
    And the response body should not contain "pagdelp-active"
