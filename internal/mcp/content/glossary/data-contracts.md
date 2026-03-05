# Data Contracts

## Overview

Data contracts are governance policies attached to schemas. They allow you to annotate fields with descriptive metadata, classify sensitive data, and define rules applied during validation, migration, or serialization. Data contracts build on top of schema registration -- they add governance without changing how schemas are parsed, fingerprinted, or compatibility-checked.

## Metadata

Metadata provides descriptive annotations for a schema. It has three components:

| Component | Type | Purpose |
|-----------|------|---------|
| **properties** | map[string]string | Key-value annotations: "owner": "payments-team", "pii": "true", "domain": "billing" |
| **tags** | map[string][]string | Field-to-tag mapping. Keys are field paths (e.g., "Order.email"), values are tag arrays (e.g., ["PII", "GDPR"]) |
| **sensitive** | []string | Field paths containing sensitive data (e.g., ["ssn", "credit_card"]) |

## How Metadata Affects Schema Identity

- Metadata is **NOT** included in the SHA-256 fingerprint.
- Same schema text with **different metadata** creates a **new version** but gets the **same global ID**.
- Same schema text with **same metadata** is treated as a duplicate -- existing version and ID returned.
- This separates content identity (what the schema describes) from governance identity (how it is annotated).

## RuleSets

A RuleSet defines executable governance policies in three categories:

| Category | Field | Purpose |
|----------|-------|---------|
| **Domain rules** | domainRules | Validation/transformation on schema content (e.g., "all field names MUST be camelCase") |
| **Migration rules** | migrationRules | Rules during schema evolution (e.g., "renamed fields MUST provide a migration path") |
| **Encoding rules** | encodingRules | Rules during serialization/deserialization (e.g., "encrypt PII-tagged fields") |

## Rule Structure

Each rule has these fields:

| Field | Type | Description |
|-------|------|-------------|
| **name** | string | Unique name for this rule |
| **kind** | string | CONDITION (validate) or TRANSFORM (modify) |
| **mode** | string | WRITE, READ, UPGRADE, DOWNGRADE, WRITEREAD, UPDOWN |
| **type** | string | Rule type: CEL, JSON_TRANSFORM, ENCRYPT, etc. |
| **tags** | []string | Optional field tags this rule applies to |
| **params** | map[string]string | Rule-specific parameters |
| **expr** | string | Rule expression |
| **onSuccess** | string | Action on success: NONE, ERROR |
| **onFailure** | string | Action on failure: NONE, ERROR, DLQ |
| **disabled** | boolean | Whether this rule is currently disabled |

## Config-Level Defaults and Overrides

Rather than attaching metadata/rules to every registration, set defaults and overrides at the config level:

| Config Field | Purpose |
|-------------|---------|
| **defaultMetadata** | Merged into registrations that do not specify their own metadata |
| **defaultRuleSet** | Merged into registrations that do not specify their own rules |
| **overrideMetadata** | ALWAYS takes precedence over both defaults and request values |
| **overrideRuleSet** | ALWAYS takes precedence over both defaults and request rules |

## The 3-Layer Merge

When a schema is registered, the registry applies a 3-layer merge:

    Layer 1: defaultMetadata / defaultRuleSet       (base, from config)
    Layer 2: request metadata / ruleSet              (from POST body)
    Layer 3: overrideMetadata / overrideRuleSet      (from config, always wins)

Properties merge by key. Tags merge by field path. Rules merge by name. The override layer always wins.

## Inheritance from Previous Versions

When registering a new version, metadata and rules from the **previous version** are carried forward unless explicitly replaced. This means governance policies accumulate across versions.

## Optimistic Concurrency

The special metadata property **confluent:version** enables optimistic concurrency control:
- Include it in a registration request with the expected metadata version number.
- If the current metadata version does not match, the registration is rejected with HTTP 409.
- This prevents concurrent updates from silently overwriting each other.

## MCP Tools

- **set_config_full / get_config_full** -- manage config with metadata and ruleSet defaults/overrides
- **get_subject_metadata** -- inspect applied metadata on a subject
- **register_schema** -- register with metadata and ruleSet in the request body
- **get_latest_schema** -- fetch current schema including metadata
