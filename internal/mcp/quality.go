package mcp

import (
	"strings"
	"unicode"
)

// CategoryScore tracks scoring for a quality category.
type CategoryScore struct {
	Score    int      `json:"score"`
	MaxScore int      `json:"max_score"`
	Details  []string `json:"details,omitempty"`
}

// QualityResult holds the overall schema quality assessment.
type QualityResult struct {
	OverallScore int                       `json:"overall_score"`
	MaxScore     int                       `json:"max_score"`
	Grade        string                    `json:"grade"`
	Categories   map[string]*CategoryScore `json:"categories"`
	QuickWins    []string                  `json:"quick_wins,omitempty"`
}

// ScoreSchemaQuality evaluates schema quality based on extracted fields and schema content.
func ScoreSchemaQuality(fields []FieldInfo, schemaStr string, schemaType string) *QualityResult {
	result := &QualityResult{
		Categories: make(map[string]*CategoryScore),
	}

	// Category 1: Naming (max 25)
	naming := &CategoryScore{MaxScore: 25}
	goodNames := 0
	for _, f := range fields {
		if isGoodFieldName(f.Name) {
			goodNames++
		} else {
			naming.Details = append(naming.Details, "Field '"+f.Name+"' does not follow snake_case convention")
			result.QuickWins = append(result.QuickWins, "Rename '"+f.Name+"' to '"+NormalizeFieldName(f.Name)+"'")
		}
	}
	if len(fields) > 0 {
		naming.Score = 25 * goodNames / len(fields)
	} else {
		naming.Score = 25
	}
	result.Categories["naming"] = naming

	// Category 2: Documentation (max 25)
	docs := &CategoryScore{MaxScore: 25}
	documented := 0
	for _, f := range fields {
		if f.Doc != "" {
			documented++
		}
	}
	if len(fields) > 0 {
		docs.Score = 25 * documented / len(fields)
		if documented == 0 {
			docs.Details = append(docs.Details, "No fields have documentation")
			result.QuickWins = append(result.QuickWins, "Add documentation/descriptions to fields")
		} else if documented < len(fields) {
			docs.Details = append(docs.Details, "Only some fields have documentation")
		}
	} else {
		docs.Score = 25
	}
	result.Categories["documentation"] = docs

	// Category 3: Type Safety (max 25)
	typeSafety := &CategoryScore{MaxScore: 25}
	safeTypes := 0
	for _, f := range fields {
		if isSpecificType(f.Type) {
			safeTypes++
		} else {
			typeSafety.Details = append(typeSafety.Details, "Field '"+f.Name+"' uses generic type '"+f.Type+"'")
		}
	}
	if len(fields) > 0 {
		typeSafety.Score = 25 * safeTypes / len(fields)
	} else {
		typeSafety.Score = 25
	}
	result.Categories["type_safety"] = typeSafety

	// Category 4: Evolution Readiness (max 25)
	evolution := &CategoryScore{MaxScore: 25}
	evScore := 0
	// Check if fields have defaults (good for evolution)
	withDefaults := 0
	for _, f := range fields {
		if f.HasDefault {
			withDefaults++
		}
	}
	if len(fields) > 0 && withDefaults > 0 {
		evScore += 10
		evolution.Details = append(evolution.Details, "Fields with defaults enable backward-compatible evolution")
	} else if len(fields) > 0 {
		evolution.Details = append(evolution.Details, "No fields have default values; consider adding defaults for evolution safety")
		result.QuickWins = append(result.QuickWins, "Add default values to optional fields")
	}
	// Check if schema has namespace/package (good practice)
	if strings.Contains(schemaStr, "namespace") || strings.Contains(schemaStr, "package") {
		evScore += 8
	} else {
		evolution.Details = append(evolution.Details, "No namespace/package declaration found")
		result.QuickWins = append(result.QuickWins, "Add a namespace to prevent naming conflicts")
	}
	// Check if schema has a doc field
	if strings.Contains(schemaStr, `"doc"`) || strings.Contains(schemaStr, `"description"`) {
		evScore += 7
	} else {
		evolution.Details = append(evolution.Details, "Schema-level documentation is missing")
	}
	evolution.Score = evScore
	result.Categories["evolution"] = evolution

	// Compute overall
	for _, cat := range result.Categories {
		result.OverallScore += cat.Score
		result.MaxScore += cat.MaxScore
	}

	// Grade
	pct := 0
	if result.MaxScore > 0 {
		pct = 100 * result.OverallScore / result.MaxScore
	}
	switch {
	case pct >= 90:
		result.Grade = "A"
	case pct >= 80:
		result.Grade = "B"
	case pct >= 70:
		result.Grade = "C"
	case pct >= 60:
		result.Grade = "D"
	default:
		result.Grade = "F"
	}

	return result
}

// isGoodFieldName checks if a field name follows snake_case convention.
func isGoodFieldName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if unicode.IsUpper(r) {
			return false
		}
		if r == '-' || r == ' ' {
			return false
		}
	}
	return true
}

// isSpecificType returns true for types that are more specific than "string" or "bytes".
func isSpecificType(t string) bool {
	generic := map[string]bool{
		"string": true,
		"bytes":  true,
		"any":    true,
		"object": true,
	}
	return !generic[strings.ToLower(t)]
}
