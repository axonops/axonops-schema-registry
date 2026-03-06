# Development

This guide covers building AxonOps Schema Registry from source, running the test suite, and contributing changes.

## Contents

- [Prerequisites](#prerequisites)
- [Building from Source](#building-from-source)
  - [Cross-Compilation](#cross-compilation)
- [Project Structure](#project-structure)
- [Running Tests](#running-tests)
  - [Unit Tests](#unit-tests)
  - [BDD Tests](#bdd-tests)
  - [Integration Tests](#integration-tests)
  - [Concurrency Tests](#concurrency-tests)
  - [Storage Conformance Tests](#storage-conformance-tests)
  - [Auth Tests](#auth-tests)
  - [Migration Tests](#migration-tests)
  - [API Endpoint Tests](#api-endpoint-tests)
  - [Compatibility Tests](#compatibility-tests)
  - [Data Contract & CSFLE Tests](#data-contract--csfle-tests)
  - [Go SerDe Data Contract & CSFLE Tests](#go-serde-data-contract--csfle-tests)
  - [Full Test Suite](#full-test-suite)
  - [Coverage](#coverage)
- [Code Quality](#code-quality)
- [Running the Server Locally](#running-the-server-locally)
- [API Documentation](#api-documentation)
  - [OpenAPI Spec](#openapi-spec)
  - [Swagger UI](#swagger-ui)
  - [Static Documentation](#static-documentation)
  - [MCP Reference](#mcp-reference)
- [Docker](#docker)
  - [Build the Image](#build-the-image)
  - [Run with Docker](#run-with-docker)
- [CI Pipeline](#ci-pipeline)
- [Code Conventions](#code-conventions)
- [Contributing](#contributing)
- [Related Documentation](#related-documentation)

## Prerequisites

- **Go 1.26+** ([download](https://go.dev/dl/))
- **Docker** (for integration, BDD, auth, KMS, and concurrency tests)
- **golangci-lint** ([install](https://golangci-lint.run/welcome/install/)) for linting
- **Node.js** (optional, for generating static API documentation with Redocly)

Optional tools for the full compatibility test suite:

- **Maven** and **Java 17+** (for Java Confluent serializer and data contract tests)
- **Python 3** (for Python Confluent serializer tests)

## Building from Source

```bash
git clone https://github.com/axonops/axonops-schema-registry.git
cd axonops-schema-registry
make build
```

The binary is placed at `./build/schema-registry`. The Makefile injects version, commit hash, and build date via linker flags automatically:

```bash
./build/schema-registry --version
```

To build manually without Make:

```bash
go build \
  -ldflags "-X main.version=$(git describe --tags --always) \
            -X main.commit=$(git rev-parse --short HEAD) \
            -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o schema-registry ./cmd/schema-registry
```

### Cross-Compilation

Build binaries for all supported platforms:

```bash
make build-all
```

This produces:

```
build/schema-registry-linux-amd64
build/schema-registry-linux-arm64
build/schema-registry-darwin-amd64
build/schema-registry-darwin-arm64
```

## Project Structure

```
cmd/schema-registry/              Entry point (cobra CLI)
cmd/generate-mcp-docs/            MCP reference doc generator
internal/
  analysis/                       Shared analysis (field extraction, quality scoring, fuzzy matching)
  api/                            HTTP server, handlers, types, OpenAPI
    handlers/                     REST endpoint handlers (schema, admin, account, analysis, DEK, exporter)
    types/                        Request/response structs + Confluent error codes
  auth/                           Authentication, RBAC, JWT, rate limiting, audit, TLS
  compatibility/                  Compatibility checkers (Avro, JSON Schema, Protobuf)
  config/                         YAML config loading with env var overrides
  context/                        Multi-tenant context management
  exporter/                       Schema exporter (Schema Linking) logic
  kms/                            KMS provider interface + implementations
    vault/                        HashiCorp Vault Transit
    aws/                          AWS KMS
    azure/                        Azure Key Vault
    gcp/                          GCP Cloud KMS
  mcp/                            MCP (Model Context Protocol) server
    content/                      Embedded markdown (16 glossary + 35 prompt files)
  metrics/                        Prometheus metrics (REST + MCP)
  registry/                       Core business logic (shared by REST + MCP)
  rules/                          Data contract rules engine (CEL, JSONata)
  schema/                         Schema parsers (Avro, JSON Schema, Protobuf)
  storage/                        Storage interface (71 methods) + backends
    memory/                       In-memory backend
    postgres/                     PostgreSQL backend
    mysql/                        MySQL backend
    cassandra/                    Cassandra backend
    vault/                        HashiCorp Vault auth storage
tests/
  api/                            External black-box API tests (38 tests)
  bdd/                            BDD/Cucumber tests (178 feature files, 2,670 scenarios)
    features/                     135 top-level feature files
    features/mcp/                 43 MCP-specific feature files (379 scenarios)
    steps/                        11 step definition files
    configs/                      Per-backend config (memory, postgres, mysql, cassandra)
    docker-compose*.yml           9 Docker Compose files
  integration/                    Integration tests (66 tests across 6 files)
  concurrency/                    Concurrency tests (29 tests)
  storage/conformance/            Storage conformance suite (shared across all backends)
  migration/                      Migration tests (5 tests + shell scripts)
  compatibility/
    go/                           Go serializer tests (31 tests)
    go-serde/                     Go SerDe data contract + CSFLE tests (30 tests, 7 files)
    java/                         Java Confluent SerDe tests (wire compat + data contracts + CSFLE)
    python/                       Python serializer tests
api/
  openapi.yaml                    OpenAPI spec (embedded into binary)
```

## Running Tests

The Makefile provides targets for every test category. Most targets accept a `BACKEND` variable to select the storage backend. Database containers are started and stopped automatically.

```bash
make help    # Show all available targets
```

### Unit Tests

```bash
make test-unit
```

Runs `go test -race -v -timeout 5m ./...`. No Docker or external services required. This is the fastest feedback loop and SHOULD be run before every commit.

### BDD Tests

BDD tests use Gherkin feature files with [godog](https://github.com/cucumber/godog) and cover all API endpoints, compatibility rules, error codes, MCP tools/resources/prompts, permissions, encryption, exporters, data contracts, and operational resilience across 178 feature files and 2,670 scenarios.

```bash
make test-bdd                        # In-process, memory backend (no Docker)
make test-bdd-functional             # Same as above (explicit alias)
make test-bdd BACKEND=postgres       # Docker Compose with PostgreSQL
make test-bdd BACKEND=mysql          # Docker Compose with MySQL
make test-bdd BACKEND=cassandra      # Docker Compose with Cassandra
make test-bdd BACKEND=confluent      # Against Confluent Schema Registry 8.1.1
make test-bdd BACKEND=all            # All backends sequentially
```

Additional BDD targets for specific scenarios:

```bash
make test-bdd-db BACKEND=postgres    # In-process server with real DB
make test-bdd-db BACKEND=all         # In-process server with all DBs
make test-bdd-auth BACKEND=postgres  # BDD auth tests with real DB
make test-bdd-auth BACKEND=all       # BDD auth tests with all DBs
make test-bdd-kms BACKEND=memory     # BDD KMS encryption tests (Vault + OpenBao)
make test-bdd-kms BACKEND=all        # BDD KMS tests with all backends
```

The `confluent` backend starts a real Confluent Schema Registry via Docker Compose to verify wire compatibility.

### Integration Tests

Integration tests exercise the full stack against real database backends. The Makefile starts the required database container, runs the tests, and tears down the container.

```bash
make test-integration BACKEND=postgres
make test-integration BACKEND=mysql
make test-integration BACKEND=cassandra
make test-integration BACKEND=all
```

There is no memory backend for integration tests -- they require a database.

### Concurrency Tests

Tests concurrent schema registration, version updates, mixed operations, hot subject contention, schema idempotency, and ID uniqueness under parallel load.

```bash
make test-concurrency BACKEND=postgres
make test-concurrency BACKEND=mysql
make test-concurrency BACKEND=cassandra
make test-concurrency BACKEND=all
```

### Storage Conformance Tests

A shared test suite that verifies identical behavior across all storage backends. This catches cases where a backend silently differs from the expected contract.

```bash
make test-conformance                     # Memory (no Docker)
make test-conformance BACKEND=postgres    # PostgreSQL
make test-conformance BACKEND=mysql       # MySQL
make test-conformance BACKEND=cassandra   # Cassandra
make test-conformance BACKEND=all         # All backends
```

### Auth Tests

Each auth test suite starts its own service container (OpenLDAP, HashiCorp Vault, or Keycloak).

```bash
make test-ldap       # LDAP integration tests
make test-vault      # Vault integration tests
make test-oidc       # OIDC integration tests
make test-auth       # All three
```

### Migration Tests

Tests schema import and migration workflows, including Go unit tests and shell-based integration tests that exercise the binary directly.

```bash
make test-migration
```

### API Endpoint Tests

Starts a schema-registry binary with in-memory storage and runs external HTTP tests against it.

```bash
make test-api
```

### Compatibility Tests

Verifies wire compatibility with real Confluent serializer clients in Go, Java (4 Confluent versions), and Python (3 confluent-kafka versions). Requires Maven and Python 3 for the full suite; missing runtimes are skipped with a warning.

```bash
make test-compatibility
```


### Data Contract & CSFLE Tests

Java integration tests using real Confluent SerDe clients to verify end-to-end data contract rule execution and CSFLE field-level encryption against the running schema registry.

```bash
# Start schema registry
make run &

# Data contract tests (no KMS required)
cd tests/compatibility/java
mvn test -P confluent-8.1 -Dschema.registry.url=http://localhost:8081 -Dgroups=data-contracts

# CSFLE tests (requires Vault)
cd tests/bdd && docker compose -f docker-compose.kms.yml up -d vault && docker compose -f docker-compose.kms.yml run --rm setup-kms
cd ../../tests/compatibility/java
mvn test -P confluent-8.1 \
  -Dschema.registry.url=http://localhost:8081 \
  -Dgroups=csfle \
  -Dvault.url=http://localhost:18200 \
  -Dvault.token=test-root-token
```

Requires Maven and Java 17+. The tests use JUnit 5 tags (`data-contracts`, `csfle`) to separate data contract tests from CSFLE tests that require KMS infrastructure.

### Go SerDe Data Contract & CSFLE Tests

Go integration tests using `confluent-kafka-go/v2` (v2.8.0) schema registry client to verify data contract rule execution (CEL conditions, CEL_FIELD transforms, JSONata migration rules, global policies) and CSFLE field-level encryption via Vault Transit KMS. No CGO is required since only the `schemaregistry/*` sub-packages are imported.

**30 tests across 7 files** in `tests/compatibility/go-serde/`.

```bash
# Start schema registry
make run &

# Data contract tests only (no KMS required)
cd tests/compatibility/go-serde
SCHEMA_REGISTRY_URL=http://localhost:8081 go test -v -run "TestCel|TestMigration|TestGlobalPolicies" ./...

# CSFLE tests (requires Vault)
cd tests/bdd && docker compose -f docker-compose.kms.yml up -d vault && docker compose -f docker-compose.kms.yml run --rm setup-kms
cd ../../tests/compatibility/go-serde
SCHEMA_REGISTRY_URL=http://localhost:8081 \
  VAULT_URL=http://localhost:18200 \
  VAULT_TOKEN=test-root-token \
  go test -v -run "TestCsfle" -timeout 10m ./...

# All Go SerDe tests
SCHEMA_REGISTRY_URL=http://localhost:8081 \
  VAULT_URL=http://localhost:18200 \
  VAULT_TOKEN=test-root-token \
  go test -v -timeout 10m ./...
```

A convenience script is also available: `tests/compatibility/go-serde/run_tests.sh`.

### Full Test Suite

Run everything:

```bash
make test                    # All tests with default backend (memory where applicable)
make test BACKEND=all        # All tests against all backends
```

### Coverage

```bash
make test-coverage
```

Generates a coverage report at `./coverage/coverage.html`.

## Code Quality

```bash
make lint           # Run golangci-lint (configuration in .golangci.yaml)
make fmt            # Format code with gofmt
```

The linter configuration is in `.golangci.yaml` at the repository root. CI runs both `golangci-lint`, `go vet`, `gosec`, and Trivy on every push.

## Running the Server Locally

Start with the default in-memory backend:

```bash
make run
```

Or with hot reload during development (requires [air](https://github.com/air-verse/air)):

```bash
make dev
```

To run with a custom configuration file:

```bash
./build/schema-registry --config config.yaml
```

To enable the MCP server alongside the REST API, add to your config:

```yaml
mcp:
  enabled: true
  host: "127.0.0.1"
  port: 9081
```

Or via environment variable:

```bash
SCHEMA_REGISTRY_MCP_ENABLED=true ./build/schema-registry --config config.yaml
```

## API Documentation

### OpenAPI Spec

The OpenAPI specification lives at `api/openapi.yaml` and is embedded into the binary at compile time. A test (`TestOpenAPISpecMatchesRoutes`) enforces bidirectional sync between the spec and the router -- every route in the router must appear in the spec and vice versa.

### Swagger UI

Set `server.docs_enabled: true` in the configuration file to serve Swagger UI at `/docs`.

### Static Documentation

Generate a static HTML version of the API documentation using Redocly:

```bash
make docs-api
```

Output is written to `docs/api/index.html`.

### MCP Reference

Generate the MCP API reference (all tools, resources, and prompts) from live server introspection:

```bash
make docs-mcp
```

Output is written to `docs/mcp-reference.md`.

## Docker

### Build the Image

```bash
make docker-build
```

This builds a multi-stage image (Go build stage, then Alpine runtime) and tags it as `axonops/schema-registry:latest` and `axonops/schema-registry:<version>`.

### Run with Docker

```bash
make docker-run
```

Or manually:

```bash
docker run -p 8081:8081 axonops/schema-registry:latest
```

To expose the MCP server:

```bash
docker run -p 8081:8081 -p 9081:9081 \
  -e SCHEMA_REGISTRY_MCP_ENABLED=true \
  -e SCHEMA_REGISTRY_MCP_HOST=0.0.0.0 \
  axonops/schema-registry:latest
```

## CI Pipeline

GitHub Actions runs 37 jobs on every push to `main` or `feature/**` branches:

- **Build & Unit Tests:** Compile binary + all test binaries, run unit tests with coverage
- **Static Analysis:** golangci-lint, go vet, gosec, Trivy vulnerability scanner
- **Docker:** Multi-stage Docker image build verification
- **Database Tests:** Integration + concurrency against PostgreSQL 15, MySQL 8, Cassandra 5.0
- **Conformance:** Storage conformance against memory, PostgreSQL, MySQL, Cassandra (4 jobs)
- **API:** Black-box API endpoint tests
- **Auth:** LDAP (OpenLDAP), Vault (HashiCorp Vault 1.15), OIDC (Keycloak 24.0)
- **Migration:** Import/migration tests
- **BDD (Docker Compose):** memory, PostgreSQL, MySQL, Cassandra, Confluent 8.1.1 (5 jobs)
- **BDD Functional:** In-process functional tests
- **BDD DB:** In-process with real DB: PostgreSQL, MySQL, Cassandra (3 jobs)
- **BDD Auth:** Auth BDD with real DB: PostgreSQL, MySQL, Cassandra (3 jobs)
- **BDD KMS:** KMS encryption: memory, PostgreSQL, MySQL, Cassandra (4 jobs)
- **Compatibility:** Go + Java (4 Confluent versions) + Python (3 versions)
- **Data Contracts:** Java SerDe, Go SerDe, Python SerDe — data contract + CSFLE with Vault (3 jobs)

The `build` job compiles all test binaries and uploads them as artifacts. Downstream jobs download pre-compiled binaries to avoid redundant compilation.

## Code Conventions

- Standard Go formatting (`gofmt`), enforced by golangci-lint
- Structured logging via `log/slog`
- Context-first function signatures: `func (s *Store) Method(ctx context.Context, ...) error`
- Multi-tenant context: all storage/registry methods take `registryCtx string` (default `"."`)
- Sentinel errors for storage operations, compared with `errors.Is()`
- Build tags to separate test categories: `//go:build integration`, `//go:build bdd`, `//go:build api`, `//go:build concurrency`, `//go:build conformance`, `//go:build ldap`, `//go:build vault`, `//go:build oidc`, `//go:build migration`
- Schema fingerprints use SHA-256 for content-addressed deduplication
- Soft-delete with a boolean flag, not physical deletion
- Global configuration uses an empty string (`""`) as the subject key
- The import API preserves original schema IDs for Confluent migration
- Both REST handlers and MCP tools call `registry.Registry` for all business logic (shared layer)
- BDD-first testing: every feature MUST have BDD coverage; Makefile targets are the canonical way to run tests

## Contributing

1. Fork the repository
2. Create a feature branch from `main`
3. Write tests for new functionality (BDD feature files + unit tests)
4. Ensure unit tests and linting pass: `make test-unit && make lint`
5. Run BDD tests: `make test-bdd`
6. Submit a pull request

For larger changes, open an issue first to discuss the approach.

## Related Documentation

- [Getting Started](getting-started.md) -- register your first schemas and integrate with Kafka clients
- [Configuration](configuration.md) -- full YAML configuration reference
- [Testing Strategy](testing.md) -- detailed testing philosophy, all test layers, how to write tests
- [Storage Backends](storage-backends.md) -- setup guides for PostgreSQL, MySQL, Cassandra, and in-memory
- [Authentication](authentication.md) -- API keys, LDAP, OIDC, JWT, and RBAC
- [Schema Types](schema-types.md) -- Avro, Protobuf, and JSON Schema support details
- [Compatibility](compatibility.md) -- compatibility levels and checking behavior
- [MCP Server](mcp.md) -- MCP server for AI-assisted schema management
- [API Reference](api-reference.md) -- complete endpoint documentation
