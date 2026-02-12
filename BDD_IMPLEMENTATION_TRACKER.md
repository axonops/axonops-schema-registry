# BDD Gap Implementation Tracker

**Created:** 2026-02-12
**Branch:** `feature/testing`
**Starting state:** 594 BDD scenarios passing, 36 feature files
**Goal:** ~180 new scenarios across 15 new feature files + code changes for Confluent compatibility

---

## HOW TO RESUME

If you are resuming after a crash/compaction, do the following:

1. Read THIS file (`BDD_IMPLEMENTATION_TRACKER.md`) — it has the current status of all work packages
2. Read `BDD_GAP_ANALYSIS.md` — it has the full gap analysis with 56 gaps
3. Look at the **STATUS** column in the work package table below — find the first `TODO` or `IN PROGRESS` item
4. Read the **detailed section** for that work package below the table
5. Run `go test -tags bdd -v ./tests/bdd/... 2>&1 | tail -20` to see current test status
6. Continue from where you left off

**IMPORTANT:** After completing each work package, UPDATE THIS FILE with the new status before moving on. This is critical for crash recovery.

---

## WORK PACKAGE OVERVIEW

| WP | Name | GAPs | Status | Feature File | Code Changes | Scenarios |
|----|------|------|--------|-------------|-------------|-----------|
| 1 | Step Definitions | All | DONE | N/A (step files) | schema_steps.go, context.go | 0 |
| 2 | Response Shapes | 08,09,17-22,52,53 | DONE | response_shapes.feature | handlers.go (schemaTypeForResponse), types.go (omitempty) | ~22 |
| 3 | Raw Schema Endpoints | 54,55,56 | DONE | raw_schema_endpoints.feature | handlers.go (referencedby 404) | ~14 |
| 4 | Subject Filtering | 11,31,34,46 | DONE | subject_filtering.feature | handlers.go (subjectPrefix) | ~9 |
| 5 | Edge Cases | 23-25,28-30,35 | DONE | edge_cases.feature | handlers.go (parseVersion 422/42202, errInvalidVersion), types.go (ErrorCodeInvalidVersion) | ~20 |
| 6 | Content Types | 29 | DONE | content_types.feature | None | ~3 |
| 7 | Schema ID Stability | 27,32 | DONE | schema_id_stability.feature | None | ~7 |
| 8 | Verbose Compat | 01 | DONE | compatibility_verbose.feature | handlers.go (verbose param) | 8 |
| 9 | Config & Mode Defaults | 04,33,40 | DONE | config_defaults.feature | handlers.go+registry.go (defaultToGlobal, GetSubjectConfig/Mode) | 12 |
| 10 | Deletion Lifecycle | 02,03,05,10,12-14,26,45,48 | DONE | deletion_lifecycle.feature | handlers.go, registry.go, memory/store.go (two-step delete enforcement) | 14 |
| 11 | Mode Enforcement | 06,07,15,41,47,49 | DONE | mode_enforcement.feature | handlers.go (checkModeForWrite), registry.go (READONLY_OVERRIDE) | 12 |
| 12 | Error Codes | 16,49-51 | DONE | error_codes_exhaustive.feature | Minimal | 17 |
| 13 | Advanced Features | 37-39,42-44 | DONE | advanced_features.feature | handlers.go (force, deletedOnly, subject filter), registry.go (SetMode force), storage.go (ErrOperationNotPermitted) | 18 |
| 14 | Confluent Conformance | NEW | DONE | confluent_conformance.feature | handlers.go (lookup 40401 vs 40403, double-delete 40404), memory/store.go (GetSchemaByFingerprint, GetSchemasBySubject, DeleteSubject) | 12 |
| 15 | Pagination | NEW | DONE | pagination.feature | handlers.go (offset/limit/deletedOnly on GET /subjects) | 8 |
| 16 | normalize (P2) | 36 | DONE | (in advanced_features.feature) | schema/types.go (Normalize), avro/parser.go, protobuf/parser.go, jsonschema/parser.go, handlers.go, registry.go (isNormalizeEnabled), storage.go (ConfigRecord.Normalize), types.go (ConfigRequest/Response.Normalize) | 5 |
| 17 | format | 42 | DONE | (in advanced_features.feature) | schema/types.go (FormattedString), avro/parser.go, protobuf/parser.go, jsonschema/parser.go, handlers.go, registry.go (FormatSchema) | 6 |
| 18 | fetchMaxId | 43 | DONE | (in advanced_features.feature) | handlers.go, types.go (MaxId), storage interface (GetMaxSchemaID), memory/postgres/mysql/cassandra stores | 3 |

**Completed: WP1-WP18 = 770 scenarios passing. All work packages done.**

---

## WP1: STEP DEFINITIONS

**Status:** DONE
**Files to modify:** `tests/bdd/steps/schema_steps.go`, `tests/bdd/steps/reference_steps.go`

### Already Existing Steps (DO NOT RECREATE)
These steps ALREADY EXIST and cover many needs for our new feature files:
```
I GET "([^"]*)"                                          # schema_steps.go:58 — raw GET
I POST "([^"]*)" with body:                              # schema_steps.go:61 — raw POST
I PUT "([^"]*)" with body:                               # schema_steps.go:68 — raw PUT
I DELETE "([^"]*)"                                       # schema_steps.go:75 — raw DELETE
the response should not have field "([^"]*)"             # schema_steps.go:322 — field absence
the response should contain "([^"]*)"                    # schema_steps.go:196 — body contains
the response body should not contain "([^"]*)"           # schema_steps.go:338 — body NOT contains
I store the response field "([^"]*)" as "([^"]*)"        # schema_steps.go:270 — store values
the response should be valid JSON                        # schema_steps.go:304 — JSON validation
the response field "([^"]*)" should be (true|false)      # schema_steps.go:311 — boolean check
I delete the mode for subject "([^"]*)"                  # mode_steps.go:27 — DELETE mode
I delete the global config                               # mode_steps.go:31 — DELETE /config
I get the raw schema by ID N                             # reference_steps.go:35 — GET /schemas/ids/N/schema
I get the raw schema for subject "X" version N           # reference_steps.go:39 — GET raw schema by version
I get the referenced by for subject "X" version N        # reference_steps.go:31 — referencedby
I get versions for schema ID N                           # reference_steps.go:43 — GET /schemas/ids/N/versions
I check compatibility of schema against subject "X" version N:  # reference_steps.go:60
I check compatibility of schema against all versions of subject "X":  # reference_steps.go:47
I permanently delete version N of subject "X"            # reference_steps.go:77
I list subjects with deleted                             # reference_steps.go:73
```

### New Steps Needed
```go
// schema_steps.go — new steps to add:

// 1. Response is a plain integer (for DELETE version response body)
// Pattern: ^the response should be an integer with value (\d+)$
// Parses tc.LastBody as integer, compares to expected

// 2. Response array contains an integer (for DELETE subject response body)
// Pattern: ^the response array should contain integer (\d+)$
// Checks tc.LastJSONArray contains the integer

// 3. Response field is an array (for verbose messages)
// Pattern: ^the response field "([^"]*)" should be an array$
// Checks that the field value is a JSON array

// 4. Response field is an array of length N
// Pattern: ^the response field "([^"]*)" should be an array of length (\d+)$
// Checks field is array with specific length

// 5. Response header check (for Content-Type)
// Pattern: ^the response header "([^"]*)" should contain "([^"]*)"$
// Checks tc.LastResponse.Header.Get(name) contains value

// reference_steps.go — new steps to add:

// 6. Compatibility check with verbose param
// Pattern: ^I check compatibility with verbose of schema against subject "([^"]*)" version "([^"]*)":$
// POST /compatibility/subjects/{subject}/versions/{version}?verbose=true

// 7. Compatibility check with verbose against all versions
// Pattern: ^I check compatibility with verbose of schema against all versions of subject "([^"]*)":$
// POST /compatibility/subjects/{subject}/versions?verbose=true

// 8. Compatibility check with verbose and schema type
// Pattern: ^I check compatibility with verbose of "([^"]*)" schema against subject "([^"]*)" version "([^"]*)":$
// POST /compatibility/subjects/{subject}/versions/{version}?verbose=true with schemaType
```

### Implementation Notes
- Most new scenarios can use the raw `I GET/POST/PUT/DELETE "..."` steps with query params in the URL
- Only add dedicated steps when the raw approach is too verbose or error-prone
- Keep step count minimal to reduce maintenance burden

---

## WP2: RESPONSE SHAPES

**Status:** DONE
**Feature file:** `tests/bdd/features/response_shapes.feature`
**Code changes needed:** None expected (verify current behavior matches Confluent)
**GAPs covered:** 08, 09, 17, 18, 19, 20, 21, 22, 52, 53

### Scenarios to write:
1. Register schema → response has only `id` field (positive integer) [GAP-17]
2. GET schema by ID (Avro) → has `schema`, NO `schemaType` field [GAP-18, 52]
3. GET schema by ID (Protobuf) → has `schema`, `schemaType` = "PROTOBUF" [GAP-18, 52]
4. GET schema by ID (JSON) → has `schema`, `schemaType` = "JSON" [GAP-18, 52]
5. GET schema by ID with references → has `references` array [GAP-18]
6. GET schema by ID without references → no `references` field or empty [GAP-18]
7. GET subject/version (Avro) → has subject, id, version, schema, NO schemaType [GAP-19, 52]
8. GET subject/version (Protobuf) → has schemaType = "PROTOBUF" [GAP-19]
9. GET subject/version (JSON) → has schemaType = "JSON" [GAP-19]
10. Lookup response (Avro) → has subject, id, version, schema, NO schemaType [GAP-20, 52]
11. Lookup response (Protobuf) → has schemaType = "PROTOBUF" [GAP-20]
12. GET /config → response has `compatibilityLevel` (not `compatibility`) [GAP-21]
13. PUT /config → response has `compatibility` (not `compatibilityLevel`) [GAP-21]
14. GET /config/{subject} → has `compatibilityLevel` [GAP-21]
15. PUT /config/{subject} → has `compatibility` [GAP-21]
16. GET /mode → has `mode` field [GAP-22]
17. PUT /mode → has `mode` field [GAP-22]
18. DELETE subject with 1 version → response is array `[1]` [GAP-08]
19. DELETE subject with 3 versions → response is array `[1,2,3]` [GAP-08]
20. DELETE version → response is the version integer [GAP-09]
21. DELETE /config/{subject} → response has removed `compatibilityLevel` [GAP-53]
22. DELETE /config/{subject} when no config → 404 [GAP-53]

---

## WP3: RAW SCHEMA ENDPOINTS

**Status:** DONE
**Feature file:** `tests/bdd/features/raw_schema_endpoints.feature`
**Code changes needed:** None expected
**GAPs covered:** 54, 55, 56

### Scenarios to write:
1. GET /schemas/ids/{id}/schema for Avro → returns valid JSON string [GAP-54]
2. GET /schemas/ids/{id}/schema for Protobuf → returns proto text [GAP-54]
3. GET /schemas/ids/{id}/schema for JSON Schema → returns valid JSON [GAP-54]
4. GET /schemas/ids/{id}/schema for non-existent ID → 40403 [GAP-54]
5. GET /subjects/{subject}/versions/{version}/schema for Avro → schema string [GAP-55]
6. GET /subjects/{subject}/versions/{version}/schema for "latest" → works [GAP-55]
7. GET /subjects/{subject}/versions/{version}/schema non-existent version → 40402 [GAP-55]
8. GET /subjects/{subject}/versions/{version}/schema non-existent subject → 40401 [GAP-55]
9. GET referencedby with no references → empty array [GAP-56]
10. GET referencedby with 1 reference → array with 1 ID [GAP-56]
11. GET referencedby with multiple references → array with multiple IDs [GAP-56]
12. GET referencedby with "latest" as version → works [GAP-56]
13. GET referencedby non-existent subject → 40401 [GAP-56]
14. GET referencedby non-existent version → 40402 [GAP-56]

---

## WP4: SUBJECT FILTERING

**Status:** DONE
**Feature file:** `tests/bdd/features/subject_filtering.feature`
**Code changes needed:** None expected
**GAPs covered:** 11, 31, 34, 46

### Scenarios to write:
1. GET /subjects?subjectPrefix=test- → only matching subjects [GAP-11]
2. GET /subjects?subjectPrefix= (empty) → all subjects [GAP-11]
3. GET /subjects?subjectPrefix=nonexistent → empty array [GAP-11]
4. GET /subjects?subjectPrefix=x&deleted=true → combines both filters [GAP-11]
5. GET /schemas/ids/{id}/subjects → shows subjects using schema [GAP-31]
6. GET /schemas/ids/{id}/subjects?deleted=true → includes soft-deleted subjects [GAP-31]
7. GET /schemas/ids/{id}/subjects without deleted → hides soft-deleted [GAP-31]
8. GET /schemas/ids/{id}/versions?deleted=true → includes soft-deleted pairs [GAP-46]
9. GET /schemas/ids/{id}/versions without deleted → hides soft-deleted [GAP-46, 34]

---

## WP5: EDGE CASES

**Status:** DONE
**Feature file:** `tests/bdd/features/edge_cases.feature`
**Code changes needed:** None expected
**GAPs covered:** 23, 24, 25, 28, 29, 30, 35

### Scenarios to write:
1. Subject with dots `com.example.Topic-value` → works [GAP-23]
2. Subject with underscores `my_topic_value` → works [GAP-23]
3. Subject with dashes `my-topic-value` → works [GAP-23]
4. Register with missing schema field → 400/422 [GAP-24]
5. Register with empty JSON body `{}` → error [GAP-24]
6. Register with non-JSON body → 400 [GAP-24]
7. PUT /config with empty body → error [GAP-24]
8. PUT /config with missing compatibility field → error [GAP-24]
9. PUT /mode with missing mode field → error [GAP-24]
10. Compatibility check with empty body → error [GAP-24]
11. GET /schemas?offset=999&limit=10 → empty array [GAP-25]
12. GET /schemas?offset=0&limit=1 → 1 result [GAP-25]
13. Duplicate registration → same ID, same version (idempotent) [GAP-28]
14. Compat check against version 1 specifically [GAP-30]
15. Compat check against version 2 specifically [GAP-30]
16. Compat check against non-existent version → 40402 [GAP-30]
17. Compat check against non-existent subject → 40401 [GAP-30]
18. Version 0 in GET → error [GAP-35]
19. Version -2 in GET → error [GAP-35]
20. Version "abc" in GET → error [GAP-35]

---

## WP6: CONTENT TYPES

**Status:** DONE
**Feature file:** `tests/bdd/features/content_types.feature`
**Code changes needed:** None expected
**GAPs covered:** 29

### Scenarios to write:
1. Response Content-Type contains `application/vnd.schemaregistry.v1+json` [GAP-29]
2. POST with Content-Type `application/json` → accepted [GAP-29]
3. POST with Content-Type `application/vnd.schemaregistry.v1+json` → accepted [GAP-29]

---

## WP7: SCHEMA ID STABILITY

**Status:** DONE
**Feature file:** `tests/bdd/features/schema_id_stability.feature`
**Code changes needed:** None expected
**GAPs covered:** 27, 32

### Scenarios to write:
1. Same Avro schema across subjects → same ID (verify with GET by ID) [GAP-27]
2. Same Protobuf schema across subjects → same ID [GAP-27]
3. Same JSON Schema across subjects → same ID [GAP-27]
4. Different schemas → different IDs [GAP-27]
5. Retrieved Avro schema is valid parseable JSON [GAP-32]
6. Retrieved Protobuf schema is valid proto text [GAP-32]
7. Retrieved JSON Schema is valid parseable JSON [GAP-32]
8. GET schema by ID → schema field matches what was registered [GAP-32]

---

## WP8: VERBOSE COMPATIBILITY

**Status:** DONE
**Feature file:** `tests/bdd/features/compatibility_verbose.feature`
**Code changes needed:** YES — `internal/api/handlers/handlers.go` CheckCompatibility
**GAPs covered:** 01

### Code change needed:
In `CheckCompatibility` handler (line ~368-415), add:
```go
verbose := r.URL.Query().Get("verbose") == "true"
// ... after getting result ...
if !verbose {
    result.Messages = nil  // omit messages when not verbose
}
```

### Scenarios to write:
1. Incompatible schema + verbose=true → has messages array [GAP-01]
2. Compatible schema + verbose=true → no messages (or empty) [GAP-01]
3. Incompatible schema without verbose → no messages field [GAP-01]
4. Compatible schema without verbose → no messages field [GAP-01]
5. Verbose with Avro incompatible schema → messages describe issue [GAP-01]
6. Verbose with Protobuf incompatible schema [GAP-01]
7. Verbose with JSON Schema incompatible schema [GAP-01]
8. Verbose against all versions (not just specific version) [GAP-01]

---

## WP9: CONFIG & MODE DEFAULTS

**Status:** DONE
**Feature file:** `tests/bdd/features/config_defaults.feature`
**Code changes needed:** YES — `internal/api/handlers/handlers.go`, `internal/registry/registry.go`
**GAPs covered:** 04, 33, 40

### Code changes needed:

**handlers.go GetConfig** (~line 311):
```go
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
    subject := chi.URLParam(r, "subject")
    defaultToGlobal := r.URL.Query().Get("defaultToGlobal") == "true"

    if subject != "" && !defaultToGlobal {
        // Only return subject-specific config, 404 if not set
        level, err := h.registry.GetSubjectConfig(ctx, subject)
        if err != nil {
            if errors.Is(err, storage.ErrNotFound) {
                writeError(w, 404, types.ErrorCodeSubjectNotFound, "Subject config not found")
                return
            }
            ...
        }
        writeJSON(w, 200, types.ConfigResponse{CompatibilityLevel: level})
        return
    }
    // Original behavior (fallback to global)
    level, err := h.registry.GetConfig(r.Context(), subject)
    ...
}
```

**handlers.go GetMode** (~line 449): Same pattern for mode endpoint.

**registry.go**: May need a `GetSubjectConfig` method that does NOT fall back to global. Or modify `GetConfig` to accept a `defaultToGlobal` parameter.

### Scenarios to write:
1. GET /config/{subject} with subject config set → returns subject config [GAP-04]
2. GET /config/{subject} without config, no param → 404 [GAP-04]
3. GET /config/{subject}?defaultToGlobal=true without config → returns global [GAP-04]
4. GET /config/{subject}?defaultToGlobal=false without config → 404 [GAP-04]
5. DELETE /config → resets to BACKWARD [GAP-33]
6. DELETE /config when already default → 200 [GAP-33]
7. GET /mode/{subject} with mode set → returns subject mode [GAP-40]
8. GET /mode/{subject} without mode, no param → 404 [GAP-40]
9. GET /mode/{subject}?defaultToGlobal=true without mode → returns global [GAP-40]
10. GET /mode/{subject}?defaultToGlobal=false without mode → 404 [GAP-40]
11. Set config → delete → GET returns global [GAP-33]
12. Global config always returns a value (never 404) [GAP-04]

---

## WP10: DELETION LIFECYCLE

**Status:** DONE
**Feature file:** `tests/bdd/features/deletion_lifecycle.feature`
**Code changes needed:** YES — `handlers.go`, `registry.go`, `memory/store.go`
**GAPs covered:** 02, 03, 05, 10, 12, 13, 14, 26, 45, 48

### Code changes needed:

**1. Two-step delete enforcement (GAP-05, 13):**
In registry.go `DeleteSubject` and `DeleteVersion`:
- If `permanent=true`, check that the resource is already soft-deleted
- If not soft-deleted, return error with code 40405

**2. Soft-deleted version GET behavior (GAP-14, 45):**
In handlers.go version GET:
- Parse `deleted` query param
- If version is soft-deleted and `deleted` param is not true → 404

**3. Version "latest" for hard delete restriction (GAP-48):**
In handlers.go DeleteVersion:
- If `permanent=true` and version is "latest", only do soft delete (or reject)

### Scenarios to write:
1. List versions with ?deleted=true shows soft-deleted versions [GAP-02]
2. List versions without param hides soft-deleted versions [GAP-02]
3. Permanent delete then list with ?deleted=true → gone [GAP-02]
4. Lookup after soft-delete without ?deleted=true → 404 [GAP-03]
5. Lookup after soft-delete with ?deleted=true → 200 [GAP-03]
6. Lookup after permanent delete with ?deleted=true → 404 [GAP-03]
7. Permanent delete without prior soft-delete → 40405 error [GAP-05, 13]
8. Soft-delete then permanent delete subject → 200 [GAP-05]
9. Soft-delete then permanent delete version → 200 [GAP-05]
10. DELETE version "latest" soft-deletes latest [GAP-10]
11. DELETE version -1 works like latest [GAP-10]
12. GET version -1 works like latest [GAP-10]
13. DELETE version "latest" then GET latest → previous version [GAP-10]
14. GET soft-deleted version without ?deleted → 404 [GAP-14, 45]
15. GET soft-deleted version with ?deleted=true → 200 [GAP-45]
16. GET permanently deleted version with ?deleted=true → 404 [GAP-45]
17. Hard delete with "latest" → only soft-deletes [GAP-48]
18. Hard delete with explicit version after soft-delete → works [GAP-48]
19. Re-register after soft-delete → version continues [GAP-26]
20. Re-register after permanent delete → version 1 [GAP-26]
21. Register under soft-deleted subject → check behavior [GAP-12]

---

## WP11: MODE ENFORCEMENT

**Status:** DONE
**Feature file:** `tests/bdd/features/mode_enforcement.feature`
**Code changes needed:** YES — `handlers.go`, `registry.go`
**GAPs covered:** 06, 07, 15, 41, 47, 49

### Code changes needed:

**1. Add mode checking to write handlers (GAP-06, 15):**
Add a helper function or middleware that checks the effective mode for a subject:
```go
func (h *Handler) checkWritePermission(ctx context.Context, subject string) error {
    mode, _ := h.registry.GetMode(ctx, subject)
    if mode == "READONLY" || mode == "READONLY_OVERRIDE" {
        return fmt.Errorf("operation not permitted in %s mode", mode)
    }
    return nil
}
```
Apply to: RegisterSchema, DeleteSubject, DeleteVersion, SetConfig, SetMode (unless READONLY_OVERRIDE)

**2. Add READONLY_OVERRIDE to valid modes (GAP-41):**
In registry.go, add "READONLY_OVERRIDE" to valid modes map.

**3. IMPORT mode on standard endpoint (GAP-07):**
In RegisterSchema handler, when mode is IMPORT, parse `id` and `version` from request body.

### Scenarios to write:
1. READONLY → register schema → 422 error 42205 [GAP-06, 15]
2. READONLY → delete subject → 422 error 42205 [GAP-06]
3. READONLY → delete version → 422 error 42205 [GAP-06]
4. READONLY → PUT config → 422 error 42205 [GAP-06]
5. READONLY → GET operations still work [GAP-06]
6. Per-subject READONLY → only that subject blocked [GAP-06]
7. Global READONLY, per-subject READWRITE → subject allowed [GAP-06]
8. READONLY_OVERRIDE → blocks writes like READONLY [GAP-41]
9. READONLY_OVERRIDE → allows PUT /mode [GAP-41]
10. Set mode READONLY_OVERRIDE → accepted [GAP-41]
11. Invalid mode value → 42204 error [GAP-49]
12. IMPORT mode → register with id field → preserved [GAP-07]
13. READWRITE mode → register with id field → ignored [GAP-07]
14. IMPORT mode → skip compat checks [GAP-07]
15. DELETE /mode/{subject} → falls back to global [GAP-47]
16. DELETE /mode/{subject} when no subject mode → 404 [GAP-47]
17. DELETE /mode/{subject} → response has removed mode [GAP-47]

---

## WP12: ERROR CODES EXHAUSTIVE

**Status:** DONE
**Feature file:** `tests/bdd/features/error_codes_exhaustive.feature`
**Code changes needed:** Minimal (most covered by WP10/WP11 code changes)
**GAPs covered:** 16, 49, 50, 51

### Scenarios to write:
1. Invalid mode → 42204 [GAP-49]
2. Invalid schema type → check error code [GAP-50]
3. Invalid version string "abc" → check error code [GAP-50]
4. Register incompatible schema → 409 [GAP-51]
5. 409 error message mentions incompatibility [GAP-51]
6. Version 0 → error with code [GAP-50]
7. Negative version (not -1) → error [GAP-50]
8. Missing required fields → appropriate error codes [GAP-50]

---

## WP13: ADVANCED FEATURES — DONE

**Status:** DONE
**Feature file:** `advanced_features.feature`
**Code changes:** handlers.go (force, deletedOnly, subject filter), registry.go (SetMode force), storage.go (ErrOperationNotPermitted), context.go (resolveVars)
**GAPs covered:** 37, 38, 39, 42-44 (@p2 tagged for normalize/format/fetchMaxId)

---

## WP14: CONFLUENT CONFORMANCE (from Confluent test suite analysis)

**Status:** DONE
**Feature file:** `confluent_conformance.feature`
**Code changes needed:** YES — handlers.go, registry.go, memory/store.go
**Source:** Analysis of Confluent's RestApiTest.java (2900+ lines)

### Issues discovered from Confluent test suite:

1. **Lookup error codes: 40401 vs 40403** — POST /subjects/{subject} (lookup) should return:
   - 40401 (SUBJECT_NOT_FOUND) when subject doesn't exist
   - 40403 (SCHEMA_NOT_FOUND) when subject exists but schema is not registered under it
   - Need to verify our implementation distinguishes these two cases

2. **Double soft-delete returns 40404** — DELETE /subjects/{subject} on already-soft-deleted subject should return error code 40404 (SUBJECT_SOFT_DELETED), not succeed silently

3. **Version continuity after soft-delete** — After soft-deleting a subject and re-registering, version numbers must CONTINUE (3, 4...) not reset to (1, 2)

4. **Schema type mixing when compatibility=NONE** — When compat is NONE, a subject should accept Avro, then JSON Schema, then Protobuf as successive versions

5. **Compatibility excludes soft-deleted versions** — Compatibility checks should skip soft-deleted versions (deleting an incompatible version should allow previously-blocked schemas)

6. **Config on non-existent subject succeeds** — PUT /config/{subject} should work even if subject has no schemas yet

7. **Canonical string idempotence** — Schemas with different whitespace but same canonical form should get the same ID

### Scenarios to write (~15):
1. Lookup on non-existent subject → 40401
2. Lookup when schema not under subject → 40403
3. Double soft-delete subject → 40404
4. Double soft-delete version → 40406
5. Version continuity after soft-delete+re-register
6. Schema type mixing (Avro→JSON→Protobuf) under NONE compat
7. Compat check ignores soft-deleted versions
8. Config on non-existent subject succeeds
9. Config on non-existent subject is retrievable
10. Whitespace-different schemas get same ID
11. Permanent delete after soft-delete returns correct version list
12. GET /subjects/{subject}/versions after all versions soft-deleted → 40401
13. Re-register after permanent delete → version 1 (vs soft-delete → version continues)
14. Register with version=-1 in request body → gets next version
15. GET version -1 returns same as GET version latest

---

## WP15: PAGINATION

**Status:** DONE
**Feature file:** `pagination.feature`
**Code changes needed:** YES — handlers.go (offset/limit on GET /subjects, deletedOnly on GET /subjects)

### Confluent behavior (from SubjectsResource.java):
- `offset` (int, default: 0) — skip N items
- `limit` (int, default: -1 = unlimited) — take N items after offset
- `deletedOnly` (bool) — only return soft-deleted subjects (takes precedence over `deleted`)

### Scenarios to write (~8):
1. GET /subjects?limit=2 → returns 2 subjects
2. GET /subjects?offset=1&limit=2 → skips 1, returns 2
3. GET /subjects?offset=999 → empty array
4. GET /subjects?limit=-1 → all subjects (unlimited)
5. GET /subjects?deletedOnly=true → only soft-deleted subjects
6. GET /subjects?deleted=true&deletedOnly=true → deletedOnly takes precedence
7. GET /subjects?subjectPrefix=x&offset=0&limit=1 → combines prefix+pagination
8. GET /subjects?limit=0 → empty array

---

## WP16: NORMALIZE PARAMETER

**Status:** DONE
**Feature file:** `normalize.feature`
**Code changes needed:** YES — schema parsers (Avro, JSON Schema, Protobuf), handlers.go, registry.go, config

### What normalize does per schema type:
- **Avro:** Fully qualifies named types, canonicalizes field order (AvroSchemaUtils.toNormalizedString)
- **JSON Schema:** Re-parses with ordered property mapper for consistent JSON property ordering
- **Protobuf:** Applies protobuf-specific normalization rules (ProtobufSchemaUtils.toNormalizedString)

### Also needed:
- `normalize` config option per subject (in PUT /config/{subject})
- If `?normalize` not passed as query param, falls back to subject/global config setting
- Affects: registration, lookup, and compatibility checking

### Scenarios to write (~10):
1. Register with normalize=true normalizes before storage
2. Lookup with normalize=true matches despite whitespace differences
3. Compat check with normalize=true normalizes before comparing
4. Avro: fields in different order but semantically identical → match with normalize
5. JSON Schema: properties in different order → match with normalize
6. Protobuf: equivalent definitions → match with normalize
7. Set normalize=true in config → applies to all registrations
8. Per-subject normalize config overrides global
9. Without normalize, different whitespace → different schemas
10. normalize=true on compat check prevents false incompatibility

---

## WP17: FORMAT PARAMETER

**Status:** DONE
**Feature file:** `format.feature`
**Code changes needed:** YES — schema parsers, handlers.go

### What format does:
- `format=resolved` — Avro: inlines all named types (no references). Returns self-contained schema.
- `format=serialized` — Protobuf: returns serialized descriptor format
- Applied on: GET /schemas/ids/{id}, GET /schemas/ids/{id}/schema, GET /subjects/{subject}/versions/{version}
- Requires `ParsedSchema.formattedString(format)` method per type

### Scenarios to write (~6):
1. GET schema with format=resolved for Avro with references → inlined schema
2. GET schema without format → schema with references
3. GET raw schema with format=resolved → inlined
4. GET subject/version with format=resolved → inlined
5. Unknown format value → behavior documented
6. Format on Protobuf schema → appropriate format

---

## WP18: FETCH MAX ID

**Status:** DONE
**Feature file:** (in advanced_features.feature, extend)
**Code changes needed:** YES — handlers.go, types.go (SchemaByIDResponse), storage interface (GetMaxSchemaID)

### What fetchMaxId does:
- GET /schemas/ids/{id}?fetchMaxId=true → response includes `maxId` field
- `maxId` equals the highest schema ID in the registry
- Without param → no `maxId` field

### Scenarios to write (~3):
1. GET schema with fetchMaxId=true → response has maxId field
2. maxId equals highest schema ID in registry
3. GET schema without fetchMaxId → no maxId field

---

## EXECUTION ORDER

**All work packages completed (WP1-WP18):**
WP1 (step definitions) → WP2-WP7 (test-only) → WP8-WP15 (code changes) → WP18 (fetchMaxId) → WP17 (format) → WP16 (normalize)

**770 BDD scenarios passing, 0 failures, all unit tests green.**

---

## RUNNING TESTS

```bash
# Build first
make build

# Run all BDD tests
go test -tags bdd -v ./tests/bdd/... 2>&1 | tail -40

# Run specific feature file
go test -tags bdd -v ./tests/bdd/... -godog.paths=tests/bdd/features/response_shapes.feature

# Run with tag filter
go test -tags bdd -v ./tests/bdd/... -godog.tags=@functional
```

---

## CHANGE LOG

| Date | WP | Action | Result |
|------|-----|--------|--------|
| 2026-02-12 | - | Created tracker | 594 scenarios passing |
| 2026-02-12 | 1 | Added 6 new step definitions + DoRawRequest | Step defs ready |
| 2026-02-12 | 2-7 | Created 6 feature files, fixed code bugs | 660 scenarios, all passing |
| 2026-02-12 | 2-7 | Code fixes: schemaType omission (AVRO), PUT /config returns compatibility, subjectPrefix, version -1, parseVersion 422/42202, referencedby 404, empty schema 422, compat error propagation | All unit+BDD tests green |
| 2026-02-12 | 8 | Verbose compat: parse verbose param, omit messages when not verbose | 668 scenarios passing |
| 2026-02-12 | 9 | defaultToGlobal: GetSubjectConfig/GetSubjectMode, handler checks defaultToGlobal param | 680 scenarios passing |
| 2026-02-12 | 10 | Two-step delete: ErrSubjectNotSoftDeleted/ErrVersionNotSoftDeleted, storage checks, handler 40405, registry.DeleteVersion fix for permanent+latest | 694 scenarios passing |
| 2026-02-12 | 11 | Mode enforcement: READONLY_OVERRIDE, checkModeForWrite on register/delete, per-subject mode | 706 scenarios passing |
| 2026-02-12 | 12 | Error codes exhaustive: 17 scenarios covering all Confluent error codes (40401-42205) | 723 scenarios passing |
| 2026-02-12 | 13 | Advanced features: force on PUT /mode, deletedOnly on versions, subject filter on schema-by-ID, {{var}} template support in BDD steps, @p2 tests for normalize/format/fetchMaxId | 741 scenarios passing |
| 2026-02-12 | 14 | Confluent conformance: lookup 40401 vs 40403, double soft-delete 40404, GET versions after all deleted 40401, GetSchemaByFingerprint/GetSchemasBySubject fixes | 753 scenarios passing |
| 2026-02-12 | 15 | Pagination: offset/limit/deletedOnly on GET /subjects | 761 scenarios passing |
| 2026-02-12 | 18 | fetchMaxId: GetMaxSchemaID storage method, MaxId field in SchemaByIDResponse, fetchMaxId query param | 763 scenarios passing |
| 2026-02-12 | 17 | format: FormattedString on ParsedSchema interface, Avro resolved, Protobuf serialized, handlers format param, FormatSchema registry method | 768 scenarios passing |
| 2026-02-12 | 16 | normalize: Normalize() on ParsedSchema interface, normalize query param on register/lookup/compat, normalize config per subject/global, isNormalizeEnabled config hierarchy | 770 scenarios passing |

---

## COMPLETION SUMMARY

All 18 work packages are complete. 770 BDD scenarios passing, 0 failures.

| WP | Priority | Complexity | Description | Status |
|----|----------|-----------|-------------|--------|
| 16 | **P2** | HIGH | normalize parameter (schema normalization per type + config) | DONE |
| 17 | **P2** | MEDIUM | format parameter (schema output formatting per type) | DONE |
| 18 | **P2** | LOW | fetchMaxId parameter (max schema ID in response) | DONE |
