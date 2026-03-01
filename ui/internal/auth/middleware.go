package auth

import (
	"context"
	"net/http"
)

type contextKey string

const usernameKey contextKey = "username"

// Middleware returns an HTTP middleware that validates the session cookie.
// Requests to paths in skipPaths bypass authentication.
func Middleware(tm *TokenManager, cookieName string, skipPaths map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			claims, err := tm.Validate(cookie.Value)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), usernameKey, claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UsernameFromContext extracts the authenticated username from the request context.
func UsernameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(usernameKey).(string)
	return v
}

// TestContextKey returns the context key used for the username.
// Exported for use in tests outside the auth package.
func TestContextKey() contextKey {
	return usernameKey
}
