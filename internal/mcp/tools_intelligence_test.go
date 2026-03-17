package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestFindSchemasByField(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "field-search-1",
		`{"type":"record","name":"User","fields":[{"name":"email","type":"string"},{"name":"id","type":"int"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err = reg.RegisterSchema(context.Background(), ".", "field-search-2",
		`{"type":"record","name":"Order","fields":[{"name":"order_id","type":"int"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "find_schemas_by_field",
		Arguments: map[string]any{
			"field": "email",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "field-search-1") {
		t.Fatalf("expected field-search-1 in results, got: %s", text)
	}
	if strings.Contains(text, "field-search-2") {
		t.Fatalf("should not contain field-search-2, got: %s", text)
	}
}

func TestFindSchemasByFieldFuzzy(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "fuzzy-search-1",
		`{"type":"record","name":"User","fields":[{"name":"email_address","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "find_schemas_by_field",
		Arguments: map[string]any{
			"field":      "email_addr",
			"match_type": "fuzzy",
			"threshold":  0.6,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "fuzzy-search-1") {
		t.Fatalf("expected fuzzy match, got: %s", text)
	}
}

func TestFindSchemasByType(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "type-search-1",
		`{"type":"record","name":"User","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "find_schemas_by_type",
		Arguments: map[string]any{
			"type_pattern": "int",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "type-search-1") {
		t.Fatalf("expected type-search-1 in results, got: %s", text)
	}
}

func TestFindSimilarSchemas(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "similar-1",
		`{"type":"record","name":"A","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"email","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err = reg.RegisterSchema(context.Background(), ".", "similar-2",
		`{"type":"record","name":"B","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"},{"name":"phone","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "find_similar_schemas",
		Arguments: map[string]any{
			"subject":   "similar-1",
			"threshold": 0.3,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "similar-2") {
		t.Fatalf("expected similar-2 in matches, got: %s", text)
	}
}

func TestScoreSchemaQuality(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "quality-test",
		`{"type":"record","name":"User","namespace":"com.example","doc":"A user record","fields":[{"name":"id","type":"int","doc":"User ID"},{"name":"name","type":"string","doc":"User name"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "score_schema_quality",
		Arguments: map[string]any{
			"subject": "quality-test",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "overall_score") {
		t.Fatalf("expected overall_score, got: %s", text)
	}
	if !strings.Contains(text, "grade") {
		t.Fatalf("expected grade, got: %s", text)
	}
}

func TestCheckFieldConsistency(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "consist-1",
		`{"type":"record","name":"A","fields":[{"name":"id","type":"int"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err = reg.RegisterSchema(context.Background(), ".", "consist-2",
		`{"type":"record","name":"B","fields":[{"name":"id","type":"int"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "check_field_consistency",
		Arguments: map[string]any{
			"field": "id",
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
	if out["consistent"] != true {
		t.Fatalf("expected consistent=true, got: %s", text)
	}
}

func TestGetSchemaComplexity(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "complexity-test",
		`{"type":"record","name":"Complex","fields":[{"name":"id","type":"int"},{"name":"address","type":{"type":"record","name":"Address","fields":[{"name":"street","type":"string"},{"name":"city","type":"string"}]}}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "get_schema_complexity",
		Arguments: map[string]any{
			"subject": "complexity-test",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "field_count") {
		t.Fatalf("expected field_count, got: %s", text)
	}
	if !strings.Contains(text, `"grade"`) {
		t.Fatalf("expected grade, got: %s", text)
	}
}

func TestDetectSchemaPatterns(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	registerTestSchema(t, reg, "detect-users-value", `{"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}`)
	registerTestSchema(t, reg, "detect-orders-value", `{"type":"record","name":"Order","fields":[{"name":"id","type":"int"}]}`)

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "detect_schema_patterns",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "total_subjects") {
		t.Fatalf("expected total_subjects, got: %s", text)
	}
	if !strings.Contains(text, "schema_types") {
		t.Fatalf("expected schema_types, got: %s", text)
	}
}

func TestSuggestSchemaEvolution(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "evolve-test",
		`{"type":"record","name":"User","fields":[{"name":"id","type":"int"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "suggest_schema_evolution",
		Arguments: map[string]any{
			"subject":     "evolve-test",
			"change_type": "add_field",
			"field_name":  "email",
			"field_type":  "string",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "snippet") {
		t.Fatalf("expected snippet in result, got: %s", text)
	}
	if !strings.Contains(text, "email") {
		t.Fatalf("expected email in result, got: %s", text)
	}
}

func TestPlanMigrationPath(t *testing.T) {
	cs, reg := newTestMCPClient(t)

	_, err := reg.RegisterSchema(context.Background(), ".", "migrate-test",
		`{"type":"record","name":"V1","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`,
		storage.SchemaTypeAvro, nil)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
		Name: "plan_migration_path",
		Arguments: map[string]any{
			"subject":       "migrate-test",
			"target_schema": `{"type":"record","name":"V2","fields":[{"name":"id","type":"long"},{"name":"email","type":"string"}]}`,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "steps") {
		t.Fatalf("expected steps in result, got: %s", text)
	}
	// Should have steps: add email, change id type, remove name
	if !strings.Contains(text, "total_steps") {
		t.Fatalf("expected total_steps in result, got: %s", text)
	}
}
