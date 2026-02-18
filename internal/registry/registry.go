// Package registry provides the core schema registry service.
package registry

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// ErrIncompatibleSchema is returned when a schema fails compatibility checks.
var ErrIncompatibleSchema = errors.New("incompatible schema")

// ErrImportIDConflict is returned when importing a schema with an ID that already
// exists but has different content.
var ErrImportIDConflict = errors.New("import ID conflict")

// Registry is the core schema registry service.
type Registry struct {
	storage       storage.Storage
	schemaParser  *schema.Registry
	compatChecker *compatibility.Checker
	defaultConfig string
}

// New creates a new Registry.
func New(store storage.Storage, parser *schema.Registry, compatChecker *compatibility.Checker, defaultCompatibility string) *Registry {
	return &Registry{
		storage:       store,
		schemaParser:  parser,
		compatChecker: compatChecker,
		defaultConfig: defaultCompatibility,
	}
}

// RegisterOpts holds optional parameters for schema registration.
type RegisterOpts struct {
	Normalize bool
	Metadata  *storage.Metadata
	RuleSet   *storage.RuleSet
}

// RegisterSchema registers a new schema for a subject.
func (r *Registry) RegisterSchema(ctx context.Context, registryCtx string, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, opts ...RegisterOpts) (*storage.SchemaRecord, error) {
	// Default to Avro if not specified
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	// Get the parser for this schema type
	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	// Resolve reference content from storage
	resolvedRefs, err := r.resolveReferences(ctx, registryCtx, refs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve references: %w", err)
	}

	// Parse the schema
	parsed, err := parser.Parse(schemaStr, resolvedRefs)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Extract options
	var opt RegisterOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Apply normalization if requested (or if subject config has normalize=true)
	shouldNormalize := opt.Normalize
	if !shouldNormalize {
		shouldNormalize = r.isNormalizeEnabled(ctx, registryCtx, subject)
	}
	if shouldNormalize {
		parsed = parsed.Normalize()
		schemaStr = parsed.CanonicalString()
	}

	// Check if schema already exists with same fingerprint AND same metadata/ruleSet.
	// Confluent behavior: same schema text + different metadata = new version (same global ID).
	// Only deduplicate when both the schema text AND metadata/ruleSet match.
	// Note: confluent:version is stripped from both sides during comparison (it's a CAS property).
	existing, err := r.storage.GetSchemaByFingerprint(ctx, registryCtx, subject, parsed.Fingerprint(), false)
	if err == nil && existing != nil {
		if metadataEqualForDedup(existing.Metadata, opt.Metadata) && ruleSetEqual(existing.RuleSet, opt.RuleSet) {
			return autoPopulateConfluentVersion(existing), nil
		}
		// Same schema text but different metadata/ruleSet — fall through to create new version
	}

	// Get compatibility level for this subject
	compatLevel, err := r.GetConfig(ctx, registryCtx, subject)
	if err != nil {
		compatLevel = r.defaultConfig
	}

	// Check compatibility if not NONE
	mode := compatibility.Mode(compatLevel)
	if mode != compatibility.ModeNone {
		// Get existing schemas for compatibility check
		existingSchemas, err := r.storage.GetSchemasBySubject(ctx, registryCtx, subject, false)
		if err != nil && !errors.Is(err, storage.ErrSubjectNotFound) {
			return nil, fmt.Errorf("failed to get existing schemas: %w", err)
		}

		// Filter by compatibility group if configured
		existingSchemas = r.filterByCompatibilityGroup(ctx, registryCtx, subject, opt.Metadata, existingSchemas)

		if len(existingSchemas) > 0 {
			// Build existing schemas with resolved references
			existingWithRefs := make([]compatibility.SchemaWithRefs, len(existingSchemas))
			for i, s := range existingSchemas {
				existingResolvedRefs, resolveErr := r.resolveReferences(ctx, registryCtx, s.References)
				if resolveErr != nil {
					return nil, fmt.Errorf("failed to resolve existing schema references: %w", resolveErr)
				}
				existingWithRefs[i] = compatibility.SchemaWithRefs{
					Schema:     s.Schema,
					References: existingResolvedRefs,
				}
			}

			// Check compatibility
			result := r.compatChecker.Check(mode, schemaType,
				compatibility.SchemaWithRefs{Schema: schemaStr, References: resolvedRefs},
				existingWithRefs)
			if !result.IsCompatible {
				return nil, fmt.Errorf("%w: %s", ErrIncompatibleSchema, strings.Join(result.Messages, "; "))
			}
		}
	}

	// Validate reserved fields if enabled
	if r.isValidateFieldsEnabled(ctx, registryCtx, subject) {
		if msgs := r.validateReservedFields(ctx, registryCtx, subject, parsed, opt.Metadata); len(msgs) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrIncompatibleSchema, strings.Join(msgs, "; "))
		}
	}

	// Check confluent:version (compare-and-set) if present in metadata
	if err := r.checkConfluentVersion(ctx, registryCtx, subject, opt.Metadata); err != nil {
		return nil, err
	}

	// Apply config default/override merge for metadata and ruleSet (Confluent 3-layer merge).
	// This merges: config.default → request-specific (or previous version) → config.override.
	prevSchema, _ := r.storage.GetLatestSchema(ctx, registryCtx, subject)
	r.maybeSetMetadataRuleSet(ctx, registryCtx, subject, &opt, prevSchema)

	// Strip confluent:version from metadata before storage — it's a CAS control property,
	// not a permanent metadata field. Will be auto-populated in the response.
	opt.Metadata = stripConfluentVersion(opt.Metadata)

	// Create new schema record
	record := &storage.SchemaRecord{
		Subject:     subject,
		SchemaType:  schemaType,
		Schema:      schemaStr,
		References:  refs,
		Metadata:    opt.Metadata,
		RuleSet:     opt.RuleSet,
		Fingerprint: parsed.Fingerprint(),
	}

	// Store the schema
	if err := r.storage.CreateSchema(ctx, registryCtx, record); err != nil {
		if errors.Is(err, storage.ErrSchemaExists) {
			// Storage detected same fingerprint+metadata — return existing
			existing, _ := r.storage.GetSchemaByFingerprint(ctx, registryCtx, subject, parsed.Fingerprint(), false)
			if existing != nil && metadataEqual(existing.Metadata, opt.Metadata) && ruleSetEqual(existing.RuleSet, opt.RuleSet) {
				return autoPopulateConfluentVersion(existing), nil
			}
		}
		return nil, fmt.Errorf("failed to store schema: %w", err)
	}

	return autoPopulateConfluentVersion(record), nil
}

// ErrVersionConflict is returned when confluent:version compare-and-set fails.
var ErrVersionConflict = errors.New("version conflict")

// checkConfluentVersion validates the confluent:version metadata property if present.
// When set to a positive integer, it enforces that the version matches the expected
// next version for the subject (optimistic concurrency control).
// Values of 0 or -1 mean auto-increment (no check).
// Includes soft-deleted versions when determining the latest version, since
// soft-deleted versions still occupy the version number space.
func (r *Registry) checkConfluentVersion(ctx context.Context, registryCtx string, subject string, metadata *storage.Metadata) error {
	if metadata == nil || metadata.Properties == nil {
		return nil
	}
	cvStr, ok := metadata.Properties["confluent:version"]
	if !ok {
		return nil
	}
	cv, err := strconv.Atoi(cvStr)
	if err != nil {
		return nil // Non-numeric value, ignore
	}
	if cv <= 0 {
		return nil // 0 or -1 = auto-increment
	}

	// Include soft-deleted versions to get the true latest version number.
	// Soft-deleted versions still occupy the version sequence (Confluent behavior).
	allSchemas, err := r.storage.GetSchemasBySubject(ctx, registryCtx, subject, true)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			// Truly new subject — only version 1 is valid
			if cv != 1 {
				return fmt.Errorf("%w: confluent:version %d but subject is new (expected 1)", ErrVersionConflict, cv)
			}
			return nil
		}
		return err
	}

	if len(allSchemas) == 0 {
		// No versions at all — only version 1 is valid
		if cv != 1 {
			return fmt.Errorf("%w: confluent:version %d but subject is new (expected 1)", ErrVersionConflict, cv)
		}
		return nil
	}

	// Find the highest version number (including soft-deleted)
	maxVersion := 0
	for _, s := range allSchemas {
		if s.Version > maxVersion {
			maxVersion = s.Version
		}
	}

	if cv != maxVersion+1 {
		return fmt.Errorf("%w: confluent:version %d but latest version is %d (expected %d)", ErrVersionConflict, cv, maxVersion, maxVersion+1)
	}
	return nil
}

// RegisterSchemaWithID registers a schema with a specific ID (for IMPORT mode).
// It validates the schema, determines the next version, and stores it with the given ID.
// Confluent behavior: if the ID already exists with the same schema content, the schema
// is associated with the new subject (succeeds). If the ID exists with different content,
// returns error code 42205.
func (r *Registry) RegisterSchemaWithID(ctx context.Context, registryCtx string, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, id int64) (*storage.SchemaRecord, error) {
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	resolvedRefs, err := r.resolveReferences(ctx, registryCtx, refs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve references: %w", err)
	}

	parsed, err := parser.Parse(schemaStr, resolvedRefs)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Check if schema already exists in this subject with same fingerprint (idempotent)
	existing, err := r.storage.GetSchemaByFingerprint(ctx, registryCtx, subject, parsed.Fingerprint(), false)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Determine next version for this subject.
	// Include soft-deleted versions to avoid version number conflicts
	// with rows that still physically exist in storage.
	nextVersion := 1
	existingSchemas, err := r.storage.GetSchemasBySubject(ctx, registryCtx, subject, true)
	if err == nil && len(existingSchemas) > 0 {
		for _, s := range existingSchemas {
			if s.Version >= nextVersion {
				nextVersion = s.Version + 1
			}
		}
	}

	record := &storage.SchemaRecord{
		ID:          id,
		Subject:     subject,
		Version:     nextVersion,
		SchemaType:  schemaType,
		Schema:      schemaStr,
		References:  refs,
		Fingerprint: parsed.Fingerprint(),
	}

	if err := r.storage.ImportSchema(ctx, registryCtx, record); err != nil {
		if errors.Is(err, storage.ErrSchemaIDConflict) {
			return nil, fmt.Errorf("overwrite schema with id %d: %w", id, ErrImportIDConflict)
		}
		return nil, fmt.Errorf("failed to store schema: %w", err)
	}

	// Advance the ID sequence so future auto-assigned IDs don't collide.
	// Only advance forward, never rewind.
	nextID := id + 1
	currentMax, err := r.storage.GetMaxSchemaID(ctx, registryCtx)
	if err == nil && currentMax+1 > nextID {
		nextID = currentMax + 1
	}
	if err := r.storage.SetNextID(ctx, registryCtx, nextID); err != nil {
		return record, fmt.Errorf("schema stored but failed to advance ID sequence: %w", err)
	}

	return record, nil
}

// CheckCompatibility checks if a schema is compatible with a specific version or all versions.
func (r *Registry) CheckCompatibility(ctx context.Context, registryCtx string, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, version string, normalize ...bool) (*compatibility.Result, error) {
	// Default to Avro if not specified
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	// Parse the new schema to validate it
	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	// Resolve reference content from storage
	resolvedRefs, err := r.resolveReferences(ctx, registryCtx, refs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve references: %w", err)
	}

	parsed, err := parser.Parse(schemaStr, resolvedRefs)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Apply normalization if requested
	shouldNormalize := len(normalize) > 0 && normalize[0]
	if !shouldNormalize {
		shouldNormalize = r.isNormalizeEnabled(ctx, registryCtx, subject)
	}
	if shouldNormalize {
		parsed = parsed.Normalize()
		schemaStr = parsed.CanonicalString()
	}

	// Get compatibility level
	compatLevel, err := r.GetConfig(ctx, registryCtx, subject)
	if err != nil {
		compatLevel = r.defaultConfig
	}

	mode := compatibility.Mode(compatLevel)
	if mode == compatibility.ModeNone {
		return compatibility.NewCompatibleResult(), nil
	}

	// Get schemas to check against
	var schemasToCheck []compatibility.SchemaWithRefs

	if version == "latest" {
		// Check against latest version only
		latest, err := r.storage.GetLatestSchema(ctx, registryCtx, subject)
		if err != nil {
			if errors.Is(err, storage.ErrSubjectNotFound) {
				// No existing schemas, always compatible
				return compatibility.NewCompatibleResult(), nil
			}
			return nil, err
		}
		latestRefs, resolveErr := r.resolveReferences(ctx, registryCtx, latest.References)
		if resolveErr != nil {
			return nil, fmt.Errorf("failed to resolve existing schema references: %w", resolveErr)
		}
		schemasToCheck = []compatibility.SchemaWithRefs{{Schema: latest.Schema, References: latestRefs}}
	} else if version == "" {
		// Empty version means check against all versions (transitive compatibility)
		existingSchemas, err := r.storage.GetSchemasBySubject(ctx, registryCtx, subject, false)
		if err != nil {
			if errors.Is(err, storage.ErrSubjectNotFound) {
				return compatibility.NewCompatibleResult(), nil
			}
			return nil, err
		}

		for _, s := range existingSchemas {
			existingRefs, resolveErr := r.resolveReferences(ctx, registryCtx, s.References)
			if resolveErr != nil {
				return nil, fmt.Errorf("failed to resolve existing schema references: %w", resolveErr)
			}
			schemasToCheck = append(schemasToCheck, compatibility.SchemaWithRefs{Schema: s.Schema, References: existingRefs})
		}
	} else {
		// Check against specific version only
		versionNum, err := strconv.Atoi(version)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", storage.ErrInvalidVersion, version)
		}
		schema, err := r.storage.GetSchemaBySubjectVersion(ctx, registryCtx, subject, versionNum)
		if err != nil {
			if errors.Is(err, storage.ErrSubjectNotFound) {
				return nil, fmt.Errorf("%w: %s", storage.ErrSubjectNotFound, subject)
			}
			if errors.Is(err, storage.ErrVersionNotFound) {
				return nil, fmt.Errorf("%w: version %d for subject %s", storage.ErrVersionNotFound, versionNum, subject)
			}
			return nil, err
		}
		schemaRefs, resolveErr := r.resolveReferences(ctx, registryCtx, schema.References)
		if resolveErr != nil {
			return nil, fmt.Errorf("failed to resolve existing schema references: %w", resolveErr)
		}
		schemasToCheck = []compatibility.SchemaWithRefs{{Schema: schema.Schema, References: schemaRefs}}
	}

	if len(schemasToCheck) == 0 {
		return compatibility.NewCompatibleResult(), nil
	}

	return r.compatChecker.Check(mode, schemaType,
		compatibility.SchemaWithRefs{Schema: schemaStr, References: resolvedRefs},
		schemasToCheck), nil
}

// GetSchemaByID retrieves a schema by its ID within a context.
func (r *Registry) GetSchemaByID(ctx context.Context, registryCtx string, id int64) (*storage.SchemaRecord, error) {
	return r.storage.GetSchemaByID(ctx, registryCtx, id)
}

// GetMaxSchemaID returns the highest schema ID currently assigned in a context.
func (r *Registry) GetMaxSchemaID(ctx context.Context, registryCtx string) (int64, error) {
	return r.storage.GetMaxSchemaID(ctx, registryCtx)
}

// FormatSchema parses a schema record and returns it formatted according to the given format.
// Returns the original schema string if format is empty or parsing fails.
func (r *Registry) FormatSchema(ctx context.Context, registryCtx string, record *storage.SchemaRecord, format string) string {
	if format == "" {
		return record.Schema
	}

	schemaType := record.SchemaType
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return record.Schema
	}

	// Resolve references
	refs := record.References
	resolvedRefs := make([]storage.Reference, len(refs))
	copy(resolvedRefs, refs)
	for i, ref := range resolvedRefs {
		if ref.Schema == "" {
			refSchema, err := r.storage.GetSchemaBySubjectVersion(ctx, registryCtx, ref.Subject, ref.Version)
			if err == nil {
				resolvedRefs[i].Schema = refSchema.Schema
			}
		}
	}

	parsed, err := parser.Parse(record.Schema, resolvedRefs)
	if err != nil {
		return record.Schema
	}

	return parsed.FormattedString(format)
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version.
func (r *Registry) GetSchemaBySubjectVersion(ctx context.Context, registryCtx string, subject string, version int) (*storage.SchemaRecord, error) {
	return r.storage.GetSchemaBySubjectVersion(ctx, registryCtx, subject, version)
}

// GetSchemasBySubject returns all schemas for a subject, optionally including deleted.
func (r *Registry) GetSchemasBySubject(ctx context.Context, registryCtx string, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	return r.storage.GetSchemasBySubject(ctx, registryCtx, subject, includeDeleted)
}

// ListSubjects returns all subject names within a context.
func (r *Registry) ListSubjects(ctx context.Context, registryCtx string, deleted bool) ([]string, error) {
	return r.storage.ListSubjects(ctx, registryCtx, deleted)
}

// GetVersions returns all versions for a subject within a context.
func (r *Registry) GetVersions(ctx context.Context, registryCtx string, subject string, deleted bool) ([]int, error) {
	schemas, err := r.storage.GetSchemasBySubject(ctx, registryCtx, subject, deleted)
	if err != nil {
		return nil, err
	}

	versions := make([]int, 0, len(schemas))
	for _, s := range schemas {
		versions = append(versions, s.Version)
	}

	return versions, nil
}

// LookupSchema finds a schema in a subject within a context.
func (r *Registry) LookupSchema(ctx context.Context, registryCtx string, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, deleted bool, normalize ...bool) (*storage.SchemaRecord, error) {
	// Default to Avro if not specified
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	// Get the parser for this schema type
	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	// Resolve reference content from storage
	resolvedRefs, err := r.resolveReferences(ctx, registryCtx, refs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve references: %w", err)
	}

	// Parse the schema to get fingerprint
	parsed, err := parser.Parse(schemaStr, resolvedRefs)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Apply normalization if requested
	shouldNormalize := len(normalize) > 0 && normalize[0]
	if !shouldNormalize {
		shouldNormalize = r.isNormalizeEnabled(ctx, registryCtx, subject)
	}
	if shouldNormalize {
		parsed = parsed.Normalize()
	}

	// Look up by fingerprint, including deleted if requested
	return r.storage.GetSchemaByFingerprint(ctx, registryCtx, subject, parsed.Fingerprint(), deleted)
}

// DeleteSubject deletes a subject within a context.
func (r *Registry) DeleteSubject(ctx context.Context, registryCtx string, subject string, permanent bool) ([]int, error) {
	if !permanent {
		// For soft-delete, check if any version in this subject is referenced by other schemas.
		// If the subject doesn't exist or is already deleted, skip the check and let
		// DeleteSubject return the appropriate error.
		schemas, err := r.storage.GetSchemasBySubject(ctx, registryCtx, subject, false)
		if err == nil {
			for _, schema := range schemas {
				refs, err := r.storage.GetReferencedBy(ctx, registryCtx, subject, schema.Version)
				if err != nil {
					return nil, err
				}
				if len(refs) > 0 {
					return nil, fmt.Errorf("One or more references exist to the schema {subject=%s,version=%d}", subject, schema.Version)
				}
			}
		}
	}
	versions, err := r.storage.DeleteSubject(ctx, registryCtx, subject, permanent)
	if err != nil {
		return nil, err
	}
	// Only clean up subject-level config and mode on permanent delete.
	// Soft-delete preserves config/mode so re-registration inherits them.
	if permanent {
		_ = r.storage.DeleteConfig(ctx, registryCtx, subject)
		_ = r.storage.DeleteMode(ctx, registryCtx, subject)
	}
	return versions, nil
}

// DeleteVersion deletes a specific version within a context.
func (r *Registry) DeleteVersion(ctx context.Context, registryCtx string, subject string, version int, permanent bool) (int, error) {
	if permanent {
		// For permanent delete, the version must already be soft-deleted.
		// The storage layer validates this (returns ErrVersionNotSoftDeleted if not).
		// We skip GetSchemaBySubjectVersion because it filters out soft-deleted versions.
		if err := r.storage.DeleteSchema(ctx, registryCtx, subject, version, permanent); err != nil {
			return 0, err
		}
		return version, nil
	}

	// Soft-delete: verify the schema exists (non-deleted)
	schema, err := r.storage.GetSchemaBySubjectVersion(ctx, registryCtx, subject, version)
	if err != nil {
		return 0, err
	}

	// Use the resolved version (handles "latest" / -1 → actual version number)
	resolvedVersion := schema.Version

	// Check for references - only block soft-delete when referenced
	refs, err := r.storage.GetReferencedBy(ctx, registryCtx, subject, resolvedVersion)
	if err != nil {
		return 0, err
	}
	if len(refs) > 0 {
		return 0, fmt.Errorf("schema is referenced by other schemas")
	}

	if err := r.storage.DeleteSchema(ctx, registryCtx, subject, resolvedVersion, permanent); err != nil {
		return 0, err
	}

	return resolvedVersion, nil
}

// GetConfig gets the compatibility configuration for a subject within a context.
// Uses the Confluent-compatible 4-tier fallback chain:
//
//	Step 1: Per-subject config
//	Step 2: Context-level global config
//	Step 3: __GLOBAL context config (cross-context default)
//	Step 4: Server hardcoded default
func (r *Registry) GetConfig(ctx context.Context, registryCtx string, subject string) (string, error) {
	// Step 1: Per-subject config
	if subject != "" {
		config, err := r.storage.GetConfig(ctx, registryCtx, subject)
		if err == nil {
			return config.CompatibilityLevel, nil
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return "", err
		}
	}

	// Step 2: Context-level global config
	config, err := r.storage.GetGlobalConfig(ctx, registryCtx)
	if err == nil {
		return config.CompatibilityLevel, nil
	}

	// Step 3: __GLOBAL context config (skip if already querying __GLOBAL)
	if registryCtx != registrycontext.GlobalContext {
		config, err = r.storage.GetGlobalConfig(ctx, registrycontext.GlobalContext)
		if err == nil {
			return config.CompatibilityLevel, nil
		}
	}

	// Step 4: Server hardcoded default
	return r.defaultConfig, nil
}

// GetSubjectConfig gets the compatibility configuration for a specific subject only,
// without falling back to the global default. Returns storage.ErrNotFound if not set.
func (r *Registry) GetSubjectConfig(ctx context.Context, registryCtx string, subject string) (string, error) {
	config, err := r.storage.GetConfig(ctx, registryCtx, subject)
	if err != nil {
		return "", err
	}
	return config.CompatibilityLevel, nil
}

// GetSubjectConfigFull gets the full configuration record for a subject only,
// without falling back to global. Returns storage.ErrNotFound if not set.
func (r *Registry) GetSubjectConfigFull(ctx context.Context, registryCtx string, subject string) (*storage.ConfigRecord, error) {
	return r.storage.GetConfig(ctx, registryCtx, subject)
}

// GetConfigFull gets the full configuration record using the 4-tier fallback chain.
func (r *Registry) GetConfigFull(ctx context.Context, registryCtx string, subject string) (*storage.ConfigRecord, error) {
	// Step 1: Per-subject config
	if subject != "" {
		config, err := r.storage.GetConfig(ctx, registryCtx, subject)
		if err == nil {
			return config, nil
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}
	}

	// Step 2: Context-level global config
	config, err := r.storage.GetGlobalConfig(ctx, registryCtx)
	if err == nil {
		return config, nil
	}

	// Step 3: __GLOBAL context config (skip if already querying __GLOBAL)
	if registryCtx != registrycontext.GlobalContext {
		config, err = r.storage.GetGlobalConfig(ctx, registrycontext.GlobalContext)
		if err == nil {
			return config, nil
		}
	}

	// Step 4: Server hardcoded default
	return &storage.ConfigRecord{CompatibilityLevel: r.defaultConfig}, nil
}

// GetGlobalConfigDirect returns the context's global config without the __GLOBAL fallback.
// Used by the API handler when defaultToGlobal=false and subject is empty.
func (r *Registry) GetGlobalConfigDirect(ctx context.Context, registryCtx string) (*storage.ConfigRecord, error) {
	config, err := r.storage.GetGlobalConfig(ctx, registryCtx)
	if err == nil {
		return config, nil
	}
	return &storage.ConfigRecord{CompatibilityLevel: r.defaultConfig}, nil
}

// SetConfigOpts holds optional fields for configuration updates.
type SetConfigOpts struct {
	Alias              string
	CompatibilityGroup string
	ValidateFields     *bool
	DefaultMetadata    *storage.Metadata
	OverrideMetadata   *storage.Metadata
	DefaultRuleSet     *storage.RuleSet
	OverrideRuleSet    *storage.RuleSet
}

// SetConfig sets the compatibility configuration for a subject within a context.
func (r *Registry) SetConfig(ctx context.Context, registryCtx string, subject string, level string, normalize *bool, opts ...SetConfigOpts) error {
	level = strings.ToUpper(level)
	if !isValidCompatibility(level) {
		return fmt.Errorf("invalid compatibility level: %s", level)
	}

	config := &storage.ConfigRecord{
		CompatibilityLevel: level,
		Normalize:          normalize,
	}

	if len(opts) > 0 {
		opt := opts[0]
		config.Alias = opt.Alias
		config.CompatibilityGroup = opt.CompatibilityGroup
		config.ValidateFields = opt.ValidateFields
		config.DefaultMetadata = opt.DefaultMetadata
		config.OverrideMetadata = opt.OverrideMetadata
		config.DefaultRuleSet = opt.DefaultRuleSet
		config.OverrideRuleSet = opt.OverrideRuleSet
	}

	if subject == "" {
		return r.storage.SetGlobalConfig(ctx, registryCtx, config)
	}

	return r.storage.SetConfig(ctx, registryCtx, subject, config)
}

// DeleteConfig deletes the compatibility configuration for a subject within a context.
// When subject is empty, this deletes the context-level global config.
func (r *Registry) DeleteConfig(ctx context.Context, registryCtx string, subject string) (string, error) {
	if subject == "" {
		return r.DeleteGlobalConfig(ctx, registryCtx)
	}

	config, err := r.storage.GetConfig(ctx, registryCtx, subject)
	if err != nil {
		return "", err
	}

	if err := r.storage.DeleteConfig(ctx, registryCtx, subject); err != nil {
		return "", err
	}

	return config.CompatibilityLevel, nil
}

// GetMode gets the mode for a subject within a context using the 4-tier fallback chain.
// Also implements the Confluent READONLY_OVERRIDE kill switch: if the default context's
// resolved global mode is READONLY_OVERRIDE, it overrides all per-subject/per-context modes.
func (r *Registry) GetMode(ctx context.Context, registryCtx string, subject string) (string, error) {
	// Confluent behavior: READONLY_OVERRIDE on default context global is a kill switch
	// that overrides everything. Check it first.
	globalMode := r.resolveGlobalMode(ctx)
	if globalMode == "READONLY_OVERRIDE" {
		return "READONLY_OVERRIDE", nil
	}

	// Step 1: Per-subject mode
	if subject != "" {
		mode, err := r.storage.GetMode(ctx, registryCtx, subject)
		if err == nil {
			return mode.Mode, nil
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return "", err
		}
	}

	// Step 2: Context-level global mode
	mode, err := r.storage.GetGlobalMode(ctx, registryCtx)
	if err == nil {
		return mode.Mode, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return "", fmt.Errorf("failed to get global mode: %w", err)
	}

	// Step 3: __GLOBAL context mode (skip if already querying __GLOBAL)
	if registryCtx != registrycontext.GlobalContext {
		mode, err = r.storage.GetGlobalMode(ctx, registrycontext.GlobalContext)
		if err == nil {
			return mode.Mode, nil
		}
	}

	// Step 4: Default
	return "READWRITE", nil
}

// resolveGlobalMode resolves the "global" mode by walking the default context chain.
// This is used for the READONLY_OVERRIDE kill switch check (Confluent compatibility).
// Chain: default context global mode → __GLOBAL context mode → READWRITE
func (r *Registry) resolveGlobalMode(ctx context.Context) string {
	// Default context global mode
	mode, err := r.storage.GetGlobalMode(ctx, registrycontext.DefaultContext)
	if err == nil {
		return mode.Mode
	}
	// __GLOBAL context mode
	mode, err = r.storage.GetGlobalMode(ctx, registrycontext.GlobalContext)
	if err == nil {
		return mode.Mode
	}
	return "READWRITE"
}

// GetGlobalModeDirect returns the context's global mode without the __GLOBAL fallback.
// Used by the API handler when defaultToGlobal=false and subject is empty.
func (r *Registry) GetGlobalModeDirect(ctx context.Context, registryCtx string) (string, error) {
	mode, err := r.storage.GetGlobalMode(ctx, registryCtx)
	if err == nil {
		return mode.Mode, nil
	}
	if errors.Is(err, storage.ErrNotFound) {
		return "READWRITE", nil
	}
	return "", fmt.Errorf("failed to get global mode: %w", err)
}

// GetSubjectMode gets the mode for a specific subject only,
// without falling back to the global default. Returns storage.ErrNotFound if not set.
func (r *Registry) GetSubjectMode(ctx context.Context, registryCtx string, subject string) (string, error) {
	mode, err := r.storage.GetMode(ctx, registryCtx, subject)
	if err != nil {
		return "", err
	}
	return mode.Mode, nil
}

// SetMode sets the mode for a subject within a context.
// If switching to IMPORT mode and force is false, it checks that no schemas exist.
func (r *Registry) SetMode(ctx context.Context, registryCtx string, subject string, mode string, force bool) error {
	mode = strings.ToUpper(mode)
	if !isValidMode(mode) {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	// Confluent behavior: switching to IMPORT requires force=true if schemas exist
	if mode == "IMPORT" && !force {
		currentMode, _ := r.GetMode(ctx, registryCtx, subject)
		if currentMode != "IMPORT" {
			hasSchemas, err := r.hasSubjects(ctx, registryCtx, subject)
			if err != nil {
				return err
			}
			if hasSchemas {
				return storage.ErrOperationNotPermitted
			}
		}
	}

	modeRecord := &storage.ModeRecord{
		Mode: mode,
	}

	if subject == "" {
		return r.storage.SetGlobalMode(ctx, registryCtx, modeRecord)
	}

	return r.storage.SetMode(ctx, registryCtx, subject, modeRecord)
}

// isNormalizeEnabled checks if normalization is enabled for a subject via the 4-tier config chain.
func (r *Registry) isNormalizeEnabled(ctx context.Context, registryCtx string, subject string) bool {
	// Step 1: Per-subject config
	if subject != "" {
		config, err := r.storage.GetConfig(ctx, registryCtx, subject)
		if err == nil && config != nil && config.Normalize != nil {
			return *config.Normalize
		}
	}
	// Step 2: Context-level global config
	config, err := r.storage.GetGlobalConfig(ctx, registryCtx)
	if err == nil && config != nil && config.Normalize != nil {
		return *config.Normalize
	}
	// Step 3: __GLOBAL context config
	if registryCtx != registrycontext.GlobalContext {
		config, err = r.storage.GetGlobalConfig(ctx, registrycontext.GlobalContext)
		if err == nil && config != nil && config.Normalize != nil {
			return *config.Normalize
		}
	}
	// Step 4: Default
	return false
}

// hasSubjects checks if any non-deleted schemas exist for the given subject (or globally if subject is empty).
func (r *Registry) hasSubjects(ctx context.Context, registryCtx string, subject string) (bool, error) {
	if subject == "" {
		// Check if any subjects exist in the context
		subjects, err := r.storage.ListSubjects(ctx, registryCtx, false)
		if err != nil {
			return false, err
		}
		return len(subjects) > 0, nil
	}
	// Check if the specific subject has schemas
	exists, err := r.storage.SubjectExists(ctx, registryCtx, subject)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetReferencedBy gets subjects/versions that reference a schema within a context.
func (r *Registry) GetReferencedBy(ctx context.Context, registryCtx string, subject string, version int) ([]storage.SubjectVersion, error) {
	return r.storage.GetReferencedBy(ctx, registryCtx, subject, version)
}

// GetSchemaTypes returns the supported schema types.
func (r *Registry) GetSchemaTypes() []string {
	return r.schemaParser.Types()
}

// GetRawSchemaByID retrieves just the schema string by ID within a context.
func (r *Registry) GetRawSchemaByID(ctx context.Context, registryCtx string, id int64) (string, error) {
	schema, err := r.storage.GetSchemaByID(ctx, registryCtx, id)
	if err != nil {
		return "", err
	}
	return schema.Schema, nil
}

// GetSubjectsBySchemaID returns all subjects where the given schema ID is registered within a context.
func (r *Registry) GetSubjectsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]string, error) {
	return r.storage.GetSubjectsBySchemaID(ctx, registryCtx, id, includeDeleted)
}

// GetVersionsBySchemaID returns all subject-version pairs for a schema ID within a context.
func (r *Registry) GetVersionsBySchemaID(ctx context.Context, registryCtx string, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	return r.storage.GetVersionsBySchemaID(ctx, registryCtx, id, includeDeleted)
}

// ListSchemas returns schemas matching the given filters within a context.
func (r *Registry) ListSchemas(ctx context.Context, registryCtx string, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	return r.storage.ListSchemas(ctx, registryCtx, params)
}

// ListContexts returns all registry context names.
// The __GLOBAL context is filtered out as it is not a real schema context
// (Confluent-compatible: __GLOBAL only holds config/mode settings).
func (r *Registry) ListContexts(ctx context.Context) ([]string, error) {
	contexts, err := r.storage.ListContexts(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]string, 0, len(contexts))
	for _, c := range contexts {
		if c != registrycontext.GlobalContext {
			filtered = append(filtered, c)
		}
	}
	return filtered, nil
}

// ImportSchemaRequest represents a single schema to import with a specified ID.
type ImportSchemaRequest struct {
	ID         int64
	Subject    string
	Version    int
	SchemaType storage.SchemaType
	Schema     string
	References []storage.Reference
}

// ImportSchemaResult represents the result of importing a single schema.
type ImportSchemaResult struct {
	ID      int64
	Subject string
	Version int
	Success bool
	Error   string
}

// ImportResult represents the result of importing multiple schemas.
type ImportResult struct {
	Imported int
	Errors   int
	Results  []ImportSchemaResult
}

// ImportSchemas imports schemas with preserved IDs (for migration) within a context.
// It validates each schema, imports it with the specified ID, and adjusts
// the ID sequence after import to prevent conflicts.
func (r *Registry) ImportSchemas(ctx context.Context, registryCtx string, schemas []ImportSchemaRequest) (*ImportResult, error) {
	result := &ImportResult{
		Results: make([]ImportSchemaResult, len(schemas)),
	}

	var maxID int64

	for i, req := range schemas {
		res := ImportSchemaResult{
			ID:      req.ID,
			Subject: req.Subject,
			Version: req.Version,
		}

		// Validate required fields
		if req.ID <= 0 {
			res.Error = "schema ID must be positive"
			result.Errors++
			result.Results[i] = res
			continue
		}
		if req.Subject == "" {
			res.Error = "subject is required"
			result.Errors++
			result.Results[i] = res
			continue
		}
		if req.Version <= 0 {
			res.Error = "version must be positive"
			result.Errors++
			result.Results[i] = res
			continue
		}
		if req.Schema == "" {
			res.Error = "schema is required"
			result.Errors++
			result.Results[i] = res
			continue
		}

		// Default to Avro if not specified
		schemaType := req.SchemaType
		if schemaType == "" {
			schemaType = storage.SchemaTypeAvro
		}

		// Validate the schema
		parser, ok := r.schemaParser.Get(schemaType)
		if !ok {
			res.Error = fmt.Sprintf("unsupported schema type: %s", schemaType)
			result.Errors++
			result.Results[i] = res
			continue
		}

		// Resolve reference content from storage
		resolvedRefs, resolveErr := r.resolveReferences(ctx, registryCtx, req.References)
		if resolveErr != nil {
			res.Error = fmt.Sprintf("failed to resolve references: %v", resolveErr)
			result.Errors++
			result.Results[i] = res
			continue
		}

		parsed, err := parser.Parse(req.Schema, resolvedRefs)
		if err != nil {
			res.Error = fmt.Sprintf("invalid schema: %v", err)
			result.Errors++
			result.Results[i] = res
			continue
		}

		// Create the schema record
		record := &storage.SchemaRecord{
			ID:          req.ID,
			Subject:     req.Subject,
			Version:     req.Version,
			SchemaType:  schemaType,
			Schema:      req.Schema,
			References:  req.References,
			Fingerprint: parsed.Fingerprint(),
		}

		// Import the schema
		if err := r.storage.ImportSchema(ctx, registryCtx, record); err != nil {
			if errors.Is(err, storage.ErrSchemaIDConflict) {
				res.Error = "schema ID already exists"
			} else if errors.Is(err, storage.ErrSchemaExists) {
				res.Error = "subject/version already exists"
			} else {
				res.Error = err.Error()
			}
			result.Errors++
			result.Results[i] = res
			continue
		}

		// Track the maximum ID for sequence adjustment
		if req.ID > maxID {
			maxID = req.ID
		}

		res.Success = true
		result.Imported++
		result.Results[i] = res
	}

	// Adjust the ID sequence to prevent conflicts.
	// Guard against rewinding: only advance the sequence, never go backward.
	if maxID > 0 {
		nextID := maxID + 1

		// Check current max to avoid rewinding the sequence
		currentMax, err := r.storage.GetMaxSchemaID(ctx, registryCtx)
		if err == nil && currentMax+1 > nextID {
			nextID = currentMax + 1
		}

		if err := r.storage.SetNextID(ctx, registryCtx, nextID); err != nil {
			return result, fmt.Errorf("imported %d schemas but failed to adjust ID sequence: %w", result.Imported, err)
		}
	}

	return result, nil
}

// GetRawSchemaBySubjectVersion retrieves just the schema string by subject and version.
func (r *Registry) GetRawSchemaBySubjectVersion(ctx context.Context, registryCtx string, subject string, version int) (string, error) {
	schema, err := r.storage.GetSchemaBySubjectVersion(ctx, registryCtx, subject, version)
	if err != nil {
		return "", err
	}
	return schema.Schema, nil
}

// DeleteGlobalConfig resets the global compatibility configuration to default for a context.
func (r *Registry) DeleteGlobalConfig(ctx context.Context, registryCtx string) (string, error) {
	config, err := r.storage.GetGlobalConfig(ctx, registryCtx)
	if err != nil {
		// If no config, use default
		config = &storage.ConfigRecord{CompatibilityLevel: r.defaultConfig}
	}
	prevLevel := config.CompatibilityLevel

	if err := r.storage.DeleteGlobalConfig(ctx, registryCtx); err != nil {
		return "", err
	}

	return prevLevel, nil
}

// DeleteMode deletes the mode configuration for a subject within a context.
// When subject is empty, this deletes the context-level global mode.
func (r *Registry) DeleteMode(ctx context.Context, registryCtx string, subject string) (string, error) {
	if subject == "" {
		return r.DeleteGlobalMode(ctx, registryCtx)
	}

	mode, err := r.storage.GetMode(ctx, registryCtx, subject)
	if err != nil {
		return "", err
	}
	prevMode := mode.Mode

	if err := r.storage.DeleteMode(ctx, registryCtx, subject); err != nil {
		return "", err
	}

	return prevMode, nil
}

// DeleteGlobalMode resets the global mode for a context by removing it.
func (r *Registry) DeleteGlobalMode(ctx context.Context, registryCtx string) (string, error) {
	mode, err := r.storage.GetGlobalMode(ctx, registryCtx)
	if err != nil {
		// If no mode, use default
		mode = &storage.ModeRecord{Mode: "READWRITE"}
	}
	prevMode := mode.Mode

	if err := r.storage.DeleteGlobalMode(ctx, registryCtx); err != nil {
		return "", err
	}

	return prevMode, nil
}

// IsHealthy returns whether the registry is healthy.
func (r *Registry) IsHealthy(ctx context.Context) bool {
	return r.storage.IsHealthy(ctx)
}

func isValidCompatibility(level string) bool {
	valid := map[string]bool{
		"NONE":                true,
		"BACKWARD":            true,
		"BACKWARD_TRANSITIVE": true,
		"FORWARD":             true,
		"FORWARD_TRANSITIVE":  true,
		"FULL":                true,
		"FULL_TRANSITIVE":     true,
	}
	return valid[level]
}

func isValidMode(mode string) bool {
	valid := map[string]bool{
		"READWRITE":         true,
		"READONLY":          true,
		"READONLY_OVERRIDE": true,
		"IMPORT":            true,
	}
	return valid[mode]
}

// filterByCompatibilityGroup filters existing schemas by the compatibility group
// metadata property if a compatibilityGroup is configured for the subject.
// The compatibilityGroup config value names a metadata property key; only schemas
// with the same value for that property are included in compatibility checks.
func (r *Registry) filterByCompatibilityGroup(ctx context.Context, registryCtx string, subject string, newMeta *storage.Metadata, schemas []*storage.SchemaRecord) []*storage.SchemaRecord {
	if len(schemas) == 0 {
		return schemas
	}

	// Get the compatibility group property name from config
	config, err := r.GetSubjectConfigFull(ctx, registryCtx, subject)
	if err != nil || config == nil || config.CompatibilityGroup == "" {
		return schemas
	}
	groupKey := config.CompatibilityGroup

	// Get the new schema's value for this property
	var newGroupVal string
	if newMeta != nil && newMeta.Properties != nil {
		newGroupVal = newMeta.Properties[groupKey]
	}

	// Filter existing schemas to those with the same group value
	filtered := make([]*storage.SchemaRecord, 0, len(schemas))
	for _, s := range schemas {
		existingVal := ""
		if s.Metadata != nil && s.Metadata.Properties != nil {
			existingVal = s.Metadata.Properties[groupKey]
		}
		if existingVal == newGroupVal {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// isValidateFieldsEnabled checks if reserved field validation is enabled via the 4-tier config chain.
func (r *Registry) isValidateFieldsEnabled(ctx context.Context, registryCtx string, subject string) bool {
	// Step 1: Per-subject config
	if subject != "" {
		config, err := r.storage.GetConfig(ctx, registryCtx, subject)
		if err == nil && config != nil && config.ValidateFields != nil {
			return *config.ValidateFields
		}
	}
	// Step 2: Context-level global config
	config, err := r.storage.GetGlobalConfig(ctx, registryCtx)
	if err == nil && config != nil && config.ValidateFields != nil {
		return *config.ValidateFields
	}
	// Step 3: __GLOBAL context config
	if registryCtx != registrycontext.GlobalContext {
		config, err = r.storage.GetGlobalConfig(ctx, registrycontext.GlobalContext)
		if err == nil && config != nil && config.ValidateFields != nil {
			return *config.ValidateFields
		}
	}
	// Step 4: Default
	return false
}

// getReservedFields extracts reserved field names from the "confluent:reserved"
// metadata property. Returns an empty set if not present.
func getReservedFields(metadata *storage.Metadata) map[string]bool {
	if metadata == nil || metadata.Properties == nil {
		return nil
	}
	val, ok := metadata.Properties["confluent:reserved"]
	if !ok || val == "" {
		return nil
	}
	fields := make(map[string]bool)
	for _, f := range strings.Split(val, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			fields[f] = true
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// validateReservedFields checks two invariants when validateFields is enabled:
// 1. Reserved fields listed in metadata must not exist as top-level schema fields.
// 2. Reserved fields from the previous version must not be removed.
func (r *Registry) validateReservedFields(ctx context.Context, registryCtx string, subject string, parsed schema.ParsedSchema, metadata *storage.Metadata) []string {
	var msgs []string

	reservedFields := getReservedFields(metadata)

	// Rule 2: Check that reserved fields from previous version are not removed
	latest, err := r.storage.GetLatestSchema(ctx, registryCtx, subject)
	if err == nil && latest != nil {
		prevReserved := getReservedFields(latest.Metadata)
		for field := range prevReserved {
			if !reservedFields[field] {
				msgs = append(msgs, fmt.Sprintf(
					"The new schema has reserved field %s removed from its metadata which is present in the old schema's metadata.", field))
			}
		}
	}

	// Rule 1: Reserved fields must not conflict with actual schema fields
	for field := range reservedFields {
		if parsed.HasTopLevelField(field) {
			msgs = append(msgs, fmt.Sprintf(
				"The new schema has field that conflicts with the reserved field %s.", field))
		}
	}

	return msgs
}

// resolveReferences looks up the schema content for each reference from storage.
func (r *Registry) resolveReferences(ctx context.Context, registryCtx string, refs []storage.Reference) ([]storage.Reference, error) {
	if len(refs) == 0 {
		return refs, nil
	}
	resolved := make([]storage.Reference, len(refs))
	for i, ref := range refs {
		record, err := r.storage.GetSchemaBySubjectVersion(ctx, registryCtx, ref.Subject, ref.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve reference %q (subject=%s, version=%d): %w",
				ref.Name, ref.Subject, ref.Version, err)
		}
		resolved[i] = storage.Reference{
			Name:    ref.Name,
			Subject: ref.Subject,
			Version: ref.Version,
			Schema:  record.Schema,
		}
	}
	return resolved, nil
}

// metadataEqual compares two Metadata pointers for equality.
// Both nil = equal. One nil, one non-nil = not equal (unless non-nil is empty).
func metadataEqual(a, b *storage.Metadata) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		a = &storage.Metadata{}
	}
	if b == nil {
		b = &storage.Metadata{}
	}
	return reflect.DeepEqual(a, b)
}

// metadataEqualForDedup compares two Metadata pointers for dedup purposes,
// ignoring the confluent:version property which is a transient CAS control property.
func metadataEqualForDedup(a, b *storage.Metadata) bool {
	return metadataEqual(stripConfluentVersion(a), stripConfluentVersion(b))
}

// ruleSetEqual compares two RuleSet pointers for equality.
func ruleSetEqual(a, b *storage.RuleSet) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		a = &storage.RuleSet{}
	}
	if b == nil {
		b = &storage.RuleSet{}
	}
	return reflect.DeepEqual(a, b)
}

// stripConfluentVersion returns a copy of the metadata without the confluent:version property.
// Returns nil if the result would be empty.
func stripConfluentVersion(meta *storage.Metadata) *storage.Metadata {
	if meta == nil || meta.Properties == nil {
		return meta
	}
	if _, ok := meta.Properties["confluent:version"]; !ok {
		return meta
	}
	// Make a copy to avoid mutating the original
	result := *meta
	newProps := make(map[string]string, len(meta.Properties))
	for k, v := range meta.Properties {
		if k != "confluent:version" {
			newProps[k] = v
		}
	}
	if len(newProps) == 0 {
		newProps = nil
	}
	result.Properties = newProps
	// If the result is effectively empty, return nil
	if result.Properties == nil && result.Tags == nil && result.Sensitive == nil {
		return nil
	}
	return &result
}

// autoPopulateConfluentVersion returns a copy of the record with confluent:version
// set in metadata to the actual assigned version number. This matches Confluent's
// behavior where confluent:version is auto-populated in the response after registration.
// The original record is not mutated.
func autoPopulateConfluentVersion(record *storage.SchemaRecord) *storage.SchemaRecord {
	if record == nil || record.Version <= 0 {
		return record
	}
	// Make a shallow copy of the record to avoid mutating stored data
	copy := *record
	// Deep copy the metadata to avoid mutating the stored version
	if copy.Metadata != nil {
		metaCopy := *copy.Metadata
		if metaCopy.Properties != nil {
			newProps := make(map[string]string, len(metaCopy.Properties)+1)
			for k, v := range metaCopy.Properties {
				newProps[k] = v
			}
			metaCopy.Properties = newProps
		} else {
			metaCopy.Properties = make(map[string]string, 1)
		}
		copy.Metadata = &metaCopy
	} else {
		copy.Metadata = &storage.Metadata{
			Properties: make(map[string]string, 1),
		}
	}
	copy.Metadata.Properties["confluent:version"] = strconv.Itoa(copy.Version)
	return &copy
}

// maybeSetMetadataRuleSet implements Confluent's 3-layer metadata/ruleSet merge
// during schema registration: merge(merge(config.default, specific), config.override).
// If the request doesn't specify metadata/ruleSet, it inherits from the previous schema version.
func (r *Registry) maybeSetMetadataRuleSet(ctx context.Context, registryCtx, subject string, opts *RegisterOpts, prevSchema *storage.SchemaRecord) {
	config, err := r.GetConfigFull(ctx, registryCtx, subject)
	if err != nil {
		return
	}

	// Determine specific metadata (from request, or inherit from previous version)
	specificMeta := opts.Metadata
	if specificMeta == nil && prevSchema != nil {
		specificMeta = prevSchema.Metadata
	}

	// 3-layer merge: default → specific → override
	merged := mergeMetadata(config.DefaultMetadata, specificMeta)
	merged = mergeMetadata(merged, config.OverrideMetadata)
	if merged != nil {
		opts.Metadata = merged
	}

	// Same for RuleSet
	specificRules := opts.RuleSet
	if specificRules == nil && prevSchema != nil {
		specificRules = prevSchema.RuleSet
	}

	mergedRules := mergeRuleSet(config.DefaultRuleSet, specificRules)
	mergedRules = mergeRuleSet(mergedRules, config.OverrideRuleSet)
	if mergedRules != nil {
		opts.RuleSet = mergedRules
	}
}

// mergeMetadata merges two Metadata objects. The override takes precedence
// for conflicting keys. Returns nil if both are nil.
func mergeMetadata(base, override *storage.Metadata) *storage.Metadata {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := &storage.Metadata{}

	// Merge properties
	if base.Properties != nil || override.Properties != nil {
		result.Properties = make(map[string]string)
		for k, v := range base.Properties {
			result.Properties[k] = v
		}
		for k, v := range override.Properties {
			result.Properties[k] = v
		}
	}

	// Merge tags
	if base.Tags != nil || override.Tags != nil {
		result.Tags = make(map[string][]string)
		for k, v := range base.Tags {
			result.Tags[k] = v
		}
		for k, v := range override.Tags {
			result.Tags[k] = v
		}
	}

	// Merge sensitive — union of both lists (deduplicated)
	if base.Sensitive != nil || override.Sensitive != nil {
		seen := make(map[string]bool)
		for _, s := range base.Sensitive {
			if !seen[s] {
				result.Sensitive = append(result.Sensitive, s)
				seen[s] = true
			}
		}
		for _, s := range override.Sensitive {
			if !seen[s] {
				result.Sensitive = append(result.Sensitive, s)
				seen[s] = true
			}
		}
	}

	return result
}

// mergeRuleSet merges two RuleSet objects. The override's rules are appended
// to the base's rules (override rules take precedence by name).
func mergeRuleSet(base, override *storage.RuleSet) *storage.RuleSet {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := &storage.RuleSet{
		MigrationRules: mergeRules(base.MigrationRules, override.MigrationRules),
		DomainRules:    mergeRules(base.DomainRules, override.DomainRules),
		EncodingRules:  mergeRules(base.EncodingRules, override.EncodingRules),
	}

	return result
}

// mergeRules merges two rule slices. Override rules replace base rules with the same name.
func mergeRules(base, override []storage.Rule) []storage.Rule {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	if len(base) == 0 {
		return override
	}
	if len(override) == 0 {
		return base
	}

	// Build map of override rules by name
	overrideMap := make(map[string]storage.Rule, len(override))
	for _, r := range override {
		overrideMap[r.Name] = r
	}

	// Start with base rules, replacing any that have overrides
	var result []storage.Rule
	seen := make(map[string]bool)
	for _, r := range base {
		if or, ok := overrideMap[r.Name]; ok {
			result = append(result, or)
			seen[r.Name] = true
		} else {
			result = append(result, r)
		}
	}
	// Append override rules not already in base
	for _, r := range override {
		if !seen[r.Name] {
			result = append(result, r)
		}
	}

	return result
}
