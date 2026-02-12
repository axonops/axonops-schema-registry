# Session Resume Guide — BDD Testing & Confluent Compatibility

**Last Updated:** 2026-02-12
**Branch:** `feature/testing`
**Base Branch:** `main`
**Current State:** 770 BDD scenarios passing, 0 failures, all unit tests green
**Status:** ALL 18 WORK PACKAGES COMPLETE

---

## HOW TO RESUME

1. Read THIS file for overview
2. Read `BDD_IMPLEMENTATION_TRACKER.md` for detailed WP status
3. Run tests to confirm state: `make build && go test -tags bdd -v ./tests/bdd/... 2>&1 | grep -e "PASS:" -c`
4. Run unit tests: `go clean -testcache && go test ./internal/... 2>&1 | grep -E "FAIL|ok"`

---

## What Was Done

### Overview
BDD-driven development to achieve 100% Confluent Schema Registry API compatibility. We used a gap analysis (BDD_GAP_ANALYSIS.md) to identify 56 behavioral gaps, organized them into 18 work packages (BDD_IMPLEMENTATION_TRACKER.md), and systematically implemented them all.

### Work Packages Completed (WP1-WP18)

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
| 16 | Normalize | Normalize() on ParsedSchema interface, normalize query param on register/lookup/compat, normalize config per subject/global, isNormalizeEnabled config hierarchy |
| 17 | Format | FormattedString on ParsedSchema interface, Avro resolved, Protobuf serialized (base64 FileDescriptorProto), FormatSchema registry method |
| 18 | fetchMaxId | GetMaxSchemaID storage method across all backends, MaxId field in SchemaByIDResponse, fetchMaxId query param |

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
- `GetSchemaByID` — fetchMaxId and format query params
- `GetRawSchemaByID`/`GetVersion`/`GetRawSchemaByVersion` — format query param
- `RegisterSchema`/`LookupSchema`/`CheckCompatibility` — normalize query param
- `SetConfig` — passes normalize to registry

**`internal/api/types/types.go`** — Added:
- `omitempty` on SchemaType fields
- `ErrorCodeInvalidVersion = 42202`
- `MaxId *int64` in SchemaByIDResponse
- `Normalize *bool` in ConfigRequest/ConfigResponse

**`internal/registry/registry.go`** — Added:
- `GetSubjectConfig()` / `GetSubjectMode()` — no global fallback
- `SetMode()` — force parameter + hasSubjects check
- `DeleteVersion()` — permanent delete + latest/-1 resolution
- `READONLY_OVERRIDE` in isValidMode
- `GetMaxSchemaID()` — delegates to storage
- `FormatSchema()` — parses schema with refs, calls FormattedString
- `isNormalizeEnabled()` — checks subject config, then global config
- `RegisterSchema`/`LookupSchema`/`CheckCompatibility` — normalize variadic param
- `SetConfig` — accepts normalize *bool parameter

**`internal/schema/types.go`** — Added to ParsedSchema interface:
- `FormattedString(format string) string`
- `Normalize() ParsedSchema`

**`internal/schema/avro/parser.go`** — Added:
- `FormattedString()` — "resolved" returns rawSchema.String()
- `Normalize()` — returns new ParsedSchema with canonical form

**`internal/schema/protobuf/parser.go`** — Added:
- `FormattedString()` — "serialized" converts to base64 FileDescriptorProto
- `Normalize()` — returns new ParsedProtobuf with normalized raw
- `toFileDescriptorProto()` + helper functions for protobuf descriptor conversion

**`internal/schema/jsonschema/parser.go`** — Added:
- `FormattedString()` — always returns canonical
- `Normalize()` — returns new ParsedJSONSchema with canonical as raw

**`internal/storage/storage.go`** — Added:
- `GetMaxSchemaID(ctx context.Context) (int64, error)` to Storage interface
- `Normalize *bool` to ConfigRecord
- Sentinel errors: `ErrSubjectNotSoftDeleted`, `ErrVersionNotSoftDeleted`, `ErrOperationNotPermitted`

**`internal/storage/memory/store.go`** — Added/Fixed:
- `GetMaxSchemaID()` — returns nextID - 1
- `DeleteSchema` — check version soft-deleted before permanent delete
- `DeleteSubject` — check all versions soft-deleted before permanent + double-delete detection
- `GetSchemaByFingerprint` — return ErrSubjectNotFound when subject doesn't exist
- `GetSchemasBySubject` — return ErrSubjectNotFound when all versions deleted

**`internal/storage/postgres/store.go`** — Added `GetMaxSchemaID()` (SELECT MAX(id))
**`internal/storage/mysql/store.go`** — Added `GetMaxSchemaID()` (SELECT MAX(id))
**`internal/storage/cassandra/store.go`** — Added `GetMaxSchemaID()` (reads from id_alloc)

**`internal/registry/registry_test.go`** — Updated all SetConfig calls with nil 4th param

**`tests/bdd/steps/context.go`** — Added `resolveVars()` for {{variable}} template substitution
**`tests/bdd/steps/schema_steps.go`** — Added steps + force=true on mode steps
**`tests/bdd/steps/mode_steps.go`** — force=true on Given/When mode steps

### Feature Files (15 new + 9 modified)

**New files (15):**
- `advanced_features.feature` (18+3+6+5=32 scenarios — force, deletedOnly, subject filter, fetchMaxId, format, normalize)
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

**Modified files (9):**
- `api_endpoints_advanced.feature` — schemaType AVRO assertions
- `api_errors.feature` — empty schema 400→422
- `configuration.feature` — defaultToGlobal
- `configuration_advanced.feature` — PUT config field name, mode fixes
- `deletion.feature` — two-step delete
- `deletion_advanced.feature` — two-step delete
- `mode_management.feature` — defaultToGlobal
- `schema_lookup.feature` — lookup error codes 40401 vs 40403

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
go test -tags bdd -v ./tests/bdd/... -godog.paths=tests/bdd/features/advanced_features.feature
```
