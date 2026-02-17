@functional @jsonschema
Feature: Advanced JSON Schema Parsing
  As a developer, I want to register complex and advanced JSON Schema constructs
  to ensure the schema registry correctly handles all Draft 7 features

  # ---------- 1. if/then/else conditional schemas ----------
  Scenario: if/then/else conditional schema
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-1":
      """
      {
        "type": "object",
        "properties": {
          "type": {"type": "string"},
          "value": {}
        },
        "required": ["type"],
        "if": {"properties": {"type": {"const": "string"}}},
        "then": {"properties": {"value": {"type": "string"}}},
        "else": {"properties": {"value": {"type": "number"}}}
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "if"
    And the response should contain "then"
    And the response should contain "else"

  # ---------- 2. Deeply nested $defs (3 levels) ----------
  Scenario: Deeply nested $defs with 3 levels of definitions referencing each other
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-2":
      """
      {
        "type": "object",
        "$defs": {
          "Country": {
            "type": "object",
            "properties": {
              "name": {"type": "string"},
              "capital": {"$ref": "#/$defs/City"}
            }
          },
          "City": {
            "type": "object",
            "properties": {
              "name": {"type": "string"},
              "mayor": {"$ref": "#/$defs/Person"}
            }
          },
          "Person": {
            "type": "object",
            "properties": {
              "firstName": {"type": "string"},
              "lastName": {"type": "string"},
              "age": {"type": "integer", "minimum": 0}
            },
            "required": ["firstName", "lastName"]
          }
        },
        "properties": {
          "country": {"$ref": "#/$defs/Country"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "$defs"
    And the response should contain "Country"
    And the response should contain "City"
    And the response should contain "Person"

  # ---------- 3. $defs with multiple cross-references ----------
  Scenario: Schema with $defs and multiple cross-references
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-3":
      """
      {
        "type": "object",
        "$defs": {
          "address": {
            "type": "object",
            "properties": {
              "street": {"type": "string"},
              "city": {"type": "string"},
              "zip": {"type": "string"}
            }
          },
          "contactInfo": {
            "type": "object",
            "properties": {
              "email": {"type": "string", "format": "email"},
              "phone": {"type": "string"},
              "mailingAddress": {"$ref": "#/$defs/address"}
            }
          }
        },
        "properties": {
          "home": {"$ref": "#/$defs/address"},
          "work": {"$ref": "#/$defs/address"},
          "contact": {"$ref": "#/$defs/contactInfo"},
          "emergencyContact": {"$ref": "#/$defs/contactInfo"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "contactInfo"
    And the response should contain "mailingAddress"

  # ---------- 4. Schema with const keyword ----------
  Scenario: Schema with const keyword
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-4":
      """
      {
        "type": "object",
        "properties": {
          "version": {"const": 1},
          "schema_type": {"const": "event"},
          "active": {"const": true},
          "payload": {"type": "object"}
        },
        "required": ["version", "schema_type", "active"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "const"

  # ---------- 5. Schema with dependencies keyword ----------
  Scenario: Schema with dependencies keyword (property dependencies)
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-5":
      """
      {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "credit_card": {"type": "string"},
          "billing_address": {"type": "string"},
          "shipping_address": {"type": "string"}
        },
        "dependencies": {
          "credit_card": ["billing_address"],
          "shipping_address": {
            "properties": {
              "shipping_type": {"type": "string", "enum": ["standard", "express", "overnight"]}
            },
            "required": ["shipping_type"]
          }
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "dependencies"

  # ---------- 6. Schema with propertyNames constraint ----------
  Scenario: Schema with propertyNames constraint
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-6":
      """
      {
        "type": "object",
        "propertyNames": {
          "type": "string",
          "pattern": "^[a-z][a-zA-Z0-9_]*$",
          "minLength": 2,
          "maxLength": 64
        },
        "additionalProperties": {"type": "string"}
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "propertyNames"

  # ---------- 7. Complex oneOf with discriminator-like pattern ----------
  Scenario: Complex oneOf with discriminator-like pattern using const
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-7":
      """
      {
        "type": "object",
        "properties": {
          "event": {
            "oneOf": [
              {
                "type": "object",
                "properties": {
                  "kind": {"const": "click"},
                  "x": {"type": "integer"},
                  "y": {"type": "integer"}
                },
                "required": ["kind", "x", "y"]
              },
              {
                "type": "object",
                "properties": {
                  "kind": {"const": "keypress"},
                  "key": {"type": "string"},
                  "modifiers": {"type": "array", "items": {"type": "string"}}
                },
                "required": ["kind", "key"]
              },
              {
                "type": "object",
                "properties": {
                  "kind": {"const": "scroll"},
                  "deltaX": {"type": "number"},
                  "deltaY": {"type": "number"}
                },
                "required": ["kind", "deltaX", "deltaY"]
              }
            ]
          }
        },
        "required": ["event"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "oneOf"
    And the response should contain "click"
    And the response should contain "keypress"
    And the response should contain "scroll"

  # ---------- 8. Nested anyOf inside oneOf ----------
  Scenario: Nested anyOf inside oneOf
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-8":
      """
      {
        "type": "object",
        "properties": {
          "target": {
            "oneOf": [
              {
                "type": "object",
                "properties": {
                  "type": {"const": "user"},
                  "identifier": {
                    "anyOf": [
                      {"type": "object", "properties": {"email": {"type": "string", "format": "email"}}, "required": ["email"]},
                      {"type": "object", "properties": {"userId": {"type": "string"}}, "required": ["userId"]}
                    ]
                  }
                },
                "required": ["type", "identifier"]
              },
              {
                "type": "object",
                "properties": {
                  "type": {"const": "group"},
                  "groupId": {"type": "string"}
                },
                "required": ["type", "groupId"]
              }
            ]
          }
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "anyOf"
    And the response should contain "oneOf"

  # ---------- 9. allOf combining multiple object schemas ----------
  Scenario: allOf combining multiple object schemas
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-9":
      """
      {
        "allOf": [
          {
            "type": "object",
            "properties": {
              "id": {"type": "string", "format": "uuid"},
              "createdAt": {"type": "string", "format": "date-time"}
            },
            "required": ["id", "createdAt"]
          },
          {
            "type": "object",
            "properties": {
              "updatedAt": {"type": "string", "format": "date-time"},
              "version": {"type": "integer", "minimum": 1}
            }
          },
          {
            "type": "object",
            "properties": {
              "name": {"type": "string", "minLength": 1},
              "description": {"type": "string"},
              "tags": {"type": "array", "items": {"type": "string"}}
            },
            "required": ["name"]
          }
        ]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "allOf"

  # ---------- 10. Schema with minimum/maximum/exclusiveMinimum/exclusiveMaximum ----------
  Scenario: Schema with numeric boundary constraints
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-10":
      """
      {
        "type": "object",
        "properties": {
          "temperature": {"type": "number", "minimum": -273.15},
          "percentage": {"type": "number", "minimum": 0, "maximum": 100},
          "age": {"type": "integer", "minimum": 0, "exclusiveMaximum": 150},
          "score": {"type": "number", "exclusiveMinimum": 0, "exclusiveMaximum": 10},
          "rating": {"type": "number", "minimum": 1, "maximum": 5}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "exclusiveMinimum"
    And the response should contain "exclusiveMaximum"

  # ---------- 11. Schema with multipleOf for decimals ----------
  Scenario: Schema with multipleOf for decimal precision
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-11":
      """
      {
        "type": "object",
        "properties": {
          "price": {"type": "number", "multipleOf": 0.01, "minimum": 0},
          "quantity": {"type": "integer", "multipleOf": 1, "minimum": 1},
          "weight_kg": {"type": "number", "multipleOf": 0.001},
          "angle_degrees": {"type": "number", "multipleOf": 0.5, "minimum": 0, "maximum": 360}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "multipleOf"

  # ---------- 12. Schema with format validators ----------
  Scenario: Schema with format validators (date-time, email, uri, uuid)
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-12":
      """
      {
        "type": "object",
        "properties": {
          "created": {"type": "string", "format": "date-time"},
          "updated": {"type": "string", "format": "date-time"},
          "email": {"type": "string", "format": "email"},
          "homepage": {"type": "string", "format": "uri"},
          "correlationId": {"type": "string", "format": "uuid"},
          "birthDate": {"type": "string", "format": "date"},
          "startTime": {"type": "string", "format": "time"},
          "hostIp": {"type": "string", "format": "ipv4"},
          "serverIp": {"type": "string", "format": "ipv6"}
        },
        "required": ["created", "email"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "date-time"
    And the response should contain "uuid"

  # ---------- 13. Schema with pattern (regex) validation ----------
  Scenario: Schema with pattern (regex) validation
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-13":
      """
      {
        "type": "object",
        "properties": {
          "zipCode": {"type": "string", "pattern": "^[0-9]{5}(-[0-9]{4})?$"},
          "phoneNumber": {"type": "string", "pattern": "^\\+?[1-9]\\d{1,14}$"},
          "slugField": {"type": "string", "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$"},
          "hexColor": {"type": "string", "pattern": "^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$"},
          "semver": {"type": "string", "pattern": "^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)$"}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "pattern"

  # ---------- 14. Complex patternProperties with multiple patterns ----------
  Scenario: Complex patternProperties with multiple patterns
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-14":
      """
      {
        "type": "object",
        "patternProperties": {
          "^x-": {"type": "string", "minLength": 1},
          "^data_[a-z]+$": {"type": "number"},
          "^is[A-Z]": {"type": "boolean"},
          "^tag_\\d+$": {"type": "string", "maxLength": 50}
        },
        "properties": {
          "id": {"type": "string"}
        },
        "additionalProperties": false
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "patternProperties"

  # ---------- 15. additionalProperties with typed schema ----------
  Scenario: additionalProperties with typed schema (not just boolean)
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-15":
      """
      {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "version": {"type": "integer"}
        },
        "additionalProperties": {
          "type": "object",
          "properties": {
            "value": {"type": "string"},
            "timestamp": {"type": "string", "format": "date-time"}
          },
          "required": ["value"]
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "additionalProperties"

  # ---------- 16. Schema with minProperties/maxProperties ----------
  Scenario: Schema with minProperties and maxProperties
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-16":
      """
      {
        "type": "object",
        "properties": {
          "key": {"type": "string"},
          "value": {"type": "string"}
        },
        "minProperties": 1,
        "maxProperties": 10,
        "additionalProperties": {"type": "string"}
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "minProperties"
    And the response should contain "maxProperties"

  # ---------- 17. Array with items as schema (homogeneous array) ----------
  Scenario: Homogeneous array with complex item schema
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-17":
      """
      {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "id": {"type": "integer"},
            "name": {"type": "string"},
            "tags": {"type": "array", "items": {"type": "string"}, "uniqueItems": true},
            "metadata": {
              "type": "object",
              "additionalProperties": {"type": "string"}
            }
          },
          "required": ["id", "name"]
        },
        "minItems": 0,
        "maxItems": 1000,
        "uniqueItems": false
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "maxItems"

  # ---------- 18. Array with contains keyword ----------
  Scenario: Array with contains keyword
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-18":
      """
      {
        "type": "object",
        "properties": {
          "values": {
            "type": "array",
            "contains": {
              "type": "object",
              "properties": {
                "priority": {"const": "high"}
              },
              "required": ["priority"]
            },
            "items": {
              "type": "object",
              "properties": {
                "priority": {"type": "string", "enum": ["low", "medium", "high"]},
                "message": {"type": "string"}
              }
            }
          }
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "contains"

  # ---------- 19. Nested object with 5 levels deep ----------
  Scenario: Nested object with 5 levels deep
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-19":
      """
      {
        "type": "object",
        "properties": {
          "level1": {
            "type": "object",
            "properties": {
              "name": {"type": "string"},
              "level2": {
                "type": "object",
                "properties": {
                  "name": {"type": "string"},
                  "level3": {
                    "type": "object",
                    "properties": {
                      "name": {"type": "string"},
                      "level4": {
                        "type": "object",
                        "properties": {
                          "name": {"type": "string"},
                          "level5": {
                            "type": "object",
                            "properties": {
                              "value": {"type": "string"},
                              "count": {"type": "integer"}
                            },
                            "required": ["value"]
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "level5"

  # ---------- 20. Object with 30+ properties (stress test) ----------
  Scenario: Object with 30+ properties (stress test)
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-20":
      """
      {
        "type": "object",
        "properties": {
          "field01": {"type": "string"},
          "field02": {"type": "string"},
          "field03": {"type": "integer"},
          "field04": {"type": "integer"},
          "field05": {"type": "number"},
          "field06": {"type": "number"},
          "field07": {"type": "boolean"},
          "field08": {"type": "boolean"},
          "field09": {"type": "string", "format": "date-time"},
          "field10": {"type": "string", "format": "email"},
          "field11": {"type": "string", "minLength": 1},
          "field12": {"type": "string", "maxLength": 255},
          "field13": {"type": "integer", "minimum": 0},
          "field14": {"type": "integer", "maximum": 1000},
          "field15": {"type": "number", "multipleOf": 0.01},
          "field16": {"type": "array", "items": {"type": "string"}},
          "field17": {"type": "array", "items": {"type": "integer"}},
          "field18": {"type": "object", "additionalProperties": {"type": "string"}},
          "field19": {"type": "string", "enum": ["a", "b", "c"]},
          "field20": {"type": "integer", "enum": [1, 2, 3, 4, 5]},
          "field21": {"type": "string", "pattern": "^[A-Z]+$"},
          "field22": {"type": "string", "format": "uri"},
          "field23": {"type": "string", "format": "uuid"},
          "field24": {"type": "boolean"},
          "field25": {"type": "string"},
          "field26": {"type": "integer"},
          "field27": {"type": "number"},
          "field28": {"type": "array", "items": {"type": "boolean"}},
          "field29": {"type": "string", "format": "ipv4"},
          "field30": {"type": "string", "format": "ipv6"},
          "field31": {"type": "object", "properties": {"nested": {"type": "string"}}},
          "field32": {"type": "string", "minLength": 5, "maxLength": 50}
        },
        "required": ["field01", "field03", "field07"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "field32"

  # ---------- 21. allOf with additional constraints ----------
  Scenario: Schema combining allOf with additional constraints
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-21":
      """
      {
        "type": "object",
        "allOf": [
          {
            "properties": {
              "id": {"type": "string"},
              "type": {"type": "string"}
            },
            "required": ["id", "type"]
          },
          {
            "properties": {
              "data": {"type": "object"}
            }
          }
        ],
        "properties": {
          "timestamp": {"type": "string", "format": "date-time"},
          "source": {"type": "string", "minLength": 1}
        },
        "required": ["timestamp"],
        "additionalProperties": false
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "allOf"
    And the response should contain "additionalProperties"

  # ---------- 22. Schema with null type ----------
  Scenario: Schema with null type
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-22":
      """
      {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "deletedAt": {"type": "null"},
          "metadata": {
            "type": "object",
            "properties": {
              "reason": {"type": "null"}
            }
          }
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "null"

  # ---------- 23. Schema with type as array ["string", "null"] ----------
  Scenario: Schema with nullable types using type array
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-23":
      """
      {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "nickname": {"type": ["string", "null"]},
          "age": {"type": ["integer", "null"]},
          "score": {"type": ["number", "null"]},
          "active": {"type": ["boolean", "null"]},
          "tags": {"type": ["array", "null"], "items": {"type": "string"}}
        },
        "required": ["name"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "nickname"

  # ---------- 24. Schema with readOnly/writeOnly annotations ----------
  Scenario: Schema with readOnly and writeOnly annotations
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-24":
      """
      {
        "type": "object",
        "properties": {
          "id": {"type": "string", "readOnly": true},
          "createdAt": {"type": "string", "format": "date-time", "readOnly": true},
          "updatedAt": {"type": "string", "format": "date-time", "readOnly": true},
          "password": {"type": "string", "writeOnly": true, "minLength": 8},
          "secretToken": {"type": "string", "writeOnly": true},
          "name": {"type": "string"},
          "email": {"type": "string", "format": "email"}
        },
        "required": ["name", "email"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "readOnly"
    And the response should contain "writeOnly"

  # ---------- 25. Complex real-world: OpenAPI-style component schema ----------
  Scenario: Complex real-world OpenAPI-style component schema
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-25":
      """
      {
        "type": "object",
        "$defs": {
          "Pagination": {
            "type": "object",
            "properties": {
              "page": {"type": "integer", "minimum": 1},
              "perPage": {"type": "integer", "minimum": 1, "maximum": 100},
              "totalItems": {"type": "integer", "minimum": 0},
              "totalPages": {"type": "integer", "minimum": 0}
            },
            "required": ["page", "perPage", "totalItems", "totalPages"]
          },
          "Error": {
            "type": "object",
            "properties": {
              "code": {"type": "string"},
              "message": {"type": "string"},
              "details": {"type": "array", "items": {"type": "object", "properties": {"field": {"type": "string"}, "reason": {"type": "string"}}}}
            },
            "required": ["code", "message"]
          },
          "Link": {
            "type": "object",
            "properties": {
              "href": {"type": "string", "format": "uri"},
              "rel": {"type": "string"},
              "method": {"type": "string", "enum": ["GET", "POST", "PUT", "DELETE", "PATCH"]}
            },
            "required": ["href", "rel"]
          }
        },
        "properties": {
          "data": {"type": "array", "items": {"type": "object"}},
          "pagination": {"$ref": "#/$defs/Pagination"},
          "errors": {"type": "array", "items": {"$ref": "#/$defs/Error"}},
          "links": {"type": "array", "items": {"$ref": "#/$defs/Link"}}
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Pagination"
    And the response should contain "Error"
    And the response should contain "Link"

  # ---------- 26. Complex real-world: GeoJSON-like schema ----------
  Scenario: Complex real-world GeoJSON-like schema
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-26":
      """
      {
        "type": "object",
        "$defs": {
          "Point": {
            "type": "object",
            "properties": {
              "type": {"const": "Point"},
              "coordinates": {
                "type": "array",
                "items": {"type": "number"},
                "minItems": 2,
                "maxItems": 3
              }
            },
            "required": ["type", "coordinates"]
          },
          "LineString": {
            "type": "object",
            "properties": {
              "type": {"const": "LineString"},
              "coordinates": {
                "type": "array",
                "items": {
                  "type": "array",
                  "items": {"type": "number"},
                  "minItems": 2,
                  "maxItems": 3
                },
                "minItems": 2
              }
            },
            "required": ["type", "coordinates"]
          },
          "Polygon": {
            "type": "object",
            "properties": {
              "type": {"const": "Polygon"},
              "coordinates": {
                "type": "array",
                "items": {
                  "type": "array",
                  "items": {
                    "type": "array",
                    "items": {"type": "number"},
                    "minItems": 2,
                    "maxItems": 3
                  },
                  "minItems": 4
                }
              }
            },
            "required": ["type", "coordinates"]
          }
        },
        "properties": {
          "type": {"const": "Feature"},
          "geometry": {
            "oneOf": [
              {"$ref": "#/$defs/Point"},
              {"$ref": "#/$defs/LineString"},
              {"$ref": "#/$defs/Polygon"}
            ]
          },
          "properties": {
            "type": "object",
            "additionalProperties": true
          }
        },
        "required": ["type", "geometry"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Point"
    And the response should contain "LineString"
    And the response should contain "Polygon"

  # ---------- 27. Complex real-world: Event schema with metadata + payload ----------
  Scenario: Complex real-world event schema with metadata and payload pattern
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-27":
      """
      {
        "type": "object",
        "$defs": {
          "EventMetadata": {
            "type": "object",
            "properties": {
              "eventId": {"type": "string", "format": "uuid"},
              "timestamp": {"type": "string", "format": "date-time"},
              "source": {"type": "string"},
              "correlationId": {"type": "string", "format": "uuid"},
              "causationId": {"type": "string", "format": "uuid"},
              "version": {"type": "integer", "minimum": 1}
            },
            "required": ["eventId", "timestamp", "source", "version"]
          }
        },
        "properties": {
          "metadata": {"$ref": "#/$defs/EventMetadata"},
          "eventType": {"type": "string", "enum": ["UserCreated", "UserUpdated", "UserDeleted", "OrderPlaced", "OrderShipped"]},
          "aggregateId": {"type": "string"},
          "aggregateType": {"type": "string"},
          "payload": {
            "type": "object",
            "additionalProperties": true
          },
          "context": {
            "type": "object",
            "properties": {
              "userId": {"type": "string"},
              "tenantId": {"type": "string"},
              "traceId": {"type": "string"},
              "spanId": {"type": "string"}
            }
          }
        },
        "required": ["metadata", "eventType", "aggregateId", "payload"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response field "schemaType" should be "JSON"
    And the response should contain "EventMetadata"
    And the response should contain "correlationId"

  # ---------- 28. Round-trip: register -> get by ID -> verify ----------
  Scenario: Round-trip register and retrieve with schema field verification
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-28":
      """
      {
        "type": "object",
        "properties": {
          "orderId": {"type": "string", "format": "uuid"},
          "items": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "sku": {"type": "string"},
                "qty": {"type": "integer", "minimum": 1}
              },
              "required": ["sku", "qty"]
            },
            "minItems": 1
          },
          "total": {"type": "number", "minimum": 0}
        },
        "required": ["orderId", "items", "total"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response field "schemaType" should be "JSON"
    When I get version 1 of subject "json-adv-28"
    Then the response status should be 200
    And the response field "subject" should be "json-adv-28"
    And the response field "version" should be 1
    And the response should contain "orderId"

  # ---------- 29. Fingerprint stability: same schema in 2 subjects -> same ID ----------
  Scenario: Fingerprint stability same schema registered in two subjects gets same ID
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-29a":
      """
      {"type":"object","properties":{"fingerprint_test":{"type":"string"},"count":{"type":"integer"}},"required":["fingerprint_test"]}
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I register a "JSON" schema under subject "json-adv-29b":
      """
      {"type":"object","properties":{"fingerprint_test":{"type":"string"},"count":{"type":"integer"}},"required":["fingerprint_test"]}
      """
    Then the response status should be 200
    And the response should have field "id"
    When I get the subjects for the stored schema ID
    Then the response status should be 200

  # ---------- 30. Deeply nested constraints (object > array > object > constraints) ----------
  Scenario: Schema with deeply nested constraints across types
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-30":
      """
      {
        "type": "object",
        "properties": {
          "departments": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "name": {"type": "string", "minLength": 1},
                "employees": {
                  "type": "array",
                  "items": {
                    "type": "object",
                    "properties": {
                      "name": {"type": "string"},
                      "role": {"type": "string", "enum": ["engineer", "manager", "director"]},
                      "skills": {
                        "type": "array",
                        "items": {"type": "string", "minLength": 1},
                        "minItems": 1,
                        "uniqueItems": true
                      },
                      "salary": {
                        "type": "object",
                        "properties": {
                          "amount": {"type": "number", "minimum": 0, "exclusiveMinimum": 0},
                          "currency": {"type": "string", "minLength": 3, "maxLength": 3}
                        },
                        "required": ["amount", "currency"]
                      }
                    },
                    "required": ["name", "role"]
                  },
                  "minItems": 1
                }
              },
              "required": ["name", "employees"]
            },
            "minItems": 1
          }
        },
        "required": ["departments"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "departments"
    And the response should contain "salary"

  # ---------- 31. Boolean schemas (true/false) ----------
  Scenario: Schema with boolean schemas true and false
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-31":
      """
      {
        "type": "object",
        "properties": {
          "anything": true,
          "nothing": false,
          "name": {"type": "string"}
        },
        "required": ["name"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "anything"
    And the response should contain "nothing"

  # ---------- 32. Schema with not keyword ----------
  Scenario: Schema with not keyword
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-32":
      """
      {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "status": {
            "type": "string",
            "not": {"enum": ["deleted", "banned", "suspended"]}
          },
          "value": {
            "not": {"type": "null"}
          },
          "code": {
            "type": "string",
            "not": {"pattern": "^INVALID_"}
          }
        },
        "required": ["name", "status"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "not"

  # ---------- 33. Schema with default values ----------
  Scenario: Schema with default values on properties
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-33":
      """
      {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "role": {"type": "string", "default": "user"},
          "active": {"type": "boolean", "default": true},
          "maxRetries": {"type": "integer", "default": 3, "minimum": 0},
          "timeout": {"type": "number", "default": 30.0, "minimum": 0},
          "tags": {"type": "array", "items": {"type": "string"}, "default": []},
          "settings": {
            "type": "object",
            "properties": {
              "theme": {"type": "string", "default": "light"},
              "language": {"type": "string", "default": "en"}
            },
            "default": {"theme": "light", "language": "en"}
          }
        },
        "required": ["name"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "default"

  # ---------- 34. Schema with examples annotation ----------
  Scenario: Schema with examples annotation
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-34":
      """
      {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "examples": ["Alice", "Bob", "Charlie"]
          },
          "age": {
            "type": "integer",
            "minimum": 0,
            "examples": [25, 30, 45]
          },
          "email": {
            "type": "string",
            "format": "email",
            "examples": ["alice@example.com"]
          },
          "address": {
            "type": "object",
            "properties": {
              "city": {"type": "string"},
              "country": {"type": "string"}
            },
            "examples": [{"city": "London", "country": "UK"}]
          }
        },
        "required": ["name"],
        "examples": [{"name": "Alice", "age": 30, "email": "alice@example.com"}]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "examples"

  # ---------- 35. Schema with multiple anyOf options ----------
  Scenario: Schema with multiple anyOf options
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-35":
      """
      {
        "type": "object",
        "properties": {
          "notification": {
            "anyOf": [
              {
                "type": "object",
                "properties": {
                  "channel": {"const": "email"},
                  "to": {"type": "string", "format": "email"},
                  "subject": {"type": "string"},
                  "body": {"type": "string"}
                },
                "required": ["channel", "to", "subject", "body"]
              },
              {
                "type": "object",
                "properties": {
                  "channel": {"const": "sms"},
                  "to": {"type": "string", "pattern": "^\\+[0-9]+$"},
                  "message": {"type": "string", "maxLength": 160}
                },
                "required": ["channel", "to", "message"]
              },
              {
                "type": "object",
                "properties": {
                  "channel": {"const": "push"},
                  "deviceToken": {"type": "string"},
                  "title": {"type": "string"},
                  "body": {"type": "string"}
                },
                "required": ["channel", "deviceToken", "title"]
              },
              {
                "type": "object",
                "properties": {
                  "channel": {"const": "webhook"},
                  "url": {"type": "string", "format": "uri"},
                  "payload": {"type": "object"}
                },
                "required": ["channel", "url"]
              }
            ]
          }
        },
        "required": ["notification"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "anyOf"
    And the response should contain "webhook"
    And the response should contain "push"

  # ---------- 36. Schema with enum on multiple types ----------
  Scenario: Schema with enum on multiple types
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-36":
      """
      {
        "type": "object",
        "properties": {
          "stringEnum": {"type": "string", "enum": ["alpha", "beta", "gamma", "delta"]},
          "integerEnum": {"type": "integer", "enum": [100, 200, 300, 400, 500]},
          "numberEnum": {"type": "number", "enum": [1.5, 2.5, 3.5]},
          "booleanEnum": {"type": "boolean", "enum": [true]},
          "mixedEnum": {"enum": ["active", 1, true, null]},
          "statusCode": {"type": "integer", "enum": [200, 201, 204, 400, 401, 403, 404, 500]}
        },
        "required": ["stringEnum", "statusCode"]
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "mixedEnum"
    And the response should contain "statusCode"

  # ---------- 37. Standalone non-object schema ----------
  Scenario: Standalone non-object schema with string type and constraints
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-37":
      """
      {"type": "string", "minLength": 1, "maxLength": 500, "pattern": "^[A-Za-z].*"}
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "minLength"
    And the response should contain "maxLength"

  # ---------- 38. Standalone array schema with tuple-like items ----------
  Scenario: Standalone array schema with tuple-like items
    Given the schema registry is running
    When I register a "JSON" schema under subject "json-adv-38":
      """
      {
        "type": "array",
        "items": [
          {"type": "string", "minLength": 1},
          {"type": "integer", "minimum": 0},
          {"type": "boolean"},
          {"type": "string", "format": "date-time"}
        ],
        "minItems": 4,
        "maxItems": 4,
        "additionalItems": false
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "additionalItems"
