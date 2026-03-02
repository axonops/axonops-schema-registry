package mcp

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerComparisonTools() {
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "check_compatibility_multi",
		Description: "Check schema compatibility against multiple subjects at once, returning per-subject results.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "check_compatibility_multi", s.handleCheckCompatibilityMulti))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "diff_schemas",
		Description: "Diff two schema versions within a subject, showing added, removed, and modified fields.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "diff_schemas", s.handleDiffSchemas))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "compare_subjects",
		Description: "Compare the latest schemas of two different subjects, showing structural differences.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "compare_subjects", s.handleCompareSubjects))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "suggest_compatible_change",
		Description: "Get rule-based advice on how to make a compatible change to a schema given its compatibility level.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "suggest_compatible_change", s.handleSuggestCompatibleChange))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "match_subjects",
		Description: "Find subjects matching a regex, glob, or substring pattern.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "match_subjects", s.handleMatchSubjects))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "explain_compatibility_failure",
		Description: "Run a compatibility check and provide detailed, human-readable explanations of any failures.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "explain_compatibility_failure", s.handleExplainCompatibilityFailure))
}

// --- check_compatibility_multi ---

type checkCompatibilityMultiInput struct {
	Subjects   []string            `json:"subjects"`
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schema_type,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
}

type compatMultiResult struct {
	Subject      string   `json:"subject"`
	IsCompatible bool     `json:"is_compatible"`
	Messages     []string `json:"messages,omitempty"`
	Error        string   `json:"error,omitempty"`
}

func (s *Server) handleCheckCompatibilityMulti(ctx context.Context, _ *gomcp.CallToolRequest, input checkCompatibilityMultiInput) (*gomcp.CallToolResult, any, error) {
	schemaType := storage.SchemaType(input.SchemaType)
	results := make([]compatMultiResult, 0, len(input.Subjects))

	for _, subj := range input.Subjects {
		result, err := s.registry.CheckCompatibility(ctx, registrycontext.DefaultContext, subj, input.Schema, schemaType, input.References, "latest")
		if err != nil {
			results = append(results, compatMultiResult{
				Subject:      subj,
				IsCompatible: false,
				Error:        err.Error(),
			})
			continue
		}
		results = append(results, compatMultiResult{
			Subject:      subj,
			IsCompatible: result.IsCompatible,
			Messages:     result.Messages,
		})
	}

	allCompatible := true
	for _, r := range results {
		if !r.IsCompatible {
			allCompatible = false
			break
		}
	}

	return jsonResult(map[string]any{
		"all_compatible": allCompatible,
		"results":        results,
	})
}

// --- diff_schemas ---

type diffSchemasInput struct {
	Subject     string `json:"subject"`
	VersionFrom int    `json:"version_from"`
	VersionTo   int    `json:"version_to"`
}

type fieldDiff struct {
	Field    string `json:"field"`
	Change   string `json:"change"` // "added", "removed", "modified"
	OldType  string `json:"old_type,omitempty"`
	NewType  string `json:"new_type,omitempty"`
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

func (s *Server) handleDiffSchemas(ctx context.Context, _ *gomcp.CallToolRequest, input diffSchemasInput) (*gomcp.CallToolResult, any, error) {
	recordFrom, err := s.registry.GetSchemaBySubjectVersion(ctx, registrycontext.DefaultContext, input.Subject, input.VersionFrom)
	if err != nil {
		return errorResult(fmt.Errorf("version %d: %w", input.VersionFrom, err)), nil, nil
	}
	recordTo, err := s.registry.GetSchemaBySubjectVersion(ctx, registrycontext.DefaultContext, input.Subject, input.VersionTo)
	if err != nil {
		return errorResult(fmt.Errorf("version %d: %w", input.VersionTo, err)), nil, nil
	}

	fieldsFrom := ExtractFields(recordFrom.Schema, recordFrom.SchemaType)
	fieldsTo := ExtractFields(recordTo.Schema, recordTo.SchemaType)

	diffs := computeFieldDiffs(fieldsFrom, fieldsTo)

	return jsonResult(map[string]any{
		"subject":      input.Subject,
		"version_from": input.VersionFrom,
		"version_to":   input.VersionTo,
		"diffs":        diffs,
		"total":        len(diffs),
	})
}

func computeFieldDiffs(from, to []FieldInfo) []fieldDiff {
	fromMap := make(map[string]FieldInfo)
	for _, f := range from {
		fromMap[f.Path] = f
	}
	toMap := make(map[string]FieldInfo)
	for _, f := range to {
		toMap[f.Path] = f
	}

	var diffs []fieldDiff

	// Check removed and modified
	for path, fOld := range fromMap {
		fNew, exists := toMap[path]
		if !exists {
			diffs = append(diffs, fieldDiff{
				Field:   path,
				Change:  "removed",
				OldType: fOld.Type,
			})
		} else if fOld.Type != fNew.Type {
			diffs = append(diffs, fieldDiff{
				Field:   path,
				Change:  "modified",
				OldType: fOld.Type,
				NewType: fNew.Type,
			})
		}
	}

	// Check added
	for path, fNew := range toMap {
		if _, exists := fromMap[path]; !exists {
			diffs = append(diffs, fieldDiff{
				Field:   path,
				Change:  "added",
				NewType: fNew.Type,
			})
		}
	}

	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Field < diffs[j].Field })
	return diffs
}

// --- compare_subjects ---

type compareSubjectsInput struct {
	SubjectA string `json:"subject_a"`
	SubjectB string `json:"subject_b"`
}

func (s *Server) handleCompareSubjects(ctx context.Context, _ *gomcp.CallToolRequest, input compareSubjectsInput) (*gomcp.CallToolResult, any, error) {
	recordA, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.SubjectA)
	if err != nil {
		return errorResult(fmt.Errorf("subject %q: %w", input.SubjectA, err)), nil, nil
	}
	recordB, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.SubjectB)
	if err != nil {
		return errorResult(fmt.Errorf("subject %q: %w", input.SubjectB, err)), nil, nil
	}

	fieldsA := ExtractFields(recordA.Schema, recordA.SchemaType)
	fieldsB := ExtractFields(recordB.Schema, recordB.SchemaType)

	diffs := computeFieldDiffs(fieldsA, fieldsB)

	// Compute common fields
	setA := make(map[string]bool)
	for _, f := range fieldsA {
		setA[f.Path] = true
	}
	setB := make(map[string]bool)
	for _, f := range fieldsB {
		setB[f.Path] = true
	}
	var common []string
	for p := range setA {
		if setB[p] {
			common = append(common, p)
		}
	}
	sort.Strings(common)

	return jsonResult(map[string]any{
		"subject_a":     input.SubjectA,
		"subject_b":     input.SubjectB,
		"type_a":        string(recordA.SchemaType),
		"type_b":        string(recordB.SchemaType),
		"fields_a":      len(fieldsA),
		"fields_b":      len(fieldsB),
		"common_fields": common,
		"diffs":         diffs,
	})
}

// --- suggest_compatible_change ---

type suggestCompatibleChangeInput struct {
	Subject    string `json:"subject"`
	ChangeType string `json:"change_type"` // "add_field", "remove_field", "rename_field", "change_type"
}

func (s *Server) handleSuggestCompatibleChange(ctx context.Context, _ *gomcp.CallToolRequest, input suggestCompatibleChangeInput) (*gomcp.CallToolResult, any, error) {
	level, err := s.registry.GetConfig(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		level = "BACKWARD"
	}

	advice := compatibilityAdvice(level, input.ChangeType)

	return jsonResult(map[string]any{
		"subject":             input.Subject,
		"compatibility_level": level,
		"change_type":         input.ChangeType,
		"advice":              advice,
	})
}

func compatibilityAdvice(level, changeType string) []string {
	var advice []string

	switch changeType {
	case "add_field":
		switch {
		case strings.Contains(level, "BACKWARD"):
			advice = append(advice, "New fields MUST have a default value for backward compatibility.")
			advice = append(advice, "Consumers using the old schema will ignore the new field.")
		case strings.Contains(level, "FORWARD"):
			advice = append(advice, "New fields can be added freely in forward-compatible mode.")
			advice = append(advice, "However, old producers won't populate the new field.")
		case strings.Contains(level, "FULL"):
			advice = append(advice, "New fields MUST have a default value for full compatibility.")
			advice = append(advice, "Both old and new consumers/producers must handle the field's presence or absence.")
		default:
			advice = append(advice, "With NONE compatibility, any change is allowed.")
		}

	case "remove_field":
		switch {
		case strings.Contains(level, "BACKWARD"):
			advice = append(advice, "Removing fields is allowed in backward-compatible mode if the field had a default value.")
			advice = append(advice, "Consumers using the new schema must not depend on the removed field.")
		case strings.Contains(level, "FORWARD"):
			advice = append(advice, "Removing fields is NOT forward-compatible. Old consumers still expect the field.")
			advice = append(advice, "Consider deprecating the field first by adding documentation.")
		case strings.Contains(level, "FULL"):
			advice = append(advice, "Removing fields is only safe if the field had a default value.")
			advice = append(advice, "Ensure no consumers depend on the removed field.")
		default:
			advice = append(advice, "With NONE compatibility, any change is allowed.")
		}

	case "rename_field":
		advice = append(advice, "Field renames are NOT directly compatible in any mode.")
		advice = append(advice, "Instead, add a new field with the desired name and a default value,")
		advice = append(advice, "then deprecate the old field. In Avro, use aliases for backward compatibility.")

	case "change_type":
		advice = append(advice, "Type changes are generally incompatible.")
		advice = append(advice, "Some promotions are allowed (e.g., int→long, float→double in Avro).")
		advice = append(advice, "For incompatible type changes, create a new field and deprecate the old one.")

	default:
		advice = append(advice, "Supported change types: add_field, remove_field, rename_field, change_type")
	}

	return advice
}

// --- match_subjects ---

type matchSubjectsInput struct {
	Pattern string `json:"pattern"`
	Regex   bool   `json:"regex,omitempty"`
}

func (s *Server) handleMatchSubjects(ctx context.Context, _ *gomcp.CallToolRequest, input matchSubjectsInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	var re *regexp.Regexp
	if input.Regex {
		re, err = regexp.Compile(input.Pattern)
		if err != nil {
			return errorResult(err), nil, nil
		}
	}

	var matches []string
	for _, subj := range subjects {
		matched := false
		if input.Regex && re != nil {
			matched = re.MatchString(subj)
		} else {
			// Substring match
			matched = strings.Contains(subj, input.Pattern)
		}
		if matched {
			matches = append(matches, subj)
		}
	}
	if matches == nil {
		matches = []string{}
	}

	return jsonResult(map[string]any{
		"matches": matches,
		"count":   len(matches),
	})
}

// --- explain_compatibility_failure ---

type explainCompatibilityFailureInput struct {
	Subject    string              `json:"subject"`
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schema_type,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
	Version    string              `json:"version,omitempty"`
}

type compatExplanation struct {
	Message     string `json:"message"`
	Explanation string `json:"explanation"`
	Suggestion  string `json:"suggestion"`
}

func (s *Server) handleExplainCompatibilityFailure(ctx context.Context, _ *gomcp.CallToolRequest, input explainCompatibilityFailureInput) (*gomcp.CallToolResult, any, error) {
	version := input.Version
	if version == "" {
		version = "latest"
	}
	schemaType := storage.SchemaType(input.SchemaType)
	result, err := s.registry.CheckCompatibility(ctx, registrycontext.DefaultContext, input.Subject, input.Schema, schemaType, input.References, version)
	if err != nil {
		return errorResult(err), nil, nil
	}

	if result.IsCompatible {
		return jsonResult(map[string]any{
			"is_compatible": true,
			"message":       "Schema is fully compatible.",
		})
	}

	level, _ := s.registry.GetConfig(ctx, registrycontext.DefaultContext, input.Subject)

	explanations := make([]compatExplanation, 0, len(result.Messages))
	for _, msg := range result.Messages {
		explanations = append(explanations, explainMessage(msg, level))
	}

	return jsonResult(map[string]any{
		"is_compatible":       false,
		"compatibility_level": level,
		"explanations":        explanations,
	})
}

func explainMessage(msg, level string) compatExplanation {
	e := compatExplanation{Message: msg}

	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "missing") || strings.Contains(lower, "removed"):
		e.Explanation = "A field was removed that consumers may still depend on."
		e.Suggestion = "Add a default value to the field before removing it, or keep the field and mark it as deprecated."
	case strings.Contains(lower, "type") && (strings.Contains(lower, "change") || strings.Contains(lower, "mismatch")):
		e.Explanation = "A field's type was changed in a way that is not promotable."
		e.Suggestion = "Use type promotion (e.g., int→long) or add a new field with the desired type."
	case strings.Contains(lower, "default"):
		e.Explanation = "A new field was added without a default value."
		e.Suggestion = "Add a default value to the new field so old data can be read with the new schema."
	case strings.Contains(lower, "enum") || strings.Contains(lower, "symbol"):
		e.Explanation = "Enum symbols were changed in an incompatible way."
		e.Suggestion = "Only add new enum symbols; do not remove or rename existing ones."
	default:
		e.Explanation = "The schema change violates the " + level + " compatibility contract."
		e.Suggestion = "Review the compatibility rules for " + level + " mode and adjust the schema accordingly."
	}

	return e
}
