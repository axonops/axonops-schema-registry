@mcp @schema-modeling @ai
Feature: MCP AI Data Modeling — Event-Driven Architecture
  An AI agent designs an event-driven microservices architecture using MCP
  tools. The AI models domain events as Avro schemas for Kafka topics,
  evolves them while maintaining backward compatibility, and verifies its
  work through schema introspection.

  # ==========================================================================
  # 1. AI DESIGNS A USER SERVICE EVENT STREAM
  # ==========================================================================

  Scenario: AI models a user registration event and evolves it safely
    # AI first configures compatibility for the topic
    When I call MCP tool "set_config" with input:
      | subject             | user-events-value |
      | compatibility_level | BACKWARD          |
    Then the MCP result should contain "BACKWARD"
    # AI designs the initial UserRegistered event
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "user-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"UserRegistered\",\"namespace\":\"com.platform.events\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}"
      }
      """
    Then the MCP result should contain "user-events-value"
    And the MCP result should contain "\"version\":1"
    # AI verifies the registered schema
    When I call MCP tool "get_latest_schema" with input:
      | subject | user-events-value |
    Then the MCP result should contain "UserRegistered"
    And the MCP result should contain "user_id"
    And the MCP result should contain "timestamp-millis"
    # AI checks compatibility before evolving — adds optional display_name
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "user-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"UserRegistered\",\"namespace\":\"com.platform.events\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"display_name\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "true"
    # AI registers the evolved schema
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "user-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"UserRegistered\",\"namespace\":\"com.platform.events\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"display_name\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":2"
    # AI verifies version history
    When I call MCP tool "list_versions" with input:
      | subject | user-events-value |
    Then the MCP result should contain "1"
    And the MCP result should contain "2"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | user-events-value      |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | list_versions          |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 2. AI DETECTS AND RECOVERS FROM INCOMPATIBLE CHANGE
  # ==========================================================================

  Scenario: AI detects incompatible change and adjusts schema design
    # AI sets up the order event topic with strict compatibility
    When I call MCP tool "set_config" with input:
      | subject             | order-events-value |
      | compatibility_level | BACKWARD           |
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "order-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"OrderCreated\",\"namespace\":\"com.platform.events\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"customer_id\",\"type\":\"string\"},{\"name\":\"total\",\"type\":\"double\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI tries to add a required field (no default) — this will be incompatible
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "order-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"OrderCreated\",\"namespace\":\"com.platform.events\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"customer_id\",\"type\":\"string\"},{\"name\":\"total\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\"}]}"
      }
      """
    Then the MCP result should contain "false"
    # AI corrects its design — adds field with default value instead
    When I call MCP tool "check_compatibility" with JSON input:
      """
      {
        "subject": "order-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"OrderCreated\",\"namespace\":\"com.platform.events\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"customer_id\",\"type\":\"string\"},{\"name\":\"total\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\",\"default\":\"USD\"}]}"
      }
      """
    Then the MCP result should contain "true"
    # AI registers the corrected evolution
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "order-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"OrderCreated\",\"namespace\":\"com.platform.events\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"customer_id\",\"type\":\"string\"},{\"name\":\"total\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\",\"default\":\"USD\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":2"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | subject                |
      | target_id            | order-events-value     |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | register_schema        |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |

  # ==========================================================================
  # 3. AI MODELS MULTIPLE EVENTS IN A DOMAIN
  # ==========================================================================

  Scenario: AI designs a complete payment domain with three event types
    # AI configures all payment topics
    When I call MCP tool "set_config" with input:
      | subject             | payment-initiated-value |
      | compatibility_level | BACKWARD_TRANSITIVE     |
    When I call MCP tool "set_config" with input:
      | subject             | payment-completed-value |
      | compatibility_level | BACKWARD_TRANSITIVE     |
    When I call MCP tool "set_config" with input:
      | subject             | payment-failed-value |
      | compatibility_level | BACKWARD_TRANSITIVE  |
    # AI registers PaymentInitiated event
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "payment-initiated-value",
        "schema": "{\"type\":\"record\",\"name\":\"PaymentInitiated\",\"namespace\":\"com.platform.payments\",\"fields\":[{\"name\":\"payment_id\",\"type\":\"string\"},{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":{\"type\":\"bytes\",\"logicalType\":\"decimal\",\"precision\":12,\"scale\":2}},{\"name\":\"currency\",\"type\":\"string\"},{\"name\":\"method\",\"type\":{\"type\":\"enum\",\"name\":\"PaymentMethod\",\"symbols\":[\"CARD\",\"BANK_TRANSFER\",\"WALLET\"]}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI registers PaymentCompleted event
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "payment-completed-value",
        "schema": "{\"type\":\"record\",\"name\":\"PaymentCompleted\",\"namespace\":\"com.platform.payments\",\"fields\":[{\"name\":\"payment_id\",\"type\":\"string\"},{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":{\"type\":\"bytes\",\"logicalType\":\"decimal\",\"precision\":12,\"scale\":2}},{\"name\":\"completed_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI registers PaymentFailed event
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "payment-failed-value",
        "schema": "{\"type\":\"record\",\"name\":\"PaymentFailed\",\"namespace\":\"com.platform.payments\",\"fields\":[{\"name\":\"payment_id\",\"type\":\"string\"},{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"reason\",\"type\":\"string\"},{\"name\":\"failed_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # AI verifies the full domain by listing subjects
    When I call MCP tool "list_subjects"
    Then the MCP result should contain "payment-initiated-value"
    And the MCP result should contain "payment-completed-value"
    And the MCP result should contain "payment-failed-value"
    # AI inspects the complete domain model
    When I call MCP tool "get_latest_schema" with input:
      | subject | payment-initiated-value |
    Then the MCP result should contain "PaymentInitiated"
    And the MCP result should contain "PaymentMethod"
    And the MCP result should contain "decimal"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | payment-initiated-value    |
      | schema_id            |                            |
      | version              |                            |
      | schema_type          |                            |
      | before_hash          |                            |
      | after_hash           |                            |
      | context              |                            |
      | transport_security   |                            |
      | source_ip            |                            |
      | user_agent           |                            |
      | method               | MCP                        |
      | path                 | get_latest_schema          |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |

  # ==========================================================================
  # 4. AI MODELS MULTI-VERSION EVOLUTION CHAIN
  # ==========================================================================

  Scenario: AI evolves a schema through 4 versions under BACKWARD_TRANSITIVE
    When I call MCP tool "set_config" with input:
      | subject             | inventory-events-value |
      | compatibility_level | BACKWARD_TRANSITIVE    |
    # v1: Initial inventory event
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "inventory-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"InventoryChanged\",\"namespace\":\"com.warehouse\",\"fields\":[{\"name\":\"sku\",\"type\":\"string\"},{\"name\":\"quantity\",\"type\":\"int\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}"
      }
      """
    Then the MCP result should contain "\"version\":1"
    # v2: Add optional warehouse location
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "inventory-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"InventoryChanged\",\"namespace\":\"com.warehouse\",\"fields\":[{\"name\":\"sku\",\"type\":\"string\"},{\"name\":\"quantity\",\"type\":\"int\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"warehouse\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":2"
    # v3: Add optional reason enum
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "inventory-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"InventoryChanged\",\"namespace\":\"com.warehouse\",\"fields\":[{\"name\":\"sku\",\"type\":\"string\"},{\"name\":\"quantity\",\"type\":\"int\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"warehouse\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"reason\",\"type\":{\"type\":\"enum\",\"name\":\"ChangeReason\",\"symbols\":[\"SALE\",\"RESTOCK\",\"ADJUSTMENT\",\"RETURN\"]},\"default\":\"ADJUSTMENT\"}]}"
      }
      """
    Then the MCP result should contain "\"version\":3"
    # v4: Add optional batch_id
    When I call MCP tool "register_schema" with JSON input:
      """
      {
        "subject": "inventory-events-value",
        "schema": "{\"type\":\"record\",\"name\":\"InventoryChanged\",\"namespace\":\"com.warehouse\",\"fields\":[{\"name\":\"sku\",\"type\":\"string\"},{\"name\":\"quantity\",\"type\":\"int\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}},{\"name\":\"warehouse\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"reason\",\"type\":{\"type\":\"enum\",\"name\":\"ChangeReason\",\"symbols\":[\"SALE\",\"RESTOCK\",\"ADJUSTMENT\",\"RETURN\"]},\"default\":\"ADJUSTMENT\"},{\"name\":\"batch_id\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
      }
      """
    Then the MCP result should contain "\"version\":4"
    # AI verifies the full version history
    When I call MCP tool "list_versions" with input:
      | subject | inventory-events-value |
    Then the MCP result should contain "1"
    And the MCP result should contain "2"
    And the MCP result should contain "3"
    And the MCP result should contain "4"
    # AI retrieves each version to inspect evolution
    When I call MCP tool "get_schema_version" with input:
      | subject | inventory-events-value |
      | version | 1                      |
    Then the MCP result should contain "InventoryChanged"
    And the MCP result should not contain "\"name\":\"warehouse\""
    When I call MCP tool "get_schema_version" with input:
      | subject | inventory-events-value |
      | version | 4                      |
    Then the MCP result should contain "batch_id"
    And the MCP result should contain "ChangeReason"
    And the audit log should contain an event:
      | event_type           | mcp_tool_call              |
      | outcome              | success                    |
      | actor_id             | mcp-anonymous              |
      | actor_type           | anonymous                  |
      | auth_method          |                            |
      | role                 |                            |
      | target_type          | subject                    |
      | target_id            | inventory-events-value     |
      | schema_id            |                            |
      | version              |                            |
      | schema_type          |                            |
      | before_hash          |                            |
      | after_hash           |                            |
      | context              |                            |
      | transport_security   |                            |
      | source_ip            |                            |
      | user_agent           |                            |
      | method               | MCP                        |
      | path                 | get_schema_version         |
      | status_code          | 0                          |
      | reason               |                            |
      | error                |                            |
      | request_body         |                            |
      | metadata             |                            |
      | timestamp            | *                          |
      | duration_ms          | *                          |
      | request_id           |                            |
