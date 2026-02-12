package auth

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{"string slice", []string{"a", "b"}, 2},
		{"interface slice", []interface{}{"a", "b", "c"}, 3},
		{"interface slice with non-strings", []interface{}{"a", 42, "b"}, 2},
		{"single string", "admin", 1},
		{"nil", nil, 0},
		{"number", 42, 0},
		{"empty interface slice", []interface{}{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toStringSlice(tt.input)
			if len(result) != tt.expected {
				t.Errorf("expected %d elements, got %d: %v", tt.expected, len(result), result)
			}
		})
	}
}

func TestContainsAudience(t *testing.T) {
	tests := []struct {
		audiences []string
		required  string
		expected  bool
	}{
		{[]string{"api", "web"}, "api", true},
		{[]string{"api", "web"}, "mobile", false},
		{nil, "api", false},
		{[]string{}, "api", false},
		{[]string{"my-app"}, "my-app", true},
	}

	for _, tt := range tests {
		got := containsAudience(tt.audiences, tt.required)
		if got != tt.expected {
			t.Errorf("containsAudience(%v, %q) = %v, want %v", tt.audiences, tt.required, got, tt.expected)
		}
	}
}

func TestOIDCProvider_ExtractStringClaim(t *testing.T) {
	p := &OIDCProvider{config: config.OIDCConfig{}}

	claims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"realm_access": map[string]interface{}{
			"roles": []interface{}{"admin", "user"},
		},
		"nested": map[string]interface{}{
			"deep": map[string]interface{}{
				"value": "found",
			},
		},
		"number": 42,
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"sub", "user123"},
		{"email", "user@example.com"},
		{"nested.deep.value", "found"},
		{"nonexistent", ""},
		{"realm_access.roles", ""}, // not a string
		{"nested.nonexistent", ""}, // missing nested key
		{"number", ""},             // not a string
	}

	for _, tt := range tests {
		got := p.extractStringClaim(claims, tt.path)
		if got != tt.expected {
			t.Errorf("extractStringClaim(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestOIDCProvider_ExtractRolesClaim(t *testing.T) {
	tests := []struct {
		name       string
		rolesClaim string
		claims     map[string]interface{}
		expected   int
	}{
		{
			name:       "empty claim config",
			rolesClaim: "",
			claims:     map[string]interface{}{"roles": []interface{}{"admin"}},
			expected:   0,
		},
		{
			name:       "top-level roles",
			rolesClaim: "roles",
			claims:     map[string]interface{}{"roles": []interface{}{"admin", "user"}},
			expected:   2,
		},
		{
			name:       "nested roles",
			rolesClaim: "realm_access.roles",
			claims: map[string]interface{}{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{"admin"},
				},
			},
			expected: 1,
		},
		{
			name:       "missing claim",
			rolesClaim: "nonexistent",
			claims:     map[string]interface{}{},
			expected:   0,
		},
		{
			name:       "single string role",
			rolesClaim: "role",
			claims:     map[string]interface{}{"role": "admin"},
			expected:   1,
		},
		{
			name:       "nested missing",
			rolesClaim: "realm_access.missing",
			claims: map[string]interface{}{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{"admin"},
				},
			},
			expected: 0,
		},
		{
			name:       "nested path not object",
			rolesClaim: "sub.roles",
			claims:     map[string]interface{}{"sub": "user123"},
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &OIDCProvider{config: config.OIDCConfig{RolesClaim: tt.rolesClaim}}
			roles := p.extractRolesClaim(tt.claims)
			if len(roles) != tt.expected {
				t.Errorf("expected %d roles, got %d: %v", tt.expected, len(roles), roles)
			}
		})
	}
}

func TestOIDCProvider_DetermineRole(t *testing.T) {
	tests := []struct {
		name        string
		rolesClaim  string
		roleMapping map[string]string
		defaultRole string
		claims      map[string]interface{}
		expected    string
	}{
		{
			name:        "no role mapping",
			rolesClaim:  "roles",
			roleMapping: nil,
			defaultRole: "readonly",
			claims:      map[string]interface{}{"roles": []interface{}{"admin"}},
			expected:    "readonly",
		},
		{
			name:        "no roles in claims",
			rolesClaim:  "roles",
			roleMapping: map[string]string{"admin": "admin"},
			defaultRole: "readonly",
			claims:      map[string]interface{}{},
			expected:    "readonly",
		},
		{
			name:        "exact match",
			rolesClaim:  "roles",
			roleMapping: map[string]string{"schema-admin": "admin"},
			defaultRole: "readonly",
			claims:      map[string]interface{}{"roles": []interface{}{"schema-admin"}},
			expected:    "admin",
		},
		{
			name:        "case insensitive match",
			rolesClaim:  "roles",
			roleMapping: map[string]string{"Admin": "admin"},
			defaultRole: "readonly",
			claims:      map[string]interface{}{"roles": []interface{}{"admin"}},
			expected:    "admin",
		},
		{
			name:        "no match returns default",
			rolesClaim:  "roles",
			roleMapping: map[string]string{"super-admin": "admin"},
			defaultRole: "readonly",
			claims:      map[string]interface{}{"roles": []interface{}{"user"}},
			expected:    "readonly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &OIDCProvider{config: config.OIDCConfig{
				RolesClaim:  tt.rolesClaim,
				RoleMapping: tt.roleMapping,
				DefaultRole: tt.defaultRole,
			}}
			got := p.determineRole(tt.claims)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
