// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/metrics"
)

// ContextKey is used for storing auth info in context.
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey ContextKey = "auth_user"
	// RoleContextKey is the context key for the user's role.
	RoleContextKey ContextKey = "auth_role"
	// UserIDContextKey is the context key for the user's database ID.
	UserIDContextKey ContextKey = "auth_user_id"
)

// User represents an authenticated user.
type User struct {
	ID       int64
	Username string
	Role     string
	Method   string // basic, api_key, jwt, oidc, mtls
}

// Authenticator handles authentication.
type Authenticator struct {
	config        config.AuthConfig
	service       *Service           // Database-backed auth service
	ldapProvider  *LDAPProvider      // LDAP authentication provider
	oidcProvider  *OIDCProvider      // OIDC authentication provider
	jwtProvider   *JWTProvider       // JWT authentication provider
	apiKeys       map[string]*APIKey // key -> APIKey (for legacy/config-based auth)
	memoryAPIKeys *MemoryAPIKeyStore // config-defined API keys (memory storage_type)
	htpasswdStore *HTPasswdStore     // htpasswd file entries (optional)
	metrics       *metrics.Metrics   // Prometheus metrics (optional)
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

// SetService sets the database-backed auth service.
func (a *Authenticator) SetService(svc *Service) {
	a.service = svc
}

// SetLDAPProvider sets the LDAP authentication provider.
func (a *Authenticator) SetLDAPProvider(p *LDAPProvider) {
	a.ldapProvider = p
}

// SetOIDCProvider sets the OIDC authentication provider.
func (a *Authenticator) SetOIDCProvider(p *OIDCProvider) {
	a.oidcProvider = p
}

// SetJWTProvider sets the JWT authentication provider.
func (a *Authenticator) SetJWTProvider(p *JWTProvider) {
	a.jwtProvider = p
}

// SetMetrics sets the Prometheus metrics instance for recording auth metrics.
func (a *Authenticator) SetMetrics(m *metrics.Metrics) {
	a.metrics = m
}

// SetHTPasswdStore sets the htpasswd file store for Basic authentication.
func (a *Authenticator) SetHTPasswdStore(store *HTPasswdStore) {
	a.htpasswdStore = store
}

// SetMemoryAPIKeyStore sets the config-defined API key store.
func (a *Authenticator) SetMemoryAPIKeyStore(store *MemoryAPIKeyStore) {
	a.memoryAPIKeys = store
}

// AddAPIKey adds an API key (for legacy/config-based auth).
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

		start := time.Now()

		// Try each enabled authentication method
		for _, method := range a.config.Methods {
			user, ok := a.authenticate(r, method)
			if ok {
				if a.metrics != nil {
					a.metrics.RecordAuthAttempt(user.Method, true, "", time.Since(start))
				}
				// Store user in context
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				ctx = context.WithValue(ctx, RoleContextKey, user.Role)
				if user.ID > 0 {
					ctx = context.WithValue(ctx, UserIDContextKey, user.ID)
				}

				// Propagate actor info to the audit middleware via the shared
				// AuditHints pointer (audit middleware runs before auth in the
				// chi middleware chain, so context-based communication is
				// one-directional — audit injects the pointer, auth fills it).
				if hints := GetAuditHints(r.Context()); hints != nil {
					hints.ActorID = user.Username
					hints.Role = user.Role
					hints.AuthMethod = user.Method
					hints.ActorType = actorTypeFromAuthMethod(user.Method)
				}

				// Record per-principal HTTP request metrics.
				if a.metrics != nil {
					rw := &principalResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
					wrappedNext := http.HandlerFunc(func(w2 http.ResponseWriter, r2 *http.Request) {
						next.ServeHTTP(w2, r2)
					})
					wrappedNext.ServeHTTP(rw, r.WithContext(ctx))
					a.metrics.RecordPrincipalRequest(user.Username, r.Method, normalizePrincipalPath(r.URL.Path), http.StatusText(rw.statusCode))
					return
				}

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// No authentication succeeded
		if a.metrics != nil {
			a.metrics.RecordAuthAttempt("unknown", false, "no_valid_credentials", time.Since(start))
		}
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
	case "oidc":
		return a.authenticateOIDC(r)
	case "mtls":
		return a.authenticateMTLS(r)
	default:
		return nil, false
	}
}

// authenticateBasic handles HTTP Basic authentication.
// It also supports API key authentication via Basic Auth format (key as username, any password)
// which is compatible with Confluent Schema Registry and Kafka producers/consumers.
// Authentication order: API key (username only) -> LDAP -> Database -> Config-based users
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

	// Try API key authentication first (username as API key, ignore password)
	// This is compatible with Confluent Schema Registry format: -u "API_KEY:API_SECRET"
	// where we only validate the API key (username), not the secret (password)
	if username != "" {
		if user, ok := a.authenticateAPIKeyValue(r, username); ok {
			return user, true
		}
	}

	// If password is empty and API key failed, reject
	// (prevents brute-force attempts with empty passwords)
	if password == "" {
		return nil, false
	}

	// Try LDAP authentication first if enabled
	if a.ldapProvider != nil {
		user, err := a.ldapProvider.Authenticate(r.Context(), username, password)
		if err == nil && user != nil {
			return user, true
		}
		// LDAP failed — check if fallback to other auth methods is allowed.
		// Default is true (allow fallback) for backward compatibility.
		if a.config.LDAP.AllowFallback != nil && !*a.config.LDAP.AllowFallback {
			return nil, false
		}
	}

	// Try database-backed authentication
	if a.service != nil {
		user, err := a.service.ValidateCredentials(r.Context(), username, password)
		if err == nil && user != nil {
			return &User{
				ID:       user.ID,
				Username: user.Username,
				Role:     user.Role,
				Method:   "basic",
			}, true
		}
	}

	// Fallback to configured users (legacy/config-based auth)
	if storedHash, ok := a.config.Basic.Users[username]; ok {
		if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err == nil {
			return &User{
				Username: username,
				Role:     a.config.RBAC.DefaultRole,
				Method:   "basic",
			}, true
		}
	}

	// Fallback to htpasswd file if configured
	if a.htpasswdStore != nil && a.htpasswdStore.Verify(username, password) {
		return &User{
			Username: username,
			Role:     a.config.RBAC.DefaultRole,
			Method:   "basic",
		}, true
	}

	return nil, false
}

// authenticateAPIKey handles API key authentication via header or query param.
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

	return a.authenticateAPIKeyValue(r, key)
}

// authenticateAPIKeyValue validates an API key value and returns the user.
// This is used by both authenticateAPIKey and authenticateBasic (for Kafka client compatibility).
func (a *Authenticator) authenticateAPIKeyValue(r *http.Request, key string) (*User, bool) {
	// Try database-backed authentication first
	if a.service != nil {
		apiKey, err := a.service.ValidateAPIKey(r.Context(), key)
		if err == nil && apiKey != nil {
			// If API key is associated with a user, get the username
			username := apiKey.Name
			if apiKey.UserID > 0 {
				user, err := a.service.GetUserByID(r.Context(), apiKey.UserID)
				if err == nil && user != nil {
					username = user.Username
				}
			}
			return &User{
				ID:       apiKey.ID,
				Username: username,
				Role:     apiKey.Role,
				Method:   "api_key",
			}, true
		}
	}

	// Try config-defined API keys (memory storage_type)
	if a.memoryAPIKeys != nil {
		if name, role, ok := a.memoryAPIKeys.Validate(key); ok {
			return &User{
				Username: name,
				Role:     role,
				Method:   "api_key",
			}, true
		}
	}

	// Fallback to in-memory API keys (legacy/config-based auth)
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
	if a.jwtProvider == nil {
		return nil, false
	}

	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, false
	}

	token := auth[7:]
	if token == "" {
		return nil, false
	}

	return a.jwtProvider.VerifyToken(r.Context(), token)
}

// authenticateOIDC handles OpenID Connect authentication.
func (a *Authenticator) authenticateOIDC(r *http.Request) (*User, bool) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, false
	}

	token := auth[7:]
	if token == "" {
		return nil, false
	}

	if a.oidcProvider == nil {
		return nil, false
	}

	return a.oidcProvider.VerifyToken(r.Context(), token)
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
		case "jwt", "oidc":
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

// GetUserID retrieves the user ID from context.
func GetUserID(ctx context.Context) int64 {
	if id, ok := ctx.Value(UserIDContextKey).(int64); ok {
		return id
	}
	return 0
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

// principalResponseWriter captures the status code for per-principal metrics.
type principalResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *principalResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Unwrap returns the underlying ResponseWriter so http.ResponseController
// can access optional interfaces like http.Flusher and http.Hijacker.
func (rw *principalResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// normalizePrincipalPath normalizes URL paths to reduce cardinality for per-principal metrics.
func normalizePrincipalPath(path string) string {
	// Strip /contexts/{context} prefix
	if strings.HasPrefix(path, "/contexts/") {
		rest := path[len("/contexts/"):]
		idx := strings.IndexByte(rest, '/')
		if idx >= 0 {
			path = rest[idx:]
		} else {
			return "/contexts"
		}
	}

	switch {
	case strings.HasPrefix(path, "/subjects/") && strings.Contains(path, "/versions"):
		return "/subjects/*/versions/*"
	case strings.HasPrefix(path, "/subjects/"):
		return "/subjects/*"
	case strings.HasPrefix(path, "/schemas/ids/"):
		return "/schemas/ids/*"
	case strings.HasPrefix(path, "/config/"):
		return "/config/*"
	case strings.HasPrefix(path, "/mode/"):
		return "/mode/*"
	case strings.HasPrefix(path, "/compatibility/"):
		return "/compatibility/*"
	case strings.HasPrefix(path, "/dek-registry/"):
		return "/dek-registry/*"
	case strings.HasPrefix(path, "/admin/"):
		return "/admin/*"
	default:
		return path
	}
}
