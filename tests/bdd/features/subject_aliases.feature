@functional
Feature: Subject Aliases
  Subject aliases allow a subject to be accessed via an alternative name.
  When a config has an "alias" field set, requests to the alias subject
  are resolved to the actual subject.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # ALIAS CONFIGURATION
  # ==========================================================================

  Scenario: Set alias via config
    When I PUT "/config/my-alias" with body:
      """
      {"compatibility": "BACKWARD", "alias": "alias-target"}
      """
    Then the response status should be 200
    When I GET "/config/my-alias"
    Then the response status should be 200
    And the response field "alias" should be "alias-target"

  Scenario: Remove alias by setting empty string
    When I PUT "/config/removable-alias" with body:
      """
      {"compatibility": "BACKWARD", "alias": "some-target"}
      """
    Then the response status should be 200
    When I PUT "/config/removable-alias" with body:
      """
      {"compatibility": "BACKWARD", "alias": ""}
      """
    Then the response status should be 200
    When I GET "/config/removable-alias"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "BACKWARD"

  # ==========================================================================
  # REGISTER AND GET VIA ALIAS
  # ==========================================================================

  Scenario: Register schema via alias — appears under actual subject
    # Create a schema under the target subject first
    When I POST "/subjects/alias-actual/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AliasTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set alias
    When I PUT "/config/alias-shortcut" with body:
      """
      {"compatibility": "BACKWARD", "alias": "alias-actual"}
      """
    Then the response status should be 200
    # Register via alias — should go to alias-actual
    When I POST "/subjects/alias-shortcut/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AliasTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Verify the schema landed under alias-actual
    When I GET "/subjects/alias-actual/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  Scenario: Get version via alias returns from actual subject
    Given subject "alias-get-target" has schema:
      """
      {"type":"record","name":"AliasGet","fields":[{"name":"val","type":"int"}]}
      """
    When I PUT "/config/alias-get-shortcut" with body:
      """
      {"compatibility": "BACKWARD", "alias": "alias-get-target"}
      """
    Then the response status should be 200
    When I GET "/subjects/alias-get-shortcut/versions/1"
    Then the response status should be 200
    And the response body should contain "AliasGet"

  Scenario: List versions via alias returns versions from actual subject
    Given subject "alias-list-target" has schema:
      """
      {"type":"record","name":"AliasList","fields":[{"name":"v","type":"string"}]}
      """
    When I PUT "/config/alias-list-shortcut" with body:
      """
      {"compatibility": "BACKWARD", "alias": "alias-list-target"}
      """
    Then the response status should be 200
    When I GET "/subjects/alias-list-shortcut/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # LOOKUP AND COMPATIBILITY VIA ALIAS
  # ==========================================================================

  Scenario: Lookup via alias finds schema in actual subject
    Given subject "alias-lookup-target" has schema:
      """
      {"type":"record","name":"AliasLookup","fields":[{"name":"key","type":"string"}]}
      """
    When I PUT "/config/alias-lookup-shortcut" with body:
      """
      {"compatibility": "BACKWARD", "alias": "alias-lookup-target"}
      """
    Then the response status should be 200
    When I POST "/subjects/alias-lookup-shortcut" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AliasLookup\",\"fields\":[{\"name\":\"key\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "subject" should be "alias-lookup-target"

  Scenario: Compatibility check via alias checks against actual subject
    Given subject "alias-compat-target" has schema:
      """
      {"type":"record","name":"AliasCompat","fields":[{"name":"x","type":"int"}]}
      """
    When I set the config for subject "alias-compat-target" to "BACKWARD"
    And I PUT "/config/alias-compat-shortcut" with body:
      """
      {"compatibility": "BACKWARD", "alias": "alias-compat-target"}
      """
    Then the response status should be 200
    # Compatible change via alias
    When I POST "/compatibility/subjects/alias-compat-shortcut/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"AliasCompat\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"},{\"name\":\"y\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # ALIAS DOES NOT RESOLVE RECURSIVELY
  # ==========================================================================

  Scenario: Alias does not resolve recursively
    # Set up chain: A → B → C
    Given subject "alias-chain-c" has schema:
      """
      {"type":"record","name":"ChainC","fields":[{"name":"c","type":"int"}]}
      """
    When I PUT "/config/alias-chain-b" with body:
      """
      {"compatibility": "BACKWARD", "alias": "alias-chain-c"}
      """
    And I PUT "/config/alias-chain-a" with body:
      """
      {"compatibility": "BACKWARD", "alias": "alias-chain-b"}
      """
    # Accessing via alias-chain-a should resolve to alias-chain-b (not alias-chain-c)
    # Since alias-chain-b has no schemas, this should fail
    When I GET "/subjects/alias-chain-a/versions"
    Then the response status should be 404
