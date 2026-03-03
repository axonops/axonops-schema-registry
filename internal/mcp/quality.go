package mcp

import "github.com/axonops/axonops-schema-registry/internal/analysis"

// CategoryScore is an alias for analysis.CategoryScore.
type CategoryScore = analysis.CategoryScore

// QualityResult is an alias for analysis.QualityResult.
type QualityResult = analysis.QualityResult

// ScoreSchemaQuality delegates to analysis.ScoreSchemaQuality.
func ScoreSchemaQuality(fields []FieldInfo, schemaStr string, schemaType string) *QualityResult {
	return analysis.ScoreSchemaQuality(fields, schemaStr, schemaType)
}
