// Package avro provides Avro schema parsing and handling.
package avro

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hamba/avro/v2"

	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Parser implements schema.Parser for Avro schemas.
type Parser struct{}

// NewParser creates a new Avro parser.
func NewParser() *Parser {
	return &Parser{}
}

// Type returns the schema type.
func (p *Parser) Type() storage.SchemaType {
	return storage.SchemaTypeAvro
}

// Parse parses an Avro schema string.
func (p *Parser) Parse(schemaStr string, references []storage.Reference) (schema.ParsedSchema, error) {
	var avroSchema avro.Schema
	var err error

	if len(references) > 0 {
		// Use a schema cache to register referenced named types first
		cache := &avro.SchemaCache{}
		for _, ref := range references {
			if ref.Schema != "" {
				if _, refErr := avro.ParseWithCache(ref.Schema, "", cache); refErr != nil {
					return nil, fmt.Errorf("invalid reference schema %q: %w", ref.Name, refErr)
				}
			}
		}
		avroSchema, err = avro.ParseWithCache(schemaStr, "", cache)
	} else {
		avroSchema, err = avro.Parse(schemaStr)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid Avro schema: %w", err)
	}

	// Generate canonical form
	canonical := canonicalize(schemaStr)

	// Generate fingerprint from canonical form
	hash := sha256.Sum256([]byte(canonical))
	fingerprint := hex.EncodeToString(hash[:])

	return &ParsedSchema{
		schemaType:  storage.SchemaTypeAvro,
		canonical:   canonical,
		fingerprint: fingerprint,
		rawSchema:   avroSchema,
	}, nil
}

// ParsedSchema implements schema.ParsedSchema for Avro.
type ParsedSchema struct {
	schemaType  storage.SchemaType
	canonical   string
	fingerprint string
	rawSchema   avro.Schema
}

// Type returns the schema type.
func (s *ParsedSchema) Type() storage.SchemaType {
	return s.schemaType
}

// CanonicalString returns the canonical form of the schema.
func (s *ParsedSchema) CanonicalString() string {
	return s.canonical
}

// Fingerprint returns the schema fingerprint.
func (s *ParsedSchema) Fingerprint() string {
	return s.fingerprint
}

// RawSchema returns the underlying Avro schema.
func (s *ParsedSchema) RawSchema() interface{} {
	return s.rawSchema
}

// Normalize returns a normalized copy of this schema using canonical form.
func (s *ParsedSchema) Normalize() schema.ParsedSchema {
	return &ParsedSchema{
		schemaType:  s.schemaType,
		canonical:   s.canonical,
		fingerprint: s.fingerprint,
		rawSchema:   s.rawSchema,
	}
}

// FormattedString returns the schema in the requested format.
// Supported formats: "resolved" (inlines all references), "default" (canonical).
func (s *ParsedSchema) FormattedString(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "resolved":
		if s.rawSchema != nil {
			return s.rawSchema.String()
		}
		return s.canonical
	default:
		return s.canonical
	}
}

// canonicalize converts an Avro schema to its canonical form.
// This follows the Avro specification for Parsing Canonical Form.
func canonicalize(schemaStr string) string {
	var obj interface{}
	if err := json.Unmarshal([]byte(schemaStr), &obj); err != nil {
		// If it's not valid JSON, return as-is (probably a primitive type name)
		return strings.TrimSpace(schemaStr)
	}

	return canonicalizeValue(obj)
}

func canonicalizeValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		// Primitive type or named type reference
		return fmt.Sprintf(`"%s"`, val)

	case []interface{}:
		// Union type
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = canonicalizeValue(item)
		}
		return "[" + strings.Join(parts, ",") + "]"

	case map[string]interface{}:
		// Complex type (record, enum, array, map, fixed)
		return canonicalizeObject(val)

	default:
		// Other JSON values (numbers, booleans)
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func canonicalizeObject(obj map[string]interface{}) string {
	schemaType, _ := obj["type"].(string)

	// Define field order based on schema type
	var fieldOrder []string
	switch schemaType {
	case "record", "error":
		fieldOrder = []string{"name", "type", "fields"}
	case "enum":
		fieldOrder = []string{"name", "type", "symbols"}
	case "array":
		fieldOrder = []string{"type", "items"}
	case "map":
		fieldOrder = []string{"type", "values"}
	case "fixed":
		fieldOrder = []string{"name", "type", "size"}
	default:
		// For other types, use alphabetical order
		fieldOrder = make([]string, 0, len(obj))
		for k := range obj {
			fieldOrder = append(fieldOrder, k)
		}
		sort.Strings(fieldOrder)
	}

	// Build canonical representation
	parts := make([]string, 0)
	for _, key := range fieldOrder {
		val, exists := obj[key]
		if !exists {
			continue
		}

		// Skip non-canonical fields
		if isNonCanonicalField(key) {
			continue
		}

		var valStr string
		switch key {
		case "fields":
			// Fields is an array of field objects
			if fields, ok := val.([]interface{}); ok {
				fieldParts := make([]string, len(fields))
				for i, f := range fields {
					if fobj, ok := f.(map[string]interface{}); ok {
						fieldParts[i] = canonicalizeField(fobj)
					}
				}
				valStr = "[" + strings.Join(fieldParts, ",") + "]"
			}
		case "symbols":
			// Symbols is an array of strings
			if symbols, ok := val.([]interface{}); ok {
				symParts := make([]string, len(symbols))
				for i, s := range symbols {
					symParts[i] = fmt.Sprintf(`"%v"`, s)
				}
				valStr = "[" + strings.Join(symParts, ",") + "]"
			}
		default:
			valStr = canonicalizeValue(val)
		}

		if valStr != "" {
			parts = append(parts, fmt.Sprintf(`"%s":%s`, key, valStr))
		}
	}

	return "{" + strings.Join(parts, ",") + "}"
}

func canonicalizeField(field map[string]interface{}) string {
	parts := make([]string, 0)

	// Field order: name, type
	if name, ok := field["name"]; ok {
		parts = append(parts, fmt.Sprintf(`"name":"%v"`, name))
	}
	if typ, ok := field["type"]; ok {
		parts = append(parts, fmt.Sprintf(`"type":%s`, canonicalizeValue(typ)))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

func isNonCanonicalField(field string) bool {
	// Fields that should be excluded from canonical form
	nonCanonical := map[string]bool{
		"doc":       true,
		"aliases":   true,
		"default":   true,
		"order":     true,
		"namespace": false, // namespace IS included for named types
	}
	return nonCanonical[field]
}
