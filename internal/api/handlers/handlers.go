// Package handlers provides HTTP request handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// errInvalidVersion is returned when a version string is not valid.
var errInvalidVersion = errors.New("invalid version")

// schemaTypeForResponse returns the schema type string for API responses.
// Always returns a non-empty type string; defaults to "AVRO" if unset.
func schemaTypeForResponse(st storage.SchemaType) string {
	if st == "" {
		return string(storage.SchemaTypeAvro)
	}
	return string(st)
}

// Handler provides HTTP handlers for the schema registry.
type Handler struct {
	registry  *registry.Registry
	clusterID string
	version   string
	commit    string
	buildTime string
}

// Config holds handler configuration.
type Config struct {
	ClusterID string
	Version   string
	Commit    string
	BuildTime string
}

// New creates a new Handler.
func New(reg *registry.Registry) *Handler {
	return &Handler{
		registry:  reg,
		clusterID: "default-cluster",
		version:   "1.0.0",
	}
}

// NewWithConfig creates a new Handler with configuration.
func NewWithConfig(reg *registry.Registry, cfg Config) *Handler {
	return &Handler{
		registry:  reg,
		clusterID: cfg.ClusterID,
		version:   cfg.Version,
		commit:    cfg.Commit,
		buildTime: cfg.BuildTime,
	}
}

// checkModeForWrite checks if the current mode allows write operations for the given subject.
// Returns:
//   - mode string: non-empty if writes are blocked by READONLY or READONLY_OVERRIDE
//   - error: non-nil if the mode check itself failed (e.g. storage unreachable)
//
// Both READONLY and READONLY_OVERRIDE block data and config writes.
func (h *Handler) checkModeForWrite(r *http.Request, registryCtx string, subject string) (string, error) {
	mode, err := h.registry.GetMode(r.Context(), registryCtx, subject)
	if err != nil {
		return "", fmt.Errorf("failed to check mode: %w", err)
	}
	if mode == "READONLY" || mode == "READONLY_OVERRIDE" {
		return mode, nil
	}
	return "", nil
}

// resolveAlias resolves a subject alias. If the subject has an alias configured,
// the alias target is returned. Otherwise the original subject is returned.
// Alias resolution is single-level (no recursive chaining).
func (h *Handler) resolveAlias(ctx context.Context, registryCtx string, subject string) string {
	if subject == "" {
		return subject
	}
	config, err := h.registry.GetSubjectConfigFull(ctx, registryCtx, subject)
	if err == nil && config.Alias != "" {
		return config.Alias
	}
	return subject
}

// HealthCheck handles GET /
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{})
}

// LivenessCheck handles GET /health/live
// Always returns 200 — confirms the process is alive and not deadlocked.
func (h *Handler) LivenessCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "UP"})
}

// ReadinessCheck handles GET /health/ready
// Returns 200 when storage is healthy, 503 when not.
func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	if h.registry.IsHealthy(r.Context()) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "UP"})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"status": "DOWN",
		"reason": "storage backend unavailable",
	})
}

// StartupCheck handles GET /health/startup
// Returns 200 when storage is connected and ready, 503 during initialization.
func (h *Handler) StartupCheck(w http.ResponseWriter, r *http.Request) {
	if h.registry.IsHealthy(r.Context()) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "UP"})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{
		"status": "DOWN",
		"reason": "storage backend unavailable",
	})
}

// GetSchemaTypes handles GET /schemas/types
func (h *Handler) GetSchemaTypes(w http.ResponseWriter, r *http.Request) {
	types := h.registry.GetSchemaTypes()
	writeJSON(w, http.StatusOK, types)
}

// GetSchemaByID handles GET /schemas/ids/{id}
func (h *Handler) GetSchemaByID(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid schema ID")
		return
	}

	schema, err := h.registry.GetSchemaByID(r.Context(), registryCtx, id)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	schemaStr := schema.Schema
	if format := r.URL.Query().Get("format"); format != "" {
		schemaStr = h.registry.FormatSchema(r.Context(), registryCtx, schema, format)
	}

	resp := types.SchemaByIDResponse{
		Schema:     schemaStr,
		SchemaType: schemaTypeForResponse(schema.SchemaType),
		References: schema.References,
		Metadata:   schema.Metadata,
		RuleSet:    schema.RuleSet,
	}

	if r.URL.Query().Get("fetchMaxId") == "true" {
		maxID, err := h.registry.GetMaxSchemaID(r.Context(), registryCtx)
		if err == nil {
			resp.MaxId = &maxID
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListSubjects handles GET /subjects
func (h *Handler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)
	deleted := r.URL.Query().Get("deleted") == "true"
	deletedOnly := r.URL.Query().Get("deletedOnly") == "true"
	subjectPrefix := r.URL.Query().Get("subjectPrefix")

	// deletedOnly implies including deleted subjects
	includeDeleted := deleted || deletedOnly

	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, includeDeleted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// For deletedOnly, filter to only deleted subjects by diffing with active set
	if deletedOnly {
		activeSubjects, _ := h.registry.ListSubjects(r.Context(), registryCtx, false)
		activeSet := make(map[string]bool, len(activeSubjects))
		for _, s := range activeSubjects {
			activeSet[s] = true
		}
		var deletedSubjects []string
		for _, s := range subjects {
			if !activeSet[s] {
				deletedSubjects = append(deletedSubjects, s)
			}
		}
		if deletedSubjects == nil {
			deletedSubjects = []string{}
		}
		subjects = deletedSubjects
	}

	// Filter by subject prefix if specified
	if subjectPrefix != "" {
		filtered := make([]string, 0)
		for _, s := range subjects {
			if strings.HasPrefix(s, subjectPrefix) {
				filtered = append(filtered, s)
			}
		}
		subjects = filtered
	}

	// Apply pagination (offset/limit)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	limit := -1 // default: unlimited
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	if offset > 0 {
		if offset >= len(subjects) {
			subjects = []string{}
		} else {
			subjects = subjects[offset:]
		}
	}
	if limit > 0 && limit < len(subjects) {
		subjects = subjects[:limit]
	}

	writeJSON(w, http.StatusOK, subjects)
}

// GetVersions handles GET /subjects/{subject}/versions
func (h *Handler) GetVersions(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)
	deleted := r.URL.Query().Get("deleted") == "true"
	deletedOnly := r.URL.Query().Get("deletedOnly") == "true"

	// deletedOnly takes precedence: if set, we include deleted and filter to only deleted
	includeDeleted := deleted || deletedOnly
	versions, err := h.registry.GetVersions(r.Context(), registryCtx, subject, includeDeleted)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	if deletedOnly {
		// Filter to only deleted versions by getting all versions and non-deleted, then diffing
		activeVersions, _ := h.registry.GetVersions(r.Context(), registryCtx, subject, false)
		activeSet := make(map[int]bool, len(activeVersions))
		for _, v := range activeVersions {
			activeSet[v] = true
		}
		var deletedVersions []int
		for _, v := range versions {
			if !activeSet[v] {
				deletedVersions = append(deletedVersions, v)
			}
		}
		if deletedVersions == nil {
			deletedVersions = []int{}
		}
		writeJSON(w, http.StatusOK, deletedVersions)
		return
	}

	// Apply pagination
	start, end := parsePagination(r, len(versions))
	writeJSON(w, http.StatusOK, versions[start:end])
}

// GetVersion handles GET /subjects/{subject}/versions/{version}
func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)
	versionStr := chi.URLParam(r, "version")

	version, err := parseVersion(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\"", versionStr))
		return
	}

	includeDeleted := r.URL.Query().Get("deleted") == "true"

	schema, err := h.registry.GetSchemaBySubjectVersion(r.Context(), registryCtx, subject, version)
	if err != nil {
		// If deleted=true and version not found, try to find the deleted version
		if includeDeleted && (errors.Is(err, storage.ErrVersionNotFound) || errors.Is(err, storage.ErrSubjectNotFound)) {
			schema, err = h.findDeletedVersion(r.Context(), registryCtx, subject, version)
		}
		if err != nil {
			if errors.Is(err, storage.ErrSubjectNotFound) {
				writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
				return
			}
			if errors.Is(err, storage.ErrVersionNotFound) {
				// Confluent returns 40401 (subject not found) when all versions
				// of a subject are soft-deleted, rather than 40402 (version not found).
				if h.isSubjectFullyDeleted(r.Context(), registryCtx, subject) {
					writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
					return
				}
				writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, "Version not found")
				return
			}
			writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
			return
		}
	}

	schemaStr := schema.Schema
	if format := r.URL.Query().Get("format"); format != "" {
		schemaStr = h.registry.FormatSchema(r.Context(), registryCtx, schema, format)
	}

	resp := types.SubjectVersionResponse{
		Subject:    schema.Subject,
		ID:         schema.ID,
		Version:    schema.Version,
		SchemaType: schemaTypeForResponse(schema.SchemaType),
		Schema:     schemaStr,
		Metadata:   schema.Metadata,
		RuleSet:    schema.RuleSet,
	}
	if len(schema.References) > 0 {
		resp.References = schema.References
	}

	writeJSON(w, http.StatusOK, resp)
}

// findDeletedVersion looks up a soft-deleted version by iterating all versions including deleted.
// When version is -1 (the "latest" sentinel), it returns the highest-versioned schema
// among all versions (including soft-deleted), matching Confluent's behavior.
func (h *Handler) findDeletedVersion(ctx context.Context, registryCtx string, subject string, version int) (*storage.SchemaRecord, error) {
	schemas, err := h.registry.GetSchemasBySubject(ctx, registryCtx, subject, true) // include deleted
	if err != nil {
		return nil, err
	}
	if len(schemas) == 0 {
		return nil, storage.ErrSubjectNotFound
	}

	// "latest" sentinel: return the highest version among all (including deleted)
	if version == -1 {
		var latest *storage.SchemaRecord
		for _, s := range schemas {
			if latest == nil || s.Version > latest.Version {
				latest = s
			}
		}
		if latest != nil {
			return latest, nil
		}
		return nil, storage.ErrVersionNotFound
	}

	for _, s := range schemas {
		if s.Version == version {
			return s, nil
		}
	}
	return nil, storage.ErrVersionNotFound
}

// isSubjectFullyDeleted returns true if the subject exists but all its versions
// are soft-deleted. Used to map ErrVersionNotFound → 40401 (Confluent behavior).
func (h *Handler) isSubjectFullyDeleted(ctx context.Context, registryCtx string, subject string) bool {
	// Check if subject has any versions including deleted ones
	allVersions, err := h.registry.GetVersions(ctx, registryCtx, subject, true)
	if err != nil || len(allVersions) == 0 {
		return false
	}
	// Check if subject has any active (non-deleted) versions
	activeVersions, err := h.registry.GetVersions(ctx, registryCtx, subject, false)
	if err != nil {
		// GetVersions returns ErrSubjectNotFound when all versions are deleted
		return errors.Is(err, storage.ErrSubjectNotFound)
	}
	return len(activeVersions) == 0
}

// RegisterSchema handles POST /subjects/{subject}/versions
func (h *Handler) RegisterSchema(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)

	// Check mode enforcement
	if mode, modeErr := h.checkModeForWrite(r, registryCtx, subject); modeErr != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeStorageError, modeErr.Error())
		return
	} else if mode != "" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted,
			fmt.Sprintf("Subject '%s' is in %s mode", subject, mode))
		return
	}

	var req types.RegisterSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if req.Schema == "" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, "Empty schema")
		return
	}

	schemaType := storage.SchemaType(strings.ToUpper(req.SchemaType))
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	normalizeSchema := r.URL.Query().Get("normalize") == "true"

	var schema *storage.SchemaRecord
	var err error

	// Explicit ID requires IMPORT mode (Confluent behavior)
	if req.ID > 0 {
		mode, modeErr := h.registry.GetMode(r.Context(), registryCtx, subject)
		if modeErr != nil {
			writeError(w, http.StatusInternalServerError, types.ErrorCodeStorageError, "Failed to check mode")
			return
		}
		if mode != "IMPORT" {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted,
				fmt.Sprintf("Subject '%s' is not in import mode. Registering schemas with explicit IDs requires IMPORT mode.", subject))
			return
		}
		schema, err = h.registry.RegisterSchemaWithID(r.Context(), registryCtx, subject, req.Schema, schemaType, req.References, req.ID)
	} else {
		schema, err = h.registry.RegisterSchema(r.Context(), registryCtx, subject, req.Schema, schemaType, req.References, registry.RegisterOpts{
			Normalize: normalizeSchema,
			Metadata:  req.Metadata,
			RuleSet:   req.RuleSet,
		})
	}
	if err != nil {
		if strings.Contains(err.Error(), "invalid schema") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
			return
		}
		if strings.Contains(err.Error(), "unsupported schema type") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
			return
		}
		if strings.Contains(err.Error(), "failed to resolve references") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
			return
		}
		if errors.Is(err, registry.ErrIncompatibleSchema) {
			writeError(w, http.StatusConflict, types.ErrorCodeIncompatibleSchema, err.Error())
			return
		}
		if errors.Is(err, registry.ErrVersionConflict) {
			writeError(w, http.StatusConflict, types.ErrorCodeIncompatibleSchema, err.Error())
			return
		}
		if errors.Is(err, registry.ErrImportIDConflict) {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted,
				fmt.Sprintf("Overwrite new schema with id %d is not permitted.", req.ID))
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.RegisterSchemaResponse{
		ID: schema.ID,
	})
}

// LookupSchema handles POST /subjects/{subject}
func (h *Handler) LookupSchema(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)
	deleted := r.URL.Query().Get("deleted") == "true"

	var req types.LookupSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if req.Schema == "" {
		writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
		return
	}

	schemaType := storage.SchemaType(strings.ToUpper(req.SchemaType))
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	normalizeSchema := r.URL.Query().Get("normalize") == "true"
	schema, err := h.registry.LookupSchema(r.Context(), registryCtx, subject, req.Schema, schemaType, req.References, deleted, normalizeSchema)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, fmt.Sprintf("Subject '%s' not found.", subject))
			return
		}
		if errors.Is(err, storage.ErrSchemaNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
			return
		}
		if strings.Contains(err.Error(), "invalid schema") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	resp := types.LookupSchemaResponse{
		Subject:    schema.Subject,
		ID:         schema.ID,
		Version:    schema.Version,
		SchemaType: schemaTypeForResponse(schema.SchemaType),
		Schema:     schema.Schema,
		Metadata:   schema.Metadata,
		RuleSet:    schema.RuleSet,
	}
	if len(schema.References) > 0 {
		resp.References = schema.References
	}

	writeJSON(w, http.StatusOK, resp)
}

// DeleteSubject handles DELETE /subjects/{subject}
func (h *Handler) DeleteSubject(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)
	permanent := r.URL.Query().Get("permanent") == "true"

	// Check mode enforcement
	if mode, modeErr := h.checkModeForWrite(r, registryCtx, subject); modeErr != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeStorageError, modeErr.Error())
		return
	} else if mode != "" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted,
			fmt.Sprintf("Subject '%s' is in %s mode", subject, mode))
		return
	}

	versions, err := h.registry.DeleteSubject(r.Context(), registryCtx, subject, permanent)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		if errors.Is(err, storage.ErrSubjectDeleted) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectSoftDeleted,
				fmt.Sprintf("Subject '%s' was soft deleted. Set permanent=true to delete permanently", subject))
			return
		}
		if errors.Is(err, storage.ErrSubjectNotSoftDeleted) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotSoftDeleted,
				fmt.Sprintf("Subject '%s' was not deleted first before being permanently deleted", subject))
			return
		}
		if strings.Contains(err.Error(), "reference") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeReferenceExists, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, versions)
}

// DeleteVersion handles DELETE /subjects/{subject}/versions/{version}
func (h *Handler) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)
	versionStr := chi.URLParam(r, "version")
	permanent := r.URL.Query().Get("permanent") == "true"

	// Check mode enforcement
	if mode, modeErr := h.checkModeForWrite(r, registryCtx, subject); modeErr != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeStorageError, modeErr.Error())
		return
	} else if mode != "" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted,
			fmt.Sprintf("Subject '%s' is in %s mode", subject, mode))
		return
	}

	// Permanent delete of "latest" or "-1" is not allowed — must use explicit version number
	if permanent && (versionStr == "latest" || versionStr == "-1") {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id for permanent delete. Use an explicit version number.", versionStr))
		return
	}

	version, err := parseVersion(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\"", versionStr))
		return
	}

	deletedVersion, err := h.registry.DeleteVersion(r.Context(), registryCtx, subject, version, permanent)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		if errors.Is(err, storage.ErrVersionNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, "Version not found")
			return
		}
		if errors.Is(err, storage.ErrVersionNotSoftDeleted) {
			writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotSoftDeleted,
				fmt.Sprintf("Subject '%s' Version %s was not deleted first before being permanently deleted", subject, versionStr))
			return
		}
		if strings.Contains(err.Error(), "referenced") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeReferenceExists, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, deletedVersion)
}

// GetConfig handles GET /config and GET /config/{subject}
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	defaultToGlobal := r.URL.Query().Get("defaultToGlobal") == "true"

	if subject != "" && !defaultToGlobal {
		// Subject-specific config only, no fallback to global
		config, err := h.registry.GetSubjectConfigFull(r.Context(), registryCtx, subject)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, types.ErrorCodeSubjectCompatNotFound,
					fmt.Sprintf("Subject '%s' does not have subject-level compatibility configured", subject))
				return
			}
			writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, types.ConfigResponse{
			CompatibilityLevel: config.CompatibilityLevel,
			Normalize:          config.Normalize,
			ValidateFields:     config.ValidateFields,
			Alias:              config.Alias,
			CompatibilityGroup: config.CompatibilityGroup,
			DefaultMetadata:    config.DefaultMetadata,
			OverrideMetadata:   config.OverrideMetadata,
			DefaultRuleSet:     config.DefaultRuleSet,
			OverrideRuleSet:    config.OverrideRuleSet,
		})
		return
	}

	config, err := h.registry.GetConfigFull(r.Context(), registryCtx, subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ConfigResponse{
		CompatibilityLevel: config.CompatibilityLevel,
		Normalize:          config.Normalize,
		ValidateFields:     config.ValidateFields,
		Alias:              config.Alias,
		CompatibilityGroup: config.CompatibilityGroup,
		DefaultMetadata:    config.DefaultMetadata,
		OverrideMetadata:   config.OverrideMetadata,
		DefaultRuleSet:     config.DefaultRuleSet,
		OverrideRuleSet:    config.OverrideRuleSet,
	})
}

// SetConfig handles PUT /config and PUT /config/{subject}
func (h *Handler) SetConfig(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)

	var req types.ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidCompatibilityLevel, "Invalid request body")
		return
	}

	// Empty body: return current config (matches Confluent behavior)
	if req.Compatibility == "" {
		level, err := h.registry.GetConfig(r.Context(), registryCtx, subject)
		if err != nil {
			writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, types.ConfigRequest{Compatibility: level})
		return
	}

	configOpts := registry.SetConfigOpts{
		Alias:              req.Alias,
		CompatibilityGroup: req.CompatibilityGroup,
		ValidateFields:     req.ValidateFields,
		DefaultMetadata:    req.DefaultMetadata,
		OverrideMetadata:   req.OverrideMetadata,
		DefaultRuleSet:     req.DefaultRuleSet,
		OverrideRuleSet:    req.OverrideRuleSet,
	}
	if err := h.registry.SetConfig(r.Context(), registryCtx, subject, req.Compatibility, req.Normalize, configOpts); err != nil {
		if strings.Contains(err.Error(), "invalid compatibility") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidCompatibilityLevel, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	resp := types.ConfigRequest{
		Compatibility:      strings.ToUpper(req.Compatibility),
		Normalize:          req.Normalize,
		ValidateFields:     req.ValidateFields,
		Alias:              req.Alias,
		CompatibilityGroup: req.CompatibilityGroup,
		DefaultMetadata:    req.DefaultMetadata,
		OverrideMetadata:   req.OverrideMetadata,
		DefaultRuleSet:     req.DefaultRuleSet,
		OverrideRuleSet:    req.OverrideRuleSet,
	}
	writeJSON(w, http.StatusOK, resp)
}

// DeleteConfig handles DELETE /config/{subject}
func (h *Handler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)

	level, err := h.registry.DeleteConfig(r.Context(), registryCtx, subject)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Config not found for subject")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ConfigResponse{
		CompatibilityLevel: level,
	})
}

// CheckCompatibility handles POST /compatibility/subjects/{subject}/versions/{version}
func (h *Handler) CheckCompatibility(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)
	versionStr := chi.URLParam(r, "version")

	var req types.CompatibilityCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if req.Schema == "" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, "Empty schema")
		return
	}

	schemaType := storage.SchemaType(strings.ToUpper(req.SchemaType))
	if schemaType == "" {
		schemaType = storage.SchemaTypeAvro
	}

	normalizeSchema := r.URL.Query().Get("normalize") == "true"
	result, err := h.registry.CheckCompatibility(r.Context(), registryCtx, subject, req.Schema, schemaType, req.References, versionStr, normalizeSchema)
	if err != nil {
		if strings.Contains(err.Error(), "invalid schema") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
			return
		}
		if errors.Is(err, storage.ErrSubjectNotFound) {
			// When checking against a specific version, Confluent returns 40402 (version not found)
			// rather than 40401 (subject not found)
			if versionStr != "" && versionStr != "latest" {
				writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound,
					fmt.Sprintf("Version %s not found", versionStr))
			} else {
				writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			}
			return
		}
		if errors.Is(err, storage.ErrVersionNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, err.Error())
			return
		}
		if errors.Is(err, storage.ErrInvalidVersion) {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	verbose := r.URL.Query().Get("verbose") == "true"
	resp := types.CompatibilityCheckResponse{
		IsCompatible: result.IsCompatible,
	}
	if verbose {
		resp.Messages = result.Messages
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetReferencedBy handles GET /subjects/{subject}/versions/{version}/referencedby
func (h *Handler) GetReferencedBy(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)
	versionStr := chi.URLParam(r, "version")

	version, err := parseVersion(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\"", versionStr))
		return
	}

	// Verify subject and version exist first
	_, err = h.registry.GetSchemaBySubjectVersion(r.Context(), registryCtx, subject, version)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		if errors.Is(err, storage.ErrVersionNotFound) || errors.Is(err, storage.ErrSchemaNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, "Version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	refs, err := h.registry.GetReferencedBy(r.Context(), registryCtx, subject, version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Convert to expected format (array of schema IDs that reference this schema)
	result := make([]int, 0, len(refs))
	for _, ref := range refs {
		schema, err := h.registry.GetSchemaBySubjectVersion(r.Context(), registryCtx, ref.Subject, ref.Version)
		if err != nil {
			// Skip schemas we can't find (might be deleted)
			continue
		}
		result = append(result, int(schema.ID))
	}

	writeJSON(w, http.StatusOK, result)
}

// GetMode handles GET /mode and GET /mode/{subject}
func (h *Handler) GetMode(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	defaultToGlobal := r.URL.Query().Get("defaultToGlobal") == "true"

	if subject != "" && !defaultToGlobal {
		// Subject-specific mode only, no fallback to global
		mode, err := h.registry.GetSubjectMode(r.Context(), registryCtx, subject)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, types.ErrorCodeSubjectModeNotFound,
					fmt.Sprintf("Subject '%s' does not have subject-level mode configured", subject))
				return
			}
			writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, types.ModeResponse{
			Mode: mode,
		})
		return
	}

	mode, err := h.registry.GetMode(r.Context(), registryCtx, subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ModeResponse{
		Mode: mode,
	})
}

// SetMode handles PUT /mode and PUT /mode/{subject}
func (h *Handler) SetMode(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	force := r.URL.Query().Get("force") == "true"

	var req types.ModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidMode, "Invalid request body")
		return
	}

	if err := h.registry.SetMode(r.Context(), registryCtx, subject, req.Mode, force); err != nil {
		if strings.Contains(err.Error(), "invalid mode") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidMode, err.Error())
			return
		}
		if errors.Is(err, storage.ErrOperationNotPermitted) {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ModeResponse{
		Mode: strings.ToUpper(req.Mode),
	})
}

// parseVersion parses a version string, handling "latest" and "-1".
// Returns errInvalidVersion for non-numeric strings, zero, or negative values (other than -1).
// parsePagination extracts offset and limit query params and applies them to a slice length.
// Returns the start and end indices for slicing.
func parsePagination(r *http.Request, total int) (start, end int) {
	start = 0
	end = total

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			start = o
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l >= 0 {
			end = start + l
		}
	}

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	return start, end
}

func parseVersion(s string) (int, error) {
	if s == "latest" || s == "-1" {
		return -1, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, errInvalidVersion
	}
	if v < 1 {
		return 0, errInvalidVersion
	}
	return v, nil
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, code int, message string) {
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(types.ErrorResponse{
		ErrorCode: code,
		Message:   message,
	})
}

// GetRawSchemaByID handles GET /schemas/ids/{id}/schema
func (h *Handler) GetRawSchemaByID(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid schema ID")
		return
	}

	schemaRecord, err := h.registry.GetSchemaByID(r.Context(), registryCtx, id)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	result := schemaRecord.Schema
	if format := r.URL.Query().Get("format"); format != "" {
		result = h.registry.FormatSchema(r.Context(), registryCtx, schemaRecord, format)
	}

	// Return raw schema as plain text
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(result)) // #nosec G705 -- schema content from storage, not user input
}

// GetSubjectsBySchemaID handles GET /schemas/ids/{id}/subjects
func (h *Handler) GetSubjectsBySchemaID(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid schema ID")
		return
	}

	deleted := r.URL.Query().Get("deleted") == "true"
	subjectFilter := r.URL.Query().Get("subject")

	subjects, err := h.registry.GetSubjectsBySchemaID(r.Context(), registryCtx, id, deleted)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	if subjectFilter != "" {
		var filtered []string
		for _, s := range subjects {
			if s == subjectFilter {
				filtered = append(filtered, s)
			}
		}
		if filtered == nil {
			filtered = []string{}
		}
		writeJSON(w, http.StatusOK, filtered)
		return
	}

	// Apply pagination
	start, end := parsePagination(r, len(subjects))
	writeJSON(w, http.StatusOK, subjects[start:end])
}

// GetVersionsBySchemaID handles GET /schemas/ids/{id}/versions
func (h *Handler) GetVersionsBySchemaID(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid schema ID")
		return
	}

	deleted := r.URL.Query().Get("deleted") == "true"
	subjectFilter := r.URL.Query().Get("subject")

	versions, err := h.registry.GetVersionsBySchemaID(r.Context(), registryCtx, id, deleted)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Convert to response format
	result := make([]types.SubjectVersionPair, 0, len(versions))
	for _, sv := range versions {
		if subjectFilter != "" && sv.Subject != subjectFilter {
			continue
		}
		result = append(result, types.SubjectVersionPair{
			Subject: sv.Subject,
			Version: sv.Version,
		})
	}

	// Apply pagination
	start, end := parsePagination(r, len(result))
	writeJSON(w, http.StatusOK, result[start:end])
}

// ListSchemas handles GET /schemas
func (h *Handler) ListSchemas(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)

	params := &storage.ListSchemasParams{
		SubjectPrefix: r.URL.Query().Get("subjectPrefix"),
		Deleted:       r.URL.Query().Get("deleted") == "true",
		LatestOnly:    r.URL.Query().Get("latestOnly") == "true",
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			params.Offset = offset
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			params.Limit = limit
		}
	}

	schemas, err := h.registry.ListSchemas(r.Context(), registryCtx, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Convert to response format
	result := make([]types.SchemaListItem, 0, len(schemas))
	for _, s := range schemas {
		result = append(result, types.SchemaListItem{
			Subject:    s.Subject,
			Version:    s.Version,
			ID:         s.ID,
			SchemaType: schemaTypeForResponse(s.SchemaType),
			Schema:     s.Schema,
			References: s.References,
		})
	}

	writeJSON(w, http.StatusOK, result)
}

// ImportSchemas handles POST /import/schemas
func (h *Handler) ImportSchemas(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)

	// Bulk import requires IMPORT mode (Confluent behavior)
	mode, modeErr := h.registry.GetMode(r.Context(), registryCtx, "")
	if modeErr != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeStorageError, "Failed to check mode")
		return
	}
	if mode != "IMPORT" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted,
			"Import is not permitted. The registry must be in IMPORT mode to import schemas.")
		return
	}

	var req types.ImportSchemasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if len(req.Schemas) == 0 {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "No schemas provided")
		return
	}

	// Convert API types to registry types
	importReqs := make([]registry.ImportSchemaRequest, len(req.Schemas))
	for i, s := range req.Schemas {
		importReqs[i] = registry.ImportSchemaRequest{
			ID:         s.ID,
			Subject:    s.Subject,
			Version:    s.Version,
			SchemaType: storage.SchemaType(strings.ToUpper(s.SchemaType)),
			Schema:     s.Schema,
			References: s.References,
		}
	}

	result, err := h.registry.ImportSchemas(r.Context(), registryCtx, importReqs)
	if err != nil {
		// Even on error, we might have partial results
		if result != nil {
			resp := types.ImportSchemasResponse{
				Imported: result.Imported,
				Errors:   result.Errors,
				Results:  make([]types.ImportSchemaResult, len(result.Results)),
			}
			for i, r := range result.Results {
				resp.Results[i] = types.ImportSchemaResult{
					ID:      r.ID,
					Subject: r.Subject,
					Version: r.Version,
					Success: r.Success,
					Error:   r.Error,
				}
			}
			// Return partial success with warning
			w.Header().Set("X-Warning", err.Error())
			writeJSON(w, http.StatusOK, resp)
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Convert result to response
	resp := types.ImportSchemasResponse{
		Imported: result.Imported,
		Errors:   result.Errors,
		Results:  make([]types.ImportSchemaResult, len(result.Results)),
	}
	for i, r := range result.Results {
		resp.Results[i] = types.ImportSchemaResult{
			ID:      r.ID,
			Subject: r.Subject,
			Version: r.Version,
			Success: r.Success,
			Error:   r.Error,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetRawSchemaByVersion handles GET /subjects/{subject}/versions/{version}/schema
func (h *Handler) GetRawSchemaByVersion(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	subject = h.resolveAlias(r.Context(), registryCtx, subject)
	versionStr := chi.URLParam(r, "version")

	version, err := parseVersion(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\"", versionStr))
		return
	}

	schemaRecord, err := h.registry.GetSchemaBySubjectVersion(r.Context(), registryCtx, subject, version)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		if errors.Is(err, storage.ErrVersionNotFound) {
			if h.isSubjectFullyDeleted(r.Context(), registryCtx, subject) {
				writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
				return
			}
			writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, "Version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	result := schemaRecord.Schema
	if format := r.URL.Query().Get("format"); format != "" {
		result = h.registry.FormatSchema(r.Context(), registryCtx, schemaRecord, format)
	}

	// Return raw schema as plain text
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(result)) // #nosec G705 -- schema content from storage, not user input
}

// DeleteGlobalConfig handles DELETE /config
func (h *Handler) DeleteGlobalConfig(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)

	level, err := h.registry.DeleteGlobalConfig(r.Context(), registryCtx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ConfigResponse{
		CompatibilityLevel: level,
	})
}

// DeleteMode handles DELETE /mode/{subject}
func (h *Handler) DeleteMode(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)

	mode, err := h.registry.DeleteMode(r.Context(), registryCtx, subject)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Mode not found for subject")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ModeResponse{
		Mode: mode,
	})
}

// GetContexts handles GET /contexts
func (h *Handler) GetContexts(w http.ResponseWriter, r *http.Request) {
	contexts, err := h.registry.ListContexts(r.Context())
	if err != nil {
		// Fallback to default context on error
		writeJSON(w, http.StatusOK, []string{"."})
		return
	}

	writeJSON(w, http.StatusOK, contexts)
}

// GetClusterID handles GET /v1/metadata/id
func (h *Handler) GetClusterID(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, types.ServerClusterIDResponse{
		ID: h.clusterID,
	})
}

// GetServerVersion handles GET /v1/metadata/version
func (h *Handler) GetServerVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, types.ServerVersionResponse{
		Version:   h.version,
		Commit:    h.commit,
		BuildTime: h.buildTime,
	})
}
