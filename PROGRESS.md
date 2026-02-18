# Issue #264: Multi-Tenant Context Support — Progress Tracker

**Branch:** `feature/contextsupport`
**Issue:** https://github.com/axonops/axonops-schema-registry/issues/264
**Plan:** `/Users/johnny/.claude/plans/eventual-plotting-garden.md`
**Last Updated:** 2026-02-18

## Overall Status: ALL PHASES COMPLETE

| Phase | Description | Status | Notes |
|-------|-------------|--------|-------|
| 1 | Storage interface (`registryCtx` param) | DONE | All 30 methods verified |
| 2 | Memory store restructure | DONE | Full per-context isolation, all methods use registryCtx |
| 3 | Registry layer (`registryCtx` plumbing) | DONE | All 28 methods forward registryCtx to storage |
| 4 | Context format fix + middleware + routing | DONE | `:.ctx:subj` format, middleware, `mountRegistryRoutes` |
| 5 | Handler changes (context-aware resolution) | DONE | All 22 handlers verified |
| 6 | Supporting (RBAC, metrics, OpenAPI test) | DONE | RBAC path normalization, metrics path normalization, OpenAPI test exclusion |
| 7 | PostgreSQL backend | DONE | 23 migrations (17-39), all SQL queries scoped by registry_ctx, per-context IDs via ctx_id_alloc |
| 8 | MySQL backend | DONE | Migrations 22-40, all SQL scoped by registry_ctx, ctx_id_alloc per-context IDs |
| 9 | Cassandra backend | DONE | All tables with registry_ctx in PK, SAI indexes, block-based per-context IDs, backfill |
| 10 | BDD tests (155 scenarios) | DONE | 155 scenarios across 13 feature files, all 1535 BDD scenarios pass |
| 11 | Unit tests (56 tests) | DONE | 56 new tests across 5 files (memory store, registry, handlers, middleware, server) |
| 12 | OpenAPI spec | DONE | Updated GET /contexts description, added Contexts tag, updated examples |
| 13 | Documentation | DONE | docs/contexts.md (341 lines), README.md updated |

## What's Working

### Memory Store (Fully Working)
- API request → middleware extracts context → handler resolves context → registry passes through → memory store selects correct `contextStore`
- Per-context schema IDs, subjects, configs, modes, fingerprints
- Default context `"."` auto-initialized
- `ListContexts()` returns sorted context names from actual data

### PostgreSQL Backend (Fully Updated, Needs Integration Testing)
- 23 new migrations (17-39) adding `registry_ctx` to all tables
- Context-scoped unique indexes on schemas, fingerprints, configs, modes
- `ctx_id_alloc` table for per-context ID sequences (replaces schemas_id_seq)
- `contexts` tracking table with default `.` seeded
- All prepared statements scoped by `registry_ctx`
- All method bodies pass `registryCtx` to queries and helpers
- `ListContexts()` queries the `contexts` table
- `ImportSchema` advances ctx_id_alloc when importing specific IDs

### MySQL Backend (Fully Updated, Needs Integration Testing)
- 19 new migrations (22-40) adding `registry_ctx` to all tables
- Context-scoped unique indexes on schemas, fingerprints, configs, modes
- `ctx_id_alloc` table for per-context ID sequences
- `contexts` tracking table with default `.` seeded
- All prepared statements scoped by `registry_ctx`
- `ListContexts()` queries the `contexts` table
- `ImportSchema` advances ctx_id_alloc when importing specific IDs

### Cassandra Backend (Fully Updated, Needs Integration Testing)
- All tables recreated with `registry_ctx` in partition keys
- SAI index on `subject_latest.registry_ctx` for filtered queries
- Block-based per-context ID allocation via `idAllocator` with `contextIDBlock` map
- `contexts` table for tracking all registry contexts
- Backfill functions for production upgrades from pre-context schemas
- `ensureContext()` helper for automatic context registration
- `ListContexts()` queries the `contexts` table

### RBAC & Metrics
- `normalizePathForRBAC()` strips `/contexts/{context}` prefix for permission matching
- `normalizePath()` in metrics handles context-scoped routes (prevents label explosion)
- Both have comprehensive test coverage

## Completed Work (Verified)

### Phase 1: Storage Interface
- [x] `internal/storage/storage.go` — `registryCtx string` on all 30 methods
- [x] `ListContexts(ctx context.Context) ([]string, error)` added
- [x] AuthStorage unchanged (global scope)

### Phase 2: Memory Store
- [x] `contextStore` struct with per-context data
- [x] `Store` restructured around `map[string]*contextStore`
- [x] `getOrCreateContext()` lazy init + `getContext()` read-only variant
- [x] Default context `"."` initialized in `NewStore()`
- [x] `ListContexts()` returns sorted names
- [x] Auth maps global (not per-context)
- [x] ALL methods use `registryCtx` to select correct context store (verified)

### Phase 3: Registry Layer
- [x] All 28 public methods accept `registryCtx`
- [x] All storage calls forward `registryCtx` (verified with code snippets)
- [x] Private helpers also forward `registryCtx`

### Phase 4: Context Middleware & Routing
- [x] `internal/api/context_middleware.go` — extracts/normalizes/validates context from URL
- [x] `internal/api/server.go` — `mountRegistryRoutes()` called at root and `/contexts/{context}`
- [x] `internal/context/context.go` — Confluent-compatible `:.ctx:subj` format
- [x] `NormalizeContextName`, `IsValidContextName`, `WithRegistryContext`, `RegistryContextFromRequest`
- [x] `DefaultContext = "."`

### Phase 5: Handler Changes
- [x] `internal/api/handlers/context_helpers.go` — `getRegistryContext`, `resolveSubjectAndContext`
- [x] All 22 registry-calling handlers use helpers and pass `registryCtx` through

### Phase 6: Supporting
- [x] `internal/api/openapi_test.go` — context routes excluded from sync test
- [x] `internal/auth/rbac.go` — `normalizePathForRBAC` with 17 test cases
- [x] `internal/auth/rbac_test.go` — `TestAuthorizeEndpoint_ContextScopedRoutes` (3 test cases)
- [x] `internal/metrics/metrics.go` — `normalizePath()` handles `/contexts/{context}/...`
- [x] `internal/metrics/metrics_test.go` — 12 context-scoped test cases

### Phase 7: PostgreSQL Backend
- [x] `internal/storage/postgres/migrations.go` — Migrations 17-39
- [x] `internal/storage/postgres/store.go` — All SQL queries scoped by `registry_ctx`
- [x] Prepared statements updated (schema, config, mode, references)
- [x] `globalSchemaID`/`globalSchemaIDTx` take `registryCtx`
- [x] `ensureContext()` helper for context tracking
- [x] `cleanupOrphanedFingerprint` context-aware
- [x] `NextID`/`SetNextID` use `ctx_id_alloc` table
- [x] `GetMaxSchemaID` queries `schema_fingerprints` per context
- [x] `ImportSchema` fully context-aware with ctx_id_alloc advancement
- [x] `ListContexts` queries `contexts` table
- [x] `DeleteGlobalConfig` uses context-scoped ON CONFLICT

### Phase 8: MySQL Backend
- [x] `internal/storage/mysql/migrations.go` — Migrations 22-40 (registry_ctx columns, indexes, ctx_id_alloc, contexts table)
- [x] `internal/storage/mysql/store.go` — All SQL queries scoped by `registry_ctx`
- [x] `ensureContext()` helper for context tracking
- [x] `NextID`/`SetNextID` use `ctx_id_alloc` table
- [x] `ImportSchema` fully context-aware
- [x] `ListContexts` queries `contexts` table

### Phase 9: Cassandra Backend
- [x] `internal/storage/cassandra/migrations.go` — All tables with `registry_ctx` in partition keys, SAI indexes, backfill functions
- [x] `internal/storage/cassandra/store.go` — All CQL queries scoped by `registryCtx`
- [x] Block-based per-context ID allocation via `idAllocator` with `contextIDBlock`
- [x] `ensureContext()` helper for context tracking
- [x] Backfill functions for production upgrades from pre-context schema
- [x] `ListContexts` queries `contexts` table
- [x] `contexts` table initialized with default context `"."`

### Conformance Tests
- [x] All 5 conformance test files updated with `registryCtx` parameter (`"."`)

### Phase 10: BDD Tests (155 scenarios across 13 feature files)
- [x] `contexts.feature` — 10 scenarios: core behavior, context listing, implicit creation, sorted list, case-sensitivity, basic isolation, delete
- [x] `contexts_isolation.feature` — 12 scenarios: schema ID isolation, version isolation, subject listing scoped, delete isolation, permanent delete isolation, lookup isolation, soft-delete isolation, fingerprint dedup per-context, default vs named context isolation
- [x] `contexts_operations.feature` — 15 scenarios: register/retrieve, latest version, list versions, lookup, soft delete, permanent delete, version delete, compatibility check, subjects by schema ID, idempotent re-registration, 404 cases, schema types
- [x] `contexts_config_mode.feature` — 9 scenarios: set/get/delete config, set/get/delete mode, config isolation between contexts, backward enforcement, incompatible rejection, NONE allows any
- [x] `contexts_schema_types.feature` — 8 scenarios: Avro/Protobuf/JSON Schema registration and compat in contexts, mixed types, cross-context same type
- [x] `contexts_edge_cases.feature` — 11 scenarios: valid context names (alphanumeric, hyphens, dots, underscores), schema dedup, 404 on non-existent context subjects, sequential IDs
- [x] `contexts_url_routing.feature` — 13 scenarios: register/retrieve/latest/list/lookup/delete via URL prefix, config/mode via URL prefix, compat via URL prefix, schema by ID via URL prefix, cross-validation (Confluent-compatible, `@axonops-only` removed)
- [x] `contexts_schema_evolution.feature` — 11 scenarios: multi-version Avro evolution, BACKWARD/FORWARD/FULL/TRANSITIVE enforcement, version deletion, independent evolution paths, idempotent re-registration, compat checks against specific/latest/all versions
- [x] `contexts_references.feature` — 8 scenarios: Avro/Protobuf references within contexts, referencedby endpoint, cross-context reference isolation, delete blocked by references, raw schema endpoint, schema version detail with references
- [x] `contexts_advanced_api.feature` — 17 scenarios: raw schema endpoint, lookup, cross-context lookup isolation, deleted subjects listing, fingerprint dedup, latest version tracking, version listing, subject scoping, schema types, compatibility checks, 404 errors
- [x] `contexts_config_mode_advanced.feature` — 12 scenarios: per-context config enforcement, per-subject override, config delete fallback to server default, READONLY mode enforcement, READWRITE after mode change, mode isolation, config leak prevention, global mode on default context, per-subject mode override, mode delete restore, compat check respects config, BACKWARD enforcement
- [x] `contexts_real_world.feature` — 12 scenarios: multi-team isolation, environment separation (dev/staging/prod), schema linking simulation, hierarchical context naming, default context coexistence, large context count (10+), migration workflow, context cleanup lifecycle, different evolution speed, config/mode isolation real-world
- [x] `contexts_validation.feature` — 17 scenarios: invalid context names (!, space, @, #, %), valid characters, default context behavior, implicit context creation, error conditions (404 for subjects/versions/config/mode/compat)
- [x] 2 new step definitions in `schema_steps.go`: variable NOT-equal comparison, array NOT-contain
- [x] All 1535 BDD scenarios pass (155 context + 1380 existing)
- [x] All 155 context scenarios are Confluent-compatible (`@functional` only, no `@axonops-only`)

### Phase 11: Unit Tests (56 tests across 5 files)
- [x] `internal/storage/memory/store_test.go` — 22 tests: schema/subject/config/mode isolation, per-context IDs, fingerprint dedup, ListContexts, default context, delete isolation, GetSchemasBySubject, DeleteSchema, SubjectExists, DeleteConfig, DeleteMode, GetSchemaByFingerprint, GetSchemaByGlobalFingerprint, GetReferencedBy, GetSubjectsBySchemaID, GetVersionsBySchemaID, ListSchemas, DeleteGlobalConfig, GetMaxSchemaID
- [x] `internal/registry/registry_test.go` — 10 tests: RegisterSchema independent contexts, GetSchemaByID/ListSubjects/LookupSchema/CheckCompatibility context-scoped, config/mode/delete isolation, ListContexts, GetSchemasBySubject context-scoped
- [x] `internal/api/handlers/context_helpers_test.go` — 8 tests: getRegistryContext default/middleware, resolveSubjectAndContext plain/qualified/URL/override
- [x] `internal/api/context_middleware_test.go` — 8 tests: set context, normalize, default mapping, invalid rejection, dash/underscore, dot-prefix
- [x] `internal/api/server_test.go` — 6 tests: GET /contexts default/after-registration, qualified subject register/retrieve, subject/schema ID isolation
- [x] Plus 2 extra tests in server_test.go from main merge (health check)
- [x] All 56 new context tests pass with zero regressions

## Compilation & Test Status

- `go build ./...` — PASSES
- `go test ./internal/...` — ALL PASS (56 new context unit tests)
- `go test -tags bdd ./tests/bdd/...` — ALL 1535 SCENARIOS PASS (155 context BDD scenarios)
- Tests only exercise memory store (no DB backend tests without running databases)

## Unit Test Coverage (context-related packages)

| Package | Coverage |
|---------|----------|
| `internal/context/` | 100% |
| `internal/api/` | 78% |
| `internal/registry/` | 63.1% |
| `internal/storage/memory/` | 62.0% |

Key context functions all at 100%: `WithRegistryContext`, `RegistryContextFromRequest`, `ResolveSubject`, `FormatSubject`, `IsValidContextName`, `NormalizeContextName`, `contextExtractionMiddleware`, `newContextStore`, `getOrCreateContext`, `getContext`, `ListContexts`, `SubjectExists`, `GetMaxSchemaID`

## Behavioral Findings

1. **Global config/mode is per-context, NOT truly global** — `PUT /config` and `PUT /mode` at root only affect the default context `"."`. Named contexts do NOT inherit root-level global config/mode. When a named context has no per-subject config and no per-context global config, the fallback is the server startup default (`BACKWARD` for compat, `READWRITE` for mode). This matches Confluent's statement: "there is no global compatibility across contexts."

2. **API returns plain subject names** — `GET /subjects/:.ctx:subj/versions/1` returns `"subject": "subj"` (plain), not `"subject": ":.ctx:subj"` (qualified). The context is separate from the subject name.

3. **Compat check on non-existent subject with "latest" returns is_compatible:true** — When there are no existing versions, there's nothing to be incompatible with (vacuous truth). Using a specific version (e.g., `/versions/1`) correctly returns 404.

## Git Commits

| Commit | Description |
|--------|-------------|
| `0375b01` | feat(contexts): add registryCtx parameter to storage interface, memory store, and registry layer |
| `95ef01a` | feat(contexts): add context middleware, routing, and handler plumbing |
| `7d009d3` | fix(contexts): add RBAC and metrics support for context-scoped routes |
| `057d2ec` | feat(contexts): add registryCtx parameter stubs to SQL/Cassandra backends |
| `3667cb2` | test(contexts): update all tests with registryCtx parameter |
| `987d9de` | docs: update progress tracker with audited phase status |
| `b6e7807` | feat(contexts): complete PostgreSQL backend context support |
| `b771bcf` | feat(contexts): complete MySQL backend context support |
| `64a1059` | feat(contexts): complete Cassandra backend context support |
| `609dc4d` | test(contexts): add 78 BDD scenarios for multi-tenant context support |
| `ba37496` | test(contexts): add 41 unit tests for context isolation across all layers |
| `ec7aa9f` | docs(contexts): update OpenAPI spec with context support documentation |
| `6dedc3c` | docs(contexts): add comprehensive context documentation and update README |
| `432c6b4` | Merge remote-tracking branch 'origin/main' into feature/contextsupport |
| `4af351c` | test(contexts): add 77 BDD scenarios for comprehensive context coverage |
| `6cdac44` | fix(contexts): correct BDD scenarios, add 15 unit tests, enable URL routing for Confluent |

## Key Design Decisions

1. **Schema IDs are per-context** — same ID in different contexts = different schemas
2. **Subject format**: `:.contextname:subject` (Confluent-compatible, NOT `:.contextname.:subject`)
3. **Default context**: `"."` everywhere (storage, registry, handlers, database)
4. **No backwards compatibility** — DB schema changes are breaking (user approved)
5. **"Global" config/mode is per-context** — applies to all subjects within a context
6. **Per-context ID allocation** — `ctx_id_alloc` table replaces global sequences
