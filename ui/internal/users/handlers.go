package users

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/axonops/schema-registry-ui/internal/auth"
)

// Handlers provides HTTP handlers for user management.
type Handlers struct {
	service *Service
}

// NewHandlers creates user management handlers.
func NewHandlers(svc *Service) *Handlers {
	return &Handlers{service: svc}
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Enabled  *bool   `json:"enabled,omitempty"`
	Password *string `json:"password,omitempty"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ListUsers handles GET /api/users.
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List()
	if err != nil {
		slog.Error("listing users failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// CreateUser handles POST /api/users.
func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	if err := h.service.Create(req.Username, req.Password); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "characters") || strings.Contains(err.Error(), "invalid") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		slog.Error("creating user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	slog.Info("user created", "username", req.Username, "by", auth.UsernameFromContext(r.Context()))
	writeJSON(w, http.StatusCreated, UserInfo{Username: req.Username, Enabled: true})
}

// UpdateUser handles PUT /api/users/{username}.
func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	username := extractUsername(r.URL.Path)
	if username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username required in path"})
		return
	}

	exists, err := h.service.Exists(username)
	if err != nil {
		slog.Error("checking user existence failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Password != nil {
		if err := h.service.SetPassword(username, *req.Password); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	if req.Enabled != nil {
		if err := h.service.SetEnabled(username, *req.Enabled); err != nil {
			if strings.Contains(err.Error(), "last active") {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	slog.Info("user updated", "username", username, "by", auth.UsernameFromContext(r.Context()))
	w.WriteHeader(http.StatusNoContent)
}

// DeleteUser handles DELETE /api/users/{username}.
func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	username := extractUsername(r.URL.Path)
	if username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username required in path"})
		return
	}

	if err := h.service.Delete(username); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "last user") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		slog.Error("deleting user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	slog.Info("user deleted", "username", username, "by", auth.UsernameFromContext(r.Context()))
	w.WriteHeader(http.StatusNoContent)
}

// ChangeMyPassword handles POST /api/users/me/password.
func (h *Handlers) ChangeMyPassword(w http.ResponseWriter, r *http.Request) {
	username := auth.UsernameFromContext(r.Context())
	if username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new_password required"})
		return
	}

	if err := h.service.SetPassword(username, req.NewPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	slog.Info("user changed password", "username", username)
	w.WriteHeader(http.StatusNoContent)
}

// extractUsername gets the username from the path /api/users/{username}.
func extractUsername(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	// Expected: api/users/{username}
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "users" {
		return parts[2]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
