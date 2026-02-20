package rules

import (
	"fmt"
	"strings"

	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// Valid rule kinds (Confluent-compatible).
var validKinds = map[string]bool{
	"CONDITION": true,
	"TRANSFORM": true,
}

// Valid modes by rule category.
var validDomainModes = map[string]bool{
	"WRITE": true, "READ": true, "WRITEREAD": true,
}
var validMigrationModes = map[string]bool{
	"UPGRADE": true, "DOWNGRADE": true, "UPDOWN": true,
}
var validEncodingModes = map[string]bool{
	"WRITE": true, "READ": true, "WRITEREAD": true,
}

// Valid onSuccess/onFailure actions.
var validActions = map[string]bool{
	"": true, "NONE": true, "DLQ": true, "ERROR": true,
}

// ValidateRuleSet validates a RuleSet for well-formedness.
// Returns nil if valid, or an error describing what is wrong.
func ValidateRuleSet(rs *storage.RuleSet) error {
	if rs == nil {
		return nil
	}

	for i, r := range rs.DomainRules {
		if err := validateRule(r, "domainRules", i, validDomainModes); err != nil {
			return err
		}
	}
	for i, r := range rs.MigrationRules {
		if err := validateRule(r, "migrationRules", i, validMigrationModes); err != nil {
			return err
		}
	}
	for i, r := range rs.EncodingRules {
		if err := validateRule(r, "encodingRules", i, validEncodingModes); err != nil {
			return err
		}
	}
	return nil
}

func validateRule(r storage.Rule, category string, index int, validModes map[string]bool) error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("invalid rule: %s[%d]: name is required", category, index)
	}

	if !validKinds[r.Kind] {
		return fmt.Errorf("invalid rule '%s': kind must be CONDITION or TRANSFORM, got '%s'", r.Name, r.Kind)
	}

	if !validModes[r.Mode] {
		modes := make([]string, 0, len(validModes))
		for m := range validModes {
			modes = append(modes, m)
		}
		return fmt.Errorf("invalid rule '%s': invalid mode '%s' for %s", r.Name, r.Mode, category)
	}

	if !validActions[r.OnSuccess] {
		return fmt.Errorf("invalid rule '%s': onSuccess must be NONE, DLQ, or ERROR, got '%s'", r.Name, r.OnSuccess)
	}
	if !validActions[r.OnFailure] {
		return fmt.Errorf("invalid rule '%s': onFailure must be NONE, DLQ, or ERROR, got '%s'", r.Name, r.OnFailure)
	}

	return nil
}
