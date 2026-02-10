//go:build bdd

package steps

import (
	"encoding/json"
	"fmt"

	"github.com/cucumber/godog"
)

// RegisterImportSteps registers import/migration step definitions.
func RegisterImportSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	ctx.Step(`^I import schemas:$`, func(body *godog.DocString) error {
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON in import body: %w", err)
		}
		return tc.POST("/import/schemas", req)
	})

	ctx.Step(`^I import a schema with ID (\d+) under subject "([^"]*)" version (\d+):$`, func(id int, subject string, version int, schema *godog.DocString) error {
		body := map[string]interface{}{
			"schemas": []map[string]interface{}{
				{
					"id":      id,
					"subject": subject,
					"version": version,
					"schema":  schema.Content,
				},
			},
		}
		return tc.POST("/import/schemas", body)
	})

	ctx.Step(`^I import a "([^"]*)" schema with ID (\d+) under subject "([^"]*)" version (\d+):$`, func(schemaType string, id int, subject string, version int, schema *godog.DocString) error {
		body := map[string]interface{}{
			"schemas": []map[string]interface{}{
				{
					"id":         id,
					"subject":    subject,
					"version":    version,
					"schemaType": schemaType,
					"schema":     schema.Content,
				},
			},
		}
		return tc.POST("/import/schemas", body)
	})

	ctx.Step(`^the import should have (\d+) imported and (\d+) errors$`, func(imported, errors int) error {
		importedVal, err := tc.JSONFieldInt("imported")
		if err != nil {
			return err
		}
		errorsVal, err2 := tc.JSONFieldInt("errors")
		if err2 != nil {
			return err2
		}
		if importedVal != imported {
			return fmt.Errorf("expected %d imported, got %d", imported, importedVal)
		}
		if errorsVal != errors {
			return fmt.Errorf("expected %d errors, got %d", errors, errorsVal)
		}
		return nil
	})
}
