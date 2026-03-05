# Core Concepts

## What Is a Schema Registry?

A schema registry is a centralized service that stores and manages **schemas** -- the formal definitions of data structures -- used by producers and consumers in an event streaming platform. It acts as the single source of truth for what data looks like, ensuring that every application writing to or reading from a Kafka topic agrees on the structure of the messages.

A schema registry prevents deserialization failures, data corruption, and silent data loss by:

1. **Storing schemas centrally** -- every data structure is registered and versioned.
2. **Enforcing compatibility** -- new schema versions are checked against previous versions before acceptance.
3. **Embedding schema IDs in messages** -- each Kafka message carries a compact reference to its schema.
4. **Decoupling producers from consumers** -- teams evolve schemas independently within compatibility contracts.

## Schemas

A **schema** is a formal definition of a data structure specifying fields, types, default values, and constraints. AxonOps Schema Registry supports three formats:

- **AVRO** -- compact binary format with rich type system, widely used in Kafka. Default when schemaType is omitted.
- **PROTOBUF** -- Google's language-neutral binary serialization (proto2 and proto3).
- **JSON** -- a vocabulary for annotating and validating JSON documents (Draft-07 and Draft 2020-12).

## Subjects

A **subject** is a named scope under which schema versions are registered. Subjects are the primary organizational unit. By default, each Kafka topic has two subjects:

- **{topic}-key** for the message key schema
- **{topic}-value** for the message value schema

Example: topic "orders" has subjects "orders-key" and "orders-value".

## Subject Naming Strategies

The naming strategy controls how subject names are derived from Kafka topics and schemas:

| Strategy | Pattern | Example | Use When |
|----------|---------|---------|----------|
| **TopicNameStrategy** (default) | {topic}-key / {topic}-value | orders-value | One schema per topic |
| **RecordNameStrategy** | {fully.qualified.RecordName} | com.example.Order | Same record type across multiple topics |
| **TopicRecordNameStrategy** | {topic}-{fully.qualified.RecordName} | orders-com.example.Order | Multiple event types per topic with topic context |

The naming strategy is a **client-side** configuration on the serializer. The registry accepts any subject name.

## Versions

Each time a new schema is registered under a subject, it gets a **version number** starting from 1. Versions are:

- **Immutable** -- once registered, a version cannot be changed.
- **Deduplicated** -- registering the same schema content again returns the existing version and ID.
- **Sequential** -- versions increment by 1 for each new registration.

## Schema IDs

Every unique schema gets a **globally unique integer ID**. This ID is embedded in Kafka messages using a 5-byte prefix (1 magic byte 0x00 + 4-byte big-endian ID). Schema IDs are:

- **Globally unique** -- no two different schemas share an ID, regardless of subject.
- **Stable** -- once assigned, an ID never changes.
- **Monotonically increasing** -- newer schemas get higher IDs.
- **Content-addressed** -- the same logical schema registered under different subjects shares a single ID.

## The Wire Format

Every message produced through a schema-aware serializer follows this binary layout:

    [Magic Byte 0x00] [Schema ID - 4 bytes big-endian] [Serialized Payload]

The 5-byte overhead is the only cost of using a schema registry.

## Schema Deduplication

The registry uses **content-addressed storage** based on SHA-256 fingerprints:

1. The schema is parsed and converted to its canonical form.
2. A SHA-256 fingerprint is computed.
3. If a schema with the same fingerprint exists, the existing ID is returned.
4. If no match exists, a new ID is allocated.

The same logical schema registered under multiple subjects shares a single schema ID.

## Schema References

A **reference** is a pointer from one schema to another, enabling cross-subject schema composition:

- **Avro**: Named type references (e.g., a Customer record defined in one subject and used in another)
- **Protobuf**: import statements referencing types from other .proto definitions
- **JSON Schema**: $ref pointing to schemas in other subjects

Each reference has three fields: **name** (how the schema refers to it), **subject** (where it is registered), and **version** (which version to use).

## Modes

Modes control whether schema registration is allowed:

| Mode | Behavior |
|------|----------|
| **READWRITE** | Normal operation. Schemas can be registered and read. (Default) |
| **READONLY** | Reads allowed, new registrations rejected. |
| **READONLY_OVERRIDE** | Like READONLY, but individual requests MAY override. |
| **IMPORT** | Allows registering schemas with specific IDs (used for migration). |

Modes can be set globally or per-subject. A per-subject mode overrides the global mode.

## The Registration and Serialization Flow

**Producer flow:**
1. Application passes a record to the serializer.
2. Serializer registers the schema with the registry (POST /subjects/{subject}/versions).
3. Registry checks compatibility and returns the schema ID (or rejects).
4. Serializer prepends the 5-byte header and serializes the payload.
5. Message is produced to Kafka.

**Consumer flow:**
1. Consumer receives raw bytes from Kafka.
2. Deserializer reads the magic byte and 4-byte schema ID from the prefix.
3. Fetches the schema from the registry by ID (GET /schemas/ids/{id}).
4. Deserializes the payload using the schema.

Both serializer and deserializer cache schemas locally after first use.

## MCP Tools for Core Operations

- **list_subjects** -- list all registered subjects
- **register_schema** -- register a new schema version
- **get_latest_schema** -- get the current schema for a subject
- **get_schema_by_id** -- fetch a schema by its global ID
- **list_versions** -- list all versions of a subject
- **get_schema_version** -- get a specific version
- **get_mode / set_mode** -- read or change the registry mode
- **get_schema_types** -- list supported schema types (AVRO, PROTOBUF, JSON)
