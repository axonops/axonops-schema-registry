package mcp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

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
