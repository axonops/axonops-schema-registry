# Schema Types

## Overview

AxonOps Schema Registry supports three schema types: **AVRO**, **PROTOBUF**, and **JSON**. Each type has its own parser, canonicalization strategy, fingerprinting algorithm, and compatibility checker.

When registering a schema via `POST /subjects/{subject}/versions`, the `schemaType` field is optional. If omitted, it defaults to `AVRO`. For Protobuf and JSON Schema, `schemaType` must be explicitly set to `PROTOBUF` or `JSON`, respectively.

You can query the supported types at any time:

```bash
curl http://localhost:8081/schemas/types
```

```json
["AVRO", "JSON", "PROTOBUF"]
```

---

## Avro

Avro is the default schema type. If you register a schema without specifying `schemaType`, the registry treats it as Avro.

### Supported Types

Avro supports the following primitive types:

- `null`, `boolean`, `int`, `long`, `float`, `double`, `bytes`, `string`

And the following complex types:

- `record` -- named type with a set of fields
- `enum` -- named type with a fixed set of symbols
- `array` -- ordered collection of items
- `map` -- key-value pairs (keys are always strings)
- `union` -- a value that matches one of several types
- `fixed` -- fixed-size byte sequence

### Logical Types

Avro logical types annotate primitive types with higher-level semantics:

| Logical Type | Underlying Type | Description |
|---|---|---|
| `date` | `int` | Days since Unix epoch |
| `time-millis` | `int` | Milliseconds since midnight |
| `time-micros` | `long` | Microseconds since midnight |
| `timestamp-millis` | `long` | Milliseconds since Unix epoch |
| `timestamp-micros` | `long` | Microseconds since Unix epoch |
| `decimal` | `bytes` or `fixed` | Arbitrary-precision decimal |
| `uuid` | `string` | RFC 4122 UUID |

### Named Types and References

Named types (records, enums, and fixed) can be shared across subjects using schema references. When a schema references a named type defined in another subject, the reference is resolved at parse time using the `references` array.

### Aliases

Avro supports aliases on records and fields. Aliases enable backward-compatible renaming: a consumer using an old schema with the original name can still read data produced with the new name, as long as the old name appears as an alias.

### Canonicalization and Fingerprinting

Canonicalization follows the Avro specification for Parsing Canonical Form:

- Fields within records are ordered as `name`, `type`, `fields` (for records), `symbols` (for enums), `items` (for arrays), `values` (for maps), `size` (for fixed)
- Non-canonical fields (`doc`, `aliases`, `order`) are stripped
- The `default` field is included in the canonical form so that schemas differing only in default values are treated as distinct

Fingerprinting computes the SHA-256 hash of the canonical form.

### Registration Example

Register a `User` record under the subject `users-value`:

```bash
curl -X POST http://localhost:8081/subjects/users-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
  }'
```

```json
{"id": 1}
```

Because `schemaType` is omitted, the registry defaults to `AVRO`.

### Avro with References

Register a shared `Address` record, then reference it from a `Customer` schema:

```bash
# Step 1: Register the Address schema
curl -X POST http://localhost:8081/subjects/address-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"},{\"name\":\"zip\",\"type\":\"string\"}]}"
  }'
```

```bash
# Step 2: Register Customer, referencing Address
curl -X POST http://localhost:8081/subjects/customer-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Customer\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"address\",\"type\":\"com.example.Address\"}]}",
    "references": [
      {
        "name": "com.example.Address",
        "subject": "address-value",
        "version": 1
      }
    ]
  }'
```

For Avro, the `name` field in the reference matches the fully qualified name of the referenced type.

---

## Protobuf

Protobuf schemas use Protocol Buffers definition syntax. The `schemaType` field must be set to `PROTOBUF` when registering.

### Supported Features

- **Syntax**: proto2 and proto3
- **Message types**: messages, nested messages, enums, oneofs, maps
- **Service definitions**: services with unary and streaming RPCs
- **Package declarations**: fully qualified naming
- **Options**: file, message, and field options are preserved
- **Imports**: resolved via schema references

### Canonicalization and Fingerprinting

Protobuf schemas are normalized by reconstructing a deterministic representation from the compiled file descriptor:

- Messages are sorted by name
- Fields within messages are sorted by field number
- Enums are sorted by name; enum values are sorted by number
- Nested messages and enums are recursively normalized
- Services and their methods are sorted by name
- Map entry types are rendered as `map<KeyType, ValueType>` syntax

Fingerprinting computes the SHA-256 hash of this normalized form.

### Registration Example

Register a `User` message under the subject `users-proto-value`:

```bash
curl -X POST http://localhost:8081/subjects/users-proto-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "PROTOBUF",
    "schema": "syntax = \"proto3\";\npackage com.example;\n\nmessage User {\n  int32 id = 1;\n  string name = 2;\n  string email = 3;\n}"
  }'
```

```json
{"id": 1}
```

### Protobuf with Imports (References)

Protobuf schemas that use `import` statements need references to resolve the imported files. Register the imported schema first, then reference it by its import path.

```bash
# Step 1: Register the common proto
curl -X POST http://localhost:8081/subjects/common-proto-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "PROTOBUF",
    "schema": "syntax = \"proto3\";\npackage com.example;\n\nmessage User {\n  int32 id = 1;\n  string name = 2;\n}"
  }'
```

```bash
# Step 2: Register Order, importing User via reference
curl -X POST http://localhost:8081/subjects/order-proto-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "PROTOBUF",
    "schema": "syntax = \"proto3\";\nimport \"common.proto\";\n\nmessage Order {\n  int32 id = 1;\n  com.example.User user = 2;\n}",
    "references": [
      {
        "name": "common.proto",
        "subject": "common-proto-value",
        "version": 1
      }
    ]
  }'
```

For Protobuf, the `name` field in the reference matches the import path used in the `import` statement.

### Complex Protobuf Example

A schema demonstrating nested messages, enums, oneofs, and maps:

```bash
curl -X POST http://localhost:8081/subjects/events-proto-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "PROTOBUF",
    "schema": "syntax = \"proto3\";\npackage com.example.events;\n\nenum EventType {\n  UNKNOWN = 0;\n  CREATED = 1;\n  UPDATED = 2;\n  DELETED = 3;\n}\n\nmessage Event {\n  string event_id = 1;\n  EventType type = 2;\n  int64 timestamp = 3;\n  map<string, string> metadata = 4;\n\n  oneof payload {\n    CreatePayload create = 10;\n    UpdatePayload update = 11;\n  }\n\n  message CreatePayload {\n    string name = 1;\n  }\n\n  message UpdatePayload {\n    string name = 1;\n    string previous_name = 2;\n  }\n}"
  }'
```

---

## JSON Schema

JSON Schema defines the structure and validation constraints for JSON data. The `schemaType` field must be set to `JSON` when registering.

### Supported Drafts

The registry uses **Draft-07** as the primary JSON Schema draft. Schemas written for Draft 2020-12 are also accepted.

### Supported Keywords

JSON Schema provides a rich vocabulary for validation:

**Type constraints**: `type`, `enum`, `const`

**Object keywords**: `properties`, `required`, `additionalProperties`, `minProperties`, `maxProperties`, `patternProperties`, `dependencies`

**Array keywords**: `items`, `minItems`, `maxItems`, `uniqueItems`, `additionalItems`, `contains`

**String keywords**: `minLength`, `maxLength`, `pattern`, `format`

**Numeric keywords**: `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`, `multipleOf`

**Composition keywords**: `allOf`, `anyOf`, `oneOf`, `not`

**Conditional keywords**: `if`, `then`, `else`

**Format values**: `email`, `uri`, `date-time`, `date`, `time`, `hostname`, `ipv4`, `ipv6`, `uuid`, and others

### Canonicalization and Fingerprinting

JSON Schema canonicalization produces a deterministic JSON representation:

- Object keys are sorted alphabetically at every level
- Numbers are normalized (integers rendered without decimal points)
- Whitespace is stripped

Fingerprinting computes the SHA-256 hash of the canonical form.

### Registration Example

Register a `User` JSON Schema under the subject `users-json-value`:

```bash
curl -X POST http://localhost:8081/subjects/users-json-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "JSON",
    "schema": "{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"integer\"},\"name\":{\"type\":\"string\"},\"email\":{\"type\":\"string\",\"format\":\"email\"}},\"required\":[\"id\",\"name\"]}"
  }'
```

```json
{"id": 1}
```

### JSON Schema with References

JSON Schema supports `$ref` for referencing schemas registered in other subjects.

```bash
# Step 1: Register the Address schema
curl -X POST http://localhost:8081/subjects/address-json-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "JSON",
    "schema": "{\"type\":\"object\",\"properties\":{\"street\":{\"type\":\"string\"},\"city\":{\"type\":\"string\"},\"zip\":{\"type\":\"string\"}},\"required\":[\"street\",\"city\"]}"
  }'
```

```bash
# Step 2: Register Customer, referencing Address via $ref
curl -X POST http://localhost:8081/subjects/customer-json-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "JSON",
    "schema": "{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"integer\"},\"name\":{\"type\":\"string\"},\"address\":{\"$ref\":\"address.json\"}},\"required\":[\"id\",\"name\"]}",
    "references": [
      {
        "name": "address.json",
        "subject": "address-json-value",
        "version": 1
      }
    ]
  }'
```

For JSON Schema, the `name` field in the reference matches the URI used in `$ref`.

### Complex JSON Schema Example

A schema using composition, conditional logic, and format validation:

```bash
curl -X POST http://localhost:8081/subjects/contacts-json-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "JSON",
    "schema": "{\"type\":\"object\",\"properties\":{\"id\":{\"type\":\"integer\"},\"name\":{\"type\":\"string\",\"minLength\":1,\"maxLength\":200},\"contact_type\":{\"type\":\"string\",\"enum\":[\"email\",\"phone\",\"address\"]},\"value\":{\"type\":\"string\"}},\"required\":[\"id\",\"name\",\"contact_type\",\"value\"],\"if\":{\"properties\":{\"contact_type\":{\"const\":\"email\"}}},\"then\":{\"properties\":{\"value\":{\"format\":\"email\"}}},\"additionalProperties\":false}"
  }'
```

---

## Schema References

All three schema types support cross-subject references, enabling schema reuse and modular design. A reference tells the registry where to find a schema that the current schema depends on.

### Reference Structure

Each reference contains three fields:

```json
{
  "name": "reference-name",
  "subject": "referenced-subject",
  "version": 1
}
```

| Field | Description |
|---|---|
| `name` | The identifier used within the schema to refer to the dependency. Interpretation varies by schema type. |
| `subject` | The subject under which the referenced schema is registered. |
| `version` | The version of the referenced schema to resolve. |

### How `name` Is Interpreted Per Schema Type

| Schema Type | `name` Matches |
|---|---|
| AVRO | Fully qualified name of the referenced type (e.g., `com.example.Address`) |
| PROTOBUF | Import path in the `import` statement (e.g., `common.proto`) |
| JSON | URI used in `$ref` (e.g., `address.json`) |

### Reference Resolution

When the registry receives a schema with references, it:

1. Looks up each referenced subject and version in the storage backend
2. Retrieves the referenced schema content
3. Passes the resolved content to the parser alongside the main schema
4. The parser uses the resolved content to validate and compile the complete schema

If any reference cannot be resolved (subject not found, version not found), the registration fails with an appropriate error.

---

## Schema Deduplication

The registry deduplicates schemas by content. When the same schema content is registered under different subjects, it receives the same global schema ID. Two schemas are considered identical when their SHA-256 fingerprints match.

For example, if you register the same Avro record under `users-value` and `customers-value`, both subjects point to the same global schema ID. This means:

- `GET /schemas/ids/{id}` returns the schema once
- `GET /schemas/ids/{id}/subjects` returns both subjects

Deduplication is based on the fingerprint of the canonical form, not the raw input string. Two schemas with different whitespace or field ordering but identical canonical forms share the same ID.

---

## Schema Normalization

When `normalize=true` is set on the subject configuration, the registry normalizes schemas before computing fingerprints and performing deduplication. This means semantically identical schemas with different formatting or non-significant ordering differences receive the same global ID.

Set normalization on a subject:

```bash
curl -X PUT http://localhost:8081/config/users-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "BACKWARD", "normalize": true}'
```

You can also pass `normalize=true` as a query parameter on individual registration requests:

```bash
curl -X POST "http://localhost:8081/subjects/users-value/versions?normalize=true" \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"},{\"name\":\"name\",\"type\":\"string\"}]}"
  }'
```

Without normalization, schemas are fingerprinted using the canonical form of the raw input. With normalization, additional formatting differences are resolved before fingerprinting, broadening the set of inputs that map to the same ID.

---

## Formatted Output

When retrieving a schema, you can request a specific output format using the `format` query parameter on `GET /schemas/ids/{id}/schema`:

| Schema Type | Format Value | Description |
|---|---|---|
| AVRO | `resolved` | Inlines all referenced types into the schema |
| PROTOBUF | `serialized` | Returns a base64-encoded `FileDescriptorProto` |
| All types | (default) | Returns the canonical form |

Example:

```bash
curl "http://localhost:8081/schemas/ids/1/schema?format=resolved"
```

---

## Related Documentation

- [Getting Started](getting-started.md) -- register your first schema in five minutes
- [Compatibility](compatibility.md) -- compatibility levels and how they apply to each schema type
- [API Reference](api-reference.md) -- complete endpoint documentation for schema registration, retrieval, and deletion
- [Configuration](configuration.md) -- server, storage, and compatibility configuration options
