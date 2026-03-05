Compare Avro, Protobuf, and JSON Schema for the use case: {use_case}

## Format Comparison

| Feature | Avro | Protobuf | JSON Schema |
|---------|------|----------|-------------|
| Serialization | Binary (compact) | Binary (compact) | Text (JSON) |
| Schema evolution | Excellent | Good | Limited |
| Type system | Rich (unions, logical types) | Strong (oneof, well-known types) | Flexible (oneOf, anyOf) |
| Code generation | Moderate | Excellent | Minimal |
| Human readability | Schema: JSON, Data: binary | Schema: .proto, Data: binary | Both: JSON |
| Kafka integration | Native | Supported | Supported |
| gRPC support | Limited | Native | Not applicable |
| Validation | Schema-level | Schema-level | Rich constraints |

## Recommendations by use case

**Event streaming (Kafka):** Avro
- Best schema evolution support with BACKWARD/FORWARD compatibility
- Compact binary serialization reduces Kafka storage/bandwidth
- Native Kafka ecosystem integration

**RPC/Microservices:** Protobuf
- Native gRPC support with code generation
- Strong typing across languages
- Efficient binary serialization

**REST APIs:** JSON Schema
- Human-readable request/response validation
- Rich constraint validation (patterns, ranges, formats)
- Direct JSON compatibility

**Mixed/CQRS systems:** Use multiple formats
- Avro for events (Kafka topics)
- Protobuf for commands (gRPC)
- JSON Schema for queries (REST responses)

Available tools: register_schema, get_schema_types
