.PHONY: build build-frontend build-backend test test-unit test-integration test-bdd test-e2e lint clean docker docker-ui docker-server-ui help

BINARY   := schema-registry-ui
BUILD_DIR := ./cmd/$(BINARY)
DIST_DIR  := web/dist

## Build -----------------------------------------------------------------------

build: build-frontend build-backend ## Build frontend + Go binary

build-frontend: ## Build React frontend
	npm ci
	npm run build
	rm -rf $(DIST_DIR)
	cp -r dist/ $(DIST_DIR)/

build-backend: ## Build Go binary (embeds frontend from web/dist)
	go build -o $(BINARY) $(BUILD_DIR)

## Test ------------------------------------------------------------------------

test: test-unit ## Run unit tests (default)

test-unit: ## Run Go unit tests
	go test -race -v ./internal/...

test-integration: ## Run Go integration tests (requires no running servers)
	go test -race -v -tags=integration ./tests/integration/...

test-bdd: ## Run BDD/Gherkin tests
	go test -race -v -tags=bdd ./tests/bdd/...

test-e2e: ## Run Playwright E2E tests (requires running SR + UI)
	npx playwright test

test-all: test-unit test-integration test-bdd ## Run all Go tests

## Lint ------------------------------------------------------------------------

lint: ## Run linters
	npm run lint
	go vet ./...

## Docker ----------------------------------------------------------------------

docker-ui: build-frontend ## Build UI Docker image
	docker build -t axonops/schema-registry-ui -f ../Dockerfile.ui ..

docker-server-ui: build-frontend ## Build combined server+UI Docker image
	@if [ -f ../Dockerfile.server-ui ]; then \
		docker build -t axonops/schema-registry-server-ui -f ../Dockerfile.server-ui ..; \
	else \
		echo "Dockerfile.server-ui not found"; \
	fi

## Misc ------------------------------------------------------------------------

clean: ## Remove build artifacts
	rm -rf dist/ $(DIST_DIR) $(BINARY) node_modules/ test-results/ playwright-report/

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
