# Deployment

## Overview

AxonOps Schema Registry is a stateless binary. All state -- schemas, subjects, configuration, users, and API keys -- is stored in the database. This means multiple instances can be deployed behind a load balancer with no coordination between them. There is no leader election, no peer discovery, and no inter-instance communication.

This guide covers deployment topologies, container-based deployments, Kubernetes manifests, systemd configuration, and operational considerations for production environments.

---

## Deployment Topologies

### Single Instance

For development, testing, or low-traffic production environments.

![Single Instance](../assets/architecture-single.svg)

A single registry instance connects directly to a storage backend. This is the simplest topology to operate and is suitable for workloads where high availability is not required.

Characteristics:

- Single binary with approximately 50 MB memory footprint
- Any storage backend (in-memory, PostgreSQL, MySQL, Cassandra)
- No external dependencies beyond the database
- Restart recovers all state from the database

### High Availability (PostgreSQL/MySQL)

For production environments requiring fault tolerance and horizontal scaling.

**Write path:**

![HA Write Path](../assets/architecture-ha-write.svg)

**Read path:**

![HA Read Path](../assets/architecture-ha-read.svg)

Key characteristics:

- **Stateless instances** -- any instance can handle any request. There is no session affinity requirement.
- **No leader election or coordination** -- instances are unaware of each other.
- **Horizontal scaling** -- add instances as needed. Performance scales linearly with instance count for read-heavy workloads.
- **API key caching** -- each instance caches API keys in memory and refreshes them periodically (configurable via `security.auth.api_key.cache_refresh_seconds`, default 60 seconds). This means a key created on one instance may take up to one refresh interval to be recognized by others.
- **Database-level concurrency control:**
  - PostgreSQL uses transactions with row-level locking to prevent race conditions during schema registration and ID allocation.
  - MySQL uses `SELECT ... FOR UPDATE` within transactions for conflict-free ID allocation.
- **Idempotent registration** -- registering the same schema content under the same subject always returns the same ID, regardless of which instance handles the request. Fingerprint-based deduplication ensures this.

Requirements:

- Load balancer (HAProxy, Nginx, AWS ALB, Kubernetes Ingress, or equivalent)
- 2 or more Schema Registry instances
- PostgreSQL or MySQL with replication configured for database-level HA
- Optional: HashiCorp Vault for centralized authentication storage (see [Configuration](configuration.md#hashicorp-vault-auth-storage))

### Distributed Multi-Datacenter (Cassandra)

For global deployments with the highest availability requirements.

![Distributed Architecture](../assets/architecture-distributed.svg)

Key characteristics:

- **Active-active** -- all datacenters serve both reads and writes simultaneously. There is no primary or standby datacenter.
- **Automatic cross-DC replication** -- Cassandra handles data replication between datacenters transparently.
- **Datacenter failure tolerance** -- the registry continues operating if an entire datacenter goes offline, provided sufficient replicas exist in the remaining datacenters.
- **Lightweight Transactions (LWT)** -- used for atomic ID allocation and fingerprint-based deduplication to prevent conflicts across concurrent writers in different datacenters.
- **SAI indexes** -- Storage Attached Indexes (Cassandra 5.0+) enable efficient secondary lookups without maintaining separate denormalized tables.

Consistency settings:

| Operation | Recommended Level | Rationale |
|-----------|-------------------|-----------|
| Write | `LOCAL_QUORUM` | Ensures durability within the local datacenter before acknowledging |
| Read (low latency) | `LOCAL_ONE` | Single local replica read; suitable for schema lookups |
| Read (strong consistency) | `LOCAL_QUORUM` | Read-your-writes guarantee; use when immediate consistency matters |
| Version assignment | LWT (`IF NOT EXISTS`) | Atomic across the cluster regardless of consistency level |

Configure read and write consistency independently:

```yaml
storage:
  type: cassandra
  cassandra:
    hosts:
      - cassandra-dc1-node1
      - cassandra-dc1-node2
    keyspace: schema_registry
    consistency: LOCAL_QUORUM
    read_consistency: LOCAL_ONE
    write_consistency: LOCAL_QUORUM
```

Requirements:

- Cassandra 5.0+ cluster with `NetworkTopologyStrategy` replication
- Schema Registry instances deployed in each datacenter
- Local load balancer per datacenter (clients connect to their local datacenter's load balancer)
- Optional: DNS-based global load balancing (Route 53, Cloudflare, or equivalent) for automatic failover

For production multi-datacenter deployments, pre-create the keyspace manually before starting the registry:

```cql
CREATE KEYSPACE schema_registry
  WITH REPLICATION = {
    'class': 'NetworkTopologyStrategy',
    'dc1': 3,
    'dc2': 3
  };
```

See [Storage Backends](storage-backends.md#cassandra) for full Cassandra configuration details.

---

## Docker Deployment

### Simple Docker Run

Run the registry with PostgreSQL as the storage backend:

```bash
docker run -d \
  --name schema-registry \
  -p 8081:8081 \
  -e SCHEMA_REGISTRY_STORAGE_TYPE=postgresql \
  -e SCHEMA_REGISTRY_PG_HOST=postgres \
  -e SCHEMA_REGISTRY_PG_DATABASE=schemaregistry \
  -e SCHEMA_REGISTRY_PG_USER=schemaregistry \
  -e SCHEMA_REGISTRY_PG_PASSWORD=secret \
  ghcr.io/axonops/axonops-schema-registry:latest
```

For evaluation with in-memory storage (no database required):

```bash
docker run -d \
  --name schema-registry \
  -p 8081:8081 \
  ghcr.io/axonops/axonops-schema-registry:latest
```

### Docker Compose with PostgreSQL

Create a `docker-compose.yaml` file:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: schemaregistry
      POSTGRES_USER: schemaregistry
      POSTGRES_PASSWORD: schemaregistry
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U schemaregistry"]
      interval: 5s
      timeout: 5s
      retries: 5

  schema-registry:
    image: ghcr.io/axonops/axonops-schema-registry:latest
    ports:
      - "8081:8081"
    environment:
      SCHEMA_REGISTRY_STORAGE_TYPE: postgresql
      SCHEMA_REGISTRY_PG_HOST: postgres
      SCHEMA_REGISTRY_PG_PORT: "5432"
      SCHEMA_REGISTRY_PG_DATABASE: schemaregistry
      SCHEMA_REGISTRY_PG_USER: schemaregistry
      SCHEMA_REGISTRY_PG_PASSWORD: schemaregistry
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8081/"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 5s

volumes:
  pgdata:
```

Start the stack:

```bash
docker compose up -d
```

Verify the registry is healthy:

```bash
curl http://localhost:8081/
```

---

## Kubernetes Deployment

The following manifests deploy a production-ready Schema Registry cluster on Kubernetes with PostgreSQL as the storage backend.

### Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: schema-registry
```

### Secret

Store database credentials in a Kubernetes Secret. In production, use an external secrets manager (Vault, AWS Secrets Manager, or Kubernetes External Secrets Operator) instead of inline `stringData`.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: schema-registry-db
  namespace: schema-registry
type: Opaque
stringData:
  postgres-user: schemaregistry
  postgres-password: changeme
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: schema-registry-config
  namespace: schema-registry
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: 8081
    storage:
      type: postgresql
      postgresql:
        host: ${POSTGRES_HOST}
        port: ${POSTGRES_PORT:5432}
        database: ${POSTGRES_DATABASE:schemaregistry}
        user: ${POSTGRES_USER}
        password: ${POSTGRES_PASSWORD}
        ssl_mode: prefer
        max_open_conns: 25
        max_idle_conns: 5
    compatibility:
      default_level: BACKWARD
    logging:
      level: info
      format: json
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: schema-registry
  namespace: schema-registry
  labels:
    app: schema-registry
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
          args:
            - --config
            - /etc/axonops-schema-registry/config.yaml
          ports:
            - name: http
              containerPort: 8081
              protocol: TCP
          env:
            - name: POSTGRES_HOST
              value: postgres-service
            - name: POSTGRES_DATABASE
              value: schemaregistry
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: schema-registry-db
                  key: postgres-user
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: schema-registry-db
                  key: postgres-password
          volumeMounts:
            - name: config
              mountPath: /etc/axonops-schema-registry
              readOnly: true
          livenessProbe:
            httpGet:
              path: /
              port: 8081
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 3
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /
              port: 8081
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 2
          resources:
            requests:
              cpu: 250m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
      volumes:
        - name: config
          configMap:
            name: schema-registry-config
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: schema-registry
  namespace: schema-registry
  labels:
    app: schema-registry
spec:
  type: ClusterIP
  selector:
    app: schema-registry
  ports:
    - name: http
      port: 8081
      targetPort: 8081
      protocol: TCP
```

### HorizontalPodAutoscaler (Optional)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: schema-registry
  namespace: schema-registry
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: schema-registry
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### Ingress (Optional)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: schema-registry
  namespace: schema-registry
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
spec:
  ingressClassName: nginx
  rules:
    - host: schema-registry.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: schema-registry
                port:
                  number: 8081
```

### Apply All Manifests

Save all resources to a single file or apply individually:

```bash
kubectl apply -f namespace.yaml
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

Verify the deployment:

```bash
kubectl -n schema-registry get pods
kubectl -n schema-registry logs -l app=schema-registry --tail=20
kubectl -n schema-registry port-forward svc/schema-registry 8081:8081
curl http://localhost:8081/
```

---

## Systemd Service

For bare-metal or VM deployments using package installation (APT or YUM), the systemd service is installed automatically. For binary installations, create the unit file manually.

### Service File

Create `/etc/systemd/system/axonops-schema-registry.service`:

```ini
[Unit]
Description=AxonOps Schema Registry
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=schemaregistry
Group=schemaregistry
ExecStart=/usr/local/bin/schema-registry --config /etc/axonops-schema-registry/config.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/axonops-schema-registry
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

### Setup

```bash
# Create a dedicated service user
sudo useradd --system --no-create-home --shell /usr/sbin/nologin schemaregistry

# Create configuration directory
sudo mkdir -p /etc/axonops-schema-registry
sudo cp config.example.yaml /etc/axonops-schema-registry/config.yaml

# Set file permissions -- config may contain database credentials
sudo chown root:schemaregistry /etc/axonops-schema-registry/config.yaml
sudo chmod 640 /etc/axonops-schema-registry/config.yaml

# Create log directory (if using file-based audit logging)
sudo mkdir -p /var/log/axonops-schema-registry
sudo chown schemaregistry:schemaregistry /var/log/axonops-schema-registry

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable axonops-schema-registry
sudo systemctl start axonops-schema-registry
```

### Verify

```bash
sudo systemctl status axonops-schema-registry
sudo journalctl -u axonops-schema-registry -f
curl http://localhost:8081/
```

---

## Health Checks

The health check endpoint is `GET /`, which returns an empty JSON object with HTTP 200 when the server is ready to accept requests:

```bash
curl http://localhost:8081/
```

```json
{}
```

The health check verifies database connectivity. A non-200 response indicates the server cannot reach its storage backend.

Use this endpoint for:

- **Load balancer health checks** -- configure your load balancer (HAProxy, Nginx, ALB) to poll `GET /` and remove unhealthy instances from rotation.
- **Kubernetes probes** -- use as both liveness and readiness probe targets (see the Kubernetes Deployment manifest above).
- **Docker HEALTHCHECK** -- use `wget --spider -q http://localhost:8081/` or `curl -f http://localhost:8081/` in Docker health check definitions.
- **Monitoring alerts** -- poll from your monitoring system and alert on non-200 responses.

---

## Graceful Shutdown

The registry handles `SIGTERM` and `SIGINT` signals for graceful shutdown:

1. On receiving the signal, the server stops accepting new connections.
2. In-flight requests are allowed to complete within the configured timeout period.
3. The server read and write timeouts (default 30 seconds each, configurable via `server.read_timeout` and `server.write_timeout`) control the drain period.
4. Database connections are closed cleanly.
5. Background services (API key cache refresh, auth service) are stopped.

No requests are dropped during normal shutdown. Kubernetes sends `SIGTERM` by default when terminating pods, and the default `terminationGracePeriodSeconds` (30 seconds) aligns with the server's default timeouts.

---

## Resource Requirements

| Deployment | CPU | Memory | Notes |
|-----------|-----|--------|-------|
| Development | 100m | 64Mi | In-memory backend, minimal load |
| Production (small) | 250m | 128Mi | Fewer than 100 schemas, low request rate |
| Production (medium) | 500m | 256Mi | 100--1000 schemas, moderate request rate |
| Production (large) | 1000m | 512Mi | 1000+ schemas, high request rate or large schemas |

These values are starting points. Actual requirements depend on schema sizes, request patterns, and whether authentication and rate limiting are enabled. Monitor memory and CPU usage after deployment and adjust accordingly.

The registry binary is statically compiled with no runtime dependencies beyond the OS. Memory usage is dominated by the in-memory API key cache (when authentication is enabled) and any in-flight request processing. Schema content is stored in the database, not in application memory.

---

## TLS Termination

Two approaches are available:

### Option 1: TLS at the Load Balancer (Recommended)

Terminate TLS at the load balancer (Nginx, HAProxy, AWS ALB, Kubernetes Ingress) and forward plain HTTP to the registry instances. This is the simpler approach and centralizes certificate management.

```
Client --[HTTPS]--> Load Balancer --[HTTP]--> Schema Registry
```

No TLS configuration is needed on the registry itself.

### Option 2: TLS at the Registry

Enable TLS directly on the registry by setting `security.tls.enabled: true` in the configuration file:

```yaml
security:
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/registry.pem
    key_file: /etc/ssl/private/registry-key.pem
    min_version: "TLS1.2"
    auto_reload: true
```

When `auto_reload` is enabled, the registry watches the certificate and key files for changes and reloads them without requiring a restart. This is useful with automated certificate rotation (for example, cert-manager in Kubernetes or ACME-based renewal).

For mutual TLS (mTLS), configure `client_auth` and provide a CA certificate:

```yaml
security:
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/registry.pem
    key_file: /etc/ssl/private/registry-key.pem
    ca_file: /etc/ssl/certs/client-ca.pem
    client_auth: verify
    min_version: "TLS1.2"
```

See [Configuration](configuration.md#tls) for all TLS options.

---

## Network Requirements

| Port | Protocol | Purpose |
|------|----------|---------|
| 8081 (configurable) | TCP | HTTP/HTTPS API endpoint |
| 5432 | TCP | PostgreSQL (if used) |
| 3306 | TCP | MySQL (if used) |
| 9042 | TCP | Cassandra (if used) |
| 8200 | TCP | HashiCorp Vault (if used for auth storage) |

There is no inter-instance communication. Each registry instance connects only to:

- Its configured storage backend
- Optionally, a Vault server for authentication storage
- Optionally, an LDAP server or OIDC provider for authentication

Firewall rules need only allow inbound traffic to the registry port and outbound traffic to the database and any configured authentication providers.

---

## Related Documentation

- [Installation](installation.md) -- all installation methods (Docker, APT, YUM, binary, source)
- [Configuration](configuration.md) -- full YAML configuration reference
- [Storage Backends](storage-backends.md) -- backend comparison, setup, and migration
- [Authentication](authentication.md) -- API keys, LDAP, OIDC, JWT, and RBAC
