@ldap @auth
Feature: LDAP Authentication and RBAC
  As an operator, I want the schema registry to authenticate users against an
  LDAP directory over TLS (LDAPS) and enforce role-based access control based
  on LDAP group memberships so that enterprise directory services can manage
  access securely.

  The test server uses OpenLDAP with TLS (LDAPS on port 636) and client
  certificate authentication (mTLS). The registry connects via ldaps:// with
  CA certificate validation and presents a client certificate.

  Four LDAP users are provisioned:
    - admin (uid=admin) -> SchemaRegistryAdmins group -> admin role
    - developer (uid=developer) -> Developers group -> developer role
    - readonly (uid=readonly) -> ReadonlyUsers group -> readonly role
    - nogroup (uid=nogroup) -> no group membership -> default readonly role

  # ---------------------------------------------------------------------------
  # Section 1: Authentication (basic connectivity)
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: LDAP admin user authenticates successfully
    Given I authenticate as "admin" with password "adminpass"
    When I GET "/subjects"
    Then the response status should be 200

  @ldap
  Scenario: LDAP developer user authenticates successfully
    Given I authenticate as "developer" with password "devpass"
    When I GET "/subjects"
    Then the response status should be 200

  @ldap
  Scenario: LDAP readonly user authenticates successfully
    Given I authenticate as "readonly" with password "readonlypass"
    When I GET "/subjects"
    Then the response status should be 200

  @ldap
  Scenario: LDAP nogroup user authenticates successfully with default role
    Given I authenticate as "nogroup" with password "nogrouppass"
    When I GET "/subjects"
    Then the response status should be 200

  @ldap
  Scenario: Invalid LDAP password returns 401
    Given I authenticate as "admin" with password "wrongpassword"
    When I GET "/subjects"
    Then the response status should be 401

  @ldap
  Scenario: Non-existent LDAP user returns 401
    Given I authenticate as "nonexistent" with password "anypassword"
    When I GET "/subjects"
    Then the response status should be 401

  @ldap
  Scenario: No credentials returns 401
    Given I clear authentication
    When I GET "/subjects"
    Then the response status should be 401

  @ldap
  Scenario: Empty password returns 401
    Given I authenticate as "admin" with password ""
    When I GET "/subjects"
    Then the response status should be 401

  # ---------------------------------------------------------------------------
  # Section 2: RBAC — Schema operations
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: Admin can register a schema
    Given I authenticate as "admin" with password "adminpass"
    When I register a "AVRO" schema under subject "ldap-admin-subject":
      """
      {"type":"record","name":"LdapAdmin","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  @ldap
  Scenario: Admin can read a schema
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-read-subject":
      """
      {"type":"record","name":"LdapRead","fields":[{"name":"id","type":"int"}]}
      """
    When I GET "/subjects/ldap-read-subject/versions/1"
    Then the response status should be 200

  @ldap
  Scenario: Admin can delete a schema version
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-del-ver":
      """
      {"type":"record","name":"LdapDelVer","fields":[{"name":"id","type":"int"}]}
      """
    When I DELETE "/subjects/ldap-del-ver/versions/1"
    Then the response status should be 200

  @ldap
  Scenario: Admin can delete a subject
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-del-subj":
      """
      {"type":"record","name":"LdapDelSubj","fields":[{"name":"id","type":"int"}]}
      """
    When I DELETE "/subjects/ldap-del-subj"
    Then the response status should be 200

  @ldap
  Scenario: Developer can register a schema
    Given I authenticate as "developer" with password "devpass"
    When I register a "AVRO" schema under subject "ldap-dev-subject":
      """
      {"type":"record","name":"LdapDev","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200

  @ldap
  Scenario: Developer can read a schema
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-dev-read":
      """
      {"type":"record","name":"LdapDevRead","fields":[{"name":"id","type":"int"}]}
      """
    Given I authenticate as "developer" with password "devpass"
    When I GET "/subjects/ldap-dev-read/versions/1"
    Then the response status should be 200

  @ldap
  Scenario: Developer cannot delete a schema version
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-dev-nodel":
      """
      {"type":"record","name":"LdapDevNoDel","fields":[{"name":"id","type":"int"}]}
      """
    Given I authenticate as "developer" with password "devpass"
    When I DELETE "/subjects/ldap-dev-nodel/versions/1"
    Then the response status should be 403

  @ldap
  Scenario: Developer cannot delete a subject
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-dev-nodelsubj":
      """
      {"type":"record","name":"LdapDevNoDelSubj","fields":[{"name":"id","type":"int"}]}
      """
    Given I authenticate as "developer" with password "devpass"
    When I DELETE "/subjects/ldap-dev-nodelsubj"
    Then the response status should be 403

  @ldap
  Scenario: Readonly can read a schema
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-ro-read":
      """
      {"type":"record","name":"LdapRORead","fields":[{"name":"id","type":"int"}]}
      """
    Given I authenticate as "readonly" with password "readonlypass"
    When I GET "/subjects/ldap-ro-read/versions/1"
    Then the response status should be 200

  @ldap
  Scenario: Readonly cannot register a schema
    Given I authenticate as "readonly" with password "readonlypass"
    When I register a "AVRO" schema under subject "ldap-ro-write":
      """
      {"type":"record","name":"LdapROWrite","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403

  @ldap
  Scenario: Readonly cannot delete a schema version
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-ro-nodel":
      """
      {"type":"record","name":"LdapRONoDel","fields":[{"name":"id","type":"int"}]}
      """
    Given I authenticate as "readonly" with password "readonlypass"
    When I DELETE "/subjects/ldap-ro-nodel/versions/1"
    Then the response status should be 403

  @ldap
  Scenario: Nogroup user (default readonly) cannot register a schema
    Given I authenticate as "nogroup" with password "nogrouppass"
    When I register a "AVRO" schema under subject "ldap-ng-write":
      """
      {"type":"record","name":"LdapNGWrite","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403

  # ---------------------------------------------------------------------------
  # Section 3: RBAC — Config operations
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: Admin can read global config
    Given I authenticate as "admin" with password "adminpass"
    When I GET "/config"
    Then the response status should be 200

  @ldap
  Scenario: Admin can update global config
    Given I authenticate as "admin" with password "adminpass"
    When I PUT "/config" with body:
      """
      {"compatibility":"FULL"}
      """
    Then the response status should be 200

  @ldap
  Scenario: Admin can delete global config
    Given I authenticate as "admin" with password "adminpass"
    When I DELETE "/config"
    Then the response status should be 200

  @ldap
  Scenario: Admin can read and write per-subject config
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-cfg-subj":
      """
      {"type":"record","name":"LdapCfgSubj","fields":[{"name":"id","type":"int"}]}
      """
    When I PUT "/config/ldap-cfg-subj" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 200
    When I GET "/config/ldap-cfg-subj"
    Then the response status should be 200

  @ldap
  Scenario: Developer can read config
    Given I authenticate as "developer" with password "devpass"
    When I GET "/config"
    Then the response status should be 200

  @ldap
  Scenario: Developer cannot update config
    Given I authenticate as "developer" with password "devpass"
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  @ldap
  Scenario: Readonly can read config
    Given I authenticate as "readonly" with password "readonlypass"
    When I GET "/config"
    Then the response status should be 200

  @ldap
  Scenario: Readonly cannot update config
    Given I authenticate as "readonly" with password "readonlypass"
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  # ---------------------------------------------------------------------------
  # Section 4: RBAC — Mode operations
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: Admin can read mode
    Given I authenticate as "admin" with password "adminpass"
    When I GET "/mode"
    Then the response status should be 200

  @ldap
  Scenario: Admin can update mode
    Given I authenticate as "admin" with password "adminpass"
    When I PUT "/mode" with body:
      """
      {"mode":"READWRITE"}
      """
    Then the response status should be 200

  @ldap
  Scenario: Admin can delete mode
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-mode-subj":
      """
      {"type":"record","name":"LdapModeSubj","fields":[{"name":"id","type":"int"}]}
      """
    And I PUT "/mode/ldap-mode-subj" with body:
      """
      {"mode":"READONLY"}
      """
    When I DELETE "/mode/ldap-mode-subj"
    Then the response status should be 200

  @ldap
  Scenario: Developer can read mode
    Given I authenticate as "developer" with password "devpass"
    When I GET "/mode"
    Then the response status should be 200

  @ldap
  Scenario: Developer cannot update mode
    Given I authenticate as "developer" with password "devpass"
    When I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 403

  @ldap
  Scenario: Readonly can read mode
    Given I authenticate as "readonly" with password "readonlypass"
    When I GET "/mode"
    Then the response status should be 200

  @ldap
  Scenario: Readonly cannot update mode
    Given I authenticate as "readonly" with password "readonlypass"
    When I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 403

  # ---------------------------------------------------------------------------
  # Section 5: RBAC — Compatibility check
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: Admin can check compatibility
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-compat":
      """
      {"type":"record","name":"LdapCompat","fields":[{"name":"id","type":"int"}]}
      """
    When I check compatibility of schema against subject "ldap-compat":
      """
      {"type":"record","name":"LdapCompat","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200
    And the response body should contain "is_compatible"

  @ldap
  Scenario: Developer can check compatibility
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-compat-dev":
      """
      {"type":"record","name":"LdapCompatDev","fields":[{"name":"id","type":"int"}]}
      """
    Given I authenticate as "developer" with password "devpass"
    When I check compatibility of schema against subject "ldap-compat-dev":
      """
      {"type":"record","name":"LdapCompatDev","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  @ldap
  Scenario: Readonly can check compatibility
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-compat-ro":
      """
      {"type":"record","name":"LdapCompatRO","fields":[{"name":"id","type":"int"}]}
      """
    Given I authenticate as "readonly" with password "readonlypass"
    When I check compatibility of schema against subject "ldap-compat-ro":
      """
      {"type":"record","name":"LdapCompatRO","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # Section 6: RBAC — Encryption (KEK/DEK)
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: Admin can create and read a KEK
    Given I authenticate as "admin" with password "adminpass"
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"ldap-test-kek","kmsType":"test-kms","kmsKeyId":"test-key-id","shared":false}
      """
    Then the response status should be 200
    When I GET "/dek-registry/v1/keks/ldap-test-kek"
    Then the response status should be 200

  @ldap
  Scenario: Developer cannot create a KEK
    Given I authenticate as "developer" with password "devpass"
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"ldap-dev-kek","kmsType":"test-kms","kmsKeyId":"test-key-id","shared":false}
      """
    Then the response status should be 403

  @ldap
  Scenario: Readonly cannot create a KEK
    Given I authenticate as "readonly" with password "readonlypass"
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"ldap-ro-kek","kmsType":"test-kms","kmsKeyId":"test-key-id","shared":false}
      """
    Then the response status should be 403

  # ---------------------------------------------------------------------------
  # Section 7: RBAC — Exporter operations
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: Admin can create and read an exporter
    Given I authenticate as "admin" with password "adminpass"
    When I POST "/exporters" with body:
      """
      {"name":"ldap-exporter","subjects":["*"],"contextType":"CUSTOM","context":"ldap-ctx","config":{"schema.registry.url":"http://localhost:8081"}}
      """
    Then the response status should be 200
    When I GET "/exporters/ldap-exporter"
    Then the response status should be 200

  @ldap
  Scenario: Developer cannot create an exporter
    Given I authenticate as "developer" with password "devpass"
    When I POST "/exporters" with body:
      """
      {"name":"ldap-dev-exporter","subjects":["*"],"contextType":"CUSTOM","context":"ldap-ctx","config":{"schema.registry.url":"http://localhost:8081"}}
      """
    Then the response status should be 403

  @ldap
  Scenario: Readonly cannot create an exporter
    Given I authenticate as "readonly" with password "readonlypass"
    When I POST "/exporters" with body:
      """
      {"name":"ldap-ro-exporter","subjects":["*"],"contextType":"CUSTOM","context":"ldap-ctx","config":{"schema.registry.url":"http://localhost:8081"}}
      """
    Then the response status should be 403

  # ---------------------------------------------------------------------------
  # Section 8: RBAC — Admin endpoints
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: Developer cannot access admin endpoints
    Given I authenticate as "developer" with password "devpass"
    When I GET "/admin/users"
    Then the response status should be 403

  @ldap
  Scenario: Readonly cannot access admin endpoints
    Given I authenticate as "readonly" with password "readonlypass"
    When I GET "/admin/users"
    Then the response status should be 403

  # ---------------------------------------------------------------------------
  # Section 9: Audit assertions
  # ---------------------------------------------------------------------------

  @ldap
  Scenario: Successful LDAP login produces audit event with auth_method=ldap
    Given I authenticate as "admin" with password "adminpass"
    When I register a "AVRO" schema under subject "ldap-audit-subj":
      """
      {"type":"record","name":"LdapAudit","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type  | schema_register |
      | outcome     | success         |
      | actor_id    | admin           |
      | actor_type  | user            |
      | auth_method | ldap            |
      | role        | admin           |
      | method      | POST            |
      | status_code | 200             |

  @ldap
  Scenario: Failed LDAP login produces auth_failure event
    # admin exists in LDAP — wrong password is rejected immediately (no fallback).
    Given I authenticate as "admin" with password "wrongpassword"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type  | auth_failure         |
      | outcome     | failure              |
      | actor_type  | anonymous            |
      | reason      | no_valid_credentials |
      | method      | GET                  |
      | path        | /subjects            |
      | status_code | 401                  |

  @ldap
  Scenario: LDAP developer schema register produces audit event with correct actor and role
    Given I authenticate as "developer" with password "devpass"
    When I register a "AVRO" schema under subject "ldap-audit-dev":
      """
      {"type":"record","name":"LdapAuditDev","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type  | schema_register |
      | outcome     | success         |
      | actor_id    | developer       |
      | actor_type  | user            |
      | auth_method | ldap            |
      | role        | developer       |
      | method      | POST            |
      | status_code | 200             |

  @ldap
  Scenario: LDAP forbidden action produces auth_forbidden event
    Given I authenticate as "readonly" with password "readonlypass"
    When I register a "AVRO" schema under subject "ldap-audit-forbidden":
      """
      {"type":"record","name":"LdapAuditForbidden","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type  | auth_forbidden    |
      | outcome     | failure           |
      | actor_id    | readonly          |
      | actor_type  | user              |
      | auth_method | ldap              |
      | role        | readonly          |
      | reason      | permission_denied |
      | method      | POST              |
      | status_code | 403               |

  @ldap
  Scenario: No credentials produces auth_failure audit event
    Given I clear authentication
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type  | auth_failure         |
      | outcome     | failure              |
      | actor_type  | anonymous            |
      | reason      | no_valid_credentials |
      | method      | GET                  |
      | path        | /subjects            |
      | status_code | 401                  |

  @ldap
  Scenario: Non-existent user produces auth_failure audit event
    Given I authenticate as "nonexistent" with password "anypassword"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type  | auth_failure         |
      | outcome     | failure              |
      | actor_type  | anonymous            |
      | reason      | no_valid_credentials |
      | method      | GET                  |
      | path        | /subjects            |
      | status_code | 401                  |

  @ldap
  Scenario: LDAP admin delete produces audit event with correct actor
    Given I authenticate as "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "ldap-audit-del":
      """
      {"type":"record","name":"LdapAuditDel","fields":[{"name":"id","type":"int"}]}
      """
    When I DELETE "/subjects/ldap-audit-del/versions/1"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type  | schema_delete |
      | outcome     | success       |
      | actor_id    | admin         |
      | actor_type  | user          |
      | auth_method | ldap          |
      | role        | admin         |
      | method      | DELETE        |
      | status_code | 200           |

  # ---------------------------------------------------------------------------
  # Section 10: LDAP fallback to database authentication
  # ---------------------------------------------------------------------------
  # The config has allow_fallback: true and a bootstrap user (localadmin/localadminpass)
  # that exists only in the database, not in LDAP.
  #
  # Fallback policy:
  #   - "user not found" in LDAP → fall back to DB/htpasswd (the user may be local-only)
  #   - "invalid credentials" in LDAP → reject immediately, NO fallback (prevents
  #     bypassing LDAP password policies: complexity, expiry, lockout, MFA)

  @ldap
  Scenario: User not in LDAP falls back to DB bootstrap user and authenticates
    # localadmin exists only in the database (via bootstrap), not in LDAP.
    # LDAP returns "user not found", fallback to DB succeeds.
    # The auth_method on subsequent requests is "ldap_fallback" (not "basic").
    # We use PUT /config (a default audit event) so we can assert auth_method.
    Given I authenticate as "localadmin" with password "localadminpass"
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type | auth_ldap_fallback                  |
      | outcome    | warning                             |
      | actor_id   | localadmin                          |
      | actor_type | user                                |
      | reason     | ldap_user_not_found_fallback_to_db  |
      | path       | /config                             |
    And the audit log should contain an event:
      | event_type  | config_update   |
      | outcome     | success         |
      | actor_id    | localadmin      |
      | auth_method | ldap_fallback   |
      | status_code | 200             |

  @ldap
  Scenario: LDAP user with wrong password is rejected immediately — no fallback
    # admin exists in LDAP. Wrong password returns "invalid credentials".
    # Fallback is NOT attempted — this prevents bypassing LDAP password policies.
    Given I authenticate as "admin" with password "wrongpassword"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type  | auth_failure         |
      | outcome     | failure              |
      | actor_type  | anonymous            |
      | reason      | no_valid_credentials |
      | status_code | 401                  |

  @ldap
  Scenario: User not in LDAP or DB returns 401 with fallback audit event
    # unknownuser doesn't exist in LDAP or DB. LDAP fails with "user not found",
    # fallback is attempted but also fails.
    Given I authenticate as "unknownuser" with password "somepassword"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type | auth_ldap_fallback                  |
      | outcome    | warning                             |
      | actor_id   | unknownuser                         |
      | reason     | ldap_user_not_found_fallback_to_db  |
    And the audit log should contain an event:
      | event_type  | auth_failure         |
      | outcome     | failure              |
      | actor_type  | anonymous            |
      | reason      | no_valid_credentials |
      | status_code | 401                  |
