@functional @avro
Feature: Avro Namespace Inheritance
  The Avro specification states that a nested named type (record, enum, fixed) without an
  explicit namespace inherits the namespace from the most tightly enclosing named type.
  The schema registry uses canonical form (with fully-qualified names) for fingerprinting
  and deduplication. These tests verify that namespace inheritance is correctly resolved
  during canonicalization so that:
    - Inherited and explicit namespaces produce the same canonical form and same schema ID
    - Different parent namespaces propagate different inherited namespaces and produce different IDs
    - Namespace inheritance works through arrays, maps, and unions

  # ==========================================================================
  # 1. INHERITED VS EXPLICIT NAMESPACE PRODUCES SAME SCHEMA ID
  # ==========================================================================

  Scenario: Inherited namespace deduplicates against explicit namespace
    # Register schema where Inner explicitly declares namespace "com.example" (same as parent)
    When I register a schema under subject "ns-inherit-explicit":
      """
      {"type":"record","name":"Outer","namespace":"com.example","fields":[
        {"name":"inner","type":{"type":"record","name":"Inner","namespace":"com.example","fields":[
          {"name":"value","type":"string"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "explicit_id"
    # Register schema where Inner inherits namespace from parent (no explicit namespace)
    When I register a schema under subject "ns-inherit-inherited":
      """
      {"type":"record","name":"Outer","namespace":"com.example","fields":[
        {"name":"inner","type":{"type":"record","name":"Inner","fields":[
          {"name":"value","type":"string"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "inherited_id"
    And the response field "id" should equal stored "explicit_id"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | ns-inherit-inherited                         |
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
      | path                 | /subjects/ns-inherit-inherited/versions      |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 2. DIFFERENT PARENT NAMESPACES PRODUCE DIFFERENT SCHEMA IDS
  # ==========================================================================

  Scenario: Different parent namespaces cause different inherited namespaces and different IDs
    # Register with parent namespace com.alpha — Inner inherits com.alpha
    When I register a schema under subject "ns-inherit-alpha":
      """
      {"type":"record","name":"Wrapper","namespace":"com.alpha","fields":[
        {"name":"child","type":{"type":"record","name":"Child","fields":[
          {"name":"code","type":"int"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "alpha_id"
    # Register with parent namespace com.beta — Inner inherits com.beta
    When I register a schema under subject "ns-inherit-beta":
      """
      {"type":"record","name":"Wrapper","namespace":"com.beta","fields":[
        {"name":"child","type":{"type":"record","name":"Child","fields":[
          {"name":"code","type":"int"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "beta_id"
    And the response field "id" should not equal stored "alpha_id"
    And the audit log should contain an event:
      | event_type           | schema_register                          |
      | outcome              | success                                  |
      | actor_id             |                                          |
      | actor_type           | anonymous                                |
      | auth_method          |                                          |
      | role                 |                                          |
      | target_type          | subject                                  |
      | target_id            | ns-inherit-beta                          |
      | schema_id            | *                                        |
      | version              |                                          |
      | schema_type          | AVRO                                     |
      | before_hash          |                                          |
      | after_hash           | sha256:*                                 |
      | context              | .                                        |
      | transport_security   | tls                                      |
      | source_ip            | *                                        |
      | user_agent           | *                                        |
      | method               | POST                                     |
      | path                 | /subjects/ns-inherit-beta/versions       |
      | status_code          | 200                                      |
      | reason               |                                          |
      | error                |                                          |
      | request_body         |                                          |
      | metadata             |                                          |
      | timestamp            | *                                        |
      | duration_ms          | *                                        |
      | request_id           | *                                        |

  # ==========================================================================
  # 3. THREE-LEVEL DEEP INHERITANCE DEDUPLICATION
  # ==========================================================================

  Scenario: Three-level nested inheritance deduplicates across subjects
    # Register a schema with 3 levels of nesting; only top level has namespace
    When I register a schema under subject "ns-inherit-deep-a":
      """
      {"type":"record","name":"Root","namespace":"com.deep","fields":[
        {"name":"mid","type":{"type":"record","name":"Middle","fields":[
          {"name":"leaf","type":{"type":"record","name":"Leaf","fields":[
            {"name":"data","type":"string"}
          ]}}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "deep_id"
    # Register the same schema under a different subject
    When I register a schema under subject "ns-inherit-deep-b":
      """
      {"type":"record","name":"Root","namespace":"com.deep","fields":[
        {"name":"mid","type":{"type":"record","name":"Middle","fields":[
          {"name":"leaf","type":{"type":"record","name":"Leaf","fields":[
            {"name":"data","type":"string"}
          ]}}
        ]}}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "deep_id"
    # Both subjects should be visible for this schema ID
    And I store the response field "id" as "schema_id"
    When I get the subjects for the stored schema ID
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain "ns-inherit-deep-a"
    And the response array should contain "ns-inherit-deep-b"
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | ns-inherit-deep-b                          |
      | schema_id            | *                                          |
      | version              |                                            |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           | sha256:*                                   |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/ns-inherit-deep-b/versions       |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  # ==========================================================================
  # 4. NESTED TYPE WITH OVERRIDDEN NAMESPACE DIFFERS FROM INHERITED
  # ==========================================================================

  Scenario: Nested type with overridden namespace produces different ID than inherited
    # Register schema where Inner inherits "com.example" from parent
    When I register a schema under subject "ns-inherit-parent-ns":
      """
      {"type":"record","name":"Container","namespace":"com.example","fields":[
        {"name":"item","type":{"type":"record","name":"Item","fields":[
          {"name":"name","type":"string"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "inherited_ns_id"
    # Register schema where Inner explicitly overrides to "com.other"
    When I register a schema under subject "ns-inherit-override-ns":
      """
      {"type":"record","name":"Container","namespace":"com.example","fields":[
        {"name":"item","type":{"type":"record","name":"Item","namespace":"com.other","fields":[
          {"name":"name","type":"string"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "override_ns_id"
    And the response field "id" should not equal stored "inherited_ns_id"
    And the audit log should contain an event:
      | event_type           | schema_register                                |
      | outcome              | success                                        |
      | actor_id             |                                                |
      | actor_type           | anonymous                                      |
      | auth_method          |                                                |
      | role                 |                                                |
      | target_type          | subject                                        |
      | target_id            | ns-inherit-override-ns                         |
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
      | path                 | /subjects/ns-inherit-override-ns/versions      |
      | status_code          | 200                                            |
      | reason               |                                                |
      | error                |                                                |
      | request_body         |                                                |
      | metadata             |                                                |
      | timestamp            | *                                              |
      | duration_ms          | *                                              |
      | request_id           | *                                              |

  # ==========================================================================
  # 5. NAMESPACE INHERITANCE THROUGH ARRAY ITEMS
  # ==========================================================================

  Scenario: Namespace inheritance through array items deduplicates correctly
    # Register schema with record nested inside array items, explicit namespace matching parent
    When I register a schema under subject "ns-inherit-array-explicit":
      """
      {"type":"record","name":"Collection","namespace":"com.arrays","fields":[
        {"name":"entries","type":{"type":"array","items":{"type":"record","name":"Entry","namespace":"com.arrays","fields":[
          {"name":"key","type":"string"}
        ]}}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "array_explicit_id"
    # Register same schema with inherited namespace on the nested record
    When I register a schema under subject "ns-inherit-array-inherited":
      """
      {"type":"record","name":"Collection","namespace":"com.arrays","fields":[
        {"name":"entries","type":{"type":"array","items":{"type":"record","name":"Entry","fields":[
          {"name":"key","type":"string"}
        ]}}}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "array_explicit_id"
    And the audit log should contain an event:
      | event_type           | schema_register                                    |
      | outcome              | success                                            |
      | actor_id             |                                                    |
      | actor_type           | anonymous                                          |
      | auth_method          |                                                    |
      | role                 |                                                    |
      | target_type          | subject                                            |
      | target_id            | ns-inherit-array-inherited                         |
      | schema_id            | *                                                  |
      | version              |                                                    |
      | schema_type          | AVRO                                               |
      | before_hash          |                                                    |
      | after_hash           | sha256:*                                           |
      | context              | .                                                  |
      | transport_security   | tls                                                |
      | source_ip            | *                                                  |
      | user_agent           | *                                                  |
      | method               | POST                                               |
      | path                 | /subjects/ns-inherit-array-inherited/versions      |
      | status_code          | 200                                                |
      | reason               |                                                    |
      | error                |                                                    |
      | request_body         |                                                    |
      | metadata             |                                                    |
      | timestamp            | *                                                  |
      | duration_ms          | *                                                  |
      | request_id           | *                                                  |

  # ==========================================================================
  # 6. NAMESPACE INHERITANCE THROUGH UNION
  # ==========================================================================

  Scenario: Namespace inheritance through union deduplicates correctly
    # Register schema with record inside union, explicit namespace matching parent
    When I register a schema under subject "ns-inherit-union-explicit":
      """
      {"type":"record","name":"Event","namespace":"com.unions","fields":[
        {"name":"payload","type":["null",{"type":"record","name":"Detail","namespace":"com.unions","fields":[
          {"name":"info","type":"string"}
        ]}],"default":null}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "union_explicit_id"
    # Register same schema with inherited namespace on the nested record
    When I register a schema under subject "ns-inherit-union-inherited":
      """
      {"type":"record","name":"Event","namespace":"com.unions","fields":[
        {"name":"payload","type":["null",{"type":"record","name":"Detail","fields":[
          {"name":"info","type":"string"}
        ]}],"default":null}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "union_explicit_id"
    And the audit log should contain an event:
      | event_type           | schema_register                                    |
      | outcome              | success                                            |
      | actor_id             |                                                    |
      | actor_type           | anonymous                                          |
      | auth_method          |                                                    |
      | role                 |                                                    |
      | target_type          | subject                                            |
      | target_id            | ns-inherit-union-inherited                         |
      | schema_id            | *                                                  |
      | version              |                                                    |
      | schema_type          | AVRO                                               |
      | before_hash          |                                                    |
      | after_hash           | sha256:*                                           |
      | context              | .                                                  |
      | transport_security   | tls                                                |
      | source_ip            | *                                                  |
      | user_agent           | *                                                  |
      | method               | POST                                               |
      | path                 | /subjects/ns-inherit-union-inherited/versions      |
      | status_code          | 200                                                |
      | reason               |                                                    |
      | error                |                                                    |
      | request_body         |                                                    |
      | metadata             |                                                    |
      | timestamp            | *                                                  |
      | duration_ms          | *                                                  |
      | request_id           | *                                                  |

  # ==========================================================================
  # 7. MIXED EXPLICIT AND INHERITED NAMESPACES
  # ==========================================================================

  Scenario: Mixed explicit and inherited namespaces deduplicate when equivalent
    # Register schema where some nested types have explicit namespaces, others inherit
    When I register a schema under subject "ns-inherit-mixed-partial":
      """
      {"type":"record","name":"Order","namespace":"com.mixed","fields":[
        {"name":"customer","type":{"type":"record","name":"Customer","fields":[
          {"name":"name","type":"string"}
        ]}},
        {"name":"status","type":{"type":"enum","name":"Status","symbols":["PENDING","SHIPPED","DELIVERED"]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "mixed_partial_id"
    # Register version where ALL namespaces are explicit (matching what would be inherited)
    When I register a schema under subject "ns-inherit-mixed-full":
      """
      {"type":"record","name":"Order","namespace":"com.mixed","fields":[
        {"name":"customer","type":{"type":"record","name":"Customer","namespace":"com.mixed","fields":[
          {"name":"name","type":"string"}
        ]}},
        {"name":"status","type":{"type":"enum","name":"Status","namespace":"com.mixed","symbols":["PENDING","SHIPPED","DELIVERED"]}}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "mixed_partial_id"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | ns-inherit-mixed-full                        |
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
      | path                 | /subjects/ns-inherit-mixed-full/versions     |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 8. NO NAMESPACE ANYWHERE — NESTED TYPES REMAIN UNQUALIFIED
  # ==========================================================================

  Scenario: No namespace at any level deduplicates across subjects
    # Register schema with no namespace anywhere
    When I register a schema under subject "ns-inherit-none-a":
      """
      {"type":"record","name":"Simple","fields":[
        {"name":"nested","type":{"type":"record","name":"Nested","fields":[
          {"name":"val","type":"int"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "none_id"
    # Register same schema under a different subject
    When I register a schema under subject "ns-inherit-none-b":
      """
      {"type":"record","name":"Simple","fields":[
        {"name":"nested","type":{"type":"record","name":"Nested","fields":[
          {"name":"val","type":"int"}
        ]}}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "none_id"
    And the audit log should contain an event:
      | event_type           | schema_register                            |
      | outcome              | success                                    |
      | actor_id             |                                            |
      | actor_type           | anonymous                                  |
      | auth_method          |                                            |
      | role                 |                                            |
      | target_type          | subject                                    |
      | target_id            | ns-inherit-none-b                          |
      | schema_id            | *                                          |
      | version              |                                            |
      | schema_type          | AVRO                                       |
      | before_hash          |                                            |
      | after_hash           | sha256:*                                   |
      | context              | .                                          |
      | transport_security   | tls                                        |
      | source_ip            | *                                          |
      | user_agent           | *                                          |
      | method               | POST                                       |
      | path                 | /subjects/ns-inherit-none-b/versions       |
      | status_code          | 200                                        |
      | reason               |                                            |
      | error                |                                            |
      | request_body         |                                            |
      | metadata             |                                            |
      | timestamp            | *                                          |
      | duration_ms          | *                                          |
      | request_id           | *                                          |

  # ==========================================================================
  # 9. OVERRIDE NAMESPACE PROPAGATES TO GRANDCHILDREN
  # ==========================================================================

  Scenario: Override namespace propagates to grandchildren and deduplicates
    # Parent is com.top, child overrides to com.middle, grandchild inherits com.middle
    # Register with grandchild inheriting from overridden child
    When I register a schema under subject "ns-inherit-propagate-inherited":
      """
      {"type":"record","name":"Top","namespace":"com.top","fields":[
        {"name":"mid","type":{"type":"record","name":"Mid","namespace":"com.middle","fields":[
          {"name":"bottom","type":{"type":"record","name":"Bottom","fields":[
            {"name":"flag","type":"boolean"}
          ]}}
        ]}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "propagate_inherited_id"
    # Register with grandchild having explicit namespace com.middle (matching inherited)
    When I register a schema under subject "ns-inherit-propagate-explicit":
      """
      {"type":"record","name":"Top","namespace":"com.top","fields":[
        {"name":"mid","type":{"type":"record","name":"Mid","namespace":"com.middle","fields":[
          {"name":"bottom","type":{"type":"record","name":"Bottom","namespace":"com.middle","fields":[
            {"name":"flag","type":"boolean"}
          ]}}
        ]}}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "propagate_inherited_id"
    And the audit log should contain an event:
      | event_type           | schema_register                                      |
      | outcome              | success                                              |
      | actor_id             |                                                      |
      | actor_type           | anonymous                                            |
      | auth_method          |                                                      |
      | role                 |                                                      |
      | target_type          | subject                                              |
      | target_id            | ns-inherit-propagate-explicit                        |
      | schema_id            | *                                                    |
      | version              |                                                      |
      | schema_type          | AVRO                                                 |
      | before_hash          |                                                      |
      | after_hash           | sha256:*                                             |
      | context              | .                                                    |
      | transport_security   | tls                                                  |
      | source_ip            | *                                                    |
      | user_agent           | *                                                    |
      | method               | POST                                                 |
      | path                 | /subjects/ns-inherit-propagate-explicit/versions     |
      | status_code          | 200                                                  |
      | reason               |                                                      |
      | error                |                                                      |
      | request_body         |                                                      |
      | metadata             |                                                      |
      | timestamp            | *                                                    |
      | duration_ms          | *                                                    |
      | request_id           | *                                                    |
