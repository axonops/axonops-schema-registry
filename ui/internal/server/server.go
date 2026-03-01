package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/axonops/schema-registry-ui/internal/auth"
	"github.com/axonops/schema-registry-ui/internal/config"
	"github.com/axonops/schema-registry-ui/internal/htpasswd"
	"github.com/axonops/schema-registry-ui/internal/proxy"
	"github.com/axonops/schema-registry-ui/internal/users"
)

// Server is the UI HTTP server.
type Server struct {
	httpServer *http.Server
	cfg        *config.Config
}

// New creates a new Server with all routes and middleware.
func New(cfg *config.Config, spaFS fs.FS) (*Server, error) {
	// Initialize htpasswd file
	hp := htpasswd.New(cfg.Auth.HtpasswdFile)
	created, err := hp.Bootstrap("admin", "admin")
	if err != nil {
		return nil, fmt.Errorf("bootstrapping htpasswd: %w", err)
	}
	if created {
		slog.Warn("default admin user created — change the password immediately")
	}

	// Initialize JWT token manager
	tm := auth.NewTokenManager(cfg.Auth.SessionSecret, cfg.Auth.SessionTTL)

	// Initialize handlers
	authHandlers := auth.NewHandlers(hp, tm, cfg.Auth.CookieName, cfg.Auth.CookieSecure)
	userSvc := users.NewService(hp)
	userHandlers := users.NewHandlers(userSvc)

	// Initialize reverse proxy
	proxyHandler, err := proxy.New(proxy.Config{
		TargetURL: cfg.Registry.URL,
		APIToken:  cfg.Registry.APIToken,
		APIKey:    cfg.Registry.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("creating proxy: %w", err)
	}

	// Build router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Compress(5))

	// Health endpoint (unauthenticated)
	r.Get("/health", authHandlers.Health)

	// Auth endpoints (unauthenticated)
	r.Post("/api/auth/login", authHandlers.Login)
	r.Get("/api/auth/config", authHandlers.Config)

	// Authenticated API routes
	skipPaths := map[string]bool{
		"/api/auth/login":  true,
		"/api/auth/config": true,
		"/health":          true,
	}

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(tm, cfg.Auth.CookieName, skipPaths))

		// Auth (authenticated)
		r.Post("/api/auth/logout", authHandlers.Logout)
		r.Get("/api/auth/session", authHandlers.Session)

		// User management
		r.Get("/api/users", userHandlers.ListUsers)
		r.Post("/api/users", userHandlers.CreateUser)
		r.Put("/api/users/{username}", userHandlers.UpdateUser)
		r.Delete("/api/users/{username}", userHandlers.DeleteUser)
		r.Post("/api/users/me/password", userHandlers.ChangeMyPassword)

		// Proxy to Schema Registry
		r.Handle("/api/v1/*", proxyHandler)
	})

	// SPA — serve React app for everything else
	if spaFS != nil {
		r.NotFound(SPAHandler(spaFS).ServeHTTP)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return &Server{httpServer: srv, cfg: cfg}, nil
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	slog.Info("server listening", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down server")
	return s.httpServer.Shutdown(ctx)
}

// Addr returns the server listen address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Handler returns the HTTP handler for use in tests.
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}
