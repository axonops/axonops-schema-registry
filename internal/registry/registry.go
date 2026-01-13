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

// RegisterSchema registers a new schema for a subject.
func (r *Registry) RegisterSchema(ctx context.Context, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference) (*storage.SchemaRecord, error) {
	// Default to Avro if not specified
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	// Get the parser for this schema type
	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	// Parse the schema
	parsed, err := parser.Parse(schemaStr, refs)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
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

		if len(existingSchemas) > 0 {
			// Extract schema strings for comparison
			schemaStrings := make([]string, len(existingSchemas))
			for i, s := range existingSchemas {
				schemaStrings[i] = s.Schema
			}

			// Check compatibility
			result := r.compatChecker.Check(mode, schemaType, schemaStr, schemaStrings)
			if !result.IsCompatible {
				return nil, fmt.Errorf("%w: %s", ErrIncompatibleSchema, strings.Join(result.Messages, "; "))
			}
		}
	}

	// Create new schema record
	record := &storage.SchemaRecord{
		Subject:     subject,
		SchemaType:  schemaType,
		Schema:      schemaStr,
		References:  refs,
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

// CheckCompatibility checks if a schema is compatible with a specific version or all versions.
func (r *Registry) CheckCompatibility(ctx context.Context, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, version string) (*compatibility.Result, error) {
	// Default to Avro if not specified
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	// Parse the new schema to validate it
	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	_, err := parser.Parse(schemaStr, refs)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
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
	var schemasToCheck []string

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
		schemasToCheck = []string{latest.Schema}
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
			schemasToCheck = append(schemasToCheck, s.Schema)
		}
	} else {
		// Check against specific version only
		versionNum, err := strconv.Atoi(version)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", storage.ErrInvalidVersion, version)
		}
		schema, err := r.storage.GetSchemaBySubjectVersion(ctx, subject, versionNum)
		if err != nil {
			if errors.Is(err, storage.ErrSubjectNotFound) || errors.Is(err, storage.ErrVersionNotFound) {
				return nil, fmt.Errorf("%w: version %d for subject %s", storage.ErrVersionNotFound, versionNum, subject)
			}
			return nil, err
		}
		schemasToCheck = []string{schema.Schema}
	}

	if len(schemasToCheck) == 0 {
		return compatibility.NewCompatibleResult(), nil
	}

	return r.compatChecker.Check(mode, schemaType, schemaStr, schemasToCheck), nil
}

// GetSchemaByID retrieves a schema by its global ID.
func (r *Registry) GetSchemaByID(ctx context.Context, id int64) (*storage.SchemaRecord, error) {
	return r.storage.GetSchemaByID(ctx, id)
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version.
func (r *Registry) GetSchemaBySubjectVersion(ctx context.Context, subject string, version int) (*storage.SchemaRecord, error) {
	return r.storage.GetSchemaBySubjectVersion(ctx, subject, version)
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
func (r *Registry) LookupSchema(ctx context.Context, subject string, schemaStr string, schemaType storage.SchemaType, refs []storage.Reference, deleted bool) (*storage.SchemaRecord, error) {
	// Default to Avro if not specified
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	// Get the parser for this schema type
	parser, ok := r.schemaParser.Get(schemaType)
	if !ok {
		return nil, fmt.Errorf("unsupported schema type: %s", schemaType)
	}

	// Parse the schema to get fingerprint
	parsed, err := parser.Parse(schemaStr, refs)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Look up by fingerprint, including deleted if requested
	return r.storage.GetSchemaByFingerprint(ctx, subject, parsed.Fingerprint(), deleted)
}

// DeleteSubject deletes a subject.
func (r *Registry) DeleteSubject(ctx context.Context, subject string, permanent bool) ([]int, error) {
	return r.storage.DeleteSubject(ctx, subject, permanent)
}

// DeleteVersion deletes a specific version.
func (r *Registry) DeleteVersion(ctx context.Context, subject string, version int, permanent bool) (int, error) {
	// First verify the schema exists
	schema, err := r.storage.GetSchemaBySubjectVersion(ctx, subject, version)
	if err != nil {
		return 0, err
	}

	// Check for references
	refs, err := r.storage.GetReferencedBy(ctx, subject, version)
	if err != nil {
		return 0, err
	}
	if len(refs) > 0 {
		return 0, fmt.Errorf("schema is referenced by other schemas")
	}

	if err := r.storage.DeleteSchema(ctx, subject, version, permanent); err != nil {
		return 0, err
	}

	return schema.Version, nil
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

// SetConfig sets the compatibility configuration for a subject.
func (r *Registry) SetConfig(ctx context.Context, subject string, level string) error {
	level = strings.ToUpper(level)
	if !isValidCompatibility(level) {
		return fmt.Errorf("invalid compatibility level: %s", level)
	}

	config := &storage.ConfigRecord{
		CompatibilityLevel: level,
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

// GetMode gets the mode for a subject.
func (r *Registry) GetMode(ctx context.Context, subject string) (string, error) {
	if subject == "" {
		mode, err := r.storage.GetGlobalMode(ctx)
		if err != nil {
			return "READWRITE", nil
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

// SetMode sets the mode for a subject.
func (r *Registry) SetMode(ctx context.Context, subject string, mode string) error {
	mode = strings.ToUpper(mode)
	if !isValidMode(mode) {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	modeRecord := &storage.ModeRecord{
		Mode: mode,
	}

	if subject == "" {
		return r.storage.SetGlobalMode(ctx, modeRecord)
	}

	return r.storage.SetMode(ctx, subject, modeRecord)
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

		parsed, err := parser.Parse(req.Schema, req.References)
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

	// Adjust the ID sequence to prevent conflicts
	if maxID > 0 {
		// Set the next ID to be one more than the maximum imported ID
		if err := r.storage.SetNextID(ctx, maxID+1); err != nil {
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
		"READWRITE": true,
		"READONLY":  true,
		"IMPORT":    true,
	}
	return valid[mode]
}
