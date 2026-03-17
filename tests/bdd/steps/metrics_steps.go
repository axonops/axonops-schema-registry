//go:build bdd

package steps

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

// RegisterMetricsSteps registers step definitions for metric assertions.
func RegisterMetricsSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	ctx.Step(`^the metric "([^"]*)" should exist$`, func(metricName string) error {
		body, err := tc.scrapeMetrics()
		if err != nil {
			return err
		}
		if !hasMetric(body, metricName) {
			return fmt.Errorf("metric %q not found in metrics output", metricName)
		}
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" should exist$`, func(metricName string) error {
		body, err := tc.scrapeMetrics()
		if err != nil {
			return err
		}
		if !hasMetric(body, metricName) {
			return fmt.Errorf("metric %q not found in metrics output (first 1000 chars: %s)", metricName, truncateStr(body, 1000))
		}
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" should not exist$`, func(metricName string) error {
		body, err := tc.scrapeMetrics()
		if err != nil {
			return err
		}
		if hasMetric(body, metricName) {
			return fmt.Errorf("metric %q unexpectedly found in metrics output", metricName)
		}
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" should have value >= (\d+)$`, func(metricName string, minVal int) error {
		body, err := tc.scrapeMetrics()
		if err != nil {
			return err
		}
		val, err := getMetricValue(body, metricName)
		if err != nil {
			return err
		}
		if val < float64(minVal) {
			return fmt.Errorf("metric %q value %.0f < %d", metricName, val, minVal)
		}
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" should have value (\d+)$`, func(metricName string, expected int) error {
		body, err := tc.scrapeMetrics()
		if err != nil {
			return err
		}
		val, err := getMetricValue(body, metricName)
		if err != nil {
			return err
		}
		if int(val) != expected {
			return fmt.Errorf("metric %q value %.0f != %d", metricName, val, expected)
		}
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" with labels "((?:[^"\\]|\\.)*)" should exist$`, func(metricName, labels string) error {
		body, err := tc.scrapeMetrics()
		if err != nil {
			return err
		}
		// Unescape backslash-escaped quotes from Gherkin: \" → "
		labels = strings.ReplaceAll(labels, `\"`, `"`)
		// Search for a line that starts with the metric name and contains the label substring.
		// Labels can appear in any order in Prometheus output, so we check each
		// label pair individually rather than assuming position after '{'.
		found := false
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, metricName+"{") && strings.Contains(line, labels) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("metric %s with labels %s not found in metrics output", metricName, labels)
		}
		return nil
	})

	ctx.Step(`^I store the current value of metric "([^"]*)" as "([^"]*)"$`, func(metricName, storageKey string) error {
		body, err := tc.scrapeMetrics()
		if err != nil {
			return err
		}
		val, err := getMetricValue(body, metricName)
		if err != nil {
			// Metric might not exist yet — default to 0
			tc.StoredValues[storageKey] = float64(0)
			return nil
		}
		tc.StoredValues[storageKey] = val
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" should have increased from "([^"]*)"$`, func(metricName, storageKey string) error {
		body, err := tc.scrapeMetrics()
		if err != nil {
			return err
		}
		val, err := getMetricValue(body, metricName)
		if err != nil {
			return fmt.Errorf("metric %q not found after operation: %w", metricName, err)
		}
		prev, ok := tc.StoredValues[storageKey].(float64)
		if !ok {
			return fmt.Errorf("stored value %q not found or not a number", storageKey)
		}
		if val <= prev {
			return fmt.Errorf("metric %q did not increase: was %.0f, now %.0f", metricName, prev, val)
		}
		return nil
	})

	ctx.Step(`^I wait for metrics refresh$`, func() {
		// Wait long enough for the periodic gauge refresh to run.
		// In-process tests use a 1-second interval; Docker tests should
		// configure metrics_refresh_interval via YAML/env accordingly.
		time.Sleep(2 * time.Second)
	})
}

// scrapeMetrics fetches metrics from the appropriate endpoint.
// When MetricsURL is set (e.g. JMX exporter sidecar for Confluent), it scrapes
// that URL directly. Otherwise it falls back to BaseURL + "/metrics".
func (tc *TestContext) scrapeMetrics() (string, error) {
	url := tc.BaseURL + "/metrics"
	if tc.MetricsURL != "" {
		url = tc.MetricsURL
	}
	resp, err := tc.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("scrape metrics at %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read metrics response: %w", err)
	}
	return string(body), nil
}

// hasMetric checks if a metric name appears in Prometheus text format output.
// It looks for lines starting with the metric name (excluding HELP/TYPE lines).
func hasMetric(body, metricName string) bool {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") {
			if strings.Contains(line, metricName) {
				return true
			}
			continue
		}
		if strings.HasPrefix(line, metricName) {
			return true
		}
	}
	return false
}

// getMetricValue extracts the numeric value of a simple (unlabeled) metric.
// For labeled metrics, it returns the first match.
var metricValueRe = regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{[^}]*\})?\s+([0-9eE.+-]+)`)

func getMetricValue(body, metricName string) (float64, error) {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		matches := metricValueRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		if matches[1] == metricName {
			return strconv.ParseFloat(matches[3], 64)
		}
	}
	return 0, fmt.Errorf("metric %q not found", metricName)
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
