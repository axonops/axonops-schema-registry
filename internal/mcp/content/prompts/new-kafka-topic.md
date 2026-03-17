End-to-end workflow for setting up schemas for a new Kafka topic.

## Step 1: Choose a Naming Strategy

Kafka subjects follow naming conventions based on the serializer's **SubjectNameStrategy**:
- **TopicNameStrategy** (default): `{topic}-key`, `{topic}-value`
- **RecordNameStrategy**: `{record.namespace}.{record.name}`
- **TopicRecordNameStrategy**: `{topic}-{record.namespace}.{record.name}`

For topic "{topic_name}", the default subjects are:
- Key: `{topic_name}-key`
- Value: `{topic_name}-value`

## Step 2: Design the Key Schema

The key schema determines message partitioning. Common patterns:
- **String key:** `{"type": "string"}` (simplest)
- **Composite key:** A record with business identifiers
- **Null key:** No key schema needed (round-robin partitioning)

## Step 3: Design the Value Schema

Design the value schema for your event/entity. Choose the format:
- **AVRO** (recommended for Kafka): compact binary, excellent evolution
- **PROTOBUF**: if you also use gRPC
- **JSON**: if consumers need human-readable messages

Use the **design-schema** prompt for format-specific guidance.

## Step 4: Validate Schemas

Before registering, validate both schemas:
```
validate_schema(schema: <key_schema>, schema_type: {format})
validate_schema(schema: <value_schema>, schema_type: {format})
```

## Step 5: Set Compatibility Level

Choose a compatibility level for each subject:
```
set_config(subject: "{topic_name}-key", compatibility_level: "FULL")
set_config(subject: "{topic_name}-value", compatibility_level: "BACKWARD")
```

**Recommendation:** FULL for keys (both producers and consumers handle changes), BACKWARD for values (default).

## Step 6: Check Compatibility (if subjects exist)

If the subject already has schemas registered:
```
check_compatibility(subject: "{topic_name}-value", schema: <new_schema>, schema_type: {format})
```

## Step 7: Register Schemas

```
register_schema(subject: "{topic_name}-key", schema: <key_schema>, schema_type: {format})
register_schema(subject: "{topic_name}-value", schema: <value_schema>, schema_type: {format})
```

## Step 8: Verify Registration

```
get_latest_schema(subject: "{topic_name}-key")
get_latest_schema(subject: "{topic_name}-value")
```

Note the returned schema IDs -- these are embedded in the Kafka message wire format.

## Step 9: Retrieve by ID

Consumers use schema IDs from messages to fetch schemas:
```
get_schema_by_id(id: <schema_id>)
```

## Step 10: Context Support

All tools accept the optional `context` parameter for multi-tenant isolation:
```
register_schema(subject: "{topic_name}-value", schema: <schema>, context: ".staging")
```

Available tools: validate_schema, check_compatibility, register_schema, get_latest_schema, get_schema_by_id, set_config, validate_subject_name
