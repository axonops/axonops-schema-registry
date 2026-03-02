package mcp

import (
	"context"
	"encoding/json"
	"fmt"
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

// extractURIParam extracts a named parameter from a resource URI by comparing
// it against the expected path pattern.
func extractURIParam(uri, param string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse URI: %w", err)
	}
	// URI format: schema://host/path... where host is the first segment
	// Examples:
	//   schema://subjects/my-subject -> host="subjects", path="/my-subject"
	//   schema://schemas/42/subjects -> host="schemas", path="/42/subjects"
	host := parsed.Host
	path := parsed.Path

	switch param {
	case "subject":
		if host == "subjects" && len(path) > 1 {
			// schema://subjects/{subject}
			// schema://subjects/{subject}/versions
			// schema://subjects/{subject}/config
			// schema://subjects/{subject}/mode
			// schema://subjects/{subject}/versions/{version}
			parts := strings.SplitN(path[1:], "/", 2) // strip leading /
			return parts[0], nil
		}
	case "version":
		if host == "subjects" && len(path) > 1 {
			// schema://subjects/{subject}/versions/{version}
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
	}
	return "", fmt.Errorf("parameter %q not found in URI %q", param, uri)
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
	config, err := s.registry.GetConfigFull(ctx, registrycontext.DefaultContext, "")
	if err != nil {
		return nil, fmt.Errorf("get global config: %w", err)
	}
	mode, err := s.registry.GetMode(ctx, registrycontext.DefaultContext, "")
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
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
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
	latest, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subject)
	if err != nil {
		return nil, fmt.Errorf("get latest schema for %q: %w", subject, err)
	}
	config, err := s.registry.GetConfigFull(ctx, registrycontext.DefaultContext, subject)
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
	versions, err := s.registry.GetVersions(ctx, registrycontext.DefaultContext, subject, false)
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
	record, err := s.registry.GetSchemaBySubjectVersion(ctx, registrycontext.DefaultContext, subject, version)
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
	config, err := s.registry.GetConfigFull(ctx, registrycontext.DefaultContext, subject)
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
	mode, err := s.registry.GetMode(ctx, registrycontext.DefaultContext, subject)
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
	record, err := s.registry.GetSchemaByID(ctx, registrycontext.DefaultContext, id)
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
	subjects, err := s.registry.GetSubjectsBySchemaID(ctx, registrycontext.DefaultContext, id, false)
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
	versions, err := s.registry.GetVersionsBySchemaID(ctx, registrycontext.DefaultContext, id, false)
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
