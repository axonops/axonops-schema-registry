@functional @jsonschema
Feature: JSON Schema Types
  As a developer, I want to register and retrieve every valid JSON Schema shape

  Scenario: Simple object with required fields
    When I register a "JSON" schema under subject "json-simple":
      """
      {"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}},"required":["name"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "name"

  Scenario: String constraints (format, minLength, maxLength, pattern)
    When I register a "JSON" schema under subject "json-string-constraints":
      """
      {"type":"object","properties":{
        "email":{"type":"string","format":"email"},
        "username":{"type":"string","minLength":3,"maxLength":20,"pattern":"^[a-zA-Z0-9_]+$"},
        "phone":{"type":"string","pattern":"^\\+[0-9]{10,15}$"}
      },"required":["email","username"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "minLength"

  Scenario: Numeric constraints
    When I register a "JSON" schema under subject "json-numeric":
      """
      {"type":"object","properties":{
        "age":{"type":"integer","minimum":0,"maximum":150},
        "score":{"type":"number","minimum":0,"maximum":100},
        "quantity":{"type":"integer","minimum":1,"multipleOf":1},
        "temperature":{"type":"number","minimum":-273.15}
      }}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "multipleOf"

  Scenario: Array constraints
    When I register a "JSON" schema under subject "json-arrays":
      """
      {"type":"object","properties":{
        "tags":{"type":"array","items":{"type":"string"},"minItems":1,"maxItems":10,"uniqueItems":true},
        "scores":{"type":"array","items":{"type":"number"}},
        "matrix":{"type":"array","items":{"type":"array","items":{"type":"integer"}}}
      }}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "uniqueItems"

  Scenario: Nested objects (2 levels)
    When I register a "JSON" schema under subject "json-nested-2":
      """
      {"type":"object","properties":{
        "id":{"type":"string"},
        "address":{"type":"object","properties":{
          "street":{"type":"string"},
          "city":{"type":"string"},
          "zip":{"type":"string"}
        },"required":["street","city"]}
      },"required":["id"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "address"

  Scenario: Deeply nested objects (3+ levels)
    When I register a "JSON" schema under subject "json-nested-3":
      """
      {"type":"object","properties":{
        "l1":{"type":"object","properties":{
          "l2":{"type":"object","properties":{
            "l3":{"type":"object","properties":{
              "l4":{"type":"object","properties":{
                "value":{"type":"string"}
              }}
            }}
          }}
        }}
      }}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "value"

  Scenario: Enum values
    When I register a "JSON" schema under subject "json-enum":
      """
      {"type":"object","properties":{
        "status":{"type":"string","enum":["active","inactive","pending","suspended"]},
        "priority":{"type":"integer","enum":[1,2,3,4,5]},
        "color":{"type":"string","enum":["red","green","blue"]}
      },"required":["status"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "suspended"

  Scenario: oneOf composition
    When I register a "JSON" schema under subject "json-oneof":
      """
      {"type":"object","properties":{
        "id":{"type":"string"},
        "payment":{"oneOf":[
          {"type":"object","properties":{"card_number":{"type":"string"},"expiry":{"type":"string"}},"required":["card_number"]},
          {"type":"object","properties":{"bank_name":{"type":"string"},"account":{"type":"string"}},"required":["bank_name"]},
          {"type":"object","properties":{"wallet_id":{"type":"string"}},"required":["wallet_id"]}
        ]}
      }}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "oneOf"

  Scenario: anyOf composition
    When I register a "JSON" schema under subject "json-anyof":
      """
      {"type":"object","properties":{
        "contact":{"anyOf":[
          {"type":"object","properties":{"email":{"type":"string","format":"email"}},"required":["email"]},
          {"type":"object","properties":{"phone":{"type":"string"}},"required":["phone"]}
        ]}
      }}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "anyOf"

  Scenario: allOf composition
    When I register a "JSON" schema under subject "json-allof":
      """
      {"allOf":[
        {"type":"object","properties":{"name":{"type":"string"}},"required":["name"]},
        {"type":"object","properties":{"age":{"type":"integer","minimum":0}}},
        {"type":"object","properties":{"email":{"type":"string"}}}
      ]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "allOf"

  Scenario: additionalProperties (false and typed)
    When I register a "JSON" schema under subject "json-additional-props":
      """
      {"type":"object","properties":{
        "name":{"type":"string"},
        "age":{"type":"integer"}
      },"additionalProperties":false}
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-additional-props-typed":
      """
      {"type":"object","properties":{
        "name":{"type":"string"}
      },"additionalProperties":{"type":"string"}}
      """
    Then the response status should be 200

  Scenario: $defs and $ref (internal)
    When I register a "JSON" schema under subject "json-defs-ref":
      """
      {"type":"object","properties":{
        "billing":{"$ref":"#/$defs/Address"},
        "shipping":{"$ref":"#/$defs/Address"}
      },"$defs":{
        "Address":{"type":"object","properties":{
          "street":{"type":"string"},
          "city":{"type":"string"},
          "zip":{"type":"string"}
        },"required":["street","city"]}
      }}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "$defs"

  Scenario: Standalone non-object types (string)
    When I register a "JSON" schema under subject "json-standalone-string":
      """
      {"type":"string","minLength":1,"maxLength":255}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "maxLength"

  Scenario: Standalone non-object types (array)
    When I register a "JSON" schema under subject "json-standalone-array":
      """
      {"type":"array","items":{"type":"number"},"minItems":1}
      """
    Then the response status should be 200

  Scenario: Standalone non-object types (integer)
    When I register a "JSON" schema under subject "json-standalone-integer":
      """
      {"type":"integer","minimum":0,"maximum":100}
      """
    Then the response status should be 200

  Scenario: patternProperties
    When I register a "JSON" schema under subject "json-pattern-props":
      """
      {"type":"object","patternProperties":{
        "^x-":{"type":"string"},
        "^[0-9]+$":{"type":"integer"}
      },"additionalProperties":false}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "patternProperties"

  Scenario: Complex real-world PaymentEvent schema
    When I register a "JSON" schema under subject "json-payment-event":
      """
      {
        "type": "object",
        "properties": {
          "event_id": {"type": "string", "format": "uuid"},
          "timestamp": {"type": "string", "format": "date-time"},
          "event_type": {"type": "string", "enum": ["INITIATED", "AUTHORIZED", "CAPTURED", "REFUNDED", "FAILED"]},
          "amount": {
            "type": "object",
            "properties": {
              "value": {"type": "number", "minimum": 0},
              "currency": {"type": "string", "minLength": 3, "maxLength": 3}
            },
            "required": ["value", "currency"]
          },
          "customer": {
            "type": "object",
            "properties": {
              "id": {"type": "string"},
              "name": {"type": "string"},
              "email": {"type": "string", "format": "email"}
            },
            "required": ["id", "name"]
          },
          "items": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "product_id": {"type": "string"},
                "name": {"type": "string"},
                "quantity": {"type": "integer", "minimum": 1},
                "unit_price": {"type": "number", "minimum": 0}
              },
              "required": ["product_id", "name", "quantity"]
            },
            "minItems": 1
          },
          "metadata": {
            "type": "object",
            "additionalProperties": {"type": "string"}
          },
          "payment_method": {
            "oneOf": [
              {
                "type": "object",
                "properties": {
                  "type": {"const": "card"},
                  "last_four": {"type": "string", "pattern": "^[0-9]{4}$"},
                  "brand": {"type": "string"}
                },
                "required": ["type", "last_four"]
              },
              {
                "type": "object",
                "properties": {
                  "type": {"const": "bank"},
                  "bank_name": {"type": "string"},
                  "account_last_four": {"type": "string"}
                },
                "required": ["type", "bank_name"]
              }
            ]
          }
        },
        "required": ["event_id", "timestamp", "event_type", "amount", "customer", "items"]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response field "schemaType" should be "JSON"
    When I get version 1 of subject "json-payment-event"
    Then the response status should be 200
    And the response field "version" should be 1

  Scenario: Retrieve JSON Schema round-trip
    Given subject "json-roundtrip" has "JSON" schema:
      """
      {"type":"object","properties":{"id":{"type":"string"},"value":{"type":"integer"}},"required":["id"]}
      """
    When I get version 1 of subject "json-roundtrip"
    Then the response status should be 200
    And the response field "subject" should be "json-roundtrip"
    And the response field "version" should be 1
    And the response field "schemaType" should be "JSON"
