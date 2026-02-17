package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

func setupTestAdminHandler(t *testing.T) (*AdminHandler, *auth.Service) {
	t.Helper()
	store := memory.NewStore()
	svc := auth.NewServiceWithConfig(store, auth.ServiceConfig{})
	t.Cleanup(func() { svc.Close() })
	authz := auth.NewAuthorizer(config.RBACConfig{
		Enabled:     true,
		DefaultRole: "readonly",
	})
	return NewAdminHandler(svc, authz), svc
}

func withUser(req *http.Request, user *auth.User) *http.Request {
	ctx := context.WithValue(req.Context(), auth.UserContextKey, user)
	return req.WithContext(ctx)
}

func superAdmin() *auth.User {
	return &auth.User{ID: 1, Username: "admin", Role: "super_admin", Method: "basic"}
}

func adminUser() *auth.User {
	return &auth.User{ID: 2, Username: "admin2", Role: "admin", Method: "basic"}
}

func readonlyUser() *auth.User {
	return &auth.User{ID: 3, Username: "reader", Role: "readonly", Method: "basic"}
}

func developerUser() *auth.User {
	return &auth.User{ID: 4, Username: "dev", Role: "developer", Method: "basic"}
}

func createTestUser(t *testing.T, h *AdminHandler, username, role string) int64 {
	t.Helper()
	r := chi.NewRouter()
	r.Post("/admin/users", h.CreateUser)

	body := types.CreateUserRequest{Username: username, Password: "password123", Role: role}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("createTestUser failed: %d %s", w.Code, w.Body.String())
	}
	var resp types.UserResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.ID
}

// --- Permission Checks ---

func TestAdmin_NoAuth_Returns401(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/users", h.ListUsers)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAdmin_ReadOnly_Returns403(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/users", h.ListUsers)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req = withUser(req, readonlyUser())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAdmin_Developer_Returns403(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/users", h.ListUsers)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req = withUser(req, developerUser())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAdmin_Admin_CanRead(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/users", h.ListUsers)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req = withUser(req, adminUser())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- ListUsers ---

func TestListUsers_Empty(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/users", h.ListUsers)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.UsersListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(resp.Users))
	}
}

func TestListUsers_WithUsers(t *testing.T) {
	h, _ := setupTestAdminHandler(t)
	createTestUser(t, h, "alice", "admin")
	createTestUser(t, h, "bob", "readonly")

	r := chi.NewRouter()
	r.Get("/admin/users", h.ListUsers)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp types.UsersListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(resp.Users))
	}
}

// --- CreateUser ---

func TestCreateUser_Success(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/users", h.CreateUser)

	body := types.CreateUserRequest{Username: "alice", Password: "pass123", Role: "admin"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.UserResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Username != "alice" {
		t.Errorf("expected username alice, got %s", resp.Username)
	}
	if resp.Role != "admin" {
		t.Errorf("expected role admin, got %s", resp.Role)
	}
}

func TestCreateUser_MissingUsername(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/users", h.CreateUser)

	body := types.CreateUserRequest{Password: "pass", Role: "admin"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_MissingPassword(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/users", h.CreateUser)

	body := types.CreateUserRequest{Username: "alice", Role: "admin"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_MissingRole(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/users", h.CreateUser)

	body := types.CreateUserRequest{Username: "alice", Password: "pass"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	h, _ := setupTestAdminHandler(t)
	createTestUser(t, h, "alice", "admin")

	r := chi.NewRouter()
	r.Post("/admin/users", h.CreateUser)

	body := types.CreateUserRequest{Username: "alice", Password: "pass", Role: "readonly"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestCreateUser_InvalidBody(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/users", h.CreateUser)

	req := httptest.NewRequest("POST", "/admin/users", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- GetUser ---

func TestGetUser_Found(t *testing.T) {
	h, _ := setupTestAdminHandler(t)
	userID := createTestUser(t, h, "alice", "admin")

	r := chi.NewRouter()
	r.Get("/admin/users/{id}", h.GetUser)

	req := httptest.NewRequest("GET", fmt.Sprintf("/admin/users/%d", userID), nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.UserResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Username != "alice" {
		t.Errorf("expected username alice, got %s", resp.Username)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/users/{id}", h.GetUser)

	req := httptest.NewRequest("GET", "/admin/users/999", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetUser_InvalidID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/users/{id}", h.GetUser)

	req := httptest.NewRequest("GET", "/admin/users/abc", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- UpdateUser ---

func TestUpdateUser_Success(t *testing.T) {
	h, _ := setupTestAdminHandler(t)
	userID := createTestUser(t, h, "alice", "admin")

	r := chi.NewRouter()
	r.Put("/admin/users/{id}", h.UpdateUser)

	email := "alice@example.com"
	body := types.UpdateUserRequest{Email: &email}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/admin/users/%d", userID), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.UserResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %s", resp.Email)
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Put("/admin/users/{id}", h.UpdateUser)

	email := "x@x.com"
	body := types.UpdateUserRequest{Email: &email}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/admin/users/999", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateUser_InvalidBody(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Put("/admin/users/{id}", h.UpdateUser)

	req := httptest.NewRequest("PUT", "/admin/users/1", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateUser_InvalidID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Put("/admin/users/{id}", h.UpdateUser)

	body := types.UpdateUserRequest{}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/admin/users/abc", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- DeleteUser ---

func TestDeleteUser_Success(t *testing.T) {
	h, _ := setupTestAdminHandler(t)
	userID := createTestUser(t, h, "alice", "admin")

	r := chi.NewRouter()
	r.Delete("/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/admin/users/%d", userID), nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Delete("/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest("DELETE", "/admin/users/999", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteUser_InvalidID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Delete("/admin/users/{id}", h.DeleteUser)

	req := httptest.NewRequest("DELETE", "/admin/users/abc", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- ListAPIKeys ---

func TestListAPIKeys_Empty(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/apikeys", h.ListAPIKeys)

	req := httptest.NewRequest("GET", "/admin/apikeys", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.APIKeysListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.APIKeys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(resp.APIKeys))
	}
}

func TestListAPIKeys_InvalidUserID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/apikeys", h.ListAPIKeys)

	req := httptest.NewRequest("GET", "/admin/apikeys?user_id=abc", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- CreateAPIKey ---

func TestCreateAPIKey_Success(t *testing.T) {
	h, _ := setupTestAdminHandler(t)
	userID := createTestUser(t, h, "alice", "admin")

	r := chi.NewRouter()
	r.Post("/admin/apikeys", h.CreateAPIKey)

	body := types.CreateAPIKeyRequest{Name: "my-key", Role: "readonly", ExpiresIn: 86400}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, &auth.User{ID: userID, Username: "alice", Role: "super_admin", Method: "basic"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.CreateAPIKeyResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Key == "" {
		t.Error("expected non-empty key")
	}
	if resp.Name != "my-key" {
		t.Errorf("expected name my-key, got %s", resp.Name)
	}
}

func TestCreateAPIKey_MissingName(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys", h.CreateAPIKey)

	body := types.CreateAPIKeyRequest{Role: "readonly", ExpiresIn: 86400}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateAPIKey_MissingRole(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys", h.CreateAPIKey)

	body := types.CreateAPIKeyRequest{Name: "my-key", ExpiresIn: 86400}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateAPIKey_MissingExpiresIn(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys", h.CreateAPIKey)

	body := types.CreateAPIKeyRequest{Name: "my-key", Role: "readonly"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateAPIKey_NoAuth(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys", h.CreateAPIKey)

	body := types.CreateAPIKeyRequest{Name: "k", Role: "readonly", ExpiresIn: 86400}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateAPIKey_UserNotInDB(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys", h.CreateAPIKey)

	// User with ID 0 (config-based auth, not in DB)
	body := types.CreateAPIKeyRequest{Name: "k", Role: "readonly", ExpiresIn: 86400}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, &auth.User{ID: 0, Username: "config-user", Role: "super_admin", Method: "basic"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// --- GetAPIKey ---

func TestGetAPIKey_NotFound(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/apikeys/{id}", h.GetAPIKey)

	req := httptest.NewRequest("GET", "/admin/apikeys/999", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetAPIKey_InvalidID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/apikeys/{id}", h.GetAPIKey)

	req := httptest.NewRequest("GET", "/admin/apikeys/abc", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- UpdateAPIKey ---

func TestUpdateAPIKey_NotFound(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Put("/admin/apikeys/{id}", h.UpdateAPIKey)

	name := "new-name"
	body := types.UpdateAPIKeyRequest{Name: &name}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/admin/apikeys/999", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdateAPIKey_InvalidBody(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Put("/admin/apikeys/{id}", h.UpdateAPIKey)

	req := httptest.NewRequest("PUT", "/admin/apikeys/1", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateAPIKey_InvalidID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Put("/admin/apikeys/{id}", h.UpdateAPIKey)

	body := types.UpdateAPIKeyRequest{}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/admin/apikeys/abc", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- DeleteAPIKey ---

func TestDeleteAPIKey_NotFound(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Delete("/admin/apikeys/{id}", h.DeleteAPIKey)

	req := httptest.NewRequest("DELETE", "/admin/apikeys/999", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteAPIKey_InvalidID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Delete("/admin/apikeys/{id}", h.DeleteAPIKey)

	req := httptest.NewRequest("DELETE", "/admin/apikeys/abc", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- RevokeAPIKey ---

func TestRevokeAPIKey_NotFound(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys/{id}/revoke", h.RevokeAPIKey)

	req := httptest.NewRequest("POST", "/admin/apikeys/999/revoke", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRevokeAPIKey_InvalidID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys/{id}/revoke", h.RevokeAPIKey)

	req := httptest.NewRequest("POST", "/admin/apikeys/abc/revoke", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- RotateAPIKey ---

func TestRotateAPIKey_NotFound(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys/{id}/rotate", h.RotateAPIKey)

	body := RotateAPIKeyRequest{ExpiresIn: 86400}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys/999/rotate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRotateAPIKey_MissingExpiresIn(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys/{id}/rotate", h.RotateAPIKey)

	body := RotateAPIKeyRequest{ExpiresIn: 0}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys/1/rotate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRotateAPIKey_InvalidBody(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys/{id}/rotate", h.RotateAPIKey)

	req := httptest.NewRequest("POST", "/admin/apikeys/1/rotate", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRotateAPIKey_InvalidID(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Post("/admin/apikeys/{id}/rotate", h.RotateAPIKey)

	body := RotateAPIKeyRequest{ExpiresIn: 86400}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/apikeys/abc/rotate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- ListRoles ---

func TestListRoles(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/roles", h.ListRoles)

	req := httptest.NewRequest("GET", "/admin/roles", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.RolesListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Roles) != 4 {
		t.Errorf("expected 4 roles, got %d", len(resp.Roles))
	}
}

// --- Admin content type ---

func TestAdmin_ContentType(t *testing.T) {
	h, _ := setupTestAdminHandler(t)

	r := chi.NewRouter()
	r.Get("/admin/users", h.ListUsers)

	req := httptest.NewRequest("GET", "/admin/users", nil)
	req = withUser(req, superAdmin())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}
