@schema-modeling @avro @conformance
Feature: Avro Conformance-Inspired Parsing
  Avro schemas derived from the Apache Avro interop test suite and specification
  exercising parsing, canonicalization, fingerprinting, and deduplication edge cases.

  # ==========================================================================
  # 1. KITCHEN-SINK INTEROP SCHEMA
  # ==========================================================================

  Scenario: Kitchen-sink interop schema with all types registers and round-trips
    When I register a schema under subject "avro-conform-interop":
      """
      {"type":"record","name":"Interop","namespace":"org.apache.avro","fields":[
        {"name":"intField","type":"int"},
        {"name":"longField","type":"long"},
        {"name":"stringField","type":"string"},
        {"name":"boolField","type":"boolean"},
        {"name":"floatField","type":"float"},
        {"name":"doubleField","type":"double"},
        {"name":"nullField","type":"null"},
        {"name":"bytesField","type":"bytes"},
        {"name":"arrayField","type":{"type":"array","items":"double"}},
        {"name":"mapField","type":{"type":"map","values":{"type":"record","name":"Foo","fields":[{"name":"label","type":"string"}]}}},
        {"name":"unionField","type":["boolean","double",{"type":"array","items":"bytes"}]},
        {"name":"enumField","type":{"type":"enum","name":"Kind","symbols":["A","B","C"]}},
        {"name":"fixedField","type":{"type":"fixed","name":"MD5","size":16}},
        {"name":"recordField","type":{"type":"record","name":"Node","fields":[
          {"name":"label","type":"string"},
          {"name":"children","type":{"type":"array","items":"Node"}}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "interop_id"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | avro-conform-interop                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-conform-interop/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 2. DEEPLY NESTED CONTAINERS
  # ==========================================================================

  Scenario: Deeply nested containers — record to array to map to union to record
    When I register a schema under subject "avro-conform-deep-nest":
      """
      {"type":"record","name":"Deep","namespace":"com.deep","fields":[
        {"name":"data","type":{"type":"array","items":{"type":"map","values":["null",{"type":"record","name":"Leaf","fields":[
          {"name":"tag","type":{"type":"enum","name":"Tag","symbols":["X","Y"]}}
        ]}]}}}
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
      | target_id            | avro-conform-deep-nest                         |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-conform-deep-nest/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 3. AVRO ERROR TYPE
  # ==========================================================================

  Scenario: Avro error type is functionally identical to record
    When I register a schema under subject "avro-conform-error-type":
      """
      {"type":"error","name":"ServiceError","namespace":"com.svc","fields":[
        {"name":"code","type":"int"},
        {"name":"message","type":"string"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "error_id"
    When I get the raw schema for subject "avro-conform-error-type" version 1
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | avro-conform-error-type                        |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-conform-error-type/versions     |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 4. LOGICAL TYPES IN NULLABLE UNION
  # ==========================================================================

  Scenario: Logical types in nullable unions register successfully
    When I register a schema under subject "avro-conform-logical":
      """
      {"type":"record","name":"Event","namespace":"com.events","fields":[
        {"name":"ts","type":["null",{"type":"long","logicalType":"timestamp-millis"}],"default":null},
        {"name":"amount","type":{"type":"bytes","logicalType":"decimal","precision":10,"scale":2}},
        {"name":"date","type":{"type":"int","logicalType":"date"}}
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
      | target_id            | avro-conform-logical                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-conform-logical/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 5. ORDER ATTRIBUTE STRIPPED FROM CANONICAL FORM
  # ==========================================================================

  @axonops-only
  Scenario: Record with order attribute deduplicates against record without
    When I register a schema under subject "avro-conform-order-a":
      """
      {"type":"record","name":"R","fields":[
        {"name":"a","type":"string","order":"ascending"},
        {"name":"b","type":"int","order":"descending"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "order_id_a"
    When I register a schema under subject "avro-conform-order-b":
      """
      {"type":"record","name":"R","fields":[
        {"name":"a","type":"string"},
        {"name":"b","type":"int"}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "order_id_a"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | avro-conform-order-b                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-conform-order-b/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 6. DOC AND ALIASES STRIPPED — SAME FINGERPRINT
  # ==========================================================================

  @axonops-only
  Scenario: Doc and aliases are stripped producing same fingerprint
    When I register a schema under subject "avro-conform-doc-a":
      """
      {"type":"record","name":"R","namespace":"com.x","doc":"A record","aliases":["OldR"],"fields":[
        {"name":"f","type":"string","doc":"a field","aliases":["old_f"]}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "doc_id"
    When I register a schema under subject "avro-conform-doc-b":
      """
      {"type":"record","name":"R","namespace":"com.x","fields":[
        {"name":"f","type":"string"}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "doc_id"
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | avro-conform-doc-b                         |
      | schema_id            | *                                          |
      | version              | *                                          |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           | sha256:*                                   |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/avro-conform-doc-b/versions      |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  # ==========================================================================
  # 7. ENUM SYMBOL ORDERING MATTERS
  # ==========================================================================

  Scenario: Enum symbol ordering matters for fingerprint
    When I register a schema under subject "avro-conform-enum-order-a":
      """
      {"type":"enum","name":"Direction","symbols":["NORTH","SOUTH","EAST","WEST"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "enum_order_a"
    When I register a schema under subject "avro-conform-enum-order-b":
      """
      {"type":"enum","name":"Direction","symbols":["WEST","EAST","SOUTH","NORTH"]}
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "enum_order_a"
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | avro-conform-enum-order-b                        |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-conform-enum-order-b/versions     |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 8. UNION ORDERING MATTERS
  # ==========================================================================

  Scenario: Union ordering matters for fingerprint
    When I register a schema under subject "avro-conform-union-order-a":
      """
      {"type":"record","name":"U","fields":[
        {"name":"v","type":["null","string"]}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "union_order_a"
    When I register a schema under subject "avro-conform-union-order-b":
      """
      {"type":"record","name":"U","fields":[
        {"name":"v","type":["string","null"]}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "union_order_a"
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | avro-conform-union-order-b                       |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-conform-union-order-b/versions    |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 9. DEFAULT VALUE DIFFERENCES PRODUCE DIFFERENT FINGERPRINTS
  # ==========================================================================

  Scenario: Default value differences produce different fingerprints
    When I register a schema under subject "avro-conform-default-a":
      """
      {"type":"record","name":"D","fields":[
        {"name":"s","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "default_a"
    When I register a schema under subject "avro-conform-default-b":
      """
      {"type":"record","name":"D","fields":[
        {"name":"s","type":"string","default":"hello"}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should not equal stored "default_a"
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | avro-conform-default-b                         |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-conform-default-b/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 10. FIELD DEFAULTS FOR EVERY TYPE
  # ==========================================================================

  Scenario: Field defaults for every Avro type register successfully
    When I register a schema under subject "avro-conform-all-defaults":
      """
      {"type":"record","name":"Defaults","fields":[
        {"name":"nullF","type":"null","default":null},
        {"name":"boolF","type":"boolean","default":false},
        {"name":"intF","type":"int","default":0},
        {"name":"longF","type":"long","default":0},
        {"name":"floatF","type":"float","default":0.0},
        {"name":"doubleF","type":"double","default":0.0},
        {"name":"stringF","type":"string","default":""},
        {"name":"arrayF","type":{"type":"array","items":"int"},"default":[]},
        {"name":"mapF","type":{"type":"map","values":"string"},"default":{}},
        {"name":"unionF","type":["null","string"],"default":null}
      ]}
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
      | target_id            | avro-conform-all-defaults                        |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-conform-all-defaults/versions     |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 11. ENUM WITH LOWERCASE SYMBOLS
  # ==========================================================================

  Scenario: Enum with lowercase symbols registers successfully
    When I register a schema under subject "avro-conform-lower-enum":
      """
      {"type":"enum","name":"Lower","symbols":["alpha","beta","gamma"]}
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
      | target_id            | avro-conform-lower-enum                        |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-conform-lower-enum/versions     |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 12. MAP WITH INLINE RECORD DEFINITION AND BACK-REFERENCE
  # ==========================================================================

  Scenario: Map with inline record definition and later back-reference
    When I register a schema under subject "avro-conform-inline-ref":
      """
      {"type":"record","name":"Container","fields":[
        {"name":"lookup","type":{"type":"map","values":{"type":"record","name":"Entry","fields":[{"name":"v","type":"int"}]}}},
        {"name":"single","type":"Entry"}
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
      | target_id            | avro-conform-inline-ref                        |
      | schema_id            | *                                              |
      | version              | *                                              |
      | schema_type          | AVRO                                           |
      | before_hash          |                                                |
      | after_hash           | sha256:*                                       |
      | context              | .                                              |
      | transport_security   | tls                                            |
      | source_ip            | *                                              |
      | user_agent           | *                                              |
      | method               | POST                                           |
      | path                 | /subjects/avro-conform-inline-ref/versions     |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 13. COMPLEX UNION BRANCHES
  # ==========================================================================

  Scenario: Complex union with multiple container branches registers
    When I register a schema under subject "avro-conform-complex-union":
      """
      {"type":"record","name":"CU","fields":[
        {"name":"val","type":["null","boolean","double",{"type":"array","items":"bytes"},{"type":"map","values":"string"}]}
      ]}
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
      | target_id            | avro-conform-complex-union                       |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-conform-complex-union/versions    |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 14. CONTENT ROUND-TRIP FOR INTEROP SCHEMA
  # ==========================================================================

  Scenario: Content round-trip verifies interop schema fields preserved
    Given subject "avro-conform-roundtrip" has schema:
      """
      {"type":"record","name":"Interop","namespace":"org.apache.avro","fields":[
        {"name":"intField","type":"int"},
        {"name":"longField","type":"long"},
        {"name":"stringField","type":"string"},
        {"name":"boolField","type":"boolean"},
        {"name":"floatField","type":"float"},
        {"name":"doubleField","type":"double"},
        {"name":"nullField","type":"null"},
        {"name":"bytesField","type":"bytes"},
        {"name":"arrayField","type":{"type":"array","items":"double"}},
        {"name":"mapField","type":{"type":"map","values":{"type":"record","name":"Foo","fields":[{"name":"label","type":"string"}]}}},
        {"name":"unionField","type":["boolean","double",{"type":"array","items":"bytes"}]},
        {"name":"enumField","type":{"type":"enum","name":"Kind","symbols":["A","B","C"]}},
        {"name":"fixedField","type":{"type":"fixed","name":"MD5","size":16}},
        {"name":"recordField","type":{"type":"record","name":"Node","fields":[
          {"name":"label","type":"string"},
          {"name":"children","type":{"type":"array","items":"Node"}}
        ]}}
      ]}
      """
    When I get version 1 of subject "avro-conform-roundtrip"
    Then the response status should be 200
    And the response body should contain "Interop"
    And the response body should contain "Node"
    And the response body should contain "Kind"
    And the response body should contain "MD5"
    And the response body should contain "Foo"

  # ==========================================================================
  # 15. SAME SCHEMA IN TWO SUBJECTS — SAME GLOBAL ID (DEDUP)
  # ==========================================================================

  Scenario: Same schema in two subjects produces same global ID
    When I register a schema under subject "avro-conform-dedup-a":
      """
      {"type":"record","name":"Shared","namespace":"com.dedup","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "dedup_id"
    When I register a schema under subject "avro-conform-dedup-b":
      """
      {"type":"record","name":"Shared","namespace":"com.dedup","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "dedup_id"
    And I store the response field "id" as "schema_id"
    When I get the subjects for the stored schema ID
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "avro-conform-dedup-a"
    And the response array should contain "avro-conform-dedup-b"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | avro-conform-dedup-b                         |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .                                            |
      | transport_security   | tls                                          |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
      | method               | POST                                         |
      | path                 | /subjects/avro-conform-dedup-b/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 16. ENUM WITH RESERVED-WORD SYMBOLS
  # ==========================================================================

  Scenario: Enum with Avro primitive type names as symbols
    When I register a schema under subject "avro-conform-reserved-enum":
      """
      {"type":"enum","name":"Keywords","symbols":["null","boolean","int","long","float","double","string","bytes"]}
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
      | target_id            | avro-conform-reserved-enum                       |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-conform-reserved-enum/versions    |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 17. ENUM WITH ALIASES
  # ==========================================================================

  Scenario: Enum with aliases attribute registers and round-trips
    When I register a schema under subject "avro-conform-enum-aliases":
      """
      {"type":"record","name":"Container","namespace":"com.alias","fields":[
        {"name":"status","type":{"type":"enum","name":"Status","aliases":["OldStatus","LegacyStatus"],"symbols":["ACTIVE","INACTIVE","PENDING"]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "enum_aliases_id"
    When I get version 1 of subject "avro-conform-enum-aliases"
    Then the response status should be 200
    And the response body should contain "aliases"
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | avro-conform-enum-aliases                        |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-conform-enum-aliases/versions     |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 18. DECIMAL LOGICAL TYPE ON FIXED BACKING TYPE
  # ==========================================================================

  Scenario: Decimal logical type on fixed backing type registers and round-trips
    When I register a schema under subject "avro-conform-decimal-fixed":
      """
      {"type":"record","name":"FinancialRecord","namespace":"com.finance","fields":[
        {"name":"amount","type":{"type":"fixed","name":"Decimal","size":8,"logicalType":"decimal","precision":18,"scale":4}},
        {"name":"balance","type":{"type":"bytes","logicalType":"decimal","precision":12,"scale":2}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "decimal_fixed_id"
    When I get version 1 of subject "avro-conform-decimal-fixed"
    Then the response status should be 200
    And the response body should contain "Decimal"
    And the response body should contain "decimal"
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | avro-conform-decimal-fixed                       |
      | schema_id            | *                                                |
      | version              | *                                                |
      | schema_type          | AVRO                                             |
      | before_hash          |                                                  |
      | after_hash           | sha256:*                                         |
      | context              | .                                                |
      | transport_security   | tls                                              |
      | source_ip            | *                                                |
      | user_agent           | *                                                |
      | method               | POST                                             |
      | path                 | /subjects/avro-conform-decimal-fixed/versions    |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |