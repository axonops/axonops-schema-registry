package mcp

import (
	"testing"
)

func TestScoreSchemaQualityGoodSchema(t *testing.T) {
	fields := []FieldInfo{
		{Name: "id", Type: "int", Required: true, Doc: "Unique identifier"},
		{Name: "name", Type: "record", Required: true, Doc: "User name"},
		{Name: "email", Type: "record", Required: false, HasDefault: true, Doc: "Email address"},
	}
	schemaStr := `{"namespace":"com.example","doc":"A user record"}`
	result := ScoreSchemaQuality(fields, schemaStr, "AVRO")

	if result.OverallScore == 0 {
		t.Fatal("expected non-zero score for good schema")
	}
	if result.Grade == "" {
		t.Fatal("expected a grade")
	}
	if result.MaxScore == 0 {
		t.Fatal("expected non-zero max score")
	}
	if result.Grade == "F" {
		t.Errorf("expected good grade for well-documented schema, got %s", result.Grade)
	}
}

func TestScoreSchemaQualityPoorSchema(t *testing.T) {
	fields := []FieldInfo{
		{Name: "UserName", Type: "string", Required: true},
		{Name: "EmailAddr", Type: "string", Required: true},
	}
	schemaStr := `{}`
	result := ScoreSchemaQuality(fields, schemaStr, "AVRO")

	if result.Grade == "A" {
		t.Error("expected poor grade for schema with bad naming and no docs")
	}
	if len(result.QuickWins) == 0 {
		t.Error("expected quick wins for poor schema")
	}
}

func TestScoreSchemaQualityEmptyFields(t *testing.T) {
	result := ScoreSchemaQuality(nil, "{}", "AVRO")
	if result.OverallScore < 0 {
		t.Error("score should not be negative")
	}
	if result.Grade == "" {
		t.Fatal("expected a grade even with no fields")
	}
}
