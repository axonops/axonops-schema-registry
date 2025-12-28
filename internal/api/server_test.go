package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/api/types"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()

	cfg := config.DefaultConfig()
	store := memory.NewStore()

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())

	reg := registry.New(store, schemaRegistry, compatChecker, cfg.Compatibility.DefaultLevel)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewServer(cfg, reg, logger)
}

func TestServer_HealthCheck(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestServer_GetSchemaTypes(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/schemas/types", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var types []string
	if err := json.NewDecoder(w.Body).Decode(&types); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(types) == 0 {
		t.Error("Expected at least one schema type")
	}
}

func TestServer_RegisterAndGetSchema(t *testing.T) {
	server := setupTestServer(t)

	// Register a schema
	schema := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}]}`
	body := types.RegisterSchemaRequest{Schema: schema}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var regResp types.RegisterSchemaResponse
	if err := json.NewDecoder(w.Body).Decode(&regResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if regResp.ID == 0 {
		t.Error("Expected non-zero schema ID")
	}

	// Get the schema by ID
	req = httptest.NewRequest("GET", "/schemas/ids/1", nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var schemaResp types.SchemaByIDResponse
	if err := json.NewDecoder(w.Body).Decode(&schemaResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if schemaResp.Schema == "" {
		t.Error("Expected non-empty schema")
	}
}

func TestServer_ListSubjects(t *testing.T) {
	server := setupTestServer(t)

	// Initially empty
	req := httptest.NewRequest("GET", "/subjects", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var subjects []string
	if err := json.NewDecoder(w.Body).Decode(&subjects); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(subjects) != 0 {
		t.Errorf("Expected 0 subjects, got %d", len(subjects))
	}

	// Register a schema
	schema := `{"type": "string"}`
	body := types.RegisterSchemaRequest{Schema: schema}
	bodyBytes, _ := json.Marshal(body)

	req = httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Now list subjects
	req = httptest.NewRequest("GET", "/subjects", nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if err := json.NewDecoder(w.Body).Decode(&subjects); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(subjects) != 1 {
		t.Errorf("Expected 1 subject, got %d", len(subjects))
	}
	if subjects[0] != "test-subject" {
		t.Errorf("Expected test-subject, got %s", subjects[0])
	}
}

func TestServer_GetVersions(t *testing.T) {
	server := setupTestServer(t)

	// Register two compatible versions (second has optional field with default)
	schemas := []string{
		`{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}]}`,
		`{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}, {"name": "name", "type": "string", "default": ""}]}`,
	}

	for _, s := range schemas {
		body := types.RegisterSchemaRequest{Schema: s}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
	}

	// Get versions
	req := httptest.NewRequest("GET", "/subjects/test-subject/versions", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var versions []int
	if err := json.NewDecoder(w.Body).Decode(&versions); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}
}

func TestServer_GetVersion(t *testing.T) {
	server := setupTestServer(t)

	// Register a schema
	schema := `{"type": "string"}`
	body := types.RegisterSchemaRequest{Schema: schema}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Get version 1
	req = httptest.NewRequest("GET", "/subjects/test-subject/versions/1", nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.SubjectVersionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Subject != "test-subject" {
		t.Errorf("Expected subject test-subject, got %s", resp.Subject)
	}
	if resp.Version != 1 {
		t.Errorf("Expected version 1, got %d", resp.Version)
	}

	// Get latest
	req = httptest.NewRequest("GET", "/subjects/test-subject/versions/latest", nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

func TestServer_LookupSchema(t *testing.T) {
	server := setupTestServer(t)

	// Register a schema
	schema := `{"type": "string"}`
	body := types.RegisterSchemaRequest{Schema: schema}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Lookup the schema
	lookupBody := types.LookupSchemaRequest{Schema: schema}
	lookupBytes, _ := json.Marshal(lookupBody)

	req = httptest.NewRequest("POST", "/subjects/test-subject", bytes.NewReader(lookupBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.LookupSchemaResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Version != 1 {
		t.Errorf("Expected version 1, got %d", resp.Version)
	}
}

func TestServer_DeleteSubject(t *testing.T) {
	server := setupTestServer(t)

	// Register a schema
	schema := `{"type": "string"}`
	body := types.RegisterSchemaRequest{Schema: schema}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Delete the subject
	req = httptest.NewRequest("DELETE", "/subjects/test-subject", nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var versions []int
	if err := json.NewDecoder(w.Body).Decode(&versions); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(versions) != 1 {
		t.Errorf("Expected 1 deleted version, got %d", len(versions))
	}
}

func TestServer_Config(t *testing.T) {
	server := setupTestServer(t)

	// Get global config
	req := httptest.NewRequest("GET", "/config", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp types.ConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.CompatibilityLevel != "BACKWARD" {
		t.Errorf("Expected BACKWARD, got %s", resp.CompatibilityLevel)
	}

	// Set global config
	body := types.ConfigRequest{Compatibility: "FULL"}
	bodyBytes, _ := json.Marshal(body)

	req = httptest.NewRequest("PUT", "/config", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it was set
	req = httptest.NewRequest("GET", "/config", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.CompatibilityLevel != "FULL" {
		t.Errorf("Expected FULL, got %s", resp.CompatibilityLevel)
	}
}

func TestServer_NotFound(t *testing.T) {
	server := setupTestServer(t)

	// Non-existent schema ID
	req := httptest.NewRequest("GET", "/schemas/ids/999", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	// Non-existent subject
	req = httptest.NewRequest("GET", "/subjects/nonexistent/versions", nil)
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestServer_InvalidSchema(t *testing.T) {
	server := setupTestServer(t)

	// Invalid JSON
	body := types.RegisterSchemaRequest{Schema: `{invalid`}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_CompatibilityCheck(t *testing.T) {
	server := setupTestServer(t)

	// Register initial schema
	schema1 := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}]}`
	body := types.RegisterSchemaRequest{Schema: schema1}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Check compatible schema
	schema2 := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}, {"name": "name", "type": "string", "default": ""}]}`
	checkBody := types.CompatibilityCheckRequest{Schema: schema2}
	checkBytes, _ := json.Marshal(checkBody)

	req = httptest.NewRequest("POST", "/compatibility/subjects/test-subject/versions/latest", bytes.NewReader(checkBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.CompatibilityCheckResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if !resp.IsCompatible {
		t.Errorf("Expected compatible, got incompatible: %v", resp.Messages)
	}
}

func TestServer_CompatibilityCheckIncompatible(t *testing.T) {
	server := setupTestServer(t)

	// Register initial schema
	schema1 := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}]}`
	body := types.RegisterSchemaRequest{Schema: schema1}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Check incompatible schema (added required field without default)
	schema2 := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}, {"name": "name", "type": "string"}]}`
	checkBody := types.CompatibilityCheckRequest{Schema: schema2}
	checkBytes, _ := json.Marshal(checkBody)

	req = httptest.NewRequest("POST", "/compatibility/subjects/test-subject/versions/latest", bytes.NewReader(checkBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.CompatibilityCheckResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if resp.IsCompatible {
		t.Error("Expected incompatible, got compatible")
	}

	if len(resp.Messages) == 0 {
		t.Error("Expected error messages")
	}
}

func TestServer_RegisterIncompatibleSchema(t *testing.T) {
	server := setupTestServer(t)

	// Register initial schema
	schema1 := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}]}`
	body := types.RegisterSchemaRequest{Schema: schema1}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Try to register incompatible schema
	schema2 := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}, {"name": "name", "type": "string"}]}`
	body = types.RegisterSchemaRequest{Schema: schema2}
	bodyBytes, _ = json.Marshal(body)

	req = httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Should fail with 409 Conflict
	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 (Conflict), got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_RegisterCompatibleSchema(t *testing.T) {
	server := setupTestServer(t)

	// Register initial schema
	schema1 := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}]}`
	body := types.RegisterSchemaRequest{Schema: schema1}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Register compatible schema (optional field with default)
	schema2 := `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}, {"name": "name", "type": "string", "default": ""}]}`
	body = types.RegisterSchemaRequest{Schema: schema2}
	bodyBytes, _ = json.Marshal(body)

	req = httptest.NewRequest("POST", "/subjects/test-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.RegisterSchemaResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if resp.ID != 2 {
		t.Errorf("Expected schema ID 2, got %d", resp.ID)
	}
}

func TestServer_ImportSchemas(t *testing.T) {
	server := setupTestServer(t)

	// Import schemas with specific IDs
	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         42,
				Subject:    "user-value",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}]}`,
			},
			{
				ID:         43,
				Subject:    "order-value",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     `{"type": "record", "name": "Order", "fields": [{"name": "order_id", "type": "long"}]}`,
			},
		},
	}
	bodyBytes, _ := json.Marshal(importReq)

	req := httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ImportSchemasResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Imported != 2 {
		t.Errorf("Expected 2 imported schemas, got %d", resp.Imported)
	}
	if resp.Errors != 0 {
		t.Errorf("Expected 0 errors, got %d", resp.Errors)
	}

	// Verify schemas can be retrieved by ID
	req = httptest.NewRequest("GET", "/schemas/ids/42", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for schema ID 42, got %d: %s", w.Code, w.Body.String())
	}

	// Verify schemas can be retrieved by subject/version
	req = httptest.NewRequest("GET", "/subjects/user-value/versions/1", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var versionResp types.SubjectVersionResponse
	_ = json.NewDecoder(w.Body).Decode(&versionResp)

	if versionResp.ID != 42 {
		t.Errorf("Expected schema ID 42, got %d", versionResp.ID)
	}

	// New schema registration should get an ID after the imported ones
	newSchema := `{"type": "record", "name": "Product", "fields": [{"name": "product_id", "type": "long"}]}`
	registerBody := types.RegisterSchemaRequest{Schema: newSchema}
	registerBytes, _ := json.Marshal(registerBody)

	req = httptest.NewRequest("POST", "/subjects/product-value/versions", bytes.NewReader(registerBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var registerResp types.RegisterSchemaResponse
	_ = json.NewDecoder(w.Body).Decode(&registerResp)

	if registerResp.ID <= 43 {
		t.Errorf("Expected new schema ID > 43, got %d", registerResp.ID)
	}
}

func TestServer_ImportSchemas_DuplicateID(t *testing.T) {
	server := setupTestServer(t)

	// Import first schema
	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         42,
				Subject:    "user-value",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     `{"type": "record", "name": "User", "fields": [{"name": "id", "type": "long"}]}`,
			},
		},
	}
	bodyBytes, _ := json.Marshal(importReq)

	req := httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Try to import again with same ID but different subject
	importReq2 := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         42,
				Subject:    "order-value",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     `{"type": "record", "name": "Order", "fields": [{"name": "order_id", "type": "long"}]}`,
			},
		},
	}
	bodyBytes2, _ := json.Marshal(importReq2)

	req = httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(bodyBytes2))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ImportSchemasResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if resp.Imported != 0 {
		t.Errorf("Expected 0 imported, got %d", resp.Imported)
	}
	if resp.Errors != 1 {
		t.Errorf("Expected 1 error, got %d", resp.Errors)
	}
	if !resp.Results[0].Success {
		if resp.Results[0].Error == "" {
			t.Error("Expected error message for failed import")
		}
	}
}

func TestServer_ImportSchemas_InvalidSchema(t *testing.T) {
	server := setupTestServer(t)

	// Try to import invalid schema
	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         42,
				Subject:    "user-value",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     `{invalid json`,
			},
		},
	}
	bodyBytes, _ := json.Marshal(importReq)

	req := httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.ImportSchemasResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if resp.Imported != 0 {
		t.Errorf("Expected 0 imported, got %d", resp.Imported)
	}
	if resp.Errors != 1 {
		t.Errorf("Expected 1 error, got %d", resp.Errors)
	}
}

func TestServer_ImportSchemas_EmptyRequest(t *testing.T) {
	server := setupTestServer(t)

	// Try to import empty list
	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{},
	}
	bodyBytes, _ := json.Marshal(importReq)

	req := httptest.NewRequest("POST", "/import/schemas", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Should return error for empty request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}
