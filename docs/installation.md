# Installation

This guide covers all supported methods for installing AxonOps Schema Registry. Choose the method that best fits your environment:

## Contents

- [Prerequisites](#prerequisites)
- [Docker](#docker)
  - [Available tags](#available-tags)
  - [Pull the image](#pull-the-image)
  - [Run with in-memory storage](#run-with-in-memory-storage)
  - [Run with environment variables](#run-with-environment-variables)
  - [Run with a config file](#run-with-a-config-file)
  - [Docker Compose with PostgreSQL](#docker-compose-with-postgresql)
- [Debian/Ubuntu (APT)](#debianubuntu-apt)
  - [Add the AxonOps repository](#add-the-axonops-repository)
  - [Install the package](#install-the-package)
  - [Configure](#configure)
  - [Enable and start the service](#enable-and-start-the-service)
  - [Check the service status](#check-the-service-status)
- [RHEL/CentOS/Fedora (YUM)](#rhelcentosfedora-yum)
  - [Add the AxonOps repository](#add-the-axonops-repository-1)
  - [Install the package](#install-the-package-1)
  - [Configure](#configure-1)
  - [Enable and start the service](#enable-and-start-the-service-1)
  - [Check the service status](#check-the-service-status-1)
- [Binary Installation](#binary-installation)
  - [Supported platforms](#supported-platforms)
  - [Download and install](#download-and-install)
  - [Create the configuration directory](#create-the-configuration-directory)
  - [Run the server](#run-the-server)
  - [Optional: create a systemd service](#optional-create-a-systemd-service)
- [Kubernetes](#kubernetes)
  - [Namespace and Secret](#namespace-and-secret)
  - [ConfigMap](#configmap)
  - [Deployment](#deployment)
  - [Service](#service)
  - [Apply the manifests](#apply-the-manifests)
  - [Verify the deployment](#verify-the-deployment)
- [Building from Source](#building-from-source)
  - [Requirements](#requirements)
  - [Clone and build](#clone-and-build)
  - [Build with version metadata](#build-with-version-metadata)
  - [Cross-compile for all platforms](#cross-compile-for-all-platforms)
  - [Run tests](#run-tests)
- [Verifying the Installation](#verifying-the-installation)
  - [Health check](#health-check)
  - [Version and metadata](#version-and-metadata)
  - [Register a test schema](#register-a-test-schema)
  - [Retrieve the schema](#retrieve-the-schema)
- [Next Steps](#next-steps)

| Method | Best for |
|--------|----------|
| [Docker](#docker) | Evaluation, development, container-based deployments |
| [Debian/Ubuntu (APT)](#debianubuntu-apt) | Production on Debian-based Linux |
| [RHEL/CentOS/Fedora (YUM)](#rhelcentosfedora-yum) | Production on RHEL-based Linux |
| [Binary](#binary-installation) | Manual installation, custom environments |
| [Kubernetes](#kubernetes) | Orchestrated container deployments |
| [From source](#building-from-source) | Development, custom builds |

## Prerequisites

AxonOps Schema Registry is a single binary with no runtime dependencies beyond a storage backend. For evaluation, the built-in in-memory storage requires no external services. For production, you need one of the following:

- **PostgreSQL** 13+ (recommended for most deployments)
- **MySQL** 8.0+
- **Apache Cassandra** 4.0+ (for multi-datacenter deployments)

The server listens on port **8081** by default.

---

## Docker

Docker images are published to GitHub Container Registry.

### Available tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `1.0.0` | Specific version (semver) |
| `1.0` | Latest patch release for the 1.0.x line |
| `1` | Latest minor and patch release for the 1.x line |

### Pull the image

```bash
docker pull ghcr.io/axonops/axonops-schema-registry:latest
```

### Run with in-memory storage

Suitable for evaluation and testing. Data does not persist across restarts.

```bash
docker run -d \
  --name schema-registry \
  -p 8081:8081 \
  ghcr.io/axonops/axonops-schema-registry:latest
```

### Run with environment variables

```bash
docker run -d \
  --name schema-registry \
  -p 8081:8081 \
  -e STORAGE_TYPE=postgresql \
  -e POSTGRES_HOST=postgres.example.com \
  -e POSTGRES_PORT=5432 \
  -e POSTGRES_DATABASE=schemaregistry \
  -e POSTGRES_USER=schemaregistry \
  -e POSTGRES_PASSWORD=secret \
  ghcr.io/axonops/axonops-schema-registry:latest
```

### Run with a config file

```bash
docker run -d \
  --name schema-registry \
  -p 8081:8081 \
  -v /path/to/config.yaml:/etc/axonops-schema-registry/config.yaml:ro \
  ghcr.io/axonops/axonops-schema-registry:latest \
  --config /etc/axonops-schema-registry/config.yaml
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
      STORAGE_TYPE: postgresql
      POSTGRES_HOST: postgres
      POSTGRES_PORT: "5432"
      POSTGRES_DATABASE: schemaregistry
      POSTGRES_USER: schemaregistry
      POSTGRES_PASSWORD: schemaregistry
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  pgdata:
```

Start the stack:

```bash
docker compose up -d
```

---

## Debian/Ubuntu (APT)

### Add the AxonOps repository

```bash
# Install prerequisites
sudo apt-get update
sudo apt-get install -y curl gnupg ca-certificates

# Add the GPG key
curl -L https://packages.axonops.com/apt/repo-signing-key.gpg \
  | sudo gpg --dearmor -o /usr/share/keyrings/axonops.gpg

# Add the repository
echo "deb [signed-by=/usr/share/keyrings/axonops.gpg] https://packages.axonops.com/apt axonops-apt main" \
  | sudo tee /etc/apt/sources.list.d/axonops-apt.list
```

### Install the package

```bash
sudo apt-get update
sudo apt-get install -y axonops-schema-registry
```

This installs:

- `/usr/bin/schema-registry` -- the server binary
- `/usr/bin/schema-registry-admin` -- the admin CLI
- `/etc/axonops-schema-registry/config.example.yaml` -- example configuration

### Configure

```bash
sudo cp /etc/axonops-schema-registry/config.example.yaml /etc/axonops-schema-registry/config.yaml
sudo editor /etc/axonops-schema-registry/config.yaml
```

At minimum, set the storage backend. See [Configuration](configuration.md) for all options.

### Enable and start the service

```bash
sudo systemctl enable axonops-schema-registry
sudo systemctl start axonops-schema-registry
```

### Check the service status

```bash
sudo systemctl status axonops-schema-registry
sudo journalctl -u axonops-schema-registry -f
```

---

## RHEL/CentOS/Fedora (YUM)

### Add the AxonOps repository

```bash
sudo tee /etc/yum.repos.d/axonops-yum.repo << 'EOF'
[axonops-yum]
name=axonops-yum
baseurl=https://packages.axonops.com/yum/
enabled=1
repo_gpgcheck=0
gpgcheck=0
EOF
```

### Install the package

```bash
sudo yum makecache
sudo yum install -y axonops-schema-registry
```

This installs the same files as the Debian package (server binary, admin CLI, and example configuration).

### Configure

```bash
sudo cp /etc/axonops-schema-registry/config.example.yaml /etc/axonops-schema-registry/config.yaml
sudo editor /etc/axonops-schema-registry/config.yaml
```

See [Configuration](configuration.md) for all available settings.

### Enable and start the service

```bash
sudo systemctl enable axonops-schema-registry
sudo systemctl start axonops-schema-registry
```

### Check the service status

```bash
sudo systemctl status axonops-schema-registry
sudo journalctl -u axonops-schema-registry -f
```

---

## Binary Installation

Pre-built binaries are available for Linux and macOS on the [GitHub Releases](https://github.com/axonops/axonops-schema-registry/releases) page.

### Supported platforms

| OS | Architecture | Binary suffix |
|----|-------------|---------------|
| Linux | x86_64 (amd64) | `linux-amd64` |
| Linux | ARM64 (aarch64) | `linux-arm64` |
| macOS | x86_64 (Intel) | `darwin-amd64` |
| macOS | ARM64 (Apple Silicon) | `darwin-arm64` |

### Download and install

Replace `linux-amd64` with your platform as appropriate.

```bash
# Download the latest release
curl -LO https://github.com/axonops/axonops-schema-registry/releases/latest/download/axonops-schema-registry-linux-amd64.tar.gz

# Extract
tar xzf axonops-schema-registry-linux-amd64.tar.gz

# Install binaries
sudo mv axonops-schema-registry-*/schema-registry /usr/local/bin/
sudo mv axonops-schema-registry-*/schema-registry-admin /usr/local/bin/

# Verify
schema-registry --version
```

The archive contains two binaries:

- `schema-registry` -- the HTTP server
- `schema-registry-admin` -- the admin CLI for managing users and API keys

### Create the configuration directory

```bash
sudo mkdir -p /etc/axonops-schema-registry
sudo cp axonops-schema-registry-*/config.example.yaml /etc/axonops-schema-registry/config.yaml
sudo editor /etc/axonops-schema-registry/config.yaml
```

### Run the server

```bash
# Run with the default config (in-memory storage, port 8081)
schema-registry

# Run with a custom config file
schema-registry --config /etc/axonops-schema-registry/config.yaml
```

### Optional: create a systemd service

If you installed from a binary rather than a package, create a systemd unit file manually:

```bash
sudo tee /etc/systemd/system/axonops-schema-registry.service << 'EOF'
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

[Install]
WantedBy=multi-user.target
EOF

# Create a dedicated service user
sudo useradd --system --no-create-home --shell /usr/sbin/nologin schemaregistry

# Set file permissions
sudo chown root:schemaregistry /etc/axonops-schema-registry/config.yaml
sudo chmod 640 /etc/axonops-schema-registry/config.yaml

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable axonops-schema-registry
sudo systemctl start axonops-schema-registry
```

---

## Kubernetes

### Namespace and Secret

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: schema-registry
---
apiVersion: v1
kind: Secret
metadata:
  name: schema-registry-secrets
  namespace: schema-registry
type: Opaque
stringData:
  postgres-user: schemaregistry
  postgres-password: schemaregistry
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
                  name: schema-registry-secrets
                  key: postgres-user
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: schema-registry-secrets
                  key: postgres-password
          volumeMounts:
            - name: config
              mountPath: /etc/axonops-schema-registry
              readOnly: true
          startupProbe:
            httpGet:
              path: /health/startup
              port: 8081
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 12    # 60s total startup window
          livenessProbe:
            httpGet:
              path: /health/live
              port: 8081
            periodSeconds: 10
            timeoutSeconds: 3
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health/ready
              port: 8081
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 2
          resources:
            requests:
              memory: "64Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "500m"
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

### Apply the manifests

```bash
kubectl apply -f namespace.yaml
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

Or, if all resources are in a single file:

```bash
kubectl apply -f schema-registry-k8s.yaml
```

### Verify the deployment

```bash
kubectl -n schema-registry get pods
kubectl -n schema-registry logs -l app=schema-registry --tail=20
kubectl -n schema-registry port-forward svc/schema-registry 8081:8081
curl http://localhost:8081/
```

---

## Building from Source

### Requirements

- **Go 1.24+** ([download](https://go.dev/dl/))
- **Git**
- **Make** (optional, for convenience targets)

### Clone and build

```bash
git clone https://github.com/axonops/axonops-schema-registry.git
cd axonops-schema-registry
make build
```

The binary is written to `./build/schema-registry`.

### Build with version metadata

The Makefile injects version, commit hash, and build date into the binary via linker flags:

```bash
make build
./build/schema-registry --version
# axonops-schema-registry v1.0.0-3-gabcdef (commit: abcdef, built: 2026-02-16T12:00:00Z)
```

To build manually without Make:

```bash
go build \
  -ldflags "-X main.version=$(git describe --tags --always) -X main.commit=$(git rev-parse --short HEAD) -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o schema-registry ./cmd/schema-registry
```

### Cross-compile for all platforms

```bash
make build-all
```

This produces binaries in `./build/`:

```
build/schema-registry-linux-amd64
build/schema-registry-linux-arm64
build/schema-registry-darwin-amd64
build/schema-registry-darwin-arm64
```

### Run tests

```bash
make test-unit
```

See the [Makefile](https://github.com/axonops/axonops-schema-registry/blob/main/Makefile) for the full set of test targets, including integration, BDD, and compatibility tests.

---

## Verifying the Installation

After starting the server by any method, confirm it is running:

### Health check

```bash
curl http://localhost:8081/
```

A healthy server returns an empty JSON object:

```json
{}
```

### Version and metadata

```bash
curl http://localhost:8081/v1/metadata/version
```

### Register a test schema

```bash
curl -X POST http://localhost:8081/subjects/test-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"schema": "{\"type\": \"record\", \"name\": \"Test\", \"fields\": [{\"name\": \"id\", \"type\": \"int\"}]}"}'
```

A successful response returns the schema ID:

```json
{"id": 1}
```

### Retrieve the schema

```bash
curl http://localhost:8081/subjects/test-value/versions/latest
```

---

## Next Steps

- [Configuration](configuration.md) -- server settings, storage backends, authentication, TLS, rate limiting
- [Storage Backends](storage-backends.md) -- detailed setup for PostgreSQL, MySQL, Cassandra, and in-memory
- [Getting Started](getting-started.md) -- register your first schemas, configure compatibility, and integrate with Kafka clients
