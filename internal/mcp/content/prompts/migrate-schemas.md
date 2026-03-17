Migrate schemas from {source} to {target} format.

## Workflow

1. Use **list_subjects** to find schemas to migrate
2. Use **get_latest_schema** for each subject to inspect the current schema
3. Convert the schema to {target} format using the mapping tables below
4. Use **validate_schema** with `schema_type: {target}` to check syntax
5. Use **register_schema** with `schema_type: {target}` on a NEW subject (e.g., `orders-value-proto`)
6. Use **check_compatibility** if the new subject already existed
7. Update serializer/deserializer configs with the new schema IDs

> **Important:** Schema IDs change when you register under a new subject. Update all producer/consumer configurations.
> All tools accept the optional `context` parameter for multi-tenant isolation.
> Use the **compare-formats** prompt if you need help choosing a target format.

---

## Type Mapping: Avro to Protobuf

| Avro | Protobuf |
|------|----------|
| `int` | `int32` |
| `long` | `int64` |
| `float` | `float` |
| `double` | `double` |
| `string` | `string` |
| `bytes` | `bytes` |
| `boolean` | `bool` |
| `null` | N/A (all proto3 fields have zero defaults) |
| `fixed` | `bytes` (fixed size not preserved) |
| record | message |
| enum | enum (prepend `UNSPECIFIED = 0` as first value) |
| array | `repeated` field |
| map | `map<string, V>` |
| union `["null", "type"]` | `optional type` field |
| union with multiple non-null types | `oneof` |
| namespace | `package` |
| doc | `//` comment |

**Logical type mapping:**

| Avro Logical Type | Protobuf |
|-------------------|----------|
| `timestamp-millis` / `timestamp-micros` | `google.protobuf.Timestamp` |
| `decimal` | `string` or custom message (precision lost) |
| `uuid` | `string` |
| `date` | `int32` (days since epoch) |
| `time-millis` | `int32` |
| `duration` | `google.protobuf.Duration` |

**What is lost:**
- Default values have no direct proto3 equivalent (proto3 uses zero values only)
- Avro `doc` fields become `//` comments (no structured doc in proto3)
- Avro `aliases` are dropped entirely (no proto3 equivalent)
- Avro named type references become Protobuf `import` paths
- Proto3 requires explicit field numbers — assign sequentially starting from 1

---

## Type Mapping: Protobuf to Avro

| Protobuf | Avro |
|----------|------|
| `int32` / `sint32` / `sfixed32` | `int` |
| `int64` / `sint64` / `sfixed64` | `long` |
| `uint32` / `fixed32` | `int` (unsigned-to-signed — warn if values may exceed 2^31) |
| `uint64` / `fixed64` | `long` (unsigned-to-signed — warn if values may exceed 2^63) |
| `float` | `float` |
| `double` | `double` |
| `string` | `string` |
| `bytes` | `bytes` |
| `bool` | `boolean` |
| message | record |
| enum | enum (drop `UNSPECIFIED` value, set default to first remaining) |
| `repeated` | array |
| `map<string, V>` | map |
| `map<K, V>` (non-string key) | **not directly supported** — Avro maps are always string-keyed (flatten or restructure) |
| `optional` | union `["null", "type"]` with default `null` |
| `oneof` | union |
| nested message | nested record |
| `package` | namespace |
| `google.protobuf.Timestamp` | `long` with logical type `timestamp-millis` |
| `google.protobuf.Duration` | `long` (milliseconds) or `fixed` (custom) |

**What is lost:**
- Field numbers are dropped (Avro uses field names on the wire)
- `reserved` fields and numbers have no Avro equivalent
- Proto3 `map<int, V>` or `map<bool, V>` requires restructuring — Avro maps only support string keys
- Service definitions and RPC methods have no Avro equivalent
- `uint32`/`uint64` are silently narrowed to signed types

---

## Type Mapping: Avro to JSON Schema

| Avro | JSON Schema |
|------|-------------|
| record | `{"type": "object", "properties": {...}}` |
| enum | `{"type": "string", "enum": [...]}` |
| array | `{"type": "array", "items": {...}}` |
| map | `{"type": "object", "additionalProperties": {...}}` |
| union `["null", "type"]` | property NOT in `required` array |
| `int` / `long` | `{"type": "integer"}` |
| `float` / `double` | `{"type": "number"}` |
| `string` | `{"type": "string"}` |
| `boolean` | `{"type": "boolean"}` |
| `bytes` | `{"type": "string", "contentEncoding": "base64"}` |
| `fixed` | `{"type": "string", "contentEncoding": "base64"}` |
| `null` | `{"type": "null"}` |

**Logical type mapping:**

| Avro | JSON Schema |
|------|-------------|
| `timestamp-millis` | `{"type": "string", "format": "date-time"}` |
| `uuid` | `{"type": "string", "format": "uuid"}` |
| `date` | `{"type": "string", "format": "date"}` |
| `time-millis` | `{"type": "string", "format": "time"}` |
| `decimal` | `{"type": "string"}` or `{"type": "number"}` |

**What is lost:**
- Avro namespace has no JSON Schema equivalent (use `$id` for naming)
- Avro `aliases` are dropped
- `default` maps to `default` keyword (preserved)
- `doc` maps to `description` keyword (preserved)

---

## Type Mapping: JSON Schema to Avro

| JSON Schema | Avro |
|-------------|------|
| `required` properties | fields without union null |
| Optional properties (not in `required`) | union `["null", "type"]` with default `null` |
| `additionalProperties` | map |
| `oneOf` / `anyOf` | union |
| `enum` | enum |
| `$ref` | named type reference |
| `{"type": "object"}` | record |
| `{"type": "array"}` | array |
| `{"type": "integer"}` | `long` (safest default) |
| `{"type": "number"}` | `double` (safest default) |
| `{"type": "string"}` | `string` |
| `{"type": "boolean"}` | `boolean` |
| `{"type": "string", "format": "date-time"}` | `long` with logical type `timestamp-millis` |
| `{"type": "string", "format": "uuid"}` | `string` with logical type `uuid` |
| `{"type": "string", "format": "date"}` | `int` with logical type `date` |

**What is lost:**
- Validation constraints: `pattern`, `minLength`, `maxLength`, `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`, `multipleOf`
- Conditional logic: `if`/`then`/`else` has no Avro equivalent
- `allOf` composition must be flattened manually into a single record
- `not` has no Avro equivalent
- `$ref` maps to Avro named type references — register referenced schemas first

---

## Type Mapping: Protobuf to JSON Schema

| Protobuf | JSON Schema |
|----------|-------------|
| message | `{"type": "object", "properties": {...}}` |
| enum | `{"type": "string", "enum": [...]}` (use string value names) |
| `int32` / `sint32` / `sfixed32` / `uint32` / `fixed32` | `{"type": "integer"}` |
| `int64` / `sint64` / `sfixed64` / `uint64` / `fixed64` | `{"type": "integer"}` |
| `float` / `double` | `{"type": "number"}` |
| `string` | `{"type": "string"}` |
| `bytes` | `{"type": "string", "contentEncoding": "base64"}` |
| `bool` | `{"type": "boolean"}` |
| `repeated` | `{"type": "array", "items": {...}}` |
| `map<K, V>` | `{"type": "object", "additionalProperties": {...}}` |
| `oneof` | `{"oneOf": [...]}` |
| `optional` | property NOT in `required` array |
| nested message | nested `{"type": "object"}` |
| `google.protobuf.Timestamp` | `{"type": "string", "format": "date-time"}` |

**What is lost:**
- Field numbers are dropped
- `reserved` fields and numbers have no JSON Schema equivalent
- Service definitions and RPC methods have no JSON Schema equivalent
- Proto `//` comments are not preserved (no standard comment field in JSON Schema)

---

## Type Mapping: JSON Schema to Protobuf

| JSON Schema | Protobuf |
|-------------|----------|
| `{"type": "object", "properties": {...}}` | message (assign field numbers sequentially from 1) |
| `{"type": "string", "enum": [...]}` | enum (prepend `UNSPECIFIED = 0` as first value) |
| `{"type": "array", "items": {...}}` | `repeated` field |
| `{"type": "object", "additionalProperties": {...}}` | `map<string, V>` |
| `{"oneOf": [...]}` | `oneof` |
| `{"type": "integer"}` | `int64` (safest default) |
| `{"type": "number"}` | `double` (safest default) |
| `{"type": "string"}` | `string` |
| `{"type": "boolean"}` | `bool` |
| `{"type": "string", "contentEncoding": "base64"}` | `bytes` |
| `{"type": "string", "format": "date-time"}` | `google.protobuf.Timestamp` |
| `$ref` | `import` + message reference |

**What is lost:**
- Validation constraints: `pattern`, `minLength`, `maxLength`, `minimum`, `maximum`, `multipleOf`
- Conditional logic: `if`/`then`/`else`, `not`
- `allOf` must be flattened manually
- `required` has no proto3 equivalent (all fields are optional by default)
- `default` values have no proto3 equivalent (proto3 uses zero values)
- `description` maps to `//` comments

---

## Known Lossy Conversions

Every format migration loses some information. The following conversions are inherently lossy and the user MUST be warned:

| Direction | What Is Lost |
|-----------|-------------|
| Any → Protobuf | Default values, doc strings (become comments), aliases, validation constraints |
| Protobuf → Any | Field numbers, reserved declarations, service/RPC definitions |
| JSON Schema → Avro | Validation constraints (`pattern`, `min`/`max`, `multipleOf`), conditional logic (`if`/`then`/`else`), `not`, complex `allOf` composition |
| JSON Schema → Protobuf | Validation constraints, conditional logic, `required` semantics, default values |
| Avro → Any | Aliases are dropped by all target formats |
| Protobuf `map<non-string, V>` → Avro | Avro maps only support string keys — restructure as array of records |
| Protobuf `uint32`/`uint64` → Avro | Unsigned integers narrowed to signed `int`/`long` — possible overflow |
| Avro `decimal` → Protobuf | Arbitrary-precision decimal becomes `string` (precision semantics lost) |

---

## General Guidance

- Create NEW subjects for the migrated format (e.g., `orders-value` becomes `orders-value-proto`)
- Do NOT change the schema format in an existing subject — this is a breaking change
- Schema IDs will change — update all serializer/deserializer configurations
- Test the converted schema with **validate_schema** before registering
- Use **diff_schemas** to compare the original and converted schemas side by side
- For complex schemas with references, migrate referenced schemas first
- Run producers and consumers in parallel during migration to ensure zero downtime
