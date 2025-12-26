#!/bin/bash
# Run all compatibility tests across all languages and versions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"

echo "=============================================="
echo "AxonOps Schema Registry Compatibility Tests"
echo "=============================================="
echo "Schema Registry: ${SCHEMA_REGISTRY_URL}"
echo ""

# Check if Schema Registry is available
echo "Checking Schema Registry availability..."
if ! curl -s "${SCHEMA_REGISTRY_URL}/subjects" > /dev/null 2>&1; then
    echo "ERROR: Schema Registry is not available at ${SCHEMA_REGISTRY_URL}"
    echo "Please start the test infrastructure first:"
    echo "  docker-compose up -d"
    exit 1
fi
echo "Schema Registry is available."
echo ""

# Track results
JAVA_RESULT=0
PYTHON_RESULT=0
GO_RESULT=0

# Run Java tests
if command -v mvn &> /dev/null; then
    echo "=============================================="
    echo "Running Java Tests"
    echo "=============================================="
    cd "${SCRIPT_DIR}/java"
    if ./run_tests.sh; then
        echo "Java tests: PASSED"
    else
        echo "Java tests: FAILED"
        JAVA_RESULT=1
    fi
else
    echo "SKIP: Maven not found, skipping Java tests"
fi

echo ""

# Run Python tests
if command -v python3 &> /dev/null; then
    echo "=============================================="
    echo "Running Python Tests"
    echo "=============================================="
    cd "${SCRIPT_DIR}/python"
    if ./run_tests.sh; then
        echo "Python tests: PASSED"
    else
        echo "Python tests: FAILED"
        PYTHON_RESULT=1
    fi
else
    echo "SKIP: Python not found, skipping Python tests"
fi

echo ""

# Run Go tests
if command -v go &> /dev/null; then
    echo "=============================================="
    echo "Running Go Tests"
    echo "=============================================="
    cd "${SCRIPT_DIR}/go"
    if ./run_tests.sh; then
        echo "Go tests: PASSED"
    else
        echo "Go tests: FAILED"
        GO_RESULT=1
    fi
else
    echo "SKIP: Go not found, skipping Go tests"
fi

echo ""
echo "=============================================="
echo "Summary"
echo "=============================================="

TOTAL_FAILED=$((JAVA_RESULT + PYTHON_RESULT + GO_RESULT))

if [ $JAVA_RESULT -eq 0 ]; then
    echo "  Java:   ✓ PASSED"
else
    echo "  Java:   ✗ FAILED"
fi

if [ $PYTHON_RESULT -eq 0 ]; then
    echo "  Python: ✓ PASSED"
else
    echo "  Python: ✗ FAILED"
fi

if [ $GO_RESULT -eq 0 ]; then
    echo "  Go:     ✓ PASSED"
else
    echo "  Go:     ✗ FAILED"
fi

echo ""

if [ $TOTAL_FAILED -eq 0 ]; then
    echo "All compatibility tests passed!"
    exit 0
else
    echo "${TOTAL_FAILED} test suite(s) failed."
    exit 1
fi
