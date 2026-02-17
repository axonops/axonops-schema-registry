# Issue #264: Multi-Tenant Context Support — Progress Tracker

**Branch:** `feature/contextsupport`
**Issue:** https://github.com/axonops/axonops-schema-registry/issues/264
**Plan:** `/Users/johnny/.claude/plans/eventual-plotting-garden.md`
**Last Updated:** 2026-02-17

## Overall Status

| Phase | Description | Status | Notes |
|-------|-------------|--------|-------|
| 1 | Storage interface (`registryCtx` param) | DONE | All 30 methods verified |
| 2 | Memory store restructure | DONE | Full per-context isolation, all methods use registryCtx |
| 3 | Registry layer (`registryCtx` plumbing) | DONE | All 28 methods forward registryCtx to storage |
| 4 | Context format fix + middleware + routing | DONE | `:.ctx:subj` format, middleware, `mountRegistryRoutes` |
| 5 | Handler changes (context-aware resolution) | DONE | All 22 handlers verified |
| 6 | Supporting (RBAC, metrics, OpenAPI test) | PARTIAL | OpenAPI test done; RBAC + metrics NOT checked |
| 7 | PostgreSQL backend | STUB ONLY | Signatures updated, but SQL queries unchanged, no migrations |
| 8 | MySQL backend | STUB ONLY | Signatures updated, but SQL queries unchanged, no migrations |
| 9 | Cassandra backend | STUB ONLY | Signatures updated, but CQL queries unchanged, no migrations |
| 10 | BDD tests (~72 scenarios) | NOT STARTED | |
| 11 | Unit tests | NOT STARTED | |
| 12 | OpenAPI spec | NOT STARTED | |
| 13 | Documentation | NOT STARTED | |

## What's Actually Working

The **memory store** is the only backend with real context support. The full flow works:
- API request → middleware extracts context → handler resolves context → registry passes through → memory store selects correct `contextStore`
- Per-context schema IDs, subjects, configs, modes, fingerprints
- Default context `"."` auto-initialized
- `ListContexts()` returns sorted context names from actual data

## What's NOT Working

**PostgreSQL, MySQL, Cassandra** — `registryCtx` parameter was added to all method signatures (compiles against interface), but:
- No SQL/CQL queries reference `registry_ctx` — parameter is silently discarded
- No database tables have a `registry_ctx` column
- No migrations exist (plan calls for PostgreSQL migrations 17-24)
- `ListContexts()` is hardcoded to `return []string{"."}, nil`
- All data lives in one flat namespace regardless of context

## Completed Work (Verified by Audit)

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
- [x] `NormalizeContextName`, `IsValidContextName`, `WithRegistryContext`, `RegistryContextFromRequest` all working
- [x] `DefaultContext = "."`

### Phase 5: Handler Changes
- [x] `internal/api/handlers/context_helpers.go` — `getRegistryContext`, `resolveSubjectAndContext`
- [x] All 22 registry-calling handlers use helpers and pass `registryCtx` through

### Phase 6: Supporting (Partial)
- [x] `internal/api/openapi_test.go` — context routes excluded from sync test
- [ ] `internal/auth/rbac.go` — `normalizePathForRBAC` NOT verified
- [ ] `internal/metrics/metrics.go` — path normalization NOT verified

### Phases 7-9: Storage Backends (Stubs Only)
- [x] PostgreSQL `store.go` — signatures updated (compiles)
- [x] MySQL `store.go` — signatures updated (compiles)
- [x] Cassandra `store.go` — signatures updated (compiles)
- [ ] PostgreSQL `migrations.go` — NO new migrations (need 17-24)
- [ ] MySQL `migrations.go` — NO new migrations
- [ ] Cassandra `migrations.go` — NO new migrations (need full table redesign)
- [ ] PostgreSQL SQL queries — NOT updated to filter by `registry_ctx`
- [ ] MySQL SQL queries — NOT updated to filter by `registry_ctx`
- [ ] Cassandra CQL queries — NOT updated to filter by `registry_ctx`
- [ ] All three `ListContexts()` hardcoded to `return []string{"."}, nil`

### Conformance Tests
- [x] All 5 conformance test files updated with `registryCtx` parameter (`"."`)

## Compilation & Test Status

- `go build ./...` — PASSES
- `go test ./internal/...` — ALL PASS
- Tests only exercise memory store (no DB backend tests without running databases)

## Key Design Decisions

1. **Schema IDs are per-context** — same ID in different contexts = different schemas
2. **Subject format**: `:.contextname:subject` (Confluent-compatible, NOT `:.contextname.:subject`)
3. **Default context**: `"."` everywhere (storage, registry, handlers, database)
4. **No backwards compatibility** — DB schema changes are breaking (user approved)
5. **"Global" config/mode is per-context** — applies to all subjects within a context
