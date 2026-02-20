package rules

import (
	"strings"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

func TestValidateRuleSet_Nil(t *testing.T) {
	if err := ValidateRuleSet(nil); err != nil {
		t.Errorf("nil ruleSet should be valid: %v", err)
	}
}

func TestValidateRuleSet_Empty(t *testing.T) {
	if err := ValidateRuleSet(&storage.RuleSet{}); err != nil {
		t.Errorf("empty ruleSet should be valid: %v", err)
	}
}

func TestValidateRuleSet_ValidDomainRule(t *testing.T) {
	rs := &storage.RuleSet{
		DomainRules: []storage.Rule{
			{Name: "check", Kind: "CONDITION", Mode: "WRITE", Type: "CEL", Expr: "true"},
		},
	}
	if err := ValidateRuleSet(rs); err != nil {
		t.Errorf("valid domain rule should pass: %v", err)
	}
}

func TestValidateRuleSet_ValidDomainModes(t *testing.T) {
	for _, mode := range []string{"WRITE", "READ", "WRITEREAD"} {
		rs := &storage.RuleSet{
			DomainRules: []storage.Rule{
				{Name: "r", Kind: "CONDITION", Mode: mode},
			},
		}
		if err := ValidateRuleSet(rs); err != nil {
			t.Errorf("domain mode %s should be valid: %v", mode, err)
		}
	}
}

func TestValidateRuleSet_ValidMigrationRule(t *testing.T) {
	for _, mode := range []string{"UPGRADE", "DOWNGRADE", "UPDOWN"} {
		rs := &storage.RuleSet{
			MigrationRules: []storage.Rule{
				{Name: "migrate", Kind: "TRANSFORM", Mode: mode},
			},
		}
		if err := ValidateRuleSet(rs); err != nil {
			t.Errorf("migration mode %s should be valid: %v", mode, err)
		}
	}
}

func TestValidateRuleSet_ValidEncodingRule(t *testing.T) {
	rs := &storage.RuleSet{
		EncodingRules: []storage.Rule{
			{Name: "encrypt", Kind: "TRANSFORM", Mode: "WRITE", Type: "ENCRYPT"},
		},
	}
	if err := ValidateRuleSet(rs); err != nil {
		t.Errorf("valid encoding rule should pass: %v", err)
	}
}

func TestValidateRuleSet_MissingName(t *testing.T) {
	rs := &storage.RuleSet{
		DomainRules: []storage.Rule{
			{Kind: "CONDITION", Mode: "WRITE"},
		},
	}
	err := ValidateRuleSet(rs)
	if err == nil {
		t.Error("missing name should fail")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("error should mention name: %v", err)
	}
}

func TestValidateRuleSet_InvalidKind(t *testing.T) {
	rs := &storage.RuleSet{
		DomainRules: []storage.Rule{
			{Name: "r1", Kind: "INVALID", Mode: "WRITE"},
		},
	}
	err := ValidateRuleSet(rs)
	if err == nil {
		t.Error("invalid kind should fail")
	}
	if !strings.Contains(err.Error(), "kind must be") {
		t.Errorf("error should mention kind: %v", err)
	}
}

func TestValidateRuleSet_InvalidDomainMode(t *testing.T) {
	rs := &storage.RuleSet{
		DomainRules: []storage.Rule{
			{Name: "r1", Kind: "CONDITION", Mode: "UPGRADE"},
		},
	}
	err := ValidateRuleSet(rs)
	if err == nil {
		t.Error("UPGRADE mode in domainRules should fail")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("error should mention mode: %v", err)
	}
}

func TestValidateRuleSet_InvalidMigrationMode(t *testing.T) {
	rs := &storage.RuleSet{
		MigrationRules: []storage.Rule{
			{Name: "r1", Kind: "TRANSFORM", Mode: "WRITE"},
		},
	}
	err := ValidateRuleSet(rs)
	if err == nil {
		t.Error("WRITE mode in migrationRules should fail")
	}
}

func TestValidateRuleSet_InvalidOnSuccess(t *testing.T) {
	rs := &storage.RuleSet{
		DomainRules: []storage.Rule{
			{Name: "r1", Kind: "CONDITION", Mode: "WRITE", OnSuccess: "INVALID"},
		},
	}
	err := ValidateRuleSet(rs)
	if err == nil {
		t.Error("invalid onSuccess should fail")
	}
	if !strings.Contains(err.Error(), "onSuccess") {
		t.Errorf("error should mention onSuccess: %v", err)
	}
}

func TestValidateRuleSet_InvalidOnFailure(t *testing.T) {
	rs := &storage.RuleSet{
		DomainRules: []storage.Rule{
			{Name: "r1", Kind: "CONDITION", Mode: "WRITE", OnFailure: "CRASH"},
		},
	}
	err := ValidateRuleSet(rs)
	if err == nil {
		t.Error("invalid onFailure should fail")
	}
	if !strings.Contains(err.Error(), "onFailure") {
		t.Errorf("error should mention onFailure: %v", err)
	}
}

func TestValidateRuleSet_ValidOnSuccessOnFailure(t *testing.T) {
	for _, action := range []string{"", "NONE", "DLQ", "ERROR"} {
		rs := &storage.RuleSet{
			DomainRules: []storage.Rule{
				{Name: "r1", Kind: "CONDITION", Mode: "WRITE", OnSuccess: action, OnFailure: action},
			},
		}
		if err := ValidateRuleSet(rs); err != nil {
			t.Errorf("action %q should be valid: %v", action, err)
		}
	}
}

func TestValidateRuleSet_AllRuleCategories(t *testing.T) {
	rs := &storage.RuleSet{
		DomainRules: []storage.Rule{
			{Name: "d1", Kind: "CONDITION", Mode: "WRITEREAD"},
		},
		MigrationRules: []storage.Rule{
			{Name: "m1", Kind: "TRANSFORM", Mode: "UPDOWN"},
		},
		EncodingRules: []storage.Rule{
			{Name: "e1", Kind: "TRANSFORM", Mode: "READ"},
		},
	}
	if err := ValidateRuleSet(rs); err != nil {
		t.Errorf("valid combined ruleSet should pass: %v", err)
	}
}

func TestValidateRuleSet_TransformKind(t *testing.T) {
	rs := &storage.RuleSet{
		DomainRules: []storage.Rule{
			{Name: "transform", Kind: "TRANSFORM", Mode: "WRITE"},
		},
	}
	if err := ValidateRuleSet(rs); err != nil {
		t.Errorf("TRANSFORM kind should be valid: %v", err)
	}
}
