@functional @audit
Feature: REST API Audit Logging
  The REST API MUST emit audit events for security-relevant operations so that
  operators can track schema changes, config updates, and deletions.

  Scenario: Schema registration emits schema_register audit event
    When I register a schema under subject "audit-rest-register":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register"
    And the audit log should contain "audit-rest-register"

  Scenario: Config update emits config_update audit event
    When I set the global compatibility level to "FULL"
    Then the response status should be 200
    And the audit log should contain event "config_update"

  Scenario: Subject deletion emits subject_delete audit event
    Given I register a schema under subject "audit-rest-delete":
      """
      {"type":"string"}
      """
    When I delete subject "audit-rest-delete"
    Then the response status should be 200
    And the audit log should contain event "subject_delete"
    And the audit log should contain "audit-rest-delete"

  Scenario: Request ID appears in audit entries
    When I register a schema under subject "audit-rest-reqid":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And the audit log should contain "request_id"
