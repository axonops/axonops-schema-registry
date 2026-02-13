@functional
Feature: Mode Management — Exhaustive (Confluent v8.1.1 Compatibility)
  Comprehensive mode management tests covering READWRITE, READONLY, and IMPORT
  modes from the Confluent Schema Registry v8.1.1 test suite.

  # ==========================================================================
  # DEFAULT MODE
  # ==========================================================================

  Scenario: Default global mode is READWRITE
    When I get the global mode
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  # ==========================================================================
  # READONLY MODE
  # ==========================================================================

  Scenario: Set global mode to READONLY blocks writes
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READONLY"
    When I register a schema under subject "mode-ex-readonly":
      """
      {"type":"record","name":"RO","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    # Reset
    When I set the global mode to "READWRITE"

  Scenario: READONLY mode still allows read operations
    Given the global mode is "READWRITE"
    And subject "mode-ex-ro-read" has schema:
      """
      {"type":"record","name":"RORead","fields":[{"name":"a","type":"string"}]}
      """
    When I set the global mode to "READONLY"
    And I get the latest version of subject "mode-ex-ro-read"
    Then the response status should be 200
    When I list all subjects
    Then the response status should be 200
    When I get the global config
    Then the response status should be 200
    # Reset
    When I set the global mode to "READWRITE"

  # ==========================================================================
  # SUBJECT MODE OVERRIDES
  # ==========================================================================

  Scenario: Subject mode overrides global mode
    When I set the global mode to "READONLY"
    When I PUT "/mode/mode-ex-subj-override" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    When I GET "/mode/mode-ex-subj-override"
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"
    # Can register under overridden subject
    When I register a schema under subject "mode-ex-subj-override":
      """
      {"type":"record","name":"Override","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    # Reset
    When I set the global mode to "READWRITE"

  Scenario: Delete subject mode falls back to global
    Given the global mode is "READWRITE"
    When I PUT "/mode/mode-ex-del-fallback" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    When I DELETE "/mode/mode-ex-del-fallback"
    Then the response status should be 200
    When I GET "/mode/mode-ex-del-fallback?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  @pending-impl
  Scenario: Get mode for subject with no override and no defaultToGlobal returns 404
    When I GET "/mode/mode-ex-no-override"
    Then the response status should be 404
    And the response should have error code 40409

  Scenario: Get mode for subject with defaultToGlobal returns global mode
    When I GET "/mode/mode-ex-dtg?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  # ==========================================================================
  # IMPORT MODE
  # ==========================================================================

  Scenario: Import mode allows registration with explicit ID
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I POST "/subjects/mode-ex-import/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 100, "version": 1}
      """
    Then the response status should be 200
    And the response field "id" should be 100
    # Reset
    When I set the global mode to "READWRITE"

  Scenario: Import same schema with same ID but different subject succeeds
    When I set the global mode to "IMPORT"
    When I POST "/subjects/mode-ex-import-reuse1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportReuse\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 200, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/mode-ex-import-reuse2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ImportReuse\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 200, "version": 1}
      """
    Then the response status should be 200
    # Reset
    When I set the global mode to "READWRITE"

  Scenario: Import different schema with same ID fails
    When I set the global mode to "IMPORT"
    When I POST "/subjects/mode-ex-import-conflict1/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 300, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/mode-ex-import-conflict2/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 300, "version": 1}
      """
    Then the response status should be 422
    # Reset
    When I set the global mode to "READWRITE"

  Scenario: Import mode skips compatibility checking
    When I set the global mode to "IMPORT"
    When I POST "/subjects/mode-ex-import-nocompat/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IC1\",\"fields\":[{\"name\":\"f1\",\"type\":\"string\"}]}", "id": 400, "version": 1}
      """
    Then the response status should be 200
    # Register incompatible schema — import mode doesn't check
    When I POST "/subjects/mode-ex-import-nocompat/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IC2\",\"fields\":[{\"name\":\"f1\",\"type\":\"int\"}]}", "id": 401, "version": 2}
      """
    Then the response status should be 200
    # Reset
    When I set the global mode to "READWRITE"

  Scenario: Register without ID after exiting import mode auto-assigns ID
    When I set the global mode to "IMPORT"
    When I POST "/subjects/mode-ex-exit-import/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 500, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I register a schema under subject "mode-ex-exit-import-auto":
      """
      {"type":"record","name":"AutoID","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 200
    And the response should have field "id"
