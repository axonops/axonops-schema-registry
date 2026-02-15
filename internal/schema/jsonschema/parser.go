// Package jsonschema provides JSON Schema parsing.
package jsonschema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Parser implements schema.Parser for JSON Schema.
type Parser struct {
	compiler *jsonschema.Compiler
}

// NewParser creates a new JSON Schema parser.
func NewParser() *Parser {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft7 // Use Draft-07 as primary
	return &Parser{
		compiler: c,
	}
}

// Type returns the schema type.
func (p *Parser) Type() storage.SchemaType {
	return storage.SchemaTypeJSON
}

// Parse parses and validates a JSON Schema.
func (p *Parser) Parse(schemaStr string, refs []storage.Reference) (schema.ParsedSchema, error) {
	// Parse the JSON to validate it's valid JSON
	var schemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(schemaStr), &schemaMap); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Create a new compiler for this parse to avoid resource conflicts
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft7

	// Add referenced schemas as resources so $ref can resolve them
	for _, ref := range refs {
		if ref.Schema != "" {
			if err := compiler.AddResource(ref.Name, strings.NewReader(ref.Schema)); err != nil {
				return nil, fmt.Errorf("failed to add reference %q: %w", ref.Name, err)
			}
		}
	}

	// Add the main schema
	schemaURL := "schema.json"
	if err := compiler.AddResource(schemaURL, strings.NewReader(schemaStr)); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	// Compile the schema to validate it
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSON schema: %w", err)
	}

	return &ParsedJSONSchema{
		raw:        schemaStr,
		schemaMap:  schemaMap,
		compiled:   compiled,
		references: refs,
	}, nil
}

// ParsedJSONSchema represents a parsed JSON Schema.
type ParsedJSONSchema struct {
	raw        string
	schemaMap  map[string]interface{}
	compiled   *jsonschema.Schema
	references []storage.Reference
}

// Type returns the schema type.
func (p *ParsedJSONSchema) Type() storage.SchemaType {
	return storage.SchemaTypeJSON
}

// CanonicalString returns the canonical form of the schema.
func (p *ParsedJSONSchema) CanonicalString() string {
	return canonicalize(p.schemaMap)
}

// Fingerprint returns a unique fingerprint for the schema.
func (p *ParsedJSONSchema) Fingerprint() string {
	canonical := p.CanonicalString()
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])
}

// RawSchema returns the underlying schema object.
func (p *ParsedJSONSchema) RawSchema() interface{} {
	return p.schemaMap
}

// Normalize returns a normalized copy of this schema with deterministic key ordering.
func (p *ParsedJSONSchema) Normalize() schema.ParsedSchema {
	return &ParsedJSONSchema{
		raw:        p.CanonicalString(),
		schemaMap:  p.schemaMap,
		compiled:   p.compiled,
		references: p.references,
	}
}

// HasTopLevelField reports whether the JSON Schema "properties" object
// contains a key with the given name.
func (p *ParsedJSONSchema) HasTopLevelField(field string) bool {
	props, ok := p.schemaMap["properties"].(map[string]interface{})
	if !ok {
		return false
	}
	_, exists := props[field]
	return exists
}

// FormattedString returns the schema in the requested format.
// JSON Schema does not support special format values; always returns canonical string.
func (p *ParsedJSONSchema) FormattedString(format string) string {
	return p.CanonicalString()
}

// Raw returns the original schema string.
func (p *ParsedJSONSchema) Raw() string {
	return p.raw
}

// Compiled returns the compiled schema for validation.
func (p *ParsedJSONSchema) Compiled() *jsonschema.Schema {
	return p.compiled
}

// SchemaMap returns the parsed schema as a map.
func (p *ParsedJSONSchema) SchemaMap() map[string]interface{} {
	return p.schemaMap
}

// canonicalize returns a canonical JSON representation.
// Keys are sorted alphabetically for consistent fingerprinting.
func canonicalize(v interface{}) string {
	result, _ := canonicalizeValue(v)
	return result
}

func canonicalizeValue(v interface{}) (string, error) {
	switch val := v.(type) {
	case nil:
		return "null", nil
	case bool:
		if val {
			return "true", nil
		}
		return "false", nil
	case float64:
		// JSON numbers are float64
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val)), nil
		}
		return fmt.Sprintf("%g", val), nil
	case string:
		// Escape and quote string
		b, _ := json.Marshal(val)
		return string(b), nil
	case []interface{}:
		var parts []string
		for _, item := range val {
			s, _ := canonicalizeValue(item)
			parts = append(parts, s)
		}
		return "[" + strings.Join(parts, ",") + "]", nil
	case map[string]interface{}:
		// Sort keys
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var parts []string
		for _, k := range keys {
			keyStr, _ := json.Marshal(k)
			valStr, _ := canonicalizeValue(val[k])
			parts = append(parts, string(keyStr)+":"+valStr)
		}
		return "{" + strings.Join(parts, ",") + "}", nil
	default:
		// Fallback to JSON encoding
		b, _ := json.Marshal(v)
		return string(b), nil
	}
}

// GetSchemaType extracts the type from a JSON Schema.
func GetSchemaType(schemaMap map[string]interface{}) string {
	if t, ok := schemaMap["type"].(string); ok {
		return t
	}
	if types, ok := schemaMap["type"].([]interface{}); ok && len(types) > 0 {
		if t, ok := types[0].(string); ok {
			return t
		}
	}
	return ""
}

// GetProperties extracts properties from a JSON Schema object type.
func GetProperties(schemaMap map[string]interface{}) map[string]interface{} {
	if props, ok := schemaMap["properties"].(map[string]interface{}); ok {
		return props
	}
	return nil
}

// GetRequired extracts required fields from a JSON Schema.
func GetRequired(schemaMap map[string]interface{}) []string {
	if required, ok := schemaMap["required"].([]interface{}); ok {
		result := make([]string, 0, len(required))
		for _, r := range required {
			if s, ok := r.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// GetItems extracts the items schema for array types.
func GetItems(schemaMap map[string]interface{}) map[string]interface{} {
	if items, ok := schemaMap["items"].(map[string]interface{}); ok {
		return items
	}
	return nil
}

// GetAdditionalProperties extracts additionalProperties from a schema.
func GetAdditionalProperties(schemaMap map[string]interface{}) (interface{}, bool) {
	val, ok := schemaMap["additionalProperties"]
	return val, ok
}

// GetEnum extracts enum values from a schema.
func GetEnum(schemaMap map[string]interface{}) []interface{} {
	if enum, ok := schemaMap["enum"].([]interface{}); ok {
		return enum
	}
	return nil
}

// Ensure Parser implements schema.Parser
var _ schema.Parser = (*Parser)(nil)
