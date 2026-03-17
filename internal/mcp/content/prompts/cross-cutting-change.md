Workflow for making a cross-cutting field change across multiple schemas.

## Step 1: Determine Scope

If the change spans multiple contexts:
```
list_contexts()
```

## Step 2: Find All Affected Schemas

Find every schema containing the field "{field_name}":
```
find_schemas_by_field(field_name: "{field_name}")
```

This returns all subjects and versions where the field appears.

## Step 3: Check Current Consistency

Verify the field's type is consistent across all schemas:
```
check_field_consistency(field_name: "{field_name}")
```

This shows if the field has different types in different schemas (type drift).

## Step 4: Plan Per-Subject Changes

For each affected subject:
```
get_latest_schema(subject: "<subject>")
get_config(subject: "<subject>")
```

Check what compatibility level each subject uses -- this determines what changes are safe.

## Step 5: Test Changes

Test compatibility for each subject before registering:
```
check_compatibility(subject: "<subject>", schema: <proposed_schema>)
```

For bulk testing across multiple subjects at once:
```
check_compatibility_multi(schemas: [
  {"subject": "subject-a", "schema": <schema_a>},
  {"subject": "subject-b", "schema": <schema_b>}
])
```

## Step 6: Handle Failures

If compatibility checks fail:
```
explain_compatibility_failure(subject: "<subject>", schema: <proposed_schema>)
suggest_compatible_change(subject: "<subject>", schema: <proposed_schema>)
```

## Step 7: Execute in Dependency Order

Use the dependency graph to determine the safe order of registration:
```
get_dependency_graph(subject: "<subject>")
```

Register referenced schemas first, then schemas that reference them.

## Step 8: Verify Changes

After all registrations:
```
diff_schemas(subject: "<subject>", version1: <old_version>, version2: "latest")
check_field_consistency(field_name: "{field_name}")
```

Confirm the field is now consistent across all schemas.

Available tools: list_contexts, find_schemas_by_field, check_field_consistency, get_latest_schema, get_config, check_compatibility, check_compatibility_multi, explain_compatibility_failure, suggest_compatible_change, get_dependency_graph, diff_schemas
