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

// --- Exporter Test Helpers ---

func createExporter(t *testing.T, h *Handler, name string, config map[string]string) {
	t.Helper()
	body := types.CreateExporterRequest{
		Name:   name,
		Config: config,
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/exporters", h.CreateExporter)

	req := httptest.NewRequest("POST", "/exporters", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("createExporter failed: %d %s", w.Code, w.Body.String())
	}
}

// --- Exporter Tests ---

func TestListExporters_Empty(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/exporters", h.ListExporters)

	req := httptest.NewRequest("GET", "/exporters", nil)
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

func TestCreateExporter_Success(t *testing.T) {
	h := setupTestHandler(t)

	body := types.CreateExporterRequest{
		Name:        "my-exporter",
		ContextType: "AUTO",
		Subjects:    []string{"test-*"},
		Config: map[string]string{
			"schema.registry.url": "http://dest:8081",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/exporters", h.CreateExporter)

	req := httptest.NewRequest("POST", "/exporters", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ExporterNameResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "my-exporter" {
		t.Errorf("expected name my-exporter, got %s", resp.Name)
	}
}

func TestCreateExporter_MissingName(t *testing.T) {
	h := setupTestHandler(t)

	body := types.CreateExporterRequest{
		Config: map[string]string{
			"schema.registry.url": "http://dest:8081",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/exporters", h.CreateExporter)

	req := httptest.NewRequest("POST", "/exporters", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExporter_Duplicate(t *testing.T) {
	h := setupTestHandler(t)
	createExporter(t, h, "dup-exporter", nil)

	// Create again with same name
	body := types.CreateExporterRequest{
		Name: "dup-exporter",
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/exporters", h.CreateExporter)

	req := httptest.NewRequest("POST", "/exporters", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeExporterExists {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeExporterExists, resp.ErrorCode)
	}
}

func TestGetExporter_Success(t *testing.T) {
	h := setupTestHandler(t)
	createExporter(t, h, "get-exporter", map[string]string{"key": "value"})

	r := chi.NewRouter()
	r.Get("/exporters/{name}", h.GetExporter)

	req := httptest.NewRequest("GET", "/exporters/get-exporter", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ExporterResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "get-exporter" {
		t.Errorf("expected name get-exporter, got %s", resp.Name)
	}
	if resp.Config["key"] != "value" {
		t.Errorf("expected config key=value, got %v", resp.Config)
	}
}

func TestGetExporter_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/exporters/{name}", h.GetExporter)

	req := httptest.NewRequest("GET", "/exporters/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeExporterNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeExporterNotFound, resp.ErrorCode)
	}
}

func TestUpdateExporter_Success(t *testing.T) {
	h := setupTestHandler(t)
	createExporter(t, h, "update-exporter", nil)

	body := types.UpdateExporterRequest{
		Subjects: []string{"new-subject-*"},
		Config: map[string]string{
			"schema.registry.url": "http://updated:8081",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Put("/exporters/{name}", h.UpdateExporter)

	req := httptest.NewRequest("PUT", "/exporters/update-exporter", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ExporterNameResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "update-exporter" {
		t.Errorf("expected name update-exporter, got %s", resp.Name)
	}
}

func TestUpdateExporter_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	body := types.UpdateExporterRequest{
		Subjects: []string{"test-*"},
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Put("/exporters/{name}", h.UpdateExporter)

	req := httptest.NewRequest("PUT", "/exporters/nonexistent", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeExporterNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeExporterNotFound, resp.ErrorCode)
	}
}

func TestDeleteExporter_Success(t *testing.T) {
	h := setupTestHandler(t)
	createExporter(t, h, "del-exporter", nil)

	r := chi.NewRouter()
	r.Delete("/exporters/{name}", h.DeleteExporter)
	r.Get("/exporters/{name}", h.GetExporter)

	// Delete
	reqDel := httptest.NewRequest("DELETE", "/exporters/del-exporter", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Verify it is gone
	reqGet := httptest.NewRequest("GET", "/exporters/del-exporter", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", wGet.Code)
	}
}

func TestDeleteExporter_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Delete("/exporters/{name}", h.DeleteExporter)

	req := httptest.NewRequest("DELETE", "/exporters/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeExporterNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeExporterNotFound, resp.ErrorCode)
	}
}

func TestGetExporterStatus_Default(t *testing.T) {
	h := setupTestHandler(t)
	createExporter(t, h, "status-exporter", nil)

	r := chi.NewRouter()
	r.Get("/exporters/{name}/status", h.GetExporterStatus)

	req := httptest.NewRequest("GET", "/exporters/status-exporter/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ExporterStatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "status-exporter" {
		t.Errorf("expected name status-exporter, got %s", resp.Name)
	}
	if resp.State != "PAUSED" {
		t.Errorf("expected state PAUSED, got %s", resp.State)
	}
}

func TestGetExporterConfig_Success(t *testing.T) {
	h := setupTestHandler(t)
	createExporter(t, h, "config-exporter", map[string]string{
		"schema.registry.url": "http://dest:8081",
		"basic.auth.enabled":  "true",
	})

	r := chi.NewRouter()
	r.Get("/exporters/{name}/config", h.GetExporterConfig)

	req := httptest.NewRequest("GET", "/exporters/config-exporter/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var config map[string]string
	json.NewDecoder(w.Body).Decode(&config)
	if config["schema.registry.url"] != "http://dest:8081" {
		t.Errorf("expected schema.registry.url=http://dest:8081, got %s", config["schema.registry.url"])
	}
	if config["basic.auth.enabled"] != "true" {
		t.Errorf("expected basic.auth.enabled=true, got %s", config["basic.auth.enabled"])
	}
}

func TestUpdateExporterConfig_Success(t *testing.T) {
	h := setupTestHandler(t)
	createExporter(t, h, "updcfg-exporter", map[string]string{
		"key": "old-value",
	})

	body := types.UpdateExporterConfigRequest{
		Config: map[string]string{
			"key": "new-value",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Put("/exporters/{name}/config", h.UpdateExporterConfig)
	r.Get("/exporters/{name}/config", h.GetExporterConfig)

	// Update config
	reqUpd := httptest.NewRequest("PUT", "/exporters/updcfg-exporter/config", bytes.NewReader(bodyBytes))
	reqUpd.Header.Set("Content-Type", "application/json")
	wUpd := httptest.NewRecorder()
	r.ServeHTTP(wUpd, reqUpd)

	if wUpd.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wUpd.Code, wUpd.Body.String())
	}

	// Verify updated config
	reqGet := httptest.NewRequest("GET", "/exporters/updcfg-exporter/config", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGet.Code)
	}

	var config map[string]string
	json.NewDecoder(wGet.Body).Decode(&config)
	if config["key"] != "new-value" {
		t.Errorf("expected key=new-value, got %s", config["key"])
	}
}

func TestPauseExporter_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/exporters/{name}/pause", h.PauseExporter)

	req := httptest.NewRequest("PUT", "/exporters/nonexistent/pause", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeExporterNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeExporterNotFound, resp.ErrorCode)
	}
}

func TestResumeExporter_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/exporters/{name}/resume", h.ResumeExporter)

	req := httptest.NewRequest("PUT", "/exporters/nonexistent/resume", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeExporterNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeExporterNotFound, resp.ErrorCode)
	}
}
