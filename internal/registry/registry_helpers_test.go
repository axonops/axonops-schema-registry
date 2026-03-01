package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsonschemacompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protobufcompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

// setupHelperTestRegistry creates a test registry with all parsers and compat checkers.
func setupHelperTestRegistry() *Registry {
	store := memory.NewStore()
	store.SetGlobalConfig(context.Background(), ".", &storage.ConfigRecord{CompatibilityLevel: "BACKWARD"})

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())
	schemaRegistry.Register(protobuf.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsonschemacompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protobufcompat.NewChecker())

	return New(store, schemaRegistry, compatChecker, "BACKWARD")
}

func TestCheckModeForWrite(t *testing.T) {
	ctx := context.Background()
	reg := setupHelperTestRegistry()

	// Default mode (READWRITE) should allow writes.
	mode, err := reg.CheckModeForWrite(ctx, ".", "test-subject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "" {
		t.Fatalf("expected empty mode (writes allowed), got %q", mode)
	}

	// Set global mode to READONLY — should block writes.
	err = reg.SetMode(ctx, ".", "", "READONLY", false)
	if err != nil {
		t.Fatalf("SetMode error: %v", err)
	}
	mode, err = reg.CheckModeForWrite(ctx, ".", "test-subject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "READONLY" {
		t.Fatalf("expected READONLY, got %q", mode)
	}

	// Set global mode to READONLY_OVERRIDE — should also block.
	err = reg.SetMode(ctx, ".", "", "READONLY_OVERRIDE", false)
	if err != nil {
		t.Fatalf("SetMode error: %v", err)
	}
	mode, err = reg.CheckModeForWrite(ctx, ".", "test-subject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "READONLY_OVERRIDE" {
		t.Fatalf("expected READONLY_OVERRIDE, got %q", mode)
	}

	// Set back to READWRITE — should allow again.
	err = reg.SetMode(ctx, ".", "", "READWRITE", false)
	if err != nil {
		t.Fatalf("SetMode error: %v", err)
	}
	mode, err = reg.CheckModeForWrite(ctx, ".", "test-subject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode != "" {
		t.Fatalf("expected empty mode (writes allowed), got %q", mode)
	}
}

func TestResolveAlias(t *testing.T) {
	ctx := context.Background()
	reg := setupHelperTestRegistry()

	// Empty subject returns empty.
	result := reg.ResolveAlias(ctx, ".", "")
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}

	// No alias configured — returns original subject.
	result = reg.ResolveAlias(ctx, ".", "my-subject")
	if result != "my-subject" {
		t.Fatalf("expected 'my-subject', got %q", result)
	}

	// Configure alias and verify resolution.
	err := reg.SetConfig(ctx, ".", "my-subject", "BACKWARD", nil, SetConfigOpts{
		Alias: "alias-target",
	})
	if err != nil {
		t.Fatalf("SetConfig error: %v", err)
	}
	result = reg.ResolveAlias(ctx, ".", "my-subject")
	if result != "alias-target" {
		t.Fatalf("expected 'alias-target', got %q", result)
	}

	// Subject without alias still returns itself.
	result = reg.ResolveAlias(ctx, ".", "other-subject")
	if result != "other-subject" {
		t.Fatalf("expected 'other-subject', got %q", result)
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"latest", -1, false},
		{"-1", -1, false},
		{"1", 1, false},
		{"100", 100, false},
		{"0", 0, true},
		{"-2", 0, true},
		{"abc", 0, true},
		{"", 0, true},
		{"1.5", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got %d", tt.input, got)
				}
				if !errors.Is(err, storage.ErrInvalidVersion) {
					t.Fatalf("expected storage.ErrInvalidVersion, got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for input %q: %v", tt.input, err)
				}
				if got != tt.want {
					t.Fatalf("ParseVersion(%q) = %d, want %d", tt.input, got, tt.want)
				}
			}
		})
	}
}
