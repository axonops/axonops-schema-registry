# MCP API Reference

> Auto-generated from the MCP server registration. Do not edit manually.
>
> Regenerate with: `go run ./cmd/generate-mcp-docs > docs/mcp-reference.md`

**105 tools** (71 read-only, 34 write) | **47 resources** (25 static, 22 templated) | **33 prompts**

## Contents

- [Tools](#tools)
- [Resources](#resources)
- [Prompts](#prompts)

---

## Tools

| # | Tool | Read-Only | Description |
|---|------|-----------|-------------|
| 1 | `change_password` |  | Change a user's password. Requires the user's ID, old password, and new password. |
| 2 | `check_compatibility` | Yes | Check if a schema is compatible with existing versions of a subject according to the configured compatibility level |
| 3 | `check_compatibility_multi` | Yes | Check schema compatibility against multiple subjects at once, returning per-subject results. |
| 4 | `check_field_consistency` | Yes | Check if a field name is used with the same type across all schemas. Generates naming variants (snake_case, camelCase... |
| 5 | `check_write_mode` | Yes | Check if write operations are allowed for a subject. Returns the blocking mode name (READONLY or READONLY_OVERRIDE) o... |
| 6 | `compare_subjects` | Yes | Compare the latest schemas of two different subjects, showing structural differences. |
| 7 | `count_subjects` | Yes | Count the total number of registered subjects in the registry. |
| 8 | `count_versions` | Yes | Count the number of schema versions registered for a subject. |
| 9 | `create_apikey` |  | Create a new API key for a user. Returns the raw key (only shown once). Requires user_id, name, role, and expires_in ... |
| 10 | `create_dek` |  | Create a new Data Encryption Key (DEK) under a KEK. The DEK is used for client-side field encryption. |
| 11 | `create_exporter` |  | Create a new schema exporter for cross-cluster schema replication. Context types: AUTO, CUSTOM, NONE. |
| 12 | `create_kek` |  | Create a new Key Encryption Key (KEK) for client-side field encryption (CSFLE). A KEK wraps Data Encryption Keys (DEK... |
| 13 | `create_user` |  | Create a new user. Requires username, password, and role (super_admin, admin, developer, readonly). |
| 14 | `delete_apikey` |  | Delete an API key by ID. |
| 15 | `delete_config` |  | Delete the compatibility configuration for a subject (reverts to global default) or delete the global config |
| 16 | `delete_dek` |  | Delete a Data Encryption Key (DEK). Use permanent=true for hard delete (default is soft-delete). |
| 17 | `delete_exporter` |  | Delete an exporter by name. |
| 18 | `delete_kek` |  | Delete a Key Encryption Key (KEK). Use permanent=true for hard delete (default is soft-delete). |
| 19 | `delete_mode` |  | Delete the mode for a subject (reverts to global default) or delete the global mode |
| 20 | `delete_subject` |  | Delete a subject and all its schema versions. Soft-deletes by default; use permanent=true for hard delete. |
| 21 | `delete_user` |  | Delete a user by ID. |
| 22 | `delete_version` |  | Delete a specific schema version. Soft-deletes by default; use permanent=true for hard delete (requires prior soft-de... |
| 23 | `detect_schema_patterns` | Yes | Scan the registry to detect naming patterns, common field groups, and evolution statistics. |
| 24 | `diff_schemas` | Yes | Diff two schema versions within a subject, showing added, removed, and type-changed fields. Fields are extracted from... |
| 25 | `explain_compatibility_failure` | Yes | Run a compatibility check and provide detailed, human-readable explanations of any failures. |
| 26 | `export_schema` | Yes | Export a single schema version with its configuration and metadata in a portable format. |
| 27 | `export_subject` | Yes | Export all schema versions for a subject with configuration and metadata. |
| 28 | `find_schemas_by_field` | Yes | Find all schemas containing a field with the given name. Exact mode auto-generates naming variants (snake_case, camel... |
| 29 | `find_schemas_by_type` | Yes | Find all schemas containing fields of a given type (e.g., 'int', 'string', 'record'). |
| 30 | `find_similar_schemas` | Yes | Find schemas structurally similar to a given subject using Jaccard similarity coefficient (|shared fields| / |total u... |
| 31 | `format_schema` | Yes | Format a schema by subject and version. Supported formats depend on schema type. Returns the formatted schema string. |
| 32 | `get_apikey` | Yes | Get an API key by ID. |
| 33 | `get_cluster_id` | Yes | Get the schema registry cluster ID. |
| 34 | `get_config` | Yes | Get the compatibility configuration for a subject or the global default. Omit subject for global config. |
| 35 | `get_config_full` | Yes | Get the full configuration record for a subject or global default, including metadata, ruleSets, alias, compatibility... |
| 36 | `get_dek` | Yes | Get a Data Encryption Key (DEK) by KEK name, subject, version, and algorithm. |
| 37 | `get_dependency_graph` | Yes | Build a dependency graph for a subject-version, showing all schemas that reference it (recursively, up to depth 10). |
| 38 | `get_exporter` | Yes | Get an exporter's configuration by name. |
| 39 | `get_exporter_config` | Yes | Get the destination configuration of an exporter. |
| 40 | `get_exporter_status` | Yes | Get the current status of an exporter (state, offset, error trace). |
| 41 | `get_global_config_direct` | Yes | Get the global configuration for the current context directly, without falling back to the __GLOBAL context. Returns ... |
| 42 | `get_kek` | Yes | Get a Key Encryption Key (KEK) by name. Use deleted=true to include soft-deleted KEKs. |
| 43 | `get_latest_schema` | Yes | Get the latest (most recent non-deleted) schema version for a subject |
| 44 | `get_max_schema_id` | Yes | Get the highest schema ID currently assigned in the registry |
| 45 | `get_mode` | Yes | Get the registry mode for a subject or the global default. Modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT |
| 46 | `get_raw_schema_by_id` | Yes | Get the raw schema string by its global ID, without any metadata |
| 47 | `get_raw_schema_version` | Yes | Get the raw schema string by subject name and version number, without any metadata |
| 48 | `get_referenced_by` | Yes | Get schemas that reference a specific subject-version pair |
| 49 | `get_registry_statistics` | Yes | Get aggregate statistics about the registry: total subjects, schemas, types breakdown, KEKs, DEKs, and exporters. |
| 50 | `get_schema_by_id` | Yes | Get a schema by its global ID, returning the full schema record including subject, version, type, and schema content |
| 51 | `get_schema_complexity` | Yes | Compute complexity metrics and grade (A-D) for a schema. Measures field_count (total fields including nested) and max... |
| 52 | `get_schema_history` | Yes | Get the full version history for a subject, including schema content and metadata for each version. |
| 53 | `get_schema_types` | Yes | Get the list of supported schema types (e.g. AVRO, PROTOBUF, JSON) |
| 54 | `get_schema_version` | Yes | Get a schema by subject name and version number |
| 55 | `get_schemas_by_subject` | Yes | Get all schema versions for a subject. Returns full schema records for every version, optionally including soft-delet... |
| 56 | `get_server_info` | Yes | Get schema registry server information including version and supported schema types |
| 57 | `get_server_version` | Yes | Get detailed server version information including version, commit hash, and build time. |
| 58 | `get_subject_config_full` | Yes | Get the full configuration record for a specific subject only, without falling back to global config. Returns error i... |
| 59 | `get_subject_metadata` | Yes | Get metadata for a subject. Without filters, returns the metadata from the latest schema version. With key/value filt... |
| 60 | `get_subjects_for_schema` | Yes | Get all subjects that use a specific schema ID |
| 61 | `get_user` | Yes | Get a user by ID. |
| 62 | `get_user_by_username` | Yes | Get a user by username. |
| 63 | `get_versions_for_schema` | Yes | Get all subject-version pairs that use a specific schema ID |
| 64 | `health_check` | Yes | Check if the schema registry is healthy and responding |
| 65 | `import_schemas` |  | Bulk import schemas with preserved IDs (for Confluent migration). Registry mode MUST be set to IMPORT first. |
| 66 | `list_apikeys` | Yes | List all API keys, optionally filtered by user_id. |
| 67 | `list_contexts` | Yes | List all tenant contexts in the schema registry. Each context is an isolated namespace for subjects and schemas. |
| 68 | `list_dek_versions` | Yes | List all version numbers for a DEK subject under a given KEK. |
| 69 | `list_deks` | Yes | List all subject names that have DEKs under a given KEK. |
| 70 | `list_exporters` | Yes | List all exporter names. Exporters replicate schemas to a destination schema registry (Schema Linking). |
| 71 | `list_keks` | Yes | List all Key Encryption Keys (KEKs). Use deleted=true to include soft-deleted KEKs. |
| 72 | `list_roles` | Yes | List all available RBAC roles with their permissions. |
| 73 | `list_schemas` | Yes | List schemas with optional filtering by subject prefix, deleted status, and pagination |
| 74 | `list_subjects` | Yes | List all registered subjects in the schema registry |
| 75 | `list_users` | Yes | List all users in the schema registry. |
| 76 | `list_versions` | Yes | List all version numbers registered for a subject |
| 77 | `lookup_schema` | Yes | Check if a schema is already registered under a subject. Returns the existing schema record if found. |
| 78 | `match_subjects` | Yes | Find subjects matching a pattern. Regex mode (regex=true) compiles as Go regex. Default mode uses case-sensitive subs... |
| 79 | `normalize_schema` | Yes | Parse and normalize a schema, returning the canonical form and fingerprint for deduplication. |
| 80 | `pause_exporter` |  | Pause a running exporter. The exporter retains its current offset and can be resumed later. |
| 81 | `plan_migration_path` | Yes | Compute a multi-step migration plan from a source schema to a target schema, decomposed into individually compatible ... |
| 82 | `register_schema` |  | Register a new schema version for a subject. If the same schema already exists, returns the existing record. |
| 83 | `reset_exporter` |  | Reset an exporter's offset back to zero, causing it to re-export all schemas. |
| 84 | `resolve_alias` | Yes | Resolve a subject alias. If the subject has an alias configured, returns the alias target. Otherwise returns the orig... |
| 85 | `resume_exporter` |  | Resume a paused exporter. The exporter continues from its last offset. |
| 86 | `revoke_apikey` |  | Revoke (disable) an API key without deleting it. |
| 87 | `rewrap_dek` |  | Re-encrypt a DEK's key material under the current KEK key version. Used after KEK rotation. |
| 88 | `rotate_apikey` |  | Rotate an API key: creates a new key with the same settings and revokes the old one. Returns the new raw key (only sh... |
| 89 | `score_schema_quality` | Yes | Score a schema's quality (0-100, grades A-F) across four categories: Naming (25 pts, checks snake_case convention), D... |
| 90 | `search_schemas` | Yes | Search schema content across all subjects using a regex or substring pattern. |
| 91 | `set_config` |  | Set the compatibility level for a subject or globally. Valid levels: NONE, BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FO... |
| 92 | `set_config_full` |  | Set the full configuration for a subject or globally, including compatibility level plus optional data contract field... |
| 93 | `set_mode` |  | Set the registry mode for a subject or globally. Valid modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT |
| 94 | `suggest_compatible_change` | Yes | Get rule-based advice for compatible schema changes based on the subject's compatibility level. BACKWARD: add fields ... |
| 95 | `suggest_schema_evolution` | Yes | Generate concrete schema code for a compatible evolution step (add field, deprecate field, add enum symbol). |
| 96 | `test_kek` |  | Test a KEK's KMS connectivity by performing a round-trip encrypt/decrypt test. Requires a KMS provider to be configured. |
| 97 | `undelete_dek` |  | Restore a soft-deleted Data Encryption Key (DEK). |
| 98 | `undelete_kek` |  | Restore a soft-deleted Key Encryption Key (KEK). |
| 99 | `update_apikey` |  | Update an API key's name, role, or enabled status. |
| 100 | `update_exporter` |  | Update an existing exporter's settings (context type, subjects, rename format, config). |
| 101 | `update_exporter_config` |  | Update the destination configuration of an exporter. |
| 102 | `update_kek` |  | Update an existing Key Encryption Key (KEK). Only kms_props, doc, and shared can be changed. |
| 103 | `update_user` |  | Update a user's email, password, role, or enabled status. |
| 104 | `validate_schema` | Yes | Validate a schema without registering it. Returns whether the schema is valid, its fingerprint, and any parse errors. |
| 105 | `validate_subject_name` | Yes | Validate a subject name against a naming strategy (topic_name, record_name, or topic_record_name). |

### Tool Details

#### `change_password`

Change a user's password. Requires the user's ID, old password, and new password.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes |  |
| `new_password` | string | Yes |  |
| `old_password` | string | Yes |  |

---

#### `check_compatibility`

Check if a schema is compatible with existing versions of a subject according to the configured compatibility level

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `references` | [null array] |  |  |
| `schema` | string | Yes |  |
| `schema_type` | string |  |  |
| `subject` | string | Yes |  |
| `version` | string |  |  |

---

#### `check_compatibility_multi`

Check schema compatibility against multiple subjects at once, returning per-subject results.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `references` | [null array] |  |  |
| `schema` | string | Yes |  |
| `schema_type` | string |  |  |
| `subjects` | [null array] | Yes |  |

---

#### `check_field_consistency`

Check if a field name is used with the same type across all schemas. Generates naming variants (snake_case, camelCase, PascalCase, kebab-case) to match fields regardless of convention. Reports type_counts map and per-subject usages. Detects type drift (e.g., user_id as long in one schema and string in another).

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `field` | string | Yes |  |

---

#### `check_write_mode`

Check if write operations are allowed for a subject. Returns the blocking mode name (READONLY or READONLY_OVERRIDE) or empty string if writes are allowed.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string |  |  |

---

#### `compare_subjects`

Compare the latest schemas of two different subjects, showing structural differences.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject_a` | string | Yes |  |
| `subject_b` | string | Yes |  |

---

#### `count_subjects`

Count the total number of registered subjects in the registry.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |

---

#### `count_versions`

Count the number of schema versions registered for a subject.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |

---

#### `create_apikey`

Create a new API key for a user. Returns the raw key (only shown once). Requires user_id, name, role, and expires_in (seconds).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `expires_in` | integer | Yes |  |
| `name` | string | Yes |  |
| `role` | string | Yes |  |
| `user_id` | integer | Yes |  |

---

#### `create_dek`

Create a new Data Encryption Key (DEK) under a KEK. The DEK is used for client-side field encryption.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `algorithm` | string |  |  |
| `encrypted_key_material` | string |  |  |
| `kek_name` | string | Yes |  |
| `subject` | string | Yes |  |
| `version` | integer |  |  |

---

#### `create_exporter`

Create a new schema exporter for cross-cluster schema replication. Context types: AUTO, CUSTOM, NONE.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `config` | object |  |  |
| `context` | string |  |  |
| `context_type` | string |  |  |
| `name` | string | Yes |  |
| `subject_rename_format` | string |  |  |
| `subjects` | [null array] |  |  |

---

#### `create_kek`

Create a new Key Encryption Key (KEK) for client-side field encryption (CSFLE). A KEK wraps Data Encryption Keys (DEKs) via a KMS provider.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `doc` | string |  |  |
| `kms_key_id` | string | Yes |  |
| `kms_props` | object |  |  |
| `kms_type` | string | Yes |  |
| `name` | string | Yes |  |
| `shared` | boolean |  |  |

---

#### `create_user`

Create a new user. Requires username, password, and role (super_admin, admin, developer, readonly).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `email` | string |  |  |
| `enabled` | [null boolean] |  |  |
| `password` | string | Yes |  |
| `role` | string | Yes |  |
| `username` | string | Yes |  |

---

#### `delete_apikey`

Delete an API key by ID.

**Annotations:** destructive

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes |  |

---

#### `delete_config`

Delete the compatibility configuration for a subject (reverts to global default) or delete the global config

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `confirm_token` | string |  |  |
| `context` | string |  |  |
| `dry_run` | boolean |  |  |
| `subject` | string |  |  |

---

#### `delete_dek`

Delete a Data Encryption Key (DEK). Use permanent=true for hard delete (default is soft-delete).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `algorithm` | string |  |  |
| `confirm_token` | string |  |  |
| `dry_run` | boolean |  |  |
| `kek_name` | string | Yes |  |
| `permanent` | boolean |  |  |
| `subject` | string | Yes |  |
| `version` | integer |  |  |

---

#### `delete_exporter`

Delete an exporter by name.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `confirm_token` | string |  |  |
| `dry_run` | boolean |  |  |
| `name` | string | Yes |  |

---

#### `delete_kek`

Delete a Key Encryption Key (KEK). Use permanent=true for hard delete (default is soft-delete).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `confirm_token` | string |  |  |
| `dry_run` | boolean |  |  |
| `name` | string | Yes |  |
| `permanent` | boolean |  |  |

---

#### `delete_mode`

Delete the mode for a subject (reverts to global default) or delete the global mode

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string |  |  |

---

#### `delete_subject`

Delete a subject and all its schema versions. Soft-deletes by default; use permanent=true for hard delete.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `confirm_token` | string |  |  |
| `context` | string |  |  |
| `dry_run` | boolean |  |  |
| `permanent` | boolean |  |  |
| `subject` | string | Yes |  |

---

#### `delete_user`

Delete a user by ID.

**Annotations:** destructive

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes |  |

---

#### `delete_version`

Delete a specific schema version. Soft-deletes by default; use permanent=true for hard delete (requires prior soft-delete).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `confirm_token` | string |  |  |
| `context` | string |  |  |
| `dry_run` | boolean |  |  |
| `permanent` | boolean |  |  |
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `detect_schema_patterns`

Scan the registry to detect naming patterns, common field groups, and evolution statistics.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |

---

#### `diff_schemas`

Diff two schema versions within a subject, showing added, removed, and type-changed fields. Fields are extracted from both versions and matched using normalized snake_case names. If version2 is omitted, diffs against the latest version.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |
| `version_from` | integer | Yes |  |
| `version_to` | integer | Yes |  |

---

#### `explain_compatibility_failure`

Run a compatibility check and provide detailed, human-readable explanations of any failures.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `references` | [null array] |  |  |
| `schema` | string | Yes |  |
| `schema_type` | string |  |  |
| `subject` | string | Yes |  |
| `version` | string |  |  |

---

#### `export_schema`

Export a single schema version with its configuration and metadata in a portable format.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `export_subject`

Export all schema versions for a subject with configuration and metadata.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |

---

#### `find_schemas_by_field`

Find all schemas containing a field with the given name. Exact mode auto-generates naming variants (snake_case, camelCase, PascalCase, kebab-case). Fuzzy mode uses Levenshtein distance with configurable threshold (default 0.7). Regex mode compiles the field name as a regular expression.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `field` | string | Yes |  |
| `match_type` | string |  |  |
| `threshold` | number |  |  |

---

#### `find_schemas_by_type`

Find all schemas containing fields of a given type (e.g., 'int', 'string', 'record').

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `regex` | boolean |  |  |
| `type_pattern` | string | Yes |  |

---

#### `find_similar_schemas`

Find schemas structurally similar to a given subject using Jaccard similarity coefficient (|shared fields| / |total unique fields|). Field names are normalized to snake_case before comparison. Returns similarity scores (0.0-1.0) and lists of shared fields.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |
| `threshold` | number |  |  |

---

#### `format_schema`

Format a schema by subject and version. Supported formats depend on schema type. Returns the formatted schema string.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `format` | string |  |  |
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `get_apikey`

Get an API key by ID.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes |  |

---

#### `get_cluster_id`

Get the schema registry cluster ID.

**Annotations:** read-only

---

#### `get_config`

Get the compatibility configuration for a subject or the global default. Omit subject for global config.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string |  |  |

---

#### `get_config_full`

Get the full configuration record for a subject or global default, including metadata, ruleSets, alias, compatibilityGroup, and all data contract fields. Uses 4-tier fallback: subject → context global → __GLOBAL → server default.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string |  |  |

---

#### `get_dek`

Get a Data Encryption Key (DEK) by KEK name, subject, version, and algorithm.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `algorithm` | string |  |  |
| `deleted` | boolean |  |  |
| `kek_name` | string | Yes |  |
| `subject` | string | Yes |  |
| `version` | integer |  |  |

---

#### `get_dependency_graph`

Build a dependency graph for a subject-version, showing all schemas that reference it (recursively, up to depth 10).

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `max_depth` | integer |  |  |
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `get_exporter`

Get an exporter's configuration by name.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes |  |

---

#### `get_exporter_config`

Get the destination configuration of an exporter.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes |  |

---

#### `get_exporter_status`

Get the current status of an exporter (state, offset, error trace).

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes |  |

---

#### `get_global_config_direct`

Get the global configuration for the current context directly, without falling back to the __GLOBAL context. Returns server default if no context-level global config is set.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |

---

#### `get_kek`

Get a Key Encryption Key (KEK) by name. Use deleted=true to include soft-deleted KEKs.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `deleted` | boolean |  |  |
| `name` | string | Yes |  |

---

#### `get_latest_schema`

Get the latest (most recent non-deleted) schema version for a subject

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |

---

#### `get_max_schema_id`

Get the highest schema ID currently assigned in the registry

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |

---

#### `get_mode`

Get the registry mode for a subject or the global default. Modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string |  |  |

---

#### `get_raw_schema_by_id`

Get the raw schema string by its global ID, without any metadata

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `id` | integer | Yes |  |

---

#### `get_raw_schema_version`

Get the raw schema string by subject name and version number, without any metadata

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `get_referenced_by`

Get schemas that reference a specific subject-version pair

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `get_registry_statistics`

Get aggregate statistics about the registry: total subjects, schemas, types breakdown, KEKs, DEKs, and exporters.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |

---

#### `get_schema_by_id`

Get a schema by its global ID, returning the full schema record including subject, version, type, and schema content

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `id` | integer | Yes |  |

---

#### `get_schema_complexity`

Compute complexity metrics and grade (A-D) for a schema. Measures field_count (total fields including nested) and max_depth (deepest nesting level via dot-notation paths). Grades: A (≤15 fields, ≤3 depth), B (≤30, ≤4), C (≤50, ≤5), D (>50 or >5). Grade D schemas should be decomposed into referenced sub-schemas.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `schema` | string |  |  |
| `schema_type` | string |  |  |
| `subject` | string |  |  |

---

#### `get_schema_history`

Get the full version history for a subject, including schema content and metadata for each version.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |

---

#### `get_schema_types`

Get the list of supported schema types (e.g. AVRO, PROTOBUF, JSON)

**Annotations:** read-only

---

#### `get_schema_version`

Get a schema by subject name and version number

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `get_schemas_by_subject`

Get all schema versions for a subject. Returns full schema records for every version, optionally including soft-deleted versions.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `deleted` | boolean |  |  |
| `subject` | string | Yes |  |

---

#### `get_server_info`

Get schema registry server information including version and supported schema types

**Annotations:** read-only

---

#### `get_server_version`

Get detailed server version information including version, commit hash, and build time.

**Annotations:** read-only

---

#### `get_subject_config_full`

Get the full configuration record for a specific subject only, without falling back to global config. Returns error if no subject-level config is set.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |

---

#### `get_subject_metadata`

Get metadata for a subject. Without filters, returns the metadata from the latest schema version. With key/value filters, searches all versions for the latest one whose metadata properties match ALL specified key/value pairs and returns a full schema record.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `deleted` | boolean |  |  |
| `metadata_filter` | object |  |  |
| `subject` | string | Yes |  |

---

#### `get_subjects_for_schema`

Get all subjects that use a specific schema ID

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `deleted` | boolean |  |  |
| `id` | integer | Yes |  |

---

#### `get_user`

Get a user by ID.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes |  |

---

#### `get_user_by_username`

Get a user by username.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `username` | string | Yes |  |

---

#### `get_versions_for_schema`

Get all subject-version pairs that use a specific schema ID

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `deleted` | boolean |  |  |
| `id` | integer | Yes |  |

---

#### `health_check`

Check if the schema registry is healthy and responding

**Annotations:** read-only

---

#### `import_schemas`

Bulk import schemas with preserved IDs (for Confluent migration). Registry mode MUST be set to IMPORT first.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `confirm_token` | string |  |  |
| `context` | string |  |  |
| `dry_run` | boolean |  |  |
| `schemas` | [null array] | Yes |  |

---

#### `list_apikeys`

List all API keys, optionally filtered by user_id.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `user_id` | integer |  |  |

---

#### `list_contexts`

List all tenant contexts in the schema registry. Each context is an isolated namespace for subjects and schemas.

**Annotations:** read-only

---

#### `list_dek_versions`

List all version numbers for a DEK subject under a given KEK.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `algorithm` | string |  |  |
| `deleted` | boolean |  |  |
| `kek_name` | string | Yes |  |
| `subject` | string | Yes |  |

---

#### `list_deks`

List all subject names that have DEKs under a given KEK.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `deleted` | boolean |  |  |
| `kek_name` | string | Yes |  |

---

#### `list_exporters`

List all exporter names. Exporters replicate schemas to a destination schema registry (Schema Linking).

**Annotations:** read-only

---

#### `list_keks`

List all Key Encryption Keys (KEKs). Use deleted=true to include soft-deleted KEKs.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `deleted` | boolean |  |  |

---

#### `list_roles`

List all available RBAC roles with their permissions.

**Annotations:** read-only

---

#### `list_schemas`

List schemas with optional filtering by subject prefix, deleted status, and pagination

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `deleted` | boolean |  |  |
| `latest_only` | boolean |  |  |
| `limit` | integer |  |  |
| `offset` | integer |  |  |
| `subject_prefix` | string |  |  |

---

#### `list_subjects`

List all registered subjects in the schema registry

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `deleted` | boolean |  |  |
| `pattern` | string |  |  |
| `prefix` | string |  |  |

---

#### `list_users`

List all users in the schema registry.

**Annotations:** read-only

---

#### `list_versions`

List all version numbers registered for a subject

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `deleted` | boolean |  |  |
| `subject` | string | Yes |  |

---

#### `lookup_schema`

Check if a schema is already registered under a subject. Returns the existing schema record if found.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `deleted` | boolean |  |  |
| `schema` | string | Yes |  |
| `schema_type` | string |  |  |
| `subject` | string | Yes |  |

---

#### `match_subjects`

Find subjects matching a pattern. Regex mode (regex=true) compiles as Go regex. Default mode uses case-sensitive substring matching.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `pattern` | string | Yes |  |
| `regex` | boolean |  |  |

---

#### `normalize_schema`

Parse and normalize a schema, returning the canonical form and fingerprint for deduplication.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `references` | [null array] |  |  |
| `schema` | string | Yes |  |
| `schema_type` | string |  |  |

---

#### `pause_exporter`

Pause a running exporter. The exporter retains its current offset and can be resumed later.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes |  |

---

#### `plan_migration_path`

Compute a multi-step migration plan from a source schema to a target schema, decomposed into individually compatible steps.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `schema_type` | string |  |  |
| `subject` | string | Yes |  |
| `target_schema` | string | Yes |  |

---

#### `register_schema`

Register a new schema version for a subject. If the same schema already exists, returns the existing record.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `metadata` | [null object] |  |  |
| `normalize` | boolean |  |  |
| `references` | [null array] |  |  |
| `rule_set` | [null object] |  |  |
| `schema` | string | Yes |  |
| `schema_type` | string |  |  |
| `subject` | string | Yes |  |

---

#### `reset_exporter`

Reset an exporter's offset back to zero, causing it to re-export all schemas.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes |  |

---

#### `resolve_alias`

Resolve a subject alias. If the subject has an alias configured, returns the alias target. Otherwise returns the original subject name. Resolution is single-level (no recursive chaining).

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `subject` | string | Yes |  |

---

#### `resume_exporter`

Resume a paused exporter. The exporter continues from its last offset.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes |  |

---

#### `revoke_apikey`

Revoke (disable) an API key without deleting it.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes |  |

---

#### `rewrap_dek`

Re-encrypt a DEK's key material under the current KEK key version. Used after KEK rotation.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `algorithm` | string |  |  |
| `kek_name` | string | Yes |  |
| `subject` | string | Yes |  |
| `version` | integer |  |  |

---

#### `rotate_apikey`

Rotate an API key: creates a new key with the same settings and revokes the old one. Returns the new raw key (only shown once). Requires id and expires_in (seconds).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `expires_in` | integer | Yes |  |
| `id` | integer | Yes |  |

---

#### `score_schema_quality`

Score a schema's quality (0-100, grades A-F) across four categories: Naming (25 pts, checks snake_case convention), Documentation (25 pts, checks field doc/description coverage), Type Safety (25 pts, penalizes generic types like string/bytes/any/object), Evolution Readiness (25 pts, checks for defaults, namespace, and schema-level docs). Returns per-category breakdown and actionable quick_wins.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `schema` | string |  |  |
| `schema_type` | string |  |  |
| `subject` | string |  |  |

---

#### `search_schemas`

Search schema content across all subjects using a regex or substring pattern.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `limit` | integer |  |  |
| `pattern` | string | Yes |  |
| `regex` | boolean |  |  |

---

#### `set_config`

Set the compatibility level for a subject or globally. Valid levels: NONE, BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `compatibility_level` | string | Yes |  |
| `context` | string |  |  |
| `normalize` | [null boolean] |  |  |
| `subject` | string |  |  |

---

#### `set_config_full`

Set the full configuration for a subject or globally, including compatibility level plus optional data contract fields: alias, compatibilityGroup, defaultMetadata, overrideMetadata, defaultRuleSet, overrideRuleSet.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `alias` | string |  |  |
| `alias_for_deks` | string |  |  |
| `compatibility_group` | string |  |  |
| `compatibility_level` | string | Yes |  |
| `compatibility_policy` | string |  |  |
| `context` | string |  |  |
| `default_metadata` | [null object] |  |  |
| `default_rule_set` | [null object] |  |  |
| `normalize` | [null boolean] |  |  |
| `override_metadata` | [null object] |  |  |
| `override_rule_set` | [null object] |  |  |
| `subject` | string |  |  |
| `validate_fields` | [null boolean] |  |  |

---

#### `set_mode`

Set the registry mode for a subject or globally. Valid modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `confirm_token` | string |  |  |
| `context` | string |  |  |
| `dry_run` | boolean |  |  |
| `force` | boolean |  |  |
| `mode` | string | Yes |  |
| `subject` | string |  |  |

---

#### `suggest_compatible_change`

Get rule-based advice for compatible schema changes based on the subject's compatibility level. BACKWARD: add fields with defaults, don't remove. FORWARD: remove fields, don't add required. FULL: only add optional fields with defaults. NONE: any change allowed.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `change_type` | string | Yes |  |
| `context` | string |  |  |
| `subject` | string | Yes |  |

---

#### `suggest_schema_evolution`

Generate concrete schema code for a compatible evolution step (add field, deprecate field, add enum symbol).

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `change_type` | string | Yes |  |
| `context` | string |  |  |
| `enum_symbol` | string |  |  |
| `field_name` | string |  |  |
| `field_type` | string |  |  |
| `subject` | string | Yes |  |

---

#### `test_kek`

Test a KEK's KMS connectivity by performing a round-trip encrypt/decrypt test. Requires a KMS provider to be configured.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `kms_key_id` | string | Yes |  |
| `kms_props` | object |  |  |
| `kms_type` | string | Yes |  |
| `name` | string | Yes |  |

---

#### `undelete_dek`

Restore a soft-deleted Data Encryption Key (DEK).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `algorithm` | string |  |  |
| `kek_name` | string | Yes |  |
| `subject` | string | Yes |  |
| `version` | integer |  |  |

---

#### `undelete_kek`

Restore a soft-deleted Key Encryption Key (KEK).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes |  |

---

#### `update_apikey`

Update an API key's name, role, or enabled status.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `enabled` | [null boolean] |  |  |
| `id` | integer | Yes |  |
| `name` | [null string] |  |  |
| `role` | [null string] |  |  |

---

#### `update_exporter`

Update an existing exporter's settings (context type, subjects, rename format, config).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `config` | object |  |  |
| `context` | string |  |  |
| `context_type` | string |  |  |
| `name` | string | Yes |  |
| `subject_rename_format` | string |  |  |
| `subjects` | [null array] |  |  |

---

#### `update_exporter_config`

Update the destination configuration of an exporter.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `config` | object | Yes |  |
| `name` | string | Yes |  |

---

#### `update_kek`

Update an existing Key Encryption Key (KEK). Only kms_props, doc, and shared can be changed.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `doc` | string |  |  |
| `kms_props` | object |  |  |
| `name` | string | Yes |  |
| `shared` | boolean |  |  |

---

#### `update_user`

Update a user's email, password, role, or enabled status.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `email` | [null string] |  |  |
| `enabled` | [null boolean] |  |  |
| `id` | integer | Yes |  |
| `password` | [null string] |  |  |
| `role` | [null string] |  |  |

---

#### `validate_schema`

Validate a schema without registering it. Returns whether the schema is valid, its fingerprint, and any parse errors.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `context` | string |  |  |
| `references` | [null array] |  |  |
| `schema` | string | Yes |  |
| `schema_type` | string |  |  |

---

#### `validate_subject_name`

Validate a subject name against a naming strategy (topic_name, record_name, or topic_record_name).

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `strategy` | string |  |  |
| `subject` | string | Yes |  |

---

## Resources

### Static Resources

| URI | Name | Description |
|-----|------|-------------|
| `schema://contexts` | `contexts` | List of all registry contexts (tenant namespaces) |
| `schema://exporters` | `exporters` | List of all schema exporter names |
| `schema://glossary/auth-and-security` | `glossary-auth-and-security` | Security model: 6 auth methods, 4 RBAC roles with permission sets, API key lifecycle, rate limiting, audit logging, and MCP permission scopes |
| `schema://glossary/best-practices` | `glossary-best-practices` | Actionable best practices for Avro, Protobuf, and JSON Schema: field naming, nullability, evolution readiness, common mistakes, and per-format guidance |
| `schema://glossary/compatibility` | `glossary-compatibility` | All 7 compatibility modes, Avro type promotions, Protobuf wire types, JSON Schema constraints, transitive semantics, and configuration resolution |
| `schema://glossary/contexts` | `glossary-contexts` | Multi-tenancy via contexts: default context, __GLOBAL, qualified subjects, URL routing, isolation guarantees, and 4-tier config/mode inheritance |
| `schema://glossary/core-concepts` | `glossary-core-concepts` | Schema registry fundamentals: what a schema registry is, subjects, versions, IDs, deduplication, modes, naming strategies, and the serialization flow |
| `schema://glossary/data-contracts` | `glossary-data-contracts` | Data contracts: metadata properties, tags, sensitive fields, rulesets (domain/migration/encoding), rule structure, 3-layer merge, and optimistic concurrency |
| `schema://glossary/design-patterns` | `glossary-design-patterns` | Common schema design patterns: event envelope, entity lifecycle, snapshot vs delta, fat vs thin events, shared types, three-phase rename, and CI/CD integration |
| `schema://glossary/encryption` | `glossary-encryption` | Client-side field level encryption (CSFLE): envelope encryption, KEK/DEK model, KMS providers, algorithms, key rotation, and rewrapping |
| `schema://glossary/error-reference` | `glossary-error-reference` | Complete error code reference: all ~30 error codes, response format, diagnostic decision tree, and per-error tool recommendations |
| `schema://glossary/exporters` | `glossary-exporters` | Schema linking via exporters: exporter model, lifecycle states (STARTING/RUNNING/PAUSED/ERROR), context types (AUTO/CUSTOM/NONE), and configuration |
| `schema://glossary/mcp-configuration` | `glossary-mcp-configuration` | MCP server configuration: all config fields, env var overrides, read-only mode, tool policy, permission scopes, presets, two-phase confirmations, and origin validation |
| `schema://glossary/migration` | `glossary-migration` | Confluent migration: step-by-step procedure, IMPORT mode, ID preservation, the import API, verification, and rollback |
| `schema://glossary/normalization-and-fingerprinting` | `glossary-normalization-and-fingerprinting` | Schema identity: fingerprinting process, per-format canonicalization rules, normalize flag, metadata identity, and deduplication scenarios |
| `schema://glossary/schema-types` | `glossary-schema-types` | Deep reference for Avro (types, logical types, aliases, canonicalization), Protobuf (proto3, well-known types, wire types), and JSON Schema (drafts, keywords, combinators) |
| `schema://glossary/storage-backends` | `glossary-storage-backends` | Storage backends: memory, PostgreSQL, MySQL, Cassandra characteristics, concurrency mechanisms, ID allocation, and choosing a backend |
| `schema://glossary/tool-selection-guide` | `glossary-tool-selection-guide` | Decision tree for choosing the right MCP tool: indexed by task category with 2-4 tools per task |
| `schema://keks` | `keks` | List of all Key Encryption Keys (KEKs) for client-side field encryption |
| `schema://mode` | `global-mode` | Global registry mode (READWRITE, READONLY, READONLY_OVERRIDE, IMPORT) |
| `schema://server/config` | `server-config` | Global compatibility level and registry mode configuration |
| `schema://server/info` | `server-info` | Schema registry server information including version, supported schema types, commit, and build time |
| `schema://status` | `server-status` | Server health status, storage connectivity, and uptime |
| `schema://subjects` | `subjects` | List of all registered subjects in the schema registry |
| `schema://types` | `schema-types` | Supported schema types (AVRO, PROTOBUF, JSON) |

### Resource Templates

| URI Template | Name | Description |
|-------------|------|-------------|
| `schema://contexts/{context}/config` | `context-config` | Global compatibility level and mode for a specific registry context |
| `schema://contexts/{context}/mode` | `context-mode` | Global registry mode for a specific registry context |
| `schema://contexts/{context}/schemas/{id}` | `context-schema-by-id` | Schema record by global ID within a specific registry context |
| `schema://contexts/{context}/schemas/{id}/subjects` | `context-schema-subjects` | All subjects that use a specific schema ID within a specific registry context |
| `schema://contexts/{context}/schemas/{id}/versions` | `context-schema-versions` | All subject-version pairs for a schema ID within a specific registry context |
| `schema://contexts/{context}/subjects` | `context-subjects` | List of subjects in a specific registry context |
| `schema://contexts/{context}/subjects/{subject}` | `context-subject-detail` | Subject details within a specific registry context |
| `schema://contexts/{context}/subjects/{subject}/config` | `context-subject-config` | Per-subject compatibility configuration within a specific registry context |
| `schema://contexts/{context}/subjects/{subject}/mode` | `context-subject-mode` | Per-subject registry mode within a specific registry context |
| `schema://contexts/{context}/subjects/{subject}/versions` | `context-subject-versions` | All version numbers for a subject within a specific registry context |
| `schema://contexts/{context}/subjects/{subject}/versions/{version}` | `context-subject-version-detail` | Schema at a specific subject version within a specific registry context |
| `schema://exporters/{name}` | `exporter-detail` | Exporter configuration and status by name |
| `schema://keks/{name}` | `kek-detail` | Key Encryption Key (KEK) details by name |
| `schema://keks/{name}/deks` | `kek-deks` | DEK subjects under a specific KEK |
| `schema://schemas/{id}` | `schema-by-id` | Schema record by global ID, including subject, version, type, and schema content |
| `schema://schemas/{id}/subjects` | `schema-subjects` | All subjects that use a specific schema ID |
| `schema://schemas/{id}/versions` | `schema-versions` | All subject-version pairs that use a specific schema ID |
| `schema://subjects/{subject}` | `subject-detail` | Subject details including latest schema version, type, and compatibility configuration |
| `schema://subjects/{subject}/config` | `subject-config` | Per-subject compatibility configuration |
| `schema://subjects/{subject}/mode` | `subject-mode` | Per-subject registry mode (READWRITE, READONLY, etc.) |
| `schema://subjects/{subject}/versions` | `subject-versions` | All version numbers registered for a subject |
| `schema://subjects/{subject}/versions/{version}` | `subject-version-detail` | Schema at a specific subject version |

---

## Prompts

| Prompt | Description | Arguments |
|--------|-------------|-----------|
| `audit-subject-history` | Review the version history and evolution of a schema subject | `subject` (required), `context` |
| `check-compatibility` | Troubleshoot schema compatibility issues and suggest fixes | `subject` (required), `context` |
| `cicd-integration` | Guide for integrating schema validation, compatibility checking, and registration into CI/CD pipelines | — |
| `compare-formats` | Help choose between Avro, Protobuf, and JSON Schema for a use case | `use_case` (required) |
| `configure-exporter` | Guide for setting up schema linking via an exporter | `exporter_type` |
| `context-management` | Guide for managing multi-tenant contexts and the 4-tier config/mode inheritance chain | — |
| `cross-cutting-change` | Workflow for making a field change across multiple schemas: find affected schemas, test, and execute safely | `field_name` (required) |
| `data-rules-deep-dive` | Comprehensive guide to data contract rules: domain, migration, and encoding rules with examples | — |
| `debug-deserialization` | Troubleshooting guide for consumer deserialization failures including wire format, schema ID extraction, and common causes | — |
| `debug-registration-error` | Debug schema registration failures by error code | `error_code` (required) |
| `deprecate-subject` | Workflow for safely deprecating and removing a schema subject with dependency checks, locking, and cleanup | `subject` (required), `context` |
| `design-schema` | Guide for designing a new schema in the chosen format | `format` (required), `domain` |
| `evolve-schema` | Guide for safely evolving an existing schema with backward compatibility | `subject` (required), `context` |
| `full-encryption-lifecycle` | End-to-end CSFLE workflow: KEK creation, DEK management, key rotation, rewrapping, and cleanup | — |
| `glossary-lookup` | Look up a schema registry concept and get directed to the relevant glossary resource | `topic` (required) |
| `governance-setup` | Guide for setting up schema governance: naming conventions, quality gates, data contracts, RBAC, and audit | — |
| `import-from-confluent` | Step-by-step guide for migrating schemas from Confluent Schema Registry with ID preservation | — |
| `migrate-schemas` | Guide for migrating schemas between formats (e.g. Avro to Protobuf) | `source_format` (required), `target_format` (required) |
| `new-kafka-topic` | End-to-end workflow for setting up key and value schemas for a new Kafka topic | `topic_name` (required), `format` |
| `plan-breaking-change` | Plan a safe breaking schema change with migration strategy | `subject` (required), `context` |
| `registry-health-audit` | Multi-step procedure for auditing registry health, configuration consistency, and schema quality | — |
| `review-schema-quality` | Analyze a schema for naming conventions, nullability, documentation, and best practices | `subject` (required), `context` |
| `schema-evolution-cookbook` | Practical recipes for common schema evolution scenarios: add fields, rename, change types, and break compatibility safely | — |
| `schema-getting-started` | Quick-start guide introducing available tools and common schema registry operations | — |
| `schema-impact-analysis` | Guided workflow for assessing the impact of a proposed schema change across dependents | `subject` (required), `context` |
| `schema-naming-conventions` | Guide to subject naming strategies (topic_name, record_name, topic_record_name) | — |
| `schema-references-guide` | Guide for cross-subject schema references with per-format name semantics (Avro, Protobuf, JSON Schema) | — |
| `schema-review-checklist` | Pre-registration checklist: syntax, compatibility, quality, naming, uniqueness, dependencies, and impact | `subject` (required), `context` |
| `setup-data-contracts` | Guide for adding metadata, tags, and data quality rules to schemas | `subject` (required), `context` |
| `setup-encryption` | Guide for setting up client-side field encryption with KEK/DEK | `kms_type` (required) |
| `setup-rbac` | Guide for configuring authentication and role-based access control (RBAC) | — |
| `team-onboarding` | Workflow for onboarding a new team with context creation, schema registration, RBAC setup, and naming conventions | `team_name` (required) |
| `troubleshooting` | Diagnostic guide for common schema registry issues and errors | — |

### Prompt Details

#### `audit-subject-history`

Review the version history and evolution of a schema subject

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to audit |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Audit the version history of subject example-subject.

Steps:
1. Use list_versions to get all version numbers for example-subject
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


</details>

---

#### `check-compatibility`

Troubleshoot schema compatibility issues and suggest fixes

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to check compatibility for |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Troubleshoot compatibility issues for subject example-subject.

Steps:
1. Use get_config to check the current compatibility level for example-subject
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


Current compatibility level: &{ BACKWARD <nil> <nil>   <nil> <nil> <nil> <nil>  }

</details>

---

#### `cicd-integration`

Guide for integrating schema validation, compatibility checking, and registration into CI/CD pipelines

<details>
<summary>Prompt content (click to expand)</summary>

Guide for integrating schema registry validation into CI/CD pipelines.

## Pre-Commit Checks

Run these before a schema change is merged:

### Syntax Validation
```
validate_schema(schema: <proposed_schema>, schema_type: "AVRO")
```
Catches malformed JSON, missing required fields, invalid types.

### Compatibility Check
```
check_compatibility(subject: "<subject>", schema: <proposed_schema>, schema_type: "AVRO")
```
Catches backward-incompatible changes (removed fields, changed types).

### Quality Gate
```
score_schema_quality(schema: <proposed_schema>, schema_type: "AVRO")
```
Enforce minimum quality scores: naming conventions, documentation, nullability patterns.

### Field Consistency
```
check_field_consistency(field_name: "customer_id", schema_type: "AVRO")
```
Ensure shared fields use consistent types across all schemas.

### Subject Name Validation
```
validate_subject_name(subject: "<subject>", strategy: "topic_name")
```
Enforce naming conventions.

## Deployment-Time Registration

When deploying a service that produces/consumes schemas:

### Register Schema
```
register_schema(subject: "<subject>", schema: <schema>, schema_type: "AVRO")
```

### Verify Registration
```
get_latest_schema(subject: "<subject>")
```
Confirm the registered version matches expectations.

### Compare with Previous
```
diff_schemas(subject: "<subject>", version1: "latest", version2: <previous_version>)
```
Log the diff for audit trail.

## Rollback Strategy

If a schema registration causes issues:

1. Do NOT delete the registered schema (consumers may already be using it)
2. Register the previous schema version as a new version (if compatible)
3. Or use `set_mode(mode: "READONLY")` to prevent further registrations while investigating

## Pipeline Architecture

Use **contexts** for environment separation:

```
# PR validation (staging context)
validate_schema(schema: <schema>)
check_compatibility(subject: "orders-value", schema: <schema>, context: ".staging")

# Merge to main (production context)
register_schema(subject: "orders-value", schema: <schema>, context: ".production")
```

## Pseudo-Code Example

```
# In CI pipeline (e.g., GitHub Actions, Jenkins)

# 1. Validate syntax
result = mcp_call("validate_schema", {schema: new_schema, schema_type: "AVRO"})
assert result.valid == true

# 2. Check compatibility
result = mcp_call("check_compatibility", {subject: subject, schema: new_schema})
assert result.is_compatible == true

# 3. Quality gate
result = mcp_call("score_schema_quality", {schema: new_schema, schema_type: "AVRO"})
assert result.overall_score >= 70

# 4. Register on deploy
result = mcp_call("register_schema", {subject: subject, schema: new_schema})
schema_id = result.id
```

Available tools: validate_schema, check_compatibility, score_schema_quality, check_field_consistency, validate_subject_name, register_schema, get_latest_schema, diff_schemas, set_mode


</details>

---

#### `compare-formats`

Help choose between Avro, Protobuf, and JSON Schema for a use case

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `use_case` | Yes | Use case description (e.g. event streaming, REST API, RPC) |

<details>
<summary>Prompt content (click to expand)</summary>

Compare Avro, Protobuf, and JSON Schema for the use case: event streaming

## Format Comparison

| Feature | Avro | Protobuf | JSON Schema |
|---------|------|----------|-------------|
| Serialization | Binary (compact) | Binary (compact) | Text (JSON) |
| Schema evolution | Excellent | Good | Limited |
| Type system | Rich (unions, logical types) | Strong (oneof, well-known types) | Flexible (oneOf, anyOf) |
| Code generation | Moderate | Excellent | Minimal |
| Human readability | Schema: JSON, Data: binary | Schema: .proto, Data: binary | Both: JSON |
| Kafka integration | Native | Supported | Supported |
| gRPC support | Limited | Native | Not applicable |
| Validation | Schema-level | Schema-level | Rich constraints |

## Recommendations by use case

**Event streaming (Kafka):** Avro
- Best schema evolution support with BACKWARD/FORWARD compatibility
- Compact binary serialization reduces Kafka storage/bandwidth
- Native Kafka ecosystem integration

**RPC/Microservices:** Protobuf
- Native gRPC support with code generation
- Strong typing across languages
- Efficient binary serialization

**REST APIs:** JSON Schema
- Human-readable request/response validation
- Rich constraint validation (patterns, ranges, formats)
- Direct JSON compatibility

**Mixed/CQRS systems:** Use multiple formats
- Avro for events (Kafka topics)
- Protobuf for commands (gRPC)
- JSON Schema for queries (REST responses)

Available tools: register_schema, get_schema_types


</details>

---

#### `configure-exporter`

Guide for setting up schema linking via an exporter

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `exporter_type` |  | Exporter context type: AUTO, CUSTOM, or NONE |

<details>
<summary>Prompt content (click to expand)</summary>

Set up schema linking with a AUTO context exporter.

Steps:
1. Create an exporter using the create_exporter tool:
   - name: descriptive name (e.g. "prod-to-dr")
   - context_type: AUTO
   - subjects: list of subjects to export (empty = all)
   - config: destination registry connection details

2. Monitor the exporter using get_exporter_status
3. Control the exporter: pause_exporter, resume_exporter, reset_exporter

Context types:
- AUTO: exports all subjects automatically
- CUSTOM: exports only specified subjects with optional rename format
- NONE: no context prefix on exported subjects

Config properties:
- schema.registry.url: destination registry URL
- basic.auth.credentials.source: auth method
- basic.auth.user.info: username:password

Available tools: create_exporter, get_exporter, list_exporters, get_exporter_status, pause_exporter, resume_exporter


</details>

---

#### `context-management`

Guide for managing multi-tenant contexts and the 4-tier config/mode inheritance chain

<details>
<summary>Prompt content (click to expand)</summary>

Guide for managing multi-tenant contexts in the schema registry.

## What are contexts?
Contexts are tenant namespaces that isolate schemas, subjects, and configuration. The default context is "." (dot). Contexts enable multi-tenancy — different teams, environments, or applications can have independent schema registries within the same server.

## Listing and navigating contexts
- **list_contexts** — list all available contexts
- **list_subjects** — lists subjects in the default context
- Subjects can be qualified with context: `:.staging:my-subject`

## The 4-tier config/mode inheritance chain
Configuration and mode settings cascade through 4 levels (highest to lowest precedence):

1. **Per-subject** (highest precedence) -- most specific, overrides everything below
2. **Context global** -- per-context default, overrides __GLOBAL and server default
3. **Global (__GLOBAL)** -- cross-context default, set via set_config/set_mode with no subject
4. **Server default** (lowest precedence) -- hardcoded BACKWARD compatibility, READWRITE mode

To check effective config: **get_config** with a subject name returns the resolved value.
To check effective mode: **get_mode** with a subject name returns the resolved value.

## Managing configuration per context
- **set_config** — set compatibility level (per-subject or global)
- **delete_config** — remove per-subject config (falls back to context global)
- **set_mode** — set mode (READWRITE, READONLY, READONLY_OVERRIDE, IMPORT)
- **delete_mode** — remove per-subject mode (falls back to context global)

## Import and migration
- Use **set_mode** with mode IMPORT to enable ID-preserving schema import
- Use **import_schemas** to bulk import schemas with preserved IDs
- Reset mode after import: **set_mode** with mode READWRITE

## Context-Scoped Resources (11 URI templates)

| URI Template | Description |
|-------------|-------------|
| `schema://contexts/{context}/subjects` | Subjects in a specific context |
| `schema://contexts/{context}/config` | Global config for a context |
| `schema://contexts/{context}/mode` | Global mode for a context |
| `schema://contexts/{context}/subjects/{subject}` | Subject details within a context |
| `schema://contexts/{context}/subjects/{subject}/versions` | Subject versions |
| `schema://contexts/{context}/subjects/{subject}/versions/{version}` | Version detail |
| `schema://contexts/{context}/subjects/{subject}/config` | Subject config |
| `schema://contexts/{context}/subjects/{subject}/mode` | Subject mode |
| `schema://contexts/{context}/schemas/{id}` | Schema by ID |
| `schema://contexts/{context}/schemas/{id}/subjects` | Schema subjects |
| `schema://contexts/{context}/schemas/{id}/versions` | Schema versions |

Non-context-prefixed URIs (e.g., `schema://subjects`) use the default context.

## Working with Contexts via MCP

78+ tools accept the optional `context` parameter for multi-tenant isolation:
```json
{"subject": "orders-value", "context": ".staging"}
```

7 prompts accept a `context` argument: evolve-schema, check-compatibility, review-schema-quality, plan-breaking-change, setup-data-contracts, audit-subject-history, schema-impact-analysis.

**Typical workflow:**
1. **list_contexts** -- see available contexts
2. **list_subjects** with `context` -- browse context subjects
3. **register_schema** with `context` -- register in context
4. **get_config** with `context` -- check context config
5. Browse via `schema://contexts/{context}/subjects` resource

Available tools: list_contexts, get_config, set_config, delete_config, get_mode, set_mode, delete_mode, import_schemas, list_subjects, register_schema


</details>

---

#### `cross-cutting-change`

Workflow for making a field change across multiple schemas: find affected schemas, test, and execute safely

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `field_name` | Yes | Field name to change across schemas |

<details>
<summary>Prompt content (click to expand)</summary>

Workflow for making a cross-cutting field change across multiple schemas.

## Step 1: Determine Scope

If the change spans multiple contexts:
```
list_contexts()
```

## Step 2: Find All Affected Schemas

Find every schema containing the field "customer_id":
```
find_schemas_by_field(field_name: "customer_id")
```

This returns all subjects and versions where the field appears.

## Step 3: Check Current Consistency

Verify the field's type is consistent across all schemas:
```
check_field_consistency(field_name: "customer_id")
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
check_field_consistency(field_name: "customer_id")
```

Confirm the field is now consistent across all schemas.

Available tools: list_contexts, find_schemas_by_field, check_field_consistency, get_latest_schema, get_config, check_compatibility, check_compatibility_multi, explain_compatibility_failure, suggest_compatible_change, get_dependency_graph, diff_schemas


</details>

---

#### `data-rules-deep-dive`

Comprehensive guide to data contract rules: domain, migration, and encoding rules with examples

<details>
<summary>Prompt content (click to expand)</summary>

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


</details>

---

#### `debug-deserialization`

Troubleshooting guide for consumer deserialization failures including wire format, schema ID extraction, and common causes

<details>
<summary>Prompt content (click to expand)</summary>

Troubleshooting guide for consumer deserialization failures.

## Kafka Wire Format

Kafka messages with schema registry use a 5-byte prefix:

```
[0x00] [4-byte schema ID (big-endian)] [serialized payload]
```

- Byte 0: Magic byte (always 0x00)
- Bytes 1-4: Schema ID as a 32-bit big-endian integer
- Remaining bytes: Avro/Protobuf/JSON-encoded payload

## Diagnostic Workflow

### Step 1: Extract the Schema ID

From the raw message bytes, extract bytes 1-4 as a big-endian integer.
If the first byte is not 0x00, the message was not serialized with the schema registry ("Unknown magic byte" error).

### Step 2: Fetch the Producer's Schema

```
get_schema_by_id(id: <extracted_id>)
```

This returns the schema the producer used to serialize the message.

### Step 3: Fetch the Consumer's Schema

```
get_latest_schema(subject: "<consumer-subject>")
```

This returns the schema the consumer is using to deserialize.

### Step 4: Compare Schemas

```
diff_schemas(subject: "<subject>", version1: <producer_version>, version2: <consumer_version>)
```

### Step 5: Check Compatibility

```
explain_compatibility_failure(subject: "<subject>", schema: <producer_schema>)
```

## Common Causes

### "Unknown magic byte"
- Message was not serialized with the schema registry
- Wrong deserializer configured (using Avro deserializer on plain JSON, etc.)
- Message was produced before schema registry was introduced
- **Fix:** Check producer serializer configuration

### "Schema not found" (ID not in registry)
- Consumer is pointing to a different registry instance than the producer
- Schema was permanently deleted from the registry
- **Fix:** Check `schema.registry.url` configuration on both producer and consumer

### "Could not deserialize" (schema mismatch)
- Producer schema evolved in a way the consumer's local schema cannot handle
- **Fix:** Use diff_schemas to see what changed, then update the consumer

### Wrong subject or naming strategy
- Producer uses TopicNameStrategy but consumer uses RecordNameStrategy
- **Fix:** Ensure both use the same SubjectNameStrategy

### Old consumer with missing fields
- A new required field was added without a default value
- **Fix:** The schema change was backward-incompatible; add a default value

## Tools Reference

- **get_schema_by_id** -- fetch schema by the ID from the message
- **get_latest_schema** -- fetch the consumer's expected schema
- **diff_schemas** -- compare producer and consumer schemas
- **explain_compatibility_failure** -- understand why schemas are incompatible
- **get_subjects_for_schema** -- find which subjects use a schema ID
- **list_versions** -- see all versions of a subject


</details>

---

#### `debug-registration-error`

Debug schema registration failures by error code

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `error_code` | Yes | Error code from failed registration (e.g. 42201, 409, 40401) |

<details>
<summary>Prompt content (click to expand)</summary>

Error 42201: Invalid schema

The schema failed validation. Common causes:
- Malformed JSON (check brackets, quotes, commas)
- Invalid Avro schema (missing type, name, or fields)
- Invalid Protobuf syntax (missing syntax declaration, package, or field numbers)
- Invalid JSON Schema (unsupported keywords or types)

Debug steps:
1. Use validate_schema to get a detailed error message
2. For Avro: ensure "type", "name", and "fields" are present for records
3. For Protobuf: ensure 'syntax = "proto3";' is the first line
4. For JSON Schema: ensure "type" is a valid JSON Schema type
5. Check for escape character issues in the schema string
6. Check for malformed JSON (missing brackets, quotes, commas)

</details>

---

#### `deprecate-subject`

Workflow for safely deprecating and removing a schema subject with dependency checks, locking, and cleanup

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to deprecate |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Workflow for safely deprecating and removing a schema subject.

## Step 1: Assess Current Usage

```
get_latest_schema(subject: "example-subject")
count_versions(subject: "example-subject")
get_referenced_by(subject: "example-subject")
```

Check if other schemas reference this subject. If so, those must be migrated first.

## Step 2: Check Dependents

```
get_dependency_graph(subject: "example-subject")
```

If the graph shows downstream dependents, migrate them to an alternative schema before proceeding.

## Step 3: Notify Consumers

Before making any changes, ensure all consumers and producers using this subject are aware of the deprecation.

## Step 4: Lock the Subject (READONLY)

Prevent new versions from being registered:
```
set_mode(subject: "example-subject", mode: "READONLY")
```

Verify writes are blocked:
```
check_write_mode(subject: "example-subject")
```

## Step 5: Add Deprecation Metadata

Mark the subject as deprecated using metadata:
```
set_config_full(subject: "example-subject", override_metadata: {"properties": {"deprecated": "true", "deprecation_date": "2026-03-06", "replacement": "<new-subject>"}})
```

## Step 6: Migration Period

Allow time for all consumers to migrate. Monitor usage via audit logs.

## Step 7: Soft-Delete

Hide the subject from listings but keep data accessible by ID:
```
delete_subject(subject: "example-subject")
```

The subject is now hidden from `list_subjects` but schemas are still resolvable by ID (important for in-flight Kafka messages).

## Step 8: Permanent Delete (Optional)

After confirming all consumers have migrated:
```
delete_subject(subject: "example-subject", permanent: true)
```

> Warning: This is irreversible. Kafka messages referencing these schema IDs will become undeserializable.

## Step 9: Clean Up DEKs (if encrypted)

If the subject had client-side encryption:
```
list_deks(kek_name: "<kek>", subject: "example-subject")
delete_dek(kek_name: "<kek>", subject: "example-subject")
```

## Context Support

All tools accept the optional `context` parameter. Ensure you pass it consistently if the subject is in a non-default context.

Available tools: get_latest_schema, count_versions, get_referenced_by, get_dependency_graph, set_mode, check_write_mode, set_config_full, delete_subject, list_deks, delete_dek


</details>

---

#### `design-schema`

Guide for designing a new schema in the chosen format

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `format` | Yes | Schema format: AVRO, PROTOBUF, or JSON |
| `domain` |  | Domain or topic for the schema (e.g. user-events, payments) |

<details>
<summary>Prompt content (click to expand)</summary>

Design a AVRO schema for the "example-events" domain.

Design an Avro schema following these best practices.

## Design Workflow

1. **Identify the entity or event** -- what does this schema represent?
2. **Choose a namespace** -- use reverse-domain (e.g., `com.company.events`)
3. **Define fields** -- name each field in snake_case, choose appropriate types
4. **Add defaults** -- every new field SHOULD have a default for backward compatibility
5. **Add documentation** -- use the `doc` property on the record and each field
6. **Validate** -- use **validate_schema** to check syntax
7. **Register** -- use **register_schema** with `schema_type: AVRO`

> All registration tools accept the optional `context` parameter for multi-tenant isolation.

## Best Practices

- Use a descriptive record name in PascalCase with a namespace (e.g., `com.company.events.OrderCreated`)
- Use `snake_case` for field names
- Use union types `["null", "type"]` with `"default": null` for optional fields
- Use logical types for dates (`timestamp-millis`), decimals (`bytes` + `decimal`), and UUIDs (`string` + `uuid`)
- Use enums for fixed sets of values (with a sensible default)
- Consider schema evolution: add new fields with defaults, avoid removing or renaming fields
- Keep records focused -- prefer composition over deeply nested structures

## Worked Example: OrderCreated Event

```json
{
  "type": "record",
  "name": "OrderCreated",
  "namespace": "com.company.events",
  "doc": "Emitted when a customer places a new order.",
  "fields": [
    {"name": "event_id", "type": {"type": "string", "logicalType": "uuid"}, "doc": "Unique event identifier"},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}, "doc": "Event timestamp in UTC"},
    {"name": "order_id", "type": "string", "doc": "Business order identifier"},
    {"name": "customer_id", "type": "string", "doc": "Customer who placed the order"},
    {"name": "status", "type": {"type": "enum", "name": "OrderStatus", "symbols": ["PENDING", "CONFIRMED", "SHIPPED", "DELIVERED", "CANCELLED"]}, "doc": "Current order status"},
    {"name": "total_cents", "type": "long", "doc": "Order total in smallest currency unit"},
    {"name": "currency", "type": {"type": "string", "default": "USD"}, "doc": "ISO 4217 currency code", "default": "USD"},
    {"name": "shipping_address", "type": ["null", {
      "type": "record", "name": "Address", "fields": [
        {"name": "street", "type": "string"},
        {"name": "city", "type": "string"},
        {"name": "postal_code", "type": "string"},
        {"name": "country", "type": "string"}
      ]
    }], "default": null, "doc": "Shipping address, null for digital orders"},
    {"name": "notes", "type": ["null", "string"], "default": null, "doc": "Optional order notes"}
  ]
}
```

## Common Mistakes

1. **Missing defaults on optional fields** -- `["null", "string"]` without `"default": null` breaks backward compatibility
2. **Using `string` for everything** -- use typed fields (`int`, `long`, `boolean`, enums) for correctness and efficiency
3. **No namespace** -- leads to naming conflicts when schemas reference each other
4. **Changing enum symbol order** -- Avro enums are ordinal; reordering is a breaking change
5. **Deeply nested unions** -- union-within-union is not allowed in Avro; flatten or use separate records

## Starter Template

```json
{
  "type": "record",
  "name": "MyEvent",
  "namespace": "com.company.events",
  "doc": "Description of the event",
  "fields": [
    {"name": "event_id", "type": {"type": "string", "logicalType": "uuid"}},
    {"name": "timestamp", "type": {"type": "long", "logicalType": "timestamp-millis"}},
    {"name": "my_field", "type": "string", "doc": "Description"}
  ]
}
```

Available tools: register_schema, validate_schema, check_compatibility, get_latest_schema, lookup_schema


</details>

---

#### `evolve-schema`

Guide for safely evolving an existing schema with backward compatibility

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name of the schema to evolve |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Evolve the schema for subject example-subject safely.

Steps:
1. Use get_latest_schema to inspect the current schema for example-subject
2. Use get_config to check the compatibility level
3. Plan your changes following the compatibility rules:
   - BACKWARD: new schema can read old data (add optional fields with defaults)
   - FORWARD: old schema can read new data (only remove optional fields)
   - FULL: both backward and forward compatible
4. Use check_compatibility to validate your changes before registering
5. Use register_schema to register the evolved schema

Common safe changes:
- Add a new optional field with a default value
- Add a new field with a union type ["null", "type"] and default null
- Widen a type (e.g. int → long in Avro)

Breaking changes to avoid:
- Removing a required field
- Changing a field type incompatibly
- Renaming a field (treated as remove + add)


</details>

---

#### `full-encryption-lifecycle`

End-to-end CSFLE workflow: KEK creation, DEK management, key rotation, rewrapping, and cleanup

<details>
<summary>Prompt content (click to expand)</summary>

End-to-end CSFLE (Client-Side Field Level Encryption) lifecycle.

## Phase 1: Create a KEK
A KEK references an external KMS key. It wraps (encrypts) the DEKs.

Use **create_kek**:
    name: "prod-kek"
    kms_type: "hcvault" (or aws-kms, azure-kms, gcp-kms, openbao)
    kms_key_id: transit key name or ARN
    kms_props: provider-specific connection details
    shared: false (recommended for client-managed keys)

Verify with **test_kek** to confirm KMS connectivity.

## Phase 2: Create DEKs
A DEK is the actual encryption key, scoped to a schema subject.

Use **create_dek**:
    kek_name: "prod-kek"
    subject: "orders-value"
    algorithm: "AES256_GCM" (or AES128_GCM, AES256_SIV)

The DEK is automatically wrapped by the KEK. The plaintext key material stays on the client.

## Phase 3: Key Rotation
Create a new DEK version for the same subject:

Use **create_dek** again with the same kek_name and subject. A new version is auto-assigned.
- New messages are encrypted with the latest DEK version.
- Old messages remain decryptable with previous DEK versions.

## Phase 4: KMS Key Rotation (Rewrap)
When the underlying KMS key is rotated:

Rewrap existing DEKs so they are encrypted with the new KMS key version. The DEK plaintext stays the same -- only the wrapper changes. No re-encryption of data is needed.

## Phase 5: Cleanup
Soft-delete old DEK versions that are no longer needed:
Use **delete_dek** -- sets deleted=true. Can be undone with undelete_dek.

Permanent delete (irreversible):
Use **delete_dek** with permanent: true.

Delete a KEK only after ALL its DEKs are permanently deleted.

## Algorithm Choice
- **AES256_GCM** (default): strongest confidentiality, non-deterministic
- **AES128_GCM**: lower key size, still authenticated
- **AES256_SIV**: deterministic -- enables equality searches but leaks value equality

## MCP Tools
create_kek, get_kek, list_keks, update_kek, delete_kek, test_kek,
create_dek, get_dek, list_deks, delete_dek, get_dek_versions

For domain knowledge, read: schema://glossary/encryption


</details>

---

#### `glossary-lookup`

Look up a schema registry concept and get directed to the relevant glossary resource

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `topic` | Yes | Keyword or concept to look up (e.g. compatibility, CSFLE, contexts, avro) |

<details>
<summary>Prompt content (click to expand)</summary>

The topic compatibility is covered in the glossary resource: schema://glossary/compatibility

Read that resource to get comprehensive domain knowledge, then answer the user's question.

All glossary resources:
- schema://glossary/core-concepts     -- subjects, versions, IDs, modes, naming
- schema://glossary/compatibility     -- 7 modes, per-format rules, transitive semantics
- schema://glossary/data-contracts    -- metadata, tags, rulesets, 3-layer merge
- schema://glossary/encryption        -- CSFLE, KEK/DEK, KMS providers, algorithms
- schema://glossary/contexts          -- multi-tenancy, 4-tier inheritance
- schema://glossary/exporters         -- schema linking, lifecycle states
- schema://glossary/schema-types      -- Avro, Protobuf, JSON Schema deep reference
- schema://glossary/design-patterns   -- event envelope, lifecycle, shared types, CI/CD
- schema://glossary/best-practices    -- per-format guidance, common mistakes
- schema://glossary/migration         -- Confluent migration, IMPORT mode, ID preservation


</details>

---

#### `governance-setup`

Guide for setting up schema governance: naming conventions, quality gates, data contracts, RBAC, and audit

<details>
<summary>Prompt content (click to expand)</summary>

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


</details>

---

#### `import-from-confluent`

Step-by-step guide for migrating schemas from Confluent Schema Registry with ID preservation

<details>
<summary>Prompt content (click to expand)</summary>

Step-by-step guide for migrating schemas from Confluent Schema Registry.

## Why This Matters
The Kafka wire format embeds a 4-byte schema ID in every message. If IDs change, existing consumers cannot deserialize messages. This procedure preserves exact schema IDs.

## Prerequisites
- Source: running Confluent Schema Registry with network access
- Target: running AxonOps Schema Registry with configured storage
- Tools: curl and jq on the migration machine

## Procedure

### Step 1: Verify target health
Use **health_check** to confirm the target registry is running.

### Step 2: Set IMPORT mode
Use **set_mode** with mode: IMPORT. This allows registering schemas with specific IDs and bypasses compatibility checks.

### Step 3: Export from Confluent
Run the migration script:
    ./scripts/migrate-from-confluent.sh --source http://confluent:8081 --dry-run
Inspect the output to verify schema count and subjects.

### Step 4: Import
    ./scripts/migrate-from-confluent.sh --source http://confluent:8081 --target http://axonops:8082 --verify
Or use the **import_schemas** tool for programmatic import.

### Step 5: Switch to READWRITE
Use **set_mode** with mode: READWRITE. New registrations will get auto-generated IDs starting after the highest imported ID.

### Step 6: Update clients
Change schema.registry.url in all Kafka serializer/deserializer configs. No code changes needed -- the API is wire-compatible.

## Verification
1. Use **list_subjects** -- count should match source.
2. Use **get_schema_by_id** -- spot-check IDs across both registries.
3. Produce/consume a test message through the new registry.

## Rollback
The migration is non-destructive. Point clients back to Confluent if issues arise.

For domain knowledge, read: schema://glossary/migration


</details>

---

#### `migrate-schemas`

Guide for migrating schemas between formats (e.g. Avro to Protobuf)

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `source_format` | Yes | Source schema format (AVRO, PROTOBUF, JSON) |
| `target_format` | Yes | Target schema format (AVRO, PROTOBUF, JSON) |

<details>
<summary>Prompt content (click to expand)</summary>

Migrate schemas from AVRO to PROTOBUF format.

## Workflow

1. Use **list_subjects** to find schemas to migrate
2. Use **get_latest_schema** for each subject to inspect the current schema
3. Convert the schema to PROTOBUF format using the mapping tables below
4. Use **validate_schema** with `schema_type: PROTOBUF` to check syntax
5. Use **register_schema** with `schema_type: PROTOBUF` on a NEW subject (e.g., `orders-value-proto`)
6. Use **check_compatibility** if the new subject already existed
7. Update serializer/deserializer configs with the new schema IDs

> **Important:** Schema IDs change when you register under a new subject. Update all producer/consumer configurations.
> All tools accept the optional `context` parameter for multi-tenant isolation.
> Use the **compare-formats** prompt if you need help choosing a target format.

---

## Type Mapping: Avro to Protobuf

| Avro | Protobuf |
|------|----------|
| `int` | `int32` |
| `long` | `int64` |
| `float` | `float` |
| `double` | `double` |
| `string` | `string` |
| `bytes` | `bytes` |
| `boolean` | `bool` |
| `null` | N/A (all proto3 fields have zero defaults) |
| `fixed` | `bytes` (fixed size not preserved) |
| record | message |
| enum | enum (prepend `UNSPECIFIED = 0` as first value) |
| array | `repeated` field |
| map | `map<string, V>` |
| union `["null", "type"]` | `optional type` field |
| union with multiple non-null types | `oneof` |
| namespace | `package` |
| doc | `//` comment |

**Logical type mapping:**

| Avro Logical Type | Protobuf |
|-------------------|----------|
| `timestamp-millis` / `timestamp-micros` | `google.protobuf.Timestamp` |
| `decimal` | `string` or custom message (precision lost) |
| `uuid` | `string` |
| `date` | `int32` (days since epoch) |
| `time-millis` | `int32` |
| `duration` | `google.protobuf.Duration` |

**What is lost:**
- Default values have no direct proto3 equivalent (proto3 uses zero values only)
- Avro `doc` fields become `//` comments (no structured doc in proto3)
- Avro `aliases` are dropped entirely (no proto3 equivalent)
- Avro named type references become Protobuf `import` paths
- Proto3 requires explicit field numbers — assign sequentially starting from 1

---

## Type Mapping: Protobuf to Avro

| Protobuf | Avro |
|----------|------|
| `int32` / `sint32` / `sfixed32` | `int` |
| `int64` / `sint64` / `sfixed64` | `long` |
| `uint32` / `fixed32` | `int` (unsigned-to-signed — warn if values may exceed 2^31) |
| `uint64` / `fixed64` | `long` (unsigned-to-signed — warn if values may exceed 2^63) |
| `float` | `float` |
| `double` | `double` |
| `string` | `string` |
| `bytes` | `bytes` |
| `bool` | `boolean` |
| message | record |
| enum | enum (drop `UNSPECIFIED` value, set default to first remaining) |
| `repeated` | array |
| `map<string, V>` | map |
| `map<K, V>` (non-string key) | **not directly supported** — Avro maps are always string-keyed (flatten or restructure) |
| `optional` | union `["null", "type"]` with default `null` |
| `oneof` | union |
| nested message | nested record |
| `package` | namespace |
| `google.protobuf.Timestamp` | `long` with logical type `timestamp-millis` |
| `google.protobuf.Duration` | `long` (milliseconds) or `fixed` (custom) |

**What is lost:**
- Field numbers are dropped (Avro uses field names on the wire)
- `reserved` fields and numbers have no Avro equivalent
- Proto3 `map<int, V>` or `map<bool, V>` requires restructuring — Avro maps only support string keys
- Service definitions and RPC methods have no Avro equivalent
- `uint32`/`uint64` are silently narrowed to signed types

---

## Type Mapping: Avro to JSON Schema

| Avro | JSON Schema |
|------|-------------|
| record | `{"type": "object", "properties": {...}}` |
| enum | `{"type": "string", "enum": [...]}` |
| array | `{"type": "array", "items": {...}}` |
| map | `{"type": "object", "additionalProperties": {...}}` |
| union `["null", "type"]` | property NOT in `required` array |
| `int` / `long` | `{"type": "integer"}` |
| `float` / `double` | `{"type": "number"}` |
| `string` | `{"type": "string"}` |
| `boolean` | `{"type": "boolean"}` |
| `bytes` | `{"type": "string", "contentEncoding": "base64"}` |
| `fixed` | `{"type": "string", "contentEncoding": "base64"}` |
| `null` | `{"type": "null"}` |

**Logical type mapping:**

| Avro | JSON Schema |
|------|-------------|
| `timestamp-millis` | `{"type": "string", "format": "date-time"}` |
| `uuid` | `{"type": "string", "format": "uuid"}` |
| `date` | `{"type": "string", "format": "date"}` |
| `time-millis` | `{"type": "string", "format": "time"}` |
| `decimal` | `{"type": "string"}` or `{"type": "number"}` |

**What is lost:**
- Avro namespace has no JSON Schema equivalent (use `$id` for naming)
- Avro `aliases` are dropped
- `default` maps to `default` keyword (preserved)
- `doc` maps to `description` keyword (preserved)

---

## Type Mapping: JSON Schema to Avro

| JSON Schema | Avro |
|-------------|------|
| `required` properties | fields without union null |
| Optional properties (not in `required`) | union `["null", "type"]` with default `null` |
| `additionalProperties` | map |
| `oneOf` / `anyOf` | union |
| `enum` | enum |
| `$ref` | named type reference |
| `{"type": "object"}` | record |
| `{"type": "array"}` | array |
| `{"type": "integer"}` | `long` (safest default) |
| `{"type": "number"}` | `double` (safest default) |
| `{"type": "string"}` | `string` |
| `{"type": "boolean"}` | `boolean` |
| `{"type": "string", "format": "date-time"}` | `long` with logical type `timestamp-millis` |
| `{"type": "string", "format": "uuid"}` | `string` with logical type `uuid` |
| `{"type": "string", "format": "date"}` | `int` with logical type `date` |

**What is lost:**
- Validation constraints: `pattern`, `minLength`, `maxLength`, `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`, `multipleOf`
- Conditional logic: `if`/`then`/`else` has no Avro equivalent
- `allOf` composition must be flattened manually into a single record
- `not` has no Avro equivalent
- `$ref` maps to Avro named type references — register referenced schemas first

---

## Type Mapping: Protobuf to JSON Schema

| Protobuf | JSON Schema |
|----------|-------------|
| message | `{"type": "object", "properties": {...}}` |
| enum | `{"type": "string", "enum": [...]}` (use string value names) |
| `int32` / `sint32` / `sfixed32` / `uint32` / `fixed32` | `{"type": "integer"}` |
| `int64` / `sint64` / `sfixed64` / `uint64` / `fixed64` | `{"type": "integer"}` |
| `float` / `double` | `{"type": "number"}` |
| `string` | `{"type": "string"}` |
| `bytes` | `{"type": "string", "contentEncoding": "base64"}` |
| `bool` | `{"type": "boolean"}` |
| `repeated` | `{"type": "array", "items": {...}}` |
| `map<K, V>` | `{"type": "object", "additionalProperties": {...}}` |
| `oneof` | `{"oneOf": [...]}` |
| `optional` | property NOT in `required` array |
| nested message | nested `{"type": "object"}` |
| `google.protobuf.Timestamp` | `{"type": "string", "format": "date-time"}` |

**What is lost:**
- Field numbers are dropped
- `reserved` fields and numbers have no JSON Schema equivalent
- Service definitions and RPC methods have no JSON Schema equivalent
- Proto `//` comments are not preserved (no standard comment field in JSON Schema)

---

## Type Mapping: JSON Schema to Protobuf

| JSON Schema | Protobuf |
|-------------|----------|
| `{"type": "object", "properties": {...}}` | message (assign field numbers sequentially from 1) |
| `{"type": "string", "enum": [...]}` | enum (prepend `UNSPECIFIED = 0` as first value) |
| `{"type": "array", "items": {...}}` | `repeated` field |
| `{"type": "object", "additionalProperties": {...}}` | `map<string, V>` |
| `{"oneOf": [...]}` | `oneof` |
| `{"type": "integer"}` | `int64` (safest default) |
| `{"type": "number"}` | `double` (safest default) |
| `{"type": "string"}` | `string` |
| `{"type": "boolean"}` | `bool` |
| `{"type": "string", "contentEncoding": "base64"}` | `bytes` |
| `{"type": "string", "format": "date-time"}` | `google.protobuf.Timestamp` |
| `$ref` | `import` + message reference |

**What is lost:**
- Validation constraints: `pattern`, `minLength`, `maxLength`, `minimum`, `maximum`, `multipleOf`
- Conditional logic: `if`/`then`/`else`, `not`
- `allOf` must be flattened manually
- `required` has no proto3 equivalent (all fields are optional by default)
- `default` values have no proto3 equivalent (proto3 uses zero values)
- `description` maps to `//` comments

---

## Known Lossy Conversions

Every format migration loses some information. The following conversions are inherently lossy and the user MUST be warned:

| Direction | What Is Lost |
|-----------|-------------|
| Any → Protobuf | Default values, doc strings (become comments), aliases, validation constraints |
| Protobuf → Any | Field numbers, reserved declarations, service/RPC definitions |
| JSON Schema → Avro | Validation constraints (`pattern`, `min`/`max`, `multipleOf`), conditional logic (`if`/`then`/`else`), `not`, complex `allOf` composition |
| JSON Schema → Protobuf | Validation constraints, conditional logic, `required` semantics, default values |
| Avro → Any | Aliases are dropped by all target formats |
| Protobuf `map<non-string, V>` → Avro | Avro maps only support string keys — restructure as array of records |
| Protobuf `uint32`/`uint64` → Avro | Unsigned integers narrowed to signed `int`/`long` — possible overflow |
| Avro `decimal` → Protobuf | Arbitrary-precision decimal becomes `string` (precision semantics lost) |

---

## General Guidance

- Create NEW subjects for the migrated format (e.g., `orders-value` becomes `orders-value-proto`)
- Do NOT change the schema format in an existing subject — this is a breaking change
- Schema IDs will change — update all serializer/deserializer configurations
- Test the converted schema with **validate_schema** before registering
- Use **diff_schemas** to compare the original and converted schemas side by side
- For complex schemas with references, migrate referenced schemas first
- Run producers and consumers in parallel during migration to ensure zero downtime


</details>

---

#### `new-kafka-topic`

End-to-end workflow for setting up key and value schemas for a new Kafka topic

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `topic_name` | Yes | Kafka topic name (e.g., orders, user-events) |
| `format` |  | Schema format: AVRO (default), PROTOBUF, or JSON |

<details>
<summary>Prompt content (click to expand)</summary>

End-to-end workflow for setting up schemas for a new Kafka topic.

## Step 1: Choose a Naming Strategy

Kafka subjects follow naming conventions based on the serializer's **SubjectNameStrategy**:
- **TopicNameStrategy** (default): `{topic}-key`, `{topic}-value`
- **RecordNameStrategy**: `{record.namespace}.{record.name}`
- **TopicRecordNameStrategy**: `{topic}-{record.namespace}.{record.name}`

For topic "orders", the default subjects are:
- Key: `orders-key`
- Value: `orders-value`

## Step 2: Design the Key Schema

The key schema determines message partitioning. Common patterns:
- **String key:** `{"type": "string"}` (simplest)
- **Composite key:** A record with business identifiers
- **Null key:** No key schema needed (round-robin partitioning)

## Step 3: Design the Value Schema

Design the value schema for your event/entity. Choose the format:
- **AVRO** (recommended for Kafka): compact binary, excellent evolution
- **PROTOBUF**: if you also use gRPC
- **JSON**: if consumers need human-readable messages

Use the **design-schema** prompt for format-specific guidance.

## Step 4: Validate Schemas

Before registering, validate both schemas:
```
validate_schema(schema: <key_schema>, schema_type: AVRO)
validate_schema(schema: <value_schema>, schema_type: AVRO)
```

## Step 5: Set Compatibility Level

Choose a compatibility level for each subject:
```
set_config(subject: "orders-key", compatibility_level: "FULL")
set_config(subject: "orders-value", compatibility_level: "BACKWARD")
```

**Recommendation:** FULL for keys (both producers and consumers handle changes), BACKWARD for values (default).

## Step 6: Check Compatibility (if subjects exist)

If the subject already has schemas registered:
```
check_compatibility(subject: "orders-value", schema: <new_schema>, schema_type: AVRO)
```

## Step 7: Register Schemas

```
register_schema(subject: "orders-key", schema: <key_schema>, schema_type: AVRO)
register_schema(subject: "orders-value", schema: <value_schema>, schema_type: AVRO)
```

## Step 8: Verify Registration

```
get_latest_schema(subject: "orders-key")
get_latest_schema(subject: "orders-value")
```

Note the returned schema IDs -- these are embedded in the Kafka message wire format.

## Step 9: Retrieve by ID

Consumers use schema IDs from messages to fetch schemas:
```
get_schema_by_id(id: <schema_id>)
```

## Step 10: Context Support

All tools accept the optional `context` parameter for multi-tenant isolation:
```
register_schema(subject: "orders-value", schema: <schema>, context: ".staging")
```

Available tools: validate_schema, check_compatibility, register_schema, get_latest_schema, get_schema_by_id, set_config, validate_subject_name


</details>

---

#### `plan-breaking-change`

Plan a safe breaking schema change with migration strategy

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name where the breaking change is planned |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Plan a safe breaking change for subject example-subject.

Steps:
1. Use get_latest_schema to understand the current schema
2. Use get_config to check the compatibility level
3. Use list_versions to see the version history

Strategy options:

**Option A: New subject (recommended for major changes)**
- Create a new subject (e.g. example-subject-v2) with the new schema
- Migrate producers to the new subject
- Keep the old subject in READONLY mode for consumers
- Tools: register_schema, set_mode READONLY

**Option B: Compatibility bypass (for minor breaking changes)**
- Set compatibility to NONE temporarily: set_config with compatibility_level: NONE
- Register the breaking schema
- Restore compatibility: set_config with original level
- WARNING: existing consumers may fail to deserialize

**Option C: Multi-step evolution**
- Add new fields alongside old fields (backward compatible)
- Migrate all consumers to use new fields
- Remove old fields in a later version
- Requires NONE compatibility for the final removal step

Always test with check_compatibility before registering.


</details>

---

#### `registry-health-audit`

Multi-step procedure for auditing registry health, configuration consistency, and schema quality

<details>
<summary>Prompt content (click to expand)</summary>

Multi-step registry health audit procedure.

## Step 1: Basic Health
Use **health_check** to verify:
- Registry is running
- Storage backend is connected and responsive

Use **get_server_info** to check:
- Version and build information
- Supported schema types

## Step 2: Registry Statistics
Use **get_registry_statistics** to get:
- Total subjects and schemas
- Schema type distribution (Avro, Protobuf, JSON)
- Total versions
- KEK and exporter counts

## Step 3: Configuration Consistency
Use **get_config** (with no subject) to check the global compatibility level.
- Is it BACKWARD (the recommended default)?
- Are there subjects with NONE that should not be?

Use **get_mode** (with no subject) to check the global mode.
- Should be READWRITE for normal operation.
- IMPORT mode should only be active during migrations.

## Step 4: Subject Health
Use **list_subjects** to get all subjects.
For suspicious subjects, use **count_versions** to check for:
- Subjects with excessive versions (>100 may indicate runaway registrations)
- Subjects with only 1 version (may be unused or abandoned)

## Step 5: Schema Quality
Use **score_schema_quality** on key subjects to check:
- Naming conventions (PascalCase records, snake_case fields)
- Documentation coverage
- Type safety (logical types, enums vs strings)
- Evolution readiness (defaults, nullable fields)

## Step 6: Dependency Health
Use **detect_schema_patterns** to check:
- Naming convention consistency across the registry
- Orphaned schemas (no references, no consumers)

Use **get_dependency_graph** on referenced subjects to verify:
- No circular dependencies
- Referenced schemas use FULL or FULL_TRANSITIVE compatibility

## Step 7: Encryption Audit (if applicable)
Use **list_keks** to check KEK inventory.
Use **test_kek** on each KEK to verify KMS connectivity.
Use **list_deks** to verify DEK coverage for encrypted subjects.

## Summary
After completing all steps, you should have a clear picture of:
- Registry availability and connectivity
- Configuration policy compliance
- Schema quality and naming consistency
- Dependency integrity
- Encryption key health


</details>

---

#### `review-schema-quality`

Analyze a schema for naming conventions, nullability, documentation, and best practices

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name of the schema to review |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Review the schema quality for subject example-subject.

Use get_latest_schema to fetch the current schema, then evaluate:

1. **Naming conventions**:
   - Record/message names: PascalCase
   - Field names: snake_case
   - Enum values: UPPER_SNAKE_CASE
   - Namespace/package: reverse domain notation

2. **Nullability**:
   - Optional fields should be nullable (Avro: union with null, Protobuf: optional)
   - Required fields should NOT be nullable
   - Default values should be meaningful

3. **Type usage**:
   - Use logical/semantic types (timestamps, UUIDs, decimals) instead of raw primitives
   - Use enums for fixed value sets instead of plain strings
   - Use appropriate numeric precision (int vs long, float vs double)

4. **Evolution readiness**:
   - All fields should have sensible defaults for backward compatibility
   - Avoid required fields that might become optional later
   - Consider using a version field or schema fingerprint

5. **Documentation**:
   - Fields should have descriptive names that are self-documenting
   - Complex fields should have doc comments (Avro: "doc" field, Protobuf: // comments)

Available tools: get_latest_schema, list_versions, get_config


</details>

---

#### `schema-evolution-cookbook`

Practical recipes for common schema evolution scenarios: add fields, rename, change types, and break compatibility safely

<details>
<summary>Prompt content (click to expand)</summary>

Practical recipes for common schema evolution scenarios.

## Recipe 1: Add an Optional Field (BACKWARD safe)

**Scenario:** Add a new "email" field to a User schema.

1. Use **get_config** to confirm compatibility is BACKWARD or FULL.
2. Add the field with a default value:
   - Avro: {"name": "email", "type": ["null", "string"], "default": null}
   - Protobuf: optional string email = 3;
   - JSON Schema: add to properties (NOT to required)
3. Use **check_compatibility** to validate.
4. Use **register_schema** to register.

## Recipe 2: Add a Required Field (needs care)

**Scenario:** Add a mandatory "created_at" timestamp.

Under BACKWARD compatibility, you CANNOT add a required field without a default.
1. Add the field with a sensible default (e.g., epoch zero, empty string).
2. Application logic treats the default as "not set."
3. Or: change compatibility to NONE temporarily (use **set_config**), register, restore.

## Recipe 3: Rename a Field (three-phase)

**Scenario:** Rename "customer_name" to "client_name".

Phase 1: Add "client_name" alongside "customer_name" (both populated).
Phase 2: Update all consumers to read "client_name". Deprecate "customer_name".
Phase 3: Remove "customer_name" (requires NONE compatibility for this step).

Use **diff_schemas** to verify each phase.

## Recipe 4: Change a Field Type

**Scenario:** Change "amount" from int to long (Avro) or int32 to int64 (Protobuf).

**Avro type promotions (safe under BACKWARD):**
- int -> long, float, double
- long -> float, double
- float -> double
- string <-> bytes

**Protobuf wire-compatible changes (safe):**
- int32 <-> uint32 <-> int64 <-> uint64 <-> bool (same wire type)
- fixed32 <-> sfixed32
- fixed64 <-> sfixed64

**Incompatible type changes:** Use the three-phase pattern (add new field, migrate, remove old).

## Recipe 5: Remove a Field

**Scenario:** Remove the deprecated "legacy_id" field.

Under BACKWARD: removing a field IS safe (old data's field is ignored).
Under FORWARD: removing a field IS NOT safe (old readers expect it).
Under FULL: removing a field IS NOT safe.

If removal is blocked, use **set_config** to temporarily set BACKWARD or NONE.
In Protobuf, use "reserved" to prevent field number reuse.

## Recipe 6: Break Compatibility Intentionally

**Scenario:** Major redesign of the schema.

Option A (recommended): Create a new subject (e.g., orders-v2-value).
1. Use **register_schema** under the new subject.
2. Migrate producers to the new subject.
3. Use **set_mode** READONLY on the old subject.

Option B: Bypass in existing subject.
1. Use **set_config** with NONE.
2. Use **register_schema** with the breaking change.
3. Use **set_config** to restore the original level.
WARNING: existing consumers may break.

## Recipe 7: Add a Schema Reference

**Scenario:** Extract Address into a shared subject.

1. Use **register_schema** to register Address under "com.example.Address".
2. Update the main schema to reference it.
3. Use **register_schema** with references array.
4. Set FULL_TRANSITIVE compatibility on the shared type.

## General Workflow

For any evolution:
1. **get_latest_schema** -- understand current state
2. **get_config** -- know the compatibility level
3. **check_compatibility** -- validate before registering
4. **explain_compatibility_failure** -- if it fails, get details
5. **register_schema** -- apply the change
6. **diff_schemas** -- verify the change

For domain knowledge, read: schema://glossary/compatibility and schema://glossary/design-patterns


</details>

---

#### `schema-getting-started`

Quick-start guide introducing available tools and common schema registry operations

<details>
<summary>Prompt content (click to expand)</summary>

Welcome to the Schema Registry MCP server. Here's a quick-start guide.

## Core operations

- **list_subjects** — see all registered subjects
- **get_latest_schema** — fetch the current schema for a subject
- **register_schema** — register a new schema version
- **check_compatibility** — test a schema before registering

## Discovery

- **search_schemas** — search schema content by keyword or regex
- **match_subjects** — find subjects by name pattern (regex, glob, or fuzzy)
- **get_registry_statistics** — overview of subjects, versions, types, KEKs, and exporters

## Schema intelligence

- **score_schema_quality** — analyze naming, docs, type safety, and evolution readiness
- **diff_schemas** — compare two schema versions structurally
- **find_similar_schemas** — find schemas with overlapping field sets
- **suggest_schema_evolution** — generate a compatible schema change
- **explain_compatibility_failure** — human-readable explanations for compat errors

## Configuration

- **get_config / set_config** — manage compatibility levels (BACKWARD, FORWARD, FULL, NONE)
- **get_mode / set_mode** — manage modes (READWRITE, READONLY, IMPORT)

## Encryption (CSFLE)

- **create_kek / create_dek** — set up client-side field encryption
- **list_keks / list_deks** — inspect encryption keys

## Resources (read-only data)

Resources are available via URI patterns like `schema://subjects`, `schema://subjects/{name}`, etc.

## Getting help

Use the other prompts for detailed guidance: design-schema, evolve-schema, check-compatibility, troubleshooting, setup-encryption, and more.


</details>

---

#### `schema-impact-analysis`

Guided workflow for assessing the impact of a proposed schema change across dependents

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to analyze impact for |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Assess the impact of a proposed schema change on subject example-subject.

## Step 1: Understand the current state
1. Use **get_latest_schema** to fetch the current schema for example-subject
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


</details>

---

#### `schema-naming-conventions`

Guide to subject naming strategies (topic_name, record_name, topic_record_name)

<details>
<summary>Prompt content (click to expand)</summary>

Guide to subject naming strategies in the schema registry.

## Naming strategies

### topic_name (default)
Pattern: `{topic}-key` or `{topic}-value`
Examples: `orders-key`, `orders-value`, `user-events-value`
Use when: one schema per Kafka topic, simple key/value distinction.

### record_name
Pattern: `{fully.qualified.RecordName}`
Examples: `com.example.Order`, `com.example.UserEvent`
Use when: multiple event types share a topic, schemas identified by record name.

### topic_record_name
Pattern: `{topic}-{fully.qualified.RecordName}`
Examples: `orders-com.example.OrderCreated`, `orders-com.example.OrderShipped`
Use when: multiple event types per topic and you want topic context in the subject name.

## Validation
Use **validate_subject_name** to check if a name conforms to a strategy:
- validate_subject_name(subject: "orders-value", strategy: "topic_name")
- validate_subject_name(subject: "com.example.Order", strategy: "record_name")

## Best practices
- Pick one strategy per environment and use it consistently
- Use **detect_schema_patterns** to check current naming convention coverage
- Use **match_subjects** to find subjects that deviate from the dominant pattern
- Avoid mixing strategies in the same registry context
- Use lowercase with hyphens for topic names: `user-events` not `UserEvents`
- Use reverse-domain namespace for record names: `com.company.domain.Type`


</details>

---

#### `schema-references-guide`

Guide for cross-subject schema references with per-format name semantics (Avro, Protobuf, JSON Schema)

<details>
<summary>Prompt content (click to expand)</summary>

Guide for cross-subject schema references.

## What Are References?
References allow one schema to depend on another schema registered in a different subject. This enables reusable, independently versioned type definitions.

## Reference Structure
Each reference has three fields:
- **name** -- how the referencing schema refers to this dependency
- **subject** -- subject where the referenced schema is registered
- **version** -- version number of the referenced schema

## Per-Format Name Semantics

### Avro
The **name** field is the fully qualified type name (namespace + name):
    name: "com.example.Address"
    subject: "com.example.Address"
    version: 1

In the schema, reference it by its fully qualified name in the type field.

### Protobuf
The **name** field is the import path:
    name: "address.proto"
    subject: "address-value"
    version: 1

In the .proto file, use: import "address.proto";

### JSON Schema
The **name** field is the reference URL:
    name: "address.json"
    subject: "address-value"
    version: 1

In the schema, use: "$ref": "address.json"

## Registering with References
Use **register_schema** with the references array:
    register_schema(
      subject: "order-value",
      schema: "...",
      schema_type: "AVRO",
      references: [
        {"name": "com.example.Address", "subject": "address-value", "version": 1}
      ]
    )

## Important Rules
1. Referenced schemas MUST be registered before the schemas that depend on them.
2. A schema that is referenced by others cannot be permanently deleted.
3. Use **get_referenced_by** to find all schemas that reference a given subject.
4. Use **get_dependency_graph** to visualize the full reference tree.
5. Use FULL_TRANSITIVE compatibility for shared referenced types.

## Resolving References
Pass ?referenceFormat=RESOLVED when fetching schemas to get resolved (inline) references.

For domain knowledge, read: schema://glossary/schema-types


</details>

---

#### `schema-review-checklist`

Pre-registration checklist: syntax, compatibility, quality, naming, uniqueness, dependencies, and impact

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name for the schema being reviewed |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Pre-registration review checklist for schema "example-subject".

## 1. Syntax Validation
```
validate_schema(schema: <proposed_schema>, schema_type: "AVRO")
```
Ensure the schema is syntactically valid.

## 2. Compatibility Check
```
check_compatibility(subject: "example-subject", schema: <proposed_schema>)
```
Verify the change is compatible with the current compatibility level.

## 3. Quality Score
```
score_schema_quality(schema: <proposed_schema>, schema_type: "AVRO")
```
Review naming conventions, documentation, nullability, and evolution readiness.

## 4. Subject Name Validation
```
validate_subject_name(subject: "example-subject", strategy: "topic_name")
```
Ensure the subject name follows established naming conventions.

## 5. Uniqueness Check
```
lookup_schema(subject: "example-subject", schema: <proposed_schema>)
find_similar_schemas(schema: <proposed_schema>, schema_type: "AVRO")
```
Check if this schema already exists (deduplication) or if similar schemas should be consolidated.

## 6. Dependency Check
```
# If the schema has references, verify they exist:
get_schema_version(subject: "<ref_subject>", version: <ref_version>)
```

For schemas that are referenced by others, use FULL_TRANSITIVE compatibility:
```
get_referenced_by(subject: "example-subject")
```

## 7. Field Consistency
```
check_field_consistency(field_name: "<shared_field>")
```
Ensure shared field names use the same type as in other schemas.

## 8. Complexity Assessment
```
get_schema_complexity(schema: <proposed_schema>, schema_type: "AVRO")
```
Review schema depth, field count, and union complexity.

## 9. Impact Analysis
```
diff_schemas(subject: "example-subject", version1: "latest", version2: <proposed_version>)
get_referenced_by(subject: "example-subject")
```
Understand what downstream schemas and consumers would be affected.

## 10. Data Contracts
If the subject has data contracts (metadata, rules):
```
get_subject_config_full(subject: "example-subject")
```
Ensure the new schema version is compatible with existing rules and metadata.

## 11. Context Verification
Ensure you are registering in the correct context:
```
list_subjects(context: "<expected_context>")
```

## 12. Register
After all checks pass:
```
register_schema(subject: "example-subject", schema: <proposed_schema>, schema_type: "AVRO")
```

Available tools: validate_schema, check_compatibility, score_schema_quality, validate_subject_name, lookup_schema, find_similar_schemas, get_referenced_by, check_field_consistency, get_schema_complexity, diff_schemas, get_subject_config_full, register_schema


</details>

---

#### `setup-data-contracts`

Guide for adding metadata, tags, and data quality rules to schemas

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to add data contracts to |
| `context` |  | Registry context for multi-tenant isolation (defaults to default context) |

<details>
<summary>Prompt content (click to expand)</summary>

Set up data contracts for subject example-subject.

Data contracts add metadata, tags, and data quality rules to schemas.

Steps:
1. Use get_latest_schema to inspect the current schema for example-subject
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


</details>

---

#### `setup-encryption`

Guide for setting up client-side field encryption with KEK/DEK

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `kms_type` | Yes | KMS provider type (e.g. aws-kms, azure-kms, gcp-kms, hcvault) |

<details>
<summary>Prompt content (click to expand)</summary>

Set up client-side field encryption (CSFLE) with hcvault.

## Step 1: Create a KEK (Key Encryption Key)

Use the **create_kek** tool:
- `name`: descriptive name (e.g., "production-kek")
- `kms_type`: hcvault
- `kms_key_id`: your KMS key identifier (see provider-specific guidance below)
- `kms_props`: provider-specific connection properties (see below)
- `shared`: false (recommended -- each KEK is used by one application)

## Step 2: Test KEK Connectivity

Use the **test_kek** tool immediately after creation:
- `name`: the KEK name from Step 1
- This verifies the registry can reach your KMS and the key is usable
- If this fails, check your kms_props and KMS permissions

## Step 3: Create a DEK (Data Encryption Key)

Use the **create_dek** tool:
- `kek_name`: name of the KEK created in Step 1
- `subject`: schema subject whose data will be encrypted
- `algorithm`: `AES256_GCM` (recommended) or `AES256_SIV` (deterministic, supports search)

The DEK is automatically generated and wrapped (encrypted) by the KEK via your KMS.

## Available Tools

- **create_kek** / **get_kek** / **update_kek** / **delete_kek** / **undelete_kek** / **list_keks** -- KEK management
- **test_kek** -- verify KMS connectivity
- **create_dek** / **get_dek** / **list_deks** / **list_dek_versions** -- DEK management
- **delete_dek** / **undelete_dek** / **rewrap_dek** -- DEK lifecycle

---

## Provider: HashiCorp Vault (hcvault)

**Prerequisites:**
1. Transit secrets engine enabled: `vault secrets enable transit`
2. Encryption key created: `vault write transit/keys/my-key type=aes256-gcm96`
3. Policy granting encrypt/decrypt on the key path

**Configuration:**
- `kms_key_id`: Transit engine key name (e.g., `my-encryption-key`)
- `kms_props`:
  - `vault.address`: Vault server URL (e.g., `http://vault:8200`)
  - `vault.token`: Vault authentication token
  - OR for AppRole auth: `vault.role.id` + `vault.secret.id`
  - For Vault Enterprise: add `vault.namespace`

**Example:**
```json
{
  "name": "prod-kek",
  "kms_type": "hcvault",
  "kms_key_id": "my-encryption-key",
  "kms_props": {
    "vault.address": "http://vault:8200",
    "vault.token": "s.xxxxxxxxxxxx"
  },
  "shared": false
}
```

---

## Provider: OpenBao (openbao)

OpenBao is an open-source fork of HashiCorp Vault with an identical Transit API.

**Configuration:**
- `kms_key_id`: Transit engine key name
- `kms_props`:
  - `openbao.address`: OpenBao server URL
  - `openbao.token`: authentication token

---

## Provider: AWS KMS (aws-kms)

**Prerequisites:**
- KMS key created in your AWS account
- IAM identity has `kms:Encrypt` and `kms:Decrypt` permissions on the key

**Configuration:**
- `kms_key_id`: full ARN (e.g., `arn:aws:kms:us-east-1:123456789:key/uuid`) or alias (e.g., `alias/my-key`)
- `kms_props`:
  - `aws.region`: AWS region (required, e.g., `us-east-1`)
  - `aws.access.key.id` + `aws.secret.access.key`: explicit credentials (optional if using IAM role)

---

## Provider: Azure Key Vault (azure-kms)

**Prerequisites:**
- Key Vault created with a key
- Service principal has `wrap` and `unwrap` permissions on the key

**Configuration:**
- `kms_key_id`: Key Vault key URL (e.g., `https://myvault.vault.azure.net/keys/my-key`)
- `kms_props`:
  - `azure.tenant.id`: Azure AD tenant ID
  - `azure.client.id`: service principal client ID
  - `azure.client.secret`: service principal secret

---

## Provider: GCP KMS (gcp-kms)

**Prerequisites:**
- Key ring and crypto key created in GCP
- Service account has `cloudkms.cryptoKeyVersions.useToEncrypt` and `useToDecrypt` roles

**Configuration:**
- `kms_key_id`: full resource name (e.g., `projects/my-project/locations/global/keyRings/my-ring/cryptoKeys/my-key`)
- `kms_props`:
  - `gcp.project.id`: GCP project ID
  - `gcp.credentials.json`: path to credentials file (optional if using application default credentials)

---

## Common Best Practices

- Use **separate KEKs** per environment (dev, staging, production)
- Set `shared: false` unless multiple applications need the same DEK
- Always run **test_kek** immediately after **create_kek** to verify connectivity
- Use **AES256_GCM** for general encryption; use **AES256_SIV** only if you need deterministic encryption for searchable fields
- Rotate DEKs periodically using **rewrap_dek** (re-encrypts the DEK with a new KMS key version without changing the DEK itself)
- See the **full-encryption-lifecycle** prompt for the complete key rotation and cleanup workflow


</details>

---

#### `setup-rbac`

Guide for configuring authentication and role-based access control (RBAC)

<details>
<summary>Prompt content (click to expand)</summary>

Guide for configuring authentication and role-based access control (RBAC).

## Authentication Methods

The registry supports multiple auth methods (configured in security.auth.methods):

| Method | Description |
|--------|-------------|
| **basic** | Username/password via HTTP Basic Auth |
| **api_key** | API keys sent as Bearer tokens |
| **jwt** | JWT tokens (for external identity providers) |
| **oidc** | OpenID Connect (delegated to an OIDC provider) |
| **ldap** | LDAP directory authentication |
| **mtls** | Mutual TLS client certificates |

## The 4 Built-in Roles

| Role | Permissions |
|------|-------------|
| **admin** | Full access: manage users, API keys, schemas, config, modes |
| **write** | Register/delete schemas, set config/mode, manage encryption keys |
| **read** | Read schemas, subjects, config, mode. Cannot modify anything. |
| **readwrite** | Read + write schemas and config. Cannot manage users or API keys. |

## Setup Steps

### Step 1: Enable auth in config
Set security.auth.enabled: true and choose methods.

### Step 2: Create admin user
Use **create_user** with role: admin.

### Step 3: Create service accounts
Use **create_api_key** for each service:
- Producers: role write or readwrite
- Consumers: role read
- CI/CD: role write (for schema registration)
- Monitoring: role read

### Step 4: Test access
Use **list_users** and **list_api_keys** to verify.

## MCP Admin Tools

- **create_user / get_user / list_users / update_user / delete_user** -- manage users
- **create_api_key / get_api_key / list_api_keys / update_api_key / delete_api_key** -- manage API keys
- **list_roles** -- list available roles and their permissions


</details>

---

#### `team-onboarding`

Workflow for onboarding a new team with context creation, schema registration, RBAC setup, and naming conventions

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `team_name` | Yes | Team name for the new context namespace |

<details>
<summary>Prompt content (click to expand)</summary>

Workflow for onboarding a new team onto the schema registry.

## Step 1: Create a Context

Create an isolated namespace for team "example-team":
```
# Contexts are created implicitly by registering a schema in them.
# First, register a bootstrap schema or set context-level config.
set_config(compatibility_level: "BACKWARD", context: ".example-team")
```

## Step 2: Register Team Schemas

Register schemas within the team context:
```
register_schema(subject: "orders-value", schema: <schema>, schema_type: "AVRO", context: ".example-team")
```

## Step 3: Set Context-Level Compatibility

Set the team's default compatibility level:
```
set_config(compatibility_level: "BACKWARD", context: ".example-team")
```

Individual subjects can override:
```
set_config(subject: "shared-types", compatibility_level: "FULL_TRANSITIVE", context: ".example-team")
```

## Step 4: Create User and API Keys

Create a user for the team:
```
create_user(username: "example-team-service", password: "<secure>", role: "developer")
```

Create API keys for services:
```
create_apikey(name: "example-team-producer", role: "developer", expires_in: 2592000)
create_apikey(name: "example-team-consumer", role: "readonly", expires_in: 2592000)
```

## Step 5: Set Up Naming Conventions

Establish naming rules for the team:
- Subject pattern: `example-team-{topic}-{key|value}`
- Avro namespace: `com.company.example-team`

Validate names:
```
validate_subject_name(subject: "example-team-orders-value", strategy: "topic_name")
```

## Step 6: Verify Context Isolation

Confirm the team's schemas are isolated from other contexts:
```
list_subjects(context: ".example-team")
```

Confirm default context does not see team schemas:
```
list_subjects()
```

## Step 7: Browse via Context-Scoped Resources

Team data is accessible via context-scoped resource URIs:
- `schema://contexts/.example-team/subjects` -- all team subjects
- `schema://contexts/.example-team/subjects/<name>` -- subject details
- `schema://contexts/.example-team/config` -- team config

## Step 8: Documentation Pointers

Direct the team to these glossary resources:
- `schema://glossary/core-concepts` -- schema registry fundamentals
- `schema://glossary/contexts` -- context isolation details
- `schema://glossary/best-practices` -- per-format guidance
- `schema://glossary/design-patterns` -- common patterns

Available tools: set_config, register_schema, create_user, create_apikey, validate_subject_name, list_subjects, list_contexts


</details>

---

#### `troubleshooting`

Diagnostic guide for common schema registry issues and errors

<details>
<summary>Prompt content (click to expand)</summary>

Diagnostic guide for common schema registry issues.

## Step 1: Check health
Use **health_check** to verify the registry is running and storage is connected.

## Step 2: Identify the error

| Error Code | Meaning | Likely cause |
|------------|---------|--------------|
| 42201 | Invalid schema | Malformed JSON, missing required Avro/Protobuf/JSON Schema fields |
| 42203 | Invalid compatibility level | Typo in compatibility level string |
| 409 | Incompatible schema | Schema violates the configured compatibility level |
| 40401 | Subject not found | Typo in subject name, or subject was soft-deleted |
| 40402 | Version not found | Version number does not exist for this subject |
| 40403 | Schema not found | Global schema ID does not exist |
| 50001 | Internal error | Storage backend issue, check server logs |

## Step 3: Debug by category

**Registration failures:**
1. Use **validate_schema** to check syntax without registering
2. Use **get_config** to check the compatibility level
3. Use **check_compatibility** to test against existing versions
4. Use **explain_compatibility_failure** for detailed fix suggestions

**Subject/version not found:**
1. Use **list_subjects** to see all subjects (add include_deleted for soft-deleted)
2. Use **match_subjects** with fuzzy mode to find similar names
3. Use **list_versions** to check available versions

**Performance issues:**
1. Use **get_registry_statistics** to check registry size
2. Use **count_versions** to check version count per subject
3. Large registries (>10k subjects) may need pagination on search operations

**Encryption issues:**
1. Use **list_keks** to verify KEK exists
2. Use **test_kek** to verify KMS connectivity
3. Use **list_deks** to check DEK status

Available tools: health_check, get_server_info, validate_schema, check_compatibility, explain_compatibility_failure, list_subjects, match_subjects


</details>

---

