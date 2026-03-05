Set up data contracts for subject {subject}.

Data contracts add metadata, tags, and data quality rules to schemas.

Steps:
1. Use get_latest_schema to inspect the current schema for {subject}
2. Use set_config_full to add metadata and rules:

   Metadata properties:
   - owner: team or person responsible
   - description: what this schema represents
   - tags: classification tags (e.g. pii, financial, internal)

   Data quality rules (ruleSet):
   - DOMAIN rules: field-level validation (e.g. email format, range checks)
   - MIGRATION rules: transform data between versions
   - All rules have: name, kind, type, mode, expr, tags

3. Use get_config_full to verify the configuration
4. Use get_subject_metadata to inspect applied metadata

Available tools: set_config_full, get_config_full, get_subject_config_full, get_subject_metadata

Example metadata structure:
{
  "properties": {"owner": "data-team", "description": "User events"},
  "ruleSet": {
    "domainRules": [
      {"name": "email_check", "kind": "CONDITION", "type": "DOMAIN", "mode": "WRITE", "expr": "email matches '^.+@.+$'"}
    ]
  }
}
