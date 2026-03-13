// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
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
	if cfg.BindPassword == "" {
		return nil, fmt.Errorf("LDAP bind password is required")
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

	// Warn if LDAP connection is not encrypted — credentials will be in plaintext.
	if !strings.HasPrefix(cfg.URL, "ldaps://") && !cfg.StartTLS {
		slog.Warn("LDAP configured without TLS — bind credentials and user passwords will be transmitted in plaintext",
			slog.String("url", cfg.URL),
			slog.String("recommendation", "use ldaps:// URL or enable start_tls"),
		)
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

	// Get user's groups from memberOf attribute
	groups := p.getUserGroups(userEntry)

	// If group search is configured, re-bind as service account and search for groups
	if p.config.GroupSearchBase != "" && p.config.GroupSearchFilter != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return nil, fmt.Errorf("failed to re-bind for group search: %w", err)
		}
		searchedGroups, err := p.searchGroups(conn, userDN)
		if err != nil {
			return nil, fmt.Errorf("group search failed: %w", err)
		}
		groups = mergeGroups(groups, searchedGroups)
	}

	role := p.mapGroupsToRole(groups)

	// Extract username and email from entry
	actualUsername := userEntry.GetAttributeValue(p.config.UsernameAttribute)
	if actualUsername == "" {
		actualUsername = username
	}

	return &User{
		Username: actualUsername,
		Role:     role,
		Method:   "ldap",
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

// IsSecure returns true if the LDAP connection uses TLS (LDAPS or StartTLS).
func (p *LDAPProvider) IsSecure() bool {
	return strings.HasPrefix(p.config.URL, "ldaps://") || p.config.StartTLS
}

// getTLSConfig returns TLS configuration for LDAP connection.
func (p *LDAPProvider) getTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: p.config.InsecureSkipVerify, // #nosec G402 -- user-configurable option for dev/test environments
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

	// Load client certificate for mTLS if provided
	if p.config.ClientCertFile != "" && p.config.ClientKeyFile != "" {
		clientCert, err := tls.LoadX509KeyPair(p.config.ClientCertFile, p.config.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load LDAP client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
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
		1, // Size limit: 1 result
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
			// Constant-time case-insensitive matching for CN
			for pattern, role := range p.config.RoleMapping {
				if constantTimeEqualFold(pattern, cn) || constantTimeEqualFold(pattern, group) {
					return role
				}
			}
		}
	}

	return p.config.DefaultRole
}

// searchGroups searches for groups that a user belongs to using an LDAP query.
// This is used when memberOf attribute is not available or when groups are stored
// in a separate subtree that requires an explicit search.
func (p *LDAPProvider) searchGroups(conn *ldap.Conn, userDN string) ([]string, error) {
	// Build search filter: replace %s with the user's DN
	filter := strings.ReplaceAll(p.config.GroupSearchFilter, "%s", ldap.EscapeFilter(userDN))

	searchRequest := ldap.NewSearchRequest(
		p.config.GroupSearchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, // No size limit
		p.config.RequestTimeout,
		false, // TypesOnly
		filter,
		[]string{"dn"},
		nil, // Controls
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	groups := make([]string, 0, len(result.Entries))
	for _, entry := range result.Entries {
		groups = append(groups, entry.DN)
	}
	return groups, nil
}

// mergeGroups merges two group lists, removing duplicates (case-insensitive).
func mergeGroups(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, g := range a {
		key := strings.ToLower(g)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, g)
		}
	}
	for _, g := range b {
		key := strings.ToLower(g)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, g)
		}
	}
	return result
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
