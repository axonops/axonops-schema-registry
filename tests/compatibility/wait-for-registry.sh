#!/bin/bash
# Wait for Schema Registry to become available

SCHEMA_REGISTRY_URL="${SCHEMA_REGISTRY_URL:-http://localhost:8081}"
MAX_WAIT="${MAX_WAIT:-60}"

echo "Waiting for Schema Registry at ${SCHEMA_REGISTRY_URL}..."

for i in $(seq 1 $MAX_WAIT); do
    if curl -s "${SCHEMA_REGISTRY_URL}/subjects" > /dev/null 2>&1; then
        echo "Schema Registry is ready!"
        exit 0
    fi
    echo "Waiting... ($i/${MAX_WAIT})"
    sleep 1
done

echo "ERROR: Schema Registry did not become available within ${MAX_WAIT} seconds"
exit 1
