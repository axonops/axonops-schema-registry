package compatibility

import (
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// SchemaWithRefs bundles a schema string with its resolved references.
type SchemaWithRefs struct {
	Schema     string
	References []storage.Reference
}

// SchemaChecker is the interface for type-specific compatibility checkers.
type SchemaChecker interface {
	// Check checks compatibility between reader and writer schemas.
	Check(reader, writer SchemaWithRefs) *Result
}

// Checker orchestrates compatibility checking across schema types.
type Checker struct {
	checkers map[storage.SchemaType]SchemaChecker
}

// NewChecker creates a new compatibility checker.
func NewChecker() *Checker {
	return &Checker{
		checkers: make(map[storage.SchemaType]SchemaChecker),
	}
}

// Register registers a schema-specific checker.
func (c *Checker) Register(schemaType storage.SchemaType, checker SchemaChecker) {
	c.checkers[schemaType] = checker
}

// Check checks if a new schema is compatible with existing schemas.
// The mode determines what compatibility checks are performed.
// existingSchemas should be ordered from oldest to newest.
func (c *Checker) Check(mode Mode, schemaType storage.SchemaType, newSchema SchemaWithRefs, existingSchemas []SchemaWithRefs) *Result {
	// NONE mode always passes
	if mode == ModeNone {
		return NewCompatibleResult()
	}

	// No existing schemas means always compatible
	if len(existingSchemas) == 0 {
		return NewCompatibleResult()
	}

	checker, ok := c.checkers[schemaType]
	if !ok {
		return NewIncompatibleResult("no compatibility checker for schema type: " + string(schemaType))
	}

	result := NewCompatibleResult()

	// Determine which schemas to check against
	var schemasToCheck []SchemaWithRefs
	if mode.IsTransitive() {
		// Check against all previous schemas
		schemasToCheck = existingSchemas
	} else {
		// Check only against the latest schema
		schemasToCheck = []SchemaWithRefs{existingSchemas[len(existingSchemas)-1]}
	}

	for i, existingSchema := range schemasToCheck {
		var checkResult *Result

		if mode.RequiresBackward() {
			// BACKWARD: new schema (reader) can read data from old schema (writer)
			checkResult = checker.Check(newSchema, existingSchema)
			if !checkResult.IsCompatible {
				for _, msg := range checkResult.Messages {
					result.AddMessage("BACKWARD compatibility check failed against version %d: %s", i+1, msg)
				}
			}
		}

		if mode.RequiresForward() {
			// FORWARD: old schema (reader) can read data from new schema (writer)
			checkResult = checker.Check(existingSchema, newSchema)
			if !checkResult.IsCompatible {
				for _, msg := range checkResult.Messages {
					result.AddMessage("FORWARD compatibility check failed against version %d: %s", i+1, msg)
				}
			}
		}
	}

	return result
}

// CheckPair checks compatibility between two specific schemas.
func (c *Checker) CheckPair(mode Mode, schemaType storage.SchemaType, newSchema, existingSchema SchemaWithRefs) *Result {
	return c.Check(mode, schemaType, newSchema, []SchemaWithRefs{existingSchema})
}
