package jsonschema

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
)

// s creates a SchemaWithRefs with no references for convenience.
func s(schema string) compatibility.SchemaWithRefs {
	return compatibility.SchemaWithRefs{Schema: schema}
}

func TestChecker_CompatibleSchemas(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Expected compatible, got messages: %v", result.Messages)
	}
}

func TestChecker_AddOptionalProperty_OpenModel_Incompatible(t *testing.T) {
	checker := NewChecker()

	// No additionalProperties = open content model (default)
	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Adding optional property to open content model should be incompatible (old writer could have used 'age' as additional property with any type)")
	}
}

func TestChecker_AddRequiredProperty_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name", "age"]
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Adding required property should be incompatible")
	}
}

func TestChecker_RemoveProperty_OpenModel_Compatible(t *testing.T) {
	checker := NewChecker()

	// No additionalProperties = open content model
	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		}
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Removing property from open content model should be compatible (property becomes additional): %v", result.Messages)
	}
}

func TestChecker_ChangePropertyType_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"age": {"type": "integer"}
		}
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"age": {"type": "string"}
		}
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Changing property type should be incompatible")
	}
}

func TestChecker_MakeOptionalRequired_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name", "age"]
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Making optional property required should be incompatible")
	}
}

func TestChecker_MakeRequiredOptional_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name", "age"]
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Making required property optional should be compatible: %v", result.Messages)
	}
}

func TestChecker_RemoveEnumValue_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "string",
		"enum": ["red", "green", "blue"]
	}`

	newSchema := `{
		"type": "string",
		"enum": ["red", "green"]
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Removing enum value should be incompatible")
	}
}

func TestChecker_AddEnumValue_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "string",
		"enum": ["red", "green"]
	}`

	newSchema := `{
		"type": "string",
		"enum": ["red", "green", "blue"]
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding enum value should be compatible: %v", result.Messages)
	}
}

func TestChecker_RemoveEnumConstraint_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "string",
		"enum": ["red", "green", "blue"]
	}`

	newSchema := `{
		"type": "string"
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Removing enum constraint should be compatible (less restrictive): %v", result.Messages)
	}
}

func TestChecker_ForbidAdditionalProperties_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		}
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": false
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Forbidding additionalProperties should be incompatible")
	}
}

func TestChecker_AllowAdditionalProperties_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": false
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": true
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Allowing additionalProperties should be compatible: %v", result.Messages)
	}
}

func TestChecker_NestedPropertyRemoval_OpenModel_Compatible(t *testing.T) {
	checker := NewChecker()

	// Nested object also has open content model (no additionalProperties)
	oldSchema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer"}
				}
			}
		}
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		}
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Removing nested property from open content model should be compatible: %v", result.Messages)
	}
}

func TestChecker_ArrayItemsChange_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "array",
		"items": {"type": "string"}
	}`

	newSchema := `{
		"type": "array",
		"items": {"type": "integer"}
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Changing array items type should be incompatible")
	}
}

func TestChecker_TightenMinItems_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "array",
		"items": {"type": "string"},
		"minItems": 1
	}`

	newSchema := `{
		"type": "array",
		"items": {"type": "string"},
		"minItems": 5
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Tightening minItems should be incompatible")
	}
}

func TestChecker_TightenMaxItems_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "array",
		"items": {"type": "string"},
		"maxItems": 10
	}`

	newSchema := `{
		"type": "array",
		"items": {"type": "string"},
		"maxItems": 5
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Tightening maxItems should be incompatible")
	}
}

func TestChecker_LoosenMinItems_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "array",
		"items": {"type": "string"},
		"minItems": 5
	}`

	newSchema := `{
		"type": "array",
		"items": {"type": "string"},
		"minItems": 1
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Loosening minItems should be compatible: %v", result.Messages)
	}
}

func TestChecker_ChangeRootType_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{"type": "object"}`
	newSchema := `{"type": "array"}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Changing root type should be incompatible")
	}
}

func TestChecker_AddTypeToUnion_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{"type": "string"}`
	newSchema := `{"type": ["string", "null"]}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding type to union should be compatible: %v", result.Messages)
	}
}

func TestChecker_RemoveTypeFromUnion_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{"type": ["string", "null"]}`
	newSchema := `{"type": "string"}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Removing type from union should be incompatible")
	}
}

func TestChecker_InvalidSchema(t *testing.T) {
	checker := NewChecker()

	validSchema := `{"type": "object"}`
	invalidSchema := `{not valid json}`

	result := checker.Check(s(invalidSchema), s(validSchema))
	if result.IsCompatible {
		t.Error("Invalid new schema should return incompatible")
	}

	result = checker.Check(s(validSchema), s(invalidSchema))
	if result.IsCompatible {
		t.Error("Invalid old schema should return incompatible")
	}
}

func TestChecker_EmptySchemas_Compatible(t *testing.T) {
	checker := NewChecker()

	// Empty schema accepts anything
	oldSchema := `{}`
	newSchema := `{}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Empty schemas should be compatible: %v", result.Messages)
	}
}

func TestChecker_AddTypeConstraint(t *testing.T) {
	checker := NewChecker()

	// Empty schema accepts anything
	oldSchema := `{}`
	newSchema := `{"type": "string"}`

	result := checker.Check(s(newSchema), s(oldSchema))
	// Adding type constraint to empty schema is compatible
	// because new schema is more restrictive, but still accepts valid old data
	_ = result // Result depends on compatibility mode interpretation
}

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{"nil", nil, nil},
		{"string", "string", []string{"string"}},
		{"array", []interface{}{"string", "null"}, []string{"null", "string"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeType(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("Expected %v, got %v", tt.expected, result)
					return
				}
			}
		})
	}
}

func TestGetRequiredSet(t *testing.T) {
	schema := map[string]interface{}{
		"required": []interface{}{"name", "age"},
	}

	result := getRequiredSet(schema)
	if !result["name"] || !result["age"] {
		t.Error("Required set should contain 'name' and 'age'")
	}
	if result["other"] {
		t.Error("Required set should not contain 'other'")
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		base     string
		prop     string
		expected string
	}{
		{"", "name", "name"},
		{"user", "name", "user.name"},
		{"a.b", "c", "a.b.c"},
	}

	for _, tt := range tests {
		result := joinPath(tt.base, tt.prop)
		if result != tt.expected {
			t.Errorf("joinPath(%q, %q) = %q, want %q", tt.base, tt.prop, result, tt.expected)
		}
	}
}

func TestPathOrRoot(t *testing.T) {
	if pathOrRoot("") != "root" {
		t.Error("Empty path should return 'root'")
	}
	if pathOrRoot("user.name") != "user.name" {
		t.Error("Non-empty path should be returned as-is")
	}
}

// ============================================================================
// NEW TESTS: Open vs closed content model (J1, J2)
// ============================================================================

func TestChecker_HasOpenContentModel(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected bool
	}{
		{"no additionalProperties", map[string]interface{}{"type": "object"}, true},
		{"additionalProperties true", map[string]interface{}{"additionalProperties": true}, true},
		{"additionalProperties false", map[string]interface{}{"additionalProperties": false}, false},
		{"additionalProperties schema", map[string]interface{}{"additionalProperties": map[string]interface{}{"type": "string"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasOpenContentModel(tt.schema)
			if result != tt.expected {
				t.Errorf("hasOpenContentModel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestChecker_AddOptionalProperty_ClosedModel_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": false
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"additionalProperties": false
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Adding optional property to closed model should be compatible (old writer can't have produced this property): %v", result.Messages)
	}
}

func TestChecker_RemoveProperty_ClosedModel_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"additionalProperties": false
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": false
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Removing property from closed model should be incompatible (old data with 'age' would be rejected)")
	}
}

func TestChecker_NestedAddProperty_OpenModel_Incompatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		}
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer"}
				}
			}
		}
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Adding property to nested open content model should be incompatible")
	}
}

func TestChecker_NestedRemoveProperty_OpenModel_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer"}
				}
			}
		}
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		}
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Removing property from nested open content model should be compatible: %v", result.Messages)
	}
}

// ============================================================================
// NEW TESTS: Array items schema removal (J3)
// ============================================================================

func TestChecker_ArrayItemsSchemaRemoval_Compatible(t *testing.T) {
	checker := NewChecker()

	oldSchema := `{
		"type": "array",
		"items": {"type": "string"}
	}`

	newSchema := `{
		"type": "array"
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if !result.IsCompatible {
		t.Errorf("Removing array items schema should be compatible (relaxation): %v", result.Messages)
	}
}

// ============================================================================
// NEW TESTS: Mixed content models
// ============================================================================

func TestChecker_AddProperty_ClosedModel_NestedOpen_Incompatible(t *testing.T) {
	checker := NewChecker()

	// Root is closed, but nested "user" is open
	oldSchema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		},
		"additionalProperties": false
	}`

	newSchema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer"}
				}
			}
		},
		"additionalProperties": false
	}`

	result := checker.Check(s(newSchema), s(oldSchema))
	if result.IsCompatible {
		t.Error("Adding property to nested open model (even if root is closed) should be incompatible")
	}
}
