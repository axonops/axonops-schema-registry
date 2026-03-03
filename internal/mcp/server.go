// Package mcp provides the MCP (Model Context Protocol) server for the schema registry.
package mcp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/metrics"
	"github.com/axonops/axonops-schema-registry/internal/registry"
)

// Server wraps the MCP protocol server and HTTP transport.
type Server struct {
	mcpServer    *gomcp.Server
	httpServer   *http.Server
	registry     *registry.Registry
	authService  *auth.Service
	config       *config.MCPConfig
	logger       *slog.Logger
	metrics      *metrics.Metrics
	auditLogger  *auth.AuditLogger
	confirmStore *ConfirmationStore
	version      string
	commit       string
	buildTime    string
	clusterID    string
}

// Option configures an MCP server.
type Option func(*Server)

// WithAuthService sets the auth service for admin user/API key management tools.
func WithAuthService(svc *auth.Service) Option {
	return func(s *Server) {
		s.authService = svc
	}
}

// WithBuildInfo sets commit hash and build time for the server version tool.
func WithBuildInfo(commit, buildTime string) Option {
	return func(s *Server) {
		s.commit = commit
		s.buildTime = buildTime
	}
}

// WithClusterID sets the cluster ID for the cluster ID tool.
func WithClusterID(id string) Option {
	return func(s *Server) {
		s.clusterID = id
	}
}

// WithMetrics sets the Prometheus metrics instance for MCP tool call tracking.
func WithMetrics(m *metrics.Metrics) Option {
	return func(s *Server) {
		s.metrics = m
	}
}

// WithAuditLogger sets the audit logger for MCP tool call audit trail.
func WithAuditLogger(al *auth.AuditLogger) Option {
	return func(s *Server) {
		s.auditLogger = al
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

	if cfg.RequireConfirmations {
		ttl := time.Duration(cfg.ConfirmationTTLSecs) * time.Second
		if ttl <= 0 {
			ttl = 300 * time.Second
		}
		s.confirmStore = NewConfirmationStore(ttl)
	}

	s.mcpServer = gomcp.NewServer(&gomcp.Implementation{
		Name:    "axonops-schema-registry",
		Version: version,
	}, &gomcp.ServerOptions{
		Instructions: serverInstructions,
	})

	s.registerTools()
	s.registerResources()
	s.registerGlossaryResources()
	s.registerPrompts()
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
	mux.Handle("/mcp", s.originMiddleware(s.authMiddleware(handler)))

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
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
	if s.confirmStore != nil {
		s.confirmStore.Close()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// serverInstructions is returned to MCP clients during the initialize handshake.
const serverInstructions = `You are connected to the AxonOps Schema Registry MCP server -- a Confluent-compatible schema registry for Avro, Protobuf, and JSON Schema.

Capabilities: 105+ tools for schema management, compatibility checking, encryption (CSFLE), exporters (schema linking), data contracts, multi-tenant contexts, and schema intelligence (quality scoring, diff, impact analysis).

Domain knowledge is available as glossary resources. Read these BEFORE answering domain questions:
  schema://glossary/core-concepts     -- subjects, versions, IDs, modes, naming
  schema://glossary/compatibility     -- 7 modes, per-format rules, transitive semantics
  schema://glossary/data-contracts    -- metadata, tags, rulesets, 3-layer merge
  schema://glossary/encryption        -- CSFLE, KEK/DEK, KMS providers, algorithms
  schema://glossary/contexts          -- multi-tenancy, 4-tier inheritance
  schema://glossary/exporters         -- schema linking, lifecycle states
  schema://glossary/schema-types      -- Avro, Protobuf, JSON Schema deep reference
  schema://glossary/design-patterns   -- event envelope, lifecycle, shared types, CI/CD
  schema://glossary/best-practices    -- per-format guidance, common mistakes
  schema://glossary/migration         -- Confluent migration, IMPORT mode, ID preservation

Critical rules:
- Schema IDs are embedded in Kafka messages. NEVER suggest changing IDs in production.
- BACKWARD is the default compatibility. Do not change it without explaining consequences.
- Deleting a subject or schema is SOFT by default. Permanent delete requires ?permanent=true.
- IMPORT mode bypasses compatibility checks. Always switch back to READWRITE after migration.`
