@mtls @auth
Feature: mTLS Transport Security
  As a security-conscious operator
  I want to require client certificates for transport security
  So that only trusted clients can connect to the schema registry

  # --- Section 1: mTLS Transport (no auth layer) ---
  # These scenarios test that mTLS works as transport-level security only.
  # The server requires a valid client certificate but does not perform
  # authentication or authorization — any valid cert gets full access.

  Scenario: Valid client certificate can list subjects
    Given I connect with mTLS certificate "client-admin"
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Valid client certificate can register a schema
    Given I connect with mTLS certificate "client-admin"
    When I POST "/subjects/mtls-test-subject/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsTest\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                      |
      | outcome            | success                              |
      | actor_id           |                                      |
      | actor_type         | anonymous                            |
      | auth_method        |                                      |
      | role               |                                      |
      | target_type        | subject                              |
      | target_id          | mtls-test-subject                    |
      | schema_id          | *                                    |
      | version            | *                                    |
      | schema_type        | AVRO                                 |
      | before_hash        |                                      |
      | after_hash         | sha256:*                             |
      | context            | .                                    |
      | transport_security | mtls                                 |
      | method             | POST                                 |
      | path               | /subjects/mtls-test-subject/versions |
      | status_code        | 200                                  |
      | reason             |                                      |
      | error              |                                      |
      | request_body       |                                      |
      | metadata           |                                      |
      | timestamp          | *                                    |
      | duration_ms        | *                                    |
      | request_id         | *                                    |
      | source_ip          | *                                    |
      | user_agent         | *                                    |

  Scenario: Valid client certificate can get schema by ID
    Given I connect with mTLS certificate "client-admin"
    And I POST "/subjects/mtls-get-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsGet\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200

  Scenario: Valid client certificate can delete a subject
    Given I connect with mTLS certificate "client-admin"
    And I POST "/subjects/mtls-del-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsDel\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/mtls-del-test"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | subject_delete_soft  |
      | outcome            | success              |
      | actor_id           |                      |
      | actor_type         | anonymous            |
      | auth_method        |                      |
      | role               |                      |
      | target_type        | subject              |
      | target_id          | mtls-del-test        |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        | AVRO                 |
      | before_hash        | sha256:*             |
      | after_hash         |                      |
      | context            | .                    |
      | transport_security | mtls                 |
      | method             | DELETE               |
      | path               | /subjects/mtls-del-test |
      | status_code        | 200                  |
      | reason             |                      |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  Scenario: Valid client certificate can manage config
    Given I connect with mTLS certificate "client-admin"
    When I PUT "/config" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | config_update        |
      | outcome            | success              |
      | actor_id           |                      |
      | actor_type         | anonymous            |
      | auth_method        |                      |
      | role               |                      |
      | target_type        | config               |
      | target_id          | _global              |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        | *                    |
      | after_hash         | sha256:*             |
      | context            | .                    |
      | transport_security | mtls                 |
      | method             | PUT                  |
      | path               | /config              |
      | status_code        | 200                  |
      | reason             |                      |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |
    When I GET "/config"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  Scenario: Connection without client certificate is refused
    Given I connect without a client certificate
    When I attempt a GET request to "/subjects"
    Then the connection should be refused

  Scenario: Expired client certificate is refused
    Given I connect with mTLS certificate "client-expired"
    When I attempt a GET request to "/subjects"
    Then the connection should be refused

  Scenario: Client certificate from wrong CA is refused
    Given I connect with mTLS certificate "client-wrong-ca"
    When I attempt a GET request to "/subjects"
    Then the connection should be refused

  # --- Section 2: mTLS + Basic Auth — Authentication ---
  # These scenarios test mTLS as transport + Basic auth for identity/RBAC.

  @mtls-auth
  Scenario: Valid cert + valid admin credentials succeeds
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I GET "/subjects"
    Then the response status should be 200

  @mtls-auth
  Scenario: Valid cert + no auth credentials returns 401
    Given I connect with mTLS certificate "client-admin"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type         | auth_failure         |
      | outcome            | failure              |
      | actor_id           |                      |
      | actor_type         | anonymous            |
      | auth_method        |                      |
      | role               |                      |
      | target_type        |                      |
      | target_id          |                      |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        |                      |
      | after_hash         |                      |
      | context            | .                     |
      | transport_security | mtls                 |
      | method             | GET                  |
      | path               | /subjects            |
      | status_code        | 401                  |
      | reason             | no_valid_credentials |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  @mtls-auth
  Scenario: Valid cert + wrong password returns 401
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "wrong-password"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type         | auth_failure         |
      | outcome            | failure              |
      | actor_id           |                      |
      | actor_type         | anonymous            |
      | auth_method        |                      |
      | role               |                      |
      | target_type        |                      |
      | target_id          |                      |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        |                      |
      | after_hash         |                      |
      | context            | .                     |
      | transport_security | mtls                 |
      | method             | GET                  |
      | path               | /subjects            |
      | status_code        | 401                  |
      | reason             | no_valid_credentials |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  @mtls-auth
  Scenario: No client certificate is refused even with valid auth
    Given I connect without a client certificate
    And I authenticate as "admin" with password "admin"
    When I attempt a GET request to "/subjects"
    Then the connection should be refused

  @mtls-auth
  Scenario: Expired client certificate is refused even with valid auth
    Given I connect with mTLS certificate "client-expired"
    And I authenticate as "admin" with password "admin"
    When I attempt a GET request to "/subjects"
    Then the connection should be refused

  # --- Section 3: mTLS + Basic Auth — RBAC Schema operations ---

  @mtls-auth
  Scenario: Admin can register a schema over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I POST "/subjects/mtls-rbac-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsRbac\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                   |
      | outcome            | success                           |
      | actor_id           | admin                             |
      | actor_type         | user                              |
      | auth_method        | basic                             |
      | role               | super_admin                       |
      | target_type        | subject                           |
      | target_id          | mtls-rbac-test                    |
      | schema_id          | *                                 |
      | version            | *                                 |
      | schema_type        | AVRO                              |
      | before_hash        |                                   |
      | after_hash         | sha256:*                          |
      | context            | .                                 |
      | transport_security | mtls                              |
      | method             | POST                              |
      | path               | /subjects/mtls-rbac-test/versions |
      | status_code        | 200                               |
      | reason             |                                   |
      | error              |                                   |
      | request_body       |                                   |
      | metadata           |                                   |
      | timestamp          | *                                 |
      | duration_ms        | *                                 |
      | request_id         | *                                 |
      | source_ip          | *                                 |
      | user_agent         | *                                 |

  @mtls-auth
  Scenario: Admin can read a schema over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    And I POST "/subjects/mtls-read-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsRead\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/mtls-read-test/versions/1"
    Then the response status should be 200

  @mtls-auth
  Scenario: Admin can delete a schema version over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    And I POST "/subjects/mtls-delvs-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsDelVs\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/mtls-delvs-test/versions/1"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_delete_soft                 |
      | outcome            | success                            |
      | actor_id           | admin                              |
      | actor_type         | user                               |
      | auth_method        | basic                              |
      | role               | super_admin                        |
      | target_type        | subject                            |
      | target_id          | mtls-delvs-test                    |
      | schema_id          | *                                  |
      | version            | *                                  |
      | schema_type        | AVRO                               |
      | before_hash        | sha256:*                           |
      | after_hash         |                                    |
      | context            | .                                  |
      | transport_security | mtls                               |
      | method             | DELETE                             |
      | path               | /subjects/mtls-delvs-test/versions/1 |
      | status_code        | 200                                |
      | reason             |                                    |
      | error              |                                    |
      | request_body       |                                    |
      | metadata           |                                    |
      | timestamp          | *                                  |
      | duration_ms        | *                                  |
      | request_id         | *                                  |
      | source_ip          | *                                  |
      | user_agent         | *                                  |

  @mtls-auth
  Scenario: Admin can delete a subject over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    And I POST "/subjects/mtls-delsub-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsDelSub\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I DELETE "/subjects/mtls-delsub-test"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | subject_delete_soft     |
      | outcome            | success                 |
      | actor_id           | admin                   |
      | actor_type         | user                    |
      | auth_method        | basic                   |
      | role               | super_admin             |
      | target_type        | subject                 |
      | target_id          | mtls-delsub-test        |
      | schema_id          |                         |
      | version            |                         |
      | schema_type        | AVRO                    |
      | before_hash        | sha256:*                |
      | after_hash         |                         |
      | context            | .                       |
      | transport_security | mtls                    |
      | method             | DELETE                  |
      | path               | /subjects/mtls-delsub-test |
      | status_code        | 200                     |
      | reason             |                         |
      | error              |                         |
      | request_body       |                         |
      | metadata           |                         |
      | timestamp          | *                       |
      | duration_ms        | *                       |
      | request_id         | *                       |
      | source_ip          | *                       |
      | user_agent         | *                       |

  @mtls-auth
  Scenario: Readonly user can read schemas but not write over mTLS
    Given I connect with mTLS certificate "client-readonly"
    And I authenticate as "admin" with password "admin"
    And I POST "/subjects/mtls-ro-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsRo\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # Create a readonly user via admin
    When I POST "/admin/users" with body:
      """
      {"username": "reader", "password": "reader-pass", "role": "readonly"}
      """
    Then the response status should be 201
    And the audit log should contain an event:
      | event_type         | user_create          |
      | outcome            | success              |
      | actor_id           | admin                |
      | actor_type         | user                 |
      | auth_method        | basic                |
      | role               | super_admin          |
      | target_type        | user                 |
      | target_id          | reader               |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        |                      |
      | after_hash         | sha256:*             |
      | context            | .                     |
      | transport_security | mtls                 |
      | method             | POST                 |
      | path               | /admin/users         |
      | status_code        | 201                  |
      | reason             |                      |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |
    # Switch to readonly user
    Given I authenticate as "reader" with password "reader-pass"
    When I GET "/subjects/mtls-ro-test/versions/1"
    Then the response status should be 200
    When I POST "/subjects/mtls-ro-write/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsRoWrite\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type         | auth_forbidden                   |
      | outcome            | failure                          |
      | actor_id           | reader                           |
      | actor_type         | user                             |
      | auth_method        | basic                            |
      | role               | readonly                         |
      | target_type        | subject                          |
      | target_id          | mtls-ro-write                    |
      | schema_id          |                                  |
      | version            |                                  |
      | schema_type        |                                  |
      | before_hash        |                                  |
      | after_hash         |                                  |
      | context            | .                                 |
      | transport_security | mtls                             |
      | method             | POST                             |
      | path               | /subjects/mtls-ro-write/versions |
      | status_code        | 403                              |
      | reason             | permission_denied    |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  @mtls-auth
  Scenario: Readonly user cannot delete schemas over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    And I POST "/subjects/mtls-ro-del/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsRoDel\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/admin/users" with body:
      """
      {"username": "reader2", "password": "reader2-pass", "role": "readonly"}
      """
    Then the response status should be 201
    Given I authenticate as "reader2" with password "reader2-pass"
    When I DELETE "/subjects/mtls-ro-del"
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type         | auth_forbidden       |
      | outcome            | failure              |
      | actor_id           | reader2              |
      | actor_type         | user                 |
      | auth_method        | basic                |
      | role               | readonly             |
      | target_type        | subject              |
      | target_id          | mtls-ro-del          |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        |                      |
      | after_hash         |                      |
      | context            | .                     |
      | transport_security | mtls                 |
      | method             | DELETE               |
      | path               | /subjects/mtls-ro-del |
      | status_code        | 403                  |
      | reason             | permission_denied    |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  @mtls-auth
  Scenario: Unauthenticated user cannot read schemas over mTLS
    Given I connect with mTLS certificate "client-admin"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type         | auth_failure         |
      | outcome            | failure              |
      | actor_id           |                      |
      | actor_type         | anonymous            |
      | auth_method        |                      |
      | role               |                      |
      | target_type        |                      |
      | target_id          |                      |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        |                      |
      | after_hash         |                      |
      | context            | .                     |
      | transport_security | mtls                 |
      | method             | GET                  |
      | path               | /subjects            |
      | status_code        | 401                  |
      | reason             | no_valid_credentials |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  # --- Section 4: mTLS + Basic Auth — RBAC Config operations ---

  @mtls-auth
  Scenario: Admin can update global config over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I PUT "/config" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | config_update        |
      | outcome            | success              |
      | actor_id           | admin                |
      | actor_type         | user                 |
      | auth_method        | basic                |
      | role               | super_admin          |
      | target_type        | config               |
      | target_id          | _global              |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        | *                    |
      | after_hash         | sha256:*             |
      | context            | .                    |
      | transport_security | mtls                 |
      | method             | PUT                  |
      | path               | /config              |
      | status_code        | 200                  |
      | reason             |                      |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  @mtls-auth
  Scenario: Admin can read global config over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I GET "/config"
    Then the response status should be 200

  @mtls-auth
  Scenario: Readonly user cannot update config over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I POST "/admin/users" with body:
      """
      {"username": "configro", "password": "configro-pass", "role": "readonly"}
      """
    Then the response status should be 201
    Given I authenticate as "configro" with password "configro-pass"
    When I PUT "/config" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type         | auth_forbidden       |
      | outcome            | failure              |
      | actor_id           | configro             |
      | actor_type         | user                 |
      | auth_method        | basic                |
      | role               | readonly             |
      | target_type        | config               |
      | target_id          | _global              |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        |                      |
      | after_hash         |                      |
      | context            | .                     |
      | transport_security | mtls                 |
      | method             | PUT                  |
      | path               | /config              |
      | status_code        | 403                  |
      | reason             | permission_denied    |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  # --- Section 5: mTLS + Basic Auth — RBAC Mode operations ---

  @mtls-auth
  Scenario: Admin can update mode over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I PUT "/mode" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | mode_update          |
      | outcome            | success              |
      | actor_id           | admin                |
      | actor_type         | user                 |
      | auth_method        | basic                |
      | role               | super_admin          |
      | target_type        | mode                 |
      | target_id          | _global              |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        | *                    |
      | after_hash         | sha256:*             |
      | context            | .                    |
      | transport_security | mtls                 |
      | method             | PUT                  |
      | path               | /mode                |
      | status_code        | 200                  |
      | reason             |                      |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |
    # Reset back
    When I PUT "/mode" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200

  @mtls-auth
  Scenario: Readonly user cannot update mode over mTLS
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I POST "/admin/users" with body:
      """
      {"username": "modero", "password": "modero-pass", "role": "readonly"}
      """
    Then the response status should be 201
    Given I authenticate as "modero" with password "modero-pass"
    When I PUT "/mode" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type         | auth_forbidden       |
      | outcome            | failure              |
      | actor_id           | modero               |
      | actor_type         | user                 |
      | auth_method        | basic                |
      | role               | readonly             |
      | target_type        | mode                 |
      | target_id          | _global              |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        |                      |
      | after_hash         |                      |
      | context            | .                     |
      | transport_security | mtls                 |
      | method             | PUT                  |
      | path               | /mode                |
      | status_code        | 403                  |
      | reason             | permission_denied    |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  # --- Section 6: Comprehensive audit assertions for transport_security ---

  @mtls-auth
  Scenario: Schema register audit event has full transport_security context
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I POST "/subjects/mtls-audit-test/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"MtlsAudit\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                    |
      | outcome            | success                            |
      | actor_id           | admin                              |
      | actor_type         | user                               |
      | auth_method        | basic                              |
      | role               | super_admin                        |
      | target_type        | subject                            |
      | target_id          | mtls-audit-test                    |
      | schema_id          | *                                  |
      | version            | *                                  |
      | schema_type        | AVRO                               |
      | before_hash        |                                    |
      | after_hash         | sha256:*                           |
      | context            | .                                  |
      | transport_security | mtls                               |
      | method             | POST                               |
      | path               | /subjects/mtls-audit-test/versions |
      | status_code        | 200                                |
      | reason             |                                    |
      | error              |                                    |
      | request_body       |                                    |
      | metadata           |                                    |
      | timestamp          | *                                  |
      | duration_ms        | *                                  |
      | request_id         | *                                  |
      | source_ip          | *                                  |
      | user_agent         | *                                  |

  @mtls-auth
  Scenario: Auth failure audit event has full transport_security context
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "wrong"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type         | auth_failure         |
      | outcome            | failure              |
      | actor_id           |                      |
      | actor_type         | anonymous            |
      | auth_method        |                      |
      | role               |                      |
      | target_type        |                      |
      | target_id          |                      |
      | schema_id          |                      |
      | version            |                      |
      | schema_type        |                      |
      | before_hash        |                      |
      | after_hash         |                      |
      | context            | .                     |
      | transport_security | mtls                 |
      | method             | GET                  |
      | path               | /subjects            |
      | status_code        | 401                  |
      | reason             | no_valid_credentials |
      | error              |                      |
      | request_body       |                      |
      | metadata           |                      |
      | timestamp          | *                    |
      | duration_ms        | *                    |
      | request_id         | *                    |
      | source_ip          | *                    |
      | user_agent         | *                    |

  @mtls-auth
  Scenario: RBAC forbidden audit event has full transport_security context
    Given I connect with mTLS certificate "client-admin"
    And I authenticate as "admin" with password "admin"
    When I POST "/admin/users" with body:
      """
      {"username": "auditro", "password": "auditro-pass", "role": "readonly"}
      """
    Then the response status should be 201
    Given I authenticate as "auditro" with password "auditro-pass"
    When I POST "/subjects/mtls-forbidden/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Forbidden\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type         | auth_forbidden                    |
      | outcome            | failure                           |
      | actor_id           | auditro                           |
      | actor_type         | user                              |
      | auth_method        | basic                             |
      | role               | readonly                          |
      | target_type        | subject                           |
      | target_id          | mtls-forbidden                    |
      | schema_id          |                                   |
      | version            |                                   |
      | schema_type        |                                   |
      | before_hash        |                                   |
      | after_hash         |                                   |
      | context            | .                                  |
      | transport_security | mtls                              |
      | method             | POST                              |
      | path               | /subjects/mtls-forbidden/versions |
      | status_code        | 403                               |
      | reason             | permission_denied                 |
      | error              |                                   |
      | request_body       |                                   |
      | metadata           |                                   |
      | timestamp          | *                                 |
      | duration_ms        | *                                 |
      | request_id         | *                                 |
      | source_ip          | *                                 |
      | user_agent         | *                                 |
