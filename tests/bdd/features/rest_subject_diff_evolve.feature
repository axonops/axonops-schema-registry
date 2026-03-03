@functional @analysis
Feature: REST Schema Diff, Evolution, Migration, and Dependencies
  REST endpoints for diffing schema versions, suggesting evolution, planning migration, and viewing dependencies.

  # --- POST /subjects/{subject}/diff ---

  Scenario: Diff with added field
    Given the global compatibility level is "NONE"
    And subject "diff-add-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    And subject "diff-add-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    When I POST "/subjects/diff-add-value/diff" with body:
      """
      {"version1": 1, "version2": 2}
      """
    Then the response status should be 200
    And the response field "subject" should be "diff-add-value"
    And the response should contain "email"

  Scenario: Diff with removed field
    Given the global compatibility level is "NONE"
    And subject "diff-rm-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    And subject "diff-rm-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/subjects/diff-rm-value/diff" with body:
      """
      {"version1": 1, "version2": 2}
      """
    Then the response status should be 200
    And the response should contain "age"

  Scenario: Diff same version shows no changes
    Given subject "diff-same-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/subjects/diff-same-value/diff" with body:
      """
      {"version1": 1, "version2": 1}
      """
    Then the response status should be 200
    And the response field "version1" should be 1
    And the response field "version2" should be 1

  Scenario: Diff nonexistent subject returns 404
    When I POST "/subjects/nonexistent-diff-value/diff" with body:
      """
      {"version1": 1, "version2": 2}
      """
    Then the response status should be 404

  Scenario: Diff nonexistent version returns 404
    Given subject "diff-nover-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/subjects/diff-nover-value/diff" with body:
      """
      {"version1": 1, "version2": 99}
      """
    Then the response status should be 404

  # --- POST /subjects/{subject}/evolve ---

  Scenario: Evolve returns subject and compat info
    Given subject "evolve-info-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/subjects/evolve-info-value/evolve" with body:
      """
      {"changes": [{"action": "add", "field": "email", "type": "string"}]}
      """
    Then the response status should be 200
    And the response field "subject" should be "evolve-info-value"
    And the response should have field "current_version"
    And the response should have field "compatibility_level"

  Scenario: Evolve nonexistent subject returns 404
    When I POST "/subjects/nonexistent-evolve-value/evolve" with body:
      """
      {"changes": []}
      """
    Then the response status should be 404

  # --- POST /subjects/{subject}/migrate ---

  Scenario: Migrate with field additions
    Given subject "migrate-add-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/subjects/migrate-add-value/migrate" with body:
      """
      {"target_schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response should contain "Add field"
    And the response should contain "email"

  Scenario: Migrate with field removals
    Given subject "migrate-rm-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    When I POST "/subjects/migrate-rm-value/migrate" with body:
      """
      {"target_schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response should contain "Remove field"
    And the response should contain "age"

  Scenario: Migrate identical schemas
    Given subject "migrate-same-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I POST "/subjects/migrate-same-value/migrate" with body:
      """
      {"target_schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response should contain "No migration steps needed"

  Scenario: Migrate missing target_schema returns 400
    Given subject "migrate-noschema-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I POST "/subjects/migrate-noschema-value/migrate" with body:
      """
      {"schema_type": "AVRO"}
      """
    Then the response status should be 400

  Scenario: Migrate nonexistent subject returns 404
    When I POST "/subjects/nonexistent-migrate-value/migrate" with body:
      """
      {"target_schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 404

  # --- GET /subjects/{subject}/versions/{version}/dependencies ---

  Scenario: Dependencies for schema with no refs
    Given subject "deps-norefs-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}
      """
    When I GET "/subjects/deps-norefs-value/versions/1/dependencies"
    Then the response status should be 200
    And the response field "subject" should be "deps-norefs-value"
    And the response field "version" should be 1
    And the response field "referenced_by" should be an array of length 0

  Scenario: Dependencies invalid version returns 400
    Given subject "deps-badver-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/subjects/deps-badver-value/versions/abc/dependencies"
    Then the response status should be 400
