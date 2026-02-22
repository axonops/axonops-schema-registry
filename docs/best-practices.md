# Schema Design and Best Practices

This guide covers how to design schemas for event-driven architectures, choose the right schema format, plan for evolution, and avoid common mistakes. It is practical and opinionated -- where multiple approaches exist, it recommends the one that works best for most teams.

If you are new to schema registries, start with the [Fundamentals](fundamentals.md) guide for core concepts, or the [Getting Started](getting-started.md) guide to run the registry and register your first schema.

## Contents

- [Why Use a Schema Registry](#why-use-a-schema-registry)
  - [The Business Case](#the-business-case)
  - [What Happens Without One](#what-happens-without-one)
  - [When You Do Not Need One](#when-you-do-not-need-one)
- [Choosing a Schema Format](#choosing-a-schema-format)
  - [Decision Matrix](#decision-matrix)
  - [Recommended Default: Avro](#recommended-default-avro)
  - [When to Choose Protobuf](#when-to-choose-protobuf)
  - [When to Choose JSON Schema](#when-to-choose-json-schema)
- [Designing Your First Schema](#designing-your-first-schema)
  - [Field Naming Conventions](#field-naming-conventions)
  - [Namespaces and Packages](#namespaces-and-packages)
  - [Nullable Fields and Defaults](#nullable-fields-and-defaults)
  - [Timestamp Handling](#timestamp-handling)
  - [Enums vs Strings](#enums-vs-strings)
  - [Walkthrough: Building an Order Schema](#walkthrough-building-an-order-schema)
- [Subject Naming Strategies](#subject-naming-strategies)
  - [Start with TopicNameStrategy](#start-with-topicnamestrategy)
  - [Domain-Prefixed Subject Names](#domain-prefixed-subject-names)
  - [When to Use RecordNameStrategy](#when-to-use-recordnamestrategy)
  - [When to Use TopicRecordNameStrategy](#when-to-use-topicrecordnamestrategy)
- [Planning for Schema Evolution](#planning-for-schema-evolution)
  - [Design for Change from Day One](#design-for-change-from-day-one)
  - [The Golden Rules](#the-golden-rules)
  - [How to Rename a Field](#how-to-rename-a-field)
  - [How to Change a Field Type](#how-to-change-a-field-type)
  - [Breaking Changes: When You Have No Choice](#breaking-changes-when-you-have-no-choice)
- [Compatibility Strategy for Your Team](#compatibility-strategy-for-your-team)
  - [Start with BACKWARD](#start-with-backward)
  - [When to Upgrade to FULL](#when-to-upgrade-to-full)
  - [Development vs Production](#development-vs-production)
  - [Per-Subject Overrides](#per-subject-overrides)
  - [Compatibility Checks in CI/CD](#compatibility-checks-in-cicd)
- [Sharing Types Across Services](#sharing-types-across-services)
  - [The Problem: Duplicated Types](#the-problem-duplicated-types)
  - [Schema References](#schema-references)
  - [Versioning Shared Types](#versioning-shared-types)
  - [When Not to Share](#when-not-to-share)
- [Common Patterns](#common-patterns)
  - [Event Envelope](#event-envelope)
  - [Entity Lifecycle Events](#entity-lifecycle-events)
  - [Snapshot vs Delta Events](#snapshot-vs-delta-events)
  - [Fat Events vs Thin Events](#fat-events-vs-thin-events)
- [Common Mistakes](#common-mistakes)
- [Advanced Patterns](#advanced-patterns)
  - [Multi-Tenant Schemas with Contexts](#multi-tenant-schemas-with-contexts)
  - [Data Contracts for Governance](#data-contracts-for-governance)
  - [Schema Evolution in CI/CD Pipelines](#schema-evolution-in-cicd-pipelines)
- [Quick Reference](#quick-reference)
- [Related Documentation](#related-documentation)

---

## Why Use a Schema Registry

### The Business Case

A schema registry is not just a developer convenience -- it is operational insurance. The value comes in three areas:

1. **Fewer production incidents.** When a producer changes the shape of its data, the registry rejects incompatible changes before they reach Kafka. The difference between "the producer changed the schema without telling anyone" and "the registry rejected the incompatible change before it shipped" is the difference between a 3 AM incident and a non-event.

2. **Self-documenting data pipelines.** The registry is the single source of truth for what every Kafka topic looks like. New engineers can browse subjects and versions to understand the data flowing through the system -- no more hunting through producer code, Confluence pages, or Slack threads.

3. **Independent team velocity.** Schema compatibility guarantees let producers and consumers deploy independently. The payments team can add a field to the order schema without coordinating a synchronized release with the analytics, shipping, and billing teams.

### What Happens Without One

Without a schema registry, every schema change is a coordination problem:

- **Silent data loss.** A consumer ignores a field it does not recognize and drops data permanently. Nobody notices until an analytics dashboard is wrong weeks later.
- **Deserialization crashes.** A producer adds a required field. Every consumer instance throws a null pointer exception simultaneously.
- **Implicit contracts.** The only source of truth for a topic's data format is "check the producer code" or "ask the team that owns it." When that team is on holiday, you are stuck.
- **Schema drift.** Multiple teams produce slightly different JSON shapes to the same topic. Each team thinks their format is correct. The downstream data lake has three variants of "order" and no way to reconcile them.
- **The "just use JSON strings" antipattern.** Without enforcement, teams serialize arbitrary JSON. No type safety, no evolution guarantees, no way to know if a field is an integer or a string without reading every producer.

### When You Do Not Need One

Be honest about when a schema registry adds overhead without proportional value:

- **Single-producer, single-consumer systems** that deploy together as a unit. If both sides change simultaneously, compatibility checking adds no value.
- **Logging and debugging topics** that no downstream system depends on. If the data is ephemeral and nobody processes it automatically, schema enforcement is unnecessary.
- **Prototyping and hackathons.** Move fast, iterate freely. If you do use a registry during prototyping, set compatibility to `NONE` so nothing blocks you.

For everything else -- any topic consumed by a different team, any topic feeding a data lake, any topic that outlives the current sprint -- use a schema registry.

---

## Choosing a Schema Format

### Decision Matrix

| Criteria | Avro | Protobuf | JSON Schema |
|----------|------|----------|-------------|
| **Binary size** | Compact (no field names in payload) | Compact (field tags, no names) | Large (full JSON text) |
| **Human readable** | Schema is JSON, data is binary | Schema is `.proto`, data is binary | Both schema and data are JSON |
| **Schema evolution** | Rich (unions, defaults, aliases) | Good (field numbers, optional fields) | Limited (additive changes) |
| **Kafka ecosystem support** | First-class everywhere | Well supported | Supported but less common |
| **Code generation** | Available but optional | Core strength (protoc) | Limited |
| **gRPC compatibility** | No | Yes | No |
| **Learning curve** | Moderate (unions, logical types) | Low (familiar C-like syntax) | Low (JSON vocabulary) |

### Recommended Default: Avro

For most Kafka use cases, **Avro is the recommended default**. Here is why:

- Avro is the native schema format of the Kafka ecosystem. Every client library (Java, Go, Python) has first-class Avro serializer support.
- Field names are not in the payload -- only field indices -- so messages are compact.
- Schema evolution rules are well-understood and battle-tested across thousands of production deployments.
- When you register a schema without specifying `schemaType`, the registry defaults to Avro. It is the path of least resistance.

### When to Choose Protobuf

Choose Protobuf when:

- You already have a large Protobuf investment (existing `.proto` files, generated code, gRPC services).
- Cross-language code generation is a priority and you want `protoc` to generate typed classes in multiple languages from a single definition.
- The same schema is used for both Kafka events and gRPC service calls, and maintaining two schema formats is unacceptable.

### When to Choose JSON Schema

Choose JSON Schema when:

- Your application already uses JSON Schema for REST API request validation, and you want the same validation rules for Kafka messages.
- Human readability of messages in Kafka is important (debugging, manual inspection with `kafkacat`).
- Your team is deeply invested in JSON tooling and the learning curve of Avro or Protobuf is a barrier.

> **Note:** Do not mix schema formats within a single Kafka topic. Each subject has one schema type. Using different formats for the key and value of the same topic is technically possible but creates operational confusion.

---

## Designing Your First Schema

### Field Naming Conventions

Pick one convention and enforce it across your organization:

| Format | Recommended Convention | Example |
|--------|----------------------|---------|
| Avro | `snake_case` | `order_id`, `created_at` |
| Protobuf | `snake_case` (proto convention, generates `camelCase` in some languages) | `order_id`, `created_at` |
| JSON Schema | `camelCase` or `snake_case` (match your existing JSON API) | `orderId` or `order_id` |

The specific convention matters less than consistency. Document it, enforce it in code review, and never deviate.

### Namespaces and Packages

Always add a namespace (Avro) or package (Protobuf) to prevent name collisions when schemas reference each other:

- **Avro:** Use `"namespace": "com.example.billing"` in the record definition.
- **Protobuf:** Use `package com.example.billing;` at the top of the `.proto` file.
- **JSON Schema:** Use a descriptive `"title"` and `"$id"` for identification.

Use a reverse domain format that reflects your organization and domain: `com.yourcompany.domain`.

### Nullable Fields and Defaults

**Every field that might be added in the future MUST have a default value.** This is the single most important rule for schema evolution.

In Avro, the standard pattern for optional fields is:

```json
{"name": "email", "type": ["null", "string"], "default": null}
```

This is both backward-compatible (new consumers can read old data that lacks the field -- they get `null`) and forward-compatible (old consumers can read new data and ignore the field).

In Protobuf (proto3), all fields are implicitly optional and default to zero values. In JSON Schema, omit the field from `required`.

> **The golden rule:** if in doubt, make it nullable with a `null` default. You can always tighten constraints later, but loosening a required field is a compatibility headache.

### Timestamp Handling

Timestamps are one of the most common sources of schema design confusion. Recommendations:

| Format | Recommended Approach |
|--------|---------------------|
| Avro | Use the `timestamp-millis` logical type (milliseconds since epoch, stored as `long`) |
| Protobuf | Use `google.protobuf.Timestamp` or `int64` milliseconds |
| JSON Schema | Use `"format": "date-time"` (ISO 8601 string) |

Avoid epoch seconds (precision loss for sub-second events) and avoid mixing timestamp formats within a schema (some fields as epoch millis, others as ISO strings).

### Enums vs Strings

**Enums** catch typos at parse time and make the set of valid values explicit. **Strings** are easier to evolve.

| Use Case | Recommendation |
|----------|---------------|
| Fixed vocabularies (currency codes, countries, HTTP methods) | Enum |
| Open-ended categories (tags, labels, custom event types) | String |
| Vocabularies that grow frequently (status codes, error types) | String with documentation |

> **Warning (Avro):** Removing an enum symbol is **never** backward-compatible. A consumer using the old schema with the removed symbol cannot read data written with the new schema. Only add symbols -- never remove them. If you need to deprecate a symbol, keep it in the enum and document it as deprecated.

### Walkthrough: Building an Order Schema

Let's build a schema progressively, starting minimal and adding fields. Each version demonstrates a design principle.

**Version 1: The minimum viable schema**

```bash
curl -X POST http://localhost:8081/subjects/billing.orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.billing\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"amount_cents\",\"type\":\"long\"},{\"name\":\"currency\",\"type\":\"string\"}]}"
  }'
```

```json
{"id": 1}
```

Note: `amount_cents` is a `long` representing cents, not a `float` representing dollars. Floating-point arithmetic causes rounding errors in financial data. Use integer cents (or micros) and carry the currency separately.

**Version 2: Add a nullable field with a default**

Add the customer's email. Because it has a default of `null`, this is backward-compatible:

```bash
curl -X POST http://localhost:8081/subjects/billing.orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.billing\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"amount_cents\",\"type\":\"long\"},{\"name\":\"currency\",\"type\":\"string\"},{\"name\":\"customer_email\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
  }'
```

```json
{"id": 2}
```

**Version 3: Add a timestamp with a logical type**

```bash
curl -X POST http://localhost:8081/subjects/billing.orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.billing\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"amount_cents\",\"type\":\"long\"},{\"name\":\"currency\",\"type\":\"string\"},{\"name\":\"customer_email\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"created_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"},\"default\":0}]}"
  }'
```

```json
{"id": 3}
```

The default of `0` (epoch) means consumers using version 2 can still read version 3 data -- they simply see `created_at` as `0`, which they can handle as "unknown."

**Version 4: Add a nested record**

```bash
curl -X POST http://localhost:8081/subjects/billing.orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.billing\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"amount_cents\",\"type\":\"long\"},{\"name\":\"currency\",\"type\":\"string\"},{\"name\":\"customer_email\",\"type\":[\"null\",\"string\"],\"default\":null},{\"name\":\"created_at\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"},\"default\":0},{\"name\":\"shipping_address\",\"type\":[\"null\",{\"type\":\"record\",\"name\":\"Address\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"},{\"name\":\"country\",\"type\":\"string\"},{\"name\":\"postal_code\",\"type\":\"string\"}]}],\"default\":null}]}"
  }'
```

```json
{"id": 4}
```

The `shipping_address` is wrapped in a `["null", Address]` union with a `null` default, making it fully optional. Avoid deeply nesting records beyond 3 levels -- it makes evolution harder because each nested record has its own compatibility scope.

Verify all four versions are registered:

```bash
curl http://localhost:8081/subjects/billing.orders-value/versions
```

```json
[1, 2, 3, 4]
```

---

## Subject Naming Strategies

### Start with TopicNameStrategy

**TopicNameStrategy** is the default in all Confluent serializers and is right for 80% of use cases. Each Kafka topic gets two subjects:

- `{topic}-key` for the message key schema
- `{topic}-value` for the message value schema

This is the simplest approach: one topic, one schema evolution history per key/value.

### Domain-Prefixed Subject Names

As the number of topics grows, flat subject names become unmanageable. Prefix subjects with a domain:

```
billing.orders-value
billing.payments-value
shipping.shipments-value
users.profiles-value
inventory.stock-levels-value
```

This groups related subjects together, makes ownership clear, and prevents naming collisions between teams. The domain prefix is purely a naming convention -- the registry treats it as part of the subject name.

### When to Use RecordNameStrategy

Use **RecordNameStrategy** when the same record type appears in multiple topics and you want a single schema evolution history for that type:

```
Topic: billing.orders     → Subject: com.example.billing.Order
Topic: billing.returns    → Subject: com.example.billing.Order
```

Both topics use the same `Order` schema, and a change to the schema is validated against the single subject `com.example.billing.Order`.

This works well for **shared domain entities** that appear in multiple data flows.

### When to Use TopicRecordNameStrategy

Use **TopicRecordNameStrategy** when multiple event types flow through a single topic:

```
Topic: billing.events
  → Subject: billing.events-com.example.billing.OrderCreated
  → Subject: billing.events-com.example.billing.OrderShipped
  → Subject: billing.events-com.example.billing.RefundIssued
```

Each event type gets its own subject with independent evolution. This is the right choice for event sourcing patterns where a single topic carries all events for an aggregate.

> **Note:** The subject naming strategy is configured on the producer's serializer, not on the registry. The registry accepts any subject name. See [Subjects, Topics, and Naming Strategies](fundamentals.md#subjects-topics-and-naming-strategies) in the fundamentals guide for details.

---

## Planning for Schema Evolution

### Design for Change from Day One

The first version of a schema sets the compatibility baseline. Every future change is measured against it. This means:

- Make every field that is not strictly required nullable with a default.
- Use a namespace/package from the start (hard to add retroactively).
- Prefer `string` over `enum` for fields whose values may grow.
- Keep the schema flat where possible (nested records are harder to evolve).

The cost of being too permissive in version 1 is minimal. The cost of being too strict is a breaking change later.

### The Golden Rules

These rules apply to all schema formats under `BACKWARD` or `FULL` compatibility:

1. **Always add new fields with a default value.** Without a default, consumers using the old schema cannot read new data.
2. **Never remove a field that lacks a default in the reader's schema.** If consumers depend on a field being present and it has no default, removing it causes deserialization failures.
3. **Never change a field's type.** Changing `string` to `int` is incompatible in all formats. Add a new field with the new type instead.
4. **Never rename a field directly.** Field identity in Avro is by name. Renaming is equivalent to removing the old field and adding a new one -- which may be incompatible. Use the three-phase rename pattern below.
5. **Never reuse a field name for a different purpose.** If you remove `status` (a string) and later add `status` (an integer), consumers with cached schemas will fail.

### How to Rename a Field

Renaming a field safely requires three phases:

**Phase 1: Add the new field alongside the old one**

```bash
# Version with both "name" and "full_name"
curl -X POST http://localhost:8081/subjects/users.profiles-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"UserProfile\",\"namespace\":\"com.example.users\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"full_name\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
  }'
```

This is backward-compatible: consumers using the old schema see `name` as before and ignore `full_name`.

**Phase 2: Migrate producers and consumers**

Update producers to write both `name` and `full_name` with the same value. Update consumers to read `full_name` with a fallback to `name`. Deploy all consumers before proceeding.

**Phase 3: Remove the old field (optional)**

Once all consumers read from `full_name`, you can stop writing `name`. If all consumers use a schema version that has `full_name`, you MAY remove `name` in a future version -- but only if consumers have a default for it or do not reference it.

In practice, many teams keep the old field forever (it costs almost nothing in storage) rather than risk the removal.

### How to Change a Field Type

Changing a field's type (e.g., `amount` from `float` to a structured `MoneyAmount` record) follows the same three-phase pattern:

1. **Add** a new field (`amount_v2` of the new type) with a default.
2. **Migrate** producers to write both fields, consumers to read the new field.
3. **Deprecate** the old field (stop writing, eventually remove).

Example: migrating from a floating-point amount to an integer-cents representation:

```bash
# Add amount_cents alongside the old float amount
curl -X POST http://localhost:8081/subjects/billing.orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.billing\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"amount\",\"type\":\"float\"},{\"name\":\"amount_cents\",\"type\":[\"null\",\"long\"],\"default\":null}]}"
  }'
```

Producers write both fields. Consumers read `amount_cents` when present, falling back to `amount`.

### Breaking Changes: When You Have No Choice

Sometimes a schema change is fundamentally incompatible. When this happens, you have three options (from cleanest to riskiest):

1. **Create a new topic.** Start fresh with `billing.orders-v2`, migrate consumers, then decommission the old topic. This is the cleanest approach and leaves a clear audit trail.

2. **Use compatibility groups.** Set a metadata property on the new schema that separates it from the old compatibility group. This allows the new schema to coexist without being checked against the old versions. See the [Compatibility Groups](compatibility.md#compatibility-groups) documentation.

3. **Temporarily disable compatibility.** Set the subject to `NONE`, register the breaking schema, then set it back to `BACKWARD`. This is dangerous -- the window between setting `NONE` and restoring compatibility allows any schema to be registered. Only use this with explicit team agreement and never in an automated pipeline.

---

## Compatibility Strategy for Your Team

### Start with BACKWARD

`BACKWARD` is the default compatibility level, and it is the right choice for most teams. It guarantees that consumers using the **new** schema can read data written with the **previous** schema.

This matches the typical Kafka deployment pattern: consumers are updated first (they need to handle the new schema), then producers start writing the new format.

### When to Upgrade to FULL

Use `FULL` compatibility when:

- You cannot control the deployment order of producers and consumers.
- Multiple independent teams consume the same topic and coordinate poorly.
- The topic is critical infrastructure and you want maximum safety.

`FULL` ensures both backward **and** forward compatibility with the previous version, meaning both old and new consumers can read both old and new data.

For long-lived data (data lake ingestion, compliance archival, event replay), consider `FULL_TRANSITIVE` -- it checks compatibility against **all** previous versions, not just the latest.

### Development vs Production

A practical workflow:

1. **During development:** Set compatibility to `NONE`. Iterate freely on schema design.

```bash
curl -X PUT http://localhost:8081/config/billing.orders-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "NONE"}'
```

2. **Before production:** Lock compatibility to `BACKWARD` (or `FULL`).

```bash
curl -X PUT http://localhost:8081/config/billing.orders-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "BACKWARD"}'
```

3. **In production:** Never change back to `NONE`. Treat compatibility as a production safety control.

### Per-Subject Overrides

Different topics serve different purposes. Use per-subject overrides to match the risk profile:

| Topic Category | Compatibility Level | Rationale |
|---------------|--------------------|----|
| Core business events (orders, payments) | `FULL_TRANSITIVE` | Maximum safety, long-lived data |
| Team-internal events | `BACKWARD` | Standard evolution |
| Internal logging / debugging | `NONE` | No consumers depend on the format |
| Data lake ingestion | `FULL_TRANSITIVE` | Historical data must remain readable |

```bash
# Core business events: maximum safety
curl -X PUT http://localhost:8081/config/billing.orders-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "FULL_TRANSITIVE"}'

# Internal logging: no restrictions
curl -X PUT http://localhost:8081/config/internal.debug-logs-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "NONE"}'
```

### Compatibility Checks in CI/CD

Use the compatibility check endpoint in your CI pipeline to catch incompatible changes before they reach the registry:

```bash
#!/usr/bin/env bash
# check-schema-compat.sh -- Run in CI before deploying
REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
SUBJECT="billing.orders-value"
SCHEMA_FILE="schemas/order.avsc"

SCHEMA=$(jq -Rs '.' < "$SCHEMA_FILE")

RESULT=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$REGISTRY_URL/compatibility/subjects/$SUBJECT/versions/latest" \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}")

if [ "$RESULT" = "200" ]; then
  echo "Schema is compatible"
  exit 0
elif [ "$RESULT" = "404" ]; then
  echo "Subject does not exist yet -- first registration, skipping check"
  exit 0
else
  echo "Schema is NOT compatible (HTTP $RESULT)"
  curl -s -X POST "$REGISTRY_URL/compatibility/subjects/$SUBJECT/versions/latest?verbose=true" \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d "{\"schema\": $SCHEMA}"
  exit 1
fi
```

This script checks compatibility and prints detailed error messages when the check fails. Add it to your CI pipeline so that incompatible schema changes fail the build.

For the full compatibility rules per schema type, see the [Compatibility](compatibility.md) documentation.

---

## Sharing Types Across Services

### The Problem: Duplicated Types

When `Address`, `Money`, or `Timestamp` records are copy-pasted across schemas, they evolve independently. The billing team adds a `postal_code` field to their copy of `Address`. The shipping team adds `zip_code` to theirs. Now you have two incompatible address formats.

### Schema References

Schema references solve this by extracting common types into their own subjects and referencing them:

**Step 1: Register the shared Address type**

```bash
curl -X POST http://localhost:8081/subjects/common.address-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Address\",\"namespace\":\"com.example.common\",\"fields\":[{\"name\":\"street\",\"type\":\"string\"},{\"name\":\"city\",\"type\":\"string\"},{\"name\":\"country\",\"type\":\"string\"},{\"name\":\"postal_code\",\"type\":\"string\"}]}"
  }'
```

**Step 2: Register the shared Money type**

```bash
curl -X POST http://localhost:8081/subjects/common.money-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Money\",\"namespace\":\"com.example.common\",\"fields\":[{\"name\":\"amount_cents\",\"type\":\"long\"},{\"name\":\"currency_code\",\"type\":\"string\"}]}"
  }'
```

**Step 3: Reference both from an Order schema**

```bash
curl -X POST http://localhost:8081/subjects/billing.orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.billing\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"total\",\"type\":\"com.example.common.Money\"},{\"name\":\"shipping_address\",\"type\":[\"null\",\"com.example.common.Address\"],\"default\":null}]}",
    "references": [
      {"name": "com.example.common.Money", "subject": "common.money-value", "version": 1},
      {"name": "com.example.common.Address", "subject": "common.address-value", "version": 1}
    ]
  }'
```

Now `Order`, `Invoice`, and `Shipment` schemas can all reference the same `Address` and `Money` types. When `Address` evolves, it evolves once in `common.address-value`.

For Protobuf, the reference `name` matches the import path (`common.proto`). For JSON Schema, the `name` matches the `$ref` URI (`address.json`). See [Schema References](schema-types.md#schema-references) for format-specific details.

### Versioning Shared Types

References require an explicit version number. This is intentional -- it ensures reproducibility:

```json
{"name": "com.example.common.Address", "subject": "common.address-value", "version": 1}
```

When `common.address-value` gets a version 2, existing schemas that reference version 1 continue to work unchanged. To use version 2, register a new version of the dependent schema with the updated reference.

This means shared type updates do not cascade automatically. You choose when to adopt a new version of a shared type.

### When Not to Share

Do not extract a type into a shared subject if it is used by only one schema. Premature abstraction makes schemas harder to understand and evolve. Wait until at least two schemas need the same type before extracting it.

---

## Common Patterns

### Event Envelope

An **event envelope** wraps domain events with standard metadata. Every event in the system carries tracing, audit, and routing information in a consistent format:

```json
{
  "type": "record",
  "name": "EventEnvelope",
  "namespace": "com.example.events",
  "fields": [
    {"name": "event_id", "type": "string"},
    {"name": "event_type", "type": "string"},
    {"name": "source", "type": "string"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "correlation_id", "type": ["null", "string"], "default": null},
    {"name": "payload", "type": "bytes"}
  ]
}
```

The `payload` field carries the domain event as serialized bytes (which could be another Avro-encoded schema). The envelope itself evolves independently of the domain events.

**When to use:** Event sourcing, audit trails, systems where all events must carry standard tracing metadata.

**When not to use:** Simple point-to-point communication where the metadata overhead is not justified. If your events already carry `event_id` and `timestamp` as domain fields, a separate envelope is redundant.

### Entity Lifecycle Events

A common pattern for CRUD-style systems: separate schemas for each lifecycle event, flowing through a single topic using `TopicRecordNameStrategy`:

```
Topic: users.lifecycle
  → Subject: users.lifecycle-com.example.users.UserCreated
  → Subject: users.lifecycle-com.example.users.UserUpdated
  → Subject: users.lifecycle-com.example.users.UserDeleted
```

Each event type evolves independently. `UserCreated` might carry all fields, while `UserDeleted` only carries `user_id`. This gives you fine-grained schema evolution per event type.

Example `UserCreated` schema:

```json
{
  "type": "record",
  "name": "UserCreated",
  "namespace": "com.example.users",
  "fields": [
    {"name": "user_id", "type": "string"},
    {"name": "email", "type": "string"},
    {"name": "full_name", "type": "string"},
    {"name": "created_at", "type": {"type": "long", "logicalType": "timestamp-millis"}}
  ]
}
```

Example `UserDeleted` schema:

```json
{
  "type": "record",
  "name": "UserDeleted",
  "namespace": "com.example.users",
  "fields": [
    {"name": "user_id", "type": "string"},
    {"name": "deleted_at", "type": {"type": "long", "logicalType": "timestamp-millis"}}
  ]
}
```

### Snapshot vs Delta Events

**Snapshot events** contain the full entity state after a change. **Delta events** contain only the fields that changed.

| Approach | Pros | Cons |
|----------|------|------|
| **Snapshot** | Simple for consumers (full state in every event), works with compacted topics, easy to debug | Larger payloads, includes unchanged fields |
| **Delta** | Smaller payloads, explicit about what changed | Consumers must maintain state, harder to debug, complex for new consumers that need the full picture |

**Recommendation:** Use snapshot events for most use cases. They are simpler, work naturally with Kafka log compaction (the latest message for a key is the full current state), and make it easy for new consumers to bootstrap.

Use delta events only when payload size is a genuine concern (high-volume, low-bandwidth scenarios) or when consumers specifically need to know which fields changed (audit logging, conflict resolution).

### Fat Events vs Thin Events

**Fat events** are self-contained and include all data a consumer might need:

```json
{
  "order_id": "ord-123",
  "customer_email": "alice@example.com",
  "customer_name": "Alice Smith",
  "items": [{"sku": "SKU-1", "name": "Widget", "price_cents": 999, "quantity": 2}],
  "total_cents": 1998,
  "currency": "USD"
}
```

**Thin events** carry only identifiers and expect consumers to look up details:

```json
{
  "order_id": "ord-123",
  "customer_id": "cust-456",
  "item_ids": ["item-789"],
  "total_cents": 1998
}
```

| Approach | Pros | Cons |
|----------|------|------|
| **Fat** | Consumer independence (no API calls needed), resilient to service outages, better for analytics | Larger payloads, data may be stale if the source changes after the event |
| **Thin** | Smaller payloads, always-fresh data via API lookup | Consumer depends on source service availability, higher latency, tight coupling |

**Recommendation:** Default to fat events. The decoupling and resilience benefits outweigh the payload size cost for most systems. The consumer should not need to call another service's API to process an event -- that defeats the purpose of event-driven architecture.

---

## Common Mistakes

**1. Not setting default values.** Adding a field without a default under `BACKWARD` compatibility is rejected by the registry. Always set defaults for new fields.

**2. Using required fields carelessly in Avro.** Every Avro field without a default is implicitly required. If you might ever remove the field, give it a default from the start.

**3. Removing enum symbols.** This is never backward-compatible in Avro. If a consumer has the old schema with the removed symbol, it cannot read data that was written with that symbol. Only add symbols.

**4. Changing field types.** `string` to `int`, `float` to `long`, `record` to `array` -- none of these are compatible. Add a new field with the new type and deprecate the old one.

**5. Inconsistent subject naming.** `orders_value`, `Orders-Value`, and `orders-value` are three different subjects. Standardize your naming convention early and document it.

**6. Registering schemas manually in production.** Use CI/CD pipelines to register schemas. Manual `curl` commands in production lead to drift, accidents, and schemas that nobody can reproduce from source control.

**7. Using NONE compatibility in production.** `NONE` means any schema is accepted, including destructive changes. It is fine during development but SHOULD never be used for production topics that have downstream consumers.

**8. Ignoring the compatibility check API.** The `POST /compatibility/subjects/{subject}/versions/latest` endpoint exists so you can test before you register. Use it in CI to catch incompatible changes before they reach the registry. A failed build is better than a failed deployment.

---

## Advanced Patterns

### Multi-Tenant Schemas with Contexts

When multiple teams or environments share a single registry, use **contexts** to isolate schema namespaces. Each context has its own schema IDs, subjects, compatibility config, and modes:

```bash
# Team A registers their schema in their own context
curl -X POST http://localhost:8081/contexts/.team-a/subjects/orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"int\"}]}"}'

# Team B registers a completely different schema under the same subject name
curl -X POST http://localhost:8081/contexts/.team-b/subjects/orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"}]}"}'
```

No conflicts -- each context is fully independent. See the [Contexts](contexts.md) guide for full documentation.

### Data Contracts for Governance

Use **metadata** to document schema ownership, classify sensitive fields, and enforce governance policies:

```bash
curl -X POST http://localhost:8081/subjects/billing.orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example.billing\",\"fields\":[{\"name\":\"order_id\",\"type\":\"string\"},{\"name\":\"customer_email\",\"type\":\"string\"}]}",
    "metadata": {
      "properties": {
        "owner": "billing-team",
        "domain": "billing",
        "classification": "internal"
      },
      "tags": {
        "Order.customer_email": ["PII", "GDPR"]
      },
      "sensitive": ["customer_email"]
    }
  }'
```

Set config-level defaults so every schema in a subject inherits baseline governance:

```bash
curl -X PUT http://localhost:8081/config/billing.orders-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "compatibility": "FULL_TRANSITIVE",
    "defaultMetadata": {
      "properties": {"domain": "billing", "owner": "billing-team"}
    }
  }'
```

See the [Data Contracts](data-contracts.md) guide for metadata, rulesets, config merge, and CAS semantics.

### Schema Evolution in CI/CD Pipelines

A production-grade workflow:

1. Schema definitions (`.avsc`, `.proto`, or `.json`) live in source control alongside the application code.
2. When a developer changes a schema, CI runs the compatibility check against the registry.
3. If compatible, the schema is registered as part of the deployment pipeline.
4. If incompatible, the build fails with a detailed error message.

```bash
#!/usr/bin/env bash
# register-schema.sh -- Called during deployment after compatibility check passes
REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
SUBJECT="$1"
SCHEMA_FILE="$2"

SCHEMA=$(jq -Rs '.' < "$SCHEMA_FILE")

RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST "$REGISTRY_URL/subjects/$SUBJECT/versions" \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "{\"schema\": $SCHEMA}")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -1)

if [ "$HTTP_CODE" = "200" ]; then
  echo "Registered schema for $SUBJECT: $BODY"
else
  echo "Failed to register schema (HTTP $HTTP_CODE): $BODY"
  exit 1
fi
```

Keep schemas in source control, check compatibility in CI, register during deployment. Never register schemas by hand in production.

---

## Quick Reference

A cheat sheet for common decisions:

| Situation | Recommendation |
|-----------|---------------|
| Adding a field | Give it a default value (`null` for optional, sensible zero for required) |
| Removing a field | Ensure all consumers have a default for it, or keep the field |
| Changing a field type | Add a new field with the new type, deprecate the old one |
| Renaming a field | Three-phase: add new, write both, remove old |
| Choosing compatibility | Start with `BACKWARD`, upgrade to `FULL` if needed |
| Naming subjects | Use `{domain}.{entity}-value` (e.g., `billing.orders-value`) |
| Sharing types | Register as separate subjects and use references |
| CI/CD | Check compatibility before registering, register during deployment |
| Production | Never use `NONE` compatibility |
| Schema format | Default to Avro unless you have a specific reason for Protobuf or JSON Schema |
| Timestamps | Avro: `timestamp-millis` logical type. Protobuf: `Timestamp`. JSON Schema: `date-time` format |
| Enums | Use for fixed vocabularies, use strings for open-ended categories |
| Event design | Default to fat snapshot events for decoupling and resilience |

---

## Related Documentation

- [Fundamentals](fundamentals.md) -- core concepts, wire format, and architecture overview
- [Getting Started](getting-started.md) -- run the registry and register your first schemas
- [Schema Types](schema-types.md) -- Avro, Protobuf, and JSON Schema reference
- [Compatibility](compatibility.md) -- all 7 compatibility modes with per-format rules
- [Contexts](contexts.md) -- multi-tenancy via contexts
- [Data Contracts](data-contracts.md) -- metadata, rulesets, and governance policies
- [Migration](migration.md) -- migrating from Confluent Schema Registry
