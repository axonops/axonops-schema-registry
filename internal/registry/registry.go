// Package registry provides the core schema registry service.
package registry

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
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
func (r *Registry) RegisterSchema(ctx context.Context, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, opts ...RegisterOpts) (*storage.SchemaRecord, error) {
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
	resolvedRefs, err := r.resolveReferences(ctx, refs)
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
		shouldNormalize = r.isNormalizeEnabled(ctx, subject)
	}
	if shouldNormalize {
		parsed = parsed.Normalize()
		schemaStr = parsed.CanonicalString()
	}

	// Check if schema already exists with same fingerprint
	existing, err := r.storage.GetSchemaByFingerprint(ctx, subject, parsed.Fingerprint(), false)
	if err == nil && existing != nil {
		// Schema already exists, return existing
		return existing, nil
	}

	// Get compatibility level for this subject
	compatLevel, err := r.GetConfig(ctx, subject)
	if err != nil {
		compatLevel = r.defaultConfig
	}

	// Check compatibility if not NONE
	mode := compatibility.Mode(compatLevel)
	if mode != compatibility.ModeNone {
		// Get existing schemas for compatibility check
		existingSchemas, err := r.storage.GetSchemasBySubject(ctx, subject, false)
		if err != nil && !errors.Is(err, storage.ErrSubjectNotFound) {
			return nil, fmt.Errorf("failed to get existing schemas: %w", err)
		}

		// Filter by compatibility group if configured
		existingSchemas = r.filterByCompatibilityGroup(ctx, subject, opt.Metadata, existingSchemas)

		if len(existingSchemas) > 0 {
			// Build existing schemas with resolved references
			existingWithRefs := make([]compatibility.SchemaWithRefs, len(existingSchemas))
			for i, s := range existingSchemas {
				existingResolvedRefs, resolveErr := r.resolveReferences(ctx, s.References)
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
	if r.isValidateFieldsEnabled(ctx, subject) {
		if msgs := r.validateReservedFields(ctx, subject, parsed, opt.Metadata); len(msgs) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrIncompatibleSchema, strings.Join(msgs, "; "))
		}
	}

	// Check confluent:version (compare-and-set) if present in metadata
	if err := r.checkConfluentVersion(ctx, subject, opt.Metadata); err != nil {
		return nil, err
	}

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
	if err := r.storage.CreateSchema(ctx, record); err != nil {
		if errors.Is(err, storage.ErrSchemaExists) {
			// Get the existing schema
			existing, _ := r.storage.GetSchemaByFingerprint(ctx, subject, parsed.Fingerprint(), false)
			if existing != nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("failed to store schema: %w", err)
	}

	return record, nil
}

// ErrVersionConflict is returned when confluent:version compare-and-set fails.
var ErrVersionConflict = errors.New("version conflict")

// checkConfluentVersion validates the confluent:version metadata property if present.
// When set to a positive integer, it enforces that the version matches the expected
// next version for the subject (optimistic concurrency control).
// Values of 0 or -1 mean auto-increment (no check).
func (r *Registry) checkConfluentVersion(ctx context.Context, subject string, metadata *storage.Metadata) error {
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

	latest, err := r.storage.GetLatestSchema(ctx, subject)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			// New subject — only version 1 is valid
			if cv != 1 {
				return fmt.Errorf("%w: confluent:version %d but subject is new (expected 1)", ErrVersionConflict, cv)
			}
			return nil
		}
		return err
	}
	if cv != latest.Version+1 {
		return fmt.Errorf("%w: confluent:version %d but latest version is %d (expected %d)", ErrVersionConflict, cv, latest.Version, latest.Version+1)
	}
	return nil
}

// RegisterSchemaWithID registers a schema with a specific ID (for IMPORT mode).
// It validates the schema, determines the next version, and stores it with the given ID.
// Confluent behavior: if the ID already exists with the same schema content, the schema
// is associated with the new subject (succeeds). If the ID exists with different content,
// returns error code 42205.
func (r *Registry) RegisterSchemaWithID(ctx context.Context, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, id int64) (*storage.SchemaRecord, error) {
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	resolvedRefs, err := r.resolveReferences(ctx, refs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve references: %w", err)
	}

	parsed, err := parser.Parse(schemaStr, resolvedRefs)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Check if schema already exists in this subject with same fingerprint (idempotent)
	existing, err := r.storage.GetSchemaByFingerprint(ctx, subject, parsed.Fingerprint(), false)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Determine next version for this subject.
	// Include soft-deleted versions to avoid version number conflicts
	// with rows that still physically exist in storage.
	nextVersion := 1
	existingSchemas, err := r.storage.GetSchemasBySubject(ctx, subject, true)
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

	if err := r.storage.ImportSchema(ctx, record); err != nil {
		if errors.Is(err, storage.ErrSchemaIDConflict) {
			return nil, fmt.Errorf("overwrite schema with id %d: %w", id, ErrImportIDConflict)
		}
		return nil, fmt.Errorf("failed to store schema: %w", err)
	}

	// Advance the ID sequence so future auto-assigned IDs don't collide.
	// Only advance forward, never rewind.
	nextID := id + 1
	currentMax, err := r.storage.GetMaxSchemaID(ctx)
	if err == nil && currentMax+1 > nextID {
		nextID = currentMax + 1
	}
	if err := r.storage.SetNextID(ctx, nextID); err != nil {
		return record, fmt.Errorf("schema stored but failed to advance ID sequence: %w", err)
	}

	return record, nil
}

// CheckCompatibility checks if a schema is compatible with a specific version or all versions.
func (r *Registry) CheckCompatibility(ctx context.Context, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, version string, normalize ...bool) (*compatibility.Result, error) {
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
	resolvedRefs, err := r.resolveReferences(ctx, refs)
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
		shouldNormalize = r.isNormalizeEnabled(ctx, subject)
	}
	if shouldNormalize {
		parsed = parsed.Normalize()
		schemaStr = parsed.CanonicalString()
	}

	// Get compatibility level
	compatLevel, err := r.GetConfig(ctx, subject)
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
		latest, err := r.storage.GetLatestSchema(ctx, subject)
		if err != nil {
			if errors.Is(err, storage.ErrSubjectNotFound) {
				// No existing schemas, always compatible
				return compatibility.NewCompatibleResult(), nil
			}
			return nil, err
		}
		latestRefs, resolveErr := r.resolveReferences(ctx, latest.References)
		if resolveErr != nil {
			return nil, fmt.Errorf("failed to resolve existing schema references: %w", resolveErr)
		}
		schemasToCheck = []compatibility.SchemaWithRefs{{Schema: latest.Schema, References: latestRefs}}
	} else if version == "" {
		// Empty version means check against all versions (transitive compatibility)
		existingSchemas, err := r.storage.GetSchemasBySubject(ctx, subject, false)
		if err != nil {
			if errors.Is(err, storage.ErrSubjectNotFound) {
				return compatibility.NewCompatibleResult(), nil
			}
			return nil, err
		}

		for _, s := range existingSchemas {
			existingRefs, resolveErr := r.resolveReferences(ctx, s.References)
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
		schema, err := r.storage.GetSchemaBySubjectVersion(ctx, subject, versionNum)
		if err != nil {
			if errors.Is(err, storage.ErrSubjectNotFound) {
				return nil, fmt.Errorf("%w: %s", storage.ErrSubjectNotFound, subject)
			}
			if errors.Is(err, storage.ErrVersionNotFound) {
				return nil, fmt.Errorf("%w: version %d for subject %s", storage.ErrVersionNotFound, versionNum, subject)
			}
			return nil, err
		}
		schemaRefs, resolveErr := r.resolveReferences(ctx, schema.References)
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

// GetSchemaByID retrieves a schema by its global ID.
func (r *Registry) GetSchemaByID(ctx context.Context, id int64) (*storage.SchemaRecord, error) {
	return r.storage.GetSchemaByID(ctx, id)
}

// GetMaxSchemaID returns the highest schema ID currently assigned.
func (r *Registry) GetMaxSchemaID(ctx context.Context) (int64, error) {
	return r.storage.GetMaxSchemaID(ctx)
}

// FormatSchema parses a schema record and returns it formatted according to the given format.
// Returns the original schema string if format is empty or parsing fails.
func (r *Registry) FormatSchema(ctx context.Context, record *storage.SchemaRecord, format string) string {
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
			refSchema, err := r.storage.GetSchemaBySubjectVersion(ctx, ref.Subject, ref.Version)
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
func (r *Registry) GetSchemaBySubjectVersion(ctx context.Context, subject string, version int) (*storage.SchemaRecord, error) {
	return r.storage.GetSchemaBySubjectVersion(ctx, subject, version)
}

// GetSchemasBySubject returns all schemas for a subject, optionally including deleted.
func (r *Registry) GetSchemasBySubject(ctx context.Context, subject string, includeDeleted bool) ([]*storage.SchemaRecord, error) {
	return r.storage.GetSchemasBySubject(ctx, subject, includeDeleted)
}

// ListSubjects returns all subject names.
func (r *Registry) ListSubjects(ctx context.Context, deleted bool) ([]string, error) {
	return r.storage.ListSubjects(ctx, deleted)
}

// GetVersions returns all versions for a subject.
func (r *Registry) GetVersions(ctx context.Context, subject string, deleted bool) ([]int, error) {
	schemas, err := r.storage.GetSchemasBySubject(ctx, subject, deleted)
	if err != nil {
		return nil, err
	}

	versions := make([]int, 0, len(schemas))
	for _, s := range schemas {
		versions = append(versions, s.Version)
	}

	return versions, nil
}

// LookupSchema finds a schema in a subject.
func (r *Registry) LookupSchema(ctx context.Context, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, deleted bool, normalize ...bool) (*storage.SchemaRecord, error) {
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
	resolvedRefs, err := r.resolveReferences(ctx, refs)
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
		shouldNormalize = r.isNormalizeEnabled(ctx, subject)
	}
	if shouldNormalize {
		parsed = parsed.Normalize()
	}

	// Look up by fingerprint, including deleted if requested
	return r.storage.GetSchemaByFingerprint(ctx, subject, parsed.Fingerprint(), deleted)
}

// DeleteSubject deletes a subject.
func (r *Registry) DeleteSubject(ctx context.Context, subject string, permanent bool) ([]int, error) {
	if !permanent {
		// For soft-delete, check if any version in this subject is referenced by other schemas.
		// If the subject doesn't exist or is already deleted, skip the check and let
		// DeleteSubject return the appropriate error.
		schemas, err := r.storage.GetSchemasBySubject(ctx, subject, false)
		if err == nil {
			for _, schema := range schemas {
				refs, err := r.storage.GetReferencedBy(ctx, subject, schema.Version)
				if err != nil {
					return nil, err
				}
				if len(refs) > 0 {
					return nil, fmt.Errorf("One or more references exist to the schema {subject=%s,version=%d}", subject, schema.Version)
				}
			}
		}
	}
	versions, err := r.storage.DeleteSubject(ctx, subject, permanent)
	if err != nil {
		return nil, err
	}
	// Only clean up subject-level config and mode on permanent delete.
	// Soft-delete preserves config/mode so re-registration inherits them.
	if permanent {
		_ = r.storage.DeleteConfig(ctx, subject)
		_ = r.storage.DeleteMode(ctx, subject)
	}
	return versions, nil
}

// DeleteVersion deletes a specific version.
func (r *Registry) DeleteVersion(ctx context.Context, subject string, version int, permanent bool) (int, error) {
	if permanent {
		// For permanent delete, the version must already be soft-deleted.
		// The storage layer validates this (returns ErrVersionNotSoftDeleted if not).
		// We skip GetSchemaBySubjectVersion because it filters out soft-deleted versions.
		if err := r.storage.DeleteSchema(ctx, subject, version, permanent); err != nil {
			return 0, err
		}
		return version, nil
	}

	// Soft-delete: verify the schema exists (non-deleted)
	schema, err := r.storage.GetSchemaBySubjectVersion(ctx, subject, version)
	if err != nil {
		return 0, err
	}

	// Use the resolved version (handles "latest" / -1 → actual version number)
	resolvedVersion := schema.Version

	// Check for references - only block soft-delete when referenced
	refs, err := r.storage.GetReferencedBy(ctx, subject, resolvedVersion)
	if err != nil {
		return 0, err
	}
	if len(refs) > 0 {
		return 0, fmt.Errorf("schema is referenced by other schemas")
	}

	if err := r.storage.DeleteSchema(ctx, subject, resolvedVersion, permanent); err != nil {
		return 0, err
	}

	return resolvedVersion, nil
}

// GetConfig gets the compatibility configuration for a subject.
func (r *Registry) GetConfig(ctx context.Context, subject string) (string, error) {
	if subject == "" {
		config, err := r.storage.GetGlobalConfig(ctx)
		if err != nil {
			return r.defaultConfig, nil
		}
		return config.CompatibilityLevel, nil
	}

	config, err := r.storage.GetConfig(ctx, subject)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// Fall back to global config
			return r.GetConfig(ctx, "")
		}
		return "", err
	}

	return config.CompatibilityLevel, nil
}

// GetSubjectConfig gets the compatibility configuration for a specific subject only,
// without falling back to the global default. Returns storage.ErrNotFound if not set.
func (r *Registry) GetSubjectConfig(ctx context.Context, subject string) (string, error) {
	config, err := r.storage.GetConfig(ctx, subject)
	if err != nil {
		return "", err
	}
	return config.CompatibilityLevel, nil
}

// GetSubjectConfigFull gets the full configuration record for a subject only,
// without falling back to global. Returns storage.ErrNotFound if not set.
func (r *Registry) GetSubjectConfigFull(ctx context.Context, subject string) (*storage.ConfigRecord, error) {
	return r.storage.GetConfig(ctx, subject)
}

// GetConfigFull gets the full configuration record with global fallback.
func (r *Registry) GetConfigFull(ctx context.Context, subject string) (*storage.ConfigRecord, error) {
	if subject == "" {
		config, err := r.storage.GetGlobalConfig(ctx)
		if err != nil {
			return &storage.ConfigRecord{CompatibilityLevel: r.defaultConfig}, nil
		}
		return config, nil
	}

	config, err := r.storage.GetConfig(ctx, subject)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return r.GetConfigFull(ctx, "")
		}
		return nil, err
	}
	return config, nil
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

// SetConfig sets the compatibility configuration for a subject.
func (r *Registry) SetConfig(ctx context.Context, subject string, level string, normalize *bool, opts ...SetConfigOpts) error {
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
		return r.storage.SetGlobalConfig(ctx, config)
	}

	return r.storage.SetConfig(ctx, subject, config)
}

// DeleteConfig deletes the compatibility configuration for a subject.
func (r *Registry) DeleteConfig(ctx context.Context, subject string) (string, error) {
	config, err := r.storage.GetConfig(ctx, subject)
	if err != nil {
		return "", err
	}

	if err := r.storage.DeleteConfig(ctx, subject); err != nil {
		return "", err
	}

	return config.CompatibilityLevel, nil
}

// GetMode gets the mode for a subject (with fallback to global).
func (r *Registry) GetMode(ctx context.Context, subject string) (string, error) {
	if subject == "" {
		mode, err := r.storage.GetGlobalMode(ctx)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				// No global mode configured: default to READWRITE
				return "READWRITE", nil
			}
			// Storage error: propagate rather than fail open
			return "", fmt.Errorf("failed to get global mode: %w", err)
		}
		return mode.Mode, nil
	}

	mode, err := r.storage.GetMode(ctx, subject)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return r.GetMode(ctx, "")
		}
		return "", err
	}

	return mode.Mode, nil
}

// GetSubjectMode gets the mode for a specific subject only,
// without falling back to the global default. Returns storage.ErrNotFound if not set.
func (r *Registry) GetSubjectMode(ctx context.Context, subject string) (string, error) {
	mode, err := r.storage.GetMode(ctx, subject)
	if err != nil {
		return "", err
	}
	return mode.Mode, nil
}

// SetMode sets the mode for a subject.
// If switching to IMPORT mode and force is false, it checks that no schemas exist.
func (r *Registry) SetMode(ctx context.Context, subject string, mode string, force bool) error {
	mode = strings.ToUpper(mode)
	if !isValidMode(mode) {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	// Confluent behavior: switching to IMPORT requires force=true if schemas exist
	if mode == "IMPORT" && !force {
		currentMode, _ := r.GetMode(ctx, subject)
		if currentMode != "IMPORT" {
			hasSchemas, err := r.hasSubjects(ctx, subject)
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
		return r.storage.SetGlobalMode(ctx, modeRecord)
	}

	return r.storage.SetMode(ctx, subject, modeRecord)
}

// hasSubjects checks if any non-deleted schemas exist for the given subject (or globally if subject is empty).
// isNormalizeEnabled checks if normalization is enabled for a subject via config.
func (r *Registry) isNormalizeEnabled(ctx context.Context, subject string) bool {
	// Check subject config first
	if subject != "" {
		config, err := r.storage.GetConfig(ctx, subject)
		if err == nil && config != nil && config.Normalize != nil {
			return *config.Normalize
		}
	}
	// Fall back to global config
	config, err := r.storage.GetGlobalConfig(ctx)
	if err == nil && config != nil && config.Normalize != nil {
		return *config.Normalize
	}
	return false
}

func (r *Registry) hasSubjects(ctx context.Context, subject string) (bool, error) {
	if subject == "" {
		// Check if any subjects exist globally
		subjects, err := r.storage.ListSubjects(ctx, false)
		if err != nil {
			return false, err
		}
		return len(subjects) > 0, nil
	}
	// Check if the specific subject has schemas
	exists, err := r.storage.SubjectExists(ctx, subject)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetReferencedBy gets subjects/versions that reference a schema.
func (r *Registry) GetReferencedBy(ctx context.Context, subject string, version int) ([]storage.SubjectVersion, error) {
	return r.storage.GetReferencedBy(ctx, subject, version)
}

// GetSchemaTypes returns the supported schema types.
func (r *Registry) GetSchemaTypes() []string {
	return r.schemaParser.Types()
}

// GetRawSchemaByID retrieves just the schema string by ID.
func (r *Registry) GetRawSchemaByID(ctx context.Context, id int64) (string, error) {
	schema, err := r.storage.GetSchemaByID(ctx, id)
	if err != nil {
		return "", err
	}
	return schema.Schema, nil
}

// GetSubjectsBySchemaID returns all subjects where the given schema ID is registered.
func (r *Registry) GetSubjectsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]string, error) {
	return r.storage.GetSubjectsBySchemaID(ctx, id, includeDeleted)
}

// GetVersionsBySchemaID returns all subject-version pairs for a schema ID.
func (r *Registry) GetVersionsBySchemaID(ctx context.Context, id int64, includeDeleted bool) ([]storage.SubjectVersion, error) {
	return r.storage.GetVersionsBySchemaID(ctx, id, includeDeleted)
}

// ListSchemas returns schemas matching the given filters.
func (r *Registry) ListSchemas(ctx context.Context, params *storage.ListSchemasParams) ([]*storage.SchemaRecord, error) {
	return r.storage.ListSchemas(ctx, params)
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

// ImportSchemas imports schemas with preserved IDs (for migration).
// It validates each schema, imports it with the specified ID, and adjusts
// the ID sequence after import to prevent conflicts.
func (r *Registry) ImportSchemas(ctx context.Context, schemas []ImportSchemaRequest) (*ImportResult, error) {
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
		resolvedRefs, resolveErr := r.resolveReferences(ctx, req.References)
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
		if err := r.storage.ImportSchema(ctx, record); err != nil {
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
		currentMax, err := r.storage.GetMaxSchemaID(ctx)
		if err == nil && currentMax+1 > nextID {
			nextID = currentMax + 1
		}

		if err := r.storage.SetNextID(ctx, nextID); err != nil {
			return result, fmt.Errorf("imported %d schemas but failed to adjust ID sequence: %w", result.Imported, err)
		}
	}

	return result, nil
}

// GetRawSchemaBySubjectVersion retrieves just the schema string by subject and version.
func (r *Registry) GetRawSchemaBySubjectVersion(ctx context.Context, subject string, version int) (string, error) {
	schema, err := r.storage.GetSchemaBySubjectVersion(ctx, subject, version)
	if err != nil {
		return "", err
	}
	return schema.Schema, nil
}

// DeleteGlobalConfig resets the global compatibility configuration to default.
func (r *Registry) DeleteGlobalConfig(ctx context.Context) (string, error) {
	config, err := r.storage.GetGlobalConfig(ctx)
	if err != nil {
		// If no config, use default
		config = &storage.ConfigRecord{CompatibilityLevel: r.defaultConfig}
	}
	prevLevel := config.CompatibilityLevel

	if err := r.storage.DeleteGlobalConfig(ctx); err != nil {
		return "", err
	}

	return prevLevel, nil
}

// DeleteMode deletes the mode configuration for a subject.
func (r *Registry) DeleteMode(ctx context.Context, subject string) (string, error) {
	mode, err := r.storage.GetMode(ctx, subject)
	if err != nil {
		return "", err
	}
	prevMode := mode.Mode

	if err := r.storage.DeleteMode(ctx, subject); err != nil {
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
func (r *Registry) filterByCompatibilityGroup(ctx context.Context, subject string, newMeta *storage.Metadata, schemas []*storage.SchemaRecord) []*storage.SchemaRecord {
	if len(schemas) == 0 {
		return schemas
	}

	// Get the compatibility group property name from config
	config, err := r.GetSubjectConfigFull(ctx, subject)
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

// isValidateFieldsEnabled checks if reserved field validation is enabled for a subject.
func (r *Registry) isValidateFieldsEnabled(ctx context.Context, subject string) bool {
	// Check subject config first
	if subject != "" {
		config, err := r.storage.GetConfig(ctx, subject)
		if err == nil && config != nil && config.ValidateFields != nil {
			return *config.ValidateFields
		}
	}
	// Fall back to global config
	config, err := r.storage.GetGlobalConfig(ctx)
	if err == nil && config != nil && config.ValidateFields != nil {
		return *config.ValidateFields
	}
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
func (r *Registry) validateReservedFields(ctx context.Context, subject string, parsed schema.ParsedSchema, metadata *storage.Metadata) []string {
	var msgs []string

	reservedFields := getReservedFields(metadata)

	// Rule 2: Check that reserved fields from previous version are not removed
	latest, err := r.storage.GetLatestSchema(ctx, subject)
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
func (r *Registry) resolveReferences(ctx context.Context, refs []storage.Reference) ([]storage.Reference, error) {
	if len(refs) == 0 {
		return refs, nil
	}
	resolved := make([]storage.Reference, len(refs))
	for i, ref := range refs {
		record, err := r.storage.GetSchemaBySubjectVersion(ctx, ref.Subject, ref.Version)
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
