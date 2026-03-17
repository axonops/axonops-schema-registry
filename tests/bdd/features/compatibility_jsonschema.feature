@functional @compatibility
Feature: JSON Schema Compatibility
  Exhaustive JSON Schema compatibility tests across all seven modes

  # ============================================================================
  # BACKWARD mode (8 scenarios)
  # New schema (reader) must be able to read data written by old schema (writer)
  # ============================================================================

  Scenario: BACKWARD - add optional property to open content model is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-back-1":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-back-1                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-back-1/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - add required property is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-back-2":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-back-2                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-back-2/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - remove property from open content model is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-back-3":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-back-3                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-back-3/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - make optional property required is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-4" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-back-4":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-back-4                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-back-4/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - widen type union is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-5" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-back-5":
      """
      {"type":"object","properties":{"value":{"type":["string","null"]}},"required":["value"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-back-5                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-back-5/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - narrow type union is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-6" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":["string","null"]}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-back-6":
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-back-6                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-back-6/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - loosen array minItems is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-7" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    When I register a "JSON" schema under subject "json-back-7":
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-back-7                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-back-7/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - tighten array minItems is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-8" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    When I register a "JSON" schema under subject "json-back-8":
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-back-8                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-back-8/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ============================================================================
  # BACKWARD_TRANSITIVE mode (5 scenarios)
  # New schema must be compatible with ALL previous versions
  # ============================================================================

  Scenario: BACKWARD_TRANSITIVE - 3-version chain all compatible (closed model)
    # With additionalProperties:false (closed content model), adding optional
    # properties is backward-compatible because the old writer couldn't have
    # produced data with the new property name.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    And subject "json-bt-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-bt-1":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-bt-1                                |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-bt-1/versions             |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD_TRANSITIVE - v3 adds required property absent in v1
    # Register v1 and v2 under NONE to avoid open-model incompatibility on v2.
    # Then switch to BACKWARD_TRANSITIVE for v3 which adds required "age" — fails vs v1.
    Given the global compatibility level is "NONE"
    And subject "json-bt-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    And subject "json-bt-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    And the global compatibility level is "BACKWARD_TRANSITIVE"
    When I register a "JSON" schema under subject "json-bt-2":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-bt-2                                |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-bt-2/versions             |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD_TRANSITIVE - loosen minItems chain is compatible
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-3" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":10}
      """
    And subject "json-bt-3" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    When I register a "JSON" schema under subject "json-bt-3":
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-bt-3                                |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-bt-3/versions             |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD_TRANSITIVE - v3 tightens minItems vs v1
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    And subject "json-bt-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    When I register a "JSON" schema under subject "json-bt-4":
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-bt-4                                |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-bt-4/versions             |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD_TRANSITIVE - optional property additions chain is compatible (closed model)
    # With closed content model, adding optional properties is backward-compatible.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-5" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"}},"required":["id"],"additionalProperties":false}
      """
    And subject "json-bt-5" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-bt-5":
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"},"email":{"type":"string"}},"required":["id"],"additionalProperties":false}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-bt-5                                |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-bt-5/versions             |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ============================================================================
  # FORWARD mode (8 scenarios)
  # Old schema (reader) must be able to read data written by new schema (writer)
  # ============================================================================

  Scenario: FORWARD - remove optional property with open content model is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-fwd-1":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-fwd-1                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-fwd-1/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD - remove required property from new is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    When I register a "JSON" schema under subject "json-fwd-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-fwd-2                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-fwd-2/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD - make required property optional in new is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    When I register a "JSON" schema under subject "json-fwd-3":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-fwd-3                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-fwd-3/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD - tighten array minItems in new is compatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    When I register a "JSON" schema under subject "json-fwd-4":
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-fwd-4                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-fwd-4/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD - loosen array minItems in new is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-5" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    When I register a "JSON" schema under subject "json-fwd-5":
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-fwd-5                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-fwd-5/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD - add enum value in new is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-6" has "JSON" schema:
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive"]}},"required":["status"]}
      """
    When I register a "JSON" schema under subject "json-fwd-6":
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive","pending"]}},"required":["status"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-fwd-6                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-fwd-6/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD - remove enum value in new is compatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-7" has "JSON" schema:
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive","pending"]}},"required":["status"]}
      """
    When I register a "JSON" schema under subject "json-fwd-7":
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive"]}},"required":["status"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-fwd-7                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-fwd-7/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD - identical schema is compatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-8" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-fwd-8":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-fwd-8                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-fwd-8/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ============================================================================
  # FORWARD_TRANSITIVE mode (4 scenarios)
  # ALL previous schemas must be able to read data from new schema
  # ============================================================================

  Scenario: FORWARD_TRANSITIVE - 3-version compatible chain (closed model)
    # With closed content model, properties in reader(old) not in writer(new) are
    # compatible because the old writer couldn't produce those properties.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    And subject "json-ft-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-ft-1":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-ft-1                                |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-ft-1/versions             |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD_TRANSITIVE - removing required property in chain fails (closed model)
    # Removing a required property makes old reader unable to find expected data.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["name","email"],"additionalProperties":false}
      """
    And subject "json-ft-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-ft-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-ft-2                                |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-ft-2/versions             |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD_TRANSITIVE - making required optional in chain fails
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    And subject "json-ft-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    When I register a "JSON" schema under subject "json-ft-3":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-ft-3                                |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-ft-3/versions             |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FORWARD_TRANSITIVE - tighten constraints chain is compatible
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    And subject "json-ft-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":3}
      """
    When I register a "JSON" schema under subject "json-ft-4":
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-ft-4                                |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-ft-4/versions             |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ============================================================================
  # FULL mode (7 scenarios)
  # Must be both backward AND forward compatible
  # ============================================================================

  Scenario: FULL - add optional property is incompatible (fails forward check)
    Given the global compatibility level is "FULL"
    And subject "json-full-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full-1":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-full-1                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-full-1/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FULL - add required property is incompatible
    Given the global compatibility level is "FULL"
    And subject "json-full-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full-2":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-full-2                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-full-2/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FULL - remove property is incompatible
    Given the global compatibility level is "FULL"
    And subject "json-full-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full-3":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-full-3                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-full-3/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FULL - make required to optional is incompatible (fails forward)
    Given the global compatibility level is "FULL"
    And subject "json-full-3b" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    When I register a "JSON" schema under subject "json-full-3b":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-full-3b                             |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-full-3b/versions          |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FULL - type change in both directions is incompatible
    Given the global compatibility level is "FULL"
    And subject "json-full-4" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-full-4":
      """
      {"type":"object","properties":{"value":{"type":"integer"}},"required":["value"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-full-4                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-full-4/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FULL - additionalProperties true to false is incompatible
    Given the global compatibility level is "FULL"
    And subject "json-full-5" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    When I register a "JSON" schema under subject "json-full-5":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-full-5                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-full-5/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FULL - identical schema is compatible
    Given the global compatibility level is "FULL"
    And subject "json-full-6" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full-6":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-full-6                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-full-6/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ============================================================================
  # FULL_TRANSITIVE mode (3 scenarios)
  # Must be both backward AND forward compatible with ALL previous versions
  # ============================================================================

  Scenario: FULL_TRANSITIVE - safe 3-version evolution is compatible
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "json-flt-1" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"]}
      """
    And subject "json-flt-1" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-flt-1":
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-flt-1                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-flt-1/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FULL_TRANSITIVE - removing property fails across versions
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "json-flt-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    And subject "json-flt-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-flt-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-flt-2                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-flt-2/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: FULL_TRANSITIVE - identical schemas across chain is compatible
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "json-flt-3" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"value":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-flt-3" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"value":{"type":"integer"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-flt-3":
      """
      {"type":"object","properties":{"id":{"type":"string"},"value":{"type":"integer"}},"required":["id"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-flt-3                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-flt-3/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ============================================================================
  # NONE mode (2 scenarios)
  # No compatibility checks, any change allowed
  # ============================================================================

  Scenario: NONE - complete restructure is allowed
    Given the global compatibility level is "NONE"
    And subject "json-none-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    When I register a "JSON" schema under subject "json-none-1":
      """
      {"type":"array","items":{"type":"number"},"minItems":1}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-none-1                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-none-1/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: NONE - root type change is allowed
    Given the global compatibility level is "NONE"
    And subject "json-none-2" has "JSON" schema:
      """
      {"type":"string","enum":["a","b","c"]}
      """
    When I register a "JSON" schema under subject "json-none-2":
      """
      {"type":"integer","minimum":0,"maximum":100}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-none-2                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-none-2/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ============================================================================
  # Edge Cases (12 scenarios)
  # ============================================================================

  Scenario: Edge - additionalProperties true to false is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    When I register a "JSON" schema under subject "json-edge-1":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-1                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-1/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - additionalProperties false to true is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-edge-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-2                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-2/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - enum value addition is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-3" has "JSON" schema:
      """
      {"type":"object","properties":{"color":{"type":"string","enum":["red","green"]}},"required":["color"]}
      """
    When I register a "JSON" schema under subject "json-edge-3":
      """
      {"type":"object","properties":{"color":{"type":"string","enum":["red","green","blue"]}},"required":["color"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-3                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-3/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - enum value removal is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-4" has "JSON" schema:
      """
      {"type":"object","properties":{"color":{"type":"string","enum":["red","green","blue"]}},"required":["color"]}
      """
    When I register a "JSON" schema under subject "json-edge-4":
      """
      {"type":"object","properties":{"color":{"type":"string","enum":["red","green"]}},"required":["color"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-4                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-4/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - nested object property removal from open model is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-5" has "JSON" schema:
      """
      {"type":"object","properties":{"user":{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}}}},"required":["user"]}
      """
    When I register a "JSON" schema under subject "json-edge-5":
      """
      {"type":"object","properties":{"user":{"type":"object","properties":{"name":{"type":"string"}}}},"required":["user"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-5                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-5/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - array items type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-6" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"}}
      """
    When I register a "JSON" schema under subject "json-edge-6":
      """
      {"type":"array","items":{"type":"integer"}}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-6                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-6/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - minItems increase is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-7" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    When I register a "JSON" schema under subject "json-edge-7":
      """
      {"type":"array","items":{"type":"string"},"minItems":10}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-7                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-7/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - maxItems decrease is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-8" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"maxItems":10}
      """
    When I register a "JSON" schema under subject "json-edge-8":
      """
      {"type":"array","items":{"type":"string"},"maxItems":5}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-8                              |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-8/versions           |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - type array expansion is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-9" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-edge-9":
      """
      {"type":"object","properties":{"value":{"type":["string","null"]}},"required":["value"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-9                              |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-9/versions           |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - maxItems increase is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-10" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"maxItems":5}
      """
    When I register a "JSON" schema under subject "json-edge-10":
      """
      {"type":"array","items":{"type":"string"},"maxItems":10}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-10                             |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-10/versions          |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - root type change object to array is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-11" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}}}
      """
    When I register a "JSON" schema under subject "json-edge-11":
      """
      {"type":"array","items":{"type":"string"}}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-11                             |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-11/versions          |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Edge - make required property optional is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-12" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    When I register a "JSON" schema under subject "json-edge-12":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-edge-12                             |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-edge-12/versions          |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ============================================================================
  # Error Validation (5 scenarios)
  # ============================================================================

  Scenario: Error - 409 response has error_code field
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-err-1":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the response status should be 409
    And the response should have error code 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-err-1                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-err-1/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Error - check endpoint returns is_compatible false for incompatible schema
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I check compatibility of "JSON" schema against subject "json-err-2":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the compatibility check should be incompatible

  Scenario: Error - per-subject NONE override bypasses global BACKWARD
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-3" has compatibility level "NONE"
    And subject "json-err-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-err-3":
      """
      {"type":"array","items":{"type":"integer"}}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-err-3                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-err-3/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: Error - check endpoint returns is_compatible false for open model property addition
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-4" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I check compatibility of "JSON" schema against subject "json-err-4":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the compatibility check should be incompatible

  Scenario: Error - delete per-subject config falls back to global
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-5" has compatibility level "NONE"
    And subject "json-err-5" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-err-5":
      """
      {"type":"array","items":{"type":"integer"}}
      """
    Then the response status should be 200
    When I delete the config for subject "json-err-5"
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-err-5":
      """
      {"type":"object","properties":{"x":{"type":"number"}}}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-err-5                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-err-5/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # --- Gap-filling: JSON Schema-specific compatibility rules ---

  Scenario: BACKWARD - additionalProperties false to true is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-gap-1":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-gap-1                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-gap-1/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - additionalProperties true to false is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    When I register a "JSON" schema under subject "json-gap-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-gap-2                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-gap-2/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - nested property removal from open model is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-3" has "JSON" schema:
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}}}}}
      """
    When I register a "JSON" schema under subject "json-gap-3":
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"}}}}}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-gap-3                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-gap-3/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - nested property addition to open model is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-4" has "JSON" schema:
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"}}}}}
      """
    When I register a "JSON" schema under subject "json-gap-4":
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"},"zip":{"type":"string"}}}}}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-gap-4                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-gap-4/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - type widening string to string-or-null is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-5" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}}}
      """
    When I register a "JSON" schema under subject "json-gap-5":
      """
      {"type":"object","properties":{"name":{"type":["string","null"]}}}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-gap-5                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-gap-5/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - type narrowing string-or-null to string is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-6" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":["string","null"]}}}
      """
    When I register a "JSON" schema under subject "json-gap-6":
      """
      {"type":"object","properties":{"name":{"type":"string"}}}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-gap-6                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-gap-6/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - array items schema removal is compatible (relaxation)
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-7" has "JSON" schema:
      """
      {"type":"object","properties":{"tags":{"type":"array","items":{"type":"string"}}}}
      """
    When I register a "JSON" schema under subject "json-gap-7":
      """
      {"type":"object","properties":{"tags":{"type":"array"}}}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-gap-7                               |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          | sha256:*                                 |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-gap-7/versions            |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  Scenario: BACKWARD - multiple simultaneous incompatible changes detected
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-8" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-gap-8":
      """
      {"type":"object","properties":{"name":{"type":"integer"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the response status should be 409
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | failure                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-gap-8                               |
      | schema_id            |                                          |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           |                                          |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-gap-8/versions            |
      | status_code          | 409                                      |
      | reason               | incompatible                             |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ==========================================================================
  # JSON Schema compatibility with external $ref — the compatibility checker
  # must resolve external $ref references when checking compatibility of
  # JSON Schemas that reference other schemas via the references mechanism.
  # ==========================================================================

  Scenario: JSON Schema BACKWARD compatibility with external reference (closed model)
    Given the global compatibility level is "BACKWARD"
    And subject "json-ref-address" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}},"required":["street","city"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-ref-person" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"address\":{\"$ref\":\"address.json\"}},\"required\":[\"name\"],\"additionalProperties\":false}",
        "references": [
          {"name": "address.json", "subject": "json-ref-address", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-ref-person                          |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-ref-person/versions       |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
    When I check compatibility of "JSON" schema with reference "address.json" from subject "json-ref-address" version 1 against subject "json-ref-person":
      """
      {"type":"object","properties":{"name":{"type":"string"},"address":{"$ref":"address.json"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema compatibility check resolves external $ref correctly
    Given the global compatibility level is "BACKWARD"
    And subject "json-compat-addr" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"}},"required":["street"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-compat-person" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"addr\":{\"$ref\":\"addr.json\"}},\"required\":[\"name\"],\"additionalProperties\":false}",
        "references": [
          {"name": "addr.json", "subject": "json-compat-addr", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-compat-person                       |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-compat-person/versions    |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
    When I check compatibility of "JSON" schema with reference "addr.json" from subject "json-compat-addr" version 1 against subject "json-compat-person":
      """
      {"type":"object","properties":{"name":{"type":"string"},"addr":{"$ref":"addr.json"}},"required":["name"],"additionalProperties":false}
      """
    Then the compatibility check should be compatible

  Scenario: JSON Schema BACKWARD incompatible change with external reference
    Given the global compatibility level is "BACKWARD"
    And subject "json-incompat-addr" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"}},"required":["street"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-incompat-person" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"addr\":{\"$ref\":\"addr.json\"}},\"required\":[\"name\"],\"additionalProperties\":false}",
        "references": [
          {"name": "addr.json", "subject": "json-incompat-addr", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | json-incompat-person                     |
      | schema_id            | *                                        |
      | version              | *                                        |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-incompat-person/versions  |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |
    When I check compatibility of "JSON" schema with reference "addr.json" from subject "json-incompat-addr" version 1 against subject "json-incompat-person":
      """
      {"type":"object","properties":{"name":{"type":"string"},"addr":{"$ref":"addr.json"}},"required":["name","addr"],"additionalProperties":false}
      """
    Then the compatibility check should be incompatible
