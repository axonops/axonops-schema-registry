package mcp

import (
	"context"
	"fmt"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func (s *Server) registerAdminTools() {
	if s.authService == nil {
		return
	}

	// User management
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "list_users",
		Description: "List all users in the schema registry.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_users", s.handleListUsers))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "create_user",
		Description: "Create a new user. Requires username, password, and role (super_admin, admin, developer, readonly).",
	}, instrumentedHandler(s, "create_user", s.handleCreateUser))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_user",
		Description: "Get a user by ID.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_user", s.handleGetUser))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "update_user",
		Description: "Update a user's email, password, role, or enabled status.",
	}, instrumentedHandler(s, "update_user", s.handleUpdateUser))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "delete_user",
		Description: "Delete a user by ID.",
		Annotations: &gomcp.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, instrumentedHandler(s, "delete_user", s.handleDeleteUser))

	// API key management
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "list_apikeys",
		Description: "List all API keys, optionally filtered by user_id.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_apikeys", s.handleListAPIKeys))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "create_apikey",
		Description: "Create a new API key for a user. Returns the raw key (only shown once). Requires user_id, name, role, and expires_in (seconds).",
	}, instrumentedHandler(s, "create_apikey", s.handleCreateAPIKey))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_apikey",
		Description: "Get an API key by ID.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_apikey", s.handleGetAPIKey))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "update_apikey",
		Description: "Update an API key's name, role, or enabled status.",
	}, instrumentedHandler(s, "update_apikey", s.handleUpdateAPIKey))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "delete_apikey",
		Description: "Delete an API key by ID.",
		Annotations: &gomcp.ToolAnnotations{DestructiveHint: boolPtr(true)},
	}, instrumentedHandler(s, "delete_apikey", s.handleDeleteAPIKey))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "revoke_apikey",
		Description: "Revoke (disable) an API key without deleting it.",
	}, instrumentedHandler(s, "revoke_apikey", s.handleRevokeAPIKey))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "change_password",
		Description: "Change a user's password. Requires the user's ID, old password, and new password.",
	}, instrumentedHandler(s, "change_password", s.handleChangePassword))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "rotate_apikey",
		Description: "Rotate an API key: creates a new key with the same settings and revokes the old one. Returns the new raw key (only shown once). Requires id and expires_in (seconds).",
	}, instrumentedHandler(s, "rotate_apikey", s.handleRotateAPIKey))

	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "get_user_by_username",
		Description: "Get a user by username.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_user_by_username", s.handleGetUserByUsername))

	// Roles
	gomcp.AddTool(s.mcpServer, &gomcp.Tool{
		Name:        "list_roles",
		Description: "List all available RBAC roles with their permissions.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_roles", s.handleListRoles))
}

func boolPtr(b bool) *bool { return &b }

// --- User handlers ---

type listUsersInput struct{}

func (s *Server) handleListUsers(ctx context.Context, _ *gomcp.CallToolRequest, _ listUsersInput) (*gomcp.CallToolResult, any, error) {
	users, err := s.authService.ListUsers(ctx)
	if err != nil {
		return errorResult(err), nil, nil
	}
	type userResp struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email,omitempty"`
		Role     string `json:"role"`
		Enabled  bool   `json:"enabled"`
	}
	resp := make([]userResp, len(users))
	for i, u := range users {
		resp[i] = userResp{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			Role:     u.Role,
			Enabled:  u.Enabled,
		}
	}
	return jsonResult(resp)
}

type createUserInput struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

func (s *Server) handleCreateUser(ctx context.Context, _ *gomcp.CallToolRequest, input createUserInput) (*gomcp.CallToolResult, any, error) {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	user, err := s.authService.CreateUser(ctx, auth.CreateUserRequest{
		Username: input.Username,
		Email:    input.Email,
		Password: input.Password,
		Role:     input.Role,
		Enabled:  enabled,
	})
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]any{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"enabled":  user.Enabled,
	})
}

type getUserInput struct {
	ID int64 `json:"id"`
}

func (s *Server) handleGetUser(ctx context.Context, _ *gomcp.CallToolRequest, input getUserInput) (*gomcp.CallToolResult, any, error) {
	user, err := s.authService.GetUserByID(ctx, input.ID)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]any{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"enabled":  user.Enabled,
	})
}

type updateUserInput struct {
	ID       int64   `json:"id"`
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
	Role     *string `json:"role,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
}

func (s *Server) handleUpdateUser(ctx context.Context, _ *gomcp.CallToolRequest, input updateUserInput) (*gomcp.CallToolResult, any, error) {
	updates := make(map[string]interface{})
	if input.Email != nil {
		updates["email"] = *input.Email
	}
	if input.Password != nil {
		updates["password"] = *input.Password
	}
	if input.Role != nil {
		updates["role"] = *input.Role
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}

	user, err := s.authService.UpdateUser(ctx, input.ID, updates)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]any{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"enabled":  user.Enabled,
	})
}

type deleteUserInput struct {
	ID int64 `json:"id"`
}

func (s *Server) handleDeleteUser(ctx context.Context, _ *gomcp.CallToolRequest, input deleteUserInput) (*gomcp.CallToolResult, any, error) {
	if err := s.authService.DeleteUser(ctx, input.ID); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"deleted": true})
}

// --- API key handlers ---

type listAPIKeysInput struct {
	UserID int64 `json:"user_id,omitempty"`
}

func (s *Server) handleListAPIKeys(ctx context.Context, _ *gomcp.CallToolRequest, input listAPIKeysInput) (*gomcp.CallToolResult, any, error) {
	var keys []*apiKeyResp
	if input.UserID > 0 {
		records, err := s.authService.ListAPIKeysByUserID(ctx, input.UserID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		keys = toAPIKeyResps(records)
	} else {
		records, err := s.authService.ListAPIKeys(ctx)
		if err != nil {
			return errorResult(err), nil, nil
		}
		keys = toAPIKeyResps(records)
	}
	return jsonResult(keys)
}

type createAPIKeyInput struct {
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	ExpiresIn int64  `json:"expires_in"` // seconds
}

func (s *Server) handleCreateAPIKey(ctx context.Context, _ *gomcp.CallToolRequest, input createAPIKeyInput) (*gomcp.CallToolResult, any, error) {
	if input.ExpiresIn <= 0 {
		return errorResult(fmt.Errorf("expires_in is required and must be positive (duration in seconds)")), nil, nil
	}
	expiresAt := time.Now().UTC().Add(time.Duration(input.ExpiresIn) * time.Second)
	result, err := s.authService.CreateAPIKey(ctx, auth.CreateAPIKeyRequest{
		UserID:    input.UserID,
		Name:      input.Name,
		Role:      input.Role,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]any{
		"id":         result.ID,
		"key":        result.Key,
		"key_prefix": result.KeyPrefix,
		"name":       result.Name,
		"role":       result.Role,
		"user_id":    result.UserID,
		"enabled":    result.Enabled,
		"expires_at": result.ExpiresAt.Format(time.RFC3339),
	})
}

type getAPIKeyInput struct {
	ID int64 `json:"id"`
}

func (s *Server) handleGetAPIKey(ctx context.Context, _ *gomcp.CallToolRequest, input getAPIKeyInput) (*gomcp.CallToolResult, any, error) {
	key, err := s.authService.GetAPIKeyByID(ctx, input.ID)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(toAPIKeyResp(key))
}

type updateAPIKeyInput struct {
	ID      int64   `json:"id"`
	Name    *string `json:"name,omitempty"`
	Role    *string `json:"role,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

func (s *Server) handleUpdateAPIKey(ctx context.Context, _ *gomcp.CallToolRequest, input updateAPIKeyInput) (*gomcp.CallToolResult, any, error) {
	updates := make(map[string]interface{})
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Role != nil {
		updates["role"] = *input.Role
	}
	if input.Enabled != nil {
		updates["enabled"] = *input.Enabled
	}

	key, err := s.authService.UpdateAPIKey(ctx, input.ID, updates)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(toAPIKeyResp(key))
}

type deleteAPIKeyInput struct {
	ID int64 `json:"id"`
}

func (s *Server) handleDeleteAPIKey(ctx context.Context, _ *gomcp.CallToolRequest, input deleteAPIKeyInput) (*gomcp.CallToolResult, any, error) {
	if err := s.authService.DeleteAPIKey(ctx, input.ID); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"deleted": true})
}

type revokeAPIKeyInput struct {
	ID int64 `json:"id"`
}

func (s *Server) handleRevokeAPIKey(ctx context.Context, _ *gomcp.CallToolRequest, input revokeAPIKeyInput) (*gomcp.CallToolResult, any, error) {
	if err := s.authService.RevokeAPIKey(ctx, input.ID); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"revoked": true})
}

// --- Password & key rotation handlers ---

type changePasswordInput struct {
	ID          int64  `json:"id"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (s *Server) handleChangePassword(ctx context.Context, _ *gomcp.CallToolRequest, input changePasswordInput) (*gomcp.CallToolResult, any, error) {
	if err := s.authService.ChangePassword(ctx, input.ID, input.OldPassword, input.NewPassword); err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]bool{"changed": true})
}

type rotateAPIKeyInput struct {
	ID        int64 `json:"id"`
	ExpiresIn int64 `json:"expires_in"` // seconds
}

func (s *Server) handleRotateAPIKey(ctx context.Context, _ *gomcp.CallToolRequest, input rotateAPIKeyInput) (*gomcp.CallToolResult, any, error) {
	if input.ExpiresIn <= 0 {
		return errorResult(fmt.Errorf("expires_in is required and must be positive (duration in seconds)")), nil, nil
	}
	expiresAt := time.Now().UTC().Add(time.Duration(input.ExpiresIn) * time.Second)
	result, err := s.authService.RotateAPIKey(ctx, input.ID, expiresAt)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]any{
		"id":         result.ID,
		"key":        result.Key,
		"key_prefix": result.KeyPrefix,
		"name":       result.Name,
		"role":       result.Role,
		"user_id":    result.UserID,
		"enabled":    result.Enabled,
		"expires_at": result.ExpiresAt.Format(time.RFC3339),
	})
}

type getUserByUsernameInput struct {
	Username string `json:"username"`
}

func (s *Server) handleGetUserByUsername(ctx context.Context, _ *gomcp.CallToolRequest, input getUserByUsernameInput) (*gomcp.CallToolResult, any, error) {
	user, err := s.authService.GetUserByUsername(ctx, input.Username)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return jsonResult(map[string]any{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
		"enabled":  user.Enabled,
	})
}

// --- Roles handler ---

type listRolesInput struct{}

func (s *Server) handleListRoles(_ context.Context, _ *gomcp.CallToolRequest, _ listRolesInput) (*gomcp.CallToolResult, any, error) {
	roles := []map[string]any{
		{"name": "super_admin", "description": "Full access to everything including user management"},
		{"name": "admin", "description": "Can manage schemas, configuration, and view admin info"},
		{"name": "developer", "description": "Can register and read schemas"},
		{"name": "readonly", "description": "Can only read schemas and configuration"},
	}
	return jsonResult(roles)
}

// --- API key response helpers ---

type apiKeyResp struct {
	ID        int64  `json:"id"`
	KeyPrefix string `json:"key_prefix"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	UserID    int64  `json:"user_id"`
	Enabled   bool   `json:"enabled"`
	ExpiresAt string `json:"expires_at"`
}

func toAPIKeyResp(k *storage.APIKeyRecord) *apiKeyResp {
	return &apiKeyResp{
		ID:        k.ID,
		KeyPrefix: k.KeyPrefix,
		Name:      k.Name,
		Role:      k.Role,
		UserID:    k.UserID,
		Enabled:   k.Enabled,
		ExpiresAt: k.ExpiresAt.Format(time.RFC3339),
	}
}

func toAPIKeyResps(keys []*storage.APIKeyRecord) []*apiKeyResp {
	resp := make([]*apiKeyResp, len(keys))
	for i, k := range keys {
		resp[i] = toAPIKeyResp(k)
	}
	return resp
}
