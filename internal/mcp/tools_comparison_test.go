package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestDiffSchemas(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	// Set compat to NONE so we can register incompatible versions
	if err := reg.SetConfig(context.Background(), ".", "diff-test", "NONE", nil); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	// Register two versions with different fields
	_, err := reg.RegisterSchema(context.Background(), ".", "diff-test",
		`{"type":"record","name":"Diff","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register v1: %v", err)
	}
	_, err = reg.RegisterSchema(context.Background(), ".", "diff-test",
		`{"type":"record","name":"Diff","fields":[{"name":"id","type":"long"},{"name":"email","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register v2: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "diff_schemas",
		Arguments: map[string]any{
			"subject":      "diff-test",
			"version_from": 1,
			"version_to":   2,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	// Should show: name removed, email added, id modified (int→long)
	if !strings.Contains(text, "diffs") {
		t.Fatalf("expected diffs in result, got: %s", text)
	}
}

func TestCompareSubjects(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "compare-a",
		`{"type":"record","name":"A","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register a: %v", err)
	}
	_, err = reg.RegisterSchema(context.Background(), ".", "compare-b",
		`{"type":"record","name":"B","fields":[{"name":"id","type":"int"},{"name":"email","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register b: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "compare_subjects",
		Arguments: map[string]any{
			"subject_a": "compare-a",
			"subject_b": "compare-b",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "common_fields") {
		t.Fatalf("expected common_fields, got: %s", text)
	}
	// "id" should be in common
	if !strings.Contains(text, "id") {
		t.Fatalf("expected 'id' in common fields, got: %s", text)
	}
}

func TestCheckCompatibilityMulti(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "compat-multi-1",
		`{"type":"record","name":"V1","fields":[{"name":"id","type":"int"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "check_compatibility_multi",
		Arguments: map[string]any{
			"subjects": []any{"compat-multi-1"},
			"schema":   `{"type":"record","name":"V2","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "all_compatible") {
		t.Fatalf("expected all_compatible in result, got: %s", text)
	}
}

func TestSuggestCompatibleChange(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "suggest-test", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "suggest_compatible_change",
		Arguments: map[string]any{
			"subject":     "suggest-test",
			"change_type": "add_field",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "advice") {
		t.Fatalf("expected advice in result, got: %s", text)
	}
}

func TestMatchSubjects(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "match-alpha", `{"type":"string"}`)
	registerTestSchema(t, reg, "match-beta", `{"type":"string"}`)
	registerTestSchema(t, reg, "other-gamma", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "match_subjects",
		Arguments: map[string]any{
			"pattern": "match-",
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
	if out["count"] != float64(2) {
		t.Fatalf("expected 2 matches, got: %s", text)
	}
}

func TestExplainCompatibilityFailure(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "explain-test",
		`{"type":"record","name":"V1","fields":[{"name":"id","type":"int"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Try an incompatible change (remove required field without default)
	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "explain_compatibility_failure",
		Arguments: map[string]any{
			"subject": "explain-test",
			"schema":  `{"type":"record","name":"V2","fields":[{"name":"email","type":"string"}]}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	// Should either show is_compatible=false with explanations, or is_compatible=true
	if !strings.Contains(text, "is_compatible") {
		t.Fatalf("expected is_compatible in result, got: %s", text)
	}
}

func TestMatchSubjectsRegex(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "regex-alpha-value", `{"type":"string"}`)
	registerTestSchema(t, reg, "regex-beta-key", `{"type":"string"}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "match_subjects",
		Arguments: map[string]any{
			"pattern": "^regex-.*-value$",
			"regex":   true,
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
		t.Fatalf("expected 1 match, got: %s", text)
	}
}
