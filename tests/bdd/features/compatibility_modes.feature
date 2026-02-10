@functional @compatibility
Feature: Compatibility Modes
  As an operator, I want all 7 compatibility modes to work correctly across all schema types

  # --- BACKWARD_TRANSITIVE with Avro ---

  Scenario: BACKWARD_TRANSITIVE rejects schema incompatible with older version (Avro)
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-bt" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"}]}
      """
    And subject "avro-bt" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-bt":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"int","default":0}]}
      """
    Then the response status should be 409

  Scenario: BACKWARD allows schema compatible with only latest (Avro)
    Given the global compatibility level is "BACKWARD"
    And subject "avro-b" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"}]}
      """
    And subject "avro-b" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-b":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"int","default":0}]}
      """
    Then the response status should be 200

  # --- FORWARD and FORWARD_TRANSITIVE with Avro ---

  Scenario: FORWARD compatible schema accepted (Avro)
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-fwd":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: FORWARD incompatible schema rejected (Avro)
    Given the global compatibility level is "FORWARD"
    And subject "avro-fwd-fail" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":"string"}]}
      """
    When I register a schema under subject "avro-fwd-fail":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE rejects schema incompatible with older version (Avro)
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "avro-ft" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"}]}
      """
    And subject "avro-ft" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-ft":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"code","type":"int","default":0},{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the response status should be 409

  # --- FULL and FULL_TRANSITIVE with Avro ---

  Scenario: FULL compatible change accepted (Avro)
    Given the global compatibility level is "FULL"
    And subject "avro-full" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-full":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the response status should be 200

  Scenario: FULL incompatible change rejected (Avro)
    Given the global compatibility level is "FULL"
    And subject "avro-full-fail" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "avro-full-fail":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"age","type":"int"}]}
      """
    Then the response status should be 409

  Scenario: FULL_TRANSITIVE rejects schema incompatible with any version (Avro)
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "avro-flt" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And subject "avro-flt" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    When I register a schema under subject "avro-flt":
      """
      {"type":"record","name":"User","fields":[{"name":"email","type":["null","string"],"default":null}]}
      """
    Then the response status should be 409

  # --- Compatibility check endpoint (without registering) ---

  Scenario: Check compatibility endpoint with BACKWARD_TRANSITIVE (Avro)
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "avro-check-bt" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    And subject "avro-check-bt" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null}]}
      """
    When I check compatibility of schema against subject "avro-check-bt":
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"},{"name":"email","type":["null","string"],"default":null},{"name":"age","type":["null","int"],"default":null}]}
      """
    Then the compatibility check should be compatible

  # --- Protobuf compatibility modes ---

  Scenario: BACKWARD compatible Protobuf schema accepted
    Given the global compatibility level is "BACKWARD"
    And subject "proto-compat" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        string name = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-compat":
      """
      syntax = "proto3";
      message User {
        string name = 1;
        optional string email = 2;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD incompatible Protobuf schema rejected
    Given the global compatibility level is "BACKWARD"
    And subject "proto-compat-fail" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        string name = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-fail":
      """
      syntax = "proto3";
      message User {
        int32 name = 1;
      }
      """
    Then the response status should be 409

  Scenario: FORWARD compatible Protobuf schema accepted (remove optional field)
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        string name = 1;
        string email = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd":
      """
      syntax = "proto3";
      message User {
        string name = 1;
      }
      """
    Then the response status should be 200

  Scenario: FULL compatible Protobuf schema accepted
    Given the global compatibility level is "FULL"
    And subject "proto-full" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        string name = 1;
        string tag = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-full":
      """
      syntax = "proto3";
      message User {
        string name = 1;
        string tag = 2;
      }
      """
    Then the response status should be 200

  # --- JSON Schema compatibility modes ---

  Scenario: BACKWARD compatible JSON Schema accepted
    Given the global compatibility level is "BACKWARD"
    And subject "json-compat" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-compat":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 200

  Scenario: BACKWARD incompatible JSON Schema rejected
    Given the global compatibility level is "BACKWARD"
    And subject "json-compat-fail" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-compat-fail":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the response status should be 409

  Scenario: FORWARD compatible JSON Schema accepted (remove optional property)
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-fwd":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 200

  Scenario: FULL compatible JSON Schema accepted
    Given the global compatibility level is "FULL"
    And subject "json-full" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 200

  # --- Per-subject override ---

  Scenario: Per-subject compatibility overrides global mode
    Given the global compatibility level is "BACKWARD"
    And subject "override-subj" has compatibility level "NONE"
    And subject "override-subj" has schema:
      """
      {"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}
      """
    When I register a schema under subject "override-subj":
      """
      {"type":"record","name":"User","fields":[{"name":"age","type":"int"}]}
      """
    Then the response status should be 200
