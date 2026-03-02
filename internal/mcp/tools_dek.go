package mcp

import (
	"context"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerDEKTools() {
	// KEK tools
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "create_kek",
		Description: "Create a new Key Encryption Key (KEK) for client-side field encryption (CSFLE). A KEK wraps Data Encryption Keys (DEKs) via a KMS provider.",
	}, instrumentedHandler(s, "create_kek", s.handleCreateKEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_kek",
		Description: "Get a Key Encryption Key (KEK) by name. Use deleted=true to include soft-deleted KEKs.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_kek", s.handleGetKEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "update_kek",
		Description: "Update an existing Key Encryption Key (KEK). Only kms_props, doc, and shared can be changed.",
	}, instrumentedHandler(s, "update_kek", s.handleUpdateKEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "delete_kek",
		Description: "Delete a Key Encryption Key (KEK). Use permanent=true for hard delete (default is soft-delete).",
	}, instrumentedHandler(s, "delete_kek", s.handleDeleteKEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "undelete_kek",
		Description: "Restore a soft-deleted Key Encryption Key (KEK).",
	}, instrumentedHandler(s, "undelete_kek", s.handleUndeleteKEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "list_keks",
		Description: "List all Key Encryption Keys (KEKs). Use deleted=true to include soft-deleted KEKs.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_keks", s.handleListKEKs))

	// DEK tools
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "create_dek",
		Description: "Create a new Data Encryption Key (DEK) under a KEK. The DEK is used for client-side field encryption.",
	}, instrumentedHandler(s, "create_dek", s.handleCreateDEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_dek",
		Description: "Get a Data Encryption Key (DEK) by KEK name, subject, version, and algorithm.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_dek", s.handleGetDEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "list_deks",
		Description: "List all subject names that have DEKs under a given KEK.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_deks", s.handleListDEKs))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "list_dek_versions",
		Description: "List all version numbers for a DEK subject under a given KEK.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_dek_versions", s.handleListDEKVersions))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "delete_dek",
		Description: "Delete a Data Encryption Key (DEK). Use permanent=true for hard delete (default is soft-delete).",
	}, instrumentedHandler(s, "delete_dek", s.handleDeleteDEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "undelete_dek",
		Description: "Restore a soft-deleted Data Encryption Key (DEK).",
	}, instrumentedHandler(s, "undelete_dek", s.handleUndeleteDEK))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "rewrap_dek",
		Description: "Re-encrypt a DEK's key material under the current KEK key version. Used after KEK rotation.",
	}, instrumentedHandler(s, "rewrap_dek", s.handleRewrapDEK))
}

// --- KEK handlers ---

type createKEKInput struct {
	Name     string            `json:"name"`
	KmsType  string            `json:"kms_type"`
	KmsKeyID string            `json:"kms_key_id"`
	KmsProps map[string]string `json:"kms_props,omitempty"`
	Doc      string            `json:"doc,omitempty"`
	Shared   bool              `json:"shared,omitempty"`
}

func (s *Server) handleCreateKEK(ctx context.Context, _ *gomcp.CallToolRequest, input createKEKInput) (*gomcp.CallToolResult, any, error) {
	kek := &storage.KEKRecord{
		Name:     input.Name,
		KmsType:  input.KmsType,
		KmsKeyID: input.KmsKeyID,
		KmsProps: input.KmsProps,
		Doc:      input.Doc,
		Shared:   input.Shared,
	}
	if err := s.registry.CreateKEK(ctx, kek); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(kek)
}

type getKEKInput struct {
	Name    string `json:"name"`
	Deleted bool   `json:"deleted,omitempty"`
}

func (s *Server) handleGetKEK(ctx context.Context, _ *gomcp.CallToolRequest, input getKEKInput) (*gomcp.CallToolResult, any, error) {
	kek, err := s.registry.GetKEK(ctx, input.Name, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(kek)
}

type updateKEKInput struct {
	Name     string            `json:"name"`
	KmsProps map[string]string `json:"kms_props,omitempty"`
	Doc      string            `json:"doc,omitempty"`
	Shared   bool              `json:"shared,omitempty"`
}

func (s *Server) handleUpdateKEK(ctx context.Context, _ *gomcp.CallToolRequest, input updateKEKInput) (*gomcp.CallToolResult, any, error) {
	kek := &storage.KEKRecord{
		Name:     input.Name,
		KmsProps: input.KmsProps,
		Doc:      input.Doc,
		Shared:   input.Shared,
	}
	if err := s.registry.UpdateKEK(ctx, kek); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(kek)
}

type deleteKEKInput struct {
	Name      string `json:"name"`
	Permanent bool   `json:"permanent,omitempty"`
}

func (s *Server) handleDeleteKEK(ctx context.Context, _ *gomcp.CallToolRequest, input deleteKEKInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.DeleteKEK(ctx, input.Name, input.Permanent); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"deleted": true})
}

type undeleteKEKInput struct {
	Name string `json:"name"`
}

func (s *Server) handleUndeleteKEK(ctx context.Context, _ *gomcp.CallToolRequest, input undeleteKEKInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.UndeleteKEK(ctx, input.Name); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"undeleted": true})
}

type listKEKsInput struct {
	Deleted bool `json:"deleted,omitempty"`
}

func (s *Server) handleListKEKs(ctx context.Context, _ *gomcp.CallToolRequest, input listKEKsInput) (*gomcp.CallToolResult, any, error) {
	keks, err := s.registry.ListKEKs(ctx, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if keks == nil {
		keks = []*storage.KEKRecord{}
	}
	return jsonResult(keks)
}

// --- DEK handlers ---

type createDEKInput struct {
	KEKName              string `json:"kek_name"`
	Subject              string `json:"subject"`
	Version              int    `json:"version,omitempty"`
	Algorithm            string `json:"algorithm,omitempty"`
	EncryptedKeyMaterial string `json:"encrypted_key_material,omitempty"`
}

func (s *Server) handleCreateDEK(ctx context.Context, _ *gomcp.CallToolRequest, input createDEKInput) (*gomcp.CallToolResult, any, error) {
	dek := &storage.DEKRecord{
		KEKName:              input.KEKName,
		Subject:              input.Subject,
		Version:              input.Version,
		Algorithm:            input.Algorithm,
		EncryptedKeyMaterial: input.EncryptedKeyMaterial,
	}
	if err := s.registry.CreateDEK(ctx, dek); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(dek)
}

type getDEKInput struct {
	KEKName   string `json:"kek_name"`
	Subject   string `json:"subject"`
	Version   int    `json:"version,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
}

func (s *Server) handleGetDEK(ctx context.Context, _ *gomcp.CallToolRequest, input getDEKInput) (*gomcp.CallToolResult, any, error) {
	version := input.Version
	if version == 0 {
		version = 1
	}
	dek, err := s.registry.GetDEK(ctx, input.KEKName, input.Subject, version, input.Algorithm, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(dek)
}

type listDEKsInput struct {
	KEKName string `json:"kek_name"`
	Deleted bool   `json:"deleted,omitempty"`
}

func (s *Server) handleListDEKs(ctx context.Context, _ *gomcp.CallToolRequest, input listDEKsInput) (*gomcp.CallToolResult, any, error) {
	subjects, err := s.registry.ListDEKs(ctx, input.KEKName, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if subjects == nil {
		subjects = []string{}
	}
	return jsonResult(subjects)
}

type listDEKVersionsInput struct {
	KEKName   string `json:"kek_name"`
	Subject   string `json:"subject"`
	Algorithm string `json:"algorithm,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
}

func (s *Server) handleListDEKVersions(ctx context.Context, _ *gomcp.CallToolRequest, input listDEKVersionsInput) (*gomcp.CallToolResult, any, error) {
	versions, err := s.registry.ListDEKVersions(ctx, input.KEKName, input.Subject, input.Algorithm, input.Deleted)
	if err != nil {
		return errorResult(err), nil, nil
	}
	if versions == nil {
		versions = []int{}
	}
	return jsonResult(versions)
}

type deleteDEKInput struct {
	KEKName   string `json:"kek_name"`
	Subject   string `json:"subject"`
	Version   int    `json:"version,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
	Permanent bool   `json:"permanent,omitempty"`
}

func (s *Server) handleDeleteDEK(ctx context.Context, _ *gomcp.CallToolRequest, input deleteDEKInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.DeleteDEK(ctx, input.KEKName, input.Subject, input.Version, input.Algorithm, input.Permanent); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"deleted": true})
}

type undeleteDEKInput struct {
	KEKName   string `json:"kek_name"`
	Subject   string `json:"subject"`
	Version   int    `json:"version,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
}

func (s *Server) handleUndeleteDEK(ctx context.Context, _ *gomcp.CallToolRequest, input undeleteDEKInput) (*gomcp.CallToolResult, any, error) {
	if err := s.registry.UndeleteDEK(ctx, input.KEKName, input.Subject, input.Version, input.Algorithm); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"undeleted": true})
}

type rewrapDEKInput struct {
	KEKName   string `json:"kek_name"`
	Subject   string `json:"subject"`
	Version   int    `json:"version,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
}

func (s *Server) handleRewrapDEK(ctx context.Context, _ *gomcp.CallToolRequest, input rewrapDEKInput) (*gomcp.CallToolResult, any, error) {
	dek, err := s.registry.RewrapDEK(ctx, input.KEKName, input.Subject, input.Version, input.Algorithm)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(dek)
}
