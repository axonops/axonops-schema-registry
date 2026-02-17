package schema

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// mockParser is a minimal Parser implementation for testing.
type mockParser struct {
	schemaType storage.SchemaType
}

func (p *mockParser) Parse(schemaStr string, references []storage.Reference) (ParsedSchema, error) {
	return nil, nil
}

func (p *mockParser) Type() storage.SchemaType {
	return p.schemaType
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	types := r.Types()
	if len(types) != 0 {
		t.Errorf("expected 0 types, got %d", len(types))
	}
}

func TestRegistry_Register_And_Get(t *testing.T) {
	r := NewRegistry()

	parser := &mockParser{schemaType: storage.SchemaTypeAvro}
	r.Register(parser)

	got, ok := r.Get(storage.SchemaTypeAvro)
	if !ok {
		t.Fatal("expected to find AVRO parser")
	}
	if got != parser {
		t.Error("expected same parser instance")
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get(storage.SchemaTypeAvro)
	if ok {
		t.Error("expected not found for unregistered type")
	}
}

func TestRegistry_Types(t *testing.T) {
	r := NewRegistry()

	r.Register(&mockParser{schemaType: storage.SchemaTypeAvro})
	r.Register(&mockParser{schemaType: storage.SchemaTypeJSON})
	r.Register(&mockParser{schemaType: storage.SchemaTypeProtobuf})

	types := r.Types()
	if len(types) != 3 {
		t.Errorf("expected 3 types, got %d", len(types))
	}

	typeSet := make(map[string]bool)
	for _, tp := range types {
		typeSet[tp] = true
	}
	for _, expected := range []string{"AVRO", "JSON", "PROTOBUF"} {
		if !typeSet[expected] {
			t.Errorf("expected type %s in list", expected)
		}
	}
}

func TestRegistry_Register_Overwrite(t *testing.T) {
	r := NewRegistry()

	parser1 := &mockParser{schemaType: storage.SchemaTypeAvro}
	parser2 := &mockParser{schemaType: storage.SchemaTypeAvro}

	r.Register(parser1)
	r.Register(parser2) // Overwrite

	got, ok := r.Get(storage.SchemaTypeAvro)
	if !ok {
		t.Fatal("expected to find parser")
	}
	if got != parser2 {
		t.Error("expected second parser to override first")
	}

	// Should still be only 1 type
	types := r.Types()
	if len(types) != 1 {
		t.Errorf("expected 1 type after overwrite, got %d", len(types))
	}
}
