//go:build bdd

package steps

import (
	"github.com/cucumber/godog"
)

// RegisterModeSteps registers mode management step definitions.
func RegisterModeSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// --- Given steps ---
	ctx.Step(`^the global mode is "([^"]*)"$`, func(mode string) error {
		body := map[string]interface{}{"mode": mode}
		return tc.PUT("/mode?force=true", body)
	})

	ctx.Step(`^subject "([^"]*)" has mode "([^"]*)"$`, func(subject, mode string) error {
		body := map[string]interface{}{"mode": mode}
		return tc.PUT("/mode/"+subject+"?force=true", body)
	})

	// --- When steps ---
	ctx.Step(`^I get the mode for subject "([^"]*)"$`, func(subject string) error {
		return tc.GET("/mode/" + subject)
	})

	ctx.Step(`^I set the mode for subject "([^"]*)" to "([^"]*)"$`, func(subject, mode string) error {
		body := map[string]interface{}{"mode": mode}
		return tc.PUT("/mode/"+subject+"?force=true", body)
	})

	ctx.Step(`^I delete the mode for subject "([^"]*)"$`, func(subject string) error {
		return tc.DELETE("/mode/" + subject)
	})

	ctx.Step(`^I delete the global config$`, func() error {
		return tc.DELETE("/config")
	})
}
