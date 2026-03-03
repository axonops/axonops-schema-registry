package mcp

import (
	"context"
	"regexp"
	"sort"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerIntelligenceTools() {
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "find_schemas_by_field",
		Description: "Find all schemas containing a field with the given name. Exact mode auto-generates naming variants (snake_case, camelCase, PascalCase, kebab-case). Fuzzy mode uses Levenshtein distance with configurable threshold (default 0.7). Regex mode compiles the field name as a regular expression.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "find_schemas_by_field", s.handleFindSchemasByField))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "find_schemas_by_type",
		Description: "Find all schemas containing fields of a given type (e.g., 'int', 'string', 'record').",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "find_schemas_by_type", s.handleFindSchemasByType))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "find_similar_schemas",
		Description: "Find schemas structurally similar to a given subject using Jaccard similarity coefficient (|shared fields| / |total unique fields|). Field names are normalized to snake_case before comparison. Returns similarity scores (0.0-1.0) and lists of shared fields.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "find_similar_schemas", s.handleFindSimilarSchemas))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "score_schema_quality",
		Description: "Score a schema's quality (0-100, grades A-F) across four categories: Naming (25 pts, checks snake_case convention), Documentation (25 pts, checks field doc/description coverage), Type Safety (25 pts, penalizes generic types like string/bytes/any/object), Evolution Readiness (25 pts, checks for defaults, namespace, and schema-level docs). Returns per-category breakdown and actionable quick_wins.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "score_schema_quality", s.handleScoreSchemaQuality))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "check_field_consistency",
		Description: "Check if a field name is used with the same type across all schemas. Generates naming variants (snake_case, camelCase, PascalCase, kebab-case) to match fields regardless of convention. Reports type_counts map and per-subject usages. Detects type drift (e.g., user_id as long in one schema and string in another).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "check_field_consistency", s.handleCheckFieldConsistency))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_schema_complexity",
		Description: "Compute complexity metrics and grade (A-D) for a schema. Measures field_count (total fields including nested) and max_depth (deepest nesting level via dot-notation paths). Grades: A (≤15 fields, ≤3 depth), B (≤30, ≤4), C (≤50, ≤5), D (>50 or >5). Grade D schemas should be decomposed into referenced sub-schemas.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_schema_complexity", s.handleGetSchemaComplexity))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "detect_schema_patterns",
		Description: "Scan the registry to detect naming patterns, common field groups, and evolution statistics.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "detect_schema_patterns", s.handleDetectSchemaPatterns))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "suggest_schema_evolution",
		Description: "Generate concrete schema code for a compatible evolution step (add field, deprecate field, add enum symbol).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "suggest_schema_evolution", s.handleSuggestSchemaEvolution))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "plan_migration_path",
		Description: "Compute a multi-step migration plan from a source schema to a target schema, decomposed into individually compatible steps.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "plan_migration_path", s.handlePlanMigrationPath))
}

// --- find_schemas_by_field ---

type findSchemasByFieldInput struct {
	Field     string  `json:"field"`
	MatchType string  `json:"match_type,omitempty"` // "exact", "fuzzy", "regex"
	Threshold float64 `json:"threshold,omitempty"`
}

type fieldSchemaMatch struct {
	Subject   string  `json:"subject"`
	Field     string  `json:"field"`
	FieldType string  `json:"field_type"`
	Path      string  `json:"path"`
	Score     float64 `json:"score,omitempty"`
}

func (s *Server) handleFindSchemasByField(ctx context.Context, _ *gomcp.CallToolRequest, input findSchemasByFieldInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	matchType := input.MatchType
	if matchType == "" {
		matchType = "exact"
	}
	threshold := input.Threshold
	if threshold <= 0 {
		threshold = 0.7
	}

	var re *regexp.Regexp
	if matchType == "regex" {
		re, err = regexp.Compile(input.Field)
		if err != nil {
			return errorResult(err), nil, nil
		}
	}

	var matches []fieldSchemaMatch
	for _, subj := range subjects {
		record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subj)
		if err != nil {
			continue
		}
		fields := ExtractFields(record.Schema, record.SchemaType)
		for _, f := range fields {
			var matched bool
			var score float64
			switch matchType {
			case "exact":
				if strings.EqualFold(f.Name, input.Field) {
					matched = true
					score = 1.0
				}
			case "fuzzy":
				score = FuzzyScore(input.Field, f.Name)
				if score >= threshold {
					matched = true
				}
			case "regex":
				if re != nil && re.MatchString(f.Name) {
					matched = true
					score = 1.0
				}
			}
			if matched {
				m := fieldSchemaMatch{
					Subject:   subj,
					Field:     f.Name,
					FieldType: f.Type,
					Path:      f.Path,
				}
				if matchType == "fuzzy" {
					m.Score = score
				}
				matches = append(matches, m)
			}
		}
	}
	if matches == nil {
		matches = []fieldSchemaMatch{}
	}

	return jsonResult(map[string]any{
		"matches": matches,
		"count":   len(matches),
	})
}

// --- find_schemas_by_type ---

type findSchemasByTypeInput struct {
	TypePattern string `json:"type_pattern"`
	Regex       bool   `json:"regex,omitempty"`
}

func (s *Server) handleFindSchemasByType(ctx context.Context, _ *gomcp.CallToolRequest, input findSchemasByTypeInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	var re *regexp.Regexp
	if input.Regex {
		re, err = regexp.Compile(input.TypePattern)
		if err != nil {
			return errorResult(err), nil, nil
		}
	}

	type typeMatch struct {
		Subject   string `json:"subject"`
		Field     string `json:"field"`
		Path      string `json:"path"`
		FieldType string `json:"field_type"`
	}

	var matches []typeMatch
	for _, subj := range subjects {
		record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subj)
		if err != nil {
			continue
		}
		fields := ExtractFields(record.Schema, record.SchemaType)
		for _, f := range fields {
			matched := false
			if input.Regex && re != nil {
				matched = re.MatchString(f.Type)
			} else {
				matched = strings.EqualFold(f.Type, input.TypePattern)
			}
			if matched {
				matches = append(matches, typeMatch{
					Subject:   subj,
					Field:     f.Name,
					Path:      f.Path,
					FieldType: f.Type,
				})
			}
		}
	}
	if matches == nil {
		matches = []typeMatch{}
	}

	return jsonResult(map[string]any{
		"matches": matches,
		"count":   len(matches),
	})
}

// --- find_similar_schemas ---

type findSimilarSchemasInput struct {
	Subject   string  `json:"subject"`
	Threshold float64 `json:"threshold,omitempty"`
}

type similarSchemaMatch struct {
	Subject      string   `json:"subject"`
	Similarity   float64  `json:"similarity"`
	CommonFields []string `json:"common_fields"`
}

func (s *Server) handleFindSimilarSchemas(ctx context.Context, _ *gomcp.CallToolRequest, input findSimilarSchemasInput) (*gomcp.CallToolResult, any, error) {
	threshold := input.Threshold
	if threshold <= 0 {
		threshold = 0.3
	}

	record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}

	sourceFields := ExtractFields(record.Schema, record.SchemaType)
	sourceSet := make(map[string]bool)
	for _, f := range sourceFields {
		sourceSet[NormalizeFieldName(f.Name)] = true
	}

	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	var matches []similarSchemaMatch
	for _, subj := range subjects {
		if subj == input.Subject {
			continue
		}
		other, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subj)
		if err != nil {
			continue
		}
		otherFields := ExtractFields(other.Schema, other.SchemaType)
		otherSet := make(map[string]bool)
		for _, f := range otherFields {
			otherSet[NormalizeFieldName(f.Name)] = true
		}

		// Jaccard similarity
		intersect := 0
		var common []string
		for name := range sourceSet {
			if otherSet[name] {
				intersect++
				common = append(common, name)
			}
		}
		union := len(sourceSet) + len(otherSet) - intersect
		if union == 0 {
			continue
		}
		similarity := float64(intersect) / float64(union)
		if similarity >= threshold {
			sort.Strings(common)
			matches = append(matches, similarSchemaMatch{
				Subject:      subj,
				Similarity:   similarity,
				CommonFields: common,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].Similarity > matches[j].Similarity })
	if matches == nil {
		matches = []similarSchemaMatch{}
	}

	return jsonResult(map[string]any{
		"subject": input.Subject,
		"matches": matches,
		"count":   len(matches),
	})
}

// --- score_schema_quality ---

type scoreSchemaQualityInput struct {
	Subject string `json:"subject,omitempty"`
	Schema  string `json:"schema,omitempty"`
	Type    string `json:"schema_type,omitempty"`
}

func (s *Server) handleScoreSchemaQuality(ctx context.Context, _ *gomcp.CallToolRequest, input scoreSchemaQualityInput) (*gomcp.CallToolResult, any, error) {
	schemaStr := input.Schema
	schemaType := storage.SchemaType(input.Type)

	if input.Subject != "" && schemaStr == "" {
		record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.Subject)
		if err != nil {
			return errorResult(err), nil, nil
		}
		schemaStr = record.Schema
		schemaType = record.SchemaType
	}

	if schemaStr == "" {
		return errorResult(errMissingInput("schema or subject")), nil, nil
	}

	fields := ExtractFields(schemaStr, schemaType)
	result := ScoreSchemaQuality(fields, schemaStr, string(schemaType))

	return jsonResult(result)
}

// --- check_field_consistency ---

type checkFieldConsistencyInput struct {
	Field string `json:"field"`
}

type fieldUsage struct {
	Subject   string `json:"subject"`
	FieldType string `json:"field_type"`
	Path      string `json:"path"`
}

func (s *Server) handleCheckFieldConsistency(ctx context.Context, _ *gomcp.CallToolRequest, input checkFieldConsistencyInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	var usages []fieldUsage
	typeCounts := map[string]int{}

	for _, subj := range subjects {
		record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subj)
		if err != nil {
			continue
		}
		fields := ExtractFields(record.Schema, record.SchemaType)
		for _, f := range fields {
			if strings.EqualFold(NormalizeFieldName(f.Name), NormalizeFieldName(input.Field)) {
				usages = append(usages, fieldUsage{
					Subject:   subj,
					FieldType: f.Type,
					Path:      f.Path,
				})
				typeCounts[f.Type]++
			}
		}
	}

	consistent := len(typeCounts) <= 1
	if usages == nil {
		usages = []fieldUsage{}
	}

	return jsonResult(map[string]any{
		"field":       input.Field,
		"consistent":  consistent,
		"type_counts": typeCounts,
		"usages":      usages,
		"total":       len(usages),
	})
}

// --- get_schema_complexity ---

type getSchemaComplexityInput struct {
	Subject string `json:"subject,omitempty"`
	Schema  string `json:"schema,omitempty"`
	Type    string `json:"schema_type,omitempty"`
}

func (s *Server) handleGetSchemaComplexity(ctx context.Context, _ *gomcp.CallToolRequest, input getSchemaComplexityInput) (*gomcp.CallToolResult, any, error) {
	schemaStr := input.Schema
	schemaType := storage.SchemaType(input.Type)

	if input.Subject != "" && schemaStr == "" {
		record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.Subject)
		if err != nil {
			return errorResult(err), nil, nil
		}
		schemaStr = record.Schema
		schemaType = record.SchemaType
	}

	if schemaStr == "" {
		return errorResult(errMissingInput("schema or subject")), nil, nil
	}

	fields := ExtractFields(schemaStr, schemaType)

	// Compute complexity metrics
	maxDepth := 0
	unionCount := 0
	for _, f := range fields {
		depth := strings.Count(f.Path, ".") + 1
		if depth > maxDepth {
			maxDepth = depth
		}
		if strings.HasPrefix(f.Type, "union[") {
			unionCount++
		}
	}

	// Count nested records via schema content
	nestedRecords := strings.Count(schemaStr, `"type":"record"`) + strings.Count(schemaStr, `"type": "record"`)
	if nestedRecords > 0 {
		nestedRecords-- // Subtract the root record
	}

	level := "low"
	if len(fields) > 20 || maxDepth > 3 {
		level = "high"
	} else if len(fields) > 10 || maxDepth > 2 {
		level = "medium"
	}

	return jsonResult(map[string]any{
		"field_count":    len(fields),
		"max_depth":      maxDepth,
		"union_count":    unionCount,
		"nested_records": nestedRecords,
		"complexity":     level,
	})
}

// --- detect_schema_patterns ---

type detectSchemaPatternsInput struct{}

func (s *Server) handleDetectSchemaPatterns(ctx context.Context, _ *gomcp.CallToolRequest, _ detectSchemaPatternsInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	// Naming patterns (suffixes)
	suffixCounts := map[string]int{}
	typeCounts := map[string]int{}
	fieldFrequency := map[string]int{} // how many schemas use each field name
	totalVersions := 0
	multiVersionSubjects := 0

	for _, subj := range subjects {
		// Naming pattern: extract suffix
		parts := strings.Split(subj, "-")
		if len(parts) > 1 {
			suffixCounts[parts[len(parts)-1]]++
		}

		versions, err := s.registry.GetVersions(ctx, registrycontext.DefaultContext, subj, false)
		if err != nil {
			continue
		}
		totalVersions += len(versions)
		if len(versions) > 1 {
			multiVersionSubjects++
		}

		record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subj)
		if err != nil {
			continue
		}
		typeCounts[string(record.SchemaType)]++

		fields := ExtractFields(record.Schema, record.SchemaType)
		seen := map[string]bool{}
		for _, f := range fields {
			normalized := NormalizeFieldName(f.Name)
			if !seen[normalized] {
				seen[normalized] = true
				fieldFrequency[normalized]++
			}
		}
	}

	// Find common fields (in >30% of schemas)
	threshold := len(subjects) * 3 / 10
	if threshold < 2 {
		threshold = 2
	}
	var commonFields []map[string]any
	for field, count := range fieldFrequency {
		if count >= threshold {
			commonFields = append(commonFields, map[string]any{
				"field": field,
				"count": count,
			})
		}
	}
	sort.Slice(commonFields, func(i, j int) bool {
		return commonFields[i]["count"].(int) > commonFields[j]["count"].(int)
	})

	// Top naming suffixes
	var topSuffixes []map[string]any
	for suffix, count := range suffixCounts {
		if count >= 2 {
			topSuffixes = append(topSuffixes, map[string]any{
				"suffix": suffix,
				"count":  count,
			})
		}
	}
	sort.Slice(topSuffixes, func(i, j int) bool {
		return topSuffixes[i]["count"].(int) > topSuffixes[j]["count"].(int)
	})

	avgVersions := 0.0
	if len(subjects) > 0 {
		avgVersions = float64(totalVersions) / float64(len(subjects))
	}

	return jsonResult(map[string]any{
		"total_subjects":         len(subjects),
		"schema_types":           typeCounts,
		"naming_suffixes":        topSuffixes,
		"common_fields":          commonFields,
		"avg_versions":           avgVersions,
		"multi_version_subjects": multiVersionSubjects,
	})
}

// --- suggest_schema_evolution ---

type suggestSchemaEvolutionInput struct {
	Subject    string `json:"subject"`
	ChangeType string `json:"change_type"` // "add_field", "deprecate_field", "add_enum_symbol"
	FieldName  string `json:"field_name,omitempty"`
	FieldType  string `json:"field_type,omitempty"`
	EnumSymbol string `json:"enum_symbol,omitempty"`
}

func (s *Server) handleSuggestSchemaEvolution(ctx context.Context, _ *gomcp.CallToolRequest, input suggestSchemaEvolutionInput) (*gomcp.CallToolResult, any, error) {
	record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}

	level, _ := s.registry.GetConfig(ctx, registrycontext.DefaultContext, input.Subject)

	var suggestion map[string]any

	switch input.ChangeType {
	case "add_field":
		suggestion = suggestAddField(record, level, input.FieldName, input.FieldType)
	case "deprecate_field":
		suggestion = suggestDeprecateField(record, input.FieldName)
	case "add_enum_symbol":
		suggestion = suggestAddEnumSymbol(record, input.EnumSymbol)
	default:
		return jsonResult(map[string]any{
			"error":           "unsupported change_type",
			"supported_types": []string{"add_field", "deprecate_field", "add_enum_symbol"},
		})
	}

	suggestion["subject"] = input.Subject
	suggestion["current_version"] = record.Version
	suggestion["compatibility_level"] = level

	return jsonResult(suggestion)
}

func suggestAddField(record *storage.SchemaRecord, level, fieldName, fieldType string) map[string]any {
	if fieldName == "" {
		fieldName = "new_field"
	}
	if fieldType == "" {
		fieldType = "string"
	}

	needsDefault := strings.Contains(level, "BACKWARD") || strings.Contains(level, "FULL")

	result := map[string]any{
		"change_type": "add_field",
		"field_name":  fieldName,
		"field_type":  fieldType,
	}

	switch record.SchemaType {
	case storage.SchemaTypeAvro:
		if needsDefault {
			result["advice"] = "Add with default value for " + level + " compatibility"
			result["snippet"] = `{"name":"` + fieldName + `","type":["null","` + fieldType + `"],"default":null}`
		} else {
			result["advice"] = "Add without default since compatibility level is " + level
			result["snippet"] = `{"name":"` + fieldName + `","type":"` + fieldType + `"}`
		}
	case storage.SchemaTypeJSON:
		result["advice"] = "Add to properties; do not add to required array if backward-compatible"
		result["snippet"] = `"` + fieldName + `":{"type":"` + fieldType + `"}`
	case storage.SchemaTypeProtobuf:
		result["advice"] = "Add with a new unique field number"
		result["snippet"] = fieldType + " " + fieldName + " = <next_number>;"
	}

	return result
}

func suggestDeprecateField(record *storage.SchemaRecord, fieldName string) map[string]any {
	result := map[string]any{
		"change_type": "deprecate_field",
		"field_name":  fieldName,
	}

	switch record.SchemaType {
	case storage.SchemaTypeAvro:
		result["advice"] = "Add @deprecated to doc, add aliases for future rename, set default value"
		result["steps"] = []string{
			"1. Add \"doc\": \"@deprecated Use new_field instead\" to the field",
			"2. Add a default value if one doesn't exist",
			"3. In a future version, the field can be removed after all consumers migrate",
		}
	case storage.SchemaTypeJSON:
		result["advice"] = "Mark as deprecated in description, remove from required array"
		result["steps"] = []string{
			"1. Add \"deprecated\": true to the field's schema",
			"2. Remove the field from the \"required\" array if present",
			"3. Add \"description\": \"Deprecated: use new_field instead\"",
		}
	case storage.SchemaTypeProtobuf:
		result["advice"] = "Use the deprecated option on the field"
		result["steps"] = []string{
			"1. Add [deprecated = true] option to the field",
			"2. Add a comment explaining the migration path",
		}
	}

	return result
}

func suggestAddEnumSymbol(record *storage.SchemaRecord, symbol string) map[string]any {
	if symbol == "" {
		symbol = "NEW_SYMBOL"
	}
	result := map[string]any{
		"change_type": "add_enum_symbol",
		"symbol":      symbol,
	}

	switch record.SchemaType {
	case storage.SchemaTypeAvro:
		result["advice"] = "Add the new symbol to the end of the symbols array. This is backward-compatible."
		result["note"] = "Never remove or reorder existing symbols."
	case storage.SchemaTypeProtobuf:
		result["advice"] = "Add the new value to the enum with the next available number."
		result["note"] = "Never reuse field numbers from removed enum values."
	default:
		result["advice"] = "Add the new value to the enum array."
	}

	return result
}

// --- plan_migration_path ---

type planMigrationPathInput struct {
	Subject      string `json:"subject"`
	TargetSchema string `json:"target_schema"`
	SchemaType   string `json:"schema_type,omitempty"`
}

type migrationStep struct {
	Step        int    `json:"step"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Snippet     string `json:"snippet,omitempty"`
}

func (s *Server) handlePlanMigrationPath(ctx context.Context, _ *gomcp.CallToolRequest, input planMigrationPathInput) (*gomcp.CallToolResult, any, error) {
	record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}

	schemaType := record.SchemaType
	if input.SchemaType != "" {
		schemaType = storage.SchemaType(input.SchemaType)
	}

	level, _ := s.registry.GetConfig(ctx, registrycontext.DefaultContext, input.Subject)

	sourceFields := ExtractFields(record.Schema, schemaType)
	targetFields := ExtractFields(input.TargetSchema, schemaType)

	sourceMap := make(map[string]FieldInfo)
	for _, f := range sourceFields {
		sourceMap[f.Path] = f
	}
	targetMap := make(map[string]FieldInfo)
	for _, f := range targetFields {
		targetMap[f.Path] = f
	}

	var steps []migrationStep
	stepNum := 1

	// Step 1: Add new fields (with defaults for backward compat)
	for path, tField := range targetMap {
		if _, exists := sourceMap[path]; !exists {
			desc := "Add field '" + path + "' of type '" + tField.Type + "'"
			if strings.Contains(level, "BACKWARD") || strings.Contains(level, "FULL") {
				desc += " with a default value"
			}
			steps = append(steps, migrationStep{
				Step:        stepNum,
				Action:      "add_field",
				Description: desc,
			})
			stepNum++
		}
	}

	// Step 2: Modify changed types
	for path, sField := range sourceMap {
		if tField, exists := targetMap[path]; exists && sField.Type != tField.Type {
			steps = append(steps, migrationStep{
				Step:        stepNum,
				Action:      "change_type",
				Description: "Change type of '" + path + "' from '" + sField.Type + "' to '" + tField.Type + "'. Consider adding a new field instead if this is not a type promotion.",
			})
			stepNum++
		}
	}

	// Step 3: Remove deprecated fields
	for path := range sourceMap {
		if _, exists := targetMap[path]; !exists {
			steps = append(steps, migrationStep{
				Step:        stepNum,
				Action:      "remove_field",
				Description: "Remove field '" + path + "'. Deprecate it first if not already deprecated. Ensure no consumers depend on it.",
			})
			stepNum++
		}
	}

	if steps == nil {
		steps = []migrationStep{}
	}

	return jsonResult(map[string]any{
		"subject":             input.Subject,
		"current_version":     record.Version,
		"compatibility_level": level,
		"steps":               steps,
		"total_steps":         len(steps),
	})
}

// errMissingInput returns an error for missing required input.
func errMissingInput(field string) error {
	return &missingInputError{field: field}
}

type missingInputError struct {
	field string
}

func (e *missingInputError) Error() string {
	return "missing required input: " + e.field
}

// allSubjectFields collects field info per subject from the registry.
func (s *Server) allSubjectFields(ctx context.Context) (map[string][]FieldInfo, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]FieldInfo)
	for _, subj := range subjects {
		record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subj)
		if err != nil {
			continue
		}
		fields := ExtractFields(record.Schema, record.SchemaType)
		if len(fields) > 0 {
			result[subj] = fields
		}
	}
	return result, nil
}
