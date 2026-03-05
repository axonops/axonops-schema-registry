# Best Practices

## Choosing a Schema Format

| Criterion | Avro | Protobuf | JSON Schema |
|-----------|------|----------|-------------|
| Kafka ecosystem fit | Best | Good | Good |
| Schema evolution | Excellent | Good | Limited |
| Binary size | Compact | Compact | Verbose (JSON text) |
| Code generation | Moderate | Excellent | Minimal |
| gRPC support | Limited | Native | None |
| Human readability | Schema: JSON, Data: binary | Schema: .proto, Data: binary | Both JSON |
| Validation richness | Schema-level | Schema-level | Rich constraints |

**Default recommendation:** Use **Avro** for Kafka event streaming unless you have a specific reason for another format (gRPC = Protobuf, REST API validation = JSON Schema).

## Avro Best Practices

1. **Always use a namespace.** Use reverse-domain notation: com.company.domain.
2. **Use PascalCase for record/enum names, snake_case for fields.**
3. **Always provide default values** for new fields. This ensures backward compatibility.
4. **Use union ["null", "type"] with default null** for optional fields.
5. **Use logical types** for dates (timestamp-millis), decimals (bytes+decimal), UUIDs (string+uuid).
6. **Use enums** for fixed value sets, not plain strings.
7. **Never remove fields without defaults** under BACKWARD compatibility.
8. **Use aliases** when you need to rename fields or records.
9. **Avoid deeply nested unions** -- they complicate evolution and code generation.
10. **Document fields** using the "doc" attribute.

## Protobuf Best Practices

1. **Use proto3 syntax** unless you have a specific need for proto2.
2. **Use a package declaration** matching your domain: package company.events.v1.
3. **Use PascalCase for messages/enums, snake_case for fields.**
4. **Never reuse deleted field numbers.** Use "reserved" to prevent accidental reuse.
5. **Use UNSPECIFIED = 0** as the first enum value.
6. **Prefer explicit field presence** (optional keyword in proto3) for fields that need null semantics.
7. **Use well-known types** (google.protobuf.Timestamp, Duration) instead of custom types.
8. **Use field numbers 1-15** for frequently accessed fields (smaller encoding).
9. **Never change field numbers** -- this breaks wire compatibility.
10. **Use oneof for variant types** instead of multiple boolean flags.

## JSON Schema Best Practices

1. **Use "type": "object" as root type.**
2. **Define a "required" array** listing mandatory fields.
3. **Use "additionalProperties": false** to prevent unexpected fields.
4. **Use format validators** (email, uri, date-time, uuid) for semantic types.
5. **Use pattern for custom string validation.**
6. **Use minimum/maximum** for number ranges, minLength/maxLength for strings.
7. **Use enum** for fixed value sets.
8. **Use $ref** for reusable type definitions.
9. **Avoid changing additionalProperties from true to false** -- this is a backward-incompatible change.
10. **Be careful with oneOf/anyOf** -- adding or removing options affects compatibility.

## Universal Best Practices

### Field Naming
- Use **snake_case** consistently across all formats.
- Use descriptive names: order_total_amount, not ota.
- Prefix boolean fields with is_, has_, or can_: is_active, has_shipping_address.

### Schema Evolution
- **Start with BACKWARD compatibility** (the default). Upgrade to FULL when you need it.
- **Always test with check_compatibility** before registering.
- **Never make breaking changes** without a migration plan.
- **Use the three-phase pattern** for renames and type changes.
- **Version your shared types** with FULL_TRANSITIVE compatibility.

### Subject Naming
- Pick ONE naming strategy per environment and use it consistently.
- Use lowercase with hyphens for topic names: user-events, not UserEvents.
- Use reverse-domain namespace for record names: com.company.domain.Type.

### Compatibility Strategy
- **Dev/staging:** BACKWARD or NONE (for rapid iteration).
- **Production:** BACKWARD or FULL (for safety).
- **Shared types:** FULL_TRANSITIVE (both producers and consumers handle changes).
- Use **per-subject overrides** for subjects that need different policies.

## Common Mistakes

1. **Not providing defaults on new fields** -- breaks backward compatibility.
2. **Reusing deleted field numbers** in Protobuf -- corrupts existing data.
3. **Changing additionalProperties to false** in JSON Schema -- breaks existing producers.
4. **Using string for everything** -- loses type safety and validation.
5. **Not using schema references** for shared types -- leads to schema duplication and drift.
6. **Setting NONE compatibility globally** -- disables all safety checks.
7. **Mixing naming strategies** in the same registry -- creates confusion.
8. **Not testing in CI** -- schema compatibility issues discovered in production.

## MCP Tools

- **score_schema_quality** -- analyze naming, docs, type safety, and evolution readiness
- **check_compatibility** -- validate changes before registering
- **detect_schema_patterns** -- check naming convention coverage
- **validate_schema** -- syntax check without registering
