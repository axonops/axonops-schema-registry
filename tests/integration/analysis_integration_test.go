//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// uniqueSubject returns a unique subject name for integration tests.
func uniqueSubject(prefix string) string {
	return fmt.Sprintf("test-analysis-%s-%d", prefix, time.Now().UnixNano())
}

// registerAnalysisSchema registers a schema and returns the response status code.
func registerAnalysisSchema(t *testing.T, subject, schemaStr string) {
	t.Helper()
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", map[string]interface{}{
		"schema": schemaStr,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register schema for %s: status %d", subject, resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAnalysis_Validate_Integration(t *testing.T) {
	t.Run("valid schema", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/validate", map[string]interface{}{
			"schema": `{"type":"record","name":"ValidateTest","fields":[{"name":"id","type":"long"}]}`,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["is_valid"] != true {
			t.Errorf("Expected is_valid=true, got %v", result["is_valid"])
		}
	})

	t.Run("invalid schema", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/validate", map[string]interface{}{
			"schema": `{invalid json`,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["is_valid"] != false {
			t.Errorf("Expected is_valid=false, got %v", result["is_valid"])
		}
	})
}

func TestAnalysis_Normalize_Integration(t *testing.T) {
	t.Run("valid schema", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/normalize", map[string]interface{}{
			"schema": `{"type":"record","name":"NormTest","fields":[{"name":"id","type":"long"}]}`,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["canonical"] == nil || result["canonical"] == "" {
			t.Error("Expected non-empty canonical")
		}
		if result["fingerprint"] == nil || result["fingerprint"] == "" {
			t.Error("Expected non-empty fingerprint")
		}
	})

	t.Run("invalid schema", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/normalize", map[string]interface{}{
			"schema": `{invalid`,
		})
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422, got %d", resp.StatusCode)
		}
	})
}

func TestAnalysis_Search_Integration(t *testing.T) {
	sub1 := uniqueSubject("search-1")
	sub2 := uniqueSubject("search-2")
	registerAnalysisSchema(t, sub1, `{"type":"record","name":"SearchA","fields":[{"name":"email_address","type":"string"}]}`)
	registerAnalysisSchema(t, sub2, `{"type":"record","name":"SearchB","fields":[{"name":"phone_number","type":"string"}]}`)

	t.Run("find by substring", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/search", map[string]interface{}{
			"query": "email_address",
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		count := int(result["count"].(float64))
		if count < 1 {
			t.Errorf("Expected at least 1 match, got %d", count)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/search", map[string]interface{}{
			"query": "nonexistent_field_xyz_12345",
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		count := int(result["count"].(float64))
		if count != 0 {
			t.Errorf("Expected 0 matches, got %d", count)
		}
	})
}

func TestAnalysis_Quality_Integration(t *testing.T) {
	sub := uniqueSubject("quality")
	registerAnalysisSchema(t, sub, `{"type":"record","name":"QualityTest","fields":[{"name":"id","type":"long"},{"name":"name","type":"string"}]}`)

	t.Run("inline schema", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/quality", map[string]interface{}{
			"schema": `{"type":"record","name":"QualInline","fields":[{"name":"id","type":"long"}]}`,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["overall_score"] == nil {
			t.Error("Expected overall_score in response")
		}
		if result["grade"] == nil {
			t.Error("Expected grade in response")
		}
	})

	t.Run("by subject", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/quality", map[string]interface{}{
			"subject": sub,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["overall_score"] == nil {
			t.Error("Expected overall_score in response")
		}
	})
}

func TestAnalysis_Complexity_Integration(t *testing.T) {
	sub := uniqueSubject("complexity")
	registerAnalysisSchema(t, sub, `{"type":"record","name":"ComplexTest","fields":[{"name":"id","type":"long"}]}`)

	t.Run("inline schema", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/complexity", map[string]interface{}{
			"schema": `{"type":"record","name":"SimpleSchema","fields":[{"name":"id","type":"long"}]}`,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["field_count"] == nil {
			t.Error("Expected field_count in response")
		}
		if result["grade"] != "A" {
			t.Errorf("Expected grade A, got %v", result["grade"])
		}
	})

	t.Run("by subject", func(t *testing.T) {
		resp := doRequest(t, "POST", "/schemas/complexity", map[string]interface{}{
			"subject": sub,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["field_count"] == nil {
			t.Error("Expected field_count in response")
		}
	})
}

func TestAnalysis_SubjectValidation_Integration(t *testing.T) {
	sub1 := uniqueSubject("subval-1")
	sub2 := uniqueSubject("subval-2")
	registerAnalysisSchema(t, sub1, `{"type":"record","name":"SubVal1","fields":[{"name":"id","type":"long"}]}`)
	registerAnalysisSchema(t, sub2, `{"type":"record","name":"SubVal2","fields":[{"name":"id","type":"long"}]}`)

	t.Run("validate subject name", func(t *testing.T) {
		resp := doRequest(t, "POST", "/subjects/validate", map[string]interface{}{
			"subject": "orders-value",
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["valid"] != true {
			t.Errorf("Expected valid=true, got %v", result["valid"])
		}
	})

	t.Run("match subjects", func(t *testing.T) {
		resp := doRequest(t, "POST", "/subjects/match", map[string]interface{}{
			"pattern": "test-analysis-subval-.*",
			"mode":    "regex",
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		count := int(result["count"].(float64))
		if count < 2 {
			t.Errorf("Expected at least 2 matches, got %d", count)
		}
	})

	t.Run("count subjects", func(t *testing.T) {
		resp := doRequest(t, "GET", "/subjects/count", nil)
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		count := int(result["count"].(float64))
		if count < 2 {
			t.Errorf("Expected at least 2 subjects, got %d", count)
		}
	})
}

func TestAnalysis_HistoryExport_Integration(t *testing.T) {
	sub := uniqueSubject("history")
	registerAnalysisSchema(t, sub, `{"type":"record","name":"HistoryTest","fields":[{"name":"id","type":"long"}]}`)

	t.Run("get history", func(t *testing.T) {
		resp := doRequest(t, "GET", "/subjects/"+sub+"/history", nil)
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		count := int(result["count"].(float64))
		if count != 1 {
			t.Errorf("Expected count=1, got %d", count)
		}
	})

	t.Run("export subject", func(t *testing.T) {
		resp := doRequest(t, "GET", "/subjects/"+sub+"/export", nil)
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		versions := result["versions"].([]interface{})
		if len(versions) < 1 {
			t.Error("Expected at least 1 version in export")
		}
	})

	t.Run("count versions", func(t *testing.T) {
		resp := doRequest(t, "GET", "/subjects/"+sub+"/versions/count", nil)
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		count := int(result["count"].(float64))
		if count != 1 {
			t.Errorf("Expected count=1, got %d", count)
		}
	})
}

func TestAnalysis_DiffMigrate_Integration(t *testing.T) {
	sub := uniqueSubject("diff")

	// Set NONE compat so we can register incompatible schemas
	doRequest(t, "PUT", "/config/"+sub, map[string]string{"compatibility": "NONE"})
	registerAnalysisSchema(t, sub, `{"type":"record","name":"DiffTest","fields":[{"name":"id","type":"long"},{"name":"old_field","type":"string"}]}`)
	registerAnalysisSchema(t, sub, `{"type":"record","name":"DiffTest","fields":[{"name":"id","type":"long"},{"name":"new_field","type":"int"}]}`)

	t.Run("diff schemas", func(t *testing.T) {
		resp := doRequest(t, "POST", "/subjects/"+sub+"/diff", map[string]interface{}{
			"version1": 1,
			"version2": 2,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["added"] == nil && result["removed"] == nil {
			t.Error("Expected added or removed fields in diff")
		}
	})

	t.Run("plan migration", func(t *testing.T) {
		resp := doRequest(t, "POST", "/subjects/"+sub+"/migrate", map[string]interface{}{
			"target_schema": `{"type":"record","name":"DiffTest","fields":[{"name":"id","type":"long"},{"name":"migrated","type":"string"}]}`,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		steps := result["steps"].([]interface{})
		if len(steps) < 1 {
			t.Error("Expected at least 1 migration step")
		}
	})
}

func TestAnalysis_CompatAnalysis_Integration(t *testing.T) {
	sub1 := uniqueSubject("compat-1")
	sub2 := uniqueSubject("compat-2")
	registerAnalysisSchema(t, sub1, `{"type":"record","name":"CompatA","fields":[{"name":"id","type":"long"},{"name":"shared","type":"string"}]}`)
	registerAnalysisSchema(t, sub2, `{"type":"record","name":"CompatB","fields":[{"name":"id","type":"long"},{"name":"shared","type":"string"},{"name":"extra","type":"int"}]}`)

	t.Run("check compatibility multi", func(t *testing.T) {
		resp := doRequest(t, "POST", "/compatibility/check", map[string]interface{}{
			"schema":   `{"type":"record","name":"CompatA","fields":[{"name":"id","type":"long"},{"name":"shared","type":"string","default":""}]}`,
			"subjects": []string{sub1},
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		results := result["results"].([]interface{})
		if len(results) < 1 {
			t.Error("Expected at least 1 result")
		}
	})

	t.Run("suggest compatible change", func(t *testing.T) {
		resp := doRequest(t, "POST", "/compatibility/subjects/"+sub1+"/suggest", nil)
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["suggestions"] == nil {
			t.Error("Expected suggestions in response")
		}
	})

	t.Run("explain compatibility", func(t *testing.T) {
		resp := doRequest(t, "POST", "/compatibility/subjects/"+sub1+"/explain", map[string]interface{}{
			"schema": `{"type":"record","name":"CompatA","fields":[{"name":"id","type":"long"},{"name":"shared","type":"string"},{"name":"new_required","type":"string"}]}`,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["is_compatible"] == nil {
			t.Error("Expected is_compatible in response")
		}
	})

	t.Run("compare subjects", func(t *testing.T) {
		resp := doRequest(t, "POST", "/compatibility/compare", map[string]interface{}{
			"subject1": sub1,
			"subject2": sub2,
		})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["shared"] == nil {
			t.Error("Expected shared in response")
		}
	})
}

func TestAnalysis_Statistics_Integration(t *testing.T) {
	sub1 := uniqueSubject("stats-1")
	sub2 := uniqueSubject("stats-2")
	registerAnalysisSchema(t, sub1, `{"type":"record","name":"Stats1","fields":[{"name":"user_id","type":"long"},{"name":"created_at","type":"string"}]}`)
	registerAnalysisSchema(t, sub2, `{"type":"record","name":"Stats2","fields":[{"name":"user_id","type":"long"},{"name":"updated_at","type":"string"}]}`)

	t.Run("get registry statistics", func(t *testing.T) {
		resp := doRequest(t, "GET", "/statistics", nil)
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["subject_count"] == nil {
			t.Error("Expected subject_count in response")
		}
		subjectCount := int(result["subject_count"].(float64))
		if subjectCount < 2 {
			t.Errorf("Expected at least 2 subjects, got %d", subjectCount)
		}
	})

	t.Run("check field consistency", func(t *testing.T) {
		resp := doRequest(t, "GET", "/statistics/fields/user_id", nil)
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["consistent"] != true {
			t.Errorf("Expected consistent=true for user_id (same type), got %v", result["consistent"])
		}
	})

	t.Run("detect schema patterns", func(t *testing.T) {
		resp := doRequest(t, "GET", "/statistics/patterns", nil)
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
		if result["common_fields"] == nil {
			t.Error("Expected common_fields in response")
		}
	})
}
