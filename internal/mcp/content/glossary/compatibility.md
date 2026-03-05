# Compatibility

## Overview

Compatibility checking ensures that new schema versions can coexist with previous versions. The registry checks compatibility at registration time -- when a new schema version is registered via POST /subjects/{subject}/versions. If the proposed schema is incompatible, the registry rejects it with HTTP 409.

Compatibility is **not enforced retroactively**. Changing the compatibility level only affects future registrations.

## The 7 Compatibility Modes

| Mode | Direction | Transitive | Checked Against | Meaning |
|------|-----------|------------|-----------------|---------|
| **NONE** | -- | -- | Nothing | No compatibility checking. Any schema is accepted. |
| **BACKWARD** | Backward | No | Latest version | New schema can read data written by the previous schema. |
| **BACKWARD_TRANSITIVE** | Backward | Yes | All versions | New schema can read data written by ANY previous schema. |
| **FORWARD** | Forward | No | Latest version | Previous schema can read data written by the new schema. |
| **FORWARD_TRANSITIVE** | Forward | Yes | All versions | ANY previous schema can read data written by the new schema. |
| **FULL** | Both | No | Latest version | Both backward and forward compatible with latest. |
| **FULL_TRANSITIVE** | Both | Yes | All versions | Both backward and forward compatible with ALL previous. |

The default mode is **BACKWARD**.

## Understanding Backward vs Forward

**Backward compatibility** answers: "Can a NEW consumer read OLD data?"
- The new schema is the reader, the old schema is the writer.
- Safe changes: add fields with defaults, remove fields.

**Forward compatibility** answers: "Can an OLD consumer read NEW data?"
- The old schema is the reader, the new schema is the writer.
- Safe changes: remove optional fields, add fields without defaults.

**Full compatibility** requires both directions simultaneously.

## Transitive vs Non-Transitive

**Non-transitive** (BACKWARD, FORWARD, FULL): checks only against the **latest** previous version. Sufficient when consumers upgrade through each version in order.

**Transitive** (BACKWARD_TRANSITIVE, FORWARD_TRANSITIVE, FULL_TRANSITIVE): checks against **all** previous versions. Necessary when consumers may skip versions -- for example, reading data from version 1 with a consumer using version 5.

## Choosing a Compatibility Mode

| Scenario | Recommended Mode |
|----------|-----------------|
| Consumers always deploy before producers | BACKWARD |
| Producers always deploy before consumers | FORWARD |
| No control over deployment order | FULL |
| Long-lived data (cold storage, replay) | BACKWARD_TRANSITIVE or FULL_TRANSITIVE |
| Rapid prototyping, no compatibility needed | NONE |

## Configuration Resolution

The compatibility level is resolved in order of precedence:

1. **Per-subject override** -- set via set_config with a subject name
2. **Global override** -- set via set_config with no subject
3. **Config file default** -- the compatibility.default_level setting in YAML

DELETE /config/{subject} removes the per-subject override (falls back to global).
DELETE /config removes the global override (falls back to config file default).

## Avro Compatibility Rules

### Backward-Compatible Changes (safe under BACKWARD)
- Add a field **with a default value** -- old data uses the default.
- Remove a field -- old data's field is ignored.

### Forward-Compatible Changes (safe under FORWARD)
- Remove a field -- old reader uses its default.
- Add a field without a default -- old reader ignores it.

### Incompatible Changes
- Adding a field **without a default** is backward-incompatible.
- Removing a field **without a default** is forward-incompatible.
- Changing a field type to an incompatible type (e.g., string to int).
- Renaming a record without using aliases.

### Type Promotions (Widening)

| Writer Type | Allowed Reader Types |
|-------------|---------------------|
| int | long, float, double |
| long | float, double |
| float | double |
| string | bytes |
| bytes | string |

### Aliases
Avro aliases support controlled renaming of records and fields without breaking compatibility.

### Unions
Both sides of a union must be compatible. A non-union type must match at least one type in the opposing union.

### Enums
Every writer symbol must exist in the reader enum (or the reader defines a default). The reader may have additional symbols.

## JSON Schema Compatibility Rules

### Backward-Compatible Changes
- Add optional properties (not in "required").
- Remove required properties from "required".
- Widen types (integer to number).
- Remove constraints (enum, const, pattern, not, dependencies).
- Relax numeric bounds (decrease minimum, increase maximum).

### Incompatible Changes
- Add required properties.
- Change type (string to integer, except integer to number).
- Add constraints where none existed.
- Remove enum values.
- Tighten bounds.
- Close content model (additionalProperties: true to false).

### Checked Keywords
type, properties, required, additionalProperties, patternProperties, items, prefixItems, additionalItems, minItems, maxItems, uniqueItems, enum, const, minimum, maximum, exclusiveMinimum, exclusiveMaximum, multipleOf, minLength, maxLength, pattern, minProperties, maxProperties, oneOf, anyOf, allOf, not, dependencies (Draft-07), dependentRequired, dependentSchemas (Draft 2020-12), $ref.

## Protobuf Compatibility Rules

Protobuf compatibility works at the **wire-format level**. Field numbers and wire types matter, not field names.

### Backward-Compatible Changes
- Add new fields (with new field numbers).
- Remove optional/repeated fields.
- Add new enum values.
- Add new messages.
- Change field names (not encoded on wire).

### Incompatible Changes
- Change a field number.
- Change a field type to a wire-incompatible type.
- Remove/add a proto2 required field.
- Remove a field from a oneof.
- Move a field into an existing oneof.
- Change the package name.

### Wire-Compatible Type Groups

| Wire Type | Compatible Types |
|-----------|-----------------|
| Varint (0) | int32, uint32, int64, uint64, bool |
| Zigzag varint (0) | sint32, sint64 |
| 32-bit fixed (5) | fixed32, sfixed32 |
| 64-bit fixed (1) | fixed64, sfixed64 |
| Length-delimited (2) | string, bytes |

Enum types are wire-compatible with all varint types.

## Compatibility Groups

Compatibility groups allow multiple independent schema lineages within the same subject. Set a compatibilityGroup property name on the subject config, then include that property in metadata when registering. Only schemas with the same group value are checked against each other.

## Checking Compatibility via API

Use **check_compatibility** to test a schema before registering:
- Check against latest: POST /compatibility/subjects/{subject}/versions/latest
- Check against all: POST /compatibility/subjects/{subject}/versions
- Add ?verbose=true for detailed incompatibility messages.

## MCP Tools

- **get_config / set_config** -- read or change compatibility level
- **check_compatibility** -- test a schema before registering
- **explain_compatibility_failure** -- get human-readable explanations for compat errors
