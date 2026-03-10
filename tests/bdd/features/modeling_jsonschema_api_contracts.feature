@schema-modeling @json @domain
Feature: JSON Schema API Contract Domain Modeling
  Real-world API contract schemas exercising multi-version evolution
  with closed content models, $ref composition, discriminated unions,
  constraint relaxation chains, and cross-subject references.

  # ==========================================================================
  # 1. API REQUEST SCHEMA EVOLVES 4 VERSIONS (CLOSED MODEL)
  # ==========================================================================

  Scenario: API request schema evolves 4 versions under BACKWARD_TRANSITIVE
    Given subject "api-request" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "api-request" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "api-request":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "api-request":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "api-request":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"},"address":{"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}}}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "api-request"

  # ==========================================================================
  # 2. RESPONSE SCHEMA WITH $DEFS + ALLOF COMPOSITION
  # ==========================================================================

  Scenario: Response schema with definitions and allOf composition
    When I register a "JSON" schema under subject "api-response":
      """
      {"type":"object","definitions":{"pagination":{"type":"object","properties":{"page":{"type":"integer"},"total":{"type":"integer"}}}},"allOf":[{"$ref":"#/definitions/pagination"}],"properties":{"items":{"type":"array","items":{"type":"object"}}}}
      """
    Then the response status should be 200
    When I get version 1 of subject "api-response"
    Then the response status should be 200
    And the response body should contain "pagination"
    And the response body should contain "allOf"
    And the audit log should contain event "schema_register" with subject "api-response"

  # ==========================================================================
  # 3. DISCRIMINATED UNION EVOLVES — ADD PAYMENT VARIANT
  # ==========================================================================

  Scenario: Discriminated union evolves by adding payment variant
    Given subject "api-payment-union" has compatibility level "BACKWARD"
    And subject "api-payment-union" has "JSON" schema:
      """
      {"oneOf":[{"type":"object","properties":{"method":{"const":"credit_card"},"card_number":{"type":"string"}},"required":["method","card_number"]},{"type":"object","properties":{"method":{"const":"bank_transfer"},"iban":{"type":"string"}},"required":["method","iban"]}]}
      """
    When I register a "JSON" schema under subject "api-payment-union":
      """
      {"oneOf":[{"type":"object","properties":{"method":{"const":"credit_card"},"card_number":{"type":"string"}},"required":["method","card_number"]},{"type":"object","properties":{"method":{"const":"bank_transfer"},"iban":{"type":"string"}},"required":["method","iban"]},{"type":"object","properties":{"method":{"const":"paypal"},"email":{"type":"string"}},"required":["method","email"]}]}
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "api-payment-union"

  # ==========================================================================
  # 4. CONSTRAINT RELAXATION CHAIN
  # ==========================================================================

  Scenario: String constraint relaxation chain under BACKWARD_TRANSITIVE
    Given subject "api-constraint-chain" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "api-constraint-chain" has "JSON" schema:
      """
      {"type":"string","maxLength":50}
      """
    When I register a "JSON" schema under subject "api-constraint-chain":
      """
      {"type":"string","maxLength":100}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "api-constraint-chain":
      """
      {"type":"string","maxLength":200}
      """
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "api-constraint-chain"

  # ==========================================================================
  # 5. ADDING REQUIRED FIELD BREAKS BACKWARD_TRANSITIVE
  # ==========================================================================

  Scenario: Adding required field breaks BACKWARD_TRANSITIVE against v1
    Given subject "api-break-trans" has compatibility level "BACKWARD_TRANSITIVE"
    And subject "api-break-trans" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "api-break-trans":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "api-break-trans":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"],"additionalProperties":false}
      """
    Then the response status should be 409

  # ==========================================================================
  # 6. CONTENT VERIFICATION
  # ==========================================================================

  Scenario: Content verification for complex JSON Schema
    Given subject "api-content-verify" has "JSON" schema:
      """
      {"type":"object","definitions":{"address":{"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"},"zip":{"type":"string"}},"required":["street","city"]}},"properties":{"name":{"type":"string"},"billing":{"$ref":"#/definitions/address"},"shipping":{"$ref":"#/definitions/address"}},"required":["name","billing"]}
      """
    When I get version 1 of subject "api-content-verify"
    Then the response status should be 200
    And the response body should contain "properties"
    And the response body should contain "required"
    And the response body should contain "definitions"
    And the response body should contain "address"

  # ==========================================================================
  # 7. JSON KEY REORDERING DEDUPLICATION
  # ==========================================================================

  @axonops-only
  Scenario: Same logical schema with different key ordering produces same ID
    When I register a "JSON" schema under subject "api-keyorder-a":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "keyorder_id"
    When I register a "JSON" schema under subject "api-keyorder-b":
      """
      {"required":["name"],"type":"object","properties":{"age":{"type":"integer"},"name":{"type":"string"}}}
      """
    Then the response status should be 200
    And the response field "id" should equal stored "keyorder_id"
    And the audit log should contain event "schema_register" with subject "api-keyorder-b"

  # ==========================================================================
  # 8. CROSS-SUBJECT REFERENCES
  # ==========================================================================

  Scenario: Cross-subject JSON Schema references with referencedby tracking
    Given subject "api-ref-address" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"},"zip":{"type":"string"}},"required":["street","city"]}
      """
    When I register a "JSON" schema under subject "api-ref-person" with references:
      """
      {
        "schemaType": "JSON",
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"home\":{\"$ref\":\"address.json\"}},\"required\":[\"name\"]}",
        "references": [
          {"name":"address.json","subject":"api-ref-address","version":1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "api-ref-address" version 1
    Then the response status should be 200
    And the audit log should contain event "schema_register" with subject "api-ref-person"
