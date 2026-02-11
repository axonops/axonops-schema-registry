#!/bin/bash
set -e

# Start the schema registry via an intermediate shell that exits immediately.
# This causes the registry process to be reparented to PID 1 (tini, via init: true).
# Tini can then properly reap the zombie when the registry is killed by webhook scripts.
# Without this, the registry would be a child of webhook (via exec below) and webhook
# does not call wait() on children it didn't create, leaving zombies that block kill -0 loops.
bash -c '/app/schema-registry "$@" & echo $! > /tmp/registry.pid' -- "$@"

# Give the registry a moment to bind its port.
sleep 0.5

# Start webhook on port 9000 (blocking).
exec webhook -hooks /etc/webhook/hooks.json -port 9000 -verbose
