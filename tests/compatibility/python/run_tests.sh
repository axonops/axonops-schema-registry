#!/bin/bash
# Run Python compatibility tests against different confluent-kafka versions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"

# Confluent Kafka Python versions to test
# Latest versions as of 2025
VERSIONS=("2.8.0" "2.6.2" "2.5.3")

echo "==================================="
echo "Python Compatibility Tests"
echo "Schema Registry: ${SCHEMA_REGISTRY_URL}"
echo "==================================="

# Create virtual environment if it doesn't exist
VENV_BASE="${SCRIPT_DIR}/.venvs"
mkdir -p "${VENV_BASE}"

run_tests_for_version() {
    local version=$1
    local venv_dir="${VENV_BASE}/confluent-${version}"

    echo ""
    echo "-----------------------------------"
    echo "Testing confluent-kafka==${version}"
    echo "-----------------------------------"

    # Create virtual environment
    if [ ! -d "${venv_dir}" ]; then
        echo "Creating virtual environment for ${version}..."
        python3 -m venv "${venv_dir}"
    fi

    # Activate and install
    source "${venv_dir}/bin/activate"

    echo "Installing confluent-kafka==${version}..."
    pip install --quiet --upgrade pip
    pip install --quiet "confluent-kafka[avro,json,protobuf]==${version}" pytest pytest-env

    # Show installed version
    python -c "import confluent_kafka; print(f'Installed: confluent-kafka {confluent_kafka.version()[0]}')"

    # Run tests
    echo "Running tests..."
    SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL}" pytest "${SCRIPT_DIR}" -v --tb=short

    deactivate
}

# Run tests for each version
for version in "${VERSIONS[@]}"; do
    run_tests_for_version "${version}"
done

echo ""
echo "==================================="
echo "All Python compatibility tests completed!"
echo "==================================="
