# BDD Test Gap Analysis — Confluent API Compatibility

**Date:** 2026-02-12 (updated)
**Branch:** `feature/testing`
**Current Status:** 594 BDD scenarios passing across 36 feature files

## Summary

After cross-referencing: (a) the Confluent Schema Registry API reference (all endpoints, all query parameters, all error codes, all behavioral notes), (b) our 36 existing feature files, (c) all step definitions, (d) our handler/router implementation, and (e) our types/error-code definitions — the following gaps remain. These are organized by priority: **P0** = Confluent compatibility-critical, **P1** = important edge cases, **P2** = nice-to-have exhaustive coverage.

**Implementation Status Legend:**
- **IMPL: YES** — Our codebase implements this feature; just needs BDD tests
- **IMPL: PARTIAL** — Partially implemented; tests may reveal behavioral divergences
- **IMPL: NO** — NOT implemented in our codebase; BDD tests will fail until code is added. These are documented for tracking but tests should be written expecting the correct Confluent behavior.

---

## SECTION 1: Missing Endpoint Behaviors (P0 — Confluent Wire Compatibility)

### GAP-01: `?verbose=true` on compatibility endpoints
**Endpoints:** `POST /compatibility/subjects/{subject}/versions/{version}`, `POST /compatibility/subjects/{subject}/versions`
**IMPL: PARTIAL** — Our handler ALWAYS returns messages (line 413 in handlers.go). It never checks the `verbose` param. So `verbose=true` behavior is the default, but `verbose=false` (or absent) should OMIT the messages field per Confluent spec.
**Confluent behavior:** `{"is_compatible": false, "messages": ["..."]}` when verbose, vs `{"is_compatible": false}` when not verbose.
**Tests needed:**
- Compatibility check with `verbose=true` returns messages array for incompatible schema
- Compatibility check with `verbose=true` returns empty/absent messages for compatible schema
- Compatibility check with `verbose=false` (default) omits messages field
- Verbose with each schema type (Avro, Protobuf, JSON Schema)

### GAP-02: `?deleted=true` on subject version listing
**Endpoint:** `GET /subjects/{subject}/versions?deleted=true`
**IMPL: PARTIAL** — We have some deletion_advanced scenarios that test this, but coverage is incomplete.
**Tests needed:**
- List versions shows only active versions by default
- List versions with `?deleted=true` includes soft-deleted versions
- List versions after soft-deleting one version — default hides it, `?deleted=true` shows it
- List versions after permanent delete — version gone even with `?deleted=true`

### GAP-03: `?deleted=true` on schema lookup (POST /subjects/{subject})
**Endpoint:** `POST /subjects/{subject}?deleted=true`
**IMPL: YES** — We have one scenario for this but it only partially covers the behavior.
**Tests needed:**
- Lookup after soft-delete of version without `?deleted=true` → 404
- Lookup after soft-delete of version with `?deleted=true` → 200 with schema details
- Lookup after permanent delete with `?deleted=true` → 404

### GAP-04: `?defaultToGlobal=true` on GET /config/{subject}
**Endpoint:** `GET /config/{subject}?defaultToGlobal=true`
**IMPL: NO** — Our `GetConfig` handler does NOT parse `defaultToGlobal` at all. It always falls back to global config when no subject config exists. Confluent returns 404 (40401) when no subject-level config and `defaultToGlobal=false` (default).
**Tests needed:**
- Get config for subject with its own config → returns subject config regardless of param
- Get config for subject without config + `?defaultToGlobal=true` → returns global config
- Get config for subject without config + `?defaultToGlobal=false` → returns 404 (40401)
- Get config for non-existent subject → behavior with and without param

### GAP-05: `?permanent=true` delete requires prior soft-delete (Confluent two-step)
**Endpoint:** `DELETE /subjects/{subject}?permanent=true`, `DELETE /subjects/{subject}/versions/{version}?permanent=true`
**IMPL: NO** — Error codes 40404, 40405, 40406 are DEFINED in `types.go` but NEVER USED in `handlers.go`. Our handlers do not enforce the two-step delete requirement. We allow permanent delete without prior soft-delete.
**Confluent behavior:**
- Hard delete (permanent=true) REQUIRES prior soft-delete. Without it → 40405 (SubjectNotSoftDeleted)
- For version hard-delete, an explicit version number is required — `"latest"` only does soft delete
- Attempting hard delete on already hard-deleted resource → 404
**Tests needed:**
- Permanent delete of non-soft-deleted subject → 40405 (Confluent) vs 200 (our behavior)
- Permanent delete of non-soft-deleted version → 40405 vs our behavior
- Soft-delete then permanent delete subject → 200
- Soft-delete then permanent delete version → 200
- Hard delete version using "latest" → should only soft-delete (Confluent restriction)
- Permanent delete of already-permanently-deleted subject → 404

### GAP-06: READONLY mode blocks write operations
**Endpoint:** ALL write endpoints when mode is READONLY
**IMPL: NO** — Our handlers have ZERO mode enforcement. No code in handlers.go checks the current mode before allowing writes, deletes, or config changes. READONLY mode is accepted via PUT /mode but never enforced.
**Confluent behavior:** When mode is READONLY (global or per-subject), ALL write operations return 42205 (OperationNotPermitted):
- Schema registration (POST /subjects/{subject}/versions)
- Schema/subject deletion (DELETE endpoints)
- Config changes (PUT /config, PUT /config/{subject})
- Mode changes (PUT /mode — except READONLY_OVERRIDE allows mode changes)
**Tests needed:**
- Set global mode READONLY → register schema → 422 with error 42205
- Set per-subject mode READONLY → register schema in that subject → 422
- Set per-subject mode READONLY → register in different subject → 200 (unaffected)
- Set global READONLY, per-subject READWRITE → register in that subject → 200 (override)
- READONLY mode still allows reads (GET operations)
- READONLY mode blocks DELETE operations
- READONLY mode blocks config changes (PUT /config)

### GAP-07: IMPORT mode enables ID preservation on registration
**Endpoint:** `POST /subjects/{subject}/versions` when mode is IMPORT
**IMPL: NO** — Our standard registration endpoint ignores `id` and `version` fields in the request body regardless of mode. We have a separate `/import/schemas` endpoint but Confluent uses the standard registration endpoint with IMPORT mode.
**Confluent behavior:** In IMPORT mode, the registration endpoint accepts `id` and `version` fields in the request body to preserve specific IDs.
**Tests needed:**
- Set mode to IMPORT → register with specific `id` field → ID preserved
- Normal mode → register with `id` field → field ignored
- IMPORT mode → register → skip compatibility checks
- IMPORT mode → register with conflicting `id` (already assigned to different schema) → error

### GAP-08: DELETE /subjects/{subject} response body
**Endpoint:** `DELETE /subjects/{subject}`
**IMPL: YES** — Response body is returned but we never verify it in BDD tests.
**Confluent response:** `[1, 2, 3]` (array of deleted version integers)
**Tests needed:**
- Delete subject with 1 version → response is `[1]`
- Delete subject with 3 versions → response is `[1, 2, 3]`
- Permanent delete subject → response is array of versions

### GAP-09: DELETE /subjects/{subject}/versions/{version} response body
**Endpoint:** `DELETE /subjects/{subject}/versions/{version}`
**IMPL: YES** — Response body is returned but we never verify it in BDD tests.
**Confluent response:** `1` (the version number as integer)
**Tests needed:**
- Delete version 1 → response body is `1`
- Delete version 3 → response body is `3`
- Permanent delete version → response body is the version number

### GAP-10: Version identifier "latest" and -1
**Endpoints:** `GET /subjects/{subject}/versions/latest`, `DELETE /subjects/{subject}/versions/latest`, `POST /compatibility/subjects/{subject}/versions/latest`
**IMPL: PARTIAL** — We support "latest" on GET but haven't tested DELETE or compatibility with "latest". `-1` support unknown.
**Tests needed:**
- GET version "latest" returns newest version (already tested)
- DELETE version "latest" soft-deletes the latest version
- Compatibility check against version "latest" works
- GET version `-1` also returns latest version
- DELETE version `-1` also targets latest version
- DELETE version "latest" then GET latest → returns previous version

### GAP-11: `?subjectPrefix` on GET /subjects
**Endpoint:** `GET /subjects?subjectPrefix=prefix`
**IMPL: YES** — The handler parses `subjectPrefix`.
**Confluent behavior:** Filters subjects by name prefix. Empty string returns all subjects.
**Tests needed:**
- List subjects with `?subjectPrefix=test-` → only subjects starting with "test-"
- List subjects with `?subjectPrefix=` (empty) → all subjects
- List subjects with `?subjectPrefix=nonexistent` → empty array
- Combine `?subjectPrefix=x` with `?deleted=true`

### GAP-40: `?defaultToGlobal=true` on GET /mode/{subject}
**Endpoint:** `GET /mode/{subject}?defaultToGlobal=true`
**IMPL: NO** — Our `GetMode` handler does NOT parse `defaultToGlobal`. It always falls back to global mode when no subject mode exists. Confluent returns 404 (40401) when no subject-level mode and `defaultToGlobal=false` (default).
**Tests needed:**
- Get mode for subject with its own mode → returns subject mode regardless of param
- Get mode for subject without mode + `?defaultToGlobal=true` → returns global mode
- Get mode for subject without mode + `?defaultToGlobal=false` → returns 404 (40401)

### GAP-41: READONLY_OVERRIDE mode
**IMPL: NO** — Our valid modes are only READWRITE, READONLY, IMPORT (line 653-655 in registry.go). READONLY_OVERRIDE is not in the list.
**Confluent behavior:** READONLY_OVERRIDE is a 4th mode. Same as READONLY (blocks writes) but allows the mode endpoint itself to be changed. Without READONLY_OVERRIDE, changing mode from READONLY is blocked.
**Tests needed:**
- Set mode to READONLY_OVERRIDE → accepted
- READONLY_OVERRIDE blocks writes like READONLY
- READONLY_OVERRIDE allows PUT /mode (unlike READONLY which blocks it)
- GET /mode returns "READONLY_OVERRIDE" when set
- Invalid mode value → 42204 error

### GAP-42: `?format` parameter on schema-by-ID endpoints
**Endpoints:** `GET /schemas/ids/{id}?format=resolved`, `GET /schemas/ids/{id}/schema?format=resolved`
**IMPL: NO** — Our handlers do not parse the `format` parameter.
**Confluent behavior:**
- For AVRO: `format=resolved` inlines all named types (no references)
- For PROTOBUF: `format=ignore_extensions` or `format=serialized`
- For JSON: not applicable
**Tests needed (if supported):**
- GET schema by ID with `format=resolved` for Avro with references → inlined schema
- GET schema by ID with default format → schema with references
- Unknown format value → behavior documented

### GAP-43: `?fetchMaxId` parameter on GET /schemas/ids/{id}
**Endpoint:** `GET /schemas/ids/{id}?fetchMaxId=true`
**IMPL: NO** — Not parsed in handlers.
**Confluent behavior:** When `fetchMaxId=true`, response includes an additional `maxId` field with the highest schema ID in the registry.
**Tests needed (if supported):**
- GET schema with `fetchMaxId=true` → response has `maxId` field
- `maxId` equals the highest schema ID in the registry
- GET schema without `fetchMaxId` → no `maxId` field

### GAP-44: `?subject` hint on schema-by-ID endpoints
**Endpoints:** `GET /schemas/ids/{id}?subject=hint`, `GET /schemas/ids/{id}/schema?subject=hint`
**IMPL: NO** — Not parsed in handlers.
**Confluent behavior:** Optional subject name hint for lookup across contexts. Helps when same schema ID exists in multiple contexts.
**Tests needed (if supported):**
- GET schema with `?subject=hint` returns schema
- GET schema without `?subject` also returns schema

### GAP-45: `?deleted=true` on GET /subjects/{subject}/versions/{version}
**Endpoint:** `GET /subjects/{subject}/versions/{version}?deleted=true`
**IMPL: PARTIAL** — Need to verify parameter is parsed for single-version retrieval.
**Confluent behavior:** Without `deleted=true`, accessing a soft-deleted version returns 40402. With `deleted=true`, the soft-deleted version is returned.
**Tests needed:**
- Get soft-deleted version without `?deleted=true` → 404 (40402)
- Get soft-deleted version with `?deleted=true` → 200 with schema details
- Get permanently deleted version with `?deleted=true` → 404

### GAP-46: `?deleted=true` on GET /schemas/ids/{id}/versions
**Endpoint:** `GET /schemas/ids/{id}/versions?deleted=true`
**IMPL: PARTIAL** — Need to verify parameter is parsed.
**Confluent behavior:** Without `deleted=true`, only active subject-version pairs returned. With `deleted=true`, soft-deleted pairs also included.
**Tests needed:**
- Schema in 2 subjects, one soft-deleted → without param shows 1 pair, with `?deleted=true` shows 2

### GAP-47: DELETE /mode/{subject} endpoint
**Endpoint:** `DELETE /mode/{subject}`
**IMPL: YES** — Handler exists at line 757 in handlers.go, route registered.
**What's missing:** Zero BDD tests for this endpoint.
**Tests needed:**
- Set subject mode → DELETE mode → GET mode falls back to global
- DELETE mode when no subject mode set → 404
- DELETE mode response returns the previously set mode

### GAP-48: Hard delete version requires explicit version number (not "latest")
**Endpoint:** `DELETE /subjects/{subject}/versions/latest?permanent=true`
**IMPL: NO** — Our handlers don't distinguish "latest" from explicit versions for permanent delete.
**Confluent behavior:** For hard-delete, `"latest"` only performs soft delete. An explicit numeric version is required for permanent delete.
**Tests needed:**
- DELETE version "latest" with `?permanent=true` → only soft-deletes (Confluent behavior)
- DELETE version 1 with `?permanent=true` (after soft-delete) → permanent delete

---

## SECTION 2: Error Code Coverage Gaps (P0)

### GAP-12: Error code 40404 — SubjectSoftDeleted
**IMPL: NO** — Error code defined in `types.go` (line 131) but NEVER used in handlers.
**What's missing:** In Confluent, attempting to register under a soft-deleted subject without first un-deleting returns 40404.
**Tests needed:**
- Register schema under soft-deleted subject → 40404 (Confluent) vs our behavior (currently allows it)

### GAP-13: Error code 40405 — SubjectNotSoftDeleted
**IMPL: NO** — Error code defined in `types.go` (line 132) but NEVER used in handlers.
**What's missing:** Trying to permanent-delete a subject that hasn't been soft-deleted first.
**Tests needed:**
- Permanent delete without prior soft-delete → 40405 (Confluent) vs our behavior

### GAP-14: Error code 40406 — SchemaVersionSoftDeleted
**IMPL: NO** — Error code defined in `types.go` (line 133) but NEVER used in handlers.
**What's missing:** Accessing a soft-deleted version without `?deleted=true`.
**Tests needed:**
- Get soft-deleted version without `?deleted=true` → should return 40406 or 40402

### GAP-15: Error code 42205 — OperationNotPermitted
**IMPL: NO** — Mode enforcement not implemented in handlers.
**What's missing:** Zero tests. This is returned when mode blocks the operation.
**Tests needed:**
- Register when READONLY → 42205
- Delete when READONLY → 42205
- Config change when READONLY → 42205

### GAP-16: Error code 50001 — InternalServerError (negative test)
**What's missing:** No test deliberately triggers a 500 error.
**Note:** This is hard to test at the BDD level without infrastructure manipulation. Lower priority.

### GAP-49: Error code 42204 — Invalid mode
**IMPL: YES** — Error code defined and used for invalid mode values.
**What's missing:** No BDD test verifies this error code.
**Tests needed:**
- PUT /mode with invalid mode value (e.g., "INVALID") → 42204
- PUT /mode/{subject} with invalid mode → 42204

### GAP-50: Error code 42202 — Invalid schema type / version
**IMPL: PARTIAL** — We define 42202 as `ErrorCodeInvalidSchemaType` but Confluent uses it for "Invalid version". Potential semantic divergence.
**Tests needed:**
- Register with invalid schemaType → check error code
- GET version with invalid version string → check error code
- Verify our error code matches Confluent for each scenario

### GAP-51: Error code 409 (40901) — Incompatible schema
**IMPL: YES** — We use error code 409 for incompatible schemas.
**What's missing:** While compatibility tests exist, none explicitly verify the 409 error code is returned for incompatible registrations (most tests check `is_compatible` field on the check endpoint, not the registration endpoint rejection).
**Tests needed:**
- Register incompatible schema → 409 with error code 409
- Verify error message mentions incompatibility

---

## SECTION 3: Response Shape & Field Validation Gaps (P1)

### GAP-17: Schema registration response fields
**Endpoint:** `POST /subjects/{subject}/versions`
**IMPL: YES**
**Confluent response:** `{"id": 1}` — just the ID field.
**Tests needed:**
- Register returns response with `id` field
- `id` is a positive integer

### GAP-18: Schema-by-ID response fields
**Endpoint:** `GET /schemas/ids/{id}`
**IMPL: PARTIAL** — Our types use `json:"schemaType,omitempty"` so AVRO type IS omitted (matching Confluent).
**Confluent response:** `{"schema": "...", "schemaType": "PROTOBUF"}` — schemaType OMITTED for AVRO.
**Tests needed:**
- Avro schema → `schemaType` field absent or omitted
- Protobuf schema → `schemaType` is "PROTOBUF"
- JSON Schema → `schemaType` is "JSON"
- Response has `schema` field (string)
- Response has `references` (omitted when empty, array when present)

### GAP-19: Subject-version response fields
**Endpoint:** `GET /subjects/{subject}/versions/{version}`
**Confluent response:** `{"subject": "s", "id": 1, "version": 1, "schemaType": "AVRO", "schema": "..."}`
**Tests needed:**
- Verify all 5 fields present: subject, id, version, schemaType (omitted for AVRO), schema
- Verify references field present when applicable
- Verify schemaType is correct for each type

### GAP-20: Lookup response fields
**Endpoint:** `POST /subjects/{subject}`
**Confluent response:** `{"subject": "s", "id": 1, "version": 1, "schemaType": "AVRO", "schema": "..."}`
**Tests needed:**
- Lookup returns all fields: subject, id, version, schema
- schemaType omitted for AVRO, present for PROTOBUF/JSON

### GAP-21: Config response field name
**Endpoints:** `GET /config`, `PUT /config`
**IMPL: YES**
**Confluent behavior:** GET returns `compatibilityLevel`, PUT returns `compatibility` (different field names!).
**Tests needed:**
- GET /config response has `compatibilityLevel` field (not `compatibility`)
- PUT /config response has `compatibility` field (not `compatibilityLevel`)
- GET /config/{subject} response has `compatibilityLevel`
- PUT /config/{subject} response has `compatibility`

### GAP-22: Mode response field validation
**Endpoints:** `GET /mode`, `PUT /mode`, `DELETE /mode/{subject}`
**Tests needed:**
- GET /mode response has `mode` field
- PUT /mode response has `mode` field
- DELETE /mode/{subject} response has `mode` field
- Mode values include: READWRITE, READONLY, READONLY_OVERRIDE (GAP-41), IMPORT
- Invalid mode returns 42204

### GAP-52: schemaType field omission for AVRO
**IMPL: YES** — Our response types use `json:"schemaType,omitempty"` which should omit AVRO.
**What's missing:** No BDD test explicitly verifies schemaType is ABSENT for AVRO schemas.
**Confluent behavior:** `schemaType` is OMITTED in responses when the type is AVRO (since AVRO is the default). For PROTOBUF and JSON, it is always included.
**Tests needed:**
- Register Avro schema → GET by ID → schemaType absent
- Register Protobuf schema → GET by ID → schemaType is "PROTOBUF"
- Register JSON Schema → GET by ID → schemaType is "JSON"
- Same check on subject-version GET and lookup responses

---

## SECTION 4: Edge Cases & Boundary Conditions (P1)

### GAP-23: Special characters in subject names
**What's missing:** No test uses subjects with dots, dashes, underscores, or URL-encoded characters.
**Tests needed:**
- Subject with dots: `com.example.Topic-value`
- Subject with dashes: `my-topic-value`
- Subject with underscores: `my_topic_value`
- Subject with URL-special characters (if supported)

### GAP-24: Empty and malformed request bodies
**What's missing:** Limited testing of malformed inputs.
**Tests needed:**
- Register with missing `schema` field → error
- Register with null schema → error
- Register with non-JSON body → error (400 or 422)
- PUT /config with missing `compatibility` field → error
- PUT /config with empty body → error
- PUT /mode with missing `mode` field → error
- Compatibility check with empty body → error

### GAP-25: Pagination edge cases for GET /schemas
**What's missing:** Limited pagination boundary testing.
**Tests needed:**
- `offset` greater than total schemas → empty array
- `limit=0` → empty array or all results (document behavior)
- `offset=0&limit=1` → exactly 1 result
- Very large offset → empty array, not error
- Negative offset → error or treated as 0

### GAP-26: Re-registration after soft-delete
**IMPL: YES** — We have some coverage in deletion_advanced.feature but not exhaustive.
**Tests needed:**
- Register v1, v2 → soft-delete subject → re-register same schema → version is 3 (not 1)
- Register v1 → soft-delete → re-register different schema → version continues
- Soft-deleted subject appears in list with `?deleted=true`

### GAP-27: Schema ID stability across subjects
**IMPL: YES** — We have schema_deduplication.feature with good coverage.
**Tests needed (verify existing):**
- Same Avro schema in 2 subjects → same ID ✓
- Same Protobuf schema in 2 subjects → same ID ✓
- Same JSON Schema in 2 subjects → same ID ✓
- Different schemas → different IDs ✓

### GAP-28: Concurrent registration idempotency
**What's missing:** Registering the exact same schema simultaneously should be safe and return the same ID.
**Tests needed:**
- Register same schema twice rapidly → same ID returned both times
- Register same schema under same subject → same version, same ID

### GAP-29: Content-Type header handling
**What's missing:** No tests verify Content-Type behavior.
**Tests needed:**
- Request with `Content-Type: application/json` → accepted
- Request with `Content-Type: application/vnd.schemaregistry.v1+json` → accepted
- Response Content-Type is `application/vnd.schemaregistry.v1+json`

### GAP-30: Compatibility check against specific version number
**Endpoint:** `POST /compatibility/subjects/{subject}/versions/{version}`
**What's missing:** We test against "latest" and all versions but limited testing against specific numbered versions.
**Tests needed:**
- Check compat against version 1 when versions 1,2,3 exist
- Check compat against version 2 specifically
- Check compat against non-existent version → 40402
- Check compat against non-existent subject → 40401

### GAP-31: GET /schemas/ids/{id}/subjects with `?deleted=true`
**What's missing:** Never tested with deleted parameter.
**Tests needed:**
- Schema used by 2 subjects, one soft-deleted → without param shows 1, with `?deleted=true` shows 2

### GAP-32: Schema string format in response
**What's missing:** The schema field in responses should contain a parseable schema string.
**Tests needed:**
- Retrieve Avro schema → response.schema is valid JSON
- Retrieve Protobuf schema → response.schema is valid proto definition
- Retrieve JSON Schema → response.schema is valid JSON

### GAP-53: DELETE /config/{subject} response shape
**Endpoint:** `DELETE /config/{subject}`
**IMPL: YES**
**Confluent behavior:** Returns the compatibility level that was previously set at the subject level.
**Tests needed:**
- Set subject config to FULL → DELETE → response has `compatibilityLevel: "FULL"` (the removed value)
- DELETE config on subject with no config → 404 (40401)

---

## SECTION 5: Missing Endpoint Tests (P1)

### GAP-33: DELETE /config (global config reset)
**IMPL: YES** — Route exists at line 141 in server.go.
**What's missing:** We test deleting subject config but the global config delete has minimal coverage.
**Tests needed:**
- Set global config to FULL → DELETE /config → GET /config returns BACKWARD (default)
- DELETE /config when already default → still returns 200

### GAP-34: GET /schemas/ids/{id}/versions with `?deleted=true`
**What's missing:** Never test with deleted subjects/versions.
**Tests needed:**
- Schema in 2 subjects, one deleted → without `?deleted=true` shows 1, with shows 2
- Verify response format: `[{"subject": "x", "version": 1}, ...]`

### GAP-35: Compatibility check against version 0 / negative / invalid
**What's missing:** No boundary tests for version parameter.
**Tests needed:**
- Version 0 → error
- Version -2 → error (only -1 is valid as alias for "latest")
- Version "abc" → error
- Version extremely large → 40402

### GAP-54: GET /schemas/ids/{id}/schema (raw schema endpoint)
**Endpoint:** `GET /schemas/ids/{id}/schema`
**IMPL: YES**
**What's missing:** Zero BDD tests for the raw schema endpoint (returns just the schema string, not wrapped in JSON metadata).
**Tests needed:**
- GET raw Avro schema → returns valid JSON string
- GET raw Protobuf schema → returns proto definition text
- GET raw JSON Schema → returns valid JSON string
- Non-existent ID → 40403

### GAP-55: GET /subjects/{subject}/versions/{version}/schema (raw schema by version)
**Endpoint:** `GET /subjects/{subject}/versions/{version}/schema`
**IMPL: YES**
**What's missing:** Limited BDD tests for the raw schema-by-version endpoint.
**Tests needed:**
- GET raw schema for specific version → returns just the schema string
- GET raw schema for "latest" → returns latest schema
- GET raw schema for non-existent version → 40402
- GET raw schema for non-existent subject → 40401

### GAP-56: GET /subjects/{subject}/versions/{version}/referencedby
**Endpoint:** `GET /subjects/{subject}/versions/{version}/referencedby`
**IMPL: YES**
**What's missing:** We test references in schema_references_advanced.feature but may not test the `referencedby` endpoint directly with various edge cases.
**Tests needed:**
- Version with no references → empty array `[]`
- Version referenced by 1 schema → `[id]`
- Version referenced by multiple schemas → array of IDs
- "latest" as version → works
- Non-existent subject/version → 40401/40402

---

## SECTION 6: Advanced Confluent Behaviors (P2)

### GAP-36: Schema normalization (`?normalize=true`)
**Endpoints:** `POST /subjects/{subject}/versions?normalize=true`, `POST /subjects/{subject}?normalize=true`, `POST /compatibility/subjects/{subject}/versions/{version}?normalize=true`
**IMPL: NO** — `normalize` parameter not parsed anywhere in handlers.
**Note:** If not supported, tests should document our behavior (param ignored or error).
**Tests needed (if supported):**
- Register with `?normalize=true` normalizes schema before storage
- Lookup with `?normalize=true` normalizes before comparison
- Compat check with `?normalize=true` normalizes before checking

### GAP-37: `?force=true` on PUT /mode
**Endpoint:** `PUT /mode?force=true`, `PUT /mode/{subject}?force=true`
**IMPL: NO** — `force` parameter not parsed in handlers.
**Confluent behavior:** Changing to IMPORT mode fails if schemas already exist unless `force=true`. Required for disaster recovery scenarios.
**Tests needed (if supported):**
- Set mode to IMPORT without force when schemas exist → error
- Set mode to IMPORT with `?force=true` when schemas exist → succeeds

### GAP-38: `?lookupDeletedSchema=true` on GET /subjects
**Endpoint:** `GET /subjects?lookupDeletedSchema=true`
**What's missing:** This Confluent param may not be supported. Document behavior.

### GAP-39: `?deletedOnly=true` on version listing
**Endpoint:** `GET /subjects/{subject}/versions?deletedOnly=true`
**What's missing:** Shows only soft-deleted versions. May not be supported.

---

## SECTION 7: Implementation Status Summary — Code Changes Needed

The following Confluent features are **NOT implemented in our codebase** and would need code changes before the corresponding BDD tests would pass:

| Feature | GAPs | Effort | Notes |
|---------|------|--------|-------|
| `?defaultToGlobal` on config endpoints | GAP-04 | Small | Parse query param, conditionally return 404 |
| `?defaultToGlobal` on mode endpoints | GAP-40 | Small | Same pattern as config |
| Two-step delete enforcement (40405) | GAP-05, GAP-13 | Medium | Add soft-delete state check before permanent delete |
| Error codes 40404, 40406 in handlers | GAP-12, GAP-14 | Medium | Use already-defined error codes |
| READONLY mode enforcement | GAP-06, GAP-15 | Medium | Add mode check middleware/guard before all write handlers |
| READONLY_OVERRIDE mode | GAP-41 | Small | Add to valid modes, same as READONLY but allows mode changes |
| IMPORT mode on standard endpoint | GAP-07 | Medium | Parse `id`/`version` from request body when mode=IMPORT |
| `?verbose` behavior (omit messages) | GAP-01 | Small | Check param, conditionally omit `messages` field |
| `?format` parameter | GAP-42 | Medium | Schema resolution/formatting |
| `?fetchMaxId` parameter | GAP-43 | Small | Query max ID, add to response |
| `?subject` hint parameter | GAP-44 | Small | Pass hint to storage lookup |
| `?normalize` parameter | GAP-36 | Large | Schema normalization logic |
| `?force` on mode change | GAP-37 | Small | Check for existing schemas before IMPORT |
| Hard delete "latest" restriction | GAP-48 | Small | Reject permanent=true with "latest" |
| Error code 42204 (invalid mode) | GAP-49 | YES | Already implemented |
| schemaType omission for AVRO | GAP-52 | YES | Already works via omitempty |

---

## IMPLEMENTATION PLAN

### Phase 1: P0 Critical Confluent Compatibility (New Feature Files)

Create these new feature files:

| File | Covers GAPs | Est. Scenarios |
|------|------------|----------------|
| `compatibility_verbose.feature` | GAP-01 | 8 |
| `deletion_lifecycle.feature` | GAP-02, GAP-03, GAP-05, GAP-10, GAP-26, GAP-45, GAP-48 | 25 |
| `mode_enforcement.feature` | GAP-06, GAP-07, GAP-15, GAP-41, GAP-47, GAP-49 | 20 |
| `response_shapes.feature` | GAP-08, GAP-09, GAP-17–GAP-22, GAP-52, GAP-53 | 25 |
| `config_defaults.feature` | GAP-04, GAP-33, GAP-40 | 12 |
| `subject_filtering.feature` | GAP-11, GAP-31, GAP-46 | 10 |

**New step definitions needed:**
1. `I GET "([^"]*)"` — already exists (raw GET step)
2. `I DELETE "([^"]*)"` — may need raw DELETE step
3. `the response should be an integer with value {n}` — for version delete response
4. `the response field "messages" should be an array` — for verbose compat
5. `the response should not have field "messages"` — already exists via `should not have field`
6. `I set the mode for subject "X" to "READONLY"` — already exists
7. `the response field "X" should not exist` — already exists via `should not have field`
8. `I delete the mode for subject "X"` — new step for DELETE /mode/{subject}
9. `the response should not have field "schemaType"` — verify AVRO omission

### Phase 2: P1 Edge Cases & Exhaustive Coverage

| File | Covers GAPs | Est. Scenarios |
|------|------------|----------------|
| `error_codes_exhaustive.feature` | GAP-12–GAP-16, GAP-49–GAP-51 | 15 |
| `edge_cases.feature` | GAP-23–GAP-25, GAP-28–GAP-30, GAP-35 | 20 |
| `content_types.feature` | GAP-29 | 4 |
| `schema_id_stability.feature` | GAP-27, GAP-32 | 10 |
| `raw_schema_endpoints.feature` | GAP-54, GAP-55, GAP-56 | 15 |

### Phase 3: P2 Advanced Confluent Features

| File | Covers GAPs | Est. Scenarios |
|------|------------|----------------|
| `normalization.feature` | GAP-36 | 6 |
| `mode_force.feature` | GAP-37 | 4 |
| `listing_params.feature` | GAP-38, GAP-39, GAP-34 | 6 |
| `schema_format.feature` | GAP-42, GAP-43, GAP-44 | 8 |

### Step Definition Changes Needed

**New steps to add to `schema_steps.go`:**
```
^I DELETE "([^"]*)"$                                    # Raw DELETE with path
^the response should be an integer$                     # For delete version response
^the response should be an integer with value (\d+)$    # Specific integer
^the response should be an array containing (\d+)$      # Array contains int
^the response field "([^"]*)" should be an array$       # Field is array
^the response field "([^"]*)" should be an array of length (\d+)$  # Field array len
^the response header "([^"]*)" should contain "([^"]*)"$  # Header check
^I delete the mode for subject "([^"]*)"$               # DELETE /mode/{subject}
^the response should not have field "([^"]*)"$           # Field absence check
```

**New steps to add to `reference_steps.go`:**
```
^I check compatibility with verbose against subject "([^"]*)" version (\d+):$
^I check compatibility with verbose against all versions of subject "([^"]*)":$
```

### Estimated Total New Scenarios: ~180

### Execution Order

1. Add new step definitions first (all files)
2. Create Phase 1 feature files (P0)
3. Run tests — document failures that need code changes vs test fixes
4. Create Phase 2 feature files (P1)
5. Run tests, fix any failures
6. Create Phase 3 feature files (P2)
7. Run full suite, target 750+ scenarios passing
8. Commit

### Files to Create/Modify

**New files (15):**
- `tests/bdd/features/compatibility_verbose.feature`
- `tests/bdd/features/deletion_lifecycle.feature`
- `tests/bdd/features/mode_enforcement.feature`
- `tests/bdd/features/response_shapes.feature`
- `tests/bdd/features/config_defaults.feature`
- `tests/bdd/features/subject_filtering.feature`
- `tests/bdd/features/error_codes_exhaustive.feature`
- `tests/bdd/features/edge_cases.feature`
- `tests/bdd/features/content_types.feature`
- `tests/bdd/features/schema_id_stability.feature`
- `tests/bdd/features/raw_schema_endpoints.feature`
- `tests/bdd/features/normalization.feature`
- `tests/bdd/features/mode_force.feature`
- `tests/bdd/features/listing_params.feature`
- `tests/bdd/features/schema_format.feature`

**Modified files (2):**
- `tests/bdd/steps/schema_steps.go` — new step definitions
- `tests/bdd/steps/reference_steps.go` — verbose compat steps

### Resume Instructions

If this session crashes, resume with:

> Read `BDD_GAP_ANALYSIS.md` in the project root. It contains the full gap analysis with 56 identified gaps (up from original 39) and a phased implementation plan. Key finding: many Confluent features (mode enforcement, defaultToGlobal, two-step delete, READONLY_OVERRIDE, format, normalize, force params) are NOT IMPLEMENTED in our codebase — the error codes exist in types.go but handlers never use them. The task is to implement all new feature files and step definitions described in the plan, starting with Phase 1 (P0 gaps). Tests for unimplemented features should still be written to document expected Confluent behavior — they will fail until code changes are made.
