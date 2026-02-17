#!/bin/bash
#
# Test the schema import API end-to-end
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/../.."

# Port for the test server (configurable to avoid conflicts)
TEST_PORT="${TEST_PORT:-28081}"

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

# Kill any leftover process on our test port
if lsof -ti :"$TEST_PORT" > /dev/null 2>&1; then
    echo "=== Killing leftover process on port $TEST_PORT ==="
    kill $(lsof -ti :"$TEST_PORT") 2>/dev/null || true
    sleep 1
fi

echo "=== Creating config ==="
cat > /tmp/test-config.yaml << EOF
server:
  host: "127.0.0.1"
  port: $TEST_PORT
storage:
  type: memory
compatibility:
  default_level: BACKWARD
EOF

echo "=== Starting server on port $TEST_PORT ==="
$SCHEMA_REGISTRY -config /tmp/test-config.yaml > /tmp/schema-registry.log 2>&1 &
SERVER_PID=$!
trap "kill $SERVER_PID 2>/dev/null || true; rm -f /tmp/test-config.yaml /tmp/schema-registry.log" EXIT

# Wait for server to start
for i in {1..30}; do
    if curl -sf http://localhost:$TEST_PORT/ > /dev/null 2>&1; then
        echo "Server started"
        break
    fi
    sleep 0.1
done

echo ""
echo "=== Setting IMPORT mode ==="
curl -sf -X PUT http://localhost:$TEST_PORT/mode \
    -H "Content-Type: application/json" \
    -d '{"mode":"IMPORT"}' > /dev/null
echo "IMPORT mode set"

echo ""
echo "=== Test 1: Import multiple schemas ==="
IMPORT_RESPONSE=$(curl -sf -X POST http://localhost:$TEST_PORT/import/schemas \
    -H "Content-Type: application/json" \
    -d '{
        "schemas": [
            {
                "id": 100,
                "subject": "user-value",
                "version": 1,
                "schemaType": "AVRO",
                "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"
            },
            {
                "id": 101,
                "subject": "user-value",
                "version": 2,
                "schemaType": "AVRO",
                "schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"},{\"name\":\"name\",\"type\":\"string\",\"default\":\"\"}]}"
            },
            {
                "id": 200,
                "subject": "order-value",
                "version": 1,
                "schemaType": "AVRO",
                "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"order_id\",\"type\":\"long\"}]}"
            }
        ]
    }')

echo "Import response: $IMPORT_RESPONSE"

IMPORTED=$(echo "$IMPORT_RESPONSE" | jq -r '.imported')
ERRORS=$(echo "$IMPORT_RESPONSE" | jq -r '.errors')

if [[ "$IMPORTED" != "3" || "$ERRORS" != "0" ]]; then
    echo "FAIL: Expected 3 imported, 0 errors. Got imported=$IMPORTED, errors=$ERRORS"
    exit 1
fi
echo "PASS: Imported 3 schemas"

echo ""
echo "=== Test 2: Verify schema ID 100 ==="
SCHEMA_100=$(curl -sf http://localhost:$TEST_PORT/schemas/ids/100)
echo "Schema 100: $SCHEMA_100"

if ! echo "$SCHEMA_100" | jq -e '.schema | contains("User")' > /dev/null; then
    echo "FAIL: Schema 100 content incorrect"
    exit 1
fi
echo "PASS: Schema ID 100 retrieved correctly"

echo ""
echo "=== Test 3: Verify subject/version mapping ==="
USER_V1=$(curl -sf http://localhost:$TEST_PORT/subjects/user-value/versions/1)
echo "user-value v1: $USER_V1"

USER_V1_ID=$(echo "$USER_V1" | jq -r '.id')
if [[ "$USER_V1_ID" != "100" ]]; then
    echo "FAIL: user-value v1 should have ID 100, got $USER_V1_ID"
    exit 1
fi
echo "PASS: user-value v1 has correct ID 100"

echo ""
echo "=== Restoring READWRITE mode ==="
curl -sf -X PUT http://localhost:$TEST_PORT/mode \
    -H "Content-Type: application/json" \
    -d '{"mode":"READWRITE"}' > /dev/null
echo "READWRITE mode restored"

echo ""
echo "=== Test 4: New schema gets ID after imported IDs ==="
NEW_SCHEMA_RESPONSE=$(curl -sf -X POST http://localhost:$TEST_PORT/subjects/product-value/versions \
    -H "Content-Type: application/json" \
    -d '{"schema": "{\"type\":\"record\",\"name\":\"Product\",\"fields\":[{\"name\":\"id\",\"type\":\"long\"}]}"}')

echo "New schema response: $NEW_SCHEMA_RESPONSE"

NEW_ID=$(echo "$NEW_SCHEMA_RESPONSE" | jq -r '.id')
if [[ "$NEW_ID" -le 200 ]]; then
    echo "FAIL: New schema ID should be > 200, got $NEW_ID"
    exit 1
fi
echo "PASS: New schema got ID $NEW_ID (> 200)"

echo ""
echo "=== Setting IMPORT mode for duplicate test ==="
curl -sf -X PUT "http://localhost:$TEST_PORT/mode?force=true" \
    -H "Content-Type: application/json" \
    -d '{"mode":"IMPORT"}' > /dev/null

echo ""
echo "=== Test 5: Duplicate ID rejected ==="
DUP_RESPONSE=$(curl -sf -X POST http://localhost:$TEST_PORT/import/schemas \
    -H "Content-Type: application/json" \
    -d '{
        "schemas": [
            {
                "id": 100,
                "subject": "duplicate-test",
                "version": 1,
                "schemaType": "AVRO",
                "schema": "{\"type\":\"string\"}"
            }
        ]
    }')

echo "Duplicate import response: $DUP_RESPONSE"

DUP_ERRORS=$(echo "$DUP_RESPONSE" | jq -r '.errors')
if [[ "$DUP_ERRORS" != "1" ]]; then
    echo "FAIL: Duplicate ID should be rejected"
    exit 1
fi
echo "PASS: Duplicate ID correctly rejected"

echo ""
echo "=== Restoring READWRITE mode ==="
curl -sf -X PUT http://localhost:$TEST_PORT/mode \
    -H "Content-Type: application/json" \
    -d '{"mode":"READWRITE"}' > /dev/null

echo ""
echo "=== Test 6: List subjects ==="
SUBJECTS=$(curl -sf http://localhost:$TEST_PORT/subjects)
echo "Subjects: $SUBJECTS"

SUBJECT_COUNT=$(echo "$SUBJECTS" | jq 'length')
if [[ "$SUBJECT_COUNT" != "3" ]]; then
    echo "FAIL: Expected 3 subjects, got $SUBJECT_COUNT"
    exit 1
fi
echo "PASS: All 3 subjects listed"

echo ""
echo "=========================================="
echo "All tests passed!"
echo "=========================================="
