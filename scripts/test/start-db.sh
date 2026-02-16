#!/usr/bin/env bash
# start-db.sh — Start a database container for testing.
# Usage: start-db.sh <postgres|mysql|cassandra>
#
# Containers use the sr-test- prefix to avoid conflicts with unrelated containers.
# Any existing container with the same name is force-removed (including volumes)
# before starting a fresh one, ensuring clean state.
set -euo pipefail

BACKEND="${1:?Usage: start-db.sh <postgres|mysql|cassandra>}"
CONTAINER_CMD="${CONTAINER_CMD:-docker}"

DB_USER="${DB_USER:-schemaregistry}"
DB_PASSWORD="${DB_PASSWORD:-schemaregistry}"
DB_DATABASE="${DB_DATABASE:-schemaregistry}"

case "$BACKEND" in
  postgres)
    CONTAINER_NAME="sr-test-postgres"
    PORT="${DB_POSTGRES_PORT:-5433}"

    echo "Starting PostgreSQL on port $PORT..."
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD run -d --name "$CONTAINER_NAME" \
      -e POSTGRES_USER="$DB_USER" \
      -e POSTGRES_PASSWORD="$DB_PASSWORD" \
      -e POSTGRES_DB="$DB_DATABASE" \
      -p "$PORT:5432" \
      postgres:15-alpine

    echo "Waiting for PostgreSQL to be ready..."
    for i in $(seq 1 60); do
      if $CONTAINER_CMD exec "$CONTAINER_NAME" pg_isready -U "$DB_USER" -d "$DB_DATABASE" 2>/dev/null; then
        echo "PostgreSQL is ready"
        exit 0
      fi
      echo "  waiting... ($i/60)"
      sleep 2
    done
    echo "ERROR: PostgreSQL did not become ready in time"
    exit 1
    ;;

  mysql)
    CONTAINER_NAME="sr-test-mysql"
    PORT="${DB_MYSQL_PORT:-3307}"

    echo "Starting MySQL on port $PORT..."
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD run -d --name "$CONTAINER_NAME" \
      -e MYSQL_ROOT_PASSWORD=root \
      -e MYSQL_USER="$DB_USER" \
      -e MYSQL_PASSWORD="$DB_PASSWORD" \
      -e MYSQL_DATABASE="$DB_DATABASE" \
      -p "$PORT:3306" \
      mysql:8.0

    echo "Waiting for MySQL to be ready..."
    for i in $(seq 1 60); do
      if $CONTAINER_CMD exec "$CONTAINER_NAME" mysqladmin ping -h 127.0.0.1 -u"$DB_USER" -p"$DB_PASSWORD" --silent 2>/dev/null; then
        echo "MySQL is ready"
        exit 0
      fi
      echo "  waiting... ($i/60)"
      sleep 2
    done
    echo "ERROR: MySQL did not become ready in time"
    exit 1
    ;;

  cassandra)
    CONTAINER_NAME="sr-test-cassandra"
    PORT="${DB_CASSANDRA_PORT:-9043}"

    echo "Starting Cassandra on port $PORT..."
    $CONTAINER_CMD rm -fv "$CONTAINER_NAME" 2>/dev/null || true
    $CONTAINER_CMD run -d --name "$CONTAINER_NAME" \
      -e CASSANDRA_CLUSTER_NAME=TestCluster \
      -e CASSANDRA_DC=dc1 \
      -e CASSANDRA_ENDPOINT_SNITCH=SimpleSnitch \
      -e MAX_HEAP_SIZE=512M \
      -e HEAP_NEWSIZE=100M \
      -p "$PORT:9042" \
      cassandra:5.0

    echo "Waiting for Cassandra to be ready (this may take several minutes)..."
    for i in $(seq 1 90); do
      if $CONTAINER_CMD exec "$CONTAINER_NAME" cqlsh -e "describe cluster" 2>/dev/null; then
        echo "Cassandra CQL is ready"
        break
      fi
      if [ "$i" -eq 90 ]; then
        echo "ERROR: Cassandra did not become ready in time"
        exit 1
      fi
      echo "  waiting... ($i/90)"
      sleep 4
    done

    echo "Creating keyspace..."
    $CONTAINER_CMD exec "$CONTAINER_NAME" cqlsh -e \
      "CREATE KEYSPACE IF NOT EXISTS $DB_DATABASE WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};"

    echo "Waiting for Cassandra to accept external connections on port $PORT..."
    for i in $(seq 1 30); do
      if (echo > /dev/tcp/127.0.0.1/"$PORT") 2>/dev/null; then
        echo "Cassandra port $PORT is open"
        sleep 5  # Extra wait for native protocol to be fully ready
        echo "Cassandra is ready"
        exit 0
      fi
      echo "  waiting for port... ($i/30)"
      sleep 2
    done
    echo "ERROR: Cassandra port did not become accessible in time"
    exit 1
    ;;

  *)
    echo "Unknown backend: $BACKEND"
    echo "Usage: start-db.sh <postgres|mysql|cassandra>"
    exit 1
    ;;
esac
