# Storage Backends

## Contents

- [Overview](#overview)
- [Backend Comparison](#backend-comparison)
- [PostgreSQL](#postgresql)
  - [Configuration](#configuration)
  - [Environment Variable Overrides](#environment-variable-overrides)
- [MySQL](#mysql)
  - [Configuration](#configuration-1)
  - [Environment Variable Overrides](#environment-variable-overrides-1)
- [Cassandra](#cassandra)
  - [Data Model](#data-model)
  - [Concurrency and Consistency](#concurrency-and-consistency)
  - [Keyspace Management](#keyspace-management)
  - [Configuration](#configuration-2)
- [Memory](#memory)
  - [Configuration](#configuration-3)
- [Auth Storage](#auth-storage)
  - [Vault Auth Storage](#vault-auth-storage)
- [Database Setup](#database-setup)
  - [PostgreSQL](#postgresql-1)
  - [MySQL](#mysql-1)
  - [Cassandra](#cassandra-1)
- [Switching Backends](#switching-backends)
- [Further Reading](#further-reading)

## Overview

All storage backends implement the same `Storage` interface, which defines approximately 40 methods covering schema operations, subject management, configuration, mode settings, ID generation, import/export, reference tracking, and authentication (user and API key management). The registry is backend-agnostic: switching backends requires only a configuration change and the appropriate database setup. Schema migrations run automatically on startup, creating all required tables and indexes.

The storage layer uses a factory pattern. Each backend registers itself at init time, and the registry creates the appropriate store based on the `storage.type` value in the configuration file.

## Backend Comparison

| Feature | PostgreSQL | MySQL | Cassandra | Memory |
|---------|-----------|-------|-----------|--------|
| Persistence | Yes | Yes | Yes | No |
| ACID Transactions | Yes | Yes | No (tunable consistency) | N/A |
| Multi-DC Replication | Via streaming replication | Via replication | Native | N/A |
| Concurrency Model | Transactions with row-level locking | Row-level locking (`SELECT ... FOR UPDATE`) | LWT + SAI indexes | `sync.RWMutex` |
| Connection Pooling | Configurable (`max_open_conns`, `max_idle_conns`) | Configurable (`max_open_conns`, `max_idle_conns`) | Driver-managed | N/A |
| Prepared Statements | Yes | Yes | No (inline CQL) | N/A |
| Minimum Version | PostgreSQL 12+ | MySQL 8.0+ | Cassandra 5.0+ | N/A |
| Production Ready | Yes | Yes | Yes | No |
| Recommended For | Most deployments | Existing MySQL infrastructure | Global / multi-DC | Development and testing |

## PostgreSQL

PostgreSQL is the recommended backend for most production deployments.

**Concurrency and consistency.** Schema registration and ID allocation use transactions with `BeginTx`. The `schemas` table enforces uniqueness on `(subject, version)` and `(subject, fingerprint)` via unique constraints, preventing duplicate registrations at the database level.

**Connection pooling.** The driver-level connection pool is configurable through `max_open_conns` (default 25), `max_idle_conns` (default 5), `conn_max_lifetime`, and `conn_max_idle_time` (both default 5 minutes).

**SSL/TLS.** The `ssl_mode` parameter supports the standard PostgreSQL modes: `disable`, `allow`, `prefer`, `require`, `verify-ca`, and `verify-full`.

**Prepared statements.** All frequently-used queries are prepared at startup for better performance, covering schema lookups, config operations, user management, and API key operations.

**Auto-migration.** On first startup, the migration system creates all required tables (`schemas`, `schema_references`, `configs`, `modes`, `users`, `api_keys`), indexes, and a `schema_versions` view. Subsequent migrations add columns incrementally using `IF NOT EXISTS` clauses for idempotency. Global defaults for compatibility (`BACKWARD`) and mode (`READWRITE`) are inserted on creation.

### Configuration

```yaml
storage:
  type: postgresql
  postgresql:
    host: localhost
    port: 5432
    database: schema_registry
    user: schemaregistry
    password: schemaregistry
    ssl_mode: disable
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300  # seconds
```

### Environment Variable Overrides

| Variable | Description |
|----------|-------------|
| `SCHEMA_REGISTRY_PG_HOST` | PostgreSQL host |
| `SCHEMA_REGISTRY_PG_PORT` | PostgreSQL port |
| `SCHEMA_REGISTRY_PG_DATABASE` | Database name |
| `SCHEMA_REGISTRY_PG_USER` | Database user |
| `SCHEMA_REGISTRY_PG_PASSWORD` | Database password |
| `SCHEMA_REGISTRY_PG_SSLMODE` | SSL mode |

## MySQL

MySQL is a good choice when MySQL is already part of the infrastructure.

**Concurrency and consistency.** ID allocation uses `SELECT ... FOR UPDATE` within a transaction to guarantee sequential, conflict-free schema IDs. The `schemas` table enforces uniqueness on `(subject, version)` and `(subject, fingerprint)` via unique keys. All tables use the InnoDB engine with `utf8mb4_unicode_ci` collation.

**Connection pooling.** Same configurable pool parameters as PostgreSQL: `max_open_conns` (default 25), `max_idle_conns` (default 5), `conn_max_lifetime`, and `conn_max_idle_time` (both default 5 minutes).

**TLS.** The `tls` parameter supports: `true`, `false`, `skip-verify`, and `preferred`.

**Prepared statements.** All frequently-used queries are prepared at startup, matching the PostgreSQL backend in scope.

**Auto-migration.** Creates all required tables, indexes, and the `id_alloc` table used for sequential ID generation. Migrations use `IF NOT EXISTS` and `INSERT IGNORE` for idempotency. Column additions for metadata, rulesets, and configuration extensions are applied incrementally.

### Configuration

```yaml
storage:
  type: mysql
  mysql:
    host: localhost
    port: 3306
    database: schema_registry
    user: schemaregistry
    password: schemaregistry
    tls: "false"
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300  # seconds
```

### Environment Variable Overrides

| Variable | Description |
|----------|-------------|
| `SCHEMA_REGISTRY_MYSQL_HOST` | MySQL host |
| `SCHEMA_REGISTRY_MYSQL_PORT` | MySQL port |
| `SCHEMA_REGISTRY_MYSQL_DATABASE` | Database name |
| `SCHEMA_REGISTRY_MYSQL_USER` | Database user |
| `SCHEMA_REGISTRY_MYSQL_PASSWORD` | Database password |
| `SCHEMA_REGISTRY_MYSQL_TLS` | TLS mode |

## Cassandra

Cassandra is the best choice for multi-datacenter and globally distributed deployments.

**Requires Cassandra 5.0+** for Storage Attached Index (SAI) support. SAI indexes replace legacy lookup tables, enabling efficient secondary queries without maintaining separate denormalized tables for each access pattern.

### Data Model

The Cassandra backend uses 15 tables:

| Table | Purpose |
|-------|---------|
| `schemas_by_id` | Primary schema lookup by global ID; SAI index on `fingerprint` for deduplication |
| `subject_versions` | Versions within a subject, partitioned by subject; SAI indexes on `schema_id` and `deleted` |
| `subject_latest` | Latest version per subject; also used for subject listing |
| `schema_references` | Schema dependencies (partitioned by `schema_id`) |
| `references_by_target` | Reverse lookup for "referenced by" queries |
| `subject_configs` | Per-subject compatibility configuration |
| `global_config` | Global compatibility configuration |
| `modes` | Registry operating mode (READWRITE/READONLY/IMPORT) |
| `id_alloc` | Block-based ID allocation via LWT |
| `schema_fingerprints` | Atomic fingerprint-to-schema-ID deduplication via LWT |
| `users_by_id` | User records |
| `users_by_email` | User lookup by email/username |
| `api_keys_by_id` | API key records |
| `api_keys_by_user` | API keys partitioned by user |
| `api_keys_by_hash` | API key lookup by hash for authentication |

### Concurrency and Consistency

**Lightweight Transactions (LWT).** Used for two critical operations:
- **ID allocation:** Block-based reservation via `INSERT ... IF NOT EXISTS` on the `id_alloc` table. The default block size is 50, meaning each LWT call reserves 50 IDs. This reduces LWT frequency by approximately 50x compared to per-ID allocation.
- **Fingerprint deduplication:** The `schema_fingerprints` table uses `INSERT ... IF NOT EXISTS` to guarantee exactly one `schema_id` per fingerprint, preventing concurrent writers from allocating duplicate IDs.

**Tunable consistency.** Read and write consistency levels can be configured independently:
- `write_consistency`: `LOCAL_QUORUM` is recommended for production.
- `read_consistency`: `LOCAL_ONE` for lower latency, `LOCAL_QUORUM` for read-your-writes guarantees.
- If neither `read_consistency` nor `write_consistency` is specified, the `consistency` value is used for both (default: `LOCAL_QUORUM`).

**Datacenter-aware routing.** When `local_dc` is configured, the driver uses `DCAwareRoundRobinPolicy` to prefer local nodes.

### Keyspace Management

The migration creates the keyspace with `SimpleStrategy` and replication factor 1 if it does not exist. For production multi-datacenter deployments, create the keyspace manually with `NetworkTopologyStrategy` before starting the registry:

```cql
CREATE KEYSPACE axonops_schema_registry
  WITH REPLICATION = {
    'class': 'NetworkTopologyStrategy',
    'dc1': 3,
    'dc2': 3
  };
```

The migration will skip keyspace creation if the keyspace already exists.

### Configuration

```yaml
storage:
  type: cassandra
  cassandra:
    hosts:
      - cassandra-node1
      - cassandra-node2
      - cassandra-node3
    keyspace: axonops_schema_registry
    consistency: LOCAL_QUORUM
    read_consistency: LOCAL_ONE
    write_consistency: LOCAL_QUORUM
    username: cassandra
    password: cassandra
    local_dc: dc1
    id_block_size: 50       # IDs reserved per LWT call (default: 50)
    max_retries: 50         # Retries for CAS operations (default: 50)
    timeout: 10s
    connect_timeout: 10s
```

## Memory

The in-memory backend stores all data in Go maps protected by a `sync.RWMutex`. It requires no external dependencies and provides the fastest possible read/write performance.

**No persistence.** All data is lost when the process exits. This backend is intended for development, testing, and CI pipelines only.

**Thread-safe.** Concurrent access is safe. Read operations acquire a read lock; write operations acquire a write lock.

**Schema deduplication.** Fingerprint-based deduplication works identically to the persistent backends, using in-memory maps for global fingerprint tracking.

### Configuration

```yaml
storage:
  type: memory
```

No additional configuration is required.

## Auth Storage

By default, user accounts and API keys are stored in the same database as schemas. All persistent backends (PostgreSQL, MySQL, Cassandra) implement the full `AuthStorage` interface alongside schema storage.

To separate authentication data from schema data, set `storage.auth_type` to use a different backend for auth operations. The primary use case is storing credentials in HashiCorp Vault.

### Vault Auth Storage

When `auth_type` is set to `vault`, user records and API keys are stored in Vault's KV v2 secrets engine. Schema data remains in the primary storage backend.

```yaml
storage:
  type: postgresql        # Schemas stored in PostgreSQL
  auth_type: vault        # Auth data stored in Vault
  postgresql:
    host: localhost
    port: 5432
    database: schema_registry
    user: schemaregistry
    password: schemaregistry
  vault:
    address: https://vault.example.com:8200
    token: ${VAULT_TOKEN}
    namespace: admin
    mount_path: secret
    base_path: schema-registry
    tls_ca_file: /etc/ssl/certs/vault-ca.pem
```

#### Vault Environment Variable Overrides

| Variable | Description |
|----------|-------------|
| `SCHEMA_REGISTRY_VAULT_ADDRESS` | Vault server address |
| `SCHEMA_REGISTRY_VAULT_TOKEN` | Vault authentication token |
| `VAULT_TOKEN` | Standard Vault token (used if `SCHEMA_REGISTRY_VAULT_TOKEN` is not set) |
| `SCHEMA_REGISTRY_VAULT_NAMESPACE` | Vault namespace |
| `VAULT_NAMESPACE` | Standard Vault namespace (used if `SCHEMA_REGISTRY_VAULT_NAMESPACE` is not set) |
| `SCHEMA_REGISTRY_VAULT_MOUNT_PATH` | KV secrets engine mount path |
| `SCHEMA_REGISTRY_VAULT_BASE_PATH` | Base path for registry data |

## Database Setup

### PostgreSQL

```sql
CREATE DATABASE schema_registry;
CREATE USER schemaregistry WITH PASSWORD 'changeme';
GRANT ALL PRIVILEGES ON DATABASE schema_registry TO schemaregistry;

-- Connect to the schema_registry database and grant schema privileges
\c schema_registry
GRANT ALL ON SCHEMA public TO schemaregistry;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO schemaregistry;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO schemaregistry;
```

All tables and indexes are created automatically on first startup.

### MySQL

```sql
CREATE DATABASE schema_registry CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'schemaregistry'@'%' IDENTIFIED BY 'changeme';
GRANT ALL PRIVILEGES ON schema_registry.* TO 'schemaregistry'@'%';
FLUSH PRIVILEGES;
```

All tables and indexes are created automatically on first startup.

### Cassandra

No manual setup is required. The migration creates the keyspace, all tables, SAI indexes, and initializes default configuration values automatically.

For production multi-datacenter deployments, pre-create the keyspace with `NetworkTopologyStrategy` as shown in the [Keyspace Management](#keyspace-management) section above, then point the registry at the existing keyspace. The migration will create tables within the existing keyspace without modifying its replication settings.

## Switching Backends

To switch from one storage backend to another:

1. Update `storage.type` in the configuration file.
2. Provide the connection details for the new backend.
3. Restart the registry. Auto-migration will create the schema in the new database.

Changing the backend does not migrate existing data. To transfer schemas between backends, use the import/export API:

1. Export schemas from the source registry using the REST API (`GET /schemas`, `GET /subjects`, etc.).
2. Import schemas into the target registry using `POST /import/schemas`, which preserves original schema IDs.
3. After import, the registry automatically adjusts the ID sequence to prevent conflicts.

For details on migrating data from Confluent Schema Registry, see [Migration](migration.md).

## Further Reading

- [Configuration](configuration.md) -- full configuration reference
- [Deployment](deployment.md) -- production deployment guidance
- [Getting Started](getting-started.md) -- quick start guide
