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
| record | message |
| enum | enum (add `UNSPECIFIED = 0` as first value) |
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

**Key differences:**
- Default values have no direct proto3 equivalent (proto3 uses zero values only)
- Avro named type references become Protobuf `import` paths
- Avro `doc` fields become `//` comments (no structured doc in proto3)
- Proto3 requires explicit field numbers (assign sequentially)

---

## Type Mapping: Protobuf to Avro

| Protobuf | Avro |
|----------|------|
| `int32` / `sint32` / `fixed32` | `int` |
| `int64` / `sint64` / `fixed64` | `long` |
| `float` | `float` |
| `double` | `double` |
| `string` | `string` |
| `bytes` | `bytes` |
| `bool` | `boolean` |
| message | record |
| enum | enum (drop `UNSPECIFIED` or map to default) |
| `repeated` | array |
| `map<K, V>` | map (Avro maps always have string keys) |
| `optional` | union `["null", "type"]` with default `null` |
| `oneof` | union |
| nested message | nested record |
| `package` | namespace |

**Key differences:**
- Field numbers are lost (Avro uses field names on the wire)
- `reserved` fields/numbers have no Avro equivalent
- Well-known types map to logical types where possible (Timestamp to timestamp-millis)
- Proto `map<int, V>` requires conversion (Avro maps are string-keyed only)

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
| `null` | `{"type": "null"}` |

**Logical type mapping:**
| Avro | JSON Schema |
|------|-------------|
| `timestamp-millis` | `{"type": "string", "format": "date-time"}` |
| `uuid` | `{"type": "string", "format": "uuid"}` |
| `date` | `{"type": "string", "format": "date"}` |
| `decimal` | `{"type": "string"}` or `{"type": "number"}` |

**Key differences:**
- `default` maps to `default` keyword
- `doc` maps to `description` keyword
- Avro namespace has no JSON Schema equivalent (use `$id`)

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
| `{"type": "integer"}` | `long` (safest) |
| `{"type": "number"}` | `double` (safest) |
| `{"type": "string"}` | `string` |
| `{"type": "boolean"}` | `boolean` |

**Key differences:**
- Format constraints lost (`pattern`, `minLength`, `maxLength`, `minimum`, `maximum`)
- `allOf` composition has no direct Avro equivalent (flatten manually)
- `$ref` maps to Avro named type references (register referenced types first)

---

## Type Mapping: Protobuf to JSON Schema

| Protobuf | JSON Schema |
|----------|-------------|
| message | `{"type": "object", "properties": {...}}` |
| enum | `{"type": "string", "enum": [...]}` (string values) |
| `repeated` | `{"type": "array", "items": {...}}` |
| `map<K, V>` | `{"type": "object", "additionalProperties": {...}}` |
| `oneof` | `{"oneOf": [...]}` |
| scalar types | corresponding JSON Schema types |

---

## Type Mapping: JSON Schema to Protobuf

| JSON Schema | Protobuf |
|-------------|----------|
| `properties` | message fields (assign field numbers sequentially) |
| `enum` | enum (add `UNSPECIFIED = 0` as first value) |
| `required` | N/A (all proto3 fields are optional by default) |
| constraints (`pattern`, `min`, `max`) | lost (no proto3 equivalent) |
| `$ref` | `import` + message reference |

---

## General Guidance

- Create NEW subjects for the migrated format (e.g., `orders-value` becomes `orders-value-proto`)
- Do NOT change the schema format in an existing subject -- this is a breaking change
- Schema IDs will change -- update all serializer/deserializer configurations
- Test the converted schema with **validate_schema** before registering
- Use **diff_schemas** to compare the original and converted schemas side by side
- For complex schemas with references, migrate referenced schemas first
