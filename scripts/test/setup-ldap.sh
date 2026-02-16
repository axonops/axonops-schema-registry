#!/usr/bin/env bash
# setup-ldap.sh — Start/stop OpenLDAP for LDAP authentication tests.
# Usage: setup-ldap.sh <start|stop>
#
# Starts an OpenLDAP container (sr-test-openldap), loads the memberOf
# overlay, and bootstraps test users and groups from the existing
# LDIF files in tests/integration/testdata/ldap/.
set -euo pipefail

CONTAINER_CMD="${CONTAINER_CMD:-docker}"
CONTAINER_NAME="sr-test-openldap"
LDAP_PORT="${LDAP_PORT:-20389}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR/../.."
LDAP_TESTDATA="$PROJECT_ROOT/tests/integration/testdata/ldap"

case "${1:?Usage: setup-ldap.sh <start|stop>}" in
  start)
    echo "Starting OpenLDAP on port $LDAP_PORT..."
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD run -d --name "$CONTAINER_NAME" \
      -p "$LDAP_PORT:389" \
      -e LDAP_ORGANISATION="Example Org" \
      -e LDAP_DOMAIN="example.org" \
      -e LDAP_BASE_DN="dc=example,dc=org" \
      -e LDAP_ADMIN_PASSWORD="adminpassword" \
      -e LDAP_CONFIG_PASSWORD="configpassword" \
      osixia/openldap:1.5.0

    echo "Waiting for OpenLDAP to start..."
    for i in $(seq 1 30); do
      if $CONTAINER_CMD exec "$CONTAINER_NAME" ldapsearch -x -H ldap://localhost:389 \
        -b "dc=example,dc=org" -D "cn=admin,dc=example,dc=org" -w adminpassword \
        "(objectClass=organization)" 2>/dev/null | grep -q "example"; then
        echo "OpenLDAP is ready"
        break
      fi
      if [ "$i" -eq 30 ]; then
        echo "ERROR: OpenLDAP did not become ready in time"
        exit 1
      fi
      echo "  waiting... ($i/30)"
      sleep 2
    done

    # Load standard schemas (may already exist, ignore errors)
    echo "Loading LDAP schemas..."
    $CONTAINER_CMD exec "$CONTAINER_NAME" ldapadd -Y EXTERNAL -H ldapi:/// \
      -f /etc/ldap/schema/cosine.ldif 2>/dev/null || true
    $CONTAINER_CMD exec "$CONTAINER_NAME" ldapadd -Y EXTERNAL -H ldapi:/// \
      -f /etc/ldap/schema/nis.ldif 2>/dev/null || true
    $CONTAINER_CMD exec "$CONTAINER_NAME" ldapadd -Y EXTERNAL -H ldapi:/// \
      -f /etc/ldap/schema/inetorgperson.ldif 2>/dev/null || true

    # Enable memberOf overlay
    echo "Configuring memberOf overlay..."
    $CONTAINER_CMD cp "$LDAP_TESTDATA/memberof.ldif" "$CONTAINER_NAME:/tmp/memberof.ldif"
    $CONTAINER_CMD exec "$CONTAINER_NAME" ldapadd -Y EXTERNAL -H ldapi:/// \
      -f /tmp/memberof.ldif 2>/dev/null || true

    # Load bootstrap data (OUs, groups, users)
    echo "Loading test data..."
    $CONTAINER_CMD cp "$LDAP_TESTDATA/bootstrap.ldif" "$CONTAINER_NAME:/tmp/bootstrap.ldif"
    $CONTAINER_CMD exec "$CONTAINER_NAME" ldapadd -x -H ldap://localhost:389 \
      -D "cn=admin,dc=example,dc=org" -w adminpassword \
      -f /tmp/bootstrap.ldif

    echo "Verifying LDAP setup..."
    $CONTAINER_CMD exec "$CONTAINER_NAME" ldapsearch -x -H ldap://localhost:389 \
      -b "ou=Users,dc=example,dc=org" -D "cn=admin,dc=example,dc=org" -w adminpassword \
      "(objectClass=inetOrgPerson)" uid 2>/dev/null | grep -c "uid:" || true

    echo "LDAP setup complete."
    ;;

  stop)
    echo "Stopping OpenLDAP..."
    $CONTAINER_CMD stop "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    echo "OpenLDAP stopped and removed."
    ;;

  *)
    echo "Usage: setup-ldap.sh <start|stop>"
    exit 1
    ;;
esac
