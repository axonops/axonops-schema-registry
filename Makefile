# AxonOps Schema Registry Makefile

# Variables
BINARY_NAME := schema-registry
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Go settings
GOCMD := go
GOTEST := $(GOCMD) test
GOBUILD := $(GOCMD) build
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Directories
CMD_DIR := ./cmd/schema-registry
BUILD_DIR := ./build
COVERAGE_DIR := ./coverage

.PHONY: all build test test-coverage clean deps lint fmt run help docker docker-build

## Default target
all: deps lint test build

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

## Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -race -v ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

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

## Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	$(GOCMD) clean

## Show help
help:
	@echo "AxonOps Schema Registry"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
