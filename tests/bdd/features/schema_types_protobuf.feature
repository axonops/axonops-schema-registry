@functional @protobuf
Feature: Protobuf Schema Types
  As a developer, I want to register and retrieve every valid Protobuf schema shape

  Scenario: Simple message with all scalar types
    When I register a "PROTOBUF" schema under subject "proto-scalars":
      """
      syntax = "proto3";
      message AllScalars {
        double f_double = 1;
        float f_float = 2;
        int32 f_int32 = 3;
        int64 f_int64 = 4;
        uint32 f_uint32 = 5;
        uint64 f_uint64 = 6;
        sint32 f_sint32 = 7;
        sint64 f_sint64 = 8;
        fixed32 f_fixed32 = 9;
        fixed64 f_fixed64 = 10;
        sfixed32 f_sfixed32 = 11;
        sfixed64 f_sfixed64 = 12;
        bool f_bool = 13;
        string f_string = 14;
        bytes f_bytes = 15;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "AllScalars"

  Scenario: Nested messages (2 levels)
    When I register a "PROTOBUF" schema under subject "proto-nested-2":
      """
      syntax = "proto3";
      message Order {
        string id = 1;
        Customer customer = 2;
        message Customer {
          string name = 1;
          string email = 2;
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "Customer"

  Scenario: Deeply nested messages (3+ levels)
    When I register a "PROTOBUF" schema under subject "proto-nested-3":
      """
      syntax = "proto3";
      message L1 {
        L2 l2 = 1;
        message L2 {
          L3 l3 = 1;
          message L3 {
            L4 l4 = 1;
            message L4 {
              string value = 1;
            }
          }
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "L4"

  Scenario: Enums (top-level and nested)
    When I register a "PROTOBUF" schema under subject "proto-enums":
      """
      syntax = "proto3";
      enum Status {
        STATUS_UNKNOWN = 0;
        STATUS_ACTIVE = 1;
        STATUS_INACTIVE = 2;
      }
      message WithEnum {
        Status status = 1;
        Priority priority = 2;
        enum Priority {
          PRIORITY_UNKNOWN = 0;
          PRIORITY_LOW = 1;
          PRIORITY_HIGH = 2;
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "STATUS_ACTIVE"

  Scenario: Repeated fields
    When I register a "PROTOBUF" schema under subject "proto-repeated":
      """
      syntax = "proto3";
      message WithRepeated {
        repeated string tags = 1;
        repeated int32 scores = 2;
        repeated Item items = 3;
        message Item {
          string name = 1;
          int32 quantity = 2;
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "repeated"

  Scenario: Simple and complex maps
    When I register a "PROTOBUF" schema under subject "proto-maps":
      """
      syntax = "proto3";
      message WithMaps {
        map<string, string> metadata = 1;
        map<string, int32> counts = 2;
        map<string, Nested> nested_map = 3;
        message Nested {
          string value = 1;
          int32 count = 2;
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "WithMaps"

  Scenario: Oneof fields
    When I register a "PROTOBUF" schema under subject "proto-oneof":
      """
      syntax = "proto3";
      message WithOneof {
        string id = 1;
        oneof payment {
          CreditCard card = 2;
          BankAccount bank = 3;
          string wallet_id = 4;
        }
        message CreditCard {
          string number = 1;
          string expiry = 2;
        }
        message BankAccount {
          string routing = 1;
          string account = 2;
        }
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "oneof"

  Scenario: Proto3 optional fields
    When I register a "PROTOBUF" schema under subject "proto-optional":
      """
      syntax = "proto3";
      message WithOptional {
        string name = 1;
        optional string email = 2;
        optional int32 age = 3;
        optional bool verified = 4;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "optional"

  Scenario: Package declarations
    When I register a "PROTOBUF" schema under subject "proto-package":
      """
      syntax = "proto3";
      package com.example.events;
      message Event {
        string id = 1;
        string type = 2;
        int64 timestamp = 3;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "com.example.events"

  Scenario: Multiple top-level messages
    When I register a "PROTOBUF" schema under subject "proto-multi-msg":
      """
      syntax = "proto3";
      message Request {
        string query = 1;
        int32 page = 2;
      }
      message Response {
        repeated Result results = 1;
        int32 total = 2;
      }
      message Result {
        string id = 1;
        string title = 2;
        double score = 3;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "Request"
    And the response should contain "Response"
    And the response should contain "Result"

  Scenario: Service definition
    When I register a "PROTOBUF" schema under subject "proto-service":
      """
      syntax = "proto3";
      message SearchRequest {
        string query = 1;
      }
      message SearchResponse {
        repeated string results = 1;
      }
      service SearchService {
        rpc Search(SearchRequest) returns (SearchResponse);
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "SearchService"

  Scenario: Proto2 syntax
    When I register a "PROTOBUF" schema under subject "proto-proto2":
      """
      syntax = "proto2";
      message LegacyMessage {
        required string name = 1;
        optional int32 age = 2;
        repeated string tags = 3;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response should contain "LegacyMessage"

  Scenario: Complex real-world PaymentEvent schema
    When I register a "PROTOBUF" schema under subject "proto-payment-event":
      """
      syntax = "proto3";
      package com.example.payments;

      enum PaymentEventType {
        PAYMENT_EVENT_TYPE_UNKNOWN = 0;
        PAYMENT_EVENT_TYPE_INITIATED = 1;
        PAYMENT_EVENT_TYPE_AUTHORIZED = 2;
        PAYMENT_EVENT_TYPE_CAPTURED = 3;
        PAYMENT_EVENT_TYPE_REFUNDED = 4;
        PAYMENT_EVENT_TYPE_FAILED = 5;
      }

      message PaymentEvent {
        string event_id = 1;
        int64 timestamp = 2;
        PaymentEventType event_type = 3;
        Money amount = 4;
        Customer customer = 5;
        repeated LineItem items = 6;
        map<string, string> metadata = 7;

        oneof payment_method {
          CardPayment card = 8;
          BankTransfer bank = 9;
        }
      }

      message Money {
        int64 units = 1;
        int32 nanos = 2;
        string currency = 3;
      }

      message Customer {
        string id = 1;
        string name = 2;
        optional string email = 3;
      }

      message LineItem {
        string product_id = 1;
        string name = 2;
        int32 quantity = 3;
        Money unit_price = 4;
      }

      message CardPayment {
        string last_four = 1;
        string brand = 2;
      }

      message BankTransfer {
        string bank_name = 1;
        string account_last_four = 2;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "PaymentEvent"
    And the response field "schemaType" should be "PROTOBUF"
    When I get version 1 of subject "proto-payment-event"
    Then the response status should be 200
    And the response field "version" should be 1

  Scenario: Retrieve Protobuf schema round-trip
    Given subject "proto-roundtrip" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message RoundTrip {
        string id = 1;
        int32 value = 2;
      }
      """
    When I get version 1 of subject "proto-roundtrip"
    Then the response status should be 200
    And the response field "subject" should be "proto-roundtrip"
    And the response field "version" should be 1
    And the response field "schemaType" should be "PROTOBUF"
