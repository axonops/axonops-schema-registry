package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/analysis"
	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// ValidateSchema handles POST /schemas/validate
func (h *Handler) ValidateSchema(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema     string `json:"schema"`
		SchemaType string `json:"schemaType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Schema == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Schema is required")
		return
	}
	st, ok := storage.ParseSchemaType(strings.ToUpper(req.SchemaType))
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema,
			fmt.Sprintf("Invalid schema type '%s'. Accepted types are AVRO, PROTOBUF, and JSON", req.SchemaType))
		return
	}

	registryCtx := getRegistryContext(r)
	result, err := h.registry.ValidateSchema(r.Context(), registryCtx, req.Schema, st, nil)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"is_valid":    result.Valid,
		"schema_type": string(st),
		"error":       result.Error,
	})
}

// NormalizeSchema handles POST /schemas/normalize
func (h *Handler) NormalizeSchema(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema     string `json:"schema"`
		SchemaType string `json:"schemaType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Schema == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Schema is required")
		return
	}
	st, ok := storage.ParseSchemaType(strings.ToUpper(req.SchemaType))
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema,
			fmt.Sprintf("Invalid schema type '%s'. Accepted types are AVRO, PROTOBUF, and JSON", req.SchemaType))
		return
	}

	registryCtx := getRegistryContext(r)
	result, err := h.registry.NormalizeSchema(r.Context(), registryCtx, req.Schema, st, nil)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schema_type": string(st),
		"canonical":   result.Normalized,
		"fingerprint": result.Fingerprint,
	})
}

// SearchSchemas handles POST /schemas/search
func (h *Handler) SearchSchemas(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Regex bool   `json:"regex"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Query is required")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}

	registryCtx := getRegistryContext(r)
	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	var re *regexp.Regexp
	if req.Regex {
		re, err = regexp.Compile(req.Query)
		if err != nil {
			writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid regex: "+err.Error())
			return
		}
	}

	type match struct {
		Subject    string `json:"subject"`
		Version    int    `json:"version"`
		SchemaType string `json:"schema_type"`
	}
	var matches []match
	for _, subj := range subjects {
		if len(matches) >= req.Limit {
			break
		}
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subj)
		if err != nil {
			continue
		}
		if req.Regex {
			if re.MatchString(latest.Schema) {
				matches = append(matches, match{Subject: subj, Version: latest.Version, SchemaType: string(latest.SchemaType)})
			}
		} else if strings.Contains(latest.Schema, req.Query) {
			matches = append(matches, match{Subject: subj, Version: latest.Version, SchemaType: string(latest.SchemaType)})
		}
	}
	if matches == nil {
		matches = []match{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"query":   req.Query,
		"count":   len(matches),
		"matches": matches,
	})
}

// FindSchemasByField handles POST /schemas/search/field
func (h *Handler) FindSchemasByField(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Field     string  `json:"field"`
		Mode      string  `json:"mode"`
		Threshold float64 `json:"threshold"`
		Limit     int     `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Field == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Field is required")
		return
	}
	if req.Mode == "" {
		req.Mode = "exact"
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.6
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}

	registryCtx := getRegistryContext(r)
	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	variants := analysis.NamingVariants(req.Field)

	var re *regexp.Regexp
	if req.Mode == "regex" {
		var err error
		re, err = regexp.Compile(req.Field)
		if err != nil {
			writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid regex: "+err.Error())
			return
		}
	}

	type fieldMatch struct {
		Subject    string  `json:"subject"`
		FieldName  string  `json:"field_name"`
		FieldType  string  `json:"field_type"`
		FieldPath  string  `json:"field_path"`
		Score      float64 `json:"score,omitempty"`
		SchemaType string  `json:"schema_type"`
	}
	var results []fieldMatch
	for _, subj := range subjects {
		if len(results) >= req.Limit {
			break
		}
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subj)
		if err != nil {
			continue
		}
		fields := analysis.ExtractFields(latest.Schema, latest.SchemaType)
		for _, f := range fields {
			switch req.Mode {
			case "exact":
				for _, v := range variants {
					if strings.EqualFold(f.Name, v) {
						results = append(results, fieldMatch{
							Subject: subj, FieldName: f.Name, FieldType: f.Type,
							FieldPath: f.Path, Score: 1.0, SchemaType: string(latest.SchemaType),
						})
						break
					}
				}
			case "fuzzy":
				score := analysis.FuzzyScore(req.Field, f.Name)
				if score >= req.Threshold {
					results = append(results, fieldMatch{
						Subject: subj, FieldName: f.Name, FieldType: f.Type,
						FieldPath: f.Path, Score: score, SchemaType: string(latest.SchemaType),
					})
				}
			case "regex":
				if re != nil && re.MatchString(f.Name) {
					results = append(results, fieldMatch{
						Subject: subj, FieldName: f.Name, FieldType: f.Type,
						FieldPath: f.Path, Score: 1.0, SchemaType: string(latest.SchemaType),
					})
				}
			}
		}
	}
	if results == nil {
		results = []fieldMatch{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"field":   req.Field,
		"mode":    req.Mode,
		"count":   len(results),
		"matches": results,
	})
}

// FindSchemasByType handles POST /schemas/search/type
func (h *Handler) FindSchemasByType(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TypePattern string `json:"type_pattern"`
		Regex       bool   `json:"regex"`
		Limit       int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.TypePattern == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "type_pattern is required")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}

	registryCtx := getRegistryContext(r)
	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	var re *regexp.Regexp
	if req.Regex {
		re, err = regexp.Compile(req.TypePattern)
		if err != nil {
			writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid regex: "+err.Error())
			return
		}
	}

	type typeMatch struct {
		Subject   string `json:"subject"`
		FieldName string `json:"field_name"`
		FieldType string `json:"field_type"`
	}
	var results []typeMatch
	for _, subj := range subjects {
		if len(results) >= req.Limit {
			break
		}
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subj)
		if err != nil {
			continue
		}
		fields := analysis.ExtractFields(latest.Schema, latest.SchemaType)
		for _, f := range fields {
			matched := false
			if req.Regex {
				matched = re.MatchString(f.Type)
			} else {
				matched = strings.EqualFold(f.Type, req.TypePattern) || strings.Contains(strings.ToLower(f.Type), strings.ToLower(req.TypePattern))
			}
			if matched {
				results = append(results, typeMatch{Subject: subj, FieldName: f.Name, FieldType: f.Type})
			}
		}
	}
	if results == nil {
		results = []typeMatch{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"type_pattern": req.TypePattern,
		"count":        len(results),
		"matches":      results,
	})
}

// FindSimilarSchemas handles POST /schemas/similar
func (h *Handler) FindSimilarSchemas(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subject   string  `json:"subject"`
		Threshold float64 `json:"threshold"`
		Limit     int     `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Subject == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Subject is required")
		return
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.3
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	registryCtx := getRegistryContext(r)
	source, err := h.registry.GetLatestSchema(r.Context(), registryCtx, req.Subject)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
		return
	}

	sourceFields := analysis.ExtractFields(source.Schema, source.SchemaType)
	sourceSet := make(map[string]bool)
	for _, f := range sourceFields {
		sourceSet[analysis.NormalizeFieldName(f.Name)] = true
	}

	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	type similar struct {
		Subject    string   `json:"subject"`
		Similarity float64  `json:"similarity"`
		Shared     []string `json:"shared_fields"`
	}
	var results []similar
	for _, subj := range subjects {
		if subj == req.Subject || len(results) >= req.Limit {
			continue
		}
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subj)
		if err != nil {
			continue
		}
		targetFields := analysis.ExtractFields(latest.Schema, latest.SchemaType)
		targetSet := make(map[string]bool)
		for _, f := range targetFields {
			targetSet[analysis.NormalizeFieldName(f.Name)] = true
		}

		// Jaccard similarity
		var shared []string
		for name := range sourceSet {
			if targetSet[name] {
				shared = append(shared, name)
			}
		}
		union := make(map[string]bool)
		for k := range sourceSet {
			union[k] = true
		}
		for k := range targetSet {
			union[k] = true
		}
		if len(union) == 0 {
			continue
		}
		sim := float64(len(shared)) / float64(len(union))
		if sim >= req.Threshold {
			results = append(results, similar{Subject: subj, Similarity: sim, Shared: shared})
		}
	}
	if results == nil {
		results = []similar{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subject": req.Subject,
		"count":   len(results),
		"similar": results,
	})
}

// ScoreSchemaQuality handles POST /schemas/quality
func (h *Handler) ScoreSchemaQuality(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema     string `json:"schema"`
		SchemaType string `json:"schemaType"`
		Subject    string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	registryCtx := getRegistryContext(r)

	schemaStr := req.Schema
	schemaType := req.SchemaType
	if schemaStr == "" && req.Subject != "" {
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, req.Subject)
		if err != nil {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		schemaStr = latest.Schema
		schemaType = string(latest.SchemaType)
	}
	if schemaStr == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Schema or subject is required")
		return
	}
	st, ok := storage.ParseSchemaType(strings.ToUpper(schemaType))
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema,
			fmt.Sprintf("Invalid schema type '%s'. Accepted types are AVRO, PROTOBUF, and JSON", schemaType))
		return
	}

	fields := analysis.ExtractFields(schemaStr, st)
	result := analysis.ScoreSchemaQuality(fields, schemaStr, string(st))
	writeJSON(w, http.StatusOK, result)
}

// GetSchemaComplexity handles POST /schemas/complexity
func (h *Handler) GetSchemaComplexity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema     string `json:"schema"`
		SchemaType string `json:"schemaType"`
		Subject    string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	registryCtx := getRegistryContext(r)

	schemaStr := req.Schema
	schemaType := req.SchemaType
	if schemaStr == "" && req.Subject != "" {
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, req.Subject)
		if err != nil {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		schemaStr = latest.Schema
		schemaType = string(latest.SchemaType)
	}
	if schemaStr == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Schema or subject is required")
		return
	}
	st, ok := storage.ParseSchemaType(strings.ToUpper(schemaType))
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, types.ErrorCodeInvalidSchema,
			fmt.Sprintf("Invalid schema type '%s'. Accepted types are AVRO, PROTOBUF, and JSON", schemaType))
		return
	}

	fields := analysis.ExtractFields(schemaStr, st)

	// Compute depth
	maxDepth := 0
	for _, f := range fields {
		depth := strings.Count(f.Path, ".") + 1
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	grade := "A"
	switch {
	case len(fields) > 50 || maxDepth > 5:
		grade = "D"
	case len(fields) > 30 || maxDepth > 4:
		grade = "C"
	case len(fields) > 15 || maxDepth > 3:
		grade = "B"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"schema_type": string(st),
		"field_count": len(fields),
		"max_depth":   maxDepth,
		"grade":       grade,
	})
}

// ValidateSubjectName handles POST /subjects/validate
func (h *Handler) ValidateSubjectName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subject  string `json:"subject"`
		Strategy string `json:"strategy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Subject == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Subject is required")
		return
	}
	if req.Strategy == "" {
		req.Strategy = "topic_name"
	}

	valid := true
	var issues []string
	var suggestion string

	switch req.Strategy {
	case "topic_name":
		if !strings.HasSuffix(req.Subject, "-key") && !strings.HasSuffix(req.Subject, "-value") {
			valid = false
			issues = append(issues, "TopicNameStrategy subjects must end with '-key' or '-value'")
			suggestion = req.Subject + "-value"
		}
	case "record_name":
		parts := strings.Split(req.Subject, ".")
		for _, p := range parts {
			if p == "" || !isValidIdentifier(p) {
				valid = false
				issues = append(issues, "RecordNameStrategy subjects must be valid qualified names (e.g., com.example.User)")
				break
			}
		}
	case "topic_record_name":
		if !strings.Contains(req.Subject, "-") {
			valid = false
			issues = append(issues, "TopicRecordNameStrategy subjects must contain a topic and record name")
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject":    req.Subject,
		"strategy":   req.Strategy,
		"valid":      valid,
		"issues":     issues,
		"suggestion": suggestion,
	})
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, ch := range s {
		if i == 0 && !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
			return false
		}
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
}

// MatchSubjects handles POST /subjects/match
func (h *Handler) MatchSubjects(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pattern   string  `json:"pattern"`
		Mode      string  `json:"mode"`
		Threshold float64 `json:"threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Pattern == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Pattern is required")
		return
	}
	if req.Mode == "" {
		req.Mode = "regex"
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.6
	}

	registryCtx := getRegistryContext(r)
	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	var matched []string
	switch req.Mode {
	case "regex":
		re, err := regexp.Compile(req.Pattern)
		if err != nil {
			writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid regex: "+err.Error())
			return
		}
		for _, s := range subjects {
			if re.MatchString(s) {
				matched = append(matched, s)
			}
		}
	case "glob":
		for _, s := range subjects {
			if globMatch(req.Pattern, s) {
				matched = append(matched, s)
			}
		}
	case "fuzzy":
		for _, s := range subjects {
			if analysis.FuzzyScore(req.Pattern, s) >= req.Threshold {
				matched = append(matched, s)
			}
		}
	}
	if matched == nil {
		matched = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pattern": req.Pattern,
		"mode":    req.Mode,
		"count":   len(matched),
		"matches": matched,
	})
}

// globMatch performs simple glob matching with * wildcards.
func globMatch(pattern, s string) bool {
	return originMatchesPattern(pattern, s)
}

// originMatchesPattern checks if a string matches a pattern with * wildcards (case-insensitive).
func originMatchesPattern(pattern, target string) bool {
	p := strings.ToLower(pattern)
	o := strings.ToLower(target)
	if !strings.Contains(p, "*") {
		return p == o
	}
	parts := strings.Split(p, "*")
	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		pos := strings.Index(o[idx:], part)
		if pos < 0 {
			return false
		}
		if i == 0 && pos != 0 {
			return false
		}
		idx += pos + len(part)
	}
	if last := parts[len(parts)-1]; last != "" {
		return strings.HasSuffix(o, last)
	}
	return true
}

// GetSchemaHistory handles GET /subjects/{subject}/history
func (h *Handler) GetSchemaHistory(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	schemas, err := h.registry.GetSchemasBySubject(r.Context(), registryCtx, subject, false)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
		return
	}

	type entry struct {
		Version    int    `json:"version"`
		SchemaID   int64  `json:"schema_id"`
		SchemaType string `json:"schema_type"`
	}
	var history []entry
	limit := 50
	for _, s := range schemas {
		if len(history) >= limit {
			break
		}
		history = append(history, entry{
			Version:    s.Version,
			SchemaID:   s.ID,
			SchemaType: string(s.SchemaType),
		})
	}
	if history == nil {
		history = []entry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subject": subject,
		"count":   len(history),
		"history": history,
	})
}

// GetDependencyGraph handles GET /subjects/{subject}/versions/{version}/dependencies
func (h *Handler) GetDependencyGraph(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	versionStr := chi.URLParam(r, "version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidVersion, "Invalid version")
		return
	}

	rec, err := h.registry.GetSchemaBySubjectVersion(r.Context(), registryCtx, subject, version)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
		return
	}

	// Build dependency graph via GetReferencedBy
	type node struct {
		Subject string `json:"subject"`
		Version int    `json:"version"`
	}
	referencedBy, _ := h.registry.GetReferencedBy(r.Context(), registryCtx, subject, version)
	var refs []node
	for _, sv := range referencedBy {
		refs = append(refs, node{Subject: sv.Subject, Version: sv.Version})
	}
	if refs == nil {
		refs = []node{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subject":       subject,
		"version":       version,
		"schema_id":     rec.ID,
		"referenced_by": refs,
	})
}

// DiffSchemas handles POST /subjects/{subject}/diff
func (h *Handler) DiffSchemas(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	var req struct {
		Version1 int `json:"version1"`
		Version2 int `json:"version2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Version1 == 0 {
		req.Version1 = 1
	}

	schema1, err := h.registry.GetSchemaBySubjectVersion(r.Context(), registryCtx, subject, req.Version1)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, fmt.Sprintf("Version %d not found", req.Version1))
		return
	}

	v2 := req.Version2
	if v2 == 0 {
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subject)
		if err != nil {
			writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
			return
		}
		v2 = latest.Version
	}
	schema2, err := h.registry.GetSchemaBySubjectVersion(r.Context(), registryCtx, subject, v2)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeVersionNotFound, fmt.Sprintf("Version %d not found", v2))
		return
	}

	fields1 := analysis.ExtractFields(schema1.Schema, schema1.SchemaType)
	fields2 := analysis.ExtractFields(schema2.Schema, schema2.SchemaType)

	set1 := make(map[string]string)
	for _, f := range fields1 {
		set1[f.Name] = f.Type
	}
	set2 := make(map[string]string)
	for _, f := range fields2 {
		set2[f.Name] = f.Type
	}

	var added, removed, changed []map[string]string
	for name, t := range set2 {
		if _, ok := set1[name]; !ok {
			added = append(added, map[string]string{"field": name, "type": t})
		} else if set1[name] != t {
			changed = append(changed, map[string]string{"field": name, "old_type": set1[name], "new_type": t})
		}
	}
	for name, t := range set1 {
		if _, ok := set2[name]; !ok {
			removed = append(removed, map[string]string{"field": name, "type": t})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject":  subject,
		"version1": req.Version1,
		"version2": v2,
		"added":    added,
		"removed":  removed,
		"changed":  changed,
	})
}

// SuggestSchemaEvolution handles POST /subjects/{subject}/evolve
func (h *Handler) SuggestSchemaEvolution(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	var req struct {
		Changes []struct {
			Action string `json:"action"`
			Field  string `json:"field"`
			Type   string `json:"type"`
		} `json:"changes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subject)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
		return
	}

	configFull, err := h.registry.GetConfigFull(r.Context(), registryCtx, subject)
	if err != nil {
		configFull = &storage.ConfigRecord{CompatibilityLevel: "BACKWARD"}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject":             subject,
		"current_version":     latest.Version,
		"compatibility_level": configFull.CompatibilityLevel,
		"changes_requested":   len(req.Changes),
		"message":             "Schema evolution suggestions are available via the MCP tool suggest_schema_evolution for richer analysis",
	})
}

// PlanMigrationPath handles POST /subjects/{subject}/migrate
func (h *Handler) PlanMigrationPath(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	var req struct {
		TargetSchema string `json:"target_schema"`
		SchemaType   string `json:"schema_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.TargetSchema == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "target_schema is required")
		return
	}

	latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subject)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
		return
	}

	configFull, err := h.registry.GetConfigFull(r.Context(), registryCtx, subject)
	if err != nil {
		configFull = &storage.ConfigRecord{CompatibilityLevel: "BACKWARD"}
	}

	sourceFields := analysis.ExtractFields(latest.Schema, latest.SchemaType)
	st := storage.SchemaType(strings.ToUpper(req.SchemaType))
	if st == "" {
		st = latest.SchemaType
	}
	targetFields := analysis.ExtractFields(req.TargetSchema, st)

	sourceSet := make(map[string]bool)
	for _, f := range sourceFields {
		sourceSet[f.Name] = true
	}
	targetSet := make(map[string]bool)
	for _, f := range targetFields {
		targetSet[f.Name] = true
	}

	var steps []string
	for _, f := range targetFields {
		if !sourceSet[f.Name] {
			steps = append(steps, fmt.Sprintf("Add field '%s' (type: %s) with a default value", f.Name, f.Type))
		}
	}
	for _, f := range sourceFields {
		if !targetSet[f.Name] {
			steps = append(steps, fmt.Sprintf("Remove field '%s' (may require compatibility level change)", f.Name))
		}
	}
	if steps == nil {
		steps = []string{"No migration steps needed — schemas have the same fields"}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject":             subject,
		"current_version":     latest.Version,
		"compatibility_level": configFull.CompatibilityLevel,
		"steps":               steps,
		"step_count":          len(steps),
	})
}

// ExportSubject handles GET /subjects/{subject}/export
func (h *Handler) ExportSubject(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	schemas, err := h.registry.GetSchemasBySubject(r.Context(), registryCtx, subject, false)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
		return
	}

	type exportEntry struct {
		Subject    string `json:"subject"`
		Version    int    `json:"version"`
		ID         int64  `json:"id"`
		Schema     string `json:"schema"`
		SchemaType string `json:"schema_type"`
	}
	var entries []exportEntry
	for _, s := range schemas {
		entries = append(entries, exportEntry{
			Subject:    subject,
			Version:    s.Version,
			ID:         s.ID,
			Schema:     s.Schema,
			SchemaType: string(s.SchemaType),
		})
	}
	if entries == nil {
		entries = []exportEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subject":  subject,
		"count":    len(entries),
		"versions": entries,
	})
}

// ExportSchema handles GET /subjects/{subject}/versions/{version}/export
func (h *Handler) ExportSchema(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	versionStr := chi.URLParam(r, "version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidVersion, "Invalid version")
		return
	}

	rec, err := h.registry.GetSchemaBySubjectVersion(r.Context(), registryCtx, subject, version)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSchemaNotFound, "Schema not found")
		return
	}

	configFull, _ := h.registry.GetConfigFull(r.Context(), registryCtx, subject)
	level := "BACKWARD"
	if configFull != nil {
		level = configFull.CompatibilityLevel
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject":             subject,
		"version":             version,
		"id":                  rec.ID,
		"schema":              rec.Schema,
		"schema_type":         string(rec.SchemaType),
		"compatibility_level": level,
	})
}

// CheckCompatibilityMulti handles POST /compatibility/check
func (h *Handler) CheckCompatibilityMulti(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema     string   `json:"schema"`
		SchemaType string   `json:"schemaType"`
		Subjects   []string `json:"subjects"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Schema == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Schema is required")
		return
	}

	registryCtx := getRegistryContext(r)
	st := storage.SchemaType(strings.ToUpper(req.SchemaType))
	if st == "" {
		st = storage.SchemaTypeAvro
	}

	type subjectResult struct {
		Subject    string `json:"subject"`
		Compatible bool   `json:"is_compatible"`
		Error      string `json:"error,omitempty"`
	}
	var results []subjectResult
	for _, subj := range req.Subjects {
		result, err := h.registry.CheckCompatibility(r.Context(), registryCtx, subj, req.Schema, st, nil, "latest")
		if err != nil {
			results = append(results, subjectResult{Subject: subj, Compatible: false, Error: err.Error()})
		} else {
			results = append(results, subjectResult{Subject: subj, Compatible: result.IsCompatible})
		}
	}
	if results == nil {
		results = []subjectResult{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}

// SuggestCompatibleChange handles POST /compatibility/subjects/{subject}/suggest
func (h *Handler) SuggestCompatibleChange(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	configFull, err := h.registry.GetConfigFull(r.Context(), registryCtx, subject)
	if err != nil {
		configFull = &storage.ConfigRecord{CompatibilityLevel: "BACKWARD"}
	}

	level := configFull.CompatibilityLevel
	var suggestions []string
	switch strings.ToUpper(level) {
	case "BACKWARD", "BACKWARD_TRANSITIVE":
		suggestions = append(suggestions, "Add new fields with default values")
		suggestions = append(suggestions, "Do NOT remove existing fields")
		suggestions = append(suggestions, "Do NOT change field types")
	case "FORWARD", "FORWARD_TRANSITIVE":
		suggestions = append(suggestions, "Remove fields (new consumers will ignore them)")
		suggestions = append(suggestions, "Do NOT add required fields without defaults")
	case "FULL", "FULL_TRANSITIVE":
		suggestions = append(suggestions, "Only add optional fields with defaults")
		suggestions = append(suggestions, "Do NOT remove or rename fields")
	case "NONE":
		suggestions = append(suggestions, "Any change is allowed (no compatibility checks)")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject":             subject,
		"compatibility_level": level,
		"suggestions":         suggestions,
	})
}

// ExplainCompatibilityFailure handles POST /compatibility/subjects/{subject}/explain
func (h *Handler) ExplainCompatibilityFailure(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	var req struct {
		Schema     string `json:"schema"`
		SchemaType string `json:"schemaType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Schema == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Schema is required")
		return
	}

	st := storage.SchemaType(strings.ToUpper(req.SchemaType))
	if st == "" {
		st = storage.SchemaTypeAvro
	}

	compatResult, err := h.registry.CheckCompatibility(r.Context(), registryCtx, subject, req.Schema, st, nil, "latest")

	configFull, _ := h.registry.GetConfigFull(r.Context(), registryCtx, subject)
	level := "BACKWARD"
	if configFull != nil {
		level = configFull.CompatibilityLevel
	}

	isCompatible := false
	if compatResult != nil {
		isCompatible = compatResult.IsCompatible
	}

	resp := map[string]any{
		"subject":             subject,
		"compatibility_level": level,
		"is_compatible":       isCompatible,
	}
	if err != nil {
		resp["error"] = err.Error()
		resp["explanation"] = "The schema is not compatible with the existing schema under " + level + " compatibility"
	}
	writeJSON(w, http.StatusOK, resp)
}

// CompareSubjects handles POST /compatibility/compare
func (h *Handler) CompareSubjects(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subject1 string `json:"subject1"`
		Subject2 string `json:"subject2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}
	if req.Subject1 == "" || req.Subject2 == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Both subject1 and subject2 are required")
		return
	}

	registryCtx := getRegistryContext(r)
	s1, err := h.registry.GetLatestSchema(r.Context(), registryCtx, req.Subject1)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject1 not found")
		return
	}
	s2, err := h.registry.GetLatestSchema(r.Context(), registryCtx, req.Subject2)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject2 not found")
		return
	}

	fields1 := analysis.ExtractFields(s1.Schema, s1.SchemaType)
	fields2 := analysis.ExtractFields(s2.Schema, s2.SchemaType)

	set1 := make(map[string]string)
	for _, f := range fields1 {
		set1[f.Name] = f.Type
	}
	set2 := make(map[string]string)
	for _, f := range fields2 {
		set2[f.Name] = f.Type
	}

	var onlyIn1, onlyIn2, shared []string
	for name := range set1 {
		if _, ok := set2[name]; ok {
			shared = append(shared, name)
		} else {
			onlyIn1 = append(onlyIn1, name)
		}
	}
	for name := range set2 {
		if _, ok := set1[name]; !ok {
			onlyIn2 = append(onlyIn2, name)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject1":     req.Subject1,
		"subject2":     req.Subject2,
		"shared":       shared,
		"only_in_sub1": onlyIn1,
		"only_in_sub2": onlyIn2,
	})
}

// GetRegistryStatistics handles GET /statistics
func (h *Handler) GetRegistryStatistics(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)
	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	typeCounts := map[string]int{}
	totalVersions := 0
	for _, subj := range subjects {
		versions, err := h.registry.GetVersions(r.Context(), registryCtx, subj, false)
		if err == nil {
			totalVersions += len(versions)
		}
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subj)
		if err == nil {
			typeCounts[string(latest.SchemaType)]++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject_count": len(subjects),
		"version_count": totalVersions,
		"type_counts":   typeCounts,
	})
}

// CheckFieldConsistency handles GET /statistics/fields/{field}
func (h *Handler) CheckFieldConsistency(w http.ResponseWriter, r *http.Request) {
	field := chi.URLParam(r, "field")
	if field == "" {
		writeError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Field name is required")
		return
	}

	registryCtx := getRegistryContext(r)
	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	variants := analysis.NamingVariants(field)

	type fieldUsage struct {
		Subject   string `json:"subject"`
		FieldName string `json:"field_name"`
		FieldType string `json:"field_type"`
	}
	var usages []fieldUsage
	typeCounts := map[string]int{}

	for _, subj := range subjects {
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subj)
		if err != nil {
			continue
		}
		fields := analysis.ExtractFields(latest.Schema, latest.SchemaType)
		for _, f := range fields {
			normalized := analysis.NormalizeFieldName(f.Name)
			for _, v := range variants {
				if analysis.NormalizeFieldName(v) == normalized {
					usages = append(usages, fieldUsage{Subject: subj, FieldName: f.Name, FieldType: f.Type})
					typeCounts[f.Type]++
					break
				}
			}
		}
	}
	if usages == nil {
		usages = []fieldUsage{}
	}

	consistent := len(typeCounts) <= 1

	writeJSON(w, http.StatusOK, map[string]any{
		"field":       field,
		"consistent":  consistent,
		"type_counts": typeCounts,
		"usages":      usages,
	})
}

// DetectSchemaPatterns handles GET /statistics/patterns
func (h *Handler) DetectSchemaPatterns(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)
	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	fieldCounts := map[string]int{}
	for _, subj := range subjects {
		latest, err := h.registry.GetLatestSchema(r.Context(), registryCtx, subj)
		if err != nil {
			continue
		}
		fields := analysis.ExtractFields(latest.Schema, latest.SchemaType)
		for _, f := range fields {
			fieldCounts[analysis.NormalizeFieldName(f.Name)]++
		}
	}

	// Find common fields (appearing in 2+ subjects)
	type commonField struct {
		Field string `json:"field"`
		Count int    `json:"count"`
	}
	var common []commonField
	for name, count := range fieldCounts {
		if count >= 2 {
			common = append(common, commonField{Field: name, Count: count})
		}
	}
	if common == nil {
		common = []commonField{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subject_count": len(subjects),
		"common_fields": common,
		"pattern_count": len(common),
	})
}

// CountSubjects handles GET /subjects/count
func (h *Handler) CountSubjects(w http.ResponseWriter, r *http.Request) {
	registryCtx := getRegistryContext(r)
	subjects, err := h.registry.ListSubjects(r.Context(), registryCtx, false)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(subjects),
	})
}

// CountVersions handles GET /subjects/{subject}/versions/count
func (h *Handler) CountVersions(w http.ResponseWriter, r *http.Request) {
	registryCtx, subject := resolveSubjectAndContext(r)
	if rejectGlobalContext(w, registryCtx) {
		return
	}

	versions, err := h.registry.GetVersions(r.Context(), registryCtx, subject, false)
	if err != nil {
		writeError(w, http.StatusNotFound, types.ErrorCodeSubjectNotFound, "Subject not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subject": subject,
		"count":   len(versions),
	})
}
