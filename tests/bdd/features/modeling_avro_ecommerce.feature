@schema-modeling @avro @domain
Feature: Avro E-Commerce Domain Modeling
  Real-world e-commerce domain schemas exercising multi-version evolution,
  cross-subject references, logical types, lifecycle operations, and
  compatibility breaking changes.

  # ==========================================================================
  # 1. PRODUCT SCHEMA EVOLVES 5 VERSIONS UNDER BACKWARD_TRANSITIVE
  # ==========================================================================

  Scenario: Product schema evolves 5 versions under BACKWARD_TRANSITIVE
    Given subject "ecom-product" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "ecom-product" has schema:
      """
      {"type":"record","name":"Product","namespace":"com.shop","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"}
      ]}
      """
    When I register a schema under subject "ecom-product":
      """
      {"type":"record","name":"Product","namespace":"com.shop","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"description","type":"string","default":""}
      ]}
      """
    Then the response status should be 200
    When I register a schema under subject "ecom-product":
      """
      {"type":"record","name":"Product","namespace":"com.shop","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"description","type":"string","default":""},
        {"name":"category","type":{"type":"enum","name":"Category","symbols":["GENERAL","ELECTRONICS","CLOTHING","FOOD"]},"default":"GENERAL"}
      ]}
      """
    Then the response status should be 200
    When I register a schema under subject "ecom-product":
      """
      {"type":"record","name":"Product","namespace":"com.shop","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"description","type":"string","default":""},
        {"name":"category","type":{"type":"enum","name":"Category","symbols":["GENERAL","ELECTRONICS","CLOTHING","FOOD"]},"default":"GENERAL"},
        {"name":"weight","type":["null","float"],"default":null}
      ]}
      """
    Then the response status should be 200
    When I register a schema under subject "ecom-product":
      """
      {"type":"record","name":"Product","namespace":"com.shop","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"description","type":"string","default":""},
        {"name":"category","type":{"type":"enum","name":"Category","symbols":["GENERAL","ELECTRONICS","CLOTHING","FOOD"]},"default":"GENERAL"},
        {"name":"weight","type":["null","float"],"default":null},
        {"name":"tags","type":{"type":"array","items":"string"},"default":[]}
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
      | target_id            | ecom-product                                 |
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
      | path                 | /subjects/ecom-product/versions              |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 2. CUSTOMER SCHEMA WITH NESTED ADDRESS RECORD
  # ==========================================================================

  Scenario: Customer schema with nested Address record evolves
    Given subject "ecom-customer" has compatibility level "BACKWARD"
    And subject "ecom-customer" has schema:
      """
      {"type":"record","name":"Customer","namespace":"com.shop","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"address","type":{"type":"record","name":"Address","fields":[
          {"name":"street","type":"string"},
          {"name":"city","type":"string"},
          {"name":"zip","type":"string"}
        ]}},
        {"name":"loyalty","type":{"type":"enum","name":"Tier","symbols":["BRONZE","SILVER","GOLD"]},"default":"BRONZE"}
      ]}
      """
    When I register a schema under subject "ecom-customer":
      """
      {"type":"record","name":"Customer","namespace":"com.shop","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"address","type":{"type":"record","name":"Address","fields":[
          {"name":"street","type":"string"},
          {"name":"city","type":"string"},
          {"name":"zip","type":"string"}
        ]}},
        {"name":"loyalty","type":{"type":"enum","name":"Tier","symbols":["BRONZE","SILVER","GOLD"]},"default":"BRONZE"},
        {"name":"phone","type":["null","string"],"default":null}
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
      | target_id            | ecom-customer                                |
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
      | path                 | /subjects/ecom-customer/versions             |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 3. ORDER REFERENCES CUSTOMER AND PRODUCT
  # ==========================================================================

  Scenario: Order schema references Customer and Product via cross-subject refs
    Given subject "ecom-ref-customer" has schema:
      """
      {"type":"record","name":"Customer","namespace":"com.shop.ref","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"}
      ]}
      """
    And subject "ecom-ref-product" has schema:
      """
      {"type":"record","name":"Product","namespace":"com.shop.ref","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"}
      ]}
      """
    When I register a schema under subject "ecom-ref-order" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.shop.ref\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"customer\",\"type\":\"com.shop.ref.Customer\"},{\"name\":\"items\",\"type\":{\"type\":\"array\",\"items\":\"com.shop.ref.Product\"}}]}",
        "references": [
          {"name":"com.shop.ref.Customer","subject":"ecom-ref-customer","version":1},
          {"name":"com.shop.ref.Product","subject":"ecom-ref-product","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "ecom-ref-customer" version 1
    Then the response status should be 200
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | ecom-ref-order                               |
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
      | path                 | /subjects/ecom-ref-order/versions            |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 4. PAYMENT WITH LOGICAL TYPES
  # ==========================================================================

  Scenario: Payment schema with logical types registers and round-trips
    When I register a schema under subject "ecom-payment":
      """
      {"type":"record","name":"Payment","namespace":"com.shop","fields":[
        {"name":"amount","type":{"type":"bytes","logicalType":"decimal","precision":12,"scale":2}},
        {"name":"timestamp","type":{"type":"long","logicalType":"timestamp-millis"}},
        {"name":"currency","type":{"type":"enum","name":"Currency","symbols":["USD","EUR","GBP"]}},
        {"name":"method","type":{"type":"enum","name":"PayMethod","symbols":["CARD","BANK","WALLET"]}}
      ]}
      """
    Then the response status should be 200
    When I get version 1 of subject "ecom-payment"
    Then the response status should be 200
    And the response body should contain "decimal"
    And the response body should contain "timestamp-millis"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | ecom-payment                                 |
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
      | path                 | /subjects/ecom-payment/versions              |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 5. SAME PRODUCT IN TWO SUBJECTS — GLOBAL DEDUP
  # ==========================================================================

  Scenario: Same Product in two subjects produces same global ID
    When I register a schema under subject "ecom-product-value":
      """
      {"type":"record","name":"ProductEvent","namespace":"com.shop.events","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"price","type":"double"}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "product_event_id"
    When I register a schema under subject "ecom-product-changelog":
      """
      {"type":"record","name":"ProductEvent","namespace":"com.shop.events","fields":[
        {"name":"id","type":"long"},
        {"name":"name","type":"string"},
        {"name":"price","type":"double"}
      ]}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "product_event_id"
    And the audit log should contain an event:
      | event_type           | schema_register                                  |
      | outcome              | success                                          |
      | actor_id             |                                                  |
      | actor_type           | anonymous                                        |
      | auth_method          |                                                  |
      | role                 |                                                  |
      | target_type          | subject                                          |
      | target_id            | ecom-product-changelog                           |
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
      | path                 | /subjects/ecom-product-changelog/versions        |
      | status_code          | 200                                              |
      | reason               |                                                  |
      | error                |                                                  |
      | request_body         |                                                  |
      | metadata             |                                                  |
      | timestamp            | *                                                |
      | duration_ms          | *                                                |
      | request_id           | *                                                |

  # ==========================================================================
  # 6. FULL LIFECYCLE
  # ==========================================================================

  Scenario: Full lifecycle — register evolve soft-delete re-register
    Given subject "ecom-lifecycle" has compatibility level "BACKWARD"
    When I register a schema under subject "ecom-lifecycle":
      """
      {"type":"record","name":"Item","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}
      """
    Then the response status should be 200
    When I register a schema under subject "ecom-lifecycle":
      """
      {"type":"record","name":"Item","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"desc","type":"string","default":""}]}
      """
    Then the response status should be 200
    When I register a schema under subject "ecom-lifecycle":
      """
      {"type":"record","name":"Item","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"desc","type":"string","default":""},{"name":"qty","type":"int","default":0}]}
      """
    Then the response status should be 200
    When I delete subject "ecom-lifecycle"
    Then the response status should be 200
    When I register a schema under subject "ecom-lifecycle":
      """
      {"type":"record","name":"Item","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"},{"name":"desc","type":"string","default":""},{"name":"qty","type":"int","default":0},{"name":"sku","type":"string","default":""}]}
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
      | target_id            | ecom-lifecycle                               |
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
      | path                 | /subjects/ecom-lifecycle/versions            |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |

  # ==========================================================================
  # 7. BREAK COMPATIBILITY — REMOVE REQUIRED FIELD
  # ==========================================================================

  Scenario: Adding field without default breaks BACKWARD compatibility
    Given subject "ecom-break-compat" has compatibility level "BACKWARD"
    And subject "ecom-break-compat" has schema:
      """
      {"type":"record","name":"Order","fields":[
        {"name":"id","type":"long"},
        {"name":"total","type":"double"}
      ]}
      """
    # In BACKWARD, new schema (reader) must read old data (writer).
    # Adding a field without default means old data won't have it and
    # the reader has no default to fill in — incompatible.
    When I register a schema under subject "ecom-break-compat":
      """
      {"type":"record","name":"Order","fields":[
        {"name":"id","type":"long"},
        {"name":"total","type":"double"},
        {"name":"customer","type":"string"}
      ]}
      """
    Then the response status should be 409

  # ==========================================================================
  # 8. IMPORT AND EVOLVE
  # ==========================================================================

  @import
  Scenario: Import schema with specific ID then evolve via normal registration
    Given subject "ecom-import" has compatibility level "BACKWARD"
    When I set the global mode to "IMPORT"
    And I import a schema with ID 50000 under subject "ecom-import" version 1:
      """
      {"type":"record","name":"Imported","fields":[{"name":"id","type":"long"},{"name":"data","type":"string"}]}
      """
    Then the response status should be 200
    When I set the global mode to "READWRITE"
    And I register a schema under subject "ecom-import":
      """
      {"type":"record","name":"Imported","fields":[{"name":"id","type":"long"},{"name":"data","type":"string"},{"name":"extra","type":"string","default":""}]}
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
      | target_id            | ecom-import                                  |
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
      | path                 | /subjects/ecom-import/versions               |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
