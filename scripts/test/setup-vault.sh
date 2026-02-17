#!/usr/bin/env bash
# setup-vault.sh — Start/stop HashiCorp Vault for authentication tests.
# Usage: setup-vault.sh <start|stop>
#
# Starts Vault in dev mode with a known root token (sr-test-vault container).
set -euo pipefail

CONTAINER_CMD="${CONTAINER_CMD:-docker}"
CONTAINER_NAME="sr-test-vault"
VAULT_PORT="${VAULT_PORT:-28200}"

case "${1:?Usage: setup-vault.sh <start|stop>}" in
  start)
    echo "Starting Vault in dev mode on port $VAULT_PORT..."
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD run -d --name "$CONTAINER_NAME" \
      -p "$VAULT_PORT:8200" \
      -e VAULT_DEV_ROOT_TOKEN_ID=root \
      -e VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200 \
      hashicorp/vault:1.15

    echo "Waiting for Vault to start..."
    for i in $(seq 1 30); do
      if curl -sf "http://localhost:$VAULT_PORT/v1/sys/health" | grep -q '"initialized":true'; then
        echo "Vault is ready"
        exit 0
      fi
      echo "  waiting... ($i/30)"
      sleep 2
    done
    echo "ERROR: Vault did not become ready in time"
    exit 1
    ;;

  stop)
    echo "Stopping Vault..."
    $CONTAINER_CMD stop "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    echo "Vault stopped and removed."
    ;;

  *)
    echo "Usage: setup-vault.sh <start|stop>"
    exit 1
    ;;
esac
