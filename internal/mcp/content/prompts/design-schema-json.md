Design a JSON Schema following these best practices:
- Use "type": "object" as the root type
- Define a "required" array listing mandatory fields
- Use "additionalProperties": false to prevent unexpected fields
- Use format validators: "email", "uri", "date-time", "uuid"
- Use pattern for custom string validation (regex)
- Use minimum/maximum for number ranges, minLength/maxLength for strings
- Use enum for fixed value sets
- Use $ref for reusable type definitions
- Consider using oneOf/anyOf for variant types

Available tools: register_schema (with schema_type: JSON), check_compatibility
