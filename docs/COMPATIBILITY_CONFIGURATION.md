# Schema Compatibility Configuration — Design Notes

This document captures design findings and future improvement proposals for how compatibility configuration works in the AxonOps Schema Registry, compared to Confluent Schema Registry.

## How Compatibility Works Today

### When Compatibility Is Checked

Compatibility is checked **only at registration time** — when a new schema version is registered via `POST /subjects/{subject}/versions`. It is never enforced retroactively on existing schemas.

This means changing the compatibility level has no effect on schemas that were already registered. It only gates what can be registered going forward.

### Resolution Order

When a schema is registered, the effective compatibility level is resolved in this order:

1. **Per-subject override** — set via `PUT /config/{subject}`. If present, this wins.
2. **Global API override** — set via `PUT /config`. Applies to all subjects without a per-subject override.
3. **Config file default** — `compatibility.default_level` in `config.yaml`. Falls back to `BACKWARD` if omitted.

### Two-Layer Default System

The system has two distinct layers:

- **Storage layer**: Returns `ErrNotFound` when no global config has been explicitly set via the API. Database backends (PostgreSQL, MySQL, Cassandra) seed a default `BACKWARD` row during migration, so they return a record instead of `ErrNotFound`.
- **Registry layer**: Catches `ErrNotFound` from storage and falls back to `r.defaultConfig` (from the config file). This ensures the config file default takes effect without being written to the database.

This design means:
- Changing `compatibility.default_level` in the config file and restarting the service takes effect immediately for all subjects without an explicit API override.
- If the default were written to the database at startup, a config file change would be silently ignored (the stored value would take precedence).
- `DELETE /config` removes the runtime API override, allowing the config file default to take effect again.

### This Matches Confluent

This behavior matches [Confluent Schema Registry](https://docs.confluent.io/platform/current/schema-registry/fundamentals/schema-evolution.html). The Confluent registry also:
- Defaults to `BACKWARD` compatibility
- Allows runtime overrides via `PUT /config` and `PUT /config/{subject}`
- Only checks compatibility at registration time, not retroactively
- Allows operators to temporarily set `NONE` to register breaking changes

## Risks of the Current Design

### Changing Compatibility Levels Can Lead to Inconsistent History

Because compatibility is only checked at registration time, the following scenario is possible:

1. Subject `orders-value` has 3 versions registered under `BACKWARD` compatibility
2. An operator changes compatibility to `NONE`
3. A completely incompatible schema (version 4) is registered
4. The operator changes compatibility back to `BACKWARD`
5. Version 5 is now checked for backward compatibility against version 4 only — but versions 1-3 are effectively orphaned from a compatibility chain perspective

This is **by design** — it is the same behavior as Confluent. The analogy is adding a linter to an existing codebase: it only catches new violations, not existing ones.

However, this means operators who carelessly change compatibility levels can create a schema history that is not internally consistent.

### No Audit Trail

There is currently no record of when or why compatibility levels were changed. An operator can weaken compatibility, register a breaking schema, and restore the level with no trace.

### No Protection Against Weakening

Any admin can change compatibility from `FULL` to `NONE` at any time. There is no confirmation, lock, or force flag required.

## Proposed Improvements (Not Yet Implemented)

### Option A: Compatibility Change Audit Trail

Add a table/log that records every compatibility level change:

```
| Timestamp           | Subject    | Old Level | New Level | Changed By |
|---------------------|------------|-----------|-----------|------------|
| 2025-01-15 10:30:00 | (global)   | BACKWARD  | NONE      | admin@co   |
| 2025-01-15 10:31:00 | (global)   | NONE      | BACKWARD  | admin@co   |
```

This would be purely informational — it doesn't prevent changes, but it provides accountability.

**Implementation notes:**
- New `config_audit` table in each storage backend
- Write an audit row on every `PUT /config` and `PUT /config/{subject}`
- Expose via a new admin API endpoint (e.g., `GET /admin/config/audit`)

### Option B: Compatibility Lock

Add an optional `locked` field to the config API that prevents weakening the compatibility level:

**API extension (Confluent-compatible):**
```json
PUT /config/orders-value
{
  "compatibility": "BACKWARD",
  "locked": true
}
```

When `locked: true`:
- Compatibility level **cannot be weakened** (e.g., `BACKWARD` → `NONE` is rejected)
- Compatibility level **can be strengthened** (e.g., `BACKWARD` → `FULL` is allowed)
- The lock itself can only be removed with `?force=true` query parameter
- Attempting to weaken returns a clear error message

**Server-side config option:**
```yaml
compatibility:
  default_level: BACKWARD
  lock_on_create: true  # Automatically lock compatibility when first schema is registered
```

When `lock_on_create: true`:
- The first time a schema is registered for a subject, the current effective compatibility level is stored as a locked per-subject config
- This prevents accidental weakening without explicit operator intent

**Confluent compatibility considerations:**
- The `locked` field is an extension — Confluent's API doesn't include it
- Confluent clients will ignore the `locked` field in responses (it's additive)
- Confluent clients sending `PUT /config` without `locked` would not break anything (the field defaults to its current value, or `false` if never set)
- The `?force=true` escape hatch follows the same pattern Confluent uses for permanent deletes

**Weakening vs. strengthening rules:**
```
NONE < BACKWARD < BACKWARD_TRANSITIVE
NONE < FORWARD < FORWARD_TRANSITIVE
NONE < FULL < FULL_TRANSITIVE
```

Lateral changes (e.g., `BACKWARD` → `FORWARD`) would be treated as neither weakening nor strengthening — they would be blocked by a lock since they change the guarantee type.

### Implementation Priority

1. **Option A (audit trail)** — low effort, high value, no API changes needed
2. **Option B (compatibility lock)** — medium effort, prevents the most common operator mistakes, API extension needed

Both options can be implemented independently. Option A provides visibility; Option B provides protection.
