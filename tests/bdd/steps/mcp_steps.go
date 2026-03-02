//go:build bdd

package steps

import (
	"context"
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

	ctx.Step(`^the MCP result should contain "([^"]*)"$`, func(expected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		if !strings.Contains(tc.MCPResultText, expected) {
			return fmt.Errorf("expected MCP result to contain %q, got: %s", expected, tc.MCPResultText)
		}
		return nil
	})

	ctx.Step(`^the MCP result should not contain "([^"]*)"$`, func(unexpected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
		if strings.Contains(tc.MCPResultText, unexpected) {
			return fmt.Errorf("expected MCP result NOT to contain %q, got: %s", unexpected, tc.MCPResultText)
		}
		return nil
	})

	ctx.Step(`^the MCP result should be "([^"]*)"$`, func(expected string) error {
		if tc.MCPError != nil {
			return fmt.Errorf("MCP call failed: %v", tc.MCPError)
		}
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
}
