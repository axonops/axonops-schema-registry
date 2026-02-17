# Resume Session Guide — Issue #264 Context Support

**Branch:** `feature/contextsupport`
**Issue:** #264 — Multi-Tenant Context Support (Confluent-compatible)

## How to Resume

If this session crashes, the next Claude Code session should:

1. Read this file first
2. Read `PROGRESS.md` for current status
3. Read the plan at the plan file path below
4. Continue from the last incomplete phase

## Plan File

The full implementation plan is at: `/Users/johnny/.claude/plans/eventual-plotting-garden.md`

If that file is missing, the plan is also summarized in PROGRESS.md and the key points are:
- 13-phase implementation adding `registryCtx` parameter throughout the codebase
- Confluent API compatibility for schema contexts (multi-tenancy)
- Per-context schema IDs, subjects, config, modes
- Subject format: `:.contextname:subject`
- Default context: `"."`

## Current State (UPDATE THIS AS YOU GO)

**Last completed step:** Phase 10 (BDD tests) — 78 scenarios across 7 feature files, all 1451 BDD scenarios pass.

**Next step:** Phase 11 — Unit tests for new context functionality

**Compilation status:** COMPILING — `go build ./...` passes

**Test status:** ALL PASS — `go test ./internal/...` passes, `go test -tags bdd ./tests/bdd/...` passes (1451 scenarios)

**Files modified so far:**
- `internal/storage/storage.go` — Added `registryCtx string` to all Storage interface methods
- `internal/storage/memory/store.go` — Per-context restructure with `contextStore`
- `internal/storage/memory/store_test.go` — Tests updated
- `internal/storage/postgres/store.go` — COMPLETE: all SQL scoped by registry_ctx
- `internal/storage/postgres/migrations.go` — COMPLETE: migrations 17-39
- `internal/storage/mysql/store.go` — COMPLETE: all SQL scoped by registry_ctx
- `internal/storage/mysql/migrations.go` — COMPLETE: migrations 22-40
- `internal/storage/cassandra/store.go` — COMPLETE: all CQL scoped by registry_ctx, per-context block IDs
- `internal/storage/cassandra/migrations.go` — COMPLETE: all tables with registry_ctx in PK, SAI indexes, backfill
- `internal/registry/registry.go` — registryCtx plumbing
- `internal/registry/registry_test.go` — Tests updated
- `internal/context/context.go` — Format fix to `:.ctx:subj`
- `internal/context/context_test.go` — Tests updated
- `internal/api/server.go` — `mountRegistryRoutes` + `/contexts/{context}` route group
- `internal/api/context_middleware.go` — NEW: context extraction middleware
- `internal/api/handlers/handlers.go` — Context-aware handler resolution
- `internal/api/handlers/context_helpers.go` — NEW: helper functions
- `internal/api/openapi_test.go` — Exclude context routes from sync test
- `internal/auth/rbac.go` — RBAC path normalization for context routes
- `internal/auth/rbac_test.go` — Tests for context-scoped RBAC
- `internal/metrics/metrics.go` — Metrics path normalization for context routes
- `internal/metrics/metrics_test.go` — Tests for context-scoped metrics
- `tests/storage/conformance/*.go` — All 5 conformance test files updated
- `tests/bdd/features/contexts*.feature` — 7 BDD feature files (78 scenarios)
- `tests/bdd/steps/schema_steps.go` — 2 new step definitions

**What still needs to be done:**
1. ~~BDD tests (~72 scenarios across 7 feature files)~~ DONE (78 scenarios)
2. Unit tests for new context functionality
3. OpenAPI spec documentation updates
4. Documentation updates

## Important Context

- User wants regular commits and pushes to `feature/contextsupport` branch
- User wants comprehensive BDD tests (~72 scenarios)
- No backwards compatibility needed for DB schemas
- Must match Confluent behavior exactly
- The `internal/context/` package has been updated with Confluent-compatible format
- PostgreSQL uses `ctx_id_alloc` table for per-context ID allocation (not schemas_id_seq)
- MySQL and Cassandra follow the same pattern
- JSON Schema backward compat: adding properties = incompatible, removing = compatible (opposite of Avro)
- POST `/subjects/.../versions` returns only `{"id": N}` (no version field)
- `GET /subjects` at root returns default context subjects only
