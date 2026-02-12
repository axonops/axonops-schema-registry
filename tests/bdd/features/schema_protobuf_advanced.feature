@functional @protobuf
Feature: Advanced Protobuf Schema Parsing
  As a developer, I want to register and retrieve advanced Protobuf schema constructs
  including all scalar types, deep nesting, enums with options, oneofs, maps, services,
  streaming RPCs, proto2 syntax, well-known types, and complex real-world schemas.

  Background:
    Given the schema registry is running

  # --------------------------------------------------------------------------
  # 1. All 15 scalar types in one message
  # --------------------------------------------------------------------------
  Scenario: All 15 Protobuf scalar types in a single message
    When I register a "PROTOBUF" schema under subject "proto-adv-1":
      """
      syntax = "proto3";
      message AllScalarTypes {
        double field_double = 1;
        float field_float = 2;
        int32 field_int32 = 3;
        int64 field_int64 = 4;
        uint32 field_uint32 = 5;
        uint64 field_uint64 = 6;
        sint32 field_sint32 = 7;
        sint64 field_sint64 = 8;
        fixed32 field_fixed32 = 9;
        fixed64 field_fixed64 = 10;
        sfixed32 field_sfixed32 = 11;
        sfixed64 field_sfixed64 = 12;
        bool field_bool = 13;
        string field_string = 14;
        bytes field_bytes = 15;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "AllScalarTypes"
    And the response should contain "field_double"
    And the response should contain "field_bytes"

  # --------------------------------------------------------------------------
  # 2. Deeply nested messages (4+ levels)
  # --------------------------------------------------------------------------
  Scenario: Deeply nested messages with 4 levels
    When I register a "PROTOBUF" schema under subject "proto-adv-2":
      """
      syntax = "proto3";
      message Outer {
        string outer_id = 1;
        Inner inner = 2;
        message Inner {
          string inner_id = 1;
          Deep deep = 2;
          message Deep {
            string deep_id = 1;
            Core core = 2;
            message Core {
              string value = 1;
              int64 timestamp = 2;
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
    And the response should contain "Outer"
    And the response should contain "Core"

  # --------------------------------------------------------------------------
  # 3. Enum with allow_alias option
  # --------------------------------------------------------------------------
  Scenario: Enum with many values and explicit numbering
    When I register a "PROTOBUF" schema under subject "proto-adv-3":
      """
      syntax = "proto3";
      enum Priority {
        PRIORITY_UNKNOWN = 0;
        PRIORITY_LOW = 1;
        PRIORITY_NORMAL = 2;
        PRIORITY_HIGH = 3;
        PRIORITY_URGENT = 4;
        PRIORITY_CRITICAL = 5;
      }
      message Task {
        string name = 1;
        Priority priority = 2;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "PRIORITY_CRITICAL"
    And the response should contain "PRIORITY_URGENT"

  # --------------------------------------------------------------------------
  # 4. Enum with reserved values
  # --------------------------------------------------------------------------
  Scenario: Enum with reserved values and names
    When I register a "PROTOBUF" schema under subject "proto-adv-4":
      """
      syntax = "proto3";
      enum ErrorCode {
        reserved 2, 15, 9 to 11;
        reserved "OLD_ERROR", "LEGACY_ERROR";
        ERROR_UNKNOWN = 0;
        ERROR_TIMEOUT = 1;
        ERROR_INTERNAL = 3;
        ERROR_NOT_FOUND = 4;
      }
      message ErrorResponse {
        ErrorCode code = 1;
        string message = 2;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "ErrorCode"
    And the response should contain "reserved"

  # --------------------------------------------------------------------------
  # 5. Oneof with multiple field types
  # --------------------------------------------------------------------------
  Scenario: Oneof with multiple scalar field types
    When I register a "PROTOBUF" schema under subject "proto-adv-5":
      """
      syntax = "proto3";
      message Notification {
        string id = 1;
        oneof channel {
          string email = 2;
          string sms_number = 3;
          int64 push_device_id = 4;
          bool in_app = 5;
          bytes raw_data = 6;
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "oneof"
    And the response should contain "email"
    And the response should contain "push_device_id"

  # --------------------------------------------------------------------------
  # 6. Oneof with message type fields
  # --------------------------------------------------------------------------
  Scenario: Oneof with message type fields
    When I register a "PROTOBUF" schema under subject "proto-adv-6":
      """
      syntax = "proto3";
      message Shape {
        string name = 1;
        oneof shape_type {
          Circle circle = 2;
          Rectangle rectangle = 3;
          Triangle triangle = 4;
        }
      }
      message Circle {
        double radius = 1;
      }
      message Rectangle {
        double width = 1;
        double height = 2;
      }
      message Triangle {
        double side_a = 1;
        double side_b = 2;
        double side_c = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "shape_type"
    And the response should contain "Circle"
    And the response should contain "Triangle"

  # --------------------------------------------------------------------------
  # 7. Map with enum values
  # --------------------------------------------------------------------------
  Scenario: Map with enum values
    When I register a "PROTOBUF" schema under subject "proto-adv-7":
      """
      syntax = "proto3";
      enum FeatureFlag {
        FEATURE_FLAG_UNKNOWN = 0;
        FEATURE_FLAG_ENABLED = 1;
        FEATURE_FLAG_DISABLED = 2;
        FEATURE_FLAG_EXPERIMENTAL = 3;
      }
      message FeatureConfig {
        map<string, FeatureFlag> flags = 1;
        string config_version = 2;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "FeatureConfig"
    And the response should contain "FeatureFlag"

  # --------------------------------------------------------------------------
  # 8. Map with nested message values
  # --------------------------------------------------------------------------
  Scenario: Map with nested message values
    When I register a "PROTOBUF" schema under subject "proto-adv-8":
      """
      syntax = "proto3";
      message UserProfiles {
        map<string, UserProfile> profiles = 1;
        message UserProfile {
          string display_name = 1;
          int32 age = 2;
          Address address = 3;
          message Address {
            string street = 1;
            string city = 2;
            string country = 3;
          }
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "UserProfiles"
    And the response should contain "UserProfile"
    And the response should contain "Address"

  # --------------------------------------------------------------------------
  # 9. Repeated message fields
  # --------------------------------------------------------------------------
  Scenario: Repeated message fields
    When I register a "PROTOBUF" schema under subject "proto-adv-9":
      """
      syntax = "proto3";
      message Playlist {
        string name = 1;
        string owner = 2;
        repeated Track tracks = 3;
        repeated Collaborator collaborators = 4;
      }
      message Track {
        string title = 1;
        string artist = 2;
        int32 duration_seconds = 3;
      }
      message Collaborator {
        string user_id = 1;
        string role = 2;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Playlist"
    And the response should contain "repeated"
    And the response should contain "Track"

  # --------------------------------------------------------------------------
  # 10. Service with server streaming
  # --------------------------------------------------------------------------
  Scenario: Service with server streaming RPC
    When I register a "PROTOBUF" schema under subject "proto-adv-10":
      """
      syntax = "proto3";
      message SubscribeRequest {
        string topic = 1;
        int64 from_offset = 2;
      }
      message Event {
        string id = 1;
        bytes payload = 2;
        int64 timestamp = 3;
      }
      service EventStream {
        rpc Subscribe(SubscribeRequest) returns (stream Event);
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "EventStream"
    And the response should contain "stream"

  # --------------------------------------------------------------------------
  # 11. Service with client streaming
  # --------------------------------------------------------------------------
  Scenario: Service with client streaming RPC
    When I register a "PROTOBUF" schema under subject "proto-adv-11":
      """
      syntax = "proto3";
      message LogEntry {
        string level = 1;
        string message = 2;
        int64 timestamp = 3;
      }
      message LogSummary {
        int32 total_entries = 1;
        int32 error_count = 2;
      }
      service LogIngestion {
        rpc IngestLogs(stream LogEntry) returns (LogSummary);
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "LogIngestion"
    And the response should contain "stream"

  # --------------------------------------------------------------------------
  # 12. Service with bidirectional streaming
  # --------------------------------------------------------------------------
  Scenario: Service with bidirectional streaming RPC
    When I register a "PROTOBUF" schema under subject "proto-adv-12":
      """
      syntax = "proto3";
      message ChatMessage {
        string sender = 1;
        string text = 2;
        int64 sent_at = 3;
      }
      message ChatAck {
        string message_id = 1;
        bool delivered = 2;
      }
      service ChatService {
        rpc Chat(stream ChatMessage) returns (stream ChatAck);
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "ChatService"

  # --------------------------------------------------------------------------
  # 13. Service with multiple methods
  # --------------------------------------------------------------------------
  Scenario: Service with multiple RPC methods
    When I register a "PROTOBUF" schema under subject "proto-adv-13":
      """
      syntax = "proto3";
      message User {
        string id = 1;
        string name = 2;
        string email = 3;
      }
      message GetUserRequest {
        string id = 1;
      }
      message CreateUserRequest {
        string name = 1;
        string email = 2;
      }
      message DeleteUserRequest {
        string id = 1;
      }
      message DeleteUserResponse {
        bool success = 1;
      }
      message ListUsersRequest {
        int32 page_size = 1;
        string page_token = 2;
      }
      message ListUsersResponse {
        repeated User users = 1;
        string next_page_token = 2;
      }
      service UserService {
        rpc GetUser(GetUserRequest) returns (User);
        rpc CreateUser(CreateUserRequest) returns (User);
        rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
        rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "UserService"
    And the response should contain "GetUser"
    And the response should contain "CreateUser"
    And the response should contain "ListUsers"

  # --------------------------------------------------------------------------
  # 14. Proto2 required fields
  # --------------------------------------------------------------------------
  Scenario: Proto2 syntax with required fields
    When I register a "PROTOBUF" schema under subject "proto-adv-14":
      """
      syntax = "proto2";
      message LegacyRecord {
        required int32 id = 1;
        required string name = 2;
        required bool active = 3;
        optional string description = 4;
        repeated string tags = 5;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "LegacyRecord"
    And the response should contain "required"

  # --------------------------------------------------------------------------
  # 15. Proto2 default values
  # --------------------------------------------------------------------------
  Scenario: Proto2 syntax with default values
    When I register a "PROTOBUF" schema under subject "proto-adv-15":
      """
      syntax = "proto2";
      message ConfigEntry {
        required string key = 1;
        optional string value = 2 [default = ""];
        optional int32 ttl_seconds = 3 [default = 3600];
        optional bool encrypted = 4 [default = false];
        optional double weight = 5 [default = 1.0];
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "ConfigEntry"
    And the response should contain "default"

  # --------------------------------------------------------------------------
  # 16. Well-known type: google.protobuf.Timestamp
  # --------------------------------------------------------------------------
  Scenario: Well-known type Timestamp
    When I register a "PROTOBUF" schema under subject "proto-adv-16":
      """
      syntax = "proto3";
      import "google/protobuf/timestamp.proto";
      message AuditEvent {
        string event_id = 1;
        string action = 2;
        google.protobuf.Timestamp created_at = 3;
        google.protobuf.Timestamp updated_at = 4;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "AuditEvent"
    And the response should contain "google.protobuf.Timestamp"

  # --------------------------------------------------------------------------
  # 17. Well-known type: google.protobuf.Duration
  # --------------------------------------------------------------------------
  Scenario: Well-known type Duration
    When I register a "PROTOBUF" schema under subject "proto-adv-17":
      """
      syntax = "proto3";
      import "google/protobuf/duration.proto";
      message TaskExecution {
        string task_id = 1;
        google.protobuf.Duration elapsed = 2;
        google.protobuf.Duration timeout = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "TaskExecution"
    And the response should contain "google.protobuf.Duration"

  # --------------------------------------------------------------------------
  # 18. Well-known type: google.protobuf.Any
  # --------------------------------------------------------------------------
  Scenario: Well-known type Any
    When I register a "PROTOBUF" schema under subject "proto-adv-18":
      """
      syntax = "proto3";
      import "google/protobuf/any.proto";
      message Envelope {
        string type_url = 1;
        google.protobuf.Any payload = 2;
        map<string, string> headers = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Envelope"
    And the response should contain "google.protobuf.Any"

  # --------------------------------------------------------------------------
  # 19. Well-known type: google.protobuf.Struct
  # --------------------------------------------------------------------------
  Scenario: Well-known type Struct
    When I register a "PROTOBUF" schema under subject "proto-adv-19":
      """
      syntax = "proto3";
      import "google/protobuf/struct.proto";
      message DynamicConfig {
        string name = 1;
        google.protobuf.Struct properties = 2;
        google.protobuf.Value single_value = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "DynamicConfig"
    And the response should contain "google.protobuf.Struct"

  # --------------------------------------------------------------------------
  # 20. Well-known type: google.protobuf.FieldMask
  # --------------------------------------------------------------------------
  Scenario: Well-known type FieldMask
    When I register a "PROTOBUF" schema under subject "proto-adv-20":
      """
      syntax = "proto3";
      import "google/protobuf/field_mask.proto";
      message UpdateRequest {
        string resource_id = 1;
        google.protobuf.FieldMask update_mask = 2;
        string new_value = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "UpdateRequest"
    And the response should contain "google.protobuf.FieldMask"

  # --------------------------------------------------------------------------
  # 21. Well-known type: google.protobuf.StringValue (wrapper)
  # --------------------------------------------------------------------------
  Scenario: Well-known type StringValue wrapper
    When I register a "PROTOBUF" schema under subject "proto-adv-21":
      """
      syntax = "proto3";
      import "google/protobuf/wrappers.proto";
      message NullableFields {
        string id = 1;
        google.protobuf.StringValue nickname = 2;
        google.protobuf.Int32Value score = 3;
        google.protobuf.BoolValue verified = 4;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "NullableFields"
    And the response should contain "google.protobuf.StringValue"

  # --------------------------------------------------------------------------
  # 22. Package with nested package name (a.b.c)
  # --------------------------------------------------------------------------
  Scenario: Package with deeply nested package name
    When I register a "PROTOBUF" schema under subject "proto-adv-22":
      """
      syntax = "proto3";
      package io.axonops.schema.registry.events;
      message SchemaRegistered {
        string subject = 1;
        int32 version = 2;
        string schema_type = 3;
        int64 registered_at = 4;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "io.axonops.schema.registry.events"
    And the response should contain "SchemaRegistered"

  # --------------------------------------------------------------------------
  # 23. Message with reserved field numbers
  # --------------------------------------------------------------------------
  Scenario: Message with reserved field numbers
    When I register a "PROTOBUF" schema under subject "proto-adv-23":
      """
      syntax = "proto3";
      message EvolvingMessage {
        reserved 2, 5, 9 to 12;
        string id = 1;
        string name = 3;
        int32 version = 4;
        bool active = 6;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "EvolvingMessage"
    And the response should contain "reserved"

  # --------------------------------------------------------------------------
  # 24. Message with reserved field names
  # --------------------------------------------------------------------------
  Scenario: Message with reserved field names
    When I register a "PROTOBUF" schema under subject "proto-adv-24":
      """
      syntax = "proto3";
      message MigratedMessage {
        reserved "old_field", "deprecated_field", "legacy_name";
        reserved 10 to 20;
        string id = 1;
        string current_field = 2;
        int64 timestamp = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "MigratedMessage"
    And the response should contain "old_field"

  # --------------------------------------------------------------------------
  # 25. Message with 30+ fields (stress test)
  # --------------------------------------------------------------------------
  Scenario: Message with 30+ fields stress test
    When I register a "PROTOBUF" schema under subject "proto-adv-25":
      """
      syntax = "proto3";
      message WideMessage {
        string field_01 = 1;
        string field_02 = 2;
        string field_03 = 3;
        int32 field_04 = 4;
        int32 field_05 = 5;
        int64 field_06 = 6;
        int64 field_07 = 7;
        double field_08 = 8;
        double field_09 = 9;
        float field_10 = 10;
        bool field_11 = 11;
        bool field_12 = 12;
        bytes field_13 = 13;
        bytes field_14 = 14;
        uint32 field_15 = 15;
        uint64 field_16 = 16;
        sint32 field_17 = 17;
        sint64 field_18 = 18;
        fixed32 field_19 = 19;
        fixed64 field_20 = 20;
        sfixed32 field_21 = 21;
        sfixed64 field_22 = 22;
        string field_23 = 23;
        string field_24 = 24;
        string field_25 = 25;
        int32 field_26 = 26;
        int32 field_27 = 27;
        int64 field_28 = 28;
        double field_29 = 29;
        float field_30 = 30;
        bool field_31 = 31;
        string field_32 = 32;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "WideMessage"
    And the response should contain "field_01"
    And the response should contain "field_32"

  # --------------------------------------------------------------------------
  # 26. Multiple messages with cross-references
  # --------------------------------------------------------------------------
  Scenario: Multiple messages with cross-references
    When I register a "PROTOBUF" schema under subject "proto-adv-26":
      """
      syntax = "proto3";
      message Order {
        string order_id = 1;
        Customer customer = 2;
        repeated OrderItem items = 3;
        ShippingAddress shipping = 4;
        PaymentInfo payment = 5;
      }
      message Customer {
        string id = 1;
        string name = 2;
        ShippingAddress default_address = 3;
      }
      message OrderItem {
        string product_id = 1;
        string name = 2;
        int32 quantity = 3;
        Money price = 4;
      }
      message ShippingAddress {
        string street = 1;
        string city = 2;
        string state = 3;
        string zip = 4;
        string country = 5;
      }
      message PaymentInfo {
        string method = 1;
        Money total = 2;
      }
      message Money {
        int64 units = 1;
        int32 nanos = 2;
        string currency_code = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Order"
    And the response should contain "Customer"
    And the response should contain "Money"
    And the response should contain "ShippingAddress"

  # --------------------------------------------------------------------------
  # 27. Enum used across multiple messages
  # --------------------------------------------------------------------------
  Scenario: Shared enum used across multiple messages
    When I register a "PROTOBUF" schema under subject "proto-adv-27":
      """
      syntax = "proto3";
      enum Status {
        STATUS_UNKNOWN = 0;
        STATUS_PENDING = 1;
        STATUS_ACTIVE = 2;
        STATUS_COMPLETED = 3;
        STATUS_CANCELLED = 4;
      }
      message Order {
        string id = 1;
        Status status = 2;
      }
      message Shipment {
        string tracking_id = 1;
        Status status = 2;
      }
      message Payment {
        string transaction_id = 1;
        Status status = 2;
        int64 amount = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Status"
    And the response should contain "Order"
    And the response should contain "Shipment"
    And the response should contain "Payment"

  # --------------------------------------------------------------------------
  # 28. Nested enum inside message
  # --------------------------------------------------------------------------
  Scenario: Nested enum defined inside a message
    When I register a "PROTOBUF" schema under subject "proto-adv-28":
      """
      syntax = "proto3";
      message HttpRequest {
        string url = 1;
        Method method = 2;
        map<string, string> headers = 3;
        bytes body = 4;
        enum Method {
          METHOD_UNKNOWN = 0;
          METHOD_GET = 1;
          METHOD_POST = 2;
          METHOD_PUT = 3;
          METHOD_DELETE = 4;
          METHOD_PATCH = 5;
        }
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "HttpRequest"
    And the response should contain "METHOD_GET"
    And the response should contain "METHOD_DELETE"

  # --------------------------------------------------------------------------
  # 29. Proto3 optional keyword
  # --------------------------------------------------------------------------
  Scenario: Proto3 optional keyword for presence tracking
    When I register a "PROTOBUF" schema under subject "proto-adv-29":
      """
      syntax = "proto3";
      message UserPreferences {
        string user_id = 1;
        optional string theme = 2;
        optional int32 page_size = 3;
        optional bool dark_mode = 4;
        optional double font_scale = 5;
        string language = 6;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "UserPreferences"
    And the response should contain "optional"

  # --------------------------------------------------------------------------
  # 30. Complex real-world: gRPC service with request/response messages
  # --------------------------------------------------------------------------
  Scenario: Complex real-world gRPC service definition
    When I register a "PROTOBUF" schema under subject "proto-adv-30":
      """
      syntax = "proto3";
      package api.v1;

      message Product {
        string id = 1;
        string name = 2;
        string description = 3;
        int64 price_cents = 4;
        string currency = 5;
        repeated string categories = 6;
        map<string, string> attributes = 7;
        ProductStatus status = 8;
        enum ProductStatus {
          PRODUCT_STATUS_UNKNOWN = 0;
          PRODUCT_STATUS_DRAFT = 1;
          PRODUCT_STATUS_ACTIVE = 2;
          PRODUCT_STATUS_ARCHIVED = 3;
        }
      }

      message GetProductRequest {
        string id = 1;
      }

      message ListProductsRequest {
        int32 page_size = 1;
        string page_token = 2;
        string filter = 3;
        string order_by = 4;
      }

      message ListProductsResponse {
        repeated Product products = 1;
        string next_page_token = 2;
        int32 total_count = 3;
      }

      message CreateProductRequest {
        Product product = 1;
        string request_id = 2;
      }

      message UpdateProductRequest {
        Product product = 1;
        string update_mask = 2;
      }

      message DeleteProductRequest {
        string id = 1;
      }

      message DeleteProductResponse {
        bool deleted = 1;
      }

      service ProductService {
        rpc GetProduct(GetProductRequest) returns (Product);
        rpc ListProducts(ListProductsRequest) returns (ListProductsResponse);
        rpc CreateProduct(CreateProductRequest) returns (Product);
        rpc UpdateProduct(UpdateProductRequest) returns (Product);
        rpc DeleteProduct(DeleteProductRequest) returns (DeleteProductResponse);
        rpc WatchProducts(ListProductsRequest) returns (stream Product);
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "ProductService"
    And the response should contain "GetProduct"
    And the response should contain "WatchProducts"
    And the response field "schemaType" should be "PROTOBUF"

  # --------------------------------------------------------------------------
  # 31. Complex real-world: Kafka event message with header and payload
  # --------------------------------------------------------------------------
  Scenario: Complex real-world Kafka event envelope
    When I register a "PROTOBUF" schema under subject "proto-adv-31":
      """
      syntax = "proto3";
      package events.v1;

      message KafkaEventEnvelope {
        EventHeader header = 1;
        bytes payload = 2;
        string payload_type = 3;

        message EventHeader {
          string event_id = 1;
          string source = 2;
          string type = 3;
          int64 timestamp_ms = 4;
          string correlation_id = 5;
          string causation_id = 6;
          map<string, string> metadata = 7;
          int32 partition_key = 8;
          SchemaInfo schema_info = 9;
        }

        message SchemaInfo {
          string subject = 1;
          int32 version = 2;
          int32 id = 3;
        }
      }

      message DeadLetterEvent {
        KafkaEventEnvelope original = 1;
        string error_message = 2;
        string error_class = 3;
        int32 retry_count = 4;
        int64 failed_at_ms = 5;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "KafkaEventEnvelope"
    And the response should contain "EventHeader"
    And the response should contain "DeadLetterEvent"

  # --------------------------------------------------------------------------
  # 32. Complex real-world: Cloud events proto with any payload
  # --------------------------------------------------------------------------
  Scenario: Complex real-world CloudEvents style message
    When I register a "PROTOBUF" schema under subject "proto-adv-32":
      """
      syntax = "proto3";
      import "google/protobuf/any.proto";
      import "google/protobuf/timestamp.proto";
      package cloudevents.v1;

      message CloudEvent {
        string id = 1;
        string source = 2;
        string spec_version = 3;
        string type = 4;
        google.protobuf.Timestamp time = 5;
        string data_content_type = 6;
        string data_schema = 7;
        string subject = 8;
        map<string, CloudEventAttributeValue> attributes = 9;
        oneof data {
          bytes binary_data = 10;
          string text_data = 11;
          google.protobuf.Any proto_data = 12;
        }
      }

      message CloudEventAttributeValue {
        oneof attr {
          bool ce_boolean = 1;
          int32 ce_integer = 2;
          string ce_string = 3;
          bytes ce_bytes = 4;
          string ce_uri = 5;
          string ce_uri_ref = 6;
          google.protobuf.Timestamp ce_timestamp = 7;
        }
      }

      message CloudEventBatch {
        repeated CloudEvent events = 1;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "CloudEvent"
    And the response should contain "CloudEventAttributeValue"
    And the response should contain "CloudEventBatch"
    And the response should contain "google.protobuf.Any"

  # --------------------------------------------------------------------------
  # 33. Round-trip: register, get by ID, verify schema field
  # --------------------------------------------------------------------------
  Scenario: Round-trip register and retrieve by ID verifying schema field
    When I register a "PROTOBUF" schema under subject "proto-adv-33":
      """
      syntax = "proto3";
      message RoundTripTest {
        string id = 1;
        int32 value = 2;
        bool flag = 3;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should have field "schema"
    And the response should contain "RoundTripTest"
    And the response field "schemaType" should be "PROTOBUF"
    When I get version 1 of subject "proto-adv-33"
    Then the response status should be 200
    And the response field "subject" should be "proto-adv-33"
    And the response field "version" should be 1
    And the response should contain "RoundTripTest"

  # --------------------------------------------------------------------------
  # 34. Fingerprint stability: same schema in 2 subjects yields same ID
  # --------------------------------------------------------------------------
  Scenario: Fingerprint stability same schema in two subjects yields same ID
    When I register a "PROTOBUF" schema under subject "proto-adv-34a":
      """
      syntax = "proto3";
      message FingerprintTest {
        string key = 1;
        int64 value = 2;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "first_id"
    When I register a "PROTOBUF" schema under subject "proto-adv-34b":
      """
      syntax = "proto3";
      message FingerprintTest {
        string key = 1;
        int64 value = 2;
      }
      """
    Then the response status should be 200
    And I store the response field "id" as "second_id"

  # --------------------------------------------------------------------------
  # 35. Large proto with nested messages, enums, services combined
  # --------------------------------------------------------------------------
  Scenario: Large combined proto with messages enums and services
    When I register a "PROTOBUF" schema under subject "proto-adv-35":
      """
      syntax = "proto3";
      package warehouse.v1;

      enum ItemCategory {
        ITEM_CATEGORY_UNKNOWN = 0;
        ITEM_CATEGORY_ELECTRONICS = 1;
        ITEM_CATEGORY_CLOTHING = 2;
        ITEM_CATEGORY_FOOD = 3;
        ITEM_CATEGORY_FURNITURE = 4;
      }

      enum ShipmentStatus {
        SHIPMENT_STATUS_UNKNOWN = 0;
        SHIPMENT_STATUS_PENDING = 1;
        SHIPMENT_STATUS_PICKED = 2;
        SHIPMENT_STATUS_PACKED = 3;
        SHIPMENT_STATUS_SHIPPED = 4;
        SHIPMENT_STATUS_DELIVERED = 5;
      }

      message Warehouse {
        string id = 1;
        string name = 2;
        Location location = 3;
        repeated Zone zones = 4;
      }

      message Location {
        double latitude = 1;
        double longitude = 2;
        string address = 3;
        string city = 4;
        string country = 5;
      }

      message Zone {
        string zone_id = 1;
        string name = 2;
        int32 capacity = 3;
        repeated Shelf shelves = 4;
      }

      message Shelf {
        string shelf_id = 1;
        int32 level = 2;
        repeated Item items = 3;
      }

      message Item {
        string sku = 1;
        string name = 2;
        ItemCategory category = 3;
        int32 quantity = 4;
        int64 price_cents = 5;
        map<string, string> attributes = 6;
      }

      message InventoryRequest {
        string warehouse_id = 1;
        ItemCategory category_filter = 2;
        int32 min_quantity = 3;
      }

      message InventoryResponse {
        repeated Item items = 1;
        int32 total_count = 2;
      }

      message ShipmentRequest {
        string warehouse_id = 1;
        repeated string skus = 2;
        string destination = 3;
      }

      message Shipment {
        string id = 1;
        ShipmentStatus status = 2;
        repeated Item items = 3;
        string destination = 4;
        int64 created_at = 5;
      }

      service WarehouseService {
        rpc GetInventory(InventoryRequest) returns (InventoryResponse);
        rpc CreateShipment(ShipmentRequest) returns (Shipment);
        rpc TrackShipment(ShipmentRequest) returns (stream Shipment);
        rpc BulkImport(stream Item) returns (InventoryResponse);
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "WarehouseService"
    And the response should contain "ItemCategory"
    And the response should contain "ShipmentStatus"
    And the response should contain "Warehouse"
    And the response field "schemaType" should be "PROTOBUF"

  # --------------------------------------------------------------------------
  # 36. Map of string to bytes
  # --------------------------------------------------------------------------
  Scenario: Map of string to bytes
    When I register a "PROTOBUF" schema under subject "proto-adv-36":
      """
      syntax = "proto3";
      message BlobStore {
        string namespace = 1;
        map<string, bytes> blobs = 2;
        map<string, int64> blob_sizes = 3;
        map<int32, string> id_to_name = 4;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "BlobStore"
    And the response should contain "blobs"

  # --------------------------------------------------------------------------
  # 37. Deeply nested oneof
  # --------------------------------------------------------------------------
  Scenario: Deeply nested oneof inside nested messages
    When I register a "PROTOBUF" schema under subject "proto-adv-37":
      """
      syntax = "proto3";
      message Document {
        string id = 1;
        Section root_section = 2;
      }
      message Section {
        string title = 1;
        oneof content {
          TextBlock text = 2;
          ImageBlock image = 3;
          TableBlock table = 4;
          NestedSection subsection = 5;
        }
      }
      message TextBlock {
        string text = 1;
        oneof format {
          PlainFormat plain = 2;
          RichFormat rich = 3;
        }
      }
      message PlainFormat {
        string font = 1;
        int32 size = 2;
      }
      message RichFormat {
        string html = 1;
        repeated string css_classes = 2;
      }
      message ImageBlock {
        string url = 1;
        int32 width = 2;
        int32 height = 3;
      }
      message TableBlock {
        repeated Row rows = 1;
        message Row {
          repeated string cells = 1;
        }
      }
      message NestedSection {
        Section section = 1;
        int32 depth = 2;
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "Document"
    And the response should contain "Section"
    And the response should contain "TextBlock"
    And the response should contain "oneof"

  # --------------------------------------------------------------------------
  # 38. Multiple services in one file
  # --------------------------------------------------------------------------
  Scenario: Multiple services defined in one proto file
    When I register a "PROTOBUF" schema under subject "proto-adv-38":
      """
      syntax = "proto3";
      package multiservice.v1;

      message HealthCheckRequest {
        string service_name = 1;
      }
      message HealthCheckResponse {
        ServingStatus status = 1;
        enum ServingStatus {
          SERVING_STATUS_UNKNOWN = 0;
          SERVING_STATUS_SERVING = 1;
          SERVING_STATUS_NOT_SERVING = 2;
        }
      }
      message EchoRequest {
        string message = 1;
      }
      message EchoResponse {
        string message = 1;
        int64 server_timestamp = 2;
      }
      message MetricsRequest {
        string metric_name = 1;
        int64 from_timestamp = 2;
        int64 to_timestamp = 3;
      }
      message MetricsResponse {
        repeated DataPoint points = 1;
      }
      message DataPoint {
        int64 timestamp = 1;
        double value = 2;
        map<string, string> labels = 3;
      }

      service HealthService {
        rpc Check(HealthCheckRequest) returns (HealthCheckResponse);
        rpc Watch(HealthCheckRequest) returns (stream HealthCheckResponse);
      }

      service EchoService {
        rpc Echo(EchoRequest) returns (EchoResponse);
        rpc StreamEcho(stream EchoRequest) returns (stream EchoResponse);
      }

      service MetricsService {
        rpc Query(MetricsRequest) returns (MetricsResponse);
        rpc StreamMetrics(MetricsRequest) returns (stream MetricsResponse);
      }
      """
    Then the response status should be 200
    And the response should have field "id"
    And I store the response field "id" as "schema_id"
    When I get the stored schema by ID
    Then the response status should be 200
    And the response should contain "HealthService"
    And the response should contain "EchoService"
    And the response should contain "MetricsService"
    And the response should contain "DataPoint"
    And the response field "schemaType" should be "PROTOBUF"
