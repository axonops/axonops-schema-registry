# Resume Document: Issue #365 — Comprehensive BDD Audit Assertion Upgrade

## Overview

Issue #365 upgrades ALL shallow single-field audit assertions (`the audit log should contain event "X"`) to comprehensive 27-field DataTable assertions (`the audit log should contain an event:` + DataTable with all 27 fields). It also fixes code-side gaps where audit hints are not properly populated.

**Branch:** `feature/mcp`
**Issue:** https://github.com/axonops/axonops-schema-registry/issues/369
**Status:** BDD assertion upgrades COMPLETE (1,742 across 162 files). Code-side fixes and CI stabilization REMAINING.

---

## What Has Been Done

### P0 Code Changes (pre-existing, triggered #365)

| Commit | Description |
|--------|-------------|
| `8ecfeb6` | Split delete events into soft/permanent, add missing event types |
| `7f15b7d` | Add before_hash/after_hash to all remaining P0 handlers |

These commits added 38 event types, hash fields, and new handler hints but had ZERO BDD validation — which is what #365 addresses.

### #365 Commits (30 total, oldest first)

| Commit | Type | Description |
|--------|------|-------------|
| `c452421` | BDD | Initial upgrade of all audit assertions to full 27-field coverage |
| `690036f` | BDD fix | Correct audit assertion field values from CI failures |
| `6e6de7e` | Code fix | Capture before_hash for permanent deletes of soft-deleted schemas |
| `2537cbd` | Code fix | Set exporter target_id on failure + fix duplicate reason assertion |
| `7519aa4` | Code fix | Add comprehensive audit hints to ImportSchemas handler |
| `c011358` | BDD fix | Fix remaining schema_id_conflict reason assertion |
| `60ffd20` | Code fix | Set early audit hints in RegisterSchema + add context/DEK assertions |
| `a1261c4` | BDD fix | Set version=* for schema_register audit assertion in context test |
| `d2b4786` | BDD | Add full 27-field audit assertions with non-default context |
| `af7da4f` | BDD fix | Correct reason assertion for 409 incompatible schema |
| `07ab57a` | BDD fix | Correct before_hash for reimport into soft-deleted subject |
| `080dc9f` | BDD fix | Update schema_id assertions for import failures with explicit IDs |
| `76fe612` | Code fix | Set schema_type for multi-subject imports using canonical default |
| `21918a8` | Code fix | Populate missing audit hints and update BDD assertions |
| `d76d763` | Code fix | Default context to ".", add CheckCompatibility hints, fix auth/KMS/exporter assertions |
| `f3cb9f6` | BDD fix | Strengthen auth_forbidden/delete audit assertions with correct target values |
| `0769e6c` | Code fix | Include fingerprint in Cassandra GetSchemaByID query |
| `c1a1fb0` | Code fix | Include fingerprint in Cassandra GetSchemaByID and GetSchemasBySubject queries |
| `8b9e92a` | BDD | Upgrade context feature files to full 27-field audit assertions |
| `aee30b5` | BDD | Upgrade contexts_schema_evolution and contexts_advanced_api |
| `6bc6269` | BDD | Upgrade 6 more context feature files |
| `78cb2eb` | BDD | Upgrade final 3 context files |
| `aa8bdb5` | BDD | Upgrade 5 MCP feature files (mcp_schema_write, mcp_config, mcp_admin, mcp_exporter, mcp_dek) |
| `4291ea4` | BDD | Upgrade remaining 34 MCP feature files |
| `9fb5bdc` | Code fix | Add audit hints to LookupSchema handler (target_type, target_id, schema_type, context, schema_id, version) |
| `dc1a347` | BDD | Upgrade 13 REST feature files (compatibility_protobuf, compatibility_jsonschema, avro_compat, schema_avro/protobuf_advanced) |
| `60ed4a7` | BDD | Upgrade 17 more REST feature files |
| `89112f4` | BDD | Upgrade 22 more REST feature files |
| `837a8ce` | BDD | Upgrade final 29 REST feature files |
| `0ec9223` | Lint fix | Remove unnecessary int64 conversion in LookupSchema handler |

### BDD Assertion Upgrade Summary

| Batch | Files | Assertions | Commit(s) |
|-------|-------|------------|-----------|
| Context files (14 files) | 14 | ~141 | `8b9e92a` through `78cb2eb` |
| MCP batch 1 | 5 | 46 | `aa8bdb5` |
| MCP batch 2 | 34 | 244 | `4291ea4` |
| REST batch 1 | 13 | 449 | `dc1a347` |
| REST batch 2 | 17 | 203 | `60ed4a7` |
| REST batch 3 | 22 | 175 | `89112f4` |
| REST batch 4 | 29 | 102 | `837a8ce` |

**Total: 1,742 full 27-field assertions across 162 feature files. Zero shallow assertions remain.**

### Code Changes Made to Handlers

| File | Handler | Change |
|------|---------|--------|
| `internal/api/handlers/handlers.go` | `LookupSchema` | Added audit hints: target_type=subject, target_id, schema_type, context, schema_id, version |
| `internal/api/handlers/handlers.go` | `RegisterSchema` | Set early audit hints before potential failure paths |
| `internal/api/handlers/handlers.go` | `ImportSchemas` | Comprehensive audit hints for import operations |
| `internal/api/handlers/handlers.go` | `DeleteSubject`/`DeleteVersion` | Capture before_hash for permanent deletes of soft-deleted schemas |
| `internal/api/handlers/handlers.go` | `CheckCompatibility` | Added audit hints for compatibility checks |
| `internal/api/handlers/exporter.go` | Various | Set target_id on failure paths |
| `internal/api/handlers/dek.go` | DEK operations | Context/DEK assertion fixes |
| `internal/auth/audit.go` | Middleware | Default context to "." |
| `internal/storage/cassandra/store.go` | `GetSchemaByID`/`GetSchemasBySubject` | Include fingerprint in queries |

---

## What Still Needs To Be Done

### Code-Side Fixes (from issue #365 comments — thorough re-review)

#### Gap 2 (P0): 34 event types missing `hints.Context`

Only 9 of 40+ event types correctly set `hints.Context`. The rest emit events with an empty context field.

**Events that SET context correctly (9):**
`schema_register`, `subject_delete_soft`, `subject_delete_permanent`, `schema_delete_soft`, `schema_delete_permanent`, `config_update`, `config_delete`, `mode_update`, `mode_delete`

**Events MISSING context (34):**

| Category | Events | File | Count |
|----------|--------|------|-------|
| KEK | `kek_create`, `kek_update`, `kek_delete_soft`, `kek_delete_permanent`, `kek_undelete` | `dek.go` | 5 |
| DEK | `dek_create`, `dek_delete_soft`, `dek_delete_permanent`, `dek_undelete` + version variants | `dek.go` | 6 |
| Exporter | `exporter_create/update/delete/pause/resume/reset/config_update` | `exporter.go` | 7 |
| Admin | `user_create`, `user_update`, `user_delete`, `password_change` | `admin.go`, `account.go` | 4 |
| API Key | `apikey_create/update/delete/revoke/rotate` | `admin.go` | 5 |
| MCP | `mcp_tool_call`, `mcp_tool_error`, `mcp_confirm_*` | `tools.go`, `confirmation.go` | 5 |
| Import | `schema_import` | `handlers.go` | 1 |
| Security | `security_warning` | `audit.go` | 1 |

**Fix:** Add `hints.Context = registryCtx` (or derive from request) in all affected handlers.

#### Gap 4 (P1): `DeleteSubject` missing `hints.SchemaType`

In `handlers.go:784-791`, `deletionSchemaType` is fetched from the latest schema but never assigned to `hints.SchemaType`.

**Fix:** Add `hints.SchemaType = deletionSchemaType` after the fetch.

#### Gap 5 (P2): DEK version ops missing `hints.Version`

- `DeleteDEKVersion` (`dek.go:445-451`) — version URL param exists but not passed to hints
- `UndeleteDEKVersion` (`dek.go:489-498`) — same issue

**Fix:** Add `hints.Version` from URL parameter in both handlers.

### BDD Assertion Fixes

#### Gap 1 (P0): Context feature files only assert default context

14 context feature files test 100+ distinct context values (e.g., `:.ops-ctx:`, `:.cfg-ctx:`) but ALL assertions use `context | .` or `context |` (empty). Zero assert a non-default context value.

**A bug where `context` is always `"."` regardless of actual registry context would pass all tests.**

**Files needing non-default context assertions:**
1. `contexts_operations.feature` — context `:.ops-ctx:`
2. `contexts_config_mode.feature` — context `:.cfg-ctx:`
3. `contexts_config_mode_advanced.feature` — contexts `:.cfgm2:`, `:.cfgm3:`
4. `contexts_isolation.feature` — context `:.ctx-id-b:`
5. `contexts.feature` — contexts `.testctx`, `.ctx-id-b`
6. `contexts_schema_evolution.feature` — contexts `:.evo1:`, `:.evo2:`
7. `contexts_real_world.feature` — contexts `:.team-bravo:`, `:.prod:`
8. `contexts_schema_types.feature` — context `:.type-ctx:`
9. `contexts_edge_cases.feature` — context `:.my-context:`
10. `contexts_validation.feature` — context `:.ctx123:`
11. `contexts_references.feature` — context `:.ref1:`
12. `contexts_advanced_api.feature` — contexts `:.api1:`, `:.api2:`
13. `contexts_url_routing.feature` — various
14. `contexts_global_config.feature` — context `:.__GLOBAL:`

**Note:** Gap 2 (code fix) MUST be done first for some of these — if the handler doesn't set `hints.Context`, the assertion will fail.

### CI Stabilization

- CI run `23132687238` (commit `0ec9223`): **FAILED** — BDD Functional Tests failed (21/22 jobs passed)
- Static analysis + all other jobs passed
- Specific failing scenarios not yet identified — need to check CI logs
- Likely cause: assertion field value mismatches from the batch upgrades (agents may have set incorrect values for edge cases)

---

## 27-Field Assertion Templates (Reference)

### Template A: REST No-Auth Success (TestFeatures, TestRESTAuditFeatures, TestKMSFeatures)
```gherkin
    And the audit log should contain an event:
      | event_type           | <EVENT_TYPE>           |
      | outcome              | success                |
      | actor_id             |                        |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | <TARGET_TYPE>          |
      | target_id            | <TARGET_ID>            |
      | schema_id            | <SCHEMA_ID_or_empty>   |
      | version              | <VERSION_or_empty>     |
      | schema_type          | <SCHEMA_TYPE_or_empty> |
      | before_hash          | <HASH_or_empty>        |
      | after_hash           | <HASH_or_empty>        |
      | context              | .                      |
      | transport_security   | tls                    |
      | method               | <METHOD>               |
      | path                 | <PATH>                 |
      | status_code          | <STATUS>               |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |
      | source_ip            | *                      |
      | user_agent           | *                      |
```

### Template B: Auth Success (TestAuthFeatures)
```gherkin
    And the audit log should contain an event:
      | event_type           | <EVENT_TYPE>           |
      | outcome              | success                |
      | actor_id             | <USERNAME>             |
      | actor_type           | user                   |
      | auth_method          | basic                  |
      | role                 | <ROLE>                 |
      | target_type          | <TARGET_TYPE>          |
      | target_id            | <TARGET_ID>            |
      | schema_id            | <SCHEMA_ID_or_empty>   |
      | version              | <VERSION_or_empty>     |
      | schema_type          | <SCHEMA_TYPE_or_empty> |
      | before_hash          | <HASH_or_empty>        |
      | after_hash           | <HASH_or_empty>        |
      | context              | .                      |
      | transport_security   | tls                    |
      | method               | <METHOD>               |
      | path                 | <PATH>                 |
      | status_code          | <STATUS>               |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           | *                      |
      | source_ip            | *                      |
      | user_agent           | *                      |
```

### Template MCP: MCP Tool Call
```gherkin
    And the audit log should contain an event:
      | event_type           | mcp_tool_call          |
      | outcome              | success                |
      | actor_id             | mcp-anonymous          |
      | actor_type           | anonymous              |
      | auth_method          |                        |
      | role                 |                        |
      | target_type          | <TARGET_TYPE_or_empty> |
      | target_id            | <TARGET_ID_or_empty>   |
      | schema_id            |                        |
      | version              |                        |
      | schema_type          |                        |
      | before_hash          |                        |
      | after_hash           |                        |
      | context              |                        |
      | transport_security   |                        |
      | source_ip            |                        |
      | user_agent           |                        |
      | method               | MCP                    |
      | path                 | <TOOL_NAME>            |
      | status_code          | 0                      |
      | reason               |                        |
      | error                |                        |
      | request_body         |                        |
      | metadata             |                        |
      | timestamp            | *                      |
      | duration_ms          | *                      |
      | request_id           |                        |
```

### Field Population Rules Per Event Type

| Event Type | target_type | schema_type | schema_id | version | before_hash | after_hash |
|------------|-------------|-------------|-----------|---------|-------------|------------|
| `schema_register` | subject | AVRO/etc | `*` | `*` | | `sha256:*` |
| `schema_lookup` | subject | | `*` | `*` | | |
| `subject_delete_soft` | subject | | | | `sha256:*` | |
| `subject_delete_permanent` | subject | | | | `sha256:*` | |
| `schema_delete_soft` | subject | | | | `sha256:*` | |
| `schema_delete_permanent` | subject | | | | `sha256:*` | |
| `config_update` | config | | | | `sha256:*` or `*` | `sha256:*` |
| `config_delete` | config | | | | `sha256:*` | |
| `mode_update` | mode | | | | `sha256:*` or `*` | `sha256:*` |
| `mode_delete` | mode | | | | `sha256:*` | |
| `compatibility_check` | subject | | | | | |
| `kek_create` | kek | | | | | `sha256:*` |
| `kek_update` | kek | | | | `sha256:*` | `sha256:*` |
| `kek_delete_soft` | kek | | | | `sha256:*` | |
| `kek_delete_permanent` | kek | | | | `sha256:*` | |
| `kek_undelete` | kek | | | | | `sha256:*` |
| `dek_create` | dek | | | | | `sha256:*` |
| `dek_delete_soft` | dek | | | | `sha256:*` | |
| `dek_delete_permanent` | dek | | | | `sha256:*` | |
| `dek_undelete` | dek | | | | | `sha256:*` |
| `exporter_create` | exporter | | | | | `sha256:*` |
| `exporter_update` | exporter | | | | `sha256:*` | `sha256:*` |
| `exporter_delete` | exporter | | | | `sha256:*` | |
| `exporter_pause` | exporter | | | | `sha256:*` | `sha256:*` |
| `exporter_resume` | exporter | | | | `sha256:*` | `sha256:*` |
| `exporter_reset` | exporter | | | | `sha256:*` | `sha256:*` |
| `exporter_config_update` | exporter | | | | `sha256:*` | `sha256:*` |
| `exporter_status` | exporter | | | | | |
| `user_create` | user | | | | | `sha256:*` |
| `user_update` | user | | | | `sha256:*` | `sha256:*` |
| `user_delete` | user | | | | `sha256:*` | |
| `apikey_create` | apikey | | | | | `sha256:*` |
| `apikey_update` | apikey | | | | `sha256:*` | `sha256:*` |
| `apikey_delete` | apikey | | | | `sha256:*` | |
| `apikey_revoke` | apikey | | | | `sha256:*` | `sha256:*` |
| `apikey_rotate` | apikey | | | | `sha256:*` | `sha256:*` |
| `password_change` | user | | | | | |
| `schema_import` | subject | AVRO/etc | `*` | | | `sha256:*` |

### BDD Step Matching Logic (mcp_steps.go:768-828)

- **Exact match**: `gotVal == wantVal` (default for all fields)
- **Contains match**: `strings.Contains(gotVal, wantVal)` (only for `path` field)
- **Prefix wildcard**: `strings.HasPrefix(gotVal, prefix)` when `wantVal` ends with `*`
- **Null match**: `rawVal == nil && wantVal == ""` — matches absent/omitempty fields
- **Non-existent field**: If field not in JSON and `wantVal == ""` → match

Key patterns:
- `sha256:*` — matches any SHA-256 hash
- `*` — matches any non-nil value (prefix="" matches everything)
- `""` (empty) — matches nil/absent fields

### Non-Deterministic Fields (always use `*`)
`timestamp`, `duration_ms`, `request_id`, `source_ip`, `user_agent`

### Always Empty (assert `""`)
`request_body` (include_body=false in all test configs), `metadata` (unused by any handler)

---

## Known Issues

1. **`extractSubject()` bug in audit.go** — hardcodes `start := 10` for `/subjects/` URL prefix offset. Fails for URL-prefix paths. Separate issue, not blocking #365.

2. **MCP `target_type`/`target_id` only populated when tool args contain literal `subject` key** — `extractSubjectFromArgs()` in tools.go only looks for "subject" key. Admin tools (username/id), exporter tools (name), KEK tools (name) all have empty target fields in MCP events. This is a design limitation, not a bug per se.

---

## Key Files Modified

### Go source files
- `internal/api/handlers/handlers.go` — LookupSchema, RegisterSchema, ImportSchemas, DeleteSubject/DeleteVersion, CheckCompatibility
- `internal/api/handlers/exporter.go` — target_id on failure paths
- `internal/api/handlers/dek.go` — DEK context/assertion fixes
- `internal/auth/audit.go` — default context to "."
- `internal/storage/cassandra/store.go` — fingerprint in queries

### Feature files (ALL 162 upgraded)
- `tests/bdd/features/mcp/` — 39 files
- `tests/bdd/features/` — 123 files (contexts, REST, auth, KMS, audit, admin, deletion, import, config, mode, exporter, DEK/KEK, compatibility, modeling, workflow, etc.)

---

## How to Resume

1. **First**: Fix CI failure — check logs for run `23132687238` to identify failing scenarios
2. **Then**: Address Gap 2 (P0) — add `hints.Context` to 34 event types in handler code
3. **Then**: Address Gap 4 (P1) — add `hints.SchemaType` to DeleteSubject
4. **Then**: Address Gap 5 (P2) — add `hints.Version` to DEK version handlers
5. **Then**: Address Gap 1 (P0) — update context feature file assertions to use actual non-default context values
6. **Finally**: Push and verify CI green (all 43+ jobs)

Note: Gap 2 code fix MUST precede Gap 1 BDD fix — if handlers don't set context, assertions for non-default context will fail.
