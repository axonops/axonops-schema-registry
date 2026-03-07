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
		Description: "Get a high-level summary of key Prometheus metrics including request rates, schema counts, error rates, and Confluent-compatible counters. Returns structured data suitable for health dashboards.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_metrics_summary", s.handleGetMetricsSummary))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "get_confluent_metrics",
		Description: "Get Confluent Schema Registry-compatible metrics (kafka_schema_registry_* prefix). These metrics match what the Confluent JMX exporter produces, enabling existing Grafana dashboards to work without changes.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "get_confluent_metrics", s.handleGetConfluentMetrics))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "query_metric",
		Description: "Query a specific Prometheus metric by name. Returns the current value(s) including all label combinations. Supports partial name matching.",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "query_metric", s.handleQueryMetric))

	addToolIfAllowed(s, &gomcp.Tool{
		Name:        "list_metrics",
		Description: "List all available Prometheus metric names grouped by category (request, schema, compatibility, storage, cache, auth, mcp, confluent, runtime).",
		Annotations: &gomcp.ToolAnnotations{ReadOnlyHint: true},
	}, instrumentedHandler(s, "list_metrics", s.handleListMetrics))
}

type metricsSummaryInput struct{}

func (s *Server) handleGetMetricsSummary(_ context.Context, _ *gomcp.CallToolRequest, _ metricsSummaryInput) (*gomcp.CallToolResult, any, error) {
	metricsText := s.scrapeMetrics()
	lines := parseMetricLines(metricsText)

	var sb strings.Builder
	sb.WriteString("# Schema Registry Metrics Summary\n\n")

	sb.WriteString("## Confluent-Compatible Counters\n")
	writeMetricValue(&sb, lines, "kafka_schema_registry_registered_count", "Schemas registered (total)")
	writeMetricValue(&sb, lines, "kafka_schema_registry_deleted_count", "Schemas deleted (total)")
	writeMetricValue(&sb, lines, "kafka_schema_registry_api_success_count", "Successful API calls")
	writeMetricValue(&sb, lines, "kafka_schema_registry_api_failure_count", "Failed API calls")
	writeMetricValue(&sb, lines, "kafka_schema_registry_master_slave_role", "Leader role (1=leader)")
	writeMetricValue(&sb, lines, "kafka_schema_registry_node_count", "Node count")
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

	sb.WriteString("## MCP Metrics\n")
	writeMetricValue(&sb, lines, "schema_registry_mcp_tool_calls_active", "Active MCP calls")
	writeMatchingMetrics(&sb, lines, "schema_registry_mcp_tool_calls_total")

	return textResult(sb.String())
}

type confluentMetricsInput struct{}

func (s *Server) handleGetConfluentMetrics(_ context.Context, _ *gomcp.CallToolRequest, _ confluentMetricsInput) (*gomcp.CallToolResult, any, error) {
	metricsText := s.scrapeMetrics()
	lines := parseMetricLines(metricsText)

	var sb strings.Builder
	sb.WriteString("# Confluent-Compatible Metrics (kafka_schema_registry_*)\n\n")
	sb.WriteString("These metrics match the Confluent Schema Registry JMX exporter output.\n")
	sb.WriteString("Existing Grafana dashboards querying these metric names will work without changes.\n\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "kafka_schema_registry_") || (strings.HasPrefix(line, "# ") && strings.Contains(line, "kafka_schema_registry_")) {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return textResult(sb.String())
}

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
		"Confluent-Compatible": {},
		"Request":              {},
		"Schema":               {},
		"Compatibility":        {},
		"Storage":              {},
		"Cache":                {},
		"Auth":                 {},
		"Rate Limit":           {},
		"MCP":                  {},
		"Principal":            {},
		"Go Runtime":           {},
		"Process":              {},
		"Other":                {},
	}

	for _, name := range names {
		switch {
		case strings.HasPrefix(name, "kafka_schema_registry_"):
			categories["Confluent-Compatible"] = append(categories["Confluent-Compatible"], name)
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
	sb.WriteString("# Available Prometheus Metrics\n\n")
	for _, cat := range []string{"Confluent-Compatible", "Request", "Schema", "Compatibility", "Storage", "Cache", "Auth", "Rate Limit", "MCP", "Principal", "Go Runtime", "Process", "Other"} {
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

// scrapeMetrics fetches the current metrics output from the Prometheus handler.
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

// parseMetricLines splits Prometheus text format into lines, trimming empty ones.
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
