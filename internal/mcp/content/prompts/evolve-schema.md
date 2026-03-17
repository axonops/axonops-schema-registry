Evolve the schema for subject {subject} safely.

Steps:
1. Use get_latest_schema to inspect the current schema for {subject}
2. Use get_config to check the compatibility level
3. Plan your changes following the compatibility rules:
   - BACKWARD: new schema can read old data (add optional fields with defaults)
   - FORWARD: old schema can read new data (only remove optional fields)
   - FULL: both backward and forward compatible
4. Use check_compatibility to validate your changes before registering
5. Use register_schema to register the evolved schema

Common safe changes:
- Add a new optional field with a default value
- Add a new field with a union type ["null", "type"] and default null
- Widen a type (e.g. int → long in Avro)

Breaking changes to avoid:
- Removing a required field
- Changing a field type incompatibly
- Renaming a field (treated as remove + add)
