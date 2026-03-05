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
