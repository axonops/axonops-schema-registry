@schema-modeling @protobuf @compatibility
Feature: Protobuf Advanced Compatibility
  Tests grounded in the protobuf wire-type compatibility groups from the
  compatibility checker (areKindsWireCompatible). Exercises varint, zigzag,
  32-bit, 64-bit, and length-delimited groups, plus oneof handling,
  package changes, service changes, and multi-version transitive chains.

  # ==========================================================================
  # 1-3. VARINT WIRE-TYPE GROUP
  # ==========================================================================

  Scenario: Varint group — int32 to uint32 is compatible
    Given subject "proto-compat-varint-u32" has compatibility level "BACKWARD"
    And subject "proto-compat-varint-u32" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  int32 value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-varint-u32":
      """
syntax = "proto3";
package test.compat;

message Msg {
  uint32 value = 1;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-varint-u32                         |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-varint-u32/versions     |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  Scenario: Varint group — int32 to int64 is compatible
    Given subject "proto-compat-varint-i64" has compatibility level "BACKWARD"
    And subject "proto-compat-varint-i64" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  int32 value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-varint-i64":
      """
syntax = "proto3";
package test.compat;

message Msg {
  int64 value = 1;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-varint-i64                         |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-varint-i64/versions     |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  Scenario: Varint group — int32 to bool is compatible
    Given subject "proto-compat-varint-bool" has compatibility level "BACKWARD"
    And subject "proto-compat-varint-bool" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  int32 value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-varint-bool":
      """
syntax = "proto3";
package test.compat;

message Msg {
  bool value = 1;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                 |
      | outcome            | success                                         |
      | actor_id           |                                                 |
      | actor_type         | anonymous                                       |
      | auth_method        |                                                 |
      | role               |                                                 |
      | target_type        | subject                                         |
      | target_id          | proto-compat-varint-bool                         |
      | schema_id          | *                                               |
      | version            |                                                 |
      | schema_type        | PROTOBUF                                        |
      | before_hash        |                                                 |
      | after_hash         | sha256:*                                        |
      | context            | .                                               |
      | transport_security | tls                                             |
      | source_ip          | *                                               |
      | user_agent         | *                                               |
      | method             | POST                                            |
      | path               | /subjects/proto-compat-varint-bool/versions      |
      | status_code        | 200                                             |
      | reason             |                                                 |
      | error              |                                                 |
      | request_body       |                                                 |
      | metadata           |                                                 |
      | timestamp          | *                                               |
      | duration_ms        | *                                               |
      | request_id         | *                                               |

  # ==========================================================================
  # 4. ENUM AND INT32 COMPATIBLE (VARINT GROUP)
  # ==========================================================================

  Scenario: Enum and int32 are wire-compatible under varint group
    Given subject "proto-compat-enum-int" has compatibility level "BACKWARD"
    And subject "proto-compat-enum-int" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  int32 value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-enum-int":
      """
syntax = "proto3";
package test.compat;

message Msg {
  Status value = 1;
  enum Status {
    UNKNOWN = 0;
    ACTIVE = 1;
  }
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-enum-int                           |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-enum-int/versions       |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  # ==========================================================================
  # 5. ZIGZAG GROUP
  # ==========================================================================

  Scenario: ZigZag group — sint32 to sint64 is compatible
    Given subject "proto-compat-zigzag" has compatibility level "BACKWARD"
    And subject "proto-compat-zigzag" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  sint32 value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-zigzag":
      """
syntax = "proto3";
package test.compat;

message Msg {
  sint64 value = 1;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-zigzag                             |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-zigzag/versions         |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  # ==========================================================================
  # 6. CROSS-GROUP INCOMPATIBLE
  # ==========================================================================

  Scenario: Cross-group sint32 to int32 is incompatible
    Given subject "proto-compat-cross-group" has compatibility level "BACKWARD"
    And subject "proto-compat-cross-group" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  sint32 value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-cross-group":
      """
syntax = "proto3";
package test.compat;

message Msg {
  int32 value = 1;
}
      """
    Then the response status should be 409

  # ==========================================================================
  # 7. 32-BIT GROUP
  # ==========================================================================

  Scenario: 32-bit group — fixed32 to sfixed32 is compatible
    Given subject "proto-compat-32bit" has compatibility level "BACKWARD"
    And subject "proto-compat-32bit" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  fixed32 value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-32bit":
      """
syntax = "proto3";
package test.compat;

message Msg {
  sfixed32 value = 1;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-32bit                              |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-32bit/versions          |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  # ==========================================================================
  # 8. 64-BIT GROUP
  # ==========================================================================

  Scenario: 64-bit group — fixed64 to sfixed64 is compatible
    Given subject "proto-compat-64bit" has compatibility level "BACKWARD"
    And subject "proto-compat-64bit" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  fixed64 value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-64bit":
      """
syntax = "proto3";
package test.compat;

message Msg {
  sfixed64 value = 1;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-64bit                              |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-64bit/versions          |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  # ==========================================================================
  # 9. LENGTH-DELIMITED GROUP
  # ==========================================================================

  Scenario: Length-delimited group — string to bytes is compatible
    Given subject "proto-compat-len-delim" has compatibility level "BACKWARD"
    And subject "proto-compat-len-delim" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  string value = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-len-delim":
      """
syntax = "proto3";
package test.compat;

message Msg {
  bytes value = 1;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-len-delim                          |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-len-delim/versions      |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  # ==========================================================================
  # 10. ADDING FIELD TO ONEOF IS COMPATIBLE
  # ==========================================================================

  Scenario: Adding field to oneof is compatible
    Given subject "proto-compat-oneof-add" has compatibility level "BACKWARD"
    And subject "proto-compat-oneof-add" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  oneof choice {
    string name = 1;
  }
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-oneof-add":
      """
syntax = "proto3";
package test.compat;

message Msg {
  oneof choice {
    string name = 1;
    int32 id = 2;
  }
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-oneof-add                          |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-oneof-add/versions      |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  # ==========================================================================
  # 11. MOVING FIELD OUT OF ONEOF IS INCOMPATIBLE
  # ==========================================================================

  Scenario: Moving field out of oneof is incompatible
    Given subject "proto-compat-oneof-move" has compatibility level "BACKWARD"
    And subject "proto-compat-oneof-move" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Msg {
  oneof choice {
    string name = 1;
    int32 id = 2;
  }
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-oneof-move":
      """
syntax = "proto3";
package test.compat;

message Msg {
  oneof choice {
    string name = 1;
  }
  int32 id = 2;
}
      """
    Then the response status should be 409

  # ==========================================================================
  # 12. PACKAGE CHANGE IS INCOMPATIBLE
  # ==========================================================================

  Scenario: Package change is incompatible
    Given subject "proto-compat-pkg-change" has compatibility level "BACKWARD"
    And subject "proto-compat-pkg-change" has "PROTOBUF" schema:
      """
syntax = "proto3";
package iot.v1;

message Reading {
  string device_id = 1;
  double value = 2;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-pkg-change":
      """
syntax = "proto3";
package iot.v2;

message Reading {
  string device_id = 1;
  double value = 2;
}
      """
    Then the response status should be 409

  # ==========================================================================
  # 13. SERVICE CHANGES ARE IGNORED IN COMPAT
  # ==========================================================================

  Scenario: Service changes are ignored in compatibility checks
    Given subject "proto-compat-service" has compatibility level "BACKWARD"
    And subject "proto-compat-service" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Request {
  string query = 1;
}

message Response {
  string result = 1;
}

service Search {
  rpc Find(Request) returns (Response);
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-service":
      """
syntax = "proto3";
package test.compat;

message Request {
  string query = 1;
}

message Response {
  string result = 1;
  int32 count = 2;
}

service Search {
  rpc Find(Request) returns (Response);
  rpc FindAll(Request) returns (Response);
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-service                            |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-service/versions        |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |

  # ==========================================================================
  # 14. 3-VERSION FIELD ADDITION CHAIN UNDER BACKWARD_TRANSITIVE
  # ==========================================================================

  Scenario: Three-version field addition chain under BACKWARD_TRANSITIVE
    Given subject "proto-compat-3v-chain" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "proto-compat-3v-chain" has "PROTOBUF" schema:
      """
syntax = "proto3";
package test.compat;

message Event {
  string name = 1;
}
      """
    When I register a "PROTOBUF" schema under subject "proto-compat-3v-chain":
      """
syntax = "proto3";
package test.compat;

message Event {
  string name = 1;
  string email = 2;
}
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "proto-compat-3v-chain":
      """
syntax = "proto3";
package test.compat;

message Event {
  string name = 1;
  string email = 2;
  string phone = 3;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type         | schema_register                                |
      | outcome            | success                                        |
      | actor_id           |                                                |
      | actor_type         | anonymous                                      |
      | auth_method        |                                                |
      | role               |                                                |
      | target_type        | subject                                        |
      | target_id          | proto-compat-3v-chain                           |
      | schema_id          | *                                              |
      | version            |                                                |
      | schema_type        | PROTOBUF                                       |
      | before_hash        |                                                |
      | after_hash         | sha256:*                                       |
      | context            | .                                              |
      | transport_security | tls                                            |
      | source_ip          | *                                              |
      | user_agent         | *                                              |
      | method             | POST                                           |
      | path               | /subjects/proto-compat-3v-chain/versions       |
      | status_code        | 200                                            |
      | reason             |                                                |
      | error              |                                                |
      | request_body       |                                                |
      | metadata           |                                                |
      | timestamp          | *                                              |
      | duration_ms        | *                                              |
      | request_id         | *                                              |
