//go:build integration

// Package integration provides integration tests for the schema registry.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/axonops/axonops-schema-registry/internal/api"
	"github.com/axonops/axonops-schema-registry/internal/compatibility"
	avrocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/avro"
	jsoncompat "github.com/axonops/axonops-schema-registry/internal/compatibility/jsonschema"
	protocompat "github.com/axonops/axonops-schema-registry/internal/compatibility/protobuf"
	"github.com/axonops/axonops-schema-registry/internal/config"
	"github.com/axonops/axonops-schema-registry/internal/registry"
	"github.com/axonops/axonops-schema-registry/internal/schema"
	"github.com/axonops/axonops-schema-registry/internal/schema/avro"
	"github.com/axonops/axonops-schema-registry/internal/schema/jsonschema"
	"github.com/axonops/axonops-schema-registry/internal/schema/protobuf"
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

	// Create schema parser registry and register parsers
	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	// Create compatibility checker and register type-specific checkers
	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	// Create registry
	reg := registry.New(store, schemaRegistry, compatChecker, "BACKWARD")

	// Create server with logger
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8081,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := api.NewServer(cfg, reg, logger)
	testServer = httptest.NewServer(server)

	// Run tests
	code := m.Run()

	// Cleanup
	testServer.Close()
	store.Close()

	os.Exit(code)
}

func createStorage(_ context.Context) (storage.Storage, error) {
	storageType := os.Getenv("STORAGE_TYPE")

	switch storageType {
	case "postgres":
		cfg := postgres.Config{
			Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
			Port:     getEnvOrDefaultInt("POSTGRES_PORT", 5432),
			Username: getEnvOrDefault("POSTGRES_USER", "schemaregistry"),
			Password: getEnvOrDefault("POSTGRES_PASSWORD", "schemaregistry"),
			Database: getEnvOrDefault("POSTGRES_DATABASE", "schemaregistry"),
			SSLMode:  "disable",
		}
		return postgres.NewStore(cfg)

	case "mysql":
		cfg := mysql.Config{
			Host:     getEnvOrDefault("MYSQL_HOST", "localhost"),
			Port:     getEnvOrDefaultInt("MYSQL_PORT", 3306),
			Username: getEnvOrDefault("MYSQL_USER", "schemaregistry"),
			Password: getEnvOrDefault("MYSQL_PASSWORD", "schemaregistry"),
			Database: getEnvOrDefault("MYSQL_DATABASE", "schemaregistry"),
		}
		return mysql.NewStore(cfg)

	case "cassandra":
		cfg := cassandra.Config{
			Hosts:               []string{getEnvOrDefault("CASSANDRA_HOSTS", "localhost")},
			Port:                getEnvOrDefaultInt("CASSANDRA_PORT", 9042),
			Keyspace:            getEnvOrDefault("CASSANDRA_KEYSPACE", "schemaregistry"),
			Consistency:         "ONE", // Use ONE for single-node test cluster (simpler than LOCAL_ONE)
			LocalDC:             "",    // Don't use DC-aware policy for single-node setup
			ReplicationStrategy: "SimpleStrategy",
			ReplicationFactor:   1,
			ConnectTimeout:      30 * time.Second, // Longer timeout for CI
			Timeout:             30 * time.Second,
			NumConns:            5, // More connections for integration tests
		}
		// Use retry logic for connection establishment
		return cassandra.NewStoreWithRetry(cfg, 5, 3*time.Second)

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
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
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
	registerResp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	if registerResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(registerResp.Body)
		t.Fatalf("Failed to register schema: status %d, body: %s", registerResp.StatusCode, body)
	}

	// Get version 1
	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions/1", nil)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d, response: %v", resp.StatusCode, result)
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
	resp1 := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)
	if resp1.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp1.Body)
		t.Fatalf("Failed to register schema1: status %d, body: %s", resp1.StatusCode, body)
	}

	schema2 := map[string]interface{}{
		"schema": `{"type":"record","name":"TestLatest","fields":[{"name":"id","type":"int"},{"name":"extra","type":["null","string"],"default":null}]}`,
	}
	resp2 := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema2)
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("Failed to register schema2: status %d, body: %s", resp2.StatusCode, body)
	}

	// Get latest
	resp := doRequest(t, "GET", "/subjects/"+subject+"/versions/latest", nil)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d, response: %v", resp.StatusCode, result)
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
	registerResp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	if registerResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(registerResp.Body)
		registerResp.Body.Close()
		t.Fatalf("Failed to register schema: status %d, body: %s", registerResp.StatusCode, body)
	}
	registerResp.Body.Close()

	// Lookup
	resp := doRequest(t, "POST", "/subjects/"+subject, schema)

	var result map[string]interface{}
	parseResponse(t, resp, &result)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d, body: %v", resp.StatusCode, result)
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
	registerResp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	if registerResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(registerResp.Body)
		registerResp.Body.Close()
		t.Fatalf("Failed to register schema: status %d, body: %s", registerResp.StatusCode, body)
	}
	registerResp.Body.Close()

	// Delete
	resp := doRequest(t, "DELETE", "/subjects/"+subject, nil)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Expected status 200, got %d, body: %s", resp.StatusCode, body)
	}

	var versions []int
	parseResponse(t, resp, &versions)

	if len(versions) == 0 {
		t.Error("Expected deleted versions in response")
	}
}

func TestConfig(t *testing.T) {
	// Get global config
	resp := doRequest(t, "GET", "/config", nil)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Expected status 200 for GET /config, got %d, body: %s", resp.StatusCode, body)
	}

	var config map[string]interface{}
	parseResponse(t, resp, &config)

	// Set config
	newConfig := map[string]string{
		"compatibility": "FULL",
	}
	resp = doRequest(t, "PUT", "/config", newConfig)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Expected status 200 for PUT /config, got %d, body: %s", resp.StatusCode, body)
	}

	parseResponse(t, resp, &config)
}

func TestSubjectConfig(t *testing.T) {
	subject := fmt.Sprintf("test-config-%d", time.Now().UnixNano())

	// Register a schema first
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestConfig","fields":[{"name":"y","type":"int"}]}`,
	}
	registerResp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	if registerResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(registerResp.Body)
		registerResp.Body.Close()
		t.Fatalf("Failed to register schema: status %d, body: %s", registerResp.StatusCode, body)
	}
	registerResp.Body.Close()

	// Set subject config
	newConfig := map[string]string{
		"compatibility": "NONE",
	}
	resp := doRequest(t, "PUT", "/config/"+subject, newConfig)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Expected status 200, got %d, body: %s", resp.StatusCode, body)
	}

	var config map[string]interface{}
	parseResponse(t, resp, &config)
}

func TestCompatibilityCheck(t *testing.T) {
	subject := fmt.Sprintf("test-compat-%d", time.Now().UnixNano())

	// Register initial schema
	schema1 := map[string]interface{}{
		"schema": `{"type":"record","name":"TestCompat","fields":[{"name":"id","type":"int"}]}`,
	}
	registerResp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)
	if registerResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(registerResp.Body)
		registerResp.Body.Close()
		t.Fatalf("Failed to register schema: status %d, body: %s", registerResp.StatusCode, body)
	}
	registerResp.Body.Close()

	// Check compatible schema
	compatSchema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestCompat","fields":[{"name":"id","type":"int"},{"name":"name","type":["null","string"],"default":null}]}`,
	}
	resp := doRequest(t, "POST", "/compatibility/subjects/"+subject+"/versions/latest", compatSchema)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Expected status 200, got %d, body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	parseResponse(t, resp, &result)

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
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register schema: status %d, body: %s", resp.StatusCode, body)
	}

	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	idVal, ok := registerResult["id"]
	if !ok || idVal == nil {
		t.Fatalf("Expected 'id' in registration response, got: %v", registerResult)
	}
	schemaID := int(idVal.(float64))

	// Get raw schema
	resp = doRequest(t, "GET", fmt.Sprintf("/schemas/ids/%d/schema", schemaID), nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d, body: %s", resp.StatusCode, body)
	}
}

func TestGetSchemaByIDSubjects(t *testing.T) {
	subject := fmt.Sprintf("test-id-subjects-%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"TestIDSubjects","fields":[{"name":"sub","type":"string"}]}`,
	}

	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register schema: status %d, body: %s", resp.StatusCode, body)
	}

	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	idVal, ok := registerResult["id"]
	if !ok || idVal == nil {
		t.Fatalf("Expected 'id' in registration response, got: %v", registerResult)
	}
	schemaID := int(idVal.(float64))

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
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register schema: status %d, body: %s", resp.StatusCode, body)
	}

	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	idVal, ok := registerResult["id"]
	if !ok || idVal == nil {
		t.Fatalf("Expected 'id' in registration response, got: %v", registerResult)
	}
	schemaID := int(idVal.(float64))

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

// Database validation tests - verify data is stored correctly in the database

func TestDatabaseValidation_SchemaStorage(t *testing.T) {
	ctx := context.Background()
	subject := fmt.Sprintf("db-validate-schema-%d", time.Now().UnixNano())
	schemaStr := `{"type":"record","name":"DBValidate","fields":[{"name":"id","type":"int"},{"name":"name","type":"string"}]}`

	// Register schema via API
	schemaReq := map[string]interface{}{
		"schema": schemaStr,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schemaReq)
	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register schema: status %d", resp.StatusCode)
	}

	schemaID := int64(registerResult["id"].(float64))

	// Validate: Query database directly to verify schema is stored correctly
	dbSchema, err := testStore.GetSchemaByID(ctx, schemaID)
	if err != nil {
		t.Fatalf("Database query failed - GetSchemaByID: %v", err)
	}

	// Verify schema ID matches
	if dbSchema.ID != schemaID {
		t.Errorf("Database validation failed: schema ID mismatch - expected %d, got %d", schemaID, dbSchema.ID)
	}

	// Verify subject matches
	if dbSchema.Subject != subject {
		t.Errorf("Database validation failed: subject mismatch - expected %s, got %s", subject, dbSchema.Subject)
	}

	// Verify version is 1
	if dbSchema.Version != 1 {
		t.Errorf("Database validation failed: version mismatch - expected 1, got %d", dbSchema.Version)
	}

	// Verify schema type is AVRO
	if dbSchema.SchemaType != storage.SchemaTypeAvro {
		t.Errorf("Database validation failed: schema type mismatch - expected AVRO, got %s", dbSchema.SchemaType)
	}

	// Verify schema content is not empty
	if dbSchema.Schema == "" {
		t.Error("Database validation failed: schema content is empty")
	}

	// Verify schema is not deleted
	if dbSchema.Deleted {
		t.Error("Database validation failed: schema should not be marked as deleted")
	}

	t.Logf("Database validation passed: Schema ID=%d, Subject=%s, Version=%d, Type=%s",
		dbSchema.ID, dbSchema.Subject, dbSchema.Version, dbSchema.SchemaType)
}

func TestDatabaseValidation_SubjectListing(t *testing.T) {
	ctx := context.Background()
	subject := fmt.Sprintf("db-validate-subject-%d", time.Now().UnixNano())

	// Register schema via API
	schemaReq := map[string]interface{}{
		"schema": `{"type":"record","name":"DBSubject","fields":[{"name":"value","type":"long"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schemaReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register schema: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Validate: Query database directly to verify subject exists
	subjects, err := testStore.ListSubjects(ctx, false)
	if err != nil {
		t.Fatalf("Database query failed - ListSubjects: %v", err)
	}

	found := false
	for _, s := range subjects {
		if s == subject {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Database validation failed: subject %s not found in database. Found subjects: %v", subject, subjects)
	}

	// Validate: SubjectExists returns true
	exists, err := testStore.SubjectExists(ctx, subject)
	if err != nil {
		t.Fatalf("Database query failed - SubjectExists: %v", err)
	}
	if !exists {
		t.Error("Database validation failed: SubjectExists returned false for registered subject")
	}

	t.Logf("Database validation passed: Subject %s exists in database", subject)
}

func TestDatabaseValidation_ConfigPersistence(t *testing.T) {
	ctx := context.Background()
	subject := fmt.Sprintf("db-validate-config-%d", time.Now().UnixNano())

	// Register schema first
	schemaReq := map[string]interface{}{
		"schema": `{"type":"record","name":"DBConfig","fields":[{"name":"x","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schemaReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register schema: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Set config via API
	configReq := map[string]string{"compatibility": "FULL"}
	resp = doRequest(t, "PUT", "/config/"+subject, configReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to set config: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Validate: Query database directly to verify config is stored
	config, err := testStore.GetConfig(ctx, subject)
	if err != nil {
		t.Fatalf("Database query failed - GetConfig: %v", err)
	}

	if config.CompatibilityLevel != "FULL" {
		t.Errorf("Database validation failed: config mismatch - expected FULL, got %s", config.CompatibilityLevel)
	}

	t.Logf("Database validation passed: Config for %s is %s", subject, config.CompatibilityLevel)
}

func TestDatabaseValidation_SchemaVersions(t *testing.T) {
	ctx := context.Background()
	subject := fmt.Sprintf("db-validate-versions-%d", time.Now().UnixNano())

	// Register first version
	schema1 := map[string]interface{}{
		"schema": `{"type":"record","name":"DBVersions","fields":[{"name":"id","type":"int"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schema1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register schema v1: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Register second version (compatible)
	schema2 := map[string]interface{}{
		"schema": `{"type":"record","name":"DBVersions","fields":[{"name":"id","type":"int"},{"name":"extra","type":["null","string"],"default":null}]}`,
	}
	resp = doRequest(t, "POST", "/subjects/"+subject+"/versions", schema2)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register schema v2: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Validate: Query database directly to verify both versions exist
	schemas, err := testStore.GetSchemasBySubject(ctx, subject, false)
	if err != nil {
		t.Fatalf("Database query failed - GetSchemasBySubject: %v", err)
	}

	if len(schemas) != 2 {
		t.Errorf("Database validation failed: expected 2 versions, got %d", len(schemas))
	}

	// Verify versions are 1 and 2
	versionMap := make(map[int]bool)
	for _, s := range schemas {
		versionMap[s.Version] = true
	}

	if !versionMap[1] || !versionMap[2] {
		t.Errorf("Database validation failed: expected versions 1 and 2, got %v", versionMap)
	}

	// Validate: GetLatestSchema returns version 2
	latest, err := testStore.GetLatestSchema(ctx, subject)
	if err != nil {
		t.Fatalf("Database query failed - GetLatestSchema: %v", err)
	}

	if latest.Version != 2 {
		t.Errorf("Database validation failed: expected latest version 2, got %d", latest.Version)
	}

	t.Logf("Database validation passed: Subject %s has %d versions, latest is v%d", subject, len(schemas), latest.Version)
}

func TestDatabaseValidation_SoftDelete(t *testing.T) {
	ctx := context.Background()
	subject := fmt.Sprintf("db-validate-delete-%d", time.Now().UnixNano())

	// Register schema
	schemaReq := map[string]interface{}{
		"schema": `{"type":"record","name":"DBDelete","fields":[{"name":"y","type":"double"}]}`,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schemaReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register schema: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Verify subject exists before delete
	existsBefore, err := testStore.SubjectExists(ctx, subject)
	if err != nil {
		t.Fatalf("Database query failed - SubjectExists (before): %v", err)
	}
	if !existsBefore {
		t.Fatal("Database validation failed: subject should exist before delete")
	}

	// Delete subject via API (soft delete)
	resp = doRequest(t, "DELETE", "/subjects/"+subject, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to delete subject: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Validate: Subject should not appear in non-deleted list
	subjects, err := testStore.ListSubjects(ctx, false)
	if err != nil {
		t.Fatalf("Database query failed - ListSubjects: %v", err)
	}

	for _, s := range subjects {
		if s == subject {
			t.Errorf("Database validation failed: deleted subject %s still appears in non-deleted list", subject)
		}
	}

	// Validate: Subject should appear in deleted list
	subjectsWithDeleted, err := testStore.ListSubjects(ctx, true)
	if err != nil {
		t.Fatalf("Database query failed - ListSubjects (with deleted): %v", err)
	}

	found := false
	for _, s := range subjectsWithDeleted {
		if s == subject {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Database validation failed: deleted subject %s not found in includeDeleted list", subject)
	}

	t.Logf("Database validation passed: Subject %s correctly soft-deleted", subject)
}

func TestDatabaseValidation_GlobalConfig(t *testing.T) {
	ctx := context.Background()

	// Set global config via API
	configReq := map[string]string{"compatibility": "FULL_TRANSITIVE"}
	resp := doRequest(t, "PUT", "/config", configReq)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to set global config: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Validate: Query database directly
	config, err := testStore.GetGlobalConfig(ctx)
	if err != nil {
		t.Fatalf("Database query failed - GetGlobalConfig: %v", err)
	}

	if config.CompatibilityLevel != "FULL_TRANSITIVE" {
		t.Errorf("Database validation failed: global config mismatch - expected FULL_TRANSITIVE, got %s", config.CompatibilityLevel)
	}

	t.Logf("Database validation passed: Global config is %s", config.CompatibilityLevel)

	// Reset to default
	resetReq := map[string]string{"compatibility": "BACKWARD"}
	resp = doRequest(t, "PUT", "/config", resetReq)
	resp.Body.Close()
}

func TestDatabaseValidation_SchemaByFingerprint(t *testing.T) {
	ctx := context.Background()
	subject := fmt.Sprintf("db-validate-fingerprint-%d", time.Now().UnixNano())
	schemaStr := `{"type":"record","name":"DBFingerprint","fields":[{"name":"fp","type":"bytes"}]}`

	// Register schema via API
	schemaReq := map[string]interface{}{
		"schema": schemaStr,
	}
	resp := doRequest(t, "POST", "/subjects/"+subject+"/versions", schemaReq)
	var registerResult map[string]interface{}
	parseResponse(t, resp, &registerResult)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register schema: status %d", resp.StatusCode)
	}

	schemaID := int64(registerResult["id"].(float64))

	// Get the schema to find its fingerprint
	dbSchema, err := testStore.GetSchemaByID(ctx, schemaID)
	if err != nil {
		t.Fatalf("Database query failed - GetSchemaByID: %v", err)
	}

	if dbSchema.Fingerprint == "" {
		t.Log("Note: Fingerprint is empty, skipping fingerprint lookup test")
		return
	}

	// Validate: Query by fingerprint
	schemaByFP, err := testStore.GetSchemaByFingerprint(ctx, subject, dbSchema.Fingerprint)
	if err != nil {
		t.Fatalf("Database query failed - GetSchemaByFingerprint: %v", err)
	}

	if schemaByFP.ID != schemaID {
		t.Errorf("Database validation failed: fingerprint lookup returned wrong schema - expected ID %d, got %d", schemaID, schemaByFP.ID)
	}

	t.Logf("Database validation passed: Schema ID=%d found by fingerprint %s", schemaID, dbSchema.Fingerprint)
}
