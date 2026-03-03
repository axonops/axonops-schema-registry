package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerGlossaryResources() {
	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/core-concepts",
		Name:        "glossary-core-concepts",
		Description: "Schema registry fundamentals: what a schema registry is, subjects, versions, IDs, deduplication, modes, naming strategies, and the serialization flow",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryCoreConceptsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/compatibility",
		Name:        "glossary-compatibility",
		Description: "All 7 compatibility modes, Avro type promotions, Protobuf wire types, JSON Schema constraints, transitive semantics, and configuration resolution",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryCompatibilityResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/data-contracts",
		Name:        "glossary-data-contracts",
		Description: "Data contracts: metadata properties, tags, sensitive fields, rulesets (domain/migration/encoding), rule structure, 3-layer merge, and optimistic concurrency",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryDataContractsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/encryption",
		Name:        "glossary-encryption",
		Description: "Client-side field level encryption (CSFLE): envelope encryption, KEK/DEK model, KMS providers, algorithms, key rotation, and rewrapping",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryEncryptionResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/contexts",
		Name:        "glossary-contexts",
		Description: "Multi-tenancy via contexts: default context, __GLOBAL, qualified subjects, URL routing, isolation guarantees, and 4-tier config/mode inheritance",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryContextsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/exporters",
		Name:        "glossary-exporters",
		Description: "Schema linking via exporters: exporter model, lifecycle states (STARTING/RUNNING/PAUSED/ERROR), context types (AUTO/CUSTOM/NONE), and configuration",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryExportersResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/schema-types",
		Name:        "glossary-schema-types",
		Description: "Deep reference for Avro (types, logical types, aliases, canonicalization), Protobuf (proto3, well-known types, wire types), and JSON Schema (drafts, keywords, combinators)",
		MIMEType:    "text/markdown",
	}, s.handleGlossarySchemaTypesResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/design-patterns",
		Name:        "glossary-design-patterns",
		Description: "Common schema design patterns: event envelope, entity lifecycle, snapshot vs delta, fat vs thin events, shared types, three-phase rename, and CI/CD integration",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryDesignPatternsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/best-practices",
		Name:        "glossary-best-practices",
		Description: "Actionable best practices for Avro, Protobuf, and JSON Schema: field naming, nullability, evolution readiness, common mistakes, and per-format guidance",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryBestPracticesResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://glossary/migration",
		Name:        "glossary-migration",
		Description: "Confluent migration: step-by-step procedure, IMPORT mode, ID preservation, the import API, verification, and rollback",
		MIMEType:    "text/markdown",
	}, s.handleGlossaryMigrationResource)
}

// --- Glossary resource handlers ---

func (s *Server) handleGlossaryCoreConceptsResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryCoreConceptsContent)
}

func (s *Server) handleGlossaryCompatibilityResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryCompatibilityContent)
}

func (s *Server) handleGlossaryDataContractsResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryDataContractsContent)
}

func (s *Server) handleGlossaryEncryptionResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryEncryptionContent)
}

func (s *Server) handleGlossaryContextsResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryContextsContent)
}

func (s *Server) handleGlossaryExportersResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryExportersContent)
}

func (s *Server) handleGlossarySchemaTypesResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossarySchemaTypesContent)
}

func (s *Server) handleGlossaryDesignPatternsResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryDesignPatternsContent)
}

func (s *Server) handleGlossaryBestPracticesResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryBestPracticesContent)
}

func (s *Server) handleGlossaryMigrationResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	return resourceMarkdown(req.Params.URI, glossaryMigrationContent)
}

// --- Glossary content constants ---
// Each constant is a complete reference document written for AI assistants.
// Content uses double-star bold instead of backtick code fences for inline
// terms because Go raw string literals cannot contain backticks.

const glossaryCoreConceptsContent = `# Core Concepts

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
`

const glossaryCompatibilityContent = `# Compatibility

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
`

const glossaryDataContractsContent = `# Data Contracts

## Overview

Data contracts are governance policies attached to schemas. They allow you to annotate fields with descriptive metadata, classify sensitive data, and define rules applied during validation, migration, or serialization. Data contracts build on top of schema registration -- they add governance without changing how schemas are parsed, fingerprinted, or compatibility-checked.

## Metadata

Metadata provides descriptive annotations for a schema. It has three components:

| Component | Type | Purpose |
|-----------|------|---------|
| **properties** | map[string]string | Key-value annotations: "owner": "payments-team", "pii": "true", "domain": "billing" |
| **tags** | map[string][]string | Field-to-tag mapping. Keys are field paths (e.g., "Order.email"), values are tag arrays (e.g., ["PII", "GDPR"]) |
| **sensitive** | []string | Field paths containing sensitive data (e.g., ["ssn", "credit_card"]) |

## How Metadata Affects Schema Identity

- Metadata is **NOT** included in the SHA-256 fingerprint.
- Same schema text with **different metadata** creates a **new version** but gets the **same global ID**.
- Same schema text with **same metadata** is treated as a duplicate -- existing version and ID returned.
- This separates content identity (what the schema describes) from governance identity (how it is annotated).

## RuleSets

A RuleSet defines executable governance policies in three categories:

| Category | Field | Purpose |
|----------|-------|---------|
| **Domain rules** | domainRules | Validation/transformation on schema content (e.g., "all field names MUST be camelCase") |
| **Migration rules** | migrationRules | Rules during schema evolution (e.g., "renamed fields MUST provide a migration path") |
| **Encoding rules** | encodingRules | Rules during serialization/deserialization (e.g., "encrypt PII-tagged fields") |

## Rule Structure

Each rule has these fields:

| Field | Type | Description |
|-------|------|-------------|
| **name** | string | Unique name for this rule |
| **kind** | string | CONDITION (validate) or TRANSFORM (modify) |
| **mode** | string | WRITE, READ, UPGRADE, DOWNGRADE, WRITEREAD, UPDOWN |
| **type** | string | Rule type: CEL, JSON_TRANSFORM, ENCRYPT, etc. |
| **tags** | []string | Optional field tags this rule applies to |
| **params** | map[string]string | Rule-specific parameters |
| **expr** | string | Rule expression |
| **onSuccess** | string | Action on success: NONE, ERROR |
| **onFailure** | string | Action on failure: NONE, ERROR, DLQ |
| **disabled** | boolean | Whether this rule is currently disabled |

## Config-Level Defaults and Overrides

Rather than attaching metadata/rules to every registration, set defaults and overrides at the config level:

| Config Field | Purpose |
|-------------|---------|
| **defaultMetadata** | Merged into registrations that do not specify their own metadata |
| **defaultRuleSet** | Merged into registrations that do not specify their own rules |
| **overrideMetadata** | ALWAYS takes precedence over both defaults and request values |
| **overrideRuleSet** | ALWAYS takes precedence over both defaults and request rules |

## The 3-Layer Merge

When a schema is registered, the registry applies a 3-layer merge:

    Layer 1: defaultMetadata / defaultRuleSet       (base, from config)
    Layer 2: request metadata / ruleSet              (from POST body)
    Layer 3: overrideMetadata / overrideRuleSet      (from config, always wins)

Properties merge by key. Tags merge by field path. Rules merge by name. The override layer always wins.

## Inheritance from Previous Versions

When registering a new version, metadata and rules from the **previous version** are carried forward unless explicitly replaced. This means governance policies accumulate across versions.

## Optimistic Concurrency

The special metadata property **confluent:version** enables optimistic concurrency control:
- Include it in a registration request with the expected metadata version number.
- If the current metadata version does not match, the registration is rejected with HTTP 409.
- This prevents concurrent updates from silently overwriting each other.

## MCP Tools

- **set_config_full / get_config_full** -- manage config with metadata and ruleSet defaults/overrides
- **get_subject_metadata** -- inspect applied metadata on a subject
- **register_schema** -- register with metadata and ruleSet in the request body
- **get_latest_schema** -- fetch current schema including metadata
`

const glossaryEncryptionContent = `# Client-Side Field Level Encryption (CSFLE)

## Overview

CSFLE allows producers to encrypt specific fields in a message before sending it to Kafka. Instead of encrypting entire topics, CSFLE targets individual fields -- an email, SSN, or credit card number -- leaving the rest in plaintext for indexing, filtering, and debugging.

The schema registry acts as the **key metadata store**. It does not perform encryption itself. It stores key metadata that serializer/deserializer clients use to encrypt and decrypt fields.

## The Envelope Encryption Pattern

CSFLE uses a two-tier key hierarchy:

| Key Type | Purpose | Where the Actual Key Lives |
|----------|---------|---------------------------|
| **KEK** (Key Encryption Key) | References an external KMS key; used to wrap (encrypt) DEKs | External KMS (Vault, AWS KMS, Azure Key Vault, GCP KMS) |
| **DEK** (Data Encryption Key) | The actual encryption key for field values | Stored in registry as encrypted bytes (wrapped by KEK) |

**How it works:**
1. A DEK is generated (by client or registry).
2. The DEK is encrypted ("wrapped") using the KEK via the external KMS.
3. The wrapped DEK is stored in the registry.
4. At encryption time, the client fetches the wrapped DEK, unwraps it via the KMS, and uses the plaintext DEK to encrypt field values.
5. The plaintext DEK never leaves the client. The registry only stores the wrapped form.

## KEK Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **name** | string | Yes | Unique identifier |
| **kmsType** | string | Yes | KMS provider: aws-kms, azure-kms, gcp-kms, hcvault, openbao |
| **kmsKeyId** | string | Yes | KMS-specific key identifier (ARN, resource name, transit path) |
| **kmsProps** | map | No | Provider-specific properties (region, endpoint, auth) |
| **doc** | string | No | Human-readable description |
| **shared** | boolean | No | If true, all DEKs share the same key material. Default: false |

The **shared** flag controls DEK creation behavior:
- shared=false (default): client generates its own key material, sends only encrypted form.
- shared=true: registry generates key material via KMS, returns plaintext on creation for client caching.

## DEK Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| **kekName** | string | Yes | Parent KEK name |
| **subject** | string | Yes | Schema subject this DEK encrypts for |
| **version** | integer | No | Auto-assigned sequential version |
| **algorithm** | string | No | AES256_GCM (default), AES128_GCM, or AES256_SIV |
| **encryptedKeyMaterial** | string | No | Wrapped DEK bytes (base64) |

DEKs are versioned per subject under a KEK, enabling key rotation.

## Supported KMS Types

| KMS Type | Value | Status |
|----------|-------|--------|
| HashiCorp Vault | hcvault | Production |
| OpenBao | openbao | Production |
| AWS KMS | aws-kms | Coming Soon |
| Azure Key Vault | azure-kms | Coming Soon |
| GCP KMS | gcp-kms | Coming Soon |

## Supported Algorithms

| Algorithm | Key Size | Mode | Use Case |
|-----------|----------|------|----------|
| **AES256_GCM** | 256-bit | Galois/Counter | Default. Authenticated encryption. Recommended for most use cases. |
| **AES128_GCM** | 128-bit | Galois/Counter | Authenticated encryption with lower key size. |
| **AES256_SIV** | 256-bit | Synthetic IV | Deterministic encryption. Same plaintext produces same ciphertext. Enables equality searches on encrypted fields but leaks value equality information. |

## Key Rotation

To rotate encryption keys:
1. Create a new DEK version for the subject under the same KEK.
2. New messages are encrypted with the latest DEK version.
3. Old messages can still be decrypted with previous DEK versions.

## Rewrapping

When the underlying KMS key is rotated, use the **rewrap** operation:
- The DEK plaintext key material stays the same.
- The encryptedKeyMaterial is re-encrypted with the new KMS key version.
- Existing encrypted data remains readable without re-encryption.

## Soft-Delete Model

Both KEKs and DEKs support three-level deletion:

| Operation | Effect | Reversible? |
|-----------|--------|-------------|
| DELETE | Sets deleted=true. Hidden from listings. | Yes, via undelete |
| DELETE ?permanent=true | Permanently removed from storage. | No |
| POST .../undelete | Clears deleted flag. | N/A |

## MCP Tools

- **create_kek / get_kek / list_keks / update_kek / delete_kek** -- manage KEKs
- **create_dek / get_dek / list_deks / delete_dek** -- manage DEKs
- **test_kek** -- test KMS connectivity for a KEK
- **get_dek_versions** -- list DEK version numbers
`

const glossaryContextsContent = `# Contexts (Multi-Tenancy)

## Overview

A **context** is a logical namespace within the registry that provides multi-tenant isolation. Each context operates as an independent schema registry -- with its own schema IDs, subjects, version histories, compatibility configuration, and modes -- while sharing a single registry deployment.

## Key Concepts

| Concept | Details |
|---------|---------|
| **Default context** | "." (dot). All operations target this context unless specified otherwise. |
| **Global context** | "__GLOBAL". Used for cross-context configuration and mode settings. |
| **Qualified subjects** | Format: :.contextname:subject. Embeds context in the subject string. |
| **URL prefix routing** | /contexts/{context}/subjects/... scopes operations to a context. |

## What Contexts Isolate

| Resource | Isolation |
|----------|-----------|
| **Schema IDs** | Each context has its own auto-incrementing ID sequence. ID 1 in .team-a is independent of ID 1 in .team-b. |
| **Subjects** | Same subject name in different contexts = different subjects with separate version histories. |
| **Versions** | Version numbering is independent per context. |
| **Compatibility config** | Global and per-subject settings are scoped to the context. |
| **Modes** | READWRITE/READONLY/IMPORT modes are scoped to the context. |

## Accessing Contexts

Two equivalent methods:

**Qualified subject names:**
    :.team-a:orders-value
    POST /subjects/:.team-a:orders-value/versions

**URL prefix routing:**
    POST /contexts/.team-a/subjects/orders-value/versions

Both produce identical results.

## Context Naming Rules

- Alphanumeric characters, hyphens, underscores, and dots.
- Maximum 255 characters.
- Context names are prefixed with a dot in listings (e.g., ".team-a", ".staging").
- The default context "." is always present.

## The 4-Tier Config/Mode Inheritance Chain

Configuration and mode settings cascade through 4 levels (highest to lowest precedence):

1. **Per-subject** -- most specific, overrides everything below. Set via set_config/set_mode with a subject.
2. **Context global** -- per-context default. Set via set_config/set_mode with no subject within the context.
3. **Global (__GLOBAL)** -- cross-context default. Set via set_config/set_mode in the __GLOBAL context.
4. **Server default** -- hardcoded: BACKWARD compatibility, READWRITE mode.

To check the effective (resolved) config: **get_config** with a subject returns the resolved value after walking the chain.

## Common Use Cases

| Use Case | Description |
|----------|-------------|
| **Team isolation** | .team-a and .team-b get independent namespaces |
| **Environment separation** | .staging and .production schemas side by side |
| **Schema Linking** | Confluent Schema Linking uses contexts for cross-cluster replication |
| **Multi-tenant SaaS** | Each tenant gets a dedicated namespace |

## MCP Tools

- **list_contexts** -- list all contexts
- **get_config / set_config / delete_config** -- manage compatibility per context/subject
- **get_mode / set_mode / delete_mode** -- manage modes per context/subject
- **list_subjects** -- list subjects in a context

## MCP Resources

- schema://contexts -- list all contexts
- schema://contexts/{context}/subjects -- subjects in a specific context
`

const glossaryExportersContent = `# Exporters (Schema Linking)

## Overview

Exporters enable **schema linking** -- replicating schemas from one registry context (or the entire registry) to a destination. They are used for disaster recovery, cross-datacenter replication, and environment promotion (e.g., staging to production).

## Exporter Data Model

| Field | Type | Description |
|-------|------|-------------|
| **name** | string | Unique identifier for this exporter |
| **contextType** | string | AUTO, CUSTOM, or NONE |
| **context** | string | Target context name (used with CUSTOM) |
| **subjects** | []string | Subjects to export (empty = all) |
| **subjectRenameFormat** | string | Optional rename pattern for exported subjects |
| **config** | map[string]string | Destination registry connection details |

## Context Types

| Type | Behavior |
|------|----------|
| **AUTO** | Exports all subjects automatically. New subjects are picked up without configuration changes. |
| **CUSTOM** | Exports only specified subjects. Optional rename format controls how subject names appear at the destination. |
| **NONE** | No context prefix on exported subjects. Subjects appear at the destination with their original names. |

## Exporter Lifecycle

Exporters have a state machine with these states:

    STARTING --> RUNNING --> PAUSED
                    |          |
                    v          v
                  ERROR    RUNNING (resume)

| State | Description |
|-------|-------------|
| **STARTING** | Exporter is initializing |
| **RUNNING** | Actively exporting schemas |
| **PAUSED** | Temporarily stopped (can be resumed) |
| **ERROR** | Failed; check status for error details |

## Lifecycle Operations

- **pause_exporter** -- pause a running exporter
- **resume_exporter** -- resume a paused exporter
- **reset_exporter** -- reset exporter state (clears offsets, restarts from beginning)

## Exporter Configuration

The config map contains destination connection details:

| Property | Description |
|----------|-------------|
| schema.registry.url | Destination registry URL |
| basic.auth.credentials.source | Auth method for destination |
| basic.auth.user.info | username:password for destination |

## MCP Tools

- **create_exporter / get_exporter / list_exporters / update_exporter / delete_exporter** -- manage exporters
- **get_exporter_status** -- check exporter state and progress
- **get_exporter_config / update_exporter_config** -- manage exporter configuration
- **pause_exporter / resume_exporter / reset_exporter** -- lifecycle control

## MCP Resources

- schema://exporters -- list all exporter names
- schema://exporters/{name} -- exporter details by name
`

const glossarySchemaTypesContent = `# Schema Types Reference

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
`

const glossaryDesignPatternsContent = `# Schema Design Patterns

## Event Envelope Pattern

Wrap every event in a standard envelope that carries metadata separate from the domain payload:

    {
      "event_id": "uuid",
      "event_type": "OrderCreated",
      "timestamp": "2026-01-15T10:30:00Z",
      "source": "orders-service",
      "correlation_id": "uuid",
      "payload": { ... domain-specific fields ... }
    }

**Benefits:** Consistent routing, filtering, and tracing across all events. The envelope schema evolves independently from payload schemas.

**Implementation:** Register the envelope as a shared schema via references. The payload type can be a union (Avro), oneof (Protobuf), or oneOf (JSON Schema).

## Entity Lifecycle Events

Model entity state changes as a sequence of typed events:

    UserCreated -> UserUpdated -> UserEmailVerified -> UserDeactivated

Each event type gets its own schema but shares common fields (entity_id, timestamp, actor). Use **TopicRecordNameStrategy** so multiple event types can share a single Kafka topic while having independent schema evolution.

## Snapshot vs Delta Events

| Pattern | Content | Use When |
|---------|---------|----------|
| **Snapshot** | Complete entity state | Consumers need the full picture; late-joining consumers; compacted topics |
| **Delta** | Only changed fields | High-frequency updates; bandwidth-sensitive; consumers track state locally |

**Snapshot example:** Every UserUpdated event contains all user fields.
**Delta example:** UserUpdated only contains the fields that changed plus the entity_id.

**Recommendation:** Start with snapshots. They are simpler and more resilient. Switch to deltas only when bandwidth or storage costs justify the added complexity.

## Fat vs Thin Events

| Pattern | Content | Use When |
|---------|---------|----------|
| **Fat events** | All data consumers might need | Consumers should be self-sufficient; avoid downstream lookups |
| **Thin events** | Minimal data + identifier for lookups | Events are high-volume; most consumers need only a subset |

**Fat event:** OrderCreated includes full customer info, product details, shipping address.
**Thin event:** OrderCreated includes only order_id, customer_id, total.

**Recommendation:** Prefer fat events for event sourcing and CQRS. Use thin events when the same data would be duplicated across many event types.

## Shared Types via References

Extract reusable types (Address, Money, ContactInfo) into their own subjects:

1. Register "Address" schema under subject "com.example.Address".
2. Reference it from "OrderCreated" using a schema reference.
3. "Address" evolves independently with its own compatibility policy.

**Versioning shared types:**
- Use FULL or FULL_TRANSITIVE compatibility for shared types (both producers and consumers must handle changes).
- Test referenced schemas across all dependents before publishing a new version.
- Use **get_referenced_by** to find all schemas that depend on a shared type.

## Three-Phase Rename Pattern

Renaming a field safely requires three schema versions:

**Phase 1: Add new field alongside old field.**
Both old_name and new_name exist. Producers write both. Consumers read from whichever exists.

**Phase 2: Deprecate old field.**
All consumers have been updated to read new_name. Mark old_name as deprecated (via doc or metadata).

**Phase 3: Remove old field.**
After all consumers have migrated, remove old_name. This step requires NONE compatibility or a new subject.

## Three-Phase Type Change Pattern

Changing a field type (e.g., string to int) follows the same three-phase approach:

**Phase 1:** Add new field with the new type alongside the old field.
**Phase 2:** Migrate all consumers to use the new field.
**Phase 3:** Remove the old field.

## CI/CD Integration

Integrate schema validation into your deployment pipeline:

1. **Pre-commit:** Run validate_schema to check syntax.
2. **PR check:** Run check_compatibility against the registry to verify the change is compatible.
3. **Deploy:** Register the schema as part of the deployment process.
4. **Post-deploy:** Verify the schema was registered successfully.

Use the MCP tools or the REST API in CI scripts:
- **validate_schema** -- syntax check without registration
- **check_compatibility** -- compatibility check without registration
- **register_schema** -- register on successful deployment

## Dead Letter Queue (DLQ) Pattern

When a consumer encounters a message it cannot deserialize:

1. Route the message to a DLQ topic.
2. Include the original topic, partition, offset, schema ID, and error message.
3. Register a DLQ schema that wraps the original message bytes.

The DLQ schema should use NONE compatibility (any message format may end up there).

## MCP Tools for Pattern Implementation

- **register_schema** -- register schemas with references
- **check_compatibility** -- validate before registering
- **get_referenced_by** -- find dependents of shared types
- **find_similar_schemas** -- find schemas with overlapping fields
- **get_dependency_graph** -- visualize the reference tree
- **diff_schemas** -- compare versions during evolution
`

const glossaryBestPracticesContent = `# Best Practices

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
`

const glossaryMigrationContent = `# Migrating from Confluent Schema Registry

## Why Schema ID Preservation Matters

The Kafka wire format embeds a 4-byte schema ID in every message. If schema IDs change during migration, existing Kafka consumers will be unable to deserialize messages already on the topic. The migration process MUST preserve exact schema IDs.

## Prerequisites

| Requirement | Details |
|-------------|---------|
| Source registry | Running Confluent Schema Registry with network access |
| Target registry | Running AxonOps Schema Registry with configured storage |
| Tools | curl and jq on the migration machine |
| Target mode | MUST be set to IMPORT mode before importing |

## Step-by-Step Migration Procedure

### Step 1: Deploy AxonOps Schema Registry
Install and configure with your storage backend. Verify health:

    curl http://localhost:8082/
    # Returns {} if healthy

### Step 2: Set Target to IMPORT Mode
IMPORT mode allows schemas to be registered with specific IDs and bypasses compatibility checking:

    set_mode with mode: IMPORT

### Step 3: Dry-Run Export
Export all schemas from Confluent without importing. The migration script (scripts/migrate-from-confluent.sh) retrieves all subjects and versions, sorts by ID for reference resolution:

    ./scripts/migrate-from-confluent.sh --source http://confluent:8081 --dry-run

Inspect the export: total schemas, subjects, highest ID.

### Step 4: Run the Migration
Execute with verification:

    ./scripts/migrate-from-confluent.sh --source http://confluent:8081 --target http://axonops:8082 --verify

### Step 5: Switch to READWRITE Mode
After successful migration:

    set_mode with mode: READWRITE

The registry will accept new registrations with auto-generated IDs starting after the highest imported ID.

### Step 6: Update Kafka Clients
Change schema.registry.url in serializer/deserializer configs. No code changes needed -- the API is wire-compatible.

## The Import API

**Endpoint:** POST /import/schemas

**Request format:**
    {
      "schemas": [
        {
          "id": 1,
          "subject": "users-value",
          "version": 1,
          "schemaType": "AVRO",
          "schema": "...",
          "references": []
        }
      ]
    }

**Import rules:**
- Schema IDs are preserved exactly.
- Same content with the same ID across subjects is allowed.
- Different content with the same ID is rejected (ErrSchemaIDConflict).
- References MUST be imported before schemas that depend on them (sort by ID).
- Compatibility checking is bypassed in IMPORT mode.
- The ID sequence is adjusted after import to prevent future conflicts.

## Verification

After migration, confirm:
1. **Schema count** -- total schemas in target matches source.
2. **Subject list** -- all subjects present in target.
3. **Schema ID preservation** -- spot-check IDs across both registries.
4. **End-to-end test** -- produce and consume a message through the new registry.

## Rollback

The migration is non-destructive to the source:
1. Point Kafka clients back to Confluent Schema Registry URL.
2. Restart affected producers and consumers.
3. No data loss -- both registries remain fully operational.

## Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| "Cannot import since found existing subjects" | Target already has schemas | Start with empty target or delete existing subjects |
| "schema ID already exists" | ID conflict with different content | Inspect export for duplicate IDs |
| Connection refused | Wrong URL or registry not running | Verify URLs and connectivity |
| Authentication errors | Wrong credentials | Check --source-user/--source-pass or --target-apikey |
| Partial import | Some schemas failed | Check per-schema errors, fix, and re-run |

## MCP Tools

- **set_mode** -- switch between IMPORT and READWRITE
- **import_schemas** -- bulk import with preserved IDs
- **list_subjects** -- verify subjects after import
- **get_schema_by_id** -- spot-check schema IDs
- **health_check** -- verify registry health
`
