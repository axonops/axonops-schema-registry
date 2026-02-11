package auth

import (
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

func TestNewLDAPProvider_Validation(t *testing.T) {
	_, err := NewLDAPProvider(config.LDAPConfig{})
	if err == nil {
		t.Error("expected error for empty URL")
	}

	_, err = NewLDAPProvider(config.LDAPConfig{URL: "ldap://localhost"})
	if err == nil {
		t.Error("expected error for empty BindDN")
	}
}

func TestNewLDAPProvider_Defaults(t *testing.T) {
	p, err := NewLDAPProvider(config.LDAPConfig{
		URL:    "ldap://localhost",
		BindDN: "cn=admin,dc=example,dc=com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.config.UserSearchFilter != "(sAMAccountName=%s)" {
		t.Errorf("expected default UserSearchFilter, got %s", p.config.UserSearchFilter)
	}
	if p.config.UsernameAttribute != "sAMAccountName" {
		t.Errorf("expected default UsernameAttribute, got %s", p.config.UsernameAttribute)
	}
	if p.config.EmailAttribute != "mail" {
		t.Errorf("expected default EmailAttribute, got %s", p.config.EmailAttribute)
	}
	if p.config.GroupAttribute != "memberOf" {
		t.Errorf("expected default GroupAttribute, got %s", p.config.GroupAttribute)
	}
	if p.config.ConnectionTimeout != 10 {
		t.Errorf("expected default ConnectionTimeout=10, got %d", p.config.ConnectionTimeout)
	}
	if p.config.RequestTimeout != 30 {
		t.Errorf("expected default RequestTimeout=30, got %d", p.config.RequestTimeout)
	}
	if p.config.DefaultRole != "readonly" {
		t.Errorf("expected default role 'readonly', got %s", p.config.DefaultRole)
	}
}

func TestNewLDAPProvider_CustomDefaults(t *testing.T) {
	p, err := NewLDAPProvider(config.LDAPConfig{
		URL:               "ldap://localhost",
		BindDN:            "cn=admin,dc=example,dc=com",
		UserSearchFilter:  "(uid=%s)",
		UsernameAttribute: "uid",
		EmailAttribute:    "email",
		GroupAttribute:     "groups",
		ConnectionTimeout: 5,
		RequestTimeout:    15,
		DefaultRole:       "admin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.config.UserSearchFilter != "(uid=%s)" {
		t.Errorf("expected custom filter, got %s", p.config.UserSearchFilter)
	}
	if p.config.ConnectionTimeout != 5 {
		t.Errorf("expected 5, got %d", p.config.ConnectionTimeout)
	}
	if p.config.DefaultRole != "admin" {
		t.Errorf("expected admin, got %s", p.config.DefaultRole)
	}
}

func TestNewLDAPProvider_Close(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:    "ldap://localhost",
		BindDN: "cn=admin,dc=example,dc=com",
	})
	if err := p.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtractCN(t *testing.T) {
	tests := []struct {
		dn       string
		expected string
	}{
		{"CN=Schema Admins,OU=Groups,DC=example,DC=com", "Schema Admins"},
		{"cn=readers,ou=groups,dc=example,dc=com", "readers"},
		{"CN=Test Group", "Test Group"},
		{"OU=Groups,DC=example,DC=com", ""},
		{"", ""},
		{"simple-string", ""},
		{"CN=,OU=Groups", ""},
	}

	for _, tt := range tests {
		got := extractCN(tt.dn)
		if got != tt.expected {
			t.Errorf("extractCN(%q) = %q, want %q", tt.dn, got, tt.expected)
		}
	}
}

func TestMapGroupsToRole_NoMapping(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:         "ldap://localhost",
		BindDN:      "cn=admin,dc=example,dc=com",
		DefaultRole: "readonly",
	})

	role := p.mapGroupsToRole([]string{"CN=SomeGroup,DC=example,DC=com"})
	if role != "readonly" {
		t.Errorf("expected default role 'readonly', got %s", role)
	}
}

func TestMapGroupsToRole_ExactDNMatch(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:    "ldap://localhost",
		BindDN: "cn=admin,dc=example,dc=com",
		RoleMapping: map[string]string{
			"CN=Admins,OU=Groups,DC=example,DC=com": "admin",
		},
		DefaultRole: "readonly",
	})

	role := p.mapGroupsToRole([]string{"CN=Admins,OU=Groups,DC=example,DC=com"})
	if role != "admin" {
		t.Errorf("expected admin, got %s", role)
	}
}

func TestMapGroupsToRole_CNMatch(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:    "ldap://localhost",
		BindDN: "cn=admin,dc=example,dc=com",
		RoleMapping: map[string]string{
			"Admins": "admin",
		},
		DefaultRole: "readonly",
	})

	// CN extracted from DN should match case-insensitively
	role := p.mapGroupsToRole([]string{"CN=Admins,OU=Groups,DC=example,DC=com"})
	if role != "admin" {
		t.Errorf("expected admin via CN match, got %s", role)
	}
}

func TestMapGroupsToRole_CaseInsensitiveCN(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:    "ldap://localhost",
		BindDN: "cn=admin,dc=example,dc=com",
		RoleMapping: map[string]string{
			"admins": "admin",
		},
		DefaultRole: "readonly",
	})

	role := p.mapGroupsToRole([]string{"CN=ADMINS,OU=Groups,DC=example,DC=com"})
	if role != "admin" {
		t.Errorf("expected admin via case-insensitive CN, got %s", role)
	}
}

func TestMapGroupsToRole_NoMatch(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:    "ldap://localhost",
		BindDN: "cn=admin,dc=example,dc=com",
		RoleMapping: map[string]string{
			"Admins": "admin",
		},
		DefaultRole: "readonly",
	})

	role := p.mapGroupsToRole([]string{"CN=Users,OU=Groups,DC=example,DC=com"})
	if role != "readonly" {
		t.Errorf("expected default 'readonly', got %s", role)
	}
}

func TestMapGroupsToRole_FirstMatchWins(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:    "ldap://localhost",
		BindDN: "cn=admin,dc=example,dc=com",
		RoleMapping: map[string]string{
			"CN=Admins,OU=Groups,DC=example,DC=com":  "admin",
			"CN=Writers,OU=Groups,DC=example,DC=com": "readwrite",
		},
		DefaultRole: "readonly",
	})

	// First group that matches should determine the role
	role := p.mapGroupsToRole([]string{
		"CN=Admins,OU=Groups,DC=example,DC=com",
		"CN=Writers,OU=Groups,DC=example,DC=com",
	})
	if role != "admin" {
		t.Errorf("expected admin (first match), got %s", role)
	}
}

func TestMapGroupsToRole_EmptyGroups(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:    "ldap://localhost",
		BindDN: "cn=admin,dc=example,dc=com",
		RoleMapping: map[string]string{
			"Admins": "admin",
		},
		DefaultRole: "readonly",
	})

	role := p.mapGroupsToRole(nil)
	if role != "readonly" {
		t.Errorf("expected default 'readonly' for empty groups, got %s", role)
	}
}
