package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestValidateSchemaValid(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "validate_schema",
		Arguments: map[string]any{
			"schema": `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, `"valid":true`) {
		t.Fatalf("expected valid:true, got: %s", text)
	}
	if !strings.Contains(text, "fingerprint") {
		t.Fatalf("expected fingerprint in result, got: %s", text)
	}
}

func TestValidateSchemaInvalid(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "validate_schema",
		Arguments: map[string]any{
			"schema": `{"type":"invalid"}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, `"valid":false`) {
		t.Fatalf("expected valid:false, got: %s", text)
	}
}

func TestNormalizeSchema(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "normalize_schema",
		Arguments: map[string]any{
			"schema": `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "normalized") {
		t.Fatalf("expected normalized in result, got: %s", text)
	}
	if !strings.Contains(text, "fingerprint") {
		t.Fatalf("expected fingerprint in result, got: %s", text)
	}
}

func TestValidateSubjectName(t *testing.T) {
	cs, _ := newTestMCPClient(t)

	tests := []struct {
		name     string
		subject  string
		strategy string
		valid    bool
	}{
		{"valid topic_name key", "my-topic-key", "topic_name", true},
		{"valid topic_name value", "my-topic-value", "topic_name", true},
		{"invalid topic_name", "my-topic", "topic_name", false},
		{"valid record_name", "com.example.User", "record_name", true},
		{"invalid record_name", "my-topic-key", "record_name", false},
		{"valid topic_record_name", "my-topic-com.example.User", "topic_record_name", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
				Name: "validate_subject_name",
				Arguments: map[string]any{
					"subject":  tc.subject,
					"strategy": tc.strategy,
				},
			})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			text := resultText(t, result)
			var out map[string]any
			if err := json.Unmarshal([]byte(text), &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if out["valid"] != tc.valid {
				t.Fatalf("expected valid=%v, got: %s", tc.valid, text)
			}
		})
	}
}

func TestSearchSchemas(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "search-test-1", `{"type":"record","name":"User","fields":[{"name":"email","type":"string"}]}`)
	registerTestSchema(t, reg, "search-test-2", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "search_schemas",
		Arguments: map[string]any{
			"pattern": "email",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "search-test-1") {
		t.Fatalf("expected search-test-1 in results, got: %s", text)
	}
	if strings.Contains(text, "search-test-2") {
		t.Fatalf("should not contain search-test-2, got: %s", text)
	}
}

func TestGetSchemaHistory(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "history-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_schema_history",
		Arguments: map[string]any{
			"subject": "history-test",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "history-test") {
		t.Fatalf("expected subject in result, got: %s", text)
	}
	if !strings.Contains(text, `"count":1`) {
		t.Fatalf("expected count:1, got: %s", text)
	}
}

func TestExportSchema(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "export-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "export_schema",
		Arguments: map[string]any{
			"subject": "export-test",
			"version": 1,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "export-test") {
		t.Fatalf("expected subject in result, got: %s", text)
	}
	if !strings.Contains(text, "schema_type") {
		t.Fatalf("expected schema_type in result, got: %s", text)
	}
}

func TestExportSubject(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "export-subj-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "export_subject",
		Arguments: map[string]any{
			"subject": "export-subj-test",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "export-subj-test") {
		t.Fatalf("expected subject in result, got: %s", text)
	}
	if !strings.Contains(text, `"count":1`) {
		t.Fatalf("expected count:1, got: %s", text)
	}
}

func TestGetRegistryStatistics(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "stats-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_registry_statistics",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "total_subjects") {
		t.Fatalf("expected total_subjects in result, got: %s", text)
	}
	if !strings.Contains(text, "total_versions") {
		t.Fatalf("expected total_versions in result, got: %s", text)
	}
}

func TestCountVersions(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "count-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "count_versions",
		Arguments: map[string]any{
			"subject": "count-test",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["count"] != float64(1) {
		t.Fatalf("expected count=1, got: %s", text)
	}
}

func TestCountSubjects(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "count-subj-1", `{"type":"string"}`)
	registerTestSchema(t, reg, "count-subj-2", `{"type":"int"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "count_subjects",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	count := out["count"].(float64)
	if count < 2 {
		t.Fatalf("expected count >= 2, got: %s", text)
	}
}
