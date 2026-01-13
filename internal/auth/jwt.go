// Package auth provides authentication and authorization for the schema registry.
package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// JWTProvider handles JWT authentication.
type JWTProvider struct {
	config    config.JWTConfig
	publicKey any // *rsa.PublicKey, *ecdsa.PublicKey, or []byte for HMAC
	mu        sync.RWMutex

	// JWKS support
	jwksKeys      map[string]any // kid -> public key
	jwksLastFetch time.Time
	jwksCacheTTL  time.Duration
	httpClient    *http.Client
}

// NewJWTProvider creates a new JWT authentication provider.
func NewJWTProvider(cfg config.JWTConfig) (*JWTProvider, error) {
	p := &JWTProvider{
		config:       cfg,
		jwksKeys:     make(map[string]any),
		jwksCacheTTL: 5 * time.Minute, // Default JWKS cache TTL
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Load the public key if specified
	if cfg.PublicKeyFile != "" {
		if err := p.loadPublicKey(cfg.PublicKeyFile, cfg.Algorithm); err != nil {
			return nil, fmt.Errorf("failed to load public key: %w", err)
		}
	}

	// Load JWKS if URL is specified
	if cfg.JWKSURL != "" {
		if err := p.refreshJWKS(); err != nil {
			return nil, fmt.Errorf("failed to load JWKS: %w", err)
		}
	}

	return p, nil
}

// loadPublicKey loads a public key from a PEM file.
func (p *JWTProvider) loadPublicKey(keyFile, algorithm string) error {
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return errors.New("failed to decode PEM block")
	}

	switch {
	case strings.HasPrefix(algorithm, "RS"):
		// RSA public key
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			// Try parsing as RSA public key directly
			pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
			if err != nil {
				return fmt.Errorf("failed to parse RSA public key: %w", err)
			}
		}
		rsaKey, ok := pub.(*rsa.PublicKey)
		if !ok {
			return errors.New("key is not an RSA public key")
		}
		p.publicKey = rsaKey

	case strings.HasPrefix(algorithm, "ES"):
		// ECDSA public key
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse ECDSA public key: %w", err)
		}
		p.publicKey = pub

	case strings.HasPrefix(algorithm, "HS"):
		// HMAC secret (raw bytes)
		p.publicKey = keyData

	default:
		return fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	return nil
}

// jwksResponse represents the JWKS endpoint response.
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

// jwkKey represents a single JWK in the key set.
type jwkKey struct {
	Kty string `json:"kty"` // Key type (RSA, EC)
	Use string `json:"use"` // Key use (sig, enc)
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm
	N   string `json:"n"`   // RSA modulus
	E   string `json:"e"`   // RSA exponent
	Crv string `json:"crv"` // EC curve
	X   string `json:"x"`   // EC x coordinate
	Y   string `json:"y"`   // EC y coordinate
}

// refreshJWKS fetches and parses the JWKS from the configured URL.
func (p *JWTProvider) refreshJWKS() error {
	resp, err := p.httpClient.Get(p.config.JWKSURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read JWKS response: %w", err)
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Build new key set (replaces old keys to evict removed/rotated keys)
	newKeys := make(map[string]any)
	for _, key := range jwks.Keys {
		if key.Use != "" && key.Use != "sig" {
			continue // Skip non-signature keys
		}

		pubKey, err := p.parseJWK(key)
		if err != nil {
			continue // Skip keys we can't parse
		}

		kid := key.Kid
		if kid == "" {
			kid = "default"
		}
		newKeys[kid] = pubKey
	}

	// Atomically replace the key set
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jwksKeys = newKeys
	p.jwksLastFetch = time.Now()
	return nil
}

// parseJWK parses a JWK into a Go public key.
func (p *JWTProvider) parseJWK(key jwkKey) (any, error) {
	switch key.Kty {
	case "RSA":
		return p.parseRSAJWK(key)
	case "EC":
		return p.parseECJWK(key)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", key.Kty)
	}
}

// parseRSAJWK parses an RSA JWK into an *rsa.PublicKey.
func (p *JWTProvider) parseRSAJWK(key jwkKey) (*rsa.PublicKey, error) {
	if key.N == "" || key.E == "" {
		return nil, errors.New("missing RSA key components")
	}

	// Decode modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// parseECJWK parses an EC JWK into an *ecdsa.PublicKey.
func (p *JWTProvider) parseECJWK(key jwkKey) (*ecdsa.PublicKey, error) {
	if key.X == "" || key.Y == "" || key.Crv == "" {
		return nil, errors.New("missing EC key components")
	}

	// Decode x coordinate
	xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode x coordinate: %w", err)
	}
	x := new(big.Int).SetBytes(xBytes)

	// Decode y coordinate
	yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode y coordinate: %w", err)
	}
	y := new(big.Int).SetBytes(yBytes)

	// Get the curve based on crv
	var curve elliptic.Curve
	switch key.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve: %s", key.Crv)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

// getKeyForToken returns the appropriate key for verifying a token.
// It uses the kid header if present, or falls back to the static key.
func (p *JWTProvider) getKeyForToken(token *jwt.Token) (any, error) {
	// If we have a static public key, use it
	if p.publicKey != nil {
		return p.publicKey, nil
	}

	// Check if JWKS is configured
	if p.config.JWKSURL == "" {
		return nil, errors.New("no key configured for JWT validation")
	}

	// Check if we need to refresh JWKS
	if time.Since(p.jwksLastFetch) > p.jwksCacheTTL {
		// Refresh in the background, but don't block if it fails
		go func() {
			p.refreshJWKS() //nolint:errcheck
		}()
	}

	// Get the key ID from the token header
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		kid = "default"
	}

	// Look up the key
	p.mu.RLock()
	defer p.mu.RUnlock()

	key, ok := p.jwksKeys[kid]
	if !ok {
		// If no matching key found and there's only one key, use it
		if len(p.jwksKeys) == 1 {
			for _, k := range p.jwksKeys {
				return k, nil
			}
		}
		return nil, fmt.Errorf("key %q not found in JWKS", kid)
	}

	return key, nil
}

// VerifyToken verifies a JWT token and returns the authenticated user.
func (p *JWTProvider) VerifyToken(ctx context.Context, rawToken string) (*User, bool) {
	// Build the key function that handles both static keys and JWKS
	keyFunc := func(token *jwt.Token) (any, error) {
		// Validate the signing method based on algorithm config
		alg := p.config.Algorithm
		if alg != "" {
			switch {
			case strings.HasPrefix(alg, "RS"):
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
			case strings.HasPrefix(alg, "ES"):
				if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
			case strings.HasPrefix(alg, "HS"):
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
			}
		}
		return p.getKeyForToken(token)
	}

	// Build parse options
	var parseOpts []jwt.ParserOption
	if p.config.Algorithm != "" {
		parseOpts = append(parseOpts, jwt.WithValidMethods([]string{p.config.Algorithm}))
	}

	// Parse and validate the token
	token, err := jwt.Parse(rawToken, keyFunc, parseOpts...)
	if err != nil {
		return nil, false
	}

	if !token.Valid {
		return nil, false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, false
	}

	// Validate issuer if configured
	if p.config.Issuer != "" {
		iss, _ := claims.GetIssuer()
		if iss != p.config.Issuer {
			return nil, false
		}
	}

	// Validate audience if configured
	if p.config.Audience != "" {
		aud, _ := claims.GetAudience()
		found := false
		for _, a := range aud {
			if a == p.config.Audience {
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
	}

	// Validate expiration
	exp, err := claims.GetExpirationTime()
	if err == nil && exp != nil {
		if time.Now().After(exp.Time) {
			return nil, false
		}
	}

	// Extract user information from claims
	username := p.extractClaim(claims, "sub")
	if username == "" {
		username = p.extractClaim(claims, "preferred_username")
	}
	if username == "" {
		username = p.extractClaim(claims, "email")
	}
	if username == "" {
		return nil, false
	}

	// Extract role from claims
	role := p.determineRole(claims)

	return &User{
		Username: username,
		Role:     role,
		Method:   "jwt",
	}, true
}

// extractClaim extracts a string claim from the token.
func (p *JWTProvider) extractClaim(claims jwt.MapClaims, key string) string {
	// Check if there's a mapping for this claim
	if mappedKey, ok := p.config.ClaimsMapping[key]; ok {
		key = mappedKey
	}

	if val, ok := claims[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// determineRole extracts the role from JWT claims.
func (p *JWTProvider) determineRole(claims jwt.MapClaims) string {
	// Check claims mapping for role
	roleKey := "role"
	if mappedKey, ok := p.config.ClaimsMapping["role"]; ok {
		roleKey = mappedKey
	}

	// Try to get role directly
	if role, ok := claims[roleKey]; ok {
		if str, ok := role.(string); ok {
			return str
		}
	}

	// Try roles array
	rolesKey := "roles"
	if mappedKey, ok := p.config.ClaimsMapping["roles"]; ok {
		rolesKey = mappedKey
	}

	if roles, ok := claims[rolesKey]; ok {
		if rolesArr, ok := roles.([]any); ok && len(rolesArr) > 0 {
			if str, ok := rolesArr[0].(string); ok {
				return str
			}
		}
	}

	// Default role
	return "readonly"
}
