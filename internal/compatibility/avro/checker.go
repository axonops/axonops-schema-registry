// Package avro provides Avro schema compatibility checking.
package avro

import (
	"fmt"

	"github.com/hamba/avro/v2"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
)

// Checker implements Avro schema compatibility checking.
type Checker struct{}

// NewChecker creates a new Avro compatibility checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Check checks compatibility between reader and writer schemas.
// For BACKWARD compatibility: reader=new schema, writer=old schema
// For FORWARD compatibility: reader=old schema, writer=new schema
func (c *Checker) Check(reader, writer compatibility.SchemaWithRefs) *compatibility.Result {
	readerSchema, err := c.parseSchema(reader)
	if err != nil {
		return compatibility.NewIncompatibleResult(fmt.Sprintf("invalid reader schema: %v", err))
	}

	writerSchema, err := c.parseSchema(writer)
	if err != nil {
		return compatibility.NewIncompatibleResult(fmt.Sprintf("invalid writer schema: %v", err))
	}

	return c.checkSchemas(readerSchema, writerSchema, "")
}

// parseSchema parses a schema string with optional reference resolution.
func (c *Checker) parseSchema(s compatibility.SchemaWithRefs) (avro.Schema, error) {
	if len(s.References) > 0 {
		cache := &avro.SchemaCache{}
		for _, ref := range s.References {
			if ref.Schema != "" {
				if _, err := avro.ParseWithCache(ref.Schema, "", cache); err != nil {
					return nil, fmt.Errorf("invalid reference schema %q: %w", ref.Name, err)
				}
			}
		}
		return avro.ParseWithCache(s.Schema, "", cache)
	}
	return avro.Parse(s.Schema)
}

// checkSchemas recursively checks compatibility between two schemas.
func (c *Checker) checkSchemas(reader, writer avro.Schema, path string) *compatibility.Result {
	result := compatibility.NewCompatibleResult()

	// Handle schema promotion (widening)
	if c.isPromotable(writer, reader) {
		return result
	}

	// Types must match (with some exceptions for unions)
	if reader.Type() != writer.Type() {
		// Check if writer is promotable to reader
		if !c.canPromote(writer, reader) {
			// Check union compatibility
			if reader.Type() == avro.Union {
				return c.checkReaderUnion(reader, writer, path)
			}
			if writer.Type() == avro.Union {
				return c.checkWriterUnion(reader, writer, path)
			}
			result.AddMessage("%s: type mismatch: reader has %s, writer has %s",
				pathOrRoot(path), reader.Type(), writer.Type())
			return result
		}
	}

	// Type-specific compatibility checks
	switch reader.Type() {
	case avro.Record:
		return c.checkRecord(reader.(*avro.RecordSchema), writer.(*avro.RecordSchema), path)
	case avro.Enum:
		return c.checkEnum(reader.(*avro.EnumSchema), writer.(*avro.EnumSchema), path)
	case avro.Array:
		return c.checkArray(reader.(*avro.ArraySchema), writer.(*avro.ArraySchema), path)
	case avro.Map:
		return c.checkMap(reader.(*avro.MapSchema), writer.(*avro.MapSchema), path)
	case avro.Union:
		return c.checkUnion(reader.(*avro.UnionSchema), writer.(*avro.UnionSchema), path)
	case avro.Fixed:
		return c.checkFixed(reader.(*avro.FixedSchema), writer.(*avro.FixedSchema), path)
	case avro.String, avro.Bytes, avro.Int, avro.Long, avro.Float, avro.Double, avro.Boolean, avro.Null:
		// Primitive types are compatible if types match (already checked above)
		return result
	default:
		return result
	}
}

// checkRecord checks compatibility between two record schemas.
func (c *Checker) checkRecord(reader, writer *avro.RecordSchema, path string) *compatibility.Result {
	result := compatibility.NewCompatibleResult()

	// Check that names match (considering aliases)
	if !c.recordNamesMatch(reader, writer) {
		result.AddMessage("%s: record name mismatch: reader has %s, writer has %s",
			pathOrRoot(path), reader.FullName(), writer.FullName())
		return result
	}

	// Build maps of writer fields by name and aliases
	writerFields := make(map[string]*avro.Field)
	for _, f := range writer.Fields() {
		writerFields[f.Name()] = f
		for _, alias := range f.Aliases() {
			writerFields[alias] = f
		}
	}

	// Check each reader field
	for _, rf := range reader.Fields() {
		fieldPath := appendPath(path, rf.Name())

		// Try to find matching writer field by name or reader's aliases
		wf := c.findWriterField(rf, writerFields)

		if wf == nil {
			// Field doesn't exist in writer - reader must have a default
			if !rf.HasDefault() {
				result.AddMessage("%s: reader field '%s' has no default and is missing from writer",
					pathOrRoot(path), rf.Name())
			}
			continue
		}

		// Check field type compatibility
		fieldResult := c.checkSchemas(rf.Type(), wf.Type(), fieldPath)
		result.Merge(fieldResult)
	}

	return result
}

// recordNamesMatch checks if reader and writer record names match,
// considering aliases on both sides per the Avro specification.
func (c *Checker) recordNamesMatch(reader, writer *avro.RecordSchema) bool {
	if reader.FullName() == writer.FullName() {
		return true
	}
	// Check if reader name matches any writer alias
	for _, alias := range writer.Aliases() {
		if reader.FullName() == alias {
			return true
		}
	}
	// Check if writer name matches any reader alias
	for _, alias := range reader.Aliases() {
		if writer.FullName() == alias {
			return true
		}
	}
	return false
}

// findWriterField finds a matching writer field for a reader field,
// checking by name and aliases per the Avro specification.
func (c *Checker) findWriterField(readerField *avro.Field, writerFields map[string]*avro.Field) *avro.Field {
	// Direct name match (also matches writer field aliases via the map)
	if wf, exists := writerFields[readerField.Name()]; exists {
		return wf
	}
	// Check reader field aliases against writer field names/aliases
	for _, alias := range readerField.Aliases() {
		if wf, exists := writerFields[alias]; exists {
			return wf
		}
	}
	return nil
}

// checkEnum checks compatibility between two enum schemas.
func (c *Checker) checkEnum(reader, writer *avro.EnumSchema, path string) *compatibility.Result {
	result := compatibility.NewCompatibleResult()

	// Check that names match
	if reader.FullName() != writer.FullName() {
		result.AddMessage("%s: enum name mismatch: reader has %s, writer has %s",
			pathOrRoot(path), reader.FullName(), writer.FullName())
		return result
	}

	// Check that all writer symbols are in reader
	// (reader can have additional symbols - those will use default if set)
	readerSymbols := make(map[string]bool)
	for _, s := range reader.Symbols() {
		readerSymbols[s] = true
	}

	for _, ws := range writer.Symbols() {
		if !readerSymbols[ws] {
			// Writer has a symbol that reader doesn't have
			// This is only compatible if reader has a default
			if reader.Default() == "" {
				result.AddMessage("%s: writer enum symbol '%s' not found in reader and no default set",
					pathOrRoot(path), ws)
			}
		}
	}

	return result
}

// checkArray checks compatibility between two array schemas.
func (c *Checker) checkArray(reader, writer *avro.ArraySchema, path string) *compatibility.Result {
	return c.checkSchemas(reader.Items(), writer.Items(), appendPath(path, "[]"))
}

// checkMap checks compatibility between two map schemas.
func (c *Checker) checkMap(reader, writer *avro.MapSchema, path string) *compatibility.Result {
	return c.checkSchemas(reader.Values(), writer.Values(), appendPath(path, "{}"))
}

// checkUnion checks compatibility between two union schemas.
func (c *Checker) checkUnion(reader, writer *avro.UnionSchema, path string) *compatibility.Result {
	result := compatibility.NewCompatibleResult()

	// Each writer type must be compatible with at least one reader type
	for _, wt := range writer.Types() {
		found := false
		for _, rt := range reader.Types() {
			if c.checkSchemas(rt, wt, path).IsCompatible {
				found = true
				break
			}
		}
		if !found {
			result.AddMessage("%s: writer union type %s is not compatible with any reader union type",
				pathOrRoot(path), wt.Type())
		}
	}

	return result
}

// checkReaderUnion handles the case where reader is a union but writer is not.
func (c *Checker) checkReaderUnion(reader, writer avro.Schema, path string) *compatibility.Result {
	union := reader.(*avro.UnionSchema)

	// Writer type must be compatible with at least one type in the reader union
	for _, rt := range union.Types() {
		if c.checkSchemas(rt, writer, path).IsCompatible {
			return compatibility.NewCompatibleResult()
		}
	}

	return compatibility.NewIncompatibleResult(
		fmt.Sprintf("%s: writer type %s is not compatible with any type in reader union",
			pathOrRoot(path), writer.Type()))
}

// checkWriterUnion handles the case where writer is a union but reader is not.
func (c *Checker) checkWriterUnion(reader, writer avro.Schema, path string) *compatibility.Result {
	union := writer.(*avro.UnionSchema)

	// All writer union types must be compatible with the reader type
	for _, wt := range union.Types() {
		result := c.checkSchemas(reader, wt, path)
		if !result.IsCompatible {
			return compatibility.NewIncompatibleResult(
				fmt.Sprintf("%s: reader type %s cannot read writer union type %s",
					pathOrRoot(path), reader.Type(), wt.Type()))
		}
	}

	return compatibility.NewCompatibleResult()
}

// checkFixed checks compatibility between two fixed schemas.
func (c *Checker) checkFixed(reader, writer *avro.FixedSchema, path string) *compatibility.Result {
	result := compatibility.NewCompatibleResult()

	if reader.FullName() != writer.FullName() {
		result.AddMessage("%s: fixed name mismatch: reader has %s, writer has %s",
			pathOrRoot(path), reader.FullName(), writer.FullName())
	}

	if reader.Size() != writer.Size() {
		result.AddMessage("%s: fixed size mismatch: reader has %d, writer has %d",
			pathOrRoot(path), reader.Size(), writer.Size())
	}

	return result
}

// isPromotable checks if writer can be promoted to reader (numeric widening).
func (c *Checker) isPromotable(writer, reader avro.Schema) bool {
	return c.canPromote(writer, reader)
}

// canPromote checks if a writer type can be promoted to a reader type.
// Avro supports: int -> long, float, double; long -> float, double; float -> double
// Also: string <-> bytes
func (c *Checker) canPromote(writer, reader avro.Schema) bool {
	wt, rt := writer.Type(), reader.Type()

	switch wt {
	case avro.Int:
		return rt == avro.Long || rt == avro.Float || rt == avro.Double
	case avro.Long:
		return rt == avro.Float || rt == avro.Double
	case avro.Float:
		return rt == avro.Double
	case avro.String:
		return rt == avro.Bytes
	case avro.Bytes:
		return rt == avro.String
	}

	return false
}

// pathOrRoot returns the path or "root" if empty.
func pathOrRoot(path string) string {
	if path == "" {
		return "root"
	}
	return path
}

// appendPath appends a segment to a path.
func appendPath(path, segment string) string {
	if path == "" {
		return segment
	}
	return path + "." + segment
}
