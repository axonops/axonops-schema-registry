//go:build bdd

package steps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
)

// concResult holds the outcome of a single concurrent HTTP request.
type concResult struct {
	StatusCode int
	SchemaID   int64
	Schema     string
	Error      string
}

// doConcRequest performs an HTTP request suitable for use inside a goroutine.
// It uses its own http.Client and does not touch TestContext fields.
func doConcRequest(client *http.Client, baseURL, method, path string, body interface{}) concResult {
	fullURL := baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return concResult{Error: fmt.Sprintf("marshal body: %v", err)}
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return concResult{Error: fmt.Sprintf("create request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")

	resp, err := client.Do(req)
	if err != nil {
		return concResult{Error: fmt.Sprintf("do request: %v", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return concResult{StatusCode: resp.StatusCode, Error: fmt.Sprintf("read body: %v", err)}
	}

	cr := concResult{StatusCode: resp.StatusCode}

	// Try to parse JSON response for id and schema fields.
	if len(respBody) > 0 && respBody[0] == '{' {
		var obj map[string]interface{}
		if json.Unmarshal(respBody, &obj) == nil {
			if id, ok := obj["id"].(float64); ok {
				cr.SchemaID = int64(id)
			}
			if s, ok := obj["schema"].(string); ok {
				cr.Schema = s
			}
		}
	}

	return cr
}

// RegisterConcurrencySteps registers step definitions for goroutine-based concurrency scenarios.
func RegisterConcurrencySteps(ctx *godog.ScenarioContext, tc *TestContext) {

	// --- When steps ---

	// N goroutines each register a unique Avro schema to separate subjects.
	ctx.Step(`^(\d+) goroutines each register a unique Avro schema to separate subjects$`, func(n int) error {
		client := &http.Client{Timeout: 10 * time.Second}
		results := make([]concResult, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(idx int) {
				defer wg.Done()
				subject := fmt.Sprintf("conc-unique-%d", idx)
				schema := fmt.Sprintf(`{"type":"record","name":"Rec%d","fields":[{"name":"f","type":"string"}]}`, idx)
				body := map[string]interface{}{"schema": schema}
				results[idx] = doConcRequest(client, tc.BaseURL, "POST", "/subjects/"+subject+"/versions", body)
			}(i)
		}
		wg.Wait()
		// Store results as []interface{} so StoredValues can hold them.
		stored := make([]interface{}, n)
		for i, r := range results {
			stored[i] = r
		}
		tc.StoredValues["conc_results"] = stored
		return nil
	})

	// N goroutines register the same Avro schema to a single subject.
	ctx.Step(`^(\d+) goroutines register the same Avro schema to subject "([^"]*)"$`, func(n int, subject string) error {
		client := &http.Client{Timeout: 10 * time.Second}
		schema := `{"type":"record","name":"Identical","fields":[{"name":"v","type":"string"}]}`
		body := map[string]interface{}{"schema": schema}
		results := make([]concResult, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(idx int) {
				defer wg.Done()
				results[idx] = doConcRequest(client, tc.BaseURL, "POST", "/subjects/"+subject+"/versions", body)
			}(i)
		}
		wg.Wait()
		stored := make([]interface{}, n)
		for i, r := range results {
			stored[i] = r
		}
		tc.StoredValues["conc_results"] = stored
		return nil
	})

	// N subjects each with one Avro schema (sequential setup via tc).
	ctx.Step(`^(\d+) subjects each with one Avro schema$`, func(n int) error {
		subjects := make([]string, n)
		for i := 0; i < n; i++ {
			subject := fmt.Sprintf("conc-del-%d", i)
			subjects[i] = subject
			schema := fmt.Sprintf(`{"type":"record","name":"Del%d","fields":[{"name":"f","type":"int"}]}`, i)
			body := map[string]interface{}{"schema": schema}
			if err := tc.POST("/subjects/"+subject+"/versions", body); err != nil {
				return fmt.Errorf("register subject %s: %w", subject, err)
			}
			if tc.LastStatusCode != 200 {
				return fmt.Errorf("expected 200 for subject %s, got %d: %s", subject, tc.LastStatusCode, string(tc.LastBody))
			}
		}
		// Store subject names for the delete step.
		stored := make([]interface{}, n)
		for i, s := range subjects {
			stored[i] = s
		}
		tc.StoredValues["conc_subjects"] = stored
		return nil
	})

	// N goroutines each soft-delete their own subject.
	ctx.Step(`^(\d+) goroutines each soft-delete their own subject$`, func(n int) error {
		subjectsRaw, ok := tc.StoredValues["conc_subjects"]
		if !ok {
			return fmt.Errorf("no stored conc_subjects")
		}
		subjects, ok := subjectsRaw.([]interface{})
		if !ok {
			return fmt.Errorf("conc_subjects is not []interface{}")
		}
		if len(subjects) < n {
			return fmt.Errorf("expected at least %d subjects, got %d", n, len(subjects))
		}
		client := &http.Client{Timeout: 10 * time.Second}
		results := make([]concResult, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(idx int) {
				defer wg.Done()
				subject := fmt.Sprintf("%v", subjects[idx])
				results[idx] = doConcRequest(client, tc.BaseURL, "DELETE", "/subjects/"+subject, nil)
			}(i)
		}
		wg.Wait()
		stored := make([]interface{}, n)
		for i, r := range results {
			stored[i] = r
		}
		tc.StoredValues["conc_results"] = stored
		return nil
	})

	// W writer goroutines add versions and R reader goroutines read latest from a subject.
	ctx.Step(`^(\d+) writer goroutines add versions and (\d+) reader goroutines read latest from subject "([^"]*)"$`, func(writers, readers int, subject string) error {
		client := &http.Client{Timeout: 10 * time.Second}
		total := writers + readers
		results := make([]concResult, total)
		var mu sync.Mutex
		var wg sync.WaitGroup

		iterations := 3

		wg.Add(writers)
		for w := 0; w < writers; w++ {
			go func(idx int) {
				defer wg.Done()
				var last concResult
				for iter := 0; iter < iterations; iter++ {
					schema := fmt.Sprintf(`{"type":"record","name":"W%dI%d","fields":[{"name":"f","type":"string"}]}`, idx, iter)
					body := map[string]interface{}{"schema": schema}
					last = doConcRequest(client, tc.BaseURL, "POST", "/subjects/"+subject+"/versions", body)
				}
				mu.Lock()
				results[idx] = last
				mu.Unlock()
			}(w)
		}

		wg.Add(readers)
		for r := 0; r < readers; r++ {
			go func(idx int) {
				defer wg.Done()
				var last concResult
				for iter := 0; iter < iterations; iter++ {
					last = doConcRequest(client, tc.BaseURL, "GET", "/subjects/"+subject+"/versions/latest", nil)
				}
				mu.Lock()
				results[writers+idx] = last
				mu.Unlock()
			}(r)
		}

		wg.Wait()
		stored := make([]interface{}, total)
		for i, r := range results {
			stored[i] = r
		}
		tc.StoredValues["conc_results"] = stored
		// Also store the writer/reader counts for later assertion steps.
		tc.StoredValues["conc_writers"] = writers
		tc.StoredValues["conc_readers"] = readers
		return nil
	})

	// N goroutines attempt to register schemas to a subject (used for READONLY mode test).
	ctx.Step(`^(\d+) goroutines attempt to register schemas to subject "([^"]*)"$`, func(n int, subject string) error {
		client := &http.Client{Timeout: 10 * time.Second}
		results := make([]concResult, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(idx int) {
				defer wg.Done()
				schema := fmt.Sprintf(`{"type":"record","name":"Blocked%d","fields":[{"name":"f","type":"int"}]}`, idx)
				body := map[string]interface{}{"schema": schema}
				results[idx] = doConcRequest(client, tc.BaseURL, "POST", "/subjects/"+subject+"/versions", body)
			}(i)
		}
		wg.Wait()
		stored := make([]interface{}, n)
		for i, r := range results {
			stored[i] = r
		}
		tc.StoredValues["conc_results"] = stored
		return nil
	})

	// --- Then steps ---

	// All concurrent results should succeed (status 200).
	ctx.Step(`^all concurrent results should succeed$`, func() error {
		results, err := getConcResults(tc)
		if err != nil {
			return err
		}
		for i, r := range results {
			if r.Error != "" {
				return fmt.Errorf("result[%d] had error: %s", i, r.Error)
			}
			if r.StatusCode != 200 {
				return fmt.Errorf("result[%d] expected status 200, got %d", i, r.StatusCode)
			}
		}
		return nil
	})

	// All returned schema IDs should be unique.
	ctx.Step(`^all returned schema IDs should be unique$`, func() error {
		results, err := getConcResults(tc)
		if err != nil {
			return err
		}
		seen := make(map[int64]int)
		for i, r := range results {
			if r.SchemaID == 0 {
				return fmt.Errorf("result[%d] has no schema ID", i)
			}
			if prev, ok := seen[r.SchemaID]; ok {
				return fmt.Errorf("duplicate schema ID %d in result[%d] and result[%d]", r.SchemaID, prev, i)
			}
			seen[r.SchemaID] = i
		}
		return nil
	})

	// All returned schema IDs should be identical.
	ctx.Step(`^all returned schema IDs should be identical$`, func() error {
		results, err := getConcResults(tc)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			return fmt.Errorf("no concurrent results")
		}
		first := results[0].SchemaID
		if first == 0 {
			return fmt.Errorf("result[0] has no schema ID")
		}
		for i, r := range results[1:] {
			if r.SchemaID != first {
				return fmt.Errorf("result[%d] schema ID %d differs from result[0] ID %d", i+1, r.SchemaID, first)
			}
		}
		return nil
	})

	// Subject should have exactly N versions.
	ctx.Step(`^subject "([^"]*)" should have exactly (\d+) versions?$`, func(subject string, expected int) error {
		if err := tc.GET("/subjects/" + subject + "/versions"); err != nil {
			return err
		}
		if tc.LastStatusCode != 200 {
			return fmt.Errorf("expected 200, got %d: %s", tc.LastStatusCode, string(tc.LastBody))
		}
		if tc.LastJSONArray == nil {
			return fmt.Errorf("response is not a JSON array: %s", string(tc.LastBody))
		}
		if len(tc.LastJSONArray) != expected {
			return fmt.Errorf("expected %d versions, got %d: %s", expected, len(tc.LastJSONArray), string(tc.LastBody))
		}
		return nil
	})

	// GET /subjects should return an empty array.
	ctx.Step(`^GET /subjects should return an empty array$`, func() error {
		if err := tc.GET("/subjects"); err != nil {
			return err
		}
		if tc.LastStatusCode != 200 {
			return fmt.Errorf("expected 200, got %d: %s", tc.LastStatusCode, string(tc.LastBody))
		}
		body := strings.TrimSpace(string(tc.LastBody))
		if body == "[]" || body == "null" || body == "" {
			return nil
		}
		if tc.LastJSONArray != nil && len(tc.LastJSONArray) == 0 {
			return nil
		}
		return fmt.Errorf("expected empty array, got: %s", body)
	})

	// No concurrent results should have a 500 status.
	ctx.Step(`^no concurrent results should have a 500 status$`, func() error {
		results, err := getConcResults(tc)
		if err != nil {
			return err
		}
		for i, r := range results {
			if r.StatusCode >= 500 {
				return fmt.Errorf("result[%d] has server error status %d", i, r.StatusCode)
			}
		}
		return nil
	})

	// All reader responses should contain a valid schema.
	ctx.Step(`^all reader responses should contain a valid schema$`, func() error {
		results, err := getConcResults(tc)
		if err != nil {
			return err
		}
		writersRaw, ok := tc.StoredValues["conc_writers"]
		if !ok {
			return fmt.Errorf("no stored conc_writers count")
		}
		writers, ok := writersRaw.(int)
		if !ok {
			return fmt.Errorf("conc_writers is not int")
		}
		// Reader results are stored after writer results.
		for i := writers; i < len(results); i++ {
			r := results[i]
			if r.StatusCode == 200 && r.Schema == "" {
				return fmt.Errorf("reader result[%d] has status 200 but empty schema", i)
			}
		}
		return nil
	})

	// All concurrent results should have a specific status code.
	ctx.Step(`^all concurrent results should have status (\d+)$`, func(expected int) error {
		results, err := getConcResults(tc)
		if err != nil {
			return err
		}
		for i, r := range results {
			if r.Error != "" {
				return fmt.Errorf("result[%d] had error: %s", i, r.Error)
			}
			if r.StatusCode != expected {
				return fmt.Errorf("result[%d] expected status %d, got %d", i, expected, r.StatusCode)
			}
		}
		return nil
	})
}

// getConcResults extracts the []concResult from tc.StoredValues["conc_results"].
func getConcResults(tc *TestContext) ([]concResult, error) {
	raw, ok := tc.StoredValues["conc_results"]
	if !ok {
		return nil, fmt.Errorf("no stored conc_results")
	}
	stored, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("conc_results is not []interface{}")
	}
	results := make([]concResult, len(stored))
	for i, v := range stored {
		r, ok := v.(concResult)
		if !ok {
			return nil, fmt.Errorf("conc_results[%d] is not concResult: %T", i, v)
		}
		results[i] = r
	}
	return results, nil
}
