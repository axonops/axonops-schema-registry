package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

func setupTestHandler(t *testing.T) *Handler {
	t.Helper()
	store := memory.NewStore()
	schemaReg := schema.NewRegistry()
	schemaReg.Register(avro.NewParser())
	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	reg := registry.New(store, schemaReg, compatChecker, "BACKWARD")
	return New(reg)
}

func registerSchema(t *testing.T, h *Handler, subject, schemaStr string) int64 {
	t.Helper()
	body := types.RegisterSchemaRequest{Schema: schemaStr}
	bodyBytes, _ := json.Marshal(body)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)

	req := httptest.NewRequest("POST", "/subjects/"+subject+"/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("registerSchema failed: %d %s", w.Code, w.Body.String())
	}
	var resp types.RegisterSchemaResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.ID
}

// setImportMode sets the global mode to IMPORT via the handler.
func setImportMode(t *testing.T, h *Handler) {
	t.Helper()
	modeReq := types.ModeRequest{Mode: "IMPORT"}
	modeBytes, _ := json.Marshal(modeReq)

	r := chi.NewRouter()
	r.Put("/mode", h.SetMode)

	req := httptest.NewRequest("PUT", "/mode?force=true", bytes.NewReader(modeBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("setImportMode failed: %d %s", w.Code, w.Body.String())
	}
}

func decodeErrorResponse(t *testing.T, w *httptest.ResponseRecorder) types.ErrorResponse {
	t.Helper()
	var resp types.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	return resp
}

// --- HealthCheck ---

func TestHealthCheck_Returns200(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/", h.HealthCheck)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

// --- GetSchemaTypes ---

func TestGetSchemaTypes(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/types", h.GetSchemaTypes)

	req := httptest.NewRequest("GET", "/schemas/types", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var schemaTypes []string
	json.NewDecoder(w.Body).Decode(&schemaTypes)
	if len(schemaTypes) == 0 {
		t.Error("expected at least one schema type")
	}
}

// --- GetSchemaByID ---

func TestGetSchemaByID_Found(t *testing.T) {
	h := setupTestHandler(t)
	id := registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}", h.GetSchemaByID)

	req := httptest.NewRequest("GET", fmt.Sprintf("/schemas/ids/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.SchemaByIDResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Schema == "" {
		t.Error("expected non-empty schema")
	}
}

func TestGetSchemaByID_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}", h.GetSchemaByID)

	req := httptest.NewRequest("GET", "/schemas/ids/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeSchemaNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeSchemaNotFound, resp.ErrorCode)
	}
}

func TestGetSchemaByID_InvalidID(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}", h.GetSchemaByID)

	req := httptest.NewRequest("GET", "/schemas/ids/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- GetRawSchemaByID ---

func TestGetRawSchemaByID_Found(t *testing.T) {
	h := setupTestHandler(t)
	id := registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/schema", h.GetRawSchemaByID)

	req := httptest.NewRequest("GET", fmt.Sprintf("/schemas/ids/%d/schema", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty raw schema")
	}
}

func TestGetRawSchemaByID_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/schema", h.GetRawSchemaByID)

	req := httptest.NewRequest("GET", "/schemas/ids/999/schema", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetRawSchemaByID_InvalidID(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/schema", h.GetRawSchemaByID)

	req := httptest.NewRequest("GET", "/schemas/ids/abc/schema", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- GetSubjectsBySchemaID ---

func TestGetSubjectsBySchemaID_Found(t *testing.T) {
	h := setupTestHandler(t)
	id := registerSchema(t, h, "test-sub", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/subjects", h.GetSubjectsBySchemaID)

	req := httptest.NewRequest("GET", fmt.Sprintf("/schemas/ids/%d/subjects", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var subjects []string
	json.NewDecoder(w.Body).Decode(&subjects)
	if len(subjects) == 0 {
		t.Error("expected at least one subject")
	}
}

func TestGetSubjectsBySchemaID_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/subjects", h.GetSubjectsBySchemaID)

	req := httptest.NewRequest("GET", "/schemas/ids/999/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetSubjectsBySchemaID_InvalidID(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/subjects", h.GetSubjectsBySchemaID)

	req := httptest.NewRequest("GET", "/schemas/ids/abc/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- GetVersionsBySchemaID ---

func TestGetVersionsBySchemaID_Found(t *testing.T) {
	h := setupTestHandler(t)
	id := registerSchema(t, h, "test-ver", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/versions", h.GetVersionsBySchemaID)

	req := httptest.NewRequest("GET", fmt.Sprintf("/schemas/ids/%d/versions", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var versions []types.SubjectVersionPair
	json.NewDecoder(w.Body).Decode(&versions)
	if len(versions) == 0 {
		t.Error("expected at least one version pair")
	}
}

func TestGetVersionsBySchemaID_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/versions", h.GetVersionsBySchemaID)

	req := httptest.NewRequest("GET", "/schemas/ids/999/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetVersionsBySchemaID_InvalidID(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas/ids/{id}/versions", h.GetVersionsBySchemaID)

	req := httptest.NewRequest("GET", "/schemas/ids/abc/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- ListSubjects ---

func TestListSubjects_Empty(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/subjects", h.ListSubjects)

	req := httptest.NewRequest("GET", "/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var subjects []string
	json.NewDecoder(w.Body).Decode(&subjects)
	if len(subjects) != 0 {
		t.Errorf("expected 0 subjects, got %d", len(subjects))
	}
}

func TestListSubjects_WithSubjects(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "sub-a", `{"type":"string"}`)
	registerSchema(t, h, "sub-b", `{"type":"int"}`)

	r := chi.NewRouter()
	r.Get("/subjects", h.ListSubjects)

	req := httptest.NewRequest("GET", "/subjects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var subjects []string
	json.NewDecoder(w.Body).Decode(&subjects)
	if len(subjects) != 2 {
		t.Errorf("expected 2 subjects, got %d", len(subjects))
	}
}

// --- GetVersions ---

func TestGetVersions_Found(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"record","name":"U","fields":[{"name":"id","type":"long"}]}`)
	registerSchema(t, h, "test", `{"type":"record","name":"U","fields":[{"name":"id","type":"long"},{"name":"n","type":"string","default":""}]}`)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions", h.GetVersions)

	req := httptest.NewRequest("GET", "/subjects/test/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var versions []int
	json.NewDecoder(w.Body).Decode(&versions)
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
}

func TestGetVersions_SubjectNotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions", h.GetVersions)

	req := httptest.NewRequest("GET", "/subjects/nonexistent/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeSubjectNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeSubjectNotFound, resp.ErrorCode)
	}
}

// --- GetVersion ---

func TestGetVersion_Found(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}", h.GetVersion)

	req := httptest.NewRequest("GET", "/subjects/test/versions/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.SubjectVersionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Subject != "test" {
		t.Errorf("expected subject test, got %s", resp.Subject)
	}
	if resp.Version != 1 {
		t.Errorf("expected version 1, got %d", resp.Version)
	}
}

func TestGetVersion_Latest(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}", h.GetVersion)

	req := httptest.NewRequest("GET", "/subjects/test/versions/latest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetVersion_SubjectNotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}", h.GetVersion)

	req := httptest.NewRequest("GET", "/subjects/nonexistent/versions/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetVersion_VersionNotFound(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}", h.GetVersion)

	req := httptest.NewRequest("GET", "/subjects/test/versions/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	resp := decodeErrorResponse(t, w)
	if resp.ErrorCode != types.ErrorCodeVersionNotFound {
		t.Errorf("expected error_code %d, got %d", types.ErrorCodeVersionNotFound, resp.ErrorCode)
	}
}

func TestGetVersion_InvalidVersion(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}", h.GetVersion)

	req := httptest.NewRequest("GET", "/subjects/test/versions/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- GetRawSchemaByVersion ---

func TestGetRawSchemaByVersion_Found(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}/schema", h.GetRawSchemaByVersion)

	req := httptest.NewRequest("GET", "/subjects/test/versions/1/schema", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() == "" {
		t.Error("expected non-empty raw schema")
	}
}

func TestGetRawSchemaByVersion_SubjectNotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}/schema", h.GetRawSchemaByVersion)

	req := httptest.NewRequest("GET", "/subjects/nonexistent/versions/1/schema", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetRawSchemaByVersion_VersionNotFound(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}/schema", h.GetRawSchemaByVersion)

	req := httptest.NewRequest("GET", "/subjects/test/versions/99/schema", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetRawSchemaByVersion_InvalidVersion(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}/schema", h.GetRawSchemaByVersion)

	req := httptest.NewRequest("GET", "/subjects/test/versions/abc/schema", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- RegisterSchema ---

func TestRegisterSchema_Success(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)

	body := types.RegisterSchemaRequest{Schema: `{"type":"string"}`}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.RegisterSchemaResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == 0 {
		t.Error("expected non-zero schema ID")
	}
}

func TestRegisterSchema_EmptySchema(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)

	body := types.RegisterSchemaRequest{Schema: ""}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

func TestRegisterSchema_InvalidJSON(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)

	req := httptest.NewRequest("POST", "/subjects/test/versions", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegisterSchema_InvalidAvro(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)

	body := types.RegisterSchemaRequest{Schema: `{"type":"invalid_type"}`}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterSchema_DefaultsToAvro(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)

	// No schemaType specified, should default to AVRO
	body := types.RegisterSchemaRequest{Schema: `{"type":"string"}`}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterSchema_Incompatible(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)

	// Register v1
	schema1 := `{"type":"record","name":"U","fields":[{"name":"id","type":"long"}]}`
	b1, _ := json.Marshal(types.RegisterSchemaRequest{Schema: schema1})
	req1 := httptest.NewRequest("POST", "/subjects/test/versions", bytes.NewReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("v1 registration failed: %d", w1.Code)
	}

	// Register incompatible v2 (required field without default)
	schema2 := `{"type":"record","name":"U","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`
	b2, _ := json.Marshal(types.RegisterSchemaRequest{Schema: schema2})
	req2 := httptest.NewRequest("POST", "/subjects/test/versions", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409 (Conflict), got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRegisterSchema_DuplicateReturnsSameID(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}/versions", h.RegisterSchema)

	schema := `{"type":"string"}`
	b, _ := json.Marshal(types.RegisterSchemaRequest{Schema: schema})

	// Register first time
	req1 := httptest.NewRequest("POST", "/subjects/test/versions", bytes.NewReader(b))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	var resp1 types.RegisterSchemaResponse
	json.NewDecoder(w1.Body).Decode(&resp1)

	// Register same schema again
	b2, _ := json.Marshal(types.RegisterSchemaRequest{Schema: schema})
	req2 := httptest.NewRequest("POST", "/subjects/test/versions", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var resp2 types.RegisterSchemaResponse
	json.NewDecoder(w2.Body).Decode(&resp2)

	if resp1.ID != resp2.ID {
		t.Errorf("expected same ID %d for duplicate, got %d", resp1.ID, resp2.ID)
	}
}

// --- LookupSchema ---

func TestLookupSchema_Found(t *testing.T) {
	h := setupTestHandler(t)
	schema := `{"type":"string"}`
	registerSchema(t, h, "test", schema)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}", h.LookupSchema)

	body := types.LookupSchemaRequest{Schema: schema}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.LookupSchemaResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Version != 1 {
		t.Errorf("expected version 1, got %d", resp.Version)
	}
}

func TestLookupSchema_NotFound(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}", h.LookupSchema)

	body := types.LookupSchemaRequest{Schema: `{"type":"int"}`}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestLookupSchema_SubjectNotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}", h.LookupSchema)

	body := types.LookupSchemaRequest{Schema: `{"type":"string"}`}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/nonexistent", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestLookupSchema_EmptySchema(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}", h.LookupSchema)

	body := types.LookupSchemaRequest{Schema: ""}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestLookupSchema_InvalidBody(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/subjects/{subject}", h.LookupSchema)

	req := httptest.NewRequest("POST", "/subjects/test", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- DeleteSubject ---

func TestDeleteSubject_SoftDelete(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Delete("/subjects/{subject}", h.DeleteSubject)

	req := httptest.NewRequest("DELETE", "/subjects/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var versions []int
	json.NewDecoder(w.Body).Decode(&versions)
	if len(versions) != 1 {
		t.Errorf("expected 1 deleted version, got %d", len(versions))
	}
}

func TestDeleteSubject_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Delete("/subjects/{subject}", h.DeleteSubject)

	req := httptest.NewRequest("DELETE", "/subjects/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteSubject_ContentType(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Delete("/subjects/{subject}", h.DeleteSubject)

	req := httptest.NewRequest("DELETE", "/subjects/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/vnd.schemaregistry.v1+json" {
		t.Errorf("expected Content-Type application/vnd.schemaregistry.v1+json, got %s", ct)
	}
}

// --- DeleteVersion ---

func TestDeleteVersion_SoftDelete(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Delete("/subjects/{subject}/versions/{version}", h.DeleteVersion)

	req := httptest.NewRequest("DELETE", "/subjects/test/versions/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteVersion_SubjectNotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Delete("/subjects/{subject}/versions/{version}", h.DeleteVersion)

	req := httptest.NewRequest("DELETE", "/subjects/nonexistent/versions/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteVersion_VersionNotFound(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Delete("/subjects/{subject}/versions/{version}", h.DeleteVersion)

	req := httptest.NewRequest("DELETE", "/subjects/test/versions/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteVersion_InvalidVersion(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Delete("/subjects/{subject}/versions/{version}", h.DeleteVersion)

	req := httptest.NewRequest("DELETE", "/subjects/test/versions/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- Config ---

func TestGetConfig_GlobalDefault(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/config", h.GetConfig)

	req := httptest.NewRequest("GET", "/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.ConfigResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.CompatibilityLevel != "BACKWARD" {
		t.Errorf("expected BACKWARD, got %s", resp.CompatibilityLevel)
	}
}

func TestSetConfig_Global(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/config", h.SetConfig)
	r.Get("/config", h.GetConfig)

	body := types.ConfigRequest{Compatibility: "FULL"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/config", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ConfigRequest
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Compatibility != "FULL" {
		t.Errorf("expected FULL, got %s", resp.Compatibility)
	}
}

func TestSetConfig_Subject(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/config/{subject}", h.SetConfig)

	body := types.ConfigRequest{Compatibility: "NONE"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/config/test-sub", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetConfig_InvalidLevel(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/config", h.SetConfig)

	body := types.ConfigRequest{Compatibility: "INVALID"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/config", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetConfig_InvalidBody(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/config", h.SetConfig)

	req := httptest.NewRequest("PUT", "/config", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeleteConfig_Found(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/config/{subject}", h.SetConfig)
	r.Delete("/config/{subject}", h.DeleteConfig)

	// Set config first
	body := types.ConfigRequest{Compatibility: "NONE"}
	bodyBytes, _ := json.Marshal(body)
	reqSet := httptest.NewRequest("PUT", "/config/test-sub", bytes.NewReader(bodyBytes))
	reqSet.Header.Set("Content-Type", "application/json")
	wSet := httptest.NewRecorder()
	r.ServeHTTP(wSet, reqSet)

	// Delete it
	reqDel := httptest.NewRequest("DELETE", "/config/test-sub", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wDel.Code, wDel.Body.String())
	}
}

func TestDeleteConfig_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Delete("/config/{subject}", h.DeleteConfig)

	req := httptest.NewRequest("DELETE", "/config/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteGlobalConfig(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Delete("/config", h.DeleteGlobalConfig)

	req := httptest.NewRequest("DELETE", "/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.ConfigResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.CompatibilityLevel == "" {
		t.Error("expected non-empty compatibility level")
	}
}

// --- Mode ---

func TestGetMode_GlobalDefault(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/mode", h.GetMode)

	req := httptest.NewRequest("GET", "/mode", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.ModeResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Mode != "READWRITE" {
		t.Errorf("expected READWRITE, got %s", resp.Mode)
	}
}

func TestSetMode_Valid(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/mode", h.SetMode)

	body := types.ModeRequest{Mode: "READONLY"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/mode", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ModeResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Mode != "READONLY" {
		t.Errorf("expected READONLY, got %s", resp.Mode)
	}
}

func TestSetMode_InvalidMode(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/mode", h.SetMode)

	body := types.ModeRequest{Mode: "INVALID"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/mode", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSetMode_InvalidBody(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/mode", h.SetMode)

	req := httptest.NewRequest("PUT", "/mode", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeleteMode_Found(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Put("/mode/{subject}", h.SetMode)
	r.Delete("/mode/{subject}", h.DeleteMode)

	body := types.ModeRequest{Mode: "READONLY"}
	bodyBytes, _ := json.Marshal(body)
	reqSet := httptest.NewRequest("PUT", "/mode/test-sub", bytes.NewReader(bodyBytes))
	reqSet.Header.Set("Content-Type", "application/json")
	wSet := httptest.NewRecorder()
	r.ServeHTTP(wSet, reqSet)

	reqDel := httptest.NewRequest("DELETE", "/mode/test-sub", nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wDel.Code, wDel.Body.String())
	}
}

func TestDeleteMode_NotFound(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Delete("/mode/{subject}", h.DeleteMode)

	req := httptest.NewRequest("DELETE", "/mode/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- CheckCompatibility ---

func TestCheckCompatibility_Compatible(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"record","name":"U","fields":[{"name":"id","type":"long"}]}`)

	r := chi.NewRouter()
	r.Post("/compatibility/subjects/{subject}/versions/{version}", h.CheckCompatibility)

	schema2 := `{"type":"record","name":"U","fields":[{"name":"id","type":"long"},{"name":"n","type":"string","default":""}]}`
	body := types.CompatibilityCheckRequest{Schema: schema2}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/compatibility/subjects/test/versions/latest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.CompatibilityCheckResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.IsCompatible {
		t.Errorf("expected compatible, got incompatible: %v", resp.Messages)
	}
}

func TestCheckCompatibility_Incompatible(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"record","name":"U","fields":[{"name":"id","type":"long"}]}`)

	r := chi.NewRouter()
	r.Post("/compatibility/subjects/{subject}/versions/{version}", h.CheckCompatibility)

	schema2 := `{"type":"record","name":"U","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`
	body := types.CompatibilityCheckRequest{Schema: schema2}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/compatibility/subjects/test/versions/latest?verbose=true", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.CompatibilityCheckResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.IsCompatible {
		t.Error("expected incompatible, got compatible")
	}
	if len(resp.Messages) == 0 {
		t.Error("expected error messages")
	}
}

func TestCheckCompatibility_SubjectNotFound_Latest(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/compatibility/subjects/{subject}/versions/{version}", h.CheckCompatibility)

	body := types.CompatibilityCheckRequest{Schema: `{"type":"string"}`}
	bodyBytes, _ := json.Marshal(body)

	// With "latest", non-existent subject means no schemas to check against = always compatible
	req := httptest.NewRequest("POST", "/compatibility/subjects/nonexistent/versions/latest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.CompatibilityCheckResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.IsCompatible {
		t.Error("expected compatible (no existing schemas)")
	}
}

func TestCheckCompatibility_EmptySchema(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Post("/compatibility/subjects/{subject}/versions/{version}", h.CheckCompatibility)

	body := types.CompatibilityCheckRequest{Schema: ""}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/compatibility/subjects/test/versions/latest", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- ListSchemas ---

func TestListSchemas_Empty(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/schemas", h.ListSchemas)

	req := httptest.NewRequest("GET", "/schemas", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var schemas []types.SchemaListItem
	json.NewDecoder(w.Body).Decode(&schemas)
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas, got %d", len(schemas))
	}
}

func TestListSchemas_WithSchemas(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "sub-a", `{"type":"string"}`)
	registerSchema(t, h, "sub-b", `{"type":"int"}`)

	r := chi.NewRouter()
	r.Get("/schemas", h.ListSchemas)

	req := httptest.NewRequest("GET", "/schemas", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var schemas []types.SchemaListItem
	json.NewDecoder(w.Body).Decode(&schemas)
	if len(schemas) < 2 {
		t.Errorf("expected at least 2 schemas, got %d", len(schemas))
	}
}

// --- ImportSchemas ---

func TestImportSchemas_Success(t *testing.T) {
	h := setupTestHandler(t)
	setImportMode(t, h)

	r := chi.NewRouter()
	r.Post("/import/schemas", h.ImportSchemas)

	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{ID: 42, Subject: "user-value", Version: 1, SchemaType: "AVRO", Schema: `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`},
			{ID: 43, Subject: "order-value", Version: 1, SchemaType: "AVRO", Schema: `{"type":"record","name":"Order","fields":[{"name":"oid","type":"long"}]}`},
		},
	}
	bodyBytes, _ := json.Marshal(importReq)

	req := httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ImportSchemasResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Imported != 2 {
		t.Errorf("expected 2 imported, got %d", resp.Imported)
	}
	if resp.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", resp.Errors)
	}
}

func TestImportSchemas_EmptyList(t *testing.T) {
	h := setupTestHandler(t)
	setImportMode(t, h)

	r := chi.NewRouter()
	r.Post("/import/schemas", h.ImportSchemas)

	importReq := types.ImportSchemasRequest{Schemas: []types.ImportSchemaRequest{}}
	bodyBytes, _ := json.Marshal(importReq)

	req := httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestImportSchemas_InvalidBody(t *testing.T) {
	h := setupTestHandler(t)
	setImportMode(t, h)

	r := chi.NewRouter()
	r.Post("/import/schemas", h.ImportSchemas)

	req := httptest.NewRequest("POST", "/import/schemas", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestImportSchemas_DuplicateID(t *testing.T) {
	h := setupTestHandler(t)
	setImportMode(t, h)

	r := chi.NewRouter()
	r.Post("/import/schemas", h.ImportSchemas)

	// Import first
	importReq1 := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{ID: 42, Subject: "user-value", Version: 1, SchemaType: "AVRO", Schema: `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`},
		},
	}
	b1, _ := json.Marshal(importReq1)
	req1 := httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(b1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	// Import duplicate ID with different subject
	importReq2 := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{ID: 42, Subject: "order-value", Version: 1, SchemaType: "AVRO", Schema: `{"type":"record","name":"Order","fields":[{"name":"oid","type":"long"}]}`},
		},
	}
	b2, _ := json.Marshal(importReq2)
	req2 := httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(b2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var resp types.ImportSchemasResponse
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Imported != 0 {
		t.Errorf("expected 0 imported, got %d", resp.Imported)
	}
	if resp.Errors != 1 {
		t.Errorf("expected 1 error, got %d", resp.Errors)
	}
}

// --- Metadata endpoints ---

func TestGetContexts(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/contexts", h.GetContexts)

	req := httptest.NewRequest("GET", "/contexts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var contexts []string
	json.NewDecoder(w.Body).Decode(&contexts)
	if len(contexts) != 1 || contexts[0] != "." {
		t.Errorf("expected [\".\"], got %v", contexts)
	}
}

func TestGetClusterID(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/v1/metadata/id", h.GetClusterID)

	req := httptest.NewRequest("GET", "/v1/metadata/id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.ServerClusterIDResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != "default-cluster" {
		t.Errorf("expected default-cluster, got %s", resp.ID)
	}
}

func TestGetServerVersion_Default(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/v1/metadata/version", h.GetServerVersion)

	req := httptest.NewRequest("GET", "/v1/metadata/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp types.ServerVersionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Version != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", resp.Version)
	}
}

func TestGetServerVersion_WithConfig(t *testing.T) {
	store := memory.NewStore()
	schemaReg := schema.NewRegistry()
	schemaReg.Register(avro.NewParser())
	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	reg := registry.New(store, schemaReg, compatChecker, "BACKWARD")
	h := NewWithConfig(reg, Config{
		ClusterID: "my-cluster",
		Version:   "2.0.0",
		Commit:    "abc123",
		BuildTime: "2024-01-01",
	})

	r := chi.NewRouter()
	r.Get("/v1/metadata/version", h.GetServerVersion)

	req := httptest.NewRequest("GET", "/v1/metadata/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp types.ServerVersionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Version != "2.0.0" {
		t.Errorf("expected 2.0.0, got %s", resp.Version)
	}
	if resp.Commit != "abc123" {
		t.Errorf("expected abc123, got %s", resp.Commit)
	}
}

// --- GetReferencedBy ---

func TestGetReferencedBy_NoRefs(t *testing.T) {
	h := setupTestHandler(t)
	registerSchema(t, h, "test", `{"type":"string"}`)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}/referencedby", h.GetReferencedBy)

	req := httptest.NewRequest("GET", "/subjects/test/versions/1/referencedby", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var refs []int
	json.NewDecoder(w.Body).Decode(&refs)
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestGetReferencedBy_InvalidVersion(t *testing.T) {
	h := setupTestHandler(t)

	r := chi.NewRouter()
	r.Get("/subjects/{subject}/versions/{version}/referencedby", h.GetReferencedBy)

	req := httptest.NewRequest("GET", "/subjects/test/versions/abc/referencedby", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}
