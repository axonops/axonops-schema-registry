// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// ContextKey is used for storing auth info in context.
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey ContextKey = "auth_user"
	// RoleContextKey is the context key for the user's role.
	RoleContextKey ContextKey = "auth_role"
)

// User represents an authenticated user.
type User struct {
	Username string
	Role     string
	Method   string // basic, api_key, jwt, mtls
}

// Authenticator handles authentication.
type Authenticator struct {
	config    config.AuthConfig
	apiKeys   map[string]*APIKey // key -> APIKey
}

// APIKey represents an API key.
type APIKey struct {
	Key         string
	Name        string
	Username    string
	Role        string
	Description string
}

// NewAuthenticator creates a new authenticator.
func NewAuthenticator(cfg config.AuthConfig) *Authenticator {
	return &Authenticator{
		config:  cfg,
		apiKeys: make(map[string]*APIKey),
	}
}

// AddAPIKey adds an API key.
func (a *Authenticator) AddAPIKey(key *APIKey) {
	a.apiKeys[key.Key] = key
}

// Middleware returns HTTP middleware for authentication.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Try each enabled authentication method
		for _, method := range a.config.Methods {
			user, ok := a.authenticate(r, method)
			if ok {
				// Store user in context
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				ctx = context.WithValue(ctx, RoleContextKey, user.Role)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// No authentication succeeded
		a.unauthorized(w, r)
	})
}

// authenticate attempts authentication with a specific method.
func (a *Authenticator) authenticate(r *http.Request, method string) (*User, bool) {
	switch method {
	case "basic":
		return a.authenticateBasic(r)
	case "api_key":
		return a.authenticateAPIKey(r)
	case "jwt":
		return a.authenticateJWT(r)
	case "mtls":
		return a.authenticateMTLS(r)
	default:
		return nil, false
	}
}

// authenticateBasic handles HTTP Basic authentication.
func (a *Authenticator) authenticateBasic(r *http.Request) (*User, bool) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return nil, false
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return nil, false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, false
	}

	username, password := parts[0], parts[1]

	// Check against configured users
	if storedHash, ok := a.config.Basic.Users[username]; ok {
		if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err == nil {
			return &User{
				Username: username,
				Role:     a.config.RBAC.DefaultRole,
				Method:   "basic",
			}, true
		}
	}

	return nil, false
}

// authenticateAPIKey handles API key authentication.
func (a *Authenticator) authenticateAPIKey(r *http.Request) (*User, bool) {
	var key string

	// Check header
	if a.config.APIKey.Header != "" {
		key = r.Header.Get(a.config.APIKey.Header)
	}

	// Check query param if header not found
	if key == "" && a.config.APIKey.QueryParam != "" {
		key = r.URL.Query().Get(a.config.APIKey.QueryParam)
	}

	if key == "" {
		return nil, false
	}

	// Look up API key
	if apiKey, ok := a.apiKeys[key]; ok {
		return &User{
			Username: apiKey.Username,
			Role:     apiKey.Role,
			Method:   "api_key",
		}, true
	}

	return nil, false
}

// authenticateJWT handles JWT authentication.
func (a *Authenticator) authenticateJWT(r *http.Request) (*User, bool) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, false
	}

	token := auth[7:]
	if token == "" {
		return nil, false
	}

	// JWT validation would be implemented here
	// For now, return false as placeholder
	_ = token
	return nil, false
}

// authenticateMTLS handles mutual TLS authentication.
func (a *Authenticator) authenticateMTLS(r *http.Request) (*User, bool) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return nil, false
	}

	cert := r.TLS.PeerCertificates[0]
	username := cert.Subject.CommonName

	if username == "" {
		return nil, false
	}

	return &User{
		Username: username,
		Role:     a.config.RBAC.DefaultRole,
		Method:   "mtls",
	}, true
}

// unauthorized sends an authentication challenge.
func (a *Authenticator) unauthorized(w http.ResponseWriter, r *http.Request) {
	// Set appropriate WWW-Authenticate header
	for _, method := range a.config.Methods {
		switch method {
		case "basic":
			realm := a.config.Basic.Realm
			if realm == "" {
				realm = "Schema Registry"
			}
			w.Header().Add("WWW-Authenticate", `Basic realm="`+realm+`"`)
		case "api_key":
			w.Header().Add("WWW-Authenticate", "API-Key")
		case "jwt":
			w.Header().Add("WWW-Authenticate", "Bearer")
		}
	}

	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// GetUser retrieves the authenticated user from context.
func GetUser(ctx context.Context) *User {
	if user, ok := ctx.Value(UserContextKey).(*User); ok {
		return user
	}
	return nil
}

// GetRole retrieves the role from context.
func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(RoleContextKey).(string); ok {
		return role
	}
	return ""
}

// HashPassword creates a bcrypt hash of a password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ConstantTimeCompare performs a constant-time string comparison.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
