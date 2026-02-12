# Session Resume Guide — BDD Testing & Confluent Compatibility

**Last Updated:** 2026-02-12
**Branch:** `feature/testing`
**Base Branch:** `main`
**Current State:** 761 BDD scenarios passing, 0 failures, all unit tests green

---

## HOW TO RESUME

1. Read THIS file for overview
2. Read `BDD_IMPLEMENTATION_TRACKER.md` for detailed WP status and remaining work
3. Run tests to confirm state: `make build && go test -tags bdd -v ./tests/bdd/... 2>&1 | grep -e "PASS:" -c`
4. Run unit tests: `go clean -testcache && go test ./internal/... 2>&1 | grep -E "FAIL|ok"`
5. Check task list in tracker — WP16-WP18 are remaining

---

## What Was Done

### Overview
BDD-driven development to achieve 100% Confluent Schema Registry API compatibility. We used a gap analysis (BDD_GAP_ANALYSIS.md) to identify 56 behavioral gaps, organized them into 18 work packages (BDD_IMPLEMENTATION_TRACKER.md), and systematically implemented them.

### Work Packages Completed (WP1-WP15)

| WP | Name | What Was Done |
|----|------|---------------|
| 1 | Step Definitions | Added 6 new step definitions + DoRawRequest + resolveVars template support |
| 2 | Response Shapes | schemaType omission (AVRO), PUT /config returns `compatibility` not `compatibilityLevel` |
| 3 | Raw Schema Endpoints | Fixed referencedby to check subject/version existence |
| 4 | Subject Filtering | Added subjectPrefix parameter parsing |
| 5 | Edge Cases | Fixed parseVersion to return 422/42202 for invalid versions |
| 6 | Content Types | Verified content type handling |
| 7 | Schema ID Stability | Verified schema deduplication across subjects |
| 8 | Verbose Compat | Added verbose query param to compatibility check |
| 9 | Config & Mode Defaults | Added defaultToGlobal param, GetSubjectConfig/GetSubjectMode methods |
| 10 | Deletion Lifecycle | Two-step delete enforcement (40405), version -1/latest handling |
| 11 | Mode Enforcement | READONLY/READONLY_OVERRIDE blocks writes (42205), checkModeForWrite |
| 12 | Error Codes | Exhaustive error code coverage (40401-42205) |
| 13 | Advanced Features | force on PUT /mode, deletedOnly on versions, subject filter on schema-by-ID |
| 14 | Confluent Conformance | Lookup 40401 vs 40403, double soft-delete 40404, GET versions after all deleted |
| 15 | Pagination | offset/limit/deletedOnly on GET /subjects |

### Key Code Changes (across all WPs)

**`internal/api/handlers/handlers.go`** — Major changes:
- `schemaTypeForResponse()` — omits "AVRO" from responses (Confluent behavior)
- `parseVersion()` — returns 422/42202 for invalid versions
- `checkModeForWrite()` — enforces READONLY/READONLY_OVERRIDE
- `SetConfig` — returns `compatibility` field (not `compatibilityLevel`)
- `GetConfig`/`GetMode` — defaultToGlobal parameter support
- `CheckCompatibility` — verbose parameter support
- `DeleteSubject` — ErrSubjectDeleted → 40404
- `SetMode` — force parameter, ErrOperationNotPermitted → 42205
- `ListSubjects` — offset/limit/deletedOnly pagination
- `GetVersions` — deletedOnly parameter
- `LookupSchema` — separate 40401 (subject not found) vs 40403 (schema not found)
- `GetSubjectsBySchemaID`/`GetVersionsBySchemaID` — subject filter parameter

**`internal/api/types/types.go`** — Added:
- `omitempty` on SchemaType fields
- `ErrorCodeInvalidVersion = 42202`

**`internal/registry/registry.go`** — Added:
- `GetSubjectConfig()` / `GetSubjectMode()` — no global fallback
- `SetMode()` — force parameter + hasSubjects check
- `DeleteVersion()` — permanent delete + latest/-1 resolution
- `READONLY_OVERRIDE` in isValidMode

**`internal/storage/storage.go`** — Added sentinel errors:
- `ErrSubjectNotSoftDeleted`, `ErrVersionNotSoftDeleted`, `ErrOperationNotPermitted`

**`internal/storage/memory/store.go`** — Fixed:
- `DeleteSchema` — check version soft-deleted before permanent delete
- `DeleteSubject` — check all versions soft-deleted before permanent + double-delete detection
- `GetSchemaByFingerprint` — return ErrSubjectNotFound when subject doesn't exist
- `GetSchemasBySubject` — return ErrSubjectNotFound when all versions deleted

**`tests/bdd/steps/context.go`** — Added:
- `resolveVars()` — {{variable}} template substitution in URLs
- Applied to DoRequest and DoRawRequest

**`tests/bdd/steps/schema_steps.go`** — Added steps + force=true on mode steps
**`tests/bdd/steps/mode_steps.go`** — force=true on Given/When mode steps

### New Feature Files Created (15 files)
- `advanced_features.feature` (18 scenarios)
- `compatibility_verbose.feature` (8 scenarios)
- `config_defaults.feature` (11 scenarios)
- `confluent_conformance.feature` (12 scenarios)
- `content_types.feature` (3 scenarios)
- `deletion_lifecycle.feature` (14 scenarios)
- `edge_cases.feature` (18 scenarios)
- `error_codes_exhaustive.feature` (17 scenarios)
- `mode_enforcement.feature` (12 scenarios)
- `pagination.feature` (8 scenarios)
- `raw_schema_endpoints.feature` (12 scenarios)
- `response_shapes.feature` (20 scenarios)
- `schema_id_stability.feature` (6 scenarios)
- `subject_filtering.feature` (7 scenarios)

### Modified Feature Files (9 files)
- `api_endpoints_advanced.feature` — schemaType AVRO assertions
- `api_errors.feature` — empty schema 400→422
- `configuration.feature` — defaultToGlobal
- `configuration_advanced.feature` — PUT config field name, mode fixes
- `deletion.feature` — two-step delete
- `deletion_advanced.feature` — two-step delete
- `mode_management.feature` — defaultToGlobal
- `schema_lookup.feature` — lookup error codes 40401 vs 40403

---

## What Remains (WP16-WP18)

These are P2 features requiring schema parser changes. They are tracked in BDD_IMPLEMENTATION_TRACKER.md.

| WP | Feature | Complexity | Description |
|----|---------|-----------|-------------|
| 16 | `normalize` | HIGH | Schema normalization per type (Avro canonical names, JSON Schema ordered props, Protobuf normalized defs). Also needs normalize config per subject. |
| 17 | `format` | MEDIUM | Schema output formatting (format=resolved inlines references). Needs ParsedSchema.formattedString(format) per type. |
| 18 | `fetchMaxId` | LOW | Add maxId field to SchemaByIDResponse. Needs GetMaxSchemaID() storage method. |

### How to Implement WP16 (normalize)
1. Add `normalize()` method to ParsedSchema interface in `internal/schema/types.go`
2. Implement per type:
   - Avro: fully qualify names, canonicalize field order
   - JSON Schema: ordered property mapper
   - Protobuf: standardize message/field definitions
3. Add `normalize` query param to RegisterSchema, LookupSchema, CheckCompatibility handlers
4. Add `normalize` config option to ConfigRecord/ConfigRequest
5. If query param not set, fall back to subject/global config setting

### How to Implement WP17 (format)
1. Add `formattedString(format string)` method to ParsedSchema interface
2. Implement per type (e.g., format=resolved inlines all references)
3. Add `format` query param to GetSchemaByID, GetRawSchemaByID, GetVersion handlers

### How to Implement WP18 (fetchMaxId)
1. Add `GetMaxSchemaID()` method to Storage interface
2. Implement in memory store: `atomic.LoadInt64(&s.nextID) - 1`
3. Implement in other backends (simple SQL/CQL query)
4. Add `fetchMaxId` query param to GetSchemaByID handler
5. Add `MaxId` field to SchemaByIDResponse (with `omitempty`)

---

## Running Tests

```bash
# Build
make build

# Run all BDD tests
go test -tags bdd -v ./tests/bdd/... 2>&1 | tail -40

# Count passing scenarios
go test -tags bdd -v ./tests/bdd/... 2>&1 | grep -e "PASS:" -c

# Run unit tests (no cache)
go clean -testcache && go test ./internal/...

# Run specific feature file
go test -tags bdd -v ./tests/bdd/... -godog.paths=tests/bdd/features/confluent_conformance.feature
```

---

## Resume Prompt

> I'm working on the `feature/testing` branch of the axonops-schema-registry project. Read `SESSION_RESUME.md` and `BDD_IMPLEMENTATION_TRACKER.md` in the project root for full context.
>
> In previous sessions, I completed WP1-WP15 (761 BDD scenarios passing, all unit tests green). The remaining work is WP16 (normalize), WP17 (format), and WP18 (fetchMaxId) — these are P2 features requiring schema parser changes.
>
> The Confluent Schema Registry source code is checked out at `/Users/johnny/Development/schema-registry` for reference.
