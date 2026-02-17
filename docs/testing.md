# Testing Strategy

Testing is a foundational pillar of AxonOps Schema Registry. The project maintains a multi-layered test suite with ~900 Go test functions, 76 BDD feature files containing ~1,400 scenarios, and ~50,000 lines of test code across Go and Gherkin. Every code change is validated across unit tests, integration tests, storage conformance tests, concurrency tests, BDD functional tests, API endpoint tests, authentication tests, migration tests, and multi-language Confluent wire-compatibility tests. The CI pipeline runs all of these against every supported storage backend on every push.

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
- [API Endpoint Tests](#api-endpoint-tests)
  - [What They Test](#what-api-tests-test)
  - [How to Run](#how-to-run-api-tests)
- [Authentication Tests](#authentication-tests)
  - [LDAP Tests](#ldap-tests)
  - [OIDC Tests](#oidc-tests)
  - [Vault Tests](#vault-tests)
  - [How to Run](#how-to-run-auth-tests)
- [Migration Tests](#migration-tests)
  - [What They Test](#what-migration-tests-test)
  - [How to Run](#how-to-run-migration-tests)
- [Confluent Wire-Compatibility Tests](#confluent-wire-compatibility-tests)
  - [What They Test](#what-compatibility-tests-test)
  - [Languages and Versions](#compatibility-languages-and-versions)
  - [How to Run](#how-to-run-compatibility-tests)
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

1. **Every layer proves something different.** Unit tests verify internal logic in isolation. Conformance tests prove all storage backends implement the same contract. Integration tests validate the full HTTP-to-database stack. BDD tests encode business requirements as executable specifications. Concurrency tests verify correctness under parallel load. Compatibility tests prove wire-level interoperability with Confluent clients. No single layer is sufficient on its own.

2. **Tests run against every storage backend.** A feature that works on PostgreSQL but fails on Cassandra is a bug. The storage conformance suite, integration tests, concurrency tests, and BDD tests all run against memory, PostgreSQL, MySQL, and Cassandra. The Makefile accepts a `BACKEND` variable to select the target, and `BACKEND=all` runs against all backends sequentially.

3. **Tests are living documentation.** The 76 BDD feature files are Gherkin specifications that describe the system's behavior in plain English. They serve as both executable tests and as the canonical reference for what the system does. When a feature file says a request SHOULD return a `40401` error, that is both a test assertion and a specification.

## Test Pyramid

```
                        ┌─────────────────────────────┐
                        │   Confluent Conformance     │  BDD against Confluent 8.1
                        │   (behavioral parity)       │
                        └──────────────┬──────────────┘
                                       │
                   ┌───────────────────┴───────────────────┐
                   │   Multi-Language Compatibility Tests   │  Go/Java/Python serializers
                   └───────────────────┬───────────────────┘
                                       │
              ┌────────────────────────┴────────────────────────┐
              │            BDD Functional Tests                 │  76 feature files
              │            (~1,400 scenarios)                   │  ~1,400 scenarios
              └───────┬────────┬────────┬────────┬─────────────┘
                      │        │        │        │
           ┌──────────┤  ┌─────┴─────┐  │  ┌────┴─────┐
           │Concurrency│  │ API Tests │  │  │Auth Tests│  LDAP/OIDC/Vault
           │  Tests    │  │(black-box)│  │  │          │
           └─────┬─────┘  └─────┬─────┘  │  └────┬────┘
                 │              │         │       │
           ┌─────┴──────────────┴─────────┴───────┴────┐
           │         Integration Tests                  │  Full HTTP + DB stack
           └─────────────────────┬─────────────────────┘
                                 │
           ┌─────────────────────┴─────────────────────┐
           │      Storage Conformance Suite             │  108 tests x 4 backends
           └─────────────────────┬─────────────────────┘
                                 │
           ┌─────────────────────┴─────────────────────┐
           │            Unit Tests                      │  ~900 functions
           └───────────────────────────────────────────┘
```

Each layer builds on the one below it. A failure at a lower layer (e.g., a conformance test) typically indicates a fundamental issue that will cascade into failures at higher layers.

## Quick Reference

Every test target accepts an optional `BACKEND` variable. Database containers are started and stopped automatically by the Makefile.

| Command | What It Runs | Docker Required | Approx. Time |
|---------|-------------|-----------------|---------------|
| `make test-unit` | Unit tests (`./internal/...`) | No | ~30s |
| `make test-conformance` | Storage conformance (memory) | No | ~10s |
| `make test-conformance BACKEND=postgres` | Storage conformance (PostgreSQL) | Yes | ~30s |
| `make test-conformance BACKEND=all` | Storage conformance (all backends) | Yes | ~2m |
| `make test-bdd` | BDD tests, in-process memory | No | ~2m |
| `make test-bdd BACKEND=postgres` | BDD tests against PostgreSQL | Yes | ~5m |
| `make test-bdd BACKEND=confluent` | BDD tests against Confluent 8.1 | Yes | ~5m |
| `make test-bdd BACKEND=all` | BDD tests against all backends | Yes | ~25m |
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

Unit tests verify the internal logic of every package in `internal/` without any external dependencies. They test schema parsing, compatibility checking, configuration loading, HTTP handler logic, authentication middleware, caching, metrics collection, RBAC enforcement, and the in-memory storage backend.

### Where Unit Tests Live

Every `_test.go` file inside `internal/` without a build tag is a unit test. Key files:

| File | Tests | What It Covers |
|------|-------|----------------|
| `internal/api/server_test.go` | 19 | HTTP server, middleware, routing, content-type handling |
| `internal/api/openapi_test.go` | 3 | OpenAPI spec/route sync, YAML validity, security schemes |
| `internal/api/handlers/handlers_test.go` | 76 | Schema, subject, config, mode, compat endpoint handlers |
| `internal/api/handlers/admin_test.go` | 45 | User and API key admin CRUD handlers |
| `internal/api/handlers/account_test.go` | 10 | Self-service account handlers |
| `internal/auth/auth_test.go` | 18 | Authenticator middleware chain |
| `internal/auth/rbac_test.go` | 6 | Role-based access control |
| `internal/auth/service_test.go` | 6 | User/API key business logic |
| `internal/auth/jwt_test.go` | 9 | JWT token generation and validation |
| `internal/auth/ratelimit_test.go` | 13 | Rate limiting middleware |
| `internal/auth/audit_test.go` | 23 | Audit logging |
| `internal/auth/tls_test.go` | 20 | TLS configuration, certificate handling |
| `internal/cache/cache_test.go` | 12 | Cache operations |
| `internal/compatibility/avro/checker_test.go` | 16 | Avro compatibility rules |
| `internal/compatibility/jsonschema/checker_test.go` | 34 | JSON Schema compatibility rules |
| `internal/compatibility/protobuf/checker_test.go` | 51 | Protobuf compatibility rules |
| `internal/compatibility/checker_test.go` | 30 | Compatibility checker registry/dispatch |
| `internal/config/config_test.go` | 25 | YAML config loading, validation, defaults |
| `internal/registry/registry_test.go` | 86 | Core business logic (schema CRUD, versioning, compat enforcement) |
| `internal/schema/avro/parser_test.go` | 23 | Avro schema parsing |
| `internal/schema/jsonschema/parser_test.go` | 35 | JSON Schema parsing |
| `internal/schema/protobuf/parser_test.go` | 33 | Protobuf parsing |
| `internal/schema/protobuf/resolver_test.go` | 11 | Protobuf reference resolution |
| `internal/storage/memory/store_test.go` | 12 | In-memory storage implementation |
| `internal/storage/factory_test.go` | 6 | Storage factory pattern |
| `internal/metrics/metrics_test.go` | 16 | Prometheus metrics registration |
| `internal/association/association_test.go` | 30 | Subject association logic |
| `internal/cluster/metadata_test.go` | 22 | Cluster metadata handling |
| `internal/context/context_test.go` | 27 | Context management |
| `internal/rules/engine_test.go` | 34 | Rules engine |
| `internal/exporter/exporter_test.go` | 40 | Schema exporter |

### How to Run Unit Tests

```bash
make test-unit
```

This runs `go test -race -v -timeout 5m ./internal/...`. No Docker or external services REQUIRED.

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

**Location:** `tests/integration/integration_test.go` (32 tests, 1,119 lines)

Build tag: `//go:build integration`

### How to Write Integration Tests

1. Add test functions to `tests/integration/integration_test.go`.
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

**Location:** `tests/concurrency/concurrency_test.go` (11 tests, 1,276 lines)

Build tag: `//go:build concurrency`

## BDD Tests

### What BDD Tests Test

BDD (Behavior-Driven Development) tests use Gherkin feature files with [godog](https://github.com/cucumber/godog) to describe the system's behavior in plain English. With 76 feature files and ~1,400 scenarios, they provide the most comprehensive functional coverage. They test every API endpoint, every compatibility rule for all three schema types, every error code, schema references, import operations, mode management, and operational resilience (service restart, crash recovery).

The BDD tests also serve as the primary mechanism for **Confluent conformance testing** -- the same feature files can be run against a real Confluent Schema Registry to verify behavioral parity.

### BDD Architecture

```
tests/bdd/
├── bdd_test.go                   # Test runner (godog initialization, backend selection)
├── features/                     # 76 .feature files (~24,000 lines of Gherkin)
│   ├── compatibility_avro.feature
│   ├── compatibility_protobuf.feature
│   ├── compatibility_jsonschema.feature
│   ├── compatibility_transitive.feature
│   ├── schema_avro_advanced.feature
│   ├── schema_references.feature
│   ├── import.feature
│   ├── ...
│   └── operational_resilience.feature
├── steps/                        # Step definitions
│   ├── context.go                # TestContext: HTTP client, state, JSON parsing
│   ├── schema_steps.go           # Schema registration, retrieval, deletion, compat, config
│   ├── reference_steps.go        # Schema reference operations
│   ├── import_steps.go           # Bulk import API
│   ├── mode_steps.go             # Mode management
│   └── infra_steps.go            # Operational scenarios (service start/stop)
├── docker-compose.base.yml       # Base service definition
├── docker-compose.postgres.yml   # PostgreSQL overlay
├── docker-compose.mysql.yml      # MySQL overlay
├── docker-compose.cassandra.yml  # Cassandra overlay
└── docker-compose.confluent.yml  # Confluent Schema Registry 8.1.1 + Kafka
```

### Feature Files

The 76 feature files are organized by functional area. The largest files by scenario count:

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

### Step Definitions

| File | Lines | What It Registers |
|------|-------|-------------------|
| `steps/context.go` | 238 | `TestContext` struct: HTTP client, response state, JSON parsing, placeholder resolution |
| `steps/schema_steps.go` | 483 | Steps for schema registration, retrieval, listing, lookup, deletion, compatibility, config |
| `steps/reference_steps.go` | 123 | Steps for schema reference registration and verification |
| `steps/import_steps.go` | 68 | Steps for the bulk import API |
| `steps/mode_steps.go` | 39 | Steps for mode management |
| `steps/infra_steps.go` | 127 | Steps for operational scenarios (service start/stop/restart via webhook) |

### BDD Tags

Tags control which scenarios run in which context:

| Tag | Purpose |
|-----|---------|
| `@functional` | Majority of scenarios. Run in-process (no Docker needed). |
| `@operational` | Require Docker infrastructure (service restart, crash recovery). |
| `@smoke` | Minimal health/metadata checks. |
| `@compatibility` | Compatibility checking scenarios. |
| `@import` | Import API scenarios. |
| `@axonops-only` | AxonOps-specific features not in Confluent (docs endpoint, import API). |
| `@pending-impl` | Not yet implemented. Always excluded. |
| `@memory`, `@postgres`, `@mysql`, `@cassandra` | Backend-specific scenarios. |
| `@avro`, `@protobuf`, `@jsonschema` | Schema-type-specific scenarios. |

### BDD Execution Modes

**1. In-process (no Docker):**

The default mode. Creates a fresh `httptest.Server` with in-memory storage per scenario. Skips `@operational` scenarios. Fast and suitable for development iteration.

```bash
make test-bdd
```

**2. Docker-based (real backends):**

Starts Docker Compose with the selected backend. Connects to the external registry. Cleans state between scenarios by truncating database tables directly.

```bash
make test-bdd BACKEND=postgres
make test-bdd BACKEND=cassandra
```

**3. Confluent conformance:**

Starts Confluent Schema Registry 8.1.1 with Kafka (KRaft mode) via Docker Compose. Runs the same feature files (excluding `@import`, `@axonops-only`, and backend-specific tags) against Confluent to verify behavioral parity.

```bash
make test-bdd BACKEND=confluent
```

### BDD State Cleanup

Between scenarios, the test runner cleans all state to ensure isolation:

- **PostgreSQL:** `TRUNCATE ... CASCADE` + `ALTER SEQUENCE ... RESTART WITH 1`
- **MySQL:** `SET FOREIGN_KEY_CHECKS = 0` + `TRUNCATE` each table
- **Cassandra:** `TRUNCATE` each table + re-seed `id_alloc`
- **Confluent/Memory:** API-based cleanup (reset mode, delete all subjects, reset config)

### How to Run BDD Tests

```bash
make test-bdd                          # In-process, memory (no Docker)
make test-bdd BACKEND=postgres         # Docker Compose with PostgreSQL
make test-bdd BACKEND=mysql            # Docker Compose with MySQL
make test-bdd BACKEND=cassandra        # Docker Compose with Cassandra
make test-bdd BACKEND=confluent        # Against Confluent Schema Registry 8.1.1
make test-bdd BACKEND=all              # All backends sequentially
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

1. **Create or update a `.feature` file** in `tests/bdd/features/`. Write scenarios in Gherkin syntax using existing step patterns where possible.
2. **Add new step definitions** if needed in the appropriate file under `tests/bdd/steps/`. Register them in the `InitializeScenario` function in `bdd_test.go`.
3. **Tag appropriately:** Use `@functional` for in-process tests, `@operational` for tests that require Docker infrastructure, and `@axonops-only` for features not present in Confluent.
4. **Run against memory first** (`make test-bdd`) for fast iteration, then verify against all backends (`make test-bdd BACKEND=all`).

## API Endpoint Tests

### What API Tests Test

API endpoint tests are black-box tests that run against a compiled, running binary. They verify that the built artifact serves all endpoints correctly. Unlike integration tests (which use `httptest.Server`), these test the actual binary including CLI argument parsing, signal handling, and startup/shutdown behavior.

### How to Run API Tests

```bash
make test-api
```

The Makefile builds the binary, starts it on port 28082 with in-memory storage, runs the tests, and shuts it down.

**Location:** `tests/api/api_test.go` (38 tests, 731 lines)

Build tag: `//go:build api`

## Authentication Tests

Authentication tests verify integration with external identity providers. Each test suite starts its own service container (OpenLDAP, Keycloak, or HashiCorp Vault), configures users and groups, and validates authentication and RBAC enforcement.

### LDAP Tests

**Location:** `tests/integration/ldap_integration_test.go` (5 tests, 624 lines)

Build tag: `//go:build ldap`

**Infrastructure:** OpenLDAP container with memberOf overlay, test users (admin, developer, readonly, nogroup), and groups (SchemaRegistryAdmins, Developers, ReadonlyUsers).

Tests: LDAP bind authentication, group-to-role mapping, RBAC enforcement on config and mode endpoints.

### OIDC Tests

**Location:** `tests/integration/oidc_integration_test.go` (6 tests, 736 lines)

Build tag: `//go:build oidc`

**Infrastructure:** Keycloak 24.0 container with a `schema-registry` realm, client, groups, and users.

Tests: Bearer token authentication, group claim to role mapping, RBAC enforcement, token validation (expiry, invalid tokens, malformed tokens).

### Vault Tests

**Location:** `tests/integration/vault_integration_test.go` (6 tests, 893 lines)

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

## Migration Tests

### What Migration Tests Test

Migration tests verify the schema import and migration workflow for users moving from Confluent Schema Registry. They test that imported schemas preserve their original IDs, that references are maintained, that duplicate detection works correctly, and that new registrations after import receive IDs higher than the maximum imported ID.

### How to Run Migration Tests

```bash
make test-migration
```

**Location:** `tests/migration/migration_test.go` (5 tests, 566 lines) + shell scripts

Build tag: `//go:build migration`

## Confluent Wire-Compatibility Tests

### What Compatibility Tests Test

These tests verify that AxonOps Schema Registry is wire-compatible with real Confluent serializer clients. They use the official Confluent client libraries to register schemas, serialize data, deserialize data, and verify round-trip compatibility. This proves that existing applications using Confluent serializers can switch to AxonOps Schema Registry without code changes.

### Compatibility Languages and Versions

| Language | Client Library | Versions Tested |
|----------|---------------|-----------------|
| **Go** | `confluent-kafka-go` | Latest |
| **Java** | `kafka-schema-registry-client` | 8.1, 7.9, 7.7.4, 7.7.3 |
| **Python** | `confluent-kafka` | 2.8.0, 2.7.0, 2.6.1 |

Each language tests Avro, Protobuf, and JSON Schema serialization/deserialization.

**Location:** `tests/compatibility/` (Go, Java, Python subdirectories)

### How to Run Compatibility Tests

```bash
make test-compatibility
```

Requires Maven (for Java tests) and Python 3 (for Python tests). Missing runtimes are skipped with a warning.

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

GitHub Actions runs the full test suite on every push to `main` or `feature/**` branches. The pipeline has 22 jobs:

| Stage | Jobs |
|-------|------|
| **Build** | Compile binary + all test binaries, run unit tests with coverage, upload artifacts |
| **Static Analysis** | golangci-lint, go vet, gosec, Trivy vulnerability scanner |
| **Docker** | Multi-stage Docker image build verification |
| **Database Tests** | Integration + concurrency tests against PostgreSQL 15, MySQL 8, Cassandra 5.0 |
| **Conformance** | Storage conformance suite against memory, PostgreSQL, MySQL, Cassandra |
| **API** | Black-box API endpoint tests |
| **Auth** | LDAP (OpenLDAP), Vault (HashiCorp Vault 1.15), OIDC (Keycloak 24.0) |
| **Migration** | Import/migration tests |
| **BDD** | In-process functional, then Docker-based: memory, PostgreSQL, MySQL, Cassandra, Confluent 8.1.1 |
| **Compatibility** | Go + Java (4 Confluent versions) + Python (3 versions) serializer tests |

The `build` job compiles all test binaries and uploads them as artifacts. Downstream jobs download pre-compiled binaries to avoid redundant compilation.

## Docker Compose Infrastructure

### Port Allocation

The project uses carefully managed port allocation to prevent conflicts between test layers:

| Context | PostgreSQL | MySQL | Cassandra | Registry |
|---------|-----------|-------|-----------|----------|
| BDD standalone | 5433 | 3307 | 9043 | 18081 |
| BDD overlay | 15432 | 13306 | 19042 | 18081 |
| Makefile DB targets | 25432 | 23306 | 29042 | -- |
| API tests | -- | -- | -- | 28082 |
| Compatibility tests | -- | -- | -- | 28083 |
| Default local dev | 5432 | 3306 | 9042 | 8081 |

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

## Related Documentation

- [Development](development.md) -- building from source, Makefile targets, code conventions
- [Configuration](configuration.md) -- all configuration options
- [API Reference](api-reference.md) -- complete endpoint documentation
- [Compatibility](compatibility.md) -- compatibility modes and checking behavior
- [Storage Backends](storage-backends.md) -- backend-specific setup and behavior
