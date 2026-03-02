package mcp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerValidationTools() {
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "validate_schema",
		Description: "Validate a schema without registering it. Returns whether the schema is valid, its fingerprint, and any parse errors.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "validate_schema", s.handleValidateSchema))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "normalize_schema",
		Description: "Parse and normalize a schema, returning the canonical form and fingerprint for deduplication.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "normalize_schema", s.handleNormalizeSchema))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "validate_subject_name",
		Description: "Validate a subject name against a naming strategy (topic_name, record_name, or topic_record_name).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "validate_subject_name", s.handleValidateSubjectName))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "search_schemas",
		Description: "Search schema content across all subjects using a regex or substring pattern.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "search_schemas", s.handleSearchSchemas))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_schema_history",
		Description: "Get the full version history for a subject, including schema content and metadata for each version.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_schema_history", s.handleGetSchemaHistory))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_dependency_graph",
		Description: "Build a dependency graph for a subject-version, showing all schemas that reference it (recursively, up to depth 10).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_dependency_graph", s.handleGetDependencyGraph))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "export_schema",
		Description: "Export a single schema version with its configuration and metadata in a portable format.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "export_schema", s.handleExportSchema))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "export_subject",
		Description: "Export all schema versions for a subject with configuration and metadata.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "export_subject", s.handleExportSubject))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_registry_statistics",
		Description: "Get aggregate statistics about the registry: total subjects, schemas, types breakdown, KEKs, DEKs, and exporters.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_registry_statistics", s.handleGetRegistryStatistics))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "count_versions",
		Description: "Count the number of schema versions registered for a subject.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "count_versions", s.handleCountVersions))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "count_subjects",
		Description: "Count the total number of registered subjects in the registry.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "count_subjects", s.handleCountSubjects))
}

// --- validate_schema ---

type validateSchemaInput struct {
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schema_type,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
}

func (s *Server) handleValidateSchema(ctx context.Context, _ *gomcp.CallToolRequest, input validateSchemaInput) (*gomcp.CallToolResult, any, error) {
	schemaType := storage.SchemaType(input.SchemaType)
	result, err := s.registry.ValidateSchema(ctx, registrycontext.DefaultContext, input.Schema, schemaType, input.References)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(result)
}

// --- normalize_schema ---

type normalizeSchemaInput struct {
	Schema     string              `json:"schema"`
	SchemaType string              `json:"schema_type,omitempty"`
	References []storage.Reference `json:"references,omitempty"`
}

func (s *Server) handleNormalizeSchema(ctx context.Context, _ *gomcp.CallToolRequest, input normalizeSchemaInput) (*gomcp.CallToolResult, any, error) {
	schemaType := storage.SchemaType(input.SchemaType)
	result, err := s.registry.NormalizeSchema(ctx, registrycontext.DefaultContext, input.Schema, schemaType, input.References)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(result)
}

// --- validate_subject_name ---

type validateSubjectNameInput struct {
	Subject  string `json:"subject"`
	Strategy string `json:"strategy,omitempty"`
}

// subjectNamePatterns defines regex patterns for each naming strategy.
var subjectNamePatterns = map[string]*regexp.Regexp{
	"topic_name":        regexp.MustCompile(`^[a-zA-Z0-9._-]+-(key|value)$`),
	"record_name":       regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`),
	"topic_record_name": regexp.MustCompile(`^[a-zA-Z0-9._-]+-[a-zA-Z_][a-zA-Z0-9_.]*$`),
}

func (s *Server) handleValidateSubjectName(_ context.Context, _ *gomcp.CallToolRequest, input validateSubjectNameInput) (*gomcp.CallToolResult, any, error) {
	strategy := input.Strategy
	if strategy == "" {
		strategy = "topic_name"
	}

	pattern, ok := subjectNamePatterns[strategy]
	if !ok {
		return jsonResult(map[string]any{
			"valid":    false,
			"subject":  input.Subject,
			"strategy": strategy,
			"error":    "unknown strategy; supported: topic_name, record_name, topic_record_name",
		})
	}

	valid := pattern.MatchString(input.Subject)
	result := map[string]any{
		"valid":    valid,
		"subject":  input.Subject,
		"strategy": strategy,
		"pattern":  pattern.String(),
	}
	if !valid {
		result["error"] = "subject name does not match the " + strategy + " naming strategy"
	}
	return jsonResult(result)
}

// --- search_schemas ---

type searchSchemasInput struct {
	Pattern string `json:"pattern"`
	Regex   bool   `json:"regex,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type searchSchemasMatch struct {
	Subject    string `json:"subject"`
	Version    int    `json:"version"`
	SchemaType string `json:"schema_type"`
}

func (s *Server) handleSearchSchemas(ctx context.Context, _ *gomcp.CallToolRequest, input searchSchemasInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	var re *regexp.Regexp
	if input.Regex {
		re, err = regexp.Compile(input.Pattern)
		if err != nil {
			return errorResult(err), nil, nil
		}
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}

	var matches []searchSchemasMatch
	for _, subj := range subjects {
		if len(matches) >= limit {
			break
		}
		record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subj)
		if err != nil {
			continue
		}
		matched := false
		if input.Regex && re != nil {
			matched = re.MatchString(record.Schema)
		} else {
			matched = strings.Contains(record.Schema, input.Pattern)
		}
		if matched {
			matches = append(matches, searchSchemasMatch{
				Subject:    record.Subject,
				Version:    record.Version,
				SchemaType: string(record.SchemaType),
			})
		}
	}
	if matches == nil {
		matches = []searchSchemasMatch{}
	}
	return jsonResult(map[string]any{
		"matches": matches,
		"count":   len(matches),
	})
}

// --- get_schema_history ---

type getSchemaHistoryInput struct {
	Subject string `json:"subject"`
}

type schemaHistoryEntry struct {
	Version    int                 `json:"version"`
	ID         int64               `json:"id"`
	SchemaType string              `json:"schema_type"`
	Schema     string              `json:"schema"`
	References []storage.Reference `json:"references,omitempty"`
}

func (s *Server) handleGetSchemaHistory(ctx context.Context, _ *gomcp.CallToolRequest, input getSchemaHistoryInput) (*gomcp.CallToolResult, any, error) {
	schemas, err := s.registry.GetSchemasBySubject(ctx, registrycontext.DefaultContext, input.Subject, false)
	if err != nil {
		return errorResult(err), nil, nil
	}
	entries := make([]schemaHistoryEntry, 0, len(schemas))
	for _, r := range schemas {
		entries = append(entries, schemaHistoryEntry{
			Version:    r.Version,
			ID:         r.ID,
			SchemaType: string(r.SchemaType),
			Schema:     r.Schema,
			References: r.References,
		})
	}
	return jsonResult(map[string]any{
		"subject":  input.Subject,
		"versions": entries,
		"count":    len(entries),
	})
}

// --- get_dependency_graph ---

type getDependencyGraphInput struct {
	Subject  string `json:"subject"`
	Version  int    `json:"version"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

type dependencyNode struct {
	Subject  string           `json:"subject"`
	Version  int              `json:"version"`
	Depth    int              `json:"depth"`
	Children []dependencyNode `json:"children,omitempty"`
}

func (s *Server) handleGetDependencyGraph(ctx context.Context, _ *gomcp.CallToolRequest, input getDependencyGraphInput) (*gomcp.CallToolResult, any, error) {
	maxDepth := input.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 10
	}

	seen := make(map[string]bool)
	root := dependencyNode{Subject: input.Subject, Version: input.Version, Depth: 0}
	s.buildDependencyTree(ctx, &root, seen, maxDepth)

	return jsonResult(root)
}

func (s *Server) buildDependencyTree(ctx context.Context, node *dependencyNode, seen map[string]bool, maxDepth int) {
	if node.Depth >= maxDepth {
		return
	}
	key := fmt.Sprintf("%s:%d", node.Subject, node.Version)
	if seen[key] {
		return
	}
	seen[key] = true

	refs, err := s.registry.GetReferencedBy(ctx, registrycontext.DefaultContext, node.Subject, node.Version)
	if err != nil || len(refs) == 0 {
		return
	}

	for _, ref := range refs {
		child := dependencyNode{
			Subject: ref.Subject,
			Version: ref.Version,
			Depth:   node.Depth + 1,
		}
		s.buildDependencyTree(ctx, &child, seen, maxDepth)
		node.Children = append(node.Children, child)
	}
}

// --- export_schema ---

type exportSchemaInput struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
}

func (s *Server) handleExportSchema(ctx context.Context, _ *gomcp.CallToolRequest, input exportSchemaInput) (*gomcp.CallToolResult, any, error) {
	version := input.Version
	if version <= 0 {
		version = -1 // latest
	}
	record, err := s.registry.GetSchemaBySubjectVersion(ctx, registrycontext.DefaultContext, input.Subject, version)
	if err != nil {
		return errorResult(err), nil, nil
	}

	config, _ := s.registry.GetConfigFull(ctx, registrycontext.DefaultContext, input.Subject)

	export := map[string]any{
		"subject":     record.Subject,
		"version":     record.Version,
		"id":          record.ID,
		"schema_type": string(record.SchemaType),
		"schema":      record.Schema,
	}
	if len(record.References) > 0 {
		export["references"] = record.References
	}
	if record.Metadata != nil {
		export["metadata"] = record.Metadata
	}
	if record.RuleSet != nil {
		export["rule_set"] = record.RuleSet
	}
	if config != nil {
		export["compatibility"] = config.CompatibilityLevel
	}

	return jsonResult(export)
}

// --- export_subject ---

type exportSubjectInput struct {
	Subject string `json:"subject"`
}

func (s *Server) handleExportSubject(ctx context.Context, _ *gomcp.CallToolRequest, input exportSubjectInput) (*gomcp.CallToolResult, any, error) {
	schemas, err := s.registry.GetSchemasBySubject(ctx, registrycontext.DefaultContext, input.Subject, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	config, _ := s.registry.GetConfigFull(ctx, registrycontext.DefaultContext, input.Subject)

	versions := make([]map[string]any, 0, len(schemas))
	for _, r := range schemas {
		v := map[string]any{
			"version":     r.Version,
			"id":          r.ID,
			"schema_type": string(r.SchemaType),
			"schema":      r.Schema,
		}
		if len(r.References) > 0 {
			v["references"] = r.References
		}
		if r.Metadata != nil {
			v["metadata"] = r.Metadata
		}
		if r.RuleSet != nil {
			v["rule_set"] = r.RuleSet
		}
		versions = append(versions, v)
	}

	export := map[string]any{
		"subject":  input.Subject,
		"versions": versions,
		"count":    len(versions),
	}
	if config != nil {
		export["compatibility"] = config.CompatibilityLevel
	}

	return jsonResult(export)
}

// --- get_registry_statistics ---

type getRegistryStatisticsInput struct{}

func (s *Server) handleGetRegistryStatistics(ctx context.Context, _ *gomcp.CallToolRequest, _ getRegistryStatisticsInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}

	stats := map[string]any{
		"total_subjects": len(subjects),
	}

	// Count schemas and types
	typeCounts := map[string]int{}
	totalVersions := 0
	for _, subj := range subjects {
		versions, err := s.registry.GetVersions(ctx, registrycontext.DefaultContext, subj, false)
		if err != nil {
			continue
		}
		totalVersions += len(versions)
		if len(versions) > 0 {
			record, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, subj)
			if err == nil {
				typeCounts[string(record.SchemaType)]++
			}
		}
	}
	stats["total_versions"] = totalVersions
	stats["types"] = typeCounts

	// Count KEKs
	keks, err := s.registry.ListKEKs(ctx, false)
	if err == nil {
		stats["total_keks"] = len(keks)
	}

	// Count exporters
	exporters, err := s.registry.ListExporters(ctx)
	if err == nil {
		stats["total_exporters"] = len(exporters)
	}

	return jsonResult(stats)
}

// --- count_versions ---

type countVersionsInput struct {
	Subject string `json:"subject"`
}

func (s *Server) handleCountVersions(ctx context.Context, _ *gomcp.CallToolRequest, input countVersionsInput) (*gomcp.CallToolResult, any, error) {
	versions, err := s.registry.GetVersions(ctx, registrycontext.DefaultContext, input.Subject, false)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]any{
		"subject": input.Subject,
		"count":   len(versions),
	})
}

// --- count_subjects ---

type countSubjectsInput struct{}

func (s *Server) handleCountSubjects(ctx context.Context, _ *gomcp.CallToolRequest, _ countSubjectsInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListSubjects(ctx, registrycontext.DefaultContext, false)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]any{
		"count": len(subjects),
	})
}
