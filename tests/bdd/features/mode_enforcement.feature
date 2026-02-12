@functional
Feature: Mode Enforcement
  Confluent Schema Registry supports READWRITE, READONLY, READONLY_OVERRIDE,
  and IMPORT modes. When a subject or global mode is READONLY or READONLY_OVERRIDE,
  write operations are blocked with error code 42205.

  # ==========================================================================
  # READONLY MODE — BLOCKS ALL WRITES
  # ==========================================================================

  Scenario: READONLY mode blocks schema registration
    Given the global mode is "READONLY"
    When I register a schema under subject "mode-ro-reg":
      """
      {"type":"record","name":"RO","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    # Reset mode
    When I set the global mode to "READWRITE"

  Scenario: READONLY mode blocks subject deletion
    Given the global mode is "READWRITE"
    And subject "mode-ro-del" has schema:
      """
      {"type":"record","name":"RODel","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I DELETE "/subjects/mode-ro-del"
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READONLY mode blocks version deletion
    Given the global mode is "READWRITE"
    And subject "mode-ro-delv" has schema:
      """
      {"type":"record","name":"RODelV","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I DELETE "/subjects/mode-ro-delv/versions/1"
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READONLY mode still allows GET operations
    Given the global mode is "READWRITE"
    And subject "mode-ro-get" has schema:
      """
      {"type":"record","name":"ROGet","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I get the latest version of subject "mode-ro-get"
    Then the response status should be 200
    When I list all subjects
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  # ==========================================================================
  # PER-SUBJECT READONLY MODE
  # ==========================================================================

  Scenario: Per-subject READONLY blocks writes only on that subject
    Given the global mode is "READWRITE"
    When I set the mode for subject "mode-per-ro" to "READONLY"
    Then the response status should be 200
    When I register a schema under subject "mode-per-ro":
      """
      {"type":"record","name":"PerRO","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    # Other subjects still work
    When I register a schema under subject "mode-per-rw":
      """
      {"type":"record","name":"PerRW","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200

  # ==========================================================================
  # READONLY_OVERRIDE MODE
  # ==========================================================================

  Scenario: READONLY_OVERRIDE is a valid mode
    When I set the global mode to "READONLY_OVERRIDE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READONLY_OVERRIDE"
    When I set the global mode to "READWRITE"

  Scenario: READONLY_OVERRIDE blocks schema registration
    Given the global mode is "READONLY_OVERRIDE"
    When I register a schema under subject "mode-override-reg":
      """
      {"type":"record","name":"Override","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READONLY_OVERRIDE allows changing mode back
    When I set the global mode to "READONLY_OVERRIDE"
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READWRITE"

  # ==========================================================================
  # DELETE /mode/{subject}
  # ==========================================================================

  Scenario: DELETE /mode/{subject} removes subject mode
    When I set the mode for subject "mode-del-test" to "READONLY"
    Then the response status should be 200
    When I GET "/mode/mode-del-test"
    Then the response field "mode" should be "READONLY"
    When I delete the mode for subject "mode-del-test"
    Then the response status should be 200
    When I GET "/mode/mode-del-test"
    Then the response status should be 404

  Scenario: DELETE /mode/{subject} when no mode returns 404
    When I delete the mode for subject "mode-del-nonexist"
    Then the response status should be 404

  # ==========================================================================
  # IMPORT MODE
  # ==========================================================================

  Scenario: IMPORT mode allows registration
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I register a schema under subject "mode-import-test":
      """
      {"type":"record","name":"Import","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  # ==========================================================================
  # INVALID MODE
  # ==========================================================================

  Scenario: Invalid mode value returns 42204
    When I PUT "/mode" with body:
      """
      {"mode": "INVALID_MODE"}
      """
    Then the response status should be 422
    And the response should have error code 42204
