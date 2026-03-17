//go:build bdd

package steps

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

// RegisterAuditOutputSteps registers step definitions for audit output testing.
// These steps verify that audit events are delivered to webhook receivers,
// syslog servers, and CEF-formatted file outputs.
func RegisterAuditOutputSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// Webhook receiver steps
	ctx.Step(`^the webhook receiver should have received an event with event_type "([^"]*)"$`, func(eventType string) error {
		return webhookReceiverHasEvent(tc, eventType)
	})

	ctx.Step(`^the webhook receiver should have received an event matching:$`, func(table *godog.Table) error {
		return webhookReceiverHasEventMatching(tc, table)
	})

	ctx.Step(`^the webhook receiver should have at least (\d+) events$`, func(count int) error {
		return webhookReceiverHasAtLeastEvents(tc, count)
	})

	// Syslog steps
	ctx.Step(`^the syslog receiver should have received a message containing "([^"]*)"$`, func(text string) error {
		return syslogReceiverHasMessage(tc, "_syslog_fetcher", text)
	})

	ctx.Step(`^the syslog TLS receiver should have received a message containing "([^"]*)"$`, func(text string) error {
		return syslogReceiverHasMessage(tc, "_syslog_tls_fetcher", text)
	})

	// CEF format steps
	ctx.Step(`^the audit CEF log should contain "([^"]*)"$`, func(text string) error {
		return auditCEFLogContains(tc, text)
	})
}

// webhookReceiverHasEvent checks that the webhook receiver has an event with the given event_type.
func webhookReceiverHasEvent(tc *TestContext, eventType string) error {
	webhookURL := getWebhookReceiverURL(tc)
	if webhookURL == "" {
		return fmt.Errorf("webhook receiver URL not configured (set _webhook_receiver_url in StoredValues)")
	}

	events, err := fetchWebhookEvents(webhookURL)
	if err != nil {
		return err
	}

	for _, raw := range events {
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			continue
		}
		if et, ok := event["event_type"].(string); ok && et == eventType {
			return nil
		}
	}

	return fmt.Errorf("webhook receiver has %d events but none with event_type=%q; events: %v", len(events), eventType, summarizeEvents(events))
}

// webhookReceiverHasEventMatching checks that the webhook receiver has an event matching all fields.
// It polls up to 10 times with 200ms delays to handle async webhook delivery.
func webhookReceiverHasEventMatching(tc *TestContext, table *godog.Table) error {
	webhookURL := getWebhookReceiverURL(tc)
	if webhookURL == "" {
		return fmt.Errorf("webhook receiver URL not configured")
	}

	expected := make(map[string]string)
	for _, row := range table.Rows {
		if len(row.Cells) >= 2 {
			expected[row.Cells[0].Value] = row.Cells[1].Value
		}
	}

	var lastEvents []string
	bestMatch := 0

	for attempt := range 10 {
		events, err := fetchWebhookEventsRaw(webhookURL)
		if err != nil {
			return err
		}
		lastEvents = events

		for _, raw := range events {
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &event); err != nil {
				continue
			}

			matched := 0
			for k, v := range expected {
				actual := fmt.Sprintf("%v", event[k])
				if k == "path" {
					if strings.Contains(actual, v) {
						matched++
					}
				} else if actual == v {
					matched++
				}
			}
			if matched == len(expected) {
				return nil
			}
			if matched > bestMatch {
				bestMatch = matched
			}
		}

		if attempt < 9 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	return fmt.Errorf("webhook receiver has %d events but none matching all %d fields (best partial match: %d/%d); events: %v",
		len(lastEvents), len(expected), bestMatch, len(expected), summarizeEvents(lastEvents))
}

// webhookReceiverHasAtLeastEvents checks that the webhook receiver has at least N events.
func webhookReceiverHasAtLeastEvents(tc *TestContext, count int) error {
	webhookURL := getWebhookReceiverURL(tc)
	if webhookURL == "" {
		return fmt.Errorf("webhook receiver URL not configured")
	}

	events, err := fetchWebhookEvents(webhookURL)
	if err != nil {
		return err
	}

	if len(events) < count {
		return fmt.Errorf("expected at least %d events, got %d", count, len(events))
	}
	return nil
}

// syslogReceiverHasMessage checks that the syslog receiver has a message containing the given text.
func syslogReceiverHasMessage(tc *TestContext, fetcherKey, text string) error {
	fetcherVal, ok := tc.StoredValues[fetcherKey]
	if !ok {
		return fmt.Errorf("syslog fetcher not configured (set %s in StoredValues)", fetcherKey)
	}
	fetcher, ok := fetcherVal.(func() (string, error))
	if !ok {
		return fmt.Errorf("_syslog_fetcher is not a function")
	}

	// Retry a few times since syslog delivery may be asynchronous
	var lastLog string
	for range 10 {
		log, err := fetcher()
		if err != nil {
			return fmt.Errorf("fetch syslog: %w", err)
		}
		lastLog = log
		if strings.Contains(log, text) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	if lastLog == "" {
		return fmt.Errorf("syslog log is empty, expected message containing %q", text)
	}
	return fmt.Errorf("syslog log does not contain %q; log content (last 500 chars): %s", text, truncateTail(lastLog, 500))
}

// auditCEFLogContains checks that the CEF audit log contains the given text.
func auditCEFLogContains(tc *TestContext, text string) error {
	fetcherVal, ok := tc.StoredValues["_cef_fetcher"]
	if !ok {
		// Fall back to regular audit fetcher — CEF may be in the same file
		fetcherVal, ok = tc.StoredValues["_audit_fetcher"]
		if !ok {
			return fmt.Errorf("CEF/audit fetcher not configured")
		}
	}
	fetcher, ok := fetcherVal.(func() (string, error))
	if !ok {
		return fmt.Errorf("fetcher is not a function")
	}

	// Retry a few times
	var lastLog string
	for range 5 {
		log, err := fetcher()
		if err != nil {
			return fmt.Errorf("fetch CEF log: %w", err)
		}
		lastLog = log
		if strings.Contains(log, text) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	if lastLog == "" {
		return fmt.Errorf("CEF log is empty, expected content containing %q", text)
	}
	return fmt.Errorf("CEF log does not contain %q; log content (last 500 chars): %s", text, truncateTail(lastLog, 500))
}

// fetchWebhookEventsRaw fetches events from the webhook receiver without retrying.
// Returns whatever events are currently available (may be empty).
func fetchWebhookEventsRaw(baseURL string) ([]string, error) {
	resp, err := http.Get(baseURL + "/events")
	if err != nil {
		return nil, fmt.Errorf("GET %s/events: %w", baseURL, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GET %s/events returned %d: %s", baseURL, resp.StatusCode, string(body))
	}

	var events []string
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, fmt.Errorf("parse webhook events: %w (body: %s)", err, truncateTail(string(body), 200))
	}
	return events, nil
}

// fetchWebhookEvents fetches events from the webhook receiver, retrying until at least one event exists.
func fetchWebhookEvents(baseURL string) ([]string, error) {
	for attempt := range 10 {
		events, err := fetchWebhookEventsRaw(baseURL)
		if err != nil {
			if attempt < 9 {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			return nil, err
		}
		if len(events) > 0 {
			return events, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, nil
}

// getWebhookReceiverURL returns the webhook receiver URL from StoredValues.
func getWebhookReceiverURL(tc *TestContext) string {
	if v, ok := tc.StoredValues["_webhook_receiver_url"]; ok {
		if url, ok := v.(string); ok {
			return url
		}
	}
	return ""
}

// summarizeEvents returns a brief summary of event types for error messages.
func summarizeEvents(events []string) string {
	var types []string
	for _, raw := range events {
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			types = append(types, "(parse error)")
			continue
		}
		if et, ok := event["event_type"].(string); ok {
			types = append(types, et)
		} else {
			types = append(types, "(no event_type)")
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(types, ", "))
}

// truncateTail returns at most n characters from the end of s.
func truncateTail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
