@audit-outputs
Feature: Enterprise Audit Log Outputs
  As a compliance officer
  I want audit events delivered to multiple outputs simultaneously
  So that events are available in file, syslog, and webhook destinations

  Background:
    Given the schema registry is running

  # ──────────────────────────────────────────────────────────
  # File output
  # ──────────────────────────────────────────────────────────

  @audit-outputs @file
  Scenario: File output delivers JSON audit events
    When I register a schema under subject "audit-file-test":
      """
      {"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type | schema_register |
      | outcome    | success         |
      | target_id  | audit-file-test |
      | method     | POST            |

  @audit-outputs @file
  Scenario: File output includes all standard audit fields
    When I register a schema under subject "audit-file-fields":
      """
      {"type":"record","name":"FieldTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type  | schema_register    |
      | outcome     | success            |
      | target_id   | audit-file-fields  |
      | actor_type  | anonymous          |
      | method      | POST               |

  # ──────────────────────────────────────────────────────────
  # Webhook output
  # ──────────────────────────────────────────────────────────

  @audit-outputs @webhook
  Scenario: Webhook output delivers audit events
    When I register a schema under subject "audit-webhook-test":
      """
      {"type":"record","name":"WebhookTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the webhook receiver should have received an event with event_type "schema_register"

  @audit-outputs @webhook
  Scenario: Webhook output delivers events for schema deletion
    Given I register a schema under subject "audit-webhook-delete":
      """
      {"type":"record","name":"DelTest","fields":[{"name":"id","type":"int"}]}
      """
    And the response status should be 200
    When I delete subject "audit-webhook-delete"
    Then the response status should be 200
    And the webhook receiver should have received an event with event_type "subject_delete_soft"

  @audit-outputs @webhook
  Scenario: Webhook output delivers config update events
    When I set the config for subject "audit-webhook-config" to "FULL"
    Then the response status should be 200
    And the webhook receiver should have received an event with event_type "config_update"

  # ──────────────────────────────────────────────────────────
  # Syslog output (TLS)
  # ──────────────────────────────────────────────────────────

  @audit-outputs @syslog @tls
  Scenario: Syslog output delivers audit events via TLS
    When I register a schema under subject "audit-syslog-test":
      """
      {"type":"record","name":"SyslogTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the syslog TLS receiver should have received a message containing "schema_register"

  @audit-outputs @syslog @tls
  Scenario: Syslog TLS output includes app name in messages
    When I register a schema under subject "audit-syslog-app":
      """
      {"type":"record","name":"AppTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the syslog TLS receiver should have received a message containing "schema-registry-test"

  @audit-outputs @syslog @tls
  Scenario: Syslog TLS output delivers deletion events
    Given I register a schema under subject "audit-syslog-del":
      """
      {"type":"record","name":"SyslogDel","fields":[{"name":"id","type":"int"}]}
      """
    And the response status should be 200
    When I delete subject "audit-syslog-del"
    Then the response status should be 200
    And the syslog TLS receiver should have received a message containing "subject_delete_soft"

  # ──────────────────────────────────────────────────────────
  # Multi-output fan-out
  # ──────────────────────────────────────────────────────────

  @audit-outputs @multi-output
  Scenario: Events are delivered to all outputs simultaneously
    When I register a schema under subject "audit-multi-test":
      """
      {"type":"record","name":"MultiTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type | schema_register  |
      | target_id  | audit-multi-test |
    And the webhook receiver should have received an event with event_type "schema_register"
    And the syslog TLS receiver should have received a message containing "schema_register"

  @audit-outputs @multi-output
  Scenario: Multiple events are delivered to all outputs
    Given I register a schema under subject "audit-multi-a":
      """
      {"type":"record","name":"MultiA","fields":[{"name":"id","type":"int"}]}
      """
    And the response status should be 200
    When I register a schema under subject "audit-multi-b":
      """
      {"type":"record","name":"MultiB","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the webhook receiver should have at least 2 events

  # ──────────────────────────────────────────────────────────
  # Graceful shutdown
  # ──────────────────────────────────────────────────────────

  @audit-outputs @shutdown
  Scenario: Pending events are flushed on graceful shutdown
    When I register a schema under subject "audit-shutdown-test":
      """
      {"type":"record","name":"ShutdownTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type | schema_register     |
      | target_id  | audit-shutdown-test |

  # ──────────────────────────────────────────────────────────
  # Event field verification across outputs
  # ──────────────────────────────────────────────────────────

  @audit-outputs @fields
  Scenario: Webhook events contain full audit fields
    When I register a schema under subject "audit-fields-test":
      """
      {"type":"record","name":"FieldsTest","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the webhook receiver should have received an event matching:
      | event_type | schema_register   |
      | outcome    | success           |
      | target_id  | audit-fields-test |
      | method     | POST              |
      | actor_type | anonymous         |

  @audit-outputs @fields
  Scenario: Failure events are recorded across outputs
    When I set the global config to "INVALID_LEVEL"
    Then the response status should be 422
    And the audit log should contain an event:
      | event_type | config_update |
      | outcome    | failure       |
    And the webhook receiver should have received an event with event_type "config_update"

  @audit-outputs @fields
  Scenario: Delete events include target information
    Given I register a schema under subject "audit-delete-target":
      """
      {"type":"record","name":"DeleteTarget","fields":[{"name":"id","type":"int"}]}
      """
    And the response status should be 200
    When I delete subject "audit-delete-target"
    Then the response status should be 200
    And the webhook receiver should have received an event matching:
      | event_type  | subject_delete_soft      |
      | outcome     | success             |
      | target_id   | audit-delete-target |
      | target_type | subject             |
