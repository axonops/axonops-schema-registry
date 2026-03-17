package auth

import "testing"

func TestConstantTimeEqualFold_MatchingStrings(t *testing.T) {
	if !constantTimeEqualFold("Admins", "admins") {
		t.Error("expected case-insensitive match")
	}
	if !constantTimeEqualFold("ADMINS", "Admins") {
		t.Error("expected case-insensitive match")
	}
	if !constantTimeEqualFold("exact", "exact") {
		t.Error("expected exact match")
	}
}

func TestConstantTimeEqualFold_NonMatchingStrings(t *testing.T) {
	if constantTimeEqualFold("admin", "user") {
		t.Error("expected no match for different strings")
	}
}

func TestConstantTimeEqualFold_DifferentLengths(t *testing.T) {
	if constantTimeEqualFold("admin", "admins") {
		t.Error("expected no match for different lengths")
	}
}

func TestConstantTimeEqualFold_EmptyStrings(t *testing.T) {
	if !constantTimeEqualFold("", "") {
		t.Error("expected empty strings to match")
	}
	if constantTimeEqualFold("", "notempty") {
		t.Error("expected no match for empty vs non-empty")
	}
}
