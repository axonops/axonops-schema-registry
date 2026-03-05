Review the schema quality for subject {subject}.

Use get_latest_schema to fetch the current schema, then evaluate:

1. **Naming conventions**:
   - Record/message names: PascalCase
   - Field names: snake_case
   - Enum values: UPPER_SNAKE_CASE
   - Namespace/package: reverse domain notation

2. **Nullability**:
   - Optional fields should be nullable (Avro: union with null, Protobuf: optional)
   - Required fields should NOT be nullable
   - Default values should be meaningful

3. **Type usage**:
   - Use logical/semantic types (timestamps, UUIDs, decimals) instead of raw primitives
   - Use enums for fixed value sets instead of plain strings
   - Use appropriate numeric precision (int vs long, float vs double)

4. **Evolution readiness**:
   - All fields should have sensible defaults for backward compatibility
   - Avoid required fields that might become optional later
   - Consider using a version field or schema fingerprint

5. **Documentation**:
   - Fields should have descriptive names that are self-documenting
   - Complex fields should have doc comments (Avro: "doc" field, Protobuf: // comments)

Available tools: get_latest_schema, list_versions, get_config
