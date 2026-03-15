@jwt @auth
Feature: JWT Authentication and RBAC
  The schema registry MUST support JWT authentication via Bearer tokens
  signed with RS256 and enforce role-based access control based on the
  `role` claim in the JWT payload.

  Tokens are generated locally using a pre-generated RSA key pair.
  The registry verifies tokens using the mounted public key.

  Test roles (embedded directly in JWT claims):
    - admin (sub=admin, role=admin) -> admin role
    - developer (sub=developer, role=developer) -> developer role
    - readonly (sub=readonly, role=readonly) -> readonly role
    - nogroup (sub=nogroup, no role claim) -> default readonly role

  # ===================================================================
  # Section 1: Authentication (token validation)
  # ===================================================================

  Scenario: Valid JWT with admin role authenticates successfully
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Valid JWT with developer role authenticates successfully
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Valid JWT with readonly role authenticates successfully
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Valid JWT with no role claim gets default readonly role
    Given I generate a JWT token with claims:
      | sub | nogroup         |
      | iss | test-issuer     |
      | aud | schema-registry |
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Valid JWT using preferred_username claim (no sub)
    Given I generate a JWT token with claims:
      | preferred_username | prefuser         |
      | role               | readonly         |
      | iss                | test-issuer      |
      | aud                | schema-registry  |
    When I GET "/subjects"
    Then the response status should be 200

  Scenario: Expired JWT returns 401
    Given I generate an expired JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Invalid signature (signed with wrong key) returns 401
    Given I generate a JWT token signed with wrong key with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
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

  Scenario: Wrong issuer claim returns 401
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | wrong-issuer    |
      | aud  | schema-registry |
    When I GET "/subjects"
    Then the response status should be 401

  Scenario: Wrong audience claim returns 401
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | wrong-audience  |
    When I GET "/subjects"
    Then the response status should be 401

  # ===================================================================
  # Section 2: RBAC - Schema operations
  # ===================================================================

  Scenario: Admin can register a schema
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I register a "AVRO" schema under subject "jwt-admin-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Admin can read a schema
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-read-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I GET "/subjects/jwt-read-subject/versions/1"
    Then the response status should be 200

  Scenario: Admin can delete a schema version
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-del-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I DELETE "/subjects/jwt-del-subject/versions/1"
    Then the response status should be 200

  Scenario: Admin can delete a subject
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-delsub-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I DELETE "/subjects/jwt-delsub-subject"
    Then the response status should be 200

  Scenario: Developer can register a schema
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I register a "AVRO" schema under subject "jwt-dev-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200

  Scenario: Developer can read a schema
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-devread-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/subjects/jwt-devread-subject/versions/1"
    Then the response status should be 200

  Scenario: Developer cannot delete a schema version
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-devdel-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I DELETE "/subjects/jwt-devdel-subject/versions/1"
    Then the response status should be 403

  Scenario: Developer cannot delete a subject
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-devdelsub":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I DELETE "/subjects/jwt-devdelsub"
    Then the response status should be 403

  Scenario: Readonly can read a schema
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-roread-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/subjects/jwt-roread-subject/versions/1"
    Then the response status should be 200

  Scenario: Readonly cannot register a schema
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I register a "AVRO" schema under subject "jwt-roreg-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 403

  Scenario: Readonly cannot delete a schema version
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-rodel-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I DELETE "/subjects/jwt-rodel-subject/versions/1"
    Then the response status should be 403

  Scenario: Nogroup user (default readonly) cannot register a schema
    Given I generate a JWT token with claims:
      | sub | nogroup         |
      | iss | test-issuer     |
      | aud | schema-registry |
    When I register a "AVRO" schema under subject "jwt-nogroup-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 3: RBAC - Config operations
  # ===================================================================

  Scenario: Admin can read global config
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/config"
    Then the response status should be 200

  Scenario: Admin can update global config
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 200

  Scenario: Admin can delete global config
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I DELETE "/config"
    Then the response status should be 200

  Scenario: Admin can read and write per-subject config
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-config-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I PUT "/config/jwt-config-subject" with body:
      """
      {"compatibility":"FULL"}
      """
    Then the response status should be 200
    When I GET "/config/jwt-config-subject"
    Then the response status should be 200
    And the response field "compatibilityLevel" should be "FULL"

  Scenario: Developer can read config
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/config"
    Then the response status should be 200

  Scenario: Developer cannot update config
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  Scenario: Readonly can read config
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/config"
    Then the response status should be 200

  Scenario: Readonly cannot update config
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 4: RBAC - Mode operations
  # ===================================================================

  Scenario: Admin can read mode
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/mode"
    Then the response status should be 200

  Scenario: Admin can update mode
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
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
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I DELETE "/mode"
    Then the response status should be 200

  Scenario: Developer can read mode
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/mode"
    Then the response status should be 200

  Scenario: Developer cannot update mode
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 403

  Scenario: Readonly can read mode
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/mode"
    Then the response status should be 200

  Scenario: Readonly cannot update mode
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I PUT "/mode" with body:
      """
      {"mode":"READONLY"}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 5: RBAC - Compatibility check
  # ===================================================================

  Scenario: Admin can check compatibility
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-compat-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    When I check compatibility of schema against subject "jwt-compat-subject":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":""}]}
      """
    Then the response status should be 200

  Scenario: Developer can check compatibility
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-compat-dev":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I check compatibility of schema against subject "jwt-compat-dev":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":""}]}
      """
    Then the response status should be 200

  Scenario: Readonly can check compatibility
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    And I register a "AVRO" schema under subject "jwt-compat-ro":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"}]}
      """
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I check compatibility of schema against subject "jwt-compat-ro":
      """
      {"type":"record","name":"Test","fields":[{"name":"f1","type":"string"},{"name":"f2","type":"string","default":""}]}
      """
    Then the response status should be 200

  # ===================================================================
  # Section 6: RBAC - KEK operations
  # ===================================================================

  Scenario: Admin can create and read a KEK
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"jwt-test-kek","kmsType":"local","kmsKeyId":"test-key","doc":"JWT test KEK"}
      """
    Then the response status should be 200
    When I GET "/dek-registry/v1/keks/jwt-test-kek"
    Then the response status should be 200

  Scenario: Developer cannot create a KEK
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"jwt-dev-kek","kmsType":"local","kmsKeyId":"test-key"}
      """
    Then the response status should be 403

  Scenario: Readonly cannot create a KEK
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"jwt-ro-kek","kmsType":"local","kmsKeyId":"test-key"}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 7: RBAC - Exporter operations
  # ===================================================================

  Scenario: Admin can create and read an exporter
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I POST "/exporters" with body:
      """
      {"name":"jwt-test-exporter","contextType":"CUSTOM","context":"jwt-ctx","subjects":["jwt-exp-*"],"config":{"schema.registry.url":"http://localhost:8082"}}
      """
    Then the response status should be 200
    When I GET "/exporters/jwt-test-exporter"
    Then the response status should be 200

  Scenario: Developer cannot create an exporter
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I POST "/exporters" with body:
      """
      {"name":"jwt-dev-exporter","contextType":"CUSTOM","context":"jwt-ctx","subjects":["test-*"],"config":{"schema.registry.url":"http://localhost:8082"}}
      """
    Then the response status should be 403

  Scenario: Readonly cannot create an exporter
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I POST "/exporters" with body:
      """
      {"name":"jwt-ro-exporter","contextType":"CUSTOM","context":"jwt-ctx","subjects":["test-*"],"config":{"schema.registry.url":"http://localhost:8082"}}
      """
    Then the response status should be 403

  # ===================================================================
  # Section 8: RBAC - Admin endpoints
  # ===================================================================

  Scenario: Developer cannot access admin endpoints
    Given I generate a JWT token with claims:
      | sub  | developer       |
      | role | developer       |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/admin/users"
    Then the response status should be 403

  Scenario: Readonly cannot access admin endpoints
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/admin/users"
    Then the response status should be 403

  # ===================================================================
  # Section 9: Audit assertions
  # ===================================================================

  Scenario: Successful JWT login produces audit event with auth_method=jwt
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | config_update |
      | outcome              | success       |
      | actor_id             | admin         |
      | actor_type           | user          |
      | auth_method          | jwt           |
      | role                 | admin         |
      | target_type          | config        |
      | target_id            | _global       |
      | schema_id            |               |
      | version              |               |
      | schema_type          |               |
      | before_hash          | *             |
      | after_hash           | sha256:*      |
      | context              | .             |
      | transport_security   | tls           |
      | method               | PUT           |
      | path                 | /config       |
      | status_code          | 200           |
      | reason               |               |
      | error                |               |
      | request_body         |               |
      | metadata             |               |
      | timestamp            | *             |
      | duration_ms          | *             |
      | request_id           | *             |
      | source_ip            | *             |
      | user_agent           | *             |

  Scenario: Failed JWT login (expired) produces auth_failure event
    Given I generate an expired JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type           | auth_failure         |
      | outcome              | failure              |
      | actor_id             |                      |
      | actor_type           | anonymous            |
      | auth_method          |                      |
      | role                 |                      |
      | target_type          |                      |
      | target_id            |                      |
      | schema_id            |                      |
      | version              |                      |
      | schema_type          |                      |
      | before_hash          |                      |
      | after_hash           |                      |
      | context              | .                     |
      | transport_security   | tls                  |
      | method               | GET                  |
      | path                 | /subjects            |
      | status_code          | 401                  |
      | reason               | no_valid_credentials |
      | error                |                      |
      | request_body         |                      |
      | metadata             |                      |
      | timestamp            | *                    |
      | duration_ms          | *                    |
      | request_id           | *                    |
      | source_ip            | *                    |
      | user_agent           | *                    |

  Scenario: JWT admin schema register produces audit event with correct actor and role
    Given I generate a JWT token with claims:
      | sub  | admin           |
      | role | admin           |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I register a "AVRO" schema under subject "jwt-audit-subject":
      """
      {"type":"record","name":"AuditTest","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register   |
      | outcome              | success           |
      | actor_id             | admin             |
      | actor_type           | user              |
      | auth_method          | jwt               |
      | role                 | admin             |
      | target_type          | subject           |
      | target_id            | jwt-audit-subject |
      | schema_id            | *                 |
      | version              | *                 |
      | schema_type          | AVRO              |
      | before_hash          |                   |
      | after_hash           | sha256:*          |
      | context              | .                 |
      | transport_security   | tls               |
      | method               | POST              |
      | path                 | /subjects         |
      | status_code          | 200               |
      | reason               |                   |
      | error                |                   |
      | request_body         |                   |
      | metadata             |                   |
      | timestamp            | *                 |
      | duration_ms          | *                 |
      | request_id           | *                 |
      | source_ip            | *                 |
      | user_agent           | *                 |

  Scenario: JWT forbidden action produces auth_forbidden event
    Given I generate a JWT token with claims:
      | sub  | readonly        |
      | role | readonly        |
      | iss  | test-issuer     |
      | aud  | schema-registry |
    When I register a "AVRO" schema under subject "jwt-forbidden-subject":
      """
      {"type":"record","name":"Forbidden","fields":[{"name":"f1","type":"string"}]}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden    |
      | outcome              | failure           |
      | actor_id             | readonly          |
      | actor_type           | user              |
      | auth_method          | jwt               |
      | role                 | readonly          |
      | target_type          |                   |
      | target_id            |                   |
      | schema_id            |                   |
      | version              |                   |
      | schema_type          |                   |
      | before_hash          |                   |
      | after_hash           |                   |
      | context              | .                  |
      | transport_security   | tls               |
      | method               | POST              |
      | path                 | /subjects         |
      | status_code          | 403               |
      | reason               | permission_denied |
      | error                |                   |
      | request_body         |                   |
      | metadata             |                   |
      | timestamp            | *                 |
      | duration_ms          | *                 |
      | request_id           | *                 |
      | source_ip            | *                 |
      | user_agent           | *                 |

  Scenario: No credentials produces auth_failure audit event
    Given I clear authentication
    When I GET "/subjects"
    Then the response status should be 401
    And the audit log should contain an event:
      | event_type           | auth_failure         |
      | outcome              | failure              |
      | actor_id             |                      |
      | actor_type           | anonymous            |
      | auth_method          |                      |
      | role                 |                      |
      | target_type          |                      |
      | target_id            |                      |
      | schema_id            |                      |
      | version              |                      |
      | schema_type          |                      |
      | before_hash          |                      |
      | after_hash           |                      |
      | context              | .                     |
      | transport_security   | tls                  |
      | method               | GET                  |
      | path                 | /subjects            |
      | status_code          | 401                  |
      | reason               | no_valid_credentials |
      | error                |                      |
      | request_body         |                      |
      | metadata             |                      |
      | timestamp            | *                    |
      | duration_ms          | *                    |
      | request_id           | *                    |
      | source_ip            | *                    |
      | user_agent           | *                    |
