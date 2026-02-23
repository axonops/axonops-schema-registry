#!/usr/bin/env bash
# Run Go data contract & CSFLE integration tests.
#
# Prerequisites:
#   - Schema registry running at $SCHEMA_REGISTRY_URL (default: http://localhost:8081)
#   - For CSFLE tests: Vault running at $VAULT_URL (default: http://localhost:18200)
#     with Transit engine enabled and test-key created
#
# Usage:
#   ./run_tests.sh                    # Run all tests
#   ./run_tests.sh -run TestCel       # Run only CEL tests
#   ./run_tests.sh -run TestCsfle     # Run only CSFLE tests (requires Vault)
#   ./run_tests.sh -run TestMigration # Run only migration tests
#   ./run_tests.sh -run TestDefault   # Run only global policy tests

set -euo pipefail
cd "$(dirname "$0")"

export SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
export VAULT_URL="${VAULT_URL:-http://localhost:18200}"
export VAULT_TOKEN="${VAULT_TOKEN:-test-root-token}"

echo "Schema Registry: ${SCHEMA_REGISTRY_URL}"
echo "Vault:           ${VAULT_URL}"
echo ""

go test -v -timeout 10m "$@" ./...
