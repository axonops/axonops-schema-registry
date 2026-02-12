@functional
Feature: Subject Filtering and Deleted Parameters
  Test query parameter filtering on subjects, schema IDs, and version endpoints
  including ?subjectPrefix, ?deleted=true on various endpoints.

  # ==========================================================================
  # GET /subjects?subjectPrefix=...
  # ==========================================================================

  Scenario: List subjects with subjectPrefix filters by prefix
    Given subject "filter-alpha-one" has schema:
      """
      {"type":"record","name":"A1","fields":[{"name":"x","type":"string"}]}
      """
    And subject "filter-alpha-two" has schema:
      """
      {"type":"record","name":"A2","fields":[{"name":"x","type":"string"}]}
      """
    And subject "filter-beta-one" has schema:
      """
      {"type":"record","name":"B1","fields":[{"name":"x","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=filter-alpha"
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "filter-alpha-one"
    And the response array should contain "filter-alpha-two"

  Scenario: List subjects with non-matching prefix returns empty array
    Given subject "filter-xxx" has schema:
      """
      {"type":"record","name":"Xxx","fields":[{"name":"x","type":"string"}]}
      """
    When I GET "/subjects?subjectPrefix=nomatch-zzz"
    Then the response status should be 200
    And the response should be an array of length 0

  Scenario: List subjects with subjectPrefix and deleted combines both filters
    Given subject "filter-del-a" has schema:
      """
      {"type":"record","name":"DA","fields":[{"name":"x","type":"string"}]}
      """
    And subject "filter-del-b" has schema:
      """
      {"type":"record","name":"DB","fields":[{"name":"x","type":"string"}]}
      """
    When I delete subject "filter-del-b"
    Then the response status should be 200
    When I GET "/subjects?subjectPrefix=filter-del"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain "filter-del-a"
    When I GET "/subjects?subjectPrefix=filter-del&deleted=true"
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "filter-del-a"
    And the response array should contain "filter-del-b"

  # ==========================================================================
  # GET /schemas/ids/{id}/subjects?deleted=true
  # ==========================================================================

  Scenario: Schema subjects endpoint hides soft-deleted subject by default
    When I register a schema under subject "sfilt-id-sub-a":
      """
      {"type":"record","name":"Shared","fields":[{"name":"k","type":"string"},{"name":"v","type":"int"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "sfilt-id-sub-b":
      """
      {"type":"record","name":"Shared","fields":[{"name":"k","type":"string"},{"name":"v","type":"int"}]}
      """
    Then the response status should be 200
    When I delete subject "sfilt-id-sub-b"
    Then the response status should be 200
    When I GET "/schemas/ids/1/subjects"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain "sfilt-id-sub-a"

  Scenario: Schema subjects endpoint shows soft-deleted subject with deleted=true
    When I register a schema under subject "sfilt-id-del-a":
      """
      {"type":"record","name":"SharedDel","fields":[{"name":"k","type":"string"},{"name":"v","type":"int"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "sfilt-id-del-b":
      """
      {"type":"record","name":"SharedDel","fields":[{"name":"k","type":"string"},{"name":"v","type":"int"}]}
      """
    Then the response status should be 200
    When I delete subject "sfilt-id-del-b"
    Then the response status should be 200
    When I GET "/schemas/ids/1/subjects?deleted=true"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # GET /schemas/ids/{id}/versions?deleted=true
  # ==========================================================================

  Scenario: Schema versions endpoint hides soft-deleted subject-version by default
    When I register a schema under subject "sfilt-ver-a":
      """
      {"type":"record","name":"SharedVer","fields":[{"name":"k","type":"string"},{"name":"v","type":"int"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "sfilt-ver-b":
      """
      {"type":"record","name":"SharedVer","fields":[{"name":"k","type":"string"},{"name":"v","type":"int"}]}
      """
    Then the response status should be 200
    When I delete subject "sfilt-ver-b"
    Then the response status should be 200
    When I get versions for schema ID 1
    Then the response status should be 200
    And the response should be an array of length 1

  Scenario: Schema versions endpoint shows soft-deleted with deleted=true
    When I register a schema under subject "sfilt-verd-a":
      """
      {"type":"record","name":"SharedVerD","fields":[{"name":"k","type":"string"},{"name":"v","type":"int"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "sfilt-verd-b":
      """
      {"type":"record","name":"SharedVerD","fields":[{"name":"k","type":"string"},{"name":"v","type":"int"}]}
      """
    Then the response status should be 200
    When I delete subject "sfilt-verd-b"
    Then the response status should be 200
    When I GET "/schemas/ids/1/versions?deleted=true"
    Then the response status should be 200
    And the response should be an array of length 2
