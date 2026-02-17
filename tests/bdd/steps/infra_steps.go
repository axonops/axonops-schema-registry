//go:build bdd

package steps

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cucumber/godog"
)

// RegisterInfraSteps registers infrastructure control step definitions.
// The webhook runs inside the schema registry container and controls the registry process.
func RegisterInfraSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	ctx.Step(`^I kill the database container$`, func() error {
		return tc.webhookAction("kill-backend")
	})

	ctx.Step(`^I restart the schema registry$`, func() error {
		return tc.webhookAction("restart-service")
	})

	ctx.Step(`^I restart the database container$`, func() error {
		return tc.webhookAction("restart-backend")
	})

	ctx.Step(`^I pause the database$`, func() error {
		return tc.webhookAction("pause-service")
	})

	ctx.Step(`^I unpause the database$`, func() error {
		return tc.webhookAction("unpause-service")
	})

	ctx.Step(`^I stop the schema registry$`, func() error {
		return tc.webhookAction("stop-service")
	})

	ctx.Step(`^I start the schema registry$`, func() error {
		return tc.webhookAction("start-service")
	})

	ctx.Step(`^I wait for the registry to become healthy$`, func() error {
		return tc.waitForHealthy(30 * time.Second)
	})

	ctx.Step(`^I wait for the registry to become unhealthy$`, func() error {
		return tc.waitForUnhealthy(15 * time.Second)
	})

	ctx.Step(`^I wait (\d+) seconds$`, func(n int) error {
		time.Sleep(time.Duration(n) * time.Second)
		return nil
	})

	ctx.Step(`^a running schema registry with (\w+) backend$`, func(backend string) error {
		// Just verify health
		return tc.GET("/")
	})

	ctx.Step(`^I have registered (\d+) schemas across multiple subjects$`, func(count int) error {
		for i := 0; i < count; i++ {
			subject := fmt.Sprintf("test-subject-%d", i)
			schema := fmt.Sprintf(`{"type":"record","name":"Test%d","fields":[{"name":"f","type":"string"}]}`, i)
			body := map[string]interface{}{"schema": schema}
			if err := tc.POST("/subjects/"+subject+"/versions", body); err != nil {
				return err
			}
			if tc.LastStatusCode != 200 {
				return fmt.Errorf("failed to register schema %d: %d %s", i, tc.LastStatusCode, string(tc.LastBody))
			}
		}
		return nil
	})

	ctx.Step(`^I have registered schemas under subjects "([^"]*)" and "([^"]*)"$`, func(s1, s2 string) error {
		schema1 := `{"type":"record","name":"Schema1","fields":[{"name":"f","type":"string"}]}`
		if err := tc.POST("/subjects/"+s1+"/versions", map[string]interface{}{"schema": schema1}); err != nil {
			return err
		}
		schema2 := `{"type":"record","name":"Schema2","fields":[{"name":"f","type":"string"}]}`
		return tc.POST("/subjects/"+s2+"/versions", map[string]interface{}{"schema": schema2})
	})
}

// webhookAction sends a POST to the webhook inside the schema registry container.
func (tc *TestContext) webhookAction(hook string) error {
	if tc.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured (set BDD_WEBHOOK_URL)")
	}
	resp, err := tc.client.Post(tc.WebhookURL+"/hooks/"+hook, "application/json", nil)
	if err != nil {
		return fmt.Errorf("webhook %s: %w", hook, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook %s returned %d", hook, resp.StatusCode)
	}
	return nil
}

// waitForHealthy polls the health endpoint until it returns 200.
func (tc *TestContext) waitForHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := tc.GET("/")
		if err == nil && tc.LastStatusCode == 200 {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("registry did not become healthy within %s", timeout)
}

// waitForUnhealthy polls the health endpoint until it fails or returns non-200.
func (tc *TestContext) waitForUnhealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := tc.GET("/")
		if err != nil || tc.LastStatusCode != 200 {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("registry is still healthy after %s", timeout)
}
