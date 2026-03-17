# Migrating from Confluent Schema Registry

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
