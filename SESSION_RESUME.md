# Session Resume Guide — BDD Testing & Confluent Compatibility

**Last Updated:** 2026-02-14
**Branch:** `feature/testing`
**Base Branch:** `main`

---

## Current State — ALL BDD TESTS PASS (ZERO PENDING)

**1305 passing BDD scenarios** (0 failures, 0 pending) across 67+ feature files.
- **1292 passing** (in-process mode, excludes `@operational`)
- **0 tagged `@pending-impl`** — all previously pending protobuf import tests now pass

### Phase Progress

| Phase | Status | Scenarios Added | Details |
|-------|--------|-----------------|---------|
| Phase 1: API BDD Tests | COMPLETE | 102 | Sections 1-13, 19-21 |
| Phase 2: Avro & Parsing BDD | COMPLETE | 86 | Sections 22-25, 32-34 |
| Phase 3: Protobuf Diff Tests | COMPLETE | 43 | Section 31 data-driven |
| Phase 4-5: JSON Schema Diff | COMPLETE | 251 | Sections 27-29 data-driven |
| Phase 6: JSON Schema Validation | COMPLETE | 40 | Section 30 reader/writer pairs |
| Phase 7: Feature Implementation | COMPLETE | — | All checkers enhanced, aliases, fingerprint fix, structural import comparison |
| Phase 8: Feature BDD Tests | NOT STARTED | — | Tests for Phase 7 features |

### Key Fixes (All Sessions)

1. **API error codes**: 40408 (subject config not found), 40409 (subject mode not found)
2. **Deletion behaviors**: `deleted=true` for GetVersion, config/mode cleanup on delete
3. **Pagination**: offset/limit for versions, schema IDs, subject IDs
4. **FORWARD_TRANSITIVE**: Fixed test data (verified against Confluent v8.1.1)
5. **Protobuf checker**: Required field removal, oneof moves (existing vs new), optional→repeated for length-delimited types, synthetic oneof handling
6. **Avro fingerprint**: Include `default` in field fingerprint so schemas differing only in defaults are treated as distinct (fixes BACKWARD_TRANSITIVE/FULL_TRANSITIVE dedup bypass)
7. **Avro aliases**: Field and record alias matching for backward-compatible renaming
8. **JSON Schema test corrections**: Verified against Confluent — adding property to open content model IS incompatible (PROPERTY_ADDED_TO_OPEN_CONTENT_MODEL); closed model allows new properties
9. **Protobuf import tests**: Rewrote diff tests 37-43 with proper reference setup. All 7 pass.
10. **Protobuf structural import comparison**: Message-typed fields now compared by structure (field numbers + wire types) instead of fully-qualified name. This handles cross-import compatibility where the same message structure appears under different package names, matching Confluent's behavior.

### Checker Enhancements

**Avro** (`internal/compatibility/avro/checker.go`):
- Field alias matching: reader field aliases checked against writer fields
- Record alias matching: reader/writer aliases compared for name compatibility

**JSON Schema** (`internal/compatibility/jsonschema/checker.go`, ~1430 lines):
- 13 new check categories matching Confluent's JsonSchemaDiff
- $ref resolution, composition, dependencies, constraints, open/closed model

**Protobuf** (`internal/compatibility/protobuf/checker.go`, ~700 lines):
- Required field removal detection (proto2)
- Field-to-oneof: FIELD_MOVED_TO_EXISTING_ONEOF = incompatible
- Real vs synthetic oneof distinction (proto3 optional)
- Optional→repeated: only compatible for string/bytes/message
- Structural message comparison for cross-import compatibility (`areMessagesStructurallyCompatible`)

**Avro Parser** (`internal/schema/avro/parser.go`):
- Field fingerprint now includes `default` values to prevent dedup bypass

### Test Data Files

| File | Cases | Source |
|------|-------|--------|
| `tests/bdd/testdata/protobuf-diff-schema-examples.json` | 43 | Confluent protobuf-provider |
| `tests/bdd/testdata/jsonschema-diff-draft07.json` | 104 | Confluent json-schema-provider |
| `tests/bdd/testdata/jsonschema-diff-draft2020.json` | 101 | Confluent json-schema-provider |
| `tests/bdd/testdata/jsonschema-combined-draft07.json` | 28 | Confluent json-schema-provider |
| `tests/bdd/testdata/jsonschema-combined-draft2020.json` | 18 | Confluent json-schema-provider |

---

## Plan Reference

Full 8-phase plan at: `/Users/johnny/.claude/plans/generic-spinning-rossum.md`

Source document: `CONFLUENT-bdd-test-scenarios.md` (~555 scenarios across 35 sections)

## Running Tests

```bash
# In-process (fast, memory backend)
go test -tags bdd -v -count=1 -timeout 10m ./tests/bdd/...

# Docker mode (memory backend, includes @operational)
BDD_BACKEND=memory go test -tags bdd -v -count=1 -timeout 15m ./tests/bdd/...

# Against Confluent
podman compose -f tests/bdd/docker-compose.confluent.yml up -d --wait
BDD_BACKEND=confluent BDD_REGISTRY_URL=http://localhost:18081 go test -tags bdd -v -count=1 -timeout 25m ./tests/bdd/...
```
