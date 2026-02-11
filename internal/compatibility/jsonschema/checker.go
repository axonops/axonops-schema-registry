// Package jsonschema provides JSON Schema compatibility checking.
package jsonschema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
)

// Checker implements compatibility.SchemaChecker for JSON Schema.
type Checker struct{}

// NewChecker creates a new JSON Schema compatibility checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Check checks compatibility between reader (new) and writer (old) JSON schemas.
func (c *Checker) Check(reader, writer compatibility.SchemaWithRefs) *compatibility.Result {
	var newSchema, oldSchema map[string]interface{}

	if err := json.Unmarshal([]byte(reader.Schema), &newSchema); err != nil {
		return compatibility.NewIncompatibleResult("failed to parse new schema: " + err.Error())
	}

	if err := json.Unmarshal([]byte(writer.Schema), &oldSchema); err != nil {
		return compatibility.NewIncompatibleResult("failed to parse old schema: " + err.Error())
	}

	result := compatibility.NewCompatibleResult()
	c.checkCompatibility(newSchema, oldSchema, "", result)
	return result
}

// checkCompatibility recursively checks compatibility between two schemas.
func (c *Checker) checkCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	// Check type compatibility
	newType := getType(newSchema)
	oldType := getType(oldSchema)

	if !c.areTypesCompatible(newType, oldType) {
		result.AddMessage("Type changed at %s from '%v' to '%v'", pathOrRoot(path), oldType, newType)
	}

	// Check based on schema type
	switch newType {
	case "object":
		c.checkObjectCompatibility(newSchema, oldSchema, path, result)
	case "array":
		c.checkArrayCompatibility(newSchema, oldSchema, path, result)
	}

	// Check enum changes
	c.checkEnumCompatibility(newSchema, oldSchema, path, result)

	// Check additionalProperties changes
	c.checkAdditionalPropertiesCompatibility(newSchema, oldSchema, path, result)
}

// checkObjectCompatibility checks compatibility of object schemas.
func (c *Checker) checkObjectCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	newProps := getProperties(newSchema)
	oldProps := getProperties(oldSchema)
	newRequired := getRequiredSet(newSchema)
	oldRequired := getRequiredSet(oldSchema)

	// Check for removed properties
	for propName := range oldProps {
		propPath := joinPath(path, propName)
		if _, exists := newProps[propName]; !exists {
			result.AddMessage("Property '%s' was removed", propPath)
		}
	}

	// Check for new required properties (breaking change)
	for propName := range newProps {
		propPath := joinPath(path, propName)
		_, existedBefore := oldProps[propName]
		wasRequired := oldRequired[propName]
		isRequired := newRequired[propName]

		if !existedBefore && isRequired {
			// New required property added - breaking
			result.AddMessage("New required property '%s' was added", propPath)
		} else if existedBefore && !wasRequired && isRequired {
			// Existing optional property made required - breaking
			result.AddMessage("Property '%s' changed from optional to required", propPath)
		}
	}

	// Check existing properties for compatibility
	for propName, newProp := range newProps {
		if oldProp, exists := oldProps[propName]; exists {
			propPath := joinPath(path, propName)
			newPropMap, newOk := newProp.(map[string]interface{})
			oldPropMap, oldOk := oldProp.(map[string]interface{})
			if newOk && oldOk {
				c.checkCompatibility(newPropMap, oldPropMap, propPath, result)
			}
		}
	}
}

// checkArrayCompatibility checks compatibility of array schemas.
func (c *Checker) checkArrayCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	newItems := getItems(newSchema)
	oldItems := getItems(oldSchema)

	if newItems != nil && oldItems != nil {
		c.checkCompatibility(newItems, oldItems, joinPath(path, "items"), result)
	} else if oldItems != nil && newItems == nil {
		result.AddMessage("Array items schema removed at '%s'", pathOrRoot(path))
	}

	// Check minItems/maxItems constraints
	c.checkConstraintChange(newSchema, oldSchema, "minItems", path, result, true)
	c.checkConstraintChange(newSchema, oldSchema, "maxItems", path, result, false)
}

// checkEnumCompatibility checks enum value changes.
func (c *Checker) checkEnumCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	newEnum := getEnum(newSchema)
	oldEnum := getEnum(oldSchema)

	if oldEnum == nil {
		return // No enum in old schema
	}

	if newEnum == nil {
		// Enum constraint removed - compatible (less restrictive)
		return
	}

	// Check for removed enum values
	oldEnumSet := make(map[string]bool)
	for _, v := range oldEnum {
		oldEnumSet[fmt.Sprintf("%v", v)] = true
	}

	newEnumSet := make(map[string]bool)
	for _, v := range newEnum {
		newEnumSet[fmt.Sprintf("%v", v)] = true
	}

	for oldVal := range oldEnumSet {
		if !newEnumSet[oldVal] {
			result.AddMessage("Enum value '%s' was removed at '%s'", oldVal, pathOrRoot(path))
		}
	}
}

// checkAdditionalPropertiesCompatibility checks additionalProperties changes.
func (c *Checker) checkAdditionalPropertiesCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	newAP, newHasAP := newSchema["additionalProperties"]
	oldAP, oldHasAP := oldSchema["additionalProperties"]

	// If old schema allowed additional properties and new doesn't
	if (!oldHasAP || oldAP == true) && newHasAP && newAP == false {
		result.AddMessage("additionalProperties changed from allowed to forbidden at '%s'", pathOrRoot(path))
	}

	// If both have schema for additional properties, check compatibility
	if newAPSchema, newOk := newAP.(map[string]interface{}); newOk {
		if oldAPSchema, oldOk := oldAP.(map[string]interface{}); oldOk {
			c.checkCompatibility(newAPSchema, oldAPSchema, joinPath(path, "additionalProperties"), result)
		}
	}
}

// checkConstraintChange checks numeric constraint changes.
func (c *Checker) checkConstraintChange(newSchema, oldSchema map[string]interface{}, constraint, path string, result *compatibility.Result, isMinConstraint bool) {
	newVal, newHas := newSchema[constraint]
	oldVal, oldHas := oldSchema[constraint]

	if !newHas && !oldHas {
		return
	}

	newNum := toFloat64(newVal)
	oldNum := toFloat64(oldVal)

	if isMinConstraint {
		// Making minimum higher is breaking
		if newHas && (!oldHas || newNum > oldNum) {
			result.AddMessage("'%s' constraint tightened at '%s' (was %v, now %v)", constraint, pathOrRoot(path), oldVal, newVal)
		}
	} else {
		// Making maximum lower is breaking
		if newHas && (!oldHas || newNum < oldNum) {
			result.AddMessage("'%s' constraint tightened at '%s' (was %v, now %v)", constraint, pathOrRoot(path), oldVal, newVal)
		}
	}
}

// areTypesCompatible checks if two types are compatible.
func (c *Checker) areTypesCompatible(newType, oldType interface{}) bool {
	// Handle nil types (no type constraint)
	if oldType == nil {
		return true // Old schema had no type constraint
	}
	if newType == nil {
		return true // New schema has no type constraint (more permissive)
	}

	// Convert to comparable format
	newTypes := normalizeType(newType)
	oldTypes := normalizeType(oldType)

	// Check if new types are a subset of old types (more restrictive is breaking)
	// Actually, for backward compatibility, new types should include all old types
	for _, ot := range oldTypes {
		found := false
		for _, nt := range newTypes {
			if nt == ot {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// normalizeType converts a type to a slice of strings.
func normalizeType(t interface{}) []string {
	if t == nil {
		return nil
	}
	if s, ok := t.(string); ok {
		return []string{s}
	}
	if arr, ok := t.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		sort.Strings(result)
		return result
	}
	return nil
}

// Helper functions

func getType(schema map[string]interface{}) interface{} {
	return schema["type"]
}

func getProperties(schema map[string]interface{}) map[string]interface{} {
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		return props
	}
	return make(map[string]interface{})
}

func getRequiredSet(schema map[string]interface{}) map[string]bool {
	result := make(map[string]bool)
	if required, ok := schema["required"].([]interface{}); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				result[s] = true
			}
		}
	}
	return result
}

func getItems(schema map[string]interface{}) map[string]interface{} {
	if items, ok := schema["items"].(map[string]interface{}); ok {
		return items
	}
	return nil
}

func getEnum(schema map[string]interface{}) []interface{} {
	if enum, ok := schema["enum"].([]interface{}); ok {
		return enum
	}
	return nil
}

func joinPath(base, prop string) string {
	if base == "" {
		return prop
	}
	return base + "." + prop
}

func pathOrRoot(path string) string {
	if path == "" {
		return "root"
	}
	return path
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

// DeepEqual compares two schema values for equality.
func DeepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

// Ensure Checker implements compatibility.SchemaChecker
var _ compatibility.SchemaChecker = (*Checker)(nil)
