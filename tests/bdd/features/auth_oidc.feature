@oidc @auth
Feature: OIDC Authentication and RBAC
  The schema registry MUST support OIDC (OpenID Connect) authentication via
  Bearer tokens issued by Keycloak 24.0 and enforce role-based access control
  based on OIDC group membership claims.

  Test users (provisioned in Keycloak):
    - admin (preferred_username=admin) -> schema-registry-admins group -> admin role
    - developer (preferred_username=developer) -> developers group -> developer role
    - readonly (preferred_username=readonly) -> readonly-users group -> readonly role
    - nogroup (preferred_username=nogroup) -> no group membership -> default readonly role

  # ===================================================================
  # Section 1: Authentication (token validation)
  # ===================================================================

  Scenario: OIDC admin user authenticates successfully with valid token
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: OIDC developer user authenticates successfully
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: OIDC readonly user authenticates successfully
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: OIDC nogroup user authenticates successfully with default role
    Given I obtain an OIDC token for "nogroup" with password "nogrouppass"
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Invalid Bearer token returns 401
    Given I authenticate with bearer token "invalid-token-string"
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Missing Authorization header returns 401
    Given I clear authentication
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Empty Bearer token returns 401
    Given I authenticate with bearer token ""
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Malformed token (not a JWT) returns 401
    Given I authenticate with bearer token "this.is.not.a.valid.jwt.token"
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Wrong auth scheme (Basic instead of Bearer) returns 401
    Given I authenticate as "admin" with password "adminpass"
    When I GET "/subjects"
    Then the response status should be 401

  # ===================================================================
  # Section 2: RBAC - Schema operations
  # ===================================================================

  Scenario: Admin can register a schema
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I register a "AVRO" schema under subject "oidc-admin-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Admin can read a schema
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-read-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I GET "/subjects/oidc-read-subject/versions/1"
    Then the response status should be 200

  Scenario: Admin can delete a schema version
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-del-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I DELETE "/subjects/oidc-del-subject/versions/1"
    Then the response status should be 200

  Scenario: Admin can delete a subject
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-delsub-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I DELETE "/subjects/oidc-delsub-subject"
    Then the response status should be 200

  Scenario: Developer can register a schema
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I register a "AVRO" schema under subject "oidc-dev-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Developer can read a schema
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-devread-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I GET "/subjects/oidc-devread-subject/versions/1"
    Then the response status should be 200

  Scenario: Developer cannot delete a schema version
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-devdel-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I DELETE "/subjects/oidc-devdel-subject/versions/1"
    Then the response status should be 403

  Scenario: Developer cannot delete a subject
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-devdelsub":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I DELETE "/subjects/oidc-devdelsub"
    Then the response status should be 403

  Scenario: Readonly can read a schema
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-roread-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I GET "/subjects/oidc-roread-subject/versions/1"
    Then the response status should be 200

  Scenario: Readonly cannot register a schema
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I register a "AVRO" schema under subject "oidc-roreg-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 403

  Scenario: Readonly cannot delete a schema version
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-rodel-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I DELETE "/subjects/oidc-rodel-subject/versions/1"
    Then the response status should be 403

  Scenario: Nogroup user (default readonly) cannot register a schema
    Given I obtain an OIDC token for "nogroup" with password "nogrouppass"
    When I register a "AVRO" schema under subject "oidc-nogroup-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 3: RBAC - Config operations
  # ===================================================================

  Scenario: Admin can read global config
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I GET "/config"
    Then the response status should be 200

  Scenario: Admin can update global config
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 200

  Scenario: Admin can delete global config
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I DELETE "/config"
    Then the response status should be 200

  Scenario: Admin can read and write per-subject config
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-config-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I PUT "/config/oidc-config-subject" with body:
      """
      {"compatibility":"FULL"}
      """
    Then the response status should be 200
    When I GET "/config/oidc-config-subject"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  Scenario: Developer can read config
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I GET "/config"
    Then the response status should be 200

  Scenario: Developer cannot update config
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  Scenario: Readonly can read config
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I GET "/config"
    Then the response status should be 200

  Scenario: Readonly cannot update config
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 4: RBAC - Mode operations
  # ===================================================================

  Scenario: Admin can read mode
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I GET "/mode"
    Then the response status should be 200

  Scenario: Admin can update mode
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 200
    # Reset mode so other tests aren't affected
    When I PUT "/mode" with body:
      """
      {"mode":"READWRITE"}
      """
    Then the response status should be 200

  Scenario: Admin can delete mode
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I DELETE "/mode"
    Then the response status should be 200

  Scenario: Developer can read mode
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I GET "/mode"
    Then the response status should be 200

  Scenario: Developer cannot update mode
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 403

  Scenario: Readonly can read mode
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I GET "/mode"
    Then the response status should be 200

  Scenario: Readonly cannot update mode
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 5: RBAC - Compatibility check
  # ===================================================================

  Scenario: Admin can check compatibility
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-compat-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "oidc-compat-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":""}]}
      """
    Then the response status should be 200

  Scenario: Developer can check compatibility
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-compat-dev":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I check compatibility of schema against subject "oidc-compat-dev":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":""}]}
      """
    Then the response status should be 200

  Scenario: Readonly can check compatibility
    Given I obtain an OIDC token for "admin" with password "adminpass"
    And I register a "AVRO" schema under subject "oidc-compat-ro":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I check compatibility of schema against subject "oidc-compat-ro":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":""}]}
      """
    Then the response status should be 200

  # ===================================================================
  # Section 6: RBAC - KEK operations
  # ===================================================================

  Scenario: Admin can create and read a KEK
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"oidc-test-kek","kmsType":"local","kmsKeyId":"test-key","doc":"OIDC test KEK"}
      """
    Then the response status should be 200
    When I GET "/dek-registry/v1/keks/oidc-test-kek"
    Then the response status should be 200

  Scenario: Developer cannot create a KEK
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"oidc-dev-kek","kmsType":"local","kmsKeyId":"test-key"}
      """
    Then the response status should be 403

  Scenario: Readonly cannot create a KEK
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"oidc-ro-kek","kmsType":"local","kmsKeyId":"test-key"}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 7: RBAC - Exporter operations
  # ===================================================================

  Scenario: Admin can create and read an exporter
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I POST "/exporters" with body:
      """
      {"name":"oidc-test-exporter","contextType":"CUSTOM","context":"oidc-ctx","subjects":["oidc-exp-*"],"config":{"schema.registry.url":"http://localhost:8082"}}
      """
    Then the response status should be 200
    When I GET "/exporters/oidc-test-exporter"
    Then the response status should be 200

  Scenario: Developer cannot create an exporter
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I POST "/exporters" with body:
      """
      {"name":"oidc-dev-exporter","contextType":"CUSTOM","context":"oidc-ctx","subjects":["test-*"],"config":{"schema.registry.url":"http://localhost:8082"}}
      """
    Then the response status should be 403

  Scenario: Readonly cannot create an exporter
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I POST "/exporters" with body:
      """
      {"name":"oidc-ro-exporter","contextType":"CUSTOM","context":"oidc-ctx","subjects":["test-*"],"config":{"schema.registry.url":"http://localhost:8082"}}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 8: RBAC - Admin endpoints
  # ===================================================================

  Scenario: Developer cannot access admin endpoints
    Given I obtain an OIDC token for "developer" with password "devpass"
    When I GET "/admin/users"
    Then the response status should be 403

  Scenario: Readonly cannot access admin endpoints
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I GET "/admin/users"
    Then the response status should be 403

  # ===================================================================
  # Section 9: Audit assertions
  # ===================================================================

  Scenario: Successful OIDC login produces audit event with auth_method=oidc
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type  | config_update |
      | outcome     | success       |
      | actor_id    | admin         |
      | auth_method | oidc          |
      | status_code | 200           |

  Scenario: Failed OIDC login produces auth_failure event
    Given I authenticate with bearer token "invalid-token"
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type | auth_failure |
      | outcome    | failure      |

  Scenario: OIDC admin schema register produces audit event with correct actor and role
    Given I obtain an OIDC token for "admin" with password "adminpass"
    When I register a "AVRO" schema under subject "oidc-audit-subject":
      """
      {"type":"record","name":"AuditTest","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type  | schema_register    |
      | outcome     | success            |
      | actor_id    | admin              |
      | auth_method | oidc               |
      | target_id   | oidc-audit-subject |
      | status_code | 200                |

  Scenario: OIDC forbidden action produces auth_forbidden event
    Given I obtain an OIDC token for "readonly" with password "readonlypass"
    When I register a "AVRO" schema under subject "oidc-forbidden-subject":
      """
      {"type":"record","name":"Forbidden","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type  | auth_forbidden |
      | outcome     | failure        |
      | actor_id    | readonly       |
      | auth_method | oidc           |

  Scenario: No credentials produces auth_failure audit event
    Given I clear authentication
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type | auth_failure |
      | outcome    | failure      |
