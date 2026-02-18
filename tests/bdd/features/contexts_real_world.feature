@functional
Feature: Contexts — Real-World Usage Patterns
  These scenarios simulate actual multi-team, multi-environment, and schema-linking
  usage patterns based on Confluent Schema Registry documentation for contexts.
  Each scenario represents a real-world situation where context isolation is critical.

  Background:
    Given the schema registry is running

  # ==========================================================================
  # SCENARIO 1: Multi-team isolation — Team Alpha and Team Bravo
  # ==========================================================================

  Scenario: Multi-team isolation with same subject name but different schemas
    # Team Alpha registers their user-events schema
    When I POST "/subjects/:.team-alpha:user-events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UserEvents\",\"namespace\":\"com.rw.scenario1.alpha\",\"fields\":[{\"name\":\"userId\",\"type\":\"long\"},{\"name\":\"action\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "alpha_id"
    # Team Bravo registers their user-events schema — completely different fields
    When I POST "/subjects/:.team-bravo:user-events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"UserEvents\",\"namespace\":\"com.rw.scenario1.bravo\",\"fields\":[{\"name\":\"uid\",\"type\":\"string\"},{\"name\":\"eventType\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "bravo_id"
    # Both contexts assign schema ID 1 independently
    And the response field "id" should be 1
    # Team Alpha also got ID 1
    Then the response field "id" should equal stored "alpha_id"
    # Verify versions for Team Alpha
    When I GET "/subjects/:.team-alpha:user-events/versions"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain integer 1
    # Verify versions for Team Bravo
    When I GET "/subjects/:.team-bravo:user-events/versions"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain integer 1
    # Root subjects should NOT contain user-events (both are in named contexts)
    When I GET "/subjects"
    Then the response status should be 200
    And the response body should not contain "user-events"
    # Contexts list includes both teams and the default
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should contain ".team-alpha"
    And the response array should contain ".team-bravo"

  # ==========================================================================
  # SCENARIO 2: Environment separation — dev, staging, prod
  # ==========================================================================

  Scenario: Environment separation with independent schema evolution
    # Dev: simple 2-field payment schema
    When I POST "/subjects/:.dev:payment-events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"PaymentEvents\",\"namespace\":\"com.rw.scenario2\",\"fields\":[{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "dev_id"
    # Staging: evolved 3-field schema (added timestamp)
    When I POST "/subjects/:.staging:payment-events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"PaymentEvents\",\"namespace\":\"com.rw.scenario2.stg\",\"fields\":[{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\"},{\"name\":\"processedAt\",\"type\":\"long\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "staging_id"
    # Prod: same simple schema as dev
    When I POST "/subjects/:.prod:payment-events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"PaymentEvents\",\"namespace\":\"com.rw.scenario2.prd\",\"fields\":[{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"currency\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "prod_id"
    # Verify contexts are sorted alphabetically
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should contain ".dev"
    And the response array should contain ".prod"
    And the response array should contain ".staging"
    # Each context has independent IDs (all got ID 1 in their respective context)
    When I GET "/subjects/:.dev:payment-events/versions/1"
    Then the response status should be 200
    And the response field "version" should be 1
    When I GET "/subjects/:.staging:payment-events/versions/1"
    Then the response status should be 200
    And the response field "version" should be 1
    When I GET "/subjects/:.prod:payment-events/versions/1"
    Then the response status should be 200
    And the response field "version" should be 1

  # ==========================================================================
  # SCENARIO 3: Schema linking simulation — source and destination clusters
  # ==========================================================================

  Scenario: Schema linking between source and destination clusters
    # Source cluster: register product-value v1
    When I POST "/subjects/:.source-cluster:product-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Product\",\"namespace\":\"com.rw.scenario3\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "src_v1_id"
    # Source cluster: evolve to v2 (add optional description field)
    When I POST "/subjects/:.source-cluster:product-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Product\",\"namespace\":\"com.rw.scenario3\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"description\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "src_v2_id"
    # Destination cluster: link only v1 (replication lag / selective linking)
    When I POST "/subjects/:.dest-cluster:product-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Product\",\"namespace\":\"com.rw.scenario3\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "dest_v1_id"
    # Source has 2 versions
    When I GET "/subjects/:.source-cluster:product-value/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    And the response array should contain integer 1
    And the response array should contain integer 2
    # Destination has only 1 version
    When I GET "/subjects/:.dest-cluster:product-value/versions"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain integer 1
    # Schema IDs are independent in each context
    When I GET "/subjects/:.source-cluster:product-value/versions/1"
    Then the response status should be 200
    And I store the response field "id" as "src_schema_id"
    When I GET "/subjects/:.dest-cluster:product-value/versions/1"
    Then the response status should be 200
    And I store the response field "id" as "dest_schema_id"

  # ==========================================================================
  # SCENARIO 4: Hierarchical context naming with dots
  # ==========================================================================

  Scenario: Hierarchical dot-separated context names are independent
    # Organization team1 payments
    When I POST "/subjects/:.org.team1.payments:transactions/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Transactions\",\"namespace\":\"com.rw.scenario4.team1\",\"fields\":[{\"name\":\"txnId\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    # Organization team2 payments — same subject name, different schema
    When I POST "/subjects/:.org.team2.payments:transactions/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Transactions\",\"namespace\":\"com.rw.scenario4.team2\",\"fields\":[{\"name\":\"transactionRef\",\"type\":\"long\"},{\"name\":\"total\",\"type\":\"double\"},{\"name\":\"status\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    # Both contexts are fully independent (no parent-child relationship)
    When I GET "/subjects/:.org.team1.payments:transactions/versions/1"
    Then the response status should be 200
    And the response body should contain "txnId"
    And the response body should not contain "transactionRef"
    When I GET "/subjects/:.org.team2.payments:transactions/versions/1"
    Then the response status should be 200
    And the response body should contain "transactionRef"
    And the response body should not contain "txnId"
    # GET /contexts includes both hierarchical names
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".org.team1.payments"
    And the response array should contain ".org.team2.payments"

  # ==========================================================================
  # SCENARIO 5: Default context coexists with named contexts
  # ==========================================================================

  Scenario: Default context coexists with named contexts independently
    # Register in default context (no prefix)
    When I POST "/subjects/orders-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Orders\",\"namespace\":\"com.rw.scenario5.default\",\"fields\":[{\"name\":\"orderId\",\"type\":\"long\"},{\"name\":\"item\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "default_order_id"
    # Register in .production context — different schema, same subject name
    When I POST "/subjects/:.production:orders-value/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Orders\",\"namespace\":\"com.rw.scenario5.prod\",\"fields\":[{\"name\":\"orderId\",\"type\":\"long\"},{\"name\":\"item\",\"type\":\"string\"},{\"name\":\"warehouse\",\"type\":\"string\",\"default\":\"default\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "prod_order_id"
    # GET /subjects at root returns only default context subjects
    When I GET "/subjects"
    Then the response status should be 200
    And the response array should contain "orders-value"
    # Default context schema — has orderId and item, no warehouse
    When I GET "/subjects/orders-value/versions/1"
    Then the response status should be 200
    And the response body should contain "orderId"
    And the response body should not contain "warehouse"
    # Production context schema — has warehouse field
    When I GET "/subjects/:.production:orders-value/versions/1"
    Then the response status should be 200
    And the response body should contain "warehouse"
    # Schemas are different, proving isolation
    And the response body should contain "com.rw.scenario5.prod"

  # ==========================================================================
  # SCENARIO 6: Large number of contexts
  # ==========================================================================

  Scenario: Large number of contexts are tracked and sorted correctly
    # Create schemas in 10 different contexts
    When I POST "/subjects/:.ctx01:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D01\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx02:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D02\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx03:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D03\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx04:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D04\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx05:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D05\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx06:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D06\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx07:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D07\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx08:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D08\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx09:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D09\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.ctx10:data/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"D10\",\"namespace\":\"com.rw.scenario6\",\"fields\":[{\"name\":\"v\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    # GET /contexts returns all 10 + default "." = 11 contexts minimum
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should contain ".ctx01"
    And the response array should contain ".ctx02"
    And the response array should contain ".ctx03"
    And the response array should contain ".ctx04"
    And the response array should contain ".ctx05"
    And the response array should contain ".ctx06"
    And the response array should contain ".ctx07"
    And the response array should contain ".ctx08"
    And the response array should contain ".ctx09"
    And the response array should contain ".ctx10"

  # ==========================================================================
  # SCENARIO 7: Migration from single-tenant to multi-tenant
  # ==========================================================================

  Scenario: Migration from single-tenant to multi-tenant preserves default context
    # Start with schemas in default context (single-tenant mode)
    When I POST "/subjects/user-value-s7/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.rw.scenario7.default\",\"fields\":[{\"name\":\"userId\",\"type\":\"long\"},{\"name\":\"email\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "default_user_id"
    When I POST "/subjects/order-value-s7/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.rw.scenario7.default\",\"fields\":[{\"name\":\"orderId\",\"type\":\"long\"},{\"name\":\"total\",\"type\":\"double\"}]}"}
      """
    Then the response status should be 200
    And I store the response field "id" as "default_order_id"
    # Now add a tenant (multi-tenant migration) — different schemas
    When I POST "/subjects/:.tenant-a:user-value-s7/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.rw.scenario7.tenanta\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"tenant\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.tenant-a:order-value-s7/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.rw.scenario7.tenanta\",\"fields\":[{\"name\":\"orderRef\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"double\"},{\"name\":\"tenant\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Default context schemas are unaffected
    When I GET "/subjects/user-value-s7/versions/1"
    Then the response status should be 200
    And the response body should contain "com.rw.scenario7.default"
    And the response body should contain "email"
    And the response body should not contain "tenant"
    When I GET "/subjects/order-value-s7/versions/1"
    Then the response status should be 200
    And the response body should contain "com.rw.scenario7.default"
    And the response body should contain "total"
    # GET /subjects returns only default context subjects
    When I GET "/subjects"
    Then the response status should be 200
    And the response array should contain "user-value-s7"
    And the response array should contain "order-value-s7"
    # GET /contexts shows both default and tenant
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain "."
    And the response array should contain ".tenant-a"

  # ==========================================================================
  # SCENARIO 8: Context cleanup — delete all subjects in a context
  # ==========================================================================

  Scenario: Context cleanup by deleting all subjects permanently
    # Register two subjects in cleanup context
    When I POST "/subjects/:.cleanup-ctx:events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Events\",\"namespace\":\"com.rw.scenario8\",\"fields\":[{\"name\":\"eventId\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.cleanup-ctx:metrics/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Metrics\",\"namespace\":\"com.rw.scenario8\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"value\",\"type\":\"double\"}]}"}
      """
    Then the response status should be 200
    # Soft-delete first subject
    When I DELETE "/subjects/:.cleanup-ctx:events"
    Then the response status should be 200
    # Permanent delete first subject
    When I DELETE "/subjects/:.cleanup-ctx:events?permanent=true"
    Then the response status should be 200
    # Soft-delete second subject
    When I DELETE "/subjects/:.cleanup-ctx:metrics"
    Then the response status should be 200
    # Permanent delete second subject
    When I DELETE "/subjects/:.cleanup-ctx:metrics?permanent=true"
    Then the response status should be 200
    # Both subjects should now return 404
    When I GET "/subjects/:.cleanup-ctx:events/versions"
    Then the response status should be 404
    When I GET "/subjects/:.cleanup-ctx:metrics/versions"
    Then the response status should be 404
    # Register a default context subject to prove it is unaffected
    When I POST "/subjects/cleanup-proof-s8/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"CleanupProof\",\"namespace\":\"com.rw.scenario8\",\"fields\":[{\"name\":\"ok\",\"type\":\"boolean\"}]}"}
      """
    Then the response status should be 200
    When I GET "/subjects/cleanup-proof-s8/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # SCENARIO 9: Cross-context schema comparison — different evolution speed
  # ==========================================================================

  Scenario: Cross-context independent evolution speed
    # Fast-evolving context: 3 versions
    When I POST "/subjects/:.fast-evo:analytics/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Analytics\",\"namespace\":\"com.rw.scenario9.fast\",\"fields\":[{\"name\":\"event\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.fast-evo:analytics/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Analytics\",\"namespace\":\"com.rw.scenario9.fast\",\"fields\":[{\"name\":\"event\",\"type\":\"string\"},{\"name\":\"source\",\"type\":\"string\",\"default\":\"unknown\"}]}"}
      """
    Then the response status should be 200
    When I POST "/subjects/:.fast-evo:analytics/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Analytics\",\"namespace\":\"com.rw.scenario9.fast\",\"fields\":[{\"name\":\"event\",\"type\":\"string\"},{\"name\":\"source\",\"type\":\"string\",\"default\":\"unknown\"},{\"name\":\"ts\",\"type\":\"long\",\"default\":0}]}"}
      """
    Then the response status should be 200
    # Slow-evolving context: 1 version
    When I POST "/subjects/:.slow-evo:analytics/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Analytics\",\"namespace\":\"com.rw.scenario9.slow\",\"fields\":[{\"name\":\"event\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Fast context has 3 versions
    When I GET "/subjects/:.fast-evo:analytics/versions"
    Then the response status should be 200
    And the response should be an array of length 3
    And the response array should contain integer 1
    And the response array should contain integer 2
    And the response array should contain integer 3
    # Slow context has only 1 version
    When I GET "/subjects/:.slow-evo:analytics/versions"
    Then the response status should be 200
    And the response should be an array of length 1
    And the response array should contain integer 1
    # Latest version in fast context is 3
    When I GET "/subjects/:.fast-evo:analytics/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 3
    # Latest version in slow context is 1
    When I GET "/subjects/:.slow-evo:analytics/versions/latest"
    Then the response status should be 200
    And the response field "version" should be 1

  # ==========================================================================
  # SCENARIO 10: Config isolation — real-world compat policies
  # ==========================================================================

  Scenario: Config isolation enforces different compatibility policies per context
    # Strict context: FULL compatibility
    When I POST "/subjects/:.strict-ctx:inventory/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Inventory\",\"namespace\":\"com.rw.scenario10.strict\",\"fields\":[{\"name\":\"sku\",\"type\":\"string\"},{\"name\":\"qty\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.strict-ctx:inventory" with body:
      """
      {"compatibility": "FULL"}
      """
    Then the response status should be 200
    # Relaxed context: NONE compatibility
    When I POST "/subjects/:.relaxed-ctx:inventory/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Inventory\",\"namespace\":\"com.rw.scenario10.relaxed\",\"fields\":[{\"name\":\"sku\",\"type\":\"string\"},{\"name\":\"qty\",\"type\":\"int\"}]}"}
      """
    Then the response status should be 200
    When I PUT "/config/:.relaxed-ctx:inventory" with body:
      """
      {"compatibility": "NONE"}
      """
    Then the response status should be 200
    # Breaking change: change field type from int to string
    # Strict context rejects it (FULL compatibility violation)
    When I POST "/subjects/:.strict-ctx:inventory/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Inventory\",\"namespace\":\"com.rw.scenario10.strict\",\"fields\":[{\"name\":\"sku\",\"type\":\"string\"},{\"name\":\"qty\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 409
    # Relaxed context accepts it (NONE allows any change)
    When I POST "/subjects/:.relaxed-ctx:inventory/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Inventory\",\"namespace\":\"com.rw.scenario10.relaxed\",\"fields\":[{\"name\":\"sku\",\"type\":\"string\"},{\"name\":\"qty\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Verify strict context still has only 1 version
    When I GET "/subjects/:.strict-ctx:inventory/versions"
    Then the response status should be 200
    And the response should be an array of length 1
    # Verify relaxed context has 2 versions
    When I GET "/subjects/:.relaxed-ctx:inventory/versions"
    Then the response status should be 200
    And the response should be an array of length 2

  # ==========================================================================
  # SCENARIO 11: Mode isolation — read-only production, writable staging
  # ==========================================================================

  Scenario: Mode isolation between read-only production and writable staging
    # Register base schema in staging
    When I POST "/subjects/:.rw-staging:events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Events\",\"namespace\":\"com.rw.scenario11.staging\",\"fields\":[{\"name\":\"eventId\",\"type\":\"string\"},{\"name\":\"payload\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Register same base schema in production
    When I POST "/subjects/:.ro-prod:events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Events\",\"namespace\":\"com.rw.scenario11.prod\",\"fields\":[{\"name\":\"eventId\",\"type\":\"string\"},{\"name\":\"payload\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    # Set production subject to READONLY
    When I PUT "/mode/:.ro-prod:events" with body:
      """
      {"mode": "READONLY"}
      """
    Then the response status should be 200
    # Try to register new version in production — rejected (READONLY)
    When I POST "/subjects/:.ro-prod:events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Events\",\"namespace\":\"com.rw.scenario11.prod\",\"fields\":[{\"name\":\"eventId\",\"type\":\"string\"},{\"name\":\"payload\",\"type\":\"string\"},{\"name\":\"source\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 422
    # Register new version in staging — succeeds (READWRITE)
    When I POST "/subjects/:.rw-staging:events/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"Events\",\"namespace\":\"com.rw.scenario11.staging\",\"fields\":[{\"name\":\"eventId\",\"type\":\"string\"},{\"name\":\"payload\",\"type\":\"string\"},{\"name\":\"source\",\"type\":\"string\",\"default\":\"\"}]}"}
      """
    Then the response status should be 200
    # Staging now has 2 versions
    When I GET "/subjects/:.rw-staging:events/versions"
    Then the response status should be 200
    And the response should be an array of length 2
    # Production still has only 1 version
    When I GET "/subjects/:.ro-prod:events/versions"
    Then the response status should be 200
    And the response should be an array of length 1

  # ==========================================================================
  # SCENARIO 12: Context with all valid character types in name
  # ==========================================================================

  Scenario: Context name with all valid character types
    # Mix of uppercase, lowercase, dash, underscore, dot, numbers
    When I POST "/subjects/:.My-Context_v2.1:test-subject/versions" with body:
      """
      {"schema": "{\"type\":\"record\",\"name\":\"ValidChars\",\"namespace\":\"com.rw.scenario12\",\"fields\":[{\"name\":\"data\",\"type\":\"string\"}]}"}
      """
    Then the response status should be 200
    And the response field "id" should be 1
    # Verify schema is retrievable
    When I GET "/subjects/:.My-Context_v2.1:test-subject/versions/1"
    Then the response status should be 200
    And the response body should contain "ValidChars"
    And the response body should contain "com.rw.scenario12"
    # GET /contexts includes the context with special characters
    When I GET "/contexts"
    Then the response status should be 200
    And the response array should contain ".My-Context_v2.1"
