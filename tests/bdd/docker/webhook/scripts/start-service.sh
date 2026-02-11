#!/bin/bash
# Start the schema registry process inside this container.
set -e
PID_FILE="/tmp/registry.pid"
CONFIG_FILE="${REGISTRY_CONFIG:-/etc/schema-registry/config.yaml}"

# Check if already running
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
        echo "Registry already running (PID $PID)"
        exit 0
    fi
fi

# Start via intermediate shell so the registry is reparented to PID 1 (tini).
# Redirect output to container stdout (/proc/1/fd/1) so the webhook pipe closes
# and the HTTP response can be sent.
bash -c '/app/schema-registry --config "$1" > /proc/1/fd/1 2>&1 & echo $! > "$2"' -- "$CONFIG_FILE" "$PID_FILE"
echo "Registry started with PID $(cat $PID_FILE)"
