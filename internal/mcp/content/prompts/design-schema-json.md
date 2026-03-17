Design a JSON Schema following these best practices.

## Design Workflow

1. **Identify the entity or event** -- what does this schema represent?
2. **Choose a schema ID** -- use `$id` with a URI (e.g., `https://company.com/schemas/order-created`)
3. **Define properties** -- use descriptive names, choose appropriate types
4. **Mark required fields** -- add field names to the `required` array
5. **Add descriptions** -- use the `description` keyword on the schema and each property
6. **Validate** -- use **validate_schema** with `schema_type: JSON`
7. **Register** -- use **register_schema** with `schema_type: JSON`

> All registration tools accept the optional `context` parameter for multi-tenant isolation.

## Best Practices

- Use `"type": "object"` as the root type
- Define a `required` array listing mandatory fields
- Use `"additionalProperties": false` to prevent unexpected fields
- Use format validators: `email`, `uri`, `date-time`, `uuid`
- Use `pattern` for custom string validation (regex)
- Use `minimum`/`maximum` for number ranges, `minLength`/`maxLength` for strings
- Use `enum` for fixed value sets
- Use `$ref` for reusable type definitions
- Use `oneOf`/`anyOf` for variant types

## Worked Example: OrderCreated Event

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://company.com/schemas/order-created",
  "title": "OrderCreated",
  "description": "Emitted when a customer places a new order.",
  "type": "object",
  "required": ["event_id", "timestamp", "order_id", "customer_id", "status", "total_cents"],
  "properties": {
    "event_id": {
      "type": "string",
      "format": "uuid",
      "description": "Unique event identifier"
    },
    "timestamp": {
      "type": "string",
      "format": "date-time",
      "description": "Event timestamp in UTC (ISO 8601)"
    },
    "order_id": {
      "type": "string",
      "description": "Business order identifier"
    },
    "customer_id": {
      "type": "string",
      "description": "Customer who placed the order"
    },
    "status": {
      "type": "string",
      "enum": ["PENDING", "CONFIRMED", "SHIPPED", "DELIVERED", "CANCELLED"],
      "description": "Current order status"
    },
    "total_cents": {
      "type": "integer",
      "minimum": 0,
      "description": "Order total in smallest currency unit"
    },
    "currency": {
      "type": "string",
      "default": "USD",
      "minLength": 3,
      "maxLength": 3,
      "description": "ISO 4217 currency code"
    },
    "shipping_address": {
      "$ref": "#/definitions/Address",
      "description": "Shipping address, null for digital orders"
    },
    "notes": {
      "type": "string",
      "description": "Optional order notes"
    }
  },
  "definitions": {
    "Address": {
      "type": "object",
      "required": ["street", "city", "postal_code", "country"],
      "properties": {
        "street": {"type": "string"},
        "city": {"type": "string"},
        "postal_code": {"type": "string"},
        "country": {"type": "string"}
      },
      "additionalProperties": false
    }
  },
  "additionalProperties": false
}
```

## Common Mistakes

1. **Missing `additionalProperties: false`** -- allows any extra fields, making the schema too permissive
2. **Confusing `required` with `type`** -- a field can be in `required` but still allow null unless typed otherwise
3. **Using `string` for everything** -- use `integer`, `number`, `boolean` for typed fields
4. **Not using `$ref` for shared types** -- leads to duplicate definitions that drift apart
5. **Overly complex `allOf`/`anyOf` compositions** -- keep schemas flat where possible for compatibility checking

## Starter Template

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "MyEvent",
  "description": "Description of the event",
  "type": "object",
  "required": ["event_id", "timestamp"],
  "properties": {
    "event_id": {"type": "string", "format": "uuid"},
    "timestamp": {"type": "string", "format": "date-time"},
    "my_field": {"type": "string", "description": "Description"}
  },
  "additionalProperties": false
}
```

Available tools: register_schema (with schema_type: JSON), validate_schema, check_compatibility
