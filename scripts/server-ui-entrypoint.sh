#!/bin/sh
set -e

# Default configuration
SR_UI_REGISTRY_URL="${SR_UI_REGISTRY_URL:-http://127.0.0.1:8081}"
export SR_UI_REGISTRY_URL

echo "Starting Schema Registry on port 8081..."
/app/schema-registry "$@" &
SR_PID=$!

# Wait for SR to be ready
for i in $(seq 1 30); do
  if wget -q -O /dev/null http://127.0.0.1:8081/ 2>/dev/null; then
    break
  fi
  sleep 0.5
done

echo "Starting UI server on port 8080..."
/app/schema-registry-ui &
UI_PID=$!

# Trap signals and forward to children
trap 'kill $SR_PID $UI_PID 2>/dev/null; wait' SIGINT SIGTERM

# Wait for either process to exit
wait -n $SR_PID $UI_PID 2>/dev/null || true

# If one exits, kill the other
kill $SR_PID $UI_PID 2>/dev/null
wait
