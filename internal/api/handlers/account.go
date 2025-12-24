// Package handlers provides HTTP request handlers.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// AccountHandler provides HTTP handlers for self-service account operations.
type AccountHandler struct {
	authService *auth.Service
}

// NewAccountHandler creates a new AccountHandler.
func NewAccountHandler(authService *auth.Service) *AccountHandler {
	return &AccountHandler{
		authService: authService,
	}
}

// GetCurrentUser handles GET /me
func (h *AccountHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeAccountError(w, http.StatusUnauthorized, types.ErrorCodeUnauthorized, "Authentication required")
		return
	}

	// Get full user record from database
	fullUser, err := h.authService.GetUserByID(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			writeAccountError(w, http.StatusNotFound, types.ErrorCodeUserNotFound, "User not found")
			return
		}
		writeAccountError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	writeAccountJSON(w, http.StatusOK, userToResponse(fullUser))
}

// ChangePassword handles POST /me/password
func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeAccountError(w, http.StatusUnauthorized, types.ErrorCodeUnauthorized, "Authentication required")
		return
	}

	if user.ID <= 0 {
		writeAccountError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Cannot change password: user not in database")
		return
	}

	var req types.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAccountError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Invalid request body")
		return
	}

	if req.OldPassword == "" {
		writeAccountError(w, http.StatusBadRequest, types.ErrorCodeInvalidSchema, "Old password is required")
		return
	}
	if req.NewPassword == "" {
		writeAccountError(w, http.StatusBadRequest, types.ErrorCodeInvalidPassword, "New password is required")
		return
	}

	err := h.authService.ChangePassword(r.Context(), user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, storage.ErrPermissionDenied) {
			writeAccountError(w, http.StatusForbidden, types.ErrorCodeForbidden, "Current password is incorrect")
			return
		}
		if errors.Is(err, storage.ErrUserNotFound) {
			writeAccountError(w, http.StatusNotFound, types.ErrorCodeUserNotFound, "User not found")
			return
		}
		writeAccountError(w, http.StatusInternalServerError, types.ErrorCodeInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeAccountJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeAccountError(w http.ResponseWriter, status int, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(types.ErrorResponse{
		ErrorCode: code,
		Message:   message,
	})
}
