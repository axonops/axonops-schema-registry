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
  - [Full Test Suite](#full-test-suite)
  - [Coverage](#coverage)
- [Code Quality](#code-quality)
- [Running the Server Locally](#running-the-server-locally)
- [API Documentation](#api-documentation)
  - [OpenAPI Spec](#openapi-spec)
  - [Swagger UI](#swagger-ui)
  - [Static Documentation](#static-documentation)
- [Docker](#docker)
  - [Build the Image](#build-the-image)
  - [Run with Docker](#run-with-docker)
- [CI Pipeline](#ci-pipeline)
- [Code Conventions](#code-conventions)
- [Contributing](#contributing)
- [Related Documentation](#related-documentation)

## Prerequisites

- **Go 1.24+** ([download](https://go.dev/dl/))
- **Docker** (for integration, BDD, auth, and concurrency tests)
- **golangci-lint** ([install](https://golangci-lint.run/welcome/install/)) for linting
- **Node.js** (optional, for generating static API documentation with Redocly)

Optional tools for the full compatibility test suite:

- **Maven** (for Java Confluent serializer tests)
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
internal/
  api/                            HTTP server, handlers, types, OpenAPI
  auth/                           Authentication, RBAC, JWT, rate limiting
  cache/                          Caching layer
  compatibility/                  Compatibility checkers
    avro/                         Avro compatibility
    jsonschema/                   JSON Schema compatibility
    protobuf/                     Protobuf compatibility
  config/                         Configuration loading
  registry/                       Core business logic
  schema/                         Schema parsers
    avro/                         Avro parser
    jsonschema/                   JSON Schema parser
    protobuf/                     Protobuf parser
  storage/                        Storage interface and backends
    memory/                       In-memory backend
    postgres/                     PostgreSQL backend
    mysql/                        MySQL backend
    cassandra/                    Cassandra backend
    vault/                        HashiCorp Vault auth storage
  metrics/                        Prometheus metrics
tests/
  bdd/                            BDD/Cucumber tests (76 feature files, 1,387 scenarios)
  integration/                    Integration tests
  concurrency/                    Concurrency tests
  storage/conformance/            Storage conformance suite
  migration/                      Migration tests
  api/                            External API tests
  compatibility/                  Multi-language compatibility tests (Go/Java/Python)
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

Runs `go test -race -v -timeout 5m ./internal/...`. No Docker or external services required. This is the fastest feedback loop and should be run before every commit.

### BDD Tests

BDD tests use Gherkin feature files with [godog](https://github.com/cucumber/godog) and cover all API endpoints, compatibility rules, error codes, and operational resilience across 76 feature files and 1,387 scenarios.

```bash
make test-bdd                        # In-process, memory backend (no Docker)
make test-bdd BACKEND=postgres       # Docker Compose with PostgreSQL
make test-bdd BACKEND=mysql          # Docker Compose with MySQL
make test-bdd BACKEND=cassandra      # Docker Compose with Cassandra
make test-bdd BACKEND=confluent      # Against Confluent Schema Registry
make test-bdd BACKEND=all            # All backends sequentially
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

## CI Pipeline

GitHub Actions runs the following on every push:

- Static analysis: golangci-lint, go vet, gosec, Trivy
- Unit tests with coverage
- Integration tests (PostgreSQL, MySQL, Cassandra)
- BDD tests (memory, PostgreSQL, MySQL, Cassandra, Confluent)
- Storage conformance tests (memory, PostgreSQL, MySQL, Cassandra)
- API endpoint tests
- Auth tests (LDAP, Vault, OIDC)
- Migration tests
- Compatibility tests (Go, Java, Python)
- Docker image build

## Code Conventions

- Standard Go formatting (`gofmt`), enforced by golangci-lint
- Structured logging via `log/slog`
- Context-first function signatures: `func (s *Store) Method(ctx context.Context, ...) error`
- Sentinel errors for storage operations, compared with `errors.Is()`
- Build tags to separate test categories: `//go:build integration`, `//go:build bdd`, `//go:build api`, `//go:build concurrency`, `//go:build conformance`
- Schema fingerprints use SHA-256 for content-addressed deduplication
- Soft-delete with a boolean flag, not physical deletion
- Global configuration uses an empty string (`""`) as the subject key
- The import API preserves original schema IDs for Confluent migration

## Contributing

1. Fork the repository
2. Create a feature branch from `main`
3. Write tests for new functionality
4. Ensure unit tests and linting pass: `make test-unit && make lint`
5. Submit a pull request

For larger changes, open an issue first to discuss the approach.

## Related Documentation

- [Getting Started](getting-started.md) -- register your first schemas and integrate with Kafka clients
- [Configuration](configuration.md) -- full YAML configuration reference
- [Storage Backends](storage-backends.md) -- setup guides for PostgreSQL, MySQL, Cassandra, and in-memory
- [Authentication](authentication.md) -- API keys, LDAP, OIDC, JWT, and RBAC
- [Schema Types](schema-types.md) -- Avro, Protobuf, and JSON Schema support details
- [Compatibility](compatibility.md) -- compatibility levels and checking behavior
- [API Reference](api-reference.md) -- complete endpoint documentation
