# Getting Started

This guide walks you through running AxonOps Schema Registry, registering your first schemas, and verifying compatibility. You should have a working registry within five minutes.

## Contents

- [Prerequisites](#prerequisites)
- [Quick Start with Docker](#quick-start-with-docker)
- [Quick Start with Binary](#quick-start-with-binary)
  - [Build from Source](#build-from-source)
  - [Run](#run)
- [Your First API Calls](#your-first-api-calls)
  - [Check Health](#check-health)
  - [Register an Avro Schema](#register-an-avro-schema)
  - [Retrieve the Schema](#retrieve-the-schema)
  - [List Subjects](#list-subjects)
  - [Register a Second Version](#register-a-second-version)
  - [Check Compatibility](#check-compatibility)
  - [Register a JSON Schema](#register-a-json-schema)
  - [Register a Protobuf Schema](#register-a-protobuf-schema)
  - [View All Subjects](#view-all-subjects)
  - [Get Supported Schema Types](#get-supported-schema-types)
- [Using with Kafka Clients](#using-with-kafka-clients)
  - [Java (Confluent Kafka Client)](#java-confluent-kafka-client)
  - [Go (confluent-kafka-go)](#go-confluent-kafka-go)
  - [Python (confluent-kafka-python)](#python-confluent-kafka-python)
- [Configuration](#configuration)
  - [Change the Default Compatibility Level](#change-the-default-compatibility-level)
- [Next Steps](#next-steps)

## Prerequisites

You need one of the following:

- **Docker** (recommended for quick start) -- any recent version
- **Go 1.24+** if building from source

No database is required for initial evaluation. The in-memory storage backend is enabled by default.

## Quick Start with Docker

Pull and run the registry with a single command:

```bash
docker run -d --name schema-registry \
  -p 8081:8081 \
  ghcr.io/axonops/axonops-schema-registry:latest
```

The registry starts on port 8081 using in-memory storage. Verify it is running:

```bash
curl http://localhost:8081/
```

Expected response:

```json
{}
```

An empty JSON object indicates the registry is healthy and ready to accept requests.

## Quick Start with Binary

### Build from Source

Clone the repository and build:

```bash
git clone https://github.com/axonops/axonops-schema-registry.git
cd axonops-schema-registry
make build
```

The binary is placed at `./build/schema-registry`.

### Run

Start the registry with the default in-memory backend:

```bash
./build/schema-registry
```

Or with a configuration file:

```bash
./build/schema-registry -config config.yaml
```

A minimal configuration file for quick evaluation:

```yaml
server:
  host: "0.0.0.0"
  port: 8081
storage:
  type: memory
compatibility:
  default_level: BACKWARD
```

## Your First API Calls

All examples below use `curl`. The registry accepts both `application/json` and `application/vnd.schemaregistry.v1+json` content types. Responses always use `application/vnd.schemaregistry.v1+json`.

### Check Health

```bash
curl http://localhost:8081/
```

```json
{}
```

### Register an Avro Schema

Register a `User` Avro schema under the subject `users-value`:

```bash
curl -X POST http://localhost:8081/subjects/users-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}"
  }'
```

Response:

```json
{"id": 1}
```

The registry assigned global schema ID `1`. The `schemaType` field defaults to `AVRO` when omitted.

### Retrieve the Schema

Fetch the schema by its global ID:

```bash
curl http://localhost:8081/schemas/ids/1
```

```json
{
  "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}",
  "schemaType": "AVRO"
}
```

Fetch by subject and version:

```bash
curl http://localhost:8081/subjects/users-value/versions/1
```

```json
{
  "subject": "users-value",
  "id": 1,
  "version": 1,
  "schemaType": "AVRO",
  "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"}]}"
}
```

### List Subjects

```bash
curl http://localhost:8081/subjects
```

```json
["users-value"]
```

### Register a Second Version

Add an `age` field with a default value. Under `BACKWARD` compatibility (the default), new fields must have defaults so that consumers using the old schema can still read new data:

```bash
curl -X POST http://localhost:8081/subjects/users-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\",\"default\":0}]}"
  }'
```

```json
{"id": 2}
```

Verify both versions exist:

```bash
curl http://localhost:8081/subjects/users-value/versions
```

```json
[1, 2]
```

### Check Compatibility

Before registering a schema, you can test whether it is compatible with existing versions. This checks a proposed schema against the latest version of `users-value`:

```bash
curl -X POST http://localhost:8081/compatibility/subjects/users-value/versions/latest \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schema": "{\"type\":\"record\",\"name\":\"User\",\"namespace\":\"com.example\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"email\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\",\"default\":0},{\"name\":\"phone\",\"type\":[\"null\",\"string\"],\"default\":null}]}"
  }'
```

```json
{"is_compatible": true}
```

An incompatible change (for example, removing a field without a default) returns `{"is_compatible": false}`.

### Register a JSON Schema

Register a JSON Schema by setting `schemaType` to `JSON`:

```bash
curl -X POST http://localhost:8081/subjects/orders-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "JSON",
    "schema": "{\"type\":\"object\",\"properties\":{\"order_id\":{\"type\":\"string\"},\"amount\":{\"type\":\"number\"},\"currency\":{\"type\":\"string\"}},\"required\":[\"order_id\",\"amount\"]}"
  }'
```

```json
{"id": 3}
```

### Register a Protobuf Schema

Register a Protobuf schema by setting `schemaType` to `PROTOBUF`:

```bash
curl -X POST http://localhost:8081/subjects/events-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{
    "schemaType": "PROTOBUF",
    "schema": "syntax = \"proto3\";\npackage com.example;\n\nmessage Event {\n  string event_id = 1;\n  string event_type = 2;\n  int64 timestamp = 3;\n}"
  }'
```

```json
{"id": 4}
```

### View All Subjects

```bash
curl http://localhost:8081/subjects
```

```json
["events-value", "orders-value", "users-value"]
```

### Get Supported Schema Types

```bash
curl http://localhost:8081/schemas/types
```

```json
["AVRO", "JSON", "PROTOBUF"]
```

## Using with Kafka Clients

AxonOps Schema Registry is wire-compatible with the Confluent Schema Registry API. Existing Kafka serializers and deserializers work without modification -- point them at your AxonOps Schema Registry URL instead of Confluent's.

### Java (Confluent Kafka Client)

```java
Properties props = new Properties();
props.put("bootstrap.servers", "localhost:9092");
props.put("key.serializer", "org.apache.kafka.common.serialization.StringSerializer");
props.put("value.serializer", "io.confluent.kafka.serializers.KafkaAvroSerializer");
props.put("schema.registry.url", "http://localhost:8081");

KafkaProducer<String, GenericRecord> producer = new KafkaProducer<>(props);
```

### Go (confluent-kafka-go)

```go
producer, err := kafka.NewProducer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
})

serializer, err := avro.NewSerializer(
    schemaregistry.NewClient(schemaregistry.NewConfig("http://localhost:8081")),
    serde.ValueSerde,
    avro.NewSerializerConfig(),
)
```

### Python (confluent-kafka-python)

```python
from confluent_kafka import SerializingProducer
from confluent_kafka.schema_registry import SchemaRegistryClient
from confluent_kafka.schema_registry.avro import AvroSerializer

schema_registry_client = SchemaRegistryClient({"url": "http://localhost:8081"})
avro_serializer = AvroSerializer(schema_registry_client, schema_str)

producer = SerializingProducer({
    "bootstrap.servers": "localhost:9092",
    "value.serializer": avro_serializer,
})
```

## Configuration

The registry uses sensible defaults. The two most common things to configure are the storage backend and the default compatibility level.

### Change the Default Compatibility Level

The global compatibility level defaults to `BACKWARD`. To change it at runtime:

```bash
curl -X PUT http://localhost:8081/config \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "FULL"}'
```

Available levels: `NONE`, `BACKWARD`, `BACKWARD_TRANSITIVE`, `FORWARD`, `FORWARD_TRANSITIVE`, `FULL`, `FULL_TRANSITIVE`.

You can also set compatibility per subject:

```bash
curl -X PUT http://localhost:8081/config/users-value \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"compatibility": "FULL_TRANSITIVE"}'
```

## Next Steps

- [Best Practices](best-practices.md) -- schema design patterns, naming conventions, evolution strategies, and common mistakes
- [Installation](installation.md) -- production deployment options, systemd units, and platform-specific packages
- [Configuration](configuration.md) -- full YAML configuration reference covering server, storage, security, and logging options
- [Storage Backends](storage-backends.md) -- choosing between PostgreSQL, MySQL, Cassandra, and in-memory storage
- [Authentication](authentication.md) -- API keys, LDAP, OIDC, JWT, and role-based access control
- [API Reference](api-reference.md) -- complete documentation for all API endpoints, request/response formats, and error codes
