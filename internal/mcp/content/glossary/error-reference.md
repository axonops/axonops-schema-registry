# Error Reference

## Response Format

All errors from the schema registry follow the Confluent-compatible format:

```json
{"error_code": <int>, "message": "<string>"}
```

MCP tool errors use `TextContent` with `IsError: true` and a textual error message.

---

## Schema Errors (422xx)

| Code | Name | Description | Diagnostic Tools |
|------|------|-------------|-----------------|
| 42201 | Invalid Schema | Schema failed syntax validation | **validate_schema** |
| 42202 | Invalid Version | Version number is not valid | **list_versions** |
| 42203 | Invalid Compatibility Level | Unrecognized compatibility level | **get_config** |
| 42204 | Invalid Mode | Unrecognized mode | **get_mode** |
| 42205 | Operation Not Permitted | Current mode forbids this operation (e.g., READONLY) | **get_mode**, **set_mode** |
| 42206 | Reference Exists | Cannot delete; other schemas reference this one | **get_referenced_by** |

## Not Found Errors (404xx)

| Code | Name | Description | Diagnostic Tools |
|------|------|-------------|-----------------|
| 40401 | Subject Not Found | Subject does not exist | **list_subjects**, **match_subjects** |
| 40402 | Version Not Found | Version does not exist for this subject | **list_versions** |
| 40403 | Schema Not Found | No schema with this global ID | **get_max_schema_id** |
| 40404 | Subject Soft-Deleted | Subject exists but is soft-deleted | **list_subjects** with `deleted: true` |
| 40405 | Subject Not Soft-Deleted | Cannot permanently delete without soft-deleting first | **delete_subject** (two-stage) |
| 40406 | Schema Version Soft-Deleted | Specific version has been soft-deleted | **list_versions** with `deleted: true` |
| 40407 | Version Not Soft-Deleted | Cannot permanently delete version without soft-deleting first | **delete_version** (two-stage) |
| 40408 | Subject Compat Config Not Found | No per-subject compatibility override | **get_config** (returns inherited value) |
| 40409 | Subject Mode Not Found | No per-subject mode override | **get_mode** (returns inherited value) |

## Compatibility Error

| Code | Name | Description | Diagnostic Tools |
|------|------|-------------|-----------------|
| 409 | Incompatible Schema | New schema is not compatible with existing versions | **explain_compatibility_failure**, **check_compatibility**, **suggest_compatible_change** |

## Server Errors (5xxxx)

| Code | Name | Description | Diagnostic Tools |
|------|------|-------------|-----------------|
| 50001 | Internal Server Error | Unexpected server error | **health_check** |
| 50002 | Storage Backend Error | Storage backend unreachable or returned error | **health_check** |

## Auth Errors

| Code | Name | Description | Diagnostic Tools |
|------|------|-------------|-----------------|
| 40101 | Unauthorized | No valid credentials provided | Check auth token/API key |
| 40103 | API Key Expired | API key has expired | **rotate_apikey**, **create_apikey** |
| 40104 | API Key Disabled | API key has been disabled | **update_apikey** (admin) |
| 40105 | User Disabled | User account has been disabled | **update_user** (admin) |
| 40301 | Forbidden | Valid credentials but insufficient permissions | **list_roles** |

## Encryption Errors

| Code | Name | Description | Diagnostic Tools |
|------|------|-------------|-----------------|
| 40470 | KEK Not Found | Key Encryption Key does not exist | **list_keks** |
| 40471 | DEK Not Found | Data Encryption Key does not exist | **list_deks** |
| 40970 | KEK Already Exists | KEK with this name already exists | **get_kek** |
| 40971 | DEK Already Exists | DEK for this subject already exists | **get_dek** |

## Exporter Errors

| Code | Name | Description | Diagnostic Tools |
|------|------|-------------|-----------------|
| 40450 | Exporter Not Found | Exporter does not exist | **list_exporters** |
| 40950 | Exporter Already Exists | Exporter with this name already exists | **get_exporter** |

---

## Diagnostic Decision Tree

```
Error received
  |
  +-- 404xx (Not Found)
  |     +-- Check names and spelling (list_subjects, match_subjects)
  |     +-- Check soft-deletion state (list_subjects with deleted: true)
  |     +-- Check you are in the correct context
  |
  +-- 409 (Incompatible)
  |     +-- explain_compatibility_failure for details
  |     +-- suggest_compatible_change for fixes
  |     +-- check_compatibility to validate proposed changes
  |
  +-- 422xx (Validation)
  |     +-- validate_schema for syntax errors
  |     +-- get_config / get_mode for config errors
  |
  +-- 401/403 (Auth)
  |     +-- Check credentials and API key status
  |     +-- list_roles to check required permissions
  |
  +-- 500xx (Server)
        +-- health_check to verify server status
        +-- Check storage backend connectivity
```
