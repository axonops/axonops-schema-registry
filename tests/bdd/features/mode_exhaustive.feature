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

  # ==========================================================================
  # IMPORT MODE — EXPLICIT VERSION SUPPORT
  # ==========================================================================

  Scenario: Import with explicit version honors the version number
    When I set the global mode to "IMPORT"
    When I POST "/subjects/mode-ex-ver-explicit/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 600, "version": 5}
      """
    Then the response status should be 200
    And the response field "id" should be 600
    When I set the global mode to "READWRITE"
    When I get version 5 of subject "mode-ex-ver-explicit"
    Then the response status should be 200
    And the response field "version" should be 5
    And the response field "id" should be 600

  Scenario: Import multiple versions out of order
    When I set the global mode to "IMPORT"
    # Import version 3 first
    When I POST "/subjects/mode-ex-ver-ooo/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"OOO\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"},{\"name\":\"c\",\"type\":\"string\"}]}", "id": 612, "version": 3}
      """
    Then the response status should be 200
    # Then import version 1
    When I POST "/subjects/mode-ex-ver-ooo/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"OOO\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}", "id": 610, "version": 1}
      """
    Then the response status should be 200
    # Then import version 2
    When I POST "/subjects/mode-ex-ver-ooo/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"OOO\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\"}]}", "id": 611, "version": 2}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    # All three versions should be retrievable
    When I get version 1 of subject "mode-ex-ver-ooo"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response field "id" should be 610
    When I get version 2 of subject "mode-ex-ver-ooo"
    Then the response status should be 200
    And the response field "version" should be 2
    And the response field "id" should be 611
    When I get version 3 of subject "mode-ex-ver-ooo"
    Then the response status should be 200
    And the response field "version" should be 3
    And the response field "id" should be 612

  Scenario: Import with non-sequential version gaps
    When I set the global mode to "IMPORT"
    When I POST "/subjects/mode-ex-ver-gaps/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 620, "version": 1}
      """
    Then the response status should be 200
    When I POST "/subjects/mode-ex-ver-gaps/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 621, "version": 10}
      """
    Then the response status should be 200
    When I POST "/subjects/mode-ex-ver-gaps/versions" with body:
      """
      {"schema": "{\"type\":\"long\"}", "id": 622, "version": 100}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I list versions of subject "mode-ex-ver-gaps"
    Then the response status should be 200
    And the response should be an array of length 3

  Scenario: Import without explicit version auto-assigns next version
    When I set the global mode to "IMPORT"
    When I POST "/subjects/mode-ex-ver-auto/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 630}
      """
    Then the response status should be 200
    When I POST "/subjects/mode-ex-ver-auto/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 631}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    When I get version 1 of subject "mode-ex-ver-auto"
    Then the response status should be 200
    And the response field "id" should be 630
    When I get version 2 of subject "mode-ex-ver-auto"
    Then the response status should be 200
    And the response field "id" should be 631

  Scenario: Import with duplicate version returns existing version
    When I set the global mode to "IMPORT"
    When I POST "/subjects/mode-ex-ver-dup/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 640, "version": 1}
      """
    Then the response status should be 200
    # Same version, different schema — Confluent allows this (returns existing)
    When I POST "/subjects/mode-ex-ver-dup/versions" with body:
      """
      {"schema": "{\"type\":\"int\"}", "id": 641, "version": 1}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"

  # ==========================================================================
  # IMPORT MODE — MUTUAL EXCLUSION WITH READWRITE
  # ==========================================================================

  Scenario: IMPORT mode rejects normal registration (no explicit ID)
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I register a schema under subject "mode-ex-import-noid":
      """
      {"type":"record","name":"NoID","fields":[{"name":"a","type":"string"}]}
      """
    Then the response status should be 422
    And the response should have error code 42205
    When I set the global mode to "READWRITE"

  Scenario: READWRITE mode rejects registration with explicit ID
    When I set the global mode to "READWRITE"
    Then the response status should be 200
    When I POST "/subjects/mode-ex-rw-with-id/versions" with body:
      """
      {"schema": "{\"type\":\"string\"}", "id": 650}
      """
    Then the response status should be 422
    And the response should have error code 42205
