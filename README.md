# AxonOps Schema Registry

A high-performance, API-compatible Kafka Schema Registry written in Go. Drop-in replacement for Confluent Schema Registry with enterprise features including multiple storage backends, flexible authentication, and comprehensive audit logging.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Feature Comparison](#feature-comparison)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [Authentication](#authentication)
- [API Reference](#api-reference)
- [Admin CLI](#admin-cli)
- [Monitoring](#monitoring)
- [License](#license)

## Overview

AxonOps Schema Registry provides centralized schema management for Apache Kafka ecosystems. It enables:

- **Schema Evolution**: Safely evolve schemas with compatibility checking
- **Data Governance**: Centralized schema storage with version control
- **Serialization**: Integration with Kafka producers/consumers for Avro, Protobuf, and JSON Schema
- **API Compatibility**: Drop-in replacement for Confluent Schema Registry API

### API Compatibility Scope

AxonOps Schema Registry implements the Confluent Schema Registry REST API v1. Compatibility includes:

- **Supported Endpoints**: All schema, subject, compatibility, config, and mode endpoints
- **Serializers**: Compatible with Confluent's Avro, Protobuf, and JSON Schema serializers (Java, Go, Python)
- **Client Libraries**: Works with `confluent-kafka-go`, `confluent-kafka-python`, and Java Kafka clients
- **Error Codes**: HTTP status codes and error response formats match Confluent behavior
- **Authentication**: API keys work with `-u "API_KEY:x"` format (key as username, password ignored)

**Known Differences**:
- No `/schemas` listing endpoint (Confluent Enterprise only)
- Contexts feature not implemented
- Schema registry cluster coordination uses database constraints instead of Kafka

### Why AxonOps Schema Registry?

- **No Kafka/ZooKeeper Dependency**: Unlike Confluent, uses standard databases (PostgreSQL, MySQL, Cassandra) for storage
- **Lightweight**: Single binary, minimal resource footprint
- **Enterprise Security**: LDAP, OIDC, mTLS, API keys, and RBAC out of the box
- **Cloud Native**: Designed for Kubernetes with health checks, metrics, and graceful shutdown

## Features

### Schema Management

| Feature | Description |
|---------|-------------|
| **Multi-Format Support** | Avro, Protocol Buffers (proto2/proto3), JSON Schema |
| **Schema References** | Schemas can reference other schemas |
| **Compatibility Modes** | NONE, BACKWARD, FORWARD, FULL (with transitive variants) |
| **Subject Naming** | TopicNameStrategy, RecordNameStrategy, TopicRecordNameStrategy |
| **Soft Delete** | Recoverable schema deletion with permanent delete option |
| **Schema Normalization** | Canonical form generation for deduplication |

### Storage Backends

| Backend | Use Case | Features |
|---------|----------|----------|
| **PostgreSQL** | Production | ACID transactions, connection pooling, SSL |
| **MySQL** | Production | ACID transactions, connection pooling, TLS |
| **Cassandra** | Distributed/HA | Multi-datacenter, tunable consistency |
| **Memory** | Development/Testing | No persistence, fast iteration |

**Auth Storage**: Users and API keys can be stored in the schema database or separately in HashiCorp Vault.

### Credential Storage

- **Passwords**: Stored as bcrypt hashes (cost factor 10). Never stored in plaintext.
- **API Keys**: Stored as SHA-256 hashes (optionally HMAC with server secret). Shown only once at creation.
- **Server Secret**: If `API_KEY_SECRET` is configured, all instances must use the same value.
- **Storage Options**: Credentials stored in the same database as schemas, or in HashiCorp Vault for centralized secrets management.

### Authentication Methods

| Method | Description |
|--------|-------------|
| **Basic Auth** | Username/password with bcrypt hashing |
| **API Keys** | Header or query parameter, with expiration |
| **LDAP/AD** | Enterprise directory integration with group mapping |
| **OIDC** | OpenID Connect (Keycloak, Okta, Auth0, etc.) |
| **mTLS** | Mutual TLS client certificate authentication |

### Authorization & Security

- **RBAC**: Role-based access control (super_admin, admin, developer, readonly)
- **Granular Permissions**: schema:read/write/delete, config:read/write, mode:read/write
- **Rate Limiting**: Token bucket algorithm, per-client or global
- **Audit Logging**: Comprehensive event logging to file
- **TLS**: Full TLS support with auto-reload certificates

### Operations

- **Prometheus Metrics**: Request latency, schema counts, error rates
- **Health Checks**: Liveness and readiness endpoints
- **Graceful Shutdown**: Clean connection draining
- **Database Migrations**: Automatic schema creation and upgrades

## Feature Comparison

*Comparison based on upstream/default configurations. Third-party plugins may extend capabilities.*

| Feature | AxonOps | Confluent OSS | Confluent Enterprise | Karapace |
|---------|---------|---------------|---------------------|----------|
| **License** | Apache 2.0 | Confluent Community | Commercial | Apache 2.0 |
| **Language** | Go | Java | Java | Python |
| **API Compatibility** | Full¹ | N/A | N/A | Full |
| **Avro Support** | Yes | Yes | Yes | Yes |
| **Protobuf Support** | Yes | Yes | Yes | Yes |
| **JSON Schema Support** | Yes | Yes | Yes | Yes |
| **Schema References** | Yes | Yes | Yes | Yes |
| **Compatibility Modes** | All 7 modes | All 7 modes | All 7 modes | All 7 modes |
| **Storage: Kafka** | No | Yes | Yes | Yes |
| **Storage: PostgreSQL** | Yes | No | No | No |
| **Storage: MySQL** | Yes | No | No | No |
| **Storage: Cassandra** | Yes | No | No | No |
| **No Kafka Dependency** | Yes | No | No | No |
| **Basic Auth** | Yes | No | Yes | Yes |
| **API Keys** | Yes | No | Yes² | No |
| **LDAP/AD** | Yes | No | Yes | No |
| **OIDC/OAuth2** | Yes | No | Yes | No |
| **mTLS** | Yes | Yes | Yes | Yes |
| **RBAC** | Yes | No | Yes | Limited |
| **Audit Logging** | Yes | No | Yes | No |
| **Rate Limiting** | Yes | No | No | No |
| **Prometheus Metrics** | Yes | Yes | Yes | Yes |
| **REST Proxy** | No | Separate | Separate | Yes |
| **Multi-Tenant** | Planned³ | No | Yes | No |
| **Schema Validation** | Yes | Yes | Yes | Yes |
| **Single Binary** | Yes | No | No | No |
| **Memory Footprint** | ~50MB | ~500MB+ | ~500MB+ | ~200MB+ |

¹ See [API Compatibility Scope](#api-compatibility-scope) for details.
² Confluent Cloud uses key:secret format; AxonOps uses key-only with password ignored.
³ Multi-tenancy roadmap: subject prefixes with per-tenant auth scope.

## Architecture

### Single Instance Deployment

For development, testing, or low-traffic environments, a single instance deployment is straightforward:

![Single Instance Architecture](assets/architecture-single.svg)

**Characteristics:**
- Single binary deployment (~50MB memory)
- No external dependencies except the database
- Suitable for development and small deployments

### High Availability Deployment (PostgreSQL/MySQL)

For production environments requiring high availability, deploy multiple stateless instances behind a load balancer.

**Write Path:**

![HA Write Path](assets/architecture-ha-write.svg)

**Read Path:**

![HA Read Path](assets/architecture-ha-read.svg)

**Key Features:**
- **Stateless Design**: Any instance can handle any request
- **No Leader Election**: No coordination required between instances
- **Horizontal Scaling**: Add instances as needed
- **API Key Caching**: In-memory cache with periodic refresh for consistency across instances
- **Database HA**: Use PostgreSQL streaming replication or MySQL group replication

**Concurrency Handling:**
- **Unique Constraints**: Database enforces `(subject, fingerprint)` uniqueness for schema deduplication
- **Transaction Isolation**: SERIALIZABLE isolation prevents concurrent version assignment races
- **Idempotent Registration**: Same schema content always returns the same ID, even under concurrent requests

**Deployment Requirements:**
- Load balancer (HAProxy, Nginx, AWS ALB, etc.)
- 2+ Schema Registry instances
- PostgreSQL/MySQL with replication for database HA
- Optional: HashiCorp Vault for centralized auth storage

### Distributed Multi-Datacenter Deployment (Cassandra)

For global deployments requiring multi-datacenter support and the highest availability:

![Distributed Architecture](assets/architecture-distributed.svg)

**Key Features:**
- **Active-Active**: All datacenters serve read and write traffic
- **Automatic Replication**: Cassandra handles cross-DC replication
- **Disaster Recovery**: Survive complete datacenter failures

**Consistency Settings:**
- **Write Consistency**: `LOCAL_QUORUM` recommended for durability within datacenter
- **Read Consistency**: `LOCAL_ONE` for low latency, `LOCAL_QUORUM` for read-your-writes
- **Version Assignment**: Uses Cassandra lightweight transactions (LWT) for atomic version assignment
- **Latest Schema**: May return slightly stale data under eventual consistency; configure `read_consistency: LOCAL_QUORUM` for strong consistency

**Deployment Requirements:**
- Cassandra cluster with multi-DC replication (NetworkTopologyStrategy)
- Schema Registry instances in each datacenter
- Local load balancer per datacenter
- DNS-based global load balancing (optional)

### Authentication Flow

The authentication module supports multiple methods that can be enabled simultaneously:

![Authentication Flow](assets/auth-flow.svg)

**Authentication Order:**
1. API Key (header or query parameter)
2. Basic Auth (database, then LDAP fallback)
3. OIDC Bearer Token
4. mTLS Client Certificate

### Schema Registration Flow

Schema registration includes parsing, validation, and compatibility checking:

![Schema Registration Flow](assets/schema-flow.svg)

**Registration Steps:**
1. Client sends schema to `/subjects/{subject}/versions`
2. Auth module validates credentials and permissions
3. Parser validates syntax and generates canonical form
4. Fingerprint (SHA-256) calculated for deduplication
5. Compatibility checked against previous versions
6. Schema stored and ID returned



## Quick Start

### Using Docker

```bash
# Start with in-memory storage (for testing)
docker run -d -p 8081:8081 ghcr.io/axonops/axonops-schema-registry:latest

# Start with PostgreSQL
docker run -d -p 8081:8081 \
  -e STORAGE_TYPE=postgresql \
  -e POSTGRES_HOST=postgres.example.com \
  -e POSTGRES_USER=schemaregistry \
  -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DATABASE=schemaregistry \
  ghcr.io/axonops/axonops-schema-registry:latest
```

### Using Binary

```bash
# Download and extract
curl -LO https://github.com/axonops/axonops-schema-registry/releases/latest/download/axonops-schema-registry-linux-amd64.tar.gz
tar xzf axonops-schema-registry-linux-amd64.tar.gz
cd axonops-schema-registry-*

# Run with default config (in-memory storage)
./schema-registry

# Run with custom config
./schema-registry --config /etc/axonops-schema-registry/config.yaml
```

### Test the API

```bash
# Check health
curl http://localhost:8081/

# Register a schema
curl -X POST http://localhost:8081/subjects/test-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"schema": "{\"type\": \"record\", \"name\": \"Test\", \"fields\": [{\"name\": \"id\", \"type\": \"int\"}]}"}'

# Get schema
curl http://localhost:8081/subjects/test-value/versions/latest
```

## Installation

### Package Installation

#### Debian/Ubuntu (APT)

```bash
# Install dependencies
sudo apt-get update
sudo apt-get install -y curl gnupg ca-certificates

# Add AxonOps repository
curl -L https://packages.axonops.com/apt/repo-signing-key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/axonops.gpg
echo "deb [signed-by=/usr/share/keyrings/axonops.gpg] https://packages.axonops.com/apt axonops-apt main" | sudo tee /etc/apt/sources.list.d/axonops-apt.list

# Install
sudo apt-get update
sudo apt-get install -y axonops-schema-registry

# Configure
sudo cp /etc/axonops-schema-registry/config.example.yaml /etc/axonops-schema-registry/config.yaml
sudo vim /etc/axonops-schema-registry/config.yaml

# Start service
sudo systemctl enable axonops-schema-registry
sudo systemctl start axonops-schema-registry
```

#### RHEL/CentOS/Fedora (YUM)

```bash
# Add AxonOps repository
sudo tee /etc/yum.repos.d/axonops-yum.repo << 'EOF'
[axonops-yum]
name=axonops-yum
baseurl=https://packages.axonops.com/yum/
enabled=1
repo_gpgcheck=0
gpgcheck=0
EOF

# Install
sudo yum makecache
sudo yum install -y axonops-schema-registry

# Configure
sudo cp /etc/axonops-schema-registry/config.example.yaml /etc/axonops-schema-registry/config.yaml
sudo vim /etc/axonops-schema-registry/config.yaml

# Start service
sudo systemctl enable axonops-schema-registry
sudo systemctl start axonops-schema-registry
```

### Binary Installation

```bash
# Download
curl -LO https://github.com/axonops/axonops-schema-registry/releases/latest/download/axonops-schema-registry-1.0.0-linux-amd64.tar.gz

# Extract
tar xzf axonops-schema-registry-1.0.0-linux-amd64.tar.gz

# Move binaries
sudo mv axonops-schema-registry-*/schema-registry /usr/local/bin/
sudo mv axonops-schema-registry-*/schema-registry-admin /usr/local/bin/

# Create config directory
sudo mkdir -p /etc/axonops-schema-registry
sudo cp axonops-schema-registry-*/config.example.yaml /etc/axonops-schema-registry/config.yaml
```

### Docker Installation

Docker images are available from GitHub Container Registry:

```bash
# Pull the image
docker pull ghcr.io/axonops/axonops-schema-registry:latest

# Available tags:
#   - latest       (latest stable release)
#   - 1.0.0        (specific version)
#   - 1.0          (latest patch for minor version)
#   - 1            (latest minor/patch for major version)

# Run with environment variables
docker run -d \
  --name schema-registry \
  -p 8081:8081 \
  -e STORAGE_TYPE=postgresql \
  -e POSTGRES_HOST=postgres \
  -e POSTGRES_DATABASE=schemaregistry \
  -e POSTGRES_USER=schemaregistry \
  -e POSTGRES_PASSWORD=secret \
  ghcr.io/axonops/axonops-schema-registry:latest

# Run with config file
docker run -d \
  --name schema-registry \
  -p 8081:8081 \
  -v /path/to/config.yaml:/etc/axonops-schema-registry/config.yaml \
  ghcr.io/axonops/axonops-schema-registry:latest --config /etc/axonops-schema-registry/config.yaml
```

### Kubernetes Installation

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: schema-registry
spec:
  replicas: 3
  selector:
    matchLabels:
      app: schema-registry
  template:
    metadata:
      labels:
        app: schema-registry
    spec:
      containers:
      - name: schema-registry
        image: ghcr.io/axonops/axonops-schema-registry:latest
        ports:
        - containerPort: 8081
        env:
        - name: STORAGE_TYPE
          value: postgresql
        - name: POSTGRES_HOST
          value: postgres-service
        - name: POSTGRES_DATABASE
          value: schemaregistry
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: schema-registry-secrets
              key: postgres-user
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: schema-registry-secrets
              key: postgres-password
        livenessProbe:
          httpGet:
            path: /
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: schema-registry
spec:
  selector:
    app: schema-registry
  ports:
  - port: 8081
    targetPort: 8081
```

## Configuration

### Configuration File

Create a configuration file at `/etc/axonops-schema-registry/config.yaml`:

```yaml
# Server settings
server:
  host: "0.0.0.0"
  port: 8081
  read_timeout: 30   # seconds
  write_timeout: 30  # seconds

# Storage backend
storage:
  type: postgresql  # memory, postgresql, mysql, cassandra

  # Optional: separate storage for auth (users/API keys)
  # auth_type: vault

  postgresql:
    host: ${POSTGRES_HOST:localhost}
    port: ${POSTGRES_PORT:5432}
    database: ${POSTGRES_DATABASE:schemaregistry}
    user: ${POSTGRES_USER:schemaregistry}
    password: ${POSTGRES_PASSWORD}
    ssl_mode: prefer  # disable, allow, prefer, require, verify-ca, verify-full
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300  # seconds

  mysql:
    host: ${MYSQL_HOST:localhost}
    port: ${MYSQL_PORT:3306}
    database: ${MYSQL_DATABASE:schemaregistry}
    user: ${MYSQL_USER:schemaregistry}
    password: ${MYSQL_PASSWORD}
    tls: "preferred"  # true, false, skip-verify, preferred
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300

  cassandra:
    hosts:
      - ${CASSANDRA_HOST:localhost}
    keyspace: ${CASSANDRA_KEYSPACE:schemaregistry}
    consistency: LOCAL_QUORUM          # Default consistency (if read/write not specified)
    read_consistency: LOCAL_ONE        # Read consistency (e.g., LOCAL_ONE for low latency)
    write_consistency: LOCAL_QUORUM    # Write consistency (e.g., LOCAL_QUORUM for durability)
    username: ${CASSANDRA_USERNAME}
    password: ${CASSANDRA_PASSWORD}

  vault:
    address: ${VAULT_ADDR:http://localhost:8200}
    token: ${VAULT_TOKEN}
    namespace: ${VAULT_NAMESPACE}
    mount_path: secret
    base_path: schema-registry

# Default compatibility level
compatibility:
  default_level: BACKWARD  # NONE, BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE

# Logging
logging:
  level: info  # debug, info, warn, error
  format: json  # json, text

# Security settings
security:
  # TLS configuration
  tls:
    enabled: false
    cert_file: /etc/axonops-schema-registry/certs/server.crt
    key_file: /etc/axonops-schema-registry/certs/server.key
    ca_file: /etc/axonops-schema-registry/certs/ca.crt
    min_version: "TLS1.2"
    client_auth: none  # none, request, require, verify
    auto_reload: true

  # Authentication
  auth:
    enabled: true
    methods:
      - api_key
      - basic
      - oidc

    # Bootstrap initial admin user
    bootstrap:
      enabled: true
      username: admin
      password: ${ADMIN_PASSWORD}
      email: admin@example.com

    # Basic authentication
    basic:
      realm: "Schema Registry"

    # API Key authentication
    api_key:
      header: "X-API-Key"
      query_param: "api_key"
      storage_type: database
      secret: ${API_KEY_SECRET}  # HMAC secret for key hashing
      key_prefix: "sr_"
      cache_refresh_seconds: 60

    # LDAP authentication
    ldap:
      enabled: false
      url: ldaps://ldap.example.com:636
      bind_dn: cn=service,dc=example,dc=com
      bind_password: ${LDAP_BIND_PASSWORD}
      base_dn: dc=example,dc=com
      user_search_base: ou=Users,dc=example,dc=com
      user_search_filter: "(sAMAccountName=%s)"
      username_attribute: sAMAccountName
      email_attribute: mail
      group_search_base: ou=Groups,dc=example,dc=com
      group_search_filter: "(member=%s)"
      group_attribute: memberOf
      role_mapping:
        "cn=SchemaRegistryAdmins,ou=Groups,dc=example,dc=com": admin
        "cn=Developers,ou=Groups,dc=example,dc=com": developer
      default_role: readonly
      start_tls: false
      insecure_skip_verify: false
      ca_cert_file: ""
      connection_timeout: 10
      request_timeout: 30

    # OIDC authentication
    oidc:
      enabled: false
      issuer_url: https://auth.example.com/realms/my-realm
      client_id: schema-registry
      client_secret: ${OIDC_CLIENT_SECRET}
      username_claim: preferred_username
      roles_claim: groups
      role_mapping:
        "/schema-registry-admins": admin
        "/developers": developer
        "/readonly-users": readonly
      default_role: readonly
      required_audience: schema-registry
      allowed_algorithms:
        - RS256
        - ES256

    # RBAC settings
    rbac:
      enabled: true
      default_role: readonly
      super_admins:
        - admin

  # Rate limiting
  rate_limiting:
    enabled: true
    requests_per_second: 100
    burst_size: 200
    per_client: true
    per_endpoint: false

  # Audit logging
  audit:
    enabled: true
    log_file: /var/log/axonops-schema-registry/audit.log
    events:
      - schema_register
      - schema_delete
      - config_change
      - user_login
      - api_key_create
    include_body: false
```

### Environment Variables

All configuration values support environment variable substitution using `${VAR_NAME}` or `${VAR_NAME:default}` syntax.

Common environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `STORAGE_TYPE` | Storage backend type | `memory` |
| `POSTGRES_HOST` | PostgreSQL host | `localhost` |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `POSTGRES_DATABASE` | PostgreSQL database | `schemaregistry` |
| `POSTGRES_USER` | PostgreSQL user | - |
| `POSTGRES_PASSWORD` | PostgreSQL password | - |
| `MYSQL_HOST` | MySQL host | `localhost` |
| `MYSQL_PORT` | MySQL port | `3306` |
| `MYSQL_DATABASE` | MySQL database | `schemaregistry` |
| `MYSQL_USER` | MySQL user | - |
| `MYSQL_PASSWORD` | MySQL password | - |
| `CASSANDRA_HOST` | Cassandra host(s) | `localhost` |
| `CASSANDRA_KEYSPACE` | Cassandra keyspace | `schemaregistry` |
| `VAULT_ADDR` | HashiCorp Vault address | - |
| `VAULT_TOKEN` | Vault token | - |
| `ADMIN_PASSWORD` | Bootstrap admin password | - |
| `API_KEY_SECRET` | Secret for API key hashing | - |
| `LDAP_BIND_PASSWORD` | LDAP service account password | - |
| `OIDC_CLIENT_SECRET` | OIDC client secret | - |

## Authentication

### Basic Authentication

```bash
# Using username and password
curl -u admin:password http://localhost:8081/subjects

# Create API key for programmatic access
curl -u admin:password -X POST http://localhost:8081/admin/apikeys \
  -H "Content-Type: application/json" \
  -d '{"name": "my-app", "role": "developer", "expires_in": 2592000}'
```

### API Key Authentication

```bash
# Confluent-compatible format (API key as username, any value as password)
curl -u "sr_live_abc123...:x" http://localhost:8081/subjects

# Using X-API-Key header
curl -H "X-API-Key: sr_live_abc123..." http://localhost:8081/subjects

# Using query parameter
curl "http://localhost:8081/subjects?api_key=sr_live_abc123..."
```

**Security Notes**: API keys are bearer tokens—treat them like passwords. Keys are scoped to a role, support expiration, and all usage is logged when audit is enabled. Rotate keys regularly via the Admin CLI.

### OIDC Authentication

```bash
# Get token from OIDC provider
TOKEN=$(curl -s -X POST https://auth.example.com/realms/my-realm/protocol/openid-connect/token \
  -d "grant_type=password" \
  -d "client_id=schema-registry" \
  -d "client_secret=secret" \
  -d "username=user" \
  -d "password=pass" | jq -r '.access_token')

# Use token with Schema Registry
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/subjects
```

### Roles and Permissions

| Role | Permissions |
|------|-------------|
| `super_admin` | Full access to all operations including user management |
| `admin` | Schema CRUD, config management, mode management |
| `developer` | Register schemas, read schemas/config/modes |
| `readonly` | Read-only access to schemas and config |

## API Reference

### Schema Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/schemas/types` | List supported schema types |
| GET | `/schemas/ids/{id}` | Get schema by global ID |
| GET | `/schemas/ids/{id}/schema` | Get raw schema string by ID |
| GET | `/schemas/ids/{id}/subjects` | Get subjects using schema ID |
| GET | `/subjects` | List all subjects |
| GET | `/subjects/{subject}/versions` | List versions for subject |
| GET | `/subjects/{subject}/versions/{version}` | Get specific version |
| POST | `/subjects/{subject}/versions` | Register new schema |
| POST | `/subjects/{subject}` | Check if schema exists |
| DELETE | `/subjects/{subject}` | Delete subject |
| DELETE | `/subjects/{subject}/versions/{version}` | Delete version |

### Compatibility

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/compatibility/subjects/{subject}/versions/{version}` | Test compatibility with version |
| POST | `/compatibility/subjects/{subject}/versions` | Test compatibility with all versions |

### Configuration

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/config` | Get global compatibility config |
| PUT | `/config` | Set global compatibility config |
| GET | `/config/{subject}` | Get subject-level config |
| PUT | `/config/{subject}` | Set subject-level config |
| DELETE | `/config/{subject}` | Delete subject-level config |

### Mode

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/mode` | Get global mode (READWRITE, READONLY, etc.) |
| PUT | `/mode` | Set global mode |
| GET | `/mode/{subject}` | Get subject-level mode |
| PUT | `/mode/{subject}` | Set subject-level mode |

### Admin API (User Management)

Requires `super_admin` or `admin` role.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/users` | List all users |
| POST | `/admin/users` | Create a new user |
| GET | `/admin/users/{id}` | Get user by ID |
| PUT | `/admin/users/{id}` | Update user |
| DELETE | `/admin/users/{id}` | Delete user |

**Create User Request:**
```json
{
  "username": "developer1",
  "password": "secure-password",
  "email": "developer1@example.com",
  "role": "developer",
  "enabled": true
}
```

### Admin API (API Key Management)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/apikeys` | List all API keys |
| POST | `/admin/apikeys` | Create a new API key |
| GET | `/admin/apikeys/{id}` | Get API key by ID |
| PUT | `/admin/apikeys/{id}` | Update API key |
| DELETE | `/admin/apikeys/{id}` | Delete API key |
| POST | `/admin/apikeys/{id}/revoke` | Revoke (disable) API key |
| POST | `/admin/apikeys/{id}/rotate` | Rotate API key (create new, revoke old) |

**Create API Key Request:**
```json
{
  "name": "ci-pipeline",
  "role": "developer",
  "expires_in": 2592000
}
```

**Create API Key Response:**
```json
{
  "id": 123,
  "key": "sr_live_abc123...",
  "key_prefix": "sr_live_abc",
  "name": "ci-pipeline",
  "role": "developer",
  "username": "admin",
  "expires_at": "2025-01-15T10:30:00Z"
}
```

### Admin API (Roles)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/roles` | List available roles with permissions |

### Account Self-Service

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/me` | Get current authenticated user info |
| POST | `/me/password` | Change own password |

## Admin CLI

The `schema-registry-admin` CLI provides user and API key management.

### User Management

```bash
# List users
schema-registry-admin user list -u admin -p password

# Create user
schema-registry-admin user create \
  --name developer1 \
  --pass 'secure-password' \
  --role developer \
  --email developer1@example.com \
  -u admin -p password

# Update user
schema-registry-admin user update 123 \
  --role admin \
  -u admin -p password

# Delete user
schema-registry-admin user delete 123 -u admin -p password
```

### API Key Management

```bash
# List API keys
schema-registry-admin apikey list -u admin -p password

# Create API key
schema-registry-admin apikey create \
  --name "ci-pipeline" \
  --role developer \
  --expires-in 8760h \
  -u admin -p password

# Rotate API key
schema-registry-admin apikey rotate 456 \
  --expires-in 8760h \
  -u admin -p password

# Revoke API key
schema-registry-admin apikey revoke 456 -u admin -p password
```

### Bootstrap Initial Admin

```bash
# Initialize database with admin user
schema-registry-admin init \
  --storage-type postgresql \
  --pg-host localhost \
  --pg-database schemaregistry \
  --pg-user postgres \
  --pg-password secret \
  --admin-username admin \
  --admin-password 'secure-admin-password'
```

## Monitoring

### Health Check

```bash
curl http://localhost:8081/
# Returns: {"status": "healthy"}
```

### Prometheus Metrics

Metrics are exposed at `/metrics` in Prometheus format:

```bash
curl http://localhost:8081/metrics
```

Key metrics:

| Metric | Description |
|--------|-------------|
| `schema_registry_http_requests_total` | Total HTTP requests |
| `schema_registry_http_request_duration_seconds` | Request latency histogram |
| `schema_registry_schemas_total` | Total registered schemas |
| `schema_registry_subjects_total` | Total subjects |
| `schema_registry_compatibility_checks_total` | Compatibility check count |
| `schema_registry_storage_operations_total` | Storage operation count |
| `schema_registry_auth_attempts_total` | Authentication attempts |
| `schema_registry_rate_limit_hits_total` | Rate limit hits |

### Logging

Logs are output to stdout in JSON format:

```json
{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"schema registered","subject":"users-value","version":1,"id":42}
```

Set log level via configuration or environment:

```bash
SCHEMA_REGISTRY_LOG_LEVEL=debug ./schema-registry
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## Support

- **GitHub Issues**: [Report bugs or request features](https://github.com/axonops/axonops-schema-registry/issues)
- **Documentation**: [Full documentation](https://docs.axonops.com/schema-registry)
