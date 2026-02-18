@functional
Feature: Contexts — URL Prefix Routing
  Verify that the /contexts/{context}/... URL prefix routes work correctly.
  All schema registry operations should be accessible via URL prefix routing
  as an alternative to qualified subject names.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # SCHEMA REGISTRATION VIA URL PREFIX
  # ==========================================================================

  Scenario: Register schema via URL prefix
    When I POST "/contexts/.url-ctx/subjects/test-subj/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlReg\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1

  Scenario: Retrieve schema via URL prefix
    When I POST "/contexts/.url-ctx2/subjects/get-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlGet\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts/.url-ctx2/subjects/get-test/versions/1"
    Then the response status should be 200
    And the response body should contain "UrlGet"
    And the response field "version" should be 1

  Scenario: Get latest version via URL prefix
    When I POST "/contexts/.url-ctx3/subjects/latest-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlLatest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/contexts/.url-ctx3/subjects/latest-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlLatest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts/.url-ctx3/subjects/latest-test/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 2

  # ==========================================================================
  # SUBJECT OPERATIONS VIA URL PREFIX
  # ==========================================================================

  Scenario: List subjects via URL prefix returns plain names
    When I POST "/contexts/.url-list/subjects/subj-a/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlListA\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/contexts/.url-list/subjects/subj-b/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlListB\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts/.url-list/subjects"
    Then the response status should be 200
    And the response array should contain "subj-a"
    And the response array should contain "subj-b"

  Scenario: List versions via URL prefix
    When I POST "/contexts/.url-ver/subjects/versioned/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlVer\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/contexts/.url-ver/subjects/versioned/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlVer\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    When I GET "/contexts/.url-ver/subjects/versioned/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  Scenario: Lookup schema via URL prefix
    When I POST "/contexts/.url-lookup/subjects/lookup-s/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlLookup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "url_lookup_id"
    When I POST "/contexts/.url-lookup/subjects/lookup-s" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlLookup\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "url_lookup_id"

  Scenario: Delete subject via URL prefix
    When I POST "/contexts/.url-del/subjects/to-delete/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlDel\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/contexts/.url-del/subjects/to-delete"
    Then the response status should be 200
    When I GET "/contexts/.url-del/subjects/to-delete/versions"
    Then the response status should be 404

  # ==========================================================================
  # CONFIG AND MODE VIA URL PREFIX
  # ==========================================================================

  Scenario: Config operations via URL prefix
    When I POST "/contexts/.url-cfg/subjects/cfg-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlCfg\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/contexts/.url-cfg/config/cfg-test" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    When I GET "/contexts/.url-cfg/config/cfg-test"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  Scenario: Mode operations via URL prefix
    When I POST "/contexts/.url-mode/subjects/mode-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlMode\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/contexts/.url-mode/mode/mode-test" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    When I GET "/contexts/.url-mode/mode/mode-test"
    Then the response status should be 200
    And the response field "mode" should be "READONLY"

  # ==========================================================================
  # COMPATIBILITY VIA URL PREFIX
  # ==========================================================================

  Scenario: Compatibility check via URL prefix
    When I POST "/contexts/.url-compat/subjects/compat-s/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlCompat\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/contexts/.url-compat/config/compat-s" with body:
      """
      {"compatibility": "BACKWARD"}
      """
    Then the response status should be 200
    When I POST "/contexts/.url-compat/compatibility/subjects/compat-s/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlCompat\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  # ==========================================================================
  # SCHEMA ID VIA URL PREFIX
  # ==========================================================================

  Scenario: Get schema by ID via URL prefix
    When I POST "/contexts/.url-byid/subjects/byid-s/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UrlById\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "url_byid"
    When I GET "/contexts/.url-byid/schemas/ids/{{url_byid}}"
    Then the response status should be 200
    And the response body should contain "UrlById"

  # ==========================================================================
  # CROSS-VALIDATION: URL PREFIX AND QUALIFIED SUBJECT
  # ==========================================================================

  Scenario: Schema registered via URL prefix is accessible via qualified subject
    When I POST "/contexts/.cross-val/subjects/cross-s/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CrossVal\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "cross_id"
    # Access via qualified subject
    When I GET "/subjects/:.cross-val:cross-s/versions/1"
    Then the response status should be 200
    And the response body should contain "CrossVal"

  Scenario: Schema registered via qualified subject is accessible via URL prefix
    When I POST "/subjects/:.cross-val2:cross-s2/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CrossVal2\",\"fields\":[{\"name\":\"b\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Access via URL prefix
    When I GET "/contexts/.cross-val2/subjects/cross-s2/versions/1"
    Then the response status should be 200
    And the response body should contain "CrossVal2"
