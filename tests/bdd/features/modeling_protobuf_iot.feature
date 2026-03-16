@schema-modeling @protobuf @domain
Feature: Protobuf IoT Domain Modeling
  Real-world IoT domain schemas exercising multi-version evolution,
  cross-subject references with proto imports, oneof evolution,
  package changes, reserved fields, and well-known types.

  # ==========================================================================
  # 1. SENSOR READING EVOLVES 5 VERSIONS
  # ==========================================================================

  Scenario: SensorReading evolves 5 versions under BACKWARD_TRANSITIVE
    Given subject "iot-sensor" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "iot-sensor" has "PROTOBUF" schema:
      """
syntax = "proto3";
package iot;

message SensorReading {
  string device_id = 1;
  double value = 2;
}
      """
    When I register a "PROTOBUF" schema under subject "iot-sensor":
      """
syntax = "proto3";
package iot;

message SensorReading {
  string device_id = 1;
  double value = 2;
  int64 timestamp = 3;
}
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "iot-sensor":
      """
syntax = "proto3";
package iot;

message SensorReading {
  string device_id = 1;
  double value = 2;
  int64 timestamp = 3;
  string unit = 4;
}
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "iot-sensor":
      """
syntax = "proto3";
package iot;

message SensorReading {
  string device_id = 1;
  double value = 2;
  int64 timestamp = 3;
  string unit = 4;
  Quality quality = 5;
  enum Quality {
    UNKNOWN = 0;
    GOOD = 1;
    DEGRADED = 2;
    BAD = 3;
  }
}
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "iot-sensor":
      """
syntax = "proto3";
package iot;

message SensorReading {
  string device_id = 1;
  double value = 2;
  int64 timestamp = 3;
  string unit = 4;
  Quality quality = 5;
  map<string, string> metadata = 6;
  enum Quality {
    UNKNOWN = 0;
    GOOD = 1;
    DEGRADED = 2;
    BAD = 3;
  }
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | iot-sensor                                   |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | PROTOBUF                                     |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/iot-sensor/versions                |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 2. DEVICE CONFIG REFERENCES SHARED TYPES
  # ==========================================================================

  Scenario: DeviceConfig references shared common types
    Given subject "iot-common" has "PROTOBUF" schema:
      """
syntax = "proto3";
package iot.common;

message Address {
  string street = 1;
  string city = 2;
  string country = 3;
}
      """
    When I register a "PROTOBUF" schema under subject "iot-device" with references:
      """
      {
        "schemaType": "PROTOBUF",
        "schema": "syntax = \"proto3\";\npackage iot;\n\nimport \"iot/common/address.proto\";\n\nmessage DeviceConfig {\n  string device_id = 1;\n  string name = 2;\n  iot.common.Address location = 3;\n}",
        "references": [
          {"name":"iot/common/address.proto","subject":"iot-common","version":1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | iot-device                                   |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | PROTOBUF                                     |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/iot-device/versions                |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 3. ALERT WITH ONEOF PAYLOAD — ADD VARIANT
  # ==========================================================================

  Scenario: Alert with oneof payload — adding variant is compatible
    Given subject "iot-alert" has compatibility level "BACKWARD"
    And subject "iot-alert" has "PROTOBUF" schema:
      """
syntax = "proto3";
package iot;

message Alert {
  string device_id = 1;
  oneof payload {
    string text = 2;
  }
}
      """
    When I register a "PROTOBUF" schema under subject "iot-alert":
      """
syntax = "proto3";
package iot;

message Alert {
  string device_id = 1;
  oneof payload {
    string text = 2;
    int32 code = 3;
  }
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | iot-alert                                    |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | PROTOBUF                                     |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/iot-alert/versions                 |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 4. PACKAGE RENAME BREAKS COMPATIBILITY
  # ==========================================================================

  Scenario: Package rename breaks compatibility
    Given subject "iot-pkg-rename" has compatibility level "BACKWARD"
    And subject "iot-pkg-rename" has "PROTOBUF" schema:
      """
syntax = "proto3";
package iot.v1;

message Telemetry {
  string device_id = 1;
  double value = 2;
}
      """
    When I register a "PROTOBUF" schema under subject "iot-pkg-rename":
      """
syntax = "proto3";
package iot.v2;

message Telemetry {
  string device_id = 1;
  double value = 2;
}
      """
    Then the response status should be 409

  # ==========================================================================
  # 5. RESERVED FIELD HANDLING
  # ==========================================================================

  Scenario: Reserved field handling allows safe field removal and addition
    Given subject "iot-reserved" has compatibility level "BACKWARD"
    And subject "iot-reserved" has "PROTOBUF" schema:
      """
syntax = "proto3";
package iot;

message Config {
  string name = 1;
  string old_setting = 2;
  int32 interval = 3;
}
      """
    When I register a "PROTOBUF" schema under subject "iot-reserved":
      """
syntax = "proto3";
package iot;

message Config {
  string name = 1;
  reserved 2;
  reserved "old_setting";
  int32 interval = 3;
  string new_setting = 4;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | iot-reserved                                 |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | PROTOBUF                                     |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/iot-reserved/versions              |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 6. NESTED MESSAGE EVOLUTION
  # ==========================================================================

  Scenario: Nested message evolution under BACKWARD_TRANSITIVE
    Given subject "iot-nested-evo" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "iot-nested-evo" has "PROTOBUF" schema:
      """
syntax = "proto3";
package iot;

message Gateway {
  string id = 1;
  Status status = 2;
  message Status {
    bool online = 1;
  }
}
      """
    When I register a "PROTOBUF" schema under subject "iot-nested-evo":
      """
syntax = "proto3";
package iot;

message Gateway {
  string id = 1;
  Status status = 2;
  message Status {
    bool online = 1;
    int64 last_seen = 2;
  }
}
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "iot-nested-evo":
      """
syntax = "proto3";
package iot;

message Gateway {
  string id = 1;
  Status status = 2;
  message Status {
    bool online = 1;
    int64 last_seen = 2;
    string firmware = 3;
  }
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | iot-nested-evo                               |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | PROTOBUF                                     |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/iot-nested-evo/versions            |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 7. ENUM GROWTH ACROSS VERSIONS
  # ==========================================================================

  Scenario: Enum growth across versions under BACKWARD_TRANSITIVE
    Given subject "iot-enum-grow" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "iot-enum-grow" has "PROTOBUF" schema:
      """
syntax = "proto3";
package iot;

message Device {
  string id = 1;
  State state = 2;
  enum State {
    ACTIVE = 0;
    INACTIVE = 1;
  }
}
      """
    When I register a "PROTOBUF" schema under subject "iot-enum-grow":
      """
syntax = "proto3";
package iot;

message Device {
  string id = 1;
  State state = 2;
  enum State {
    ACTIVE = 0;
    INACTIVE = 1;
    DEPRECATED = 2;
  }
}
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "iot-enum-grow":
      """
syntax = "proto3";
package iot;

message Device {
  string id = 1;
  State state = 2;
  enum State {
    ACTIVE = 0;
    INACTIVE = 1;
    DEPRECATED = 2;
    MAINTENANCE = 3;
  }
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | iot-enum-grow                                |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | PROTOBUF                                     |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/iot-enum-grow/versions             |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 8. PROTO WITH WELL-KNOWN TYPE IMPORTS
  # ==========================================================================

  Scenario: Proto with well-known type imports registers successfully
    When I register a "PROTOBUF" schema under subject "iot-wkt":
      """
syntax = "proto3";
package iot;

import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/wrappers.proto";

message Measurement {
  string sensor_id = 1;
  google.protobuf.Timestamp recorded_at = 2;
  google.protobuf.Duration sampling_interval = 3;
  google.protobuf.DoubleValue value = 4;
  google.protobuf.StringValue unit = 5;
}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | iot-wkt                                      |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | PROTOBUF                                     |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/iot-wkt/versions                   |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
