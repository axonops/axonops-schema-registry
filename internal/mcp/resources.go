package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"strconv"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerResources() {
	// --- Static resources ---

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://server/info",
		Name:        "server-info",
		Description: "Schema registry server information including version, supported schema types, commit, and build time",
		MIMEType:    "application/json",
	}, s.handleServerInfoResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://server/config",
		Name:        "server-config",
		Description: "Global compatibility level and registry mode configuration",
		MIMEType:    "application/json",
	}, s.handleServerConfigResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://subjects",
		Name:        "subjects",
		Description: "List of all registered subjects in the schema registry",
		MIMEType:    "application/json",
	}, s.handleSubjectsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://types",
		Name:        "schema-types",
		Description: "Supported schema types (AVRO, PROTOBUF, JSON)",
		MIMEType:    "application/json",
	}, s.handleTypesResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://contexts",
		Name:        "contexts",
		Description: "List of all registry contexts (tenant namespaces)",
		MIMEType:    "application/json",
	}, s.handleContextsResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://mode",
		Name:        "global-mode",
		Description: "Global registry mode (READWRITE, READONLY, READONLY_OVERRIDE, IMPORT)",
		MIMEType:    "application/json",
	}, s.handleGlobalModeResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://keks",
		Name:        "keks",
		Description: "List of all Key Encryption Keys (KEKs) for client-side field encryption",
		MIMEType:    "application/json",
	}, s.handleKEKsListResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://exporters",
		Name:        "exporters",
		Description: "List of all schema exporter names",
		MIMEType:    "application/json",
	}, s.handleExportersListResource)

	s.mcpServer.AddResource(&gomcp.Resource{
		URI:         "schema://status",
		Name:        "server-status",
		Description: "Server health status, storage connectivity, and uptime",
		MIMEType:    "application/json",
	}, s.handleStatusResource)

	// --- Templated resources ---

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://subjects/{subject}",
		Name:        "subject-detail",
		Description: "Subject details including latest schema version, type, and compatibility configuration",
		MIMEType:    "application/json",
	}, s.handleSubjectDetailResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://subjects/{subject}/versions",
		Name:        "subject-versions",
		Description: "All version numbers registered for a subject",
		MIMEType:    "application/json",
	}, s.handleSubjectVersionsResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://subjects/{subject}/versions/{version}",
		Name:        "subject-version-detail",
		Description: "Schema at a specific subject version",
		MIMEType:    "application/json",
	}, s.handleSubjectVersionDetailResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://subjects/{subject}/config",
		Name:        "subject-config",
		Description: "Per-subject compatibility configuration",
		MIMEType:    "application/json",
	}, s.handleSubjectConfigResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://subjects/{subject}/mode",
		Name:        "subject-mode",
		Description: "Per-subject registry mode (READWRITE, READONLY, etc.)",
		MIMEType:    "application/json",
	}, s.handleSubjectModeResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://schemas/{id}",
		Name:        "schema-by-id",
		Description: "Schema record by global ID, including subject, version, type, and schema content",
		MIMEType:    "application/json",
	}, s.handleSchemaByIDResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://schemas/{id}/subjects",
		Name:        "schema-subjects",
		Description: "All subjects that use a specific schema ID",
		MIMEType:    "application/json",
	}, s.handleSchemaSubjectsResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://schemas/{id}/versions",
		Name:        "schema-versions",
		Description: "All subject-version pairs that use a specific schema ID",
		MIMEType:    "application/json",
	}, s.handleSchemaVersionsResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://exporters/{name}",
		Name:        "exporter-detail",
		Description: "Exporter configuration and status by name",
		MIMEType:    "application/json",
	}, s.handleExporterDetailResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://keks/{name}",
		Name:        "kek-detail",
		Description: "Key Encryption Key (KEK) details by name",
		MIMEType:    "application/json",
	}, s.handleKEKDetailResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://keks/{name}/deks",
		Name:        "kek-deks",
		Description: "DEK subjects under a specific KEK",
		MIMEType:    "application/json",
	}, s.handleKEKDEKsResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/subjects",
		Name:        "context-subjects",
		Description: "List of subjects in a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSubjectsResource)

	// --- Context-scoped resource templates ---
	// These provide access to registry data within a specific multi-tenant context.
	// The context is extracted from the URI prefix: schema://contexts/{context}/...
	// Handlers are shared with the non-context versions above; they use
	// resolveResourceContext() to determine the context from the URI.

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/config",
		Name:        "context-config",
		Description: "Global compatibility level and mode for a specific registry context",
		MIMEType:    "application/json",
	}, s.handleServerConfigResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/mode",
		Name:        "context-mode",
		Description: "Global registry mode for a specific registry context",
		MIMEType:    "application/json",
	}, s.handleGlobalModeResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/subjects/{subject}",
		Name:        "context-subject-detail",
		Description: "Subject details within a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSubjectDetailResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/subjects/{subject}/versions",
		Name:        "context-subject-versions",
		Description: "All version numbers for a subject within a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSubjectVersionsResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/subjects/{subject}/versions/{version}",
		Name:        "context-subject-version-detail",
		Description: "Schema at a specific subject version within a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSubjectVersionDetailResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/subjects/{subject}/config",
		Name:        "context-subject-config",
		Description: "Per-subject compatibility configuration within a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSubjectConfigResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/subjects/{subject}/mode",
		Name:        "context-subject-mode",
		Description: "Per-subject registry mode within a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSubjectModeResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/schemas/{id}",
		Name:        "context-schema-by-id",
		Description: "Schema record by global ID within a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSchemaByIDResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/schemas/{id}/subjects",
		Name:        "context-schema-subjects",
		Description: "All subjects that use a specific schema ID within a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSchemaSubjectsResource)

	s.mcpServer.AddResourceTemplate(&gomcp.ResourceTemplate{
		URITemplate: "schema://contexts/{context}/schemas/{id}/versions",
		Name:        "context-schema-versions",
		Description: "All subject-version pairs for a schema ID within a specific registry context",
		MIMEType:    "application/json",
	}, s.handleSchemaVersionsResource)
}

// --- Resource result helpers ---

func resourceJSON(uri string, v any) (*gomcp.ReadResourceResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal resource: %w", err)
	}
	return &gomcp.ReadResourceResult{
		Contents: []*gomcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func resourceMarkdown(uri, text string) (*gomcp.ReadResourceResult, error) {
	return &gomcp.ReadResourceResult{
		Contents: []*gomcp.ResourceContents{{
			URI:      uri,
			MIMEType: "text/markdown",
			Text:     text,
		}},
	}, nil
}

// resourceMarkdownFromFS reads a markdown file from an embed.FS and returns it
// as a ReadResourceResult. This enables glossary and other static content to be
// maintained as standalone .md files rather than Go string literals.
func resourceMarkdownFromFS(fsys fs.FS, path, uri string) (*gomcp.ReadResourceResult, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read embedded file %s: %w", path, err)
	}
	return resourceMarkdown(uri, string(data))
}

// extractURIParam extracts a named parameter from a resource URI by comparing
// it against the expected path pattern. It handles both direct URIs
// (schema://subjects/{subject}) and context-prefixed URIs
// (schema://contexts/{context}/subjects/{subject}).
func extractURIParam(uri, param string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse URI: %w", err)
	}
	// URI format: schema://host/path... where host is the first segment
	// Examples:
	//   schema://subjects/my-subject -> host="subjects", path="/my-subject"
	//   schema://schemas/42/subjects -> host="schemas", path="/42/subjects"
	//   schema://contexts/staging/subjects/my-subject -> host="contexts", path="/staging/subjects/my-subject"
	host := parsed.Host
	path := parsed.Path

	// Handle context-prefixed URIs: schema://contexts/{context}/{type}/{rest...}
	// For any param other than "context", strip the context prefix and re-route.
	if host == "contexts" && param != "context" && len(path) > 1 {
		segments := strings.SplitN(path[1:], "/", 3)
		if len(segments) >= 2 {
			host = segments[1]
			if len(segments) == 3 {
				path = "/" + segments[2]
			} else {
				path = ""
			}
		}
	}

	switch param {
	case "subject":
		if host == "subjects" && len(path) > 1 {
			parts := strings.SplitN(path[1:], "/", 2) // strip leading /
			return parts[0], nil
		}
	case "version":
		if host == "subjects" && len(path) > 1 {
			parts := strings.Split(path[1:], "/")
			if len(parts) >= 3 && parts[1] == "versions" {
				return parts[2], nil
			}
		}
	case "id":
		if host == "schemas" && len(path) > 1 {
			parts := strings.SplitN(path[1:], "/", 2)
			return parts[0], nil
		}
	case "name":
		if (host == "exporters" || host == "keks") && len(path) > 1 {
			parts := strings.SplitN(path[1:], "/", 2)
			return parts[0], nil
		}
	case "context":
		if host == "contexts" && len(path) > 1 {
			parts := strings.SplitN(path[1:], "/", 2)
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("parameter %q not found in URI %q", param, uri)
}

// resolveResourceContext extracts the registry context from a resource URI.
// URIs prefixed with schema://contexts/{context}/... use the specified context.
// All other URIs default to the default context.
func resolveResourceContext(uri string) string {
	registryCtx, err := extractURIParam(uri, "context")
	if err != nil {
		return registrycontext.DefaultContext
	}
	return registryCtx
}

// --- Static resource handlers ---

func (s *Server) handleServerInfoResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	info := map[string]any{
		"version":      s.version,
		"commit":       s.commit,
		"build_time":   s.buildTime,
		"schema_types": []string{"AVRO", "PROTOBUF", "JSON"},
	}
	return resourceJSON(req.Params.URI, info)
}

func (s *Server) handleServerConfigResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	registryCtx := resolveResourceContext(req.Params.URI)
	config, err := s.registry.GetConfigFull(ctx, registryCtx, "")
	if err != nil {
		return nil, fmt.Errorf("get global config: %w", err)
	}
	mode, err := s.registry.GetMode(ctx, registryCtx, "")
	if err != nil {
		return nil, fmt.Errorf("get global mode: %w", err)
	}
	result := map[string]any{
		"compatibility": config,
		"mode":          mode,
	}
	return resourceJSON(req.Params.URI, result)
}

func (s *Server) handleSubjectsResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	registryCtx := resolveResourceContext(req.Params.URI)
	subjects, err := s.registry.ListSubjects(ctx, registryCtx, false)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	if subjects == nil {
		subjects = []string{}
	}
	return resourceJSON(req.Params.URI, subjects)
}

func (s *Server) handleTypesResource(_ context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	types := []string{"AVRO", "PROTOBUF", "JSON"}
	return resourceJSON(req.Params.URI, types)
}

func (s *Server) handleContextsResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	contexts, err := s.registry.ListContexts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list contexts: %w", err)
	}
	if contexts == nil {
		contexts = []string{}
	}
	return resourceJSON(req.Params.URI, contexts)
}

// --- Templated resource handlers ---

func (s *Server) handleSubjectDetailResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	subject, err := extractURIParam(req.Params.URI, "subject")
	if err != nil {
		return nil, err
	}
	registryCtx := resolveResourceContext(req.Params.URI)
	latest, err := s.registry.GetLatestSchema(ctx, registryCtx, subject)
	if err != nil {
		return nil, fmt.Errorf("get latest schema for %q: %w", subject, err)
	}
	config, err := s.registry.GetConfigFull(ctx, registryCtx, subject)
	if err != nil {
		return nil, fmt.Errorf("get config for %q: %w", subject, err)
	}
	result := map[string]any{
		"subject":       subject,
		"latest":        latest,
		"compatibility": config,
	}
	return resourceJSON(req.Params.URI, result)
}

func (s *Server) handleSubjectVersionsResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	subject, err := extractURIParam(req.Params.URI, "subject")
	if err != nil {
		return nil, err
	}
	registryCtx := resolveResourceContext(req.Params.URI)
	versions, err := s.registry.GetVersions(ctx, registryCtx, subject, false)
	if err != nil {
		return nil, fmt.Errorf("get versions for %q: %w", subject, err)
	}
	return resourceJSON(req.Params.URI, versions)
}

func (s *Server) handleSubjectVersionDetailResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	subject, err := extractURIParam(req.Params.URI, "subject")
	if err != nil {
		return nil, err
	}
	versionStr, err := extractURIParam(req.Params.URI, "version")
	if err != nil {
		return nil, err
	}
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q: %w", versionStr, err)
	}
	registryCtx := resolveResourceContext(req.Params.URI)
	record, err := s.registry.GetSchemaBySubjectVersion(ctx, registryCtx, subject, version)
	if err != nil {
		return nil, fmt.Errorf("get schema %s version %d: %w", subject, version, err)
	}
	return resourceJSON(req.Params.URI, record)
}

func (s *Server) handleSubjectConfigResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	subject, err := extractURIParam(req.Params.URI, "subject")
	if err != nil {
		return nil, err
	}
	registryCtx := resolveResourceContext(req.Params.URI)
	config, err := s.registry.GetConfigFull(ctx, registryCtx, subject)
	if err != nil {
		return nil, fmt.Errorf("get config for %q: %w", subject, err)
	}
	return resourceJSON(req.Params.URI, config)
}

func (s *Server) handleSubjectModeResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	subject, err := extractURIParam(req.Params.URI, "subject")
	if err != nil {
		return nil, err
	}
	registryCtx := resolveResourceContext(req.Params.URI)
	mode, err := s.registry.GetMode(ctx, registryCtx, subject)
	if err != nil {
		return nil, fmt.Errorf("get mode for %q: %w", subject, err)
	}
	return resourceJSON(req.Params.URI, map[string]string{"mode": mode})
}

func (s *Server) handleSchemaByIDResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	idStr, err := extractURIParam(req.Params.URI, "id")
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid schema ID %q: %w", idStr, err)
	}
	if id <= 0 {
		return nil, fmt.Errorf("schema ID must be positive, got %d", id)
	}
	registryCtx := resolveResourceContext(req.Params.URI)
	record, err := s.registry.GetSchemaByID(ctx, registryCtx, id)
	if err != nil {
		return nil, fmt.Errorf("get schema %d: %w", id, err)
	}
	return resourceJSON(req.Params.URI, record)
}

func (s *Server) handleSchemaSubjectsResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	idStr, err := extractURIParam(req.Params.URI, "id")
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid schema ID %q: %w", idStr, err)
	}
	if id <= 0 {
		return nil, fmt.Errorf("schema ID must be positive, got %d", id)
	}
	registryCtx := resolveResourceContext(req.Params.URI)
	subjects, err := s.registry.GetSubjectsBySchemaID(ctx, registryCtx, id, false)
	if err != nil {
		return nil, fmt.Errorf("get subjects for schema %d: %w", id, err)
	}
	if subjects == nil {
		subjects = []string{}
	}
	return resourceJSON(req.Params.URI, subjects)
}

func (s *Server) handleSchemaVersionsResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	idStr, err := extractURIParam(req.Params.URI, "id")
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid schema ID %q: %w", idStr, err)
	}
	registryCtx := resolveResourceContext(req.Params.URI)
	versions, err := s.registry.GetVersionsBySchemaID(ctx, registryCtx, id, false)
	if err != nil {
		return nil, fmt.Errorf("get versions for schema %d: %w", id, err)
	}
	if versions == nil {
		versions = []storage.SubjectVersion{}
	}
	return resourceJSON(req.Params.URI, versions)
}

func (s *Server) handleExporterDetailResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	name, err := extractURIParam(req.Params.URI, "name")
	if err != nil {
		return nil, err
	}
	exporter, err := s.registry.GetExporter(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get exporter %q: %w", name, err)
	}
	return resourceJSON(req.Params.URI, exporter)
}

func (s *Server) handleKEKDetailResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	name, err := extractURIParam(req.Params.URI, "name")
	if err != nil {
		return nil, err
	}
	kek, err := s.registry.GetKEK(ctx, name, false)
	if err != nil {
		return nil, fmt.Errorf("get KEK %q: %w", name, err)
	}
	return resourceJSON(req.Params.URI, kek)
}

func (s *Server) handleGlobalModeResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	registryCtx := resolveResourceContext(req.Params.URI)
	mode, err := s.registry.GetMode(ctx, registryCtx, "")
	if err != nil {
		return nil, fmt.Errorf("get global mode: %w", err)
	}
	return resourceJSON(req.Params.URI, map[string]string{"mode": mode})
}

func (s *Server) handleKEKsListResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	keks, err := s.registry.ListKEKs(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("list KEKs: %w", err)
	}
	if keks == nil {
		keks = []*storage.KEKRecord{}
	}
	return resourceJSON(req.Params.URI, keks)
}

func (s *Server) handleExportersListResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	exporters, err := s.registry.ListExporters(ctx)
	if err != nil {
		return nil, fmt.Errorf("list exporters: %w", err)
	}
	if exporters == nil {
		exporters = []string{}
	}
	return resourceJSON(req.Params.URI, exporters)
}

func (s *Server) handleStatusResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	healthy := s.registry.IsHealthy(ctx)
	status := map[string]any{
		"healthy":    healthy,
		"version":    s.version,
		"cluster_id": s.clusterID,
	}
	return resourceJSON(req.Params.URI, status)
}

func (s *Server) handleKEKDEKsResource(ctx context.Context, req *gomcp.ReadResourceRequest) (*gomcp.ReadResourceResult, error) {
	name, err := extractURIParam(req.Params.URI, "name")
	if err != nil {
		return nil, err
	}
	deks, err := s.registry.ListDEKs(ctx, name, false)
	if err != nil {
		return nil, fmt.Errorf("list DEKs for KEK %q: %w", name, err)
	}
	if deks == nil {
		deks = []string{}
	}
	return resourceJSON(req.Params.URI, deks)
}
