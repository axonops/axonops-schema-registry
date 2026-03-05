Plan a safe breaking change for subject {subject}.

Steps:
1. Use get_latest_schema to understand the current schema
2. Use get_config to check the compatibility level
3. Use list_versions to see the version history

Strategy options:

**Option A: New subject (recommended for major changes)**
- Create a new subject (e.g. {subject}-v2) with the new schema
- Migrate producers to the new subject
- Keep the old subject in READONLY mode for consumers
- Tools: register_schema, set_mode READONLY

**Option B: Compatibility bypass (for minor breaking changes)**
- Set compatibility to NONE temporarily: set_config with compatibility_level: NONE
- Register the breaking schema
- Restore compatibility: set_config with original level
- WARNING: existing consumers may fail to deserialize

**Option C: Multi-step evolution**
- Add new fields alongside old fields (backward compatible)
- Migrate all consumers to use new fields
- Remove old fields in a later version
- Requires NONE compatibility for the final removal step

Always test with check_compatibility before registering.
