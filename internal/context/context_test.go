package context

import (
	gocontext "context"
	"testing"
)

func TestResolveSubject_Plain(t *testing.T) {
	ctx, subject := ResolveSubject("my-subject")
	if ctx != DefaultContext {
		t.Errorf("expected default context %q, got %q", DefaultContext, ctx)
	}
	if subject != "my-subject" {
		t.Errorf("expected my-subject, got %s", subject)
	}
}

func TestResolveSubject_WithContext(t *testing.T) {
	ctx, subject := ResolveSubject(":.TestContext:orders-value")
	if ctx != ".TestContext" {
		t.Errorf("expected .TestContext, got %s", ctx)
	}
	if subject != "orders-value" {
		t.Errorf("expected orders-value, got %s", subject)
	}
}

func TestResolveSubject_DefaultContext(t *testing.T) {
	// A subject that starts with ":." but has no second colon is treated as plain
	ctx, subject := ResolveSubject(":.no-close")
	if ctx != DefaultContext {
		t.Errorf("expected default context for malformed, got %s", ctx)
	}
	if subject != ":.no-close" {
		t.Errorf("expected raw subject, got %s", subject)
	}
}

func TestResolveSubject_EmptySubjectAfterContext(t *testing.T) {
	// :.TestContext: with empty subject → context-level operation
	ctx, subject := ResolveSubject(":.TestContext:")
	if ctx != ".TestContext" {
		t.Errorf("expected .TestContext for context-level op, got %s", ctx)
	}
	if subject != "" {
		t.Errorf("expected empty subject for context-level op, got %s", subject)
	}
}

func TestResolveSubject_GlobalContext(t *testing.T) {
	ctx, subject := ResolveSubject(":.__GLOBAL:")
	if ctx != ".__GLOBAL" {
		t.Errorf("expected .__GLOBAL, got %s", ctx)
	}
	if subject != "" {
		t.Errorf("expected empty subject, got %s", subject)
	}
}

func TestResolveSubject_GlobalContextWithSubject(t *testing.T) {
	ctx, subject := ResolveSubject(":.__GLOBAL:some-subject")
	if ctx != ".__GLOBAL" {
		t.Errorf("expected .__GLOBAL, got %s", ctx)
	}
	if subject != "some-subject" {
		t.Errorf("expected some-subject, got %s", subject)
	}
}

func TestResolveSubject_PlainColons(t *testing.T) {
	// Subject with colons that doesn't start with ":." should be plain
	ctx, subject := ResolveSubject("foo:bar:baz")
	if ctx != DefaultContext {
		t.Errorf("expected default context, got %s", ctx)
	}
	if subject != "foo:bar:baz" {
		t.Errorf("expected foo:bar:baz, got %s", subject)
	}
}

func TestFormatSubject_DefaultContext(t *testing.T) {
	result := FormatSubject(DefaultContext, "my-subject")
	if result != "my-subject" {
		t.Errorf("expected plain subject, got %s", result)
	}
}

func TestFormatSubject_EmptyContext(t *testing.T) {
	result := FormatSubject("", "my-subject")
	if result != "my-subject" {
		t.Errorf("expected plain subject, got %s", result)
	}
}

func TestFormatSubject_NamedContext(t *testing.T) {
	result := FormatSubject(".TestContext", "orders-value")
	if result != ":.TestContext:orders-value" {
		t.Errorf("expected ':.TestContext:orders-value', got %s", result)
	}
}

func TestFormatSubject_RoundTrip(t *testing.T) {
	original := ":.production:my-topic"
	ctx, subject := ResolveSubject(original)
	formatted := FormatSubject(ctx, subject)
	if formatted != original {
		t.Errorf("round-trip failed: %q -> (%q, %q) -> %q", original, ctx, subject, formatted)
	}
}

func TestIsValidContextName(t *testing.T) {
	valid := []string{".", ".test", ".my-ctx", ".my_ctx", ".ctx.v1", ".ABC", ".a1b2", "a", "test-1"}
	for _, name := range valid {
		if !IsValidContextName(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}

	invalid := []string{"", "has space", "has/slash", "has@at", "has!bang"}
	for _, name := range invalid {
		if IsValidContextName(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}

func TestIsValidContextName_TooLong(t *testing.T) {
	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}
	if IsValidContextName(string(long)) {
		t.Error("expected too-long name to be invalid")
	}
}

func TestNormalizeContextName_Default(t *testing.T) {
	if NormalizeContextName("") != DefaultContext {
		t.Errorf("expected default for empty")
	}
	if NormalizeContextName(":.:") != DefaultContext {
		t.Errorf("expected default for :.:")
	}
}

func TestNormalizeContextName_PrependDot(t *testing.T) {
	if NormalizeContextName("TestContext") != ".TestContext" {
		t.Errorf("expected .TestContext")
	}
}

func TestNormalizeContextName_AlreadyDotted(t *testing.T) {
	if NormalizeContextName(".TestContext") != ".TestContext" {
		t.Errorf("expected .TestContext unchanged")
	}
}

func TestWithRegistryContext_FromRequest(t *testing.T) {
	ctx := WithRegistryContext(gocontext.Background(), ".TestContext")
	got := RegistryContextFromRequest(ctx)
	if got != ".TestContext" {
		t.Errorf("expected .TestContext, got %s", got)
	}
}

func TestRegistryContextFromRequest_NotSet(t *testing.T) {
	got := RegistryContextFromRequest(gocontext.Background())
	if got != DefaultContext {
		t.Errorf("expected default context when not set, got %s", got)
	}
}

func TestRegistryContextFromRequest_EmptyString(t *testing.T) {
	ctx := WithRegistryContext(gocontext.Background(), "")
	got := RegistryContextFromRequest(ctx)
	if got != DefaultContext {
		t.Errorf("expected default context for empty string, got %s", got)
	}
}

func TestGlobalContextConstant(t *testing.T) {
	if GlobalContext != ".__GLOBAL" {
		t.Errorf("expected .__GLOBAL, got %s", GlobalContext)
	}
}

func TestIsGlobalContext(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"global context", ".__GLOBAL", true},
		{"default context", ".", false},
		{"named context", ".myctx", false},
		{"empty", "", false},
		{"similar name", ".__GLOBAL2", false},
		{"no dot prefix", "__GLOBAL", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGlobalContext(tt.input); got != tt.expect {
				t.Errorf("IsGlobalContext(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestIsValidContextName_GlobalContext(t *testing.T) {
	// __GLOBAL should be a valid context name (it uses alphanumeric, underscore, dot)
	if !IsValidContextName(GlobalContext) {
		t.Errorf("expected GlobalContext %q to be a valid context name", GlobalContext)
	}
}
