// Package analysis provides schema analysis utilities shared between the MCP
// server and the REST API. It includes field extraction for Avro, JSON Schema,
// and Protobuf, plus naming normalization helpers.
package analysis

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	avrolib "github.com/hamba/avro/v2"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// FieldInfo describes a single field extracted from a schema.
type FieldInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Required   bool   `json:"required"`
	HasDefault bool   `json:"has_default"`
	Doc        string `json:"doc,omitempty"`
}

// ExtractFields extracts field information from a schema string based on its type.
func ExtractFields(schemaStr string, schemaType storage.SchemaType) []FieldInfo {
	switch schemaType {
	case storage.SchemaTypeAvro:
		return extractAvroFields(schemaStr)
	case storage.SchemaTypeJSON:
		return extractJSONSchemaFields(schemaStr)
	case storage.SchemaTypeProtobuf:
		return extractProtobufFields(schemaStr)
	default:
		return nil
	}
}

func extractAvroFields(schemaStr string) []FieldInfo {
	schema, err := avrolib.Parse(schemaStr)
	if err != nil {
		return nil
	}
	var fields []FieldInfo
	walkAvroSchema(schema, "", &fields)
	return fields
}

func walkAvroSchema(s avrolib.Schema, prefix string, fields *[]FieldInfo) {
	switch v := s.(type) {
	case *avrolib.RecordSchema:
		for _, f := range v.Fields() {
			path := f.Name()
			if prefix != "" {
				path = prefix + "." + f.Name()
			}
			fieldType := avroSchemaTypeName(f.Type())
			hasDefault := f.HasDefault()
			doc := ""
			if f.Doc() != "" {
				doc = f.Doc()
			}
			required := !isAvroNullable(f.Type())
			*fields = append(*fields, FieldInfo{
				Name:       f.Name(),
				Path:       path,
				Type:       fieldType,
				Required:   required,
				HasDefault: hasDefault,
				Doc:        doc,
			})
			walkAvroFieldType(f.Type(), path, fields)
		}
	}
}

func walkAvroFieldType(s avrolib.Schema, prefix string, fields *[]FieldInfo) {
	switch v := s.(type) {
	case *avrolib.RecordSchema:
		walkAvroSchema(v, prefix, fields)
	case *avrolib.ArraySchema:
		walkAvroFieldType(v.Items(), prefix+"[]", fields)
	case *avrolib.MapSchema:
		walkAvroFieldType(v.Values(), prefix+"{}", fields)
	case *avrolib.UnionSchema:
		for _, t := range v.Types() {
			if _, ok := t.(*avrolib.NullSchema); !ok {
				walkAvroFieldType(t, prefix, fields)
			}
		}
	}
}

func avroSchemaTypeName(s avrolib.Schema) string {
	switch v := s.(type) {
	case *avrolib.PrimitiveSchema:
		return string(v.Type())
	case *avrolib.RecordSchema:
		return "record"
	case *avrolib.ArraySchema:
		return "array"
	case *avrolib.MapSchema:
		return "map"
	case *avrolib.EnumSchema:
		return "enum"
	case *avrolib.FixedSchema:
		return "fixed"
	case *avrolib.UnionSchema:
		types := make([]string, 0, len(v.Types()))
		for _, t := range v.Types() {
			types = append(types, avroSchemaTypeName(t))
		}
		return "union[" + strings.Join(types, ",") + "]"
	case *avrolib.RefSchema:
		return "ref"
	case *avrolib.NullSchema:
		return "null"
	default:
		return "unknown"
	}
}

func isAvroNullable(s avrolib.Schema) bool {
	u, ok := s.(*avrolib.UnionSchema)
	if !ok {
		return false
	}
	for _, t := range u.Types() {
		if _, ok := t.(*avrolib.NullSchema); ok {
			return true
		}
	}
	return false
}

func extractJSONSchemaFields(schemaStr string) []FieldInfo {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
		return nil
	}

	required := map[string]bool{}
	if reqArr, ok := schema["required"].([]interface{}); ok {
		for _, r := range reqArr {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}

	var fields []FieldInfo
	walkJSONSchemaProperties(schema, "", required, &fields)
	return fields
}

func walkJSONSchemaProperties(obj map[string]interface{}, prefix string, parentRequired map[string]bool, fields *[]FieldInfo) {
	props, ok := obj["properties"].(map[string]interface{})
	if !ok {
		return
	}

	for name, propRaw := range props {
		prop, ok := propRaw.(map[string]interface{})
		if !ok {
			continue
		}
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		typeName := "object"
		if t, ok := prop["type"].(string); ok {
			typeName = t
		}
		_, hasDefault := prop["default"]
		doc := ""
		if d, ok := prop["description"].(string); ok {
			doc = d
		}

		*fields = append(*fields, FieldInfo{
			Name:       name,
			Path:       path,
			Type:       typeName,
			Required:   parentRequired[name],
			HasDefault: hasDefault,
			Doc:        doc,
		})

		if typeName == "object" {
			childRequired := map[string]bool{}
			if reqArr, ok := prop["required"].([]interface{}); ok {
				for _, r := range reqArr {
					if s, ok := r.(string); ok {
						childRequired[s] = true
					}
				}
			}
			walkJSONSchemaProperties(prop, path, childRequired, fields)
		}
		if typeName == "array" {
			if items, ok := prop["items"].(map[string]interface{}); ok {
				if it, ok := items["type"].(string); ok && it == "object" {
					childRequired := map[string]bool{}
					if reqArr, ok := items["required"].([]interface{}); ok {
						for _, r := range reqArr {
							if s, ok := r.(string); ok {
								childRequired[s] = true
							}
						}
					}
					walkJSONSchemaProperties(items, path+"[]", childRequired, fields)
				}
			}
		}
	}
}

func extractProtobufFields(schemaStr string) []FieldInfo {
	var fields []FieldInfo
	fieldRe := regexp.MustCompile(`(?m)^\s*(?:(optional|required|repeated)\s+)?(\w+)\s+(\w+)\s*=\s*\d+\s*;`)
	matches := fieldRe.FindAllStringSubmatch(schemaStr, -1)
	for _, m := range matches {
		modifier := m[1]
		typeName := m[2]
		name := m[3]
		required := modifier == "required"
		if modifier == "" {
			required = false
		}
		fields = append(fields, FieldInfo{
			Name:     name,
			Path:     name,
			Type:     typeName,
			Required: required,
		})
	}
	return fields
}

// NormalizeFieldName converts a field name from any casing to snake_case.
func NormalizeFieldName(name string) string {
	var result []rune
	for i, r := range name {
		if r == '-' || r == '.' || r == ' ' {
			result = append(result, '_')
			continue
		}
		if unicode.IsUpper(r) && i > 0 {
			prev := rune(name[i-1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}
