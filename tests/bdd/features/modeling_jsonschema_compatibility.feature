@schema-modeling @json @compatibility
Feature: JSON Schema Advanced Compatibility
  Tests grounded in the JSON Schema compatibility checker implementation
  (isTypePromotion, checkSumTypeCompatibility, checkAdditionalPropertiesCompatibility,
  checkEnumCompatibility, checkStringConstraints, checkNumericConstraints).
  Exercises type widening, composition changes, constraint relaxation/tightening,
  and multi-version transitive chains.

  # ==========================================================================
  # 1. TYPE WIDENING — INTEGER TO NUMBER
  # ==========================================================================

  Scenario: Type widening integer to number is backward compatible
    Given subject "json-compat-int-num" has compatibility level "BACKWARD"
    And subject "json-compat-int-num" has "JSON" schema:
      """
      {"type":"integer"}
      """
    When I register a "JSON" schema under subject "json-compat-int-num":
      """
      {"type":"number"}
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
      | target_id            | json-compat-int-num                          |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-compat-int-num/versions       |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 2. TYPE WIDENING — STRING TO NULLABLE STRING
  # ==========================================================================

  Scenario: Type widening string to nullable string is backward compatible
    Given subject "json-compat-nullable" has compatibility level "BACKWARD"
    And subject "json-compat-nullable" has "JSON" schema:
      """
      {"type":"string"}
      """
    When I register a "JSON" schema under subject "json-compat-nullable":
      """
      {"type":["string","null"]}
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
      | target_id            | json-compat-nullable                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-compat-nullable/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 3. ADDITIONAL PROPERTIES FALSE TO TRUE (RELAXING)
  # ==========================================================================

  Scenario: additionalProperties false to true is backward compatible
    Given subject "json-compat-addl-open" has compatibility level "BACKWARD"
    And subject "json-compat-addl-open" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-compat-addl-open":
      """
      {"type":"object","properties":{"a":{"type":"string"}}}
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
      | target_id            | json-compat-addl-open                        |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-compat-addl-open/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 4. ADDITIONAL PROPERTIES TRUE TO FALSE (TIGHTENING)
  # ==========================================================================

  Scenario: additionalProperties true to false is backward incompatible
    Given subject "json-compat-addl-close" has compatibility level "BACKWARD"
    And subject "json-compat-addl-close" has "JSON" schema:
      """
      {"type":"object","properties":{"a":{"type":"string"}}}
      """
    When I register a "JSON" schema under subject "json-compat-addl-close":
      """
      {"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 409

  # ==========================================================================
  # 5. ONEOF — ADD VARIANT (RELAXING)
  # ==========================================================================

  Scenario: oneOf add variant is backward compatible
    Given subject "json-compat-oneof-add" has compatibility level "BACKWARD"
    And subject "json-compat-oneof-add" has "JSON" schema:
      """
      {"oneOf":[{"type":"string"},{"type":"integer"}]}
      """
    When I register a "JSON" schema under subject "json-compat-oneof-add":
      """
      {"oneOf":[{"type":"string"},{"type":"integer"},{"type":"boolean"}]}
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
      | target_id            | json-compat-oneof-add                        |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-compat-oneof-add/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 6. ONEOF — REMOVE VARIANT (TIGHTENING)
  # ==========================================================================

  Scenario: oneOf remove variant is backward incompatible
    Given subject "json-compat-oneof-rm" has compatibility level "BACKWARD"
    And subject "json-compat-oneof-rm" has "JSON" schema:
      """
      {"oneOf":[{"type":"string"},{"type":"integer"},{"type":"boolean"}]}
      """
    When I register a "JSON" schema under subject "json-compat-oneof-rm":
      """
      {"oneOf":[{"type":"string"},{"type":"integer"}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # 7. ENUM VALUE ADDITION (RELAXING)
  # ==========================================================================

  Scenario: Enum value addition is backward compatible
    Given subject "json-compat-enum-add" has compatibility level "BACKWARD"
    And subject "json-compat-enum-add" has "JSON" schema:
      """
      {"enum":["A","B"]}
      """
    When I register a "JSON" schema under subject "json-compat-enum-add":
      """
      {"enum":["A","B","C"]}
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
      | target_id            | json-compat-enum-add                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-compat-enum-add/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 8. ENUM VALUE REMOVAL (TIGHTENING)
  # ==========================================================================

  Scenario: Enum value removal is backward incompatible
    Given subject "json-compat-enum-rm" has compatibility level "BACKWARD"
    And subject "json-compat-enum-rm" has "JSON" schema:
      """
      {"enum":["A","B","C"]}
      """
    When I register a "JSON" schema under subject "json-compat-enum-rm":
      """
      {"enum":["A","B"]}
      """
    Then the response status should be 409

  # ==========================================================================
  # 9. MAXLENGTH DECREASE (TIGHTENING)
  # ==========================================================================

  Scenario: maxLength decrease is backward incompatible
    Given subject "json-compat-maxlen" has compatibility level "BACKWARD"
    And subject "json-compat-maxlen" has "JSON" schema:
      """
      {"type":"string","maxLength":200}
      """
    When I register a "JSON" schema under subject "json-compat-maxlen":
      """
      {"type":"string","maxLength":100}
      """
    Then the response status should be 409

  # ==========================================================================
  # 10. MINLENGTH INCREASE (TIGHTENING)
  # ==========================================================================

  Scenario: minLength increase is backward incompatible
    Given subject "json-compat-minlen" has compatibility level "BACKWARD"
    And subject "json-compat-minlen" has "JSON" schema:
      """
      {"type":"string","minLength":1}
      """
    When I register a "JSON" schema under subject "json-compat-minlen":
      """
      {"type":"string","minLength":5}
      """
    Then the response status should be 409

  # ==========================================================================
  # 11. ADD REQUIRED PROPERTY (TIGHTENING)
  # ==========================================================================

  Scenario: Adding required property is backward incompatible
    Given subject "json-compat-add-req" has compatibility level "BACKWARD"
    And subject "json-compat-add-req" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}}}
      """
    When I register a "JSON" schema under subject "json-compat-add-req":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the response status should be 409

  # ==========================================================================
  # 12. ADD OPTIONAL PROPERTY TO CLOSED MODEL (RELAXING)
  # ==========================================================================

  Scenario: Adding optional property to closed model is backward compatible
    Given subject "json-compat-closed-add" has compatibility level "BACKWARD"
    And subject "json-compat-closed-add" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-compat-closed-add":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | json-compat-closed-add                         |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-compat-closed-add/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 13. NUMERIC CONSTRAINT RELAXATION CHAIN (3 VERSIONS)
  # ==========================================================================

  Scenario: Numeric constraint relaxation chain under BACKWARD_TRANSITIVE
    Given subject "json-compat-num-chain" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "json-compat-num-chain" has "JSON" schema:
      """
      {"type":"integer","minimum":0,"maximum":100}
      """
    When I register a "JSON" schema under subject "json-compat-num-chain":
      """
      {"type":"integer","minimum":0,"maximum":200}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-compat-num-chain":
      """
      {"type":"integer","minimum":0,"maximum":500}
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
      | target_id            | json-compat-num-chain                        |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-compat-num-chain/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 14. 4-VERSION CLOSED MODEL CHAIN UNDER BACKWARD_TRANSITIVE
  # ==========================================================================

  Scenario: Four-version closed model evolution under BACKWARD_TRANSITIVE
    Given subject "json-compat-4v-chain" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "json-compat-4v-chain" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-compat-4v-chain":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-compat-4v-chain":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-compat-4v-chain":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"},"age":{"type":"integer"}},"required":["name"],"additionalProperties":false}
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
      | target_id            | json-compat-4v-chain                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-compat-4v-chain/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
