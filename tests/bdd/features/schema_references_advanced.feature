@functional
Feature: Advanced Schema References
  Exhaustive testing of cross-subject schema references for Avro, JSON Schema, and Protobuf,
  including reference chains, multiple references, error cases, and referencedby tracking.

  # ==========================================================================
  # AVRO REFERENCES
  # ==========================================================================

  Scenario: Avro cross-subject reference resolves named type
    Given the global compatibility level is "NONE"
    And subject "avro-ref-address" has schema:
      """
      {"type":"record","name":"Address","namespace":"com.test","fields":[{"name":"street","type":"string"},{"name":"city","type":"string"},{"name":"zip","type":"string"}]}
      """
    When I register a schema under subject "avro-ref-person" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Person\",\"namespace\":\"com.test\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"home\",\"type\":\"com.test.Address\"}]}",
        "references": [
          {"name": "com.test.Address", "subject": "avro-ref-address", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Person"

  Scenario: Avro multiple cross-subject references
    Given the global compatibility level is "NONE"
    And subject "avro-ref-customer2" has schema:
      """
      {"type":"record","name":"Customer","namespace":"com.multi","fields":[{"name":"id","type":"string"},{"name":"name","type":"string"}]}
      """
    And subject "avro-ref-product" has schema:
      """
      {"type":"record","name":"Product","namespace":"com.multi","fields":[{"name":"sku","type":"string"},{"name":"price","type":"double"}]}
      """
    When I register a schema under subject "avro-ref-order2" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.multi\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"customer\",\"type\":\"com.multi.Customer\"},{\"name\":\"product\",\"type\":\"com.multi.Product\"},{\"name\":\"quantity\",\"type\":\"int\"}]}",
        "references": [
          {"name": "com.multi.Customer", "subject": "avro-ref-customer2", "version": 1},
          {"name": "com.multi.Product", "subject": "avro-ref-product", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: Avro reference chain - A references B which references C
    Given the global compatibility level is "NONE"
    And subject "avro-chain-c" has schema:
      """
      {"type":"record","name":"GeoPoint","namespace":"com.chain","fields":[{"name":"lat","type":"double"},{"name":"lng","type":"double"}]}
      """
    When I register a schema under subject "avro-chain-b" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Location\",\"namespace\":\"com.chain\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"point\",\"type\":\"com.chain.GeoPoint\"}]}",
        "references": [
          {"name": "com.chain.GeoPoint", "subject": "avro-chain-c", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I register a schema under subject "avro-chain-a" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Event\",\"namespace\":\"com.chain\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"location\",\"type\":\"com.chain.Location\"}]}",
        "references": [
          {"name": "com.chain.GeoPoint", "subject": "avro-chain-c", "version": 1},
          {"name": "com.chain.Location", "subject": "avro-chain-b", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: Avro reference with version pinning
    Given the global compatibility level is "NONE"
    # Register v1 of the referenced schema
    And subject "avro-pin-ref" has schema:
      """
      {"type":"record","name":"Status","namespace":"com.pin","fields":[{"name":"code","type":"int"}]}
      """
    # Register v2 of the referenced schema (different fields)
    And subject "avro-pin-ref" has schema:
      """
      {"type":"record","name":"Status","namespace":"com.pin","fields":[{"name":"code","type":"int"},{"name":"label","type":"string","default":""}]}
      """
    # Reference version 1 specifically
    When I register a schema under subject "avro-pin-main" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.pin\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"},{\"name\":\"status\",\"type\":\"com.pin.Status\"}]}",
        "references": [
          {"name": "com.pin.Status", "subject": "avro-pin-ref", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # JSON SCHEMA REFERENCES
  # ==========================================================================

  Scenario: JSON Schema cross-subject $ref resolves
    Given the global compatibility level is "NONE"
    And subject "json-ref-address" has "JSON" schema:
      """
      {"type":"object","properties":{"street":{"type":"string"},"city":{"type":"string"},"zip":{"type":"string"}},"required":["street","city"]}
      """
    When I register a "JSON" schema under subject "json-ref-person" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"address\":{\"$ref\":\"address.json\"}},\"required\":[\"name\"]}",
        "schemaType": "JSON",
        "references": [
          {"name": "address.json", "subject": "json-ref-address", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: JSON Schema multiple external $refs
    Given the global compatibility level is "NONE"
    And subject "json-ref-billing" has "JSON" schema:
      """
      {"type":"object","properties":{"card_number":{"type":"string"},"expiry":{"type":"string"}},"required":["card_number"]}
      """
    And subject "json-ref-shipping" has "JSON" schema:
      """
      {"type":"object","properties":{"address":{"type":"string"},"method":{"type":"string","enum":["standard","express"]}},"required":["address"]}
      """
    When I register a "JSON" schema under subject "json-ref-checkout" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"order_id\":{\"type\":\"string\"},\"billing\":{\"$ref\":\"billing.json\"},\"shipping\":{\"$ref\":\"shipping.json\"}},\"required\":[\"order_id\"]}",
        "schemaType": "JSON",
        "references": [
          {"name": "billing.json", "subject": "json-ref-billing", "version": 1},
          {"name": "shipping.json", "subject": "json-ref-shipping", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: JSON Schema external $ref combined with internal $defs
    Given the global compatibility level is "NONE"
    And subject "json-ref-ext-type" has "JSON" schema:
      """
      {"type":"object","properties":{"code":{"type":"string"},"name":{"type":"string"}},"required":["code"]}
      """
    When I register a "JSON" schema under subject "json-ref-combo" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"$defs\":{\"Metadata\":{\"type\":\"object\",\"properties\":{\"created\":{\"type\":\"string\"},\"version\":{\"type\":\"integer\"}}}},\"properties\":{\"type\":{\"$ref\":\"external-type.json\"},\"meta\":{\"$ref\":\"#/$defs/Metadata\"}},\"required\":[\"type\"]}",
        "schemaType": "JSON",
        "references": [
          {"name": "external-type.json", "subject": "json-ref-ext-type", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: JSON Schema reference chain (3 levels)
    Given the global compatibility level is "NONE"
    And subject "json-chain-c" has "JSON" schema:
      """
      {"type":"object","properties":{"lat":{"type":"number"},"lng":{"type":"number"}},"required":["lat","lng"]}
      """
    When I register a "JSON" schema under subject "json-chain-b" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"},\"coords\":{\"$ref\":\"geo.json\"}},\"required\":[\"name\"]}",
        "schemaType": "JSON",
        "references": [
          {"name": "geo.json", "subject": "json-chain-c", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I register a "JSON" schema under subject "json-chain-a" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"event_id\":{\"type\":\"string\"},\"location\":{\"$ref\":\"location.json\"}},\"required\":[\"event_id\"]}",
        "schemaType": "JSON",
        "references": [
          {"name": "geo.json", "subject": "json-chain-c", "version": 1},
          {"name": "location.json", "subject": "json-chain-b", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # PROTOBUF REFERENCES
  # ==========================================================================

  Scenario: Protobuf cross-subject import resolves
    Given the global compatibility level is "NONE"
    And subject "proto-ref-common" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package common;
      message Address {
        string street = 1;
        string city = 2;
        string zip = 3;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ref-person" with references:
      """
      {
        "schema": "syntax = \"proto3\";\nimport \"common.proto\";\npackage main;\nmessage Person {\n  string name = 1;\n  common.Address home = 2;\n}",
        "schemaType": "PROTOBUF",
        "references": [
          {"name": "common.proto", "subject": "proto-ref-common", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: Protobuf multiple imports from different subjects
    Given the global compatibility level is "NONE"
    And subject "proto-ref-customer" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package customers;
      message Customer {
        string id = 1;
        string name = 2;
      }
      """
    And subject "proto-ref-product2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package products;
      message Product {
        string sku = 1;
        double price = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ref-order2" with references:
      """
      {
        "schema": "syntax = \"proto3\";\nimport \"customer.proto\";\nimport \"product.proto\";\npackage orders;\nmessage Order {\n  string id = 1;\n  customers.Customer customer = 2;\n  products.Product product = 3;\n  int32 quantity = 4;\n}",
        "schemaType": "PROTOBUF",
        "references": [
          {"name": "customer.proto", "subject": "proto-ref-customer", "version": 1},
          {"name": "product.proto", "subject": "proto-ref-product2", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: Protobuf import chain - 3 levels
    Given the global compatibility level is "NONE"
    And subject "proto-chain-c" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package geo;
      message GeoPoint {
        double lat = 1;
        double lng = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-chain-b" with references:
      """
      {
        "schema": "syntax = \"proto3\";\nimport \"geo.proto\";\npackage locations;\nmessage Location {\n  string name = 1;\n  geo.GeoPoint point = 2;\n}",
        "schemaType": "PROTOBUF",
        "references": [
          {"name": "geo.proto", "subject": "proto-chain-c", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I register a "PROTOBUF" schema under subject "proto-chain-a" with references:
      """
      {
        "schema": "syntax = \"proto3\";\nimport \"location.proto\";\npackage events;\nmessage Event {\n  string id = 1;\n  locations.Location location = 2;\n}",
        "schemaType": "PROTOBUF",
        "references": [
          {"name": "geo.proto", "subject": "proto-chain-c", "version": 1},
          {"name": "location.proto", "subject": "proto-chain-b", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: Protobuf import with well-known types plus custom import
    Given the global compatibility level is "NONE"
    And subject "proto-ref-metadata" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package meta;
      message Metadata {
        string source = 1;
        string version = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ref-wkt-plus" with references:
      """
      {
        "schema": "syntax = \"proto3\";\nimport \"google/protobuf/timestamp.proto\";\nimport \"meta.proto\";\npackage events;\nmessage Event {\n  string id = 1;\n  google.protobuf.Timestamp created_at = 2;\n  meta.Metadata metadata = 3;\n}",
        "schemaType": "PROTOBUF",
        "references": [
          {"name": "meta.proto", "subject": "proto-ref-metadata", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # ERROR CASES
  # ==========================================================================

  Scenario: Reference to non-existent subject returns 422
    Given the global compatibility level is "NONE"
    When I register a schema under subject "ref-err-missing" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Bad\",\"fields\":[{\"name\":\"data\",\"type\":\"com.missing.Type\"}]}",
        "references": [
          {"name": "com.missing.Type", "subject": "does-not-exist", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  Scenario: Reference to non-existent version returns 422
    Given the global compatibility level is "NONE"
    And subject "ref-err-ver-src" has schema:
      """
      {"type":"record","name":"Source","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-err-ver-main" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Main\",\"fields\":[{\"name\":\"src\",\"type\":\"Source\"}]}",
        "references": [
          {"name": "Source", "subject": "ref-err-ver-src", "version": 999}
        ]
      }
      """
    Then the response status should be 422

  Scenario: Protobuf reference to non-existent import returns 422
    Given the global compatibility level is "NONE"
    When I register a "PROTOBUF" schema under subject "proto-ref-missing" with references:
      """
      {
        "schema": "syntax = \"proto3\";\nimport \"missing.proto\";\nmessage Msg {\n  string id = 1;\n}",
        "schemaType": "PROTOBUF",
        "references": [
          {"name": "missing.proto", "subject": "does-not-exist-proto", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  Scenario: JSON Schema reference to non-existent subject returns 422
    Given the global compatibility level is "NONE"
    When I register a "JSON" schema under subject "json-ref-missing" with references:
      """
      {
        "schema": "{\"type\":\"object\",\"properties\":{\"ref\":{\"$ref\":\"missing.json\"}}}",
        "schemaType": "JSON",
        "references": [
          {"name": "missing.json", "subject": "does-not-exist-json", "version": 1}
        ]
      }
      """
    Then the response status should be 422

  # ==========================================================================
  # REFERENCEDBY TRACKING
  # ==========================================================================

  Scenario: Verify referencedby endpoint after registration
    Given the global compatibility level is "NONE"
    And subject "ref-by-src" has schema:
      """
      {"type":"record","name":"RefSource","namespace":"com.refby","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-by-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.refby\",\"fields\":[{\"name\":\"src\",\"type\":\"com.refby.RefSource\"}]}",
        "references": [
          {"name": "com.refby.RefSource", "subject": "ref-by-src", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "ref-by-src" version 1
    Then the response status should be 200

  Scenario: Verify referencedby with multiple consumers
    Given the global compatibility level is "NONE"
    And subject "ref-by-shared" has schema:
      """
      {"type":"record","name":"Shared","namespace":"com.shared","fields":[{"name":"val","type":"string"}]}
      """
    When I register a schema under subject "ref-by-use1" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Use1\",\"namespace\":\"com.shared\",\"fields\":[{\"name\":\"s\",\"type\":\"com.shared.Shared\"}]}",
        "references": [
          {"name": "com.shared.Shared", "subject": "ref-by-shared", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I register a schema under subject "ref-by-use2" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Use2\",\"namespace\":\"com.shared\",\"fields\":[{\"name\":\"s\",\"type\":\"com.shared.Shared\"}]}",
        "references": [
          {"name": "com.shared.Shared", "subject": "ref-by-shared", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    When I get the referenced by for subject "ref-by-shared" version 1
    Then the response status should be 200

  # ==========================================================================
  # SPECIAL CASES
  # ==========================================================================

  Scenario: Schema with unused reference still valid
    Given the global compatibility level is "NONE"
    And subject "ref-unused-src" has schema:
      """
      {"type":"record","name":"Unused","fields":[{"name":"id","type":"string"}]}
      """
    When I register a schema under subject "ref-unused-main" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Main\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "references": [
          {"name": "Unused", "subject": "ref-unused-src", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: Register same schema with different references creates separate entries
    Given the global compatibility level is "NONE"
    And subject "ref-diff-a" has schema:
      """
      {"type":"record","name":"TypeA","namespace":"com.diff","fields":[{"name":"a","type":"string"}]}
      """
    And subject "ref-diff-b" has schema:
      """
      {"type":"record","name":"TypeB","namespace":"com.diff","fields":[{"name":"b","type":"string"}]}
      """
    When I register a schema under subject "ref-diff-main1" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Main\",\"namespace\":\"com.diff\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "references": [
          {"name": "com.diff.TypeA", "subject": "ref-diff-a", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "id1"
    When I register a schema under subject "ref-diff-main2" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Main\",\"namespace\":\"com.diff\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
        "references": [
          {"name": "com.diff.TypeB", "subject": "ref-diff-b", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: Large reference count - 5 references in one Avro schema
    Given the global compatibility level is "NONE"
    And subject "ref-large-1" has schema:
      """
      {"type":"record","name":"T1","namespace":"com.large","fields":[{"name":"v","type":"string"}]}
      """
    And subject "ref-large-2" has schema:
      """
      {"type":"record","name":"T2","namespace":"com.large","fields":[{"name":"v","type":"int"}]}
      """
    And subject "ref-large-3" has schema:
      """
      {"type":"record","name":"T3","namespace":"com.large","fields":[{"name":"v","type":"long"}]}
      """
    And subject "ref-large-4" has schema:
      """
      {"type":"record","name":"T4","namespace":"com.large","fields":[{"name":"v","type":"float"}]}
      """
    And subject "ref-large-5" has schema:
      """
      {"type":"record","name":"T5","namespace":"com.large","fields":[{"name":"v","type":"double"}]}
      """
    When I register a schema under subject "ref-large-main" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"BigRef\",\"namespace\":\"com.large\",\"fields\":[{\"name\":\"a\",\"type\":\"com.large.T1\"},{\"name\":\"b\",\"type\":\"com.large.T2\"},{\"name\":\"c\",\"type\":\"com.large.T3\"},{\"name\":\"d\",\"type\":\"com.large.T4\"},{\"name\":\"e\",\"type\":\"com.large.T5\"}]}",
        "references": [
          {"name": "com.large.T1", "subject": "ref-large-1", "version": 1},
          {"name": "com.large.T2", "subject": "ref-large-2", "version": 1},
          {"name": "com.large.T3", "subject": "ref-large-3", "version": 1},
          {"name": "com.large.T4", "subject": "ref-large-4", "version": 1},
          {"name": "com.large.T5", "subject": "ref-large-5", "version": 1}
        ]
      }
      """
    Then the response status should be 200

  Scenario: Update referenced schema and register with new version
    Given the global compatibility level is "NONE"
    And subject "ref-update-src" has schema:
      """
      {"type":"record","name":"Src","namespace":"com.update","fields":[{"name":"id","type":"string"}]}
      """
    # Register consumer using v1
    When I register a schema under subject "ref-update-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.update\",\"fields\":[{\"name\":\"src\",\"type\":\"com.update.Src\"}]}",
        "references": [
          {"name": "com.update.Src", "subject": "ref-update-src", "version": 1}
        ]
      }
      """
    Then the response status should be 200
    # Update the referenced schema to v2
    And subject "ref-update-src" has schema:
      """
      {"type":"record","name":"Src","namespace":"com.update","fields":[{"name":"id","type":"string"},{"name":"name","type":"string","default":""}]}
      """
    # Register new consumer version using v2 of reference
    When I register a schema under subject "ref-update-consumer" with references:
      """
      {
        "schema": "{\"type\":\"record\",\"name\":\"Consumer\",\"namespace\":\"com.update\",\"fields\":[{\"name\":\"src\",\"type\":\"com.update.Src\"},{\"name\":\"tag\",\"type\":\"string\",\"default\":\"\"}]}",
        "references": [
          {"name": "com.update.Src", "subject": "ref-update-src", "version": 2}
        ]
      }
      """
    Then the response status should be 200
