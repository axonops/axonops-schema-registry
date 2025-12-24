// Package handlers provides HTTP request handlers.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// AdminHandler provides HTTP handlers for admin operations.
type AdminHandler struct {
	authService *auth.Service
	authorizer  *auth.Authorizer
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(authService *auth.Service, authorizer *auth.Authorizer) *AdminHandler {
	return &AdminHandler{
		authService: authService,
		authorizer:  authorizer,
	}
}

// ListUsers handles GET /admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRead(w, r) {
		return
	}

	users, err := h.authService.ListUsers(r.Context())
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	resp := types.UsersListResponse{
		Users: make([]types.UserResponse, 0, len(users)),
	}
	for _, u := range users {
		resp.Users = append(resp.Users, userToResponse(u))
	}

	writeAdminJSON(w, http.StatusOK, resp)
}

// CreateUser handles POST /admin/users
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminWrite(w, r) {
		return
	}

	var req types.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if req.Username == "" {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Username is required")
		return
	}
	if req.Password == "" {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidPassword, "Password is required")
		return
	}
	if req.Role == "" {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidRole, "Role is required")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	user, err := h.authService.CreateUser(r.Context(), auth.CreateUserRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		Enabled:  enabled,
	})
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			writeAdminError(w, http.StatusConflict, types.ErrorCodeUserExists, "User already exists")
			return
		}
		if errors.Is(err, storage.ErrInvalidRole) {
			writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidRole, err.Error())
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeAdminJSON(w, http.StatusCreated, userToResponse(user))
}

// GetUser handles GET /admin/users/{id}
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRead(w, r) {
		return
	}

	id, err := parseUserID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid user ID")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			writeAdminError(w, http.StatusNotFound, types.ErrorCodeUserNotFound, "User not found")
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeAdminJSON(w, http.StatusOK, userToResponse(user))
}

// UpdateUser handles PUT /admin/users/{id}
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminWrite(w, r) {
		return
	}

	id, err := parseUserID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid user ID")
		return
	}

	var req types.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	updates := make(map[string]interface{})
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Password != nil {
		updates["password"] = *req.Password
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	user, err := h.authService.UpdateUser(r.Context(), id, updates)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			writeAdminError(w, http.StatusNotFound, types.ErrorCodeUserNotFound, "User not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidRole) {
			writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidRole, err.Error())
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeAdminJSON(w, http.StatusOK, userToResponse(user))
}

// DeleteUser handles DELETE /admin/users/{id}
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminWrite(w, r) {
		return
	}

	id, err := parseUserID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid user ID")
		return
	}

	if err := h.authService.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			writeAdminError(w, http.StatusNotFound, types.ErrorCodeUserNotFound, "User not found")
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListAPIKeys handles GET /admin/apikeys
func (h *AdminHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRead(w, r) {
		return
	}

	// Check if filtering by user
	userIDStr := r.URL.Query().Get("user_id")
	var keys []*storage.APIKeyRecord
	var err error

	if userIDStr != "" {
		userID, parseErr := strconv.ParseInt(userIDStr, 10, 64)
		if parseErr != nil {
			writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid user ID")
			return
		}
		keys, err = h.authService.ListAPIKeysByUserID(r.Context(), userID)
	} else {
		keys, err = h.authService.ListAPIKeys(r.Context())
	}

	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	resp := types.APIKeysListResponse{
		APIKeys: make([]types.APIKeyResponse, 0, len(keys)),
	}
	for _, k := range keys {
		resp.APIKeys = append(resp.APIKeys, h.apiKeyToResponse(r.Context(), k))
	}

	writeAdminJSON(w, http.StatusOK, resp)
}

// CreateAPIKey handles POST /admin/apikeys
func (h *AdminHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminWrite(w, r) {
		return
	}

	var req types.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Name is required")
		return
	}
	if req.Role == "" {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidRole, "Role is required")
		return
	}
	if req.ExpiresIn <= 0 {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "expires_in is required and must be positive (duration in seconds)")
		return
	}

	// Get the authenticated user
	currentUser := auth.GetUser(r.Context())
	if currentUser == nil {
		writeAdminError(w, http.StatusUnauthorized, types.ErrorCodeUnauthorized, "Authentication required")
		return
	}

	// Determine the owner of the API key
	var ownerUserID int64
	var ownerUsername string

	if req.ForUserID != nil && *req.ForUserID > 0 {
		// Creating for another user - only super_admin can do this
		if !h.authorizer.HasPermission(currentUser, auth.PermissionAdminWrite) || currentUser.Role != string(auth.RoleSuperAdmin) {
			writeAdminError(w, http.StatusForbidden, types.ErrorCodeForbidden, "Only super admins can create API keys for other users")
			return
		}
		// Verify the target user exists
		targetUser, err := h.authService.GetUserByID(r.Context(), *req.ForUserID)
		if err != nil {
			writeAdminError(w, http.StatusBadRequest, types.ErrorCodeUserNotFound, "Target user not found")
			return
		}
		ownerUserID = targetUser.ID
		ownerUsername = targetUser.Username
	} else {
		// Creating for self - use the authenticated user's ID
		ownerUserID = currentUser.ID
		ownerUsername = currentUser.Username

		// If the user ID is 0 (e.g., authenticated via config-based auth), we can't create API keys
		if ownerUserID <= 0 {
			writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Cannot create API key: user not in database")
			return
		}
	}

	expiresAt := time.Now().UTC().Add(time.Duration(req.ExpiresIn) * time.Second)

	result, err := h.authService.CreateAPIKey(r.Context(), auth.CreateAPIKeyRequest{
		UserID:    ownerUserID,
		Name:      req.Name,
		Role:      req.Role,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		if errors.Is(err, storage.ErrInvalidRole) {
			writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidRole, err.Error())
			return
		}
		if errors.Is(err, storage.ErrAPIKeyNameExists) {
			writeAdminError(w, http.StatusConflict, types.ErrorCodeAPIKeyExists, "API key name already exists for this user")
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	resp := types.CreateAPIKeyResponse{
		ID:        result.ID,
		Key:       result.Key,
		KeyPrefix: result.KeyPrefix,
		Name:      result.Name,
		Role:      result.Role,
		UserID:    result.UserID,
		Username:  ownerUsername,
		Enabled:   result.Enabled,
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	}

	writeAdminJSON(w, http.StatusCreated, resp)
}

// GetAPIKey handles GET /admin/apikeys/{id}
func (h *AdminHandler) GetAPIKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRead(w, r) {
		return
	}

	id, err := parseAPIKeyID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid API key ID")
		return
	}

	key, err := h.authService.GetAPIKeyByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrAPIKeyNotFound) {
			writeAdminError(w, http.StatusNotFound, types.ErrorCodeAPIKeyNotFound, "API key not found")
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeAdminJSON(w, http.StatusOK, h.apiKeyToResponse(r.Context(), key))
}

// UpdateAPIKey handles PUT /admin/apikeys/{id}
func (h *AdminHandler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminWrite(w, r) {
		return
	}

	id, err := parseAPIKeyID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid API key ID")
		return
	}

	var req types.UpdateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	key, err := h.authService.UpdateAPIKey(r.Context(), id, updates)
	if err != nil {
		if errors.Is(err, storage.ErrAPIKeyNotFound) {
			writeAdminError(w, http.StatusNotFound, types.ErrorCodeAPIKeyNotFound, "API key not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidRole) {
			writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidRole, err.Error())
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeAdminJSON(w, http.StatusOK, h.apiKeyToResponse(r.Context(), key))
}

// DeleteAPIKey handles DELETE /admin/apikeys/{id}
func (h *AdminHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminWrite(w, r) {
		return
	}

	id, err := parseAPIKeyID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid API key ID")
		return
	}

	if err := h.authService.DeleteAPIKey(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrAPIKeyNotFound) {
			writeAdminError(w, http.StatusNotFound, types.ErrorCodeAPIKeyNotFound, "API key not found")
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RevokeAPIKey handles POST /admin/apikeys/{id}/revoke
func (h *AdminHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminWrite(w, r) {
		return
	}

	id, err := parseAPIKeyID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid API key ID")
		return
	}

	if err := h.authService.RevokeAPIKey(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrAPIKeyNotFound) {
			writeAdminError(w, http.StatusNotFound, types.ErrorCodeAPIKeyNotFound, "API key not found")
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	key, err := h.authService.GetAPIKeyByID(r.Context(), id)
	if err != nil {
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeAdminJSON(w, http.StatusOK, h.apiKeyToResponse(r.Context(), key))
}

// RotateAPIKeyRequest is the request body for rotating an API key.
type RotateAPIKeyRequest struct {
	ExpiresIn int64 `json:"expires_in"` // Required: expiry duration in seconds for the new key
}

// RotateAPIKey handles POST /admin/apikeys/{id}/rotate
func (h *AdminHandler) RotateAPIKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminWrite(w, r) {
		return
	}

	id, err := parseAPIKeyID(r)
	if err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid API key ID")
		return
	}

	var req RotateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if req.ExpiresIn <= 0 {
		writeAdminError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "expires_in is required and must be positive (duration in seconds)")
		return
	}

	newExpiresAt := time.Now().UTC().Add(time.Duration(req.ExpiresIn) * time.Second)

	result, err := h.authService.RotateAPIKey(r.Context(), id, newExpiresAt)
	if err != nil {
		if errors.Is(err, storage.ErrAPIKeyNotFound) {
			writeAdminError(w, http.StatusNotFound, types.ErrorCodeAPIKeyNotFound, "API key not found")
			return
		}
		writeAdminError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	// Get username for the response
	username := ""
	if user, err := h.authService.GetUserByID(r.Context(), result.UserID); err == nil {
		username = user.Username
	}

	newKeyResp := types.CreateAPIKeyResponse{
		ID:        result.ID,
		Key:       result.Key,
		KeyPrefix: result.KeyPrefix,
		Name:      result.Name,
		Role:      result.Role,
		UserID:    result.UserID,
		Username:  username,
		Enabled:   result.Enabled,
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	}

	resp := types.RotateAPIKeyResponse{
		NewKey:    newKeyResp,
		RevokedID: id,
	}

	writeAdminJSON(w, http.StatusOK, resp)
}

// ListRoles handles GET /admin/roles
func (h *AdminHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminRead(w, r) {
		return
	}

	roles := []types.RoleInfo{
		{
			Name:        string(auth.RoleSuperAdmin),
			Description: "Full access to everything including user management",
			Permissions: permissionsToStrings(auth.GetRolePermissions(auth.RoleSuperAdmin)),
		},
		{
			Name:        string(auth.RoleAdmin),
			Description: "Can manage schemas, configuration, and view admin info",
			Permissions: permissionsToStrings(auth.GetRolePermissions(auth.RoleAdmin)),
		},
		{
			Name:        string(auth.RoleDeveloper),
			Description: "Can register and read schemas",
			Permissions: permissionsToStrings(auth.GetRolePermissions(auth.RoleDeveloper)),
		},
		{
			Name:        string(auth.RoleReadOnly),
			Description: "Can only read schemas and configuration",
			Permissions: permissionsToStrings(auth.GetRolePermissions(auth.RoleReadOnly)),
		},
	}

	writeAdminJSON(w, http.StatusOK, types.RolesListResponse{Roles: roles})
}

// Helper functions

func (h *AdminHandler) requireAdminRead(w http.ResponseWriter, r *http.Request) bool {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeAdminError(w, http.StatusUnauthorized, types.ErrorCodeUnauthorized, "Authentication required")
		return false
	}
	if !h.authorizer.HasPermission(user, auth.PermissionAdminRead) {
		writeAdminError(w, http.StatusForbidden, types.ErrorCodeForbidden, "Admin read permission required")
		return false
	}
	return true
}

func (h *AdminHandler) requireAdminWrite(w http.ResponseWriter, r *http.Request) bool {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeAdminError(w, http.StatusUnauthorized, types.ErrorCodeUnauthorized, "Authentication required")
		return false
	}
	if !h.authorizer.HasPermission(user, auth.PermissionAdminWrite) {
		writeAdminError(w, http.StatusForbidden, types.ErrorCodeForbidden, "Admin write permission required")
		return false
	}
	return true
}

func parseUserID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	return strconv.ParseInt(idStr, 10, 64)
}

func parseAPIKeyID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	return strconv.ParseInt(idStr, 10, 64)
}

func userToResponse(u *storage.UserRecord) types.UserResponse {
	return types.UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Enabled:   u.Enabled,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AdminHandler) apiKeyToResponse(ctx context.Context, k *storage.APIKeyRecord) types.APIKeyResponse {
	// Get username for the response
	username := ""
	if user, err := h.authService.GetUserByID(ctx, k.UserID); err == nil {
		username = user.Username
	}

	resp := types.APIKeyResponse{
		ID:        k.ID,
		KeyPrefix: k.KeyPrefix,
		Name:      k.Name,
		Role:      k.Role,
		UserID:    k.UserID,
		Username:  username,
		Enabled:   k.Enabled,
		CreatedAt: k.CreatedAt.Format(time.RFC3339),
		ExpiresAt: k.ExpiresAt.Format(time.RFC3339),
	}
	if k.LastUsed != nil {
		lastUsed := k.LastUsed.Format(time.RFC3339)
		resp.LastUsed = &lastUsed
	}
	return resp
}

func permissionsToStrings(perms []auth.Permission) []string {
	result := make([]string, len(perms))
	for i, p := range perms {
		result[i] = string(p)
	}
	return result
}

func writeAdminJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeAdminError(w http.ResponseWriter, status int, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(types.ErrorResponse{
		ErrorCode: code,
		Message:   message,
	})
}
