@schema-modeling @json @conformance
Feature: JSON Schema Conformance-Inspired Parsing
  JSON Schema features from the official JSON Schema Test Suite (Draft-07)
  exercising all major keywords, composition patterns, and edge cases.

  # ==========================================================================
  # 1. EMPTY SCHEMA
  # ==========================================================================

  Scenario: Empty schema accepts everything
    When I register a "JSON" schema under subject "json-conform-empty":
      """
      {}
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
      | target_id            | json-conform-empty                       |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-conform-empty/versions    |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ==========================================================================
  # 2. ALL 7 PRIMITIVE TYPES
  # ==========================================================================

  Scenario: All 7 primitive type schemas register successfully
    When I register a "JSON" schema under subject "json-conform-type-string":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-type-integer":
      """
      {"type":"integer"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-type-number":
      """
      {"type":"number"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-type-boolean":
      """
      {"type":"boolean"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-type-null":
      """
      {"type":"null"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-type-object":
      """
      {"type":"object"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-type-array":
      """
      {"type":"array"}
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
      | target_id            | json-conform-type-array                      |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-conform-type-array/versions   |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 3. MULTI-TYPE ARRAY
  # ==========================================================================

  Scenario: Multi-type array produces different fingerprint than single type
    When I register a "JSON" schema under subject "json-conform-multitype-a":
      """
      {"type":"string"}
      """
    Then the response status should be 200
    And I store the response field "id" as "single_type_id"
    When I register a "JSON" schema under subject "json-conform-multitype-b":
      """
      {"type":["string","null"]}
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "single_type_id"
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | json-conform-multitype-b                       |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-conform-multitype-b/versions    |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 4. OBJECT WITH ALL PROPERTY KEYWORDS
  # ==========================================================================

  Scenario: Object with properties required and additionalProperties
    When I register a "JSON" schema under subject "json-conform-obj-props":
      """
      {"type":"object","properties":{"foo":{"type":"string"},"bar":{"type":"integer"}},"required":["foo"],"additionalProperties":false}
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
      | target_id            | json-conform-obj-props                       |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-conform-obj-props/versions    |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 5. PATTERN PROPERTIES + ADDITIONAL PROPERTIES
  # ==========================================================================

  Scenario: patternProperties with additionalProperties interaction
    When I register a "JSON" schema under subject "json-conform-pattern-props":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"patternProperties":{"^x-":{"type":"string"}},"additionalProperties":{"type":"integer"}}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | json-conform-pattern-props                       |
      | schema_id            | *                                                |
      | version              |                                                  |
      | schema_type          | JSON                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/json-conform-pattern-props/versions    |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 6. RECURSIVE $REF TO ROOT
  # ==========================================================================

  Scenario: Recursive ref to root schema
    When I register a "JSON" schema under subject "json-conform-recursive-ref":
      """
      {"type":"object","properties":{"children":{"type":"array","items":{"$ref":"#"}}},"additionalProperties":false}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | json-conform-recursive-ref                       |
      | schema_id            | *                                                |
      | version              |                                                  |
      | schema_type          | JSON                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/json-conform-recursive-ref/versions    |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 7. $REF THROUGH DEFINITIONS CHAIN
  # ==========================================================================

  Scenario: Ref through definitions chain a to b to c
    When I register a "JSON" schema under subject "json-conform-ref-chain":
      """
      {"definitions":{"a":{"type":"integer"},"b":{"$ref":"#/definitions/a"},"c":{"$ref":"#/definitions/b"}},"allOf":[{"$ref":"#/definitions/c"}]}
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
      | target_id            | json-conform-ref-chain                       |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-conform-ref-chain/versions    |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 8. IF/THEN/ELSE
  # ==========================================================================

  Scenario: Conditional schema with if then else
    When I register a "JSON" schema under subject "json-conform-conditional":
      """
      {"type":"object","properties":{"country":{"type":"string"},"postal_code":{"type":"string"}},"if":{"properties":{"country":{"const":"US"}},"required":["country"]},"then":{"properties":{"postal_code":{"pattern":"^[0-9]{5}$"}}},"else":{"properties":{"postal_code":{"pattern":"^[A-Z][0-9][A-Z] [0-9][A-Z][0-9]$"}}}}
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
      | target_id            | json-conform-conditional                       |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-conform-conditional/versions    |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 9. NOT KEYWORD
  # ==========================================================================

  Scenario: Not keyword registers successfully
    When I register a "JSON" schema under subject "json-conform-not":
      """
      {"not":{"type":"null"}}
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
      | target_id            | json-conform-not                         |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | JSON                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/json-conform-not/versions      |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ==========================================================================
  # 10. CONST KEYWORD
  # ==========================================================================

  Scenario: Const keyword in properties
    When I register a "JSON" schema under subject "json-conform-const":
      """
      {"type":"object","properties":{"status":{"const":"active"},"name":{"type":"string"}},"required":["status"]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | json-conform-const                         |
      | schema_id            | *                                          |
      | version              |                                            |
      | schema_type          | JSON                                       |
      | before_hash          |                                            |
      | after_hash           | sha256:*                                   |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/json-conform-const/versions      |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  # ==========================================================================
  # 11. CONTAINS KEYWORD
  # ==========================================================================

  Scenario: Contains keyword for array validation
    When I register a "JSON" schema under subject "json-conform-contains":
      """
      {"type":"array","contains":{"type":"integer"}}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | json-conform-contains                         |
      | schema_id            | *                                             |
      | version              |                                               |
      | schema_type          | JSON                                          |
      | before_hash          |                                               |
      | after_hash           | sha256:*                                      |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/json-conform-contains/versions      |
      | status_code          | 200                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  # ==========================================================================
  # 12. DEPENDENCIES
  # ==========================================================================

  Scenario: Dependencies keyword
    When I register a "JSON" schema under subject "json-conform-dependencies":
      """
      {"type":"object","properties":{"name":{"type":"string"},"credit_card":{"type":"string"},"billing_address":{"type":"string"}},"dependencies":{"credit_card":["billing_address"]}}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | json-conform-dependencies                        |
      | schema_id            | *                                                |
      | version              |                                                  |
      | schema_type          | JSON                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/json-conform-dependencies/versions     |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 13. COMBINED ALLOF + ONEOF
  # ==========================================================================

  Scenario: Combined allOf and oneOf composition
    When I register a "JSON" schema under subject "json-conform-allof-oneof":
      """
      {"allOf":[{"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}],"oneOf":[{"properties":{"type":{"const":"circle"},"radius":{"type":"number"}},"required":["type","radius"]},{"properties":{"type":{"const":"rect"},"w":{"type":"number"},"h":{"type":"number"}},"required":["type","w","h"]}]}
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
      | target_id            | json-conform-allof-oneof                       |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-conform-allof-oneof/versions    |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 14. ENUM WITH HETEROGENEOUS VALUES
  # ==========================================================================

  Scenario: Enum with heterogeneous value types
    When I register a "JSON" schema under subject "json-conform-hetero-enum":
      """
      {"enum":[1,"two",true,null,{"key":"val"},[1,2]]}
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
      | target_id            | json-conform-hetero-enum                       |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-conform-hetero-enum/versions    |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 15. FORMAT ANNOTATIONS
  # ==========================================================================

  Scenario: Format annotations for various string formats
    When I register a "JSON" schema under subject "json-conform-fmt-datetime":
      """
      {"type":"string","format":"date-time"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-email":
      """
      {"type":"string","format":"email"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-uri":
      """
      {"type":"string","format":"uri"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-ipv4":
      """
      {"type":"string","format":"ipv4"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-ipv6":
      """
      {"type":"string","format":"ipv6"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-uuid":
      """
      {"type":"string","format":"uuid"}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | json-conform-fmt-uuid                         |
      | schema_id            | *                                             |
      | version              |                                               |
      | schema_type          | JSON                                          |
      | before_hash          |                                               |
      | after_hash           | sha256:*                                      |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/json-conform-fmt-uuid/versions      |
      | status_code          | 200                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  # ==========================================================================
  # 16. PROPERTY NAMES
  # ==========================================================================

  Scenario: propertyNames constraint on object keys
    When I register a "JSON" schema under subject "json-conform-propnames":
      """
      {"type":"object","propertyNames":{"maxLength":5}}
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
      | target_id            | json-conform-propnames                         |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-conform-propnames/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 17. DEFINITIONS WITH $REF COMPOSITION
  # ==========================================================================

  Scenario: Definitions with ref composition and reuse
    When I register a "JSON" schema under subject "json-conform-defs-ref":
      """
      {"type":"object","definitions":{"address":{"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}},"required":["street","city"]}},"properties":{"home":{"$ref":"#/definitions/address"},"work":{"$ref":"#/definitions/address"}}}
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
      | target_id            | json-conform-defs-ref                        |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-conform-defs-ref/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 18. CONTENT ROUND-TRIP
  # ==========================================================================

  Scenario: Content round-trip verifies JSON Schema keywords preserved
    Given subject "json-conform-roundtrip" has "JSON" schema:
      """
      {"type":"object","definitions":{"address":{"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}},"required":["street","city"]}},"properties":{"name":{"type":"string"},"age":{"type":"integer"},"home":{"$ref":"#/definitions/address"}},"required":["name"]}
      """
    When I get version 1 of subject "json-conform-roundtrip"
    Then the response status should be 200
    And the response body should contain "properties"
    And the response body should contain "required"
    And the response body should contain "definitions"

  # ==========================================================================
  # 19. STANDALONE BOOLEAN ROOT SCHEMA — TRUE ACCEPTS EVERYTHING
  # ==========================================================================

  Scenario: Standalone boolean root schema true accepts everything
    When I register a "JSON" schema under subject "json-conform-bool-true":
      """
      true
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
      | target_id            | json-conform-bool-true                       |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | JSON                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/json-conform-bool-true/versions    |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 20. STANDALONE BOOLEAN ROOT SCHEMA — FALSE REJECTS EVERYTHING
  # ==========================================================================

  Scenario: Standalone boolean root schema false rejects everything
    When I register a "JSON" schema under subject "json-conform-bool-false":
      """
      false
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                               |
      | outcome              | success                                       |
      | actor_id             |                                               |
      | actor_type           | anonymous                                     |
      | auth_method          |                                               |
      | role                 |                                               |
      | target_type          | subject                                       |
      | target_id            | json-conform-bool-false                       |
      | schema_id            | *                                             |
      | version              |                                               |
      | schema_type          | JSON                                          |
      | before_hash          |                                               |
      | after_hash           | sha256:*                                      |
      | context              | .                                             |
      | transport_security   | tls                                           |
      | source_ip            | *                                             |
      | user_agent           | *                                             |
      | method               | POST                                          |
      | path                 | /subjects/json-conform-bool-false/versions    |
      | status_code          | 200                                           |
      | reason               |                                               |
      | error                |                                               |
      | request_body         |                                               |
      | metadata             |                                               |
      | timestamp            | *                                             |
      | duration_ms          | *                                             |
      | request_id           | *                                             |

  # ==========================================================================
  # 21. ADDITIONAL FORMAT ANNOTATIONS
  # ==========================================================================

  Scenario: Additional format annotations register successfully
    When I register a "JSON" schema under subject "json-conform-fmt-hostname":
      """
      {"type":"string","format":"hostname"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-json-pointer":
      """
      {"type":"string","format":"json-pointer"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-regex":
      """
      {"type":"string","format":"regex"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-uri-ref":
      """
      {"type":"string","format":"uri-reference"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-uri-tmpl":
      """
      {"type":"string","format":"uri-template"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-iri":
      """
      {"type":"string","format":"iri"}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-conform-fmt-iri-ref":
      """
      {"type":"string","format":"iri-reference"}
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
      | target_id            | json-conform-fmt-iri-ref                       |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | JSON                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/json-conform-fmt-iri-ref/versions    |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |
