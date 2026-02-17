// Package protobuf provides Protobuf schema parsing.
package protobuf

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Parser implements schema.Parser for Protobuf schemas.
type Parser struct {
	resolver *referenceResolver
}

// NewParser creates a new Protobuf parser.
func NewParser() *Parser {
	return &Parser{
		resolver: newReferenceResolver(),
	}
}

// Type returns the schema type.
func (p *Parser) Type() storage.SchemaType {
	return storage.SchemaTypeProtobuf
}

// Parse parses and validates a Protobuf schema.
func (p *Parser) Parse(schemaStr string, refs []storage.Reference) (schema.ParsedSchema, error) {
	// Create a resolver with references and the schema content
	resolver := p.resolver.withReferencesAndSchema(schemaStr, refs)

	// Create compiler
	compiler := protocompile.Compiler{
		Resolver:       resolver,
		SourceInfoMode: protocompile.SourceInfoStandard,
	}

	// Compile to get the file descriptor
	ctx := context.Background()
	files, err := compiler.Compile(ctx, "schema.proto")
	if err != nil {
		return nil, fmt.Errorf("failed to compile protobuf: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files compiled")
	}

	fd := files[0]

	return &ParsedProtobuf{
		raw:        schemaStr,
		descriptor: fd,
		references: refs,
	}, nil
}

// ParsedProtobuf represents a parsed Protobuf schema.
type ParsedProtobuf struct {
	raw        string
	descriptor protoreflect.FileDescriptor
	references []storage.Reference
}

// Type returns the schema type.
func (p *ParsedProtobuf) Type() storage.SchemaType {
	return storage.SchemaTypeProtobuf
}

// CanonicalString returns the canonical form of the schema.
func (p *ParsedProtobuf) CanonicalString() string {
	return p.normalize()
}

// Fingerprint returns a unique fingerprint for the schema.
func (p *ParsedProtobuf) Fingerprint() string {
	// Normalize and hash the schema
	normalized := p.normalize()
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// RawSchema returns the underlying schema object.
func (p *ParsedProtobuf) RawSchema() interface{} {
	return p.descriptor
}

// Raw returns the original schema string.
func (p *ParsedProtobuf) Raw() string {
	return p.raw
}

// Descriptor returns the file descriptor.
func (p *ParsedProtobuf) Descriptor() protoreflect.FileDescriptor {
	return p.descriptor
}

// Normalize returns a normalized copy of this schema.
func (p *ParsedProtobuf) Normalize() schema.ParsedSchema {
	return &ParsedProtobuf{
		raw:        p.normalize(),
		descriptor: p.descriptor,
		references: p.references,
	}
}

// HasTopLevelField reports whether any top-level message in the Protobuf
// schema contains a field with the given name.
func (p *ParsedProtobuf) HasTopLevelField(field string) bool {
	if p.descriptor == nil {
		return false
	}
	msgs := p.descriptor.Messages()
	for i := 0; i < msgs.Len(); i++ {
		fields := msgs.Get(i).Fields()
		for j := 0; j < fields.Len(); j++ {
			if string(fields.Get(j).Name()) == field {
				return true
			}
		}
	}
	return false
}

// FormattedString returns the schema in the requested format.
// Supported formats: "serialized" (base64-encoded FileDescriptorProto), "default" (canonical).
func (p *ParsedProtobuf) FormattedString(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "serialized":
		fdp := toFileDescriptorProto(p.descriptor)
		data, err := proto.Marshal(fdp)
		if err != nil {
			return p.normalize()
		}
		return base64.StdEncoding.EncodeToString(data)
	default:
		return p.normalize()
	}
}

// toFileDescriptorProto converts a protoreflect.FileDescriptor to a descriptorpb.FileDescriptorProto.
func toFileDescriptorProto(fd protoreflect.FileDescriptor) *descriptorpb.FileDescriptorProto {
	fdp := &descriptorpb.FileDescriptorProto{}
	name := fd.Path()
	fdp.Name = &name
	if fd.Package() != "" {
		pkg := string(fd.Package())
		fdp.Package = &pkg
	}
	syntax := "proto3"
	if fd.Syntax() == protoreflect.Proto2 {
		syntax = "proto2"
	}
	fdp.Syntax = &syntax

	// Dependencies
	for i := 0; i < fd.Imports().Len(); i++ {
		fdp.Dependency = append(fdp.Dependency, fd.Imports().Get(i).Path())
	}

	// Messages
	for i := 0; i < fd.Messages().Len(); i++ {
		fdp.MessageType = append(fdp.MessageType, messageToProto(fd.Messages().Get(i)))
	}

	// Enums
	for i := 0; i < fd.Enums().Len(); i++ {
		fdp.EnumType = append(fdp.EnumType, enumToProto(fd.Enums().Get(i)))
	}

	// Services
	for i := 0; i < fd.Services().Len(); i++ {
		fdp.Service = append(fdp.Service, serviceToProto(fd.Services().Get(i)))
	}

	return fdp
}

func messageToProto(md protoreflect.MessageDescriptor) *descriptorpb.DescriptorProto {
	dp := &descriptorpb.DescriptorProto{}
	name := string(md.Name())
	dp.Name = &name

	for i := 0; i < md.Fields().Len(); i++ {
		dp.Field = append(dp.Field, fieldToProto(md.Fields().Get(i)))
	}
	for i := 0; i < md.Oneofs().Len(); i++ {
		oo := md.Oneofs().Get(i)
		ooName := string(oo.Name())
		dp.OneofDecl = append(dp.OneofDecl, &descriptorpb.OneofDescriptorProto{Name: &ooName})
	}
	for i := 0; i < md.Messages().Len(); i++ {
		dp.NestedType = append(dp.NestedType, messageToProto(md.Messages().Get(i)))
	}
	for i := 0; i < md.Enums().Len(); i++ {
		dp.EnumType = append(dp.EnumType, enumToProto(md.Enums().Get(i)))
	}
	return dp
}

func fieldToProto(fd protoreflect.FieldDescriptor) *descriptorpb.FieldDescriptorProto {
	fp := &descriptorpb.FieldDescriptorProto{}
	name := string(fd.Name())
	fp.Name = &name
	num := int32(fd.Number())
	fp.Number = &num
	fdType := descriptorpb.FieldDescriptorProto_Type(fd.Kind())
	fp.Type = &fdType
	label := descriptorpb.FieldDescriptorProto_Label(fd.Cardinality())
	fp.Label = &label
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.EnumKind {
		tn := string(fd.Message().FullName())
		if fd.Kind() == protoreflect.EnumKind {
			tn = string(fd.Enum().FullName())
		}
		fp.TypeName = &tn
	}
	if fd.ContainingOneof() != nil {
		idx := int32(fd.ContainingOneof().Index()) // #nosec G115 -- oneof index is always small
		fp.OneofIndex = &idx
	}
	return fp
}

func enumToProto(ed protoreflect.EnumDescriptor) *descriptorpb.EnumDescriptorProto {
	ep := &descriptorpb.EnumDescriptorProto{}
	name := string(ed.Name())
	ep.Name = &name
	for i := 0; i < ed.Values().Len(); i++ {
		v := ed.Values().Get(i)
		vName := string(v.Name())
		vNum := int32(v.Number())
		ep.Value = append(ep.Value, &descriptorpb.EnumValueDescriptorProto{
			Name:   &vName,
			Number: &vNum,
		})
	}
	return ep
}

func serviceToProto(sd protoreflect.ServiceDescriptor) *descriptorpb.ServiceDescriptorProto {
	sp := &descriptorpb.ServiceDescriptorProto{}
	name := string(sd.Name())
	sp.Name = &name
	for i := 0; i < sd.Methods().Len(); i++ {
		m := sd.Methods().Get(i)
		mName := string(m.Name())
		input := string(m.Input().FullName())
		output := string(m.Output().FullName())
		sp.Method = append(sp.Method, &descriptorpb.MethodDescriptorProto{
			Name:       &mName,
			InputType:  &input,
			OutputType: &output,
		})
	}
	return sp
}

// normalize returns a normalized form of the schema.
func (p *ParsedProtobuf) normalize() string {
	// Build normalized representation from descriptor
	var sb strings.Builder

	fd := p.descriptor

	// Package
	if fd.Package() != "" {
		sb.WriteString(fmt.Sprintf("package %s;\n", fd.Package()))
	}

	// Syntax
	if fd.Syntax() == protoreflect.Proto3 {
		sb.WriteString("syntax = \"proto3\";\n")
	} else {
		sb.WriteString("syntax = \"proto2\";\n")
	}

	// Messages (sorted by name)
	messages := make([]string, 0, fd.Messages().Len())
	for i := 0; i < fd.Messages().Len(); i++ {
		msg := fd.Messages().Get(i)
		messages = append(messages, normalizeMessage(msg, 0))
	}
	sort.Strings(messages)
	for _, m := range messages {
		sb.WriteString(m)
	}

	// Enums (sorted by name)
	enums := make([]string, 0, fd.Enums().Len())
	for i := 0; i < fd.Enums().Len(); i++ {
		enum := fd.Enums().Get(i)
		enums = append(enums, normalizeEnum(enum, 0))
	}
	sort.Strings(enums)
	for _, e := range enums {
		sb.WriteString(e)
	}

	// Services (sorted by name)
	services := make([]string, 0, fd.Services().Len())
	for i := 0; i < fd.Services().Len(); i++ {
		svc := fd.Services().Get(i)
		services = append(services, normalizeService(svc))
	}
	sort.Strings(services)
	for _, s := range services {
		sb.WriteString(s)
	}

	return sb.String()
}

// normalizeMessage normalizes a message descriptor.
func normalizeMessage(msg protoreflect.MessageDescriptor, indent int) string {
	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)

	sb.WriteString(fmt.Sprintf("%smessage %s {\n", prefix, msg.Name()))

	// Fields (sorted by number)
	type fieldInfo struct {
		number int
		text   string
	}
	fields := make([]fieldInfo, 0, msg.Fields().Len())
	for i := 0; i < msg.Fields().Len(); i++ {
		f := msg.Fields().Get(i)
		fields = append(fields, fieldInfo{
			number: int(f.Number()),
			text:   normalizeField(f, indent+1),
		})
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].number < fields[j].number
	})
	for _, f := range fields {
		sb.WriteString(f.text)
	}

	// Nested messages
	nested := make([]string, 0, msg.Messages().Len())
	for i := 0; i < msg.Messages().Len(); i++ {
		nm := msg.Messages().Get(i)
		// Skip map entry types
		if !nm.IsMapEntry() {
			nested = append(nested, normalizeMessage(nm, indent+1))
		}
	}
	sort.Strings(nested)
	for _, n := range nested {
		sb.WriteString(n)
	}

	// Nested enums
	enums := make([]string, 0, msg.Enums().Len())
	for i := 0; i < msg.Enums().Len(); i++ {
		e := msg.Enums().Get(i)
		enums = append(enums, normalizeEnum(e, indent+1))
	}
	sort.Strings(enums)
	for _, e := range enums {
		sb.WriteString(e)
	}

	// Oneofs
	oneofs := make([]string, 0, msg.Oneofs().Len())
	for i := 0; i < msg.Oneofs().Len(); i++ {
		o := msg.Oneofs().Get(i)
		// Skip synthetic oneofs (for optional fields in proto3)
		if !o.IsSynthetic() {
			oneofs = append(oneofs, normalizeOneof(o, indent+1))
		}
	}
	sort.Strings(oneofs)
	for _, o := range oneofs {
		sb.WriteString(o)
	}

	sb.WriteString(fmt.Sprintf("%s}\n", prefix))
	return sb.String()
}

// normalizeField normalizes a field descriptor.
func normalizeField(f protoreflect.FieldDescriptor, indent int) string {
	prefix := strings.Repeat("  ", indent)

	var label string
	if f.Cardinality() == protoreflect.Repeated {
		if f.IsMap() {
			// Map field
			keyType := protoTypeName(f.MapKey())
			valueType := protoTypeName(f.MapValue())
			return fmt.Sprintf("%smap<%s, %s> %s = %d;\n", prefix, keyType, valueType, f.Name(), f.Number())
		}
		label = "repeated "
	} else if f.Cardinality() == protoreflect.Optional && f.ParentFile().Syntax() == protoreflect.Proto2 {
		label = "optional "
	} else if f.Cardinality() == protoreflect.Required {
		label = "required "
	}

	typeName := protoTypeName(f)

	return fmt.Sprintf("%s%s%s %s = %d;\n", prefix, label, typeName, f.Name(), f.Number())
}

// protoTypeName returns the type name for a field.
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

// normalizeEnum normalizes an enum descriptor.
func normalizeEnum(e protoreflect.EnumDescriptor, indent int) string {
	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)

	sb.WriteString(fmt.Sprintf("%senum %s {\n", prefix, e.Name()))

	// Values (sorted by number)
	type valueInfo struct {
		number int
		text   string
	}
	values := make([]valueInfo, 0, e.Values().Len())
	for i := 0; i < e.Values().Len(); i++ {
		v := e.Values().Get(i)
		values = append(values, valueInfo{
			number: int(v.Number()),
			text:   fmt.Sprintf("%s  %s = %d;\n", prefix, v.Name(), v.Number()),
		})
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].number < values[j].number
	})
	for _, v := range values {
		sb.WriteString(v.text)
	}

	sb.WriteString(fmt.Sprintf("%s}\n", prefix))
	return sb.String()
}

// normalizeOneof normalizes a oneof descriptor.
func normalizeOneof(o protoreflect.OneofDescriptor, indent int) string {
	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)

	sb.WriteString(fmt.Sprintf("%soneof %s {\n", prefix, o.Name()))

	// Fields (sorted by number)
	type fieldInfo struct {
		number int
		text   string
	}
	fields := make([]fieldInfo, 0, o.Fields().Len())
	for i := 0; i < o.Fields().Len(); i++ {
		f := o.Fields().Get(i)
		typeName := protoTypeName(f)
		fields = append(fields, fieldInfo{
			number: int(f.Number()),
			text:   fmt.Sprintf("%s  %s %s = %d;\n", prefix, typeName, f.Name(), f.Number()),
		})
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].number < fields[j].number
	})
	for _, f := range fields {
		sb.WriteString(f.text)
	}

	sb.WriteString(fmt.Sprintf("%s}\n", prefix))
	return sb.String()
}

// normalizeService normalizes a service descriptor.
func normalizeService(s protoreflect.ServiceDescriptor) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("service %s {\n", s.Name()))

	// Methods (sorted by name)
	methods := make([]string, 0, s.Methods().Len())
	for i := 0; i < s.Methods().Len(); i++ {
		m := s.Methods().Get(i)
		inputStream := ""
		outputStream := ""
		if m.IsStreamingClient() {
			inputStream = "stream "
		}
		if m.IsStreamingServer() {
			outputStream = "stream "
		}
		methods = append(methods, fmt.Sprintf("  rpc %s (%s%s) returns (%s%s);\n",
			m.Name(), inputStream, m.Input().FullName(), outputStream, m.Output().FullName()))
	}
	sort.Strings(methods)
	for _, m := range methods {
		sb.WriteString(m)
	}

	sb.WriteString("}\n")
	return sb.String()
}

// Ensure Parser implements schema.Parser
var _ schema.Parser = (*Parser)(nil)
