Guide for managing multi-tenant contexts in the schema registry.

## What are contexts?
Contexts are tenant namespaces that isolate schemas, subjects, and configuration. The default context is "." (dot). Contexts enable multi-tenancy — different teams, environments, or applications can have independent schema registries within the same server.

## Listing and navigating contexts
- **list_contexts** — list all available contexts
- **list_subjects** — lists subjects in the default context
- Subjects can be qualified with context: `:.staging:my-subject`

## The 4-tier config/mode inheritance chain
Configuration and mode settings cascade through 4 levels:

1. **Server default** — hardcoded BACKWARD compatibility, READWRITE mode
2. **Global (__GLOBAL)** — set via set_config/set_mode with no subject
3. **Context global** — per-context default (overrides __GLOBAL)
4. **Per-subject** — most specific (overrides everything above)

To check effective config: **get_config** with a subject name returns the resolved value.
To check effective mode: **get_mode** with a subject name returns the resolved value.

## Managing configuration per context
- **set_config** — set compatibility level (per-subject or global)
- **delete_config** — remove per-subject config (falls back to context global)
- **set_mode** — set mode (READWRITE, READONLY, READONLY_OVERRIDE, IMPORT)
- **delete_mode** — remove per-subject mode (falls back to context global)

## Import and migration
- Use **set_mode** with mode IMPORT to enable ID-preserving schema import
- Use **import_schemas** to bulk import schemas with preserved IDs
- Reset mode after import: **set_mode** with mode READWRITE

## Resources
- `schema://contexts` — list all contexts
- `schema://contexts/{context}/subjects` — subjects in a specific context

Available tools: list_contexts, get_config, set_config, delete_config, get_mode, set_mode, delete_mode, import_schemas
