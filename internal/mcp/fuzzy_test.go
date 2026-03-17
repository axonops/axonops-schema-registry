package mcp

import (
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		dist int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
	}
	for _, tc := range tests {
		t.Run(tc.a+"_"+tc.b, func(t *testing.T) {
			got := LevenshteinDistance(tc.a, tc.b)
			if got != tc.dist {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.dist)
			}
		})
	}
}

func TestFuzzyScore(t *testing.T) {
	if s := FuzzyScore("hello", "hello"); s != 1.0 {
		t.Errorf("exact match should be 1.0, got %f", s)
	}
	if s := FuzzyScore("hello", "world"); s >= 0.5 {
		t.Errorf("very different strings should have low score, got %f", s)
	}
	if s := FuzzyScore("email", "email_address"); s < 0.3 {
		t.Errorf("related strings should have moderate score, got %f", s)
	}
	if s := FuzzyScore("", ""); s != 1.0 {
		t.Errorf("two empty strings should be 1.0, got %f", s)
	}
}

func TestNamingVariants(t *testing.T) {
	variants := NamingVariants("user_name")
	varMap := make(map[string]bool)
	for _, v := range variants {
		varMap[v] = true
	}

	expected := []string{"user_name", "userName", "UserName", "user-name"}
	for _, e := range expected {
		if !varMap[e] {
			t.Errorf("expected variant %q in %v", e, variants)
		}
	}
}

func TestNamingVariantsCamelCase(t *testing.T) {
	variants := NamingVariants("firstName")
	varMap := make(map[string]bool)
	for _, v := range variants {
		varMap[v] = true
	}

	if !varMap["first_name"] {
		t.Errorf("expected 'first_name' in variants %v", variants)
	}
	if !varMap["firstName"] {
		t.Errorf("expected 'firstName' in variants %v", variants)
	}
}

func TestMatchFuzzy(t *testing.T) {
	candidates := []string{"email", "email_address", "phone", "name"}
	matches := MatchFuzzy("email", candidates, 0.5)

	found := false
	for _, m := range matches {
		if m.Value == "email" && m.Score == 1.0 {
			found = true
		}
	}
	if !found {
		t.Error("expected exact match for 'email'")
	}

	// phone should NOT match
	for _, m := range matches {
		if m.Value == "phone" {
			t.Error("phone should not fuzzy-match 'email'")
		}
	}
}
