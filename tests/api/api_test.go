//go:build api

// Package api provides API endpoint tests for the schema registry.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var baseURL = "http://localhost:8081"

func init() {
	if url := os.Getenv("SCHEMA_REGISTRY_URL"); url != "" {
		baseURL = url
	}
}

func doRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	return resp
}

func parseResponse(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, string(body))
	}
}

func expectStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status %d, got %d. Body: %s", expected, resp.StatusCode, string(body))
	}
}

// Health and metadata tests

func TestHealthEndpoint(t *testing.T) {
	resp := doRequest(t, "GET", "/", nil)
	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestSchemaTypesEndpoint(t *testing.T) {
	resp := doRequest(t, "GET", "/schemas/types", nil)
	expectStatus(t, resp, http.StatusOK)

	var types []string
	parseResponse(t, resp, &types)

	if len(types) == 0 {
		t.Error("Expected at least one schema type")
	}

	// Should include AVRO, PROTOBUF, JSON
	expectedTypes := map[string]bool{"AVRO": false, "PROTOBUF": false, "JSON": false}
	for _, schemaType := range types {
		if _, ok := expectedTypes[schemaType]; ok {
			expectedTypes[schemaType] = true
		}
	}

	for schemaType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected schema type %s not found", schemaType)
		}
	}
}

func TestContextsEndpoint(t *testing.T) {
	resp := doRequest(t, "GET", "/contexts", nil)
	expectStatus(t, resp, http.StatusOK)

	var contexts []string
	parseResponse(t, resp, &contexts)

	if len(contexts) == 0 {
		t.Error("Expected at least one context")
	}
}

func TestMetadataIDEndpoint(t *testing.T) {
	resp := doRequest(t, "GET", "/v1/metadata/id", nil)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if _, ok := result["id"]; !ok {
		t.Error("Expected 'id' field in response")
	}
}

func TestMetadataVersionEndpoint(t *testing.T) {
	resp := doRequest(t, "GET", "/v1/metadata/version", nil)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if _, ok := result["version"]; !ok {
		t.Error("Expected 'version' field in response")
	}
}

// Schema registration tests

func TestRegisterAvroSchema(t *testing.T) {
	subject := fmt.Sprintf("api-test-avro-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema":     `{"type":"record","name":"User","namespace":"com.example","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`,
		"schemaType": "AVRO",
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if _, ok := result["id"]; !ok {
		t.Error("Expected 'id' in response")
	}
}

func TestRegisterJsonSchema(t *testing.T) {
	subject := fmt.Sprintf("api-test-json-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema":     `{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}},"required":["id"]}`,
		"schemaType": "JSON",
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if _, ok := result["id"]; !ok {
		t.Error("Expected 'id' in response")
	}
}

func TestRegisterProtobufSchema(t *testing.T) {
	subject := fmt.Sprintf("api-test-proto-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema":     `syntax = "proto3"; message User { int32 id = 1; string name = 2; }`,
		"schemaType": "PROTOBUF",
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if _, ok := result["id"]; !ok {
		t.Error("Expected 'id' in response")
	}
}

func TestRegisterInvalidSchema(t *testing.T) {
	subject := fmt.Sprintf("api-test-invalid-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema":     `{invalid json}`,
		"schemaType": "AVRO",
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	// Should fail with 422 Unprocessable Entity
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 422 or 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// Schema retrieval tests

func TestGetSchemaByID(t *testing.T) {
	subject := fmt.Sprintf("api-test-get-id-%d", time.Now().UnixNano())

	// Register schema
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"GetByID","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var regResult map[string]interface{}
	parseResponse(t, resp, &regResult)

	schemaID := int(regResult["id"].(float64))

	// Get by ID
	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d", schemaID), nil)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if result["schema"] == nil {
		t.Error("Expected 'schema' in response")
	}
}

func TestGetRawSchemaByID(t *testing.T) {
	subject := fmt.Sprintf("api-test-raw-id-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"RawByID","fields":[{"name":"data","type":"string"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var regResult map[string]interface{}
	parseResponse(t, resp, &regResult)

	schemaID := int(regResult["id"].(float64))

	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d/schema", schemaID), nil)
	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestGetSubjectsBySchemaID(t *testing.T) {
	subject := fmt.Sprintf("api-test-subjects-id-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"SubjectsByID","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var regResult map[string]interface{}
	parseResponse(t, resp, &regResult)

	schemaID := int(regResult["id"].(float64))

	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d/subjects", schemaID), nil)
	expectStatus(t, resp, http.StatusOK)

	var subjects []string
	parseResponse(t, resp, &subjects)

	if len(subjects) == 0 {
		t.Error("Expected at least one subject")
	}
}

func TestGetVersionsBySchemaID(t *testing.T) {
	subject := fmt.Sprintf("api-test-versions-id-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"VersionsByID","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var regResult map[string]interface{}
	parseResponse(t, resp, &regResult)

	schemaID := int(regResult["id"].(float64))

	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d/versions", schemaID), nil)
	expectStatus(t, resp, http.StatusOK)

	var versions []map[string]interface{}
	parseResponse(t, resp, &versions)
}

func TestGetNonExistentSchema(t *testing.T) {
	resp := doRequest(t, "GET", "/schemas/ids/999999999", nil)
	expectStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// Subject tests

func TestListSubjects(t *testing.T) {
	// Create some subjects
	for i := 0; i < 3; i++ {
		subject := fmt.Sprintf("api-test-list-%d-%d", time.Now().UnixNano(), i)
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"List%d","fields":[{"name":"id","type":"int"}]}`, i),
		}
		resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
		resp.Body.Close()
	}

	resp := doRequest(t, "GET", "/subjects", nil)
	expectStatus(t, resp, http.StatusOK)

	var subjects []string
	parseResponse(t, resp, &subjects)

	if len(subjects) < 3 {
		t.Errorf("Expected at least 3 subjects, got %d", len(subjects))
	}
}

func TestListSchemas(t *testing.T) {
	resp := doRequest(t, "GET", "/schemas", nil)
	expectStatus(t, resp, http.StatusOK)

	var schemas []map[string]interface{}
	parseResponse(t, resp, &schemas)
}

func TestListSchemasWithFilters(t *testing.T) {
	// With subjectPrefix
	resp := doRequest(t, "GET", "/schemas?subjectPrefix=api-test", nil)
	expectStatus(t, resp, http.StatusOK)

	var schemas []map[string]interface{}
	parseResponse(t, resp, &schemas)

	// With latestOnly
	resp = doRequest(t, "GET", "/schemas?latestOnly=true", nil)
	expectStatus(t, resp, http.StatusOK)
	parseResponse(t, resp, &schemas)

	// With pagination
	resp = doRequest(t, "GET", "/schemas?offset=0&limit=10", nil)
	expectStatus(t, resp, http.StatusOK)
	parseResponse(t, resp, &schemas)
}

// Version tests

func TestGetVersions(t *testing.T) {
	subject := fmt.Sprintf("api-test-versions-%d", time.Now().UnixNano())

	// Register multiple versions
	for i := 0; i < 3; i++ {
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"Versions","fields":[{"name":"id","type":"int"},{"name":"v%d","type":["null","string"],"default":null}]}`, i),
		}
		resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
		resp.Body.Close()
	}

	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions", nil)
	expectStatus(t, resp, http.StatusOK)

	var versions []int
	parseResponse(t, resp, &versions)

	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}
}

func TestGetSpecificVersion(t *testing.T) {
	subject := fmt.Sprintf("api-test-specific-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Specific","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	resp.Body.Close()

	resp = doRequest(t, "GET", "/subjects/"+subject+"/versions/1", nil)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if result["version"].(float64) != 1 {
		t.Errorf("Expected version 1, got %v", result["version"])
	}
}

func TestGetLatestVersion(t *testing.T) {
	subject := fmt.Sprintf("api-test-latest-%d", time.Now().UnixNano())

	// Register 2 versions
	for i := 0; i < 2; i++ {
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"Latest","fields":[{"name":"id","type":"int"},{"name":"v%d","type":["null","int"],"default":null}]}`, i),
		}
		resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
		resp.Body.Close()
	}

	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions/latest", nil)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if result["version"].(float64) != 2 {
		t.Errorf("Expected version 2, got %v", result["version"])
	}
}

func TestGetRawSchemaByVersion(t *testing.T) {
	subject := fmt.Sprintf("api-test-raw-version-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"RawVersion","fields":[{"name":"data","type":"string"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	resp.Body.Close()

	resp = doRequest(t, "GET", "/subjects/"+subject+"/versions/1/schema", nil)
	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

// Lookup tests

func TestLookupSchema(t *testing.T) {
	subject := fmt.Sprintf("api-test-lookup-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Lookup","fields":[{"name":"id","type":"int"}]}`,
	}

	// Register
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var regResult map[string]interface{}
	parseResponse(t, resp, &regResult)

	// Lookup
	resp = doRequest(t, "POST", "/subjects/"+subject, schema)
	expectStatus(t, resp, http.StatusOK)

	var lookupResult map[string]interface{}
	parseResponse(t, resp, &lookupResult)

	if lookupResult["id"].(float64) != regResult["id"].(float64) {
		t.Error("Lookup returned different ID than registration")
	}
}

func TestLookupNonExistentSchema(t *testing.T) {
	subject := fmt.Sprintf("api-test-lookup-notfound-%d", time.Now().UnixNano())

	// Register with one schema
	schema1 := map[string]interface{}{
		"schema": `{"type":"record","name":"LookupNF1","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)
	resp.Body.Close()

	// Lookup with different schema
	schema2 := map[string]interface{}{
		"schema": `{"type":"record","name":"LookupNF2","fields":[{"name":"different","type":"string"}]}`,
	}
	resp = doRequest(t, "POST", "/subjects/"+subject, schema2)
	expectStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// Config tests

func TestGetGlobalConfig(t *testing.T) {
	resp := doRequest(t, "GET", "/config", nil)
	expectStatus(t, resp, http.StatusOK)

	var config map[string]interface{}
	parseResponse(t, resp, &config)

	if _, ok := config["compatibilityLevel"]; !ok {
		t.Error("Expected 'compatibilityLevel' in response")
	}
}

func TestSetGlobalConfig(t *testing.T) {
	config := map[string]string{"compatibility": "FULL"}

	resp := doRequest(t, "PUT", "/config", config)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)
}

func TestDeleteGlobalConfig(t *testing.T) {
	resp := doRequest(t, "DELETE", "/config", nil)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)
}

func TestSubjectConfig(t *testing.T) {
	subject := fmt.Sprintf("api-test-config-%d", time.Now().UnixNano())

	// Register schema first
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Config","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	resp.Body.Close()

	// Set config
	config := map[string]string{"compatibility": "NONE"}
	resp = doRequest(t, "PUT", "/config/"+subject, config)
	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Get config
	resp = doRequest(t, "GET", "/config/"+subject, nil)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	// Delete config
	resp = doRequest(t, "DELETE", "/config/"+subject, nil)
	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

// Mode tests

func TestGetGlobalMode(t *testing.T) {
	resp := doRequest(t, "GET", "/mode", nil)
	expectStatus(t, resp, http.StatusOK)

	var mode map[string]interface{}
	parseResponse(t, resp, &mode)

	if _, ok := mode["mode"]; !ok {
		t.Error("Expected 'mode' in response")
	}
}

func TestSetGlobalMode(t *testing.T) {
	mode := map[string]string{"mode": "READWRITE"}

	resp := doRequest(t, "PUT", "/mode", mode)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)
}

// Compatibility tests

func TestCompatibilityCheck(t *testing.T) {
	subject := fmt.Sprintf("api-test-compat-%d", time.Now().UnixNano())

	// Register initial schema
	schema1 := map[string]interface{}{
		"schema": `{"type":"record","name":"Compat","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)
	resp.Body.Close()

	// Check compatible schema
	schema2 := map[string]interface{}{
		"schema": `{"type":"record","name":"Compat","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}`,
	}
	resp = doRequest(t, "POST", "/compatibility/subjects/"+subject+"/versions/latest", schema2)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if !result["is_compatible"].(bool) {
		t.Error("Expected schema to be compatible")
	}
}

func TestIncompatibilityCheck(t *testing.T) {
	subject := fmt.Sprintf("api-test-incompat-%d", time.Now().UnixNano())

	// Register initial schema
	schema1 := map[string]interface{}{
		"schema": `{"type":"record","name":"Incompat","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)
	resp.Body.Close()

	// Check incompatible schema (removes required field)
	schema2 := map[string]interface{}{
		"schema": `{"type":"record","name":"Incompat","fields":[{"name":"different","type":"string"}]}`,
	}
	resp = doRequest(t, "POST", "/compatibility/subjects/"+subject+"/versions/latest", schema2)
	expectStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if result["is_compatible"].(bool) {
		t.Error("Expected schema to be incompatible")
	}
}

// Delete tests

func TestDeleteSubject(t *testing.T) {
	subject := fmt.Sprintf("api-test-delete-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Delete","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	resp.Body.Close()

	// Soft delete
	resp = doRequest(t, "DELETE", "/subjects/"+subject, nil)
	expectStatus(t, resp, http.StatusOK)

	var versions []int
	parseResponse(t, resp, &versions)

	if len(versions) == 0 {
		t.Error("Expected deleted versions in response")
	}
}

func TestDeleteVersion(t *testing.T) {
	subject := fmt.Sprintf("api-test-delete-ver-%d", time.Now().UnixNano())

	// Register multiple versions
	for i := 0; i < 2; i++ {
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"DeleteVer","fields":[{"name":"id","type":"int"},{"name":"v%d","type":["null","int"],"default":null}]}`, i),
		}
		resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
		resp.Body.Close()
	}

	// Delete version 1
	resp := doRequest(t, "DELETE", "/subjects/"+subject+"/versions/1", nil)
	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

// Referenced by tests

func TestGetReferencedBy(t *testing.T) {
	subject := fmt.Sprintf("api-test-refby-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"RefBy","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	resp.Body.Close()

	resp = doRequest(t, "GET", "/subjects/"+subject+"/versions/1/referencedby", nil)
	expectStatus(t, resp, http.StatusOK)

	var refs []int
	parseResponse(t, resp, &refs)
}

// Error handling tests

func TestNotFoundSubject(t *testing.T) {
	resp := doRequest(t, "GET", "/subjects/nonexistent-subject-12345/versions", nil)
	expectStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestNotFoundVersion(t *testing.T) {
	subject := fmt.Sprintf("api-test-nf-ver-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"NFVer","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	resp.Body.Close()

	resp = doRequest(t, "GET", "/subjects/"+subject+"/versions/999", nil)
	expectStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func TestInvalidCompatibilityLevel(t *testing.T) {
	config := map[string]string{"compatibility": "INVALID_LEVEL"}

	resp := doRequest(t, "PUT", "/config", config)

	// Should fail with 422 Unprocessable Entity
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 422 or 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestInvalidMode(t *testing.T) {
	mode := map[string]string{"mode": "INVALID_MODE"}

	resp := doRequest(t, "PUT", "/mode", mode)

	// Should fail
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 422 or 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
