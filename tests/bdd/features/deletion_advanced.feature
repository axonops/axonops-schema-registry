@functional
Feature: Advanced Schema Deletion
  Comprehensive edge-case testing for soft-delete and permanent-delete operations
  on versions and subjects, including interactions with references, re-registration,
  and schema ID lookups.

  # ==========================================================================
  # SOFT-DELETE VERSION VISIBILITY
  # ==========================================================================

  Scenario: Soft-deleted version is not visible in version list
    Given the global compatibility level is "NONE"
    And subject "del-adv-vis" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-adv-vis" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "del-adv-vis"
    Then the response status should be 200
    When I list versions of subject "del-adv-vis"
    Then the response status should be 200
    And the response should be an array of length 1
    And the audit log should contain an event:
      | event_type          | schema_delete_soft                   |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-vis                          |
      | schema_id           | *                                    |
      | version             | *                                    |
      | schema_type         | AVRO                                 |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-vis/versions       |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  Scenario: Soft-deleted version is visible with deleted=true query parameter
    Given the global compatibility level is "NONE"
    And subject "del-adv-vis-del" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-adv-vis-del" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "del-adv-vis-del"
    Then the response status should be 200
    When I GET "/subjects/del-adv-vis-del/versions?deleted=true"
    Then the response status should be 200
    And the response should be an array of length 2
    And the audit log should contain an event:
      | event_type          | schema_delete_soft                   |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-vis-del                      |
      | schema_id           | *                                    |
      | version             | *                                    |
      | schema_type         | AVRO                                 |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-vis-del/versions   |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  # ==========================================================================
  # SOFT-DELETE SUBJECT VISIBILITY WITH MULTIPLE SUBJECTS
  # ==========================================================================

  Scenario: Soft-deleted subject visible with deleted=true among multiple subjects
    Given subject "del-adv-multi-a" has schema:
      """
      {"type":"record","name":"A","fields":[{"name":"x","type":"string"}]}
      """
    And subject "del-adv-multi-b" has schema:
      """
      {"type":"record","name":"B","fields":[{"name":"y","type":"string"}]}
      """
    And subject "del-adv-multi-c" has schema:
      """
      {"type":"record","name":"C","fields":[{"name":"z","type":"string"}]}
      """
    When I delete subject "del-adv-multi-b"
    Then the response status should be 200
    When I list all subjects
    Then the response should be an array of length 2
    And the response array should contain "del-adv-multi-a"
    And the response array should contain "del-adv-multi-c"
    When I list subjects with deleted
    Then the response should be an array of length 3
    And the response array should contain "del-adv-multi-b"
    And the audit log should contain an event:
      | event_type          | subject_delete_soft                  |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-multi-b                      |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-multi-b            |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  # ==========================================================================
  # PERMANENT DELETE REMOVES FROM DELETED LIST
  # ==========================================================================

  Scenario: Permanent delete version removes it from deleted=true list too
    Given the global compatibility level is "NONE"
    And subject "del-adv-perm-ver" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-adv-perm-ver" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "del-adv-perm-ver"
    Then the response status should be 200
    When I permanently delete version 1 of subject "del-adv-perm-ver"
    Then the response status should be 200
    When I GET "/subjects/del-adv-perm-ver/versions?deleted=true"
    Then the response status should be 200
    And the response should be an array of length 1
    And the audit log should contain an event:
      | event_type          | schema_delete_permanent              |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-perm-ver                     |
      | schema_id           | *                                    |
      | version             | *                                    |
      | schema_type         | AVRO                                 |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-perm-ver/versions  |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  Scenario: Permanent delete subject removes it from deleted=true list too
    Given subject "del-adv-perm-sub" has schema:
      """
      {"type":"record","name":"Gone","fields":[{"name":"v","type":"string"}]}
      """
    When I delete subject "del-adv-perm-sub"
    Then the response status should be 200
    When I permanently delete subject "del-adv-perm-sub"
    Then the response status should be 200
    When I list subjects with deleted
    Then the response should be an array of length 0
    And the audit log should contain an event:
      | event_type          | subject_delete_permanent             |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-perm-sub                     |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-perm-sub           |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  # ==========================================================================
  # RE-REGISTRATION AFTER DELETION
  # ==========================================================================

  Scenario: Re-register schema after soft-delete subject gets new version number
    Given the global compatibility level is "NONE"
    And subject "del-adv-reregister" has schema:
      """
      {"type":"record","name":"First","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-adv-reregister" has schema:
      """
      {"type":"record","name":"Second","fields":[{"name":"b","type":"string"}]}
      """
    When I delete subject "del-adv-reregister"
    Then the response status should be 200
    When I register a schema under subject "del-adv-reregister":
      """
      {"type":"record","name":"Third","fields":[{"name":"c","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "del-adv-reregister"
    Then the response status should be 200
    And the response field "version" should be 3
    And the audit log should contain an event:
      | event_type          | subject_delete_soft                  |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-reregister                   |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-reregister         |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |
    And the audit log should contain an event:
      | event_type          | schema_register                       |
      | outcome             | success                               |
      | actor_id            |                                       |
      | actor_type          | anonymous                             |
      | auth_method         |                                       |
      | role                |                                       |
      | target_type         | subject                               |
      | target_id           | del-adv-reregister                    |
      | schema_id           | *                                     |
      | version             | *                                     |
      | schema_type         | AVRO                                  |
      | method              | POST                                  |
      | path                | /subjects/del-adv-reregister/versions |
      | status_code         | 200                                   |
      | before_hash         |                                       |
      | after_hash          | sha256:*                              |
      | context             | .                                     |
      | transport_security  | tls                                   |
      | reason              |                                       |
      | error               |                                       |
      | request_body        |                                       |
      | metadata            |                                       |
      | timestamp           | *                                     |
      | duration_ms         | *                                     |
      | request_id          | *                                     |
      | source_ip           | *                                     |
      | user_agent          | *                                     |

  Scenario: Permanent delete then re-register starts from version 1
    Given the global compatibility level is "NONE"
    And subject "del-adv-fresh" has schema:
      """
      {"type":"record","name":"Old","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-adv-fresh" has schema:
      """
      {"type":"record","name":"Older","fields":[{"name":"b","type":"string"}]}
      """
    When I delete subject "del-adv-fresh"
    Then the response status should be 200
    When I permanently delete subject "del-adv-fresh"
    Then the response status should be 200
    When I register a schema under subject "del-adv-fresh":
      """
      {"type":"record","name":"New","fields":[{"name":"c","type":"string"}]}
      """
    Then the response status should be 200
    When I get the latest version of subject "del-adv-fresh"
    Then the response status should be 200
    And the response field "version" should be 1
    And the audit log should contain an event:
      | event_type          | subject_delete_permanent             |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-fresh                        |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-fresh              |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |
    And the audit log should contain an event:
      | event_type          | schema_register                      |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-fresh                        |
      | schema_id           | *                                    |
      | version             | *                                    |
      | schema_type         | AVRO                                 |
      | method              | POST                                 |
      | path                | /subjects/del-adv-fresh/versions     |
      | status_code         | 200                                  |
      | before_hash         |                                      |
      | after_hash          | sha256:*                             |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  # ==========================================================================
  # DELETION WITH ACTIVE REFERENCES
  # ==========================================================================

  Scenario: Delete version with active reference is blocked
    Given the global compatibility level is "NONE"
    And subject "del-adv-ref-base" has schema:
      """
      {"type":"record","name":"Base","namespace":"com.delref","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "del-adv-ref-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.delref\",\"fields\":[{\"name\":\"base\",\"type\":\"com.delref.Base\"}]}",
        "references": [
          {"name": "com.delref.Base", "subject": "del-adv-ref-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I delete version 1 of subject "del-adv-ref-base"
    Then the response status should be 422
    And the response should have error code 42206
    And the response should contain "reference"

  Scenario: Delete subject with active reference is blocked
    Given the global compatibility level is "NONE"
    And subject "del-adv-refsub-base" has schema:
      """
      {"type":"record","name":"BaseSub","namespace":"com.delrefsub","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "del-adv-refsub-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"ConsumerSub\",\"namespace\":\"com.delrefsub\",\"fields\":[{\"name\":\"base\",\"type\":\"com.delrefsub.BaseSub\"}]}",
        "references": [
          {"name": "com.delrefsub.BaseSub", "subject": "del-adv-refsub-base", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Subject-level soft-delete blocked when any version is referenced
    When I delete subject "del-adv-refsub-base"
    Then the response status should be 422
    And the response should have error code 42206
    And the response should contain "reference"

  # ==========================================================================
  # SOFT DELETE THEN PERMANENT DELETE SAME RESOURCE
  # ==========================================================================

  Scenario: Soft delete then permanent delete version removes it
    Given the global compatibility level is "NONE"
    And subject "del-adv-two-step-ver" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "del-adv-two-step-ver" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "del-adv-two-step-ver"
    Then the response status should be 200
    When I permanently delete version 1 of subject "del-adv-two-step-ver"
    Then the response status should be 200
    When I list versions of subject "del-adv-two-step-ver"
    Then the response status should be 200
    And the response should be an array of length 1
    And the audit log should contain an event:
      | event_type          | schema_delete_permanent                    |
      | outcome             | success                                    |
      | actor_id            |                                            |
      | actor_type          | anonymous                                  |
      | auth_method         |                                            |
      | role                |                                            |
      | target_type         | subject                                    |
      | target_id           | del-adv-two-step-ver                       |
      | schema_id           | *                                          |
      | version             | *                                          |
      | schema_type         | AVRO                                       |
      | method              | DELETE                                     |
      | path                | /subjects/del-adv-two-step-ver/versions    |
      | status_code         | 200                                        |
      | before_hash         | sha256:*                                   |
      | after_hash          |                                            |
      | context             | .                                          |
      | transport_security  | tls                                        |
      | reason              |                                            |
      | error               |                                            |
      | request_body        |                                            |
      | metadata            |                                            |
      | timestamp           | *                                          |
      | duration_ms         | *                                          |
      | request_id          | *                                          |
      | source_ip           | *                                          |
      | user_agent          | *                                          |

  Scenario: Soft delete then permanent delete same subject works
    Given subject "del-adv-two-step-sub" has schema:
      """
      {"type":"record","name":"TwoStep","fields":[{"name":"a","type":"string"}]}
      """
    When I delete subject "del-adv-two-step-sub"
    Then the response status should be 200
    When I permanently delete subject "del-adv-two-step-sub"
    Then the response status should be 200
    When I list subjects with deleted
    Then the response should be an array of length 0
    And the audit log should contain an event:
      | event_type          | subject_delete_permanent             |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-two-step-sub                 |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-two-step-sub       |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  # ==========================================================================
  # DELETE NON-EXISTENT RESOURCES
  # ==========================================================================

  Scenario: Delete non-existent version returns 404
    Given subject "del-adv-404-ver" has schema:
      """
      {"type":"record","name":"Exists","fields":[{"name":"a","type":"string"}]}
      """
    When I delete version 99 of subject "del-adv-404-ver"
    Then the response status should be 404
    And the response should have error code 40402

  Scenario: Delete non-existent subject returns 404
    When I delete subject "del-adv-totally-missing"
    Then the response status should be 404
    And the response should have error code 40401

  # ==========================================================================
  # SCHEMA ID BEHAVIOR AFTER DELETION
  # ==========================================================================

  Scenario: Soft-deleted schema is still retrievable by global schema ID
    Given subject "del-adv-id-lookup" has schema:
      """
      {"type":"record","name":"ByID","fields":[{"name":"val","type":"string"}]}
      """
    When I get version 1 of subject "del-adv-id-lookup"
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I delete subject "del-adv-id-lookup"
    Then the response status should be 200
    # Schema content is still available by its global ID even after subject soft-delete
    When I get schema by ID {{schema_id}}
    Then the response status should be 200
    And the response should contain "ByID"
    And the audit log should contain an event:
      | event_type          | subject_delete_soft                  |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-id-lookup                    |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-id-lookup          |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |

  Scenario: Permanent delete of last subject-version for a schema ID removes it from ID lookup
    Given the global compatibility level is "NONE"
    When I register a schema under subject "del-adv-id-gone":
      """
      {"type":"record","name":"WillVanish","fields":[{"name":"gone","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "gone_id"
    When I delete subject "del-adv-id-gone"
    Then the response status should be 200
    When I permanently delete subject "del-adv-id-gone"
    Then the response status should be 200
    # After permanent deletion of the only subject-version, the schema ID should no longer resolve
    When I get the subjects for schema ID {{gone_id}}
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type          | subject_delete_permanent             |
      | outcome             | success                              |
      | actor_id            |                                      |
      | actor_type          | anonymous                            |
      | auth_method         |                                      |
      | role                |                                      |
      | target_type         | subject                              |
      | target_id           | del-adv-id-gone                      |
      | schema_id           |                                      |
      | version             |                                      |
      | schema_type         |                                      |
      | method              | DELETE                               |
      | path                | /subjects/del-adv-id-gone            |
      | status_code         | 200                                  |
      | before_hash         | sha256:*                             |
      | after_hash          |                                      |
      | context             | .                                    |
      | transport_security  | tls                                  |
      | reason              |                                      |
      | error               |                                      |
      | request_body        |                                      |
      | metadata            |                                      |
      | timestamp           | *                                    |
      | duration_ms         | *                                    |
      | request_id          | *                                    |
      | source_ip           | *                                    |
      | user_agent          | *                                    |
