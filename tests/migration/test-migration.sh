#!/bin/bash
#
# Test migration from real Confluent Schema Registry to AxonOps
# Starts Kafka + Confluent SR as source, AxonOps as target
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/../.."

CONTAINER_CMD="${CONTAINER_CMD:-docker}"
TARGET_PID=""

cleanup() {
    echo "Cleaning up..."
    if [[ -n "$TARGET_PID" ]]; then
        kill "$TARGET_PID" 2>/dev/null || true
    fi
    $CONTAINER_CMD rm -f confluent-sr kafka zookeeper 2>/dev/null || true
    $CONTAINER_CMD network rm sr-migration-net 2>/dev/null || true
    rm -f /tmp/target-config.yaml /tmp/test-export.json /tmp/target.log
}
trap cleanup EXIT

# Use pre-built binary if available (CI), otherwise build
if [[ -x "./build/schema-registry" ]]; then
    echo "=== Using build/schema-registry ==="
    SCHEMA_REGISTRY="./build/schema-registry"
elif [[ -x "./schema-registry" ]]; then
    echo "=== Using pre-built schema-registry ==="
    SCHEMA_REGISTRY="./schema-registry"
else
    echo "=== Building schema-registry ==="
    go build -o ./build/schema-registry ./cmd/schema-registry
    SCHEMA_REGISTRY="./build/schema-registry"
fi

echo ""
echo "=========================================="
echo "Migration Test: Confluent SR -> AxonOps"
echo "=========================================="
echo ""

# Create a dedicated network so containers can reach each other by name
$CONTAINER_CMD network rm sr-migration-net 2>/dev/null || true
$CONTAINER_CMD network create sr-migration-net 2>/dev/null || true

echo "=== Starting Zookeeper ==="
$CONTAINER_CMD run -d --name zookeeper --network sr-migration-net \
    -e ZOOKEEPER_CLIENT_PORT=2181 \
    -e ZOOKEEPER_TICK_TIME=2000 \
    confluentinc/cp-zookeeper:7.5.0

echo "=== Starting Kafka ==="
$CONTAINER_CMD run -d --name kafka --network sr-migration-net \
    -p 29092:9092 \
    -e KAFKA_BROKER_ID=1 \
    -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
    -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092 \
    -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
    confluentinc/cp-kafka:7.5.0

echo "Waiting for Kafka to start..."
for i in {1..30}; do
    if $CONTAINER_CMD exec kafka kafka-topics --bootstrap-server kafka:9092 --list 2>/dev/null; then
        echo "Kafka is ready"
        break
    fi
    echo "Waiting for Kafka... ($i)"
    sleep 2
done

echo ""
echo "=== Starting Confluent Schema Registry ==="
$CONTAINER_CMD run -d --name confluent-sr --network sr-migration-net \
    -p 28084:8081 \
    -e SCHEMA_REGISTRY_HOST_NAME=confluent-sr \
    -e SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS=kafka:9092 \
    -e SCHEMA_REGISTRY_LISTENERS=http://0.0.0.0:8081 \
    confluentinc/cp-schema-registry:7.5.0

echo "Waiting for Confluent Schema Registry to start..."
for i in {1..30}; do
    if curl -sf http://localhost:28084/subjects > /dev/null 2>&1; then
        echo "Confluent Schema Registry is ready"
        break
    fi
    echo "Waiting for Confluent SR... ($i)"
    sleep 2
done

echo ""
echo "=== Starting AxonOps Schema Registry (target) on port 18082 ==="
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

echo "Waiting for AxonOps Schema Registry to start..."
for i in {1..30}; do
    if curl -sf http://localhost:18082/ > /dev/null 2>&1; then
        echo "AxonOps Schema Registry is ready"
        break
    fi
    sleep 0.5
done

echo ""
echo "=== Populating Confluent Schema Registry with test schemas ==="

# Register schemas on Confluent SR
echo "Registering user-value v1..."
curl -sf -X POST http://localhost:28084/subjects/user-value/versions \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}' | jq .

echo "Registering user-value v2..."
curl -sf -X POST http://localhost:28084/subjects/user-value/versions \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\",\"default\":\"\"}]}"}' | jq .

echo "Registering order-value..."
curl -sf -X POST http://localhost:28084/subjects/order-value/versions \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"Order\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"order_id\",\"type\":\"long\"},{\"name\":\"user_id\",\"type\":\"long\"}]}"}' | jq .

echo "Registering product-value..."
curl -sf -X POST http://localhost:28084/subjects/product-value/versions \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"Product\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"product_id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\"}]}"}' | jq .

echo ""
echo "=== Confluent Schema Registry contents ==="
echo "Subjects:"
curl -sf http://localhost:28084/subjects | jq .
echo ""
echo "Schema IDs:"
for subject in user-value order-value product-value; do
    echo "$subject versions:"
    curl -sf "http://localhost:28084/subjects/$subject/versions" | jq .
done

echo ""
echo "=== Running migration script ==="
./scripts/migrate-from-confluent.sh \
    --source http://localhost:28084 \
    --target http://localhost:18082 \
    --output /tmp/test-export.json \
    --verify

echo ""
echo "=== Verifying AxonOps Schema Registry contents ==="
echo "Subjects:"
curl -sf http://localhost:18082/subjects | jq .

echo ""
echo "=== Verifying schema IDs match between Confluent and AxonOps ==="
PASS=true

for subject in user-value order-value product-value; do
    VERSIONS=$(curl -sf "http://localhost:28084/subjects/$subject/versions")
    for version in $(echo "$VERSIONS" | jq -r '.[]'); do
        CONFLUENT_ID=$(curl -sf "http://localhost:28084/subjects/$subject/versions/$version" | jq -r '.id')
        AXONOPS_ID=$(curl -sf "http://localhost:18082/subjects/$subject/versions/$version" | jq -r '.id')

        if [[ "$CONFLUENT_ID" == "$AXONOPS_ID" ]]; then
            echo "✓ $subject v$version: ID $CONFLUENT_ID matches"
        else
            echo "✗ $subject v$version: Confluent ID=$CONFLUENT_ID, AxonOps ID=$AXONOPS_ID"
            PASS=false
        fi
    done
done

echo ""
echo "=== Verifying new registrations get correct IDs ==="
echo "Registering new schema on AxonOps..."
NEW_RESPONSE=$(curl -sf -X POST http://localhost:18082/subjects/new-subject/versions \
    -H "Content-Type: application/vnd.schemaregistry.v1+json" \
    -d '{"schema": "{\"type\":\"string\"}"}')
echo "New schema response: $NEW_RESPONSE"

NEW_ID=$(echo "$NEW_RESPONSE" | jq -r '.id')
MAX_IMPORTED=$(jq '[.schemas[].id] | max' /tmp/test-export.json)
echo "New ID: $NEW_ID, Max imported ID: $MAX_IMPORTED"

if [[ "$NEW_ID" -le "$MAX_IMPORTED" ]]; then
    echo "✗ FAIL: New ID ($NEW_ID) should be greater than max imported ID ($MAX_IMPORTED)"
    PASS=false
else
    echo "✓ New ID is correctly greater than max imported ID"
fi

echo ""
if [[ "$PASS" == "true" ]]; then
    echo "=========================================="
    echo "Migration test PASSED!"
    echo "=========================================="
else
    echo "=========================================="
    echo "Migration test FAILED!"
    echo "=========================================="
    exit 1
fi
