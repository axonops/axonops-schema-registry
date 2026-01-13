package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/axonops/axonops-schema-registry/internal/config"
)

// generateTestRSAKey generates an RSA key pair for testing.
func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return key
}

// createTestToken creates a JWT token for testing.
func createTestToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims, method jwt.SigningMethod) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenStr
}

// writePublicKey writes an RSA public key to a PEM file.
func writePublicKey(path string, key *rsa.PublicKey) error {
	keyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return err
	}
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: keyBytes,
	}
	return os.WriteFile(path, pem.EncodeToMemory(pemBlock), 0600)
}

func TestJWTProvider_VerifyToken_Valid(t *testing.T) {
	key := generateTestRSAKey(t)

	// Write public key to temp file
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "public.pem")
	if err := writePublicKey(keyFile, &key.PublicKey); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	provider, err := NewJWTProvider(config.JWTConfig{
		Algorithm:     "RS256",
		PublicKeyFile: keyFile,
		Issuer:        "test-issuer",
		Audience:      "test-audience",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": "testuser",
		"iss": "test-issuer",
		"aud": []string{"test-audience"},
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := createTestToken(t, key, claims, jwt.SigningMethodRS256)
	user, ok := provider.VerifyToken(context.Background(), token)
	if !ok {
		t.Fatal("expected token to be valid")
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}
	if user.Method != "jwt" {
		t.Errorf("expected method 'jwt', got %q", user.Method)
	}
}

func TestJWTProvider_VerifyToken_ExpiredToken(t *testing.T) {
	key := generateTestRSAKey(t)

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "public.pem")
	if err := writePublicKey(keyFile, &key.PublicKey); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	provider, err := NewJWTProvider(config.JWTConfig{
		Algorithm:     "RS256",
		PublicKeyFile: keyFile,
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": "testuser",
		"exp": time.Now().Add(-time.Hour).Unix(), // Expired
	}

	token := createTestToken(t, key, claims, jwt.SigningMethodRS256)
	_, ok := provider.VerifyToken(context.Background(), token)
	if ok {
		t.Error("expected expired token to be rejected")
	}
}

func TestJWTProvider_VerifyToken_WrongIssuer(t *testing.T) {
	key := generateTestRSAKey(t)

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "public.pem")
	if err := writePublicKey(keyFile, &key.PublicKey); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	provider, err := NewJWTProvider(config.JWTConfig{
		Algorithm:     "RS256",
		PublicKeyFile: keyFile,
		Issuer:        "expected-issuer",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": "testuser",
		"iss": "wrong-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := createTestToken(t, key, claims, jwt.SigningMethodRS256)
	_, ok := provider.VerifyToken(context.Background(), token)
	if ok {
		t.Error("expected token with wrong issuer to be rejected")
	}
}

func TestJWTProvider_VerifyToken_WrongAudience(t *testing.T) {
	key := generateTestRSAKey(t)

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "public.pem")
	if err := writePublicKey(keyFile, &key.PublicKey); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	provider, err := NewJWTProvider(config.JWTConfig{
		Algorithm:     "RS256",
		PublicKeyFile: keyFile,
		Audience:      "expected-audience",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": "testuser",
		"aud": []string{"wrong-audience"},
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := createTestToken(t, key, claims, jwt.SigningMethodRS256)
	_, ok := provider.VerifyToken(context.Background(), token)
	if ok {
		t.Error("expected token with wrong audience to be rejected")
	}
}

func TestJWTProvider_VerifyToken_RoleExtraction(t *testing.T) {
	key := generateTestRSAKey(t)

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "public.pem")
	if err := writePublicKey(keyFile, &key.PublicKey); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	tests := []struct {
		name     string
		claims   jwt.MapClaims
		wantRole string
	}{
		{
			name: "role from role claim",
			claims: jwt.MapClaims{
				"sub":  "testuser",
				"role": "admin",
				"exp":  time.Now().Add(time.Hour).Unix(),
			},
			wantRole: "admin",
		},
		{
			name: "role from roles array",
			claims: jwt.MapClaims{
				"sub":   "testuser",
				"roles": []any{"developer", "reader"},
				"exp":   time.Now().Add(time.Hour).Unix(),
			},
			wantRole: "developer",
		},
		{
			name: "default role when missing",
			claims: jwt.MapClaims{
				"sub": "testuser",
				"exp": time.Now().Add(time.Hour).Unix(),
			},
			wantRole: "readonly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewJWTProvider(config.JWTConfig{
				Algorithm:     "RS256",
				PublicKeyFile: keyFile,
			})
			if err != nil {
				t.Fatalf("failed to create provider: %v", err)
			}

			token := createTestToken(t, key, tt.claims, jwt.SigningMethodRS256)
			user, ok := provider.VerifyToken(context.Background(), token)
			if !ok {
				t.Fatal("expected token to be valid")
			}
			if user.Role != tt.wantRole {
				t.Errorf("expected role %q, got %q", tt.wantRole, user.Role)
			}
		})
	}
}

func TestJWTProvider_VerifyToken_ClaimsMapping(t *testing.T) {
	key := generateTestRSAKey(t)

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "public.pem")
	if err := writePublicKey(keyFile, &key.PublicKey); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	provider, err := NewJWTProvider(config.JWTConfig{
		Algorithm:     "RS256",
		PublicKeyFile: keyFile,
		ClaimsMapping: map[string]string{
			"sub":  "user_id",
			"role": "user_role",
		},
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	claims := jwt.MapClaims{
		"user_id":   "custom-user",
		"user_role": "custom-admin",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}

	token := createTestToken(t, key, claims, jwt.SigningMethodRS256)
	user, ok := provider.VerifyToken(context.Background(), token)
	if !ok {
		t.Fatal("expected token to be valid")
	}
	if user.Username != "custom-user" {
		t.Errorf("expected username 'custom-user', got %q", user.Username)
	}
	if user.Role != "custom-admin" {
		t.Errorf("expected role 'custom-admin', got %q", user.Role)
	}
}

func TestJWTProvider_VerifyToken_InvalidSignature(t *testing.T) {
	key := generateTestRSAKey(t)
	wrongKey := generateTestRSAKey(t)

	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "public.pem")
	if err := writePublicKey(keyFile, &key.PublicKey); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	provider, err := NewJWTProvider(config.JWTConfig{
		Algorithm:     "RS256",
		PublicKeyFile: keyFile,
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": "testuser",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	// Sign with wrong key
	token := createTestToken(t, wrongKey, claims, jwt.SigningMethodRS256)
	_, ok := provider.VerifyToken(context.Background(), token)
	if ok {
		t.Error("expected token with invalid signature to be rejected")
	}
}

func TestJWTProvider_JWKS_Valid(t *testing.T) {
	key := generateTestRSAKey(t)

	// Create a mock JWKS server
	jwks := createTestJWKS(t, &key.PublicKey, "test-kid")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwks)
	}))
	defer server.Close()

	provider, err := NewJWTProvider(config.JWTConfig{
		Algorithm: "RS256",
		JWKSURL:   server.URL,
		Issuer:    "test-issuer",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": "testuser",
		"iss": "test-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	// Create token with kid header
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	jwtToken.Header["kid"] = "test-kid"
	tokenStr, err := jwtToken.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	user, ok := provider.VerifyToken(context.Background(), tokenStr)
	if !ok {
		t.Fatal("expected token to be valid")
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}
}

func TestJWTProvider_JWKS_InvalidKid(t *testing.T) {
	key := generateTestRSAKey(t)
	wrongKey := generateTestRSAKey(t)

	// Create a mock JWKS server with a different key
	jwks := createTestJWKS(t, &wrongKey.PublicKey, "wrong-kid")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwks)
	}))
	defer server.Close()

	provider, err := NewJWTProvider(config.JWTConfig{
		Algorithm: "RS256",
		JWKSURL:   server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": "testuser",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	// Create token with mismatched kid
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	jwtToken.Header["kid"] = "nonexistent-kid"
	tokenStr, err := jwtToken.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// Should fail because the kid doesn't match and there's only one key with a different kid
	_, ok := provider.VerifyToken(context.Background(), tokenStr)
	// This will succeed because when there's only one key in JWKS, it uses that key
	// regardless of kid mismatch. But signature verification should fail.
	if ok {
		t.Error("expected token to be rejected due to wrong key")
	}
}

// createTestJWKS creates a JWKS JSON response for testing.
func createTestJWKS(t *testing.T, key *rsa.PublicKey, kid string) []byte {
	t.Helper()

	// Encode the modulus and exponent as base64url
	n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": kid,
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	}

	data, err := json.Marshal(jwks)
	if err != nil {
		t.Fatalf("failed to marshal JWKS: %v", err)
	}
	return data
}
