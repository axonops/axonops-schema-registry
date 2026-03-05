Audit the version history of subject {subject}.

Steps:
1. Use list_versions to get all version numbers for {subject}
2. Use get_schema_version for each version to see the full schema
3. Compare consecutive versions to identify changes:
   - Added fields
   - Removed fields
   - Type changes
   - Default value changes
4. Use get_config to check the compatibility policy
5. Use get_referenced_by to find schemas that reference this subject

This helps you understand:
- How the schema has evolved over time
- Whether evolution has followed best practices
- If any versions introduced breaking changes
- Which other schemas depend on this one

Available tools: list_versions, get_schema_version, get_latest_schema, get_config, get_referenced_by
