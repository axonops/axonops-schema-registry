//go:build bdd

package steps

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/cucumber/godog"
)

// RegisterRateLimitSteps registers rate limiting step definitions.
func RegisterRateLimitSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	ctx.Step(`^I send (\d+) rapid requests to "([^"]*)"$`, func(count int, path string) error {
		tc.StoredValues["_rapid_statuses"] = sendRapidRequests(tc, count, path)
		return nil
	})

	ctx.Step(`^at least one response should have status (\d+)$`, func(expected int) error {
		val, ok := tc.StoredValues["_rapid_statuses"]
		if !ok {
			return fmt.Errorf("no rapid request statuses stored; call 'I send N rapid requests' first")
		}
		statuses, ok := val.([]int)
		if !ok {
			return fmt.Errorf("stored rapid statuses is not []int")
		}
		for _, s := range statuses {
			if s == expected {
				return nil
			}
		}
		return fmt.Errorf("none of the %d responses had status %d; statuses: %v", len(statuses), expected, statuses)
	})
}

// sendRapidRequests fires count sequential GET requests as fast as possible and returns all status codes.
func sendRapidRequests(tc *TestContext, count int, path string) []int {
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	url := tc.BaseURL + path

	statuses := make([]int, 0, count)
	for i := 0; i < count; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			statuses = append(statuses, 0)
			continue
		}
		req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")
		if tc.AuthHeader != "" {
			req.Header.Set("Authorization", tc.AuthHeader)
		}
		resp, err := client.Do(req)
		if err != nil {
			statuses = append(statuses, 0)
			continue
		}
		statuses = append(statuses, resp.StatusCode)
		resp.Body.Close()
	}
	return statuses
}
