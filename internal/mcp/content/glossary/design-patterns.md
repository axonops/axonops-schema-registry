# Schema Design Patterns

## Event Envelope Pattern

Wrap every event in a standard envelope that carries metadata separate from the domain payload:

    {
      "event_id": "uuid",
      "event_type": "OrderCreated",
      "timestamp": "2026-01-15T10:30:00Z",
      "source": "orders-service",
      "correlation_id": "uuid",
      "payload": { ... domain-specific fields ... }
    }

**Benefits:** Consistent routing, filtering, and tracing across all events. The envelope schema evolves independently from payload schemas.

**Implementation:** Register the envelope as a shared schema via references. The payload type can be a union (Avro), oneof (Protobuf), or oneOf (JSON Schema).

**Avro example:**
```json
{
  "type": "record",
  "name": "EventEnvelope",
  "namespace": "com.company.events",
  "fields": [
    {"name": "event_id", "type": {"type": "string", "logicalType": "uuid"}},
    {"name": "event_type", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "source", "type": "string"},
    {"name": "correlation_id", "type": ["null", {"type": "string", "logicalType": "uuid"}], "default": null},
    {"name": "payload", "type": "bytes", "doc": "Serialized domain event"}
  ]
}
```

## Entity Lifecycle Events

Model entity state changes as a sequence of typed events:

    UserCreated -> UserUpdated -> UserEmailVerified -> UserDeactivated

Each event type gets its own schema but shares common fields (entity_id, timestamp, actor). Use **TopicRecordNameStrategy** so multiple event types can share a single Kafka topic while having independent schema evolution.

**Avro example -- two related lifecycle schemas:**
```json
{
  "type": "record", "name": "UserCreated", "namespace": "com.company.users",
  "fields": [
    {"name": "user_id", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "email", "type": "string"},
    {"name": "display_name", "type": "string"}
  ]
}
```
```json
{
  "type": "record", "name": "UserDeactivated", "namespace": "com.company.users",
  "fields": [
    {"name": "user_id", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "reason", "type": ["null", "string"], "default": null}
  ]
}
```

## Snapshot vs Delta Events

| Pattern | Content | Use When |
|---------|---------|----------|
| **Snapshot** | Complete entity state | Consumers need the full picture; late-joining consumers; compacted topics |
| **Delta** | Only changed fields | High-frequency updates; bandwidth-sensitive; consumers track state locally |

**Snapshot example:** Every UserUpdated event contains all user fields.
**Delta example:** UserUpdated only contains the fields that changed plus the entity_id.

**Recommendation:** Start with snapshots. They are simpler and more resilient. Switch to deltas only when bandwidth or storage costs justify the added complexity.

## Fat vs Thin Events

| Pattern | Content | Use When |
|---------|---------|----------|
| **Fat events** | All data consumers might need | Consumers should be self-sufficient; avoid downstream lookups |
| **Thin events** | Minimal data + identifier for lookups | Events are high-volume; most consumers need only a subset |

**Fat event:** OrderCreated includes full customer info, product details, shipping address.
**Thin event:** OrderCreated includes only order_id, customer_id, total.

**Recommendation:** Prefer fat events for event sourcing and CQRS. Use thin events when the same data would be duplicated across many event types.

## Shared Types via References

Extract reusable types (Address, Money, ContactInfo) into their own subjects:

1. Register "Address" schema under subject "com.example.Address".
2. Reference it from "OrderCreated" using a schema reference.
3. "Address" evolves independently with its own compatibility policy.

**Avro example -- shared Address type:**
```json
{"type": "record", "name": "Address", "namespace": "com.company.common",
 "fields": [
   {"name": "street", "type": "string"},
   {"name": "city", "type": "string"},
   {"name": "postal_code", "type": "string"},
   {"name": "country", "type": "string"}
 ]}
```
**Order referencing Address:**
```json
{"type": "record", "name": "Order", "namespace": "com.company.orders",
 "fields": [
   {"name": "order_id", "type": "string"},
   {"name": "shipping", "type": "com.company.common.Address"}
 ]}
```
Register with: `register_schema` using `references: [{"name": "com.company.common.Address", "subject": "com.company.Address", "version": 1}]`

**Versioning shared types:**
- Use FULL or FULL_TRANSITIVE compatibility for shared types (both producers and consumers must handle changes).
- Test referenced schemas across all dependents before publishing a new version.
- Use **get_referenced_by** to find all schemas that depend on a shared type.

## Three-Phase Rename Pattern

Renaming a field safely requires three schema versions:

**Phase 1: Add new field alongside old field.**
Both old_name and new_name exist. Producers write both. Consumers read from whichever exists.

**Phase 2: Deprecate old field.**
All consumers have been updated to read new_name. Mark old_name as deprecated (via doc or metadata).

**Phase 3: Remove old field.**
After all consumers have migrated, remove old_name. This step requires NONE compatibility or a new subject.

**Avro example -- three versions side by side:**
```
v1: {"type":"record","name":"User","fields":[{"name":"full_name","type":"string"}]}
v2: {"type":"record","name":"User","fields":[{"name":"full_name","type":"string"},{"name":"display_name","type":["null","string"],"default":null}]}
v3: {"type":"record","name":"User","fields":[{"name":"display_name","type":"string"}]}
```

## Three-Phase Type Change Pattern

Changing a field type (e.g., string to int) follows the same three-phase approach:

**Phase 1:** Add new field with the new type alongside the old field.
**Phase 2:** Migrate all consumers to use the new field.
**Phase 3:** Remove the old field.

## CI/CD Integration

Integrate schema validation into your deployment pipeline:

1. **Pre-commit:** Run validate_schema to check syntax.
2. **PR check:** Run check_compatibility against the registry to verify the change is compatible.
3. **Deploy:** Register the schema as part of the deployment process.
4. **Post-deploy:** Verify the schema was registered successfully.

Use the MCP tools or the REST API in CI scripts:
- **validate_schema** -- syntax check without registration
- **check_compatibility** -- compatibility check without registration
- **register_schema** -- register on successful deployment

## Dead Letter Queue (DLQ) Pattern

When a consumer encounters a message it cannot deserialize:

1. Route the message to a DLQ topic.
2. Include the original topic, partition, offset, schema ID, and error message.
3. Register a DLQ schema that wraps the original message bytes.

The DLQ schema should use NONE compatibility (any message format may end up there).

## MCP Tools for Pattern Implementation

- **register_schema** -- register schemas with references
- **check_compatibility** -- validate before registering
- **get_referenced_by** -- find dependents of shared types
- **find_similar_schemas** -- find schemas with overlapping fields
- **get_dependency_graph** -- visualize the reference tree
- **diff_schemas** -- compare versions during evolution
