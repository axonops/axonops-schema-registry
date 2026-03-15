@auth
Feature: Authentication flows and RBAC
  As an operator, I want the schema registry to enforce authentication and
  role-based access control so that only authorized users can perform
  operations matching their role.

  The test server has auth enabled with basic + api_key methods, RBAC enabled,
  and a pre-seeded super_admin user: admin / admin-password (user ID=1).

  # ---------------------------------------------------------------------------
  # Authentication flows
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Valid basic auth succeeds on protected endpoint
    Given I authenticate as "admin" with password "admin-password"
    When I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: Invalid password returns 401
    Given I authenticate as "admin" with password "wrong-password"
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

  @auth
  Scenario: Non-existent user returns 401
    Given I authenticate as "nonexistent" with password "any-password"
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

  @auth
  Scenario: No auth header on protected endpoint returns 401
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

  @auth
  Scenario: Public health endpoint works without auth
    Given I clear authentication
    When I GET "/"
    Then the response status should be 200

  @auth
  Scenario: API key auth via Basic format works
    Given I authenticate as "admin" with password "admin-password"
    And I create an API key with name "test-key" role "developer" expires_in 3600
    Then the response status should be 201
    And I store the response field "key" as "my_api_key"
    When I authenticate with stored API key "my_api_key"
    And I GET "/subjects"
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # RBAC - readonly role
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Readonly user can GET /subjects
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer" password "viewer-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "viewer" with password "viewer-pass"
    And I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: Readonly user cannot POST schema
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer2" password "viewer-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "viewer2" with password "viewer-pass"
    And I register a schema under subject "test-value":
      """
      {"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden                |
      | outcome              | failure                       |
      | actor_id             | viewer2                       |
      | actor_type           | user                          |
      | auth_method          | basic                         |
      | role                 | readonly                      |
      | target_type          |                               |
      | target_id            |                               |
      | schema_id            |                               |
      | version              |                               |
      | schema_type          |                               |
      | before_hash          |                               |
      | after_hash           |                               |
      | context              | .                              |
      | transport_security   | tls                           |
      | method               | POST                          |
      | path                 | /subjects/test-value/versions |
      | status_code          | 403                           |
      | reason               | permission_denied             |
      | error                |                               |
      | request_body         |                               |
      | metadata             |                               |
      | timestamp            | *                             |
      | duration_ms          | *                             |
      | request_id           | *                             |
      | source_ip            | *                             |
      | user_agent           | *                             |

  @auth
  Scenario: Readonly user cannot DELETE subject
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer3" password "viewer-pass" role "readonly"
    Then the response status should be 201
    # First register a schema as admin so there is something to delete
    When I authenticate as "admin" with password "admin-password"
    And I register a schema under subject "to-delete-value":
      """
      {"type":"record","name":"ToDelete","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    When I authenticate as "viewer3" with password "viewer-pass"
    And I DELETE "/subjects/to-delete-value"
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden            |
      | outcome              | failure                   |
      | actor_id             | viewer3                   |
      | actor_type           | user                      |
      | auth_method          | basic                     |
      | role                 | readonly                  |
      | target_type          |                           |
      | target_id            |                           |
      | schema_id            |                           |
      | version              |                           |
      | schema_type          |                           |
      | before_hash          |                           |
      | after_hash           |                           |
      | context              | .                          |
      | transport_security   | tls                       |
      | method               | DELETE                    |
      | path                 | /subjects/to-delete-value |
      | status_code          | 403                       |
      | reason               | permission_denied         |
      | error                |                           |
      | request_body         |                           |
      | metadata             |                           |
      | timestamp            | *                         |
      | duration_ms          | *                         |
      | request_id           | *                         |
      | source_ip            | *                         |
      | user_agent           | *                         |

  @auth
  Scenario: Readonly user can GET /config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer4" password "viewer-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "viewer4" with password "viewer-pass"
    And I GET "/config"
    Then the response status should be 200

  @auth
  Scenario: Readonly user cannot PUT /config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "viewer5" password "viewer-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "viewer5" with password "viewer-pass"
    And I PUT "/config" with body:
      """
      {"compatibility":"NONE"}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden    |
      | outcome              | failure           |
      | actor_id             | viewer5           |
      | actor_type           | user              |
      | auth_method          | basic             |
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
      | method               | PUT               |
      | path                 | /config           |
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

  # ---------------------------------------------------------------------------
  # RBAC - developer role
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Developer can GET /subjects
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "dev1" password "dev-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "dev1" with password "dev-pass"
    And I GET "/subjects"
    Then the response status should be 200

  @auth
  Scenario: Developer can POST schema
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "dev2" password "dev-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "dev2" with password "dev-pass"
    And I register a schema under subject "dev-test-value":
      """
      {"type":"record","name":"DevTest","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                   |
      | outcome              | success                           |
      | actor_id             | dev2                              |
      | actor_type           | user                              |
      | auth_method          | basic                             |
      | role                 | developer                         |
      | target_type          | subject                           |
      | target_id            | dev-test-value                    |
      | schema_id            | *                                 |
      | version              | *                                 |
      | schema_type          | AVRO                              |
      | before_hash          |                                   |
      | after_hash           | sha256:*                          |
      | context              | .                                 |
      | transport_security   | tls                               |
      | method               | POST                              |
      | path                 | /subjects/dev-test-value/versions |
      | status_code          | 200                               |
      | reason               |                                   |
      | error                |                                   |
      | request_body         |                                   |
      | metadata             |                                   |
      | timestamp            | *                                 |
      | duration_ms          | *                                 |
      | request_id           | *                                 |
      | source_ip            | *                                 |
      | user_agent           | *                                 |

  @auth
  Scenario: Developer cannot DELETE subject
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "dev3" password "dev-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "admin" with password "admin-password"
    And I register a schema under subject "dev-nodelete-value":
      """
      {"type":"record","name":"DevNoDelete","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    When I authenticate as "dev3" with password "dev-pass"
    And I DELETE "/subjects/dev-nodelete-value"
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden               |
      | outcome              | failure                      |
      | actor_id             | dev3                         |
      | actor_type           | user                         |
      | auth_method          | basic                        |
      | role                 | developer                    |
      | target_type          |                              |
      | target_id            |                              |
      | schema_id            |                              |
      | version              |                              |
      | schema_type          |                              |
      | before_hash          |                              |
      | after_hash           |                              |
      | context              | .                             |
      | transport_security   | tls                          |
      | method               | DELETE                       |
      | path                 | /subjects/dev-nodelete-value |
      | status_code          | 403                          |
      | reason               | permission_denied            |
      | error                |                              |
      | request_body         |                              |
      | metadata             |                              |
      | timestamp            | *                            |
      | duration_ms          | *                            |
      | request_id           | *                            |
      | source_ip            | *                            |
      | user_agent           | *                            |

  @auth
  Scenario: Developer can GET /config
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "dev4" password "dev-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "dev4" with password "dev-pass"
    And I GET "/config"
    Then the response status should be 200

  # ---------------------------------------------------------------------------
  # RBAC - admin role
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Admin can manage schemas (register and delete)
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "mgr1" password "mgr-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "mgr1" with password "mgr-pass"
    And I register a schema under subject "admin-test-value":
      """
      {"type":"record","name":"AdminTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    When I DELETE "/subjects/admin-test-value"
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | subject_delete_soft        |
      | outcome              | success                    |
      | actor_id             | mgr1                       |
      | actor_type           | user                       |
      | auth_method          | basic                      |
      | role                 | admin                      |
      | target_type          | subject                    |
      | target_id            | admin-test-value           |
      | schema_id            |                            |
      | version              |                            |
      | schema_type          | AVRO                       |
      | before_hash          | sha256:*                   |
      | after_hash           |                            |
      | context              | .                          |
      | transport_security   | tls                        |
      | method               | DELETE                     |
      | path                 | /subjects/admin-test-value |
      | status_code          | 200                        |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           | *                          |
      | source_ip            | *                          |
      | user_agent           | *                          |

  @auth
  Scenario: Admin can read admin endpoints
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "mgr2" password "mgr-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "mgr2" with password "mgr-pass"
    And I GET "/admin/users"
    Then the response status should be 200

  @auth
  Scenario: Admin cannot write admin endpoints
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "mgr3" password "mgr-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "mgr3" with password "mgr-pass"
    And I create a user with username "should-fail" password "nope" role "readonly"
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden    |
      | outcome              | failure           |
      | actor_id             | mgr3              |
      | actor_type           | user              |
      | auth_method          | basic             |
      | role                 | admin             |
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
      | path                 | /admin/users      |
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

  # ---------------------------------------------------------------------------
  # RBAC - DEK Registry (encryption) endpoints
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Readonly user can GET /dek-registry/v1/keks
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "enc-viewer1" password "enc-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "enc-viewer1" with password "enc-pass"
    And I GET "/dek-registry/v1/keks"
    Then the response status should be 200

  @auth
  Scenario: Readonly user cannot POST KEK
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "enc-viewer2" password "enc-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "enc-viewer2" with password "enc-pass"
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"rbac-test-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123:key/test"}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden        |
      | outcome              | failure               |
      | actor_id             | enc-viewer2           |
      | actor_type           | user                  |
      | auth_method          | basic                 |
      | role                 | readonly              |
      | target_type          |                       |
      | target_id            |                       |
      | schema_id            |                       |
      | version              |                       |
      | schema_type          |                       |
      | before_hash          |                       |
      | after_hash           |                       |
      | context              | .                      |
      | transport_security   | tls                   |
      | method               | POST                  |
      | path                 | /dek-registry/v1/keks |
      | status_code          | 403                   |
      | reason               | permission_denied     |
      | error                |                       |
      | request_body         |                       |
      | metadata             |                       |
      | timestamp            | *                     |
      | duration_ms          | *                     |
      | request_id           | *                     |
      | source_ip            | *                     |
      | user_agent           | *                     |

  @auth
  Scenario: Developer can GET /dek-registry/v1/keks
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "enc-dev1" password "enc-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "enc-dev1" with password "enc-pass"
    And I GET "/dek-registry/v1/keks"
    Then the response status should be 200

  @auth
  Scenario: Developer cannot POST KEK
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "enc-dev2" password "enc-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "enc-dev2" with password "enc-pass"
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"rbac-dev-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123:key/test"}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden        |
      | outcome              | failure               |
      | actor_id             | enc-dev2              |
      | actor_type           | user                  |
      | auth_method          | basic                 |
      | role                 | developer             |
      | target_type          |                       |
      | target_id            |                       |
      | schema_id            |                       |
      | version              |                       |
      | schema_type          |                       |
      | before_hash          |                       |
      | after_hash           |                       |
      | context              | .                      |
      | transport_security   | tls                   |
      | method               | POST                  |
      | path                 | /dek-registry/v1/keks |
      | status_code          | 403                   |
      | reason               | permission_denied     |
      | error                |                       |
      | request_body         |                       |
      | metadata             |                       |
      | timestamp            | *                     |
      | duration_ms          | *                     |
      | request_id           | *                     |
      | source_ip            | *                     |
      | user_agent           | *                     |

  @auth
  Scenario: Admin can create and read KEK
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "enc-admin1" password "enc-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "enc-admin1" with password "enc-pass"
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"admin-rbac-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123:key/test"}
      """
    Then the response status should be 200
    When I GET "/dek-registry/v1/keks"
    Then the response status should be 200

  @auth
  Scenario: Admin can delete KEK
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "enc-admin2" password "enc-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "enc-admin2" with password "enc-pass"
    And I POST "/dek-registry/v1/keks" with body:
      """
      {"name":"admin-delete-kek","kmsType":"aws-kms","kmsKeyId":"arn:aws:kms:us-east-1:123:key/del"}
      """
    Then the response status should be 200
    When I DELETE "/dek-registry/v1/keks/admin-delete-kek"
    Then the response status should be 204

  # ---------------------------------------------------------------------------
  # RBAC - Exporter endpoints
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Readonly user can GET /exporters
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "exp-viewer1" password "exp-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "exp-viewer1" with password "exp-pass"
    And I GET "/exporters"
    Then the response status should be 200

  @auth
  Scenario: Readonly user cannot POST exporter
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "exp-viewer2" password "exp-pass" role "readonly"
    Then the response status should be 201
    When I authenticate as "exp-viewer2" with password "exp-pass"
    And I POST "/exporters" with body:
      """
      {"name":"rbac-test-exporter","subjects":["test"],"contextType":"CUSTOM","context":"test-ctx","config":{"schema.registry.url":"http://remote:8081"}}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden    |
      | outcome              | failure           |
      | actor_id             | exp-viewer2       |
      | actor_type           | user              |
      | auth_method          | basic             |
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
      | path                 | /exporters        |
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

  @auth
  Scenario: Developer cannot POST exporter
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "exp-dev1" password "exp-pass" role "developer"
    Then the response status should be 201
    When I authenticate as "exp-dev1" with password "exp-pass"
    And I POST "/exporters" with body:
      """
      {"name":"dev-test-exporter","subjects":["test"],"contextType":"CUSTOM","context":"test-ctx","config":{"schema.registry.url":"http://remote:8081"}}
      """
    Then the response status should be 403
    And the audit log should contain an event:
      | event_type           | auth_forbidden    |
      | outcome              | failure           |
      | actor_id             | exp-dev1          |
      | actor_type           | user              |
      | auth_method          | basic             |
      | role                 | developer         |
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
      | path                 | /exporters        |
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

  @auth
  Scenario: Admin can create and list exporters
    Given I authenticate as "admin" with password "admin-password"
    And I create a user with username "exp-admin1" password "exp-pass" role "admin"
    Then the response status should be 201
    When I authenticate as "exp-admin1" with password "exp-pass"
    And I POST "/exporters" with body:
      """
      {"name":"admin-test-exporter","subjects":["test"],"contextType":"CUSTOM","context":"test-ctx","config":{"schema.registry.url":"http://remote:8081"}}
      """
    Then the response status should be 200
    When I GET "/exporters"
    Then the response status should be 200


  # ---------------------------------------------------------------------------
  # Metrics (public endpoint, no auth needed)
  # ---------------------------------------------------------------------------

  @auth
  Scenario: Metrics endpoint returns Prometheus format without auth
    Given I clear authentication
    When I get the metrics
    Then the response status should be 200
    And the response should contain "# HELP"

  @auth
  Scenario: Metrics contain schema_registry_requests_total
    Given I clear authentication
    # Make a request first to ensure the request counter is initialized
    When I GET "/"
    Then the response status should be 200
    When I get the metrics
    Then the response status should be 200
    And the response should contain Prometheus metric "schema_registry_requests_total"
