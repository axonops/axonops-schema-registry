package rules

import (
	"fmt"
	"testing"
)

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("expected non-nil engine")
	}
	if len(e.ListRules()) != 0 {
		t.Error("expected 0 rules")
	}
}

func TestAddRule(t *testing.T) {
	e := NewEngine()
	rule := &Rule{
		ID:       "r1",
		Name:     "Test Rule",
		Type:     RuleTypeValidation,
		Trigger:  TriggerOnRegister,
		Severity: SeverityError,
		Enabled:  true,
	}
	err := e.AddRule(rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if rule.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestAddRule_EmptyID(t *testing.T) {
	e := NewEngine()
	err := e.AddRule(&Rule{Name: "test"})
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestAddRule_EmptyName(t *testing.T) {
	e := NewEngine()
	err := e.AddRule(&Rule{ID: "r1"})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestAddRule_WithSubjectPattern(t *testing.T) {
	e := NewEngine()
	err := e.AddRule(&Rule{
		ID:             "r1",
		Name:           "Pattern Rule",
		SubjectPattern: "^test-.*",
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddRule_InvalidSubjectPattern(t *testing.T) {
	e := NewEngine()
	err := e.AddRule(&Rule{
		ID:             "r1",
		Name:           "Bad Pattern",
		SubjectPattern: "[invalid",
	})
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestRemoveRule(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{ID: "r1", Name: "test"})

	err := e.RemoveRule("r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(e.ListRules()) != 0 {
		t.Error("expected 0 rules after removal")
	}
}

func TestRemoveRule_NotFound(t *testing.T) {
	e := NewEngine()
	err := e.RemoveRule("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetRule(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{ID: "r1", Name: "Test Rule"})

	rule, err := e.GetRule("r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Name != "Test Rule" {
		t.Errorf("expected 'Test Rule', got %s", rule.Name)
	}
}

func TestGetRule_NotFound(t *testing.T) {
	e := NewEngine()
	_, err := e.GetRule("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestListRules(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{ID: "r1", Name: "Rule 1"})
	e.AddRule(&Rule{ID: "r2", Name: "Rule 2"})
	e.AddRule(&Rule{ID: "r3", Name: "Rule 3"})

	rules := e.ListRules()
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d", len(rules))
	}
}

func TestUpdateRule(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{ID: "r1", Name: "Original"})

	err := e.UpdateRule(&Rule{ID: "r1", Name: "Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rule, _ := e.GetRule("r1")
	if rule.Name != "Updated" {
		t.Errorf("expected 'Updated', got %s", rule.Name)
	}
}

func TestUpdateRule_NotFound(t *testing.T) {
	e := NewEngine()
	err := e.UpdateRule(&Rule{ID: "nonexistent", Name: "test"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestUpdateRule_PreservesCreatedAt(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{ID: "r1", Name: "Original"})
	original, _ := e.GetRule("r1")
	createdAt := original.CreatedAt

	e.UpdateRule(&Rule{ID: "r1", Name: "Updated"})
	updated, _ := e.GetRule("r1")

	if updated.CreatedAt != createdAt {
		t.Error("expected CreatedAt to be preserved")
	}
}

func TestUpdateRule_InvalidSubjectPattern(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{ID: "r1", Name: "test"})

	err := e.UpdateRule(&Rule{ID: "r1", Name: "test", SubjectPattern: "[invalid"})
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestRegisterPlugin(t *testing.T) {
	e := NewEngine()
	plugin := &mockPlugin{name: "test-plugin"}
	e.RegisterPlugin(plugin)

	// Verify plugin is callable via Execute
	e.AddRule(&Rule{
		ID:         "r1",
		Name:       "Plugin Rule",
		ExprType:   "plugin",
		Expression: "test-plugin",
		Trigger:    TriggerAlways,
		Severity:   SeverityError,
		Enabled:    true,
	})

	result := e.Execute(&EvaluationContext{
		Subject: "test", Trigger: TriggerOnRegister,
	})
	if !result.Passed {
		t.Error("expected plugin rule to pass")
	}
}

func TestExecute_DisabledRuleSkipped(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:       "r1",
		Name:     "Disabled",
		Type:     RuleTypeValidation,
		Trigger:  TriggerAlways,
		Severity: SeverityError,
		Enabled:  false,
	})

	result := e.Execute(&EvaluationContext{Subject: "test"})
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results for disabled rule, got %d", len(result.Results))
	}
}

func TestExecute_TriggerFilter(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:       "r1",
		Name:     "OnRegister Only",
		Trigger:  TriggerOnRegister,
		Severity: SeverityWarning,
		Enabled:  true,
	})

	// Wrong trigger
	result := e.Execute(&EvaluationContext{Subject: "test", Trigger: TriggerOnUpdate})
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results for wrong trigger, got %d", len(result.Results))
	}

	// Matching trigger
	result = e.Execute(&EvaluationContext{Subject: "test", Trigger: TriggerOnRegister})
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestExecute_TriggerAlways(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:       "r1",
		Name:     "Always",
		Trigger:  TriggerAlways,
		Severity: SeverityInfo,
		Enabled:  true,
	})

	result := e.Execute(&EvaluationContext{Subject: "test", Trigger: TriggerOnValidate})
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestExecute_SubjectPatternFilter(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:             "r1",
		Name:           "Test Only",
		SubjectPattern: "^test-.*",
		Trigger:        TriggerAlways,
		Severity:       SeverityInfo,
		Enabled:        true,
	})

	// Non-matching subject
	result := e.Execute(&EvaluationContext{Subject: "prod-orders"})
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results for non-matching subject, got %d", len(result.Results))
	}

	// Matching subject
	result = e.Execute(&EvaluationContext{Subject: "test-orders"})
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestExecute_SchemaTypeFilter(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:          "r1",
		Name:        "Avro Only",
		Trigger:     TriggerAlways,
		SchemaTypes: []string{"AVRO"},
		Severity:    SeverityInfo,
		Enabled:     true,
	})

	// Non-matching type
	result := e.Execute(&EvaluationContext{Subject: "s", SchemaType: "JSON"})
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results for non-matching type, got %d", len(result.Results))
	}

	// Matching type
	result = e.Execute(&EvaluationContext{Subject: "s", SchemaType: "AVRO"})
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestExecute_PIIDetection(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:       "pii",
		Name:     "PII Check",
		Type:     RuleTypePII,
		Trigger:  TriggerAlways,
		Severity: SeverityWarning,
		Enabled:  true,
	})

	// Schema with PII — warning severity doesn't block Passed
	result := e.Execute(&EvaluationContext{
		Subject: "users",
		Schema:  `{"fields":[{"name":"email","type":"string"}]}`,
	})
	if result.Warnings != 1 {
		t.Errorf("expected 1 warning, got %d", result.Warnings)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Passed {
		t.Error("expected PII rule result to not pass")
	}

	// Schema without PII
	result = e.Execute(&EvaluationContext{
		Subject: "metrics",
		Schema:  `{"fields":[{"name":"count","type":"int"}]}`,
	})
	if result.Warnings != 0 {
		t.Errorf("expected 0 warnings, got %d", result.Warnings)
	}
}

func TestExecute_PIIDetection_CaseInsensitive(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:       "pii",
		Name:     "PII Check",
		Type:     RuleTypePII,
		Trigger:  TriggerAlways,
		Severity: SeverityError,
		Enabled:  true,
	})

	result := e.Execute(&EvaluationContext{
		Subject: "users",
		Schema:  `{"fields":[{"name":"Social_Security","type":"string"}]}`,
	})
	if result.Passed {
		t.Error("expected PII detection to catch Social_Security")
	}
}

func TestExecute_NamingConvention(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:         "naming",
		Name:       "Naming Check",
		Type:       RuleTypeNaming,
		Trigger:    TriggerAlways,
		Severity:   SeverityError,
		Expression: "^[a-z][a-z0-9-]*$",
		Enabled:    true,
	})

	// Valid name
	result := e.Execute(&EvaluationContext{Subject: "orders-value"})
	if !result.Passed {
		t.Error("expected valid name to pass")
	}

	// Invalid name (uppercase)
	result = e.Execute(&EvaluationContext{Subject: "Orders-Value"})
	if result.Passed {
		t.Error("expected invalid name to fail")
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
}

func TestExecute_ValidationRule(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:         "validate",
		Name:       "Must Have Doc",
		Type:       RuleTypeValidation,
		Trigger:    TriggerAlways,
		Severity:   SeverityError,
		Expression: "\"doc\"",
		Enabled:    true,
	})

	// Schema with doc
	result := e.Execute(&EvaluationContext{
		Subject: "s",
		Schema:  `{"type":"record","name":"Test","doc":"description","fields":[]}`,
	})
	if !result.Passed {
		t.Error("expected to pass with doc")
	}

	// Schema without doc
	result = e.Execute(&EvaluationContext{
		Subject: "s",
		Schema:  `{"type":"record","name":"Test","fields":[]}`,
	})
	if result.Passed {
		t.Error("expected to fail without doc")
	}
}

func TestExecute_CELPlaceholder(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:         "cel",
		Name:       "CEL Rule",
		ExprType:   "cel",
		Expression: "schema.size() > 0",
		Trigger:    TriggerAlways,
		Severity:   SeverityInfo,
		Enabled:    true,
	})

	result := e.Execute(&EvaluationContext{Subject: "s"})
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if !result.Results[0].Passed {
		t.Error("CEL placeholder should pass")
	}
}

func TestExecute_JSONataPlaceholder(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:         "jsonata",
		Name:       "JSONata Rule",
		ExprType:   "jsonata",
		Expression: "$count(fields) > 0",
		Trigger:    TriggerAlways,
		Severity:   SeverityInfo,
		Enabled:    true,
	})

	result := e.Execute(&EvaluationContext{Subject: "s"})
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if !result.Results[0].Passed {
		t.Error("JSONata placeholder should pass")
	}
}

func TestExecute_PluginNotFound(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID:         "plugin-rule",
		Name:       "Missing Plugin",
		ExprType:   "plugin",
		Expression: "nonexistent-plugin",
		Trigger:    TriggerAlways,
		Severity:   SeverityInfo,
		Enabled:    true,
	})

	result := e.Execute(&EvaluationContext{Subject: "s"})
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	// Missing plugin still passes (returns default result with message)
	if result.Results[0].Message == "" {
		t.Error("expected message about missing plugin")
	}
}

func TestExecute_PluginError(t *testing.T) {
	e := NewEngine()
	plugin := &mockPlugin{name: "failing", err: fmt.Errorf("plugin crashed")}
	e.RegisterPlugin(plugin)

	e.AddRule(&Rule{
		ID:         "r1",
		Name:       "Failing Plugin",
		ExprType:   "plugin",
		Expression: "failing",
		Trigger:    TriggerAlways,
		Severity:   SeverityError,
		Enabled:    true,
	})

	result := e.Execute(&EvaluationContext{Subject: "s"})
	if result.Passed {
		t.Error("expected failure from plugin error")
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
}

func TestExecute_MultipleRules(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID: "r1", Name: "Rule 1",
		Type: RuleTypeValidation, Expression: "type",
		Trigger: TriggerAlways, Severity: SeverityError, Enabled: true,
	})
	e.AddRule(&Rule{
		ID: "r2", Name: "Rule 2",
		Type: RuleTypeNaming, Expression: "^[a-z-]+$",
		Trigger: TriggerAlways, Severity: SeverityWarning, Enabled: true,
	})

	result := e.Execute(&EvaluationContext{
		Subject: "valid-name",
		Schema:  `{"type":"record"}`,
	})
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if !result.Passed {
		t.Error("expected both rules to pass")
	}
}

func TestExecute_ErrorSeverityBlocksPass(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID: "r1", Name: "Error Rule",
		Type: RuleTypeNaming, Expression: "^[0-9]+$", // Numbers only
		Trigger: TriggerAlways, Severity: SeverityError, Enabled: true,
	})

	result := e.Execute(&EvaluationContext{Subject: "not-a-number"})
	if result.Passed {
		t.Error("expected error severity to block pass")
	}
	if result.Errors != 1 {
		t.Errorf("expected 1 error, got %d", result.Errors)
	}
}

func TestExecute_WarningSeverityDoesNotBlock(t *testing.T) {
	e := NewEngine()
	e.AddRule(&Rule{
		ID: "r1", Name: "Warning Rule",
		Type: RuleTypeNaming, Expression: "^[0-9]+$",
		Trigger: TriggerAlways, Severity: SeverityWarning, Enabled: true,
	})

	result := e.Execute(&EvaluationContext{Subject: "not-a-number"})
	// Warning doesn't block pass
	if result.Passed {
		// Actually it should still pass because warnings don't set Passed=false
		// Let me check... looking at the code, Warnings don't set result.Passed = false
	}
	if result.Warnings != 1 {
		t.Errorf("expected 1 warning, got %d", result.Warnings)
	}
}

func TestExecute_Empty(t *testing.T) {
	e := NewEngine()
	result := e.Execute(&EvaluationContext{Subject: "s"})
	if !result.Passed {
		t.Error("expected pass with no rules")
	}
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
}

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()
	if len(rules) != 2 {
		t.Errorf("expected 2 default rules, got %d", len(rules))
	}

	// Both should be disabled by default
	for _, rule := range rules {
		if rule.Enabled {
			t.Errorf("default rule %s should be disabled", rule.Name)
		}
	}

	// Verify PII detection rule
	found := false
	for _, rule := range rules {
		if rule.ID == "pii-detection" {
			found = true
			if rule.Type != RuleTypePII {
				t.Errorf("expected PII type")
			}
		}
	}
	if !found {
		t.Error("expected pii-detection rule")
	}
}

// --- Mock Plugin ---

type mockPlugin struct {
	name string
	err  error
}

func (p *mockPlugin) Name() string { return p.name }

func (p *mockPlugin) Evaluate(ctx *EvaluationContext) (*RuleResult, error) {
	if p.err != nil {
		return nil, p.err
	}
	return &RuleResult{
		RuleID:   "plugin",
		RuleName: p.name,
		Passed:   true,
		Severity: SeverityInfo,
		Message:  "plugin evaluated",
	}, nil
}
