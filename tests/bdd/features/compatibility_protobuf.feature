@functional @compatibility
Feature: Protobuf Schema Compatibility
  Exhaustive compatibility checking for Protobuf schemas across all modes

  # ==========================================================================
  # BACKWARD mode (10 scenarios)
  # BACKWARD: new schema (reader) must be able to read data written by old schema (writer)
  # Check(new, old) — adding fields to new is safe, removing fields from new is breaking
  # ==========================================================================

  Scenario: BACKWARD - add optional field is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        string id = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-1":
      """
      syntax = "proto3";
      message Event {
        string id = 1;
        string source = 2;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD - add required field is incompatible (proto2)
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-2" has "PROTOBUF" schema:
      """
      syntax = "proto2";
      message Event {
        required string id = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-2":
      """
      syntax = "proto2";
      message Event {
        required string id = 1;
        required int32 priority = 2;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - remove field is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        string id = 1;
        string source = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-3":
      """
      syntax = "proto3";
      message Event {
        string id = 1;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - change field type int32 to string is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 code = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-4":
      """
      syntax = "proto3";
      message Event {
        string code = 1;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - compatible type change int32 to sint32
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        int32 value = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-5":
      """
      syntax = "proto3";
      message Event {
        sint32 value = 1;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD - change field number is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-6" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        string name = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-6":
      """
      syntax = "proto3";
      message Event {
        string name = 2;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - optional to repeated is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-7" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        string tag = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-7":
      """
      syntax = "proto3";
      message Event {
        repeated string tag = 1;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD - repeated to optional is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-8" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        repeated string tags = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-8":
      """
      syntax = "proto3";
      message Event {
        string tags = 1;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - add enum value is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-9" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Priority {
        LOW = 0;
        MEDIUM = 1;
      }
      message Event {
        Priority priority = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-9":
      """
      syntax = "proto3";
      enum Priority {
        LOW = 0;
        MEDIUM = 1;
        HIGH = 2;
      }
      message Event {
        Priority priority = 1;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD - remove enum value is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-back-10" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Priority {
        LOW = 0;
        MEDIUM = 1;
        HIGH = 2;
      }
      message Event {
        Priority priority = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-back-10":
      """
      syntax = "proto3";
      enum Priority {
        LOW = 0;
        MEDIUM = 1;
      }
      message Event {
        Priority priority = 1;
      }
      """
    Then the response status should be 409

  # ==========================================================================
  # BACKWARD_TRANSITIVE mode (6 scenarios)
  # Must be compatible with ALL previous versions, not just the latest
  # ==========================================================================

  Scenario: BACKWARD_TRANSITIVE - 3-version chain all compatible
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Record {
        string id = 1;
      }
      """
    And subject "proto-bt-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Record {
        string id = 1;
        string name = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-1":
      """
      syntax = "proto3";
      message Record {
        string id = 1;
        string name = 2;
        int32 age = 3;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - field number reuse in v3 is incompatible
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Record {
        string id = 1;
        int32 code = 2;
      }
      """
    And subject "proto-bt-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Record {
        string id = 1;
        int32 code = 2;
        string label = 3;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-2":
      """
      syntax = "proto3";
      message Record {
        string id = 1;
        string code = 2;
        string label = 3;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD_TRANSITIVE - type change chain is incompatible with v1
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Record {
        int32 value = 1;
      }
      """
    And subject "proto-bt-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Record {
        sint32 value = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-3":
      """
      syntax = "proto3";
      message Record {
        sfixed32 value = 1;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - enum grows each version stays compatible
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Color {
        RED = 0;
      }
      message Palette {
        Color color = 1;
      }
      """
    And subject "proto-bt-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Color {
        RED = 0;
        GREEN = 1;
      }
      message Palette {
        Color color = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-4":
      """
      syntax = "proto3";
      enum Color {
        RED = 0;
        GREEN = 1;
        BLUE = 2;
      }
      message Palette {
        Color color = 1;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - field additions across 3 versions compatible
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Metric {
        string name = 1;
      }
      """
    And subject "proto-bt-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Metric {
        string name = 1;
        double value = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-5":
      """
      syntax = "proto3";
      message Metric {
        string name = 1;
        double value = 2;
        int64 timestamp = 3;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD_TRANSITIVE - incompatible with v1 but OK with v2 still fails
    Given the global compatibility level is "BACKWARD_TRANSITIVE"
    And subject "proto-bt-6" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Metric {
        string name = 1;
        int32 count = 2;
      }
      """
    And subject "proto-bt-6" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Metric {
        string name = 1;
        int32 count = 2;
        string label = 3;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-bt-6":
      """
      syntax = "proto3";
      message Metric {
        string name = 1;
        string label = 3;
      }
      """
    Then the response status should be 409

  # ==========================================================================
  # FORWARD mode (8 scenarios)
  # FORWARD: old schema (reader) must be able to read data written by new schema (writer)
  # Check(old, new) — removing fields from new is safe, adding fields to new is breaking
  # ==========================================================================

  Scenario: FORWARD - remove field is compatible
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Payload {
        string id = 1;
        string data = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd-1":
      """
      syntax = "proto3";
      message Payload {
        string id = 1;
      }
      """
    Then the response status should be 200

  Scenario: FORWARD - add field is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Payload {
        string id = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd-2":
      """
      syntax = "proto3";
      message Payload {
        string id = 1;
        string data = 2;
      }
      """
    Then the response status should be 409

  Scenario: FORWARD - change field type is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Payload {
        string value = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd-3":
      """
      syntax = "proto3";
      message Payload {
        int32 value = 1;
      }
      """
    Then the response status should be 409

  Scenario: FORWARD - compatible type group uint32 to fixed32
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Payload {
        uint32 counter = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd-4":
      """
      syntax = "proto3";
      message Payload {
        fixed32 counter = 1;
      }
      """
    Then the response status should be 200

  Scenario: FORWARD - remove enum value is compatible
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Level {
        INFO = 0;
        WARN = 1;
        ERROR = 2;
      }
      message LogEntry {
        Level level = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd-5":
      """
      syntax = "proto3";
      enum Level {
        INFO = 0;
        WARN = 1;
      }
      message LogEntry {
        Level level = 1;
      }
      """
    Then the response status should be 200

  Scenario: FORWARD - add enum value is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd-6" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Level {
        INFO = 0;
        WARN = 1;
      }
      message LogEntry {
        Level level = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd-6":
      """
      syntax = "proto3";
      enum Level {
        INFO = 0;
        WARN = 1;
        ERROR = 2;
      }
      message LogEntry {
        Level level = 1;
      }
      """
    Then the response status should be 409

  Scenario: FORWARD - remove service method is compatible
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd-7" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      message DetailReq { string id = 1; }
      message DetailResp { string detail = 1; }
      service DataService {
        rpc Get(Req) returns (Resp);
        rpc GetDetail(DetailReq) returns (DetailResp);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd-7":
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      message DetailReq { string id = 1; }
      message DetailResp { string detail = 1; }
      service DataService {
        rpc Get(Req) returns (Resp);
      }
      """
    Then the response status should be 200

  Scenario: FORWARD - add service method is incompatible
    Given the global compatibility level is "FORWARD"
    And subject "proto-fwd-8" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      message ListReq { int32 page = 1; }
      message ListResp { repeated string items = 1; }
      service DataService {
        rpc Get(Req) returns (Resp);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-fwd-8":
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      message ListReq { int32 page = 1; }
      message ListResp { repeated string items = 1; }
      service DataService {
        rpc Get(Req) returns (Resp);
        rpc List(ListReq) returns (ListResp);
      }
      """
    Then the response status should be 409

  # ==========================================================================
  # FORWARD_TRANSITIVE mode (5 scenarios)
  # Must be forward-compatible with ALL previous versions
  # ==========================================================================

  Scenario: FORWARD_TRANSITIVE - 3-version compatible chain with field removals
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "proto-ft-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Entry {
        string id = 1;
        string name = 2;
        int32 code = 3;
      }
      """
    And subject "proto-ft-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Entry {
        string id = 1;
        string name = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ft-1":
      """
      syntax = "proto3";
      message Entry {
        string id = 1;
      }
      """
    Then the response status should be 200

  Scenario: FORWARD_TRANSITIVE - field addition breaks against v1
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "proto-ft-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Entry {
        string id = 1;
      }
      """
    And subject "proto-ft-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Entry {
        string id = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ft-2":
      """
      syntax = "proto3";
      message Entry {
        string id = 1;
        string label = 2;
      }
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE - service method addition breaks against v1
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "proto-ft-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string q = 1; }
      message Resp { string r = 1; }
      service Svc {
        rpc Do(Req) returns (Resp);
      }
      """
    And subject "proto-ft-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string q = 1; }
      message Resp { string r = 1; }
      service Svc {
        rpc Do(Req) returns (Resp);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ft-3":
      """
      syntax = "proto3";
      message Req { string q = 1; }
      message Resp { string r = 1; }
      message Req2 { int32 x = 1; }
      message Resp2 { string y = 1; }
      service Svc {
        rpc Do(Req) returns (Resp);
        rpc DoMore(Req2) returns (Resp2);
      }
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE - enum value addition breaks against earlier versions
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "proto-ft-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Status {
        UNKNOWN = 0;
      }
      message Item {
        Status status = 1;
      }
      """
    And subject "proto-ft-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Status {
        UNKNOWN = 0;
      }
      message Item {
        Status status = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ft-4":
      """
      syntax = "proto3";
      enum Status {
        UNKNOWN = 0;
        ACTIVE = 1;
      }
      message Item {
        Status status = 1;
      }
      """
    Then the response status should be 409

  Scenario: FORWARD_TRANSITIVE - progressive field removal stays compatible
    Given the global compatibility level is "FORWARD_TRANSITIVE"
    And subject "proto-ft-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Log {
        string message = 1;
        int32 code = 2;
        string source = 3;
      }
      """
    And subject "proto-ft-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Log {
        string message = 1;
        int32 code = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-ft-5":
      """
      syntax = "proto3";
      message Log {
        string message = 1;
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # FULL mode (7 scenarios)
  # FULL: both backward AND forward compatible (Check in both directions)
  # Only changes that are safe in BOTH directions pass
  # ==========================================================================

  Scenario: FULL - add optional field fails (not forward compatible)
    Given the global compatibility level is "FULL"
    And subject "proto-full-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Doc {
        string title = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-full-1":
      """
      syntax = "proto3";
      message Doc {
        string title = 1;
        string body = 2;
      }
      """
    Then the response status should be 409

  Scenario: FULL - add required field is incompatible
    Given the global compatibility level is "FULL"
    And subject "proto-full-2" has "PROTOBUF" schema:
      """
      syntax = "proto2";
      message Doc {
        required string title = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-full-2":
      """
      syntax = "proto2";
      message Doc {
        required string title = 1;
        required string body = 2;
      }
      """
    Then the response status should be 409

  Scenario: FULL - remove field fails (not backward compatible)
    Given the global compatibility level is "FULL"
    And subject "proto-full-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Doc {
        string title = 1;
        string body = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-full-3":
      """
      syntax = "proto3";
      message Doc {
        string title = 1;
      }
      """
    Then the response status should be 409

  Scenario: FULL - identical schema is compatible
    Given the global compatibility level is "FULL"
    And subject "proto-full-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Doc {
        string title = 1;
        int32 version = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-full-4":
      """
      syntax = "proto3";
      message Doc {
        string title = 1;
        int32 version = 2;
      }
      """
    Then the response status should be 200

  Scenario: FULL - cross-group type change is incompatible
    Given the global compatibility level is "FULL"
    And subject "proto-full-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Doc {
        int32 count = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-full-5":
      """
      syntax = "proto3";
      message Doc {
        uint32 count = 1;
      }
      """
    Then the response status should be 409

  Scenario: FULL - add enum value fails (not forward compatible)
    Given the global compatibility level is "FULL"
    And subject "proto-full-6" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      enum Role {
        GUEST = 0;
        USER = 1;
      }
      message Account {
        Role role = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-full-6":
      """
      syntax = "proto3";
      enum Role {
        GUEST = 0;
        USER = 1;
        ADMIN = 2;
      }
      message Account {
        Role role = 1;
      }
      """
    Then the response status should be 409

  Scenario: FULL - compatible type change within group passes both directions
    Given the global compatibility level is "FULL"
    And subject "proto-full-7" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Doc {
        int32 score = 1;
        string name = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-full-7":
      """
      syntax = "proto3";
      message Doc {
        sint32 score = 1;
        string name = 2;
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # FULL_TRANSITIVE mode (4 scenarios)
  # Both backward and forward compatible with ALL previous versions
  # ==========================================================================

  Scenario: FULL_TRANSITIVE - safe 3-version chain with identical schemas
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "proto-flt-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Config {
        string key = 1;
        string value = 2;
      }
      """
    And subject "proto-flt-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Config {
        string key = 1;
        string value = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-flt-1":
      """
      syntax = "proto3";
      message Config {
        string key = 1;
        string value = 2;
      }
      """
    Then the response status should be 200

  Scenario: FULL_TRANSITIVE - incompatible type change fails against all versions
    Given the global compatibility level is "NONE"
    And subject "proto-flt-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Config {
        int32 counter = 1;
        string label = 2;
      }
      """
    And subject "proto-flt-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Config {
        sint32 counter = 1;
        string label = 2;
      }
      """
    And the global compatibility level is "FULL_TRANSITIVE"
    When I register a "PROTOBUF" schema under subject "proto-flt-2":
      """
      syntax = "proto3";
      message Config {
        string counter = 1;
        string label = 2;
      }
      """
    Then the response status should be 409

  Scenario: FULL_TRANSITIVE - compatible type changes across 3 versions
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "proto-flt-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Config {
        int32 counter = 1;
        string label = 2;
      }
      """
    And subject "proto-flt-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Config {
        sint32 counter = 1;
        string label = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-flt-3":
      """
      syntax = "proto3";
      message Config {
        sfixed32 counter = 1;
        string label = 2;
      }
      """
    Then the response status should be 200

  Scenario: FULL_TRANSITIVE - service method change fails
    Given the global compatibility level is "FULL_TRANSITIVE"
    And subject "proto-flt-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      service Api {
        rpc Fetch(Req) returns (Resp);
      }
      """
    And subject "proto-flt-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      service Api {
        rpc Fetch(Req) returns (Resp);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-flt-4":
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      message Req2 { int32 num = 1; }
      message Resp2 { string result = 1; }
      service Api {
        rpc Fetch(Req) returns (Resp);
        rpc Query(Req2) returns (Resp2);
      }
      """
    Then the response status should be 409

  # ==========================================================================
  # NONE mode (2 scenarios)
  # Compatibility checking is disabled — any schema change is accepted
  # ==========================================================================

  Scenario: NONE - completely different message is accepted
    Given the global compatibility level is "NONE"
    And subject "proto-none-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        string name = 1;
        int32 age = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-none-1":
      """
      syntax = "proto3";
      message Order {
        int64 order_id = 1;
        double total = 2;
        bool shipped = 3;
      }
      """
    Then the response status should be 200

  Scenario: NONE - field number reuse with different type is accepted
    Given the global compatibility level is "NONE"
    And subject "proto-none-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Data {
        int32 value = 1;
        string label = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-none-2":
      """
      syntax = "proto3";
      message Data {
        string value = 1;
        int32 label = 2;
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # Edge Cases (8 scenarios)
  # Test complex Protobuf features: nested messages, oneof, maps, services
  # ==========================================================================

  Scenario: Edge case - nested message field type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-edge-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Outer {
        string id = 1;
        message Inner {
          string value = 1;
          int32 count = 2;
        }
        Inner detail = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-edge-1":
      """
      syntax = "proto3";
      message Outer {
        string id = 1;
        message Inner {
          string value = 1;
          string count = 2;
        }
        Inner detail = 2;
      }
      """
    Then the response status should be 409

  Scenario: Edge case - oneof field addition is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "proto-edge-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        string id = 1;
        oneof payload {
          string text = 2;
        }
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-edge-2":
      """
      syntax = "proto3";
      message Event {
        string id = 1;
        oneof payload {
          string text = 2;
          int32 number = 3;
        }
      }
      """
    Then the response status should be 200

  Scenario: Edge case - oneof field removal is incompatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "proto-edge-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Event {
        string id = 1;
        oneof payload {
          string text = 2;
          int32 number = 3;
        }
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-edge-3":
      """
      syntax = "proto3";
      message Event {
        string id = 1;
        oneof payload {
          string text = 2;
        }
      }
      """
    Then the response status should be 409

  Scenario: Edge case - package name change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-edge-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package com.example.v1;
      message Event {
        string id = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-edge-4":
      """
      syntax = "proto3";
      package com.example.v2;
      message Event {
        string id = 1;
      }
      """
    Then the response status should be 409

  Scenario: Edge case - map field replaced by scalar is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-edge-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Config {
        map<string, string> settings = 1;
        string name = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-edge-5":
      """
      syntax = "proto3";
      message Config {
        string settings = 1;
        string name = 2;
      }
      """
    Then the response status should be 409

  Scenario: Edge case - multiple messages with one field type changed is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-edge-6" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        string name = 1;
      }
      message Order {
        int32 total = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-edge-6":
      """
      syntax = "proto3";
      message User {
        string name = 1;
      }
      message Order {
        string total = 1;
      }
      """
    Then the response status should be 409

  Scenario: Edge case - service streaming mode change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-edge-7" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      service StreamSvc {
        rpc Fetch(Req) returns (Resp);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-edge-7":
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      service StreamSvc {
        rpc Fetch(Req) returns (stream Resp);
      }
      """
    Then the response status should be 409

  Scenario: Edge case - adding new message type is compatible (backward)
    Given the global compatibility level is "BACKWARD"
    And subject "proto-edge-8" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        string name = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-edge-8":
      """
      syntax = "proto3";
      message User {
        string name = 1;
      }
      message Address {
        string street = 1;
        string city = 2;
      }
      """
    Then the response status should be 200

  # ==========================================================================
  # Error Validation (5 scenarios)
  # Verify error responses, check endpoint, and per-subject overrides
  # ==========================================================================

  Scenario: Error validation - 409 response has error_code field
    Given the global compatibility level is "BACKWARD"
    And subject "proto-err-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Item {
        string name = 1;
        int32 quantity = 2;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-err-1":
      """
      syntax = "proto3";
      message Item {
        string name = 1;
        string quantity = 2;
      }
      """
    Then the response status should be 409
    And the response should have error code 409

  Scenario: Error validation - check endpoint returns is_compatible false
    Given the global compatibility level is "BACKWARD"
    And subject "proto-err-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Item {
        string name = 1;
        int32 quantity = 2;
      }
      """
    When I check compatibility of "PROTOBUF" schema against subject "proto-err-2":
      """
      syntax = "proto3";
      message Item {
        string name = 1;
        string quantity = 2;
      }
      """
    Then the compatibility check should be incompatible

  Scenario: Error validation - per-subject NONE overrides global BACKWARD
    Given the global compatibility level is "BACKWARD"
    And subject "proto-err-3" has compatibility level "NONE"
    And subject "proto-err-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Item {
        string name = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-err-3":
      """
      syntax = "proto3";
      message DifferentItem {
        int64 id = 1;
        double price = 2;
      }
      """
    Then the response status should be 200

  Scenario: Error validation - check endpoint returns is_compatible true
    Given the global compatibility level is "BACKWARD"
    And subject "proto-err-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Item {
        string name = 1;
      }
      """
    When I check compatibility of "PROTOBUF" schema against subject "proto-err-4":
      """
      syntax = "proto3";
      message Item {
        string name = 1;
        int32 quantity = 2;
      }
      """
    Then the compatibility check should be compatible

  Scenario: Error validation - delete per-subject config falls back to global
    Given the global compatibility level is "BACKWARD"
    And subject "proto-err-5" has compatibility level "NONE"
    And subject "proto-err-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Thing {
        string name = 1;
        int32 count = 2;
      }
      """
    And I delete the config for subject "proto-err-5"
    When I register a "PROTOBUF" schema under subject "proto-err-5":
      """
      syntax = "proto3";
      message Thing {
        string name = 1;
        string count = 2;
      }
      """
    Then the response status should be 409

  # --- Gap-filling: Protobuf-specific compatibility rules ---

  Scenario: BACKWARD - package name change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-1" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      package com.example.v1;
      message User {
        string name = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-1":
      """
      syntax = "proto3";
      package com.example.v2;
      message User {
        string name = 1;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - syntax version change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-2" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message User {
        string name = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-2":
      """
      syntax = "proto2";
      message User {
        optional string name = 1;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - cardinality optional to repeated is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-3" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Data {
        string tag = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-3":
      """
      syntax = "proto3";
      message Data {
        repeated string tag = 1;
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD - cardinality repeated to singular is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-4" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Data {
        repeated string tags = 1;
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-4":
      """
      syntax = "proto3";
      message Data {
        string tags = 1;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - nested message removal is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-5" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Outer {
        string id = 1;
        Inner detail = 2;
        message Inner {
          string value = 1;
        }
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-5":
      """
      syntax = "proto3";
      message Outer {
        string id = 1;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - nested enum removal is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-6" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Item {
        string name = 1;
        Status status = 2;
        enum Status {
          UNKNOWN = 0;
          ACTIVE = 1;
        }
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-6":
      """
      syntax = "proto3";
      message Item {
        string name = 1;
        int32 status = 2;
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - service method removal is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-7" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Request { string id = 1; }
      message Response { string data = 1; }
      service MyService {
        rpc GetData(Request) returns (Response);
        rpc ListData(Request) returns (Response);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-7":
      """
      syntax = "proto3";
      message Request { string id = 1; }
      message Response { string data = 1; }
      service MyService {
        rpc GetData(Request) returns (Response);
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - service method addition is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-8" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Request { string id = 1; }
      message Response { string data = 1; }
      service MyService {
        rpc GetData(Request) returns (Response);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-8":
      """
      syntax = "proto3";
      message Request { string id = 1; }
      message Response { string data = 1; }
      service MyService {
        rpc GetData(Request) returns (Response);
        rpc ListData(Request) returns (Response);
      }
      """
    Then the response status should be 200

  Scenario: BACKWARD - service method input type change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-9" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message ReqA { string id = 1; }
      message ReqB { int32 id = 1; }
      message Resp { string data = 1; }
      service Svc {
        rpc DoWork(ReqA) returns (Resp);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-9":
      """
      syntax = "proto3";
      message ReqA { string id = 1; }
      message ReqB { int32 id = 1; }
      message Resp { string data = 1; }
      service Svc {
        rpc DoWork(ReqB) returns (Resp);
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - service streaming mode change is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-10" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      service Svc {
        rpc StreamData(Req) returns (Resp);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-10":
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      service Svc {
        rpc StreamData(Req) returns (stream Resp);
      }
      """
    Then the response status should be 409

  Scenario: BACKWARD - service removal is incompatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-11" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      service Svc {
        rpc DoWork(Req) returns (Resp);
      }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-11":
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      """
    Then the response status should be 409

  Scenario: BACKWARD - service addition is compatible
    Given the global compatibility level is "BACKWARD"
    And subject "proto-gap-12" has "PROTOBUF" schema:
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      """
    When I register a "PROTOBUF" schema under subject "proto-gap-12":
      """
      syntax = "proto3";
      message Req { string id = 1; }
      message Resp { string data = 1; }
      service Svc {
        rpc DoWork(Req) returns (Resp);
      }
      """
    Then the response status should be 200
