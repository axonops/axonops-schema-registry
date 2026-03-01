// Package auth provides JWT-based authentication for the UI server.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims for a UI session.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// TokenManager creates and validates JWT tokens.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenManager creates a new TokenManager with the given secret and TTL.
func NewTokenManager(secret string, ttlSeconds int) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

// Create generates a new signed JWT token for the given username.
func (tm *TokenManager) Create(username string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(tm.ttl)

	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "schema-registry-ui",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(tm.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing token: %w", err)
	}

	return signed, expiresAt, nil
}

// Validate parses and validates a JWT token string, returning the claims.
func (tm *TokenManager) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// TTL returns the token time-to-live duration.
func (tm *TokenManager) TTL() time.Duration {
	return tm.ttl
}
