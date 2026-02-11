#!/bin/bash
# Unpause the schema registry process (SIGCONT).
set -e
PID_FILE="/tmp/registry.pid"

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    kill -CONT "$PID" 2>/dev/null || true
    echo "Registry unpaused (PID $PID)"
else
    echo "No PID file found"
fi
