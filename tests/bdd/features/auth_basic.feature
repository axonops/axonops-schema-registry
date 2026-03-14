@auth @basic-auth
Feature: Internal User Authentication and RBAC
  As an operator, I want the schema registry to authenticate users stored in
  its internal database (via bootstrap or admin API) and enforce role-based
  access control so that I can manage access without an external identity
  provider.

  The test server has auth enabled with basic + api_key methods, RBAC enabled,
  and a bootstrapped super_admin user: admin / admin-password (created on
  startup via security.auth.bootstrap config).

  Roles under test:
    - super_admin (bootstrap: admin / admin-password) -> full access + admin:write
    - admin (created via admin API) -> full access except admin:write
    - developer (created via admin API) -> register + read schemas
    - readonly (created via admin API) -> read-only access

  # ===================================================================
  # Section 1: Authentication (basic connectivity)
  # ===================================================================

  Scenario: Bootstrap super_admin authenticates successfully
    Given I authenticate as "admin" with password "admin-password"
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Created admin user authenticates successfully
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-admin1" password "admin1-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "ba-admin1" with password "admin1-pass"
    And I GET "/subjects"
    Then the response status should be 200

  Scenario: Created developer user authenticates successfully
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-dev1" password "dev1-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "ba-dev1" with password "dev1-pass"
    And I GET "/subjects"
    Then the response status should be 200

  Scenario: Created readonly user authenticates successfully
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-ro1" password "ro1-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "ba-ro1" with password "ro1-pass"
    And I GET "/subjects"
    Then the response status should be 200

  Scenario: Invalid password returns 401
    Given I authenticate as "admin" with password "wrong-password"
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Non-existent user returns 401
    Given I authenticate as "nonexistent" with password "any-password"
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: No credentials returns 401
    Given I clear authentication
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Empty password returns 401
    Given I authenticate as "admin" with password ""
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Disabled user returns 401
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-disabled" password "disabled-pass" role "developer"
    Then the response status should be 201
    And I store the response field "id" as "disabled_user_id"
    # Verify user can auth first
    When I authenticate as "ba-disabled" with password "disabled-pass"
    And I GET "/subjects"
    Then the response status should be 200
    # Now disable the user
    When I authenticate as "admin" with password "admin-password"
    And I update user "{{disabled_user_id}}" with:
      """
      {"enabled": false}
      """
    Then the response status should be 200
    # Disabled user should get 401
    When I authenticate as "ba-disabled" with password "disabled-pass"
    And I GET "/subjects"
    Then the response status should be 401

  # ===================================================================
  # Section 2: RBAC — Schema operations
  # ===================================================================

  Scenario: Admin can register a schema
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-s-admin" password "pass" role "admin"
    When I authenticate as "ba-s-admin" with password "pass"
    And I register a "AVRO" schema under subject "ba-admin-subject":
      """
      {"type":"record","name":"BaAdmin","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  Scenario: Admin can read a schema
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-read-subject":
      """
      {"type":"record","name":"BaRead","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/subjects/ba-read-subject/versions/1"
    Then the response status should be 200

  Scenario: Admin can delete a schema version
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-del-ver":
      """
      {"type":"record","name":"BaDelVer","fields":[{"name":"id","type":"int"}]}
      """
    When I DELETE "/subjects/ba-del-ver/versions/1"
    Then the response status should be 200

  Scenario: Admin can delete a subject
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-del-subj":
      """
      {"type":"record","name":"BaDelSubj","fields":[{"name":"id","type":"int"}]}
      """
    When I DELETE "/subjects/ba-del-subj"
    Then the response status should be 200

  Scenario: Developer can register a schema
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-s-dev" password "pass" role "developer"
    When I authenticate as "ba-s-dev" with password "pass"
    And I register a "AVRO" schema under subject "ba-dev-subject":
      """
      {"type":"record","name":"BaDev","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  Scenario: Developer can read a schema
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-dev-read":
      """
      {"type":"record","name":"BaDevRead","fields":[{"name":"id","type":"int"}]}
      """
    And I create a user with username "ba-s-devr" password "pass" role "developer"
    When I authenticate as "ba-s-devr" with password "pass"
    And I GET "/subjects/ba-dev-read/versions/1"
    Then the response status should be 200

  Scenario: Developer cannot delete a schema version
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-dev-nodel":
      """
      {"type":"record","name":"BaDevNoDel","fields":[{"name":"id","type":"int"}]}
      """
    And I create a user with username "ba-s-devd" password "pass" role "developer"
    When I authenticate as "ba-s-devd" with password "pass"
    And I DELETE "/subjects/ba-dev-nodel/versions/1"
    Then the response status should be 403

  Scenario: Developer cannot delete a subject
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-dev-nodelsubj":
      """
      {"type":"record","name":"BaDevNoDelSubj","fields":[{"name":"id","type":"int"}]}
      """
    And I create a user with username "ba-s-devs" password "pass" role "developer"
    When I authenticate as "ba-s-devs" with password "pass"
    And I DELETE "/subjects/ba-dev-nodelsubj"
    Then the response status should be 403

  Scenario: Readonly can read a schema
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-ro-read":
      """
      {"type":"record","name":"BaRORead","fields":[{"name":"id","type":"int"}]}
      """
    And I create a user with username "ba-s-ror" password "pass" role "readonly"
    When I authenticate as "ba-s-ror" with password "pass"
    And I GET "/subjects/ba-ro-read/versions/1"
    Then the response status should be 200

  Scenario: Readonly cannot register a schema
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-s-rowr" password "pass" role "readonly"
    When I authenticate as "ba-s-rowr" with password "pass"
    And I register a "AVRO" schema under subject "ba-ro-write":
      """
      {"type":"record","name":"BaROWrite","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403

  Scenario: Readonly cannot delete a schema version
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-ro-nodel":
      """
      {"type":"record","name":"BaRONoDel","fields":[{"name":"id","type":"int"}]}
      """
    And I create a user with username "ba-s-rod" password "pass" role "readonly"
    When I authenticate as "ba-s-rod" with password "pass"
    And I DELETE "/subjects/ba-ro-nodel/versions/1"
    Then the response status should be 403

  # ===================================================================
  # Section 3: RBAC — Config operations
  # ===================================================================

  Scenario: Admin can read global config
    Given I authenticate as "admin" with password "admin-password"
    When I GET "/config"
    Then the response status should be 200

  Scenario: Admin can update global config
    Given I authenticate as "admin" with password "admin-password"
    When I PUT "/config" with body:
      """
      {"compatibility":"FULL"}
      """
    Then the response status should be 200

  Scenario: Admin can delete global config
    Given I authenticate as "admin" with password "admin-password"
    When I DELETE "/config"
    Then the response status should be 200

  Scenario: Admin can read and write per-subject config
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-cfg-subj":
      """
      {"type":"record","name":"BaCfgSubj","fields":[{"name":"id","type":"int"}]}
      """
    When I PUT "/config/ba-cfg-subj" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 200
    When I GET "/config/ba-cfg-subj"
    Then the response status should be 200

  Scenario: Developer can read config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-c-dev" password "pass" role "developer"
    When I authenticate as "ba-c-dev" with password "pass"
    And I GET "/config"
    Then the response status should be 200

  Scenario: Developer cannot update config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-c-devw" password "pass" role "developer"
    When I authenticate as "ba-c-devw" with password "pass"
    And I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  Scenario: Readonly can read config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-c-ro" password "pass" role "readonly"
    When I authenticate as "ba-c-ro" with password "pass"
    And I GET "/config"
    Then the response status should be 200

  Scenario: Readonly cannot update config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-c-row" password "pass" role "readonly"
    When I authenticate as "ba-c-row" with password "pass"
    And I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 4: RBAC — Mode operations
  # ===================================================================

  Scenario: Admin can read mode
    Given I authenticate as "admin" with password "admin-password"
    When I GET "/mode"
    Then the response status should be 200

  Scenario: Admin can update mode
    Given I authenticate as "admin" with password "admin-password"
    When I PUT "/mode" with body:
      """
      {"mode":"READWRITE"}
      """
    Then the response status should be 200

  Scenario: Admin can delete mode
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-mode-subj":
      """
      {"type":"record","name":"BaModeSubj","fields":[{"name":"id","type":"int"}]}
      """
    And I PUT "/mode/ba-mode-subj" with body:
      """
      {"mode":"READONLY"}
      """
    When I DELETE "/mode/ba-mode-subj"
    Then the response status should be 200

  Scenario: Developer can read mode
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-m-dev" password "pass" role "developer"
    When I authenticate as "ba-m-dev" with password "pass"
    And I GET "/mode"
    Then the response status should be 200

  Scenario: Developer cannot update mode
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-m-devw" password "pass" role "developer"
    When I authenticate as "ba-m-devw" with password "pass"
    And I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 403

  Scenario: Readonly can read mode
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-m-ro" password "pass" role "readonly"
    When I authenticate as "ba-m-ro" with password "pass"
    And I GET "/mode"
    Then the response status should be 200

  Scenario: Readonly cannot update mode
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-m-row" password "pass" role "readonly"
    When I authenticate as "ba-m-row" with password "pass"
    And I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 5: RBAC — Compatibility check
  # ===================================================================

  Scenario: Admin can check compatibility
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-compat":
      """
      {"type":"record","name":"BaCompat","fields":[{"name":"id","type":"int"}]}
      """
    When I check compatibility of schema against subject "ba-compat":
      """
      {"type":"record","name":"BaCompat","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200
    And the response body should contain "is_compatible"

  Scenario: Developer can check compatibility
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-compat-dev":
      """
      {"type":"record","name":"BaCompatDev","fields":[{"name":"id","type":"int"}]}
      """
    And I create a user with username "ba-cp-dev" password "pass" role "developer"
    When I authenticate as "ba-cp-dev" with password "pass"
    And I check compatibility of schema against subject "ba-compat-dev":
      """
      {"type":"record","name":"BaCompatDev","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: Readonly can check compatibility
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-compat-ro":
      """
      {"type":"record","name":"BaCompatRO","fields":[{"name":"id","type":"int"}]}
      """
    And I create a user with username "ba-cp-ro" password "pass" role "readonly"
    When I authenticate as "ba-cp-ro" with password "pass"
    And I check compatibility of schema against subject "ba-compat-ro":
      """
      {"type":"record","name":"BaCompatRO","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  # ===================================================================
  # Section 6: RBAC — Encryption (KEK/DEK)
  # ===================================================================

  Scenario: Admin can create and read a KEK
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-k-admin" password "pass" role "admin"
    When I authenticate as "ba-k-admin" with password "pass"
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"ba-test-kek","kmsType":"test-kms","kmsKeyId":"test-key-id","shared":false}
      """
    Then the response status should be 200
    When I GET "/dek-registry/v1/keks/ba-test-kek"
    Then the response status should be 200

  Scenario: Developer cannot create a KEK
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-k-dev" password "pass" role "developer"
    When I authenticate as "ba-k-dev" with password "pass"
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"ba-dev-kek","kmsType":"test-kms","kmsKeyId":"test-key-id","shared":false}
      """
    Then the response status should be 403

  Scenario: Readonly cannot create a KEK
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-k-ro" password "pass" role "readonly"
    When I authenticate as "ba-k-ro" with password "pass"
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"ba-ro-kek","kmsType":"test-kms","kmsKeyId":"test-key-id","shared":false}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 7: RBAC — Exporter operations
  # ===================================================================

  Scenario: Admin can create and read an exporter
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-e-admin" password "pass" role "admin"
    When I authenticate as "ba-e-admin" with password "pass"
    And I POST "/exporters" with body:
      """
      {"name":"ba-exporter","subjects":["*"],"contextType":"CUSTOM","context":"ba-ctx","config":{"schema.registry.url":"http://localhost:8081"}}
      """
    Then the response status should be 200
    When I GET "/exporters/ba-exporter"
    Then the response status should be 200

  Scenario: Developer cannot create an exporter
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-e-dev" password "pass" role "developer"
    When I authenticate as "ba-e-dev" with password "pass"
    And I POST "/exporters" with body:
      """
      {"name":"ba-dev-exporter","subjects":["*"],"contextType":"CUSTOM","context":"ba-ctx","config":{"schema.registry.url":"http://localhost:8081"}}
      """
    Then the response status should be 403

  Scenario: Readonly cannot create an exporter
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-e-ro" password "pass" role "readonly"
    When I authenticate as "ba-e-ro" with password "pass"
    And I POST "/exporters" with body:
      """
      {"name":"ba-ro-exporter","subjects":["*"],"contextType":"CUSTOM","context":"ba-ctx","config":{"schema.registry.url":"http://localhost:8081"}}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 8: RBAC — Admin endpoints
  # ===================================================================

  Scenario: Super admin can read and write admin endpoints
    Given I authenticate as "admin" with password "admin-password"
    When I GET "/admin/users"
    Then the response status should be 200
    When I create a user with username "ba-sa-test" password "pass" role "readonly"
    Then the response status should be 201

  Scenario: Admin role can read admin endpoints but cannot write
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-a-test" password "pass" role "admin"
    When I authenticate as "ba-a-test" with password "pass"
    And I GET "/admin/users"
    Then the response status should be 200
    When I create a user with username "ba-a-should-fail" password "nope" role "readonly"
    Then the response status should be 403

  Scenario: Developer cannot access admin endpoints
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-a-dev" password "pass" role "developer"
    When I authenticate as "ba-a-dev" with password "pass"
    And I GET "/admin/users"
    Then the response status should be 403

  Scenario: Readonly cannot access admin endpoints
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-a-ro" password "pass" role "readonly"
    When I authenticate as "ba-a-ro" with password "pass"
    And I GET "/admin/users"
    Then the response status should be 403

  # ===================================================================
  # Section 9: Audit assertions
  # ===================================================================

  Scenario: Successful basic auth login produces audit event with auth_method=basic
    Given I authenticate as "admin" with password "admin-password"
    When I register a "AVRO" schema under subject "ba-audit-reg":
      """
      {"type":"record","name":"BaAudit","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register |
      | outcome            | success         |
      | actor_id           | admin           |
      | actor_type         | user            |
      | auth_method        | basic           |
      | role               | super_admin     |
      | target_type        | subject         |
      | target_id          | ba-audit-reg    |
      | transport_security | tls             |
      | method             | POST            |
      | status_code        | 200             |
      | schema_type        | AVRO            |
      | after_hash         | sha256:*        |

  Scenario: Failed basic auth login produces auth_failure event
    Given I authenticate as "admin" with password "wrongpassword"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type         | auth_failure         |
      | outcome            | failure              |
      | actor_type         | anonymous            |
      | reason             | no_valid_credentials |
      | transport_security | tls                  |
      | method             | GET                  |
      | path               | /subjects            |
      | status_code        | 401                  |

  Scenario: Developer schema register produces audit event with correct actor and role
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-aud-dev" password "pass" role "developer"
    When I authenticate as "ba-aud-dev" with password "pass"
    And I register a "AVRO" schema under subject "ba-audit-dev":
      """
      {"type":"record","name":"BaAuditDev","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register |
      | outcome            | success         |
      | actor_id           | ba-aud-dev      |
      | actor_type         | user            |
      | auth_method        | basic           |
      | role               | developer       |
      | target_type        | subject         |
      | target_id          | ba-audit-dev    |
      | transport_security | tls             |
      | method             | POST            |
      | status_code        | 200             |

  Scenario: Forbidden action produces auth_forbidden event
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-aud-ro" password "pass" role "readonly"
    When I authenticate as "ba-aud-ro" with password "pass"
    And I register a "AVRO" schema under subject "ba-audit-forbidden":
      """
      {"type":"record","name":"BaAuditForbidden","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type         | auth_forbidden    |
      | outcome            | failure           |
      | actor_id           | ba-aud-ro         |
      | actor_type         | user              |
      | auth_method        | basic             |
      | role               | readonly          |
      | reason             | permission_denied |
      | transport_security | tls               |
      | method             | POST              |
      | status_code        | 403               |

  Scenario: No credentials produces auth_failure audit event
    Given I clear authentication
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type         | auth_failure         |
      | outcome            | failure              |
      | actor_type         | anonymous            |
      | reason             | no_valid_credentials |
      | transport_security | tls                  |
      | method             | GET                  |
      | path               | /subjects            |
      | status_code        | 401                  |

  Scenario: Admin delete produces audit event with correct actor
    Given I authenticate as "admin" with password "admin-password"
    And I register a "AVRO" schema under subject "ba-audit-del":
      """
      {"type":"record","name":"BaAuditDel","fields":[{"name":"id","type":"int"}]}
      """
    When I DELETE "/subjects/ba-audit-del/versions/1"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_delete_soft |
      | outcome            | success       |
      | actor_id           | admin         |
      | actor_type         | user          |
      | auth_method        | basic         |
      | role               | super_admin   |
      | transport_security | tls           |
      | method             | DELETE        |
      | status_code        | 200           |

  Scenario: Config update produces audit event with correct actor
    Given I authenticate as "admin" with password "admin-password"
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | config_update |
      | outcome            | success       |
      | actor_id           | admin         |
      | actor_type         | user          |
      | auth_method        | basic         |
      | role               | super_admin   |
      | transport_security | tls           |
      | method             | PUT           |
      | path               | /config       |
      | status_code        | 200           |

  # ===================================================================
  # Section 10: User lifecycle
  # ===================================================================

  Scenario: Password change via self-service works and new password authenticates
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-pwchange" password "old-pass" role "developer"
    Then the response status should be 201
    # Change password via self-service
    When I authenticate as "ba-pwchange" with password "old-pass"
    And I POST "/me/password" with body:
      """
      {"old_password":"old-pass","new_password":"new-pass"}
      """
    Then the response status should be 204
    # Old password should now fail
    When I authenticate as "ba-pwchange" with password "old-pass"
    And I GET "/subjects"
    Then the response status should be 401
    # New password should work
    When I authenticate as "ba-pwchange" with password "new-pass"
    And I GET "/subjects"
    Then the response status should be 200

  Scenario: Self-service /me returns current user info
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-me-test" password "me-pass" role "developer"
    When I authenticate as "ba-me-test" with password "me-pass"
    And I GET "/me"
    Then the response status should be 200
    And the response field "username" should be "ba-me-test"
    And the response field "role" should be "developer"

  Scenario: API key inherits role and enforces RBAC
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "ba-apikey-user" password "pass" role "readonly"
    Then the response status should be 201
    # Create API key with developer role for the user
    And I create an API key with name "ba-dev-key" role "developer" expires_in 3600
    Then the response status should be 201
    And I store the response field "key" as "ba_dev_api_key"
    # API key should allow schema registration (developer role)
    When I authenticate with stored API key "ba_dev_api_key"
    And I register a "AVRO" schema under subject "ba-apikey-subject":
      """
      {"type":"record","name":"BaApiKey","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  Scenario: API key with readonly role cannot register schema
    Given I authenticate as "admin" with password "admin-password"
    And I create an API key with name "ba-ro-key" role "readonly" expires_in 3600
    Then the response status should be 201
    And I store the response field "key" as "ba_ro_api_key"
    When I authenticate with stored API key "ba_ro_api_key"
    And I register a "AVRO" schema under subject "ba-apikey-ro-subject":
      """
      {"type":"record","name":"BaApiKeyRO","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type         | auth_forbidden    |
      | outcome            | failure           |
      | actor_type         | api_key           |
      | role               | readonly          |
      | reason             | permission_denied |
      | transport_security | tls               |
      | method             | POST              |
      | status_code        | 403               |

  Scenario: Super admin can create users but admin role cannot
    Given I authenticate as "admin" with password "admin-password"
    # Super admin creates an admin user
    And I create a user with username "ba-admin-test" password "pass" role "admin"
    Then the response status should be 201
    # The admin user tries to create another user (admin:write required)
    When I authenticate as "ba-admin-test" with password "pass"
    And I create a user with username "ba-should-fail" password "fail" role "readonly"
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type         | auth_forbidden    |
      | outcome            | failure           |
      | actor_id           | ba-admin-test     |
      | actor_type         | user              |
      | auth_method        | basic             |
      | role               | admin             |
      | reason             | permission_denied |
      | transport_security | tls               |
      | method             | POST              |
      | path               | /admin/users      |
      | status_code        | 403               |
