# Resume Session Guide — Issue #267 MCP Server

**Branch:** `feature/mcp`
**Issue:** #267 — MCP Server for AI-Assisted Schema Management
**Last Updated:** 2026-03-06 (Issue #287 complete, CI 37/37 green)

## Current Status

**All phases COMPLETE — committed and pushed, CI fully green (37/37 jobs)**
**Latest commit:** `5d8ec48` — encryption workflow CI fix
**Issue #287:** Complete — comprehensive format migration guidance with all 6 format-pair type mappings

All code builds, all unit tests pass (`go build ./...`, `go vet ./...`, `go test ./internal/...`).

## Verified Totals (from source code audit)

| Component | Count | Location |
|-----------|-------|----------|
| MCP Tools | 105 | 12 files (`internal/mcp/tools*.go`) |
| MCP Resources | 47 (25 static + 22 templated) | `resources.go` + `glossary.go` |
| MCP Prompts | 33 | `internal/mcp/prompts.go` |
| Permission Scopes | 14 | `internal/mcp/permissions.go` |
| Permission Presets | 5 | readonly, developer, operator, admin, full |
| Server Instructions | set | `internal/mcp/server.go` (16 glossary URIs + 5 critical rules) |
| REST Analysis Endpoints | 26 | `internal/api/handlers/analysis.go` |
| MCP Unit tests | 208 | 8 test files |
| MCP BDD feature files | 43 | `tests/bdd/features/mcp/` |
| MCP BDD scenarios | 384 | across 43 files |
| REST Analysis BDD files | 9 | `tests/bdd/features/rest_*.feature` |
| REST Analysis BDD scenarios | 113 | across 9 files |
| **Total BDD scenarios** | **2608** | **across 178 feature files** |
| CI jobs | 37 | `.github/workflows/ci.yaml` |
| Makefile test targets | 22 | `Makefile` |

## Recent Work (2026-03-06)

| Phase | Description | Commit |
|-------|-------------|--------|
| 37 | Developer docs overhaul (CLAUDE.md, development.md, testing.md) | `6cdd230` |
| 38 | Issue #287: migration prompt expansion + BDD fixes (8 workflow files) | `7ece464` |
| 39 | gofmt CI fixes (permissions.go, metrics.go) | `ff62a67`+`01816d1` |
| 40 | MCP reference with rendered prompt content in `<details>` blocks | `c512268` |
| 41 | Encryption workflow CI fix (remove @kms, replace test_kek, explicit versions) | `5d8ec48` |

## Key Files

1. `PROGRESS.md` — Full phase tracker with commit hashes and exact tool/resource/prompt lists
2. `Makefile` — 22 test targets with CI awareness, aligned with CI pipeline
3. `internal/mcp/server.go` — Server struct, `New()`, `Start()`, options pattern, server instructions
4. `internal/mcp/content/` — Embedded `.md` files (16 glossary + 27 prompts) via `embed.FS`
5. `internal/mcp/glossary.go` — 16 glossary resource handlers (read from `content.GlossaryFS`)
6. `internal/mcp/prompts.go` — 33 prompt handlers (read from `content.PromptsFS`)
7. `internal/mcp/permissions.go` — 14 scopes, 5 presets, tool-to-scope mapping
8. `internal/mcp/tools.go` — Tool registration, `addToolIfAllowed`, `instrumentedHandler`, `resolveContext`
9. `internal/api/handlers/analysis.go` — 26 REST analysis endpoint handlers
10. `internal/analysis/` — Shared analysis package (field extraction, quality, fuzzy matching)
11. `internal/auth/rbac.go` — RBAC permissions with deny-by-default model
12. `tests/bdd/features/mcp/` — MCP BDD tests (379 scenarios across 43 files)
13. `tests/bdd/steps/mcp_steps.go` — MCP BDD step definitions with `$variable` resolution
14. `.github/workflows/ci.yaml` — 37 CI jobs, 17 using Makefile targets

## Running Tests (use Makefile targets — same as CI)

```bash
# Unit tests
make test-unit

# BDD functional (in-process, memory)
make test-bdd-functional

# BDD with Docker Compose
make test-bdd BACKEND=memory|postgres|mysql|cassandra|confluent|all

# BDD with real DB (in-process)
make test-bdd-db BACKEND=postgres|mysql|cassandra|all

# BDD auth with real DB
make test-bdd-auth BACKEND=postgres|mysql|cassandra|all

# BDD KMS (Vault + OpenBao)
make test-bdd-kms BACKEND=memory|postgres|mysql|cassandra|all

# All tests
make test

# See all available targets
make help
```
