Guide for integrating schema registry validation into CI/CD pipelines.

## Pre-Commit Checks

Run these before a schema change is merged:

### Syntax Validation
```
validate_schema(schema: <proposed_schema>, schema_type: "AVRO")
```
Catches malformed JSON, missing required fields, invalid types.

### Compatibility Check
```
check_compatibility(subject: "<subject>", schema: <proposed_schema>, schema_type: "AVRO")
```
Catches backward-incompatible changes (removed fields, changed types).

### Quality Gate
```
score_schema_quality(schema: <proposed_schema>, schema_type: "AVRO")
```
Enforce minimum quality scores: naming conventions, documentation, nullability patterns.

### Field Consistency
```
check_field_consistency(field_name: "customer_id", schema_type: "AVRO")
```
Ensure shared fields use consistent types across all schemas.

### Subject Name Validation
```
validate_subject_name(subject: "<subject>", strategy: "topic_name")
```
Enforce naming conventions.

## Deployment-Time Registration

When deploying a service that produces/consumes schemas:

### Register Schema
```
register_schema(subject: "<subject>", schema: <schema>, schema_type: "AVRO")
```

### Verify Registration
```
get_latest_schema(subject: "<subject>")
```
Confirm the registered version matches expectations.

### Compare with Previous
```
diff_schemas(subject: "<subject>", version1: "latest", version2: <previous_version>)
```
Log the diff for audit trail.

## Rollback Strategy

If a schema registration causes issues:

1. Do NOT delete the registered schema (consumers may already be using it)
2. Register the previous schema version as a new version (if compatible)
3. Or use `set_mode(mode: "READONLY")` to prevent further registrations while investigating

## Pipeline Architecture

Use **contexts** for environment separation:

```
# PR validation (staging context)
validate_schema(schema: <schema>)
check_compatibility(subject: "orders-value", schema: <schema>, context: ".staging")

# Merge to main (production context)
register_schema(subject: "orders-value", schema: <schema>, context: ".production")
```

## Pseudo-Code Example

```
# In CI pipeline (e.g., GitHub Actions, Jenkins)

# 1. Validate syntax
result = mcp_call("validate_schema", {schema: new_schema, schema_type: "AVRO"})
assert result.valid == true

# 2. Check compatibility
result = mcp_call("check_compatibility", {subject: subject, schema: new_schema})
assert result.is_compatible == true

# 3. Quality gate
result = mcp_call("score_schema_quality", {schema: new_schema, schema_type: "AVRO"})
assert result.overall_score >= 70

# 4. Register on deploy
result = mcp_call("register_schema", {subject: subject, schema: new_schema})
schema_id = result.id
```

Available tools: validate_schema, check_compatibility, score_schema_quality, check_field_consistency, validate_subject_name, register_schema, get_latest_schema, diff_schemas, set_mode
