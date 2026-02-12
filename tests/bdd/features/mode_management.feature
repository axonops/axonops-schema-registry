@functional
Feature: Mode Management
  As an operator, I want to manage read/write modes globally and per subject

  Scenario: Get default global mode
    When I get the global mode
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  Scenario: Set global mode to READONLY
    When I set the global mode to "READONLY"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "READONLY"

  Scenario: Set global mode to IMPORT
    When I set the global mode to "IMPORT"
    Then the response status should be 200
    When I get the global mode
    Then the response field "mode" should be "IMPORT"

  Scenario: Set per-subject mode
    When I set the mode for subject "my-subject" to "READONLY"
    Then the response status should be 200
    When I get the mode for subject "my-subject"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"

  Scenario: Delete per-subject mode falls back to global with defaultToGlobal
    Given subject "mode-subj" has mode "READONLY"
    When I delete the mode for subject "mode-subj"
    Then the response status should be 200
    When I get the mode for subject "mode-subj"
    Then the response status should be 404
    When I GET "/mode/mode-subj?defaultToGlobal=true"
    Then the response status should be 200
    And the response field "mode" should be "READWRITE"

  Scenario: Per-subject mode isolation
    When I set the mode for subject "subj-a" to "READONLY"
    And I set the mode for subject "subj-b" to "IMPORT"
    When I get the mode for subject "subj-a"
    Then the response field "mode" should be "READONLY"
    When I get the mode for subject "subj-b"
    Then the response field "mode" should be "IMPORT"
