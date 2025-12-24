// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// LDAPProvider handles LDAP authentication.
type LDAPProvider struct {
	config config.LDAPConfig
}

// NewLDAPProvider creates a new LDAP authentication provider.
func NewLDAPProvider(cfg config.LDAPConfig) (*LDAPProvider, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("LDAP URL is required")
	}
	if cfg.BindDN == "" {
		return nil, fmt.Errorf("LDAP bind DN is required")
	}
	if cfg.UserSearchFilter == "" {
		cfg.UserSearchFilter = "(sAMAccountName=%s)"
	}
	if cfg.UsernameAttribute == "" {
		cfg.UsernameAttribute = "sAMAccountName"
	}
	if cfg.EmailAttribute == "" {
		cfg.EmailAttribute = "mail"
	}
	if cfg.GroupAttribute == "" {
		cfg.GroupAttribute = "memberOf"
	}
	if cfg.ConnectionTimeout == 0 {
		cfg.ConnectionTimeout = 10
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 30
	}
	if cfg.DefaultRole == "" {
		cfg.DefaultRole = "readonly"
	}

	return &LDAPProvider{
		config: cfg,
	}, nil
}

// Authenticate validates user credentials against LDAP and returns the user if valid.
func (p *LDAPProvider) Authenticate(ctx context.Context, username, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	// Create connection with timeout
	conn, err := p.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer conn.Close()

	// Bind with service account to search for user
	if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
		return nil, fmt.Errorf("failed to bind with service account: %w", err)
	}

	// Search for user
	userEntry, err := p.searchUser(conn, username)
	if err != nil {
		return nil, fmt.Errorf("user search failed: %w", err)
	}
	if userEntry == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Re-bind with user's credentials to verify password
	userDN := userEntry.DN
	if err := conn.Bind(userDN, password); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Get user's groups and determine role
	groups := p.getUserGroups(userEntry)
	role := p.mapGroupsToRole(groups)

	// Extract username and email from entry
	actualUsername := userEntry.GetAttributeValue(p.config.UsernameAttribute)
	if actualUsername == "" {
		actualUsername = username
	}

	return &User{
		Username: actualUsername,
		Role:     role,
		Method:   "basic", // LDAP is used via basic auth
	}, nil
}

// connect establishes a connection to the LDAP server.
func (p *LDAPProvider) connect() (*ldap.Conn, error) {
	timeout := time.Duration(p.config.ConnectionTimeout) * time.Second

	var conn *ldap.Conn
	var err error

	// Check if using LDAPS (ldaps://)
	if strings.HasPrefix(p.config.URL, "ldaps://") {
		tlsConfig, tlsErr := p.getTLSConfig()
		if tlsErr != nil {
			return nil, tlsErr
		}
		conn, err = ldap.DialURL(p.config.URL, ldap.DialWithTLSConfig(tlsConfig))
	} else {
		conn, err = ldap.DialURL(p.config.URL)
	}

	if err != nil {
		return nil, err
	}

	// Set timeout for operations
	conn.SetTimeout(timeout)

	// Upgrade to TLS if StartTLS is enabled
	if p.config.StartTLS && !strings.HasPrefix(p.config.URL, "ldaps://") {
		tlsConfig, tlsErr := p.getTLSConfig()
		if tlsErr != nil {
			conn.Close()
			return nil, tlsErr
		}
		if err := conn.StartTLS(tlsConfig); err != nil {
			conn.Close()
			return nil, fmt.Errorf("StartTLS failed: %w", err)
		}
	}

	return conn, nil
}

// getTLSConfig returns TLS configuration for LDAP connection.
func (p *LDAPProvider) getTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: p.config.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	// Load CA certificate if provided
	if p.config.CACertFile != "" {
		caCert, err := os.ReadFile(p.config.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// searchUser searches for a user in LDAP by username.
func (p *LDAPProvider) searchUser(conn *ldap.Conn, username string) (*ldap.Entry, error) {
	// Determine search base
	searchBase := p.config.UserSearchBase
	if searchBase == "" {
		searchBase = p.config.BaseDN
	}

	// Build search filter with username
	filter := strings.ReplaceAll(p.config.UserSearchFilter, "%s", ldap.EscapeFilter(username))

	// Attributes to retrieve
	attributes := []string{
		"dn",
		p.config.UsernameAttribute,
		p.config.EmailAttribute,
		p.config.GroupAttribute,
	}

	searchRequest := ldap.NewSearchRequest(
		searchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1,    // Size limit: 1 result
		p.config.RequestTimeout,
		false, // TypesOnly
		filter,
		attributes,
		nil, // Controls
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if len(result.Entries) == 0 {
		return nil, nil
	}

	return result.Entries[0], nil
}

// getUserGroups extracts group memberships from a user entry.
func (p *LDAPProvider) getUserGroups(entry *ldap.Entry) []string {
	groups := entry.GetAttributeValues(p.config.GroupAttribute)
	return groups
}

// mapGroupsToRole maps LDAP groups to a registry role.
func (p *LDAPProvider) mapGroupsToRole(groups []string) string {
	if p.config.RoleMapping == nil {
		return p.config.DefaultRole
	}

	// Check each group against role mappings
	// Priority: first match wins, so order in config matters
	for _, group := range groups {
		// Try exact match first (full DN)
		if role, ok := p.config.RoleMapping[group]; ok {
			return role
		}

		// Try matching just the CN (common name)
		cn := extractCN(group)
		if cn != "" {
			// Case-insensitive matching for CN
			for pattern, role := range p.config.RoleMapping {
				if strings.EqualFold(pattern, cn) || strings.EqualFold(pattern, group) {
					return role
				}
			}
		}
	}

	return p.config.DefaultRole
}

// extractCN extracts the Common Name (CN) from a Distinguished Name (DN).
func extractCN(dn string) string {
	// Parse DN components
	parts := strings.Split(dn, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "cn=") {
			return part[3:] // Return value after "CN="
		}
	}
	return ""
}

// Close closes any resources held by the LDAP provider.
func (p *LDAPProvider) Close() error {
	// No persistent connections to close
	return nil
}
