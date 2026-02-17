#!/bin/bash
set -e

# Start the schema registry via an intermediate shell that exits immediately.
# This causes the registry process to be reparented to PID 1 (tini, via init: true).
# Tini can then properly reap the zombie when the registry is killed by webhook scripts.
# Without this, the registry would be a child of webhook (via exec below) and webhook
# does not call wait() on children it didn't create, leaving zombies that block kill -0 loops.
#
# Retry logic: For database backends (Cassandra, PostgreSQL, MySQL), the DB healthcheck
# may pass inside the DB container before external connections are accepted. The registry
# may die immediately if it can't connect. We retry up to MAX_RETRIES times.

MAX_RETRIES=10
RETRY_DELAY=5

for i in $(seq 1 $MAX_RETRIES); do
    bash -c '/app/schema-registry "$@" & echo $! > /tmp/registry.pid' -- "$@"

    # Give the registry a moment to start up or crash.
    sleep 3

    PID=$(cat /tmp/registry.pid)
    if kill -0 "$PID" 2>/dev/null; then
        echo "Registry started (PID $PID) on attempt $i"
        break
    fi

    if [ "$i" -eq "$MAX_RETRIES" ]; then
        echo "Registry failed to start after $MAX_RETRIES attempts"
        exit 1
    fi

    echo "Registry died on attempt $i/$MAX_RETRIES, retrying in ${RETRY_DELAY}s..."
    sleep "$RETRY_DELAY"
done

# Start webhook on port 9000 (blocking).
exec webhook -hooks /etc/webhook/hooks.json -port 9000 -verbose
