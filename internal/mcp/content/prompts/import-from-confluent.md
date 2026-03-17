Step-by-step guide for migrating schemas from Confluent Schema Registry.

## Why This Matters
The Kafka wire format embeds a 4-byte schema ID in every message. If IDs change, existing consumers cannot deserialize messages. This procedure preserves exact schema IDs.

## Prerequisites
- Source: running Confluent Schema Registry with network access
- Target: running AxonOps Schema Registry with configured storage
- Tools: curl and jq on the migration machine

## Procedure

### Step 1: Verify target health
Use **health_check** to confirm the target registry is running.

### Step 2: Set IMPORT mode
Use **set_mode** with mode: IMPORT. This allows registering schemas with specific IDs and bypasses compatibility checks.

### Step 3: Export from Confluent
Run the migration script:
    ./scripts/migrate-from-confluent.sh --source http://confluent:8081 --dry-run
Inspect the output to verify schema count and subjects.

### Step 4: Import
    ./scripts/migrate-from-confluent.sh --source http://confluent:8081 --target http://axonops:8082 --verify
Or use the **import_schemas** tool for programmatic import.

### Step 5: Switch to READWRITE
Use **set_mode** with mode: READWRITE. New registrations will get auto-generated IDs starting after the highest imported ID.

### Step 6: Update clients
Change schema.registry.url in all Kafka serializer/deserializer configs. No code changes needed -- the API is wire-compatible.

## Verification
1. Use **list_subjects** -- count should match source.
2. Use **get_schema_by_id** -- spot-check IDs across both registries.
3. Produce/consume a test message through the new registry.

## Rollback
The migration is non-destructive. Point clients back to Confluent if issues arise.

For domain knowledge, read: schema://glossary/migration
