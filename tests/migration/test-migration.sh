#!/bin/bash
#
# Test the full migration script end-to-end
# Uses two instances: one as "Confluent" (source) and one as "AxonOps" (target)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/../.."

cleanup() {
    echo "Cleaning up..."
    kill $SOURCE_PID 2>/dev/null || true
    kill $TARGET_PID 2>/dev/null || true
    rm -f /tmp/source-config.yaml /tmp/target-config.yaml /tmp/test-export.json
    rm -f /tmp/source.log /tmp/target.log
}
trap cleanup EXIT

# Use pre-built binary if available (CI), otherwise build
if [[ -x "./schema-registry" ]]; then
    echo "=== Using pre-built schema-registry ==="
    SCHEMA_REGISTRY="./schema-registry"
elif [[ -x "/tmp/schema-registry" ]]; then
    echo "=== Using existing /tmp/schema-registry ==="
    SCHEMA_REGISTRY="/tmp/schema-registry"
else
    echo "=== Building schema-registry ==="
    go build -o /tmp/schema-registry ./cmd/schema-registry
    SCHEMA_REGISTRY="/tmp/schema-registry"
fi

echo "=== Starting SOURCE server (simulating Confluent) on port 18081 ==="
cat > /tmp/source-config.yaml << 'EOF'
server:
  host: "127.0.0.1"
  port: 18081
storage:
  type: memory
compatibility:
  default_level: BACKWARD
EOF
$SCHEMA_REGISTRY -config /tmp/source-config.yaml > /tmp/source.log 2>&1 &
SOURCE_PID=$!

echo "=== Starting TARGET server (AxonOps) on port 18082 ==="
cat > /tmp/target-config.yaml << 'EOF'
server:
  host: "127.0.0.1"
  port: 18082
storage:
  type: memory
compatibility:
  default_level: BACKWARD
EOF
$SCHEMA_REGISTRY -config /tmp/target-config.yaml > /tmp/target.log 2>&1 &
TARGET_PID=$!

# Wait for servers to start
echo "Waiting for servers to start..."
for i in {1..30}; do
    if curl -sf http://localhost:18081/ > /dev/null 2>&1 && \
       curl -sf http://localhost:18082/ > /dev/null 2>&1; then
        echo "Both servers started"
        break
    fi
    sleep 0.2
done

echo ""
echo "=== Populating SOURCE with test schemas ==="

# Register schemas on source (simulating Confluent SR)
curl -sf -X POST http://localhost:18081/subjects/user-value/versions \
    -H "Content-Type: application/json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}' | jq .

curl -sf -X POST http://localhost:18081/subjects/user-value/versions \
    -H "Content-Type: application/json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\",\"default\":\"\"}]}"}' | jq .

curl -sf -X POST http://localhost:18081/subjects/order-value/versions \
    -H "Content-Type: application/json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"order_id\",\"type\":\"long\"}]}"}' | jq .

curl -sf -X POST http://localhost:18081/subjects/product-value/versions \
    -H "Content-Type: application/json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"Product\",\"fields\":[{\"name\":\"product_id\",\"type\":\"long\"}]}"}' | jq .

echo ""
echo "=== Source subjects ==="
curl -sf http://localhost:18081/subjects | jq .

echo ""
echo "=== Running migration script ==="
./scripts/migrate-from-confluent.sh \
    --source http://localhost:18081 \
    --target http://localhost:18082 \
    --output /tmp/test-export.json \
    --verify

echo ""
echo "=== Verifying target subjects ==="
curl -sf http://localhost:18082/subjects | jq .

echo ""
echo "=== Verifying IDs match ==="
echo "Source user-value v1:"
curl -sf http://localhost:18081/subjects/user-value/versions/1 | jq '{id, subject, version}'
echo "Target user-value v1:"
curl -sf http://localhost:18082/subjects/user-value/versions/1 | jq '{id, subject, version}'

echo ""
echo "=== Verifying new registrations get correct IDs ==="
echo "Registering new schema on target..."
NEW_RESPONSE=$(curl -sf -X POST http://localhost:18082/subjects/new-subject/versions \
    -H "Content-Type: application/json" \
    -d '{"schema": "{\"type\":\"string\"}"}')
echo "New schema response: $NEW_RESPONSE"

NEW_ID=$(echo "$NEW_RESPONSE" | jq -r '.id')
MAX_IMPORTED=$(jq '[.schemas[].id] | max' /tmp/test-export.json)
echo "New ID: $NEW_ID, Max imported ID: $MAX_IMPORTED"

if [[ "$NEW_ID" -le "$MAX_IMPORTED" ]]; then
    echo "FAIL: New ID should be greater than max imported ID"
    exit 1
fi

echo ""
echo "=========================================="
echo "Migration test PASSED!"
echo "=========================================="
