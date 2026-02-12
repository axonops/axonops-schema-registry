//go:build bdd

package steps

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
)

// RegisterSchemaSteps registers all schema-related step definitions.
func RegisterSchemaSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// --- Given steps ---
	ctx.Step(`^the schema registry is running$`, func() error {
		return tc.GET("/")
	})
	ctx.Step(`^no subjects exist$`, func() error {
		// The server starts fresh for each scenario (memory backend)
		return nil
	})
	ctx.Step(`^subject "([^"]*)" has schema:$`, func(subject string, schema *godog.DocString) error {
		body := map[string]interface{}{"schema": schema.Content}
		if err := tc.POST("/subjects/"+subject+"/versions", body); err != nil {
			return err
		}
		if tc.LastStatusCode != 200 {
			return fmt.Errorf("expected 200 registering schema, got %d: %s", tc.LastStatusCode, string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^subject "([^"]*)" has "([^"]*)" schema:$`, func(subject, schemaType string, schema *godog.DocString) error {
		body := map[string]interface{}{
			"schema":     schema.Content,
			"schemaType": schemaType,
		}
		if err := tc.POST("/subjects/"+subject+"/versions", body); err != nil {
			return err
		}
		if tc.LastStatusCode != 200 {
			return fmt.Errorf("expected 200 registering %s schema, got %d: %s", schemaType, tc.LastStatusCode, string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^the global compatibility level is "([^"]*)"$`, func(level string) error {
		body := map[string]interface{}{"compatibility": level}
		return tc.PUT("/config", body)
	})
	ctx.Step(`^subject "([^"]*)" has compatibility level "([^"]*)"$`, func(subject, level string) error {
		body := map[string]interface{}{"compatibility": level}
		return tc.PUT("/config/"+subject, body)
	})

	// --- Generic HTTP steps ---
	ctx.Step(`^I GET "([^"]*)"$`, func(path string) error {
		return tc.GET(path)
	})
	ctx.Step(`^I POST "([^"]*)" with body:$`, func(path string, body *godog.DocString) error {
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		return tc.POST(path, req)
	})
	ctx.Step(`^I PUT "([^"]*)" with body:$`, func(path string, body *godog.DocString) error {
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		return tc.PUT(path, req)
	})
	ctx.Step(`^I DELETE "([^"]*)"$`, func(path string) error {
		return tc.DELETE(path)
	})

	// --- When steps ---
	ctx.Step(`^I register a schema under subject "([^"]*)":$`, func(subject string, schema *godog.DocString) error {
		body := map[string]interface{}{"schema": schema.Content}
		return tc.POST("/subjects/"+subject+"/versions", body)
	})
	ctx.Step(`^I register a "([^"]*)" schema under subject "([^"]*)":$`, func(schemaType, subject string, schema *godog.DocString) error {
		body := map[string]interface{}{
			"schema":     schema.Content,
			"schemaType": schemaType,
		}
		return tc.POST("/subjects/"+subject+"/versions", body)
	})
	ctx.Step(`^I get schema by ID (\d+)$`, func(id int) error {
		return tc.GET("/schemas/ids/" + strconv.Itoa(id))
	})
	ctx.Step(`^I get the stored schema by ID$`, func() error {
		id, ok := tc.StoredValues["schema_id"]
		if !ok {
			return fmt.Errorf("no stored schema_id")
		}
		return tc.GET(fmt.Sprintf("/schemas/ids/%v", id))
	})
	ctx.Step(`^I get version (\d+) of subject "([^"]*)"$`, func(version int, subject string) error {
		return tc.GET(fmt.Sprintf("/subjects/%s/versions/%d", subject, version))
	})
	ctx.Step(`^I get the latest version of subject "([^"]*)"$`, func(subject string) error {
		return tc.GET(fmt.Sprintf("/subjects/%s/versions/latest", subject))
	})
	ctx.Step(`^I list all subjects$`, func() error {
		return tc.GET("/subjects")
	})
	ctx.Step(`^I list versions of subject "([^"]*)"$`, func(subject string) error {
		return tc.GET("/subjects/" + subject + "/versions")
	})
	ctx.Step(`^I delete subject "([^"]*)"$`, func(subject string) error {
		return tc.DELETE("/subjects/" + subject)
	})
	ctx.Step(`^I permanently delete subject "([^"]*)"$`, func(subject string) error {
		return tc.DoRequest("DELETE", "/subjects/"+subject+"?permanent=true", nil)
	})
	ctx.Step(`^I delete version (\d+) of subject "([^"]*)"$`, func(version int, subject string) error {
		return tc.DELETE(fmt.Sprintf("/subjects/%s/versions/%d", subject, version))
	})
	ctx.Step(`^I get the schema types$`, func() error {
		return tc.GET("/schemas/types")
	})
	ctx.Step(`^I lookup schema in subject "([^"]*)":$`, func(subject string, schema *godog.DocString) error {
		body := map[string]interface{}{"schema": schema.Content}
		return tc.POST("/subjects/"+subject, body)
	})
	ctx.Step(`^I lookup a "([^"]*)" schema in subject "([^"]*)":$`, func(schemaType, subject string, schema *godog.DocString) error {
		body := map[string]interface{}{
			"schema":     schema.Content,
			"schemaType": schemaType,
		}
		return tc.POST("/subjects/"+subject, body)
	})
	ctx.Step(`^I lookup schema in subject "([^"]*)" with deleted:$`, func(subject string, schema *godog.DocString) error {
		body := map[string]interface{}{"schema": schema.Content}
		return tc.POST("/subjects/"+subject+"?deleted=true", body)
	})
	ctx.Step(`^I check compatibility of schema against subject "([^"]*)":$`, func(subject string, schema *godog.DocString) error {
		body := map[string]interface{}{"schema": schema.Content}
		return tc.POST("/compatibility/subjects/"+subject+"/versions/latest", body)
	})
	ctx.Step(`^I check compatibility of "([^"]*)" schema against subject "([^"]*)":$`, func(schemaType, subject string, schema *godog.DocString) error {
		body := map[string]interface{}{
			"schema":     schema.Content,
			"schemaType": schemaType,
		}
		return tc.POST("/compatibility/subjects/"+subject+"/versions/latest", body)
	})
	ctx.Step(`^I get the global config$`, func() error {
		return tc.GET("/config")
	})
	ctx.Step(`^I set the global config to "([^"]*)"$`, func(level string) error {
		body := map[string]interface{}{"compatibility": level}
		return tc.PUT("/config", body)
	})
	ctx.Step(`^I get the config for subject "([^"]*)"$`, func(subject string) error {
		return tc.GET("/config/" + subject)
	})
	ctx.Step(`^I set the config for subject "([^"]*)" to "([^"]*)"$`, func(subject, level string) error {
		body := map[string]interface{}{"compatibility": level}
		return tc.PUT("/config/"+subject, body)
	})
	ctx.Step(`^I delete the config for subject "([^"]*)"$`, func(subject string) error {
		return tc.DELETE("/config/" + subject)
	})
	ctx.Step(`^I get the global mode$`, func() error {
		return tc.GET("/mode")
	})
	ctx.Step(`^I set the global mode to "([^"]*)"$`, func(mode string) error {
		body := map[string]interface{}{"mode": mode}
		return tc.PUT("/mode", body)
	})
	ctx.Step(`^I get the subjects for schema ID (\d+)$`, func(id int) error {
		return tc.GET(fmt.Sprintf("/schemas/ids/%d/subjects", id))
	})
	ctx.Step(`^I get the subjects for the stored schema ID$`, func() error {
		id, ok := tc.StoredValues["schema_id"]
		if !ok {
			return fmt.Errorf("no stored schema_id")
		}
		return tc.GET(fmt.Sprintf("/schemas/ids/%v/subjects", id))
	})
	ctx.Step(`^I list all schemas$`, func() error {
		return tc.GET("/schemas")
	})

	// --- Then steps ---
	ctx.Step(`^the response status should be (\d+)$`, func(expected int) error {
		if tc.LastStatusCode != expected {
			return fmt.Errorf("expected status %d, got %d: %s", expected, tc.LastStatusCode, string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^the response should contain "([^"]*)"$`, func(expected string) error {
		if !strings.Contains(string(tc.LastBody), expected) {
			return fmt.Errorf("response does not contain %q: %s", expected, string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^the response should have field "([^"]*)"$`, func(field string) error {
		_, err := tc.JSONField(field)
		return err
	})
	ctx.Step(`^the response field "([^"]*)" should be "([^"]*)"$`, func(field, expected string) error {
		val, err := tc.JSONFieldString(field)
		if err != nil {
			return err
		}
		if val != expected {
			return fmt.Errorf("field %q: expected %q, got %q", field, expected, val)
		}
		return nil
	})
	ctx.Step(`^the response field "([^"]*)" should be (\d+)$`, func(field string, expected int) error {
		val, err := tc.JSONFieldInt(field)
		if err != nil {
			return err
		}
		if val != expected {
			return fmt.Errorf("field %q: expected %d, got %d", field, expected, val)
		}
		return nil
	})
	ctx.Step(`^the response should be an array of length (\d+)$`, func(expected int) error {
		// Handle null/empty body as empty array
		if expected == 0 {
			body := strings.TrimSpace(string(tc.LastBody))
			if body == "null" || body == "" || body == "[]" {
				return nil
			}
			if tc.LastJSONArray != nil && len(tc.LastJSONArray) == 0 {
				return nil
			}
			if tc.LastJSONArray != nil {
				return fmt.Errorf("expected empty array, got length %d: %s", len(tc.LastJSONArray), body)
			}
			return fmt.Errorf("expected empty array, got: %s", body)
		}
		if tc.LastJSONArray == nil {
			return fmt.Errorf("response is not a JSON array: %s", string(tc.LastBody))
		}
		if len(tc.LastJSONArray) != expected {
			return fmt.Errorf("expected array length %d, got %d: %s", expected, len(tc.LastJSONArray), string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^the response array should contain "([^"]*)"$`, func(expected string) error {
		if tc.LastJSONArray == nil {
			return fmt.Errorf("response is not a JSON array")
		}
		for _, v := range tc.LastJSONArray {
			if fmt.Sprintf("%v", v) == expected {
				return nil
			}
		}
		return fmt.Errorf("array does not contain %q: %s", expected, string(tc.LastBody))
	})
	ctx.Step(`^the response should have error code (\d+)$`, func(code int) error {
		val, err := tc.JSONFieldInt("error_code")
		if err != nil {
			return err
		}
		if val != code {
			return fmt.Errorf("expected error_code %d, got %d", code, val)
		}
		return nil
	})
	ctx.Step(`^I store the response field "([^"]*)" as "([^"]*)"$`, func(field, key string) error {
		val, err := tc.JSONField(field)
		if err != nil {
			return err
		}
		tc.StoredValues[key] = val
		return nil
	})
	ctx.Step(`^the compatibility check should be compatible$`, func() error {
		if tc.LastStatusCode != 200 {
			return fmt.Errorf("expected 200, got %d: %s", tc.LastStatusCode, string(tc.LastBody))
		}
		val, err := tc.JSONField("is_compatible")
		if err != nil {
			return err
		}
		if val != true {
			return fmt.Errorf("expected is_compatible=true, got %v", val)
		}
		return nil
	})
	ctx.Step(`^the compatibility check should be incompatible$`, func() error {
		if tc.LastStatusCode != 200 {
			return fmt.Errorf("expected 200, got %d: %s", tc.LastStatusCode, string(tc.LastBody))
		}
		val, err := tc.JSONField("is_compatible")
		if err != nil {
			return err
		}
		if val != false {
			return fmt.Errorf("expected is_compatible=false, got %v", val)
		}
		return nil
	})
	ctx.Step(`^the response should be valid JSON$`, func() error {
		var obj interface{}
		if err := json.Unmarshal(tc.LastBody, &obj); err != nil {
			return fmt.Errorf("invalid JSON: %w\n%s", err, string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^the response field "([^"]*)" should be (true|false)$`, func(field, expected string) error {
		val, err := tc.JSONField(field)
		if err != nil {
			return err
		}
		expectedBool := expected == "true"
		if val != expectedBool {
			return fmt.Errorf("field %q: expected %v, got %v", field, expectedBool, val)
		}
		return nil
	})
	ctx.Step(`^the response should not have field "([^"]*)"$`, func(field string) error {
		if tc.LastJSON == nil {
			return nil // no JSON object means field is absent
		}
		_, ok := tc.LastJSON[field]
		if ok {
			return fmt.Errorf("field %q should not be present in response: %s", field, string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^the response body should contain "([^"]*)"$`, func(expected string) error {
		if !strings.Contains(string(tc.LastBody), expected) {
			return fmt.Errorf("response body does not contain %q: %s", expected, string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^the response body should not contain "([^"]*)"$`, func(expected string) error {
		if strings.Contains(string(tc.LastBody), expected) {
			return fmt.Errorf("response body should not contain %q but does: %s", expected, string(tc.LastBody))
		}
		return nil
	})
	ctx.Step(`^the response should contain error message matching "([^"]*)"$`, func(pattern string) error {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
		msg, msgErr := tc.JSONFieldString("message")
		if msgErr != nil {
			return fmt.Errorf("no message field in response: %s", string(tc.LastBody))
		}
		if !re.MatchString(msg) {
			return fmt.Errorf("message %q does not match pattern %q", msg, pattern)
		}
		return nil
	})
}
