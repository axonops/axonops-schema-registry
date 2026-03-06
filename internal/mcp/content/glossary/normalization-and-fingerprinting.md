# Normalization and Fingerprinting

## Overview

Schema identity in the registry is determined by **content-addressed fingerprinting**. Two schemas are considered identical if they produce the same fingerprint, regardless of whitespace, field ordering, or comments.

## Fingerprinting Process

1. **Parse** the schema string into a structured representation
2. **Canonicalize** -- convert to a canonical form (format-specific rules below)
3. **Hash** -- compute SHA-256 of the canonical form
4. The resulting hex digest is the **fingerprint**

## Per-Format Canonicalization Rules

### Avro (Parsing Canonical Form -- PCF)

Avro uses the Parsing Canonical Form defined in the Avro specification:

- Remove `doc`, `aliases`, `default`, `order`, and other non-essential attributes
- Sort record fields by name
- Inline named types on first use, reference by name thereafter
- Normalize `null` unions
- Remove whitespace

**Example:** `{"type":"record","name":"User","fields":[{"name":"id","type":"string"},{"name":"name","type":"string"}]}` -- field order is alphabetical, no extra attributes.

### Protobuf

- Strip comments (`//` and `/* */`)
- Sort fields by field number
- Normalize whitespace
- Retain `syntax`, `package`, `import`, `message`, `enum`, `oneof`, `reserved`

### JSON Schema

- Sort all object keys alphabetically (recursive)
- Remove whitespace
- Normalize numeric values

## The `normalize` Flag

When `normalize: true` is passed to **register_schema**, the schema is canonicalized before storage:

- Whitespace and formatting differences are eliminated
- The canonical form is stored and returned
- Useful for ensuring consistent fingerprints across different producers

Without `normalize`, the original schema string is stored, but the fingerprint is still computed from the canonical form.

## How Metadata and RuleSet Affect Identity

Metadata (`metadata`) and data contract rules (`ruleSet`) are NOT included in the fingerprint calculation:

- Same schema content + different metadata = **new version but same schema ID**
- The schema ID is shared (content-addressed), but the version is unique
- This means re-registering a schema with updated metadata creates a new version that reuses the existing schema ID

## Deduplication Scenarios

| Scenario | Result |
|----------|--------|
| Identical schema registered twice to same subject | Returns existing ID and version (no new version created) |
| Identical schema registered to different subjects | Same schema ID, different subject-version pairs |
| Same schema with different whitespace | Same fingerprint, deduplicated |
| Same schema with different metadata | New version, same schema ID |
| Same schema with `normalize: true` vs `false` | Same fingerprint (fingerprint always uses canonical form) |
| Semantically equivalent but syntactically different | Different fingerprint (canonicalization has limits) |

## MCP Tools

- **normalize_schema** -- canonicalize a schema without registering it
- **validate_schema** -- check syntax and see the canonical form
- **lookup_schema** -- find if a schema already exists (uses fingerprint matching)
- **find_similar_schemas** -- find schemas with overlapping structure
- **get_schema_by_id** -- retrieve by global ID (content-addressed)
