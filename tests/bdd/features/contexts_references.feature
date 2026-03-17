@functional @contexts
Feature: Contexts — Schema References
  Verify that schema references (Avro named types, Protobuf imports) work
  correctly within named contexts. References MUST be resolved within the
  same context — the registry uses the registryCtx when looking up reference
  subjects. Cross-context reference resolution MUST NOT succeed.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # AVRO REFERENCE WITHIN SAME CONTEXT
  # ==========================================================================

  Scenario: Avro schema with reference to another subject in same context
    # Register the referenced "Address" schema in context .ref1
    When I POST "/subjects/:.ref1:address-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "addr_id_ref1"
    # Register "Order" that references Address in the same context
    When I POST "/subjects/:.ref1:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "order_id_ref1"
    # Verify GET version 1 of order-value returns the schema with references
    When I GET "/subjects/:.ref1:order-value/versions/1"
    Then the response status should be 200
    And the response should contain "Order"
    And the response should contain "references"
    And the response should contain "com.example.Address"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ref1:order-value                           |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ref1                                        |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ref1:order-value/versions        |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # REFERENCEDBY ENDPOINT IN CONTEXT
  # ==========================================================================

  Scenario: ReferencedBy endpoint shows referencing schemas in context
    # Register Address in context .ref2
    When I POST "/subjects/:.ref2:address-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "addr_id"
    # Register Order referencing Address in same context
    When I POST "/subjects/:.ref2:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "order_id"
    # Check referencedby on the Address schema
    When I GET "/subjects/:.ref2:address-value/versions/1/referencedby"
    Then the response status should be 200
    And the response should be valid JSON
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ref2:order-value                           |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ref2                                        |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ref2:order-value/versions        |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # CROSS-CONTEXT REFERENCE ISOLATION
  # ==========================================================================

  Scenario: References are context-isolated — reference in one context not visible in another
    # Register Address in context .ref3a
    When I POST "/subjects/:.ref3a:address-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register Order referencing Address in context .ref3a — should succeed
    When I POST "/subjects/:.ref3a:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 200
    # Try to register Order in context .ref3b referencing address-value
    # This MUST fail because address-value does not exist in context .ref3b
    When I POST "/subjects/:.ref3b:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 422
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ref3a:order-value                          |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ref3a                                       |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ref3a:order-value/versions       |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # SAME SUBJECT NAME, DIFFERENT CONTEXTS, INDEPENDENT REFERENCES
  # ==========================================================================

  Scenario: Same reference subject name in different contexts are independent
    # Register Address in context .ref4a with fields (street, city)
    When I POST "/subjects/:.ref4a:address-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "addr_4a_id"
    # Register Address in context .ref4b with DIFFERENT fields (line1, line2, zip)
    When I POST "/subjects/:.ref4b:address-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"line1\",\"type\":\"string\"},{\"name\":\"line2\",\"type\":\"string\"},{\"name\":\"zip\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "addr_4b_id"
    # Register Order in context .ref4a referencing address-value — resolves to (street, city)
    When I POST "/subjects/:.ref4a:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "order_4a_id"
    # Register Order in context .ref4b referencing address-value — resolves to (line1, line2, zip)
    When I POST "/subjects/:.ref4b:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "order_4b_id"
    # Verify each context has its own Address — check the raw schemas
    When I GET "/subjects/:.ref4a:address-value/versions/1/schema"
    Then the response status should be 200
    And the response body should contain "street"
    And the response body should contain "city"
    And the response body should not contain "line1"
    When I GET "/subjects/:.ref4b:address-value/versions/1/schema"
    Then the response status should be 200
    And the response body should contain "line1"
    And the response body should contain "zip"
    And the response body should not contain "street"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ref4b:order-value                          |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ref4b                                       |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ref4b:order-value/versions       |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # RAW SCHEMA ENDPOINT FOR SCHEMA WITH REFERENCES
  # ==========================================================================

  Scenario: Raw schema endpoint for schema with references in context
    # Register Address in context .ref5
    When I POST "/subjects/:.ref5:address-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register Order with reference to Address in same context
    When I POST "/subjects/:.ref5:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 200
    # GET the raw schema for order-value — returns the schema JSON string
    When I GET "/subjects/:.ref5:order-value/versions/1/schema"
    Then the response status should be 200
    And the response body should contain "Order"
    And the response body should contain "com.example.Address"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ref5:order-value                           |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ref5                                        |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ref5:order-value/versions        |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # DELETE PROTECTION FOR REFERENCED SCHEMAS IN CONTEXT
  # ==========================================================================

  Scenario: Delete referenced subject is blocked in context
    # Register Address in context .ref6
    When I POST "/subjects/:.ref6:address-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register Order referencing Address in same context
    When I POST "/subjects/:.ref6:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 200
    # Try to delete the referenced Address subject — should be blocked
    When I DELETE "/subjects/:.ref6:address-value"
    Then the response status should be 422
    And the response body should contain "42206"
    # Also try to delete the specific version — should be blocked
    When I DELETE "/subjects/:.ref6:address-value/versions/1"
    Then the response status should be 422
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ref6:order-value                           |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ref6                                        |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ref6:order-value/versions        |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # PROTOBUF IMPORT IN SAME CONTEXT
  # ==========================================================================

  Scenario: Protobuf schema with import in same context
    # Register the base proto "common" in context .ref7
    When I POST "/subjects/:.ref7:common-proto/versions" with body:
      """
      {"schemaType": "PROTOBUF", "schema": "syntax = \"proto3\";\npackage common;\nmessage Timestamp {\n  int64 seconds = 1;\n  int32 nanos = 2;\n}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "common_id"
    # Register the order proto that imports common-proto in same context
    When I POST "/subjects/:.ref7:order-proto/versions" with body:
      """
      {"schemaType": "PROTOBUF", "schema": "syntax = \"proto3\";\npackage orders;\nimport \"common.proto\";\nmessage Order {\n  int64 id = 1;\n  common.Timestamp created = 2;\n}", "references": [{"name": "common.proto", "subject": "common-proto", "version": 1}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "order_proto_id"
    # Verify the order proto was registered
    When I GET "/subjects/:.ref7:order-proto/versions/1"
    Then the response status should be 200
    And the response field "schemaType" should be "PROTOBUF"
    And the response should contain "references"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ref7:order-proto                           |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | PROTOBUF                                     |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ref7                                        |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ref7:order-proto/versions        |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |

  # ==========================================================================
  # SCHEMA VERSION DETAIL INCLUDES REFERENCES FIELD
  # ==========================================================================

  Scenario: Schema version detail includes references field for schema with references
    # Register Address in context .ref8
    When I POST "/subjects/:.ref8:address-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "addr_id_ref8"
    # Register Order with reference in same context
    When I POST "/subjects/:.ref8:order-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}", "references": [{"name": "com.example.Address", "subject": "address-value", "version": 1}]}
      """
    Then the response status should be 200
    And I store the response field "id" as "order_id_ref8"
    # GET the version detail — should include the references array
    When I GET "/subjects/:.ref8:order-value/versions/1"
    Then the response status should be 200
    And the response should contain "references"
    And the response should contain "com.example.Address"
    And the response should contain "address-value"
    And the response field "version" should be 1
    And the response field "subject" should be "order-value"
    And the audit log should contain an event:
      | event_type           | schema_register                              |
      | outcome              | success                                      |
      | actor_id             |                                              |
      | actor_type           | anonymous                                    |
      | auth_method          |                                              |
      | role                 |                                              |
      | target_type          | subject                                      |
      | target_id            | :.ref8:order-value                           |
      | schema_id            | *                                            |
      | version              | *                                            |
      | schema_type          | AVRO                                         |
      | before_hash          |                                              |
      | after_hash           | sha256:*                                     |
      | context              | .ref8                                        |
      | transport_security   | tls                                          |
      | method               | POST                                         |
      | path                 | /subjects/:.ref8:order-value/versions        |
      | status_code          | 200                                          |
      | reason               |                                              |
      | error                |                                              |
      | request_body         |                                              |
      | metadata             |                                              |
      | timestamp            | *                                            |
      | duration_ms          | *                                            |
      | request_id           | *                                            |
      | source_ip            | *                                            |
      | user_agent           | *                                            |
