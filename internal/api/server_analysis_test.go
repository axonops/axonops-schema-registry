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
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

// setupAnalysisTestServer creates a test server with all 3 schema parsers
// (Avro, Protobuf, JSON Schema) and all compatibility checkers registered.
func setupAnalysisTestServer(t *testing.T) *Server {
	t.Helper()

	cfg := config.DefaultConfig()
	store := memory.NewStore()

	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaRegistry, compatChecker, cfg.Compatibility.DefaultLevel)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewServer(cfg, reg, logger)
}

// registerTestSchema registers an Avro schema under the given subject and returns the schema ID.
func registerTestSchema(t *testing.T, server *Server, subject, schemaStr string) int64 {
	t.Helper()
	body := types.RegisterSchemaRequest{Schema: schemaStr}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/subjects/"+subject+"/versions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("registerTestSchema(%s): expected 200, got %d: %s", subject, w.Code, w.Body.String())
	}

	var resp types.RegisterSchemaResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("registerTestSchema(%s): decode error: %v", subject, err)
	}
	return resp.ID
}

// doAnalysisRequest is a helper that performs an HTTP request and returns the recorder.
func doAnalysisRequest(t *testing.T, server *Server, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	var req *http.Request
	if bodyReader != nil {
		req = httptest.NewRequest(method, path, bodyReader)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

// parseAnalysisResponse decodes the response body into a map.
func parseAnalysisResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v\nBody: %s", err, w.Body.String())
	}
	return result
}

// --- Schema Analysis ---

func TestAnalysis_ValidateSchema(t *testing.T) {
	server := setupAnalysisTestServer(t)

	t.Run("valid schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/validate", map[string]string{
			"schema": `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["is_valid"] != true {
			t.Errorf("Expected is_valid=true, got %v", result["is_valid"])
		}
	})

	t.Run("invalid schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/validate", map[string]string{
			"schema": `{invalid`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["is_valid"] != false {
			t.Errorf("Expected is_valid=false, got %v", result["is_valid"])
		}
	})

	t.Run("missing schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/validate", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_NormalizeSchema(t *testing.T) {
	server := setupAnalysisTestServer(t)

	t.Run("valid schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/normalize", map[string]string{
			"schema": `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["canonical"] == nil || result["canonical"] == "" {
			t.Error("Expected non-empty canonical")
		}
		if result["fingerprint"] == nil || result["fingerprint"] == "" {
			t.Error("Expected non-empty fingerprint")
		}
	})

	t.Run("invalid schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/normalize", map[string]string{
			"schema": `{invalid`,
		})
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422, got %d", w.Code)
		}
	})

	t.Run("missing schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/normalize", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_SearchSchemas(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "search-user", `{"type":"record","name":"User","fields":[{"name":"user_id","type":"long"}]}`)
	registerTestSchema(t, server, "search-order", `{"type":"record","name":"Order","fields":[{"name":"order_id","type":"long"}]}`)

	t.Run("find by substring", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/search", map[string]interface{}{
			"query": "user_id",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count != 1 {
			t.Errorf("Expected count=1, got %d", count)
		}
	})

	t.Run("missing query", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/search", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_FindSchemasByField(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "field-test", `{"type":"record","name":"FieldTest","fields":[{"name":"email","type":"string"},{"name":"age","type":"int"}]}`)

	t.Run("exact mode", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/search/field", map[string]interface{}{
			"field": "email",
			"mode":  "exact",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count < 1 {
			t.Errorf("Expected at least 1 match, got %d", count)
		}
	})

	t.Run("regex mode", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/search/field", map[string]interface{}{
			"field": "e.*l",
			"mode":  "regex",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count < 1 {
			t.Errorf("Expected at least 1 match for regex, got %d", count)
		}
	})

	t.Run("missing field", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/search/field", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_FindSchemasByType(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "type-test", `{"type":"record","name":"TypeTest","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`)

	t.Run("find by type", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/search/type", map[string]interface{}{
			"type_pattern": "long",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count < 1 {
			t.Errorf("Expected at least 1 match, got %d", count)
		}
	})

	t.Run("missing type_pattern", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/search/type", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_FindSimilarSchemas(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "similar-a", `{"type":"record","name":"A","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`)
	registerTestSchema(t, server, "similar-b", `{"type":"record","name":"B","fields":[{"name":"id","type":"long"},{"name":"email","type":"string"}]}`)

	t.Run("finds similar", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/similar", map[string]interface{}{
			"subject":   "similar-a",
			"threshold": 0.1,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count < 1 {
			t.Errorf("Expected at least 1 similar schema, got %d", count)
		}
		similar := result["similar"].([]interface{})
		first := similar[0].(map[string]interface{})
		if first["subject"] == "similar-a" {
			t.Error("Source subject should be excluded from similar results")
		}
	})

	t.Run("subject not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/similar", map[string]interface{}{
			"subject": "nonexistent",
		})
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("missing subject", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/similar", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// --- Schema Quality & Complexity ---

func TestAnalysis_ScoreSchemaQuality(t *testing.T) {
	server := setupAnalysisTestServer(t)

	t.Run("inline schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/quality", map[string]interface{}{
			"schema": `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["overall_score"] == nil {
			t.Error("Expected overall_score in response")
		}
		if result["grade"] == nil {
			t.Error("Expected grade in response")
		}
	})

	t.Run("by subject", func(t *testing.T) {
		registerTestSchema(t, server, "quality-test", `{"type":"record","name":"Quality","fields":[{"name":"id","type":"long"}]}`)
		w := doAnalysisRequest(t, server, "POST", "/schemas/quality", map[string]interface{}{
			"subject": "quality-test",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["overall_score"] == nil {
			t.Error("Expected overall_score in response")
		}
	})

	t.Run("missing schema and subject", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/quality", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_GetSchemaComplexity(t *testing.T) {
	server := setupAnalysisTestServer(t)

	t.Run("simple schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/complexity", map[string]interface{}{
			"schema": `{"type":"record","name":"Simple","fields":[{"name":"id","type":"long"}]}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["field_count"] == nil {
			t.Error("Expected field_count in response")
		}
		if result["max_depth"] == nil {
			t.Error("Expected max_depth in response")
		}
		if result["grade"] != "A" {
			t.Errorf("Expected grade A for simple schema, got %v", result["grade"])
		}
	})

	t.Run("by subject", func(t *testing.T) {
		registerTestSchema(t, server, "complexity-test", `{"type":"record","name":"Complex","fields":[{"name":"a","type":"int"},{"name":"b","type":"string"}]}`)
		w := doAnalysisRequest(t, server, "POST", "/schemas/complexity", map[string]interface{}{
			"subject": "complexity-test",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["field_count"] == nil {
			t.Error("Expected field_count in response")
		}
	})

	t.Run("missing schema and subject", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/schemas/complexity", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

// --- Subject Operations ---

func TestAnalysis_ValidateSubjectName(t *testing.T) {
	server := setupAnalysisTestServer(t)

	t.Run("valid topic_name", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/validate", map[string]interface{}{
			"subject": "my-topic-value",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["valid"] != true {
			t.Errorf("Expected valid=true for -value suffix, got %v", result["valid"])
		}
		if result["strategy"] != "topic_name" {
			t.Errorf("Expected strategy=topic_name, got %v", result["strategy"])
		}
	})

	t.Run("invalid topic_name", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/validate", map[string]interface{}{
			"subject": "my-topic",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["valid"] != false {
			t.Errorf("Expected valid=false without -value/-key suffix, got %v", result["valid"])
		}
	})

	t.Run("missing subject", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/validate", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_MatchSubjects(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "match-alpha", `{"type":"record","name":"Alpha","fields":[{"name":"id","type":"long"}]}`)
	registerTestSchema(t, server, "match-beta", `{"type":"record","name":"Beta","fields":[{"name":"id","type":"long"}]}`)

	t.Run("regex mode", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/match", map[string]interface{}{
			"pattern": "match-.*",
			"mode":    "regex",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count < 2 {
			t.Errorf("Expected at least 2 matches, got %d", count)
		}
	})

	t.Run("glob mode", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/match", map[string]interface{}{
			"pattern": "match-*",
			"mode":    "glob",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count < 2 {
			t.Errorf("Expected at least 2 matches for glob, got %d", count)
		}
	})

	t.Run("missing pattern", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/match", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_CountSubjects(t *testing.T) {
	server := setupAnalysisTestServer(t)

	t.Run("empty registry", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/count", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count != 0 {
			t.Errorf("Expected count=0, got %d", count)
		}
	})

	t.Run("after registration", func(t *testing.T) {
		registerTestSchema(t, server, "count-a", `{"type":"record","name":"CountA","fields":[{"name":"id","type":"long"}]}`)
		registerTestSchema(t, server, "count-b", `{"type":"record","name":"CountB","fields":[{"name":"id","type":"long"}]}`)

		w := doAnalysisRequest(t, server, "GET", "/subjects/count", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count != 2 {
			t.Errorf("Expected count=2, got %d", count)
		}
	})
}

// --- History & Export ---

func TestAnalysis_GetSchemaHistory(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "history-test", `{"type":"record","name":"History","fields":[{"name":"id","type":"long"}]}`)

	t.Run("get history", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/history-test/history", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count != 1 {
			t.Errorf("Expected count=1, got %d", count)
		}
		if result["subject"] != "history-test" {
			t.Errorf("Expected subject=history-test, got %v", result["subject"])
		}
	})

	t.Run("subject not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/nonexistent/history", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

func TestAnalysis_CountVersions(t *testing.T) {
	server := setupAnalysisTestServer(t)

	// Set compat to NONE so we can register any schema
	setGlobalMode(t, server, "READWRITE")
	setAnalysisConfig(t, server, "NONE")

	registerTestSchema(t, server, "vcount-test", `{"type":"record","name":"VCount","fields":[{"name":"id","type":"long"}]}`)
	registerTestSchema(t, server, "vcount-test", `{"type":"record","name":"VCount","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`)

	t.Run("count versions", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/vcount-test/versions/count", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		count := int(result["count"].(float64))
		if count != 2 {
			t.Errorf("Expected count=2, got %d", count)
		}
	})

	t.Run("subject not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/nonexistent/versions/count", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

func TestAnalysis_ExportSubject(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "export-test", `{"type":"record","name":"Export","fields":[{"name":"id","type":"long"}]}`)

	t.Run("export subject", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/export-test/export", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["subject"] != "export-test" {
			t.Errorf("Expected subject=export-test, got %v", result["subject"])
		}
		versions := result["versions"].([]interface{})
		if len(versions) < 1 {
			t.Error("Expected at least 1 version in export")
		}
		first := versions[0].(map[string]interface{})
		if first["schema"] == nil || first["schema"] == "" {
			t.Error("Expected non-empty schema in export entry")
		}
	})

	t.Run("subject not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/nonexistent/export", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

func TestAnalysis_ExportSchema(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "export-schema-test", `{"type":"record","name":"ExportSchema","fields":[{"name":"id","type":"long"}]}`)

	t.Run("export version", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/export-schema-test/versions/1/export", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["schema"] == nil || result["schema"] == "" {
			t.Error("Expected non-empty schema")
		}
		if result["compatibility_level"] == nil {
			t.Error("Expected compatibility_level in response")
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/export-schema-test/versions/abc/export", nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("version not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/export-schema-test/versions/999/export", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

// --- Diff, Evolution & Migration ---

func TestAnalysis_DiffSchemas(t *testing.T) {
	server := setupAnalysisTestServer(t)

	// Set compat to NONE to register incompatible schemas
	setAnalysisConfig(t, server, "NONE")

	registerTestSchema(t, server, "diff-test", `{"type":"record","name":"Diff","fields":[{"name":"id","type":"long"},{"name":"old_field","type":"string"}]}`)
	registerTestSchema(t, server, "diff-test", `{"type":"record","name":"Diff","fields":[{"name":"id","type":"long"},{"name":"new_field","type":"string"}]}`)

	t.Run("detect changes", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/diff-test/diff", map[string]interface{}{
			"version1": 1,
			"version2": 2,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["subject"] != "diff-test" {
			t.Errorf("Expected subject=diff-test, got %v", result["subject"])
		}
		// Should detect added and removed fields
		added := result["added"]
		removed := result["removed"]
		if added == nil && removed == nil {
			t.Error("Expected added or removed fields in diff")
		}
	})

	t.Run("subject not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/nonexistent/diff", map[string]interface{}{
			"version1": 1,
			"version2": 2,
		})
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

func TestAnalysis_SuggestSchemaEvolution(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "evolve-test", `{"type":"record","name":"Evolve","fields":[{"name":"id","type":"long"}]}`)

	t.Run("get suggestions", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/evolve-test/evolve", map[string]interface{}{
			"changes": []map[string]string{},
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["subject"] != "evolve-test" {
			t.Errorf("Expected subject=evolve-test, got %v", result["subject"])
		}
		if result["current_version"] == nil {
			t.Error("Expected current_version in response")
		}
		if result["compatibility_level"] == nil {
			t.Error("Expected compatibility_level in response")
		}
	})

	t.Run("subject not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/nonexistent/evolve", map[string]interface{}{})
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

func TestAnalysis_PlanMigrationPath(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "migrate-test", `{"type":"record","name":"Migrate","fields":[{"name":"id","type":"long"},{"name":"old_field","type":"string"}]}`)

	t.Run("plan migration", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/migrate-test/migrate", map[string]interface{}{
			"target_schema": `{"type":"record","name":"Migrate","fields":[{"name":"id","type":"long"},{"name":"new_field","type":"int"}]}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		steps := result["steps"].([]interface{})
		if len(steps) < 1 {
			t.Error("Expected at least 1 migration step")
		}
	})

	t.Run("missing target_schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/migrate-test/migrate", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("subject not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/subjects/nonexistent/migrate", map[string]interface{}{
			"target_schema": `{"type":"string"}`,
		})
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

// --- Dependencies ---

func TestAnalysis_GetDependencyGraph(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "dep-test", `{"type":"record","name":"Dep","fields":[{"name":"id","type":"long"}]}`)

	t.Run("get dependencies", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/dep-test/versions/1/dependencies", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		refs := result["referenced_by"].([]interface{})
		if len(refs) != 0 {
			t.Errorf("Expected empty referenced_by, got %d entries", len(refs))
		}
		if result["subject"] != "dep-test" {
			t.Errorf("Expected subject=dep-test, got %v", result["subject"])
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/dep-test/versions/abc/dependencies", nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("version not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/subjects/dep-test/versions/999/dependencies", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

// --- Compatibility Analysis ---

func TestAnalysis_CheckCompatibilityMulti(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "compat-multi-a", `{"type":"record","name":"CompatA","fields":[{"name":"id","type":"long"}]}`)
	registerTestSchema(t, server, "compat-multi-b", `{"type":"record","name":"CompatB","fields":[{"name":"id","type":"long"}]}`)

	t.Run("check multiple subjects", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/check", map[string]interface{}{
			"schema":   `{"type":"record","name":"CompatA","fields":[{"name":"id","type":"long"},{"name":"extra","type":"string","default":""}]}`,
			"subjects": []string{"compat-multi-a", "compat-multi-b"},
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		results := result["results"].([]interface{})
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("missing schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/check", map[string]interface{}{
			"subjects": []string{"compat-multi-a"},
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_SuggestCompatibleChange(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "suggest-test", `{"type":"record","name":"Suggest","fields":[{"name":"id","type":"long"}]}`)

	t.Run("backward suggestions", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/subjects/suggest-test/suggest", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["subject"] != "suggest-test" {
			t.Errorf("Expected subject=suggest-test, got %v", result["subject"])
		}
		suggestions := result["suggestions"].([]interface{})
		if len(suggestions) == 0 {
			t.Error("Expected at least 1 suggestion")
		}
	})

	t.Run("NONE compat", func(t *testing.T) {
		setAnalysisSubjectConfig(t, server, "suggest-test", "NONE")
		w := doAnalysisRequest(t, server, "POST", "/compatibility/subjects/suggest-test/suggest", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		suggestions := result["suggestions"].([]interface{})
		found := false
		for _, s := range suggestions {
			if s == "Any change is allowed (no compatibility checks)" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'Any change is allowed' suggestion for NONE compat")
		}
	})
}

func TestAnalysis_ExplainCompatibilityFailure(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "explain-test", `{"type":"record","name":"Explain","fields":[{"name":"id","type":"long"}]}`)

	t.Run("compatible schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/subjects/explain-test/explain", map[string]interface{}{
			"schema": `{"type":"record","name":"Explain","fields":[{"name":"id","type":"long"},{"name":"extra","type":"string","default":""}]}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["is_compatible"] != true {
			t.Errorf("Expected is_compatible=true, got %v", result["is_compatible"])
		}
	})

	t.Run("incompatible schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/subjects/explain-test/explain", map[string]interface{}{
			"schema": `{"type":"record","name":"Explain","fields":[{"name":"id","type":"long"},{"name":"required_field","type":"string"}]}`,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["is_compatible"] != false {
			t.Errorf("Expected is_compatible=false, got %v", result["is_compatible"])
		}
	})

	t.Run("missing schema", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/subjects/explain-test/explain", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})
}

func TestAnalysis_CompareSubjects(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "compare-a", `{"type":"record","name":"CompareA","fields":[{"name":"id","type":"long"},{"name":"shared","type":"string"},{"name":"only_a","type":"int"}]}`)
	registerTestSchema(t, server, "compare-b", `{"type":"record","name":"CompareB","fields":[{"name":"id","type":"long"},{"name":"shared","type":"string"},{"name":"only_b","type":"int"}]}`)

	t.Run("compare two subjects", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/compare", map[string]interface{}{
			"subject1": "compare-a",
			"subject2": "compare-b",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		shared := result["shared"].([]interface{})
		if len(shared) < 2 {
			t.Errorf("Expected at least 2 shared fields (id, shared), got %d", len(shared))
		}
		onlyIn1 := result["only_in_sub1"].([]interface{})
		if len(onlyIn1) < 1 {
			t.Error("Expected at least 1 field only in subject1")
		}
		onlyIn2 := result["only_in_sub2"].([]interface{})
		if len(onlyIn2) < 1 {
			t.Error("Expected at least 1 field only in subject2")
		}
	})

	t.Run("missing subjects", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/compare", map[string]interface{}{
			"subject1": "compare-a",
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("subject not found", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "POST", "/compatibility/compare", map[string]interface{}{
			"subject1": "compare-a",
			"subject2": "nonexistent",
		})
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})
}

// --- Statistics ---

func TestAnalysis_GetRegistryStatistics(t *testing.T) {
	server := setupAnalysisTestServer(t)

	t.Run("empty registry", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/statistics", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["subject_count"] == nil {
			t.Error("Expected subject_count in response")
		}
		if result["version_count"] == nil {
			t.Error("Expected version_count in response")
		}
		if result["type_counts"] == nil {
			t.Error("Expected type_counts in response")
		}
	})

	t.Run("with schemas", func(t *testing.T) {
		registerTestSchema(t, server, "stats-a", `{"type":"record","name":"StatsA","fields":[{"name":"id","type":"long"}]}`)
		registerTestSchema(t, server, "stats-b", `{"type":"record","name":"StatsB","fields":[{"name":"id","type":"long"}]}`)

		w := doAnalysisRequest(t, server, "GET", "/statistics", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		subjectCount := int(result["subject_count"].(float64))
		if subjectCount < 2 {
			t.Errorf("Expected at least 2 subjects, got %d", subjectCount)
		}
	})
}

func TestAnalysis_CheckFieldConsistency(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "consistency-a", `{"type":"record","name":"ConsA","fields":[{"name":"user_id","type":"long"}]}`)
	registerTestSchema(t, server, "consistency-b", `{"type":"record","name":"ConsB","fields":[{"name":"user_id","type":"long"}]}`)

	t.Run("consistent field", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/statistics/fields/user_id", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["consistent"] != true {
			t.Errorf("Expected consistent=true for same type, got %v", result["consistent"])
		}
	})

	t.Run("inconsistent field", func(t *testing.T) {
		registerTestSchema(t, server, "consistency-c", `{"type":"record","name":"ConsC","fields":[{"name":"user_id","type":"string"}]}`)

		w := doAnalysisRequest(t, server, "GET", "/statistics/fields/user_id", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		if result["consistent"] != false {
			t.Errorf("Expected consistent=false for different types, got %v", result["consistent"])
		}
	})
}

func TestAnalysis_DetectSchemaPatterns(t *testing.T) {
	server := setupAnalysisTestServer(t)

	registerTestSchema(t, server, "pattern-a", `{"type":"record","name":"PatA","fields":[{"name":"id","type":"long"},{"name":"created_at","type":"string"}]}`)
	registerTestSchema(t, server, "pattern-b", `{"type":"record","name":"PatB","fields":[{"name":"id","type":"long"},{"name":"updated_at","type":"string"}]}`)

	t.Run("detect common fields", func(t *testing.T) {
		w := doAnalysisRequest(t, server, "GET", "/statistics/patterns", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
		result := parseAnalysisResponse(t, w)
		commonFields := result["common_fields"].([]interface{})
		// "id" should appear in both subjects
		found := false
		for _, cf := range commonFields {
			entry := cf.(map[string]interface{})
			if entry["field"] == "id" {
				found = true
				count := int(entry["count"].(float64))
				if count < 2 {
					t.Errorf("Expected id count >= 2, got %d", count)
				}
				break
			}
		}
		if !found {
			t.Error("Expected 'id' in common_fields")
		}
	})
}

// --- Test helpers ---

// setAnalysisConfig sets the global compatibility level.
func setAnalysisConfig(t *testing.T, server *Server, level string) {
	t.Helper()
	body := types.ConfigRequest{Compatibility: level}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/config", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Failed to set config to %s: %d %s", level, w.Code, w.Body.String())
	}
}

// setAnalysisSubjectConfig sets the compatibility level for a specific subject.
func setAnalysisSubjectConfig(t *testing.T, server *Server, subject, level string) {
	t.Helper()
	body := types.ConfigRequest{Compatibility: level}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/config/"+subject, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Failed to set subject config to %s: %d %s", level, w.Code, w.Body.String())
	}
}
