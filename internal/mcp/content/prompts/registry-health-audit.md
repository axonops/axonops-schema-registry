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
