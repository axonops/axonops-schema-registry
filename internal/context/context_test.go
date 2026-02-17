package context

import (
	gocontext "context"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

func TestNewContextManager(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	if cm == nil {
		t.Fatal("expected non-nil manager")
	}

	// Should have default context
	contexts := cm.ListContexts()
	if len(contexts) != 1 {
		t.Errorf("expected 1 default context, got %d", len(contexts))
	}

	def := cm.GetDefaultContext()
	if def.Name != "." {
		t.Errorf("expected default context name '.', got %s", def.Name)
	}
	if def.Config.CompatibilityLevel != "BACKWARD" {
		t.Errorf("expected BACKWARD, got %s", def.Config.CompatibilityLevel)
	}
	if def.Config.Mode != "READWRITE" {
		t.Errorf("expected READWRITE, got %s", def.Config.Mode)
	}
}

func TestCreateContext(t *testing.T) {
	cm := NewContextManager(memory.NewStore())

	ctx := &Context{
		Name:        "tenant-1",
		Description: "First tenant",
		Config: &ContextConfig{
			CompatibilityLevel: "FULL",
			Mode:               "READWRITE",
		},
	}
	err := cm.CreateContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, err := cm.GetContext("tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Description != "First tenant" {
		t.Errorf("expected 'First tenant', got %s", found.Description)
	}
}

func TestCreateContext_EmptyName(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	err := cm.CreateContext(&Context{})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateContext_InvalidName(t *testing.T) {
	cm := NewContextManager(memory.NewStore())

	invalidNames := []string{"has space", "has/slash", "has@at"}
	for _, name := range invalidNames {
		err := cm.CreateContext(&Context{Name: name})
		if err == nil {
			t.Errorf("expected error for invalid name %q", name)
		}
	}
}

func TestCreateContext_ValidNames(t *testing.T) {
	cm := NewContextManager(memory.NewStore())

	validNames := []string{"tenant-1", "my_context", "prod.v2", "ABC123"}
	for _, name := range validNames {
		err := cm.CreateContext(&Context{Name: name})
		if err != nil {
			t.Errorf("expected valid name %q, got error: %v", name, err)
		}
	}
}

func TestCreateContext_Duplicate(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	cm.CreateContext(&Context{Name: "dup"})

	err := cm.CreateContext(&Context{Name: "dup"})
	if err == nil {
		t.Error("expected error for duplicate")
	}
}

func TestCreateContext_InheritsConfig(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	ctx := &Context{Name: "no-config"} // No explicit Config
	cm.CreateContext(ctx)

	found, _ := cm.GetContext("no-config")
	if found.Config == nil {
		t.Fatal("expected config to be inherited")
	}
	if found.Config.CompatibilityLevel != "BACKWARD" {
		t.Errorf("expected inherited BACKWARD, got %s", found.Config.CompatibilityLevel)
	}
}

func TestGetContext_NotFound(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	_, err := cm.GetContext("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestListContexts(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	cm.CreateContext(&Context{Name: "a"})
	cm.CreateContext(&Context{Name: "b"})

	list := cm.ListContexts()
	if len(list) != 3 { // default + a + b
		t.Errorf("expected 3 contexts, got %d", len(list))
	}
}

func TestUpdateContext(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	cm.CreateContext(&Context{Name: "ctx", Description: "old"})

	err := cm.UpdateContext(&Context{Name: "ctx", Description: "new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := cm.GetContext("ctx")
	if found.Description != "new" {
		t.Errorf("expected 'new', got %s", found.Description)
	}
}

func TestUpdateContext_NotFound(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	err := cm.UpdateContext(&Context{Name: "nonexistent"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestDeleteContext(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	cm.CreateContext(&Context{Name: "doomed"})

	err := cm.DeleteContext("doomed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = cm.GetContext("doomed")
	if err == nil {
		t.Error("expected not found after deletion")
	}
}

func TestDeleteContext_DefaultProtected(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	err := cm.DeleteContext(".")
	if err == nil {
		t.Error("expected error when deleting default context")
	}
}

func TestDeleteContext_NotFound(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	err := cm.DeleteContext("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestSetDefaultConfig(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	cm.SetDefaultConfig(&ContextConfig{
		CompatibilityLevel: "FULL",
		Mode:               "READONLY",
	})

	def := cm.GetDefaultContext()
	if def.Config.CompatibilityLevel != "FULL" {
		t.Errorf("expected FULL, got %s", def.Config.CompatibilityLevel)
	}
	if def.Config.Mode != "READONLY" {
		t.Errorf("expected READONLY, got %s", def.Config.Mode)
	}
}

func TestResolveSubject_Plain(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	ctxName, subject := cm.ResolveSubject("my-subject")
	if ctxName != "." {
		t.Errorf("expected default context, got %s", ctxName)
	}
	if subject != "my-subject" {
		t.Errorf("expected my-subject, got %s", subject)
	}
}

func TestResolveSubject_WithContext(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	ctxName, subject := cm.ResolveSubject(":.tenant-1.:orders-value")
	if ctxName != "tenant-1" {
		t.Errorf("expected tenant-1, got %s", ctxName)
	}
	if subject != "orders-value" {
		t.Errorf("expected orders-value, got %s", subject)
	}
}

func TestResolveSubject_MalformedPrefix(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	// Missing the closing ".:" — should treat as plain subject
	ctxName, subject := cm.ResolveSubject(":.tenant-1-no-close")
	if ctxName != "." {
		t.Errorf("expected default context for malformed, got %s", ctxName)
	}
	if subject != ":.tenant-1-no-close" {
		t.Errorf("expected raw subject, got %s", subject)
	}
}

func TestFormatSubject_DefaultContext(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	result := cm.FormatSubject(".", "my-subject")
	if result != "my-subject" {
		t.Errorf("expected plain subject, got %s", result)
	}
}

func TestFormatSubject_EmptyContext(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	result := cm.FormatSubject("", "my-subject")
	if result != "my-subject" {
		t.Errorf("expected plain subject, got %s", result)
	}
}

func TestFormatSubject_NamedContext(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	result := cm.FormatSubject("tenant-1", "orders-value")
	if result != ":.tenant-1.:orders-value" {
		t.Errorf("expected ':.tenant-1.:orders-value', got %s", result)
	}
}

func TestGetContextConfig(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	cm.CreateContext(&Context{
		Name: "custom",
		Config: &ContextConfig{
			CompatibilityLevel: "NONE",
			Mode:               "READONLY",
		},
	})

	cfg, err := cm.GetContextConfig("custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CompatibilityLevel != "NONE" {
		t.Errorf("expected NONE, got %s", cfg.CompatibilityLevel)
	}
}

func TestGetContextConfig_InheritsDefault(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	cm.CreateContext(&Context{Name: "inherit"})

	// Override Config to nil after creation to test inheritance path
	cm.contexts["inherit"].Config = nil

	cfg, err := cm.GetContextConfig("inherit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CompatibilityLevel != "BACKWARD" {
		t.Errorf("expected default BACKWARD, got %s", cfg.CompatibilityLevel)
	}
}

func TestGetContextConfig_NotFound(t *testing.T) {
	cm := NewContextManager(memory.NewStore())
	_, err := cm.GetContextConfig("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestWithContext_FromContext(t *testing.T) {
	schemaCtx := &Context{Name: "test-ctx"}
	ctx := WithContext(gocontext.Background(), schemaCtx)

	found := FromContext(ctx)
	if found == nil {
		t.Fatal("expected context from request context")
	}
	if found.Name != "test-ctx" {
		t.Errorf("expected test-ctx, got %s", found.Name)
	}
}

func TestFromContext_NotSet(t *testing.T) {
	found := FromContext(gocontext.Background())
	if found != nil {
		t.Error("expected nil when context not set")
	}
}

func TestIsValidContextName(t *testing.T) {
	valid := []string{"a", "test", "my-ctx", "my_ctx", "ctx.v1", "ABC", "a1b2"}
	for _, name := range valid {
		if !isValidContextName(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	invalid := []string{"", "has space", "has/slash", "has@at", "has!bang"}
	for _, name := range invalid {
		if isValidContextName(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}
