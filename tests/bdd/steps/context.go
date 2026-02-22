//go:build bdd

// Package steps provides godog step definitions for BDD tests.
package steps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TestContext holds state shared across steps within a single scenario.
type TestContext struct {
	BaseURL        string
	WebhookURL     string
	LastResponse   *http.Response
	LastBody       []byte
	LastStatusCode int
	LastJSON       map[string]interface{}
	LastJSONArray  []interface{}
	StoredValues   map[string]interface{} // for passing values between steps
	AuthHeader     string                 // Authorization header value (e.g., "Basic ...")
	client         *http.Client
}

// NewTestContext creates a fresh test context.
func NewTestContext(baseURL string) *TestContext {
	return &TestContext{
		BaseURL:      baseURL,
		StoredValues: make(map[string]interface{}),
		client:       &http.Client{Timeout: 5 * time.Second},
	}
}

// Reset clears per-scenario state.
func (tc *TestContext) Reset() {
	tc.LastResponse = nil
	tc.LastBody = nil
	tc.LastStatusCode = 0
	tc.LastJSON = nil
	tc.LastJSONArray = nil
	tc.StoredValues = make(map[string]interface{})
	tc.AuthHeader = ""
}

// resolveVars replaces {{key}} placeholders in a string with stored values.
func (tc *TestContext) resolveVars(s string) string {
	for key, val := range tc.StoredValues {
		placeholder := "{{" + key + "}}"
		s = strings.ReplaceAll(s, placeholder, fmt.Sprintf("%v", val))
	}
	return s
}

// DoRequest sends an HTTP request and stores the response.
func (tc *TestContext) DoRequest(method, path string, body interface{}) error {
	path = tc.resolveVars(path)
	url := tc.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")
	if tc.AuthHeader != "" {
		req.Header.Set("Authorization", tc.AuthHeader)
	}

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	tc.LastResponse = resp
	tc.LastStatusCode = resp.StatusCode
	tc.LastBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	// Try to parse as JSON
	tc.LastJSON = nil
	tc.LastJSONArray = nil
	if len(tc.LastBody) > 0 {
		if tc.LastBody[0] == '{' {
			var obj map[string]interface{}
			if err := json.Unmarshal(tc.LastBody, &obj); err == nil {
				tc.LastJSON = obj
			}
		} else if tc.LastBody[0] == '[' {
			var arr []interface{}
			if err := json.Unmarshal(tc.LastBody, &arr); err == nil {
				tc.LastJSONArray = arr
			}
		}
	}

	return nil
}

// GET sends a GET request.
func (tc *TestContext) GET(path string) error {
	return tc.DoRequest("GET", path, nil)
}

// POST sends a POST request with JSON body.
func (tc *TestContext) POST(path string, body interface{}) error {
	return tc.DoRequest("POST", path, body)
}

// PUT sends a PUT request with JSON body.
func (tc *TestContext) PUT(path string, body interface{}) error {
	return tc.DoRequest("PUT", path, body)
}

// DELETE sends a DELETE request.
func (tc *TestContext) DELETE(path string) error {
	return tc.DoRequest("DELETE", path, nil)
}

// DoRawRequest sends an HTTP request with a raw string body (not JSON-marshaled).
func (tc *TestContext) DoRawRequest(method, path string, body string) error {
	path = tc.resolveVars(path)
	url := tc.BaseURL + path

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")
	if tc.AuthHeader != "" {
		req.Header.Set("Authorization", tc.AuthHeader)
	}

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	tc.LastResponse = resp
	tc.LastStatusCode = resp.StatusCode
	tc.LastBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	// Try to parse as JSON
	tc.LastJSON = nil
	tc.LastJSONArray = nil
	if len(tc.LastBody) > 0 {
		if tc.LastBody[0] == '{' {
			var obj map[string]interface{}
			if err := json.Unmarshal(tc.LastBody, &obj); err == nil {
				tc.LastJSON = obj
			}
		} else if tc.LastBody[0] == '[' {
			var arr []interface{}
			if err := json.Unmarshal(tc.LastBody, &arr); err == nil {
				tc.LastJSONArray = arr
			}
		}
	}

	return nil
}

// JSONField extracts a field from the last JSON response.
func (tc *TestContext) JSONField(key string) (interface{}, error) {
	if tc.LastJSON == nil {
		return nil, fmt.Errorf("no JSON object in last response")
	}
	val, ok := tc.LastJSON[key]
	if !ok {
		return nil, fmt.Errorf("field %q not found in response: %s", key, string(tc.LastBody))
	}
	return val, nil
}

// JSONFieldInt extracts an integer field from the last JSON response.
func (tc *TestContext) JSONFieldInt(key string) (int, error) {
	val, err := tc.JSONField(key)
	if err != nil {
		return 0, err
	}
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case json.Number:
		n, err := v.Int64()
		return int(n), err
	default:
		return 0, fmt.Errorf("field %q is not a number: %T", key, val)
	}
}

// JSONFieldString extracts a string field from the last JSON response.
func (tc *TestContext) JSONFieldString(key string) (string, error) {
	val, err := tc.JSONField(key)
	if err != nil {
		return "", err
	}
	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("field %q is not a string: %T", key, val)
	}
	return s, nil
}

// ReplacePlaceholders replaces {stored_key} placeholders in strings with stored values.
func (tc *TestContext) ReplacePlaceholders(s string) string {
	for key, val := range tc.StoredValues {
		placeholder := "{" + key + "}"
		switch v := val.(type) {
		case string:
			s = strings.ReplaceAll(s, placeholder, v)
		case int:
			s = strings.ReplaceAll(s, placeholder, strconv.Itoa(v))
		case float64:
			s = strings.ReplaceAll(s, placeholder, strconv.Itoa(int(v)))
		}
	}
	return s
}
