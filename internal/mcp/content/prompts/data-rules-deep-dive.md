Comprehensive guide to data contract rules.

## Rule Categories

### Domain Rules (domainRules)
Validation and transformation rules applied to schema content at registration time.

Example: enforce camelCase field naming
    {
      "name": "checkCamelCase",
      "kind": "CONDITION",
      "mode": "WRITE",
      "type": "CEL",
      "expr": "name.matches('^[a-z][a-zA-Z0-9]*$')",
      "onFailure": "ERROR"
    }

### Migration Rules (migrationRules)
Rules applied during schema version transitions (upgrades and downgrades).

Example: rename a field during upgrade
    {
      "name": "renameCustomerToClient",
      "kind": "TRANSFORM",
      "mode": "UPGRADE",
      "type": "JSON_TRANSFORM",
      "expr": "$.customer -> $.client"
    }

### Encoding Rules (encodingRules)
Rules applied during serialization/deserialization, typically for field-level encryption.

Example: encrypt PII-tagged fields
    {
      "name": "encryptPII",
      "kind": "TRANSFORM",
      "mode": "WRITE",
      "type": "ENCRYPT",
      "tags": ["PII"]
    }

## Rule Fields

| Field | Values | Description |
|-------|--------|-------------|
| **name** | any string | Unique identifier for this rule |
| **kind** | CONDITION, TRANSFORM | Validate (CONDITION) or modify (TRANSFORM) |
| **mode** | WRITE, READ, UPGRADE, DOWNGRADE, WRITEREAD, UPDOWN | When the rule applies |
| **type** | CEL, JSON_TRANSFORM, ENCRYPT, etc. | Rule engine/evaluator type |
| **tags** | string array | Field tags this rule targets (e.g., ["PII", "GDPR"]) |
| **params** | map[string]string | Rule-specific configuration |
| **expr** | string | Rule expression (CEL expression, JSONPath, etc.) |
| **onSuccess** | NONE, ERROR | Action when rule passes |
| **onFailure** | NONE, ERROR, DLQ | Action when rule fails |
| **disabled** | boolean | Whether this rule is currently inactive |

## The 3-Layer Merge

When registering a schema:
1. **defaultRuleSet** from config -- base rules applied when request has none
2. **request ruleSet** -- rules from the POST body
3. **overrideRuleSet** from config -- always wins, overrides everything

Rules merge by name: if two layers define a rule with the same name, the higher layer wins.

## Setting Config-Level Rules

Use **set_config_full** to set defaults and overrides:
- defaultMetadata / defaultRuleSet: baseline governance
- overrideMetadata / overrideRuleSet: mandatory governance (always applied)

## Inheritance
Rules from the previous version carry forward unless explicitly replaced. This means governance accumulates across versions.

## MCP Tools
- **set_config_full / get_config_full** -- manage rules at the config level
- **register_schema** -- register with ruleSet in the request
- **get_latest_schema** -- inspect current rules

For domain knowledge, read: schema://glossary/data-contracts
