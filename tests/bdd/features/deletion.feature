@functional
Feature: Schema Deletion
  As a developer, I want to soft-delete and permanently delete schemas

  Scenario: Soft-delete hides version from get
    Given subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I delete version 1 of subject "user-value"
    Then the response status should be 200
    When I get version 1 of subject "user-value"
    Then the response status should be 404
    And the audit log should contain an event:
      | event_type           | schema_delete_soft                       |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | user-value                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | DELETE                                   |
      | path                 | /subjects/user-value/versions/1          |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Soft-delete subject hides it from list
    Given subject "temp-value" has schema:
      """
      {"type":"record","name":"Temp","fields":[{"name":"v","type":"string"}]}
      """
    When I delete subject "temp-value"
    Then the response status should be 200
    When I list all subjects
    Then the response should be an array of length 0
    And the audit log should contain an event:
      | event_type           | subject_delete_soft                      |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | temp-value                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | DELETE                                   |
      | path                 | /subjects/temp-value                     |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Soft-deleted subject visible with deleted=true
    Given subject "soft-del-vis" has schema:
      """
      {"type":"record","name":"SoftDelVis","fields":[{"name":"v","type":"string"}]}
      """
    When I delete subject "soft-del-vis"
    Then the response status should be 200
    When I list subjects with deleted
    Then the response should be an array of length 1
    And the response array should contain "soft-del-vis"
    And the audit log should contain an event:
      | event_type           | subject_delete_soft                      |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | soft-del-vis                             |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | DELETE                                   |
      | path                 | /subjects/soft-del-vis                   |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Permanent delete removes subject completely (two-step)
    Given subject "perm-value" has schema:
      """
      {"type":"record","name":"Perm","fields":[{"name":"v","type":"string"}]}
      """
    When I delete subject "perm-value"
    Then the response status should be 200
    When I permanently delete subject "perm-value"
    Then the response status should be 200
    When I list all subjects
    Then the response should be an array of length 0
    And the audit log should contain an event:
      | event_type           | subject_delete_permanent                 |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | perm-value                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | DELETE                                   |
      | path                 | /subjects/perm-value                     |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Re-register after soft-delete creates new version
    Given the global compatibility level is "NONE"
    And subject "user-value" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I delete subject "user-value"
    And I register a schema under subject "user-value":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | subject_delete_soft                      |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | user-value                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | DELETE                                   |
      | path                 | /subjects/user-value                     |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | user-value                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/user-value/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Delete specific version leaves other versions intact
    Given the global compatibility level is "NONE"
    And subject "multi-ver-del" has schema:
      """
      {"type":"record","name":"V1","fields":[{"name":"a","type":"string"}]}
      """
    And subject "multi-ver-del" has schema:
      """
      {"type":"record","name":"V2","fields":[{"name":"b","type":"string"}]}
      """
    When I delete version 1 of subject "multi-ver-del"
    Then the response status should be 200
    When I get version 2 of subject "multi-ver-del"
    Then the response status should be 200
    When I get the latest version of subject "multi-ver-del"
    Then the response status should be 200
    And the response field "version" should be 2
    And the audit log should contain an event:
      | event_type           | schema_delete_soft                       |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | multi-ver-del                            |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | AVRO                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | DELETE                                   |
      | path                 | /subjects/multi-ver-del/versions/1       |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
