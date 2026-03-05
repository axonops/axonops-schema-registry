Practical recipes for common schema evolution scenarios.

## Recipe 1: Add an Optional Field (BACKWARD safe)

**Scenario:** Add a new "email" field to a User schema.

1. Use **get_config** to confirm compatibility is BACKWARD or FULL.
2. Add the field with a default value:
   - Avro: {"name": "email", "type": ["null", "string"], "default": null}
   - Protobuf: optional string email = 3;
   - JSON Schema: add to properties (NOT to required)
3. Use **check_compatibility** to validate.
4. Use **register_schema** to register.

## Recipe 2: Add a Required Field (needs care)

**Scenario:** Add a mandatory "created_at" timestamp.

Under BACKWARD compatibility, you CANNOT add a required field without a default.
1. Add the field with a sensible default (e.g., epoch zero, empty string).
2. Application logic treats the default as "not set."
3. Or: change compatibility to NONE temporarily (use **set_config**), register, restore.

## Recipe 3: Rename a Field (three-phase)

**Scenario:** Rename "customer_name" to "client_name".

Phase 1: Add "client_name" alongside "customer_name" (both populated).
Phase 2: Update all consumers to read "client_name". Deprecate "customer_name".
Phase 3: Remove "customer_name" (requires NONE compatibility for this step).

Use **diff_schemas** to verify each phase.

## Recipe 4: Change a Field Type

**Scenario:** Change "amount" from int to long (Avro) or int32 to int64 (Protobuf).

**Avro type promotions (safe under BACKWARD):**
- int -> long, float, double
- long -> float, double
- float -> double
- string <-> bytes

**Protobuf wire-compatible changes (safe):**
- int32 <-> uint32 <-> int64 <-> uint64 <-> bool (same wire type)
- fixed32 <-> sfixed32
- fixed64 <-> sfixed64

**Incompatible type changes:** Use the three-phase pattern (add new field, migrate, remove old).

## Recipe 5: Remove a Field

**Scenario:** Remove the deprecated "legacy_id" field.

Under BACKWARD: removing a field IS safe (old data's field is ignored).
Under FORWARD: removing a field IS NOT safe (old readers expect it).
Under FULL: removing a field IS NOT safe.

If removal is blocked, use **set_config** to temporarily set BACKWARD or NONE.
In Protobuf, use "reserved" to prevent field number reuse.

## Recipe 6: Break Compatibility Intentionally

**Scenario:** Major redesign of the schema.

Option A (recommended): Create a new subject (e.g., orders-v2-value).
1. Use **register_schema** under the new subject.
2. Migrate producers to the new subject.
3. Use **set_mode** READONLY on the old subject.

Option B: Bypass in existing subject.
1. Use **set_config** with NONE.
2. Use **register_schema** with the breaking change.
3. Use **set_config** to restore the original level.
WARNING: existing consumers may break.

## Recipe 7: Add a Schema Reference

**Scenario:** Extract Address into a shared subject.

1. Use **register_schema** to register Address under "com.example.Address".
2. Update the main schema to reference it.
3. Use **register_schema** with references array.
4. Set FULL_TRANSITIVE compatibility on the shared type.

## General Workflow

For any evolution:
1. **get_latest_schema** -- understand current state
2. **get_config** -- know the compatibility level
3. **check_compatibility** -- validate before registering
4. **explain_compatibility_failure** -- if it fails, get details
5. **register_schema** -- apply the change
6. **diff_schemas** -- verify the change

For domain knowledge, read: schema://glossary/compatibility and schema://glossary/design-patterns
