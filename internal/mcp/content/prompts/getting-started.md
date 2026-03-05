Welcome to the Schema Registry MCP server. Here's a quick-start guide.

## Core operations

- **list_subjects** — see all registered subjects
- **get_latest_schema** — fetch the current schema for a subject
- **register_schema** — register a new schema version
- **check_compatibility** — test a schema before registering

## Discovery

- **search_schemas** — search schema content by keyword or regex
- **match_subjects** — find subjects by name pattern (regex, glob, or fuzzy)
- **get_registry_statistics** — overview of subjects, versions, types, KEKs, and exporters

## Schema intelligence

- **score_schema_quality** — analyze naming, docs, type safety, and evolution readiness
- **diff_schemas** — compare two schema versions structurally
- **find_similar_schemas** — find schemas with overlapping field sets
- **suggest_schema_evolution** — generate a compatible schema change
- **explain_compatibility_failure** — human-readable explanations for compat errors

## Configuration

- **get_config / set_config** — manage compatibility levels (BACKWARD, FORWARD, FULL, NONE)
- **get_mode / set_mode** — manage modes (READWRITE, READONLY, IMPORT)

## Encryption (CSFLE)

- **create_kek / create_dek** — set up client-side field encryption
- **list_keks / list_deks** — inspect encryption keys

## Resources (read-only data)

Resources are available via URI patterns like `schema://subjects`, `schema://subjects/{name}`, etc.

## Getting help

Use the other prompts for detailed guidance: design-schema, evolve-schema, check-compatibility, troubleshooting, setup-encryption, and more.
