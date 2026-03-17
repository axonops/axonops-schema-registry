Assess the impact of a proposed schema change on subject {subject}.

## Step 1: Understand the current state
1. Use **get_latest_schema** to fetch the current schema for {subject}
2. Use **get_config** to check the compatibility level
3. Use **list_versions** to see the version history

## Step 2: Find dependents
1. Use **get_referenced_by** to find schemas that reference this subject
2. Use **get_dependency_graph** to see the full transitive dependency tree
3. Use **find_similar_schemas** to identify structurally related schemas

## Step 3: Check field usage
1. Use **check_field_consistency** for fields you plan to change
2. Use **find_schemas_by_field** to find other schemas using the same field names

## Step 4: Validate the change
1. Use **check_compatibility** to test your proposed schema
2. Use **check_compatibility_multi** if the schema is used across multiple subjects
3. Use **explain_compatibility_failure** if compatibility fails
4. Use **diff_schemas** to see a structural comparison

## Step 5: Plan the rollout
1. Use **suggest_schema_evolution** to generate a compatible schema
2. Use **plan_migration_path** if the change requires multiple steps
3. Consider using **set_mode** READONLY on the subject during migration
