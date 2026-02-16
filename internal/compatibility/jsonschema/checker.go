// Package jsonschema provides JSON Schema compatibility checking.
package jsonschema

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	"github.com/axonops/axonops-schema-registry/internal/storage"
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

	// Build external reference maps from resolved references
	newExtRefs := buildExternalRefMap(reader.References)
	oldExtRefs := buildExternalRefMap(writer.References)

	// Resolve $ref references within each schema (local + external)
	resolveAllRefs(newSchema, newExtRefs)
	resolveAllRefs(oldSchema, oldExtRefs)

	result := compatibility.NewCompatibleResult()
	c.checkCompatibility(newSchema, oldSchema, "", result)
	return result
}

// checkCompatibility recursively checks compatibility between two schemas.
func (c *Checker) checkCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	// Handle composition keywords (oneOf, anyOf, allOf)
	newHasComp := hasCompositionKeyword(newSchema)
	oldHasComp := hasCompositionKeyword(oldSchema)

	if newHasComp || oldHasComp {
		c.checkCompositionCompatibility(newSchema, oldSchema, path, result)
		// If both schemas are purely compositional (no "type" and no object/array keywords),
		// return to avoid false type-change errors
		if getType(newSchema) == nil && getType(oldSchema) == nil &&
			!hasObjectKeywords(newSchema) && !hasObjectKeywords(oldSchema) {
			return
		}
	}

	// Check type compatibility (with number/integer promotion)
	newType := getType(newSchema)
	oldType := getType(oldSchema)

	if !c.areTypesCompatible(newType, oldType) {
		result.AddMessage("Type changed at %s from '%v' to '%v'", pathOrRoot(path), oldType, newType)
	}

	// Check based on schema type — detect implicit types via keywords
	newTypeStr := typeString(newType)
	oldTypeStr := typeString(oldType)

	isObject := newTypeStr == "object" || oldTypeStr == "object" ||
		hasObjectKeywords(newSchema) || hasObjectKeywords(oldSchema)
	isArray := newTypeStr == "array" || oldTypeStr == "array" ||
		hasArrayKeywords(newSchema) || hasArrayKeywords(oldSchema)

	if isObject {
		c.checkObjectCompatibility(newSchema, oldSchema, path, result)
	}
	if isArray {
		c.checkArrayCompatibility(newSchema, oldSchema, path, result)
	}

	// Check enum changes
	c.checkEnumCompatibility(newSchema, oldSchema, path, result)

	// Check const changes
	c.checkConstCompatibility(newSchema, oldSchema, path, result)

	// Check additionalProperties changes
	c.checkAdditionalPropertiesCompatibility(newSchema, oldSchema, path, result)

	// Check string constraints (minLength, maxLength, pattern)
	c.checkStringConstraints(newSchema, oldSchema, path, result)

	// Check numeric constraints (minimum, maximum, exclusiveMinimum, exclusiveMaximum, multipleOf)
	c.checkNumericConstraints(newSchema, oldSchema, path, result)

	// Check property count constraints (maxProperties, minProperties)
	c.checkPropertyCountConstraints(newSchema, oldSchema, path, result)

	// Check not schema
	c.checkNotSchema(newSchema, oldSchema, path, result)

	// Check dependencies (Draft-07)
	c.checkDependencies(newSchema, oldSchema, path, result)

	// Check dependentRequired/dependentSchemas (Draft-2020)
	c.checkDependentRequired(newSchema, oldSchema, path, result)
	c.checkDependentSchemas(newSchema, oldSchema, path, result)

	// Check uniqueItems
	c.checkUniqueItems(newSchema, oldSchema, path, result)

	// Check additionalItems
	c.checkAdditionalItems(newSchema, oldSchema, path, result)

	// Check items as boolean (Draft-2020: items: true/false)
	c.checkItemsBoolean(newSchema, oldSchema, path, result)
}

// ==========================================================================
// $REF RESOLUTION
// ==========================================================================

// buildExternalRefMap builds a map of reference name → parsed schema from resolved
// external references. This allows $ref resolution for cross-subject references.
func buildExternalRefMap(refs []storage.Reference) map[string]map[string]interface{} {
	if len(refs) == 0 {
		return nil
	}
	result := make(map[string]map[string]interface{}, len(refs))
	for _, ref := range refs {
		if ref.Schema == "" {
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(ref.Schema), &parsed); err == nil {
			result[ref.Name] = parsed
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// resolveAllRefs resolves all $ref references within a schema using both
// local definitions and external references from other subjects.
func resolveAllRefs(schema map[string]interface{}, extRefs map[string]map[string]interface{}) {
	defs := getDefinitions(schema)
	resolveRefsInMap(schema, defs, extRefs)
}

// getDefinitions returns the definitions map from a schema.
func getDefinitions(schema map[string]interface{}) map[string]interface{} {
	if defs, ok := schema["definitions"].(map[string]interface{}); ok {
		return defs
	}
	if defs, ok := schema["$defs"].(map[string]interface{}); ok {
		return defs
	}
	return nil
}

// resolveRefsInMap recursively replaces $ref with the referenced definition content.
// Resolves both local ($ref: "#/definitions/...") and external ($ref: "RefName") references.
func resolveRefsInMap(schema map[string]interface{}, defs map[string]interface{}, extRefs map[string]map[string]interface{}) {
	for key, val := range schema {
		if key == "definitions" || key == "$defs" {
			continue
		}
		switch v := val.(type) {
		case map[string]interface{}:
			if ref, ok := v["$ref"].(string); ok {
				if resolved := resolveRef(ref, defs, extRefs); resolved != nil {
					schema[key] = resolved
				}
			} else {
				resolveRefsInMap(v, defs, extRefs)
			}
		case []interface{}:
			for i, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					if ref, ok := m["$ref"].(string); ok {
						if resolved := resolveRef(ref, defs, extRefs); resolved != nil {
							v[i] = resolved
						}
					} else {
						resolveRefsInMap(m, defs, extRefs)
					}
				}
			}
		}
	}
}

// resolveRef resolves a $ref string, trying local definitions first, then external references.
func resolveRef(ref string, defs map[string]interface{}, extRefs map[string]map[string]interface{}) map[string]interface{} {
	// Try local $ref first (e.g., "#/definitions/someRef")
	if resolved := resolveLocalRef(ref, defs); resolved != nil {
		return resolved
	}
	// Try external references (e.g., "Address" or "com.example.Address")
	if extRefs != nil {
		if resolved, ok := extRefs[ref]; ok {
			// Return a copy to avoid mutation
			result := make(map[string]interface{}, len(resolved))
			for k, v := range resolved {
				result[k] = v
			}
			return result
		}
	}
	return nil
}

// resolveLocalRef resolves a local $ref string to its definition.
func resolveLocalRef(ref string, defs map[string]interface{}) map[string]interface{} {
	if defs == nil {
		return nil
	}
	// Handle "#/definitions/name" and "#/$defs/name" patterns
	for _, prefix := range []string{"#/definitions/", "#/$defs/"} {
		if strings.HasPrefix(ref, prefix) {
			name := ref[len(prefix):]
			if def, ok := defs[name]; ok {
				if defMap, ok := def.(map[string]interface{}); ok {
					// Return a copy to avoid mutation
					result := make(map[string]interface{}, len(defMap))
					for k, v := range defMap {
						result[k] = v
					}
					return result
				}
			}
		}
	}
	return nil
}

// ==========================================================================
// IMPLICIT TYPE DETECTION
// ==========================================================================

// hasObjectKeywords returns true if the schema has keywords that imply object type.
func hasObjectKeywords(schema map[string]interface{}) bool {
	for _, key := range []string{"properties", "required", "patternProperties", "additionalProperties"} {
		if _, ok := schema[key]; ok {
			return true
		}
	}
	return false
}

// hasArrayKeywords returns true if the schema has keywords that imply array type.
func hasArrayKeywords(schema map[string]interface{}) bool {
	if _, ok := schema["prefixItems"]; ok {
		return true
	}
	if _, ok := schema["additionalItems"]; ok {
		return true
	}
	// "items" as array (tuple) or schema object or boolean implies array validation
	if _, ok := schema["items"]; ok {
		return true
	}
	if _, ok := schema["minItems"]; ok {
		return true
	}
	if _, ok := schema["maxItems"]; ok {
		return true
	}
	if _, ok := schema["uniqueItems"]; ok {
		return true
	}
	return false
}

// ==========================================================================
// OBJECT COMPATIBILITY
// ==========================================================================

// checkObjectCompatibility checks compatibility of object schemas.
func (c *Checker) checkObjectCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	newProps := getProperties(newSchema)
	oldProps := getProperties(oldSchema)
	newRequired := getRequiredSet(newSchema)
	oldRequired := getRequiredSet(oldSchema)

	// Determine content model type for the reader (new schema)
	readerOpen := hasOpenContentModel(newSchema)
	readerAPSchema := getAdditionalPropertiesSchema(newSchema)

	// Check for removed properties
	for propName := range oldProps {
		propPath := joinPath(path, propName)
		if _, exists := newProps[propName]; !exists {
			// Skip if old property schema was false (already forbidden)
			if oldPropVal, ok := oldProps[propName].(bool); ok && !oldPropVal {
				continue
			}
			if !readerOpen {
				// Closed model: check if removed property is covered by patternProperties or additionalProperties schema
				if hasCoveringPatternProperties(newSchema) {
					continue // patternProperties may cover the removed property
				}
				if readerAPSchema != nil {
					oldPropMap, oldOk := oldProps[propName].(map[string]interface{})
					if oldOk {
						localResult := compatibility.NewCompatibleResult()
						c.checkCompatibility(readerAPSchema, oldPropMap, propPath, localResult)
						if !localResult.IsCompatible {
							result.AddMessage("Property '%s' removed but not covered by additionalProperties", propPath)
						}
					}
				} else {
					result.AddMessage("Property '%s' was removed", propPath)
				}
			}
		}
	}

	// Check for new properties
	for propName := range newProps {
		propPath := joinPath(path, propName)
		_, existedBefore := oldProps[propName]
		isRequired := newRequired[propName]

		if !existedBefore {
			// Skip if new property schema is boolean true (accepts anything — no new constraint)
			if newPropVal, ok := newProps[propName].(bool); ok && newPropVal {
				continue
			}
			if isRequired {
				// New required property added — always incompatible for backward compat
				result.AddMessage("New required property '%s' was added", propPath)
			} else if hasOpenContentModel(oldSchema) {
				// Open content model: old writer could have used this property name
				// with any type, conflicting with the new typed constraint
				result.AddMessage("Property '%s' was added to open content model", propPath)
			} else if getAdditionalPropertiesSchema(oldSchema) != nil {
				// Partially open: check if new property type matches the AP schema
				newPropMap, newOk := newProps[propName].(map[string]interface{})
				apSchema := getAdditionalPropertiesSchema(oldSchema)
				if newOk && apSchema != nil {
					localResult := compatibility.NewCompatibleResult()
					c.checkCompatibility(newPropMap, apSchema, propPath, localResult)
					if !localResult.IsCompatible {
						result.AddMessage("Property '%s' added with type incompatible with additionalProperties", propPath)
					}
				}
			}
			// Closed model (additionalProperties:false) + non-required → compatible
			// (old writer couldn't produce this property)
		} else if !oldRequired[propName] && isRequired {
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

	// Check if required array was added (old had none, new has some)
	if len(oldRequired) == 0 && len(newRequired) > 0 {
		for propName := range newRequired {
			if _, existed := oldProps[propName]; existed && !oldRequired[propName] {
				// Already handled above in the "changed from optional to required" check
				continue
			}
		}
	}
}

// getAdditionalPropertiesSchema returns the additionalProperties value as a schema map,
// or nil if not present or if it's a boolean.
func getAdditionalPropertiesSchema(schema map[string]interface{}) map[string]interface{} {
	ap, ok := schema["additionalProperties"]
	if !ok {
		return nil
	}
	if apSchema, ok := ap.(map[string]interface{}); ok {
		return apSchema
	}
	return nil
}

// ==========================================================================
// ARRAY COMPATIBILITY
// ==========================================================================

// checkArrayCompatibility checks compatibility of array schemas.
func (c *Checker) checkArrayCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	// Handle single-schema items (not tuple)
	newItems := getItems(newSchema)
	oldItems := getItems(oldSchema)

	if newItems != nil && oldItems != nil {
		c.checkCompatibility(newItems, oldItems, joinPath(path, "items"), result)
	} else if newItems != nil && oldItems == nil {
		// Only flag if old schema truly had no items constraint (not items:false)
		_, oldHasItems := oldSchema["items"]
		if !oldHasItems {
			// Adding items constraint to unconstrained array — more restrictive
			result.AddMessage("items schema added at '%s'", pathOrRoot(path))
		}
	}

	// Handle tuple-style items (items as array in Draft-07, prefixItems in Draft-2020)
	c.checkTupleItems(newSchema, oldSchema, path, result)

	// Check minItems/maxItems constraints
	c.checkConstraintChange(newSchema, oldSchema, "minItems", path, result, true)
	c.checkConstraintChange(newSchema, oldSchema, "maxItems", path, result, false)
}

// checkTupleItems checks tuple-style array items compatibility.
// Draft-07 uses "items" as array, Draft-2020 uses "prefixItems".
func (c *Checker) checkTupleItems(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldTuple := getTupleItems(oldSchema)
	newTuple := getTupleItems(newSchema)

	if len(oldTuple) == 0 && len(newTuple) == 0 {
		return
	}

	// Get the "additional items" schema for content model checks
	oldAISchema := getAdditionalItemsSchema(oldSchema)
	newAISchema := getAdditionalItemsSchema(newSchema)

	// Compare items at each position
	minLen := len(oldTuple)
	if len(newTuple) < minLen {
		minLen = len(newTuple)
	}

	for i := 0; i < minLen; i++ {
		oldItem, oldOk := oldTuple[i].(map[string]interface{})
		newItem, newOk := newTuple[i].(map[string]interface{})
		if oldOk && newOk {
			c.checkCompatibility(newItem, oldItem, joinPath(path, fmt.Sprintf("items/%d", i)), result)
		}
	}

	// Items added to tuple
	if len(newTuple) > len(oldTuple) {
		for i := len(oldTuple); i < len(newTuple); i++ {
			newItem, newOk := newTuple[i].(map[string]interface{})
			if !newOk {
				continue
			}
			// Check if old had additionalItems schema covering this position
			if oldAISchema != nil {
				localResult := compatibility.NewCompatibleResult()
				c.checkCompatibility(newItem, oldAISchema, joinPath(path, fmt.Sprintf("items/%d", i)), localResult)
				if !localResult.IsCompatible {
					result.AddMessage("Item added at position %d not covered by additionalItems", i)
				}
			}
		}
	}

	// Items removed from tuple
	if len(oldTuple) > len(newTuple) {
		for i := len(newTuple); i < len(oldTuple); i++ {
			oldItem, oldOk := oldTuple[i].(map[string]interface{})
			if !oldOk {
				continue
			}
			// Check if new has additionalItems schema covering this position
			if newAISchema != nil {
				localResult := compatibility.NewCompatibleResult()
				c.checkCompatibility(newAISchema, oldItem, joinPath(path, fmt.Sprintf("items/%d", i)), localResult)
				if !localResult.IsCompatible {
					result.AddMessage("Item removed at position %d not covered by additionalItems", i)
				}
			}
		}
	}
}

// getTupleItems returns the tuple-style items array from a schema.
// Handles both Draft-07 (items as array) and Draft-2020 (prefixItems).
func getTupleItems(schema map[string]interface{}) []interface{} {
	// Draft-2020: prefixItems
	if prefixItems, ok := schema["prefixItems"].([]interface{}); ok {
		return prefixItems
	}
	// Draft-07: items as array
	if items, ok := schema["items"].([]interface{}); ok {
		return items
	}
	return nil
}

// getAdditionalItemsSchema returns the schema for additional items beyond tuple items.
// Draft-07: additionalItems schema, Draft-2020: items schema (when prefixItems present)
func getAdditionalItemsSchema(schema map[string]interface{}) map[string]interface{} {
	// If schema has prefixItems, then "items" is the additional items schema (Draft-2020)
	if _, hasPrefixItems := schema["prefixItems"]; hasPrefixItems {
		if items, ok := schema["items"].(map[string]interface{}); ok {
			return items
		}
		return nil
	}
	// Draft-07: additionalItems
	if ai, ok := schema["additionalItems"].(map[string]interface{}); ok {
		return ai
	}
	return nil
}

// ==========================================================================
// ENUM COMPATIBILITY
// ==========================================================================

// checkEnumCompatibility checks enum value changes.
func (c *Checker) checkEnumCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	newEnum := getEnum(newSchema)
	oldEnum := getEnum(oldSchema)

	if oldEnum == nil && newEnum != nil {
		// Enum constraint added — more restrictive
		result.AddMessage("Enum constraint added at '%s'", pathOrRoot(path))
		return
	}

	if oldEnum == nil {
		return
	}

	if newEnum == nil {
		// Enum constraint removed — compatible (less restrictive)
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

// ==========================================================================
// CONST COMPATIBILITY
// ==========================================================================

// checkConstCompatibility checks const value changes.
// const is semantically equivalent to an enum with a single value.
func (c *Checker) checkConstCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldConst, oldHas := oldSchema["const"]
	newConst, newHas := newSchema["const"]

	if !oldHas && !newHas {
		return
	}

	if oldHas && !newHas {
		// Removing const constraint — compatible (less restrictive)
		return
	}

	if !oldHas && newHas {
		// Adding const constraint — more restrictive
		result.AddMessage("const constraint added at '%s'", pathOrRoot(path))
		return
	}

	// Both have const — check if values differ
	if !reflect.DeepEqual(oldConst, newConst) {
		result.AddMessage("const value changed at '%s' from '%v' to '%v'", pathOrRoot(path), oldConst, newConst)
	}
}

// ==========================================================================
// ADDITIONAL PROPERTIES COMPATIBILITY
// ==========================================================================

// checkAdditionalPropertiesCompatibility checks additionalProperties changes.
func (c *Checker) checkAdditionalPropertiesCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	newAP, newHasAP := newSchema["additionalProperties"]
	oldAP, oldHasAP := oldSchema["additionalProperties"]

	// If old schema allowed additional properties and new doesn't
	if (!oldHasAP || oldAP == true) && newHasAP && newAP == false {
		result.AddMessage("additionalProperties changed from allowed to forbidden at '%s'", pathOrRoot(path))
	}

	// If old allowed additional properties schema and new narrows it
	if newAPSchema, newOk := newAP.(map[string]interface{}); newOk {
		if oldAPSchema, oldOk := oldAP.(map[string]interface{}); oldOk {
			c.checkCompatibility(newAPSchema, oldAPSchema, joinPath(path, "additionalProperties"), result)
		} else if !oldHasAP || oldAP == true {
			// Old was unrestricted, new has schema constraint — narrowing
			result.AddMessage("additionalProperties narrowed at '%s'", pathOrRoot(path))
		}
	}
}

// ==========================================================================
// COMPOSITION COMPATIBILITY (oneOf, anyOf, allOf)
// ==========================================================================

// hasCompositionKeyword returns true if the schema uses oneOf, anyOf, or allOf.
func hasCompositionKeyword(schema map[string]interface{}) bool {
	_, hasOneOf := schema["oneOf"]
	_, hasAnyOf := schema["anyOf"]
	_, hasAllOf := schema["allOf"]
	return hasOneOf || hasAnyOf || hasAllOf
}

// checkCompositionCompatibility handles oneOf, anyOf, allOf compatibility.
func (c *Checker) checkCompositionCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	// Handle sum type (oneOf/anyOf) compatibility
	c.checkSumTypeCompatibility(newSchema, oldSchema, path, result)

	// Handle allOf (product type) compatibility
	c.checkAllOfCompatibility(newSchema, oldSchema, path, result)

	// Check subschema compatibility for matching composition elements
	c.checkCompositionSubschemas(newSchema, oldSchema, path, result)
}

// checkCompositionSubschemas recursively checks internal structure of composition elements.
func (c *Checker) checkCompositionSubschemas(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	// Only check when both schemas use the same composition keyword with same element count
	for _, keyword := range []string{"oneOf", "anyOf"} {
		oldElems := getSchemaArrayValue(oldSchema, keyword)
		newElems := getSchemaArrayValue(newSchema, keyword)

		if len(oldElems) > 0 && len(newElems) > 0 && len(oldElems) == len(newElems) {
			// Schemas have the same number of elements — check each for internal compatibility
			for i := 0; i < len(oldElems); i++ {
				oldElem := oldElems[i]
				newElem := newElems[i]

				// Check if internal structure is compatible (e.g., property type changes)
				localResult := compatibility.NewCompatibleResult()
				c.checkCompatibility(newElem, oldElem, path, localResult)
				if !localResult.IsCompatible {
					result.AddMessage("Composed schema element changed at '%s'", pathOrRoot(path))
					return
				}
			}
		}
	}
}

// checkSumTypeCompatibility checks oneOf/anyOf compatibility.
// For backward compat: new schema must accept all type options from old schema.
func (c *Checker) checkSumTypeCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldOptions := c.collectSumTypeOptions(oldSchema)
	newOptions := c.collectSumTypeOptions(newSchema)

	if len(oldOptions) == 0 && len(newOptions) == 0 {
		return
	}
	if len(oldOptions) == 0 {
		return // Adding sum types to schema with no previous types — compatible (widening)
	}

	// For backward compat: each old option must have a compatible match in new
	for _, oldOpt := range oldOptions {
		if !c.hasCompatibleSumOption(newOptions, oldOpt) {
			oldType := getTypeString(oldOpt)
			if oldType == "" {
				oldType = "schema"
			}
			result.AddMessage("Type option '%s' removed at '%s'", oldType, pathOrRoot(path))
		}
	}
}

// collectSumTypeOptions extracts type options from oneOf, anyOf, allOf, or plain type.
func (c *Checker) collectSumTypeOptions(schema map[string]interface{}) []map[string]interface{} {
	if opts := getSchemaArrayValue(schema, "oneOf"); len(opts) > 0 {
		return opts
	}
	if opts := getSchemaArrayValue(schema, "anyOf"); len(opts) > 0 {
		return opts
	}
	// For allOf, compute the effective type (intersection).
	// allOf with a single type = that type. Multiple conflicting types = empty.
	if opts := getSchemaArrayValue(schema, "allOf"); len(opts) > 0 {
		typeSet := make(map[string]bool)
		for _, opt := range opts {
			if t := getTypeString(opt); t != "" {
				typeSet[t] = true
			}
		}
		if len(typeSet) == 1 {
			for t := range typeSet {
				return []map[string]interface{}{{"type": t}}
			}
		}
		// Multiple conflicting types = empty intersection = no valid options
		// Return nil so sum type check treats old schema as having no options
	}
	// Fall back to the schema's type as a single option
	t := getType(schema)
	if t != nil {
		types := normalizeType(t)
		opts := make([]map[string]interface{}, len(types))
		for i, typ := range types {
			opts[i] = map[string]interface{}{"type": typ}
		}
		return opts
	}
	return nil
}

// hasCompatibleSumOption checks if any new option is compatible with the old option.
func (c *Checker) hasCompatibleSumOption(newOptions []map[string]interface{}, oldOpt map[string]interface{}) bool {
	oldType := getTypeString(oldOpt)
	for _, newOpt := range newOptions {
		newType := getTypeString(newOpt)
		if newType == oldType {
			return true
		}
		if oldType != "" && newType != "" && isTypePromotion(oldType, newType) {
			return true
		}
	}
	return false
}

// checkAllOfCompatibility checks allOf (product type) compatibility.
func (c *Checker) checkAllOfCompatibility(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldAllOf := getSchemaArrayValue(oldSchema, "allOf")
	newAllOf := getSchemaArrayValue(newSchema, "allOf")

	if len(oldAllOf) == 0 && len(newAllOf) == 0 {
		return
	}

	// Deduplicate
	oldDeduped := deduplicateSchemas(oldAllOf)
	newDeduped := deduplicateSchemas(newAllOf)

	// Old has allOf, new doesn't — removing allOf constraints is compatible
	if len(oldDeduped) > 0 && len(newDeduped) == 0 {
		return
	}

	// New has allOf, old doesn't — adding allOf constraints
	if len(oldDeduped) == 0 && len(newDeduped) > 0 {
		// Collect old schema's effective types (from type, oneOf, anyOf)
		oldSumOptions := c.collectSumTypeOptionsExcludeAllOf(oldSchema)
		for _, newElem := range newDeduped {
			newType := getTypeString(newElem)
			oldType := typeString(getType(oldSchema))
			if newType != "" && oldType != "" && (newType == oldType || isTypePromotion(oldType, newType)) {
				continue // Same or compatible type constraint
			}
			// Check against old schema's sum type options (oneOf/anyOf)
			if newType != "" && len(oldSumOptions) > 0 {
				found := false
				for _, oldOpt := range oldSumOptions {
					oldOptType := getTypeString(oldOpt)
					if newType == oldOptType || isTypePromotion(oldOptType, newType) {
						found = true
						break
					}
				}
				if found {
					continue
				}
			}
			if !schemaSubsumedBy(newElem, oldSchema) {
				result.AddMessage("New constraint added to allOf at '%s'", pathOrRoot(path))
				return
			}
		}
		return
	}

	// Both have allOf — compare elements
	// New elements not in old = added constraints = incompatible
	for _, newElem := range newDeduped {
		if schemaExistsIn(newElem, oldDeduped) {
			continue // Exact match found
		}
		// Try matching by type
		newType := getTypeString(newElem)
		if newType != "" {
			found := false
			for _, oldElem := range oldDeduped {
				oldType := getTypeString(oldElem)
				if oldType == newType || isTypePromotion(oldType, newType) {
					found = true
					break
				}
			}
			if found {
				continue
			}
		}
		// Try matching by enum (both old and new have enum elements)
		if getEnum(newElem) != nil {
			found := false
			for _, oldElem := range oldDeduped {
				if getEnum(oldElem) != nil {
					// Both have enums — check if new is a compatible change
					localResult := compatibility.NewCompatibleResult()
					c.checkEnumCompatibility(newElem, oldElem, path, localResult)
					if !localResult.IsCompatible {
						for _, msg := range localResult.Messages {
							result.AddMessage("%s", msg)
						}
					}
					found = true
					break
				}
			}
			if found {
				continue
			}
		}
		// Try matching by shared keys (structural similarity)
		if c.hasMatchingElement(newElem, oldDeduped) {
			continue
		}
		result.AddMessage("New constraint added to allOf at '%s'", pathOrRoot(path))
	}

	// Check type changes within matching allOf elements
	for _, oldElem := range oldDeduped {
		oldType := getTypeString(oldElem)
		if oldType == "" {
			continue
		}
		for _, newElem := range newDeduped {
			newType := getTypeString(newElem)
			if newType != "" && oldType != "" && newType != oldType {
				if !isTypePromotion(oldType, newType) {
					// Check if both are type schemas (to avoid false positives with enum elements)
					_, oldHasType := oldElem["type"]
					_, newHasType := newElem["type"]
					if oldHasType && newHasType && len(oldElem) == 1 && len(newElem) == 1 {
						result.AddMessage("Type changed in allOf at '%s' from '%s' to '%s'", pathOrRoot(path), oldType, newType)
					}
				}
			}
		}
	}
}

// ==========================================================================
// STRING CONSTRAINTS
// ==========================================================================

// checkStringConstraints checks minLength, maxLength, and pattern changes.
func (c *Checker) checkStringConstraints(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	// minLength: increasing = incompatible
	c.checkConstraintChange(newSchema, oldSchema, "minLength", path, result, true)
	// maxLength: decreasing = incompatible
	c.checkConstraintChange(newSchema, oldSchema, "maxLength", path, result, false)

	// pattern changes
	oldPattern, oldHas := oldSchema["pattern"]
	newPattern, newHas := newSchema["pattern"]

	if oldHas && newHas && oldPattern != newPattern {
		result.AddMessage("pattern changed at '%s' from '%v' to '%v'", pathOrRoot(path), oldPattern, newPattern)
	} else if !oldHas && newHas {
		result.AddMessage("pattern constraint added at '%s'", pathOrRoot(path))
	}
	// Removing pattern is compatible (less restrictive)
}

// ==========================================================================
// NUMERIC CONSTRAINTS
// ==========================================================================

// checkNumericConstraints checks minimum, maximum, exclusiveMinimum, exclusiveMaximum, multipleOf.
func (c *Checker) checkNumericConstraints(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	// minimum: increasing = incompatible
	c.checkConstraintChange(newSchema, oldSchema, "minimum", path, result, true)
	// maximum: decreasing = incompatible
	c.checkConstraintChange(newSchema, oldSchema, "maximum", path, result, false)
	// exclusiveMinimum: increasing = incompatible
	c.checkConstraintChange(newSchema, oldSchema, "exclusiveMinimum", path, result, true)
	// exclusiveMaximum: decreasing = incompatible
	c.checkConstraintChange(newSchema, oldSchema, "exclusiveMaximum", path, result, false)

	// multipleOf changes
	oldMul, oldHas := oldSchema["multipleOf"]
	newMul, newHas := newSchema["multipleOf"]

	if oldHas && newHas {
		oldVal := toFloat64(oldMul)
		newVal := toFloat64(newMul)
		if oldVal != 0 && newVal != 0 {
			ratio := oldVal / newVal
			if math.Abs(ratio-math.Round(ratio)) > 1e-9 {
				result.AddMessage("multipleOf changed at '%s' from %v to %v", pathOrRoot(path), oldMul, newMul)
			}
		}
	} else if !oldHas && newHas {
		result.AddMessage("multipleOf constraint added at '%s'", pathOrRoot(path))
	}
}

// ==========================================================================
// PROPERTY COUNT CONSTRAINTS
// ==========================================================================

// checkPropertyCountConstraints checks maxProperties and minProperties.
func (c *Checker) checkPropertyCountConstraints(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	c.checkConstraintChange(newSchema, oldSchema, "minProperties", path, result, true)
	c.checkConstraintChange(newSchema, oldSchema, "maxProperties", path, result, false)
}

// ==========================================================================
// NOT SCHEMA
// ==========================================================================

// checkNotSchema checks "not" keyword changes.
func (c *Checker) checkNotSchema(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldNot, oldHas := oldSchema["not"]
	newNot, newHas := newSchema["not"]

	if !oldHas && !newHas {
		return
	}

	if !oldHas && newHas {
		result.AddMessage("'not' constraint added at '%s'", pathOrRoot(path))
		return
	}

	if oldHas && !newHas {
		return
	}

	oldNotMap, oldOk := oldNot.(map[string]interface{})
	newNotMap, newOk := newNot.(map[string]interface{})

	if oldOk && newOk {
		oldNotType := getTypeString(oldNotMap)
		newNotType := getTypeString(newNotMap)

		if oldNotType != "" && newNotType != "" && oldNotType != newNotType {
			if !isTypePromotion(newNotType, oldNotType) {
				result.AddMessage("'not' schema changed at '%s' from '%s' to '%s'", pathOrRoot(path), oldNotType, newNotType)
			}
		}

		if !reflect.DeepEqual(oldNotMap, newNotMap) && oldNotType == newNotType {
			if len(newNotMap) < len(oldNotMap) {
				result.AddMessage("'not' schema broadened at '%s'", pathOrRoot(path))
			}
		}
	}
}

// ==========================================================================
// DEPENDENCIES (Draft-07)
// ==========================================================================

// checkDependencies checks "dependencies" keyword changes.
func (c *Checker) checkDependencies(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldDeps, oldHas := oldSchema["dependencies"]
	newDeps, newHas := newSchema["dependencies"]

	if !oldHas && !newHas {
		return
	}

	if !oldHas && newHas {
		result.AddMessage("dependencies added at '%s'", pathOrRoot(path))
		return
	}

	if oldHas && !newHas {
		return
	}

	oldDepsMap, oldOk := oldDeps.(map[string]interface{})
	newDepsMap, newOk := newDeps.(map[string]interface{})
	if !oldOk || !newOk {
		return
	}

	// Check for added dependencies
	for propName := range newDepsMap {
		if _, exists := oldDepsMap[propName]; !exists {
			result.AddMessage("dependency added for property '%s' at '%s'", propName, pathOrRoot(path))
		}
	}

	// Check for changed/removed dependencies
	for propName, oldDep := range oldDepsMap {
		newDep, exists := newDepsMap[propName]
		if !exists {
			if _, isSchema := oldDep.(map[string]interface{}); isSchema {
				continue // Schema dependency removed — compatible
			}
			result.AddMessage("dependency removed for property '%s' at '%s'", propName, pathOrRoot(path))
			continue
		}

		// Both exist — check type-specific compatibility
		oldDepSchema, oldIsSchema := oldDep.(map[string]interface{})
		newDepSchema, newIsSchema := newDep.(map[string]interface{})
		if oldIsSchema && newIsSchema {
			c.checkCompatibility(newDepSchema, oldDepSchema, joinPath(path, "dependencies/"+propName), result)
		} else if !reflect.DeepEqual(oldDep, newDep) {
			result.AddMessage("dependency changed for property '%s' at '%s'", propName, pathOrRoot(path))
		}
	}
}

// ==========================================================================
// DEPENDENT REQUIRED (Draft-2020)
// ==========================================================================

// checkDependentRequired checks "dependentRequired" keyword changes (Draft-2020).
func (c *Checker) checkDependentRequired(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldDeps, oldHas := oldSchema["dependentRequired"]
	newDeps, newHas := newSchema["dependentRequired"]

	if !oldHas && !newHas {
		return
	}

	if !oldHas && newHas {
		result.AddMessage("dependentRequired added at '%s'", pathOrRoot(path))
		return
	}

	if oldHas && !newHas {
		return
	}

	oldDepsMap, oldOk := oldDeps.(map[string]interface{})
	newDepsMap, newOk := newDeps.(map[string]interface{})
	if !oldOk || !newOk {
		return
	}

	// Check for added dependency keys
	for propName := range newDepsMap {
		if _, exists := oldDepsMap[propName]; !exists {
			result.AddMessage("dependentRequired added for property '%s' at '%s'", propName, pathOrRoot(path))
		}
	}

	// Check for removed dependency keys
	for propName := range oldDepsMap {
		if _, exists := newDepsMap[propName]; !exists {
			result.AddMessage("dependentRequired removed for property '%s' at '%s'", propName, pathOrRoot(path))
		}
	}

	// Check for changed dependencies
	for propName, oldDep := range oldDepsMap {
		newDep, exists := newDepsMap[propName]
		if !exists {
			continue // Already handled above
		}
		if !reflect.DeepEqual(oldDep, newDep) {
			result.AddMessage("dependentRequired changed for property '%s' at '%s'", propName, pathOrRoot(path))
		}
	}
}

// ==========================================================================
// DEPENDENT SCHEMAS (Draft-2020)
// ==========================================================================

// checkDependentSchemas checks "dependentSchemas" keyword changes (Draft-2020).
func (c *Checker) checkDependentSchemas(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldDeps, oldHas := oldSchema["dependentSchemas"]
	newDeps, newHas := newSchema["dependentSchemas"]

	if !oldHas && !newHas {
		return
	}

	if !oldHas && newHas {
		result.AddMessage("dependentSchemas added at '%s'", pathOrRoot(path))
		return
	}

	if oldHas && !newHas {
		return
	}

	oldDepsMap, oldOk := oldDeps.(map[string]interface{})
	newDepsMap, newOk := newDeps.(map[string]interface{})
	if !oldOk || !newOk {
		return
	}

	// Check for added dependency keys
	for propName := range newDepsMap {
		if _, exists := oldDepsMap[propName]; !exists {
			result.AddMessage("dependentSchema added for property '%s' at '%s'", propName, pathOrRoot(path))
		}
	}

	// Check for removed dependency keys
	for propName := range oldDepsMap {
		if _, exists := newDepsMap[propName]; !exists {
			// Schema dependency removed — this is compatible (relaxing)
			continue
		}
	}

	// Check changed dependency schemas
	for propName, oldDep := range oldDepsMap {
		newDep, exists := newDepsMap[propName]
		if !exists {
			continue
		}
		oldDepSchema, oldIsSchema := oldDep.(map[string]interface{})
		newDepSchema, newIsSchema := newDep.(map[string]interface{})
		if oldIsSchema && newIsSchema {
			c.checkCompatibility(newDepSchema, oldDepSchema, joinPath(path, "dependencies/"+propName), result)
		}
	}
}

// ==========================================================================
// UNIQUE ITEMS
// ==========================================================================

// checkUniqueItems checks uniqueItems constraint changes.
func (c *Checker) checkUniqueItems(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldVal, oldHas := oldSchema["uniqueItems"]
	newVal, newHas := newSchema["uniqueItems"]

	if !newHas {
		return
	}
	if newHas && newVal == true && (!oldHas || oldVal != true) {
		result.AddMessage("uniqueItems constraint added at '%s'", pathOrRoot(path))
	}
}

// ==========================================================================
// ADDITIONAL ITEMS
// ==========================================================================

// checkAdditionalItems checks additionalItems constraint changes.
func (c *Checker) checkAdditionalItems(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	newAI, newHasAI := newSchema["additionalItems"]
	oldAI, oldHasAI := oldSchema["additionalItems"]

	if (!oldHasAI || oldAI == true) && newHasAI && newAI == false {
		result.AddMessage("additionalItems changed from allowed to forbidden at '%s'", pathOrRoot(path))
	}

	if newAISchema, newOk := newAI.(map[string]interface{}); newOk {
		if oldAISchema, oldOk := oldAI.(map[string]interface{}); oldOk {
			c.checkCompatibility(newAISchema, oldAISchema, joinPath(path, "additionalItems"), result)
		}
	}
}

// ==========================================================================
// ITEMS AS BOOLEAN (Draft-2020)
// ==========================================================================

// checkItemsBoolean checks items: true → items: false changes.
// In Draft-2020, items as boolean controls whether additional items beyond prefixItems are allowed.
func (c *Checker) checkItemsBoolean(newSchema, oldSchema map[string]interface{}, path string, result *compatibility.Result) {
	oldItems, oldHas := oldSchema["items"]
	newItems, newHas := newSchema["items"]

	// Only check boolean items values
	oldBool, oldIsBool := oldItems.(bool)
	newBool, newIsBool := newItems.(bool)

	if oldHas && newHas && oldIsBool && newIsBool {
		if oldBool && !newBool {
			// items: true → items: false = closing the model = incompatible
			result.AddMessage("items changed from allowed to forbidden at '%s'", pathOrRoot(path))
		}
	} else if oldHas && newHas && oldIsBool && oldBool && !newIsBool {
		// items: true → items: {schema} = narrowing = could be incompatible
		// but we handle this in checkArrayCompatibility via getItems
	} else if oldHas && newHas && !oldIsBool && newIsBool && !newBool {
		// items: {schema} → items: false = closing the model
		result.AddMessage("items changed from schema to forbidden at '%s'", pathOrRoot(path))
	}
}

// ==========================================================================
// CONSTRAINT CHECKING
// ==========================================================================

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
		if newHas && (!oldHas || newNum > oldNum) {
			result.AddMessage("'%s' constraint tightened at '%s' (was %v, now %v)", constraint, pathOrRoot(path), oldVal, newVal)
		}
	} else {
		if newHas && (!oldHas || newNum < oldNum) {
			result.AddMessage("'%s' constraint tightened at '%s' (was %v, now %v)", constraint, pathOrRoot(path), oldVal, newVal)
		}
	}
}

// ==========================================================================
// TYPE COMPATIBILITY
// ==========================================================================

// areTypesCompatible checks if two types are compatible.
func (c *Checker) areTypesCompatible(newType, oldType interface{}) bool {
	if oldType == nil {
		return true
	}
	if newType == nil {
		return true
	}

	newTypes := normalizeType(newType)
	oldTypes := normalizeType(oldType)

	for _, ot := range oldTypes {
		found := false
		for _, nt := range newTypes {
			if nt == ot || isTypePromotion(ot, nt) {
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

// isTypePromotion checks if oldType can be promoted to newType.
func isTypePromotion(oldType, newType string) bool {
	return oldType == "integer" && newType == "number"
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

// ==========================================================================
// HELPER FUNCTIONS
// ==========================================================================

func getType(schema map[string]interface{}) interface{} {
	return schema["type"]
}

func typeString(t interface{}) string {
	if s, ok := t.(string); ok {
		return s
	}
	return ""
}

func getTypeString(schema map[string]interface{}) string {
	return typeString(getType(schema))
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

// hasOpenContentModel determines if a JSON Schema has an open content model.
func hasOpenContentModel(schema map[string]interface{}) bool {
	ap, hasAP := schema["additionalProperties"]
	if !hasAP {
		return true
	}
	if boolVal, ok := ap.(bool); ok {
		return boolVal
	}
	// additionalProperties is a schema object — partially open (not fully open)
	return false
}

// hasCoveringPatternProperties checks if a schema has patternProperties that
// could cover a removed named property. If the new schema has patternProperties,
// the removed property may still be validated by a pattern match.
func hasCoveringPatternProperties(schema map[string]interface{}) bool {
	pp, has := schema["patternProperties"]
	if !has {
		return false
	}
	ppMap, ok := pp.(map[string]interface{})
	return ok && len(ppMap) > 0
}

// getSchemaArrayValue extracts an array of schema objects from a keyword.
func getSchemaArrayValue(schema map[string]interface{}, key string) []map[string]interface{} {
	arr, ok := schema[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result
}

// deduplicateSchemas removes duplicate schemas from a slice.
func deduplicateSchemas(schemas []map[string]interface{}) []map[string]interface{} {
	if len(schemas) == 0 {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(schemas))
	for _, s := range schemas {
		isDup := false
		for _, existing := range result {
			if reflect.DeepEqual(s, existing) {
				isDup = true
				break
			}
		}
		if !isDup {
			result = append(result, s)
		}
	}
	return result
}

// schemaExistsIn checks if a schema exists in a slice (by deep equality).
func schemaExistsIn(schema map[string]interface{}, schemas []map[string]interface{}) bool {
	for _, s := range schemas {
		if reflect.DeepEqual(schema, s) {
			return true
		}
	}
	return false
}

// collectSumTypeOptionsExcludeAllOf extracts type options from oneOf, anyOf, or plain type (not allOf).
func (c *Checker) collectSumTypeOptionsExcludeAllOf(schema map[string]interface{}) []map[string]interface{} {
	if opts := getSchemaArrayValue(schema, "oneOf"); len(opts) > 0 {
		return opts
	}
	if opts := getSchemaArrayValue(schema, "anyOf"); len(opts) > 0 {
		return opts
	}
	t := getType(schema)
	if t != nil {
		types := normalizeType(t)
		opts := make([]map[string]interface{}, len(types))
		for i, typ := range types {
			opts[i] = map[string]interface{}{"type": typ}
		}
		return opts
	}
	return nil
}

// hasMatchingElement checks if newElem has a structurally similar match in oldSchemas.
func (c *Checker) hasMatchingElement(newElem map[string]interface{}, oldSchemas []map[string]interface{}) bool {
	for _, oldElem := range oldSchemas {
		if len(newElem) > 0 && len(oldElem) > 0 {
			sharedKeys := 0
			for k := range newElem {
				if _, ok := oldElem[k]; ok {
					sharedKeys++
				}
			}
			if sharedKeys > 0 && sharedKeys == len(newElem) {
				return true
			}
		}
	}
	return false
}

// schemaSubsumedBy checks if newConstraint is already satisfied by oldSchema.
func schemaSubsumedBy(newConstraint, oldSchema map[string]interface{}) bool {
	newType := getTypeString(newConstraint)
	oldType := typeString(getType(oldSchema))
	if newType != "" && oldType != "" {
		return newType == oldType || isTypePromotion(oldType, newType)
	}
	return reflect.DeepEqual(newConstraint, oldSchema)
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

// Ensure Checker implements compatibility.SchemaChecker
var _ compatibility.SchemaChecker = (*Checker)(nil)
