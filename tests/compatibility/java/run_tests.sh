#!/bin/bash
# Run Java compatibility tests against different Confluent versions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"

# Confluent versions to test
PROFILES=("confluent-8.1" "confluent-7.9" "confluent-7.7.4" "confluent-7.7.3")

echo "==================================="
echo "Java Compatibility Tests"
echo "Schema Registry: ${SCHEMA_REGISTRY_URL}"
echo "==================================="

cd "${SCRIPT_DIR}"

# Run tests for each profile
for profile in "${PROFILES[@]}"; do
    echo ""
    echo "-----------------------------------"
    echo "Testing with profile: ${profile}"
    echo "-----------------------------------"

    mvn test -P "${profile}" \
        -Dschema.registry.url="${SCHEMA_REGISTRY_URL}" \
        -q

    echo "Profile ${profile}: PASSED"
done

echo ""
echo "==================================="
echo "All Java compatibility tests completed!"
echo "==================================="
