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
// Supports "*" wildcards that match any sequence of characters within
// URL components. Matching is case-insensitive.
//
// The pattern is split into scheme, host, and port parts for precise matching:
//   - Scheme must match exactly (no wildcards in scheme)
//   - Host wildcard "*" only matches within domain boundaries (segments split by ".")
//   - Port wildcard "*" matches any port number
//
// This prevents subdomain spoofing: "*.example.com" will NOT match
// "evil.example.com.attacker.com".
func originMatchesPattern(pattern, origin string) bool {
	p := strings.ToLower(pattern)
	o := strings.ToLower(origin)

	// Fast path: no wildcard, exact match.
	if !strings.Contains(p, "*") {
		return p == o
	}

	// Bare wildcard matches everything.
	if p == "*" {
		return true
	}

	// Split into scheme and authority (host:port).
	pScheme, pAuthority := splitOrigin(p)
	oScheme, oAuthority := splitOrigin(o)

	// Scheme must match exactly.
	if pScheme != oScheme {
		return false
	}

	// Split authority into host and port.
	pHost, pPort := splitHostPort(pAuthority)
	oHost, oPort := splitHostPort(oAuthority)

	// Match port: "*" matches any non-empty port, otherwise must match exactly.
	if pPort == "*" {
		if oPort == "" {
			return false
		}
	} else if pPort != oPort {
		return false
	}

	// Match host segments by domain boundary.
	return matchHostPattern(pHost, oHost)
}

// splitOrigin splits "scheme://authority" into scheme and authority.
func splitOrigin(origin string) (scheme, authority string) {
	if idx := strings.Index(origin, "://"); idx >= 0 {
		return origin[:idx], origin[idx+3:]
	}
	return "", origin
}

// splitHostPort splits "host:port" into host and port.
func splitHostPort(authority string) (host, port string) {
	if idx := strings.LastIndex(authority, ":"); idx >= 0 {
		return authority[:idx], authority[idx+1:]
	}
	return authority, ""
}

// matchHostPattern matches a host pattern against an origin host using
// domain-boundary-aware wildcard matching. The pattern is split on "."
// and each segment is matched individually. A "*" segment matches exactly
// one domain segment, preventing cross-boundary matching.
func matchHostPattern(pattern, host string) bool {
	pParts := strings.Split(pattern, ".")
	oParts := strings.Split(host, ".")

	if len(pParts) != len(oParts) {
		return false
	}

	for i, pp := range pParts {
		if pp == "*" {
			continue
		}
		if pp != oParts[i] {
			return false
		}
	}
	return true
}
