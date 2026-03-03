@functional @analysis
Feature: REST Schema Analysis
  REST endpoints for finding similar schemas, scoring quality, and measuring complexity.

  # --- POST /schemas/similar ---

  Scenario: Find similar schemas with shared fields
    Given subject "similar-user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "similar-order-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"id","type":"int"},{"name":"amount","type":"double"}]}
      """
    When I POST "/schemas/similar" with body:
      """
      {"subject": "similar-user-value", "threshold": 0.1}
      """
    Then the response status should be 200
    And the response field "subject" should be "similar-user-value"
    And the response should contain "similar-order-value"

  Scenario: Similar schemas excludes source subject
    Given subject "similar-src-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/schemas/similar" with body:
      """
      {"subject": "similar-src-value", "threshold": 0.0}
      """
    Then the response status should be 200
    And the response field "count" should be 0

  Scenario: Similar with nonexistent subject returns 404
    When I POST "/schemas/similar" with body:
      """
      {"subject": "nonexistent-subject-value"}
      """
    Then the response status should be 404

  Scenario: Similar with missing subject returns 400
    When I POST "/schemas/similar" with body:
      """
      {"threshold": 0.5}
      """
    Then the response status should be 400

  Scenario: Similar with high threshold returns empty
    Given subject "similar-high1-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "similar-high2-value" has schema:
      """
      {"type":"record","name":"Order","fields":[{"name":"order_id","type":"long"},{"name":"total","type":"double"}]}
      """
    When I POST "/schemas/similar" with body:
      """
      {"subject": "similar-high1-value", "threshold": 1.0}
      """
    Then the response status should be 200
    And the response field "count" should be 0

  # --- POST /schemas/quality ---

  Scenario: Quality score by inline schema
    When I POST "/schemas/quality" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response should have field "overall_score"
    And the response should have field "grade"
    And the response should have field "categories"

  Scenario: Quality score by subject
    Given subject "quality-sub-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/schemas/quality" with body:
      """
      {"subject": "quality-sub-value"}
      """
    Then the response status should be 200
    And the response should have field "grade"

  Scenario: Quality with missing schema and subject returns 400
    When I POST "/schemas/quality" with body:
      """
      {"schemaType": "AVRO"}
      """
    Then the response status should be 400

  Scenario: Quality for nonexistent subject returns 404
    When I POST "/schemas/quality" with body:
      """
      {"subject": "nonexistent-quality-value"}
      """
    Then the response status should be 404

  Scenario: Quality response includes max_score
    When I POST "/schemas/quality" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Test\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response should have field "max_score"
    And the response should have field "overall_score"

  # --- POST /schemas/complexity ---

  Scenario: Complexity by inline schema
    When I POST "/schemas/complexity" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response should have field "field_count"
    And the response should have field "max_depth"
    And the response should have field "grade"

  Scenario: Complexity by subject
    Given subject "complexity-sub-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/schemas/complexity" with body:
      """
      {"subject": "complexity-sub-value"}
      """
    Then the response status should be 200
    And the response should have field "field_count"

  Scenario: Complexity with missing schema and subject returns 400
    When I POST "/schemas/complexity" with body:
      """
      {"schemaType": "AVRO"}
      """
    Then the response status should be 400

  Scenario: Complexity grade A for simple schema
    When I POST "/schemas/complexity" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Simple\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "grade" should be "A"
    And the response field "field_count" should be 2
