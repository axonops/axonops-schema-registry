// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// OIDCProvider handles OpenID Connect authentication.
type OIDCProvider struct {
	config   config.OIDCConfig
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
}

// NewOIDCProvider creates a new OIDC authentication provider.
func NewOIDCProvider(ctx context.Context, cfg config.OIDCConfig) (*OIDCProvider, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("OIDC issuer URL is required")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("OIDC client ID is required")
	}

	// Set defaults
	if cfg.UsernameClaim == "" {
		cfg.UsernameClaim = "sub"
	}
	if cfg.DefaultRole == "" {
		cfg.DefaultRole = "readonly"
	}

	// Create OIDC provider (fetches discovery document)
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Configure ID token verifier
	verifierConfig := &oidc.Config{
		ClientID:        cfg.ClientID,
		SkipIssuerCheck: cfg.SkipIssuerCheck,
		SkipExpiryCheck: cfg.SkipExpiryCheck,
	}

	// Set supported algorithms if specified
	if len(cfg.AllowedAlgorithms) > 0 {
		verifierConfig.SupportedSigningAlgs = cfg.AllowedAlgorithms
	}

	verifier := provider.Verifier(verifierConfig)

	return &OIDCProvider{
		config:   cfg,
		provider: provider,
		verifier: verifier,
	}, nil
}

// VerifyToken validates an OIDC/JWT token and returns the user if valid.
func (p *OIDCProvider) VerifyToken(ctx context.Context, rawToken string) (*User, bool) {
	if rawToken == "" {
		return nil, false
	}

	// Verify the token
	idToken, err := p.verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, false
	}

	// Validate audience if required
	if p.config.RequiredAudience != "" {
		if !containsAudience(idToken.Audience, p.config.RequiredAudience) {
			return nil, false
		}
	}

	// Extract claims
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, false
	}

	// Extract username
	username := p.extractStringClaim(claims, p.config.UsernameClaim)
	if username == "" {
		// Fall back to subject if configured claim is empty
		username = idToken.Subject
	}

	// Extract roles and determine role
	role := p.determineRole(claims)

	return &User{
		Username: username,
		Role:     role,
		Method:   "oidc",
	}, true
}

// extractStringClaim extracts a string value from claims.
// Supports nested claims using dot notation (e.g., "realm_access.roles").
func (p *OIDCProvider) extractStringClaim(claims map[string]interface{}, path string) string {
	parts := strings.Split(path, ".")
	current := claims

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return ""
		}

		// If this is the last part, try to get it as a string
		if i == len(parts)-1 {
			if str, ok := val.(string); ok {
				return str
			}
			return ""
		}

		// Otherwise, it should be a nested object
		if nested, ok := val.(map[string]interface{}); ok {
			current = nested
		} else {
			return ""
		}
	}

	return ""
}

// extractRolesClaim extracts roles/groups from claims.
// Returns a slice of role/group names.
func (p *OIDCProvider) extractRolesClaim(claims map[string]interface{}) []string {
	if p.config.RolesClaim == "" {
		return nil
	}

	parts := strings.Split(p.config.RolesClaim, ".")
	current := claims

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil
		}

		// If this is the last part, try to get it as an array
		if i == len(parts)-1 {
			return toStringSlice(val)
		}

		// Otherwise, it should be a nested object
		if nested, ok := val.(map[string]interface{}); ok {
			current = nested
		} else {
			return nil
		}
	}

	return nil
}

// toStringSlice converts an interface{} to []string.
func toStringSlice(val interface{}) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case string:
		// Single role as string
		return []string{v}
	default:
		return nil
	}
}

// determineRole maps OIDC roles/groups to a registry role.
func (p *OIDCProvider) determineRole(claims map[string]interface{}) string {
	roles := p.extractRolesClaim(claims)

	if p.config.RoleMapping == nil || len(roles) == 0 {
		return p.config.DefaultRole
	}

	// Check each role against role mappings
	for _, role := range roles {
		// Try exact match
		if mappedRole, ok := p.config.RoleMapping[role]; ok {
			return mappedRole
		}
		// Try case-insensitive match
		for pattern, mappedRole := range p.config.RoleMapping {
			if strings.EqualFold(pattern, role) {
				return mappedRole
			}
		}
	}

	return p.config.DefaultRole
}

// containsAudience checks if the audience list contains the required audience.
func containsAudience(audiences []string, required string) bool {
	for _, aud := range audiences {
		if aud == required {
			return true
		}
	}
	return false
}
