package api

import (
	"bytes"
	"context"
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

	req = httptest.NewRequest("POST", "/compatibility/subjects/test-subject/versions/latest?verbose=true", bytes.NewReader(checkBytes))
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

// setGlobalMode sets the global mode via the API (e.g., "IMPORT", "READWRITE").
func setGlobalMode(t *testing.T, server *Server, mode string) {
	t.Helper()
	modeReq := types.ModeRequest{Mode: mode}
	modeBytes, _ := json.Marshal(modeReq)
	req := httptest.NewRequest("PUT", "/mode?force=true", bytes.NewReader(modeBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Failed to set mode to %s: %d %s", mode, w.Code, w.Body.String())
	}
}

func TestServer_ImportSchemas(t *testing.T) {
	server := setupTestServer(t)

	// Set IMPORT mode (required for import operations)
	setGlobalMode(t, server, "IMPORT")

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

	// Switch back to READWRITE for normal registration
	setGlobalMode(t, server, "READWRITE")

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

	// Set IMPORT mode (required for import operations)
	setGlobalMode(t, server, "IMPORT")

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

	// Set IMPORT mode (required for import operations)
	setGlobalMode(t, server, "IMPORT")

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

	// Set IMPORT mode (required for import operations)
	setGlobalMode(t, server, "IMPORT")

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

// --- Context routing tests ---

// registerSchema is a test helper that registers an Avro schema under the given
// subject (which may be a qualified subject like ":.testctx:my-subject") and
// returns the schema ID. It fails the test on any unexpected error.
func registerSchema(t *testing.T, server *Server, subject, schemaStr string) int64 {
	t.Helper()
	body := types.RegisterSchemaRequest{Schema: schemaStr}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/"+subject+"/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("registerSchema(%s): expected status 200, got %d: %s", subject, w.Code, w.Body.String())
	}

	var resp types.RegisterSchemaResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("registerSchema(%s): failed to decode response: %v", subject, err)
	}
	return resp.ID
}

func TestServer_GetContexts_Default(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/contexts", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var contexts []string
	if err := json.NewDecoder(w.Body).Decode(&contexts); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// With no schemas registered, the memory store still has the default context "."
	if len(contexts) != 1 {
		t.Fatalf("Expected 1 context, got %d: %v", len(contexts), contexts)
	}
	if contexts[0] != "." {
		t.Errorf("Expected default context \".\", got %q", contexts[0])
	}
}

func TestServer_GetContexts_AfterRegistration(t *testing.T) {
	server := setupTestServer(t)

	// Register a schema using a qualified subject that targets context ".testctx"
	schema := `{"type": "record", "name": "Ctx", "fields": [{"name": "id", "type": "long"}]}`
	registerSchema(t, server, ":.testctx:my-subject", schema)

	// GET /contexts should now return both "." and ".testctx"
	req := httptest.NewRequest("GET", "/contexts", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var contexts []string
	if err := json.NewDecoder(w.Body).Decode(&contexts); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(contexts) != 2 {
		t.Fatalf("Expected 2 contexts, got %d: %v", len(contexts), contexts)
	}

	contextSet := make(map[string]bool, len(contexts))
	for _, c := range contexts {
		contextSet[c] = true
	}
	if !contextSet["."] {
		t.Error("Expected default context \".\" in contexts list")
	}
	if !contextSet[".testctx"] {
		t.Error("Expected context \".testctx\" in contexts list")
	}
}

func TestServer_RegisterSchema_QualifiedSubject(t *testing.T) {
	server := setupTestServer(t)

	schema := `{"type": "record", "name": "QualTest", "fields": [{"name": "id", "type": "long"}]}`
	body := types.RegisterSchemaRequest{Schema: schema}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/:.testctx:my-subject/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.RegisterSchemaResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.ID != 1 {
		t.Errorf("Expected schema ID 1, got %d", resp.ID)
	}
}

func TestServer_ContextIsolation_Subjects(t *testing.T) {
	server := setupTestServer(t)

	// Register a schema in the default context
	defaultSchema := `{"type": "record", "name": "Default", "fields": [{"name": "id", "type": "long"}]}`
	registerSchema(t, server, "default-subject", defaultSchema)

	// Register a schema in a non-default context using qualified subject
	ctxSchema := `{"type": "record", "name": "InCtx", "fields": [{"name": "id", "type": "long"}]}`
	registerSchema(t, server, ":.otherctx:ctx-subject", ctxSchema)

	// GET /subjects at root level should return only default context subjects
	req := httptest.NewRequest("GET", "/subjects", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var subjects []string
	if err := json.NewDecoder(w.Body).Decode(&subjects); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(subjects) != 1 {
		t.Fatalf("Expected 1 subject in default context, got %d: %v", len(subjects), subjects)
	}
	if subjects[0] != "default-subject" {
		t.Errorf("Expected \"default-subject\", got %q", subjects[0])
	}
}

func TestServer_ContextIsolation_SchemaByID(t *testing.T) {
	server := setupTestServer(t)

	// Register a schema in context ".ctxA" — it will get ID 1 in that context
	schema := `{"type": "record", "name": "Isolated", "fields": [{"name": "id", "type": "long"}]}`
	id := registerSchema(t, server, ":.ctxA:isolated-subject", schema)

	if id != 1 {
		t.Fatalf("Expected schema ID 1 in context .ctxA, got %d", id)
	}

	// GET /schemas/ids/1 at root level should return not found because ID 1
	// belongs to context ".ctxA", not the default context "."
	req := httptest.NewRequest("GET", "/schemas/ids/1", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for schema ID 1 at root (default context), got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_QualifiedSubject_GetVersion(t *testing.T) {
	server := setupTestServer(t)

	// Register a schema with a qualified subject targeting context ".ctx"
	schema := `{"type": "record", "name": "Versioned", "fields": [{"name": "id", "type": "long"}]}`
	registerSchema(t, server, ":.ctx:subj", schema)

	// GET /subjects/:.ctx:subj/versions/1 should return the schema
	req := httptest.NewRequest("GET", "/subjects/:.ctx:subj/versions/1", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp types.SubjectVersionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Subject != "subj" {
		t.Errorf("Expected subject \"subj\", got %q", resp.Subject)
	}
	if resp.Version != 1 {
		t.Errorf("Expected version 1, got %d", resp.Version)
	}
	if resp.ID != 1 {
		t.Errorf("Expected schema ID 1, got %d", resp.ID)
	}
	if resp.Schema == "" {
		t.Error("Expected non-empty schema")
	}
}

// --- Health check tests ---

// unhealthyStore wraps memory.Store but always reports unhealthy.
type unhealthyStore struct {
	*memory.Store
}

func (u *unhealthyStore) IsHealthy(_ context.Context) bool {
	return false
}

// setupUnhealthyTestServer creates a server whose storage reports unhealthy.
func setupUnhealthyTestServer(t *testing.T) *Server {
	t.Helper()

	cfg := config.DefaultConfig()
	store := &unhealthyStore{Store: memory.NewStore()}

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())

	reg := registry.New(store, schemaRegistry, compatChecker, cfg.Compatibility.DefaultLevel)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewServer(cfg, reg, logger)
}

func TestServer_LivenessCheck(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["status"] != "UP" {
		t.Errorf("Expected status UP, got %q", resp["status"])
	}
}

func TestServer_ReadinessCheck_Healthy(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["status"] != "UP" {
		t.Errorf("Expected status UP, got %q", resp["status"])
	}
}

func TestServer_ReadinessCheck_Unhealthy(t *testing.T) {
	server := setupUnhealthyTestServer(t)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["status"] != "DOWN" {
		t.Errorf("Expected status DOWN, got %q", resp["status"])
	}
	if resp["reason"] == "" {
		t.Error("Expected non-empty reason field")
	}
}

func TestServer_StartupCheck_Healthy(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/health/startup", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["status"] != "UP" {
		t.Errorf("Expected status UP, got %q", resp["status"])
	}
}

func TestServer_StartupCheck_Unhealthy(t *testing.T) {
	server := setupUnhealthyTestServer(t)

	req := httptest.NewRequest("GET", "/health/startup", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["status"] != "DOWN" {
		t.Errorf("Expected status DOWN, got %q", resp["status"])
	}
	if resp["reason"] == "" {
		t.Error("Expected non-empty reason field")
	}
}

func TestServer_LivenessCheck_AlwaysUp(t *testing.T) {
	// Liveness should return 200 even when storage is unhealthy
	server := setupUnhealthyTestServer(t)

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 even with unhealthy storage, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["status"] != "UP" {
		t.Errorf("Expected status UP, got %q", resp["status"])
	}
}

func TestServer_MethodNotAllowed(t *testing.T) {
	server := setupTestServer(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"PATCH on /subjects", "PATCH", "/subjects"},
		{"DELETE on /schemas/types", "DELETE", "/schemas/types"},
		{"POST on /config", "POST", "/config"},
		{"PATCH on /config", "PATCH", "/config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected 405, got %d", w.Code)
			}

			ct := w.Header().Get("Content-Type")
			if ct != "application/vnd.schemaregistry.v1+json" {
				t.Errorf("Expected Confluent content type, got %q", ct)
			}

			var resp types.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			if resp.ErrorCode != 405 {
				t.Errorf("Expected error_code 405, got %d", resp.ErrorCode)
			}
		})
	}
}
