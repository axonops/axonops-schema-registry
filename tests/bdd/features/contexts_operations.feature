@functional @contexts
Feature: Contexts — Full API Operations
  Verify all schema registry API operations work correctly with context-prefixed
  subjects using the :.contextname:subject format.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # SCHEMA REGISTRATION & RETRIEVAL
  # ==========================================================================

  Scenario: Register and retrieve schema via qualified subject
    When I POST "/subjects/:.ops-ctx:register-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"OpsRegister\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    When I GET "/subjects/:.ops-ctx:register-test/versions/1"
    Then the response status should be 200
    And the response field "version" should be 1
    And the response body should contain "OpsRegister"
    And the audit log should contain an event:
      | event_type           | schema_register                         |
      | outcome              | success                                 |
      | actor_id             |                                         |
      | actor_type           | anonymous                               |
      | auth_method          |                                         |
      | role                 |                                         |
      | target_type          | subject                                 |
      | target_id            | :.ops-ctx:register-test                 |
      | schema_id            | *                                       |
      | version              | *                                       |
      | schema_type          | AVRO                                    |
      | before_hash          |                                         |
      | after_hash           | sha256:*                                |
      | context              | .ops-ctx                                |
      | transport_security   | tls                                     |
      | method               | POST                                    |
      | path                 | /subjects/:.ops-ctx:register-test/versions |
      | status_code          | 200                                     |
      | reason               |                                         |
      | error                |                                         |
      | request_body         |                                         |
      | metadata             |                                         |
      | timestamp            | *                                       |
      | duration_ms          | *                                       |
      | request_id           | *                                       |
      | source_ip            | *                                       |
      | user_agent           | *                                       |

  Scenario: Get latest version via qualified subject
    When I POST "/subjects/:.ops-ctx2:latest-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LatestTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ops-ctx2:latest-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LatestTest\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.ops-ctx2:latest-test/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 2
    And the audit log should contain event "schema_register" with subject ":.ops-ctx2:latest-test"

  Scenario: List versions for context-prefixed subject
    When I POST "/subjects/:.ops-ctx3:list-ver/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ListVer\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ops-ctx3:list-ver/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ListVer\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"int\",\"default\":0}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.ops-ctx3:list-ver/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain integer 1
    And the response array should contain integer 2
    And the audit log should contain event "schema_register" with subject ":.ops-ctx3:list-ver"

  # ==========================================================================
  # SCHEMA LOOKUP
  # ==========================================================================

  Scenario: Lookup schema via qualified subject
    When I POST "/subjects/:.ops-ctx4:lookup-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LookupOps\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "lookup_id"
    When I POST "/subjects/:.ops-ctx4:lookup-test" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"LookupOps\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "lookup_id"
    And the audit log should contain event "schema_lookup" with subject ":.ops-ctx4:lookup-test"

  Scenario: Lookup non-existent schema in context returns 404
    When I POST "/subjects/:.ops-ctx5:s1/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Exists\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ops-ctx5:s1" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"DoesNotExist\",\"fields\":[{\"name\":\"x\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 404

  # ==========================================================================
  # DELETE OPERATIONS
  # ==========================================================================

  Scenario: Soft-delete subject via qualified subject
    When I POST "/subjects/:.ops-ctx6:to-delete/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SoftDel\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/:.ops-ctx6:to-delete"
    Then the response status should be 200
    When I GET "/subjects/:.ops-ctx6:to-delete/versions"
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type           | subject_delete_soft                    |
      | outcome              | success                                |
      | actor_id             |                                        |
      | actor_type           | anonymous                              |
      | auth_method          |                                        |
      | role                 |                                        |
      | target_type          | subject                                |
      | target_id            | :.ops-ctx6:to-delete                   |
      | schema_id            |                                        |
      | version              |                                        |
      | schema_type          |                                        |
      | before_hash          | sha256:*                               |
      | after_hash           |                                        |
      | context              | .ops-ctx6                              |
      | transport_security   | tls                                    |
      | method               | DELETE                                 |
      | path                 | /subjects/:.ops-ctx6:to-delete         |
      | status_code          | 200                                    |
      | reason               |                                        |
      | error                |                                        |
      | request_body         |                                        |
      | metadata             |                                        |
      | timestamp            | *                                      |
      | duration_ms          | *                                      |
      | request_id           | *                                      |
      | source_ip            | *                                      |
      | user_agent           | *                                      |

  Scenario: Permanently delete subject via qualified subject
    When I POST "/subjects/:.ops-ctx7:to-perm-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"PermDel\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/:.ops-ctx7:to-perm-del"
    Then the response status should be 200
    When I DELETE "/subjects/:.ops-ctx7:to-perm-del?permanent=true"
    Then the response status should be 200
    And the audit log should contain event "subject_delete_permanent" with subject ":.ops-ctx7:to-perm-del"

  Scenario: Delete specific version via qualified subject
    When I POST "/subjects/:.ops-ctx8:ver-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"VerDel\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ops-ctx8:ver-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"VerDel\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/:.ops-ctx8:ver-del/versions/1"
    Then the response status should be 200
    # Version 2 still exists
    When I GET "/subjects/:.ops-ctx8:ver-del/versions/2"
    Then the response status should be 200
    And the audit log should contain event "schema_delete_soft" with subject ":.ops-ctx8:ver-del"

  # ==========================================================================
  # COMPATIBILITY
  # ==========================================================================

  Scenario: Compatibility check via qualified subject
    When I POST "/subjects/:.ops-ctx9:compat-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CompatOps\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I set the config for subject ":.ops-ctx9:compat-test" to "BACKWARD"
    When I POST "/compatibility/subjects/:.ops-ctx9:compat-test/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CompatOps\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"},{\"name\":\"b\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be true

  Scenario: Incompatible schema detected in context
    When I POST "/subjects/:.ops-ctx10:compat-fail/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IncompatOps\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I set the config for subject ":.ops-ctx10:compat-fail" to "BACKWARD"
    When I POST "/compatibility/subjects/:.ops-ctx10:compat-fail/versions/latest" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"IncompatOps\",\"fields\":[{\"name\":\"a\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the response field "is_compatible" should be false

  # ==========================================================================
  # SUBJECTS BY SCHEMA ID
  # ==========================================================================

  Scenario: Get subjects for schema ID in context via URL prefix
    When I POST "/subjects/:.ops-ctx11:subj-by-id/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"SubjById\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "sbid"
    When I GET "/contexts/.ops-ctx11/schemas/ids/{{sbid}}/subjects"
    Then the response status should be 200
    And the response array should contain "subj-by-id"
    And the audit log should contain event "schema_register" with subject ":.ops-ctx11:subj-by-id"

  Scenario: Re-registering same schema returns existing (idempotent)
    When I POST "/subjects/:.ops-ctx12:idempotent/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Idempotent\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    # Register same schema again — should return same ID
    When I POST "/subjects/:.ops-ctx12:idempotent/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Idempotent\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "first_id"
    And the audit log should contain event "schema_register" with subject ":.ops-ctx12:idempotent"

  Scenario: Get non-existent subject in context returns 404
    When I GET "/subjects/:.ops-ctx13:nonexistent/versions"
    Then the response status should be 404

  Scenario: Get non-existent version in context returns 404
    When I POST "/subjects/:.ops-ctx14:exists/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ExistsVer\",\"fields\":[{\"name\":\"a\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/:.ops-ctx14:exists/versions/99"
    Then the response status should be 404

  Scenario: Schema types endpoint is global (not context-scoped)
    When I GET "/schemas/types"
    Then the response status should be 200
    And the response array should contain "AVRO"
    And the response array should contain "PROTOBUF"
    And the response array should contain "JSON"
