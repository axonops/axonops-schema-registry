package mcp

import (
	"github.com/axonops/axonops-schema-registry/internal/analysis"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// FieldInfo is an alias for analysis.FieldInfo for backward compatibility
// within the mcp package and its tests.
type FieldInfo = analysis.FieldInfo

// ExtractFields delegates to analysis.ExtractFields.
func ExtractFields(schemaStr string, schemaType storage.SchemaType) []FieldInfo {
	return analysis.ExtractFields(schemaStr, schemaType)
}

// NormalizeFieldName delegates to analysis.NormalizeFieldName.
func NormalizeFieldName(name string) string {
	return analysis.NormalizeFieldName(name)
}
