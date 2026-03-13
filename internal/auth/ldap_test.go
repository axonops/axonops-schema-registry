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

	_, err = NewLDAPProvider(config.LDAPConfig{URL: "ldap://localhost", BindDN: "cn=admin,dc=example,dc=com"})
	if err == nil {
		t.Error("expected error for empty BindPassword")
	}
}

func TestNewLDAPProvider_Defaults(t *testing.T) {
	p, err := NewLDAPProvider(config.LDAPConfig{
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
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
		BindPassword:      "secret",
		UserSearchFilter:  "(uid=%s)",
		UsernameAttribute: "uid",
		EmailAttribute:    "email",
		GroupAttribute:    "groups",
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
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
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
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
		DefaultRole:  "readonly",
	})

	role := p.mapGroupsToRole([]string{"CN=SomeGroup,DC=example,DC=com"})
	if role != "readonly" {
		t.Errorf("expected default role 'readonly', got %s", role)
	}
}

func TestMapGroupsToRole_ExactDNMatch(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
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
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
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
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
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
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
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
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
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

func TestMergeGroups_NoDuplicates(t *testing.T) {
	a := []string{"CN=Group1,DC=example,DC=com", "CN=Group2,DC=example,DC=com"}
	b := []string{"CN=Group3,DC=example,DC=com"}
	result := mergeGroups(a, b)
	if len(result) != 3 {
		t.Errorf("expected 3 groups, got %d", len(result))
	}
}

func TestMergeGroups_DeduplicatesCaseInsensitive(t *testing.T) {
	a := []string{"CN=Admins,DC=example,DC=com"}
	b := []string{"cn=admins,dc=example,dc=com"}
	result := mergeGroups(a, b)
	if len(result) != 1 {
		t.Errorf("expected 1 group after dedup, got %d", len(result))
	}
	// Should keep the first occurrence
	if result[0] != "CN=Admins,DC=example,DC=com" {
		t.Errorf("expected first occurrence preserved, got %s", result[0])
	}
}

func TestMergeGroups_BothEmpty(t *testing.T) {
	result := mergeGroups(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 groups, got %d", len(result))
	}
}

func TestMergeGroups_OneEmpty(t *testing.T) {
	a := []string{"CN=Group1,DC=example,DC=com"}
	result := mergeGroups(a, nil)
	if len(result) != 1 {
		t.Errorf("expected 1 group, got %d", len(result))
	}
}

func TestMergeGroups_ExactDuplicates(t *testing.T) {
	a := []string{"CN=Admins,DC=example,DC=com"}
	b := []string{"CN=Admins,DC=example,DC=com"}
	result := mergeGroups(a, b)
	if len(result) != 1 {
		t.Errorf("expected 1 group after exact dedup, got %d", len(result))
	}
}

func TestLDAPProvider_IsSecure(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		startTLS bool
		want     bool
	}{
		{"ldaps is secure", "ldaps://ldap.example.com:636", false, true},
		{"starttls is secure", "ldap://ldap.example.com:389", true, true},
		{"plain ldap is insecure", "ldap://ldap.example.com:389", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewLDAPProvider(config.LDAPConfig{
				URL:          tt.url,
				BindDN:       "cn=admin,dc=example,dc=com",
				BindPassword: "secret",
				StartTLS:     tt.startTLS,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := p.IsSecure(); got != tt.want {
				t.Errorf("IsSecure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLDAPProvider_GetTLSConfig_ClientCert(t *testing.T) {
	// Use the test certs generated for BDD testing
	certFile := "../../tests/bdd/certs/ldap/client.pem"
	keyFile := "../../tests/bdd/certs/ldap/client-key.pem"
	caFile := "../../tests/bdd/certs/ldap/ca.pem"

	p, err := NewLDAPProvider(config.LDAPConfig{
		URL:            "ldaps://ldap.example.com:636",
		BindDN:         "cn=admin,dc=example,dc=com",
		BindPassword:   "secret",
		CACertFile:     caFile,
		ClientCertFile: certFile,
		ClientKeyFile:  keyFile,
	})
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	tlsConfig, err := p.getTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error getting TLS config: %v", err)
	}
	if tlsConfig.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
	if len(tlsConfig.Certificates) != 1 {
		t.Errorf("expected 1 client certificate, got %d", len(tlsConfig.Certificates))
	}
}

func TestLDAPProvider_GetTLSConfig_InvalidClientCert(t *testing.T) {
	p, err := NewLDAPProvider(config.LDAPConfig{
		URL:            "ldaps://ldap.example.com:636",
		BindDN:         "cn=admin,dc=example,dc=com",
		BindPassword:   "secret",
		ClientCertFile: "/nonexistent/client.pem",
		ClientKeyFile:  "/nonexistent/client-key.pem",
	})
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	_, err = p.getTLSConfig()
	if err == nil {
		t.Error("expected error for invalid client cert path")
	}
}

func TestMapGroupsToRole_EmptyGroups(t *testing.T) {
	p, _ := NewLDAPProvider(config.LDAPConfig{
		URL:          "ldap://localhost",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
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
