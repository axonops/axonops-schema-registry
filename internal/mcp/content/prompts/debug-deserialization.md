Troubleshooting guide for consumer deserialization failures.

## Kafka Wire Format

Kafka messages with schema registry use a 5-byte prefix:

```
[0x00] [4-byte schema ID (big-endian)] [serialized payload]
```

- Byte 0: Magic byte (always 0x00)
- Bytes 1-4: Schema ID as a 32-bit big-endian integer
- Remaining bytes: Avro/Protobuf/JSON-encoded payload

## Diagnostic Workflow

### Step 1: Extract the Schema ID

From the raw message bytes, extract bytes 1-4 as a big-endian integer.
If the first byte is not 0x00, the message was not serialized with the schema registry ("Unknown magic byte" error).

### Step 2: Fetch the Producer's Schema

```
get_schema_by_id(id: <extracted_id>)
```

This returns the schema the producer used to serialize the message.

### Step 3: Fetch the Consumer's Schema

```
get_latest_schema(subject: "<consumer-subject>")
```

This returns the schema the consumer is using to deserialize.

### Step 4: Compare Schemas

```
diff_schemas(subject: "<subject>", version1: <producer_version>, version2: <consumer_version>)
```

### Step 5: Check Compatibility

```
explain_compatibility_failure(subject: "<subject>", schema: <producer_schema>)
```

## Common Causes

### "Unknown magic byte"
- Message was not serialized with the schema registry
- Wrong deserializer configured (using Avro deserializer on plain JSON, etc.)
- Message was produced before schema registry was introduced
- **Fix:** Check producer serializer configuration

### "Schema not found" (ID not in registry)
- Consumer is pointing to a different registry instance than the producer
- Schema was permanently deleted from the registry
- **Fix:** Check `schema.registry.url` configuration on both producer and consumer

### "Could not deserialize" (schema mismatch)
- Producer schema evolved in a way the consumer's local schema cannot handle
- **Fix:** Use diff_schemas to see what changed, then update the consumer

### Wrong subject or naming strategy
- Producer uses TopicNameStrategy but consumer uses RecordNameStrategy
- **Fix:** Ensure both use the same SubjectNameStrategy

### Old consumer with missing fields
- A new required field was added without a default value
- **Fix:** The schema change was backward-incompatible; add a default value

## Tools Reference

- **get_schema_by_id** -- fetch schema by the ID from the message
- **get_latest_schema** -- fetch the consumer's expected schema
- **diff_schemas** -- compare producer and consumer schemas
- **explain_compatibility_failure** -- understand why schemas are incompatible
- **get_subjects_for_schema** -- find which subjects use a schema ID
- **list_versions** -- see all versions of a subject
