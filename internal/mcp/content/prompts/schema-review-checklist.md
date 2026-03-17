Pre-registration review checklist for schema "{subject}".

## 1. Syntax Validation
```
validate_schema(schema: <proposed_schema>, schema_type: "AVRO")
```
Ensure the schema is syntactically valid.

## 2. Compatibility Check
```
check_compatibility(subject: "{subject}", schema: <proposed_schema>)
```
Verify the change is compatible with the current compatibility level.

## 3. Quality Score
```
score_schema_quality(schema: <proposed_schema>, schema_type: "AVRO")
```
Review naming conventions, documentation, nullability, and evolution readiness.

## 4. Subject Name Validation
```
validate_subject_name(subject: "{subject}", strategy: "topic_name")
```
Ensure the subject name follows established naming conventions.

## 5. Uniqueness Check
```
lookup_schema(subject: "{subject}", schema: <proposed_schema>)
find_similar_schemas(schema: <proposed_schema>, schema_type: "AVRO")
```
Check if this schema already exists (deduplication) or if similar schemas should be consolidated.

## 6. Dependency Check
```
# If the schema has references, verify they exist:
get_schema_version(subject: "<ref_subject>", version: <ref_version>)
```

For schemas that are referenced by others, use FULL_TRANSITIVE compatibility:
```
get_referenced_by(subject: "{subject}")
```

## 7. Field Consistency
```
check_field_consistency(field_name: "<shared_field>")
```
Ensure shared field names use the same type as in other schemas.

## 8. Complexity Assessment
```
get_schema_complexity(schema: <proposed_schema>, schema_type: "AVRO")
```
Review schema depth, field count, and union complexity.

## 9. Impact Analysis
```
diff_schemas(subject: "{subject}", version1: "latest", version2: <proposed_version>)
get_referenced_by(subject: "{subject}")
```
Understand what downstream schemas and consumers would be affected.

## 10. Data Contracts
If the subject has data contracts (metadata, rules):
```
get_subject_config_full(subject: "{subject}")
```
Ensure the new schema version is compatible with existing rules and metadata.

## 11. Context Verification
Ensure you are registering in the correct context:
```
list_subjects(context: "<expected_context>")
```

## 12. Register
After all checks pass:
```
register_schema(subject: "{subject}", schema: <proposed_schema>, schema_type: "AVRO")
```

Available tools: validate_schema, check_compatibility, score_schema_quality, validate_subject_name, lookup_schema, find_similar_schemas, get_referenced_by, check_field_consistency, get_schema_complexity, diff_schemas, get_subject_config_full, register_schema
