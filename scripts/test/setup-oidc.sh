#!/usr/bin/env bash
# setup-oidc.sh — Start/stop Keycloak for OIDC authentication tests.
# Usage: setup-oidc.sh <start|stop>
#
# Starts Keycloak (sr-test-keycloak), creates the schema-registry realm,
# client, groups, and users via the Admin REST API. This replicates the
# exact setup from .github/workflows/ci.yaml.
set -euo pipefail

CONTAINER_CMD="${CONTAINER_CMD:-docker}"
CONTAINER_NAME="sr-test-keycloak"
KC_PORT="${KC_PORT:-28080}"

case "${1:?Usage: setup-oidc.sh <start|stop>}" in
  start)
    echo "Starting Keycloak on port $KC_PORT..."
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD run -d --name "$CONTAINER_NAME" \
      -p "$KC_PORT:8080" \
      -e KEYCLOAK_ADMIN=admin \
      -e KEYCLOAK_ADMIN_PASSWORD=admin \
      -e KC_HTTP_PORT=8080 \
      -e KC_HEALTH_ENABLED=true \
      quay.io/keycloak/keycloak:24.0 start-dev

    echo "Waiting for Keycloak to start (this may take 1-2 minutes)..."
    for i in $(seq 1 60); do
      if curl -sf "http://localhost:$KC_PORT/health/ready" 2>/dev/null | grep -q '"status": "UP"'; then
        echo "Keycloak is ready"
        break
      fi
      if [ "$i" -eq 60 ]; then
        echo "ERROR: Keycloak did not become ready in time"
        exit 1
      fi
      echo "  waiting... ($i/60)"
      sleep 3
    done

    KC_BASE="http://localhost:$KC_PORT"

    # Get admin token
    echo "Getting admin token..."
    ADMIN_TOKEN=$(curl -sf -X POST "$KC_BASE/realms/master/protocol/openid-connect/token" \
      -H "Content-Type: application/x-www-form-urlencoded" \
      -d "username=admin" \
      -d "password=admin" \
      -d "grant_type=password" \
      -d "client_id=admin-cli" | jq -r '.access_token')

    if [ "$ADMIN_TOKEN" = "null" ] || [ -z "$ADMIN_TOKEN" ]; then
      echo "ERROR: Failed to get admin token"
      exit 1
    fi

    # Create the schema-registry realm
    echo "Creating schema-registry realm..."
    curl -sf -X POST "$KC_BASE/admin/realms" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "realm": "schema-registry",
        "enabled": true,
        "sslRequired": "none",
        "registrationAllowed": false,
        "loginWithEmailAllowed": true,
        "duplicateEmailsAllowed": false,
        "resetPasswordAllowed": false,
        "editUsernameAllowed": false,
        "bruteForceProtected": false
      }' || true

    # Create the client
    echo "Creating schema-registry client..."
    curl -sf -X POST "$KC_BASE/admin/realms/schema-registry/clients" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "clientId": "schema-registry",
        "enabled": true,
        "publicClient": false,
        "secret": "schema-registry-secret",
        "directAccessGrantsEnabled": true,
        "serviceAccountsEnabled": false,
        "standardFlowEnabled": true,
        "implicitFlowEnabled": false,
        "redirectUris": ["http://localhost:8081/*"],
        "webOrigins": ["*"],
        "protocol": "openid-connect",
        "fullScopeAllowed": true
      }' || true

    # Get the client UUID
    CLIENT_UUID=$(curl -sf "$KC_BASE/admin/realms/schema-registry/clients" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[] | select(.clientId=="schema-registry") | .id')

    # Add groups mapper to the client
    echo "Adding groups mapper..."
    curl -sf -X POST "$KC_BASE/admin/realms/schema-registry/clients/$CLIENT_UUID/protocol-mappers/models" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "name": "groups",
        "protocol": "openid-connect",
        "protocolMapper": "oidc-group-membership-mapper",
        "consentRequired": false,
        "config": {
          "full.path": "true",
          "id.token.claim": "true",
          "access.token.claim": "true",
          "claim.name": "groups",
          "userinfo.token.claim": "true"
        }
      }' || true

    # Create groups
    echo "Creating groups..."
    for group in schema-registry-admins developers readonly-users; do
      curl -sf -X POST "$KC_BASE/admin/realms/schema-registry/groups" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$group\"}" || true
    done

    # Get group IDs
    ADMIN_GROUP_ID=$(curl -sf "$KC_BASE/admin/realms/schema-registry/groups" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[] | select(.name=="schema-registry-admins") | .id')
    DEV_GROUP_ID=$(curl -sf "$KC_BASE/admin/realms/schema-registry/groups" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[] | select(.name=="developers") | .id')
    RO_GROUP_ID=$(curl -sf "$KC_BASE/admin/realms/schema-registry/groups" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[] | select(.name=="readonly-users") | .id')

    # Create users
    echo "Creating users..."
    curl -sf -X POST "$KC_BASE/admin/realms/schema-registry/users" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "username": "admin", "email": "admin@example.org",
        "firstName": "Admin", "lastName": "User",
        "enabled": true, "emailVerified": true,
        "credentials": [{"type": "password", "value": "adminpass", "temporary": false}]
      }' || true

    curl -sf -X POST "$KC_BASE/admin/realms/schema-registry/users" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "username": "developer", "email": "developer@example.org",
        "firstName": "Developer", "lastName": "User",
        "enabled": true, "emailVerified": true,
        "credentials": [{"type": "password", "value": "devpass", "temporary": false}]
      }' || true

    curl -sf -X POST "$KC_BASE/admin/realms/schema-registry/users" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "username": "readonly", "email": "readonly@example.org",
        "firstName": "ReadOnly", "lastName": "User",
        "enabled": true, "emailVerified": true,
        "credentials": [{"type": "password", "value": "readonlypass", "temporary": false}]
      }' || true

    # Get user IDs
    echo "Assigning users to groups..."
    ADMIN_USER_ID=$(curl -sf "$KC_BASE/admin/realms/schema-registry/users?username=admin" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')
    DEV_USER_ID=$(curl -sf "$KC_BASE/admin/realms/schema-registry/users?username=developer" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')
    RO_USER_ID=$(curl -sf "$KC_BASE/admin/realms/schema-registry/users?username=readonly" \
      -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.[0].id')

    # Add users to groups
    curl -sf -X PUT "$KC_BASE/admin/realms/schema-registry/users/$ADMIN_USER_ID/groups/$ADMIN_GROUP_ID" \
      -H "Authorization: Bearer $ADMIN_TOKEN" || true
    curl -sf -X PUT "$KC_BASE/admin/realms/schema-registry/users/$DEV_USER_ID/groups/$DEV_GROUP_ID" \
      -H "Authorization: Bearer $ADMIN_TOKEN" || true
    curl -sf -X PUT "$KC_BASE/admin/realms/schema-registry/users/$RO_USER_ID/groups/$RO_GROUP_ID" \
      -H "Authorization: Bearer $ADMIN_TOKEN" || true

    # Verify setup
    echo "Verifying Keycloak setup..."
    TOKEN_RESPONSE=$(curl -sf -X POST "$KC_BASE/realms/schema-registry/protocol/openid-connect/token" \
      -H "Content-Type: application/x-www-form-urlencoded" \
      -d "username=admin" \
      -d "password=adminpass" \
      -d "grant_type=password" \
      -d "client_id=schema-registry" \
      -d "client_secret=schema-registry-secret" \
      -d "scope=openid" 2>/dev/null) || true

    if echo "$TOKEN_RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
      echo "OIDC setup complete and verified."
    else
      echo "WARNING: Could not verify token retrieval"
    fi
    ;;

  stop)
    echo "Stopping Keycloak..."
    $CONTAINER_CMD stop "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    echo "Keycloak stopped and removed."
    ;;

  *)
    echo "Usage: setup-oidc.sh <start|stop>"
    exit 1
    ;;
esac
