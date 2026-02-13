// Package protobuf provides Protobuf schema compatibility checking.
package protobuf

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Checker implements compatibility.SchemaChecker for Protobuf schemas.
type Checker struct{}

// NewChecker creates a new Protobuf compatibility checker.
func NewChecker() *Checker {
	return &Checker{}
}

// Check checks compatibility between reader and writer Protobuf schemas.
// For Protobuf, the "reader" is the new schema and "writer" is the old schema.
// This follows the same convention as Avro.
func (c *Checker) Check(reader, writer compatibility.SchemaWithRefs) *compatibility.Result {
	// Parse both schemas
	readerFD, err := parseSchemaWithRefs(reader)
	if err != nil {
		return compatibility.NewIncompatibleResult("failed to parse new schema: " + err.Error())
	}

	writerFD, err := parseSchemaWithRefs(writer)
	if err != nil {
		return compatibility.NewIncompatibleResult("failed to parse old schema: " + err.Error())
	}

	result := compatibility.NewCompatibleResult()

	// Check package compatibility
	if readerFD.Package() != writerFD.Package() {
		result.AddMessage("Package changed from '%s' to '%s'", writerFD.Package(), readerFD.Package())
	}

	// Syntax keyword is a source-level annotation only; proto2 optional and proto3
	// fields produce identical wire bytes. Confluent ignores syntax changes.

	// Check messages
	c.checkMessages(readerFD, writerFD, result)

	// Check enums
	c.checkEnums(readerFD, writerFD, result)

	// Services are gRPC metadata with zero wire-format impact.
	// Confluent ignores service definitions entirely.

	return result
}

// parseSchemaWithRefs parses a Protobuf schema string with optional references.
func parseSchemaWithRefs(s compatibility.SchemaWithRefs) (protoreflect.FileDescriptor, error) {
	handler := reporter.NewHandler(nil)
	_, err := parser.Parse("schema.proto", strings.NewReader(s.Schema), handler)
	if err != nil {
		return nil, err
	}

	resolver := newCheckerResolver(s.Schema, s.References)

	compiler := protocompile.Compiler{
		Resolver: resolver,
	}

	ctx := context.Background()
	files, err := compiler.Compile(ctx, "schema.proto")
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files compiled")
	}

	return files[0], nil
}

// checkerResolver resolves protobuf imports from schema references and well-known types.
type checkerResolver struct {
	content   string
	refs      map[string]string
	wellKnown map[string]string
}

// newCheckerResolver creates a resolver for the compatibility checker.
func newCheckerResolver(content string, refs []storage.Reference) *checkerResolver {
	r := &checkerResolver{
		content:   content,
		refs:      make(map[string]string),
		wellKnown: checkerWellKnownTypes(),
	}
	for _, ref := range refs {
		if ref.Name != "" {
			r.refs[ref.Name] = ref.Schema
		}
	}
	return r
}

func (r *checkerResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	if path == "schema.proto" {
		return protocompile.SearchResult{
			Source: strings.NewReader(r.content),
		}, nil
	}
	// Check well-known types
	if content, ok := r.wellKnown[path]; ok {
		return protocompile.SearchResult{
			Source: strings.NewReader(content),
		}, nil
	}
	// Check references
	if content, ok := r.refs[path]; ok && content != "" {
		return protocompile.SearchResult{
			Source: strings.NewReader(content),
		}, nil
	}
	return protocompile.SearchResult{}, fmt.Errorf("file not found: %s", path)
}

// checkMessages checks compatibility of messages.
func (c *Checker) checkMessages(reader, writer protoreflect.FileDescriptor, result *compatibility.Result) {
	// Build map of old messages
	oldMessages := make(map[string]protoreflect.MessageDescriptor)
	for i := 0; i < writer.Messages().Len(); i++ {
		msg := writer.Messages().Get(i)
		oldMessages[string(msg.FullName())] = msg
	}

	// Check each new message
	for i := 0; i < reader.Messages().Len(); i++ {
		newMsg := reader.Messages().Get(i)
		name := string(newMsg.FullName())

		oldMsg, exists := oldMessages[name]
		if !exists {
			// New message added - always compatible
			continue
		}

		c.checkMessageCompatibility(newMsg, oldMsg, result)
		delete(oldMessages, name)
	}

	// Messages removed from new schema
	for name := range oldMessages {
		result.AddMessage("Message '%s' was removed", name)
	}
}

// checkMessageCompatibility checks compatibility between two message descriptors.
func (c *Checker) checkMessageCompatibility(newMsg, oldMsg protoreflect.MessageDescriptor, result *compatibility.Result) {
	msgName := string(newMsg.FullName())

	// Build map of old fields by number
	oldFields := make(map[int32]protoreflect.FieldDescriptor)
	for i := 0; i < oldMsg.Fields().Len(); i++ {
		f := oldMsg.Fields().Get(i)
		oldFields[int32(f.Number())] = f
	}

	// Check each new field
	for i := 0; i < newMsg.Fields().Len(); i++ {
		newField := newMsg.Fields().Get(i)
		num := int32(newField.Number())

		oldField, exists := oldFields[num]
		if !exists {
			// New field added
			// For backward compatibility, new required fields are problematic
			if newField.Cardinality() == protoreflect.Required {
				result.AddMessage("Message '%s': new required field '%s' (number %d) added",
					msgName, newField.Name(), num)
			}
			continue
		}

		// Check field compatibility
		c.checkFieldCompatibility(newField, oldField, msgName, result)
		delete(oldFields, num)
	}

	// In proto3, field removal is wire-safe: readers ignore unknown fields and use
	// defaults for missing ones. However, removing a field from a oneof IS
	// incompatible because it changes oneof semantics.
	for num, oldField := range oldFields {
		if oldField.ContainingOneof() != nil {
			result.AddMessage("Message '%s': field '%s' (number %d) was removed from oneof",
				msgName, oldField.Name(), num)
		}
	}

	// Check nested messages
	c.checkNestedMessages(newMsg, oldMsg, result)

	// Check nested enums
	c.checkNestedEnums(newMsg, oldMsg, result)
}

// checkFieldCompatibility checks compatibility between two field descriptors.
func (c *Checker) checkFieldCompatibility(newField, oldField protoreflect.FieldDescriptor, msgName string, result *compatibility.Result) {
	fieldName := string(newField.Name())
	fieldNum := newField.Number()

	// Check name change (allowed but worth noting)
	if newField.Name() != oldField.Name() {
		// Name change is allowed in protobuf (wire format uses number)
		// But it's worth noting for documentation
	}

	// Check type compatibility
	if !c.areTypesCompatible(newField, oldField) {
		result.AddMessage("Message '%s': field %d type changed from '%s' to '%s'",
			msgName, fieldNum, protoTypeName(oldField), protoTypeName(newField))
	}

	// Check cardinality changes
	oldCard := oldField.Cardinality()
	newCard := newField.Cardinality()

	if oldCard != newCard {
		// Some cardinality changes are compatible
		if oldCard == protoreflect.Optional && newCard == protoreflect.Repeated {
			// Optional to repeated - compatible for reading
		} else if oldCard == protoreflect.Required && newCard != protoreflect.Required {
			// Required to optional/repeated - compatible
		} else if newCard == protoreflect.Required && oldCard != protoreflect.Required {
			// Non-required to required - breaking
			result.AddMessage("Message '%s': field '%s' changed from optional to required",
				msgName, fieldName)
		} else if oldCard == protoreflect.Repeated && newCard != protoreflect.Repeated {
			// Per protobuf spec: "For string, bytes, and message fields, singular is
			// compatible with repeated." These use length-delimited encoding which
			// is the same wire format for both singular and repeated.
			kind := newField.Kind()
			if kind != protoreflect.StringKind && kind != protoreflect.BytesKind && kind != protoreflect.MessageKind {
				result.AddMessage("Message '%s': field '%s' changed from repeated to singular",
					msgName, fieldName)
			}
		}
	}

	// Check oneof membership changes
	oldOneof := oldField.ContainingOneof()
	newOneof := newField.ContainingOneof()
	if (oldOneof == nil) != (newOneof == nil) {
		result.AddMessage("Message '%s': field '%s' oneof membership changed",
			msgName, fieldName)
	}
}

// areTypesCompatible checks if two field types are compatible.
func (c *Checker) areTypesCompatible(newField, oldField protoreflect.FieldDescriptor) bool {
	newKind := newField.Kind()
	oldKind := oldField.Kind()

	if newKind == oldKind {
		// Same kind - check message/enum types
		if newKind == protoreflect.MessageKind {
			return newField.Message().FullName() == oldField.Message().FullName()
		}
		if newKind == protoreflect.EnumKind {
			return newField.Enum().FullName() == oldField.Enum().FullName()
		}
		return true
	}

	// Wire-type compatible groups per official protobuf specification:
	// - Varint (wire type 0): int32, uint32, int64, uint64, bool
	// - Zigzag varint (wire type 0): sint32, sint64
	// - 32-bit (wire type 5): fixed32, sfixed32
	// - 64-bit (wire type 1): fixed64, sfixed64
	// - Length-delimited (wire type 2): string, bytes
	compatibleGroups := [][]protoreflect.Kind{
		{protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind, protoreflect.BoolKind},
		{protoreflect.Sint32Kind, protoreflect.Sint64Kind},
		{protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind},
		{protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind},
		{protoreflect.StringKind, protoreflect.BytesKind},
	}

	for _, group := range compatibleGroups {
		oldInGroup := false
		newInGroup := false
		for _, k := range group {
			if oldKind == k {
				oldInGroup = true
			}
			if newKind == k {
				newInGroup = true
			}
		}
		if oldInGroup && newInGroup {
			return true
		}
	}

	// Enum is wire-compatible with varint types (int32, uint32, int64, uint64, bool)
	// because enums use varint encoding on the wire.
	varintKinds := []protoreflect.Kind{
		protoreflect.Int32Kind, protoreflect.Uint32Kind,
		protoreflect.Int64Kind, protoreflect.Uint64Kind,
		protoreflect.BoolKind,
	}
	if oldKind == protoreflect.EnumKind || newKind == protoreflect.EnumKind {
		otherKind := newKind
		if oldKind != protoreflect.EnumKind {
			otherKind = oldKind
		}
		if otherKind == protoreflect.EnumKind {
			// Both are enums but different enum types - handled by same-kind check above
			return false
		}
		for _, k := range varintKinds {
			if otherKind == k {
				return true
			}
		}
	}

	return false
}

// checkNestedMessages checks compatibility of nested messages.
func (c *Checker) checkNestedMessages(newMsg, oldMsg protoreflect.MessageDescriptor, result *compatibility.Result) {
	oldNested := make(map[string]protoreflect.MessageDescriptor)
	for i := 0; i < oldMsg.Messages().Len(); i++ {
		nm := oldMsg.Messages().Get(i)
		if !nm.IsMapEntry() {
			oldNested[string(nm.Name())] = nm
		}
	}

	for i := 0; i < newMsg.Messages().Len(); i++ {
		nm := newMsg.Messages().Get(i)
		if nm.IsMapEntry() {
			continue
		}
		name := string(nm.Name())

		if oldNm, exists := oldNested[name]; exists {
			c.checkMessageCompatibility(nm, oldNm, result)
			delete(oldNested, name)
		}
	}

	for name := range oldNested {
		result.AddMessage("Nested message '%s.%s' was removed", oldMsg.FullName(), name)
	}
}

// checkNestedEnums checks compatibility of nested enums.
func (c *Checker) checkNestedEnums(newMsg, oldMsg protoreflect.MessageDescriptor, result *compatibility.Result) {
	oldEnums := make(map[string]protoreflect.EnumDescriptor)
	for i := 0; i < oldMsg.Enums().Len(); i++ {
		e := oldMsg.Enums().Get(i)
		oldEnums[string(e.Name())] = e
	}

	for i := 0; i < newMsg.Enums().Len(); i++ {
		e := newMsg.Enums().Get(i)
		name := string(e.Name())

		if oldE, exists := oldEnums[name]; exists {
			c.checkEnumCompatibility(e, oldE, result)
			delete(oldEnums, name)
		}
	}

	// Enum type removal is wire-compatible: enum fields are just integers on
	// the wire, so removing the type definition doesn't affect wire format.
}

// checkEnums checks compatibility of top-level enums.
func (c *Checker) checkEnums(reader, writer protoreflect.FileDescriptor, result *compatibility.Result) {
	oldEnums := make(map[string]protoreflect.EnumDescriptor)
	for i := 0; i < writer.Enums().Len(); i++ {
		e := writer.Enums().Get(i)
		oldEnums[string(e.FullName())] = e
	}

	for i := 0; i < reader.Enums().Len(); i++ {
		newEnum := reader.Enums().Get(i)
		name := string(newEnum.FullName())

		if oldEnum, exists := oldEnums[name]; exists {
			c.checkEnumCompatibility(newEnum, oldEnum, result)
			delete(oldEnums, name)
		}
	}

	// Enum type removal is wire-compatible: enum fields are integers on the
	// wire, removing the type definition doesn't affect wire format.
}

// checkEnumCompatibility checks compatibility between two enum descriptors.
func (c *Checker) checkEnumCompatibility(newEnum, oldEnum protoreflect.EnumDescriptor, result *compatibility.Result) {
	// Build map of old values by number
	oldValues := make(map[int32]protoreflect.EnumValueDescriptor)
	for i := 0; i < oldEnum.Values().Len(); i++ {
		v := oldEnum.Values().Get(i)
		oldValues[int32(v.Number())] = v
	}

	// Check each new value
	for i := 0; i < newEnum.Values().Len(); i++ {
		newValue := newEnum.Values().Get(i)
		num := int32(newValue.Number())

		if oldValue, exists := oldValues[num]; exists {
			// Value exists - check if name changed (allowed but notable)
			if newValue.Name() != oldValue.Name() {
				// Name change is allowed
			}
			delete(oldValues, num)
		}
		// New values are always compatible (added to enum)
	}

	// Enum values are integers on the wire. Removing a value name doesn't affect
	// wire compatibility — unknown values are preserved as numeric values.
	// Confluent allows enum value removal.
}

// checkServices checks compatibility of services.
func (c *Checker) checkServices(reader, writer protoreflect.FileDescriptor, result *compatibility.Result) {
	oldServices := make(map[string]protoreflect.ServiceDescriptor)
	for i := 0; i < writer.Services().Len(); i++ {
		s := writer.Services().Get(i)
		oldServices[string(s.FullName())] = s
	}

	for i := 0; i < reader.Services().Len(); i++ {
		newSvc := reader.Services().Get(i)
		name := string(newSvc.FullName())

		if oldSvc, exists := oldServices[name]; exists {
			c.checkServiceCompatibility(newSvc, oldSvc, result)
			delete(oldServices, name)
		}
	}

	for name := range oldServices {
		result.AddMessage("Service '%s' was removed", name)
	}
}

// checkServiceCompatibility checks compatibility between two service descriptors.
func (c *Checker) checkServiceCompatibility(newSvc, oldSvc protoreflect.ServiceDescriptor, result *compatibility.Result) {
	svcName := string(newSvc.FullName())

	oldMethods := make(map[string]protoreflect.MethodDescriptor)
	for i := 0; i < oldSvc.Methods().Len(); i++ {
		m := oldSvc.Methods().Get(i)
		oldMethods[string(m.Name())] = m
	}

	for i := 0; i < newSvc.Methods().Len(); i++ {
		newMethod := newSvc.Methods().Get(i)
		name := string(newMethod.Name())

		if oldMethod, exists := oldMethods[name]; exists {
			// Check method compatibility
			if newMethod.Input().FullName() != oldMethod.Input().FullName() {
				result.AddMessage("Service '%s': method '%s' input type changed from '%s' to '%s'",
					svcName, name, oldMethod.Input().FullName(), newMethod.Input().FullName())
			}
			if newMethod.Output().FullName() != oldMethod.Output().FullName() {
				result.AddMessage("Service '%s': method '%s' output type changed from '%s' to '%s'",
					svcName, name, oldMethod.Output().FullName(), newMethod.Output().FullName())
			}
			if newMethod.IsStreamingClient() != oldMethod.IsStreamingClient() {
				result.AddMessage("Service '%s': method '%s' client streaming changed",
					svcName, name)
			}
			if newMethod.IsStreamingServer() != oldMethod.IsStreamingServer() {
				result.AddMessage("Service '%s': method '%s' server streaming changed",
					svcName, name)
			}
			delete(oldMethods, name)
		}
	}

	for name := range oldMethods {
		result.AddMessage("Service '%s': method '%s' was removed", svcName, name)
	}
}

// protoTypeName returns a human-readable type name for a field.
func protoTypeName(f protoreflect.FieldDescriptor) string {
	switch f.Kind() {
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.Int32Kind:
		return "int32"
	case protoreflect.Sint32Kind:
		return "sint32"
	case protoreflect.Uint32Kind:
		return "uint32"
	case protoreflect.Int64Kind:
		return "int64"
	case protoreflect.Sint64Kind:
		return "sint64"
	case protoreflect.Uint64Kind:
		return "uint64"
	case protoreflect.Sfixed32Kind:
		return "sfixed32"
	case protoreflect.Fixed32Kind:
		return "fixed32"
	case protoreflect.FloatKind:
		return "float"
	case protoreflect.Sfixed64Kind:
		return "sfixed64"
	case protoreflect.Fixed64Kind:
		return "fixed64"
	case protoreflect.DoubleKind:
		return "double"
	case protoreflect.StringKind:
		return "string"
	case protoreflect.BytesKind:
		return "bytes"
	case protoreflect.MessageKind:
		return string(f.Message().FullName())
	case protoreflect.EnumKind:
		return string(f.Enum().FullName())
	case protoreflect.GroupKind:
		return "group"
	default:
		return "unknown"
	}
}

// checkerWellKnownTypes returns proto definitions for commonly-used well-known types.
func checkerWellKnownTypes() map[string]string {
	return map[string]string{
		"google/protobuf/any.proto": `
syntax = "proto3";
package google.protobuf;
message Any {
  string type_url = 1;
  bytes value = 2;
}`,
		"google/protobuf/timestamp.proto": `
syntax = "proto3";
package google.protobuf;
message Timestamp {
  int64 seconds = 1;
  int32 nanos = 2;
}`,
		"google/protobuf/duration.proto": `
syntax = "proto3";
package google.protobuf;
message Duration {
  int64 seconds = 1;
  int32 nanos = 2;
}`,
		"google/protobuf/empty.proto": `
syntax = "proto3";
package google.protobuf;
message Empty {}`,
		"google/protobuf/wrappers.proto": `
syntax = "proto3";
package google.protobuf;
message DoubleValue { double value = 1; }
message FloatValue { float value = 1; }
message Int64Value { int64 value = 1; }
message UInt64Value { uint64 value = 1; }
message Int32Value { int32 value = 1; }
message UInt32Value { uint32 value = 1; }
message BoolValue { bool value = 1; }
message StringValue { string value = 1; }
message BytesValue { bytes value = 1; }`,
		"google/protobuf/struct.proto": `
syntax = "proto3";
package google.protobuf;
message Struct {
  map<string, Value> fields = 1;
}
message Value {
  oneof kind {
    NullValue null_value = 1;
    double number_value = 2;
    string string_value = 3;
    bool bool_value = 4;
    Struct struct_value = 5;
    ListValue list_value = 6;
  }
}
message ListValue {
  repeated Value values = 1;
}
enum NullValue {
  NULL_VALUE = 0;
}`,
		"google/protobuf/field_mask.proto": `
syntax = "proto3";
package google.protobuf;
message FieldMask {
  repeated string paths = 1;
}`,
	}
}

// Ensure Checker implements compatibility.SchemaChecker
var _ compatibility.SchemaChecker = (*Checker)(nil)
