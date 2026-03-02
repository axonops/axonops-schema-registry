// Package mcp provides the MCP (Model Context Protocol) server for the schema registry.
package mcp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/registry"
)

// Server wraps the MCP protocol server and HTTP transport.
type Server struct {
	mcpServer   *gomcp.Server
	httpServer  *http.Server
	registry    *registry.Registry
	authService *auth.Service
	config      *config.MCPConfig
	logger      *slog.Logger
	version     string
}

// Option configures an MCP server.
type Option func(*Server)

// WithAuthService sets the auth service for admin user/API key management tools.
func WithAuthService(svc *auth.Service) Option {
	return func(s *Server) {
		s.authService = svc
	}
}

// New creates a new MCP server.
func New(cfg *config.MCPConfig, reg *registry.Registry, logger *slog.Logger, version string, opts ...Option) *Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	s := &Server{
		registry: reg,
		config:   cfg,
		logger:   logger,
		version:  version,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.mcpServer = gomcp.NewServer(&gomcp.Implementation{
		Name:    "axonops-schema-registry",
		Version: version,
	}, nil)

	s.registerTools()
	return s
}

// MCPServer returns the underlying MCP server (for testing with InMemoryTransport).
func (s *Server) MCPServer() *gomcp.Server {
	return s.mcpServer
}

// Start starts the MCP HTTP server. Blocks until the server stops.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	handler := gomcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *gomcp.Server { return s.mcpServer },
		nil,
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("mcp listen: %w", err)
	}
	s.logger.Info("MCP server listening", slog.String("address", addr))
	return s.httpServer.Serve(ln)
}

// Shutdown gracefully shuts down the MCP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
