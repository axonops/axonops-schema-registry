# CLAUDE.md — AxonOps Schema Registry

## Project Overview

AxonOps Schema Registry is a Go-based, API-compatible drop-in replacement for Confluent's Kafka Schema Registry. It supports Avro, Protobuf, and JSON Schema types with multiple storage backends, built-in enterprise security, multi-tenant contexts, data contracts (metadata/ruleSet), client-side field-level encryption (CSFLE via DEK/KEK), schema exporters (Schema Linking), and an MCP (Model Context Protocol) server for AI-assisted schema management.

- **Repository:** https://github.com/axonops/axonops-schema-registry
- **License:** Apache 2.0
- **Language:** Go 1.26+
- **Go Module:** `github.com/axonops/axonops-schema-registry`

## Documentation standards

## Writing Style Guidelines

### RFC 2119 Compliance

All documentation **MUST** adhere to RFC 2119 for requirement level terminology.

The key words **"MUST"**, **"MUST NOT"**, **"REQUIRED"**, **"SHALL"**, **"SHALL NOT"**, **"SHOULD"**, **"SHOULD NOT"**, **"RECOMMENDED"**, **"MAY"**, and **"OPTIONAL"** in this documentation **ARE TO BE INTERPRETED AS DESCRIBED IN RFC 2119**.

Use these terms precisely:

- **MUST / REQUIRED / SHALL**: Absolute requirements for safety or correctness.
- **MUST NOT / SHALL NOT**: Absolute prohibitions that prevent harm or corruption.
- **SHOULD / RECOMMENDED**: Strong recommendations; exceptions MAY exist but the implications MUST be understood and carefully weighed.
- **SHOULD NOT / NOT RECOMMENDED**: Discouraged practices; MAY be acceptable in rare, well-understood cases.
- **MAY / OPTIONAL**: Truly optional choices that do not break interoperability or correctness when omitted.

When documenting procedures with safety, data integrity, or availability implications, you **MUST** use RFC 2119 terms to indicate requirement levels clearly.

### Technical Content Standards

- Use clear, concise language suitable for DevOps engineers and database administrators.
- Include specific error messages, stack traces, and log excerpts when relevant.
- Provide context about versions and configurations affected.
- Structure content with clear headings (H2, H3, H4) for scannability.
- Use code blocks for configuration examples, commands, and log output.

### Formatting Requirements

- **Commands**: Use `backticks` for inline commands and triple backticks for code blocks.
- **File paths & config options**: Use `backticks` for file paths and configuration parameters.
- **Error messages**: Use blockquotes (`>`) for error messages and critical warnings.
- **Version numbers**: Always specify exact versions (for example, `Cassandra 5.0.5`).
- **RFC 2119 terms**: Capitalize RFC 2119 keywords exactly as specified (MUST, SHOULD, MAY, etc.).

### SDRR Framework

When creating troubleshooting playbooks, follow the SDRR framework:

- **Symptoms**: Observable behaviors and error messages.
- **Diagnostics**: Steps to identify root cause.
- **Resolution**: Specific actions to fix the issue.
- **Root Cause Analysis**: Explanation of why the issue occurred and how to prevent recurrence.

## Adding New Content

### Creating Troubleshooting Playbooks

1. Create new markdown files in `docs/troubleshooting/`.
2. Name files descriptively, for example: `paxos-repair-bootstrap-failure.md`.
3. Add frontmatter with title and description.
4. Include metadata about affected versions and configurations.
5. Cross-reference related documentation (architecture, operations, performance, etc.).

### Updating Existing Pages

- Check for existing content that SHOULD be updated or expanded instead of duplicating.
- Maintain consistency with established patterns and terminology.
- Update navigation in `mkdocs.yml` if adding new sections or pages.

### Code Examples and Commands

- Commands and configuration snippets **SHOULD** be tested whenever feasible.
- Provide both JMX and `cassandra.yaml` configuration examples when applicable.
- Include warnings about potential risks or side effects using RFC 2119 language.
- Specify where commands MUST be run (for example, "on the joining node", "on all nodes in the DC", "on all nodes in the cluster").

## Common Patterns

### Configuration Options

When documenting configuration parameters, you SHOULD:

- Show YAML syntax with proper indentation.
- Explain defaults and when they SHOULD or MUST be changed.
- Document risks and trade-offs using RFC 2119 terminology.
- Provide examples for different scenarios (for example, LWT vs non-LWT keyspaces).

### Command Sequences

For multi-step procedures:

1. Use numbered lists for sequential steps.
2. Include expected output or verification steps where useful.
3. Add troubleshooting notes for common failures at each step.
4. Specify rollback procedures when applicable.

### Version-Specific Notes

- Clearly mark content that applies to specific versions only.
- Use callouts for version-specific warnings (for example, "Cassandra 5.0.5+ only").
- Document behavioral changes across versions when they affect operational procedures.

## Architecture

```
cmd/schema-registry/              Entry point, CLI (cobra)
cmd/generate-mcp-docs/            MCP reference doc generator
internal/
  analysis/                       Shared analysis (field extraction, quality scoring, fuzzy matching)
  api/
    server.go                     HTTP server, chi router, middleware chain, all route registration
    handlers/
      handlers.go                 Schema/subject/config/mode/compat endpoint handlers
      admin.go                    User and API key admin handlers
      account.go                  Self-service account handlers
      analysis.go                 26 REST analysis endpoint handlers
      dek.go                      DEK/KEK encryption handlers
      exporter.go                 Exporter (Schema Linking) handlers
    types/
      types.go                    All request/response structs + Confluent error codes
  auth/
    auth.go                       Authenticator middleware
    rbac.go                       Role-based access control
    service.go                    User/API key business logic
    jwt.go                        JWT auth
    ratelimit.go                  Rate limiting
    ldap.go, oidc.go, tls.go
    audit.go                      Audit logger, AuditOutput interface, multi-output fan-out
    audit_file.go                 FileOutput with lumberjack rotation
    audit_syslog.go               SyslogOutput (RFC 5424, TCP/UDP/TLS via srslog)
    audit_webhook.go              WebhookOutput with batching, retry, overflow drop
    audit_format.go               CEF format serialization
  compatibility/
    checker.go                    Compatibility checker registry
    modes.go                      7 compat modes (NONE, BACKWARD, FORWARD, FULL + _TRANSITIVE)
    result.go                     Compatibility check result types
    avro/checker.go               Avro compatibility
    jsonschema/checker.go         JSON Schema compatibility
    protobuf/checker.go           Protobuf compatibility
  config/
    config.go                     YAML config loading with env var overrides
  context/
    context.go                    Multi-tenant context management
  exporter/
    exporter.go                   Schema exporter (Schema Linking) logic
  kms/
    provider.go                   KMS provider interface
    vault/provider.go             HashiCorp Vault Transit KMS
    aws/provider.go               AWS KMS
    azure/provider.go             Azure Key Vault
    gcp/provider.go               GCP Cloud KMS
  mcp/
    server.go                     MCP server struct, New(), Start(), options pattern, server instructions
    tools.go                      Tool registration, addToolIfAllowed, instrumentedHandler, resolveContext
    tools_schema.go               Schema read/write tools (13)
    tools_config.go               Config tools (6)
    tools_write.go                Write operation tools (4)
    tools_context.go              Context tools (2)
    tools_admin.go                Admin tools — users, API keys, KEK, DEK (15)
    tools_dek.go                  DEK/KEK encryption tools (13)
    tools_metadata.go             Metadata tools (12)
    tools_exporter.go             Exporter tools (11)
    tools_validation.go           Validation tools (11)
    tools_intelligence.go         Intelligence/analysis tools (9)
    tools_comparison.go           Comparison tools (6)
    resources.go                  9 static resources + 22 templated resources
    glossary.go                   16 glossary static resources
    prompts.go                    33 prompt handlers
    permissions.go                14 scopes, 5 presets, tool-to-scope mapping
    content/                      Embedded .md files via embed.FS
      glossary/                   16 glossary markdown files
      prompts/                    35 prompt markdown files
  metrics/
    metrics.go                    Prometheus metrics (REST + MCP + audit outputs)
  registry/
    registry.go                   Core business logic (shared by REST and MCP)
  rules/
    engine.go                     Data contract rules engine (CEL, JSONata)
    validator.go                  Rule validation
  schema/
    types.go                      Parser + ParsedSchema interfaces, parser Registry
    avro/parser.go                Avro parser
    jsonschema/parser.go          JSON Schema parser
    protobuf/parser.go + resolver.go  Protobuf parser + reference resolution
  storage/
    storage.go                    Storage interface (71 methods), types, sentinel errors
    factory.go                    Factory pattern: Register(), Create(), StorageType constants
    memory/store.go               In-memory backend
    postgres/store.go + migrations.go   PostgreSQL backend
    mysql/store.go + migrations.go      MySQL backend
    cassandra/store.go + migrations.go  Cassandra backend
    vault/                        HashiCorp Vault auth storage
tests/
  api/api_test.go                 External HTTP tests (38 tests, build tag: api)
  bdd/                            BDD/Cucumber tests (192 feature files, 2,843 scenarios)
    bdd_test.go                   Test runner (godog init, backend selection)
    features/                     148 top-level .feature files
    features/mcp/                 44 MCP-specific .feature files
    steps/                        13 step definition files
    configs/                      9 per-backend/feature config files
    docker-compose*.yml           17 Docker Compose files (base, backends, KMS, audit, auth, mcp)
    docker/webhook-receiver/      Webhook receiver container for audit output BDD tests
    docker/syslog-ng/             syslog-ng TLS config for audit output BDD tests
    certs/                        Self-signed ECDSA P256 certs for TLS syslog testing
  integration/
    integration_test.go           httptest integration (32 tests, build tag: integration)
    analysis_integration_test.go  Analysis integration (10 tests)
    auth_integration_test.go      Auth integration (7 tests)
    ldap_integration_test.go      LDAP integration (5 tests, build tag: ldap)
    oidc_integration_test.go      OIDC integration (6 tests, build tag: oidc)
    vault_integration_test.go     Vault integration (6 tests, build tag: vault)
  concurrency/concurrency_test.go Concurrent operations (29 tests, build tag: concurrency)
  storage/conformance/            Storage conformance suite (shared across all backends)
  migration/migration_test.go     Migration/import (5 tests, build tag: migration)
  compatibility/                  Multi-language compat tests (Go/Java/Python, docker-compose)
    go/                           Go serializer tests (31 tests)
    go-serde/                     Go SerDe data contract + CSFLE tests (30 tests, 7 files)
    java/                         Java Confluent SerDe tests (wire compat + data contracts + CSFLE)
    python/                       Python serializer tests
api/
  openapi.yaml                    OpenAPI spec (embedded into binary via embed.go)
Dockerfile                        Multi-stage build (golang:1.26-alpine -> alpine:3.19)
```

**Test inventory: 82 test files, 1,436 test functions, ~48,000 lines of Go test code + ~49,000 lines of Gherkin.**

## Key Interfaces

### Storage Interface (`internal/storage/storage.go`)

The `Storage` interface embeds `AuthStorage`, `EncryptionStorage`, and `ExporterStorage` — 71 methods total:

**Schema ops:** CreateSchema, GetSchemaByID, GetSchemaBySubjectVersion, GetSchemasBySubject, GetSchemaByFingerprint, GetSchemaByGlobalFingerprint, GetLatestSchema, DeleteSchema, ListSchemas

**Subject ops:** ListSubjects, DeleteSubject, SubjectExists

**Config ops:** GetConfig, SetConfig, DeleteConfig, GetGlobalConfig, SetGlobalConfig, DeleteGlobalConfig

**Mode ops:** GetMode, SetMode, DeleteMode, GetGlobalMode, SetGlobalMode, DeleteGlobalMode

**ID ops:** NextID, GetMaxSchemaID, SetNextID

**Import ops:** ImportSchema (with specific ID, for Confluent migration)

**Lookups:** GetReferencedBy, GetSubjectsBySchemaID, GetVersionsBySchemaID

**Context ops:** ListContexts

**Encryption ops (KEK):** CreateKEK, GetKEK, UpdateKEK, DeleteKEK, UndeleteKEK, ListKEKs

**Encryption ops (DEK):** CreateDEK, GetDEK, ListDEKs, ListDEKVersions, DeleteDEK, UndeleteDEK, UpdateDEK

**Exporter ops:** CreateExporter, GetExporter, UpdateExporter, DeleteExporter, ListExporters, GetExporterStatus, SetExporterStatus, GetExporterConfig, UpdateExporterConfig

**Auth ops (via AuthStorage):** CreateUser, GetUserByID, GetUserByUsername, UpdateUser, DeleteUser, ListUsers, CreateAPIKey, GetAPIKeyByID, GetAPIKeyByHash, GetAPIKeyByUserAndName, UpdateAPIKey, DeleteAPIKey, ListAPIKeys, ListAPIKeysByUserID, UpdateAPIKeyLastUsed

**Lifecycle:** Close, IsHealthy

All schema, subject, config, mode, and ID operations take a `registryCtx string` parameter for multi-tenant context isolation.

**Sentinel errors:** ErrNotFound, ErrSubjectNotFound, ErrSchemaNotFound, ErrVersionNotFound, ErrInvalidVersion, ErrSubjectDeleted, ErrSchemaExists, ErrSchemaIDConflict, ErrUserNotFound, ErrUserExists, ErrAPIKeyNotFound, ErrAPIKeyExists, ErrAPIKeyNameExists, ErrInvalidAPIKey, ErrAPIKeyExpired, ErrAPIKeyDisabled, ErrUserDisabled, ErrInvalidRole, ErrPermissionDenied

### Schema Parser Interface (`internal/schema/types.go`)

```go
type Parser interface {
    Parse(schemaStr string, references []storage.Reference) (ParsedSchema, error)
    Type() storage.SchemaType
}
type ParsedSchema interface {
    Type() storage.SchemaType
    CanonicalString() string
    Fingerprint() string
    RawSchema() interface{}
}
```

### Schema References (`internal/storage/storage.go`)

```go
type Reference struct {
    Name    string `json:"name"`
    Subject string `json:"subject"`
    Version int    `json:"version"`
}
```

Used for: Avro named type references, Protobuf imports, JSON Schema $ref across subjects.

### Factory Pattern (`internal/storage/factory.go`)

```go
storage.Register(storageType, factoryFunc)  // Called via init() in each backend
store, err := storage.Create(storageType, configMap)
```

### Registry (shared business logic — `internal/registry/registry.go`)

Both REST handlers and MCP tools call `registry.Registry` for all business logic. This is the single source of truth for schema operations, compatibility checking, and configuration management. Do not duplicate business logic in handlers or tools.

## MCP Server (`internal/mcp/`)

The MCP server provides AI assistants with structured access to the schema registry via the Model Context Protocol.

- **Transport:** Streamable HTTP only (port 9081, configurable)
- **SDK:** `github.com/modelcontextprotocol/go-sdk` (official Go SDK)
- **Spec version:** 2025-11-25

### MCP Components

| Component | Count | Location |
|-----------|-------|----------|
| Tools | 107 | 12 files (`internal/mcp/tools*.go`) |
| Resources | 47 (25 static + 22 templated) | `resources.go` + `glossary.go` |
| Prompts | 33 | `prompts.go` |
| Permission Scopes | 14 | `permissions.go` |
| Permission Presets | 5 | readonly, developer, operator, admin, full |
| Embedded content | 51 files | `content/glossary/` (16) + `content/prompts/` (35) |
| Unit tests | 208 | 8 test files |
| BDD scenarios | 379 | 43 feature files in `tests/bdd/features/mcp/` |

### Key MCP Patterns

- `addToolIfAllowed()` checks permission scopes + read-only mode + tool policy before registering a tool
- `instrumentedHandler()` wraps every tool with Prometheus metrics, slog logging, and audit trail
- `resolveContext()` extracts `context` parameter from tool arguments, falls back to `DefaultContext`
- `resolveResourceContext()` does the same for resources
- Glossary and prompt content is embedded via `embed.FS` from `internal/mcp/content/`
- Permission scopes use 4-level resolution: preset > scopes > read_only > tool_policy > default (full)
- System tools (empty scope string) are always allowed regardless of permission configuration

## API Routes (Confluent Wire-Compatible)

### Schema Operations
```
GET  /schemas/types                                         -> Supported schema types
GET  /schemas                                               -> List schemas (with filters)
GET  /schemas/ids/{id}                                      -> Schema by global ID
GET  /schemas/ids/{id}/schema                               -> Raw schema string by ID
GET  /schemas/ids/{id}/subjects                             -> Subjects using this schema ID
GET  /schemas/ids/{id}/versions                             -> Subject-version pairs for schema ID
POST /subjects/{subject}/versions                           -> Register schema
POST /subjects/{subject}                                    -> Lookup (check if schema exists)
```

### Subject Operations
```
GET    /subjects                                            -> List subjects (?deleted=true)
GET    /subjects/{subject}/versions                         -> List versions
GET    /subjects/{subject}/versions/{version}               -> Get version detail
GET    /subjects/{subject}/versions/{version}/schema        -> Raw schema by version
GET    /subjects/{subject}/versions/{version}/referencedby  -> Referencing schemas
DELETE /subjects/{subject}                                  -> Soft-delete (?permanent=true)
DELETE /subjects/{subject}/versions/{version}               -> Delete version
```

### Configuration & Mode
```
GET/PUT/DELETE  /config                                     -> Global compatibility
GET/PUT/DELETE  /config/{subject}                           -> Per-subject compatibility
GET/PUT         /mode                                       -> Global mode
GET/PUT/DELETE  /mode/{subject}                             -> Per-subject mode
```

### Compatibility Checking
```
POST /compatibility/subjects/{subject}/versions/{version}   -> Check against specific version
POST /compatibility/subjects/{subject}/versions             -> Check against all versions
```

### Encryption (DEK/KEK)
```
GET/POST       /dek-registry/v1/keks                        -> List/create KEKs
GET/PUT/DELETE /dek-registry/v1/keks/{name}                 -> KEK CRUD
POST           /dek-registry/v1/keks/{name}/undelete        -> Undelete KEK
GET/POST       /dek-registry/v1/keks/{name}/deks            -> List/create DEKs
GET/DELETE     /dek-registry/v1/keks/{name}/deks/{subject}  -> DEK CRUD
POST           /dek-registry/v1/keks/{name}/deks/{subject}/undelete -> Undelete DEK
GET            /dek-registry/v1/keks/{name}/deks/{subject}/versions -> DEK versions
```

### Exporters (Schema Linking)
```
GET/POST       /exporters                                   -> List/create exporters
GET/PUT/DELETE /exporters/{name}                            -> Exporter CRUD
PUT            /exporters/{name}/pause                      -> Pause exporter
PUT            /exporters/{name}/resume                     -> Resume exporter
PUT            /exporters/{name}/reset                      -> Reset exporter
GET            /exporters/{name}/status                     -> Exporter status
GET/PUT        /exporters/{name}/config                     -> Exporter config
```

### Contexts
```
GET  /contexts                                              -> List contexts
```

### Import, Metadata & Admin
```
POST /import/schemas                                        -> Bulk import preserving IDs
GET  /                                                      -> Health check
GET  /metrics                                               -> Prometheus metrics
GET  /v1/metadata/id, /v1/metadata/version                  -> Metadata endpoints
/admin/users/*, /admin/apikeys/*, /admin/roles              -> RBAC-protected admin
/admin/account, /admin/account/password                     -> Self-service account
```

### Analysis (AxonOps-only)
```
GET  /analysis/subjects/{subject}/fields                    -> Extract fields from latest schema
POST /analysis/schemas/fields                               -> Extract fields from schema body
POST /analysis/schemas/quality                              -> Score schema quality
GET  /analysis/subjects/{subject}/quality                   -> Score subject quality
GET  /analysis/subjects/{subject}/diff                      -> Diff between versions
POST /analysis/schemas/validate                             -> Validate schema
GET  /analysis/search/fields                                -> Find schemas by field name
GET  /analysis/search/subjects                              -> Fuzzy search subjects
GET  /analysis/consistency/fields                           -> Check field consistency
GET  /analysis/statistics                                   -> Registry-wide statistics
```

## Content Types & Error Format

Content-Type: `application/vnd.schemaregistry.v1+json` (also accepts `application/json`)

```json
{"error_code": 40401, "message": "Subject 'foo' not found"}
```

Error codes: 40401 (subject not found), 40402 (version not found), 40403 (schema not found), 42201 (invalid schema), 42203 (invalid compat level), 409 (incompatible), 50001 (internal error).

## Compatibility Levels

NONE, BACKWARD (default), BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE

## Configuration

Config file loaded via `--config` flag. All fields support env var overrides with `SCHEMA_REGISTRY_` prefix (nested fields use `_` separator, e.g. `SCHEMA_REGISTRY_MCP_ENABLED=true`).

```yaml
server:
  host: "0.0.0.0"
  port: 8081
  docs_enabled: false
storage:
  type: memory   # memory | postgresql | mysql | cassandra
  postgresql:
    host: localhost
    port: 5432
    database: schemaregistry
    user: schemaregistry
    password: schemaregistry
    ssl_mode: disable
  mysql:
    host: localhost
    port: 3306
    database: schemaregistry
    user: schemaregistry
    password: schemaregistry
  cassandra:
    hosts: [localhost]
    port: 9042
    keyspace: schemaregistry
    consistency: ONE
    migrate: true
compatibility:
  default_level: BACKWARD
logging:
  level: info
  format: json
security:
  auth:
    enabled: false
    methods: [api_key, basic]
mcp:
  enabled: false
  host: "127.0.0.1"
  port: 9081
  read_only: false
  auth_token: ""
  default_context: "."
  tool_policy: "allow_all"     # allow_all | deny_list | allow_list
  allowed_tools: []
  denied_tools: []
  permission_preset: ""        # readonly | developer | operator | admin | full
  permission_scopes: []        # fine-grained: schema_read, schema_write, config_read, etc.
```

## Build & Test

```bash
# Build
make build                     # Build binary -> ./build/schema-registry
make build-all                 # Cross-compile (linux/darwin, amd64/arm64)

# Lint & format
make fmt                       # gofmt -s -w .
make lint                      # golangci-lint run ./...

# Run
make run                       # Build + run with in-memory storage
make dev                       # Hot reload (requires air)

# Unit tests (fast, no Docker)
make test-unit                 # go test -race -v -timeout 5m ./...

# BDD tests
make test-bdd                  # In-process, memory backend (no Docker)
make test-bdd-functional       # Same as above (explicit name)
make test-bdd BACKEND=postgres # Docker Compose with PostgreSQL
make test-bdd BACKEND=all      # All backends: memory, postgres, mysql, cassandra, confluent
make test-bdd-db BACKEND=all   # In-process server with real DB (postgres, mysql, cassandra)
make test-bdd-auth BACKEND=all # BDD auth tests with real DB
make test-bdd-kms BACKEND=all  # BDD KMS encryption tests (Vault + OpenBao)
make test-bdd-rest-audit       # REST audit BDD tests
make test-bdd-mcp-audit        # MCP audit BDD tests
make test-bdd-mcp-metrics      # MCP metrics BDD tests
make test-bdd-audit-outputs    # Audit outputs BDD (file + syslog TLS + webhook)

# Integration & concurrency (require DB)
make test-integration BACKEND=postgres
make test-concurrency BACKEND=postgres

# Storage conformance
make test-conformance                    # Memory (no Docker)
make test-conformance BACKEND=all        # All backends

# Auth tests
make test-ldap                 # LDAP (starts OpenLDAP)
make test-vault                # Vault (starts HashiCorp Vault)
make test-oidc                 # OIDC (starts Keycloak)
make test-auth                 # All three

# Other
make test-api                  # Black-box API endpoint tests
make test-migration            # Migration/import tests
make test-compatibility        # Go/Java/Python Confluent serializer tests
make test-coverage             # Unit tests with HTML coverage report

# Everything
make test                      # ALL tests
make test BACKEND=all          # ALL tests against ALL backends

# Documentation
make docs-api                  # Generate REST API docs from OpenAPI spec
make docs-mcp                  # Generate MCP API reference from live introspection
```

## CI Pipeline (`.github/workflows/ci.yaml`)

43 jobs run on every push to `main` or `feature/**` branches:

| Stage | Jobs |
|-------|------|
| **Build** | Compile binary + all test binaries, run unit tests with coverage, upload artifacts |
| **Static Analysis** | golangci-lint, go vet, gosec, Trivy vulnerability scanner |
| **Gate** | Dependency gate — all downstream jobs wait for build + lint + security |
| **Docker** | Multi-stage Docker image build verification |
| **Database Tests** | Integration + concurrency against PostgreSQL 15, MySQL 8, Cassandra 5.0 |
| **Conformance** | Storage conformance against memory, PostgreSQL, MySQL, Cassandra (4 jobs) |
| **API** | Black-box API endpoint tests |
| **Auth** | LDAP (OpenLDAP), Vault (HashiCorp Vault 1.15), OIDC (Keycloak 24.0) |
| **Migration** | Import/migration tests |
| **BDD (Docker)** | memory, PostgreSQL, MySQL, Cassandra, Confluent 8.1.1 (5 jobs) |
| **BDD Functional** | In-process functional tests |
| **BDD DB** | In-process with real DB: PostgreSQL, MySQL, Cassandra (3 jobs) |
| **BDD Auth** | Auth BDD with real DB: PostgreSQL, MySQL, Cassandra (3 jobs) |
| **BDD KMS** | KMS encryption: memory, PostgreSQL, MySQL, Cassandra (4 jobs) |
| **BDD Audit** | REST audit, MCP audit, MCP metrics, audit outputs (syslog TLS + webhook) (4 jobs) |
| **Compatibility** | Go + Java (4 Confluent versions) + Python (3 versions) serializer tests |
| **Data Contracts** | Java SerDe, Go SerDe, Python SerDe — data contract + CSFLE with Vault (3 jobs) |

The `build` job compiles all test binaries and uploads them as artifacts. Downstream jobs download pre-compiled binaries to avoid redundant compilation.

## Key Dependencies

| Purpose | Package |
|---------|---------|
| Router | github.com/go-chi/chi/v5 |
| Avro | github.com/hamba/avro/v2 |
| Protobuf | github.com/bufbuild/protocompile |
| JSON Schema | github.com/santhosh-tekuri/jsonschema/v5 |
| PostgreSQL | github.com/lib/pq |
| MySQL | github.com/go-sql-driver/mysql |
| Cassandra | github.com/apache/cassandra-gocql-driver/v2 |
| Vault | github.com/hashicorp/vault/api |
| OIDC | github.com/coreos/go-oidc/v3 |
| LDAP | github.com/go-ldap/ldap/v3 |
| Metrics | github.com/prometheus/client_golang |
| CLI | github.com/spf13/cobra |
| BDD | github.com/cucumber/godog |
| MCP SDK | github.com/modelcontextprotocol/go-sdk |
| File rotation | gopkg.in/natefinch/lumberjack.v2 |
| Syslog RFC 5424 | github.com/RackSec/srslog |

## API Documentation

- **OpenAPI spec:** `api/openapi.yaml` — the single source of truth for the REST API contract. Embedded into the binary via `api/embed.go` (`//go:embed`).
- **Swagger UI:** Served at `GET /docs` when `server.docs_enabled: true` in config (default: `false`). Uses Swagger UI from unpkg CDN.
- **Raw spec:** Served at `GET /openapi.yaml` when docs are enabled.
- **Spec-route sync test:** `internal/api/openapi_test.go` bidirectionally validates that every chi router route exists in the OpenAPI spec and vice versa. The build WILL fail if the spec and routes drift apart.
- **Static docs generation:** Run `make docs-api` to generate static ReDoc HTML documentation at `docs/api/index.html` (requires `npx`).
- **MCP reference:** Run `make docs-mcp` to generate `docs/mcp-reference.md` from live server introspection.

### Documentation Standards (RFC 2119)

All API documentation (OpenAPI descriptions, doc comments, user-facing text) MUST follow these writing conventions:

- Use RFC 2119 keywords (`MUST`, `MUST NOT`, `REQUIRED`, `SHALL`, `SHALL NOT`, `SHOULD`, `SHOULD NOT`, `RECOMMENDED`, `MAY`, `OPTIONAL`) when specifying requirements. These keywords MUST be written in uppercase when used in their normative sense.
- Use `backticks` for field names, parameter names, endpoint paths, header names, and literal values.
- Use **bold** for emphasis on key concepts and terms.
- Use blockquotes (`>`) for important notes, warnings, and caveats.
- Descriptions SHOULD explain concepts for users unfamiliar with schema registries, Avro, Protobuf, and JSON Schema.
- Include error conditions and error codes in endpoint descriptions.
- API descriptions SHOULD explain *why* an endpoint exists, not just *what* it does.

## Code Conventions

- Standard Go (`gofmt`), golangci-lint (`.golangci.yaml`)
- Structured logging via `log/slog`
- Context-first signatures: `func (s *Store) Method(ctx context.Context, ...) error`
- Multi-tenant context: all storage/registry methods take `registryCtx string` (default `"."`)
- Sentinel errors — use `errors.Is()` for comparison
- Build tags: `//go:build integration`, `//go:build api`, `//go:build bdd`, `//go:build concurrency`, `//go:build conformance`, `//go:build ldap`, `//go:build vault`, `//go:build oidc`, `//go:build migration`
- Schema fingerprints are SHA-256 content-addressed for deduplication
- Soft-delete uses boolean flag, not physical deletion
- Global config uses empty string (`""`) as subject key
- Import API preserves original IDs (Kafka wire format embeds schema IDs)
- Multi-stage Docker build: golang:1.26-alpine -> alpine:3.19, binary at /app/schema-registry
- BDD-first testing: every feature MUST have BDD coverage; if it is not BDD tested in a black-box manner, it is not considered tested
- BDD tests MUST assert audit events for write operations using composite table assertions (`the audit log should contain an event:` + DataTable)
- All BDD tests run against Docker compose-deployed binaries — never in-process
- Audit output architecture: `AuditOutput` interface with fan-out (stdout, file+rotation, syslog RFC 5424/TLS, webhook with batching/retry)
- Makefile targets are the canonical way to run tests — they match what CI runs
- **Test assertion integrity: You MUST NOT weaken, relax, or remove test assertions to make failing tests pass.** If a test asserts a value and the code does not produce that value, the code has a bug — fix the code, not the test. Weakening assertions hides real bugs and degrades system quality. Before changing any test expectation, you MUST: (1) analyse the scenario being tested to understand what the correct behaviour SHOULD be, (2) determine whether the test or the code is wrong, and (3) if the test expectation genuinely needs changing, confirm with the user before proceeding. This applies to ALL test types (BDD, unit, integration, conformance).
