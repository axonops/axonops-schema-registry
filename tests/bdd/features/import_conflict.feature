@functional @import @axonops-only @edge-case
Feature: Import with Conflicting IDs
  As an operator migrating schemas
  I want to understand what happens when importing schemas with IDs that conflict
  So that I can handle migration edge cases safely

  Background:
    Given the schema registry is running
    And the global compatibility level is "NONE"

  # ---------------------------------------------------------------------------
  # Import with an ID that already exists
  # ---------------------------------------------------------------------------

  Scenario: Import schema with an ID already used by another schema
    # First, switch to IMPORT mode and import a schema with ID 1000
    Given the global mode is "IMPORT"
    When I import a schema with ID 1000 under subject "import-existing" version 1:
      """
      {"type":"record","name":"Existing","fields":[{"name":"id","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    # Attempt to import a DIFFERENT schema with the same ID 1000
    When I import a schema with ID 1000 under subject "import-conflict" version 1:
      """
      {"type":"record","name":"Conflict","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 422
    # The import should report an error for the conflicting ID
    And the import should have 0 imported and 1 errors
    # Reset mode
    When I set the global mode to "READWRITE"
    # First import (succeeded) — audit entry for the successful import
    And the audit log should contain an event:
      | event_type          | schema_import     |
      | outcome             | success           |
      | actor_id            |                   |
      | actor_type          | anonymous         |
      | auth_method         |                   |
      | role                |                   |
      | target_type         | subject           |
      | target_id           | import-existing   |
      | schema_id           | *                 |
      | version             |                   |
      | schema_type         | AVRO              |
      | method              | POST              |
      | path                | /import/schemas   |
      | status_code         | 200               |
      | before_hash         |                   |
      | after_hash          | sha256:*          |
      | context             |                   |
      | transport_security  | tls               |
      | reason              |                   |
      | error               |                   |
      | request_body        |                   |
      | metadata            |                   |
      | timestamp           | *                 |
      | duration_ms         | *                 |
      | request_id          | *                 |
      | source_ip           | *                 |
      | user_agent          | *                 |
    # Second import (conflicting ID, 0 imported / 1 error) — returns 422
    And the audit log should contain an event:
      | event_type          | schema_import     |
      | outcome             | failure           |
      | actor_id            |                   |
      | actor_type          | anonymous         |
      | auth_method         |                   |
      | role                |                   |
      | target_type         | subject           |
      | target_id           | import-conflict   |
      | schema_id           | *                 |
      | version             |                   |
      | schema_type         | AVRO              |
      | method              | POST              |
      | path                | /import/schemas   |
      | status_code         | 422               |
      | before_hash         |                   |
      | after_hash          |                   |
      | context             |                   |
      | transport_security  | tls               |
      | reason              | invalid_schema     |
      | error               |                   |
      | request_body        |                   |
      | metadata            |                   |
      | timestamp           | *                 |
      | duration_ms         | *                 |
      | request_id          | *                 |
      | source_ip           | *                 |
      | user_agent          | *                 |

  # ---------------------------------------------------------------------------
  # Import with an ID that already exists but same schema content
  # ---------------------------------------------------------------------------

  Scenario: Import identical schema with same ID and subject/version is rejected
    Given the global mode is "IMPORT"
    When I import a schema with ID 1100 under subject "import-idem" version 1:
      """
      {"type":"record","name":"Idem","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    # Re-importing the same subject/version returns a conflict error even
    # when the schema content is identical — subject/version uniqueness is
    # enforced by the storage layer.
    When I import a schema with ID 1100 under subject "import-idem" version 1:
      """
      {"type":"record","name":"Idem","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 422
    And the import should have 0 imported and 1 errors
    When I set the global mode to "READWRITE"
    # First import succeeded
    And the audit log should contain an event:
      | event_type          | schema_import   |
      | outcome             | success         |
      | actor_id            |                 |
      | actor_type          | anonymous       |
      | auth_method         |                 |
      | role                |                 |
      | target_type         | subject         |
      | target_id           | import-idem     |
      | schema_id           | *               |
      | version             |                 |
      | schema_type         | AVRO            |
      | method              | POST            |
      | path                | /import/schemas |
      | status_code         | 200             |
      | before_hash         |                 |
      | after_hash          | sha256:*        |
      | context             |                 |
      | transport_security  | tls             |
      | reason              |                 |
      | error               |                 |
      | request_body        |                 |
      | metadata            |                 |
      | timestamp           | *               |
      | duration_ms         | *               |
      | request_id          | *               |
      | source_ip           | *               |
      | user_agent          | *               |
    # Re-import returns 422
    And the audit log should contain an event:
      | event_type          | schema_import   |
      | outcome             | failure         |
      | actor_id            |                 |
      | actor_type          | anonymous       |
      | auth_method         |                 |
      | role                |                 |
      | target_type         | subject         |
      | target_id           | import-idem     |
      | schema_id           | *               |
      | version             |                 |
      | schema_type         | AVRO            |
      | method              | POST            |
      | path                | /import/schemas |
      | status_code         | 422             |
      | before_hash         |                 |
      | after_hash          |                 |
      | context             |                 |
      | transport_security  | tls             |
      | reason              | invalid_schema     |
      | error               |                 |
      | request_body        |                 |
      | metadata            |                 |
      | timestamp           | *               |
      | duration_ms         | *               |
      | request_id          | *               |
      | source_ip           | *               |
      | user_agent          | *               |

  # ---------------------------------------------------------------------------
  # Import requires IMPORT mode
  # ---------------------------------------------------------------------------

  Scenario: Import fails when not in IMPORT mode
    # Default mode is READWRITE — import should fail
    When I import a schema with ID 2000 under subject "import-blocked" version 1:
      """
      {"type":"record","name":"Blocked","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 422
    # No target_id — the handler rejects before parsing the body, so the
    # subject is never extracted from the request.
    And the audit log should contain an event:
      | event_type          | schema_import     |
      | outcome             | failure           |
      | actor_id            |                   |
      | actor_type          | anonymous         |
      | auth_method         |                   |
      | role                |                   |
      | target_type         | subject           |
      | target_id           |                   |
      | schema_id           |                   |
      | version             |                   |
      | schema_type         |                   |
      | method              | POST              |
      | path                | /import/schemas   |
      | status_code         | 422               |
      | before_hash         |                   |
      | after_hash          |                   |
      | context             |                   |
      | transport_security  | tls               |
      | reason              | invalid_schema    |
      | error               |                   |
      | request_body        |                   |
      | metadata            |                   |
      | timestamp           | *                 |
      | duration_ms         | *                 |
      | request_id          | *                 |
      | source_ip           | *                 |
      | user_agent          | *                 |

  # ---------------------------------------------------------------------------
  # Import with version that already exists under same subject
  # ---------------------------------------------------------------------------

  Scenario: Import schema with version that already exists under same subject
    Given the global mode is "IMPORT"
    When I import a schema with ID 3000 under subject "import-ver-dup" version 1:
      """
      {"type":"record","name":"VerDup","fields":[{"name":"id","type":"int"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    # Import a different schema with a different ID but same subject and version
    When I import a schema with ID 3001 under subject "import-ver-dup" version 1:
      """
      {"type":"record","name":"VerDup2","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 422
    And the import should have 0 imported and 1 errors
    When I set the global mode to "READWRITE"
    # First import succeeded
    And the audit log should contain an event:
      | event_type          | schema_import    |
      | outcome             | success          |
      | actor_id            |                  |
      | actor_type          | anonymous        |
      | auth_method         |                  |
      | role                |                  |
      | target_type         | subject          |
      | target_id           | import-ver-dup   |
      | schema_id           | *                |
      | version             |                  |
      | schema_type         | AVRO             |
      | method              | POST             |
      | path                | /import/schemas  |
      | status_code         | 200              |
      | before_hash         |                  |
      | after_hash          | sha256:*         |
      | context             |                  |
      | transport_security  | tls              |
      | reason              |                  |
      | error               |                  |
      | request_body        |                  |
      | metadata            |                  |
      | timestamp           | *                |
      | duration_ms         | *                |
      | request_id          | *                |
      | source_ip           | *                |
      | user_agent          | *                |
    # Second import (0 imported / 1 error) returns 422 and audits as failure
    And the audit log should contain an event:
      | event_type          | schema_import    |
      | outcome             | failure          |
      | actor_id            |                  |
      | actor_type          | anonymous        |
      | auth_method         |                  |
      | role                |                  |
      | target_type         | subject          |
      | target_id           | import-ver-dup   |
      | schema_id           | *                |
      | version             |                  |
      | schema_type         | AVRO             |
      | method              | POST             |
      | path                | /import/schemas  |
      | status_code         | 422              |
      | before_hash         |                  |
      | after_hash          |                  |
      | context             |                  |
      | transport_security  | tls              |
      | reason              | invalid_schema     |
      | error               |                  |
      | request_body        |                  |
      | metadata            |                  |
      | timestamp           | *                |
      | duration_ms         | *                |
      | request_id          | *                |
      | source_ip           | *                |
      | user_agent          | *                |

  # ---------------------------------------------------------------------------
  # Import Protobuf and JSON Schema with conflicting IDs
  # ---------------------------------------------------------------------------

  Scenario: Import Protobuf with conflicting ID is rejected
    Given the global mode is "IMPORT"
    When I import a "PROTOBUF" schema with ID 4000 under subject "import-proto-1" version 1:
      """
      syntax = "proto3";
      message ProtoOne {
        string name = 1;
      }
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I import a "PROTOBUF" schema with ID 4000 under subject "import-proto-conflict" version 1:
      """
      syntax = "proto3";
      message ProtoConflict {
        int32 id = 1;
      }
      """
    Then the response status should be 422
    And the import should have 0 imported and 1 errors
    When I set the global mode to "READWRITE"
    # First import (succeeded)
    And the audit log should contain an event:
      | event_type          | schema_import          |
      | outcome             | success                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | target_type         | subject                |
      | target_id           | import-proto-1         |
      | schema_id           | *                      |
      | version             |                        |
      | schema_type         | PROTOBUF               |
      | method              | POST                   |
      | path                | /import/schemas        |
      | status_code         | 200                    |
      | before_hash         |                        |
      | after_hash          | sha256:*               |
      | context             |                        |
      | transport_security  | tls                    |
      | reason              |                        |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |
    # Second import (conflicting ID, 0 imported / 1 error) — returns 422
    And the audit log should contain an event:
      | event_type          | schema_import          |
      | outcome             | failure                |
      | actor_id            |                        |
      | actor_type          | anonymous              |
      | auth_method         |                        |
      | role                |                        |
      | target_type         | subject                |
      | target_id           | import-proto-conflict  |
      | schema_id           | *                      |
      | version             |                        |
      | schema_type         | PROTOBUF               |
      | method              | POST                   |
      | path                | /import/schemas        |
      | status_code         | 422                    |
      | before_hash         |                        |
      | after_hash          |                        |
      | context             |                        |
      | transport_security  | tls                    |
      | reason              | schema_id_conflict     |
      | error               |                        |
      | request_body        |                        |
      | metadata            |                        |
      | timestamp           | *                      |
      | duration_ms         | *                      |
      | request_id          | *                      |
      | source_ip           | *                      |
      | user_agent          | *                      |

  # ---------------------------------------------------------------------------
  # IDs after import continue above highest imported
  # ---------------------------------------------------------------------------

  Scenario: Schema IDs after import start above the highest imported ID
    Given the global mode is "IMPORT"
    When I import a schema with ID 50000 under subject "import-high-id" version 1:
      """
      {"type":"record","name":"HighId","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200
    And the import should have 1 imported and 0 errors
    When I set the global mode to "READWRITE"
    # Register a new schema — should get an ID > 50000
    When I register a schema under subject "after-import":
      """
      {"type":"record","name":"AfterImport","fields":[{"name":"f","type":"string"}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "new_id"
    And the stored "new_id" should be greater than 50000
    And the audit log should contain an event:
      | event_type          | schema_import    |
      | outcome             | success          |
      | actor_id            |                  |
      | actor_type          | anonymous        |
      | auth_method         |                  |
      | role                |                  |
      | target_type         | subject          |
      | target_id           | import-high-id   |
      | schema_id           | *                |
      | version             |                  |
      | schema_type         | AVRO             |
      | method              | POST             |
      | path                | /import/schemas  |
      | status_code         | 200              |
      | before_hash         |                  |
      | after_hash          | sha256:*         |
      | context             |                  |
      | transport_security  | tls              |
      | reason              |                  |
      | error               |                  |
      | request_body        |                  |
      | metadata            |                  |
      | timestamp           | *                |
      | duration_ms         | *                |
      | request_id          | *                |
      | source_ip           | *                |
      | user_agent          | *                |
