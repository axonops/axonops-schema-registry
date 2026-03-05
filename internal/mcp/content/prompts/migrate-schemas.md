Migrate schemas from {source} to {target} format.

Steps:
1. Use list_subjects to find schemas to migrate
2. Use get_latest_schema to inspect each schema
3. Convert the schema to {target} format following these guidelines:
4. Use register_schema with schema_type: {target} to register the converted schema
5. Use check_compatibility to validate if needed

Migration considerations from {source} to {target}:
