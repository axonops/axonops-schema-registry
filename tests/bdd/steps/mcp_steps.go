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

	"github.com/cucumber/godog"
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/config"
	mcpkg "github.com/axonops/axonops-schema-registry/internal/mcp"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/storage"
)

// mcpState holds per-scenario MCP client state.
type mcpState struct {
	session *gomcp.ClientSession
	ss      *gomcp.ServerSession
}

// getMCPSession lazily creates an MCP in-process client for the scenario.
func getMCPSession(tc *TestContext) (*gomcp.ClientSession, error) {
	// Check if we already have a session stored
	if s, ok := tc.StoredValues["_mcp_state"].(*mcpState); ok && s.session != nil {
		return s.session, nil
	}

	reg, ok := tc.Registry.(*registry.Registry)
	if !ok || reg == nil {
		return nil, fmt.Errorf("MCP tests require a *registry.Registry on TestContext")
	}

	cfg := &config.MCPConfig{Host: "localhost", Port: 0}

	var mcpOpts []mcpkg.Option
	if st, ok := tc.StoredValues["_storage"].(storage.Storage); ok {
		authSvc := auth.NewServiceWithConfig(st, auth.ServiceConfig{})
		mcpOpts = append(mcpOpts, mcpkg.WithAuthService(authSvc))
	}

	srv := mcpkg.New(cfg, reg, nil, "bdd-test", mcpOpts...)

	ctx := context.Background()
	ct, st := gomcp.NewInMemoryTransports()

	ss, err := srv.MCPServer().Connect(ctx, st, nil)
	if err != nil {
		return nil, fmt.Errorf("MCP server connect: %w", err)
	}

	client := gomcp.NewClient(&gomcp.Implementation{Name: "bdd-client", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		ss.Close()
		return nil, fmt.Errorf("MCP client connect: %w", err)
	}

	tc.StoredValues["_mcp_state"] = &mcpState{session: cs, ss: ss}
	return cs, nil
}

// closeMCPSession cleans up the MCP session if one was created.
func closeMCPSession(tc *TestContext) {
	if s, ok := tc.StoredValues["_mcp_state"].(*mcpState); ok {
		if s.session != nil {
			s.session.Close()
		}
		if s.ss != nil {
			s.ss.Close()
		}
		delete(tc.StoredValues, "_mcp_state")
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
			return nil
		}
		tc.MCPError = nil
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
			return nil
		}
		tc.MCPError = nil
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

		var args map[string]any
		if err := json.Unmarshal([]byte(body.Content), &args); err != nil {
			return fmt.Errorf("invalid JSON input: %w", err)
		}

		result, err := cs.CallTool(context.Background(), &gomcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			tc.MCPError = err
			tc.MCPResultText = ""
			return nil
		}
		tc.MCPError = nil
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
