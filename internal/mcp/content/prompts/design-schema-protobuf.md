Design a Protobuf schema following these best practices.

## Design Workflow

1. **Identify the entity or event** -- what does this schema represent?
2. **Choose a package** -- use dot-separated domain (e.g., `company.events.v1`)
3. **Define messages** -- PascalCase names, snake_case fields, assign field numbers
4. **Use well-known types** -- Timestamp, Duration, etc. for standard semantics
5. **Add documentation** -- use `//` comments on messages and fields
6. **Validate** -- use **validate_schema** with `schema_type: PROTOBUF`
7. **Register** -- use **register_schema** with `schema_type: PROTOBUF`

> All registration tools accept the optional `context` parameter for multi-tenant isolation.

## Best Practices

- Use `syntax = "proto3";` (required)
- Use a package declaration matching your domain (e.g., `package company.events.v1;`)
- Use PascalCase for message and enum names, snake_case for field names
- Use explicit field numbers and NEVER reuse deleted field numbers (use `reserved`)
- Use `oneof` for variant/union types
- Use `repeated` for arrays, `map<K,V>` for key-value pairs
- Use well-known types (`google.protobuf.Timestamp`, `Duration`, `Struct`) when appropriate
- Use enums with `UNSPECIFIED = 0` as the first value
- Consider backward compatibility: only add new fields, never change field numbers or types

## Worked Example: OrderCreated Event

```protobuf
syntax = "proto3";

package company.events.v1;

import "google/protobuf/timestamp.proto";

// Emitted when a customer places a new order.
message OrderCreated {
  // Unique event identifier (UUID format)
  string event_id = 1;
  // Event timestamp in UTC
  google.protobuf.Timestamp timestamp = 2;
  // Business order identifier
  string order_id = 3;
  // Customer who placed the order
  string customer_id = 4;

  enum OrderStatus {
    ORDER_STATUS_UNSPECIFIED = 0;
    ORDER_STATUS_PENDING = 1;
    ORDER_STATUS_CONFIRMED = 2;
    ORDER_STATUS_SHIPPED = 3;
    ORDER_STATUS_DELIVERED = 4;
    ORDER_STATUS_CANCELLED = 5;
  }
  OrderStatus status = 5;

  // Order total in smallest currency unit (cents)
  int64 total_cents = 6;
  // ISO 4217 currency code
  string currency = 7;
  // Shipping address, empty for digital orders
  Address shipping_address = 8;
  // Optional order notes
  string notes = 9;
}

message Address {
  string street = 1;
  string city = 2;
  string postal_code = 3;
  string country = 4;
}
```

## Common Mistakes

1. **Reusing deleted field numbers** -- always use `reserved` for removed fields (e.g., `reserved 5, 8;`)
2. **Missing UNSPECIFIED enum value** -- proto3 enums must have a zero value; name it `*_UNSPECIFIED = 0`
3. **Changing field numbers** -- field numbers are part of the wire format; changing them breaks all existing data
4. **Not versioning packages** -- use `.v1`, `.v2` in package names for major API versions
5. **Using `int32` for IDs** -- prefer `int64` or `string` for IDs to avoid overflow

## Starter Template

```protobuf
syntax = "proto3";

package company.events.v1;

import "google/protobuf/timestamp.proto";

// Description of the event.
message MyEvent {
  string event_id = 1;
  google.protobuf.Timestamp timestamp = 2;
  string my_field = 3;
}
```

Available tools: register_schema (with schema_type: PROTOBUF), validate_schema, check_compatibility
