//go:build bdd

package steps

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cucumber/godog"
)

// RegisterAuthSteps registers authentication and admin-related step definitions.
func RegisterAuthSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// --- Auth credential steps ---
	ctx.Step(`^I authenticate as "([^"]*)" with password "([^"]*)"$`, func(username, password string) error {
		tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
		return nil
	})

	ctx.Step(`^I authenticate with API key "([^"]*)"$`, func(key string) error {
		key = tc.resolveVars(key)
		tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(key+":ignored"))
		return nil
	})

	ctx.Step(`^I clear authentication$`, func() error {
		tc.AuthHeader = ""
		return nil
	})

	ctx.Step(`^I authenticate with stored API key "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		key := fmt.Sprintf("%v", val)
		tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(key+":ignored"))
		return nil
	})

	// --- Admin user management steps ---
	ctx.Step(`^I create a user with username "([^"]*)" password "([^"]*)" role "([^"]*)"$`, func(username, password, role string) error {
		body := map[string]interface{}{
			"username": username,
			"password": password,
			"role":     role,
		}
		return tc.POST("/admin/users", body)
	})

	ctx.Step(`^I create a user with username "([^"]*)" password "([^"]*)" role "([^"]*)" email "([^"]*)"$`, func(username, password, role, email string) error {
		body := map[string]interface{}{
			"username": username,
			"password": password,
			"role":     role,
			"email":    email,
		}
		return tc.POST("/admin/users", body)
	})

	ctx.Step(`^I list all users$`, func() error {
		return tc.GET("/admin/users")
	})

	ctx.Step(`^I get user by ID "([^"]*)"$`, func(idVar string) error {
		resolved := tc.resolveVars(idVar)
		return tc.GET("/admin/users/" + resolved)
	})

	ctx.Step(`^I get user by stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.GET(fmt.Sprintf("/admin/users/%v", val))
	})

	ctx.Step(`^I update user "([^"]*)" with:$`, func(idVar string, body *godog.DocString) error {
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		resolved := tc.resolveVars(idVar)
		return tc.PUT("/admin/users/"+resolved, req)
	})

	ctx.Step(`^I update user with stored ID "([^"]*)" with:$`, func(varName string, body *godog.DocString) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		return tc.PUT(fmt.Sprintf("/admin/users/%v", val), req)
	})

	ctx.Step(`^I delete user with stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.DELETE(fmt.Sprintf("/admin/users/%v", val))
	})

	// --- Admin API key management steps ---
	ctx.Step(`^I create an API key with name "([^"]*)" role "([^"]*)" expires_in (\d+)$`, func(name, role string, expiresIn int) error {
		body := map[string]interface{}{
			"name":       name,
			"role":       role,
			"expires_in": expiresIn,
		}
		return tc.POST("/admin/apikeys", body)
	})

	ctx.Step(`^I create an API key with name "([^"]*)" role "([^"]*)" expires_in (\d+) for_user_id (\d+)$`, func(name, role string, expiresIn, userID int) error {
		body := map[string]interface{}{
			"name":        name,
			"role":        role,
			"expires_in":  expiresIn,
			"for_user_id": userID,
		}
		return tc.POST("/admin/apikeys", body)
	})

	ctx.Step(`^I list all API keys$`, func() error {
		return tc.GET("/admin/apikeys")
	})

	ctx.Step(`^I get API key by stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.GET(fmt.Sprintf("/admin/apikeys/%v", val))
	})

	ctx.Step(`^I update API key with stored ID "([^"]*)" with:$`, func(varName string, body *godog.DocString) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		return tc.PUT(fmt.Sprintf("/admin/apikeys/%v", val), req)
	})

	ctx.Step(`^I delete API key with stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.DELETE(fmt.Sprintf("/admin/apikeys/%v", val))
	})

	ctx.Step(`^I revoke API key with stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.POST(fmt.Sprintf("/admin/apikeys/%v/revoke", val), nil)
	})

	ctx.Step(`^I rotate API key with stored ID "([^"]*)" expires_in (\d+)$`, func(varName string, expiresIn int) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		body := map[string]interface{}{
			"expires_in": expiresIn,
		}
		return tc.POST(fmt.Sprintf("/admin/apikeys/%v/rotate", val), body)
	})

	ctx.Step(`^I list roles$`, func() error {
		return tc.GET("/admin/roles")
	})

	// --- Metrics steps ---
	ctx.Step(`^I get the metrics$`, func() error {
		return tc.GET("/metrics")
	})

	ctx.Step(`^the response should contain Prometheus metric "([^"]*)"$`, func(metricName string) error {
		body := string(tc.LastBody)
		if !strings.Contains(body, metricName) {
			return fmt.Errorf("metrics response does not contain %q (first 500 chars: %s)", metricName, truncate(body, 500))
		}
		return nil
	})

	// --- Response assertions for nested fields ---
	ctx.Step(`^the response users array should have length (\d+)$`, func(expected int) error {
		if tc.LastJSON == nil {
			return fmt.Errorf("no JSON object in last response")
		}
		users, ok := tc.LastJSON["users"]
		if !ok {
			return fmt.Errorf("no 'users' field in response: %s", string(tc.LastBody))
		}
		arr, ok := users.([]interface{})
		if !ok {
			return fmt.Errorf("'users' field is not an array: %T", users)
		}
		if len(arr) != expected {
			return fmt.Errorf("expected users array length %d, got %d", expected, len(arr))
		}
		return nil
	})

	ctx.Step(`^the response apikeys array should have length (\d+)$`, func(expected int) error {
		if tc.LastJSON == nil {
			return fmt.Errorf("no JSON object in last response")
		}
		keys, ok := tc.LastJSON["api_keys"]
		if !ok {
			return fmt.Errorf("no 'api_keys' field in response: %s", string(tc.LastBody))
		}
		arr, ok := keys.([]interface{})
		if !ok {
			return fmt.Errorf("'api_keys' field is not an array: %T", keys)
		}
		if len(arr) != expected {
			return fmt.Errorf("expected api_keys array length %d, got %d", expected, len(arr))
		}
		return nil
	})

	ctx.Step(`^the response roles array should have length (\d+)$`, func(expected int) error {
		if tc.LastJSON == nil {
			return fmt.Errorf("no JSON object in last response")
		}
		roles, ok := tc.LastJSON["roles"]
		if !ok {
			return fmt.Errorf("no 'roles' field in response: %s", string(tc.LastBody))
		}
		arr, ok := roles.([]interface{})
		if !ok {
			return fmt.Errorf("'roles' field is not an array: %T", roles)
		}
		if len(arr) != expected {
			return fmt.Errorf("expected roles array length %d, got %d", expected, len(arr))
		}
		return nil
	})

	ctx.Step(`^the response field "([^"]*)" should not be empty$`, func(field string) error {
		val, err := tc.JSONFieldString(field)
		if err != nil {
			return err
		}
		if val == "" {
			return fmt.Errorf("field %q is empty", field)
		}
		return nil
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
