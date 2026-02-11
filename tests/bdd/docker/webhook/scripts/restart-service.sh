#!/bin/bash
# Restart the schema registry process inside this container.
set -e
PID_FILE="/tmp/registry.pid"
CONFIG_FILE="${REGISTRY_CONFIG:-/etc/schema-registry/config.yaml}"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    kill "$PID" 2>/dev/null || true
    while kill -0 "$PID" 2>/dev/null; do sleep 0.1; done
fi

# Start via intermediate shell so the registry is reparented to PID 1 (tini).
# Redirect output to container stdout (/proc/1/fd/1) so the webhook pipe closes
# and the HTTP response can be sent.
bash -c '/app/schema-registry --config "$1" > /proc/1/fd/1 2>&1 & echo $! > "$2"' -- "$CONFIG_FILE" "$PID_FILE"
echo "Registry restarted with PID $(cat $PID_FILE)"
