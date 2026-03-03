@functional @analysis
Feature: REST Subject History and Export
  REST endpoints for viewing schema history, counting versions, and exporting schemas.

  # --- GET /subjects/{subject}/history ---

  Scenario: Get history with one version
    Given subject "history-one-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I GET "/subjects/history-one-value/history"
    Then the response status should be 200
    And the response field "subject" should be "history-one-value"
    And the response field "count" should be 1

  Scenario: Get history with two versions
    Given the global compatibility level is "NONE"
    And subject "history-two-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "history-two-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    When I GET "/subjects/history-two-value/history"
    Then the response status should be 200
    And the response field "count" should be 2

  Scenario: History for nonexistent subject returns 404
    When I GET "/subjects/nonexistent-history-value/history"
    Then the response status should be 404

  # --- GET /subjects/{subject}/versions/count ---

  Scenario: Count versions for subject
    Given subject "vcount-one-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/subjects/vcount-one-value/versions/count"
    Then the response status should be 200
    And the response field "subject" should be "vcount-one-value"
    And the response field "count" should be 1

  Scenario: Count versions after second version
    Given the global compatibility level is "NONE"
    And subject "vcount-two-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    And subject "vcount-two-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"age","type":"int"}]}
      """
    When I GET "/subjects/vcount-two-value/versions/count"
    Then the response status should be 200
    And the response field "count" should be 2

  Scenario: Count versions nonexistent subject returns 404
    When I GET "/subjects/nonexistent-vcount-value/versions/count"
    Then the response status should be 404

  # --- GET /subjects/{subject}/export ---

  Scenario: Export all versions
    Given subject "export-all-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I GET "/subjects/export-all-value/export"
    Then the response status should be 200
    And the response field "subject" should be "export-all-value"
    And the response field "count" should be 1
    And the response field "versions" should be an array

  Scenario: Export nonexistent subject returns 404
    When I GET "/subjects/nonexistent-export-value/export"
    Then the response status should be 404

  # --- GET /subjects/{subject}/versions/{version}/export ---

  Scenario: Export version 1
    Given subject "export-v1-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I GET "/subjects/export-v1-value/versions/1/export"
    Then the response status should be 200
    And the response should have field "schema"
    And the response should have field "schema_type"
    And the response should have field "compatibility_level"
    And the response field "version" should be 1

  Scenario: Export nonexistent version returns 404
    Given subject "export-nover-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/subjects/export-nover-value/versions/99/export"
    Then the response status should be 404

  Scenario: Export invalid version returns 400
    Given subject "export-badver-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/subjects/export-badver-value/versions/abc/export"
    Then the response status should be 400

  Scenario: Export returns compatibility level
    Given subject "export-compat-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "export-compat-value" has compatibility level "FULL"
    When I GET "/subjects/export-compat-value/versions/1/export"
    Then the response status should be 200
    And the response field "compatibility_level" should be "FULL"
