@schema-modeling @avro @compatibility
Feature: Avro Advanced Compatibility
  Tests grounded in the Avro specification's type promotion rules and the
  compatibility checker implementation (canPromote, checkEnum, checkFixed,
  recordNamesMatch). Exercises every promotion path, enum evolution,
  fixed constraints, aliases, and multi-version transitive chains.

  # ==========================================================================
  # 1-6. COMPLETE TYPE PROMOTION MATRIX
  # ==========================================================================

  Scenario: Type promotion int to long under BACKWARD
    Given subject "avro-compat-int-long" has compatibility level "BACKWARD"
    And subject "avro-compat-int-long" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"int"}]}
      """
    When I register a schema under subject "avro-compat-int-long":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"long"}]}
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
      | target_id            | avro-compat-int-long                         |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-compat-int-long/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Type promotion int to float under BACKWARD
    Given subject "avro-compat-int-float" has compatibility level "BACKWARD"
    And subject "avro-compat-int-float" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"int"}]}
      """
    When I register a schema under subject "avro-compat-int-float":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"float"}]}
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
      | target_id            | avro-compat-int-float                        |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-compat-int-float/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Type promotion int to double under BACKWARD
    Given subject "avro-compat-int-double" has compatibility level "BACKWARD"
    And subject "avro-compat-int-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"int"}]}
      """
    When I register a schema under subject "avro-compat-int-double":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"double"}]}
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
      | target_id            | avro-compat-int-double                       |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-compat-int-double/versions    |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Type promotion long to float under BACKWARD
    Given subject "avro-compat-long-float" has compatibility level "BACKWARD"
    And subject "avro-compat-long-float" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"long"}]}
      """
    When I register a schema under subject "avro-compat-long-float":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"float"}]}
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
      | target_id            | avro-compat-long-float                       |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-compat-long-float/versions    |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Type promotion long to double under BACKWARD
    Given subject "avro-compat-long-double" has compatibility level "BACKWARD"
    And subject "avro-compat-long-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"long"}]}
      """
    When I register a schema under subject "avro-compat-long-double":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"double"}]}
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
      | target_id            | avro-compat-long-double                      |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-compat-long-double/versions   |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  Scenario: Type promotion float to double under BACKWARD
    Given subject "avro-compat-float-double" has compatibility level "BACKWARD"
    And subject "avro-compat-float-double" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"float"}]}
      """
    When I register a schema under subject "avro-compat-float-double":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"double"}]}
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
      | target_id            | avro-compat-float-double                     |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-compat-float-double/versions  |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 7. STRING BYTES BIDIRECTIONAL (FULL)
  # ==========================================================================

  Scenario: String and bytes are bidirectionally compatible under FULL
    Given subject "avro-compat-str-bytes" has compatibility level "FULL"
    And subject "avro-compat-str-bytes" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"string"}]}
      """
    When I register a schema under subject "avro-compat-str-bytes":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":"bytes"}]}
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
      | target_id            | avro-compat-str-bytes                        |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-compat-str-bytes/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 8. TYPE PROMOTION INSIDE UNION
  # ==========================================================================

  Scenario: Type promotion inside union under BACKWARD
    Given subject "avro-compat-union-promote" has compatibility level "BACKWARD"
    And subject "avro-compat-union-promote" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":["null","int"]}]}
      """
    When I register a schema under subject "avro-compat-union-promote":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":["null","long"]}]}
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
      | target_id            | avro-compat-union-promote                        |
      | schema_id            | *                                                |
      | version              |                                                  |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-compat-union-promote/versions     |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 9. COLLECTION ITEM PROMOTION
  # ==========================================================================

  Scenario: Array item type promotion under BACKWARD
    Given subject "avro-compat-array-promote" has compatibility level "BACKWARD"
    And subject "avro-compat-array-promote" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":{"type":"array","items":"int"}}]}
      """
    When I register a schema under subject "avro-compat-array-promote":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":{"type":"array","items":"long"}}]}
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
      | target_id            | avro-compat-array-promote                        |
      | schema_id            | *                                                |
      | version              |                                                  |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-compat-array-promote/versions     |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 10. MAP VALUE PROMOTION
  # ==========================================================================

  Scenario: Map value type promotion under BACKWARD
    Given subject "avro-compat-map-promote" has compatibility level "BACKWARD"
    And subject "avro-compat-map-promote" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":{"type":"map","values":"int"}}]}
      """
    When I register a schema under subject "avro-compat-map-promote":
      """
      {"type":"record","name":"R","fields":[{"name":"v","type":{"type":"map","values":"long"}}]}
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
      | target_id            | avro-compat-map-promote                        |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-compat-map-promote/versions     |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 11. ENUM REMOVING SYMBOL WITH DEFAULT — COMPATIBLE
  # ==========================================================================

  Scenario: Enum removing symbol with default is compatible under BACKWARD
    Given subject "avro-compat-enum-default" has compatibility level "BACKWARD"
    And subject "avro-compat-enum-default" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B","C"]}}]}
      """
    When I register a schema under subject "avro-compat-enum-default":
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B"],"default":"A"}}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                 |
      | outcome              | success                                         |
      | actor_id             |                                                 |
      | actor_type           | anonymous                                       |
      | auth_method          |                                                 |
      | role                 |                                                 |
      | target_type          | subject                                         |
      | target_id            | avro-compat-enum-default                        |
      | schema_id            | *                                               |
      | version              |                                                 |
      | schema_type          | AVRO                                            |
      | before_hash          |                                                 |
      | after_hash           | sha256:*                                        |
      | context              | .                                               |
      | transport_security   | tls                                             |
      | source_ip            | *                                               |
      | user_agent           | *                                               |
      | method               | POST                                            |
      | path                 | /subjects/avro-compat-enum-default/versions     |
      | status_code          | 200                                             |
      | reason               |                                                 |
      | error                |                                                 |
      | request_body         |                                                 |
      | metadata             |                                                 |
      | timestamp            | *                                               |
      | duration_ms          | *                                               |
      | request_id           | *                                               |

  # ==========================================================================
  # 12. ENUM REMOVING SYMBOL WITHOUT DEFAULT — INCOMPATIBLE
  # ==========================================================================

  Scenario: Enum removing symbol without default is incompatible under BACKWARD
    Given subject "avro-compat-enum-no-default" has compatibility level "BACKWARD"
    And subject "avro-compat-enum-no-default" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B","C"]}}]}
      """
    When I register a schema under subject "avro-compat-enum-no-default":
      """
      {"type":"record","name":"R","fields":[{"name":"e","type":{"type":"enum","name":"E","symbols":["A","B"]}}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # 13. FIXED SIZE MISMATCH — INCOMPATIBLE
  # ==========================================================================

  Scenario: Fixed type size mismatch is incompatible
    Given subject "avro-compat-fixed-size" has compatibility level "BACKWARD"
    And subject "avro-compat-fixed-size" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"h","type":{"type":"fixed","name":"Hash","size":16}}]}
      """
    When I register a schema under subject "avro-compat-fixed-size":
      """
      {"type":"record","name":"R","fields":[{"name":"h","type":{"type":"fixed","name":"Hash","size":32}}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # 14. FIXED NAME MISMATCH — INCOMPATIBLE
  # ==========================================================================

  Scenario: Fixed type name mismatch is incompatible
    Given subject "avro-compat-fixed-name" has compatibility level "BACKWARD"
    And subject "avro-compat-fixed-name" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"h","type":{"type":"fixed","name":"MD5","size":16}}]}
      """
    When I register a schema under subject "avro-compat-fixed-name":
      """
      {"type":"record","name":"R","fields":[{"name":"h","type":{"type":"fixed","name":"SHA256","size":16}}]}
      """
    Then the response status should be 409

  # ==========================================================================
  # 15. RECORD ALIAS ENABLES BACKWARD-COMPATIBLE RENAME
  # ==========================================================================

  Scenario: Record alias enables backward-compatible rename
    Given subject "avro-compat-record-alias" has compatibility level "BACKWARD"
    And subject "avro-compat-record-alias" has schema:
      """
      {"type":"record","name":"OldName","fields":[{"name":"x","type":"int"}]}
      """
    When I register a schema under subject "avro-compat-record-alias":
      """
      {"type":"record","name":"NewName","aliases":["OldName"],"fields":[{"name":"x","type":"int"}]}
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
      | target_id            | avro-compat-record-alias                         |
      | schema_id            | *                                                |
      | version              |                                                  |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-compat-record-alias/versions      |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 16. FIELD ALIAS ENABLES BACKWARD-COMPATIBLE RENAME
  # ==========================================================================

  Scenario: Field alias enables backward-compatible field rename
    Given subject "avro-compat-field-alias" has compatibility level "BACKWARD"
    And subject "avro-compat-field-alias" has schema:
      """
      {"type":"record","name":"R","fields":[{"name":"old_field","type":"string"}]}
      """
    When I register a schema under subject "avro-compat-field-alias":
      """
      {"type":"record","name":"R","fields":[{"name":"new_field","type":"string","aliases":["old_field"]}]}
      """
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                 |
      | outcome              | success                                         |
      | actor_id             |                                                 |
      | actor_type           | anonymous                                       |
      | auth_method          |                                                 |
      | role                 |                                                 |
      | target_type          | subject                                         |
      | target_id            | avro-compat-field-alias                         |
      | schema_id            | *                                               |
      | version              |                                                 |
      | schema_type          | AVRO                                            |
      | before_hash          |                                                 |
      | after_hash           | sha256:*                                        |
      | context              | .                                               |
      | transport_security   | tls                                             |
      | source_ip            | *                                               |
      | user_agent           | *                                               |
      | method               | POST                                            |
      | path                 | /subjects/avro-compat-field-alias/versions      |
      | status_code          | 200                                             |
      | reason               |                                                 |
      | error                |                                                 |
      | request_body         |                                                 |
      | metadata             |                                                 |
      | timestamp            | *                                               |
      | duration_ms          | *                                               |
      | request_id           | *                                               |

  # ==========================================================================
  # 17. 5-VERSION EVOLUTION CHAIN UNDER BACKWARD_TRANSITIVE
  # ==========================================================================

  Scenario: Five-version evolution chain under BACKWARD_TRANSITIVE
    Given subject "avro-compat-5v-chain" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "avro-compat-5v-chain" has schema:
      """
      {"type":"record","name":"Person","fields":[
        {"name":"name","type":"string"},
        {"name":"age","type":"int"}
      ]}
      """
    When I register a schema under subject "avro-compat-5v-chain":
      """
      {"type":"record","name":"Person","fields":[
        {"name":"name","type":"string"},
        {"name":"age","type":"int"},
        {"name":"email","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    When I register a schema under subject "avro-compat-5v-chain":
      """
      {"type":"record","name":"Person","fields":[
        {"name":"name","type":"string"},
        {"name":"age","type":"int"},
        {"name":"email","type":"string","default":""},
        {"name":"phone","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    When I register a schema under subject "avro-compat-5v-chain":
      """
      {"type":"record","name":"Person","fields":[
        {"name":"name","type":"string"},
        {"name":"age","type":"int"},
        {"name":"email","type":"string","default":""},
        {"name":"phone","type":"string","default":""},
        {"name":"address","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    When I register a schema under subject "avro-compat-5v-chain":
      """
      {"type":"record","name":"Person","fields":[
        {"name":"name","type":"string"},
        {"name":"age","type":"int"},
        {"name":"email","type":"string","default":""},
        {"name":"phone","type":"string","default":""},
        {"name":"address","type":"string","default":""},
        {"name":"city","type":"string","default":""}
      ]}
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
      | target_id            | avro-compat-5v-chain                         |
      | schema_id            | *                                            |
      | version              |                                              |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-compat-5v-chain/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 18. FULL_TRANSITIVE 3-VERSION CHAIN
  # ==========================================================================

  Scenario: Three-version chain under FULL_TRANSITIVE with bidirectional defaults
    Given subject "avro-compat-full-trans" has compatibility level "FULL_TRANSITIVE"
    And subject "avro-compat-full-trans" has schema:
      """
      {"type":"record","name":"Cfg","fields":[
        {"name":"host","type":"string","default":"localhost"},
        {"name":"port","type":"int","default":8080}
      ]}
      """
    When I register a schema under subject "avro-compat-full-trans":
      """
      {"type":"record","name":"Cfg","fields":[
        {"name":"host","type":"string","default":"localhost"},
        {"name":"port","type":"int","default":8080},
        {"name":"timeout","type":"int","default":30}
      ]}
      """
    Then the response status should be 200
    When I register a schema under subject "avro-compat-full-trans":
      """
      {"type":"record","name":"Cfg","fields":[
        {"name":"host","type":"string","default":"localhost"},
        {"name":"port","type":"int","default":8080},
        {"name":"timeout","type":"int","default":30},
        {"name":"retries","type":"int","default":3}
      ]}
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
      | target_id            | avro-compat-full-trans                         |
      | schema_id            | *                                              |
      | version              |                                                |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-compat-full-trans/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |
