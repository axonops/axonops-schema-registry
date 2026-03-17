# Tool Selection Guide

## Finding Schemas

| Task | Tools |
|------|-------|
| Browse all subjects | **list_subjects** (with optional `prefix`, `pattern`, `context`) |
| Find a schema by name | **match_subjects** (fuzzy matching for misspelled names) |
| Find schemas containing a field | **find_schemas_by_field** (search across all schemas) |
| Find schemas by type/pattern | **find_schemas_by_type**, **detect_schema_patterns** |
| Find similar/duplicate schemas | **find_similar_schemas** |
| Search by content | **search_schemas** (keyword search across schema content) |
| Get a schema by ID | **get_schema_by_id**, **get_raw_schema_by_id** |
| Get the latest version | **get_latest_schema** |
| Get a specific version | **get_schema_version**, **get_raw_schema_version** |

## Registering and Validating

| Task | Tools |
|------|-------|
| Check syntax before registering | **validate_schema** |
| Check compatibility before registering | **check_compatibility** |
| Register a new schema | **register_schema** |
| Check if a schema already exists | **lookup_schema** |
| Validate a subject name | **validate_subject_name** |
| Normalize a schema | **normalize_schema** |

## Comparing and Analyzing

| Task | Tools |
|------|-------|
| Diff two schema versions | **diff_schemas** |
| Compare two subjects | **compare_subjects** |
| Score schema quality | **score_schema_quality** |
| Check field consistency across schemas | **check_field_consistency** |
| Measure schema complexity | **get_schema_complexity** |
| Check compatibility against multiple versions | **check_compatibility_multi** |

## Evolving Schemas

| Task | Tools |
|------|-------|
| Suggest compatible changes | **suggest_compatible_change** |
| Explain why a change is incompatible | **explain_compatibility_failure** |
| Suggest evolution path | **suggest_schema_evolution** |
| Plan a migration path | **plan_migration_path** |
| Format/pretty-print a schema | **format_schema** |

## Configuration

| Task | Tools |
|------|-------|
| Get compatibility level | **get_config** |
| Set compatibility level | **set_config** |
| Get full config (with metadata/rules) | **get_config_full**, **get_subject_config_full** |
| Set full config | **set_config_full** |
| Get/set mode | **get_mode**, **set_mode** |
| Check if writes are allowed | **check_write_mode** |
| Get global config directly | **get_global_config_direct** |

## Dependencies and References

| Task | Tools |
|------|-------|
| Find what references this schema | **get_referenced_by** |
| Get full dependency graph | **get_dependency_graph** |
| List subjects using a schema ID | **get_subjects_for_schema** |
| List version pairs for a schema ID | **get_versions_for_schema** |

## Administration

| Task | Tools |
|------|-------|
| Manage users | **list_users**, **create_user**, **update_user**, **delete_user**, **get_user**, **get_user_by_username** |
| Manage API keys | **list_apikeys**, **create_apikey**, **update_apikey**, **delete_apikey**, **rotate_apikey**, **revoke_apikey** |
| List roles | **list_roles** |
| Change password | **change_password** |

## Encryption (CSFLE)

| Task | Tools |
|------|-------|
| Manage KEKs | **create_kek**, **get_kek**, **update_kek**, **delete_kek**, **undelete_kek**, **list_keks** |
| Test KMS connectivity | **test_kek** |
| Manage DEKs | **create_dek**, **get_dek**, **list_deks**, **list_dek_versions**, **delete_dek**, **undelete_dek** |
| Rotate DEK encryption | **rewrap_dek** |

## Exporters (Schema Linking)

| Task | Tools |
|------|-------|
| Manage exporters | **create_exporter**, **get_exporter**, **update_exporter**, **delete_exporter**, **list_exporters** |
| Exporter status | **get_exporter_status**, **pause_exporter**, **resume_exporter**, **reset_exporter** |
| Exporter config | **get_exporter_config**, **update_exporter_config** |

## Working with Contexts

| Task | Tools |
|------|-------|
| List all contexts | **list_contexts** |
| List subjects in a context | **list_subjects** with `context` parameter |
| Register in a context | **register_schema** with `context` parameter |
| Browse context data via resources | `schema://contexts/{context}/subjects`, `schema://contexts/{context}/config` |
| Import schemas | **import_schemas** |

## System and Monitoring

| Task | Tools |
|------|-------|
| Health check | **health_check** |
| Server info | **get_server_info**, **get_server_version**, **get_cluster_id** |
| Registry statistics | **get_registry_statistics**, **count_subjects**, **count_versions** |
| Schema types | **get_schema_types** |
| Export schema/subject | **export_schema**, **export_subject** |
| Schema history | **get_schema_history** |
