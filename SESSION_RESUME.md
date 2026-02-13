# Session Resume Guide — BDD Testing & Confluent Compatibility

**Last Updated:** 2026-02-13
**Branch:** `feature/testing`
**Base Branch:** `main`

---

## Current State — EXHAUSTIVE BDD TESTING IN PROGRESS

**1292 total BDD scenarios** across 67+ feature files.
- **1257 passing** (memory backend, in-process)
- **35 tagged `@pending-impl`** (tests for unimplemented features, excluded from CI)

### Phase Progress

| Phase | Status | Scenarios Added | Details |
|-------|--------|-----------------|---------|
| Phase 1: API BDD Tests | COMPLETE | 102 (88 pass, 14 pending) | Sections 1-13, 19-21 |
| Phase 2: Avro & Parsing BDD | COMPLETE | 86 (81 pass, 5 pending) | Sections 22-25, 32-34 |
| Phase 3: Protobuf Diff Tests | COMPLETE | 43 (29 pass, 14 pending) | Section 31 data-driven |
| Phase 4-5: JSON Schema Diff | COMPLETE | 251 (244 pass, 7 pending) | Sections 27-29 data-driven |
| Phase 6: JSON Schema Validation | COMPLETE | 40 (35 pass, 5 pending) | Section 30 reader/writer pairs |
| Phase 7: Feature Implementation | IN PROGRESS | — | JSON Schema checker enhanced |
| Phase 8: Feature BDD Tests | NOT STARTED | — | Tests for Phase 7 features |

### JSON Schema Checker Enhancement (Phase 4/7)

Major expansion of `internal/compatibility/jsonschema/checker.go` (~360 → ~1430 lines):

- **$ref resolution** — Resolves local `#/definitions/` and `#/$defs/` references
- **Implicit type detection** — Object/array by keywords without explicit `"type"`
- **String constraints** — minLength, maxLength, pattern
- **Numeric constraints** — minimum, maximum, exclusiveMin/Max, multipleOf
- **Composition** — oneOf/anyOf/allOf with recursive subschema checking
- **Dependencies** — dependentRequired, dependentSchemas (Draft-2020)
- **Tuple items** — items-as-array (Draft-07), prefixItems (Draft-2020)
- **Items as boolean** — Draft-2020 `items: true → false`
- **Const compatibility** — Value change detection
- **Property count** — maxProperties, minProperties
- **Not schema** — not keyword changes
- **patternProperties** — Covering pattern detection for removed properties
- **Boolean property schemas** — `true`/`false` property handling
- **Closed vs open model** — Correct property addition semantics

### Remaining 35 `@pending-impl` Failures (by category)

| Category | Count | Files | Details |
|----------|-------|-------|---------|
| Protobuf imports | 7 | protobuf_diff | google.proto not found |
| Protobuf checker | 6 | protobuf_diff | required fields, oneof, field labels |
| JSON Schema validation | 5 | jsonschema_validation | Record evolution, transitive chains |
| Config/Mode errors | 5 | config_exhaustive, mode_exhaustive | Error codes 40408/40409 |
| Avro gaps | 4 | avro_exhaustive | Aliases, transitive modes |
| Deletion | 3 | deletion_exhaustive | Soft-delete query, config cleanup |
| Pagination | 3 | pagination_exhaustive | offset/limit params |
| Error handling | 1 | error_handling_exhaustive | Version 0 validation |
| Schema parsing | 1 | schema_parsing_exhaustive | Avro alias compat |

### Files Modified (checker enhancement)

| File | Change |
|------|--------|
| `internal/compatibility/jsonschema/checker.go` | +1200 lines — 13 new check categories |
| `internal/compatibility/jsonschema/checker_test.go` | Updated 4 unit tests for Confluent semantics |
| `tests/bdd/steps/schema_steps.go` | Added variable-resolved assertion steps |
| `tests/bdd/bdd_test.go` | Added `~@pending-impl` to all tag filters |
| `tests/bdd/features/*.feature` | Removed @pending-impl from 77 now-passing scenarios |

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
go test -tags bdd -v -count=1 -timeout 15m ./tests/bdd/...

# Including pending-impl scenarios (to see what fails)
BDD_TAGS="~@operational" go test -tags bdd -v -count=1 -timeout 15m ./tests/bdd/...
```
