@functional @audit
Feature: REST API Audit Logging
  The REST API MUST emit audit events for security-relevant operations so that
  operators can track schema changes, config updates, and deletions.
  Unauthenticated requests MUST still be audited with an empty user field.

  # --- Schema Events ---

  Scenario: Schema registration emits schema_register audit event
    When I register a schema under subject "audit-rest-register":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register"
    And the audit log should contain event "schema_register" with subject "audit-rest-register"
    And the audit log should contain event "schema_register" with method "POST"
    And the audit log should contain event "schema_register" with path containing "/subjects/audit-rest-register/versions"

  Scenario: Schema version deletion emits schema_delete audit event
    Given I register a schema under subject "audit-rest-verdel":
      """
      {"type":"string"}
      """
    When I delete version 1 of subject "audit-rest-verdel"
    Then the response status should be 200
    And the audit log should contain event "schema_delete"
    And the audit log should contain event "schema_delete" with subject "audit-rest-verdel"

  Scenario: Schema lookup emits schema_lookup audit event
    Given I register a schema under subject "audit-rest-lookup":
      """
      {"type":"string"}
      """
    When I lookup schema in subject "audit-rest-lookup":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain event "schema_lookup"
    And the audit log should contain event "schema_lookup" with subject "audit-rest-lookup"

  # --- Subject Events ---

  Scenario: Subject deletion emits subject_delete audit event
    Given I register a schema under subject "audit-rest-delete":
      """
      {"type":"string"}
      """
    When I delete subject "audit-rest-delete"
    Then the response status should be 200
    And the audit log should contain event "subject_delete"
    And the audit log should contain event "subject_delete" with subject "audit-rest-delete"

  Scenario: Permanent subject deletion emits subject_delete audit event
    Given I register a schema under subject "audit-rest-permdel":
      """
      {"type":"string"}
      """
    And I delete subject "audit-rest-permdel"
    When I permanently delete subject "audit-rest-permdel"
    Then the response status should be 200
    And the audit log should contain event "subject_delete"
    And the audit log should contain event "subject_delete" with subject "audit-rest-permdel"

  # --- Config Events ---

  Scenario: Config update emits config_update audit event
    When I set the global compatibility level to "FULL"
    Then the response status should be 200
    And the audit log should contain event "config_update"
    And the audit log should contain event "config_update" with method "PUT"

  Scenario: Subject config update emits config_update audit event
    Given I register a schema under subject "audit-rest-cfgupd":
      """
      {"type":"string"}
      """
    When I PUT "/config/audit-rest-cfgupd" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    And the audit log should contain event "config_update"

  Scenario: Config delete emits config_delete audit event
    Given I register a schema under subject "audit-rest-cfgdel":
      """
      {"type":"string"}
      """
    And I PUT "/config/audit-rest-cfgdel" with body:
      """
      {"compatibility": "FULL"}
      """
    When I DELETE "/config/audit-rest-cfgdel"
    Then the response status should be 200
    And the audit log should contain event "config_delete"

  # --- Mode Events ---

  Scenario: Mode update emits mode_update audit event
    When I PUT "/mode" with body:
      """
      {"mode": "READWRITE"}
      """
    Then the response status should be 200
    And the audit log should contain event "mode_update"
    And the audit log should contain event "mode_update" with method "PUT"

  Scenario: Subject mode update emits mode_update audit event
    Given I register a schema under subject "audit-rest-modeupd":
      """
      {"type":"string"}
      """
    When I PUT "/mode/audit-rest-modeupd" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    And the audit log should contain event "mode_update"

  Scenario: Mode delete emits mode_delete audit event
    Given I register a schema under subject "audit-rest-modedel":
      """
      {"type":"string"}
      """
    And I PUT "/mode/audit-rest-modedel" with body:
      """
      {"mode": "READONLY"}
      """
    When I DELETE "/mode/audit-rest-modedel"
    Then the response status should be 200
    And the audit log should contain event "mode_delete"

  # --- Import Events ---

  Scenario: Schema import emits schema_import audit event
    When I PUT "/mode" with body:
      """
      {"mode": "IMPORT"}
      """
    And I POST "/import/schemas" with body:
      """
      {
        "schemas": [
          {
            "subject": "audit-rest-import",
            "version": 1,
            "id": 99901,
            "schemaType": "AVRO",
            "schema": "{\"type\":\"string\"}"
          }
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain event "schema_import"

  # --- Exporter Events ---

  Scenario: Exporter creation emits exporter_create audit event
    When I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-create",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    Then the response status should be 200
    And the audit log should contain event "exporter_create"
    And the audit log should contain event "exporter_create" with method "POST"

  Scenario: Exporter update emits exporter_update audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-update",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    When I PUT "/exporters/audit-exp-update" with body:
      """
      {
        "name": "audit-exp-update",
        "contextType": "NONE",
        "subjects": ["audit-exp-sub-new"]
      }
      """
    Then the response status should be 200
    And the audit log should contain event "exporter_update"

  Scenario: Exporter deletion emits exporter_delete audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-delete",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    When I DELETE "/exporters/audit-exp-delete"
    Then the response status should be 200
    And the audit log should contain event "exporter_delete"

  Scenario: Exporter pause emits exporter_pause audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-pause",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    When I PUT "/exporters/audit-exp-pause/pause" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain event "exporter_pause"

  Scenario: Exporter resume emits exporter_resume audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-resume",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    And I PUT "/exporters/audit-exp-resume/pause" with body:
      """
      {}
      """
    When I PUT "/exporters/audit-exp-resume/resume" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain event "exporter_resume"

  Scenario: Exporter reset emits exporter_reset audit event
    Given I POST "/exporters" with body:
      """
      {
        "name": "audit-exp-reset",
        "contextType": "AUTO",
        "subjects": ["audit-exp-sub"]
      }
      """
    When I PUT "/exporters/audit-exp-reset/reset" with body:
      """
      {}
      """
    Then the response status should be 200
    And the audit log should contain event "exporter_reset"

  # --- Cross-cutting Audit Properties ---

  Scenario: Request ID appears in audit entries
    When I register a schema under subject "audit-rest-reqid":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain "request_id"

  Scenario: Unauthenticated requests have empty user in audit log
    When I register a schema under subject "audit-rest-nouser":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register" for user ""

  Scenario: Read-only operations are not audited by default
    Given I register a schema under subject "audit-rest-readonly":
      """
      {"type":"string"}
      """
    When I GET "/subjects"
    Then the response status should be 200
    And the audit log should not contain event "subject_list"

  Scenario: Multiple write operations produce separate audit entries
    When I register a schema under subject "audit-rest-multi-1":
      """
      {"type":"string"}
      """
    And I register a schema under subject "audit-rest-multi-2":
      """
      {"type":"int"}
      """
    Then the audit log should contain "audit-rest-multi-1"
    And the audit log should contain "audit-rest-multi-2"
