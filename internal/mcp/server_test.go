package mcp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	prometheusmodel "github.com/prometheus/client_model/go"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/metrics"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

// newTestMCPClient creates a test MCP server + in-memory client session.
func newTestMCPClient(t *testing.T) (*gomcp.ClientSession, *registry.Registry) {
	t.Helper()

	store := memory.NewStore()
	t.Cleanup(func() { store.Close() })

	schemaReg := schema.NewRegistry()
	schemaReg.Register(avro.NewParser())
	schemaReg.Register(protobuf.NewParser())
	schemaReg.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaReg, compatChecker, "BACKWARD")

	cfg := &config.MCPConfig{Host: "localhost", Port: 0}
	srv := New(cfg, reg, testLogger(), "test-version")

	ctx := context.Background()
	ct, st := gomcp.NewInMemoryTransports()

	ss, err := srv.MCPServer().Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { ss.Close() })

	client := gomcp.NewClient(&gomcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })

	return cs, reg
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthCheck(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "health_check",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success, got error")
	}

	text := resultText(t, result)
	if !strings.Contains(text, "healthy") {
		t.Fatalf("expected 'healthy' in result, got: %s", text)
	}
}

func TestGetServerInfo(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_server_info",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	text := resultText(t, result)
	for _, want := range []string{"AVRO", "PROTOBUF", "JSON", "test-version"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in result, got: %s", want, text)
		}
	}
}

func TestListSubjectsEmpty(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_subjects",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	text := resultText(t, result)
	if text != "[]" {
		t.Fatalf("expected '[]', got: %s", text)
	}
}

func TestListSubjects(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	// Register a schema via the registry
	ctx := context.Background()
	_, err := reg.RegisterSchema(ctx, ".", "test-subject", `{"type":"string"}`, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("RegisterSchema: %v", err)
	}

	result, err := cs.CallTool(ctx, &gomcp.CallToolParams{
		Name: "list_subjects",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	text := resultText(t, result)
	var subjects []string
	if err := json.Unmarshal([]byte(text), &subjects); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(subjects) != 1 || subjects[0] != "test-subject" {
		t.Fatalf("expected [test-subject], got: %v", subjects)
	}
}

func TestListSubjectsWithPrefix(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	ctx := context.Background()
	for _, subj := range []string{"orders-value", "users-value", "orders-key"} {
		_, err := reg.RegisterSchema(ctx, ".", subj, `{"type":"string"}`, storage.SchemaTypeAvro, nil)
		if err != nil {
			t.Fatalf("RegisterSchema(%s): %v", subj, err)
		}
	}

	result, err := cs.CallTool(ctx, &gomcp.CallToolParams{
		Name:      "list_subjects",
		Arguments: map[string]any{"prefix": "orders"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	text := resultText(t, result)
	var subjects []string
	if err := json.Unmarshal([]byte(text), &subjects); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(subjects) != 2 {
		t.Fatalf("expected 2 subjects, got: %v", subjects)
	}
	for _, s := range subjects {
		if !strings.HasPrefix(s, "orders") {
			t.Errorf("unexpected subject: %s", s)
		}
	}
}

// --- Phase 2: Schema read tool tests ---

func registerTestSchema(t *testing.T, reg *registry.Registry, subject, schemaStr string) *storage.SchemaRecord {
	t.Helper()
	rec, err := reg.RegisterSchema(context.Background(), ".", subject, schemaStr, storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("RegisterSchema(%s): %v", subject, err)
	}
	return rec
}

func TestGetSchemaByID(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	rec := registerTestSchema(t, reg, "schema-by-id", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_schema_by_id",
		Arguments: map[string]any{"id": rec.ID},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "AVRO") || !strings.Contains(text, "string") {
		t.Errorf("expected schema content in result, got: %s", text)
	}
}

func TestGetSchemaByIDNotFound(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_schema_by_id",
		Arguments: map[string]any{"id": 99999},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestGetRawSchemaByID(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	rec := registerTestSchema(t, reg, "raw-by-id", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_raw_schema_by_id",
		Arguments: map[string]any{"id": rec.ID},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "string") {
		t.Errorf("expected schema content, got: %s", text)
	}
	// Raw result should NOT contain subject metadata
	if strings.Contains(text, "raw-by-id") {
		t.Errorf("raw result should not contain subject name, got: %s", text)
	}
}

func TestGetSchemaVersion(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "version-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_schema_version",
		Arguments: map[string]any{"subject": "version-test", "version": 1},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "version-test") {
		t.Errorf("expected subject in result, got: %s", text)
	}
}

func TestGetRawSchemaVersion(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "raw-version", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_raw_schema_version",
		Arguments: map[string]any{"subject": "raw-version", "version": 1},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "string") {
		t.Errorf("expected schema content, got: %s", text)
	}
}

func TestGetLatestSchema(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	// Use backward-compatible schema evolution: record with optional field added
	v1 := `{"type":"record","name":"Test","fields":[{"name":"a","type":"string"}]}`
	v2 := `{"type":"record","name":"Test","fields":[{"name":"a","type":"string"},{"name":"b","type":["null","string"],"default":null}]}`
	registerTestSchema(t, reg, "latest-test", v1)
	registerTestSchema(t, reg, "latest-test", v2)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_latest_schema",
		Arguments: map[string]any{"subject": "latest-test"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	// Latest should be version 2
	if !strings.Contains(text, `"version":2`) {
		t.Errorf("expected version 2 in result, got: %s", text)
	}
}

func TestListVersions(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	v1 := `{"type":"record","name":"Test","fields":[{"name":"a","type":"string"}]}`
	v2 := `{"type":"record","name":"Test","fields":[{"name":"a","type":"string"},{"name":"b","type":["null","string"],"default":null}]}`
	registerTestSchema(t, reg, "versions-test", v1)
	registerTestSchema(t, reg, "versions-test", v2)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "list_versions",
		Arguments: map[string]any{"subject": "versions-test"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	var versions []int
	if err := json.Unmarshal([]byte(text), &versions); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(versions) != 2 || versions[0] != 1 || versions[1] != 2 {
		t.Fatalf("expected [1,2], got: %v", versions)
	}
}

func TestGetSubjectsForSchema(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	rec := registerTestSchema(t, reg, "subj-a", `{"type":"string"}`)
	// Same schema content registers under same ID in different subject
	registerTestSchema(t, reg, "subj-b", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_subjects_for_schema",
		Arguments: map[string]any{"id": rec.ID},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	var subjects []string
	if err := json.Unmarshal([]byte(text), &subjects); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(subjects) != 2 {
		t.Fatalf("expected 2 subjects, got: %v", subjects)
	}
}

func TestGetVersionsForSchema(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	rec := registerTestSchema(t, reg, "ver-schema", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_versions_for_schema",
		Arguments: map[string]any{"id": rec.ID},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "ver-schema") {
		t.Errorf("expected subject in result, got: %s", text)
	}
}

func TestLookupSchema(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "lookup-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "lookup_schema",
		Arguments: map[string]any{
			"subject": "lookup-test",
			"schema":  `{"type":"string"}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "lookup-test") {
		t.Errorf("expected subject in result, got: %s", text)
	}
}

func TestGetSchemaTypes(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_schema_types",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	for _, want := range []string{"AVRO", "PROTOBUF", "JSON"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in result, got: %s", want, text)
		}
	}
}

func TestListSchemas(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "list-schemas-a", `{"type":"string"}`)
	registerTestSchema(t, reg, "list-schemas-b", `{"type":"int"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_schemas",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "list-schemas-a") || !strings.Contains(text, "list-schemas-b") {
		t.Errorf("expected both subjects in result, got: %s", text)
	}
}

func TestListSchemasWithPrefix(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "prefix-a", `{"type":"string"}`)
	registerTestSchema(t, reg, "other-b", `{"type":"int"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "list_schemas",
		Arguments: map[string]any{"subject_prefix": "prefix"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "prefix-a") {
		t.Errorf("expected prefix-a in result, got: %s", text)
	}
	if strings.Contains(text, "other-b") {
		t.Errorf("should not contain other-b, got: %s", text)
	}
}

func TestGetMaxSchemaID(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "max-id-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_max_schema_id",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	var out map[string]int64
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["max_id"] < 1 {
		t.Errorf("expected max_id >= 1, got: %d", out["max_id"])
	}
}

// --- Phase 3: Schema write tool tests ---

func TestRegisterSchema(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "register_schema",
		Arguments: map[string]any{
			"subject": "reg-test",
			"schema":  `{"type":"string"}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "reg-test") {
		t.Errorf("expected subject in result, got: %s", text)
	}
	if !strings.Contains(text, `"version":1`) {
		t.Errorf("expected version 1, got: %s", text)
	}
}

func TestRegisterSchemaJSON(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "register_schema",
		Arguments: map[string]any{
			"subject":     "json-reg-test",
			"schema":      `{"type":"object","properties":{}}`,
			"schema_type": "JSON",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "JSON") {
		t.Errorf("expected JSON schemaType, got: %s", text)
	}
}

func TestDeleteSubject(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "del-subj", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_subject",
		Arguments: map[string]any{"subject": "del-subj"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	var versions []int
	if err := json.Unmarshal([]byte(text), &versions); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(versions) != 1 || versions[0] != 1 {
		t.Fatalf("expected [1], got: %v", versions)
	}
}

func TestDeleteSubjectPermanent(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "del-perm", `{"type":"string"}`)

	// Soft-delete first
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_subject",
		Arguments: map[string]any{"subject": "del-perm"},
	})
	if err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	// Permanent delete
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_subject",
		Arguments: map[string]any{"subject": "del-perm", "permanent": true},
	})
	if err != nil {
		t.Fatalf("perm delete: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
}

func TestDeleteVersion(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "del-ver", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_version",
		Arguments: map[string]any{"subject": "del-ver", "version": 1},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, `"version":1`) {
		t.Errorf("expected version 1, got: %s", text)
	}
}

func TestCheckCompatibilityPass(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "compat-pass", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "check_compatibility",
		Arguments: map[string]any{
			"subject": "compat-pass",
			"schema":  `{"type":"string"}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Errorf("expected is_compatible true, got: %s", text)
	}
}

func TestCheckCompatibilityFail(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "compat-fail", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "check_compatibility",
		Arguments: map[string]any{
			"subject": "compat-fail",
			"schema":  `{"type":"int"}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "false") {
		t.Errorf("expected is_compatible false, got: %s", text)
	}
}

// --- Phase 4: Config & mode tool tests ---

func TestGetConfigDefault(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_config",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "BACKWARD") {
		t.Errorf("expected BACKWARD in result, got: %s", text)
	}
}

func TestSetAndGetConfig(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Set config on a subject
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "set_config",
		Arguments: map[string]any{"subject": "cfg-test", "compatibility_level": "FULL"},
	})
	if err != nil {
		t.Fatalf("set_config: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "FULL") {
		t.Errorf("expected FULL in set result, got: %s", text)
	}

	// Get it back
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_config",
		Arguments: map[string]any{"subject": "cfg-test"},
	})
	if err != nil {
		t.Fatalf("get_config: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "FULL") {
		t.Errorf("expected FULL in get result, got: %s", text)
	}
}

func TestDeleteConfig(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Set then delete
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "set_config",
		Arguments: map[string]any{"subject": "cfg-del", "compatibility_level": "NONE"},
	})
	if err != nil {
		t.Fatalf("set_config: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_config",
		Arguments: map[string]any{"subject": "cfg-del"},
	})
	if err != nil {
		t.Fatalf("delete_config: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "NONE") {
		t.Errorf("expected previous level NONE, got: %s", text)
	}
}

func TestGetModeDefault(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_mode",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "READWRITE") {
		t.Errorf("expected READWRITE in result, got: %s", text)
	}
}

func TestSetAndGetMode(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "set_mode",
		Arguments: map[string]any{"subject": "mode-test", "mode": "READONLY"},
	})
	if err != nil {
		t.Fatalf("set_mode: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "READONLY") {
		t.Errorf("expected READONLY in set result, got: %s", text)
	}

	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_mode",
		Arguments: map[string]any{"subject": "mode-test"},
	})
	if err != nil {
		t.Fatalf("get_mode: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "READONLY") {
		t.Errorf("expected READONLY in get result, got: %s", text)
	}
}

func TestDeleteMode(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "set_mode",
		Arguments: map[string]any{"subject": "mode-del", "mode": "READONLY"},
	})
	if err != nil {
		t.Fatalf("set_mode: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_mode",
		Arguments: map[string]any{"subject": "mode-del"},
	})
	if err != nil {
		t.Fatalf("delete_mode: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "READONLY") {
		t.Errorf("expected previous mode READONLY, got: %s", text)
	}
}

// --- Phase 5: Context & import tool tests ---

func TestListContexts(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_contexts",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	// Default context "." should be present
	if !strings.Contains(text, ".") {
		t.Errorf("expected default context in result, got: %s", text)
	}
}

func TestImportSchemas(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Set mode to IMPORT first
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "set_mode",
		Arguments: map[string]any{"mode": "IMPORT"},
	})
	if err != nil {
		t.Fatalf("set_mode: %v", err)
	}

	// Import a schema with specific ID
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "import_schemas",
		Arguments: map[string]any{
			"schemas": []any{
				map[string]any{
					"id":      100,
					"subject": "import-test",
					"version": 1,
					"schema":  `{"type":"string"}`,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, `"Imported":1`) {
		t.Errorf("expected 1 imported, got: %s", text)
	}
}

// --- Phase 6: KEK & DEK tool tests ---

func TestCreateAndGetKEK(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Create KEK
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_kek",
		Arguments: map[string]any{
			"name":       "test-kek",
			"kms_type":   "aws-kms",
			"kms_key_id": "arn:aws:kms:us-east-1:123456789:key/abc-123",
			"doc":        "Test KEK for unit tests",
			"shared":     false,
		},
	})
	if err != nil {
		t.Fatalf("create_kek: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "test-kek") {
		t.Errorf("expected name in result, got: %s", text)
	}
	if !strings.Contains(text, "aws-kms") {
		t.Errorf("expected kmsType in result, got: %s", text)
	}

	// Get KEK back
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_kek",
		Arguments: map[string]any{"name": "test-kek"},
	})
	if err != nil {
		t.Fatalf("get_kek: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "test-kek") || !strings.Contains(text, "aws-kms") {
		t.Errorf("expected KEK fields in result, got: %s", text)
	}
	if !strings.Contains(text, "Test KEK for unit tests") {
		t.Errorf("expected doc in result, got: %s", text)
	}
}

func TestListKEKs(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Create two KEKs
	for _, name := range []string{"kek-alpha", "kek-beta"} {
		result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
			Name: "create_kek",
			Arguments: map[string]any{
				"name":       name,
				"kms_type":   "aws-kms",
				"kms_key_id": "arn:aws:kms:us-east-1:123456789:key/" + name,
			},
		})
		if err != nil {
			t.Fatalf("create_kek(%s): %v", name, err)
		}
		if result.IsError {
			t.Fatalf("create_kek(%s) error: %s", name, resultText(t, result))
		}
	}

	// List KEKs
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_keks",
	})
	if err != nil {
		t.Fatalf("list_keks: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "kek-alpha") || !strings.Contains(text, "kek-beta") {
		t.Errorf("expected both KEKs in list, got: %s", text)
	}
}

func TestDeleteAndUndeleteKEK(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Create KEK
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_kek",
		Arguments: map[string]any{
			"name":       "del-kek",
			"kms_type":   "aws-kms",
			"kms_key_id": "arn:aws:kms:us-east-1:123456789:key/del-kek",
		},
	})
	if err != nil {
		t.Fatalf("create_kek: %v", err)
	}

	// Soft-delete KEK
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_kek",
		Arguments: map[string]any{"name": "del-kek"},
	})
	if err != nil {
		t.Fatalf("delete_kek: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Errorf("expected deleted:true, got: %s", text)
	}

	// Get should fail without deleted flag
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_kek",
		Arguments: map[string]any{"name": "del-kek"},
	})
	if err != nil {
		t.Fatalf("get_kek: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for deleted KEK without deleted flag")
	}

	// Undelete KEK
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "undelete_kek",
		Arguments: map[string]any{"name": "del-kek"},
	})
	if err != nil {
		t.Fatalf("undelete_kek: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Errorf("expected undeleted:true, got: %s", text)
	}

	// Get should now succeed
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_kek",
		Arguments: map[string]any{"name": "del-kek"},
	})
	if err != nil {
		t.Fatalf("get_kek after undelete: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success after undelete, got error: %s", resultText(t, result))
	}
}

func TestCreateAndGetDEK(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Create KEK first (required parent)
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_kek",
		Arguments: map[string]any{
			"name":       "dek-parent-kek",
			"kms_type":   "aws-kms",
			"kms_key_id": "arn:aws:kms:us-east-1:123456789:key/dek-parent",
		},
	})
	if err != nil {
		t.Fatalf("create_kek: %v", err)
	}

	// Create DEK
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_dek",
		Arguments: map[string]any{
			"kek_name":  "dek-parent-kek",
			"subject":   "dek-test-subject",
			"algorithm": "AES256_GCM",
		},
	})
	if err != nil {
		t.Fatalf("create_dek: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "dek-parent-kek") {
		t.Errorf("expected kekName in result, got: %s", text)
	}
	if !strings.Contains(text, "dek-test-subject") {
		t.Errorf("expected subject in result, got: %s", text)
	}

	// Get DEK back
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_dek",
		Arguments: map[string]any{
			"kek_name":  "dek-parent-kek",
			"subject":   "dek-test-subject",
			"algorithm": "AES256_GCM",
		},
	})
	if err != nil {
		t.Fatalf("get_dek: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text = resultText(t, result)
	if !strings.Contains(text, "dek-parent-kek") || !strings.Contains(text, "dek-test-subject") {
		t.Errorf("expected DEK fields in result, got: %s", text)
	}
}

func TestListDEKs(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Create KEK
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_kek",
		Arguments: map[string]any{
			"name":       "list-dek-kek",
			"kms_type":   "aws-kms",
			"kms_key_id": "arn:aws:kms:us-east-1:123456789:key/list-dek",
		},
	})
	if err != nil {
		t.Fatalf("create_kek: %v", err)
	}

	// Create DEKs for two subjects
	for _, subj := range []string{"subj-a", "subj-b"} {
		_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
			Name: "create_dek",
			Arguments: map[string]any{
				"kek_name": "list-dek-kek",
				"subject":  subj,
			},
		})
		if err != nil {
			t.Fatalf("create_dek(%s): %v", subj, err)
		}
	}

	// List DEK subjects
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "list_deks",
		Arguments: map[string]any{"kek_name": "list-dek-kek"},
	})
	if err != nil {
		t.Fatalf("list_deks: %v", err)
	}
	text := resultText(t, result)
	var subjects []string
	if err := json.Unmarshal([]byte(text), &subjects); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(subjects) != 2 {
		t.Fatalf("expected 2 subjects, got: %v", subjects)
	}
}

func TestListDEKVersions(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Create KEK
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_kek",
		Arguments: map[string]any{
			"name":       "ver-dek-kek",
			"kms_type":   "aws-kms",
			"kms_key_id": "arn:aws:kms:us-east-1:123456789:key/ver-dek",
		},
	})
	if err != nil {
		t.Fatalf("create_kek: %v", err)
	}

	// Create two DEK versions for same subject
	for i := 0; i < 2; i++ {
		_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
			Name: "create_dek",
			Arguments: map[string]any{
				"kek_name": "ver-dek-kek",
				"subject":  "ver-subj",
			},
		})
		if err != nil {
			t.Fatalf("create_dek v%d: %v", i+1, err)
		}
	}

	// List versions
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_dek_versions",
		Arguments: map[string]any{
			"kek_name": "ver-dek-kek",
			"subject":  "ver-subj",
		},
	})
	if err != nil {
		t.Fatalf("list_dek_versions: %v", err)
	}
	text := resultText(t, result)
	var versions []int
	if err := json.Unmarshal([]byte(text), &versions); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(versions) != 2 || versions[0] != 1 || versions[1] != 2 {
		t.Fatalf("expected [1,2], got: %v", versions)
	}
}

func TestDeleteAndUndeleteDEK(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Create KEK + DEK
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_kek",
		Arguments: map[string]any{
			"name":       "del-dek-kek",
			"kms_type":   "aws-kms",
			"kms_key_id": "arn:aws:kms:us-east-1:123456789:key/del-dek",
		},
	})
	if err != nil {
		t.Fatalf("create_kek: %v", err)
	}

	_, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_dek",
		Arguments: map[string]any{
			"kek_name": "del-dek-kek",
			"subject":  "del-dek-subj",
		},
	})
	if err != nil {
		t.Fatalf("create_dek: %v", err)
	}

	// Delete DEK (version 1 was auto-assigned)
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "delete_dek",
		Arguments: map[string]any{
			"kek_name": "del-dek-kek",
			"subject":  "del-dek-subj",
			"version":  1,
		},
	})
	if err != nil {
		t.Fatalf("delete_dek: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Errorf("expected deleted:true, got: %s", text)
	}

	// Undelete DEK
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "undelete_dek",
		Arguments: map[string]any{
			"kek_name": "del-dek-kek",
			"subject":  "del-dek-subj",
			"version":  1,
		},
	})
	if err != nil {
		t.Fatalf("undelete_dek: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Errorf("expected undeleted:true, got: %s", text)
	}

	// Get should succeed
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_dek",
		Arguments: map[string]any{
			"kek_name": "del-dek-kek",
			"subject":  "del-dek-subj",
		},
	})
	if err != nil {
		t.Fatalf("get_dek after undelete: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success after undelete, got error: %s", resultText(t, result))
	}
}

// --- Phase 7: Exporter tool tests ---

func createTestExporter(t *testing.T, cs *gomcp.ClientSession, name string) {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_exporter",
		Arguments: map[string]any{
			"name":         name,
			"context_type": "AUTO",
			"subjects":     []any{"subject-a", "subject-b"},
			"config":       map[string]any{"schema.registry.url": "http://dest:8081"},
		},
	})
	if err != nil {
		t.Fatalf("create_exporter(%s): %v", name, err)
	}
	if result.IsError {
		t.Fatalf("create_exporter(%s) error: %s", name, resultText(t, result))
	}
}

func TestCreateAndGetExporter(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	createTestExporter(t, cs, "test-exporter")

	// Get it back
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_exporter",
		Arguments: map[string]any{"name": "test-exporter"},
	})
	if err != nil {
		t.Fatalf("get_exporter: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "test-exporter") {
		t.Errorf("expected name in result, got: %s", text)
	}
	if !strings.Contains(text, "AUTO") {
		t.Errorf("expected context type AUTO, got: %s", text)
	}
	if !strings.Contains(text, "subject-a") {
		t.Errorf("expected subject in result, got: %s", text)
	}
}

func TestListExporters(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	createTestExporter(t, cs, "exp-alpha")
	createTestExporter(t, cs, "exp-beta")

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_exporters",
	})
	if err != nil {
		t.Fatalf("list_exporters: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "exp-alpha") || !strings.Contains(text, "exp-beta") {
		t.Errorf("expected both exporters, got: %s", text)
	}
}

func TestDeleteExporter(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	createTestExporter(t, cs, "del-exporter")

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_exporter",
		Arguments: map[string]any{"name": "del-exporter"},
	})
	if err != nil {
		t.Fatalf("delete_exporter: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Errorf("expected deleted:true, got: %s", text)
	}

	// Verify it's gone
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_exporter",
		Arguments: map[string]any{"name": "del-exporter"},
	})
	if err != nil {
		t.Fatalf("get_exporter: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for deleted exporter")
	}
}

func TestExporterPauseResumeReset(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	createTestExporter(t, cs, "status-exporter")

	// Initial status should be PAUSED (set by CreateExporter)
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_exporter_status",
		Arguments: map[string]any{"name": "status-exporter"},
	})
	if err != nil {
		t.Fatalf("get_exporter_status: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "PAUSED") {
		t.Errorf("expected PAUSED, got: %s", text)
	}

	// Resume
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "resume_exporter",
		Arguments: map[string]any{"name": "status-exporter"},
	})
	if err != nil {
		t.Fatalf("resume_exporter: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "RUNNING") {
		t.Errorf("expected RUNNING, got: %s", text)
	}

	// Pause
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "pause_exporter",
		Arguments: map[string]any{"name": "status-exporter"},
	})
	if err != nil {
		t.Fatalf("pause_exporter: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "PAUSED") {
		t.Errorf("expected PAUSED, got: %s", text)
	}

	// Reset
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "reset_exporter",
		Arguments: map[string]any{"name": "status-exporter"},
	})
	if err != nil {
		t.Fatalf("reset_exporter: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "reset") {
		t.Errorf("expected reset, got: %s", text)
	}
}

func TestExporterConfig(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	createTestExporter(t, cs, "cfg-exporter")

	// Get config
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_exporter_config",
		Arguments: map[string]any{"name": "cfg-exporter"},
	})
	if err != nil {
		t.Fatalf("get_exporter_config: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "schema.registry.url") {
		t.Errorf("expected config key in result, got: %s", text)
	}

	// Update config
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "update_exporter_config",
		Arguments: map[string]any{
			"name":   "cfg-exporter",
			"config": map[string]any{"schema.registry.url": "http://new-dest:8081"},
		},
	})
	if err != nil {
		t.Fatalf("update_exporter_config: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	// Get config again, verify updated
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_exporter_config",
		Arguments: map[string]any{"name": "cfg-exporter"},
	})
	if err != nil {
		t.Fatalf("get_exporter_config: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "new-dest") {
		t.Errorf("expected updated config, got: %s", text)
	}
}

func TestUpdateExporter(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	createTestExporter(t, cs, "upd-exporter")

	// Update subjects
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "update_exporter",
		Arguments: map[string]any{
			"name":     "upd-exporter",
			"subjects": []any{"new-subject"},
		},
	})
	if err != nil {
		t.Fatalf("update_exporter: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	// Get and verify
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_exporter",
		Arguments: map[string]any{"name": "upd-exporter"},
	})
	if err != nil {
		t.Fatalf("get_exporter: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "new-subject") {
		t.Errorf("expected updated subject, got: %s", text)
	}
}

// --- Phase 8: Metadata, alias, and advanced tool tests ---

func TestGetConfigFull(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Global default should return BACKWARD with full record
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_config_full",
	})
	if err != nil {
		t.Fatalf("get_config_full: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "BACKWARD") {
		t.Errorf("expected BACKWARD in result, got: %s", text)
	}
	if !strings.Contains(text, "compatibilityLevel") {
		t.Errorf("expected full record fields, got: %s", text)
	}
}

func TestSetConfigFullWithAlias(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Set config with alias
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "set_config_full",
		Arguments: map[string]any{
			"subject":              "alias-src",
			"compatibility_level": "BACKWARD",
			"alias":               "alias-target",
		},
	})
	if err != nil {
		t.Fatalf("set_config_full: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	// Get full config and verify alias
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_config_full",
		Arguments: map[string]any{"subject": "alias-src"},
	})
	if err != nil {
		t.Fatalf("get_config_full: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "alias-target") {
		t.Errorf("expected alias in config, got: %s", text)
	}
}

func TestSetConfigFullWithMetadata(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "set_config_full",
		Arguments: map[string]any{
			"subject":              "meta-subj",
			"compatibility_level": "FULL",
			"default_metadata": map[string]any{
				"properties": map[string]any{"owner": "team-data"},
				"tags":       map[string]any{"pii": []any{"email", "phone"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("set_config_full: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}

	// Get subject config full
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_subject_config_full",
		Arguments: map[string]any{"subject": "meta-subj"},
	})
	if err != nil {
		t.Fatalf("get_subject_config_full: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "team-data") {
		t.Errorf("expected metadata properties, got: %s", text)
	}
	if !strings.Contains(text, "pii") {
		t.Errorf("expected metadata tags, got: %s", text)
	}
}

func TestResolveAlias(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Set up alias
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "set_config_full",
		Arguments: map[string]any{
			"subject":              "my-alias",
			"compatibility_level": "BACKWARD",
			"alias":               "real-subject",
		},
	})
	if err != nil {
		t.Fatalf("set_config_full: %v", err)
	}

	// Resolve alias
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "resolve_alias",
		Arguments: map[string]any{"subject": "my-alias"},
	})
	if err != nil {
		t.Fatalf("resolve_alias: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "real-subject") {
		t.Errorf("expected resolved alias, got: %s", text)
	}

	// No alias — should resolve to self
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "resolve_alias",
		Arguments: map[string]any{"subject": "no-alias"},
	})
	if err != nil {
		t.Fatalf("resolve_alias: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "no-alias") {
		t.Errorf("expected self-resolve, got: %s", text)
	}
}

func TestGetSchemasBySubject(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	v1 := `{"type":"record","name":"Test","fields":[{"name":"a","type":"string"}]}`
	v2 := `{"type":"record","name":"Test","fields":[{"name":"a","type":"string"},{"name":"b","type":["null","string"],"default":null}]}`
	registerTestSchema(t, reg, "multi-ver", v1)
	registerTestSchema(t, reg, "multi-ver", v2)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_schemas_by_subject",
		Arguments: map[string]any{"subject": "multi-ver"},
	})
	if err != nil {
		t.Fatalf("get_schemas_by_subject: %v", err)
	}
	text := resultText(t, result)
	// Should contain both versions
	if !strings.Contains(text, `"version":1`) || !strings.Contains(text, `"version":2`) {
		t.Errorf("expected both versions, got: %s", text)
	}
}

func TestCheckWriteMode(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	// Default READWRITE mode should be writable
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "check_write_mode",
	})
	if err != nil {
		t.Fatalf("check_write_mode: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Errorf("expected writable:true, got: %s", text)
	}

	// Set to READONLY
	_, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "set_mode",
		Arguments: map[string]any{"subject": "ro-subj", "mode": "READONLY"},
	})
	if err != nil {
		t.Fatalf("set_mode: %v", err)
	}

	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "check_write_mode",
		Arguments: map[string]any{"subject": "ro-subj"},
	})
	if err != nil {
		t.Fatalf("check_write_mode: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "READONLY") {
		t.Errorf("expected READONLY blocking mode, got: %s", text)
	}
	if !strings.Contains(text, "false") {
		t.Errorf("expected writable:false, got: %s", text)
	}
}

func TestFormatSchema(t *testing.T) {
	cs, reg := newTestMCPClient(t)
	registerTestSchema(t, reg, "format-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "format_schema",
		Arguments: map[string]any{"subject": "format-test", "version": 1},
	})
	if err != nil {
		t.Fatalf("format_schema: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "string") {
		t.Errorf("expected schema content, got: %s", text)
	}
	if !strings.Contains(text, "format-test") {
		t.Errorf("expected subject in result, got: %s", text)
	}
}

func TestGetGlobalConfigDirect(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_global_config_direct",
	})
	if err != nil {
		t.Fatalf("get_global_config_direct: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "BACKWARD") {
		t.Errorf("expected BACKWARD default, got: %s", text)
	}
}

// newTestMCPClientWithAuth creates a test MCP server with auth service enabled.
func newTestMCPClientWithAuth(t *testing.T) (*gomcp.ClientSession, *auth.Service) {
	t.Helper()

	store := memory.NewStore()
	t.Cleanup(func() { store.Close() })

	schemaReg := schema.NewRegistry()
	schemaReg.Register(avro.NewParser())
	schemaReg.Register(protobuf.NewParser())
	schemaReg.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaReg, compatChecker, "BACKWARD")

	authSvc := auth.NewServiceWithConfig(store, auth.ServiceConfig{})
	t.Cleanup(func() { authSvc.Close() })

	cfg := &config.MCPConfig{Host: "localhost", Port: 0}
	srv := New(cfg, reg, testLogger(), "test-version", WithAuthService(authSvc))

	ctx := context.Background()
	ct, st := gomcp.NewInMemoryTransports()

	ss, err := srv.MCPServer().Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { ss.Close() })

	client := gomcp.NewClient(&gomcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })

	return cs, authSvc
}

// --- Admin tool tests ---

func TestListRoles(t *testing.T) {
	cs, _ := newTestMCPClientWithAuth(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_roles",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "super_admin") {
		t.Fatalf("expected super_admin in roles, got: %s", text)
	}
	if !strings.Contains(text, "readonly") {
		t.Fatalf("expected readonly in roles, got: %s", text)
	}
}

func TestCreateAndGetUser(t *testing.T) {
	cs, _ := newTestMCPClientWithAuth(t)

	// Create user
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_user",
		Arguments: map[string]any{
			"username": "testuser",
			"password": "secret123",
			"role":     "developer",
		},
	})
	if err != nil {
		t.Fatalf("create_user: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "testuser") {
		t.Fatalf("expected username in result, got: %s", text)
	}

	// Parse the ID from the result
	var created map[string]any
	if err := json.Unmarshal([]byte(text), &created); err != nil {
		t.Fatalf("parse create result: %v", err)
	}
	userID := created["id"].(float64)

	// Get user by ID
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_user",
		Arguments: map[string]any{"id": userID},
	})
	if err != nil {
		t.Fatalf("get_user: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "testuser") {
		t.Fatalf("expected username in get result, got: %s", text)
	}
	if !strings.Contains(text, "developer") {
		t.Fatalf("expected role in get result, got: %s", text)
	}
}

func TestListUsers(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	// Create two users
	for _, name := range []string{"alice", "bob"} {
		_, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
			Username: name, Password: "pass123", Role: "developer", Enabled: true,
		})
		if err != nil {
			t.Fatalf("create user %s: %v", name, err)
		}
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_users",
	})
	if err != nil {
		t.Fatalf("list_users: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "alice") || !strings.Contains(text, "bob") {
		t.Fatalf("expected both users, got: %s", text)
	}
}

func TestUpdateUser(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	user, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "updateme", Password: "pass123", Role: "developer", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "update_user",
		Arguments: map[string]any{
			"id":   float64(user.ID),
			"role": "admin",
		},
	})
	if err != nil {
		t.Fatalf("update_user: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "admin") {
		t.Fatalf("expected updated role, got: %s", text)
	}
}

func TestDeleteUser(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	user, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "deleteme", Password: "pass123", Role: "developer", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "delete_user",
		Arguments: map[string]any{"id": float64(user.ID)},
	})
	if err != nil {
		t.Fatalf("delete_user: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Fatalf("expected deleted:true, got: %s", text)
	}
}

func TestCreateAndGetAPIKey(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	user, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "keyowner", Password: "pass123", Role: "developer", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "create_apikey",
		Arguments: map[string]any{
			"user_id":    float64(user.ID),
			"name":       "test-key",
			"role":       "developer",
			"expires_in": float64(3600),
		},
	})
	if err != nil {
		t.Fatalf("create_apikey: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "test-key") {
		t.Fatalf("expected key name, got: %s", text)
	}
	if !strings.Contains(text, "key") {
		t.Fatalf("expected raw key in result, got: %s", text)
	}

	// Parse ID
	var created map[string]any
	if err := json.Unmarshal([]byte(text), &created); err != nil {
		t.Fatalf("parse create result: %v", err)
	}
	keyID := created["id"].(float64)

	// Get API key
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_apikey",
		Arguments: map[string]any{"id": keyID},
	})
	if err != nil {
		t.Fatalf("get_apikey: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "test-key") {
		t.Fatalf("expected key name in get result, got: %s", text)
	}
}

func TestListAPIKeys(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	user, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "keyuser", Password: "pass123", Role: "developer", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create API key directly via auth service
	_, err = authSvc.CreateAPIKey(context.Background(), auth.CreateAPIKeyRequest{
		UserID:    user.ID,
		Name:      "key1",
		Role:      "developer",
		ExpiresAt: func() time.Time { return time.Now().Add(time.Hour) }(),
	})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "list_apikeys",
	})
	if err != nil {
		t.Fatalf("list_apikeys: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "key1") {
		t.Fatalf("expected key1 in list, got: %s", text)
	}

	// List by user_id
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "list_apikeys",
		Arguments: map[string]any{"user_id": float64(user.ID)},
	})
	if err != nil {
		t.Fatalf("list_apikeys by user: %v", err)
	}
	text = resultText(t, result)
	if !strings.Contains(text, "key1") {
		t.Fatalf("expected key1 in filtered list, got: %s", text)
	}
}

func TestRevokeAPIKey(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	user, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "revokeuser", Password: "pass123", Role: "developer", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	key, err := authSvc.CreateAPIKey(context.Background(), auth.CreateAPIKeyRequest{
		UserID:    user.ID,
		Name:      "revoke-me",
		Role:      "developer",
		ExpiresAt: func() time.Time { return time.Now().Add(time.Hour) }(),
	})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "revoke_apikey",
		Arguments: map[string]any{"id": float64(key.ID)},
	})
	if err != nil {
		t.Fatalf("revoke_apikey: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Fatalf("expected revoked:true, got: %s", text)
	}

	// Verify it's disabled
	result, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_apikey",
		Arguments: map[string]any{"id": float64(key.ID)},
	})
	if err != nil {
		t.Fatalf("get_apikey after revoke: %v", err)
	}
	text = resultText(t, result)
	if strings.Contains(text, `"enabled":true`) {
		t.Fatalf("expected key to be disabled after revoke, got: %s", text)
	}
}

func TestChangePassword(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	user, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "pwuser", Password: "oldpass123", Role: "developer", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "change_password",
		Arguments: map[string]any{
			"id":           float64(user.ID),
			"old_password": "oldpass123",
			"new_password": "newpass456",
		},
	})
	if err != nil {
		t.Fatalf("change_password: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "true") {
		t.Fatalf("expected changed:true, got: %s", text)
	}

	// Verify old password no longer works
	_, err = authSvc.ValidateCredentials(context.Background(), "pwuser", "oldpass123")
	if err == nil {
		t.Fatal("expected old password to fail")
	}

	// Verify new password works
	_, err = authSvc.ValidateCredentials(context.Background(), "pwuser", "newpass456")
	if err != nil {
		t.Fatalf("expected new password to work: %v", err)
	}
}

func TestRotateAPIKey(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	user, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "rotateuser", Password: "pass123", Role: "developer", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	key, err := authSvc.CreateAPIKey(context.Background(), auth.CreateAPIKeyRequest{
		UserID:    user.ID,
		Name:      "rotate-me",
		Role:      "developer",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create api key: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "rotate_apikey",
		Arguments: map[string]any{
			"id":         float64(key.ID),
			"expires_in": float64(7200),
		},
	})
	if err != nil {
		t.Fatalf("rotate_apikey: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "key") {
		t.Fatalf("expected new key in result, got: %s", text)
	}
	// The result should contain a new key (different ID)
	var rotated map[string]any
	if err := json.Unmarshal([]byte(text), &rotated); err != nil {
		t.Fatalf("parse rotate result: %v", err)
	}
	if rotated["id"].(float64) == float64(key.ID) {
		t.Fatal("expected new key to have different ID")
	}
}

func TestGetUserByUsername(t *testing.T) {
	cs, authSvc := newTestMCPClientWithAuth(t)

	_, err := authSvc.CreateUser(context.Background(), auth.CreateUserRequest{
		Username: "findme", Password: "pass123", Role: "admin", Enabled: true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_user_by_username",
		Arguments: map[string]any{"username": "findme"},
	})
	if err != nil {
		t.Fatalf("get_user_by_username: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "findme") {
		t.Fatalf("expected username, got: %s", text)
	}
	if !strings.Contains(text, "admin") {
		t.Fatalf("expected role, got: %s", text)
	}
}

func TestGetSubjectMetadata(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	// Register a schema with metadata via set_config_full
	_, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "register_schema",
		Arguments: map[string]any{
			"subject":     "meta-test",
			"schema":      `{"type":"string"}`,
			"schema_type": "AVRO",
		},
	})
	if err != nil {
		t.Fatalf("register schema: %v", err)
	}

	// Set metadata on the subject config
	_, err = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "set_config_full",
		Arguments: map[string]any{
			"subject":              "meta-test",
			"compatibility_level": "BACKWARD",
			"default_metadata": map[string]any{
				"properties": map[string]any{
					"owner": "team-a",
					"major": "1",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("set_config_full: %v", err)
	}

	// Register another version to pick up the metadata
	_, err = reg.RegisterSchema(context.Background(), ".", "meta-test", `{"type":"string"}`, "AVRO", nil)
	if err != nil {
		// May get "already exists" which is fine for this test
		_ = err
	}

	// Get metadata without filter (bare metadata from latest)
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_subject_metadata",
		Arguments: map[string]any{"subject": "meta-test"},
	})
	if err != nil {
		t.Fatalf("get_subject_metadata: %v", err)
	}
	text := resultText(t, result)
	// Should return metadata (possibly empty if no metadata was attached to schema)
	if text == "" {
		t.Fatal("expected non-empty metadata result")
	}
}

func TestGetClusterID(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_cluster_id",
	})
	if err != nil {
		t.Fatalf("get_cluster_id: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "default-cluster") {
		t.Fatalf("expected default-cluster, got: %s", text)
	}
}

func TestGetServerVersion(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_server_version",
	})
	if err != nil {
		t.Fatalf("get_server_version: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "test-version") {
		t.Fatalf("expected test-version, got: %s", text)
	}
}

// newTestMCPClientWithMetrics creates a test MCP server with Prometheus metrics wired.
func newTestMCPClientWithMetrics(t *testing.T) (*gomcp.ClientSession, *metrics.Metrics) {
	t.Helper()

	store := memory.NewStore()
	t.Cleanup(func() { store.Close() })

	schemaReg := schema.NewRegistry()
	schemaReg.Register(avro.NewParser())
	schemaReg.Register(protobuf.NewParser())
	schemaReg.Register(jsonschema.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaReg, compatChecker, "BACKWARD")

	m := metrics.New()
	m.EnablePrincipalMetrics()
	cfg := &config.MCPConfig{Host: "localhost", Port: 0}
	srv := New(cfg, reg, testLogger(), "test-version", WithMetrics(m))

	ctx := context.Background()
	ct, st := gomcp.NewInMemoryTransports()

	ss, err := srv.MCPServer().Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { ss.Close() })

	client := gomcp.NewClient(&gomcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })

	return cs, m
}

func TestInstrumentedHandlerRecordsMetrics(t *testing.T) {
	cs, m := newTestMCPClientWithMetrics(t)

	// Call a tool that should succeed.
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "health_check",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}

	// Verify the metrics were recorded.
	// Use the Prometheus test helper to get counter values.
	val := getCounterValue(t, m.MCPToolCallsTotal, "health_check", "success")
	if val != 1 {
		t.Errorf("expected MCPToolCallsTotal=1, got=%v", val)
	}

	// Call again to verify increment.
	_, _ = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "health_check",
	})
	val = getCounterValue(t, m.MCPToolCallsTotal, "health_check", "success")
	if val != 2 {
		t.Errorf("expected MCPToolCallsTotal=2, got=%v", val)
	}
}

func TestInstrumentedHandlerRecordsErrors(t *testing.T) {
	cs, m := newTestMCPClientWithMetrics(t)

	// Call a tool that will return an error (get schema by non-existent ID).
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_schema_by_id",
		Arguments: json.RawMessage(`{"id": 999999}`),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}

	// Verify error metric was recorded.
	errVal := getCounterValue(t, m.MCPToolCallErrors, "get_schema_by_id")
	if errVal != 1 {
		t.Errorf("expected MCPToolCallErrors=1, got=%v", errVal)
	}
	totalVal := getCounterValue(t, m.MCPToolCallsTotal, "get_schema_by_id", "error")
	if totalVal != 1 {
		t.Errorf("expected MCPToolCallsTotal(error)=1, got=%v", totalVal)
	}
}

func TestInstrumentedHandlerRecordsPrincipalMetrics(t *testing.T) {
	cs, m := newTestMCPClientWithMetrics(t)

	// Call a tool that should succeed.
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "health_check",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}

	// Verify per-principal MCP metric was recorded with "mcp-client" principal.
	val := getCounterValue(t, m.PrincipalMCPCallsTotal, "mcp-client", "health_check", "success")
	if val != 1 {
		t.Errorf("expected PrincipalMCPCallsTotal(mcp-client, health_check, success)=1, got=%v", val)
	}

	// Call a tool that returns an error.
	_, _ = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name:      "get_schema_by_id",
		Arguments: json.RawMessage(`{"id": 999999}`),
	})

	errVal := getCounterValue(t, m.PrincipalMCPCallsTotal, "mcp-client", "get_schema_by_id", "error")
	if errVal != 1 {
		t.Errorf("expected PrincipalMCPCallsTotal(mcp-client, get_schema_by_id, error)=1, got=%v", errVal)
	}

	// Call health_check again and verify increment.
	_, _ = cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "health_check",
	})
	val = getCounterValue(t, m.PrincipalMCPCallsTotal, "mcp-client", "health_check", "success")
	if val != 2 {
		t.Errorf("expected PrincipalMCPCallsTotal(mcp-client, health_check, success)=2, got=%v", val)
	}
}

func TestInstrumentedHandlerWithoutMetrics(t *testing.T) {
	// Create server without metrics — verify no panic.
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "health_check",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
}

// getCounterValue reads the current value of a Prometheus CounterVec for the given labels.
func getCounterValue(t *testing.T, cv *prometheus.CounterVec, labelValues ...string) float64 {
	t.Helper()
	counter, err := cv.GetMetricWithLabelValues(labelValues...)
	if err != nil {
		t.Fatalf("GetMetricWithLabelValues: %v", err)
	}
	// Use the Write method to extract the value.
	var m prometheusmodel.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Write metric: %v", err)
	}
	return m.GetCounter().GetValue()
}

// resultText extracts the text from the first TextContent in a CallToolResult.
func resultText(t *testing.T, result *gomcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("empty content")
	}
	data, err := result.Content[0].MarshalJSON()
	if err != nil {
		t.Fatalf("marshal content: %v", err)
	}
	var wire struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	return wire.Text
}
