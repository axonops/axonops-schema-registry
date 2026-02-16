#!/usr/bin/env bash
# stop-db.sh — Stop and remove a database test container.
# Usage: stop-db.sh <postgres|mysql|cassandra>
#
# Removes the container and its anonymous volumes to ensure the next
# run starts with completely clean state. Only touches containers with
# the sr-test- prefix — never affects unrelated containers.
set -euo pipefail

BACKEND="${1:?Usage: stop-db.sh <postgres|mysql|cassandra>}"
CONTAINER_CMD="${CONTAINER_CMD:-docker}"

case "$BACKEND" in
  postgres)  CONTAINER_NAME="sr-test-postgres" ;;
  mysql)     CONTAINER_NAME="sr-test-mysql" ;;
  cassandra) CONTAINER_NAME="sr-test-cassandra" ;;
  *)
    echo "Unknown backend: $BACKEND"
    exit 1
    ;;
esac

echo "Stopping $CONTAINER_NAME..."
$CONTAINER_CMD stop "$CONTAINER_NAME" 2>/dev/null || true
$CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
echo "$CONTAINER_NAME stopped and removed."
