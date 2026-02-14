# Session Resume Guide — BDD Testing & Confluent Compatibility

**Last Updated:** 2026-02-14
**Branch:** `feature/testing`
**Base Branch:** `main`

---

## Current State — ALL NON-PENDING BDD TESTS PASS

**1283 passing BDD scenarios** (0 failures) across 67+ feature files.
- **1283 passing** (memory backend, in-process)
- **23 tagged `@pending-impl`** (tests for unimplemented features, excluded from CI)

### Phase Progress

| Phase | Status | Scenarios Added | Details |
|-------|--------|-----------------|---------|
| Phase 1: API BDD Tests | COMPLETE | 102 | Sections 1-13, 19-21 |
| Phase 2: Avro & Parsing BDD | COMPLETE | 86 | Sections 22-25, 32-34 |
| Phase 3: Protobuf Diff Tests | COMPLETE | 43 | Section 31 data-driven |
| Phase 4-5: JSON Schema Diff | COMPLETE | 251 | Sections 27-29 data-driven |
| Phase 6: JSON Schema Validation | COMPLETE | 40 | Section 30 reader/writer pairs |
| Phase 7: Feature Implementation | IN PROGRESS | — | JSON Schema + Protobuf checkers enhanced |
| Phase 8: Feature BDD Tests | NOT STARTED | — | Tests for Phase 7 features |

### Key Fixes This Session

1. **API error codes**: 40408 (subject config not found), 40409 (subject mode not found)
2. **Deletion behaviors**: `deleted=true` for GetVersion, config/mode cleanup on delete
3. **Pagination**: offset/limit for versions, schema IDs, subject IDs
4. **FORWARD_TRANSITIVE**: Fixed test data (verified against Confluent v8.1.1)
5. **Protobuf checker**: Required field removal, oneof moves (existing vs new), optional→repeated for length-delimited types, synthetic oneof handling
6. **All @pending-impl tags removed** from tests that now pass

### Remaining 23 `@pending-impl` Scenarios

| Category | Count | Details |
|----------|-------|---------|
| Protobuf imports | 13 | Proto import resolution (google.proto, custom imports) |
| JSON Schema validation | 5 | Record evolution, union compat, transitive chains |
| Avro gaps | 4 | Aliases (field/record), transitive mode dedup issue |
| Schema parsing | 1 | Avro alias compatibility in parsing |

### Checker Enhancements

**JSON Schema** (`internal/compatibility/jsonschema/checker.go`, ~1430 lines):
- 13 new check categories matching Confluent's JsonSchemaDiff
- $ref resolution, composition, dependencies, constraints, open/closed model

**Protobuf** (`internal/compatibility/protobuf/checker.go`, ~660 lines):
- Required field removal detection (proto2)
- Field-to-oneof: FIELD_MOVED_TO_EXISTING_ONEOF = incompatible
- Real vs synthetic oneof distinction (proto3 optional)
- Optional→repeated: only compatible for string/bytes/message

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
BDD_BACKEND=memory go test -tags bdd -v -count=1 -timeout 15m ./tests/bdd/...

# Including pending-impl scenarios (to see what fails)
BDD_BACKEND=memory BDD_TAGS="~@operational" go test -tags bdd -v -count=1 -timeout 15m ./tests/bdd/...

# Against Confluent
podman compose -f tests/bdd/docker-compose.confluent.yml up -d --wait
BDD_BACKEND=confluent BDD_REGISTRY_URL=http://localhost:18081 go test -tags bdd -v -count=1 -timeout 25m ./tests/bdd/...
```
