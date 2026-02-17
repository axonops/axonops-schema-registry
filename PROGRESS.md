# Issue #264: Multi-Tenant Context Support — Progress Tracker

**Branch:** `feature/contextsupport`
**Issue:** https://github.com/axonops/axonops-schema-registry/issues/264
**Plan:** `/Users/johnny/.claude/plans/eventual-plotting-garden.md`
**Last Updated:** 2026-02-17

## Overall Status

| Phase | Description | Status | Notes |
|-------|-------------|--------|-------|
| 1 | Storage interface (`registryCtx` param) | IN PROGRESS | Interface updated, need memory store |
| 2 | Memory store restructure | NOT STARTED | Per-context `contextStore` |
| 3 | Registry layer (`registryCtx` plumbing) | NOT STARTED | |
| 4 | Context format fix + middleware + routing | NOT STARTED | `:.ctx:subj` format |
| 5 | Handler changes (context-aware resolution) | NOT STARTED | |
| 6 | Supporting (RBAC, metrics, OpenAPI test) | NOT STARTED | |
| 7 | PostgreSQL backend | NOT STARTED | Migrations 17-24 |
| 8 | MySQL backend | NOT STARTED | |
| 9 | Cassandra backend | NOT STARTED | Full table redesign |
| 10 | BDD tests (~72 scenarios) | NOT STARTED | |
| 11 | Unit tests | NOT STARTED | |
| 12 | OpenAPI spec | NOT STARTED | |
| 13 | Documentation | NOT STARTED | |

## Current Session Work Log

### Phase 1: Storage Interface
- [x] Updated `internal/storage/storage.go` — added `registryCtx string` param to all 28+ methods
- [x] Added `ListContexts(ctx context.Context) ([]string, error)` method
- [x] AuthStorage unchanged (global scope)
- [ ] Memory store needs updating to match new interface (Phase 2)

### Phase 2: Memory Store
- [ ] Create `contextStore` struct
- [ ] Restructure `Store` around `map[string]*contextStore`
- [ ] Per-context ID sequences
- [ ] Per-context fingerprint dedup
- [ ] `getOrCreateContext()` lazy init
- [ ] Default context `"."` always initialized
- [ ] `ListContexts()` returns sorted names
- [ ] Auth maps remain global

## Files Modified

- `internal/storage/storage.go` — Interface updated with `registryCtx`

## Key Design Decisions

1. **Schema IDs are per-context** — same ID in different contexts = different schemas
2. **Subject format**: `:.contextname:subject` (Confluent-compatible, NOT `:.contextname.:subject`)
3. **Default context**: `"."` everywhere (storage, registry, handlers, database)
4. **No backwards compatibility** — DB schema changes are breaking (user approved)
5. **"Global" config/mode is per-context** — applies to all subjects within a context
