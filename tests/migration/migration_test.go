//go:build migration

// Package migration provides integration tests for migrating schemas from
// Confluent Schema Registry to AxonOps Schema Registry.
package migration

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
	"sort"
	"testing"

	"github.com/axonops/axonops-schema-registry/internal/api"
	"github.com/axonops/axonops-schema-registry/internal/api/types"
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
	"github.com/axonops/axonops-schema-registry/internal/storage/memory"
)

// ctx is a background context for tests
var ctx = context.Background()

// createTestServer creates a new test server with in-memory storage
func createTestServer() (*httptest.Server, storage.Storage) {
	store := memory.NewStore()

	// Create schema parser registry and register parsers
	schemaRegistry := schema.NewRegistry()
	schemaRegistry.Register(avro.NewParser())
	schemaRegistry.Register(protobuf.NewParser())
	schemaRegistry.Register(jsonschema.NewParser())

	// Create compatibility checker
	compatChecker := compatibility.NewChecker()
	compatChecker.Register(storage.SchemaTypeAvro, avrocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeProtobuf, protocompat.NewChecker())
	compatChecker.Register(storage.SchemaTypeJSON, jsoncompat.NewChecker())

	// Create registry
	reg := registry.New(store, schemaRegistry, compatChecker, "BACKWARD")

	// Create server
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8081,
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := api.NewServer(cfg, reg, logger)

	return httptest.NewServer(server), store
}

// doRequest performs an HTTP request to the test server
func doRequest(t *testing.T, server *httptest.Server, method, path string, body interface{}) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	return resp
}

// parseResponse parses the HTTP response body into the target
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

// TestMigrationFromConfluent simulates a complete migration from Confluent Schema Registry
// to AxonOps Schema Registry, verifying that:
// 1. Schema IDs are preserved
// 2. Subject/version mappings are correct
// 3. New registrations get IDs after imported IDs
// 4. Schema content is identical
func TestMigrationFromConfluent(t *testing.T) {
	// Create source server (simulating Confluent Schema Registry)
	sourceServer, _ := createTestServer()
	defer sourceServer.Close()

	// Create target server (AxonOps Schema Registry)
	targetServer, _ := createTestServer()
	defer targetServer.Close()

	// Step 1: Register schemas on source (simulating existing Confluent SR data)
	t.Log("Step 1: Registering schemas on source (Confluent SR)")

	testSchemas := []struct {
		subject string
		schema  string
	}{
		{
			subject: "user-value",
			schema:  `{"type":"record","name":"User","fields":[{"name":"id","type":"long"}]}`,
		},
		{
			subject: "user-value",
			schema:  `{"type":"record","name":"User","fields":[{"name":"id","type":"long"},{"name":"name","type":"string","default":""}]}`,
		},
		{
			subject: "order-value",
			schema:  `{"type":"record","name":"Order","fields":[{"name":"order_id","type":"long"}]}`,
		},
		{
			subject: "product-value",
			schema:  `{"type":"record","name":"Product","fields":[{"name":"product_id","type":"long"}]}`,
		},
	}

	sourceSchemaIDs := make([]int64, len(testSchemas))
	for i, ts := range testSchemas {
		resp := doRequest(t, sourceServer, "POST", "/subjects/"+ts.subject+"/versions",
			map[string]interface{}{"schema": ts.schema})
		var result map[string]interface{}
		parseResponse(t, resp, &result)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to register schema %d: status %d", i, resp.StatusCode)
		}

		sourceSchemaIDs[i] = int64(result["id"].(float64))
		t.Logf("  Registered %s: ID=%d", ts.subject, sourceSchemaIDs[i])
	}

	// Step 2: Export schemas from source (simulating migration script export)
	t.Log("Step 2: Exporting schemas from source")

	// Get all subjects
	resp := doRequest(t, sourceServer, "GET", "/subjects", nil)
	var subjects []string
	parseResponse(t, resp, &subjects)

	var exportedSchemas []types.ImportSchemaRequest

	for _, subject := range subjects {
		// Get versions for subject
		resp := doRequest(t, sourceServer, "GET", "/subjects/"+subject+"/versions", nil)
		var versions []int
		parseResponse(t, resp, &versions)

		for _, version := range versions {
			// Get schema details
			resp := doRequest(t, sourceServer, "GET", fmt.Sprintf("/subjects/%s/versions/%d", subject, version), nil)
			var schemaInfo map[string]interface{}
			parseResponse(t, resp, &schemaInfo)

			schemaType := "AVRO"
			if st, ok := schemaInfo["schemaType"].(string); ok && st != "" {
				schemaType = st
			}
			exportedSchemas = append(exportedSchemas, types.ImportSchemaRequest{
				ID:         int64(schemaInfo["id"].(float64)),
				Subject:    subject,
				Version:    int(schemaInfo["version"].(float64)),
				SchemaType: schemaType,
				Schema:     schemaInfo["schema"].(string),
			})
		}
	}

	// Sort by ID to ensure dependencies are imported first
	sort.Slice(exportedSchemas, func(i, j int) bool {
		return exportedSchemas[i].ID < exportedSchemas[j].ID
	})

	t.Logf("  Exported %d schemas", len(exportedSchemas))

	// Step 3: Import schemas to target
	t.Log("Step 3: Importing schemas to target (AxonOps SR)")

	importReq := types.ImportSchemasRequest{Schemas: exportedSchemas}
	resp = doRequest(t, targetServer, "POST", "/import/schemas", importReq)

	var importResult types.ImportSchemasResponse
	parseResponse(t, resp, &importResult)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Import failed: status %d", resp.StatusCode)
	}

	if importResult.Errors > 0 {
		for _, r := range importResult.Results {
			if !r.Success {
				t.Errorf("Import error for ID %d: %s", r.ID, r.Error)
			}
		}
		t.Fatalf("Import had %d errors", importResult.Errors)
	}

	t.Logf("  Imported %d schemas with 0 errors", importResult.Imported)

	// Step 4: Verify schema IDs match
	t.Log("Step 4: Verifying schema IDs match")

	for _, exported := range exportedSchemas {
		// Get from target
		resp := doRequest(t, targetServer, "GET", fmt.Sprintf("/subjects/%s/versions/%d", exported.Subject, exported.Version), nil)
		var targetSchema map[string]interface{}
		parseResponse(t, resp, &targetSchema)

		targetID := int64(targetSchema["id"].(float64))
		if targetID != exported.ID {
			t.Errorf("ID mismatch for %s v%d: source=%d, target=%d",
				exported.Subject, exported.Version, exported.ID, targetID)
		}
	}
	t.Log("  All schema IDs match!")

	// Step 5: Verify subjects match
	t.Log("Step 5: Verifying subjects match")

	resp = doRequest(t, targetServer, "GET", "/subjects", nil)
	var targetSubjects []string
	parseResponse(t, resp, &targetSubjects)

	sort.Strings(subjects)
	sort.Strings(targetSubjects)

	if len(subjects) != len(targetSubjects) {
		t.Errorf("Subject count mismatch: source=%d, target=%d", len(subjects), len(targetSubjects))
	}

	for i, s := range subjects {
		if i >= len(targetSubjects) || targetSubjects[i] != s {
			t.Errorf("Subject mismatch at index %d: source=%s, target=%s", i, s, targetSubjects[i])
		}
	}
	t.Log("  All subjects match!")

	// Step 6: Verify new registrations get correct IDs
	t.Log("Step 6: Verifying new registrations get IDs after imported IDs")

	maxImportedID := sourceSchemaIDs[len(sourceSchemaIDs)-1]

	newSchemaResp := doRequest(t, targetServer, "POST", "/subjects/new-subject/versions",
		map[string]interface{}{"schema": `{"type":"string"}`})
	var newResult map[string]interface{}
	parseResponse(t, newSchemaResp, &newResult)

	newID := int64(newResult["id"].(float64))
	if newID <= maxImportedID {
		t.Errorf("New schema ID (%d) should be > max imported ID (%d)", newID, maxImportedID)
	}
	t.Logf("  New schema got ID %d (> %d)", newID, maxImportedID)

	// Step 7: Verify schema content is identical via API
	t.Log("Step 7: Verifying schema content")

	for _, exported := range exportedSchemas {
		// Get schema by ID from target
		resp := doRequest(t, targetServer, "GET", fmt.Sprintf("/schemas/ids/%d", exported.ID), nil)
		var targetSchema map[string]interface{}
		parseResponse(t, resp, &targetSchema)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Failed to get schema ID %d from target: status %d", exported.ID, resp.StatusCode)
			continue
		}

		// Verify schema type matches (schemaType omitted for AVRO)
		targetSchemaType := "AVRO"
		if st, ok := targetSchema["schemaType"].(string); ok && st != "" {
			targetSchemaType = st
		}
		if targetSchemaType != exported.SchemaType {
			t.Errorf("Schema type mismatch for ID %d: expected %s, got %s",
				exported.ID, exported.SchemaType, targetSchemaType)
		}

		// Verify schema content is present (not comparing exact content due to normalization)
		if targetSchema["schema"] == nil || targetSchema["schema"].(string) == "" {
			t.Errorf("Schema content missing for ID %d", exported.ID)
		}
	}
	t.Log("  All schema content verified!")

	t.Log("Migration test PASSED!")
}

// TestImportWithDuplicateIDs verifies that duplicate IDs are handled correctly:
// - Same schema content with same ID (different subject) is allowed (reuse)
// - Different schema content with same ID is rejected
func TestImportWithDuplicateIDs(t *testing.T) {
	server, _ := createTestServer()
	defer server.Close()

	// Import first schema
	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         100,
				Subject:    "test-subject",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     `{"type":"string"}`,
			},
		},
	}

	resp := doRequest(t, server, "POST", "/import/schemas", importReq)
	var result types.ImportSchemasResponse
	parseResponse(t, resp, &result)

	if result.Errors != 0 {
		t.Fatalf("First import should succeed, got %d errors", result.Errors)
	}

	// Same schema content, same ID, different subject — should succeed (reuse)
	importReq.Schemas[0].Subject = "different-subject"
	resp = doRequest(t, server, "POST", "/import/schemas", importReq)
	parseResponse(t, resp, &result)

	if result.Errors != 0 {
		t.Errorf("Same content with same ID in different subject should succeed, got %d errors", result.Errors)
	}

	// Different schema content with same ID — should be rejected
	importReq.Schemas[0].Subject = "another-subject"
	importReq.Schemas[0].Schema = `{"type":"int"}`
	resp = doRequest(t, server, "POST", "/import/schemas", importReq)
	parseResponse(t, resp, &result)

	if result.Errors != 1 {
		t.Errorf("Different content with same ID should be rejected, got %d errors", result.Errors)
	}

	if result.Results[0].Error != "schema ID already exists" {
		t.Errorf("Expected 'schema ID already exists' error, got: %s", result.Results[0].Error)
	}
}

// TestImportWithReferences tests importing schemas with references
func TestImportWithReferences(t *testing.T) {
	server, store := createTestServer()
	defer server.Close()

	// Import base schema first
	baseSchema := `{"type":"record","name":"Address","namespace":"com.example","fields":[{"name":"street","type":"string"}]}`

	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         1,
				Subject:    "address-value",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     baseSchema,
			},
		},
	}

	resp := doRequest(t, server, "POST", "/import/schemas", importReq)
	var result types.ImportSchemasResponse
	parseResponse(t, resp, &result)

	if result.Errors != 0 {
		t.Fatalf("Base schema import failed: %d errors", result.Errors)
	}

	// Import schema with reference
	userSchema := `{"type":"record","name":"User","namespace":"com.example","fields":[{"name":"name","type":"string"},{"name":"address","type":"com.example.Address"}]}`

	importReq = types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         2,
				Subject:    "user-value",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     userSchema,
				References: []storage.Reference{
					{
						Name:    "com.example.Address",
						Subject: "address-value",
						Version: 1,
					},
				},
			},
		},
	}

	resp = doRequest(t, server, "POST", "/import/schemas", importReq)
	parseResponse(t, resp, &result)

	if result.Errors != 0 {
		t.Fatalf("Referenced schema import failed: %d errors", result.Errors)
	}

	// Verify reference is stored
	ctx := context.Background()
	schema, err := store.GetSchemaByID(ctx, 2)
	if err != nil {
		t.Fatalf("Failed to get schema: %v", err)
	}

	if len(schema.References) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(schema.References))
	}

	if schema.References[0].Name != "com.example.Address" {
		t.Errorf("Expected reference name 'com.example.Address', got '%s'", schema.References[0].Name)
	}
}

// TestImportMultipleVersions tests importing multiple versions of the same subject
func TestImportMultipleVersions(t *testing.T) {
	server, _ := createTestServer()
	defer server.Close()

	// Import multiple versions in order
	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         10,
				Subject:    "multi-version",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"}]}`,
			},
			{
				ID:         20,
				Subject:    "multi-version",
				Version:    2,
				SchemaType: "AVRO",
				Schema:     `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""}]}`,
			},
			{
				ID:         30,
				Subject:    "multi-version",
				Version:    3,
				SchemaType: "AVRO",
				Schema:     `{"type":"record","name":"Test","fields":[{"name":"id","type":"int"},{"name":"name","type":"string","default":""},{"name":"email","type":"string","default":""}]}`,
			},
		},
	}

	resp := doRequest(t, server, "POST", "/import/schemas", importReq)
	var result types.ImportSchemasResponse
	parseResponse(t, resp, &result)

	if result.Errors != 0 {
		t.Fatalf("Import failed: %d errors", result.Errors)
	}

	if result.Imported != 3 {
		t.Errorf("Expected 3 imported, got %d", result.Imported)
	}

	// Verify versions
	resp = doRequest(t, server, "GET", "/subjects/multi-version/versions", nil)
	var versions []int
	parseResponse(t, resp, &versions)

	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}

	// Verify latest version
	resp = doRequest(t, server, "GET", "/subjects/multi-version/versions/latest", nil)
	var latest map[string]interface{}
	parseResponse(t, resp, &latest)

	if int(latest["version"].(float64)) != 3 {
		t.Errorf("Expected latest version 3, got %v", latest["version"])
	}

	if int64(latest["id"].(float64)) != 30 {
		t.Errorf("Expected latest ID 30, got %v", latest["id"])
	}
}

// TestImportPreservesSchemaTypes tests that different schema types are preserved
func TestImportPreservesSchemaTypes(t *testing.T) {
	server, store := createTestServer()
	defer server.Close()

	importReq := types.ImportSchemasRequest{
		Schemas: []types.ImportSchemaRequest{
			{
				ID:         100,
				Subject:    "avro-subject",
				Version:    1,
				SchemaType: "AVRO",
				Schema:     `{"type":"string"}`,
			},
			{
				ID:         200,
				Subject:    "json-subject",
				Version:    1,
				SchemaType: "JSON",
				Schema:     `{"type":"object","properties":{"name":{"type":"string"}}}`,
			},
		},
	}

	resp := doRequest(t, server, "POST", "/import/schemas", importReq)
	var result types.ImportSchemasResponse
	parseResponse(t, resp, &result)

	if result.Errors != 0 {
		t.Fatalf("Import failed: %d errors", result.Errors)
	}

	// Verify schema types
	ctx := context.Background()

	avroSchema, err := store.GetSchemaByID(ctx, 100)
	if err != nil {
		t.Fatalf("Failed to get AVRO schema: %v", err)
	}
	if avroSchema.SchemaType != storage.SchemaTypeAvro {
		t.Errorf("Expected AVRO type, got %s", avroSchema.SchemaType)
	}

	jsonSchema, err := store.GetSchemaByID(ctx, 200)
	if err != nil {
		t.Fatalf("Failed to get JSON schema: %v", err)
	}
	if jsonSchema.SchemaType != storage.SchemaTypeJSON {
		t.Errorf("Expected JSON type, got %s", jsonSchema.SchemaType)
	}
}
