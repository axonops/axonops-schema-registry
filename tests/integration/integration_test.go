//go:build integration

// Package integration provides integration tests for the schema registry.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/api"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/storage"
	"github.com/axonops/axonops-schema-registry/internal/storage/cassandra"
	"github.com/axonops/axonops-schema-registry/internal/storage/mysql"
	"github.com/axonops/axonops-schema-registry/internal/storage/postgres"
)

var (
	testServer *httptest.Server
	testStore  storage.Storage
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Create storage based on environment
	store, err := createStorage(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create storage: %v\n", err)
		os.Exit(1)
	}
	testStore = store

	// Create registry
	reg := registry.New(store)

	// Create server
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8081,
		},
	}
	server := api.NewServer(cfg, reg, nil)
	testServer = httptest.NewServer(server)

	// Run tests
	code := m.Run()

	// Cleanup
	testServer.Close()
	store.Close()

	os.Exit(code)
}

func createStorage(ctx context.Context) (storage.Storage, error) {
	storageType := os.Getenv("STORAGE_TYPE")

	switch storageType {
	case "postgres":
		cfg := &postgres.Config{
			Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
			Port:     getEnvOrDefaultInt("POSTGRES_PORT", 5432),
			User:     getEnvOrDefault("POSTGRES_USER", "schemaregistry"),
			Password: getEnvOrDefault("POSTGRES_PASSWORD", "schemaregistry"),
			Database: getEnvOrDefault("POSTGRES_DATABASE", "schemaregistry"),
			SSLMode:  "disable",
		}
		return postgres.New(ctx, cfg)

	case "mysql":
		cfg := &mysql.Config{
			Host:     getEnvOrDefault("MYSQL_HOST", "localhost"),
			Port:     getEnvOrDefaultInt("MYSQL_PORT", 3306),
			User:     getEnvOrDefault("MYSQL_USER", "schemaregistry"),
			Password: getEnvOrDefault("MYSQL_PASSWORD", "schemaregistry"),
			Database: getEnvOrDefault("MYSQL_DATABASE", "schemaregistry"),
		}
		return mysql.New(ctx, cfg)

	case "cassandra":
		cfg := &cassandra.Config{
			Hosts:    []string{getEnvOrDefault("CASSANDRA_HOSTS", "localhost")},
			Port:     getEnvOrDefaultInt("CASSANDRA_PORT", 9042),
			Keyspace: getEnvOrDefault("CASSANDRA_KEYSPACE", "schemaregistry"),
		}
		return cassandra.New(ctx, cfg)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		fmt.Sscanf(value, "%d", &intValue)
		return intValue
	}
	return defaultValue
}

// Test helper functions
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

	req, err := http.NewRequest(method, testServer.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

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

// Tests

func TestHealthCheck(t *testing.T) {
	resp := doRequest(t, "GET", "/", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestSchemaTypes(t *testing.T) {
	resp := doRequest(t, "GET", "/schemas/types", nil)

	var types []string
	parseResponse(t, resp, &types)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Should include at least AVRO
	found := false
	for _, schemaType := range types {
		if schemaType == "AVRO" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected AVRO in schema types")
	}
}

func TestRegisterSchema(t *testing.T) {
	subject := fmt.Sprintf("test-subject-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`,
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if _, ok := result["id"]; !ok {
		t.Error("Expected 'id' in response")
	}
}

func TestGetSchemaByID(t *testing.T) {
	// First register a schema
	subject := fmt.Sprintf("test-schema-id-%d", time.Now().UnixNano())
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestByID","fields":[{"name":"name","type":"string"}]}`,
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	schemaID := int(registerResult["id"].(float64))

	// Get schema by ID
	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d", schemaID), nil)

	var schemaResult map[string]interface{}
	parseResponse(t, resp, &schemaResult)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if _, ok := schemaResult["schema"]; !ok {
		t.Error("Expected 'schema' in response")
	}
}

func TestGetSubjects(t *testing.T) {
	// Register a schema first
	subject := fmt.Sprintf("test-subjects-%d", time.Now().UnixNano())
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestSubjects","fields":[{"name":"value","type":"long"}]}`,
	}

	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	// Get subjects
	resp := doRequest(t, "GET", "/subjects", nil)

	var subjects []string
	parseResponse(t, resp, &subjects)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Should include our subject
	found := false
	for _, s := range subjects {
		if s == subject {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected subject %s in list", subject)
	}
}

func TestGetVersions(t *testing.T) {
	subject := fmt.Sprintf("test-versions-%d", time.Now().UnixNano())

	// Register first version
	schema1 := map[string]interface{}{
		"schema": `{"type":"record","name":"TestVersions","fields":[{"name":"id","type":"int"}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)

	// Register second version (compatible change)
	schema2 := map[string]interface{}{
		"schema": `{"type":"record","name":"TestVersions","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema2)

	// Get versions
	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions", nil)

	var versions []int
	parseResponse(t, resp, &versions)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if len(versions) < 2 {
		t.Errorf("Expected at least 2 versions, got %d", len(versions))
	}
}

func TestGetVersion(t *testing.T) {
	subject := fmt.Sprintf("test-get-version-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestGetVersion","fields":[{"name":"data","type":"bytes"}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	// Get version 1
	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions/1", nil)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if result["version"].(float64) != 1 {
		t.Errorf("Expected version 1, got %v", result["version"])
	}
}

func TestGetLatestVersion(t *testing.T) {
	subject := fmt.Sprintf("test-latest-%d", time.Now().UnixNano())

	// Register multiple versions
	schema1 := map[string]interface{}{
		"schema": `{"type":"record","name":"TestLatest","fields":[{"name":"id","type":"int"}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)

	schema2 := map[string]interface{}{
		"schema": `{"type":"record","name":"TestLatest","fields":[{"name":"id","type":"int"},{"name":"extra","type":["null","string"],"default":null}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema2)

	// Get latest
	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions/latest", nil)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if result["version"].(float64) != 2 {
		t.Errorf("Expected version 2 (latest), got %v", result["version"])
	}
}

func TestLookupSchema(t *testing.T) {
	subject := fmt.Sprintf("test-lookup-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestLookup","fields":[{"name":"field1","type":"float"}]}`,
	}

	// Register
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	// Lookup
	resp := doRequest(t, "POST", "/subjects/"+subject, schema)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if _, ok := result["id"]; !ok {
		t.Error("Expected 'id' in lookup response")
	}
}

func TestDeleteSubject(t *testing.T) {
	subject := fmt.Sprintf("test-delete-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestDelete","fields":[{"name":"x","type":"double"}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	// Delete
	resp := doRequest(t, "DELETE", "/subjects/"+subject, nil)

	var versions []int
	parseResponse(t, resp, &versions)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if len(versions) == 0 {
		t.Error("Expected deleted versions in response")
	}
}

func TestConfig(t *testing.T) {
	// Get global config
	resp := doRequest(t, "GET", "/config", nil)

	var config map[string]interface{}
	parseResponse(t, resp, &config)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Set config
	newConfig := map[string]string{
		"compatibility": "FULL",
	}
	resp = doRequest(t, "PUT", "/config", newConfig)

	parseResponse(t, resp, &config)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestSubjectConfig(t *testing.T) {
	subject := fmt.Sprintf("test-config-%d", time.Now().UnixNano())

	// Register a schema first
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestConfig","fields":[{"name":"y","type":"int"}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	// Set subject config
	newConfig := map[string]string{
		"compatibility": "NONE",
	}
	resp := doRequest(t, "PUT", "/config/"+subject, newConfig)

	var config map[string]interface{}
	parseResponse(t, resp, &config)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestCompatibilityCheck(t *testing.T) {
	subject := fmt.Sprintf("test-compat-%d", time.Now().UnixNano())

	// Register initial schema
	schema1 := map[string]interface{}{
		"schema": `{"type":"record","name":"TestCompat","fields":[{"name":"id","type":"int"}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)

	// Check compatible schema
	compatSchema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestCompat","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}`,
	}
	resp := doRequest(t, "POST", "/compatibility/subjects/"+subject+"/versions/latest", compatSchema)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if isCompat, ok := result["is_compatible"].(bool); !ok || !isCompat {
		t.Error("Expected schema to be compatible")
	}
}

func TestMode(t *testing.T) {
	// Get global mode
	resp := doRequest(t, "GET", "/mode", nil)

	var mode map[string]interface{}
	parseResponse(t, resp, &mode)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetSchemaByIDSchema(t *testing.T) {
	subject := fmt.Sprintf("test-raw-schema-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestRaw","fields":[{"name":"raw","type":"string"}]}`,
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	schemaID := int(registerResult["id"].(float64))

	// Get raw schema
	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d/schema", schemaID), nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetSchemaByIDSubjects(t *testing.T) {
	subject := fmt.Sprintf("test-id-subjects-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestIDSubjects","fields":[{"name":"sub","type":"string"}]}`,
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	schemaID := int(registerResult["id"].(float64))

	// Get subjects for schema ID
	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d/subjects", schemaID), nil)

	var subjects []string
	parseResponse(t, resp, &subjects)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetSchemaByIDVersions(t *testing.T) {
	subject := fmt.Sprintf("test-id-versions-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestIDVersions","fields":[{"name":"ver","type":"int"}]}`,
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	schemaID := int(registerResult["id"].(float64))

	// Get versions for schema ID
	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d/versions", schemaID), nil)

	var versions []map[string]interface{}
	parseResponse(t, resp, &versions)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestListSchemas(t *testing.T) {
	// Register some schemas
	for i := 0; i < 3; i++ {
		subject := fmt.Sprintf("test-list-schemas-%d-%d", time.Now().UnixNano(), i)
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"TestList%d","fields":[{"name":"id","type":"int"}]}`, i),
		}
		doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	}

	// List schemas
	resp := doRequest(t, "GET", "/schemas", nil)

	var schemas []map[string]interface{}
	parseResponse(t, resp, &schemas)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetRawSchemaByVersion(t *testing.T) {
	subject := fmt.Sprintf("test-raw-version-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestRawVer","fields":[{"name":"raw","type":"string"}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	// Get raw schema by version
	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions/1/schema", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestContexts(t *testing.T) {
	resp := doRequest(t, "GET", "/contexts", nil)

	var contexts []string
	parseResponse(t, resp, &contexts)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Should include default context
	if len(contexts) == 0 {
		t.Error("Expected at least one context")
	}
}

func TestMetadataID(t *testing.T) {
	resp := doRequest(t, "GET", "/v1/metadata/id", nil)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if _, ok := result["id"]; !ok {
		t.Error("Expected 'id' in response")
	}
}

func TestMetadataVersion(t *testing.T) {
	resp := doRequest(t, "GET", "/v1/metadata/version", nil)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if _, ok := result["version"]; !ok {
		t.Error("Expected 'version' in response")
	}
}

func TestDeleteGlobalConfig(t *testing.T) {
	// Set a config first
	newConfig := map[string]string{
		"compatibility": "FULL",
	}
	doRequest(t, "PUT", "/config", newConfig)

	// Delete global config
	resp := doRequest(t, "DELETE", "/config", nil)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestReferencedBy(t *testing.T) {
	subject := fmt.Sprintf("test-refby-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestRefBy","fields":[{"name":"id","type":"int"}]}`,
	}
	doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)

	// Get referenced by
	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions/1/referencedby", nil)

	var refs []int
	parseResponse(t, resp, &refs)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
