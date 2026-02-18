// Package api provides the HTTP server and routing.
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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

// WithRateLimiter configures rate limiting for the server.
func WithRateLimiter(rateLimiter *auth.RateLimiter) ServerOption {
	return func(s *Server) {
		s.rateLimiter = rateLimiter
	}
}

// NewServer creates a new HTTP server.
func NewServer(cfg *config.Config, reg *registry.Registry, logger *slog.Logger, opts ...ServerOption) *Server {
	s := &Server{
		config:   cfg,
		registry: reg,
		logger:   logger,
		metrics:  metrics.New(),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
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

	// Common middleware for all routes
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.loggingMiddleware)
	r.Use(s.metrics.Middleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Create handlers
	h := handlers.New(s.registry)

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

	// Metadata (v1 API)
	r.Get("/v1/metadata/id", h.GetClusterID)
	r.Get("/v1/metadata/version", h.GetServerVersion)
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
		tlsConfig, err := auth.CreateServerTLSConfig(s.config.Security.TLS)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		s.server.TLSConfig = tlsConfig
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
