@functional @analysis
Feature: REST Registry Statistics
  REST endpoints for registry-wide statistics, field consistency checks, and pattern detection.

  # --- GET /statistics ---

  Scenario: Statistics with empty registry
    When I GET "/statistics"
    Then the response status should be 200
    And the response field "subject_count" should be 0
    And the response field "version_count" should be 0

  Scenario: Statistics with schemas registered
    Given subject "stats-user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "stats-order-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"double"}]}
      """
    When I GET "/statistics"
    Then the response status should be 200
    And the response field "subject_count" should be 2

  Scenario: Statistics includes type counts
    Given subject "stats-type-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/statistics"
    Then the response status should be 200
    And the response should have field "type_counts"
    And the response should contain "AVRO"

  # --- GET /statistics/fields/{field} ---

  Scenario: Field consistent same type
    Given subject "field-con-a-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "field-con-b-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"total","type":"double"}]}
      """
    When I GET "/statistics/fields/id"
    Then the response status should be 200
    And the response field "field" should be "id"
    And the response field "consistent" should be true

  Scenario: Field inconsistent different types
    Given subject "field-incon-a-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    And subject "field-incon-b-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"string"}]}
      """
    When I GET "/statistics/fields/id"
    Then the response status should be 200
    And the response field "consistent" should be false

  Scenario: Field not found returns empty usages
    Given subject "field-nf-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/statistics/fields/nonexistent_field"
    Then the response status should be 200
    And the response field "usages" should be an array of length 0

  Scenario: Field consistency includes type_counts
    Given subject "field-tc-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I GET "/statistics/fields/name"
    Then the response status should be 200
    And the response should have field "type_counts"

  # --- GET /statistics/patterns ---

  Scenario: Detect patterns empty registry
    When I GET "/statistics/patterns"
    Then the response status should be 200
    And the response field "common_fields" should be an array of length 0

  Scenario: Detect patterns with shared fields
    Given subject "pattern-a-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "pattern-b-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"double"}]}
      """
    When I GET "/statistics/patterns"
    Then the response status should be 200
    And the response should contain "id"

  Scenario: Detect patterns no common fields
    Given subject "nopattern-a-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And subject "nopattern-b-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"amount","type":"double"}]}
      """
    When I GET "/statistics/patterns"
    Then the response status should be 200
    And the response field "pattern_count" should be 0

  Scenario: Pattern detection returns subject count
    Given subject "patsub-a-value" has schema:
      """
      {"type":"record","name":"A","fields":[{"name":"id","type":"int"}]}
      """
    And subject "patsub-b-value" has schema:
      """
      {"type":"record","name":"B","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/statistics/patterns"
    Then the response status should be 200
    And the response field "subject_count" should be 2
