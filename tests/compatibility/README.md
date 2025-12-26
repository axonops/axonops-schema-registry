# Confluent Serializer/Deserializer Compatibility Tests

This directory contains compatibility tests to verify that AxonOps Schema Registry works correctly with Confluent serializers and deserializers across multiple versions and languages.

## Purpose

These tests verify:
1. **Schema Fingerprinting**: Identical schemas produce the same schema ID (deduplication)
2. **Wire Format**: Serialized data follows the Confluent wire format (magic byte + schema ID + payload)
3. **Round-trip Serialization**: Data can be serialized and deserialized correctly
4. **Schema Evolution**: Backward-compatible schema changes work as expected

## Test Matrix

### Java (Confluent Platform)
| Version | Kafka Version | Status |
|---------|---------------|--------|
| 8.1.0   | 3.8.0         | ✓      |
| 7.9.0   | 3.7.0         | ✓      |
| 7.7.4   | 3.7.0         | ✓      |
| 7.7.3   | 3.7.0         | ✓      |

### Python (confluent-kafka)
| Version | Status |
|---------|--------|
| 2.8.0   | ✓      |
| 2.6.2   | ✓      |
| 2.5.3   | ✓      |

### Go (srclient)
| Version | Status |
|---------|--------|
| 0.7.0   | ✓      |

## Prerequisites

- Docker and Docker Compose
- Java 11+ and Maven (for Java tests)
- Python 3.8+ (for Python tests)
- Go 1.21+ (for Go tests)

## Running Tests

### Start Test Infrastructure

First, start the Schema Registry and supporting services:

```bash
# Start services
docker-compose up -d

# Wait for Schema Registry to be healthy
./wait-for-registry.sh
```

### Run All Tests

```bash
./run_all_tests.sh
```

### Run Tests by Language

```bash
# Java tests (all Confluent versions)
cd java && ./run_tests.sh

# Python tests (all confluent-kafka versions)
cd python && ./run_tests.sh

# Go tests
cd go && ./run_tests.sh
```

### Run Tests for Specific Version

```bash
# Java with specific Confluent version
cd java && mvn test -P confluent-8.1

# Python with specific version
cd python
python -m venv venv && source venv/bin/activate
pip install "confluent-kafka[avro,json,protobuf]==2.6.1" pytest
SCHEMA_REGISTRY_URL=http://localhost:8081 pytest
```

## Schema Types Tested

### Avro
- Simple record types
- Complex types with enums
- Nullable fields (union types)
- Schema evolution (adding fields with defaults)

### Protobuf
- Proto3 message types
- Map fields
- Schema evolution (adding fields)

### JSON Schema
- Draft-07 schemas
- Nested objects and arrays
- Required vs optional properties

## Wire Format Verification

The tests verify the Confluent wire format:

```
+--------+------------+---------+
| Magic  | Schema ID  | Payload |
| (1B)   | (4B BE)    | (var)   |
+--------+------------+---------+
```

- **Magic byte**: Always `0x00`
- **Schema ID**: 4-byte big-endian integer
- **Payload**: Schema-specific encoded data

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SCHEMA_REGISTRY_URL` | `http://localhost:8081` | Schema Registry endpoint |

## Troubleshooting

### Tests fail with connection refused
Ensure the Schema Registry is running and healthy:
```bash
curl http://localhost:8081/subjects
```

### Python tests fail with import errors
Install the required dependencies:
```bash
pip install "confluent-kafka[avro,json,protobuf]" pytest
```

### Go tests fail with module errors
Download dependencies:
```bash
cd go && go mod download
```

## Related

- GitHub Issue: #254
- [Confluent Schema Registry](https://docs.confluent.io/platform/current/schema-registry/index.html)
- [Wire Format Specification](https://docs.confluent.io/platform/current/schema-registry/fundamentals/serdes-develop/index.html#wire-format)
