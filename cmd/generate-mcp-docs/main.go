// Command generate-mcp-docs creates an in-process MCP server, queries it via
// the MCP protocol (tools/list, resources/list, prompts/list), and outputs a
// markdown reference document to stdout.
//
// Usage:
//
//	go run ./cmd/generate-mcp-docs > docs/mcp-reference.md
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"sort"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/axonops/axonops-schema-registry/internal/auth"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/mcp"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	jsonschemaparser "github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

func main() {
	ctx := context.Background()

	store := memory.NewStore()
	defer store.Close()

	schemaReg := schema.NewRegistry()
	schemaReg.Register(avro.NewParser())
	schemaReg.Register(protobuf.NewParser())
	schemaReg.Register(jsonschemaparser.NewParser())

	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	reg := registry.New(store, schemaReg, compatChecker, "BACKWARD")

	// Create MCP server with auth service so admin tools are registered.
	authSvc := auth.NewService(store)
	cfg := &config.MCPConfig{Host: "localhost", Port: 0}
	srv := mcp.New(cfg, reg, slog.New(slog.NewTextHandler(io.Discard, nil)), "dev",
		mcp.WithAuthService(authSvc),
		mcp.WithBuildInfo("dev", "generated"),
		mcp.WithClusterID("doc-gen"),
	)

	// Connect via in-memory transport.
	ct, st := gomcp.NewInMemoryTransports()

	ss, err := srv.MCPServer().Connect(ctx, st, nil)
	if err != nil {
		log.Fatalf("server connect: %v", err)
	}
	defer ss.Close()

	client := gomcp.NewClient(&gomcp.Implementation{Name: "doc-gen", Version: "1.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		log.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	// Collect tools (handle pagination).
	var tools []*gomcp.Tool
	for tool, err := range cs.Tools(ctx, nil) {
		if err != nil {
			log.Fatalf("list tools: %v", err)
		}
		tools = append(tools, tool)
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

	// Collect resources.
	var resources []*gomcp.Resource
	for res, err := range cs.Resources(ctx, nil) {
		if err != nil {
			log.Fatalf("list resources: %v", err)
		}
		resources = append(resources, res)
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].URI < resources[j].URI })

	// Collect resource templates.
	var templates []*gomcp.ResourceTemplate
	for tmpl, err := range cs.ResourceTemplates(ctx, nil) {
		if err != nil {
			log.Fatalf("list resource templates: %v", err)
		}
		templates = append(templates, tmpl)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].URITemplate < templates[j].URITemplate })

	// Collect prompts.
	var prompts []*gomcp.Prompt
	for p, err := range cs.Prompts(ctx, nil) {
		if err != nil {
			log.Fatalf("list prompts: %v", err)
		}
		prompts = append(prompts, p)
	}
	sort.Slice(prompts, func(i, j int) bool { return prompts[i].Name < prompts[j].Name })

	// Generate markdown.
	w := os.Stdout
	fmt.Fprintln(w, "# MCP API Reference")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "> Auto-generated from the MCP server registration. Do not edit manually.")
	fmt.Fprintln(w, ">")
	fmt.Fprintln(w, "> Regenerate with: `go run ./cmd/generate-mcp-docs > docs/mcp-reference.md`")
	fmt.Fprintln(w)

	// Summary counts.
	readOnly := 0
	for _, t := range tools {
		if t.Annotations != nil && t.Annotations.ReadOnlyHint {
			readOnly++
		}
	}
	fmt.Fprintf(w, "**%d tools** (%d read-only, %d write) | **%d resources** (%d static, %d templated) | **%d prompts**\n\n",
		len(tools), readOnly, len(tools)-readOnly,
		len(resources)+len(templates), len(resources), len(templates),
		len(prompts))

	// Table of Contents
	fmt.Fprintln(w, "## Contents")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "- [Tools](#tools)")
	fmt.Fprintln(w, "- [Resources](#resources)")
	fmt.Fprintln(w, "- [Prompts](#prompts)")
	fmt.Fprintln(w)

	// Tools section.
	fmt.Fprintln(w, "---")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Tools")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "| # | Tool | Read-Only | Description |\n")
	fmt.Fprintf(w, "|---|------|-----------|-------------|\n")
	for i, t := range tools {
		ro := ""
		if t.Annotations != nil && t.Annotations.ReadOnlyHint {
			ro = "Yes"
		}
		desc := strings.ReplaceAll(t.Description, "\n", " ")
		if len(desc) > 120 {
			desc = desc[:117] + "..."
		}
		fmt.Fprintf(w, "| %d | `%s` | %s | %s |\n", i+1, t.Name, ro, desc)
	}
	fmt.Fprintln(w)

	// Detailed tool docs with input schemas.
	fmt.Fprintln(w, "### Tool Details")
	fmt.Fprintln(w)
	for _, t := range tools {
		fmt.Fprintf(w, "#### `%s`\n\n", t.Name)
		fmt.Fprintln(w, t.Description)
		fmt.Fprintln(w)

		if t.Annotations != nil {
			var annotations []string
			if t.Annotations.ReadOnlyHint {
				annotations = append(annotations, "read-only")
			}
			if t.Annotations.DestructiveHint != nil && *t.Annotations.DestructiveHint {
				annotations = append(annotations, "destructive")
			}
			if t.Annotations.IdempotentHint {
				annotations = append(annotations, "idempotent")
			}
			if len(annotations) > 0 {
				fmt.Fprintf(w, "**Annotations:** %s\n\n", strings.Join(annotations, ", "))
			}
		}

		// Extract input schema properties.
		if t.InputSchema != nil {
			schemaBytes, err := json.Marshal(t.InputSchema)
			if err == nil {
				var schemaMap map[string]interface{}
				if json.Unmarshal(schemaBytes, &schemaMap) == nil {
					if props, ok := schemaMap["properties"].(map[string]interface{}); ok && len(props) > 0 {
						required := map[string]bool{}
						if req, ok := schemaMap["required"].([]interface{}); ok {
							for _, r := range req {
								if s, ok := r.(string); ok {
									required[s] = true
								}
							}
						}

						// Sort property names for stable output.
						propNames := make([]string, 0, len(props))
						for name := range props {
							propNames = append(propNames, name)
						}
						sort.Strings(propNames)

						fmt.Fprintln(w, "**Parameters:**")
						fmt.Fprintln(w)
						fmt.Fprintln(w, "| Parameter | Type | Required | Description |")
						fmt.Fprintln(w, "|-----------|------|----------|-------------|")
						for _, name := range propNames {
							prop := props[name]
							propMap, ok := prop.(map[string]interface{})
							if !ok {
								continue
							}
							pType := fmt.Sprintf("%v", propMap["type"])
							pDesc := ""
							if d, ok := propMap["description"]; ok {
								pDesc = strings.ReplaceAll(fmt.Sprintf("%v", d), "\n", " ")
							}
							req := ""
							if required[name] {
								req = "Yes"
							}
							fmt.Fprintf(w, "| `%s` | %s | %s | %s |\n", name, pType, req, pDesc)
						}
						fmt.Fprintln(w)
					}
				}
			}
		}

		fmt.Fprintln(w, "---")
		fmt.Fprintln(w)
	}

	// Resources section.
	fmt.Fprintln(w, "## Resources")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Static Resources")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| URI | Name | Description |")
	fmt.Fprintln(w, "|-----|------|-------------|")
	for _, r := range resources {
		fmt.Fprintf(w, "| `%s` | `%s` | %s |\n", r.URI, r.Name, r.Description)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "### Resource Templates")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| URI Template | Name | Description |")
	fmt.Fprintln(w, "|-------------|------|-------------|")
	for _, r := range templates {
		fmt.Fprintf(w, "| `%s` | `%s` | %s |\n", r.URITemplate, r.Name, r.Description)
	}
	fmt.Fprintln(w)

	// Prompts section.
	fmt.Fprintln(w, "---")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Prompts")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Prompt | Description | Arguments |")
	fmt.Fprintln(w, "|--------|-------------|-----------|")
	for _, p := range prompts {
		var args []string
		for _, a := range p.Arguments {
			marker := ""
			if a.Required {
				marker = " (required)"
			}
			args = append(args, fmt.Sprintf("`%s`%s", a.Name, marker))
		}
		argStr := strings.Join(args, ", ")
		if argStr == "" {
			argStr = "—"
		}
		desc := strings.ReplaceAll(p.Description, "\n", " ")
		fmt.Fprintf(w, "| `%s` | %s | %s |\n", p.Name, desc, argStr)
	}
	fmt.Fprintln(w)

	// Detailed prompt docs.
	fmt.Fprintln(w, "### Prompt Details")
	fmt.Fprintln(w)
	for _, p := range prompts {
		fmt.Fprintf(w, "#### `%s`\n\n", p.Name)
		fmt.Fprintln(w, p.Description)
		fmt.Fprintln(w)
		if len(p.Arguments) > 0 {
			fmt.Fprintln(w, "**Arguments:**")
			fmt.Fprintln(w)
			fmt.Fprintln(w, "| Name | Required | Description |")
			fmt.Fprintln(w, "|------|----------|-------------|")
			for _, a := range p.Arguments {
				req := ""
				if a.Required {
					req = "Yes"
				}
				fmt.Fprintf(w, "| `%s` | %s | %s |\n", a.Name, req, a.Description)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, "---")
		fmt.Fprintln(w)
	}
}
