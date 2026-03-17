package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerMetricsTools() {
	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_metrics_summary",
		Description: "Get a health-oriented summary of all key schema registry metrics across every category: request rates, schema counts, error rates, compatibility checks, storage health, cache performance, authentication, rate limiting, MCP operations, and wire-compatible counters.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_metrics_summary", s.handleGetMetricsSummary))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_metrics_by_category",
		Description: "Get all metrics for a specific category. Valid categories: request, schema, compatibility, storage, cache, auth, rate_limit, mcp, principal, wire_compatible, runtime. Returns current values with all label combinations.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_metrics_by_category", s.handleGetMetricsByCategory))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "query_metric",
		Description: "Query a specific metric by name or partial name. Returns current value(s) including all label combinations. Use this to inspect a single metric in detail.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "query_metric", s.handleQueryMetric))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "list_metrics",
		Description: "List all available metric names grouped by category (request, schema, compatibility, storage, cache, auth, rate_limit, mcp, principal, wire_compatible, runtime, process).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_metrics", s.handleListMetrics))
}

// --- get_metrics_summary ---

type metricsSummaryInput struct{}

func (s *Server) handleGetMetricsSummary(_ context.Context, _ *gomcp.CallToolRequest, _ metricsSummaryInput) (*gomcp.CallToolResult, any, error) {
	metricsText := s.scrapeMetrics()
	lines := parseMetricLines(metricsText)

	var sb strings.Builder
	sb.WriteString("# Schema Registry Metrics Summary\n\n")

	sb.WriteString("## Wire-Compatible Counters\n")
	sb.WriteString("These counters use the `kafka_schema_registry_` prefix for Grafana dashboard compatibility.\n")
	writeMetricValue(&sb, lines, "kafka_schema_registry_registered_count", "Schemas registered (total)")
	writeMetricValue(&sb, lines, "kafka_schema_registry_deleted_count", "Schemas deleted (total)")
	writeMetricValue(&sb, lines, "kafka_schema_registry_api_success_count", "Successful API calls (2xx/3xx)")
	writeMetricValue(&sb, lines, "kafka_schema_registry_api_failure_count", "Failed API calls (4xx/5xx)")
	writeMetricValue(&sb, lines, "kafka_schema_registry_master_slave_role", "Leader role (1=leader, 0=follower)")
	writeMetricValue(&sb, lines, "kafka_schema_registry_node_count", "Cluster node count")
	sb.WriteString("\n")

	sb.WriteString("## Schema Counts by Type\n")
	writeMatchingMetrics(&sb, lines, "kafka_schema_registry_schemas_created")
	writeMatchingMetrics(&sb, lines, "kafka_schema_registry_schemas_deleted")
	sb.WriteString("\n")

	sb.WriteString("## Request Metrics\n")
	writeMetricValue(&sb, lines, "schema_registry_requests_in_flight", "Requests in flight")
	writeMatchingMetrics(&sb, lines, "schema_registry_requests_total")
	sb.WriteString("\n")

	sb.WriteString("## Schema Metrics\n")
	writeMatchingMetrics(&sb, lines, "schema_registry_schemas_total")
	writeMetricValue(&sb, lines, "schema_registry_subjects_total", "Total subjects")
	writeMatchingMetrics(&sb, lines, "schema_registry_registrations_total")
	sb.WriteString("\n")

	sb.WriteString("## Compatibility Metrics\n")
	writeMatchingMetrics(&sb, lines, "schema_registry_compatibility_checks_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_compatibility_errors_total")
	sb.WriteString("\n")

	sb.WriteString("## Storage Metrics\n")
	writeMatchingMetrics(&sb, lines, "schema_registry_storage_operations_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_storage_errors_total")
	sb.WriteString("\n")

	sb.WriteString("## Cache Metrics\n")
	writeMatchingMetrics(&sb, lines, "schema_registry_cache_hits_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_cache_misses_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_cache_size")
	sb.WriteString("\n")

	sb.WriteString("## Auth Metrics\n")
	writeMatchingMetrics(&sb, lines, "schema_registry_auth_attempts_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_auth_failures_total")
	sb.WriteString("\n")

	sb.WriteString("## Rate Limit Metrics\n")
	writeMatchingMetrics(&sb, lines, "schema_registry_rate_limit_hits_total")
	sb.WriteString("\n")

	sb.WriteString("## MCP Metrics\n")
	writeMetricValue(&sb, lines, "schema_registry_mcp_tool_calls_active", "Active MCP tool calls")
	writeMatchingMetrics(&sb, lines, "schema_registry_mcp_tool_calls_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_mcp_tool_call_errors_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_mcp_confirmations_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_mcp_policy_denials_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_mcp_permission_denied_total")
	sb.WriteString("\n")

	sb.WriteString("## Per-Principal Metrics\n")
	writeMatchingMetrics(&sb, lines, "schema_registry_principal_requests_total")
	writeMatchingMetrics(&sb, lines, "schema_registry_principal_mcp_calls_total")

	return textResult(sb.String())
}

// --- get_metrics_by_category ---

type metricsByCategoryInput struct {
	Category string `json:"category"`
}

// categoryPrefixes maps category names to metric name prefixes.
var categoryPrefixes = map[string][]string{
	"request":         {"schema_registry_request"},
	"schema":          {"schema_registry_schema", "schema_registry_subject", "schema_registry_registration"},
	"compatibility":   {"schema_registry_compatibility"},
	"storage":         {"schema_registry_storage"},
	"cache":           {"schema_registry_cache"},
	"auth":            {"schema_registry_auth"},
	"rate_limit":      {"schema_registry_rate_limit"},
	"mcp":             {"schema_registry_mcp"},
	"principal":       {"schema_registry_principal"},
	"wire_compatible": {"kafka_schema_registry_"},
	"runtime":         {"go_"},
	"process":         {"process_"},
}

func (s *Server) handleGetMetricsByCategory(_ context.Context, _ *gomcp.CallToolRequest, input metricsByCategoryInput) (*gomcp.CallToolResult, any, error) {
	if input.Category == "" {
		cats := make([]string, 0, len(categoryPrefixes))
		for k := range categoryPrefixes {
			cats = append(cats, k)
		}
		sort.Strings(cats)
		return errorResult(fmt.Errorf("category is required. Valid categories: %s", strings.Join(cats, ", "))), nil, nil
	}

	prefixes, ok := categoryPrefixes[input.Category]
	if !ok {
		cats := make([]string, 0, len(categoryPrefixes))
		for k := range categoryPrefixes {
			cats = append(cats, k)
		}
		sort.Strings(cats)
		return errorResult(fmt.Errorf("unknown category %q. Valid categories: %s", input.Category, strings.Join(cats, ", "))), nil, nil
	}

	metricsText := s.scrapeMetrics()
	lines := parseMetricLines(metricsText)

	var matches []string
	for _, line := range lines {
		for _, prefix := range prefixes {
			if strings.HasPrefix(line, prefix) || (strings.HasPrefix(line, "# ") && strings.Contains(line, prefix)) {
				matches = append(matches, line)
				break
			}
		}
	}

	if len(matches) == 0 {
		return textResult(fmt.Sprintf("No metrics found for category %q. Metrics in this category may not have been initialized yet (counters appear after their first increment).", input.Category))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s Metrics\n\n", strings.ReplaceAll(strings.Title(input.Category), "_", " ")))
	for _, line := range matches {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return textResult(sb.String())
}

// --- query_metric ---

type queryMetricInput struct {
	Name string `json:"name"`
}

func (s *Server) handleQueryMetric(_ context.Context, _ *gomcp.CallToolRequest, input queryMetricInput) (*gomcp.CallToolResult, any, error) {
	if input.Name == "" {
		return errorResult(fmt.Errorf("metric name is required")), nil, nil
	}

	metricsText := s.scrapeMetrics()
	lines := parseMetricLines(metricsText)

	var matches []string
	for _, line := range lines {
		if strings.Contains(line, input.Name) {
			matches = append(matches, line)
		}
	}

	if len(matches) == 0 {
		return errorResult(fmt.Errorf("no metrics found matching %q", input.Name)), nil, nil
	}

	return textResult(strings.Join(matches, "\n"))
}

// --- list_metrics ---

type listMetricsInput struct{}

func (s *Server) handleListMetrics(_ context.Context, _ *gomcp.CallToolRequest, _ listMetricsInput) (*gomcp.CallToolResult, any, error) {
	metricsText := s.scrapeMetrics()
	lines := parseMetricLines(metricsText)

	nameSet := make(map[string]bool)
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		name := line
		if idx := strings.IndexAny(name, "{ "); idx >= 0 {
			name = name[:idx]
		}
		nameSet[name] = true
	}

	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	categories := map[string][]string{
		"Wire-Compatible": {},
		"Request":         {},
		"Schema":          {},
		"Compatibility":   {},
		"Storage":         {},
		"Cache":           {},
		"Auth":            {},
		"Rate Limit":      {},
		"MCP":             {},
		"Principal":       {},
		"Go Runtime":      {},
		"Process":         {},
		"Other":           {},
	}

	for _, name := range names {
		switch {
		case strings.HasPrefix(name, "kafka_schema_registry_"):
			categories["Wire-Compatible"] = append(categories["Wire-Compatible"], name)
		case strings.HasPrefix(name, "schema_registry_request"):
			categories["Request"] = append(categories["Request"], name)
		case strings.HasPrefix(name, "schema_registry_schema") || strings.HasPrefix(name, "schema_registry_subject") || strings.HasPrefix(name, "schema_registry_registration"):
			categories["Schema"] = append(categories["Schema"], name)
		case strings.HasPrefix(name, "schema_registry_compatibility"):
			categories["Compatibility"] = append(categories["Compatibility"], name)
		case strings.HasPrefix(name, "schema_registry_storage"):
			categories["Storage"] = append(categories["Storage"], name)
		case strings.HasPrefix(name, "schema_registry_cache"):
			categories["Cache"] = append(categories["Cache"], name)
		case strings.HasPrefix(name, "schema_registry_auth"):
			categories["Auth"] = append(categories["Auth"], name)
		case strings.HasPrefix(name, "schema_registry_rate_limit"):
			categories["Rate Limit"] = append(categories["Rate Limit"], name)
		case strings.HasPrefix(name, "schema_registry_mcp"):
			categories["MCP"] = append(categories["MCP"], name)
		case strings.HasPrefix(name, "schema_registry_principal"):
			categories["Principal"] = append(categories["Principal"], name)
		case strings.HasPrefix(name, "go_"):
			categories["Go Runtime"] = append(categories["Go Runtime"], name)
		case strings.HasPrefix(name, "process_"):
			categories["Process"] = append(categories["Process"], name)
		default:
			categories["Other"] = append(categories["Other"], name)
		}
	}

	var sb strings.Builder
	sb.WriteString("# Available Metrics\n\n")
	for _, cat := range []string{"Wire-Compatible", "Request", "Schema", "Compatibility", "Storage", "Cache", "Auth", "Rate Limit", "MCP", "Principal", "Go Runtime", "Process", "Other"} {
		metricList := categories[cat]
		if len(metricList) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s (%d)\n", cat, len(metricList)))
		for _, m := range metricList {
			sb.WriteString(fmt.Sprintf("- %s\n", m))
		}
		sb.WriteString("\n")
	}

	return textResult(sb.String())
}

// scrapeMetrics fetches the current metrics output from the handler.
func (s *Server) scrapeMetrics() string {
	if s.metrics == nil {
		return ""
	}
	handler := s.metrics.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	body, _ := io.ReadAll(rr.Body)
	return string(body)
}

// parseMetricLines splits metrics text format into lines, trimming empty ones.
func parseMetricLines(text string) []string {
	raw := strings.Split(text, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// writeMetricValue writes a single metric's value to the builder.
func writeMetricValue(sb *strings.Builder, lines []string, name, desc string) {
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, name+" ") || strings.HasPrefix(line, name+"{") {
			sb.WriteString(fmt.Sprintf("- **%s**: %s = `%s`\n", desc, name, strings.TrimPrefix(line, name+" ")))
			return
		}
	}
	sb.WriteString(fmt.Sprintf("- **%s**: %s = (not yet initialized)\n", desc, name))
}

// writeMatchingMetrics writes all lines matching a metric name prefix.
func writeMatchingMetrics(sb *strings.Builder, lines []string, prefix string) {
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			sb.WriteString(fmt.Sprintf("- `%s`\n", line))
		}
	}
}
