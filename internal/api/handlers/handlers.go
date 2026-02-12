// Package handlers provides HTTP request handlers.
package handlers

import (
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
// Per Confluent convention, AVRO is the default and is omitted (empty string)
// so that JSON omitempty will exclude it. Non-AVRO types are returned as-is.
func schemaTypeForResponse(st storage.SchemaType) string {
	if st == storage.SchemaTypeAvro || st == "" {
		return ""
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
// Returns an error message if writes are blocked, or empty string if allowed.
func (h *Handler) checkModeForWrite(r *http.Request, subject string) string {
	mode, _ := h.registry.GetMode(r.Context(), subject)
	if mode == "READONLY" || mode == "READONLY_OVERRIDE" {
		return mode
	}
	return ""
}

// HealthCheck handles GET /
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{})
}

// GetSchemaTypes handles GET /schemas/types
func (h *Handler) GetSchemaTypes(w http.ResponseWriter, r *http.Request) {
	types := h.registry.GetSchemaTypes()
	writeJSON(w, http.StatusOK, types)
}

// GetSchemaByID handles GET /schemas/ids/{id}
func (h *Handler) GetSchemaByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid schema ID")
		return
	}

	schema, err := h.registry.GetSchemaByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.SchemaByIDResponse{
		Schema:     schema.Schema,
		SchemaType: schemaTypeForResponse(schema.SchemaType),
		References: schema.References,
	})
}

// ListSubjects handles GET /subjects
func (h *Handler) ListSubjects(w http.ResponseWriter, r *http.Request) {
	deleted := r.URL.Query().Get("deleted") == "true"
	deletedOnly := r.URL.Query().Get("deletedOnly") == "true"
	subjectPrefix := r.URL.Query().Get("subjectPrefix")

	// deletedOnly implies including deleted subjects
	includeDeleted := deleted || deletedOnly

	subjects, err := h.registry.ListSubjects(r.Context(), includeDeleted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// For deletedOnly, filter to only deleted subjects by diffing with active set
	if deletedOnly {
		activeSubjects, _ := h.registry.ListSubjects(r.Context(), false)
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
	if limit >= 0 && limit < len(subjects) {
		subjects = subjects[:limit]
	}

	writeJSON(w, http.StatusOK, subjects)
}

// GetVersions handles GET /subjects/{subject}/versions
func (h *Handler) GetVersions(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	deleted := r.URL.Query().Get("deleted") == "true"
	deletedOnly := r.URL.Query().Get("deletedOnly") == "true"

	// deletedOnly takes precedence: if set, we include deleted and filter to only deleted
	includeDeleted := deleted || deletedOnly
	versions, err := h.registry.GetVersions(r.Context(), subject, includeDeleted)
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
		activeVersions, _ := h.registry.GetVersions(r.Context(), subject, false)
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

	writeJSON(w, http.StatusOK, versions)
}

// GetVersion handles GET /subjects/{subject}/versions/{version}
func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")

	version, err := parseVersion(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\"", versionStr))
		return
	}

	schema, err := h.registry.GetSchemaBySubjectVersion(r.Context(), subject, version)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		if errors.Is(err, storage.ErrVersionNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, "Version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	resp := types.SubjectVersionResponse{
		Subject:    schema.Subject,
		ID:         schema.ID,
		Version:    schema.Version,
		SchemaType: schemaTypeForResponse(schema.SchemaType),
		Schema:     schema.Schema,
	}
	if len(schema.References) > 0 {
		resp.References = schema.References
	}

	writeJSON(w, http.StatusOK, resp)
}

// RegisterSchema handles POST /subjects/{subject}/versions
func (h *Handler) RegisterSchema(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	// Check mode enforcement
	if mode := h.checkModeForWrite(r, subject); mode != "" {
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

	schema, err := h.registry.RegisterSchema(r.Context(), subject, req.Schema, schemaType, req.References)
	if err != nil {
		if strings.Contains(err.Error(), "invalid schema") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
			return
		}
		if strings.Contains(err.Error(), "unsupported schema type") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchemaType, err.Error())
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
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.RegisterSchemaResponse{
		ID: schema.ID,
	})
}

// LookupSchema handles POST /subjects/{subject}
func (h *Handler) LookupSchema(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	deleted := r.URL.Query().Get("deleted") == "true"

	var req types.LookupSchemaRequest
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

	schema, err := h.registry.LookupSchema(r.Context(), subject, req.Schema, schemaType, req.References, deleted)
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
	}
	if len(schema.References) > 0 {
		resp.References = schema.References
	}

	writeJSON(w, http.StatusOK, resp)
}

// DeleteSubject handles DELETE /subjects/{subject}
func (h *Handler) DeleteSubject(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	permanent := r.URL.Query().Get("permanent") == "true"

	// Check mode enforcement
	if mode := h.checkModeForWrite(r, subject); mode != "" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted,
			fmt.Sprintf("Subject '%s' is in %s mode", subject, mode))
		return
	}

	versions, err := h.registry.DeleteSubject(r.Context(), subject, permanent)
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
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, versions)
}

// DeleteVersion handles DELETE /subjects/{subject}/versions/{version}
func (h *Handler) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")
	permanent := r.URL.Query().Get("permanent") == "true"

	// Check mode enforcement
	if mode := h.checkModeForWrite(r, subject); mode != "" {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeOperationNotPermitted,
			fmt.Sprintf("Subject '%s' is in %s mode", subject, mode))
		return
	}

	version, err := parseVersion(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\"", versionStr))
		return
	}

	deletedVersion, err := h.registry.DeleteVersion(r.Context(), subject, version, permanent)
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
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotSoftDeleted,
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
	subject := chi.URLParam(r, "subject")
	defaultToGlobal := r.URL.Query().Get("defaultToGlobal") == "true"

	if subject != "" && !defaultToGlobal {
		// Subject-specific config only, no fallback to global
		level, err := h.registry.GetSubjectConfig(r.Context(), subject)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound,
					fmt.Sprintf("Subject '%s' does not have subject-level compatibility configured", subject))
				return
			}
			writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, types.ConfigResponse{
			CompatibilityLevel: level,
		})
		return
	}

	level, err := h.registry.GetConfig(r.Context(), subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ConfigResponse{
		CompatibilityLevel: level,
	})
}

// SetConfig handles PUT /config and PUT /config/{subject}
func (h *Handler) SetConfig(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	var req types.ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidCompatibilityLevel, "Invalid request body")
		return
	}

	if err := h.registry.SetConfig(r.Context(), subject, req.Compatibility); err != nil {
		if strings.Contains(err.Error(), "invalid compatibility") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidCompatibilityLevel, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, types.ConfigRequest{
		Compatibility: strings.ToUpper(req.Compatibility),
	})
}

// DeleteConfig handles DELETE /config/{subject}
func (h *Handler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	level, err := h.registry.DeleteConfig(r.Context(), subject)
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
	subject := chi.URLParam(r, "subject")
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

	result, err := h.registry.CheckCompatibility(r.Context(), subject, req.Schema, schemaType, req.References, versionStr)
	if err != nil {
		if strings.Contains(err.Error(), "invalid schema") {
			writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
			return
		}
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
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
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")

	version, err := parseVersion(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\"", versionStr))
		return
	}

	// Verify subject and version exist first
	_, err = h.registry.GetSchemaBySubjectVersion(r.Context(), subject, version)
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

	refs, err := h.registry.GetReferencedBy(r.Context(), subject, version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Convert to expected format (array of schema IDs that reference this schema)
	result := make([]int, 0, len(refs))
	for _, ref := range refs {
		schema, err := h.registry.GetSchemaBySubjectVersion(r.Context(), ref.Subject, ref.Version)
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
	subject := chi.URLParam(r, "subject")
	defaultToGlobal := r.URL.Query().Get("defaultToGlobal") == "true"

	if subject != "" && !defaultToGlobal {
		// Subject-specific mode only, no fallback to global
		mode, err := h.registry.GetSubjectMode(r.Context(), subject)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound,
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

	mode, err := h.registry.GetMode(r.Context(), subject)
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
	subject := chi.URLParam(r, "subject")
	force := r.URL.Query().Get("force") == "true"

	var req types.ModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidMode, "Invalid request body")
		return
	}

	if err := h.registry.SetMode(r.Context(), subject, req.Mode, force); err != nil {
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
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid schema ID")
		return
	}

	schema, err := h.registry.GetRawSchemaByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrSchemaNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Return raw schema as plain text
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(schema)) // #nosec G705 -- schema content from storage, not user input
}

// GetSubjectsBySchemaID handles GET /schemas/ids/{id}/subjects
func (h *Handler) GetSubjectsBySchemaID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid schema ID")
		return
	}

	deleted := r.URL.Query().Get("deleted") == "true"
	subjectFilter := r.URL.Query().Get("subject")

	subjects, err := h.registry.GetSubjectsBySchemaID(r.Context(), id, deleted)
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

	writeJSON(w, http.StatusOK, subjects)
}

// GetVersionsBySchemaID handles GET /schemas/ids/{id}/versions
func (h *Handler) GetVersionsBySchemaID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid schema ID")
		return
	}

	deleted := r.URL.Query().Get("deleted") == "true"
	subjectFilter := r.URL.Query().Get("subject")

	versions, err := h.registry.GetVersionsBySchemaID(r.Context(), id, deleted)
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

	writeJSON(w, http.StatusOK, result)
}

// ListSchemas handles GET /schemas
func (h *Handler) ListSchemas(w http.ResponseWriter, r *http.Request) {
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

	schemas, err := h.registry.ListSchemas(r.Context(), params)
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

	result, err := h.registry.ImportSchemas(r.Context(), importReqs)
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
	subject := chi.URLParam(r, "subject")
	versionStr := chi.URLParam(r, "version")

	version, err := parseVersion(versionStr)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidVersion,
			fmt.Sprintf("The specified version '%s' is not a valid version id. Allowed values are between [1, 2^31-1] and the string \"latest\"", versionStr))
		return
	}

	schema, err := h.registry.GetRawSchemaBySubjectVersion(r.Context(), subject, version)
	if err != nil {
		if errors.Is(err, storage.ErrSubjectNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		if errors.Is(err, storage.ErrVersionNotFound) {
			writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, "Version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Return raw schema as plain text
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(schema)) // #nosec G705 -- schema content from storage, not user input
}

// DeleteGlobalConfig handles DELETE /config
func (h *Handler) DeleteGlobalConfig(w http.ResponseWriter, r *http.Request) {
	level, err := h.registry.DeleteGlobalConfig(r.Context())
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
	subject := chi.URLParam(r, "subject")

	mode, err := h.registry.DeleteMode(r.Context(), subject)
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
	// Return default context for single-tenant deployment
	writeJSON(w, http.StatusOK, []string{"."})
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
