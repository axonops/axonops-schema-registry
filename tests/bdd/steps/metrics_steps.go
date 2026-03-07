//go:build bdd

package steps

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
)

// RegisterMetricsSteps registers step definitions for Prometheus metric assertions.
func RegisterMetricsSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	ctx.Step(`^the metric "([^"]*)" should exist$`, func(metricName string) error {
		return tc.GET("/metrics")
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" should exist$`, func(metricName string) error {
		if err := tc.GET("/metrics"); err != nil {
			return err
		}
		body := string(tc.LastBody)
		if !hasMetric(body, metricName) {
			return fmt.Errorf("metric %q not found in /metrics output (first 1000 chars: %s)", metricName, truncateStr(body, 1000))
		}
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" should not exist$`, func(metricName string) error {
		if err := tc.GET("/metrics"); err != nil {
			return err
		}
		body := string(tc.LastBody)
		if hasMetric(body, metricName) {
			return fmt.Errorf("metric %q unexpectedly found in /metrics output", metricName)
		}
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" should have value >= (\d+)$`, func(metricName string, minVal int) error {
		if err := tc.GET("/metrics"); err != nil {
			return err
		}
		body := string(tc.LastBody)
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
		if err := tc.GET("/metrics"); err != nil {
			return err
		}
		body := string(tc.LastBody)
		val, err := getMetricValue(body, metricName)
		if err != nil {
			return err
		}
		if int(val) != expected {
			return fmt.Errorf("metric %q value %.0f != %d", metricName, val, expected)
		}
		return nil
	})

	ctx.Step(`^the Prometheus metric "([^"]*)" with labels "([^"]*)" should exist$`, func(metricName, labels string) error {
		if err := tc.GET("/metrics"); err != nil {
			return err
		}
		body := string(tc.LastBody)
		search := metricName + "{" + labels
		if !strings.Contains(body, search) {
			return fmt.Errorf("metric %s{%s...} not found in /metrics output", metricName, labels)
		}
		return nil
	})

	ctx.Step(`^I store the current value of metric "([^"]*)" as "([^"]*)"$`, func(metricName, storageKey string) error {
		if err := tc.GET("/metrics"); err != nil {
			return err
		}
		body := string(tc.LastBody)
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
		if err := tc.GET("/metrics"); err != nil {
			return err
		}
		body := string(tc.LastBody)
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
