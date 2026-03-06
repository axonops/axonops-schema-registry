Workflow for onboarding a new team onto the schema registry.

## Step 1: Create a Context

Create an isolated namespace for team "{team_name}":
```
# Contexts are created implicitly by registering a schema in them.
# First, register a bootstrap schema or set context-level config.
set_config(compatibility_level: "BACKWARD", context: ".{team_name}")
```

## Step 2: Register Team Schemas

Register schemas within the team context:
```
register_schema(subject: "orders-value", schema: <schema>, schema_type: "AVRO", context: ".{team_name}")
```

## Step 3: Set Context-Level Compatibility

Set the team's default compatibility level:
```
set_config(compatibility_level: "BACKWARD", context: ".{team_name}")
```

Individual subjects can override:
```
set_config(subject: "shared-types", compatibility_level: "FULL_TRANSITIVE", context: ".{team_name}")
```

## Step 4: Create User and API Keys

Create a user for the team:
```
create_user(username: "{team_name}-service", password: "<secure>", role: "developer")
```

Create API keys for services:
```
create_apikey(name: "{team_name}-producer", role: "developer", expires_in: 2592000)
create_apikey(name: "{team_name}-consumer", role: "readonly", expires_in: 2592000)
```

## Step 5: Set Up Naming Conventions

Establish naming rules for the team:
- Subject pattern: `{team_name}-{topic}-{key|value}`
- Avro namespace: `com.company.{team_name}`

Validate names:
```
validate_subject_name(subject: "{team_name}-orders-value", strategy: "topic_name")
```

## Step 6: Verify Context Isolation

Confirm the team's schemas are isolated from other contexts:
```
list_subjects(context: ".{team_name}")
```

Confirm default context does not see team schemas:
```
list_subjects()
```

## Step 7: Browse via Context-Scoped Resources

Team data is accessible via context-scoped resource URIs:
- `schema://contexts/.{team_name}/subjects` -- all team subjects
- `schema://contexts/.{team_name}/subjects/<name>` -- subject details
- `schema://contexts/.{team_name}/config` -- team config

## Step 8: Documentation Pointers

Direct the team to these glossary resources:
- `schema://glossary/core-concepts` -- schema registry fundamentals
- `schema://glossary/contexts` -- context isolation details
- `schema://glossary/best-practices` -- per-format guidance
- `schema://glossary/design-patterns` -- common patterns

Available tools: set_config, register_schema, create_user, create_apikey, validate_subject_name, list_subjects, list_contexts
