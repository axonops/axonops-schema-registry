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
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/metrics"
	"github.com/axonops/axonops-schema-registry/internal/registry"
)

// Server represents the HTTP server.
type Server struct {
	config   *config.Config
	registry *registry.Registry
	router   chi.Router
	server   *http.Server
	logger   *slog.Logger
	metrics  *metrics.Metrics
}

// NewServer creates a new HTTP server.
func NewServer(cfg *config.Config, reg *registry.Registry, logger *slog.Logger) *Server {
	s := &Server{
		config:   cfg,
		registry: reg,
		logger:   logger,
		metrics:  metrics.New(),
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

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.loggingMiddleware)
	r.Use(s.metrics.Middleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Create handlers
	h := handlers.New(s.registry)

	// Health check
	r.Get("/", h.HealthCheck)

	// Metrics endpoint
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		s.metrics.Handler().ServeHTTP(w, r)
	})

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

	// Compatibility
	r.Post("/compatibility/subjects/{subject}/versions/{version}", h.CheckCompatibility)
	r.Post("/compatibility/subjects/{subject}/versions", h.CheckCompatibility) // Check against all versions

	// Contexts
	r.Get("/contexts", h.GetContexts)

	// Metadata (v1 API)
	r.Get("/v1/metadata/id", h.GetClusterID)
	r.Get("/v1/metadata/version", h.GetServerVersion)

	s.router = r
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
	return fmt.Sprintf("http://%s", s.config.Address())
}
