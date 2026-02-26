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

  Scenario: IMPORT mode allows registration with explicit ID
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-import-with-id/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportWithId\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 99990}
      """
    Then the response status should be 200
    And the response field "id" should be 99990
    When I set the global mode to "READWRITE"

  Scenario: IMPORT mode rejects different schema with same ID
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-import-dup1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportDup1\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 99991}
      """
    Then the response status should be 200
    And the response field "id" should be 99991
    # Try to register a DIFFERENT schema with the SAME ID
    When I POST "/subjects/mode-import-dup2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportDup2\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}", "id": 99991}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: IMPORT mode allows same schema with same ID in different subject
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-import-share1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportShare\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 99992}
      """
    Then the response status should be 200
    And the response field "id" should be 99992
    # Same schema content, same ID, different subject — should succeed
    When I POST "/subjects/mode-import-share2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportShare\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 99992}
      """
    Then the response status should be 200
    And the response field "id" should be 99992
    When I set the global mode to "READWRITE"

  Scenario: IMPORT mode rejects registration without explicit ID
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I register a schema under subject "mode-import-no-id":
      """
      {"type":"record","name":"ImportNoId","fields":[{"name":"b","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
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

  # ==========================================================================
  # Explicit ID enforcement — explicit schema IDs in register requests
  # are only allowed when the mode is IMPORT.
  # ==========================================================================

  Scenario: Explicit ID in READWRITE mode is rejected with 42205
    Given the global mode is "READWRITE"
    When I POST "/subjects/mode-rw-explicit/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ExplicitRW\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12345}
      """
    Then the response status should be 422
    And the response should have error code 42205

  Scenario: Explicit ID in IMPORT mode succeeds
    Given the global mode is "IMPORT"
    When I POST "/subjects/mode-import-explicit/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ExplicitImp\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12345}
      """
    Then the response status should be 200
    And the response field "id" should be 12345
    When I set the global mode to "READWRITE"

  Scenario: Per-subject IMPORT mode allows explicit ID
    Given the global mode is "READWRITE"
    When I set the mode for subject "mode-subj-import" to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-subj-import/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SubjImp\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 12346}
      """
    Then the response status should be 200
    And the response field "id" should be 12346
