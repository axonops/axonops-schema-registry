package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
)

// --- KEK/DEK Test Helpers ---

func createKEK(t *testing.T, h *Handler, name, kmsType, kmsKeyID string) {
	t.Helper()
	body := types.CreateKEKRequest{
		Name:     name,
		KmsType:  kmsType,
		KmsKeyID: kmsKeyID,
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/dek-registry/v1/keks", h.CreateKEK)

	req := httptest.NewRequest("POST", "/dek-registry/v1/keks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("createKEK failed: %d %s", w.Code, w.Body.String())
	}
}

func createDEK(t *testing.T, h *Handler, kekName, subject string) {
	t.Helper()
	body := types.CreateDEKRequest{
		Subject: subject,
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/dek-registry/v1/keks/{name}/deks", h.CreateDEK)

	req := httptest.NewRequest("POST", "/dek-registry/v1/keks/"+kekName+"/deks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("createDEK failed: %d %s", w.Code, w.Body.String())
	}
}

// --- KEK Tests ---

func TestListKEKs_Empty(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks", h.ListKEKs)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var names []string
	json.NewDecoder(w.Body).Decode(&names)
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestCreateKEK_Success(t *testing.T) {
	h := setupTestHandler(t)

	body := types.CreateKEKRequest{
		Name:     "my-kek",
		KmsType:  "aws-kms",
		KmsKeyID: "arn:aws:kms:us-east-1:123456789:key/test-key",
		Doc:      "Test KEK",
		Shared:   true,
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/dek-registry/v1/keks", h.CreateKEK)

	req := httptest.NewRequest("POST", "/dek-registry/v1/keks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.KEKResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "my-kek" {
		t.Errorf("expected name my-kek, got %s", resp.Name)
	}
	if resp.KmsType != "aws-kms" {
		t.Errorf("expected kmsType aws-kms, got %s", resp.KmsType)
	}
	if resp.KmsKeyID != "arn:aws:kms:us-east-1:123456789:key/test-key" {
		t.Errorf("expected kmsKeyId arn:aws:kms:us-east-1:123456789:key/test-key, got %s", resp.KmsKeyID)
	}
	if resp.Doc != "Test KEK" {
		t.Errorf("expected doc 'Test KEK', got %s", resp.Doc)
	}
	if !resp.Shared {
		t.Error("expected shared=true")
	}
}

func TestCreateKEK_MissingFields(t *testing.T) {
	h := setupTestHandler(t)

	// Missing kmsType and kmsKeyId
	body := types.CreateKEKRequest{
		Name: "missing-fields-kek",
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/dek-registry/v1/keks", h.CreateKEK)

	req := httptest.NewRequest("POST", "/dek-registry/v1/keks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateKEK_Duplicate(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "dup-kek", "aws-kms", "key-1")

	// Create again with same name
	body := types.CreateKEKRequest{
		Name:     "dup-kek",
		KmsType:  "aws-kms",
		KmsKeyID: "key-2",
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/dek-registry/v1/keks", h.CreateKEK)

	req := httptest.NewRequest("POST", "/dek-registry/v1/keks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeKEKExists {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeKEKExists, resp.ErrorCode)
	}
}

func TestGetKEK_Success(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "get-kek", "gcp-kms", "gcp-key-1")

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks/{name}", h.GetKEK)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks/get-kek", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.KEKResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "get-kek" {
		t.Errorf("expected name get-kek, got %s", resp.Name)
	}
	if resp.KmsType != "gcp-kms" {
		t.Errorf("expected kmsType gcp-kms, got %s", resp.KmsType)
	}
	if resp.KmsKeyID != "gcp-key-1" {
		t.Errorf("expected kmsKeyId gcp-key-1, got %s", resp.KmsKeyID)
	}
}

func TestGetKEK_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks/{name}", h.GetKEK)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeKEKNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeKEKNotFound, resp.ErrorCode)
	}
}

func TestUpdateKEK_Success(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "update-kek", "aws-kms", "key-1")

	shared := true
	body := types.UpdateKEKRequest{
		Doc:    "Updated documentation",
		Shared: &shared,
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Put("/dek-registry/v1/keks/{name}", h.UpdateKEK)

	req := httptest.NewRequest("PUT", "/dek-registry/v1/keks/update-kek", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.KEKResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Doc != "Updated documentation" {
		t.Errorf("expected doc 'Updated documentation', got %s", resp.Doc)
	}
	if !resp.Shared {
		t.Error("expected shared=true after update")
	}
}

func TestUpdateKEK_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	body := types.UpdateKEKRequest{
		Doc: "Does not matter",
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Put("/dek-registry/v1/keks/{name}", h.UpdateKEK)

	req := httptest.NewRequest("PUT", "/dek-registry/v1/keks/nonexistent", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeKEKNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeKEKNotFound, resp.ErrorCode)
	}
}

func TestDeleteKEK_Soft(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "soft-del-kek", "aws-kms", "key-1")

	r := chi.NewRouter()
	r.Delete("/dek-registry/v1/keks/{name}", h.DeleteKEK)
	r.Get("/dek-registry/v1/keks/{name}", h.GetKEK)

	// Soft delete
	reqDel := httptest.NewRequest("DELETE", "/dek-registry/v1/keks/soft-del-kek", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Should not be found without deleted=true
	reqGet := httptest.NewRequest("GET", "/dek-registry/v1/keks/soft-del-kek", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusNotFound {
		t.Errorf("expected 404 without deleted=true, got %d", wGet.Code)
	}

	// Should be found with deleted=true
	reqGetDel := httptest.NewRequest("GET", "/dek-registry/v1/keks/soft-del-kek?deleted=true", nil)
	wGetDel := httptest.NewRecorder()
	r.ServeHTTP(wGetDel, reqGetDel)

	if wGetDel.Code != http.StatusOK {
		t.Errorf("expected 200 with deleted=true, got %d: %s", wGetDel.Code, wGetDel.Body.String())
	}

	var resp types.KEKResponse
	json.NewDecoder(wGetDel.Body).Decode(&resp)
	if !resp.Deleted {
		t.Error("expected deleted=true in response")
	}
}

func TestDeleteKEK_Permanent(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "perm-del-kek", "aws-kms", "key-1")

	r := chi.NewRouter()
	r.Delete("/dek-registry/v1/keks/{name}", h.DeleteKEK)
	r.Get("/dek-registry/v1/keks/{name}", h.GetKEK)

	// Permanent delete
	reqDel := httptest.NewRequest("DELETE", "/dek-registry/v1/keks/perm-del-kek?permanent=true", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Should not be found even with deleted=true
	reqGet := httptest.NewRequest("GET", "/dek-registry/v1/keks/perm-del-kek?deleted=true", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusNotFound {
		t.Errorf("expected 404 after permanent delete, got %d", wGet.Code)
	}
}

func TestUndeleteKEK_Success(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "undel-kek", "aws-kms", "key-1")

	r := chi.NewRouter()
	r.Delete("/dek-registry/v1/keks/{name}", h.DeleteKEK)
	r.Put("/dek-registry/v1/keks/{name}/undelete", h.UndeleteKEK)
	r.Get("/dek-registry/v1/keks/{name}", h.GetKEK)

	// Soft delete
	reqDel := httptest.NewRequest("DELETE", "/dek-registry/v1/keks/undel-kek", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("soft-delete failed: %d", wDel.Code)
	}

	// Undelete
	reqUndel := httptest.NewRequest("PUT", "/dek-registry/v1/keks/undel-kek/undelete", nil)
	wUndel := httptest.NewRecorder()
	r.ServeHTTP(wUndel, reqUndel)

	if wUndel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wUndel.Code, wUndel.Body.String())
	}

	// Should be found again without deleted=true
	reqGet := httptest.NewRequest("GET", "/dek-registry/v1/keks/undel-kek", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Errorf("expected 200 after undelete, got %d", wGet.Code)
	}
}

func TestUndeleteKEK_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/dek-registry/v1/keks/{name}/undelete", h.UndeleteKEK)

	req := httptest.NewRequest("PUT", "/dek-registry/v1/keks/nonexistent/undelete", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeKEKNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeKEKNotFound, resp.ErrorCode)
	}
}

// --- DEK Tests ---

func TestListDEKs_Empty(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "list-dek-kek", "aws-kms", "key-1")

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks/{name}/deks", h.ListDEKs)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks/list-dek-kek/deks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var subjects []string
	json.NewDecoder(w.Body).Decode(&subjects)
	if len(subjects) != 0 {
		t.Errorf("expected empty list, got %v", subjects)
	}
}

func TestCreateDEK_Success(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "dek-create-kek", "aws-kms", "key-1")

	body := types.CreateDEKRequest{
		Subject:              "test-subject",
		Algorithm:            "AES256_GCM",
		EncryptedKeyMaterial: "encrypted-data",
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/dek-registry/v1/keks/{name}/deks", h.CreateDEK)

	req := httptest.NewRequest("POST", "/dek-registry/v1/keks/dek-create-kek/deks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.DEKResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.KEKName != "dek-create-kek" {
		t.Errorf("expected kekName dek-create-kek, got %s", resp.KEKName)
	}
	if resp.Subject != "test-subject" {
		t.Errorf("expected subject test-subject, got %s", resp.Subject)
	}
	if resp.Version != 1 {
		t.Errorf("expected version 1, got %d", resp.Version)
	}
	if resp.Algorithm != "AES256_GCM" {
		t.Errorf("expected algorithm AES256_GCM, got %s", resp.Algorithm)
	}
}

func TestCreateDEK_KEKNotFound(t *testing.T) {
	h := setupTestHandler(t)

	body := types.CreateDEKRequest{
		Subject: "test-subject",
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/dek-registry/v1/keks/{name}/deks", h.CreateDEK)

	req := httptest.NewRequest("POST", "/dek-registry/v1/keks/nonexistent-kek/deks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeKEKNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeKEKNotFound, resp.ErrorCode)
	}
}

func TestGetDEK_Success(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "get-dek-kek", "aws-kms", "key-1")
	createDEK(t, h, "get-dek-kek", "my-subject")

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks/{name}/deks/{subject}", h.GetDEK)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks/get-dek-kek/deks/my-subject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.DEKResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.KEKName != "get-dek-kek" {
		t.Errorf("expected kekName get-dek-kek, got %s", resp.KEKName)
	}
	if resp.Subject != "my-subject" {
		t.Errorf("expected subject my-subject, got %s", resp.Subject)
	}
	if resp.Version != 1 {
		t.Errorf("expected version 1, got %d", resp.Version)
	}
}

func TestGetDEK_NotFound(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "get-dek-nf-kek", "aws-kms", "key-1")

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks/{name}/deks/{subject}", h.GetDEK)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks/get-dek-nf-kek/deks/no-such-subject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeDEKNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeDEKNotFound, resp.ErrorCode)
	}
}

func TestGetDEKVersion_Success(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "ver-dek-kek", "aws-kms", "key-1")
	createDEK(t, h, "ver-dek-kek", "ver-subject")

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks/{name}/deks/{subject}/versions/{version}", h.GetDEKVersion)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks/ver-dek-kek/deks/ver-subject/versions/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.DEKResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Version != 1 {
		t.Errorf("expected version 1, got %d", resp.Version)
	}
	if resp.Subject != "ver-subject" {
		t.Errorf("expected subject ver-subject, got %s", resp.Subject)
	}
}

func TestGetDEKVersion_InvalidVersion(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks/{name}/deks/{subject}/versions/{version}", h.GetDEKVersion)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks/some-kek/deks/some-subject/versions/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListDEKVersions(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "listver-kek", "aws-kms", "key-1")

	// Create multiple DEKs (each gets auto-versioned)
	createDEK(t, h, "listver-kek", "listver-subject")
	createDEK(t, h, "listver-kek", "listver-subject") // version 2

	r := chi.NewRouter()
	r.Get("/dek-registry/v1/keks/{name}/deks/{subject}/versions", h.ListDEKVersions)

	req := httptest.NewRequest("GET", "/dek-registry/v1/keks/listver-kek/deks/listver-subject/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var versions []int
	json.NewDecoder(w.Body).Decode(&versions)
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d: %v", len(versions), versions)
	}
}

func TestDeleteDEK_Soft(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "deldek-kek", "aws-kms", "key-1")
	createDEK(t, h, "deldek-kek", "deldek-subject")

	r := chi.NewRouter()
	r.Delete("/dek-registry/v1/keks/{name}/deks/{subject}", h.DeleteDEK)
	r.Get("/dek-registry/v1/keks/{name}/deks/{subject}", h.GetDEK)

	// Soft delete
	reqDel := httptest.NewRequest("DELETE", "/dek-registry/v1/keks/deldek-kek/deks/deldek-subject", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Should not be found without deleted=true
	reqGet := httptest.NewRequest("GET", "/dek-registry/v1/keks/deldek-kek/deks/deldek-subject", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusNotFound {
		t.Errorf("expected 404 after soft delete, got %d", wGet.Code)
	}

	// Should be found with deleted=true
	reqGetDel := httptest.NewRequest("GET", "/dek-registry/v1/keks/deldek-kek/deks/deldek-subject?deleted=true", nil)
	wGetDel := httptest.NewRecorder()
	r.ServeHTTP(wGetDel, reqGetDel)

	if wGetDel.Code != http.StatusOK {
		t.Errorf("expected 200 with deleted=true, got %d: %s", wGetDel.Code, wGetDel.Body.String())
	}
}

func TestUndeleteDEK_Success(t *testing.T) {
	h := setupTestHandler(t)
	createKEK(t, h, "undeldek-kek", "aws-kms", "key-1")
	createDEK(t, h, "undeldek-kek", "undeldek-subject")

	r := chi.NewRouter()
	r.Delete("/dek-registry/v1/keks/{name}/deks/{subject}", h.DeleteDEK)
	r.Put("/dek-registry/v1/keks/{name}/deks/{subject}/undelete", h.UndeleteDEK)
	r.Get("/dek-registry/v1/keks/{name}/deks/{subject}", h.GetDEK)

	// Soft delete
	reqDel := httptest.NewRequest("DELETE", "/dek-registry/v1/keks/undeldek-kek/deks/undeldek-subject", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("soft-delete failed: %d", wDel.Code)
	}

	// Undelete
	reqUndel := httptest.NewRequest("PUT", "/dek-registry/v1/keks/undeldek-kek/deks/undeldek-subject/undelete", nil)
	wUndel := httptest.NewRecorder()
	r.ServeHTTP(wUndel, reqUndel)

	if wUndel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wUndel.Code, wUndel.Body.String())
	}

	// Should be found again without deleted=true
	reqGet := httptest.NewRequest("GET", "/dek-registry/v1/keks/undeldek-kek/deks/undeldek-subject", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Errorf("expected 200 after undelete, got %d: %s", wGet.Code, wGet.Body.String())
	}
}
