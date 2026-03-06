Guide for setting up schema governance policies.

## Step 1: Naming Conventions

Validate subject names follow an established pattern:
```
validate_subject_name(subject: "<subject>", strategy: "topic_name")
```

Detect naming patterns across the registry:
```
detect_schema_patterns()
```

Enforce consistent naming: TopicNameStrategy for Kafka, reverse-domain namespace for Avro.

## Step 2: Global Compatibility

Set a safe default compatibility level:
```
set_config(compatibility_level: "BACKWARD")
```

For shared types used by multiple teams:
```
set_config(subject: "com.company.Address", compatibility_level: "FULL_TRANSITIVE")
```

## Step 3: Quality Gates

Score schema quality before registration:
```
score_schema_quality(schema: <schema>, schema_type: "AVRO")
```

Set minimum thresholds:
- Documentation score >= 70 (fields have doc attributes)
- Naming score >= 80 (consistent snake_case)
- Evolution readiness >= 60 (defaults on optional fields)

Check field type consistency across schemas:
```
check_field_consistency(field_name: "customer_id", schema_type: "AVRO")
```

Ensure shared fields use the same type everywhere.

## Step 4: Data Contracts

Add governance metadata to schemas:
```
set_config_full(
  subject: "<subject>",
  override_metadata: {
    "properties": {
      "owner": "team-orders",
      "pii": "true",
      "classification": "CONFIDENTIAL"
    },
    "tags": ["pii", "gdpr"]
  }
)
```

Add data quality rules:
```
set_config_full(
  subject: "<subject>",
  default_rule_set: {
    "domainRules": [
      {"name": "pii-check", "kind": "CONDITION", "type": "CEL", "expr": "has(message.email)"}
    ]
  }
)
```

## Step 5: Contexts for Governance Boundaries

Use contexts to create governance boundaries:
- `.production` -- strict compatibility (FULL), quality gates enforced
- `.staging` -- relaxed compatibility (BACKWARD), quality gates advisory
- `.sandbox` -- NONE compatibility, no quality gates

## Step 6: RBAC Enforcement

Set up roles appropriate to governance needs:
- **readonly** for monitoring dashboards and auditors
- **developer** for schema producers (can register, cannot delete)
- **admin** for schema governance team (can set config, modes)

## Step 7: Audit Logging

Enable audit logging to track all schema changes:
- Who registered/deleted/modified schemas
- When compatibility levels were changed
- Which subjects were deprecated

## Step 8: CI/CD Integration

See the **cicd-integration** prompt for pipeline setup.

Available tools: validate_subject_name, detect_schema_patterns, set_config, score_schema_quality, check_field_consistency, set_config_full, list_contexts, list_roles
