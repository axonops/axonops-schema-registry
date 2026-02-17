# AxonOps Schema Registry Makefile
# Usage: make <target> [BACKEND=memory|postgres|mysql|cassandra|confluent|all]

# =====================================================================
# Variables
# =====================================================================
BINARY_NAME    := schema-registry
VERSION        := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT         := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE     := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS        := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Go settings
GOCMD          := go
GOTEST         := $(GOCMD) test
GOBUILD        := $(GOCMD) build
GOMOD          := $(GOCMD) mod
GOFMT          := gofmt
GOLINT         := golangci-lint

# Directories
CMD_DIR        := ./cmd/schema-registry
BUILD_DIR      := ./build
COVERAGE_DIR   := ./coverage
SCRIPTS_DIR    := ./scripts/test

# Backend selection (default: memory)
BACKEND        ?= memory

# Container runtime (auto-detect: prefer docker, fall back to podman)
CONTAINER_CMD  ?= $(shell command -v docker 2>/dev/null || command -v podman 2>/dev/null || echo docker)

# Port assignments for standalone DB containers.
# These must not conflict with:
#   - BDD standalone compose (tests/bdd/docker-compose.yml): 5433, 3307, 9043
#   - BDD overlay compose: 15432, 13306, 19042
#   - Default local DB ports: 5432, 3306, 9042
DB_POSTGRES_PORT    := 25432
DB_MYSQL_PORT       := 23306
DB_CASSANDRA_PORT   := 29042

# Common DB credentials
DB_USER        := schemaregistry
DB_PASSWORD    := schemaregistry
DB_DATABASE    := schemaregistry

# Timeout configuration (per test type and backend)
TIMEOUT_UNIT           := 5m
TIMEOUT_BDD_MEMORY     := 10m
TIMEOUT_BDD_POSTGRES   := 15m
TIMEOUT_BDD_MYSQL      := 15m
TIMEOUT_BDD_CASSANDRA  := 20m
TIMEOUT_BDD_CONFLUENT  := 25m
TIMEOUT_INT_DEFAULT    := 10m
TIMEOUT_INT_CASSANDRA  := 15m
TIMEOUT_CONC_DEFAULT   := 15m
TIMEOUT_CONC_CASSANDRA := 20m
TIMEOUT_CONF_MEMORY    := 5m
TIMEOUT_CONF_DEFAULT   := 10m
TIMEOUT_CONF_CASSANDRA := 15m
TIMEOUT_API            := 10m
TIMEOUT_LDAP           := 10m
TIMEOUT_VAULT          := 10m
TIMEOUT_OIDC           := 10m
TIMEOUT_MIGRATION      := 5m
TIMEOUT_COMPAT         := 10m

# =====================================================================
# Phony targets
# =====================================================================
.PHONY: all build build-all \
        test test-unit test-bdd test-integration test-concurrency test-conformance \
        test-migration test-api test-ldap test-vault test-oidc test-auth \
        test-compatibility test-coverage \
        deps lint fmt run dev clean \
        docker-build docker-run docs-api help

# =====================================================================
# Default target
# =====================================================================

## Default: deps, lint, test, build
all: deps lint test build

# =====================================================================
# Build targets
# =====================================================================

## Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

## Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

# =====================================================================
# Master test target — runs everything
# =====================================================================

## Run ALL tests (unit, BDD, integration, conformance, concurrency, migration, API, auth, compatibility)
test: test-unit test-bdd test-integration test-conformance test-concurrency test-migration test-api test-auth test-compatibility

# =====================================================================
# Unit tests (no Docker, no build tags)
# =====================================================================

## Run unit tests (internal packages, no Docker needed)
test-unit:
	@echo "=== Unit Tests ==="
	$(GOTEST) -race -v -timeout $(TIMEOUT_UNIT) ./internal/...

# =====================================================================
# BDD tests — Docker managed by Go test code (bdd_test.go)
# =====================================================================

## Run BDD tests [BACKEND=memory|postgres|mysql|cassandra|confluent|all]
test-bdd:
ifeq ($(BACKEND),all)
	@for b in memory postgres mysql cassandra confluent; do \
		echo ""; \
		echo "==================================================="; \
		echo "=== BDD Tests: $$b backend"; \
		echo "==================================================="; \
		$(MAKE) --no-print-directory _test-bdd-single BACKEND=$$b || exit 1; \
	done
else
	@$(MAKE) --no-print-directory _test-bdd-single BACKEND=$(BACKEND)
endif

.PHONY: _test-bdd-single
_test-bdd-single:
	@TIMEOUT=""; \
	case "$(BACKEND)" in \
		memory)    TIMEOUT=$(TIMEOUT_BDD_MEMORY) ;; \
		postgres)  TIMEOUT=$(TIMEOUT_BDD_POSTGRES) ;; \
		mysql)     TIMEOUT=$(TIMEOUT_BDD_MYSQL) ;; \
		cassandra) TIMEOUT=$(TIMEOUT_BDD_CASSANDRA) ;; \
		confluent) TIMEOUT=$(TIMEOUT_BDD_CONFLUENT) ;; \
		*)         echo "Unknown BDD backend: $(BACKEND)"; exit 1 ;; \
	esac; \
	echo "=== BDD Tests ($(BACKEND), timeout $$TIMEOUT) ==="; \
	if [ "$(BACKEND)" = "memory" ]; then \
		$(GOTEST) -tags bdd -v -count=1 -timeout $$TIMEOUT ./tests/bdd/...; \
	else \
		BDD_BACKEND=$(BACKEND) CONTAINER_CMD=$(CONTAINER_CMD) \
			$(GOTEST) -tags bdd -v -count=1 -timeout $$TIMEOUT ./tests/bdd/...; \
	fi

# =====================================================================
# Integration tests — Makefile manages DB containers
# =====================================================================

## Run integration tests [BACKEND=postgres|mysql|cassandra|all] (no memory)
test-integration:
ifeq ($(BACKEND),all)
	@for b in postgres mysql cassandra; do \
		echo ""; \
		echo "==================================================="; \
		echo "=== Integration Tests: $$b backend"; \
		echo "==================================================="; \
		$(MAKE) --no-print-directory _test-integration-single BACKEND=$$b || exit 1; \
	done
else ifeq ($(BACKEND),memory)
	@echo "SKIP: Integration tests require a database backend (postgres, mysql, cassandra)."
	@echo "      Run with: make test-integration BACKEND=postgres|mysql|cassandra|all"
else
	@$(MAKE) --no-print-directory _test-integration-single BACKEND=$(BACKEND)
endif

.PHONY: _test-integration-single
_test-integration-single:
	@echo "=== Integration Tests ($(BACKEND)) ==="; \
	TIMEOUT=$(TIMEOUT_INT_DEFAULT); \
	if [ "$(BACKEND)" = "cassandra" ]; then TIMEOUT=$(TIMEOUT_INT_CASSANDRA); fi; \
	DB_POSTGRES_PORT=$(DB_POSTGRES_PORT) DB_MYSQL_PORT=$(DB_MYSQL_PORT) DB_CASSANDRA_PORT=$(DB_CASSANDRA_PORT) \
		DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_DATABASE=$(DB_DATABASE) \
		CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/start-db.sh $(BACKEND); \
	rc=0; \
	STORAGE_TYPE=$(BACKEND) \
		$(call db_env,$(BACKEND)) \
		$(GOTEST) -tags integration -race -v -timeout $$TIMEOUT ./tests/integration/... || rc=$$?; \
	CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/stop-db.sh $(BACKEND); \
	exit $$rc

# =====================================================================
# Concurrency tests — Makefile manages DB containers
# =====================================================================

## Run concurrency tests [BACKEND=postgres|mysql|cassandra|all] (no memory)
test-concurrency:
ifeq ($(BACKEND),all)
	@for b in postgres mysql cassandra; do \
		echo ""; \
		echo "==================================================="; \
		echo "=== Concurrency Tests: $$b backend"; \
		echo "==================================================="; \
		$(MAKE) --no-print-directory _test-concurrency-single BACKEND=$$b || exit 1; \
	done
else ifeq ($(BACKEND),memory)
	@echo "SKIP: Concurrency tests require a database backend (postgres, mysql, cassandra)."
	@echo "      Run with: make test-concurrency BACKEND=postgres|mysql|cassandra|all"
else
	@$(MAKE) --no-print-directory _test-concurrency-single BACKEND=$(BACKEND)
endif

.PHONY: _test-concurrency-single
_test-concurrency-single:
	@echo "=== Concurrency Tests ($(BACKEND)) ==="; \
	TIMEOUT=$(TIMEOUT_CONC_DEFAULT); \
	if [ "$(BACKEND)" = "cassandra" ]; then TIMEOUT=$(TIMEOUT_CONC_CASSANDRA); fi; \
	DB_POSTGRES_PORT=$(DB_POSTGRES_PORT) DB_MYSQL_PORT=$(DB_MYSQL_PORT) DB_CASSANDRA_PORT=$(DB_CASSANDRA_PORT) \
		DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_DATABASE=$(DB_DATABASE) \
		CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/start-db.sh $(BACKEND); \
	rc=0; \
	STORAGE_TYPE=$(BACKEND) \
		$(call db_env,$(BACKEND)) \
		$(GOTEST) -tags concurrency -race -v -timeout $$TIMEOUT ./tests/concurrency/... || rc=$$?; \
	CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/stop-db.sh $(BACKEND); \
	exit $$rc

# =====================================================================
# Conformance tests — memory needs no Docker, DB backends need containers
# =====================================================================

## Run storage conformance tests [BACKEND=memory|postgres|mysql|cassandra|all]
test-conformance:
ifeq ($(BACKEND),all)
	@for b in memory postgres mysql cassandra; do \
		echo ""; \
		echo "==================================================="; \
		echo "=== Conformance Tests: $$b backend"; \
		echo "==================================================="; \
		$(MAKE) --no-print-directory _test-conformance-single BACKEND=$$b || exit 1; \
	done
else
	@$(MAKE) --no-print-directory _test-conformance-single BACKEND=$(BACKEND)
endif

.PHONY: _test-conformance-single
_test-conformance-single:
ifeq ($(BACKEND),memory)
	@echo "=== Conformance Tests (memory) ==="
	$(GOTEST) -race -v -timeout $(TIMEOUT_CONF_MEMORY) -run TestMemoryBackend ./tests/storage/conformance/...
else
	@echo "=== Conformance Tests ($(BACKEND)) ==="; \
	TIMEOUT=$(TIMEOUT_CONF_DEFAULT); \
	if [ "$(BACKEND)" = "cassandra" ]; then TIMEOUT=$(TIMEOUT_CONF_CASSANDRA); fi; \
	TEST_RUN=""; \
	case "$(BACKEND)" in \
		postgres)  TEST_RUN=TestPostgresBackend ;; \
		mysql)     TEST_RUN=TestMySQLBackend ;; \
		cassandra) TEST_RUN=TestCassandraBackend ;; \
		*)         echo "Unknown conformance backend: $(BACKEND)"; exit 1 ;; \
	esac; \
	DB_POSTGRES_PORT=$(DB_POSTGRES_PORT) DB_MYSQL_PORT=$(DB_MYSQL_PORT) DB_CASSANDRA_PORT=$(DB_CASSANDRA_PORT) \
		DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_DATABASE=$(DB_DATABASE) \
		CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/start-db.sh $(BACKEND); \
	rc=0; \
	$(call db_env,$(BACKEND)) \
		$(GOTEST) -tags conformance -race -v -timeout $$TIMEOUT -run $$TEST_RUN ./tests/storage/conformance/... || rc=$$?; \
	CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/stop-db.sh $(BACKEND); \
	exit $$rc
endif

# =====================================================================
# Migration tests — Go tests use in-memory, shell tests need binary
# =====================================================================

## Run migration tests (Go unit tests + shell integration tests)
test-migration: build
	@echo "=== Migration Tests ==="
	@echo "--- Go migration tests (in-memory) ---"
	$(GOTEST) -tags migration -v -timeout $(TIMEOUT_MIGRATION) ./tests/migration/...
	@echo ""
	@echo "--- Shell migration tests (import API) ---"
	@chmod +x ./tests/migration/test-import.sh && TEST_PORT=28081 ./tests/migration/test-import.sh
	@echo ""
	@echo "--- Shell migration tests (full migration) ---"
	@chmod +x ./tests/migration/test-migration.sh && CONTAINER_CMD=$(CONTAINER_CMD) ./tests/migration/test-migration.sh

# =====================================================================
# API endpoint tests — needs running schema-registry binary
# =====================================================================

## Run API endpoint tests (starts schema-registry binary automatically)
test-api: build
	@echo "=== API Endpoint Tests ==="; \
	API_PORT=28082; \
	echo "Starting schema-registry on port $$API_PORT..."; \
	echo "server:" > /tmp/sr-api-test.yaml; \
	echo "  host: 127.0.0.1" >> /tmp/sr-api-test.yaml; \
	echo "  port: $$API_PORT" >> /tmp/sr-api-test.yaml; \
	echo "storage:" >> /tmp/sr-api-test.yaml; \
	echo "  type: memory" >> /tmp/sr-api-test.yaml; \
	$(BUILD_DIR)/$(BINARY_NAME) -config /tmp/sr-api-test.yaml > /tmp/sr-api-test.log 2>&1 & \
	SR_PID=$$!; \
	echo "Waiting for schema-registry (PID $$SR_PID) to start..."; \
	for i in $$(seq 1 30); do \
		if curl -sf http://localhost:$$API_PORT/ > /dev/null 2>&1; then \
			echo "Schema registry is ready"; \
			break; \
		fi; \
		sleep 1; \
	done; \
	rc=0; \
	SCHEMA_REGISTRY_URL=http://localhost:$$API_PORT \
		$(GOTEST) -tags api -v -timeout $(TIMEOUT_API) ./tests/api/... || rc=$$?; \
	kill $$SR_PID 2>/dev/null || true; \
	wait $$SR_PID 2>/dev/null || true; \
	rm -f /tmp/sr-api-test.yaml /tmp/sr-api-test.log; \
	exit $$rc

# =====================================================================
# Auth tests — LDAP, Vault, OIDC (each needs its own service container)
# =====================================================================

## Run all auth tests (LDAP + Vault + OIDC)
test-auth: test-ldap test-vault test-oidc

## Run LDAP authentication tests (starts OpenLDAP container)
test-ldap:
	@echo "=== LDAP Auth Tests ==="; \
	LDAP_PORT=20389; \
	CONTAINER_CMD=$(CONTAINER_CMD) LDAP_PORT=$$LDAP_PORT $(SCRIPTS_DIR)/setup-ldap.sh start; \
	rc=0; \
	LDAP_URL=ldap://localhost:$$LDAP_PORT \
		$(GOTEST) -tags ldap -race -v -timeout $(TIMEOUT_LDAP) ./tests/integration/... || rc=$$?; \
	CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/setup-ldap.sh stop; \
	exit $$rc

## Run Vault authentication tests (starts HashiCorp Vault container)
test-vault:
	@echo "=== Vault Auth Tests ==="; \
	VAULT_PORT=28200; \
	CONTAINER_CMD=$(CONTAINER_CMD) VAULT_PORT=$$VAULT_PORT $(SCRIPTS_DIR)/setup-vault.sh start; \
	rc=0; \
	VAULT_ADDR=http://localhost:$$VAULT_PORT VAULT_TOKEN=root \
		$(GOTEST) -tags vault -race -v -timeout $(TIMEOUT_VAULT) ./tests/integration/... || rc=$$?; \
	CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/setup-vault.sh stop; \
	exit $$rc

## Run OIDC authentication tests (starts Keycloak container)
test-oidc:
	@echo "=== OIDC Auth Tests ==="; \
	KC_PORT=28080; \
	CONTAINER_CMD=$(CONTAINER_CMD) KC_PORT=$$KC_PORT $(SCRIPTS_DIR)/setup-oidc.sh start; \
	rc=0; \
	OIDC_ISSUER_URL=http://localhost:$$KC_PORT/realms/schema-registry \
		$(GOTEST) -tags oidc -race -v -timeout $(TIMEOUT_OIDC) ./tests/integration/... || rc=$$?; \
	CONTAINER_CMD=$(CONTAINER_CMD) $(SCRIPTS_DIR)/setup-oidc.sh stop; \
	exit $$rc

# =====================================================================
# Confluent serializer compatibility tests (Go/Java/Python)
# =====================================================================

## Run Confluent serializer compatibility tests (starts schema-registry binary)
test-compatibility: build
	@echo "=== Confluent Compatibility Tests ==="; \
	COMPAT_PORT=28083; \
	echo "Starting schema-registry on port $$COMPAT_PORT..."; \
	echo "server:" > /tmp/sr-compat-test.yaml; \
	echo "  host: 127.0.0.1" >> /tmp/sr-compat-test.yaml; \
	echo "  port: $$COMPAT_PORT" >> /tmp/sr-compat-test.yaml; \
	echo "storage:" >> /tmp/sr-compat-test.yaml; \
	echo "  type: memory" >> /tmp/sr-compat-test.yaml; \
	$(BUILD_DIR)/$(BINARY_NAME) -config /tmp/sr-compat-test.yaml > /tmp/sr-compat-test.log 2>&1 & \
	SR_PID=$$!; \
	echo "Waiting for schema-registry (PID $$SR_PID) to start..."; \
	for i in $$(seq 1 30); do \
		if curl -sf http://localhost:$$COMPAT_PORT/subjects > /dev/null 2>&1; then \
			echo "Schema registry is ready"; \
			break; \
		fi; \
		sleep 1; \
	done; \
	rc=0; \
	echo ""; \
	echo "--- Go compatibility tests ---"; \
	(cd tests/compatibility/go && go mod download && \
		SCHEMA_REGISTRY_URL=http://localhost:$$COMPAT_PORT go test -v -timeout $(TIMEOUT_COMPAT) ./...) || rc=1; \
	echo ""; \
	echo "--- Java compatibility tests ---"; \
	if command -v mvn > /dev/null 2>&1; then \
		for profile in confluent-8.1 confluent-7.9 confluent-7.7.4 confluent-7.7.3; do \
			echo "Testing with profile: $$profile"; \
			(cd tests/compatibility/java && \
				SCHEMA_REGISTRY_URL=http://localhost:$$COMPAT_PORT \
				mvn test -P "$$profile" -Dschema.registry.url=http://localhost:$$COMPAT_PORT -q) || rc=1; \
		done; \
	else \
		echo "SKIP: mvn not found, skipping Java tests"; \
	fi; \
	echo ""; \
	echo "--- Python compatibility tests ---"; \
	if command -v python3 > /dev/null 2>&1; then \
		for version in 2.8.0 2.7.0 2.6.1; do \
			echo "Testing with confluent-kafka==$$version"; \
			(cd tests/compatibility/python && \
				python3 -m venv ".venv-$$version" && \
				. ".venv-$$version/bin/activate" && \
				pip install --quiet --upgrade pip && \
				pip install --quiet "confluent-kafka[avro,json,protobuf]==$$version" pytest && \
				SCHEMA_REGISTRY_URL=http://localhost:$$COMPAT_PORT pytest -v --tb=short && \
				deactivate) || rc=1; \
		done; \
	else \
		echo "SKIP: python3 not found, skipping Python tests"; \
	fi; \
	kill $$SR_PID 2>/dev/null || true; \
	wait $$SR_PID 2>/dev/null || true; \
	rm -f /tmp/sr-compat-test.yaml /tmp/sr-compat-test.log; \
	exit $$rc

# =====================================================================
# Test coverage
# =====================================================================

## Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

# =====================================================================
# Development targets
# =====================================================================

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## Run linter
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

## Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## Run the server
run: build
	@echo "Starting schema registry..."
	$(BUILD_DIR)/$(BINARY_NAME)

## Run with hot reload (requires air)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

## Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t axonops/schema-registry:$(VERSION) .
	docker tag axonops/schema-registry:$(VERSION) axonops/schema-registry:latest

## Run with Docker
docker-run:
	docker run -p 8081:8081 axonops/schema-registry:latest

## Generate API documentation from OpenAPI spec (markdown + ReDoc HTML)
docs-api:
	@./scripts/generate-api-docs.sh

## Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	$(GOCMD) clean

# =====================================================================
# Helper functions (GNU Make)
# =====================================================================

# db_env returns env var assignments for connecting to test DB containers.
# These are exported as prefixed environment variables for go test.
define db_env
$(if $(filter postgres,$(1)),POSTGRES_HOST=localhost POSTGRES_PORT=$(DB_POSTGRES_PORT) POSTGRES_USER=$(DB_USER) POSTGRES_PASSWORD=$(DB_PASSWORD) POSTGRES_DATABASE=$(DB_DATABASE))$(if $(filter mysql,$(1)),MYSQL_HOST=localhost MYSQL_PORT=$(DB_MYSQL_PORT) MYSQL_USER=$(DB_USER) MYSQL_PASSWORD=$(DB_PASSWORD) MYSQL_DATABASE=$(DB_DATABASE))$(if $(filter cassandra,$(1)),CASSANDRA_HOSTS=127.0.0.1 CASSANDRA_PORT=$(DB_CASSANDRA_PORT) CASSANDRA_KEYSPACE=$(DB_DATABASE))
endef

# =====================================================================
# Help
# =====================================================================

## Show help
help:
	@echo "AxonOps Schema Registry"
	@echo ""
	@echo "Usage:"
	@echo "  make <target> [BACKEND=memory|postgres|mysql|cassandra|confluent|all]"
	@echo ""
	@echo "Build:"
	@echo "  build               Build the binary"
	@echo "  build-all           Build for multiple platforms"
	@echo "  docker-build        Build Docker image"
	@echo "  docker-run          Run with Docker"
	@echo ""
	@echo "Test (comprehensive):"
	@echo "  test                Run ALL tests (unit + BDD + integration + conformance"
	@echo "                      + concurrency + migration + API + auth + compatibility)"
	@echo "  test-unit           Unit tests (no Docker, no build tags)"
	@echo "  test-bdd            BDD/Gherkin tests                     [BACKEND=]"
	@echo "  test-integration    Integration tests against DB backends [BACKEND=] (no memory)"
	@echo "  test-concurrency    Concurrency tests against DB backends [BACKEND=] (no memory)"
	@echo "  test-conformance    Storage conformance tests             [BACKEND=]"
	@echo "  test-migration      Migration tests (Go + shell scripts)"
	@echo "  test-api            API endpoint tests (starts binary)"
	@echo "  test-ldap           LDAP auth tests (starts OpenLDAP)"
	@echo "  test-vault          Vault auth tests (starts Vault)"
	@echo "  test-oidc           OIDC auth tests (starts Keycloak)"
	@echo "  test-auth           All auth tests (LDAP + Vault + OIDC)"
	@echo "  test-compatibility  Confluent serializer tests (Go/Java/Python)"
	@echo "  test-coverage       Unit tests with coverage report"
	@echo ""
	@echo "BACKEND values:"
	@echo "  memory              In-memory (default, no Docker for most tests)"
	@echo "  postgres            PostgreSQL backend"
	@echo "  mysql               MySQL backend"
	@echo "  cassandra           Cassandra backend"
	@echo "  confluent           Confluent Schema Registry (BDD only)"
	@echo "  all                 Run against all applicable backends"
	@echo ""
	@echo "Development:"
	@echo "  deps                Download dependencies"
	@echo "  lint                Run golangci-lint"
	@echo "  fmt                 Format code"
	@echo "  run                 Build and run the server"
	@echo "  dev                 Run with hot reload (requires air)"
	@echo "  docs-api            Generate API docs from OpenAPI (markdown + HTML)"
	@echo "  clean               Clean build artifacts"
