// Package proxy provides a reverse proxy to the Schema Registry backend.
package proxy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Config holds proxy configuration.
type Config struct {
	TargetURL string
	APIToken  string
	APIKey    string
}

// New creates a reverse proxy handler that forwards /api/v1/* to the Schema Registry.
// It strips the /api/v1 prefix and injects authentication headers.
func New(cfg Config) (http.Handler, error) {
	target, err := url.Parse(cfg.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("parsing target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Strip /api/v1 prefix: /api/v1/subjects → /subjects
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}

		// Inject authentication
		if cfg.APIToken != "" {
			req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
		} else if cfg.APIKey != "" {
			req.Header.Set("X-API-Key", cfg.APIKey)
		}

		// Remove cookie header — SR doesn't need UI cookies
		req.Header.Del("Cookie")

		req.Host = target.Host
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("proxy error", "error", err, "path", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "schema registry unavailable",
		})
	}

	return proxy, nil
}
