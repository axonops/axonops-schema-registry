package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

func setupTestAccountHandler(t *testing.T) (*AccountHandler, *auth.Service) {
	t.Helper()
	store := memory.NewStore()
	svc := auth.NewServiceWithConfig(store, auth.ServiceConfig{})
	t.Cleanup(func() { svc.Close() })
	return NewAccountHandler(svc), svc
}

// --- GetCurrentUser ---

func TestGetCurrentUser_Success(t *testing.T) {
	h, svc := setupTestAccountHandler(t)

	// Create a user in the DB
	user, err := svc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "alice",
		Password: "pass123",
		Role:     "admin",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/me", h.GetCurrentUser)

	req := httptest.NewRequest("GET", "/me", nil)
	req = withUser(req, &auth.User{ID: user.ID, Username: "alice", Role: "admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.UserResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Username != "alice" {
		t.Errorf("expected username alice, got %s", resp.Username)
	}
}

func TestGetCurrentUser_NoAuth(t *testing.T) {
	h, _ := setupTestAccountHandler(t)

	r := chi.NewRouter()
	r.Get("/me", h.GetCurrentUser)

	req := httptest.NewRequest("GET", "/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetCurrentUser_UserNotInDB(t *testing.T) {
	h, _ := setupTestAccountHandler(t)

	r := chi.NewRouter()
	r.Get("/me", h.GetCurrentUser)

	// User with ID that doesn't exist in DB
	req := httptest.NewRequest("GET", "/me", nil)
	req = withUser(req, &auth.User{ID: 999, Username: "ghost", Role: "admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- ChangePassword ---

func TestChangePassword_Success(t *testing.T) {
	h, svc := setupTestAccountHandler(t)

	user, err := svc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "alice",
		Password: "oldpass123",
		Role:     "admin",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	r := chi.NewRouter()
	r.Post("/me/password", h.ChangePassword)

	body := types.ChangePasswordRequest{OldPassword: "oldpass123", NewPassword: "newpass456"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/me/password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, &auth.User{ID: user.ID, Username: "alice", Role: "admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangePassword_NoAuth(t *testing.T) {
	h, _ := setupTestAccountHandler(t)

	r := chi.NewRouter()
	r.Post("/me/password", h.ChangePassword)

	body := types.ChangePasswordRequest{OldPassword: "old", NewPassword: "new"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/me/password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	h, svc := setupTestAccountHandler(t)

	user, err := svc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "alice",
		Password: "correct-password",
		Role:     "admin",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	r := chi.NewRouter()
	r.Post("/me/password", h.ChangePassword)

	body := types.ChangePasswordRequest{OldPassword: "wrong-password", NewPassword: "new"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/me/password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, &auth.User{ID: user.ID, Username: "alice", Role: "admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangePassword_MissingOldPassword(t *testing.T) {
	h, _ := setupTestAccountHandler(t)

	r := chi.NewRouter()
	r.Post("/me/password", h.ChangePassword)

	body := types.ChangePasswordRequest{NewPassword: "new"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/me/password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, &auth.User{ID: 1, Username: "alice", Role: "admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChangePassword_MissingNewPassword(t *testing.T) {
	h, _ := setupTestAccountHandler(t)

	r := chi.NewRouter()
	r.Post("/me/password", h.ChangePassword)

	body := types.ChangePasswordRequest{OldPassword: "old"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/me/password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, &auth.User{ID: 1, Username: "alice", Role: "admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChangePassword_InvalidBody(t *testing.T) {
	h, _ := setupTestAccountHandler(t)

	r := chi.NewRouter()
	r.Post("/me/password", h.ChangePassword)

	req := httptest.NewRequest("POST", "/me/password", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, &auth.User{ID: 1, Username: "alice", Role: "admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChangePassword_UserNotInDB(t *testing.T) {
	h, _ := setupTestAccountHandler(t)

	r := chi.NewRouter()
	r.Post("/me/password", h.ChangePassword)

	// User with ID 0 — config-based auth, no DB record
	body := types.ChangePasswordRequest{OldPassword: "old", NewPassword: "new"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/me/password", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, &auth.User{ID: 0, Username: "config-user", Role: "admin"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
