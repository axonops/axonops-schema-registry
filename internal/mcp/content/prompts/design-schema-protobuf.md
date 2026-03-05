Design a Protobuf schema following these best practices:
- Use syntax = "proto3" (required)
- Use a package declaration matching your domain (e.g. package company.events.v1)
- Use PascalCase for message and enum names, snake_case for field names
- Use explicit field numbers and never reuse deleted field numbers
- Use oneof for variant/union types
- Use repeated for arrays, map<K,V> for key-value pairs
- Use well-known types (google.protobuf.Timestamp, Duration, etc.) when appropriate
- Use enums with UNSPECIFIED = 0 as the first value
- Consider backward compatibility: only add new fields, never change field numbers

Available tools: register_schema (with schema_type: PROTOBUF), check_compatibility
