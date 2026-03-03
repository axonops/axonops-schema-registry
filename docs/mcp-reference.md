# MCP API Reference

> Auto-generated from the MCP server registration. Do not edit manually.
>
> Regenerate with: `go run ./cmd/generate-mcp-docs > docs/mcp-reference.md`

**105 tools** (71 read-only, 34 write) | **31 resources** (19 static, 12 templated) | **25 prompts**

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
| 78 | `match_subjects` | Yes | Find subjects matching a pattern. Regex mode compiles as Go regex. Glob mode uses wildcard matching (case-insensitive... |
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
| `field` | string | Yes |  |

---

#### `check_write_mode`

Check if write operations are allowed for a subject. Returns the blocking mode name (READONLY or READONLY_OVERRIDE) or empty string if writes are allowed.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subject` | string |  |  |

---

#### `compare_subjects`

Compare the latest schemas of two different subjects, showing structural differences.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subject_a` | string | Yes |  |
| `subject_b` | string | Yes |  |

---

#### `count_subjects`

Count the total number of registered subjects in the registry.

**Annotations:** read-only

---

#### `count_versions`

Count the number of schema versions registered for a subject.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `subject` | string |  |  |

---

#### `delete_subject`

Delete a subject and all its schema versions. Soft-deletes by default; use permanent=true for hard delete.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `confirm_token` | string |  |  |
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
| `dry_run` | boolean |  |  |
| `permanent` | boolean |  |  |
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `detect_schema_patterns`

Scan the registry to detect naming patterns, common field groups, and evolution statistics.

**Annotations:** read-only

---

#### `diff_schemas`

Diff two schema versions within a subject, showing added, removed, and type-changed fields. Fields are extracted from both versions and matched using normalized snake_case names. If version2 is omitted, diffs against the latest version.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `export_subject`

Export all schema versions for a subject with configuration and metadata.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subject` | string | Yes |  |

---

#### `find_schemas_by_field`

Find all schemas containing a field with the given name. Exact mode auto-generates naming variants (snake_case, camelCase, PascalCase, kebab-case). Fuzzy mode uses Levenshtein distance with configurable threshold (default 0.7). Regex mode compiles the field name as a regular expression.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `regex` | boolean |  |  |
| `type_pattern` | string | Yes |  |

---

#### `find_similar_schemas`

Find schemas structurally similar to a given subject using Jaccard similarity coefficient (|shared fields| / |total unique fields|). Field names are normalized to snake_case before comparison. Returns similarity scores (0.0-1.0) and lists of shared fields.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subject` | string | Yes |  |
| `threshold` | number |  |  |

---

#### `format_schema`

Format a schema by subject and version. Supported formats depend on schema type. Returns the formatted schema string.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `subject` | string |  |  |

---

#### `get_config_full`

Get the full configuration record for a subject or global default, including metadata, ruleSets, alias, compatibilityGroup, and all data contract fields. Uses 4-tier fallback: subject → context global → __GLOBAL → server default.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `subject` | string | Yes |  |

---

#### `get_max_schema_id`

Get the highest schema ID currently assigned in the registry

**Annotations:** read-only

---

#### `get_mode`

Get the registry mode for a subject or the global default. Modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subject` | string |  |  |

---

#### `get_raw_schema_by_id`

Get the raw schema string by its global ID, without any metadata

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes |  |

---

#### `get_raw_schema_version`

Get the raw schema string by subject name and version number, without any metadata

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `get_referenced_by`

Get schemas that reference a specific subject-version pair

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `get_registry_statistics`

Get aggregate statistics about the registry: total subjects, schemas, types breakdown, KEKs, DEKs, and exporters.

**Annotations:** read-only

---

#### `get_schema_by_id`

Get a schema by its global ID, returning the full schema record including subject, version, type, and schema content

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes |  |

---

#### `get_schema_complexity`

Compute complexity metrics and grade (A-D) for a schema. Measures field_count (total fields including nested) and max_depth (deepest nesting level via dot-notation paths). Grades: A (≤15 fields, ≤3 depth), B (≤30, ≤4), C (≤50, ≤5), D (>50 or >5). Grade D schemas should be decomposed into referenced sub-schemas.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `subject` | string | Yes |  |
| `version` | integer | Yes |  |

---

#### `get_schemas_by_subject`

Get all schema versions for a subject. Returns full schema records for every version, optionally including soft-deleted versions.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `subject` | string | Yes |  |

---

#### `get_subject_metadata`

Get metadata for a subject. Without filters, returns the metadata from the latest schema version. With key/value filters, searches all versions for the latest one whose metadata properties match ALL specified key/value pairs and returns a full schema record.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `deleted` | boolean |  |  |
| `subject` | string | Yes |  |

---

#### `lookup_schema`

Check if a schema is already registered under a subject. Returns the existing schema record if found.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `deleted` | boolean |  |  |
| `schema` | string | Yes |  |
| `schema_type` | string |  |  |
| `subject` | string | Yes |  |

---

#### `match_subjects`

Find subjects matching a pattern. Regex mode compiles as Go regex. Glob mode uses wildcard matching (case-insensitive). Fuzzy mode uses Levenshtein distance with configurable threshold (default 0.6).

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `pattern` | string | Yes |  |
| `regex` | boolean |  |  |

---

#### `normalize_schema`

Parse and normalize a schema, returning the canonical form and fingerprint for deduplication.

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `schema_type` | string |  |  |
| `subject` | string | Yes |  |
| `target_schema` | string | Yes |  |

---

#### `register_schema`

Register a new schema version for a subject. If the same schema already exists, returns the existing record.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
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
| `subject` | string | Yes |  |

---

#### `suggest_schema_evolution`

Generate concrete schema code for a compatible evolution step (add field, deprecate field, add enum symbol).

**Annotations:** read-only

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `change_type` | string | Yes |  |
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
| `schema://glossary/best-practices` | `glossary-best-practices` | Actionable best practices for Avro, Protobuf, and JSON Schema: field naming, nullability, evolution readiness, common mistakes, and per-format guidance |
| `schema://glossary/compatibility` | `glossary-compatibility` | All 7 compatibility modes, Avro type promotions, Protobuf wire types, JSON Schema constraints, transitive semantics, and configuration resolution |
| `schema://glossary/contexts` | `glossary-contexts` | Multi-tenancy via contexts: default context, __GLOBAL, qualified subjects, URL routing, isolation guarantees, and 4-tier config/mode inheritance |
| `schema://glossary/core-concepts` | `glossary-core-concepts` | Schema registry fundamentals: what a schema registry is, subjects, versions, IDs, deduplication, modes, naming strategies, and the serialization flow |
| `schema://glossary/data-contracts` | `glossary-data-contracts` | Data contracts: metadata properties, tags, sensitive fields, rulesets (domain/migration/encoding), rule structure, 3-layer merge, and optimistic concurrency |
| `schema://glossary/design-patterns` | `glossary-design-patterns` | Common schema design patterns: event envelope, entity lifecycle, snapshot vs delta, fat vs thin events, shared types, three-phase rename, and CI/CD integration |
| `schema://glossary/encryption` | `glossary-encryption` | Client-side field level encryption (CSFLE): envelope encryption, KEK/DEK model, KMS providers, algorithms, key rotation, and rewrapping |
| `schema://glossary/exporters` | `glossary-exporters` | Schema linking via exporters: exporter model, lifecycle states (STARTING/RUNNING/PAUSED/ERROR), context types (AUTO/CUSTOM/NONE), and configuration |
| `schema://glossary/migration` | `glossary-migration` | Confluent migration: step-by-step procedure, IMPORT mode, ID preservation, the import API, verification, and rollback |
| `schema://glossary/schema-types` | `glossary-schema-types` | Deep reference for Avro (types, logical types, aliases, canonicalization), Protobuf (proto3, well-known types, wire types), and JSON Schema (drafts, keywords, combinators) |
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
| `schema://contexts/{context}/subjects` | `context-subjects` | List of subjects in a specific registry context |
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
| `audit-subject-history` | Review the version history and evolution of a schema subject | `subject` (required) |
| `check-compatibility` | Troubleshoot schema compatibility issues and suggest fixes | `subject` (required) |
| `compare-formats` | Help choose between Avro, Protobuf, and JSON Schema for a use case | `use_case` (required) |
| `configure-exporter` | Guide for setting up schema linking via an exporter | `exporter_type` |
| `context-management` | Guide for managing multi-tenant contexts and the 4-tier config/mode inheritance chain | — |
| `data-rules-deep-dive` | Comprehensive guide to data contract rules: domain, migration, and encoding rules with examples | — |
| `debug-registration-error` | Debug schema registration failures by error code | `error_code` (required) |
| `design-schema` | Guide for designing a new schema in the chosen format | `format` (required), `domain` |
| `evolve-schema` | Guide for safely evolving an existing schema with backward compatibility | `subject` (required) |
| `full-encryption-lifecycle` | End-to-end CSFLE workflow: KEK creation, DEK management, key rotation, rewrapping, and cleanup | — |
| `glossary-lookup` | Look up a schema registry concept and get directed to the relevant glossary resource | `topic` (required) |
| `import-from-confluent` | Step-by-step guide for migrating schemas from Confluent Schema Registry with ID preservation | — |
| `migrate-schemas` | Guide for migrating schemas between formats (e.g. Avro to Protobuf) | `source_format` (required), `target_format` (required) |
| `plan-breaking-change` | Plan a safe breaking schema change with migration strategy | `subject` (required) |
| `registry-health-audit` | Multi-step procedure for auditing registry health, configuration consistency, and schema quality | — |
| `review-schema-quality` | Analyze a schema for naming conventions, nullability, documentation, and best practices | `subject` (required) |
| `schema-evolution-cookbook` | Practical recipes for common schema evolution scenarios: add fields, rename, change types, and break compatibility safely | — |
| `schema-getting-started` | Quick-start guide introducing available tools and common schema registry operations | — |
| `schema-impact-analysis` | Guided workflow for assessing the impact of a proposed schema change across dependents | `subject` (required) |
| `schema-naming-conventions` | Guide to subject naming strategies (topic_name, record_name, topic_record_name) | — |
| `schema-references-guide` | Guide for cross-subject schema references with per-format name semantics (Avro, Protobuf, JSON Schema) | — |
| `setup-data-contracts` | Guide for adding metadata, tags, and data quality rules to schemas | `subject` (required) |
| `setup-encryption` | Guide for setting up client-side field encryption with KEK/DEK | `kms_type` (required) |
| `setup-rbac` | Guide for configuring authentication and role-based access control (RBAC) | — |
| `troubleshooting` | Diagnostic guide for common schema registry issues and errors | — |

### Prompt Details

#### `audit-subject-history`

Review the version history and evolution of a schema subject

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to audit |

---

#### `check-compatibility`

Troubleshoot schema compatibility issues and suggest fixes

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to check compatibility for |

---

#### `compare-formats`

Help choose between Avro, Protobuf, and JSON Schema for a use case

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `use_case` | Yes | Use case description (e.g. event streaming, REST API, RPC) |

---

#### `configure-exporter`

Guide for setting up schema linking via an exporter

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `exporter_type` |  | Exporter context type: AUTO, CUSTOM, or NONE |

---

#### `context-management`

Guide for managing multi-tenant contexts and the 4-tier config/mode inheritance chain

---

#### `data-rules-deep-dive`

Comprehensive guide to data contract rules: domain, migration, and encoding rules with examples

---

#### `debug-registration-error`

Debug schema registration failures by error code

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `error_code` | Yes | Error code from failed registration (e.g. 42201, 409, 40401) |

---

#### `design-schema`

Guide for designing a new schema in the chosen format

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `format` | Yes | Schema format: AVRO, PROTOBUF, or JSON |
| `domain` |  | Domain or topic for the schema (e.g. user-events, payments) |

---

#### `evolve-schema`

Guide for safely evolving an existing schema with backward compatibility

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name of the schema to evolve |

---

#### `full-encryption-lifecycle`

End-to-end CSFLE workflow: KEK creation, DEK management, key rotation, rewrapping, and cleanup

---

#### `glossary-lookup`

Look up a schema registry concept and get directed to the relevant glossary resource

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `topic` | Yes | Keyword or concept to look up (e.g. compatibility, CSFLE, contexts, avro) |

---

#### `import-from-confluent`

Step-by-step guide for migrating schemas from Confluent Schema Registry with ID preservation

---

#### `migrate-schemas`

Guide for migrating schemas between formats (e.g. Avro to Protobuf)

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `source_format` | Yes | Source schema format (AVRO, PROTOBUF, JSON) |
| `target_format` | Yes | Target schema format (AVRO, PROTOBUF, JSON) |

---

#### `plan-breaking-change`

Plan a safe breaking schema change with migration strategy

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name where the breaking change is planned |

---

#### `registry-health-audit`

Multi-step procedure for auditing registry health, configuration consistency, and schema quality

---

#### `review-schema-quality`

Analyze a schema for naming conventions, nullability, documentation, and best practices

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name of the schema to review |

---

#### `schema-evolution-cookbook`

Practical recipes for common schema evolution scenarios: add fields, rename, change types, and break compatibility safely

---

#### `schema-getting-started`

Quick-start guide introducing available tools and common schema registry operations

---

#### `schema-impact-analysis`

Guided workflow for assessing the impact of a proposed schema change across dependents

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to analyze impact for |

---

#### `schema-naming-conventions`

Guide to subject naming strategies (topic_name, record_name, topic_record_name)

---

#### `schema-references-guide`

Guide for cross-subject schema references with per-format name semantics (Avro, Protobuf, JSON Schema)

---

#### `setup-data-contracts`

Guide for adding metadata, tags, and data quality rules to schemas

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `subject` | Yes | Subject name to add data contracts to |

---

#### `setup-encryption`

Guide for setting up client-side field encryption with KEK/DEK

**Arguments:**

| Name | Required | Description |
|------|----------|-------------|
| `kms_type` | Yes | KMS provider type (e.g. aws-kms, azure-kms, gcp-kms, hcvault) |

---

#### `setup-rbac`

Guide for configuring authentication and role-based access control (RBAC)

---

#### `troubleshooting`

Diagnostic guide for common schema registry issues and errors

---

