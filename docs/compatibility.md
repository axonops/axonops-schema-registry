# Compatibility

## Contents

- [Overview](#overview)
- [Compatibility Modes](#compatibility-modes)
  - [Understanding Backward vs Forward](#understanding-backward-vs-forward)
  - [Transitive vs Non-Transitive](#transitive-vs-non-transitive)
- [Configuration Resolution](#configuration-resolution)
  - [Setting Compatibility](#setting-compatibility)
- [Avro Compatibility Rules](#avro-compatibility-rules)
  - [Backward-Compatible Changes (safe to make under BACKWARD mode)](#backward-compatible-changes-safe-to-make-under-backward-mode)
  - [Forward-Compatible Changes (safe to make under FORWARD mode)](#forward-compatible-changes-safe-to-make-under-forward-mode)
  - [Incompatible Changes](#incompatible-changes)
  - [Type Promotions](#type-promotions)
  - [Aliases](#aliases)
  - [Unions](#unions)
  - [Enums](#enums)
- [JSON Schema Compatibility Rules](#json-schema-compatibility-rules)
  - [Backward-Compatible Changes](#backward-compatible-changes)
  - [Incompatible Changes](#incompatible-changes-1)
  - [Checked Constraints](#checked-constraints)
- [Protobuf Compatibility Rules](#protobuf-compatibility-rules)
  - [Backward-Compatible Changes](#backward-compatible-changes-1)
  - [Incompatible Changes](#incompatible-changes-2)
  - [Wire-Compatible Type Groups](#wire-compatible-type-groups)
  - [Cardinality Changes](#cardinality-changes)
  - [Syntax Changes](#syntax-changes)
  - [Service Definitions](#service-definitions)
- [Checking Compatibility via API](#checking-compatibility-via-api)
  - [Check Against a Specific Version](#check-against-a-specific-version)
  - [Check Against All Versions](#check-against-all-versions)
  - [Request Body](#request-body)
  - [Response](#response)
  - [Verbose Mode](#verbose-mode)
  - [Example: Check Before Registering](#example-check-before-registering)
- [Compatibility Groups](#compatibility-groups)
  - [How It Works](#how-it-works)
  - [Configuration](#configuration)
  - [Registering Schemas with Groups](#registering-schemas-with-groups)
- [Related Documentation](#related-documentation)

## Overview

Compatibility checking ensures that new schema versions can coexist with previous versions. The registry checks compatibility at registration time -- when a new schema version is registered via `POST /subjects/{subject}/versions`. If the proposed schema is incompatible with existing versions (under the active compatibility mode), the registry rejects the registration with HTTP 409 and an error message describing the incompatibility.

Compatibility is not enforced retroactively. Changing the compatibility level for a subject does not re-validate previously registered schemas. It only affects future registrations.

## Compatibility Modes

| Mode | Backward | Forward | Transitive | Description |
|------|----------|---------|------------|-------------|
| NONE | No | No | No | No compatibility checking |
| BACKWARD | Yes | No | No | New schema can read data written by the latest previous schema |
| BACKWARD_TRANSITIVE | Yes | No | Yes | New schema can read data written by ALL previous schemas |
| FORWARD | No | Yes | No | Latest previous schema can read data written by new schema |
| FORWARD_TRANSITIVE | No | Yes | Yes | ALL previous schemas can read data written by new schema |
| FULL | Yes | Yes | No | Both backward and forward compatible with latest |
| FULL_TRANSITIVE | Yes | Yes | Yes | Both backward and forward compatible with ALL previous |

The default mode is **BACKWARD**.

### Understanding Backward vs Forward

**Backward compatibility** answers the question: "Can a new consumer read old data?"

A new consumer using the new schema can deserialize messages that were written with the previous schema. In reader/writer terminology, the new schema is the reader and the old schema is the writer.

**Forward compatibility** answers the question: "Can an old consumer read new data?"

An existing consumer using the old schema can deserialize messages that were written with the new schema. The old schema is the reader and the new schema is the writer.

**Full compatibility** requires both backward and forward compatibility simultaneously.

### Transitive vs Non-Transitive

Non-transitive modes (BACKWARD, FORWARD, FULL) check compatibility only against the **latest** previous version. This is sufficient when schemas evolve incrementally and every consumer upgrades through each version in order.

Transitive modes (BACKWARD_TRANSITIVE, FORWARD_TRANSITIVE, FULL_TRANSITIVE) check compatibility against **all** previous versions. This is necessary when consumers may skip versions -- for example, when a consumer running schema version 1 needs to read data written with schema version 5 without having processed versions 2 through 4.

## Configuration Resolution

The compatibility level for a subject is resolved in this order of precedence:

1. **Per-subject override** -- set via `PUT /config/{subject}`
2. **Global override** -- set via `PUT /config`
3. **Config file default** -- the `compatibility.default_level` setting in the YAML config

This means:

- The config file default is the baseline that applies when no runtime overrides exist.
- An administrator can override the default globally at runtime via the API. This override is persisted in the storage backend.
- An administrator can override the compatibility level for individual subjects. This takes highest priority.
- `DELETE /config/{subject}` removes the per-subject override, causing the subject to fall back to the global setting.
- `DELETE /config` removes the global runtime override, causing it to fall back to the config file default.
- Changes only affect future registrations. Existing schemas are not re-validated.

### Setting Compatibility

Set the global compatibility level:

```bash
curl -X PUT http://localhost:8081/config \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "FULL"}'
```

Set the compatibility level for a specific subject:

```bash
curl -X PUT http://localhost:8081/config/my-subject \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "NONE"}'
```

Read the effective compatibility level for a subject (reflects the full resolution chain):

```bash
curl http://localhost:8081/config/my-subject
```

Response:

```json
{"compatibilityLevel": "NONE"}
```

Remove the per-subject override:

```bash
curl -X DELETE http://localhost:8081/config/my-subject
```

Remove the global runtime override:

```bash
curl -X DELETE http://localhost:8081/config
```

## Avro Compatibility Rules

The Avro compatibility checker follows the [Avro specification](https://avro.apache.org/docs/current/specification/) rules for schema resolution. Compatibility is checked by attempting to read data written with one schema using the other schema.

### Backward-Compatible Changes (safe to make under BACKWARD mode)

- **Add a field with a default value.** The new reader schema has a field that the old writer schema does not. When reading old data, the default value is used.
- **Remove a field.** The old writer schema has a field that the new reader schema does not. The old field is ignored when reading.

### Forward-Compatible Changes (safe to make under FORWARD mode)

- **Remove a field.** The old reader still has the field; the new writer no longer writes it. The old reader uses its default value (if present).
- **Add a field without a default.** The old reader ignores the new field.

### Incompatible Changes

- **Adding a field without a default** is backward-incompatible. The new reader cannot construct a value for this field when reading old data.
- **Removing a field without a default** is forward-incompatible. The old reader cannot construct a value for the removed field when reading new data.
- **Changing a field type** to an incompatible type (for example, `string` to `int`).
- **Renaming a record** without using aliases.

### Type Promotions

Avro supports the following numeric promotions (widening):

| Writer Type | Allowed Reader Types |
|-------------|---------------------|
| `int` | `long`, `float`, `double` |
| `long` | `float`, `double` |
| `float` | `double` |
| `string` | `bytes` |
| `bytes` | `string` |

### Aliases

Avro aliases are supported for both record and field renaming. If a reader schema renames a record or field, the checker resolves the old name through the aliases defined on the reader or writer schema. This allows controlled renaming without breaking compatibility.

### Unions

When one side of a compatibility check is a union and the other is not, the checker verifies that the non-union type is compatible with at least one type in the union. When both sides are unions, every writer union type must be compatible with at least one reader union type.

### Enums

When comparing enums, every writer symbol must exist in the reader enum (or the reader must define a default enum value). The reader may have additional symbols.

## JSON Schema Compatibility Rules

The JSON Schema compatibility checker supports both **Draft-07** and **Draft 2020-12** schemas. Compatibility is determined by analyzing whether the new schema accepts all values that the old schema accepts (backward) or vice versa (forward).

### Backward-Compatible Changes

- **Add optional properties.** New properties that are not in the `required` array.
- **Remove required properties.** Properties removed from `required`.
- **Widen types.** For example, changing `integer` to `number`.
- **Remove constraints.** Removing `enum`, `const`, `pattern`, `not`, or `dependencies`.
- **Relax numeric bounds.** Decreasing `minimum`/`minLength`/`minItems`/`minProperties` or increasing `maximum`/`maxLength`/`maxItems`/`maxProperties`.

### Incompatible Changes

- **Add required properties.** A new property in `required` that did not exist before.
- **Change property from optional to required.**
- **Change type.** For example, changing `string` to `integer` (except `integer` to `number`, which is a valid widening).
- **Add constraints.** Adding `enum`, `const`, `pattern`, `not`, `uniqueItems`, `dependencies`, `dependentRequired`, or `dependentSchemas` where none existed.
- **Remove enum values.** Removing a value from an `enum` array.
- **Tighten numeric bounds.** Increasing `minimum`/`minLength`/`minItems`/`minProperties` or decreasing `maximum`/`maxLength`/`maxItems`/`maxProperties`.
- **Close the content model.** Changing `additionalProperties` from `true` (or absent) to `false`.
- **Add properties to an open content model.** If the old schema allowed `additionalProperties` (the default), adding a new typed property is incompatible because old data may already contain values of a different type for that property name.

### Checked Constraints

The JSON Schema checker evaluates the following keywords:

- `type` (with `integer` to `number` promotion)
- `properties`, `required`, `additionalProperties`, `patternProperties`
- `items`, `prefixItems`, `additionalItems`, `minItems`, `maxItems`, `uniqueItems`
- `enum`, `const`
- `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`, `multipleOf`
- `minLength`, `maxLength`, `pattern`
- `minProperties`, `maxProperties`
- `oneOf`, `anyOf`, `allOf`
- `not`
- `dependencies` (Draft-07)
- `dependentRequired`, `dependentSchemas` (Draft 2020-12)
- `$ref` resolution (local references via `#/definitions/` and `#/$defs/`)

## Protobuf Compatibility Rules

The Protobuf compatibility checker works at the wire-format level. Protobuf's binary encoding uses field numbers and wire types rather than field names, so many changes that look significant at the source level are actually wire-compatible.

### Backward-Compatible Changes

- **Add new fields** (with new field numbers). Old data simply lacks the new fields; readers use default values.
- **Remove optional/repeated fields.** New readers ignore unknown field numbers in old data.
- **Add new enum values.** Unknown enum values are preserved as their numeric value.
- **Add new messages.** No wire impact on existing messages.
- **Change field names.** Field names are not encoded on the wire; only numbers matter.
- **Remove enum type definitions.** Enum fields are integers on the wire.

### Incompatible Changes

- **Change a field number.** The reader cannot match the old data to the correct field.
- **Change a field type to a wire-incompatible type.** For example, `int32` to `string` uses different wire encoding.
- **Remove a required field** (proto2 `required` keyword).
- **Add a required field** (proto2 `required` keyword).
- **Remove a field from a oneof.** Changes the semantics of the oneof group.
- **Move a field into an existing oneof** that already has other members. This adds a mutual exclusion constraint that did not exist before.
- **Remove a message** that is referenced by other messages.
- **Change the package name.**

### Wire-Compatible Type Groups

Fields within the same wire-type group can be changed between each other without breaking compatibility:

| Wire Type | Compatible Types |
|-----------|-----------------|
| Varint (wire type 0) | `int32`, `uint32`, `int64`, `uint64`, `bool` |
| Zigzag varint (wire type 0) | `sint32`, `sint64` |
| 32-bit fixed (wire type 5) | `fixed32`, `sfixed32` |
| 64-bit fixed (wire type 1) | `fixed64`, `sfixed64` |
| Length-delimited (wire type 2) | `string`, `bytes` |

Additionally, `enum` types are wire-compatible with all varint types (`int32`, `uint32`, `int64`, `uint64`, `bool`).

### Cardinality Changes

- `optional` to `repeated` is wire-compatible for `string`, `bytes`, and `message` fields (all use length-delimited encoding). For other types, this is incompatible.
- `required` to `optional` or `repeated` is compatible.
- `optional` to `required` is incompatible.

### Syntax Changes

Changing between `proto2` and `proto3` syntax is not treated as incompatible. The syntax keyword is a source-level annotation; `proto2 optional` and `proto3` fields produce identical wire bytes.

### Service Definitions

Service definitions are gRPC metadata with no wire-format impact on message serialization. The checker does not flag service changes as incompatible.

## Checking Compatibility via API

You can check whether a proposed schema is compatible with existing versions before registering it. This is useful for CI/CD pipelines or pre-registration validation.

### Check Against a Specific Version

```
POST /compatibility/subjects/{subject}/versions/{version}
```

The `{version}` parameter accepts an integer version number or the string `latest`.

### Check Against All Versions

```
POST /compatibility/subjects/{subject}/versions
```

When no version is specified, the proposed schema is checked against all existing versions of the subject.

### Request Body

```json
{
  "schema": "{...}",
  "schemaType": "AVRO",
  "references": []
}
```

The `schemaType` field defaults to `AVRO` if omitted. Valid values are `AVRO`, `JSON`, and `PROTOBUF`. The `references` array is optional and is used for schemas that reference other schemas by subject.

### Response

A successful compatibility check returns:

```json
{"is_compatible": true}
```

An incompatible schema returns:

```json
{"is_compatible": false}
```

### Verbose Mode

Add `?verbose=true` to get detailed incompatibility messages:

```
POST /compatibility/subjects/my-subject/versions/latest?verbose=true
```

Response:

```json
{
  "is_compatible": false,
  "messages": [
    "BACKWARD compatibility check failed against version 1: root: reader field 'email' has no default and is missing from writer"
  ]
}
```

### Example: Check Before Registering

```bash
curl -X POST http://localhost:8081/compatibility/subjects/users-value/versions/latest \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\",\"default\":0}]}"
  }'
```

```json
{"is_compatible": true}
```

## Compatibility Groups

Compatibility groups allow multiple independent schema lineages within the same subject. This is useful when a subject contains schemas that represent different major versions or different logical schema families that should not be checked against each other.

### How It Works

The `compatibilityGroup` configuration property names a **metadata property key**. When set, the registry only checks compatibility against existing schemas that have the **same value** for that metadata property. Schemas with a different value (or no value) for the property are excluded from the compatibility check.

### Configuration

Set a compatibility group for a subject:

```bash
curl -X PUT http://localhost:8081/config/my-subject \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "BACKWARD", "compatibilityGroup": "major_version"}'
```

### Registering Schemas with Groups

When registering a schema, include the group value in the `metadata.properties` field:

```bash
# Register v1 schema in group "1"
curl -X POST http://localhost:8081/subjects/my-subject/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
    "metadata": {"properties": {"major_version": "1"}}
  }'

# Register a completely different schema in group "2" — compatibility is not
# checked against group "1" schemas, so this succeeds even though the schemas
# are incompatible
curl -X POST http://localhost:8081/subjects/my-subject/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"event_id\",\"type\":\"int\"}]}",
    "metadata": {"properties": {"major_version": "2"}}
  }'
```

Without a `compatibilityGroup` configured, all schemas under the subject are compared against each other regardless of metadata values.

## Related Documentation

- [API Reference](api-reference.md) -- complete endpoint documentation including error codes
- [Schema Types](schema-types.md) -- Avro, JSON Schema, and Protobuf parser details
- [Configuration](configuration.md) -- full YAML configuration reference
- [Getting Started](getting-started.md) -- register your first schemas and test compatibility
