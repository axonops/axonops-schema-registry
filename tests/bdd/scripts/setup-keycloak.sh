#!/bin/sh
# setup-keycloak.sh — Provision Keycloak for OIDC BDD tests.
# Runs as a Docker Compose init container (curlimages/curl:8.5.0).
#
# Creates: realm, client, groups mapper, 3 groups, 4 users with assignments.
# Replicates the setup from scripts/test/setup-oidc.sh adapted for Docker Compose.
set -eu

KC_BASE="${KC_BASE:-http://keycloak:8080}"
MAX_RETRIES=60
RETRY_INTERVAL=3

echo "Waiting for Keycloak to become ready..."
i=0
while [ "$i" -lt "$MAX_RETRIES" ]; do
  if curl -sf "$KC_BASE/health/ready" 2>/dev/null | grep -q '"UP"'; then
    echo "Keycloak is ready"
    break
  fi
  i=$((i + 1))
  if [ "$i" -eq "$MAX_RETRIES" ]; then
    echo "ERROR: Keycloak did not become ready in time"
    exit 1
  fi
  echo "  waiting... ($i/$MAX_RETRIES)"
  sleep "$RETRY_INTERVAL"
done

# Get admin token
echo "Getting admin token..."
ADMIN_TOKEN=$(curl -sf -X POST "$KC_BASE/realms/master/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin" \
  -d "password=admin" \
  -d "grant_type=password" \
  -d "client_id=admin-cli" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')

if [ -z "$ADMIN_TOKEN" ]; then
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
  -H "Authorization: Bearer $ADMIN_TOKEN" | sed -n 's/.*"id":"\([^"]*\)".*"clientId":"schema-registry".*/\1/p')

if [ -z "$CLIENT_UUID" ]; then
  # Try alternative parsing — the client may appear at different position in JSON array
  CLIENT_UUID=$(curl -sf "$KC_BASE/admin/realms/schema-registry/clients?clientId=schema-registry" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1)
fi

echo "Client UUID: $CLIENT_UUID"

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

# Helper: get group ID by name
get_group_id() {
  curl -sf "$KC_BASE/admin/realms/schema-registry/groups?search=$1" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | sed -n "s/.*\"id\":\"\([^\"]*\)\".*\"name\":\"$1\".*/\1/p" | head -1
}

ADMIN_GROUP_ID=$(get_group_id "schema-registry-admins")
DEV_GROUP_ID=$(get_group_id "developers")
RO_GROUP_ID=$(get_group_id "readonly-users")

echo "Group IDs: admins=$ADMIN_GROUP_ID, devs=$DEV_GROUP_ID, ro=$RO_GROUP_ID"

# Create users
echo "Creating users..."
create_user() {
  local username=$1 email=$2 firstname=$3 password=$4
  curl -sf -X POST "$KC_BASE/admin/realms/schema-registry/users" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"username\": \"$username\",
      \"email\": \"$email\",
      \"firstName\": \"$firstname\",
      \"lastName\": \"User\",
      \"enabled\": true,
      \"emailVerified\": true,
      \"credentials\": [{\"type\": \"password\", \"value\": \"$password\", \"temporary\": false}]
    }" || true
}

create_user "admin" "admin@example.org" "Admin" "adminpass"
create_user "developer" "developer@example.org" "Developer" "devpass"
create_user "readonly" "readonly@example.org" "ReadOnly" "readonlypass"
create_user "nogroup" "nogroup@example.org" "NoGroup" "nogrouppass"

# Helper: get user ID by username
get_user_id() {
  curl -sf "$KC_BASE/admin/realms/schema-registry/users?username=$1&exact=true" \
    -H "Authorization: Bearer $ADMIN_TOKEN" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p' | head -1
}

# Assign users to groups
echo "Assigning users to groups..."
ADMIN_USER_ID=$(get_user_id "admin")
DEV_USER_ID=$(get_user_id "developer")
RO_USER_ID=$(get_user_id "readonly")

curl -sf -X PUT "$KC_BASE/admin/realms/schema-registry/users/$ADMIN_USER_ID/groups/$ADMIN_GROUP_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" || true
curl -sf -X PUT "$KC_BASE/admin/realms/schema-registry/users/$DEV_USER_ID/groups/$DEV_GROUP_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" || true
curl -sf -X PUT "$KC_BASE/admin/realms/schema-registry/users/$RO_USER_ID/groups/$RO_GROUP_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" || true

# nogroup user intentionally has no group assignment

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

if echo "$TOKEN_RESPONSE" | grep -q '"access_token"'; then
  echo "OIDC setup complete and verified."
else
  echo "WARNING: Could not verify token retrieval"
  echo "Response: $TOKEN_RESPONSE"
fi
