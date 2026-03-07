// Package api provides the HTTP server and routing.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/axonops/axonops-schema-registry/internal/api/handlers"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/metrics"
	"github.com/axonops/axonops-schema-registry/internal/registry"
)

// Server represents the HTTP server.
type Server struct {
	config        *config.Config
	registry      *registry.Registry
	router        chi.Router
	server        *http.Server
	logger        *slog.Logger
	metrics       *metrics.Metrics
	authenticator *auth.Authenticator
	authorizer    *auth.Authorizer
	authService   *auth.Service
	rateLimiter   *auth.RateLimiter
	auditLogger   *auth.AuditLogger
	tlsManager    *auth.TLSManager
	version       string
	commit        string
}

// ServerOption is a function that configures the server.
type ServerOption func(*Server)

// WithAuth configures authentication and authorization for the server.
func WithAuth(authenticator *auth.Authenticator, authorizer *auth.Authorizer, authService *auth.Service) ServerOption {
	return func(s *Server) {
		s.authenticator = authenticator
		s.authorizer = authorizer
		s.authService = authService
	}
}

// WithBuildInfo configures the build version and commit for the server.
func WithBuildInfo(version, commit string) ServerOption {
	return func(s *Server) {
		s.version = version
		s.commit = commit
	}
}

// WithRateLimiter configures rate limiting for the server.
func WithRateLimiter(rateLimiter *auth.RateLimiter) ServerOption {
	return func(s *Server) {
		s.rateLimiter = rateLimiter
	}
}

// WithAuditLogger configures audit logging for the server.
func WithAuditLogger(al *auth.AuditLogger) ServerOption {
	return func(s *Server) {
		s.auditLogger = al
	}
}

// WithMetrics provides a pre-created metrics instance.
// When set, NewServer uses this instead of creating a new one.
func WithMetrics(m *metrics.Metrics) ServerOption {
	return func(s *Server) {
		s.metrics = m
	}
}

// NewServer creates a new HTTP server.
func NewServer(cfg *config.Config, reg *registry.Registry, logger *slog.Logger, opts ...ServerOption) *Server {
	s := &Server{
		config:   cfg,
		registry: reg,
		logger:   logger,
	}

	// Apply options (may set metrics via WithMetrics)
	for _, opt := range opts {
		opt(s)
	}

	// Create metrics if not provided via WithMetrics
	if s.metrics == nil {
		s.metrics = metrics.New()
	}

	// Wire metrics to auth components so they can record Prometheus metrics.
	if s.authenticator != nil {
		s.authenticator.SetMetrics(s.metrics)
	}
	if s.rateLimiter != nil {
		s.rateLimiter.SetMetrics(s.metrics)
	}

	s.setupRouter()
	return s
}

// Metrics returns the metrics instance for recording custom metrics.
func (s *Server) Metrics() *metrics.Metrics {
	return s.metrics
}

// setupRouter configures the HTTP router.
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Return 405 with Confluent-compatible JSON error for unsupported methods
	r.MethodNotAllowed(methodNotAllowedHandler)
	r.NotFound(notFoundHandler)

	// Common middleware for all routes
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.loggingMiddleware)
	if s.auditLogger != nil {
		r.Use(s.auditLogger.Middleware)
	}
	r.Use(s.metrics.Middleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Clean double slashes from URL paths. Some clients (e.g., confluent-kafka-python's
	// DekRegistryClient) construct URLs with "/".join() which produces double slashes
	// when the path starts with "/".
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "//") {
				r.URL.Path = path.Clean(r.URL.Path)
				if r.URL.RawPath != "" {
					r.URL.RawPath = path.Clean(r.URL.RawPath)
				}
			}
			next.ServeHTTP(w, r)
		})
	})

	// Request body size limit
	maxBodySize := int64(10 << 20) // 10MB default
	if s.config.Server.MaxRequestBodySize > 0 {
		maxBodySize = s.config.Server.MaxRequestBodySize
	}
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
			next.ServeHTTP(w, r)
		})
	})

	// Create handlers
	h := handlers.NewWithConfig(s.registry, handlers.Config{
		ClusterID: s.config.Server.ClusterID,
		Version:   s.version,
		Commit:    s.commit,
	})
	h.SetMetrics(s.metrics)

	// Public endpoints (no auth required) - health checks, metrics, and documentation
	r.Get("/", h.HealthCheck)
	r.Get("/health/live", h.LivenessCheck)
	r.Get("/health/ready", h.ReadinessCheck)
	r.Get("/health/startup", h.StartupCheck)
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		s.metrics.Handler().ServeHTTP(w, r)
	})
	if s.config.Server.DocsEnabled {
		r.Get("/docs", handleSwaggerUI)
		r.Get("/openapi.yaml", handleOpenAPISpec)
	}

	// Protected routes group (auth required when configured)
	r.Group(func(r chi.Router) {
		// Add auth middleware if configured
		if s.authenticator != nil {
			r.Use(s.authenticator.Middleware)
		}

		// Add authorization middleware if configured
		if s.authorizer != nil {
			r.Use(s.authorizer.AuthorizeEndpoint(auth.DefaultEndpointPermissions()))
		}

		// Add rate limiting middleware if configured
		if s.rateLimiter != nil {
			r.Use(s.rateLimiter.Middleware)
		}

		// Mount all schema registry routes at root level (default context)
		s.mountRegistryRoutes(r, h)

		// DEK Registry routes (Confluent CSFLE compatible).
		// KEK/DEK endpoints are intentionally global (not mounted under /contexts/{context})
		// because encryption keys are shared resources across all contexts, matching
		// Confluent's behavior.
		r.Route("/dek-registry/v1", func(r chi.Router) {
			// KEK endpoints
			r.Get("/keks", h.ListKEKs)
			r.Post("/keks", h.CreateKEK)
			r.Get("/keks/{name}", h.GetKEK)
			r.Put("/keks/{name}", h.UpdateKEK)
			r.Delete("/keks/{name}", h.DeleteKEK)
			r.Post("/keks/{name}/undelete", h.UndeleteKEK)
			r.Post("/keks/{name}/test", h.TestKEK)

			// DEK endpoints
			r.Get("/keks/{name}/deks", h.ListDEKs)
			r.Post("/keks/{name}/deks", h.CreateDEK)
			r.Get("/keks/{name}/deks/{subject}", h.GetDEK)
			r.Post("/keks/{name}/deks/{subject}", h.CreateDEKWithSubject)
			r.Delete("/keks/{name}/deks/{subject}", h.DeleteDEK)
			r.Post("/keks/{name}/deks/{subject}/undelete", h.UndeleteDEK)
			r.Get("/keks/{name}/deks/{subject}/versions", h.ListDEKVersions)
			r.Get("/keks/{name}/deks/{subject}/versions/{version}", h.GetDEKVersion)
			r.Delete("/keks/{name}/deks/{subject}/versions/{version}", h.DeleteDEKVersion)
			r.Post("/keks/{name}/deks/{subject}/versions/{version}/undelete", h.UndeleteDEKVersion)
		})

		// Account endpoints (self-service, requires auth)
		if s.authService != nil {
			accountHandler := handlers.NewAccountHandler(s.authService)
			r.Route("/me", func(r chi.Router) {
				r.Get("/", accountHandler.GetCurrentUser)
				r.Post("/password", accountHandler.ChangePassword)
			})
		}

		// Admin endpoints (requires auth)
		if s.authService != nil && s.authorizer != nil {
			adminHandler := handlers.NewAdminHandler(s.authService, s.authorizer)
			r.Route("/admin", func(r chi.Router) {
				// User management
				r.Get("/users", adminHandler.ListUsers)
				r.Post("/users", adminHandler.CreateUser)
				r.Get("/users/{id}", adminHandler.GetUser)
				r.Put("/users/{id}", adminHandler.UpdateUser)
				r.Delete("/users/{id}", adminHandler.DeleteUser)

				// API Key management
				r.Get("/apikeys", adminHandler.ListAPIKeys)
				r.Post("/apikeys", adminHandler.CreateAPIKey)
				r.Get("/apikeys/{id}", adminHandler.GetAPIKey)
				r.Put("/apikeys/{id}", adminHandler.UpdateAPIKey)
				r.Delete("/apikeys/{id}", adminHandler.DeleteAPIKey)
				r.Post("/apikeys/{id}/revoke", adminHandler.RevokeAPIKey)
				r.Post("/apikeys/{id}/rotate", adminHandler.RotateAPIKey)

				// Roles
				r.Get("/roles", adminHandler.ListRoles)
			})
		}
	})

	// Context-scoped routes: /contexts/{context}/...
	// These mirror the registry routes but scoped to a specific context.
	r.Route("/contexts/{context}", func(r chi.Router) {
		r.Use(contextExtractionMiddleware)

		// Add auth middleware if configured
		if s.authenticator != nil {
			r.Use(s.authenticator.Middleware)
		}

		// Add authorization middleware if configured
		if s.authorizer != nil {
			r.Use(s.authorizer.AuthorizeEndpoint(auth.DefaultEndpointPermissions()))
		}

		// Add rate limiting middleware if configured
		if s.rateLimiter != nil {
			r.Use(s.rateLimiter.Middleware)
		}

		// Mount schema registry routes under context prefix
		s.mountRegistryRoutes(r, h)
	})

	s.router = r
}

// mountRegistryRoutes registers all schema registry API routes on the given router.
// This is called twice: once at root level (default context) and once under /contexts/{context}.
func (s *Server) mountRegistryRoutes(r chi.Router, h *handlers.Handler) {
	// Schema types
	r.Get("/schemas/types", h.GetSchemaTypes)

	// Schema listing
	r.Get("/schemas", h.ListSchemas)

	// Schema by ID
	r.Get("/schemas/ids/{id}", h.GetSchemaByID)
	r.Get("/schemas/ids/{id}/schema", h.GetRawSchemaByID)
	r.Get("/schemas/ids/{id}/subjects", h.GetSubjectsBySchemaID)
	r.Get("/schemas/ids/{id}/versions", h.GetVersionsBySchemaID)

	// Subjects
	r.Get("/subjects", h.ListSubjects)
	r.Get("/subjects/{subject}/versions", h.GetVersions)
	r.Get("/subjects/{subject}/versions/{version}", h.GetVersion)
	r.Get("/subjects/{subject}/versions/{version}/schema", h.GetRawSchemaByVersion)
	r.Get("/subjects/{subject}/versions/{version}/referencedby", h.GetReferencedBy)
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)
	r.Post("/subjects/{subject}", h.LookupSchema)
	r.Delete("/subjects/{subject}", h.DeleteSubject)
	r.Delete("/subjects/{subject}/versions/{version}", h.DeleteVersion)
	r.Get("/subjects/{subject}/metadata", h.GetSubjectMetadata)

	// Config
	r.Get("/config", h.GetConfig)
	r.Put("/config", h.SetConfig)
	r.Delete("/config", h.DeleteGlobalConfig)
	r.Get("/config/{subject}", h.GetConfig)
	r.Put("/config/{subject}", h.SetConfig)
	r.Delete("/config/{subject}", h.DeleteConfig)

	// Mode
	r.Get("/mode", h.GetMode)
	r.Put("/mode", h.SetMode)
	r.Delete("/mode", h.DeleteGlobalMode)
	r.Get("/mode/{subject}", h.GetMode)
	r.Put("/mode/{subject}", h.SetMode)
	r.Delete("/mode/{subject}", h.DeleteMode)

	// Import (for migration from other schema registries)
	r.Post("/import/schemas", h.ImportSchemas)

	// Compatibility
	r.Post("/compatibility/subjects/{subject}/versions/{version}", h.CheckCompatibility)
	r.Post("/compatibility/subjects/{subject}/versions", h.CheckCompatibility)

	// Contexts
	r.Get("/contexts", h.GetContexts)

	// Exporters (Confluent Schema Linking compatible)
	r.Get("/exporters", h.ListExporters)
	r.Post("/exporters", h.CreateExporter)
	r.Get("/exporters/{name}", h.GetExporter)
	r.Put("/exporters/{name}", h.UpdateExporter)
	r.Delete("/exporters/{name}", h.DeleteExporter)
	r.Put("/exporters/{name}/pause", h.PauseExporter)
	r.Put("/exporters/{name}/resume", h.ResumeExporter)
	r.Put("/exporters/{name}/reset", h.ResetExporter)
	r.Get("/exporters/{name}/status", h.GetExporterStatus)
	r.Get("/exporters/{name}/config", h.GetExporterConfig)
	r.Put("/exporters/{name}/config", h.UpdateExporterConfig)

	// Metadata (v1 API)
	r.Get("/v1/metadata/id", h.GetClusterID)
	r.Get("/v1/metadata/version", h.GetServerVersion)

	// Analysis & Intelligence endpoints (mirror MCP tools)
	r.Post("/schemas/validate", h.ValidateSchema)
	r.Post("/schemas/normalize", h.NormalizeSchema)
	r.Post("/schemas/search", h.SearchSchemas)
	r.Post("/schemas/search/field", h.FindSchemasByField)
	r.Post("/schemas/search/type", h.FindSchemasByType)
	r.Post("/schemas/similar", h.FindSimilarSchemas)
	r.Post("/schemas/quality", h.ScoreSchemaQuality)
	r.Post("/schemas/complexity", h.GetSchemaComplexity)
	r.Post("/subjects/validate", h.ValidateSubjectName)
	r.Post("/subjects/match", h.MatchSubjects)
	r.Get("/subjects/count", h.CountSubjects)
	r.Get("/subjects/{subject}/history", h.GetSchemaHistory)
	r.Get("/subjects/{subject}/versions/{version}/dependencies", h.GetDependencyGraph)
	r.Get("/subjects/{subject}/versions/{version}/export", h.ExportSchema)
	r.Post("/subjects/{subject}/diff", h.DiffSchemas)
	r.Post("/subjects/{subject}/evolve", h.SuggestSchemaEvolution)
	r.Post("/subjects/{subject}/migrate", h.PlanMigrationPath)
	r.Get("/subjects/{subject}/export", h.ExportSubject)
	r.Get("/subjects/{subject}/versions/count", h.CountVersions)
	r.Post("/compatibility/check", h.CheckCompatibilityMulti)
	r.Post("/compatibility/subjects/{subject}/suggest", h.SuggestCompatibleChange)
	r.Post("/compatibility/subjects/{subject}/explain", h.ExplainCompatibilityFailure)
	r.Post("/compatibility/compare", h.CompareSubjects)
	r.Get("/statistics", h.GetRegistryStatistics)
	r.Get("/statistics/fields/{field}", h.CheckFieldConsistency)
	r.Get("/statistics/patterns", h.DetectSchemaPatterns)
}

// loggingMiddleware logs HTTP requests.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			s.logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote", r.RemoteAddr),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	addr := s.config.Address()
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	// Configure TLS if enabled
	if s.config.Security.TLS.Enabled {
		tlsConfig, tm, err := auth.CreateServerTLSConfig(s.config.Security.TLS)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		s.server.TLSConfig = tlsConfig
		s.tlsManager = tm
		s.logger.Info("starting server with TLS", slog.String("address", addr))
		return s.server.ListenAndServeTLS("", "") // Certs loaded via GetCertificate
	}

	s.logger.Info("starting server", slog.String("address", addr))
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// ReloadTLS reloads TLS certificates from disk.
// Returns nil if TLS is not enabled or no TLSManager is configured.
func (s *Server) ReloadTLS() error {
	if s.tlsManager == nil {
		return nil
	}
	return s.tlsManager.Reload()
}

// Router returns the HTTP router for testing.
func (s *Server) Router() http.Handler {
	return s.router
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Address returns the server address.
func (s *Server) Address() string {
	if s.config.Security.TLS.Enabled {
		return fmt.Sprintf("https://%s", s.config.Address())
	}
	return fmt.Sprintf("http://%s", s.config.Address())
}

// methodNotAllowedHandler returns a JSON error response matching Confluent's format
// when an HTTP method is not supported for the matched route.
func methodNotAllowedHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error_code": 405,
		"message":    "HTTP 405 Method Not Allowed",
	})
}

// notFoundHandler returns a JSON error response matching Confluent's format
// when no route matches the request path.
func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error_code":404,"message":"HTTP 404 Not Found"}`))
}
