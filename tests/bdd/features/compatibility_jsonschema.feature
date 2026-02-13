@functional @compatibility
Feature: JSON Schema Compatibility
  Exhaustive JSON Schema compatibility tests across all seven modes

  # ============================================================================
  # BACKWARD mode (8 scenarios)
  # New schema (reader) must be able to read data written by old schema (writer)
  # ============================================================================

  Scenario: BACKWARD - add optional property to open content model is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-back-1":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - add required property is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-back-2":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - remove property from open content model is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-back-3":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - make optional property required is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-4" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-back-4":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - widen type union is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-5" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-back-5":
      """
      {"type":"object","properties":{"value":{"type":["string","null"]}},"required":["value"]}
      """
    Then the response status should be 200

  Scenario: BACKWARD - narrow type union is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-6" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":["string","null"]}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-back-6":
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    Then the response status should be 409

  Scenario: BACKWARD - loosen array minItems is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-7" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    When I register a "JSON" schema under subject "json-back-7":
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    Then the response status should be 200

  Scenario: BACKWARD - tighten array minItems is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-back-8" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    When I register a "JSON" schema under subject "json-back-8":
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    Then the response status should be 409

  # ============================================================================
  # BACKWARD_TRANSITIVE mode (5 scenarios)
  # New schema must be compatible with ALL previous versions
  # ============================================================================

  Scenario: BACKWARD_TRANSITIVE - 3-version chain all compatible (closed model)
    # With additionalProperties:false (closed content model), adding optional
    # properties is backward-compatible because the old writer couldn't have
    # produced data with the new property name.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    And subject "json-bt-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-bt-1":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - v3 adds required property absent in v1
    # Register v1 and v2 under NONE to avoid open-model incompatibility on v2.
    # Then switch to BACKWARD_TRANSITIVE for v3 which adds required "age" — fails vs v1.
    Given the global compatibility level is "NONE"
    And subject "json-bt-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    And subject "json-bt-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    And the global compatibility level is "BACKWARD_TRANSITIVE"
    When I register a "JSON" schema under subject "json-bt-2":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    Then the response status should be 409

  Scenario: BACKWARD_TRANSITIVE - loosen minItems chain is compatible
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-3" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":10}
      """
    And subject "json-bt-3" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    When I register a "JSON" schema under subject "json-bt-3":
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - v3 tightens minItems vs v1
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    And subject "json-bt-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    When I register a "JSON" schema under subject "json-bt-4":
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    Then the response status should be 409

  Scenario: BACKWARD_TRANSITIVE - optional property additions chain is compatible (closed model)
    # With closed content model, adding optional properties is backward-compatible.
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "json-bt-5" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"}},"required":["id"],"additionalProperties":false}
      """
    And subject "json-bt-5" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-bt-5":
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"},"email":{"type":"string"}},"required":["id"],"additionalProperties":false}
      """
    Then the response status should be 200

  # ============================================================================
  # FORWARD mode (8 scenarios)
  # Old schema (reader) must be able to read data written by new schema (writer)
  # ============================================================================

  Scenario: FORWARD - remove optional property with open content model is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-fwd-1":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: FORWARD - remove required property from new is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    When I register a "JSON" schema under subject "json-fwd-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: FORWARD - make required property optional in new is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    When I register a "JSON" schema under subject "json-fwd-3":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: FORWARD - tighten array minItems in new is compatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    When I register a "JSON" schema under subject "json-fwd-4":
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    Then the response status should be 200

  Scenario: FORWARD - loosen array minItems in new is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-5" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    When I register a "JSON" schema under subject "json-fwd-5":
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    Then the response status should be 409

  Scenario: FORWARD - add enum value in new is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-6" has "JSON" schema:
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive"]}},"required":["status"]}
      """
    When I register a "JSON" schema under subject "json-fwd-6":
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive","pending"]}},"required":["status"]}
      """
    Then the response status should be 409

  Scenario: FORWARD - remove enum value in new is compatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-7" has "JSON" schema:
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive","pending"]}},"required":["status"]}
      """
    When I register a "JSON" schema under subject "json-fwd-7":
      """
      {"type":"object","properties":{"status":{"type":"string","enum":["active","inactive"]}},"required":["status"]}
      """
    Then the response status should be 200

  Scenario: FORWARD - identical schema is compatible
    Given the global compatibility level is "FORWARD"
    And subject "json-fwd-8" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-fwd-8":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200

  # ============================================================================
  # FORWARD_TRANSITIVE mode (4 scenarios)
  # ALL previous schemas must be able to read data from new schema
  # ============================================================================

  Scenario: FORWARD_TRANSITIVE - 3-version compatible chain (closed model)
    # With closed content model, properties in reader(old) not in writer(new) are
    # compatible because the old writer couldn't produce those properties.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    And subject "json-ft-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-ft-1":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 200

  Scenario: FORWARD_TRANSITIVE - removing required property in chain fails (closed model)
    # Removing a required property makes old reader unable to find expected data.
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"},"phone":{"type":"string"}},"required":["name","email"],"additionalProperties":false}
      """
    And subject "json-ft-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"],"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-ft-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE - making required optional in chain fails
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    And subject "json-ft-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    When I register a "JSON" schema under subject "json-ft-3":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE - tighten constraints chain is compatible
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "json-ft-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    And subject "json-ft-4" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":3}
      """
    When I register a "JSON" schema under subject "json-ft-4":
      """
      {"type":"array","items":{"type":"string"},"minItems":5}
      """
    Then the response status should be 200

  # ============================================================================
  # FULL mode (7 scenarios)
  # Must be both backward AND forward compatible
  # ============================================================================

  Scenario: FULL - add optional property is incompatible (fails forward check)
    Given the global compatibility level is "FULL"
    And subject "json-full-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full-1":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: FULL - add required property is incompatible
    Given the global compatibility level is "FULL"
    And subject "json-full-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full-2":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the response status should be 409

  Scenario: FULL - remove property is incompatible
    Given the global compatibility level is "FULL"
    And subject "json-full-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full-3":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: FULL - make required to optional is incompatible (fails forward)
    Given the global compatibility level is "FULL"
    And subject "json-full-3b" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    When I register a "JSON" schema under subject "json-full-3b":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: FULL - type change in both directions is incompatible
    Given the global compatibility level is "FULL"
    And subject "json-full-4" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-full-4":
      """
      {"type":"object","properties":{"value":{"type":"integer"}},"required":["value"]}
      """
    Then the response status should be 409

  Scenario: FULL - additionalProperties true to false is incompatible
    Given the global compatibility level is "FULL"
    And subject "json-full-5" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    When I register a "JSON" schema under subject "json-full-5":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 409

  Scenario: FULL - identical schema is compatible
    Given the global compatibility level is "FULL"
    And subject "json-full-6" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-full-6":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200

  # ============================================================================
  # FULL_TRANSITIVE mode (3 scenarios)
  # Must be both backward AND forward compatible with ALL previous versions
  # ============================================================================

  Scenario: FULL_TRANSITIVE - safe 3-version evolution is compatible
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "json-flt-1" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"]}
      """
    And subject "json-flt-1" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-flt-1":
      """
      {"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"}},"required":["id"]}
      """
    Then the response status should be 200

  Scenario: FULL_TRANSITIVE - removing property fails across versions
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "json-flt-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    And subject "json-flt-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-flt-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    Then the response status should be 409

  Scenario: FULL_TRANSITIVE - identical schemas across chain is compatible
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "json-flt-3" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"value":{"type":"integer"}},"required":["id"]}
      """
    And subject "json-flt-3" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"value":{"type":"integer"}},"required":["id"]}
      """
    When I register a "JSON" schema under subject "json-flt-3":
      """
      {"type":"object","properties":{"id":{"type":"string"},"value":{"type":"integer"}},"required":["id"]}
      """
    Then the response status should be 200

  # ============================================================================
  # NONE mode (2 scenarios)
  # No compatibility checks, any change allowed
  # ============================================================================

  Scenario: NONE - complete restructure is allowed
    Given the global compatibility level is "NONE"
    And subject "json-none-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    When I register a "JSON" schema under subject "json-none-1":
      """
      {"type":"array","items":{"type":"number"},"minItems":1}
      """
    Then the response status should be 200

  Scenario: NONE - root type change is allowed
    Given the global compatibility level is "NONE"
    And subject "json-none-2" has "JSON" schema:
      """
      {"type":"string","enum":["a","b","c"]}
      """
    When I register a "JSON" schema under subject "json-none-2":
      """
      {"type":"integer","minimum":0,"maximum":100}
      """
    Then the response status should be 200

  # ============================================================================
  # Edge Cases (12 scenarios)
  # ============================================================================

  Scenario: Edge - additionalProperties true to false is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    When I register a "JSON" schema under subject "json-edge-1":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 409

  Scenario: Edge - additionalProperties false to true is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-edge-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    Then the response status should be 200

  Scenario: Edge - enum value addition is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-3" has "JSON" schema:
      """
      {"type":"object","properties":{"color":{"type":"string","enum":["red","green"]}},"required":["color"]}
      """
    When I register a "JSON" schema under subject "json-edge-3":
      """
      {"type":"object","properties":{"color":{"type":"string","enum":["red","green","blue"]}},"required":["color"]}
      """
    Then the response status should be 200

  Scenario: Edge - enum value removal is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-4" has "JSON" schema:
      """
      {"type":"object","properties":{"color":{"type":"string","enum":["red","green","blue"]}},"required":["color"]}
      """
    When I register a "JSON" schema under subject "json-edge-4":
      """
      {"type":"object","properties":{"color":{"type":"string","enum":["red","green"]}},"required":["color"]}
      """
    Then the response status should be 409

  Scenario: Edge - nested object property removal from open model is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-5" has "JSON" schema:
      """
      {"type":"object","properties":{"user":{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}}}},"required":["user"]}
      """
    When I register a "JSON" schema under subject "json-edge-5":
      """
      {"type":"object","properties":{"user":{"type":"object","properties":{"name":{"type":"string"}}}},"required":["user"]}
      """
    Then the response status should be 200

  Scenario: Edge - array items type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-6" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"}}
      """
    When I register a "JSON" schema under subject "json-edge-6":
      """
      {"type":"array","items":{"type":"integer"}}
      """
    Then the response status should be 409

  Scenario: Edge - minItems increase is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-7" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"minItems":1}
      """
    When I register a "JSON" schema under subject "json-edge-7":
      """
      {"type":"array","items":{"type":"string"},"minItems":10}
      """
    Then the response status should be 409

  Scenario: Edge - maxItems decrease is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-8" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"maxItems":10}
      """
    When I register a "JSON" schema under subject "json-edge-8":
      """
      {"type":"array","items":{"type":"string"},"maxItems":5}
      """
    Then the response status should be 409

  Scenario: Edge - type array expansion is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-9" has "JSON" schema:
      """
      {"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}
      """
    When I register a "JSON" schema under subject "json-edge-9":
      """
      {"type":"object","properties":{"value":{"type":["string","null"]}},"required":["value"]}
      """
    Then the response status should be 200

  Scenario: Edge - maxItems increase is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-10" has "JSON" schema:
      """
      {"type":"array","items":{"type":"string"},"maxItems":5}
      """
    When I register a "JSON" schema under subject "json-edge-10":
      """
      {"type":"array","items":{"type":"string"},"maxItems":10}
      """
    Then the response status should be 200

  Scenario: Edge - root type change object to array is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-11" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}}}
      """
    When I register a "JSON" schema under subject "json-edge-11":
      """
      {"type":"array","items":{"type":"string"}}
      """
    Then the response status should be 409

  Scenario: Edge - make required property optional is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "json-edge-12" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name","age"]}
      """
    When I register a "JSON" schema under subject "json-edge-12":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200

  # ============================================================================
  # Error Validation (5 scenarios)
  # ============================================================================

  Scenario: Error - 409 response has error_code field
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-err-1":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the response status should be 409
    And the response should have error code 409

  Scenario: Error - check endpoint returns is_compatible false for incompatible schema
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I check compatibility of "JSON" schema against subject "json-err-2":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the compatibility check should be incompatible

  Scenario: Error - per-subject NONE override bypasses global BACKWARD
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-3" has compatibility level "NONE"
    And subject "json-err-3" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-err-3":
      """
      {"type":"array","items":{"type":"integer"}}
      """
    Then the response status should be 200

  Scenario: Error - check endpoint returns is_compatible false for open model property addition
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-4" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I check compatibility of "JSON" schema against subject "json-err-4":
      """
      {"type":"object","properties":{"name":{"type":"string"},"email":{"type":"string"}},"required":["name"]}
      """
    Then the compatibility check should be incompatible

  Scenario: Error - delete per-subject config falls back to global
    Given the global compatibility level is "BACKWARD"
    And subject "json-err-5" has compatibility level "NONE"
    And subject "json-err-5" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-err-5":
      """
      {"type":"array","items":{"type":"integer"}}
      """
    Then the response status should be 200
    When I delete the config for subject "json-err-5"
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-err-5":
      """
      {"type":"object","properties":{"x":{"type":"number"}}}
      """
    Then the response status should be 409

  # --- Gap-filling: JSON Schema-specific compatibility rules ---

  Scenario: BACKWARD - additionalProperties false to true is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-1" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    When I register a "JSON" schema under subject "json-gap-1":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    Then the response status should be 200

  Scenario: BACKWARD - additionalProperties true to false is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-2" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}
      """
    When I register a "JSON" schema under subject "json-gap-2":
      """
      {"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}
      """
    Then the response status should be 409

  Scenario: BACKWARD - nested property removal from open model is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-3" has "JSON" schema:
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"}}}}}
      """
    When I register a "JSON" schema under subject "json-gap-3":
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"}}}}}
      """
    Then the response status should be 200

  Scenario: BACKWARD - nested property addition to open model is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-4" has "JSON" schema:
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"}}}}}
      """
    When I register a "JSON" schema under subject "json-gap-4":
      """
      {"type":"object","properties":{"address":{"type":"object","properties":{"street":{"type":"string"},"zip":{"type":"string"}}}}}
      """
    Then the response status should be 409

  Scenario: BACKWARD - type widening string to string-or-null is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-5" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"}}}
      """
    When I register a "JSON" schema under subject "json-gap-5":
      """
      {"type":"object","properties":{"name":{"type":["string","null"]}}}
      """
    Then the response status should be 200

  Scenario: BACKWARD - type narrowing string-or-null to string is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-6" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":["string","null"]}}}
      """
    When I register a "JSON" schema under subject "json-gap-6":
      """
      {"type":"object","properties":{"name":{"type":"string"}}}
      """
    Then the response status should be 409

  Scenario: BACKWARD - array items schema removal is compatible (relaxation)
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-7" has "JSON" schema:
      """
      {"type":"object","properties":{"tags":{"type":"array","items":{"type":"string"}}}}
      """
    When I register a "JSON" schema under subject "json-gap-7":
      """
      {"type":"object","properties":{"tags":{"type":"array"}}}
      """
    Then the response status should be 200

  Scenario: BACKWARD - multiple simultaneous incompatible changes detected
    Given the global compatibility level is "BACKWARD"
    And subject "json-gap-8" has "JSON" schema:
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"},"email":{"type":"string"}},"required":["name"]}
      """
    When I register a "JSON" schema under subject "json-gap-8":
      """
      {"type":"object","properties":{"name":{"type":"integer"},"email":{"type":"string"}},"required":["name","email"]}
      """
    Then the response status should be 409
