#!/bin/bash
# Pause the schema registry process (SIGSTOP).
set -e
PID_FILE="/tmp/registry.pid"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    kill -STOP "$PID" 2>/dev/null || true
    echo "Registry paused (PID $PID)"
else
    echo "No PID file found"
fi
