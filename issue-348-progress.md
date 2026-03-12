# Issue #348: Enterprise Audit Log Outputs — Progress

## Overview
Multi-output audit log delivery: stdout + file with rotation + syslog (RFC 5424/TLS) + webhook (Splunk HEC, Elasticsearch).

## Phase Status

| Phase | Description | Status | Commit |
|-------|-------------|--------|--------|
| 1 | AuditOutput interface + refactor Log() | DONE | `dd0455f` |
| 2 | Config structure migration + env vars | DONE | (pending) |
| 3 | File output with rotation (lumberjack) | TODO | |
| 4 | CEF format | TODO | |
| 5 | Syslog output (srslog) | TODO | |
| 6 | Webhook output | TODO | |
| 7 | Prometheus metrics | TODO | |
| 8 | BDD tests | TODO | |
| 9 | Documentation + CLAUDE.md | TODO | |

## Phase 1: AuditOutput Interface (DONE)
- Added `AuditOutput` interface: `Write([]byte) error`, `Close() error`, `Name() string`
- Added `formattedOutput` struct pairing output with format type ("json"/"cef")
- Added `StdoutOutput`, `WriterOutput` concrete implementations
- Replaced `*slog.Logger` + `*os.File` with `[]formattedOutput` in AuditLogger
- Replaced slog `LogAttrs()` with `json.Marshal(event)` + newline + fan-out
- BDD safe: parseAuditEvents only checks audit field names, never slog envelope
- New tests: MultipleOutputs, FailingOutputDoesNotBlockOthers, ProducesValidJSON
- All 48 audit unit tests pass

## Phase 2: Config Structure Migration (DONE)
- Added `AuditOutputsConfig` with sub-configs: Stdout, File, Syslog, Webhook
- Legacy `log_file` still works (lower priority than new `outputs` config)
- Added 26 env var overrides for all audit output fields
- Added `validateAuditConfig()`: format type validation, required fields when enabled
- Migrated BDD configs: `config.memory-audit.yaml`, `config.memory-mcp-audit.yaml`
- Updated main.go startup log to show per-output enabled status
- All unit tests pass, build compiles
