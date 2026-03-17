Diagnostic guide for schema registration errors.

## Diagnostic Decision Tree

1. **Is it a 4xx error?** Check the error code category:
   - 404xx: Resource not found -- check names, soft-deletion state
   - 409: Incompatibility -- use **explain_compatibility_failure**
   - 422xx: Validation error -- check input syntax and values
   - 401/403: Auth error -- check credentials and role
2. **Is it a 5xx error?** Server-side issue:
   - 500xx: Use **health_check**, check storage backend connectivity
3. **Unknown error?** Use **health_check** + **get_server_info** to verify the server is operational

---

## Schema Errors (422xx)

### 42201 -- Invalid Schema
The schema failed syntax validation.
- Use **validate_schema** to get a detailed error message
- For Avro: ensure `type`, `name`, and `fields` are present for records
- For Protobuf: ensure `syntax = "proto3";` is the first line
- For JSON Schema: ensure `type` is a valid JSON Schema type
- Check for malformed JSON (brackets, quotes, commas)
- Check for escape character issues in schema strings

### 42202 -- Invalid Version
The version number is not valid.
- Use **list_versions** to see valid version numbers
- Version must be a positive integer or the string "latest"

### 42203 -- Invalid Compatibility Level
The requested compatibility level is not recognized.
- Valid levels: NONE, BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE
- Use **get_config** to check the current level

### 42204 -- Invalid Mode
The requested mode is not valid.
- Valid modes: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT
- Use **get_mode** to check the current mode

### 42205 -- Operation Not Permitted
The current mode forbids this operation.
- Use **get_mode** to check the mode -- is it READONLY?
- Switch to READWRITE with **set_mode** if registration is needed
- Or use READONLY_OVERRIDE for temporary write access

### 42206 -- Reference Exists
Cannot delete because other schemas reference this one.
- Use **get_referenced_by** to find dependent schemas
- Delete or update dependents before deleting this schema

---

## Not Found Errors (404xx)

### 40401 -- Subject Not Found
The subject does not exist.
- Use **list_subjects** to see all subjects
- Use **match_subjects** with a fuzzy pattern to find misspelled subjects
- Check with `deleted: true` -- the subject may be soft-deleted
- Verify you are in the correct context (pass `context` parameter)

### 40402 -- Version Not Found
The version does not exist for this subject.
- Use **list_versions** to see available versions
- Version may have been soft-deleted

### 40403 -- Schema Not Found
No schema exists with this global ID.
- Use **get_max_schema_id** to check the highest assigned ID
- The schema may have been permanently deleted

### 40404 -- Subject Soft-Deleted
The subject exists but has been soft-deleted.
- Use **list_subjects** with `deleted: true` to confirm
- Re-register a schema to recreate the subject
- Or permanently delete with **delete_subject** and `permanent: true`

### 40405 -- Subject Not Soft-Deleted
Cannot permanently delete a subject that has not been soft-deleted first.
- First soft-delete: **delete_subject** (without `permanent: true`)
- Then permanently delete: **delete_subject** with `permanent: true`

### 40406 -- Schema Version Soft-Deleted
The specific version has been soft-deleted.

### 40407 -- Version Not Soft-Deleted
Cannot permanently delete a version that has not been soft-deleted first.

### 40408 -- Subject Compatibility Config Not Found
No per-subject compatibility override exists.
- The subject uses the inherited config (context global or server default)
- Use **get_config** to see the effective (resolved) config

### 40409 -- Subject Mode Not Found
No per-subject mode override exists.
- The subject uses the inherited mode
- Use **get_mode** to see the effective (resolved) mode

---

## Compatibility Error (409)

### 409 -- Incompatible Schema
The new schema is not compatible with existing versions under the current compatibility level.
- Use **explain_compatibility_failure** to get a detailed explanation
- Use **check_compatibility** to see the specific incompatible changes
- Use **get_config** to check the compatibility level
- Use **get_latest_schema** to compare with the current schema
- Use **suggest_compatible_change** for AI-suggested fixes
- Common fixes: add default values, make new fields optional, avoid removing fields

---

## Server Errors (5xxxx)

### 50001 -- Internal Server Error
An unexpected server error occurred.
- Use **health_check** to verify the server is running
- Check server logs for details
- May indicate a storage backend connectivity issue

### 50002 -- Storage Backend Error
The storage backend is unreachable or returned an error.
- Use **health_check** to check connectivity
- Verify storage backend (PostgreSQL, MySQL, Cassandra) is running and accessible

---

## Auth Errors (401/403)

### 40101 -- Unauthorized
No valid credentials provided.
- Check that your auth token or API key is correct
- Verify the auth method is enabled in the server configuration

### 40103 -- API Key Expired
The API key has expired.
- Use **rotate_apikey** to create a new key (requires admin access)
- Or create a new API key with **create_apikey**

### 40104 -- API Key Disabled
The API key has been disabled.
- An admin must re-enable it with **update_apikey**

### 40105 -- User Disabled
The user account has been disabled.
- An admin must re-enable it with **update_user**

### 40301 -- Forbidden
Valid credentials but insufficient permissions.
- Check the user's role -- use **list_roles** to see available roles
- Request role upgrade from an admin

---

## Encryption Errors

### 40470 -- KEK Not Found
The Key Encryption Key does not exist.
- Use **list_keks** to see available KEKs
- Check for typos in the KEK name

### 40471 -- DEK Not Found
The Data Encryption Key does not exist.
- Use **list_deks** to see DEKs for a KEK

### 40970 -- KEK Already Exists
A KEK with this name already exists.
- Use **get_kek** to inspect the existing KEK

### 40971 -- DEK Already Exists
A DEK for this subject already exists under this KEK.
- Use **get_dek** to inspect the existing DEK

---

## Exporter Errors

### 40450 -- Exporter Not Found
The exporter does not exist.
- Use **list_exporters** to see available exporters

### 40950 -- Exporter Already Exists
An exporter with this name already exists.
- Use **get_exporter** to inspect the existing exporter
