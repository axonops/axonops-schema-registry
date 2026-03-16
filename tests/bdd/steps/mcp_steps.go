//go:build bdd

package steps

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// getMCPSession lazily creates an MCP client for the scenario.
// Connects to the Docker-deployed MCP server via HTTP Streamable transport.
func getMCPSession(tc *TestContext) (*gomcp.ClientSession, error) {
	// Check if we already have a session stored
	if s, ok := tc.StoredValues["_mcp_session"].(*gomcp.ClientSession); ok && s != nil {
		return s, nil
	}

	mcpURL, ok := tc.StoredValues["_mcp_url"].(string)
	if !ok || mcpURL == "" {
		return nil, fmt.Errorf("MCP URL not set (StoredValues[\"_mcp_url\"] is empty)")
	}

	transport := &gomcp.StreamableClientTransport{
		Endpoint: mcpURL,
	}

	client := gomcp.NewClient(&gomcp.Implementation{Name: "bdd-client", Version: "1.0"}, nil)
	cs, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		return nil, fmt.Errorf("MCP HTTP connect to %s: %w", mcpURL, err)
	}

	tc.StoredValues["_mcp_session"] = cs
	return cs, nil
}

// closeMCPSession cleans up the MCP session if one was created.
func closeMCPSession(tc *TestContext) {
	if s, ok := tc.StoredValues["_mcp_session"].(*gomcp.ClientSession); ok && s != nil {
		s.Close()
		delete(tc.StoredValues, "_mcp_session")
	}
}

// extractText extracts the text from the first TextContent in a CallToolResult.
func extractText(result *gomcp.CallToolResult) (string, error) {
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty MCP result content")
	}
	data, err := result.Content[0].MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("marshal MCP content: %w", err)
	}
	var wire struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return "", fmt.Errorf("unmarshal MCP content: %w", err)
	}
	return wire.Text, nil
}

// resolveStoredVars replaces $variable references in a string with values
// from tc.StoredValues. For example, "$user_id" is replaced with the string
// representation of tc.StoredValues["user_id"]. This allows feature files to
// use dynamic IDs instead of hardcoded values.
func resolveStoredVars(tc *TestContext, s string) string {
	for key, val := range tc.StoredValues {
		placeholder := "$" + key
		if strings.Contains(s, placeholder) {
			// For float64 values (from JSON), format as integer if whole number
			if f, ok := val.(float64); ok && f == float64(int64(f)) {
				s = strings.ReplaceAll(s, placeholder, strconv.FormatInt(int64(f), 10))
			} else {
				s = strings.ReplaceAll(s, placeholder, fmt.Sprintf("%v", val))
			}
		}
	}
	return s
}

// RegisterMCPSteps registers MCP-related step definitions.
func RegisterMCPSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// Clean up MCP session after each scenario
	ctx.After(func(gctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		closeMCPSession(tc)
		return gctx, nil
	})

	ctx.Step(`^I call MCP tool "([^"]*)"$`, func(toolName string) error {
		cs, err := getMCPSession(tc)
		if err != nil {
			return err
		}
		result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
			Name: toolName,
		})
		if err != nil {
			tc.MCPError = err
			tc.MCPResultText = ""
			tc.MCPResultIsError = false
			return nil
		}
		tc.MCPError = nil
		tc.MCPResultIsError = result.IsError
		text, err := extractText(result)
		if err != nil {
			return err
		}
		tc.MCPResultText = text
		return nil
	})

	ctx.Step(`^I call MCP tool "([^"]*)" with input:$`, func(toolName string, table *godog.Table) error {
		cs, err := getMCPSession(tc)
		if err != nil {
			return err
		}

		args := make(map[string]any)
		for _, row := range table.Rows {
			if len(row.Cells) >= 2 {
				key := row.Cells[0].Value
				val := row.Cells[1].Value
				// Resolve $variable references from StoredValues
				if strings.HasPrefix(val, "$") {
					varName := val[1:]
					if stored, ok := tc.StoredValues[varName]; ok {
						args[key] = stored
						continue
					}
				}
				// Try to parse as integer, bool, otherwise keep as string
				if n, err := strconv.Atoi(val); err == nil {
					args[key] = n
				} else {
					switch val {
					case "true":
						args[key] = true
					case "false":
						args[key] = false
					default:
						args[key] = val
					}
				}
			}
		}

		result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			tc.MCPError = err
			tc.MCPResultText = ""
			tc.MCPResultIsError = false
			return nil
		}
		tc.MCPError = nil
		tc.MCPResultIsError = result.IsError
		text, err := extractText(result)
		if err != nil {
			return err
		}
		tc.MCPResultText = text
		return nil
	})

	ctx.Step(`^I call MCP tool "([^"]*)" with JSON input:$`, func(toolName string, body *godog.DocString) error {
		cs, err := getMCPSession(tc)
		if err != nil {
			return err
		}

		// Resolve $variable references before JSON parsing
		content := resolveStoredVars(tc, body.Content)

		var args map[string]any
		if err := json.Unmarshal([]byte(content), &args); err != nil {
			return fmt.Errorf("invalid JSON input: %w", err)
		}

		result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			tc.MCPError = err
			tc.MCPResultText = ""
			tc.MCPResultIsError = false
			return nil
		}
		tc.MCPError = nil
		tc.MCPResultIsError = result.IsError
		text, err := extractText(result)
		if err != nil {
			return err
		}
		tc.MCPResultText = text
		return nil
	})

	ctx.Step(`^the MCP result should contain "(.+)"$`, func(expected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		expected = strings.ReplaceAll(expected, `\"`, `"`)
		expected = resolveStoredVars(tc, expected)
		if !strings.Contains(tc.MCPResultText, expected) {
			return fmt.Errorf("expected MCP result to contain %q, got: %s", expected, tc.MCPResultText)
		}
		return nil
	})

	ctx.Step(`^the MCP result should not contain "(.+)"$`, func(unexpected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		unexpected = strings.ReplaceAll(unexpected, `\"`, `"`)
		unexpected = resolveStoredVars(tc, unexpected)
		if strings.Contains(tc.MCPResultText, unexpected) {
			return fmt.Errorf("expected MCP result NOT to contain %q, got: %s", unexpected, tc.MCPResultText)
		}
		return nil
	})

	ctx.Step(`^the MCP result should be "(.+)"$`, func(expected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		expected = strings.ReplaceAll(expected, `\"`, `"`)
		if tc.MCPResultText != expected {
			return fmt.Errorf("expected MCP result to be %q, got: %q", expected, tc.MCPResultText)
		}
		return nil
	})

	ctx.Step(`^I register an Avro schema for subject "([^"]*)"$`, func(subject string) error {
		body := map[string]interface{}{
			"schema": `{"type":"string"}`,
		}
		if err := tc.POST("/subjects/"+subject+"/versions", body); err != nil {
			return err
		}
		if tc.LastStatusCode != 200 {
			return fmt.Errorf("expected 200 registering schema, got %d: %s", tc.LastStatusCode, string(tc.LastBody))
		}
		return nil
	})

	// --- MCP JSON field extraction and KMS verification steps ---

	ctx.Step(`^the MCP result field "([^"]*)" should be non-empty$`, func(field string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		val, err := mcpJSONField(tc.MCPResultText, field)
		if err != nil {
			return err
		}
		if val == nil {
			return fmt.Errorf("MCP result field %q is null", field)
		}
		if s, ok := val.(string); ok && s == "" {
			return fmt.Errorf("MCP result field %q is an empty string", field)
		}
		return nil
	})

	ctx.Step(`^the MCP result field "([^"]*)" should be empty or absent$`, func(field string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(tc.MCPResultText), &obj); err != nil {
			return nil // not even JSON — field is absent
		}
		val, ok := obj[field]
		if !ok || val == nil {
			return nil // absent or null
		}
		if s, ok := val.(string); ok && s == "" {
			return nil // empty string
		}
		return fmt.Errorf("MCP result field %q is present and non-empty: %v", field, val)
	})

	ctx.Step(`^I store the MCP result field "([^"]*)" as "([^"]*)"$`, func(field, key string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		val, err := mcpJSONField(tc.MCPResultText, field)
		if err != nil {
			return err
		}
		tc.StoredValues[key] = val
		return nil
	})

	ctx.Step(`^the MCP result field "([^"]*)" should not equal stored "([^"]*)"$`, func(field, key string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		val, err := mcpJSONField(tc.MCPResultText, field)
		if err != nil {
			return err
		}
		stored, ok := tc.StoredValues[key]
		if !ok {
			return fmt.Errorf("no stored value for key %q", key)
		}
		if fmt.Sprintf("%v", val) == fmt.Sprintf("%v", stored) {
			return fmt.Errorf("MCP result field %q equals stored %q: both are %v", field, key, val)
		}
		return nil
	})

	ctx.Step(`^the MCP result field "([^"]*)" should equal stored "([^"]*)"$`, func(field, key string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		val, err := mcpJSONField(tc.MCPResultText, field)
		if err != nil {
			return err
		}
		stored, ok := tc.StoredValues[key]
		if !ok {
			return fmt.Errorf("no stored value for key %q", key)
		}
		if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", stored) {
			return fmt.Errorf("MCP result field %q (%v) does not equal stored %q (%v)", field, val, key, stored)
		}
		return nil
	})

	// --- MCP tool listing steps ---

	ctx.Step(`^I list MCP tools$`, func() error {
		cs, err := getMCPSession(tc)
		if err != nil {
			return err
		}
		result, err := cs.ListTools(context.Background(), &gomcp.ListToolsParams{})
		if err != nil {
			return fmt.Errorf("ListTools: %w", err)
		}
		var names []string
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		data, err := json.Marshal(names)
		if err != nil {
			return fmt.Errorf("failed to marshal tool names: %w", err)
		}
		tc.MCPResultText = string(data)
		tc.MCPError = nil
		return nil
	})

	// --- MCP Resource steps ---

	ctx.Step(`^I read MCP resource "([^"]*)"$`, func(uri string) error {
		// Resolve $variable references in the URI
		uri = resolveStoredVars(tc, uri)
		cs, err := getMCPSession(tc)
		if err != nil {
			return err
		}
		result, err := cs.ReadResource(context.Background(), &gomcp.ReadResourceParams{
			URI: uri,
		})
		if err != nil {
			tc.MCPError = err
			tc.MCPResourceText = ""
			return nil
		}
		tc.MCPError = nil
		if len(result.Contents) == 0 {
			tc.MCPResourceText = ""
			return nil
		}
		tc.MCPResourceText = result.Contents[0].Text
		return nil
	})

	ctx.Step(`^the MCP resource result should contain "(.+)"$`, func(expected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP resource read failed: %v", tc.MCPError)
		}
		expected = strings.ReplaceAll(expected, `\"`, `"`)
		if !strings.Contains(tc.MCPResourceText, expected) {
			return fmt.Errorf("expected MCP resource result to contain %q, got: %s", expected, tc.MCPResourceText)
		}
		return nil
	})

	ctx.Step(`^the MCP resource result should not contain "(.+)"$`, func(unexpected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP resource read failed: %v", tc.MCPError)
		}
		unexpected = strings.ReplaceAll(unexpected, `\"`, `"`)
		if strings.Contains(tc.MCPResourceText, unexpected) {
			return fmt.Errorf("expected MCP resource result NOT to contain %q, got: %s", unexpected, tc.MCPResourceText)
		}
		return nil
	})

	ctx.Step(`^the MCP resource read should fail$`, func() error {
		if tc.MCPError == nil {
			return fmt.Errorf("expected MCP resource read to fail, but it succeeded with: %s", tc.MCPResourceText)
		}
		return nil
	})

	// --- MCP Prompt steps ---

	ctx.Step(`^I get MCP prompt "([^"]*)"$`, func(name string) error {
		cs, err := getMCPSession(tc)
		if err != nil {
			return err
		}
		result, err := cs.GetPrompt(context.Background(), &gomcp.GetPromptParams{
			Name: name,
		})
		if err != nil {
			tc.MCPError = err
			tc.MCPPromptText = ""
			tc.MCPPromptDesc = ""
			return nil
		}
		tc.MCPError = nil
		tc.MCPPromptDesc = result.Description
		var texts []string
		for _, msg := range result.Messages {
			data, merr := msg.Content.MarshalJSON()
			if merr != nil {
				continue
			}
			var wire struct {
				Text string `json:"text"`
			}
			if jerr := json.Unmarshal(data, &wire); jerr == nil {
				texts = append(texts, wire.Text)
			}
		}
		tc.MCPPromptText = strings.Join(texts, "\n")
		return nil
	})

	ctx.Step(`^I get MCP prompt "([^"]*)" with arguments:$`, func(name string, table *godog.Table) error {
		cs, err := getMCPSession(tc)
		if err != nil {
			return err
		}
		args := make(map[string]string)
		for _, row := range table.Rows {
			if len(row.Cells) >= 2 {
				args[row.Cells[0].Value] = row.Cells[1].Value
			}
		}
		result, err := cs.GetPrompt(context.Background(), &gomcp.GetPromptParams{
			Name:      name,
			Arguments: args,
		})
		if err != nil {
			tc.MCPError = err
			tc.MCPPromptText = ""
			tc.MCPPromptDesc = ""
			return nil
		}
		tc.MCPError = nil
		tc.MCPPromptDesc = result.Description
		var texts []string
		for _, msg := range result.Messages {
			data, merr := msg.Content.MarshalJSON()
			if merr != nil {
				continue
			}
			var wire struct {
				Text string `json:"text"`
			}
			if jerr := json.Unmarshal(data, &wire); jerr == nil {
				texts = append(texts, wire.Text)
			}
		}
		tc.MCPPromptText = strings.Join(texts, "\n")
		return nil
	})

	ctx.Step(`^the MCP prompt result should contain "(.+)"$`, func(expected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP prompt get failed: %v", tc.MCPError)
		}
		expected = strings.ReplaceAll(expected, `\"`, `"`)
		if !strings.Contains(tc.MCPPromptText, expected) {
			return fmt.Errorf("expected MCP prompt result to contain %q, got: %s", expected, tc.MCPPromptText)
		}
		return nil
	})

	ctx.Step(`^the MCP prompt description should contain "(.+)"$`, func(expected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP prompt get failed: %v", tc.MCPError)
		}
		expected = strings.ReplaceAll(expected, `\"`, `"`)
		if !strings.Contains(tc.MCPPromptDesc, expected) {
			return fmt.Errorf("expected MCP prompt description to contain %q, got: %s", expected, tc.MCPPromptDesc)
		}
		return nil
	})

	ctx.Step(`^the MCP prompt get should fail$`, func() error {
		if tc.MCPError == nil {
			return fmt.Errorf("expected MCP prompt get to fail, but it succeeded")
		}
		return nil
	})

	ctx.Step(`^MCP confirmations are enabled$`, func() error {
		tc.StoredValues["_mcp_confirmations_enabled"] = true
		return nil
	})

	ctx.Step(`^MCP permission preset is "([^"]*)"$`, func(preset string) error {
		tc.StoredValues["_mcp_permission_preset"] = preset
		return nil
	})

	ctx.Step(`^MCP permission scopes are "([^"]*)"$`, func(scopeList string) error {
		scopes := strings.Split(scopeList, ",")
		for i := range scopes {
			scopes[i] = strings.TrimSpace(scopes[i])
		}
		tc.StoredValues["_mcp_permission_scopes"] = scopes
		return nil
	})

	ctx.Step(`^I call MCP tool "([^"]*)" with JSON input using stored "([^"]*)":$`, func(toolName, storedKey string, body *godog.DocString) error {
		cs, err := getMCPSession(tc)
		if err != nil {
			return err
		}

		var args map[string]any
		if err := json.Unmarshal([]byte(body.Content), &args); err != nil {
			return fmt.Errorf("invalid JSON input: %w", err)
		}

		// Inject the stored value as confirm_token
		stored, ok := tc.StoredValues[storedKey]
		if !ok {
			return fmt.Errorf("no stored value for key %q", storedKey)
		}
		args["confirm_token"] = fmt.Sprintf("%v", stored)

		result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			tc.MCPError = err
			tc.MCPResultText = ""
			tc.MCPResultIsError = false
			return nil
		}
		tc.MCPError = nil
		tc.MCPResultIsError = result.IsError
		text, err := extractText(result)
		if err != nil {
			return err
		}
		tc.MCPResultText = text
		return nil
	})

	ctx.Step(`^the MCP result should not be an error$`, func() error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed with error: %v", tc.MCPError)
		}
		if tc.MCPResultIsError {
			return fmt.Errorf("MCP result has IsError=true: %s", tc.MCPResultText)
		}
		return nil
	})

	ctx.Step(`^the MCP result should be an error$`, func() error {
		if tc.MCPError == nil && !tc.MCPResultIsError {
			return fmt.Errorf("expected MCP result to be an error, but it succeeded: %s", tc.MCPResultText)
		}
		return nil
	})

	// --- Audit log assertion steps ---

	// getAuditLog returns the audit log content from either the fsnotify watcher,
	// in-process buffer, or Docker container's audit log file (via _audit_fetcher).
	// When no audit infrastructure is configured (e.g., running against Confluent's
	// registry which has no audit logging), it returns ("", false, nil) so that
	// callers can silently skip audit assertions rather than failing.
	getAuditLog := func() (string, bool, error) {
		if tc.AuditWatcher != nil {
			tc.AuditWatcher.ReadNewData()
			logStr, err := tc.AuditWatcher.LogString()
			return logStr, true, err
		}
		if tc.AuditBuffer != nil {
			return tc.AuditBuffer.String(), true, nil
		}
		if fetcher, ok := tc.StoredValues["_audit_fetcher"].(func() (string, error)); ok {
			log, err := fetcher()
			return log, true, err
		}
		return "", false, nil
	}

	ctx.Step(`^the audit log should contain event "([^"]*)"$`, func(eventType string) error {
		log, available, err := getAuditLog()
		if err != nil {
			return err
		}
		if !available {
			return nil // No audit infrastructure configured; skip assertion.
		}
		if !strings.Contains(log, eventType) {
			return fmt.Errorf("expected audit log to contain event %q, got: %s", eventType, log)
		}
		return nil
	})

	ctx.Step(`^the audit log should contain "([^"]*)"$`, func(text string) error {
		log, available, err := getAuditLog()
		if err != nil {
			return err
		}
		if !available {
			return nil // No audit infrastructure configured; skip assertion.
		}
		if !strings.Contains(log, text) {
			return fmt.Errorf("expected audit log to contain %q, got: %s", text, log)
		}
		return nil
	})

	ctx.Step(`^the audit log should not contain event "([^"]*)"$`, func(eventType string) error {
		log, available, err := getAuditLog()
		if err != nil {
			return err
		}
		if !available {
			return nil // No audit infrastructure configured; skip assertion.
		}
		if strings.Contains(log, eventType) {
			return fmt.Errorf("expected audit log NOT to contain event %q, got: %s", eventType, log)
		}
		return nil
	})

	// parseAuditEvents parses the audit log into structured events for field-level assertions.
	parseAuditEvents := func(logStr string) []map[string]interface{} {
		var events []map[string]interface{}
		for _, line := range strings.Split(logStr, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Strip null bytes that can appear when reading a file that was
			// truncated while lumberjack still held an fd at a non-zero offset
			// (sparse file).
			if idx := strings.IndexByte(line, '{'); idx > 0 {
				line = line[idx:]
			}
			var event map[string]interface{}
			if json.Unmarshal([]byte(line), &event) == nil {
				events = append(events, event)
			}
		}
		return events
	}

	ctx.Step(`^the audit log should contain event "([^"]*)" for user "([^"]*)"$`, func(eventType, user string) error {
		logStr, available, err := getAuditLog()
		if err != nil {
			return err
		}
		if !available {
			return nil // No audit infrastructure configured; skip assertion.
		}
		for _, event := range parseAuditEvents(logStr) {
			if fmt.Sprintf("%v", event["event_type"]) == eventType && fmt.Sprintf("%v", event["actor_id"]) == user {
				return nil
			}
		}
		return fmt.Errorf("expected audit event %q for actor_id %q not found in log:\n%s", eventType, user, logStr)
	})

	ctx.Step(`^the audit log should contain event "([^"]*)" with subject "([^"]*)"$`, func(eventType, subject string) error {
		logStr, available, err := getAuditLog()
		if err != nil {
			return err
		}
		if !available {
			return nil // No audit infrastructure configured; skip assertion.
		}
		for _, event := range parseAuditEvents(logStr) {
			if fmt.Sprintf("%v", event["event_type"]) == eventType && fmt.Sprintf("%v", event["target_id"]) == subject {
				return nil
			}
		}
		return fmt.Errorf("expected audit event %q with target_id %q not found in log:\n%s", eventType, subject, logStr)
	})

	ctx.Step(`^the audit log should contain event "([^"]*)" with method "([^"]*)"$`, func(eventType, method string) error {
		logStr, available, err := getAuditLog()
		if err != nil {
			return err
		}
		if !available {
			return nil // No audit infrastructure configured; skip assertion.
		}
		for _, event := range parseAuditEvents(logStr) {
			if fmt.Sprintf("%v", event["event_type"]) == eventType && fmt.Sprintf("%v", event["method"]) == method {
				return nil
			}
		}
		return fmt.Errorf("expected audit event %q with method %q not found in log:\n%s", eventType, method, logStr)
	})

	ctx.Step(`^the audit log should contain event "([^"]*)" with path containing "([^"]*)"$`, func(eventType, pathFragment string) error {
		logStr, available, err := getAuditLog()
		if err != nil {
			return err
		}
		if !available {
			return nil // No audit infrastructure configured; skip assertion.
		}
		for _, event := range parseAuditEvents(logStr) {
			if fmt.Sprintf("%v", event["event_type"]) == eventType {
				if path, ok := event["path"].(string); ok && strings.Contains(path, pathFragment) {
					return nil
				}
			}
		}
		return fmt.Errorf("expected audit event %q with path containing %q not found in log:\n%s", eventType, pathFragment, logStr)
	})

	// Composite audit assertion: validates multiple fields of a single audit event at once.
	// Usage:
	//   And the audit log should contain an event:
	//     | event_type  | apikey_create  |
	//     | actor_id    | admin          |
	//     | actor_type  | user           |
	//     | auth_method | basic          |
	//     | role        | admin          |
	//     | method      | POST           |
	//     | path        | /admin/apikeys |
	//     | status_code | 201            |
	//
	// Supported fields: any JSON field in the audit event (event_type, outcome, actor_id,
	// actor_type, role, auth_method, target_type, target_id, source_ip, user_agent, method,
	// path, status_code, reason, error, before_hash, after_hash, etc.).
	// The "path" field uses "contains" matching; all others use exact match.
	// A value ending with "*" uses prefix matching (e.g., | after_hash | sha256:* |).
	// An empty value (e.g., | actor_id | |) matches the empty string.
	ctx.Step(`^the audit log should contain an event:$`, func(table *godog.Table) error {
		// Build expected field map from the table.
		expected := make(map[string]string)
		for _, row := range table.Rows {
			if len(row.Cells) >= 2 {
				expected[row.Cells[0].Value] = row.Cells[1].Value
			}
		}

		if _, ok := expected["event_type"]; !ok {
			return fmt.Errorf("audit assertion table must include event_type")
		}

		// matchEvents checks the audit log for a matching event.
		matchEvents := func(events []map[string]interface{}) (bool, map[string]interface{}, int) {
			var bestMatch map[string]interface{}
			bestMatchCount := 0
			for _, event := range events {
				matchCount := 0
				allMatch := true
				for field, wantVal := range expected {
					rawVal := event[field]
					gotVal := fmt.Sprintf("%v", rawVal)
					if field == "status_code" {
						if num, ok := rawVal.(float64); ok {
							gotVal = fmt.Sprintf("%d", int(num))
						}
					}
					if rawVal == nil && wantVal == "" {
						matchCount++
					} else if field == "path" {
						if path, ok := event["path"].(string); ok && strings.Contains(path, wantVal) {
							matchCount++
						} else {
							allMatch = false
						}
					} else if strings.HasSuffix(wantVal, "*") {
						prefix := strings.TrimSuffix(wantVal, "*")
						if strings.HasPrefix(gotVal, prefix) {
							matchCount++
						} else {
							allMatch = false
						}
					} else {
						if gotVal == wantVal {
							matchCount++
						} else {
							allMatch = false
						}
					}
				}
				if allMatch {
					return true, nil, 0
				}
				if matchCount > bestMatchCount {
					bestMatchCount = matchCount
					bestMatch = event
				}
			}
			return false, bestMatch, bestMatchCount
		}

		// Use AuditWatcher channel-based wait if available (fast path).
		var logStr string
		var events []map[string]interface{}
		var bestMatch map[string]interface{}

		if tc.AuditWatcher != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			found, bm, _ := tc.AuditWatcher.WaitForMatch(ctx, matchEvents)
			if found {
				return nil
			}
			bestMatch = bm
			events = tc.AuditWatcher.Events()
			logStr, _ = tc.AuditWatcher.LogString()
		} else {
			// Fallback: polling via docker exec or in-process buffer (legacy path).
			for attempt := 0; attempt < 5; attempt++ {
				var available bool
				var err error
				logStr, available, err = getAuditLog()
				if err != nil {
					return err
				}
				if !available {
					return nil
				}

				events = parseAuditEvents(logStr)
				found, bm, _ := matchEvents(events)
				if found {
					return nil
				}
				bestMatch = bm
				time.Sleep(100 * time.Millisecond)
			}
		}

		// Build a helpful error message showing the best partial match.
		var detail strings.Builder
		detail.WriteString("expected audit event not found. Wanted:\n")
		for field, val := range expected {
			detail.WriteString(fmt.Sprintf("  %s = %q\n", field, val))
		}
		if bestMatch != nil {
			detail.WriteString("best partial match:\n")
			for field := range expected {
				detail.WriteString(fmt.Sprintf("  %s = %q\n", field, fmt.Sprintf("%v", bestMatch[field])))
			}
		}
		detail.WriteString(fmt.Sprintf("full audit log (%d events):\n%s", len(events), logStr))
		return fmt.Errorf("%s", detail.String())
	})

	ctx.Step(`^I can unwrap the MCP result encrypted key material using KMS type "([^"]*)" and key ID "([^"]*)"$`, func(kmsType, keyID string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(tc.MCPResultText), &obj); err != nil {
			return fmt.Errorf("parse MCP result as JSON: %w", err)
		}

		encMaterial, ok := obj["encryptedKeyMaterial"]
		if !ok || encMaterial == nil {
			return fmt.Errorf("encryptedKeyMaterial not found in MCP result")
		}
		ciphertext, ok := encMaterial.(string)
		if !ok || ciphertext == "" {
			return fmt.Errorf("encryptedKeyMaterial is not a non-empty string: %v", encMaterial)
		}

		keyMaterial, ok := obj["keyMaterial"]
		if !ok || keyMaterial == nil {
			return fmt.Errorf("keyMaterial not found in MCP result")
		}
		expectedPlaintext, ok := keyMaterial.(string)
		if !ok || expectedPlaintext == "" {
			return fmt.Errorf("keyMaterial is not a non-empty string: %v", keyMaterial)
		}

		// Decode encryptedKeyMaterial (base64-encoded ciphertext like "vault:v1:...")
		rawCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
		if err != nil {
			return fmt.Errorf("base64 decode of encryptedKeyMaterial: %w", err)
		}

		// Decrypt via KMS Transit
		decryptedBase64, err := transitDecrypt(kmsType, keyID, string(rawCiphertext))
		if err != nil {
			return fmt.Errorf("transit decrypt: %w", err)
		}

		// Compare decrypted plaintext with keyMaterial
		decryptedBytes, err := base64.StdEncoding.DecodeString(decryptedBase64)
		if err != nil {
			return fmt.Errorf("base64 decode of decrypted plaintext: %w", err)
		}

		expectedBytes, err := base64.StdEncoding.DecodeString(expectedPlaintext)
		if err != nil {
			return fmt.Errorf("base64 decode of keyMaterial: %w", err)
		}

		if !bytes.Equal(decryptedBytes, expectedBytes) {
			return fmt.Errorf("unwrapped key material does not match: decrypted %d bytes, expected %d bytes",
				len(decryptedBytes), len(expectedBytes))
		}

		return nil
	})
}

// mcpJSONField extracts a field from the MCP result text (JSON string).
func mcpJSONField(resultText, field string) (interface{}, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(resultText), &obj); err != nil {
		return nil, fmt.Errorf("parse MCP result as JSON: %w (text: %s)", err, resultText)
	}
	val, ok := obj[field]
	if !ok {
		return nil, fmt.Errorf("field %q not found in MCP result: %s", field, resultText)
	}
	return val, nil
}
