#!/bin/bash
# Kill the schema registry process forcefully (SIGKILL).
set -e
PID_FILE="/tmp/registry.pid"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    kill -9 "$PID" 2>/dev/null || true
    while kill -0 "$PID" 2>/dev/null; do sleep 0.1; done
    echo "Registry killed (PID $PID)"
else
    echo "No PID file found"
fi
