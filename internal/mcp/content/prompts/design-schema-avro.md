Design an Avro schema following these best practices.

## Design Workflow

1. **Identify the entity or event** -- what does this schema represent?
2. **Choose a namespace** -- use reverse-domain (e.g., `com.company.events`)
3. **Define fields** -- name each field in snake_case, choose appropriate types
4. **Add defaults** -- every new field SHOULD have a default for backward compatibility
5. **Add documentation** -- use the `doc` property on the record and each field
6. **Validate** -- use **validate_schema** to check syntax
7. **Register** -- use **register_schema** with `schema_type: AVRO`

> All registration tools accept the optional `context` parameter for multi-tenant isolation.

## Best Practices

- Use a descriptive record name in PascalCase with a namespace (e.g., `com.company.events.OrderCreated`)
- Use `snake_case` for field names
- Use union types `["null", "type"]` with `"default": null` for optional fields
- Use logical types for dates (`timestamp-millis`), decimals (`bytes` + `decimal`), and UUIDs (`string` + `uuid`)
- Use enums for fixed sets of values (with a sensible default)
- Consider schema evolution: add new fields with defaults, avoid removing or renaming fields
- Keep records focused -- prefer composition over deeply nested structures

## Worked Example: OrderCreated Event

```json
{
  "type": "record",
  "name": "OrderCreated",
  "namespace": "com.company.events",
  "doc": "Emitted when a customer places a new order.",
  "fields": [
    {"name": "event_id", "type": {"type": "string", "logicalType": "uuid"}, "doc": "Unique event identifier"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}, "doc": "Event timestamp in UTC"},
    {"name": "order_id", "type": "string", "doc": "Business order identifier"},
    {"name": "customer_id", "type": "string", "doc": "Customer who placed the order"},
    {"name": "status", "type": {"type": "enum", "name": "OrderStatus", "symbols": ["PENDING", "CONFIRMED", "SHIPPED", "DELIVERED", "CANCELLED"]}, "doc": "Current order status"},
    {"name": "total_cents", "type": "long", "doc": "Order total in smallest currency unit"},
    {"name": "currency", "type": {"type": "string", "default": "USD"}, "doc": "ISO 4217 currency code", "default": "USD"},
    {"name": "shipping_address", "type": ["null", {
      "type": "record", "name": "Address", "fields": [
        {"name": "street", "type": "string"},
        {"name": "city", "type": "string"},
        {"name": "postal_code", "type": "string"},
        {"name": "country", "type": "string"}
      ]
    }], "default": null, "doc": "Shipping address, null for digital orders"},
    {"name": "notes", "type": ["null", "string"], "default": null, "doc": "Optional order notes"}
  ]
}
```

## Common Mistakes

1. **Missing defaults on optional fields** -- `["null", "string"]` without `"default": null` breaks backward compatibility
2. **Using `string` for everything** -- use typed fields (`int`, `long`, `boolean`, enums) for correctness and efficiency
3. **No namespace** -- leads to naming conflicts when schemas reference each other
4. **Changing enum symbol order** -- Avro enums are ordinal; reordering is a breaking change
5. **Deeply nested unions** -- union-within-union is not allowed in Avro; flatten or use separate records

## Starter Template

```json
{
  "type": "record",
  "name": "MyEvent",
  "namespace": "com.company.events",
  "doc": "Description of the event",
  "fields": [
    {"name": "event_id", "type": {"type": "string", "logicalType": "uuid"}},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "my_field", "type": "string", "doc": "Description"}
  ]
}
```

Available tools: register_schema, validate_schema, check_compatibility, get_latest_schema, lookup_schema
