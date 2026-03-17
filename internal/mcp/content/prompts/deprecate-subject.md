Workflow for safely deprecating and removing a schema subject.

## Step 1: Assess Current Usage

```
get_latest_schema(subject: "{subject}")
count_versions(subject: "{subject}")
get_referenced_by(subject: "{subject}")
```

Check if other schemas reference this subject. If so, those must be migrated first.

## Step 2: Check Dependents

```
get_dependency_graph(subject: "{subject}")
```

If the graph shows downstream dependents, migrate them to an alternative schema before proceeding.

## Step 3: Notify Consumers

Before making any changes, ensure all consumers and producers using this subject are aware of the deprecation.

## Step 4: Lock the Subject (READONLY)

Prevent new versions from being registered:
```
set_mode(subject: "{subject}", mode: "READONLY")
```

Verify writes are blocked:
```
check_write_mode(subject: "{subject}")
```

## Step 5: Add Deprecation Metadata

Mark the subject as deprecated using metadata:
```
set_config_full(subject: "{subject}", override_metadata: {"properties": {"deprecated": "true", "deprecation_date": "2026-03-06", "replacement": "<new-subject>"}})
```

## Step 6: Migration Period

Allow time for all consumers to migrate. Monitor usage via audit logs.

## Step 7: Soft-Delete

Hide the subject from listings but keep data accessible by ID:
```
delete_subject(subject: "{subject}")
```

The subject is now hidden from `list_subjects` but schemas are still resolvable by ID (important for in-flight Kafka messages).

## Step 8: Permanent Delete (Optional)

After confirming all consumers have migrated:
```
delete_subject(subject: "{subject}", permanent: true)
```

> Warning: This is irreversible. Kafka messages referencing these schema IDs will become undeserializable.

## Step 9: Clean Up DEKs (if encrypted)

If the subject had client-side encryption:
```
list_deks(kek_name: "<kek>", subject: "{subject}")
delete_dek(kek_name: "<kek>", subject: "{subject}")
```

## Context Support

All tools accept the optional `context` parameter. Ensure you pass it consistently if the subject is in a non-default context.

Available tools: get_latest_schema, count_versions, get_referenced_by, get_dependency_graph, set_mode, check_write_mode, set_config_full, delete_subject, list_deks, delete_dek
