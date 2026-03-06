# Testing Strategy

Testing is a foundational pillar of AxonOps Schema Registry. The project maintains a comprehensive multi-layered test suite with extensive Go test functions, BDD feature files containing thousands of scenarios, and tens of thousands of lines of test code across Go and Gherkin. Every code change is validated across unit tests, integration tests, storage conformance tests, concurrency tests, BDD functional tests, API endpoint tests, authentication tests, KMS encryption tests, migration tests, MCP protocol tests, and multi-language Confluent wire-compatibility tests. The CI pipeline runs all of these against every supported storage backend on every push.

This document explains the testing philosophy, describes every test layer in detail, and provides the exact commands and patterns needed to run, write, and extend tests.

## Contents

- [Testing Philosophy](#testing-philosophy)
- [Test Pyramid](#test-pyramid)
- [Quick Reference](#quick-reference)
- [Unit Tests](#unit-tests)
  - [What They Test](#what-unit-tests-test)
  - [Where They Live](#where-unit-tests-live)
  - [How to Run](#how-to-run-unit-tests)
  - [How to Write Unit Tests](#how-to-write-unit-tests)
  - [Conventions](#unit-test-conventions)
- [Storage Conformance Tests](#storage-conformance-tests)
  - [What They Test](#what-conformance-tests-test)
  - [Architecture](#conformance-architecture)
  - [How to Run](#how-to-run-conformance-tests)
  - [How to Extend](#how-to-extend-conformance-tests)
- [Integration Tests](#integration-tests)
  - [What They Test](#what-integration-tests-test)
  - [How to Run](#how-to-run-integration-tests)
  - [How to Write Integration Tests](#how-to-write-integration-tests)
- [Concurrency Tests](#concurrency-tests)
  - [What They Test](#what-concurrency-tests-test)
  - [Test Scenarios](#concurrency-test-scenarios)
  - [How to Run](#how-to-run-concurrency-tests)
- [BDD Tests](#bdd-tests)
  - [What They Test](#what-bdd-tests-test)
  - [Architecture](#bdd-architecture)
  - [Feature Files](#feature-files)
  - [Step Definitions](#step-definitions)
  - [Tags](#bdd-tags)
  - [Execution Modes](#bdd-execution-modes)
  - [State Cleanup](#bdd-state-cleanup)
  - [How to Run](#how-to-run-bdd-tests)
  - [How to Write BDD Tests](#how-to-write-bdd-tests)
- [MCP Tests](#mcp-tests)
  - [Unit Tests](#mcp-unit-tests)
  - [BDD Tests](#mcp-bdd-tests)
  - [How to Run](#how-to-run-mcp-tests)
  - [How to Write MCP Tests](#how-to-write-mcp-tests)
- [API Endpoint Tests](#api-endpoint-tests)
  - [What They Test](#what-api-tests-test)
  - [How to Run](#how-to-run-api-tests)
- [Authentication Tests](#authentication-tests)
  - [LDAP Tests](#ldap-tests)
  - [OIDC Tests](#oidc-tests)
  - [Vault Tests](#vault-tests)
  - [How to Run](#how-to-run-auth-tests)
- [KMS Encryption Tests](#kms-encryption-tests)
  - [What They Test](#what-kms-tests-test)
  - [How to Run](#how-to-run-kms-tests)
- [Migration Tests](#migration-tests)
  - [What They Test](#what-migration-tests-test)
  - [How to Run](#how-to-run-migration-tests)
- [Confluent Wire-Compatibility Tests](#confluent-wire-compatibility-tests)
  - [What They Test](#what-compatibility-tests-test)
  - [Languages and Versions](#compatibility-languages-and-versions)
  - [How to Run](#how-to-run-compatibility-tests)
- [Data Contract & CSFLE Tests](#data-contract--csfle-tests)
  - [Java SerDe Tests](#java-serde-tests)
  - [Go SerDe Tests](#go-serde-tests)
  - [How to Run](#how-to-run-data-contract-tests)
- [OpenAPI Validation](#openapi-validation)
  - [What It Tests](#what-openapi-validation-tests)
  - [What to Do When It Fails](#what-to-do-when-openapi-validation-fails)
- [Confluent Conformance via BDD](#confluent-conformance-via-bdd)
- [CI Pipeline](#ci-pipeline)
- [Docker Compose Infrastructure](#docker-compose-infrastructure)
  - [Port Allocation](#port-allocation)
- [Pre-Commit Validation Workflow](#pre-commit-validation-workflow)
- [Which Tests to Write for a Given Change](#which-tests-to-write-for-a-given-change)
- [Related Documentation](#related-documentation)

## Testing Philosophy

The test suite is designed around three principles:

1. **Every layer proves something different.** Unit tests verify internal logic in isolation. Conformance tests prove all storage backends implement the same contract. Integration tests validate the full HTTP-to-database stack. BDD tests encode business requirements as executable specifications. Concurrency tests verify correctness under parallel load. Compatibility tests prove wire-level interoperability with Confluent clients. MCP tests verify AI assistant protocol compliance. No single layer is sufficient on its own.

2. **Tests run against every storage backend.** A feature that works on PostgreSQL but fails on Cassandra is a bug. The storage conformance suite, integration tests, concurrency tests, and BDD tests all run against memory, PostgreSQL, MySQL, and Cassandra. The Makefile accepts a `BACKEND` variable to select the target, and `BACKEND=all` runs against all backends sequentially.

3. **BDD tests are the primary specification.** The 178 BDD feature files are Gherkin specifications that describe the system's behavior in plain English. They serve as both executable tests and as the canonical reference for what the system does. When a feature file says a request SHOULD return a `40401` error, that is both a test assertion and a specification. If a feature is not BDD tested in a black-box manner, it is not considered tested.

## Test Pyramid

```
                        +-------------------------------+
                        |   Confluent Conformance       |  BDD against Confluent 8.1
                        |   (behavioral parity)         |
                        +---------------+---------------+
                                        |
                   +--------------------+--------------------+
                   |   Multi-Language Compatibility Tests     |  Go/Java/Python serializers
                   +--------------------+--------------------+
                                        |
              +-------------------------+-------------------------+
              |            BDD Functional Tests                   |  Feature files
              |            (thousands of scenarios)               |  All backends
              +---------+--------+--------+--------+-------------+
                        |        |        |        |
             +----------+  +-----+-----+  |  +----+-----+
             |Concurrency|  | API Tests |  |  |Auth Tests|  LDAP/OIDC/Vault
             |  Tests    |  |(black-box)|  |  |          |
             +-----+-----+  +-----+-----+  |  +----+----+
                   |              |         |       |
             +-----+--------------+---------+-------+----+
             |         Integration Tests                  |  Full HTTP + DB stack
             +---------------------+---------------------+
                                   |
             +---------------------+---------------------+
             |      Storage Conformance Suite             |  108 tests x 4 backends
             +---------------------+---------------------+
                                   |
             +---------------------+---------------------+
             |            Unit Tests                      |  1,436 functions
             +-------------------------------------------+
```

Each layer builds on the one below it. A failure at a lower layer (e.g., a conformance test) typically indicates a fundamental issue that will cascade into failures at higher layers.

## Quick Reference

Every test target accepts an optional `BACKEND` variable. Database containers are started and stopped automatically by the Makefile.

| Command | What It Runs | Docker Required | Approx. Time |
|---------|-------------|-----------------|---------------|
| `make test-unit` | Unit tests (`./...`) | No | ~30s |
| `make test-conformance` | Storage conformance (memory) | No | ~10s |
| `make test-conformance BACKEND=postgres` | Storage conformance (PostgreSQL) | Yes | ~30s |
| `make test-conformance BACKEND=all` | Storage conformance (all backends) | Yes | ~2m |
| `make test-bdd` | BDD tests, in-process memory | No | ~2m |
| `make test-bdd-functional` | Same as `make test-bdd` | No | ~2m |
| `make test-bdd BACKEND=postgres` | BDD tests against PostgreSQL | Yes | ~5m |
| `make test-bdd BACKEND=confluent` | BDD tests against Confluent 8.1 | Yes | ~5m |
| `make test-bdd BACKEND=all` | BDD tests against all backends | Yes | ~25m |
| `make test-bdd-db BACKEND=postgres` | BDD with in-process server + real DB | Yes | ~5m |
| `make test-bdd-db BACKEND=all` | BDD with in-process server + all DBs | Yes | ~15m |
| `make test-bdd-auth BACKEND=postgres` | BDD auth tests with real DB | Yes | ~5m |
| `make test-bdd-auth BACKEND=all` | BDD auth tests with all DBs | Yes | ~15m |
| `make test-bdd-kms` | BDD KMS tests (memory) | Yes | ~3m |
| `make test-bdd-kms BACKEND=all` | BDD KMS tests (all backends) | Yes | ~15m |
| `make test-integration BACKEND=postgres` | Integration tests (PostgreSQL) | Yes | ~2m |
| `make test-integration BACKEND=all` | Integration tests (all backends) | Yes | ~8m |
| `make test-concurrency BACKEND=postgres` | Concurrency tests (PostgreSQL) | Yes | ~3m |
| `make test-concurrency BACKEND=all` | Concurrency tests (all backends) | Yes | ~10m |
| `make test-api` | API endpoint tests (black-box) | No | ~1m |
| `make test-ldap` | LDAP auth tests | Yes | ~1m |
| `make test-vault` | Vault auth tests | Yes | ~1m |
| `make test-oidc` | OIDC auth tests | Yes | ~2m |
| `make test-auth` | All auth tests | Yes | ~4m |
| `make test-migration` | Migration/import tests | No | ~1m |
| `make test-compatibility` | Go/Java/Python Confluent clients | Yes | ~5m |
| `make test` | Everything | Yes | ~45m |
| `make test-coverage` | Unit tests with HTML coverage report | No | ~1m |

## Unit Tests

### What Unit Tests Test

Unit tests verify the internal logic of every package in `internal/` without any external dependencies. They test schema parsing, compatibility checking, configuration loading, HTTP handler logic, authentication middleware, caching, metrics collection, RBAC enforcement, MCP server tools/resources/prompts, MCP permissions, analysis functions, rules engine, KMS providers, context management, exporters, and the in-memory storage backend.

### Where Unit Tests Live

Every `_test.go` file inside `internal/` without a build tag is a unit test. Key files:

| File | Tests | What It Covers |
|------|-------|----------------|
| `internal/mcp/server_test.go` | 154 | MCP server, tools, resources, prompts, context resolution |
| `internal/registry/registry_test.go` | 136 | Core business logic (schema CRUD, versioning, compat enforcement) |
| `internal/api/handlers/handlers_test.go` | 82 | Schema, subject, config, mode, compat endpoint handlers |
| `internal/compatibility/protobuf/checker_test.go` | 51 | Protobuf compatibility rules |
| `internal/api/handlers/admin_test.go` | 45 | User and API key admin CRUD handlers |
| `internal/api/handlers/dek_test.go` | 38 | DEK/KEK encryption handlers |
| `internal/schema/avro/parser_test.go` | 38 | Avro schema parsing |
| `internal/storage/memory/store_test.go` | 38 | In-memory storage implementation |
| `internal/auth/audit_test.go` | 36 | Audit logging |
| `internal/schema/jsonschema/parser_test.go` | 35 | JSON Schema parsing |
| `internal/compatibility/jsonschema/checker_test.go` | 34 | JSON Schema compatibility rules |
| `internal/schema/protobuf/parser_test.go` | 33 | Protobuf parsing |
| `internal/api/server_test.go` | 32 | HTTP server, middleware, routing, content-type handling |
| `internal/compatibility/checker_test.go` | 30 | Compatibility checker registry/dispatch |
| `internal/config/config_test.go` | 28 | YAML config loading, validation, defaults, env overrides |
| `internal/api/server_analysis_test.go` | 26 | REST analysis endpoints |
| `internal/context/context_test.go` | 22 | Context management |
| `internal/auth/tls_test.go` | 20 | TLS configuration, certificate handling |
| `internal/auth/auth_test.go` | 18 | Authenticator middleware chain |
| `internal/metrics/metrics_test.go` | 23 | Prometheus metrics (REST + MCP) |
| `internal/compatibility/avro/checker_test.go` | 16 | Avro compatibility rules |
| `internal/rules/validator_test.go` | 16 | Data contract rules validation |
| `internal/exporter/exporter_test.go` | 40 | Schema exporter logic |
| `internal/api/handlers/exporter_test.go` | 15 | Exporter endpoint handlers |
| `internal/api/handlers/context_helpers_test.go` | 14 | Context helper functions |
| `internal/auth/rbac_test.go` | 14 | Role-based access control |
| `internal/auth/ratelimit_test.go` | 13 | Rate limiting middleware |
| `internal/auth/ldap_test.go` | 12 | LDAP authentication |
| `internal/mcp/permissions_test.go` | 11 | MCP permission scopes and presets |
| `internal/mcp/tools_validation_test.go` | 11 | MCP validation tools |
| `internal/mcp/tools_intelligence_test.go` | 10 | MCP intelligence/analysis tools |
| `internal/schema/protobuf/resolver_test.go` | 11 | Protobuf reference resolution |
| `internal/compatibility/modes_test.go` | 10 | Compatibility mode logic |
| `internal/compatibility/result_test.go` | 10 | Compatibility result formatting |
| `internal/api/context_middleware_test.go` | 8 | Context middleware |
| `internal/mcp/tools_comparison_test.go` | 7 | MCP comparison tools |
| `internal/mcp/schema_utils_test.go` | 7 | MCP schema utilities |
| `internal/auth/service_test.go` | 6 | User/API key business logic |
| `internal/storage/factory_test.go` | 6 | Storage factory pattern |
| `internal/mcp/fuzzy_test.go` | 5 | MCP fuzzy matching |
| `internal/mcp/quality_test.go` | 3 | MCP quality scoring |
| `internal/api/openapi_test.go` | 3 | OpenAPI spec/route sync, YAML validity |

### How to Run Unit Tests

```bash
make test-unit
```

This runs `go test -race -v -timeout 5m ./...`. No Docker or external services REQUIRED.

### How to Write Unit Tests

1. Create a `_test.go` file in the same package as the code being tested.
2. Do **NOT** add a build tag -- unit tests MUST run without any tag.
3. Use the in-memory storage backend (`internal/storage/memory`) for any test that needs a store.
4. Use `httptest.NewRecorder()` for handler tests.
5. Run with `-race` to catch data races.

### Unit Test Conventions

- Test functions follow `TestFunctionName` or `TestFunctionName_Subcase` naming.
- Table-driven tests are RECOMMENDED for functions with multiple input/output combinations.
- Use `t.Helper()` in test helper functions.
- Avoid external network calls -- unit tests MUST be hermetic.

## Storage Conformance Tests

### What Conformance Tests Test

The storage conformance suite is a shared set of 108 test cases that run identically against every storage backend. Its purpose is to guarantee that all backends implement the `Storage` interface with identical behavior. If a test passes on PostgreSQL but fails on Cassandra, the conformance suite catches it.

### Conformance Architecture

The suite is defined in `tests/storage/conformance/suite.go` which exports `RunAll(t, StoreFactory)`. Each backend provides a test file that calls `RunAll` with a factory function that creates a fresh store instance.

| Category File | Sub-tests | What It Covers |
|---------------|-----------|----------------|
| `schema_tests.go` | 25 | CreateSchema, GetSchemaByID, GetSchemaBySubjectVersion, GetSchemasBySubject, GetSchemaByFingerprint, GetLatestSchema, DeleteSchema, ListSchemas, deduplication |
| `subject_tests.go` | 9 | ListSubjects, DeleteSubject, SubjectExists, soft-delete semantics |
| `config_tests.go` | 16 | Config CRUD (global and per-subject), defaults, deletion |
| `auth_tests.go` | 20 | User CRUD, API key CRUD, listing, filtering |
| `import_tests.go` | 8 | ImportSchema with specific IDs, duplicate handling, version ordering |
| `error_tests.go` | 30 | All sentinel error conditions (ErrNotFound, ErrSubjectNotFound, etc.) |

Backend test files:

| File | Build Tag | Backend |
|------|-----------|---------|
| `conformance_test.go` | none | Memory (always runs with `make test-unit`) |
| `postgres_test.go` | `conformance` | PostgreSQL |
| `mysql_test.go` | `conformance` | MySQL |
| `cassandra_test.go` | `conformance` | Cassandra |

### How to Run Conformance Tests

```bash
make test-conformance                     # Memory (no Docker)
make test-conformance BACKEND=postgres    # PostgreSQL
make test-conformance BACKEND=mysql       # MySQL
make test-conformance BACKEND=cassandra   # Cassandra
make test-conformance BACKEND=all         # All backends sequentially
```

### How to Extend Conformance Tests

To add a new conformance test:

1. Identify which category file in `tests/storage/conformance/` the test belongs to (schema, subject, config, auth, import, or error).
2. Add a new sub-test function within the appropriate `Run*Tests` function.
3. The test MUST use only the `storage.Storage` interface -- no backend-specific code.
4. Run `make test-conformance BACKEND=all` to verify the test passes on all backends.

To add a new storage backend to the conformance suite:

1. Create a new `<backend>_test.go` file in `tests/storage/conformance/`.
2. Add a `//go:build conformance` build tag.
3. Implement a `TestMain` that sets up the database connection.
4. Call `conformance.RunAll(t, factory)` with a factory function that returns a fresh store.
5. Ensure tables are truncated between sub-tests to prevent state leakage.

## Integration Tests

### What Integration Tests Test

Integration tests exercise the full HTTP API stack against real database backends. They create a real `api.Server` backed by a real storage implementation, wrapped in `httptest.Server`. Tests issue actual HTTP requests and verify both the HTTP responses and the underlying database state.

### How to Run Integration Tests

```bash
make test-integration BACKEND=postgres     # PostgreSQL
make test-integration BACKEND=mysql        # MySQL
make test-integration BACKEND=cassandra    # Cassandra
make test-integration BACKEND=all          # All backends
```

There is no memory backend for integration tests -- they REQUIRE a real database.

**Location:** `tests/integration/` (66 tests across 6 files)

Build tag: `//go:build integration`

### How to Write Integration Tests

1. Add test functions to the appropriate file in `tests/integration/`.
2. Use the shared `httptest.Server` created in `TestMain`.
3. Issue HTTP requests using `http.DefaultClient`.
4. Verify both HTTP response status/body and direct storage queries where appropriate.
5. Clean up created resources at the end of each test to avoid polluting other tests.

## Concurrency Tests

### What Concurrency Tests Test

Concurrency tests validate correctness when multiple clients perform simultaneous operations against the same database. They create multiple server instances sharing a single database and run parallel workers that perform schema registrations, version updates, reads, deletes, and compatibility checks concurrently.

### Concurrency Test Scenarios

| Test | Workers | Operations | What It Proves |
|------|---------|------------|----------------|
| `TestConcurrentSchemaRegistration` | 10 | 100 each | Unique schemas registered concurrently get unique IDs |
| `TestConcurrentVersionUpdates` | 10 | 10 each | Versions on a single subject are contiguous (no gaps) |
| `TestConcurrentReads` | 10 | 100 each | Reads return correct data under write load |
| `TestConcurrentMixedOperations` | 10 | mixed | Writes, reads, and deletes interleave safely |
| `TestConcurrentCompatibilityChecks` | 10 | 50 each | Compatibility checks are safe under load |
| `TestConcurrentConfigUpdates` | 10 | 20 each | Config updates from multiple instances converge |
| `TestDataConsistency` | 10 | 10 each | Write on instance 0, read from all instances |
| `TestHotSubjectContention` | 20 | 10 each | All writers to ONE subject get contiguous versions and unique IDs |
| `TestSchemaIdempotency` | 10 | 1 each | Identical schema posted by all workers returns same ID |
| `TestSchemaIDUniqueness` | 10 | 10 each | Different schemas across subjects get globally unique IDs |

The design creates 3 server instances (1 for Cassandra) sharing the same database. Workers use round-robin instance selection. Cassandra tests use reduced parallelism (5 workers, 20 operations) to account for lightweight transaction overhead.

### How to Run Concurrency Tests

```bash
make test-concurrency BACKEND=postgres     # PostgreSQL
make test-concurrency BACKEND=mysql        # MySQL
make test-concurrency BACKEND=cassandra    # Cassandra
make test-concurrency BACKEND=all          # All backends
```

**Location:** `tests/concurrency/concurrency_test.go` (29 tests)

Build tag: `//go:build concurrency`

## BDD Tests

### What BDD Tests Test

BDD (Behavior-Driven Development) tests use Gherkin feature files with [godog](https://github.com/cucumber/godog) to describe the system's behavior in plain English. They provide the most comprehensive functional coverage, testing every API endpoint, every compatibility rule for all three schema types, every error code, schema references, import operations, mode management, multi-tenant contexts, MCP tools/resources/prompts, MCP permissions, DEK/KEK encryption, exporter operations, data contracts, analysis endpoints, audit logging, rate limiting, and operational resilience (service restart, crash recovery).

The BDD tests also serve as the primary mechanism for **Confluent conformance testing** -- the same feature files can be run against a real Confluent Schema Registry to verify behavioral parity.

### BDD Architecture

```
tests/bdd/
├── bdd_test.go                   # Test runner (godog initialization, backend selection)
├── features/                     # 135 top-level .feature files
│   ├── compatibility_avro.feature
│   ├── compatibility_protobuf.feature
│   ├── compatibility_jsonschema.feature
│   ├── compatibility_transitive.feature
│   ├── schema_avro_advanced.feature
│   ├── schema_references.feature
│   ├── import.feature
│   ├── encryption_*.feature      # DEK/KEK encryption features
│   ├── exporter_*.feature        # Exporter (Schema Linking) features
│   ├── data_contracts_*.feature  # Data contract features
│   ├── contexts_*.feature        # Multi-tenant context features
│   ├── ...
│   ├── operational_resilience.feature
│   └── mcp/                      # MCP-specific .feature files
│       ├── mcp_tools.feature
│       ├── mcp_resources.feature
│       ├── mcp_prompts.feature
│       ├── mcp_glossary.feature
│       ├── mcp_permissions.feature
│       ├── mcp_workflow_*.feature  # 9 workflow scenario files
│       └── ...
├── steps/                        # 11 step definition files
│   ├── context.go                # TestContext: HTTP client, state, JSON parsing
│   ├── schema_steps.go           # Schema registration, retrieval, deletion, compat, config
│   ├── mcp_steps.go              # MCP tool calls, resource reads, prompt gets
│   ├── concurrency_steps.go      # Concurrent operation steps
│   ├── auth_steps.go             # Authentication and RBAC steps
│   ├── encryption_steps.go       # DEK/KEK encryption steps
│   ├── reference_steps.go        # Schema reference operations
│   ├── infra_steps.go            # Operational scenarios (service start/stop)
│   ├── import_steps.go           # Bulk import API
│   ├── mode_steps.go             # Mode management
│   └── rate_limit_steps.go       # Rate limiting steps
├── configs/                      # Per-backend BDD config files
│   ├── config.memory.yaml
│   ├── config.postgres.yaml
│   ├── config.mysql.yaml
│   └── config.cassandra.yaml
├── docker-compose.yml            # BDD standalone compose
├── docker-compose.base.yml       # Base service definition
├── docker-compose.postgres.yml   # PostgreSQL overlay
├── docker-compose.mysql.yml      # MySQL overlay
├── docker-compose.cassandra.yml  # Cassandra overlay
├── docker-compose.confluent.yml  # Confluent Schema Registry 8.1.1 + Kafka
├── docker-compose.memory.yml     # Memory overlay
├── docker-compose.kms.yml        # KMS services (Vault + OpenBao)
└── docker-compose.kms-overlay.yml # KMS overlay for backend compose
```

### Feature Files

The feature files are organized by functional area. The largest files by scenario count:

| Feature File | Scenarios | Area |
|-------------|-----------|------|
| `compatibility_jsonschema_diff_draft07.feature` | 104 | JSON Schema Draft-07 compatibility |
| `compatibility_jsonschema_diff_draft2020.feature` | 101 | JSON Schema Draft-2020 compatibility |
| `compatibility_protobuf.feature` | 67 | Protobuf compatibility |
| `compatibility_avro.feature` | 65 | Avro compatibility |
| `compatibility_jsonschema.feature` | 62 | JSON Schema compatibility |
| `avro_compatibility_exhaustive.feature` | 47 | Exhaustive Avro compat matrix |
| `compatibility_protobuf_diff.feature` | 43 | Protobuf diff compatibility |
| `schema_avro_advanced.feature` | 40 | Advanced Avro schema features |
| `compatibility_transitive.feature` | 40 | Transitive compatibility (3+ version chains) |

MCP feature files:

| Feature File | Scenarios | Area |
|-------------|-----------|------|
| `mcp_tools.feature` | — | Core MCP tool operations |
| `mcp_resources.feature` | — | MCP resource access |
| `mcp_prompts.feature` | — | MCP prompt retrieval |
| `mcp_glossary.feature` | — | Glossary resource content |
| `mcp_permissions.feature` | — | Permission scopes and presets |
| `mcp_workflow_*.feature` | 45 | 9 prompt-guided workflow scenarios |

### Step Definitions

| File | Lines | What It Registers |
|------|-------|-------------------|
| `steps/mcp_steps.go` | 756 | MCP tool calls, resource reads, prompt gets, permission config, session management |
| `steps/schema_steps.go` | 549 | Steps for schema registration, retrieval, listing, lookup, deletion, compatibility, config |
| `steps/concurrency_steps.go` | 430 | Concurrent operation steps |
| `steps/auth_steps.go` | 269 | Authentication and RBAC steps |
| `steps/encryption_steps.go` | 268 | DEK/KEK encryption steps |
| `steps/context.go` | 263 | `TestContext` struct: HTTP client, response state, JSON parsing, placeholder resolution |
| `steps/infra_steps.go` | 127 | Steps for operational scenarios (service start/stop/restart via webhook) |
| `steps/reference_steps.go` | 123 | Steps for schema reference registration and verification |
| `steps/import_steps.go` | 68 | Steps for the bulk import API |
| `steps/rate_limit_steps.go` | 63 | Rate limiting steps |
| `steps/mode_steps.go` | 39 | Steps for mode management |

### BDD Tags

Tags control which scenarios run in which context:

| Tag | Purpose |
|-----|---------|
| `@functional` | Majority of scenarios. Run in-process (no Docker needed). |
| `@operational` | Require Docker infrastructure (service restart, crash recovery). |
| `@smoke` | Minimal health/metadata checks. |
| `@compatibility` | Compatibility checking scenarios. |
| `@import` | Import API scenarios. |
| `@axonops-only` | AxonOps-specific features not in Confluent (docs, import API, analysis, MCP). |
| `@memory`, `@postgres`, `@mysql`, `@cassandra` | Backend-specific scenarios. |
| `@avro`, `@protobuf`, `@jsonschema` | Schema-type-specific scenarios. |
| `@kms` | KMS encryption tests (Vault/OpenBao Transit). Require KMS infrastructure. |
| `@data-contracts` | Data contract features (rules, metadata, DEK Registry). |
| `@encryption` | DEK/KEK encryption scenarios. |
| `@contexts` | Multi-tenant context scenarios. |
| `@mcp` | MCP protocol tests. |
| `@mcp-permissions` | MCP permission scope tests. |
| `@mcp-workflow` | MCP prompt-guided workflow tests. |
| `@mcp-glossary` | MCP glossary resource tests. |
| `@mcp-prompts` | MCP prompt tests. |
| `@mcp-resources` | MCP resource tests. |
| `@mcp-security` | MCP security tests. |
| `@mcp-confirmation` | MCP confirmation dialog tests. |
| `@auth` | Authentication scenarios. |
| `@admin` | Admin endpoint scenarios. |
| `@account` | Self-service account scenarios. |
| `@audit` | Audit logging scenarios. |
| `@rate-limiting` | Rate limiting scenarios. |
| `@concurrency` | Concurrency scenarios. |
| `@analysis` | Analysis endpoint scenarios. |
| `@references` | Schema reference scenarios. |
| `@security` | Security-related scenarios. |
| `@observability` | Metrics/monitoring scenarios. |
| `@pending-impl` | Not yet implemented. Always excluded. |

### BDD Execution Modes

**1. In-process (no Docker):**

The default mode. Creates a fresh `httptest.Server` with in-memory storage per scenario. Skips `@operational` scenarios. Fast and suitable for development iteration.

```bash
make test-bdd
# or equivalently:
make test-bdd-functional
```

**2. Docker-based (real backends):**

Starts Docker Compose with the selected backend. Connects to the external registry. Cleans state between scenarios by truncating database tables directly.

```bash
make test-bdd BACKEND=postgres
make test-bdd BACKEND=cassandra
```

**3. In-process with real DB:**

Uses an in-process server but connects to a real database. Useful for debugging backend-specific issues without Docker Compose complexity.

```bash
make test-bdd-db BACKEND=postgres
```

**4. Confluent conformance:**

Starts Confluent Schema Registry 8.1.1 with Kafka (KRaft mode) via Docker Compose. Runs the same feature files (excluding `@import`, `@axonops-only`, and backend-specific tags) against Confluent to verify behavioral parity.

```bash
make test-bdd BACKEND=confluent
```

**5. KMS encryption:**

Starts Vault and OpenBao KMS services. Runs scenarios tagged `@kms`.

```bash
make test-bdd-kms BACKEND=memory
make test-bdd-kms BACKEND=all
```

### BDD State Cleanup

Between scenarios, the test runner cleans all state to ensure isolation:

- **PostgreSQL:** `TRUNCATE ... CASCADE` + `ALTER SEQUENCE ... RESTART WITH 1`
- **MySQL:** `SET FOREIGN_KEY_CHECKS = 0` + `TRUNCATE` each table
- **Cassandra:** `TRUNCATE` each table + re-seed `id_alloc`
- **Confluent/Memory:** API-based cleanup (reset mode, delete all subjects, reset config)
- **MCP:** Session state is reset per scenario; `InMemoryTransport` is recreated

### How to Run BDD Tests

```bash
make test-bdd                          # In-process, memory (no Docker)
make test-bdd-functional               # Same as above (explicit alias)
make test-bdd BACKEND=postgres         # Docker Compose with PostgreSQL
make test-bdd BACKEND=mysql            # Docker Compose with MySQL
make test-bdd BACKEND=cassandra        # Docker Compose with Cassandra
make test-bdd BACKEND=confluent        # Against Confluent Schema Registry 8.1.1
make test-bdd BACKEND=all              # All backends sequentially
make test-bdd-db BACKEND=postgres      # In-process server + real PostgreSQL
make test-bdd-db BACKEND=all           # In-process server + all DBs
make test-bdd-auth BACKEND=postgres    # BDD auth tests + real PostgreSQL
make test-bdd-auth BACKEND=all         # BDD auth tests + all DBs
make test-bdd-kms BACKEND=memory       # BDD KMS tests (Vault + OpenBao)
make test-bdd-kms BACKEND=all          # BDD KMS tests + all backends
```

To run a specific feature file:

```bash
go test -tags bdd -v -run 'TestFeatures/compatibility_avro' ./tests/bdd/...
```

To run scenarios with a specific tag:

```bash
go test -tags bdd -v ./tests/bdd/... -godog.tags="@smoke"
```

### How to Write BDD Tests

1. **Create or update a `.feature` file** in `tests/bdd/features/` (or `tests/bdd/features/mcp/` for MCP tests). Write scenarios in Gherkin syntax using existing step patterns where possible.
2. **Add new step definitions** if needed in the appropriate file under `tests/bdd/steps/`. Register them in the `InitializeScenario` function in `bdd_test.go`.
3. **Tag appropriately:** Use `@functional` for in-process tests, `@operational` for tests that require Docker infrastructure, `@axonops-only` for features not present in Confluent, and `@mcp` for MCP tests.
4. **Run against memory first** (`make test-bdd`) for fast iteration, then verify against all backends (`make test-bdd BACKEND=all`).

## MCP Tests

### MCP Unit Tests

MCP unit tests live in `internal/mcp/`. They cover:

- Server initialization and options pattern
- Tool registration and permission filtering
- Resource and resource template registration
- Prompt handler registration and content
- Context resolution (`resolveContext`, `resolveResourceContext`)
- Permission scopes, presets, and resolution precedence
- Fuzzy matching, quality scoring, schema field analysis
- Instrumented handler metrics and logging

### MCP BDD Tests

MCP BDD tests live in `tests/bdd/features/mcp/`. They use `InMemoryTransport` for protocol-level testing (no HTTP network I/O). They cover:

- All tools: invocation, parameter validation, error handling
- All resources: static content and templated resolution
- All prompts: content validation and context parameter handling
- Permission scopes: readonly/developer/operator/admin/full presets
- Workflow scenarios: 9 end-to-end prompt-guided workflows (45 scenarios)
- Security: auth token validation, read-only mode enforcement

Step definitions are in `tests/bdd/steps/mcp_steps.go` (756 lines). Key patterns:

- `_mcp_permission_preset` / `_mcp_permission_scopes` — set before lazy session creation
- `getMCPSession()` — creates or returns cached MCP session with `InMemoryTransport`
- `$variable` resolution — BDD variables like `$schema_id` resolved from prior step results

### How to Run MCP Tests

```bash
# Unit tests (included in make test-unit)
make test-unit

# BDD tests (included in make test-bdd)
make test-bdd

# MCP BDD tests only
go test -tags bdd -v ./tests/bdd/... -godog.tags="@mcp"
```

### How to Write MCP Tests

1. **New tool:** Add the tool implementation in the appropriate `tools_*.go` file. Add it to `toolPermissionScope` in `permissions.go`. Write unit tests in the corresponding `*_test.go`. Add BDD scenarios in `tests/bdd/features/mcp/`.
2. **New resource:** Add to `resources.go` (or `glossary.go` for glossary content). Add BDD scenarios in `mcp_resources.feature`.
3. **New prompt:** Add the handler in `prompts.go`, create the content markdown in `content/prompts/`. Add BDD scenarios in `mcp_prompts.feature`.
4. **New permission scope:** Add the scope constant in `permissions.go`, update preset definitions, update `toolPermissionScope` mapping. Add BDD scenarios in `mcp_permissions.feature`.

## API Endpoint Tests

### What API Tests Test

API endpoint tests are black-box tests that run against a compiled, running binary. They verify that the built artifact serves all endpoints correctly. Unlike integration tests (which use `httptest.Server`), these test the actual binary including CLI argument parsing, signal handling, and startup/shutdown behavior.

### How to Run API Tests

```bash
make test-api
```

The Makefile builds the binary, starts it on port 28082 with in-memory storage, runs the tests, and shuts it down.

**Location:** `tests/api/api_test.go` (38 tests)

Build tag: `//go:build api`

## Authentication Tests

Authentication tests verify integration with external identity providers. Each test suite starts its own service container (OpenLDAP, Keycloak, or HashiCorp Vault), configures users and groups, and validates authentication and RBAC enforcement.

### LDAP Tests

**Location:** `tests/integration/ldap_integration_test.go` (5 tests)

Build tag: `//go:build ldap`

**Infrastructure:** OpenLDAP container with memberOf overlay, test users (admin, developer, readonly, nogroup), and groups (SchemaRegistryAdmins, Developers, ReadonlyUsers).

Tests: LDAP bind authentication, group-to-role mapping, RBAC enforcement on config and mode endpoints.

### OIDC Tests

**Location:** `tests/integration/oidc_integration_test.go` (6 tests)

Build tag: `//go:build oidc`

**Infrastructure:** Keycloak 24.0 container with a `schema-registry` realm, client, groups, and users.

Tests: Bearer token authentication, group claim to role mapping, RBAC enforcement, token validation (expiry, invalid tokens, malformed tokens).

### Vault Tests

**Location:** `tests/integration/vault_integration_test.go` (6 tests)

Build tag: `//go:build vault`

**Infrastructure:** HashiCorp Vault 1.15 in dev mode. Uses Vault KV v2 as auth storage backend.

Tests: User CRUD stored in Vault, API key CRUD stored in Vault, password authentication with Vault-backed store, RBAC enforcement, Vault health connectivity.

### How to Run Auth Tests

```bash
make test-ldap       # LDAP tests only
make test-vault      # Vault tests only
make test-oidc       # OIDC tests only
make test-auth       # All three
```

## KMS Encryption Tests

### What KMS Tests Test

KMS encryption tests verify client-side field-level encryption (CSFLE) using DEK/KEK management with external KMS providers. Tests cover KEK creation, DEK generation, encryption/decryption round-trips, key rotation, soft-delete/undelete, and multi-backend storage of encryption metadata.

Both HashiCorp Vault Transit and OpenBao are tested as KMS providers.

### How to Run KMS Tests

```bash
make test-bdd-kms                        # Memory backend with Vault + OpenBao
make test-bdd-kms BACKEND=postgres       # PostgreSQL with Vault + OpenBao
make test-bdd-kms BACKEND=all            # All backends with Vault + OpenBao
```

The Makefile starts Vault and OpenBao containers, runs the KMS setup script, executes `@kms`-tagged BDD scenarios, and tears down the containers.

**Docker Compose:** `tests/bdd/docker-compose.kms.yml`

**KMS endpoints:**
- Vault: `http://localhost:18200` (token: `test-root-token`)
- OpenBao: `http://localhost:18201` (token: `test-bao-token`)

## Migration Tests

### What Migration Tests Test

Migration tests verify the schema import and migration workflow for users moving from Confluent Schema Registry. They test that imported schemas preserve their original IDs, that references are maintained, that duplicate detection works correctly, and that new registrations after import receive IDs higher than the maximum imported ID.

### How to Run Migration Tests

```bash
make test-migration
```

**Location:** `tests/migration/migration_test.go` (5 tests) + shell scripts

Build tag: `//go:build migration`

## Confluent Wire-Compatibility Tests

### What Compatibility Tests Test

These tests verify that AxonOps Schema Registry is wire-compatible with real Confluent serializer clients. They use the official Confluent client libraries to register schemas, serialize data, deserialize data, and verify round-trip compatibility. This proves that existing applications using Confluent serializers can switch to AxonOps Schema Registry without code changes.

### Compatibility Languages and Versions

| Language | Client Library | Versions Tested |
|----------|---------------|-----------------|
| **Go** | `confluent-kafka-go` | Latest |
| **Go (SerDe)** | `confluent-kafka-go/v2` | v2.8.0 (data contracts + CSFLE) |
| **Java** | `kafka-schema-registry-client` | 8.1, 7.9, 7.7.4, 7.7.3 |
| **Python** | `confluent-kafka` | 2.8.0, 2.7.0, 2.6.1 |

Each language tests Avro, Protobuf, and JSON Schema serialization/deserialization. The Go SerDe suite additionally tests data contract rules (CEL, JSONata, global policies) and CSFLE field-level encryption.

**Location:** `tests/compatibility/` (Go, Go SerDe, Java, Python subdirectories)

### How to Run Compatibility Tests

```bash
make test-compatibility
```

Requires Maven (for Java tests) and Python 3 (for Python tests). Missing runtimes are skipped with a warning.

## Data Contract & CSFLE Tests

### Java SerDe Tests

Java integration tests using real Confluent SerDe clients to verify end-to-end rule execution and field-level encryption.

| Test Class | Tests | What It Covers |
|-----------|-------|----------------|
| `DataContractCelRulesTest` | 8 | CEL CONDITION (write/read), CEL_FIELD TRANSFORM (PII masking, normalization), disabled rules |
| `DataContractMigrationTest` | 4 | JSONata UPGRADE/DOWNGRADE, field rename, field addition, breaking change bridging |
| `CsfleVaultEncryptionTest` | 7 | CSFLE round-trip, raw byte inspection, multi-field encryption, DEK/KEK auto-creation via Vault |
| `DataContractGlobalPoliciesTest` | 4 | Default ruleSet execution, override ruleSet enforcement, rule inheritance, tag propagation |

**Location:** `tests/compatibility/java/src/test/java/com/axonops/schemaregistry/compat/`

**JUnit tags:** `data-contracts`, `csfle`

### Go SerDe Tests

Go integration tests using `confluent-kafka-go/v2` (v2.8.0). 30 tests across 7 files.

| Test File | Tests | What It Covers |
|-----------|-------|----------------|
| `cel_rules_test.go` | 8 | CEL CONDITION (write/read), CEL_FIELD TRANSFORM (PII masking, normalization), disabled rules |
| `migration_rules_test.go` | 6 | JSONata UPGRADE/DOWNGRADE, field rename, field addition, breaking change bridging |
| `csfle_vault_test.go` | 7 | CSFLE round-trip, raw byte inspection, multi-field encryption, DEK/KEK auto-creation via Vault Transit |
| `global_policies_test.go` | 4 | Default ruleSet execution, override ruleSet enforcement, rule inheritance, PII tag propagation |
| `cel_extras_test.go` | 4 | Go-only extra CEL scenarios (chained transforms, nested field conditions) |
| `csfle_extras_test.go` | 3 | Go-only extra CSFLE scenarios (key rotation, multi-tenant encryption) |
| `cel_jsonschema_test.go` | 2 | CEL rules with JSON Schema format |
| `cel_protobuf_test.go` | 2 | CEL rules with Protobuf format |

**Location:** `tests/compatibility/go-serde/`

### How to Run Data Contract Tests

```bash
# Java data contracts (no KMS)
cd tests/compatibility/java
mvn test -P confluent-8.1 -Dschema.registry.url=http://localhost:8081 -Dgroups=data-contracts

# Java CSFLE (requires Vault)
mvn test -P confluent-8.1 \
  -Dschema.registry.url=http://localhost:8081 \
  -Dgroups=csfle \
  -Dvault.url=http://localhost:18200 \
  -Dvault.token=test-root-token

# Go data contracts (no KMS)
cd tests/compatibility/go-serde
SCHEMA_REGISTRY_URL=http://localhost:8081 go test -v -run "TestCel|TestMigration|TestGlobalPolicies" ./...

# Go CSFLE (requires Vault)
SCHEMA_REGISTRY_URL=http://localhost:8081 \
  VAULT_URL=http://localhost:18200 \
  VAULT_TOKEN=test-root-token \
  go test -v -run "TestCsfle" -timeout 10m ./...
```

## OpenAPI Validation

### What OpenAPI Validation Tests

The test `TestOpenAPISpecMatchesRoutes` in `internal/api/openapi_test.go` enforces bidirectional sync between the OpenAPI spec (`api/openapi.yaml`) and the chi router:

1. Every route registered in the router MUST exist in the OpenAPI spec.
2. Every path in the OpenAPI spec MUST exist in the router.

This test runs as part of the unit test suite (no build tag). It prevents the spec and implementation from drifting apart.

Two additional tests validate that the embedded spec is valid YAML and that security schemes (`basicAuth`, `apiKey`, `bearerAuth`) are defined.

### What to Do When OpenAPI Validation Fails

If you add, remove, or rename an API endpoint:

1. Update the route in `internal/api/server.go`.
2. Update `api/openapi.yaml` to match.
3. Run `make test-unit` to verify the sync test passes.
4. Run `make docs-api` to regenerate the Markdown and HTML API documentation.

## Confluent Conformance via BDD

Running the BDD tests against Confluent Schema Registry is the primary mechanism for proving behavioral parity:

```bash
make test-bdd BACKEND=confluent
```

This starts Confluent Schema Registry 8.1.1 with Kafka in KRaft mode (via `tests/bdd/docker-compose.confluent.yml`) and runs the same Gherkin scenarios. Tags `@import`, `@axonops-only`, and backend-specific tags are excluded since those test AxonOps-specific features.

If a scenario passes against AxonOps but fails against Confluent, it indicates either:
- A behavioral difference that SHOULD be documented in the compatibility notes.
- A bug in the feature file's expectations.

If a scenario passes against Confluent but fails against AxonOps, it indicates a compatibility bug in AxonOps that MUST be fixed.

## CI Pipeline

GitHub Actions runs 37 jobs on every push to `main` or `feature/**` branches. The pipeline is defined in `.github/workflows/ci.yaml`.

| Stage | Jobs |
|-------|------|
| **Build** | `build` — compile binary + all test binaries, run unit tests with coverage, upload artifacts |
| **Static Analysis** | `lint`, `security` — golangci-lint, go vet, gosec, Trivy |
| **Gate** | `gate` — dependency gate, all downstream jobs wait for build + lint + security |
| **Docker** | `docker` — multi-stage Docker image build verification |
| **Database Tests** | `postgres-tests`, `mysql-tests`, `cassandra-tests` — integration + concurrency |
| **Conformance** | `conformance-memory`, `conformance-postgres`, `conformance-mysql`, `conformance-cassandra` |
| **API** | `api-tests` — black-box API endpoint tests |
| **Auth** | `ldap-tests`, `vault-tests`, `oidc-tests` |
| **Migration** | `migration-tests` |
| **BDD Docker** | `bdd-memory-tests`, `bdd-postgres-tests`, `bdd-mysql-tests`, `bdd-cassandra-tests`, `bdd-confluent-tests` |
| **BDD Functional** | `bdd-functional-tests` — in-process functional tests |
| **BDD DB** | `bdd-db-postgres-tests`, `bdd-db-mysql-tests`, `bdd-db-cassandra-tests` |
| **BDD Auth** | `bdd-auth-postgres-tests`, `bdd-auth-mysql-tests`, `bdd-auth-cassandra-tests` |
| **BDD KMS** | `bdd-kms-tests`, `bdd-kms-postgres-tests`, `bdd-kms-mysql-tests`, `bdd-kms-cassandra-tests` |
| **Compatibility** | `compatibility-tests` — Go + Java + Python serializer tests |
| **Data Contracts** | `java-data-contract-csfle-tests`, `go-data-contract-csfle-tests`, `python-data-contract-csfle-tests` |

The `build` job compiles all test binaries and uploads them as artifacts. Downstream jobs download pre-compiled binaries to avoid redundant compilation.

## Docker Compose Infrastructure

### Port Allocation

The project uses carefully managed port allocation to prevent conflicts between test layers:

| Context | PostgreSQL | MySQL | Cassandra | Registry | KMS (Vault) | KMS (OpenBao) |
|---------|-----------|-------|-----------|----------|-------------|---------------|
| BDD standalone | 5433 | 3307 | 9043 | 18081 | — | — |
| BDD overlay | 15432 | 13306 | 19042 | 18081 | — | — |
| BDD KMS | — | — | — | — | 18200 | 18201 |
| Makefile DB targets | 25432 | 23306 | 29042 | — | — | — |
| API tests | — | — | — | 28082 | — | — |
| Compatibility tests | — | — | — | 28083 | — | — |
| Auth tests | — | — | — | — | 28200 (Vault) | — |
| Default local dev | 5432 | 3306 | 9042 | 8081 | — | — |

## Pre-Commit Validation Workflow

Before submitting any change, developers SHOULD run the following sequence:

```bash
# 1. Format and lint (fast, catches style issues)
make fmt && make lint

# 2. Unit tests (fast, catches logic errors)
make test-unit

# 3. Storage conformance (catches backend contract violations)
make test-conformance BACKEND=all

# 4. BDD tests in-process (catches functional regressions)
make test-bdd

# 5. BDD tests against all backends (catches backend-specific issues)
make test-bdd BACKEND=all

# 6. If API endpoints changed: regenerate docs and verify sync
make docs-api && make test-unit
```

For changes affecting storage, concurrency, or authentication, also run:

```bash
make test-integration BACKEND=all
make test-concurrency BACKEND=all
make test-auth
```

For changes affecting MCP:

```bash
# MCP BDD tests
go test -tags bdd -v ./tests/bdd/... -godog.tags="@mcp"

# Regenerate MCP reference
make docs-mcp
```

For changes affecting encryption:

```bash
make test-bdd-kms BACKEND=all
```

## Which Tests to Write for a Given Change

| Type of Change | Tests REQUIRED |
|---------------|---------------|
| New API endpoint | Handler unit test, BDD feature scenarios, OpenAPI spec update, API endpoint test |
| New storage method | Conformance test in `tests/storage/conformance/`, integration test, BDD coverage |
| New storage backend | Conformance backend file calling `RunAll`, integration test support, BDD Docker Compose file |
| Compatibility rule change | Compatibility checker unit test, BDD compatibility scenarios |
| New schema type | Parser unit test, compatibility checker unit test, BDD schema scenarios |
| Auth method change | Auth unit test, auth integration test (LDAP/OIDC/Vault as applicable) |
| Configuration option | Config unit test, integration test verifying the option takes effect |
| Bug fix | Unit test reproducing the bug, plus BDD scenario if it affects API behavior |
| New MCP tool | Unit test, BDD scenario in `features/mcp/`, `toolPermissionScope` entry in `permissions.go` |
| New MCP resource | BDD scenario in `mcp_resources.feature` or `mcp_glossary.feature` |
| New MCP prompt | BDD scenario in `mcp_prompts.feature`, content file in `content/prompts/` |
| New MCP permission scope | Unit test in `permissions_test.go`, BDD scenario in `mcp_permissions.feature` |
| DEK/KEK encryption change | BDD KMS scenario, handler unit test |
| Exporter change | Exporter unit test, BDD exporter scenario |
| Data contract rule change | BDD data contract scenario, rules engine unit test |

## Related Documentation

- [Development](development.md) -- building from source, Makefile targets, code conventions
- [Configuration](configuration.md) -- all configuration options
- [API Reference](api-reference.md) -- complete endpoint documentation
- [MCP Server](mcp.md) -- MCP server for AI-assisted schema management
- [Compatibility](compatibility.md) -- compatibility modes and checking behavior
- [Storage Backends](storage-backends.md) -- backend-specific setup and behavior
- [Encryption](encryption.md) -- DEK/KEK encryption and KMS providers
- [Exporters](exporters.md) -- schema exporter (Schema Linking) configuration
