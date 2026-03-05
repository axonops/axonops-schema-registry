Design an Avro schema following these best practices:
- Use a descriptive record name in PascalCase with a namespace (e.g. com.company.events)
- Use snake_case for field names
- Always include a namespace to avoid naming conflicts
- Use union types ["null", "type"] with default null for optional fields
- Use logical types for dates (timestamp-millis), decimals (bytes + decimal), and UUIDs (string + uuid)
- Consider schema evolution: add new fields with defaults, avoid removing or renaming fields
- Use enums for fixed sets of values

Available tools: register_schema, check_compatibility, get_latest_schema, lookup_schema
