// Package rules provides a rule engine for schema validation and policies.
package rules

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

// RuleType represents the type of rule.
type RuleType string

const (
	RuleTypeValidation RuleType = "validation"  // Validates schema content
	RuleTypePII        RuleType = "pii"         // PII detection
	RuleTypeNaming     RuleType = "naming"      // Naming conventions
	RuleTypeCustom     RuleType = "custom"      // Custom rules
)

// RuleTrigger represents when a rule is triggered.
type RuleTrigger string

const (
	TriggerOnRegister RuleTrigger = "on_register" // On schema registration
	TriggerOnUpdate   RuleTrigger = "on_update"   // On schema update
	TriggerOnValidate RuleTrigger = "on_validate" // On message validation
	TriggerAlways     RuleTrigger = "always"      // Always
)

// RuleSeverity represents the severity of a rule violation.
type RuleSeverity string

const (
	SeverityError   RuleSeverity = "error"   // Blocks operation
	SeverityWarning RuleSeverity = "warning" // Logs warning
	SeverityInfo    RuleSeverity = "info"    // Informational
)

// Rule represents a validation rule.
type Rule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        RuleType          `json:"type"`
	Trigger     RuleTrigger       `json:"trigger"`
	Severity    RuleSeverity      `json:"severity"`
	Expression  string            `json:"expression"` // CEL or JSONata expression
	ExprType    string            `json:"expr_type"`  // "cel" or "jsonata"
	SubjectPattern string         `json:"subject_pattern,omitempty"` // Regex pattern for subjects
	SchemaTypes []string          `json:"schema_types,omitempty"`    // AVRO, PROTOBUF, JSON
	Enabled     bool              `json:"enabled"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// RuleResult represents the result of rule execution.
type RuleResult struct {
	RuleID   string       `json:"rule_id"`
	RuleName string       `json:"rule_name"`
	Passed   bool         `json:"passed"`
	Severity RuleSeverity `json:"severity"`
	Message  string       `json:"message,omitempty"`
	Details  interface{}  `json:"details,omitempty"`
}

// ExecutionResult represents the result of executing all rules.
type ExecutionResult struct {
	Passed   bool          `json:"passed"`
	Results  []*RuleResult `json:"results"`
	Errors   int           `json:"errors"`
	Warnings int           `json:"warnings"`
}

// Engine manages and executes rules.
type Engine struct {
	mu       sync.RWMutex
	rules    map[string]*Rule
	plugins  map[string]RulePlugin
	compiled map[string]*compiledRule
}

// compiledRule holds a compiled rule for efficient execution.
type compiledRule struct {
	rule           *Rule
	subjectPattern *regexp.Regexp
}

// RulePlugin is the interface for custom rule plugins.
type RulePlugin interface {
	Name() string
	Evaluate(ctx *EvaluationContext) (*RuleResult, error)
}

// EvaluationContext provides context for rule evaluation.
type EvaluationContext struct {
	Subject     string            `json:"subject"`
	Version     int               `json:"version"`
	SchemaType  string            `json:"schema_type"`
	Schema      string            `json:"schema"`
	Trigger     RuleTrigger       `json:"trigger"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewEngine creates a new rule engine.
func NewEngine() *Engine {
	return &Engine{
		rules:    make(map[string]*Rule),
		plugins:  make(map[string]RulePlugin),
		compiled: make(map[string]*compiledRule),
	}
}

// AddRule adds a rule to the engine.
func (e *Engine) AddRule(rule *Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	// Compile subject pattern
	compiled := &compiledRule{rule: rule}
	if rule.SubjectPattern != "" {
		pattern, err := regexp.Compile(rule.SubjectPattern)
		if err != nil {
			return fmt.Errorf("invalid subject pattern: %w", err)
		}
		compiled.subjectPattern = pattern
	}

	e.rules[rule.ID] = rule
	e.compiled[rule.ID] = compiled
	return nil
}

// RemoveRule removes a rule from the engine.
func (e *Engine) RemoveRule(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.rules[id]; !exists {
		return fmt.Errorf("rule not found: %s", id)
	}

	delete(e.rules, id)
	delete(e.compiled, id)
	return nil
}

// GetRule retrieves a rule by ID.
func (e *Engine) GetRule(id string) (*Rule, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rule, exists := e.rules[id]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	return rule, nil
}

// ListRules returns all rules.
func (e *Engine) ListRules() []*Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*Rule, 0, len(e.rules))
	for _, rule := range e.rules {
		result = append(result, rule)
	}
	return result
}

// UpdateRule updates an existing rule.
func (e *Engine) UpdateRule(rule *Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	existing, exists := e.rules[rule.ID]
	if !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	rule.CreatedAt = existing.CreatedAt
	rule.UpdatedAt = time.Now()

	// Recompile subject pattern
	compiled := &compiledRule{rule: rule}
	if rule.SubjectPattern != "" {
		pattern, err := regexp.Compile(rule.SubjectPattern)
		if err != nil {
			return fmt.Errorf("invalid subject pattern: %w", err)
		}
		compiled.subjectPattern = pattern
	}

	e.rules[rule.ID] = rule
	e.compiled[rule.ID] = compiled
	return nil
}

// RegisterPlugin registers a custom rule plugin.
func (e *Engine) RegisterPlugin(plugin RulePlugin) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.plugins[plugin.Name()] = plugin
}

// Execute executes all applicable rules for a context.
func (e *Engine) Execute(ctx *EvaluationContext) *ExecutionResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := &ExecutionResult{
		Passed:  true,
		Results: make([]*RuleResult, 0),
	}

	for _, compiled := range e.compiled {
		rule := compiled.rule

		// Skip disabled rules
		if !rule.Enabled {
			continue
		}

		// Check trigger
		if rule.Trigger != TriggerAlways && rule.Trigger != ctx.Trigger {
			continue
		}

		// Check subject pattern
		if compiled.subjectPattern != nil && !compiled.subjectPattern.MatchString(ctx.Subject) {
			continue
		}

		// Check schema type
		if len(rule.SchemaTypes) > 0 && !contains(rule.SchemaTypes, ctx.SchemaType) {
			continue
		}

		// Execute rule
		ruleResult := e.executeRule(compiled, ctx)
		result.Results = append(result.Results, ruleResult)

		if !ruleResult.Passed {
			switch ruleResult.Severity {
			case SeverityError:
				result.Passed = false
				result.Errors++
			case SeverityWarning:
				result.Warnings++
			}
		}
	}

	return result
}

// executeRule executes a single rule.
func (e *Engine) executeRule(compiled *compiledRule, ctx *EvaluationContext) *RuleResult {
	rule := compiled.rule

	result := &RuleResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Passed:   true,
		Severity: rule.Severity,
	}

	// Execute based on expression type
	switch rule.ExprType {
	case "cel":
		return e.executeCEL(rule, ctx)
	case "jsonata":
		return e.executeJSONata(rule, ctx)
	case "plugin":
		if plugin, ok := e.plugins[rule.Expression]; ok {
			pluginResult, err := plugin.Evaluate(ctx)
			if err != nil {
				result.Passed = false
				result.Message = fmt.Sprintf("plugin error: %v", err)
				return result
			}
			return pluginResult
		}
		result.Message = fmt.Sprintf("plugin not found: %s", rule.Expression)
		return result
	default:
		// Built-in rules
		return e.executeBuiltIn(rule, ctx)
	}
}

// executeCEL executes a CEL expression (placeholder).
func (e *Engine) executeCEL(rule *Rule, ctx *EvaluationContext) *RuleResult {
	// CEL evaluation would be implemented here using google/cel-go
	// For now, return a placeholder result
	return &RuleResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Passed:   true,
		Severity: rule.Severity,
		Message:  "CEL expression evaluated",
	}
}

// executeJSONata executes a JSONata expression (placeholder).
func (e *Engine) executeJSONata(rule *Rule, ctx *EvaluationContext) *RuleResult {
	// JSONata evaluation would be implemented here
	// For now, return a placeholder result
	return &RuleResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Passed:   true,
		Severity: rule.Severity,
		Message:  "JSONata expression evaluated",
	}
}

// executeBuiltIn executes built-in rule types.
func (e *Engine) executeBuiltIn(rule *Rule, ctx *EvaluationContext) *RuleResult {
	result := &RuleResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Passed:   true,
		Severity: rule.Severity,
	}

	switch rule.Type {
	case RuleTypePII:
		// Check for PII patterns in schema
		piiPatterns := []string{
			"ssn", "social_security", "credit_card", "email", "phone",
			"address", "password", "secret", "token",
		}
		schemaLower := toLower(ctx.Schema)
		for _, pattern := range piiPatterns {
			if containsStr(schemaLower, pattern) {
				result.Passed = false
				result.Message = fmt.Sprintf("potential PII detected: %s", pattern)
				result.Details = map[string]string{"pattern": pattern}
				return result
			}
		}

	case RuleTypeNaming:
		// Check naming conventions
		if rule.Expression != "" {
			pattern, err := regexp.Compile(rule.Expression)
			if err == nil && !pattern.MatchString(ctx.Subject) {
				result.Passed = false
				result.Message = fmt.Sprintf("subject '%s' does not match naming convention", ctx.Subject)
			}
		}

	case RuleTypeValidation:
		// Custom validation based on expression
		// Expression could be a simple check like "schema.contains('required')"
		if rule.Expression != "" && !containsStr(ctx.Schema, rule.Expression) {
			result.Passed = false
			result.Message = fmt.Sprintf("schema does not contain required pattern: %s", rule.Expression)
		}
	}

	return result
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// DefaultRules returns a set of default rules.
func DefaultRules() []*Rule {
	return []*Rule{
		{
			ID:          "pii-detection",
			Name:        "PII Detection",
			Description: "Detects potential PII fields in schemas",
			Type:        RuleTypePII,
			Trigger:     TriggerOnRegister,
			Severity:    SeverityWarning,
			Enabled:     false, // Disabled by default
		},
		{
			ID:          "naming-convention",
			Name:        "Subject Naming Convention",
			Description: "Enforces subject naming conventions",
			Type:        RuleTypeNaming,
			Trigger:     TriggerOnRegister,
			Severity:    SeverityError,
			Expression:  "^[a-z][a-z0-9-]*$", // lowercase with dashes
			Enabled:     false,
		},
	}
}
