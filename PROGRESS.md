# Issue #267: MCP Server for AI-Assisted Schema Management — Progress Tracker

**Branch:** `feature/mcp`
**Issue:** https://github.com/axonops/axonops-schema-registry/issues/267
**Last Updated:** 2026-03-06 (Issue #287 complete, CI 37/37 green)

## Overall Status: IMPLEMENTATION COMPLETE — ALL COMMITTED, CI GREEN

All MCP server features are implemented, tested, and committed.
Test suite hardening complete: all BDD tests run against database backends, CI has 37 jobs.
Issue #287 complete: comprehensive format migration guidance with all 6 format-pair type mappings.
MCP reference docs include rendered prompt content. All 37 CI jobs green on `5d8ec48`.

## Phase Tracker

| Phase | Description | Status | Commit | Notes |
|-------|-------------|--------|--------|-------|
| Analysis | Technical review, MCP spec research, SDK selection | DONE | — | Posted as issue comment |
| 0 | Code restructuring (extract shared logic) | DONE | `6194d5e` | Merged to main |
| 1 | MCP scaffolding + 3 starter tools + BDD infra | DONE | `c703fe9` | `server.go`, `tools.go`, `mcp_steps.go` |
| 2 | Core schema read tools (13 tools) | DONE | `ca3018d` | `tools_schema.go` |
| 3 | Schema write tools (4 tools) | DONE | `e68a63f` | `tools_write.go` |
| 4 | Config & mode tools (6 tools) | DONE | `04e4d91` | `tools_config.go` |
| 5 | Context management (2 tools) | DONE | `638b48f` | `tools_context.go` |
| 6 | KEK/DEK tools (13 tools) | DONE | `45ca908` | `tools_dek.go` |
| 7 | Exporter tools (11 tools) | DONE | `5ab0879` | `tools_exporter.go` |
| 8 | Metadata & data contracts (12 tools) | DONE | `43510a9` + `047a8c5` | `tools_metadata.go` |
| 9 | Admin tools (15 tools) | DONE | `0f18cd9` + `251bbcf` | `tools_admin.go` |
| 10 | Observability (metrics, audit, logging) | DONE | `2bacf45` + `67acb1e` | `instrumentedHandler` in `tools.go` |
| 11 | MCP Resources (15 resources) | DONE | `7f3e485` | `resources.go` — 5 static + 10 templated |
| 12 | MCP Prompts (12 prompts) | DONE | `7f3e485` | `prompts.go` — context-aware guidance |
| 13 | Security & guardrails | DONE | `7f3e485` | `middleware.go`, `addToolIfAllowed` in `tools.go` |
| 14 | Validation & export tools (10 tools) | DONE | `2c6848a` | `tools_validation.go`, `ValidateSchema`/`NormalizeSchema` in registry |
| 15 | Comparison & search tools (6 tools) | DONE | `2c6848a` | `tools_comparison.go`, `schema_utils.go` field extraction |
| 16 | AI intelligence tools (9 tools) | DONE | `2c6848a` | `tools_intelligence.go`, `fuzzy.go`, `quality.go` |
| 16b | Origin validation (P0) | DONE | `74c9c34` | `middleware.go` — allowlist, wildcard, 403 |
| 16c | Two-phase confirmations (P1) | DONE | `74c9c34` | `confirmation.go` — 8 tools, dry_run/confirm_token flow |
| 16d | Security hardening | DONE | `3fa2148` | Localhost binding, wildcard origins, log_schemas |
| 16e | list_subjects pattern filtering | DONE | `48ba7d4` | Regex `pattern` parameter |
| 17 | Shared analysis package | DONE | `8200d51` | `internal/analysis/` extracted from `internal/mcp/` |
| 18 | REST analysis endpoints (26) | DONE | `efedc32` | `internal/api/handlers/analysis.go`, OpenAPI spec |
| 19 | REST analysis BDD tests (115) | DONE | `77738ed` | 9 REST + 1 MCP feature files |
| 20 | Glossary resources (10) + prompts (8) + server instructions | DONE | `ef273ef` | `glossary.go`, 8 new prompts, `ServerOptions{Instructions}` |
| 20b | Glossary + prompts tests | DONE | `35487db` | 19 unit tests, 18 BDD scenarios |
| 20c | Glossary + prompts docs | DONE | `af19605` | `mcp.md`, `mcp-reference.md` |
| 21 | Test suite hardening — DB backend BDD | DONE | `8f64223` | All BDD tests run against PG/MySQL/Cassandra |
| 21b | CI jobs for DB backend BDD | DONE | `aee5cfd` | 6 new CI jobs (3 general-DB + 3 auth-DB) |
| 21c | Fingerprint dedup + Cassandra concurrency | DONE | `de92f5f` | 30%→10% tolerance, Cassandra 1/30→1/6 load |
| 21d | Rate limiting BDD tests | DONE | `9f1564a` | 2 step defs, removed `@pending-impl` tag |
| 21e | CI reliability fixes | DONE | `f55d7c4`→`a978dbd` | MySQL dedup race, Cassandra ID cache, dynamic IDs, RBAC permissions |
| 22 | CI-Makefile alignment | DONE | `22e8840` | Fix timeouts, unit scope, add CI awareness |
| 22b | New BDD Makefile targets | DONE | `22e8840` | `test-bdd-functional`, `test-bdd-db`, `test-bdd-auth`, `test-bdd-kms` |
| 22c | CI migration to Makefile | DONE | `3b58d87` | 17 CI jobs now use `make` targets |
| 22d | Conformance CI optimization | DONE | `30f3692` | Remove unnecessary conformance→integration deps |
| 22e | Code quality fixes | DONE | `302c03e` | Fix json.Marshal error handling in mcp_steps.go |
| 23 | Multi-tenant context parameter | DONE | `1bea1f8` | `resolveContext()` helper, Context field on 58 input structs, 4 BDD scenarios |
| 24 | Content extraction to embed.FS | DONE | `c5b2370` | 10 glossary + 27 prompt `.md` files, glossary.go 1213→124 lines, prompts.go 1610→744 lines |
| 25 | MCP API docs regeneration | DONE | `1ec7ed6` | Regenerated mcp-reference.md with context parameter |
| 26 | Context support in resources & prompts | DONE | `a1f9341` | 10 new resource templates (31→41), 7 prompts gain context arg, 9 unit + 18 BDD tests |
| 27 | NEWWORK Part 1: Fix broken prompts | DONE | `8290205` | Fix migrate-schemas, setup-encryption, debug-registration-error, context-management |
| 28 | NEWWORK Part 2: Enhance content + mcp-config glossary | DONE | `7ca0f16` | Expand 6 prompt files, 3 glossary files, add mcp-configuration glossary |
| 29 | NEWWORK Part 3: 5 new glossary resources | DONE | `a65d6f6` | error-reference, auth-and-security, storage-backends, normalization, tool-selection-guide |
| 30 | NEWWORK Part 4: 8 new prompts | DONE | `93b8980` | new-kafka-topic, debug-deserialization, deprecate-subject, cicd-integration, team-onboarding, governance-setup, cross-cutting-change, schema-review-checklist |
| 31 | NEWWORK Part 5: Server instructions update | DONE | `0767709` | 6 new glossary URIs + 5 new critical rules |
| 32 | NEWWORK Part 6: Context feature integration | DONE | `a847ca1` | Context note in core-concepts, MCP support section in contexts glossary |
| 33 | NEWWORK Part 7: Permission scopes | DONE | `32ae0cf` | 14 scopes, 5 presets, tool-to-scope mapping, Prometheus counter, 11 unit + 10 BDD tests |
| 34 | NEWWORK Part 8: BDD workflow tests | DONE | `0bf91ba` | 45 scenarios across 9 feature files |
| 35 | NEWWORK Part 9: Documentation updates | DONE | `3380d3e` | mcp.md, configuration.md, security.md, deployment.md, mcp-reference.md |
| 36 | NEWWORK Part 10: Documentation audit | DONE | `40aba79` | Verified all counts, updated README, added MCP to complete config example |
| 37 | Developer docs overhaul | DONE | `6cdd230` | Rewrote CLAUDE.md, development.md, testing.md with accurate project state |
| 38 | Issue #287: Migration prompt + BDD fixes | DONE | `7ece464` | Expanded migrate-schemas.md, fixed 8 workflow feature files, added `should be an error` step |
| 39 | gofmt CI fixes | DONE | `ff62a67` | Fix pre-existing gofmt in permissions.go and metrics.go |
| 40 | MCP reference with prompt content | DONE | `c512268` | Doc generator calls prompts/get, renders content in collapsible `<details>` blocks |
| 41 | Encryption workflow CI fix | DONE | `5d8ec48` | Remove @kms tag, replace test_kek with list_keks, add explicit version to DEK ops |

## Verified Implementation Counts (from source audit)

### Tools: 105 (105 `addToolIfAllowed` calls)

| Group | Count | File | Tool Names |
|-------|-------|------|------------|
| Server | 3 | `tools.go` | `health_check`, `get_server_info`, `list_subjects` |
| Schema Read | 13 | `tools_schema.go` | `get_schema_by_id`, `get_raw_schema_by_id`, `get_schema_version`, `get_raw_schema_version`, `get_latest_schema`, `list_versions`, `get_subjects_for_schema`, `get_versions_for_schema`, `get_referenced_by`, `lookup_schema`, `get_schema_types`, `list_schemas`, `get_max_schema_id` |
| Schema Write | 4 | `tools_write.go` | `register_schema`, `delete_subject`, `delete_version`, `check_compatibility` |
| Config/Mode | 6 | `tools_config.go` | `get_config`, `set_config`, `delete_config`, `get_mode`, `set_mode`, `delete_mode` |
| Context | 2 | `tools_context.go` | `list_contexts`, `import_schemas` |
| KEK/DEK | 13 | `tools_dek.go` | `create_kek`, `get_kek`, `update_kek`, `delete_kek`, `undelete_kek`, `list_keks`, `rewrap_dek`, `create_dek`, `get_dek`, `list_deks`, `list_dek_versions`, `delete_dek`, `undelete_dek` |
| Exporter | 11 | `tools_exporter.go` | `create_exporter`, `get_exporter`, `update_exporter`, `delete_exporter`, `list_exporters`, `get_exporter_config`, `update_exporter_config`, `get_exporter_status`, `pause_exporter`, `resume_exporter`, `reset_exporter` |
| Metadata | 12 | `tools_metadata.go` | `get_config_full`, `set_config_full`, `get_subject_config_full`, `resolve_alias`, `get_schemas_by_subject`, `check_write_mode`, `test_kek`, `format_schema`, `get_global_config_direct`, `get_subject_metadata`, `get_cluster_id`, `get_server_version` |
| Admin | 15 | `tools_admin.go` | `list_users`, `create_user`, `get_user`, `get_user_by_username`, `update_user`, `delete_user`, `list_apikeys`, `create_apikey`, `get_apikey`, `update_apikey`, `delete_apikey`, `revoke_apikey`, `rotate_apikey`, `list_roles`, `change_password` |
| Validation/Export | 11 | `tools_validation.go` | `validate_schema`, `normalize_schema`, `validate_subject_name`, `search_schemas`, `get_schema_history`, `get_dependency_graph`, `export_schema`, `export_subject`, `get_registry_statistics`, `count_versions`, `count_subjects` |
| Comparison/Search | 6 | `tools_comparison.go` | `check_compatibility_multi`, `diff_schemas`, `compare_subjects`, `suggest_compatible_change`, `match_subjects`, `explain_compatibility_failure` |
| Intelligence | 9 | `tools_intelligence.go` | `find_schemas_by_field`, `find_schemas_by_type`, `find_similar_schemas`, `score_schema_quality`, `check_field_consistency`, `get_schema_complexity`, `detect_schema_patterns`, `suggest_schema_evolution`, `plan_migration_path` |
| **Old subtotal** | **79** | — | Phases 1-13 |
| **New subtotal** | **26** | — | Phases 14-16 |

- 71 tools marked `ReadOnlyHint: true` (visible in read-only mode)
- 34 tools are write operations (hidden in read-only mode)

### Resources: 47 (25 static + 22 templated)

Glossary resources serve content from embedded `.md` files via `content.GlossaryFS`.

**Static (25)** — `AddResource`:
| # | URI | Name |
|---|-----|------|
| 1 | `schema://server/info` | Server version, commit, build time, supported types |
| 2 | `schema://server/config` | Global compatibility level and mode |
| 3 | `schema://subjects` | List of all registered subjects |
| 4 | `schema://types` | Supported schema types (AVRO, PROTOBUF, JSON) |
| 5 | `schema://contexts` | List of all registry contexts |
| 6 | `schema://mode` | Global registry mode |
| 7 | `schema://keks` | List of all KEKs |
| 8 | `schema://exporters` | List of all exporter names |
| 9 | `schema://status` | Server health and status |
| 10 | `schema://glossary/core-concepts` | Schema registry fundamentals |
| 11 | `schema://glossary/compatibility` | 7 compatibility modes, per-format rules |
| 12 | `schema://glossary/data-contracts` | Metadata, tags, rulesets, 3-layer merge |
| 13 | `schema://glossary/encryption` | CSFLE, KEK/DEK, KMS providers |
| 14 | `schema://glossary/contexts` | Multi-tenancy, 4-tier inheritance |
| 15 | `schema://glossary/exporters` | Schema linking, lifecycle states |
| 16 | `schema://glossary/schema-types` | Avro, Protobuf, JSON Schema reference |
| 17 | `schema://glossary/design-patterns` | Event envelope, lifecycle, shared types |
| 18 | `schema://glossary/best-practices` | Per-format guidance, common mistakes |
| 19 | `schema://glossary/migration` | Confluent migration, IMPORT mode |
| 20 | `schema://glossary/mcp-configuration` | MCP server config, permissions, security |
| 21 | `schema://glossary/error-reference` | All error codes, diagnostic guidance |
| 22 | `schema://glossary/auth-and-security` | RBAC roles, auth methods, rate limiting |
| 23 | `schema://glossary/storage-backends` | PostgreSQL, MySQL, Cassandra characteristics |
| 24 | `schema://glossary/normalization-and-fingerprinting` | Canonical forms, deduplication |
| 25 | `schema://glossary/tool-selection-guide` | Decision tree for choosing the right tool |

**Templated (22)** — `AddResourceTemplate`:
| # | URI Template | Name |
|---|-------------|------|
| 20 | `schema://subjects/{subject}` | Subject details: latest version, type, config |
| 21 | `schema://subjects/{subject}/versions` | All version numbers for a subject |
| 22 | `schema://subjects/{subject}/versions/{version}` | Schema at specific version |
| 23 | `schema://subjects/{subject}/config` | Per-subject compatibility config |
| 24 | `schema://subjects/{subject}/mode` | Per-subject mode |
| 25 | `schema://schemas/{id}` | Schema by global ID |
| 26 | `schema://schemas/{id}/subjects` | Subjects using a schema ID |
| 27 | `schema://schemas/{id}/versions` | Subject-version pairs for schema ID |
| 28 | `schema://exporters/{name}` | Exporter details by name |
| 29 | `schema://keks/{name}` | KEK details by name |
| 30 | `schema://keks/{name}/deks` | DEK subjects under a KEK |
| 31 | `schema://contexts/{context}/subjects` | Subjects in a specific context |
| 32 | `schema://contexts/{context}/config` | Global config/mode for a specific context |
| 33 | `schema://contexts/{context}/mode` | Global mode for a specific context |
| 34 | `schema://contexts/{context}/subjects/{subject}` | Subject details within a context |
| 35 | `schema://contexts/{context}/subjects/{subject}/versions` | Subject versions within a context |
| 36 | `schema://contexts/{context}/subjects/{subject}/versions/{version}` | Version detail within a context |
| 37 | `schema://contexts/{context}/subjects/{subject}/config` | Subject config within a context |
| 38 | `schema://contexts/{context}/subjects/{subject}/mode` | Subject mode within a context |
| 39 | `schema://contexts/{context}/schemas/{id}` | Schema by ID within a context |
| 40 | `schema://contexts/{context}/schemas/{id}/subjects` | Schema subjects within a context |
| 41 | `schema://contexts/{context}/schemas/{id}/versions` | Schema versions within a context |

### Prompts: 33

| # | Name | Required Args | Optional Args |
|---|------|---------------|---------------|
| 1 | `design-schema` | `format` | `domain` |
| 2 | `evolve-schema` | `subject` | — |
| 3 | `check-compatibility` | `subject` | — |
| 4 | `migrate-schemas` | `source_format`, `target_format` | — |
| 5 | `setup-encryption` | `kms_type` | — |
| 6 | `configure-exporter` | `exporter_type` | — |
| 7 | `review-schema-quality` | `subject` | — |
| 8 | `plan-breaking-change` | `subject` | — |
| 9 | `debug-registration-error` | `error_code` | — |
| 10 | `setup-data-contracts` | `subject` | — |
| 11 | `audit-subject-history` | `subject` | — |
| 12 | `compare-formats` | `use_case` | — |
| 13 | `schema-getting-started` | — | — |
| 14 | `troubleshooting` | — | — |
| 15 | `schema-impact-analysis` | `subject` | — |
| 16 | `schema-naming-conventions` | — | — |
| 17 | `context-management` | — | — |
| 18 | `glossary-lookup` | `topic` | — |
| 19 | `import-from-confluent` | — | — |
| 20 | `setup-rbac` | — | — |
| 21 | `schema-references-guide` | — | — |
| 22 | `full-encryption-lifecycle` | — | — |
| 23 | `data-rules-deep-dive` | — | — |
| 24 | `registry-health-audit` | — | — |
| 25 | `schema-evolution-cookbook` | — | — |
| 26 | `new-kafka-topic` | `topic_name` | `format` |
| 27 | `debug-deserialization` | — | — |
| 28 | `deprecate-subject` | `subject` | `context` |
| 29 | `cicd-integration` | — | — |
| 30 | `team-onboarding` | `team_name` | — |
| 31 | `governance-setup` | — | — |
| 32 | `cross-cutting-change` | `field_name` | — |
| 33 | `schema-review-checklist` | `subject` | `context` |

### Server Instructions

Server instructions are returned to MCP clients during the `initialize` handshake via `gomcp.ServerOptions{Instructions}`. They provide a capabilities overview, all 16 glossary resource URIs, and critical rules for schema registry operations.

### Security

- **Bearer token auth** (`middleware.go`): HTTP middleware on `/mcp`, configurable via `mcp.auth_token`
- **Read-only mode**: `mcp.read_only: true` hides tools without `ReadOnlyHint: true` annotation
- **Permission scopes** (`permissions.go`): 14 scopes, 5 presets (readonly/developer/operator/admin/full), tool-to-scope mapping
- **Tool policy** (`tools.go`): `allow_all` (default), `deny_list`, `allow_list` via `mcp.tool_policy`
- **All tools instrumented**: Prometheus metrics, structured `slog` logging, audit trail via `instrumentedHandler`

### Config (`internal/config/config.go` MCPConfig)

```go
type MCPConfig struct {
    Enabled              bool     `yaml:"enabled"`
    Host                 string   `yaml:"host"`
    Port                 int      `yaml:"port"`
    AuthToken            string   `yaml:"auth_token"`
    ReadOnly             bool     `yaml:"read_only"`
    ToolPolicy           string   `yaml:"tool_policy"`
    AllowedTools         []string `yaml:"allowed_tools"`
    DeniedTools          []string `yaml:"denied_tools"`
    AllowedOrigins       []string `yaml:"allowed_origins"`
    RequireConfirmations bool     `yaml:"require_confirmations"`
    ConfirmationTTLSecs  int      `yaml:"confirmation_ttl"`
    LogSchemas           bool     `yaml:"log_schemas"`
    PermissionPreset     string   `yaml:"permission_preset"`
    PermissionScopes     []string `yaml:"permission_scopes"`
}
```

## Testing (verified counts)

### Unit Tests: 208 across 8 test files

| File | Tests | Notes |
|------|-------|-------|
| `server_test.go` | 154 | Phases 1-13 + glossary resources + extended prompts + context-scoped resources/prompts |
| `tools_validation_test.go` | 11 | Phase 14 validation/export/statistics tools |
| `tools_comparison_test.go` | 7 | Phase 15 comparison/search tools |
| `tools_intelligence_test.go` | 10 | Phase 16 intelligence/evolution tools |
| `schema_utils_test.go` | 7 | Field extraction for Avro, JSON Schema, Protobuf |
| `fuzzy_test.go` | 5 | Levenshtein distance, fuzzy scoring, naming variants |
| `quality_test.go` | 3 | Schema quality scoring |
| `permissions_test.go` | 11 | Permission scopes, presets, precedence |

### BDD Tests: 2608 scenarios across 178 feature files (make test-bdd-functional)

#### MCP BDD Tests: 384 scenarios across 43 feature files

| Feature File | Tag | Scenarios |
|-------------|-----|-----------|
| `mcp_server.feature` | `@mcp-server` | 7 |
| `mcp_schema_read.feature` | `@mcp-schema-read` | 13 |
| `mcp_schema_write.feature` | `@mcp-schema-write` | 6 |
| `mcp_config.feature` | `@mcp-config` | 6 |
| `mcp_context.feature` | `@mcp-context` | 2 |
| `mcp_dek.feature` | `@mcp-dek` | 6 |
| `mcp_exporter.feature` | `@mcp-exporter` | 6 |
| `mcp_admin.feature` | `@mcp-admin` | 9 |
| `mcp_metadata.feature` | `@mcp-metadata` | 13 |
| `mcp_data_contracts.feature` | `@mcp-data-contracts` | 12 |
| `mcp_data_rules_e2e.feature` | `@mcp-data-rules` | 8 |
| `mcp_encryption_lifecycle.feature` | `@mcp-encryption` | 11 |
| `mcp_kms_e2e.feature` | `@kms` | 15 |
| `mcp_modeling_domain.feature` | `@mcp-modeling` | 5 |
| `mcp_modeling_errors.feature` | `@mcp-modeling` | 9 |
| `mcp_modeling_event_driven.feature` | `@mcp-modeling` | 4 |
| `mcp_modeling_lifecycle.feature` | `@mcp-modeling` | 7 |
| `mcp_modeling_multiformat.feature` | `@mcp-modeling` | 6 |
| `mcp_observability.feature` | `@mcp-observability` | 9 |
| `mcp_resources.feature` | `@mcp-resources` | 19 |
| `mcp_prompts.feature` | `@mcp-prompts` | 26 |
| `mcp_prompts_extended.feature` | `@mcp-prompts` | 8 |
| `mcp_glossary.feature` | `@mcp-glossary` | 10 |
| `mcp_security.feature` | `@mcp-security` | 4 |
| `mcp_confirmation.feature` | `@mcp` | 11 |
| `mcp_audit.feature` | `@mcp` | 7 |
| `mcp_validation.feature` | `@mcp` | 15 |
| `mcp_comparison.feature` | `@mcp` | 9 |
| `mcp_intelligence.feature` | `@mcp` | 13 |
| `mcp_evolution.feature` | `@mcp` | 7 |
| `mcp_dependency_graph.feature` | `@mcp` | 2 |
| `mcp_context_isolation.feature` | `@mcp` | 4 |
| `mcp_resource_context.feature` | `@mcp` | 18 |

#### REST Analysis BDD Tests: 113 scenarios across 9 feature files (`77738ed`)

| Feature File | Tag | Scenarios |
|-------------|-----|-----------|
| `rest_schema_validation.feature` | `@functional @analysis` | 12 |
| `rest_schema_search.feature` | `@functional @analysis` | 16 |
| `rest_schema_analysis.feature` | `@functional @analysis` | 14 |
| `rest_subject_validation.feature` | `@functional @analysis` | 14 |
| `rest_subject_history_export.feature` | `@functional @analysis` | 12 |
| `rest_subject_diff_evolve.feature` | `@functional @analysis` | 14 |
| `rest_compatibility_analysis.feature` | `@functional @analysis` | 14 |
| `rest_statistics.feature` | `@functional @analysis` | 11 |
| `rest_analysis_edge_cases.feature` | `@functional @analysis` | 6 |

### BDD Step Definitions: `tests/bdd/steps/mcp_steps.go` (702 lines)

Supports: tool calls (table + JSON input), tool listing, resource reads, prompt gets, error assertions.

### Test Commands (use Makefile targets — same as CI)

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
```

## Commit History (feature/mcp branch, newest first)

```
5d8ec48 fix(bdd): fix encryption workflow tests for CI compatibility
c512268 docs: include rendered prompt content in MCP reference
ff62a67 style: fix gofmt formatting in metrics.go
01816d1 style: fix gofmt formatting in permissions.go
7ece464 feat(mcp): comprehensive format migration guidance and BDD fixes (#287)
6cdd230 docs: overhaul CLAUDE.md, development.md, and testing.md with accurate project state
a4b71ce docs: update PROGRESS.md and RESUME_SESSION.md with NEWWORK.md results
40aba79 docs: add MCP section to complete config example, update README counts
3380d3e docs: update MCP, configuration, security, and deployment documentation
0bf91ba test(mcp): add 45 BDD workflow scenarios covering prompt-guided workflows
...
a1f9341 feat(mcp): add context support to resources and prompts
1ec7ed6 docs: regenerate MCP API reference with context parameter
c5b2370 refactor(mcp): extract glossary and prompt content to embedded markdown files
1bea1f8 feat(mcp): add optional context parameter to all registry-calling tools
bf5a0da style: fix gofmt trailing newline in tools_intelligence.go
99502bf fix: address PR #286 review findings
302c03e fix(test): handle json.Marshal error in mcp_steps.go
30f3692 fix(ci): remove unnecessary conformance job dependencies
3b58d87 refactor(ci): migrate BDD and unit test jobs to use Makefile targets
22e8840 feat(makefile): add CI awareness, fix timeouts, and add BDD test targets
a978dbd fix(api): handle ErrVersionNotFound in LookupSchema as 404 not 500
e49004b fix(cassandra): query actual data for GetMaxSchemaID instead of block allocator
6ec9d8a fix(mysql): use FOR UPDATE in globalSchemaIDTx to fix REPEATABLE READ race
50791dc fix(auth): add RBAC permissions for analysis and statistics endpoints
c2dcb85 fix(bdd): reset Cassandra ID cache during cleanup to prevent stale IDs
c8241b8 fix(bdd): replace hardcoded IDs with dynamic $variable resolution in MCP tests
956b29d fix(bdd): fix KMS and DB backend CI failures
27ed38d fix(mysql): move ID allocation into main transaction to fix dedup race
f55d7c4 fix(test): tune rate limiting and dedup tolerance for CI reliability
9f1564a feat(bdd): implement rate limiting BDD tests
de92f5f fix(test): tighten fingerprint dedup tolerance and increase Cassandra load
aee5cfd feat(ci): add database backend CI jobs for BDD and auth tests
8f64223 feat(bdd): run all in-process BDD tests against database backends
af19605 docs: update MCP documentation with glossary and new prompts
35487db test(mcp): add unit and BDD tests for glossary resources and prompts
ef273ef feat(mcp): add glossary resources, guided prompts, and server instructions
c0abf02 docs: enrich API and MCP tool descriptions with implementation detail
18b7307 fix(bdd): exclude @audit tag from Docker and Confluent backend tests
e1029da feat(audit): enterprise-grade logging and auditing for REST and MCP
12d40b7 chore: remove docs/ui directory (Web UI not in scope for this project)
2fec734 docs: add API reference links to MCP guide intro and contents
013d172 docs: add MCP server documentation and auto-generated API reference
02419bf test: add handler unit tests and integration tests for 26 REST analysis endpoints
411daf2 fix(bdd): exclude @analysis tag from Confluent backend tests
77738ed feat(bdd): add 115 BDD scenarios for REST analysis endpoints and MCP dependency graph
efedc32 feat(api): add 26 REST analysis endpoints with OpenAPI spec and ReDoc docs
8200d51 refactor: extract shared analysis utilities to internal/analysis package
48ba7d4 feat(mcp): add regex pattern filtering to list_subjects tool
3fa2148 feat(mcp): harden security defaults — localhost binding, wildcard origins, log_schemas
74c9c34 feat(mcp): add origin validation and two-phase confirmation security
f77df97 fix(bdd): fix BDD tag for confirmation scenarios — use @mcp not @mcp-confirmation
2c6848a feat(mcp): add 26 advanced tools — validation, comparison, search, AI intelligence
40f52f5 feat(ci): enable MCP KMS tests against database backends
5a76b61 fix(ci): exclude @mcp from KMS database backend tests
7cd47be fix: resolve CI lint failures — gofmt imports and gosec G112
a10f956 feat(mcp): add KMS E2E tests and fix godog escaped-quote step matching
67acb1e feat(mcp): wire per-principal metrics with hardcoded mcp-client principal
e1c1471 feat(mcp): add comprehensive BDD tests for AI data modeling, data contracts, and encryption
2bacf45 feat: add comprehensive observability — metrics, audit logging, and per-principal tracking
047a8c5 feat(mcp): add get_subject_metadata, get_cluster_id, and get_server_version tools
251bbcf feat(mcp): add change_password, rotate_apikey, and get_user_by_username tools
0f18cd9 feat(mcp): add 12 admin user and API key management tools with unit and BDD tests
43510a9 feat(mcp): add 9 metadata, alias, and advanced tools with unit and BDD tests
5ab0879 feat(mcp): add 11 exporter tools with unit and BDD tests
45ca908 feat(mcp): add 13 KEK & DEK encryption tools with unit and BDD tests
638b48f feat(mcp): add context and import tools with unit and BDD tests
04e4d91 feat(mcp): add 6 config and mode tools with unit and BDD tests
e68a63f feat(mcp): add 4 schema write tools with unit and BDD tests
ca3018d feat(mcp): add 13 schema read tools with unit and BDD tests
c703fe9 feat(mcp): add MCP server scaffolding with 3 starter tools and BDD tests
8f20667 fix(test): relax MySQL fingerprint dedup concurrency tolerance to 30%
f47a768 fix(ci): fix misspell lint and bump jackson-core for Trivy CVE
847bea4 fix(auth): use context.WithoutCancel for background API key update
0862af9 style: fix gofmt formatting in handlers.go and storage_test.go
6194d5e refactor: extract shared logic from handlers to registry/storage for MCP reuse
```

## Test Suite Hardening (Phases 21-21e)

### Problem
While the project had 2570 BDD scenarios, several critical gaps existed:
- MCP/auth BDD tests (345 scenarios) only ran against memory storage
- Fingerprint dedup test allowed 30% failure rate, masking real bugs
- Rate limiting BDD tests were tagged `@pending-impl` and never ran
- Cassandra concurrency tests ran at 1/30th load
- Go serde CSFLE tests could silently skip

### What Was Done

1. **All in-process BDD tests now run against database backends** (`8f64223`):
   - Added `newTestServerWithStore()`, `newTestServerWithStoreAndAudit()`, `newAuthTestServerWithStore()` functions
   - Modified `BeforeScenario` hooks in `TestFeatures` and `TestAuthFeatures` to use `sharedDBStore` when `BDD_STORAGE` is set
   - All 2570 scenarios run against PostgreSQL, MySQL, and Cassandra in CI

2. **6 new CI jobs** (`aee5cfd`):
   - `bdd-db-postgres-tests`, `bdd-db-mysql-tests`, `bdd-db-cassandra-tests` — full BDD suite against each DB
   - `bdd-auth-postgres-tests`, `bdd-auth-mysql-tests`, `bdd-auth-cassandra-tests` — auth tests against each DB
   - Total CI jobs: 37

3. **Fingerprint dedup tolerance tightened** (`de92f5f`): 30% → 10% error rate, Cassandra load increased from 1/30 to 1/6

4. **Rate limiting BDD tests implemented** (`9f1564a`):
   - 2 new step definitions: `I send N rapid requests to PATH`, `at least one response should have status CODE`
   - Removed `@pending-impl` tag from `rate_limiting.feature`

5. **MySQL concurrency fix** (`27ed38d` + `6ec9d8a`):
   - Moved ID allocation into main transaction to fix dedup race
   - Added `FOR UPDATE` in `globalSchemaIDTx` to bypass MySQL REPEATABLE READ snapshot isolation

6. **Cassandra fixes** (`c2dcb85` + `e49004b`):
   - Reset ID cache during `cleanDBStore()` to prevent stale block allocator IDs
   - Changed `GetMaxSchemaID` to query `MAX(schema_id) FROM schemas_by_id` instead of block allocator ceiling

7. **MCP BDD dynamic IDs** (`c8241b8`):
   - Added `resolveStoredVars()` helper for `$variable` resolution in MCP step definitions
   - Updated 6 MCP feature files to use stored dynamic IDs instead of hardcoded `"id": 1`

8. **RBAC permissions audit** (`50791dc`):
   - Added missing RBAC entries for 27 analysis endpoints (`POST /schemas`) and 3 statistics endpoints (`GET /statistics`)
   - Added `POST /subjects/validate` and `POST /subjects/match` as read-only (before generic `POST /subjects` write entry)
   - 2 new unit tests verifying permission entries and readonly user access

9. **LookupSchema error handling** (`a978dbd`):
   - Handle `ErrVersionNotFound` as 404 (not 500) during concurrent soft-delete races on Cassandra

### CI Status
All 37 CI jobs pass on commit `5d8ec48`.

## Key Technical Decisions

- **SDK**: `github.com/modelcontextprotocol/go-sdk` v1.4.0+ (official Go MCP SDK)
- **MCP Spec**: 2025-11-25 (latest)
- **Transport**: Streamable HTTP only (port 9081, separate from REST 8081)
- **Architecture**: Both REST and MCP call `registry.Registry` directly (shared business logic)
- **MCP auth**: Independent from REST auth (Bearer token only)
- **Tool policy**: `addToolIfAllowed` enforces read-only mode and tool policy at registration time
- **Instrumentation**: `instrumentedHandler` wraps every tool with Prometheus metrics, `slog` logging, audit trail
- **BDD transport**: `InMemoryTransport` for protocol-level BDD (no HTTP network I/O)
- **BDD storage**: `BDD_STORAGE` env var enables testing against database backends
