@functional @contexts
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
    And the audit log should contain an event:
      | event_type           | schema_register                                       |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | test-subj                                             |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           | sha256:*                                              |
      | context              | .url-ctx                                              |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /contexts/.url-ctx/subjects/test-subj/versions        |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | schema_register                                       |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | get-test                                              |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           | sha256:*                                              |
      | context              | .url-ctx2                                             |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /contexts/.url-ctx2/subjects/get-test/versions        |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | schema_register                                       |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | latest-test                                           |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           | sha256:*                                              |
      | context              | .url-ctx3                                             |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /contexts/.url-ctx3/subjects/latest-test/versions     |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | schema_register                                       |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | subj-b                                                |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           | sha256:*                                              |
      | context              | .url-list                                             |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /contexts/.url-list/subjects/subj-b/versions          |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | schema_register                                       |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | versioned                                             |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           | sha256:*                                              |
      | context              | .url-ver                                              |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /contexts/.url-ver/subjects/versioned/versions        |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | schema_lookup                                         |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | *                                                     |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           |                                                       |
      | context              | .url-lookup                                           |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /contexts/.url-lookup/subjects/lookup-s               |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | subject_delete_soft                                   |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | to-delete                                             |
      | schema_id            |                                                       |
      | version              |                                                       |
      | schema_type          | AVRO                                                  |
      | before_hash          | sha256:*                                              |
      | after_hash           |                                                       |
      | context              | .url-del                                              |
      | transport_security   | tls                                                   |
      | method               | DELETE                                                |
      | path                 | /contexts/.url-del/subjects/to-delete                 |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | config_update                                         |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | config                                                |
      | target_id            | cfg-test                                              |
      | schema_id            |                                                       |
      | version              |                                                       |
      | schema_type          |                                                       |
      | before_hash          | *                                                     |
      | after_hash           | sha256:*                                              |
      | context              | .url-cfg                                              |
      | transport_security   | tls                                                   |
      | method               | PUT                                                   |
      | path                 | /contexts/.url-cfg/config/cfg-test                    |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | mode_update                                           |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | mode                                                  |
      | target_id            | mode-test                                             |
      | schema_id            |                                                       |
      | version              |                                                       |
      | schema_type          |                                                       |
      | before_hash          | *                                                     |
      | after_hash           | sha256:*                                              |
      | context              | .url-mode                                             |
      | transport_security   | tls                                                   |
      | method               | PUT                                                   |
      | path                 | /contexts/.url-mode/mode/mode-test                    |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | schema_register                                       |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | byid-s                                                |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           | sha256:*                                              |
      | context              | .url-byid                                             |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /contexts/.url-byid/subjects/byid-s/versions          |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | schema_register                                       |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | cross-s                                               |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           | sha256:*                                              |
      | context              | .cross-val                                            |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /contexts/.cross-val/subjects/cross-s/versions        |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |

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
    And the audit log should contain an event:
      | event_type           | schema_register                                       |
      | outcome              | success                                               |
      | actor_id             |                                                       |
      | actor_type           | anonymous                                             |
      | auth_method          |                                                       |
      | role                 |                                                       |
      | target_type          | subject                                               |
      | target_id            | :.cross-val2:cross-s2                                 |
      | schema_id            | *                                                     |
      | version              | *                                                     |
      | schema_type          | AVRO                                                  |
      | before_hash          |                                                       |
      | after_hash           | sha256:*                                              |
      | context              | .cross-val2                                           |
      | transport_security   | tls                                                   |
      | method               | POST                                                  |
      | path                 | /subjects/:.cross-val2:cross-s2/versions              |
      | status_code          | 200                                                   |
      | reason               |                                                       |
      | error                |                                                       |
      | request_body         |                                                       |
      | metadata             |                                                       |
      | timestamp            | *                                                     |
      | duration_ms          | *                                                     |
      | request_id           | *                                                     |
      | source_ip            | *                                                     |
      | user_agent           | *                                                     |
