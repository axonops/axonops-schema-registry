# Schema Types Reference

## Avro

### Primitive Types
null, boolean, int (32-bit), long (64-bit), float (32-bit IEEE 754), double (64-bit IEEE 754), bytes, string

### Complex Types
- **record** -- named type with fields. Fields have name, type, default, doc, order, aliases.
- **enum** -- named type with symbol list. Has name, symbols, default, aliases, namespace.
- **array** -- ordered collection. Has items (element type).
- **map** -- string-keyed dictionary. Has values (value type).
- **union** -- one of several types. Written as JSON array: ["null", "string"].
- **fixed** -- fixed-size byte array. Has name and size.

### Logical Types
Logical types annotate primitive types with semantic meaning:

| Logical Type | Underlying Type | Description |
|-------------|----------------|-------------|
| date | int | Days since Unix epoch |
| time-millis | int | Milliseconds since midnight |
| time-micros | long | Microseconds since midnight |
| timestamp-millis | long | Milliseconds since Unix epoch |
| timestamp-micros | long | Microseconds since Unix epoch |
| decimal | bytes or fixed | Arbitrary precision decimal (requires precision and scale) |
| uuid | string | RFC 4122 UUID |
| duration | fixed(12) | Three unsigned 32-bit ints: months, days, milliseconds |

### Named Types and References
Named types (record, enum, fixed) have a fully qualified name (namespace + name). When referencing a named type defined in another subject, use schema references with the fully qualified name as the reference **name**.

### Canonicalization
Avro schemas are normalized to a canonical form: fields ordered deterministically, whitespace removed, optional attributes stripped. The SHA-256 fingerprint is computed from this canonical form.

### Aliases
Records and fields can have aliases -- alternative names used for compatibility resolution. Aliases allow renaming without breaking backward/forward compatibility.

## Protobuf

### Supported Features
- proto3 syntax (recommended) and proto2 syntax.
- Messages, enums, oneofs, maps, repeated fields.
- Nested messages and enums.
- Import statements (resolved via schema references).
- Well-known types (google.protobuf.Timestamp, Duration, Struct, etc.).
- Service definitions (gRPC metadata, no wire impact).
- Extensions and custom options.

### Wire Types
Protobuf encodes fields using a (field_number, wire_type) pair:

| Wire Type | ID | Types |
|-----------|-----|-------|
| Varint | 0 | int32, int64, uint32, uint64, sint32, sint64, bool, enum |
| 64-bit | 1 | fixed64, sfixed64, double |
| Length-delimited | 2 | string, bytes, messages, packed repeated |
| 32-bit | 5 | fixed32, sfixed32, float |

### Field Numbers
- MUST be unique within a message.
- MUST NOT be reused after deletion (reserve them instead).
- Range 1-15 use one byte for the tag (prefer for frequently used fields).
- Range 16-2047 use two bytes.

### Canonicalization
Protobuf schemas are normalized: comments stripped, whitespace normalized, fields sorted by number. The fingerprint is computed from this canonical form.

## JSON Schema

### Supported Drafts
- **Draft-07** (most widely used)
- **Draft 2020-12** (latest standard)

### Core Keywords

**Type system:** type (string, number, integer, boolean, object, array, null)

**Object keywords:** properties, required, additionalProperties, patternProperties, minProperties, maxProperties

**Array keywords:** items, prefixItems (2020-12), additionalItems (Draft-07), minItems, maxItems, uniqueItems

**String keywords:** minLength, maxLength, pattern, format (email, uri, date-time, uuid, etc.)

**Numeric keywords:** minimum, maximum, exclusiveMinimum, exclusiveMaximum, multipleOf

**Enumeration:** enum, const

**Composition:** oneOf, anyOf, allOf, not

**References:** $ref (local via #/definitions/ or #/$defs/, cross-subject via schema references)

**Conditional:** if, then, else (Draft-07+)

**Dependencies:** dependencies (Draft-07), dependentRequired, dependentSchemas (2020-12)

### Canonicalization
JSON Schema is normalized: keys sorted, whitespace removed, semantically equivalent constructs unified. The fingerprint is computed from the canonical form.

## Schema References

References enable cross-subject schema composition. Each reference has:

| Field | Type | Description |
|-------|------|-------------|
| **name** | string | How the referencing schema refers to this dependency |
| **subject** | string | Subject where the referenced schema is registered |
| **version** | integer | Version number of the referenced schema |

The **name** field is interpreted differently per schema type:
- **Avro**: fully qualified type name (e.g., "com.example.Address")
- **Protobuf**: import path (e.g., "address.proto")
- **JSON Schema**: reference URL (e.g., "address.json")

## Schema Normalization

Pass ?normalize=true on registration to normalize the schema before storing. Normalization produces a canonical form that makes fingerprint comparison more reliable. Without normalization, semantically equivalent schemas with different formatting get different fingerprints.

## MCP Tools

- **register_schema** -- register with schemaType and references
- **validate_schema** -- validate syntax without registering
- **get_schema_types** -- list supported types (AVRO, PROTOBUF, JSON)
