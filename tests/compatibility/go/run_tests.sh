#!/bin/bash
# Run Go compatibility tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"

echo "==================================="
echo "Go Compatibility Tests"
echo "Schema Registry: ${SCHEMA_REGISTRY_URL}"
echo "==================================="

cd "${SCRIPT_DIR}"

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Show Go version
go version

# Show srclient version
echo "Testing with srclient (Confluent Schema Registry Go client)"

# Run tests
echo ""
echo "Running tests..."
SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL}" go test -v ./...

echo ""
echo "==================================="
echo "All Go compatibility tests completed!"
echo "==================================="
