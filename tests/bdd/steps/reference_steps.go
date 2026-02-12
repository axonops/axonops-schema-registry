//go:build bdd

package steps

import (
	"encoding/json"
	"fmt"

	"github.com/cucumber/godog"
)

// RegisterReferenceSteps registers schema reference step definitions.
func RegisterReferenceSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	ctx.Step(`^I register a schema under subject "([^"]*)" with references:$`, func(subject string, body *godog.DocString) error {
		var req map[string]interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		return tc.POST("/subjects/"+subject+"/versions", req)
	})

	ctx.Step(`^I register a "([^"]*)" schema under subject "([^"]*)" with references:$`, func(schemaType, subject string, body *godog.DocString) error {
		var req map[string]interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		req["schemaType"] = schemaType
		return tc.POST("/subjects/"+subject+"/versions", req)
	})

	ctx.Step(`^I get the referenced by for subject "([^"]*)" version (\d+)$`, func(subject string, version int) error {
		return tc.GET(fmt.Sprintf("/subjects/%s/versions/%d/referencedby", subject, version))
	})

	ctx.Step(`^I get the raw schema by ID (.+)$`, func(idStr string) error {
		resolved := tc.resolveVars(idStr)
		return tc.GET("/schemas/ids/" + resolved + "/schema")
	})

	ctx.Step(`^I get the raw schema for subject "([^"]*)" version (\d+)$`, func(subject string, version int) error {
		return tc.GET(fmt.Sprintf("/subjects/%s/versions/%d/schema", subject, version))
	})

	ctx.Step(`^I get versions for schema ID (.+)$`, func(idStr string) error {
		resolved := tc.resolveVars(idStr)
		return tc.GET("/schemas/ids/" + resolved + "/versions")
	})

	ctx.Step(`^I check compatibility of schema against all versions of subject "([^"]*)":$`, func(subject string, schema *godog.DocString) error {
		body := map[string]interface{}{"schema": schema.Content}
		return tc.POST(fmt.Sprintf("/compatibility/subjects/%s/versions", subject), body)
	})

	ctx.Step(`^I check compatibility of "([^"]*)" schema against all versions of subject "([^"]*)":$`, func(schemaType, subject string, schema *godog.DocString) error {
		body := map[string]interface{}{
			"schema":     schema.Content,
			"schemaType": schemaType,
		}
		return tc.POST(fmt.Sprintf("/compatibility/subjects/%s/versions", subject), body)
	})

	ctx.Step(`^I check compatibility of schema against subject "([^"]*)" version (\d+):$`, func(subject string, version int, schema *godog.DocString) error {
		body := map[string]interface{}{"schema": schema.Content}
		return tc.POST(fmt.Sprintf("/compatibility/subjects/%s/versions/%d", subject, version), body)
	})

	ctx.Step(`^I check compatibility of "([^"]*)" schema against subject "([^"]*)" version (\d+):$`, func(schemaType, subject string, version int, schema *godog.DocString) error {
		body := map[string]interface{}{
			"schema":     schema.Content,
			"schemaType": schemaType,
		}
		return tc.POST(fmt.Sprintf("/compatibility/subjects/%s/versions/%d", subject, version), body)
	})

	ctx.Step(`^I list subjects with deleted$`, func() error {
		return tc.GET("/subjects?deleted=true")
	})

	ctx.Step(`^I permanently delete version (\d+) of subject "([^"]*)"$`, func(version int, subject string) error {
		return tc.DoRequest("DELETE", fmt.Sprintf("/subjects/%s/versions/%d?permanent=true", subject, version), nil)
	})

	ctx.Step(`^I get the cluster ID$`, func() error {
		return tc.GET("/v1/metadata/id")
	})

	ctx.Step(`^I get the server version$`, func() error {
		return tc.GET("/v1/metadata/version")
	})

	ctx.Step(`^I get the contexts$`, func() error {
		return tc.GET("/contexts")
	})
}
