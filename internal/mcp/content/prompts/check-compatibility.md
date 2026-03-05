Troubleshoot compatibility issues for subject {subject}.

Steps:
1. Use get_config to check the current compatibility level for {subject}
2. Use list_versions to see all registered versions
3. Use get_latest_schema to inspect the current schema
4. Use check_compatibility to test your new schema against existing versions
5. If incompatible, review the error details and adjust your schema

Common compatibility fixes:
- BACKWARD violations: Add a default value to new required fields, or make them optional
- FORWARD violations: Don't remove fields that consumers might depend on
- FULL violations: Only add optional fields with defaults

If you need to make a breaking change:
- Consider using set_config to temporarily change the compatibility level
- Or create a new subject (e.g. subject-v2) for the breaking change
- Use set_mode READONLY to protect finalized subjects
