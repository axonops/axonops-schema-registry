# Migrating from Confluent Schema Registry

## Overview

AxonOps Schema Registry includes a migration script and an import API for migrating schemas from Confluent Schema Registry. The migration preserves schema IDs, which is critical because the Kafka wire format embeds a 4-byte schema ID in every message. If schema IDs change, existing Kafka consumers will be unable to deserialize messages already on the topic.

The migration is non-destructive to the source registry. No data is modified or deleted on the Confluent side.

## Prerequisites

Before starting the migration, ensure the following are in place:

| Requirement | Details |
|-------------|---------|
| Source registry | A running Confluent Schema Registry with network access from the machine running the migration script. |
| Target registry | A running AxonOps Schema Registry instance with a configured storage backend. |
| Tools | `curl` and `jq` must be installed on the machine running the migration script. |
| Target mode | The target registry must be set to `IMPORT` mode before importing schemas. IMPORT mode allows schemas to be registered with specific IDs and bypasses compatibility checking. |

## Migration Script

The migration script is located at `scripts/migrate-from-confluent.sh`. It exports all subjects and schema versions from a Confluent Schema Registry, then imports them into AxonOps Schema Registry in a single bulk request.

### Usage

```bash
./scripts/migrate-from-confluent.sh \
  --source http://confluent-sr:8081 \
  --target http://axonops-sr:8082 \
  --verify
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--source URL` | Confluent Schema Registry URL | `http://localhost:8081` |
| `--target URL` | AxonOps Schema Registry URL | `http://localhost:8082` |
| `--source-user USER` | Basic auth username for the source registry | (none) |
| `--source-pass PASS` | Basic auth password for the source registry | (none) |
| `--target-user USER` | Basic auth username for the target registry | (none) |
| `--target-pass PASS` | Basic auth password for the target registry | (none) |
| `--target-apikey KEY` | API key for the target registry (sent as Bearer token) | (none) |
| `--dry-run` | Export schemas to a file without importing them | `false` |
| `--verify` | After import, compare every schema between source and target | `false` |
| `--output FILE` | File path for the exported schema data | `schemas-export.json` |
| `--help` | Print usage information and exit | -- |

### What the Script Does

1. Checks that `curl` and `jq` are installed.
2. Tests connectivity to the source registry, and to the target registry (unless `--dry-run` is set).
3. Retrieves all subjects from the source, then fetches every version of every subject, including schema content, schema type, schema ID, and references.
4. Sorts all exported schemas by ID so that referenced schemas are imported before the schemas that depend on them.
5. Writes the export to a JSON file (default: `schemas-export.json`).
6. Sends all schemas to the target registry in a single `POST /import/schemas` request.
7. If `--verify` is set, compares every subject, version, ID, and schema content between source and target, reporting any mismatches.
8. Prints a summary with the number of schemas exported, subjects, and the highest schema ID.

## Import API

The import endpoint accepts a batch of schemas with explicit IDs and versions.

**Endpoint:** `POST /import/schemas`

The target registry must be in `IMPORT` mode. Set the mode before importing:

```bash
curl -X PUT http://localhost:8082/mode \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"mode": "IMPORT"}'
```

### Request Format

```json
{
  "schemas": [
    {
      "id": 1,
      "subject": "users-value",
      "version": 1,
      "schemaType": "AVRO",
      "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"
    },
    {
      "id": 2,
      "subject": "users-value",
      "version": 2,
      "schemaType": "AVRO",
      "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\",\"default\":\"\"}]}",
      "references": []
    }
  ]
}
```

Each schema object has the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | integer | Yes | The global schema ID to assign. Must match the original Confluent ID. |
| `subject` | string | Yes | The subject name (e.g., `users-value`). |
| `version` | integer | Yes | The version number within the subject. |
| `schemaType` | string | No | `AVRO`, `JSON`, or `PROTOBUF`. Defaults to `AVRO` if omitted. |
| `schema` | string | Yes | The schema content as a JSON-encoded string. |
| `references` | array | No | Schema references for cross-subject dependencies. Each reference has `name`, `subject`, and `version` fields. |

### Response Format

```json
{
  "imported": 2,
  "errors": 0,
  "results": [
    {"id": 1, "subject": "users-value", "version": 1, "success": true},
    {"id": 2, "subject": "users-value", "version": 2, "success": true}
  ]
}
```

Each result object contains:

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | The schema ID that was imported. |
| `subject` | string | The subject name. |
| `version` | integer | The version number. |
| `success` | boolean | Whether this schema was imported successfully. |
| `error` | string | Present only when `success` is `false`. Describes the failure reason. |

### Import Rules

- **Schema IDs are preserved exactly.** The ID specified in the request is the ID stored in the target registry.
- **Same content with the same ID across different subjects is allowed.** This is normal when multiple subjects reference the same underlying schema.
- **Different content with the same ID is rejected.** The import returns an error for that schema (`ErrSchemaIDConflict`).
- **References are resolved during import.** Referenced schemas must be imported before the schemas that depend on them. The migration script handles this by sorting schemas by ID.
- **Compatibility checking is bypassed.** IMPORT mode disables compatibility checks, allowing the exact historical schema sequence to be reproduced.
- **The ID sequence is adjusted after import.** The registry updates its internal ID counter to start after the highest imported ID, preventing conflicts with future registrations.

## Step-by-Step Migration

### 1. Deploy AxonOps Schema Registry

Install and configure AxonOps Schema Registry with your chosen storage backend. See [Installation](installation.md) and [Storage Backends](storage-backends.md) for setup instructions.

Verify the target registry is running:

```bash
curl http://localhost:8082/
```

An empty JSON object `{}` confirms the registry is healthy.

### 2. Set the Target to IMPORT Mode

IMPORT mode must be enabled before importing schemas with specific IDs:

```bash
curl -X PUT http://localhost:8082/mode \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"mode": "IMPORT"}'
```

### 3. Run a Dry-Run Export

Export all schemas from Confluent without importing anything. This allows you to inspect the export file and confirm the schema count before committing to the migration.

```bash
./scripts/migrate-from-confluent.sh \
  --source http://confluent-sr:8081 \
  --target http://axonops-sr:8082 \
  --dry-run \
  --output schemas-export.json
```

Inspect the export:

```bash
# Total number of schemas
jq '.schemas | length' schemas-export.json

# List of subjects
jq '[.schemas[].subject] | unique' schemas-export.json

# Highest schema ID
jq '[.schemas[].id] | max' schemas-export.json
```

### 4. Run the Migration

Execute the migration with verification enabled:

```bash
./scripts/migrate-from-confluent.sh \
  --source http://confluent-sr:8081 \
  --target http://axonops-sr:8082 \
  --verify
```

If the source or target requires authentication:

```bash
./scripts/migrate-from-confluent.sh \
  --source http://confluent-sr:8081 \
  --source-user admin \
  --source-pass secret \
  --target http://axonops-sr:8082 \
  --target-apikey sr_live_abc123 \
  --verify
```

The script prints a summary at the end showing the number of schemas exported, the number of subjects, and the highest schema ID.

### 5. Switch to READWRITE Mode

After a successful migration, switch the target registry back to normal operating mode:

```bash
curl -X PUT http://localhost:8082/mode \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"mode": "READWRITE"}'
```

The registry will now accept new schema registrations with auto-generated IDs starting after the highest imported ID.

### 6. Update Kafka Clients

Point your Kafka serializer and deserializer configurations to the new registry URL. The only change required is the `schema.registry.url` property. No code changes are needed because the API is wire-compatible.

**Java:**

```java
props.put("schema.registry.url", "http://axonops-sr:8082");
```

**Go:**

```go
client, err := schemaregistry.NewClient(schemaregistry.NewConfig("http://axonops-sr:8082"))
```

**Python:**

```python
schema_registry_client = SchemaRegistryClient({"url": "http://axonops-sr:8082"})
```

## Verification

After migration, confirm the following:

**Schema count.** The total number of schemas in the target should match the source.

```bash
# Source
curl -s http://confluent-sr:8081/subjects | jq 'length'

# Target
curl -s http://axonops-sr:8082/subjects | jq 'length'
```

**Subject list.** All subjects should be present in the target.

```bash
diff <(curl -s http://confluent-sr:8081/subjects | jq -r '.[]' | sort) \
     <(curl -s http://axonops-sr:8082/subjects | jq -r '.[]' | sort)
```

**Schema ID preservation.** Spot-check a few schemas to confirm IDs match.

```bash
# Check schema ID 1 on both registries
curl -s http://confluent-sr:8081/schemas/ids/1 | jq .
curl -s http://axonops-sr:8082/schemas/ids/1 | jq .
```

**Kafka producer and consumer.** Run a test message through a Kafka topic using the new registry URL to confirm serialization and deserialization work end-to-end.

## Rollback

The migration does not modify or delete any data on the source Confluent Schema Registry. If problems are discovered after switching:

1. Update Kafka client configurations to point back to the Confluent Schema Registry URL.
2. Restart the affected producers and consumers.

No data loss occurs because both registries remain fully operational. The AxonOps instance can be left running (for example, to debug issues) or shut down.

## Troubleshooting

**"Cannot import since found existing subjects"** -- The target registry already contains schemas. The import API rejects imports when subjects already exist unless you are adding new subjects. Start with an empty target registry or delete existing subjects before importing.

**"schema ID already exists"** -- Two schemas in the import have the same ID but different content. This typically indicates data corruption in the source export. Inspect the export file to identify the conflicting entries.

**Connection refused** -- Confirm the source and target URLs are correct and that the registries are running. The script tests connectivity before attempting the export.

**Authentication errors** -- Verify the credentials passed via `--source-user`/`--source-pass` or `--target-user`/`--target-pass`/`--target-apikey`. The source uses HTTP Basic Auth. The target accepts either Basic Auth or a Bearer token (API key).

**Partial import** -- If some schemas fail to import, the response includes per-schema error details. Fix the failing schemas in the export file and re-run the import. Schemas that were already imported successfully will conflict on re-import if the content and ID match (this is safe and reported as a success).

## Related Documentation

- [Getting Started](getting-started.md) -- initial setup and first API calls
- [Storage Backends](storage-backends.md) -- choosing and configuring a persistence backend
- [Configuration](configuration.md) -- full configuration reference
- [Installation](installation.md) -- deployment methods and platform packages
