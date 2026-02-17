#!/bin/bash
# Stop the schema registry process inside this container.
set -e
PID_FILE="/tmp/registry.pid"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    kill "$PID" 2>/dev/null || true
    while kill -0 "$PID" 2>/dev/null; do sleep 0.1; done
    echo "Registry stopped (PID $PID)"
else
    echo "No PID file found"
fi
