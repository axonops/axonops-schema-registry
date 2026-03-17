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

## Anti-Patterns

### 1. "String Everything"
Using `string` for all fields (timestamps, amounts, booleans, enums).
**Consequence:** No type safety, no validation, no efficient serialization. Consumers must parse and validate every field manually.
**Fix:** Use typed fields (`long` + `timestamp-millis`, `int`, `boolean`, `enum`).

### 2. NONE Compatibility Globally
Setting `NONE` as the global default "because it's easier."
**Consequence:** Any schema change registers successfully, even breaking ones. Consumers fail at runtime with deserialization errors in production.
**Fix:** Use BACKWARD (default) or FULL globally. Use per-subject NONE only during initial development, then switch before production.

### 3. Changing Schema IDs
Manually reassigning schema IDs or using IMPORT mode for non-migration purposes.
**Consequence:** Kafka messages embed schema IDs in their wire format. Changing IDs makes existing messages unreadable. Data corruption.
**Fix:** Never change IDs in production. Use IMPORT mode only for Confluent migration.

### 4. Nested Union Hell (Avro)
Using unions inside unions: `["null", ["null", "string"]]`.
**Consequence:** Not valid in Avro. Even nested records with union fields can create complex deserialization code that is hard to evolve.
**Fix:** Flatten unions. Use separate records for complex variant types.

### 5. Missing `reserved` in Protobuf
Deleting a field without adding its number to `reserved`.
**Consequence:** Future developers reuse the field number for a different field. Old messages deserialize the wrong type into the new field. Silent data corruption.
**Fix:** Always add `reserved N;` when removing fields.

### 6. Producing Without Registering
Serializing data without first registering the schema.
**Consequence:** The schema registry rejects the schema at serialization time, causing producer failures. Or worse, a consumer gets data for an unknown schema ID.
**Fix:** Register schemas in CI/CD before deploying producers. Use **validate_schema** and **check_compatibility** as pre-commit gates.

### 7. Soft-Delete Confusion
Not understanding the two-stage delete model.
**Consequence:** Attempting permanent delete without soft-delete first (error 40405). Or soft-deleting and expecting the subject to be fully gone (it is hidden but data still exists).
**Fix:** Stage 1: `delete_subject` (soft-delete, reversible). Stage 2: `delete_subject` with `permanent: true` (irreversible). Check with `list_subjects(deleted: true)`.

## Context Best Practices

### Environment Isolation
Use contexts to separate environments: `.dev`, `.staging`, `.production`. Each gets independent schemas, IDs, and compatibility settings.

### Team Isolation
Use contexts for team namespaces: `.team-orders`, `.team-payments`. Teams can iterate independently without affecting each other.

### Context-Level Config Defaults
Set compatibility at the context level (not just global) so different teams can have different defaults. Use the 4-tier inheritance chain: per-subject > context global > __GLOBAL > server default.

### Consistent Context Parameter
Always pass the `context` parameter consistently across all MCP tool calls within a workflow. Mixing default context with explicit context in the same session leads to confusion about which schemas are visible.

### Avoid Mixing Qualified Subjects with Context Parameter
Do NOT use qualified subjects (`:.team:my-subject`) AND the `context` parameter simultaneously. Pick one approach:
- **Context parameter:** `register_schema(subject: "my-subject", context: ".team")`
- **Qualified subject:** `register_schema(subject: ":.team:my-subject")`

### Context Naming Conventions
- Use lowercase with hyphens: `.team-orders`, `.staging`
- Prefix with purpose: `.env-staging`, `.team-payments`
- Keep names short but descriptive

## MCP Tools

- **score_schema_quality** -- analyze naming, docs, type safety, and evolution readiness
- **check_compatibility** -- validate changes before registering
- **detect_schema_patterns** -- check naming convention coverage
- **validate_schema** -- syntax check without registering
