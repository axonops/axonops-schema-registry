package mcp

import (
	"context"
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	registrycontext "github.com/axonops/axonops-schema-registry/internal/context"
	registrypkg "github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerMetadataTools() {
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_config_full",
		Description: "Get the full configuration record for a subject or global default, including metadata, ruleSets, alias, compatibilityGroup, and all data contract fields. Uses 4-tier fallback: subject → context global → __GLOBAL → server default.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_config_full", s.handleGetConfigFull))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "set_config_full",
		Description: "Set the full configuration for a subject or globally, including compatibility level plus optional data contract fields: alias, compatibilityGroup, defaultMetadata, overrideMetadata, defaultRuleSet, overrideRuleSet.",
	}, instrumentedHandler(s, "set_config_full", s.handleSetConfigFull))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_subject_config_full",
		Description: "Get the full configuration record for a specific subject only, without falling back to global config. Returns error if no subject-level config is set.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_subject_config_full", s.handleGetSubjectConfigFull))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "resolve_alias",
		Description: "Resolve a subject alias. If the subject has an alias configured, returns the alias target. Otherwise returns the original subject name. Resolution is single-level (no recursive chaining).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "resolve_alias", s.handleResolveAlias))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_schemas_by_subject",
		Description: "Get all schema versions for a subject. Returns full schema records for every version, optionally including soft-deleted versions.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_schemas_by_subject", s.handleGetSchemasBySubject))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "check_write_mode",
		Description: "Check if write operations are allowed for a subject. Returns the blocking mode name (READONLY or READONLY_OVERRIDE) or empty string if writes are allowed.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "check_write_mode", s.handleCheckWriteMode))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "test_kek",
		Description: "Test a KEK's KMS connectivity by performing a round-trip encrypt/decrypt test. Requires a KMS provider to be configured.",
	}, instrumentedHandler(s, "test_kek", s.handleTestKEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "format_schema",
		Description: "Format a schema by subject and version. Supported formats depend on schema type. Returns the formatted schema string.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "format_schema", s.handleFormatSchema))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_global_config_direct",
		Description: "Get the global configuration for the current context directly, without falling back to the __GLOBAL context. Returns server default if no context-level global config is set.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_global_config_direct", s.handleGetGlobalConfigDirect))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_subject_metadata",
		Description: "Get metadata for a subject. Without filters, returns the metadata from the latest schema version. With key/value filters, searches all versions for the latest one whose metadata properties match ALL specified key/value pairs and returns a full schema record.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_subject_metadata", s.handleGetSubjectMetadata))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_cluster_id",
		Description: "Get the schema registry cluster ID.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_cluster_id", s.handleGetClusterID))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_server_version",
		Description: "Get detailed server version information including version, commit hash, and build time.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_server_version", s.handleGetServerVersion))
}

// --- Handler types and implementations ---

type getConfigFullInput struct {
	Subject string `json:"subject,omitempty"`
}

func (s *Server) handleGetConfigFull(ctx context.Context, _ *gomcp.CallToolRequest, input getConfigFullInput) (*gomcp.CallToolResult, any, error) {
	config, err := s.registry.GetConfigFull(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(config)
}

type setConfigFullInput struct {
	Subject             string            `json:"subject,omitempty"`
	CompatibilityLevel  string            `json:"compatibility_level"`
	Normalize           *bool             `json:"normalize,omitempty"`
	Alias               string            `json:"alias,omitempty"`
	CompatibilityGroup  string            `json:"compatibility_group,omitempty"`
	ValidateFields      *bool             `json:"validate_fields,omitempty"`
	DefaultMetadata     *storage.Metadata `json:"default_metadata,omitempty"`
	OverrideMetadata    *storage.Metadata `json:"override_metadata,omitempty"`
	DefaultRuleSet      *storage.RuleSet  `json:"default_rule_set,omitempty"`
	OverrideRuleSet     *storage.RuleSet  `json:"override_rule_set,omitempty"`
	AliasForDeks        string            `json:"alias_for_deks,omitempty"`
	CompatibilityPolicy string            `json:"compatibility_policy,omitempty"`
}

func (s *Server) handleSetConfigFull(ctx context.Context, _ *gomcp.CallToolRequest, input setConfigFullInput) (*gomcp.CallToolResult, any, error) {
	opts := registrypkg.SetConfigOpts{
		Alias:               input.Alias,
		CompatibilityGroup:  input.CompatibilityGroup,
		ValidateFields:      input.ValidateFields,
		DefaultMetadata:     input.DefaultMetadata,
		OverrideMetadata:    input.OverrideMetadata,
		DefaultRuleSet:      input.DefaultRuleSet,
		OverrideRuleSet:     input.OverrideRuleSet,
		AliasForDeks:        input.AliasForDeks,
		CompatibilityPolicy: input.CompatibilityPolicy,
	}
	err := s.registry.SetConfig(ctx, registrycontext.DefaultContext, input.Subject, input.CompatibilityLevel, input.Normalize, opts)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]string{"compatibilityLevel": input.CompatibilityLevel})
}

type getSubjectConfigFullInput struct {
	Subject string `json:"subject"`
}

func (s *Server) handleGetSubjectConfigFull(ctx context.Context, _ *gomcp.CallToolRequest, input getSubjectConfigFullInput) (*gomcp.CallToolResult, any, error) {
	config, err := s.registry.GetSubjectConfigFull(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(config)
}

type resolveAliasInput struct {
	Subject string `json:"subject"`
}

func (s *Server) handleResolveAlias(ctx context.Context, _ *gomcp.CallToolRequest, input resolveAliasInput) (*gomcp.CallToolResult, any, error) {
	resolved := s.registry.ResolveAlias(ctx, registrycontext.DefaultContext, input.Subject)
	return jsonResult(map[string]string{"subject": input.Subject, "resolved": resolved})
}

type getSchemasBySubjectInput struct {
	Subject string `json:"subject"`
	Deleted bool   `json:"deleted,omitempty"`
}

func (s *Server) handleGetSchemasBySubject(ctx context.Context, _ *gomcp.CallToolRequest, input getSchemasBySubjectInput) (*gomcp.CallToolResult, any, error) {
	schemas, err := s.registry.GetSchemasBySubject(ctx, registrycontext.DefaultContext, input.Subject, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if schemas == nil {
		schemas = []*storage.SchemaRecord{}
	}
	return jsonResult(schemas)
}

type checkWriteModeInput struct {
	Subject string `json:"subject,omitempty"`
}

func (s *Server) handleCheckWriteMode(ctx context.Context, _ *gomcp.CallToolRequest, input checkWriteModeInput) (*gomcp.CallToolResult, any, error) {
	blockingMode, err := s.registry.CheckModeForWrite(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	result := map[string]any{
		"writable": blockingMode == "",
	}
	if blockingMode != "" {
		result["blocking_mode"] = blockingMode
	}
	return jsonResult(result)
}

type testKEKInput struct {
	Name     string            `json:"name"`
	KmsType  string            `json:"kms_type"`
	KmsKeyID string            `json:"kms_key_id"`
	KmsProps map[string]string `json:"kms_props,omitempty"`
}

func (s *Server) handleTestKEK(ctx context.Context, _ *gomcp.CallToolRequest, input testKEKInput) (*gomcp.CallToolResult, any, error) {
	kek := &storage.KEKRecord{
		Name:     input.Name,
		KmsType:  input.KmsType,
		KmsKeyID: input.KmsKeyID,
		KmsProps: input.KmsProps,
	}
	if err := s.registry.TestKEK(ctx, kek); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"success": true})
}

type formatSchemaInput struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
	Format  string `json:"format,omitempty"`
}

func (s *Server) handleFormatSchema(ctx context.Context, _ *gomcp.CallToolRequest, input formatSchemaInput) (*gomcp.CallToolResult, any, error) {
	version := input.Version
	if version == 0 {
		version = -1 // latest
	}

	var record *storage.SchemaRecord
	var err error
	if version == -1 {
		record, err = s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.Subject)
	} else {
		record, err = s.registry.GetSchemaBySubjectVersion(ctx, registrycontext.DefaultContext, input.Subject, version)
	}
	if err != nil {
		return errorResult(err), nil, nil
	}

	formatted := s.registry.FormatSchema(ctx, registrycontext.DefaultContext, record, input.Format)
	return jsonResult(map[string]any{
		"subject":    input.Subject,
		"version":    record.Version,
		"schemaType": string(record.SchemaType),
		"schema":     formatted,
	})
}

type getGlobalConfigDirectInput struct{}

func (s *Server) handleGetGlobalConfigDirect(ctx context.Context, _ *gomcp.CallToolRequest, _ getGlobalConfigDirectInput) (*gomcp.CallToolResult, any, error) {
	config, err := s.registry.GetGlobalConfigDirect(ctx, registrycontext.DefaultContext)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(config)
}

// --- Subject metadata handler ---

type getSubjectMetadataInput struct {
	Subject        string            `json:"subject"`
	MetadataFilter map[string]string `json:"metadata_filter,omitempty"`
	Deleted        bool              `json:"deleted,omitempty"`
}

func (s *Server) handleGetSubjectMetadata(ctx context.Context, _ *gomcp.CallToolRequest, input getSubjectMetadataInput) (*gomcp.CallToolResult, any, error) {
	if len(input.MetadataFilter) > 0 {
		// Search all versions for the latest matching the metadata filter.
		schemas, err := s.registry.GetSchemasBySubject(ctx, registrycontext.DefaultContext, input.Subject, input.Deleted)
		if err != nil {
			return errorResult(err), nil, nil
		}

		for i := len(schemas) - 1; i >= 0; i-- {
			rec := schemas[i]
			if rec.Metadata == nil || rec.Metadata.Properties == nil {
				continue
			}
			allMatch := true
			for k, v := range input.MetadataFilter {
				if propVal, ok := rec.Metadata.Properties[k]; !ok || propVal != v {
					allMatch = false
					break
				}
			}
			if allMatch {
				return jsonResult(rec)
			}
		}
		return errorResult(fmt.Errorf("no schema version found matching the specified metadata")), nil, nil
	}

	// No filter: return bare metadata from latest version.
	schema, err := s.registry.GetLatestSchema(ctx, registrycontext.DefaultContext, input.Subject)
	if err != nil {
		return errorResult(err), nil, nil
	}
	meta := schema.Metadata
	if meta == nil {
		meta = &storage.Metadata{}
	}
	return jsonResult(meta)
}

// --- Cluster ID and server version handlers ---

type getClusterIDInput struct{}

func (s *Server) handleGetClusterID(_ context.Context, _ *gomcp.CallToolRequest, _ getClusterIDInput) (*gomcp.CallToolResult, any, error) {
	id := s.clusterID
	if id == "" {
		id = "default-cluster"
	}
	return jsonResult(map[string]string{"id": id})
}

type getServerVersionInput struct{}

func (s *Server) handleGetServerVersion(_ context.Context, _ *gomcp.CallToolRequest, _ getServerVersionInput) (*gomcp.CallToolResult, any, error) {
	resp := map[string]string{
		"version": s.version,
	}
	if s.commit != "" {
		resp["commit"] = s.commit
	}
	if s.buildTime != "" {
		resp["build_time"] = s.buildTime
	}
	return jsonResult(resp)
}
