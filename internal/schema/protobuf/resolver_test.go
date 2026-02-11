package protobuf

import (
	"io"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestNewReferenceResolver(t *testing.T) {
	r := newReferenceResolver()
	if r == nil {
		t.Fatal("expected non-nil resolver")
	}
	if len(r.refs) != 0 {
		t.Errorf("expected empty refs, got %d", len(r.refs))
	}
	if len(r.wellKnown) == 0 {
		t.Error("expected well-known types to be populated")
	}
}

func TestGetWellKnownTypes(t *testing.T) {
	types := getWellKnownTypes()

	expectedKeys := []string{
		"google/protobuf/any.proto",
		"google/protobuf/timestamp.proto",
		"google/protobuf/duration.proto",
		"google/protobuf/empty.proto",
		"google/protobuf/struct.proto",
		"google/protobuf/wrappers.proto",
		"google/protobuf/field_mask.proto",
		"google/protobuf/descriptor.proto",
	}

	for _, key := range expectedKeys {
		if _, ok := types[key]; !ok {
			t.Errorf("expected well-known type %q", key)
		}
	}
}

func TestWithReferencesAndSchema(t *testing.T) {
	r := newReferenceResolver()

	schema := `syntax = "proto3"; message Test { string name = 1; }`
	refs := []storage.Reference{
		{Name: "common.proto", Subject: "common", Version: 1, Schema: `syntax = "proto3"; message Common { int32 id = 1; }`},
		{Name: "other.proto", Subject: "other", Version: 1, Schema: `syntax = "proto3"; message Other { string val = 1; }`},
	}

	resolver := r.withReferencesAndSchema(schema, refs)
	if resolver == nil {
		t.Fatal("expected non-nil resolver")
	}

	// Should find the main schema
	result, err := resolver.FindFileByPath("schema.proto")
	if err != nil {
		t.Fatalf("unexpected error finding schema.proto: %v", err)
	}
	content, _ := io.ReadAll(result.Source)
	if string(content) != schema {
		t.Errorf("expected main schema content")
	}

	// Should find references
	result, err = resolver.FindFileByPath("common.proto")
	if err != nil {
		t.Fatalf("unexpected error finding common.proto: %v", err)
	}
	content, _ = io.ReadAll(result.Source)
	if len(content) == 0 {
		t.Error("expected reference content")
	}
}

func TestWithReferencesAndSchema_EmptyRefs(t *testing.T) {
	r := newReferenceResolver()

	schema := `syntax = "proto3"; message Test { string name = 1; }`
	resolver := r.withReferencesAndSchema(schema, nil)

	// Main schema should still be accessible
	result, err := resolver.FindFileByPath("schema.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, _ := io.ReadAll(result.Source)
	if string(content) != schema {
		t.Error("expected schema content")
	}
}

func TestWithReferencesAndSchema_SkipsEmptyName(t *testing.T) {
	r := newReferenceResolver()

	refs := []storage.Reference{
		{Name: "", Subject: "s", Version: 1, Schema: "content"},
		{Name: "valid.proto", Subject: "s", Version: 1, Schema: "valid content"},
	}

	resolver := r.withReferencesAndSchema("main", refs)

	// Empty name ref should be skipped
	_, err := resolver.FindFileByPath("")
	if err == nil {
		t.Error("expected error for empty path (empty name not added)")
	}

	// Valid ref should be found
	result, err := resolver.FindFileByPath("valid.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, _ := io.ReadAll(result.Source)
	if string(content) != "valid content" {
		t.Error("expected valid content")
	}
}

func TestFindFileByPath_WellKnown(t *testing.T) {
	r := newReferenceResolver()

	wellKnownPaths := []string{
		"google/protobuf/timestamp.proto",
		"google/protobuf/any.proto",
		"google/protobuf/empty.proto",
	}

	for _, path := range wellKnownPaths {
		result, err := r.FindFileByPath(path)
		if err != nil {
			t.Errorf("unexpected error for %s: %v", path, err)
			continue
		}
		content, _ := io.ReadAll(result.Source)
		if len(content) == 0 {
			t.Errorf("expected content for %s", path)
		}
	}
}

func TestFindFileByPath_NotFound(t *testing.T) {
	r := newReferenceResolver()

	_, err := r.FindFileByPath("nonexistent.proto")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if err.Error() != "file not found: nonexistent.proto" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFindFileByPath_EmptyContent(t *testing.T) {
	r := newReferenceResolver()
	// Add a ref with empty content
	r.refs["empty.proto"] = ""

	_, err := r.FindFileByPath("empty.proto")
	if err == nil {
		t.Error("expected error for empty content ref")
	}
}

func TestFindFileByPath_WellKnownPriority(t *testing.T) {
	r := newReferenceResolver()
	// Override a well-known type with a custom ref
	r.refs["google/protobuf/timestamp.proto"] = "custom content"

	// Well-known types take priority
	result, err := r.FindFileByPath("google/protobuf/timestamp.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, _ := io.ReadAll(result.Source)
	// Should get well-known content, not custom
	if string(content) == "custom content" {
		t.Error("expected well-known type to take priority over refs")
	}
}

func TestFileNotFoundError(t *testing.T) {
	err := &fileNotFoundError{path: "test.proto"}
	expected := "file not found: test.proto"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestReferenceResolver_InterfaceCompliance(t *testing.T) {
	// Verify the resolver implements protocompile.Resolver
	r := newReferenceResolver()
	resolver := r.withReferencesAndSchema("syntax = \"proto3\";", nil)
	if resolver == nil {
		t.Fatal("expected non-nil resolver")
	}
}
