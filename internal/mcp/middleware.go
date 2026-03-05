package mcp

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
)

// authMiddleware enforces bearer token authentication for MCP HTTP requests.
// If no auth token is configured, all requests are allowed.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.AuthToken == "" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" || token == auth {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.config.AuthToken)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// originMiddleware validates the Origin header against the configured allowlist.
// This prevents DNS rebinding attacks per the MCP specification.
// If AllowedOrigins is empty, all origins are allowed (backward-compatible).
// If the Origin header is absent (non-browser clients), the request is allowed.
func (s *Server) originMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || len(s.config.AllowedOrigins) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		if !s.isOriginAllowed(origin) {
			s.logger.Warn("MCP origin rejected",
				slog.String("origin", origin),
				slog.String("remote_addr", r.RemoteAddr),
			)
			if s.metrics != nil {
				s.metrics.RecordMCPPolicyDenial("origin_rejected")
			}
			http.Error(w, "Forbidden: origin not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks whether the given origin is in the allowlist.
// Supports exact match (case-insensitive), "*" wildcard (allow all),
// and glob-style patterns where "*" matches any substring within a
// component (e.g., "http://localhost:*" matches "http://localhost:3000",
// "vscode-webview://*" matches any vscode-webview origin).
func (s *Server) isOriginAllowed(origin string) bool {
	for _, allowed := range s.config.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if originMatchesPattern(allowed, origin) {
			return true
		}
	}
	return false
}

// originMatchesPattern checks if an origin matches a pattern.
// The pattern may contain "*" wildcards that match any sequence of
// characters. Matching is case-insensitive.
func originMatchesPattern(pattern, origin string) bool {
	p := strings.ToLower(pattern)
	o := strings.ToLower(origin)

	// Fast path: no wildcard, exact match.
	if !strings.Contains(p, "*") {
		return p == o
	}

	// Split on "*" and ensure each part appears in order.
	parts := strings.Split(p, "*")
	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		pos := strings.Index(o[idx:], part)
		if pos < 0 {
			return false
		}
		// First segment must be a prefix.
		if i == 0 && pos != 0 {
			return false
		}
		idx += pos + len(part)
	}
	// Last segment must be a suffix (unless pattern ends with *).
	if last := parts[len(parts)-1]; last != "" {
		return strings.HasSuffix(o, last)
	}
	return true
}
