# Session Resume Guide — BDD Testing & Confluent Compatibility

**Last Updated:** 2026-02-13
**Branch:** `feature/testing`
**Base Branch:** `main`

---

## Current State — ALL COMPLETE

**796 BDD scenarios** total across 50 feature files.
**CI fully green** — 23/23 jobs pass.

| Backend | Scenarios | Status |
|---------|-----------|--------|
| Memory | 785 | PASS |
| PostgreSQL | 777 | PASS |
| MySQL | 775 | PASS |
| Cassandra | 775 | PASS |
| Confluent 8.1.1 | 752 | PASS |
| Functional (in-process) | 772 | PASS |

**0 `@confluent-compat-diff` tags** remain — all 56 were fixed and removed.
All unit tests pass. `go vet` clean.

---

## Compatibility Checker Bug Fixes — COMPLETE

### Protobuf Checker (`internal/compatibility/protobuf/checker.go`)

8 fixes applied:
- **P1**: Corrected wire-type compatible groups (varint, zigzag, 32-bit, 64-bit, bytes)
- **P2**: Added enum-to-integer compatibility (enum ↔ varint group)
- **P3**: Field removal now compatible (except oneof fields)
- **P4**: Enum value removal now compatible
- **P5**: Service definitions ignored (gRPC metadata, no wire impact)
- **P6**: Syntax version check removed (source-level annotation only)
- **P7**: repeated→singular compatible for string/bytes/message types
- **P8**: Enum type removal now compatible

### JSON Schema Checker (`internal/compatibility/jsonschema/checker.go`)

3 fixes applied:
- **J1**: Added `hasOpenContentModel()` helper
- **J2**: Content-model-aware property handling:
  - Open model: property removal = compatible, property addition = incompatible
  - Closed model (additionalProperties:false): property removal = incompatible, property addition = compatible
- **J3**: Array items schema removal now compatible (relaxation)

### Unit Tests Updated

- `internal/compatibility/protobuf/checker_test.go`: 8 tests flipped, 19 new tests added (40 total)
- `internal/compatibility/jsonschema/checker_test.go`: 3 tests flipped, 8 new tests added (35 total)
- `internal/compatibility/checker_test.go`: 3 integration tests updated

### BDD Scenarios Updated

- `compatibility_protobuf.feature`: 27 scenarios updated (tags removed)
- `compatibility_jsonschema.feature`: 13 scenarios updated (tags removed)
- `compatibility_transitive.feature`: 14 scenarios updated (7 protobuf + 7 JSON Schema, tags removed)
- `compatibility_modes.feature`: 2 scenarios updated (tags removed)

---

## Confluent 8.1.1 Compatibility Fixes — COMPLETE

Tested against `confluentinc/cp-schema-registry:8.1.1` + `confluentinc/cp-kafka:8.1.1`.
Docker Compose: `tests/bdd/docker-compose.confluent.yml`

### Code Fixes (match Confluent behavior)

- [x] **Fix 1** (4 failures): Always include `schemaType` in responses
- [x] **Fix 2** (2 failures): Error code 40407 for version-level permanent delete
- [x] **Fix 3** (2 failures): Error code 40402 for compat check on non-existent subject
- [x] **Fix 4** (1 failure): Error code 42201 for invalid schema type
- [x] **Fix 6** (1 failure): Lookup with empty schema returns 404/40403
- [x] **Fix 7** (1 failure): limit=0 means unlimited
- [x] **Fix 10**: Reference deletion — subject soft-delete now blocked when versions have active references (422/42206)
- [x] **Fix 11**: PUT /config with empty body returns current config (200)
- [x] **Fix 12**: IMPORT mode with explicit ID — stores schema with requested ID
- [x] **Fix 13**: IMPORT mode ID conflict — same content reuses ID (200), different content rejects (422/42205)
- [x] **Fix 14**: All storage backends (memory, postgres, mysql, cassandra) updated for import ID reuse

### @axonops-only Scenarios (6 remaining)

These represent intentional divergences (our extensions or Confluent OSS bugs):

| Scenario | File | Reason |
|----------|------|--------|
| PUT /mode with empty body returns 422 | edge_cases.feature:68 | Confluent returns 500 NPE (bug) |
| GET /schemas/ids/99999/schema returns 404 | raw_schema_endpoints.feature:42 | Confluent returns 500 NPE (bug) |
| Schema type case-insensitive | api_endpoints_advanced.feature:21 | Our enhancement (Confluent is case-sensitive) |
| IMPORT mode without explicit ID | mode_enforcement.feature:178 | Our extension (Confluent requires explicit ID) |
| Subject filter on /schemas/ids/{id}/subjects | advanced_features.feature:336 | Documented in Confluent API but not implemented in OSS (bug) |
| Subject filter on /schemas/ids/{id}/versions | advanced_features.feature:352 | Documented in Confluent API but not implemented in OSS (bug) |

### @import Scenarios (14 excluded from Confluent)

These test `POST /import/schemas` — our custom bulk import endpoint that doesn't exist in Confluent:
- `schema_import.feature`: 6 scenarios
- `import_advanced.feature`: 8 scenarios

### Tag Filter Cleanup

- Removed `~@confluent-compat-diff` from `bdd_test.go` (tag no longer exists on any scenario)

---

## CI Pipeline — COMPLETE

Added `BDD Tests (Confluent 8.1.1)` job to `.github/workflows/ci.yaml`.
All 23 CI jobs pass including:
- BDD tests against all 5 backends (Memory, PostgreSQL, MySQL, Cassandra, Confluent)
- Confluent compatibility tests (Go, Java, Python)
- Migration tests (import ID reuse + conflict rejection)
- All conformance, unit, integration, auth, and Docker build jobs

---

## Summary of Changed Files

### Checker code
- `internal/compatibility/protobuf/checker.go` — 8 fixes (P1-P8)
- `internal/compatibility/jsonschema/checker.go` — 3 fixes (J1-J3)

### Registry and handler code
- `internal/registry/registry.go` — `RegisterSchemaWithID()`, `ErrImportIDConflict`, reference checking in `DeleteSubject()`
- `internal/api/handlers/handlers.go` — Import ID routing, ID conflict error handling, reference delete error, empty config body
- `internal/api/types/types.go` — Added `ID` field to `RegisterSchemaRequest`

### Storage backends (import ID reuse)
- `internal/storage/memory/store.go` — Same-fingerprint ID reuse in `ImportSchema`
- `internal/storage/postgres/store.go` — Same-fingerprint ID reuse in `ImportSchema`
- `internal/storage/mysql/store.go` — Same-fingerprint ID reuse in `ImportSchema`
- `internal/storage/cassandra/store.go` — Same-fingerprint ID reuse in `ImportSchema`

### Unit tests
- `internal/compatibility/protobuf/checker_test.go` — 8 flipped + 19 new
- `internal/compatibility/jsonschema/checker_test.go` — 3 flipped + 8 new
- `internal/compatibility/checker_test.go` — 3 integration tests updated

### BDD scenarios
- `tests/bdd/features/compatibility_protobuf.feature` — 27 scenarios updated
- `tests/bdd/features/compatibility_jsonschema.feature` — 13 scenarios updated
- `tests/bdd/features/compatibility_transitive.feature` — 14 scenarios updated
- `tests/bdd/features/compatibility_modes.feature` — 2 scenarios updated
- `tests/bdd/features/mode_enforcement.feature` — 4 IMPORT mode scenarios (2 new)
- `tests/bdd/features/edge_cases.feature` — empty config, pagination tags
- `tests/bdd/features/deletion_advanced.feature` — reference deletion
- `tests/bdd/features/schema_listing_advanced.feature` — removed file-level @axonops-only
- `tests/bdd/features/advanced_features.feature` — subject filter scenarios
- `tests/bdd/bdd_test.go` — removed dead `~@confluent-compat-diff` filter

### CI
- `.github/workflows/ci.yaml` — Added `bdd-confluent-tests` job

### Compatibility tests (Go/Java/Python)
- `tests/compatibility/go/jsonschema_test.go` — closed content model for evolution test
- `tests/compatibility/python/test_jsonschema_compatibility.py` — closed content model for evolution test

### Migration tests
- `tests/migration/migration_test.go` — updated `TestImportWithDuplicateIDs` for fingerprint-aware reuse

---

## Running Tests

```bash
# Unit tests
go test -race -count=1 ./internal/...

# Memory backend BDD tests (785 scenarios)
BDD_BACKEND=memory go test -tags bdd -v -count=1 -timeout 10m ./tests/bdd/...

# Confluent backend (752 scenarios, requires docker-compose up first)
podman compose -f tests/bdd/docker-compose.confluent.yml up -d --wait
BDD_BACKEND=confluent BDD_REGISTRY_URL=http://localhost:18081 go test -tags bdd -v -count=1 -timeout 25m ./tests/bdd/...
```

## Next Steps

All planned work is complete. Ready for new tasks.
