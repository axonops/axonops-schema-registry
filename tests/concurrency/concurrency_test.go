//go:build concurrency

// Package concurrency provides concurrency tests for the schema registry.
package concurrency

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	// Default values - can be reduced for Cassandra
	numInstances   = 3
	numConcurrent  = 10
	numOperations  = 100
	requestTimeout = 30 * time.Second
)

func init() {
	// Reduce load for Cassandra in CI (single-node with limited resources)
	if os.Getenv("STORAGE_TYPE") == "cassandra" {
		numInstances = 1
		numConcurrent = 5
		numOperations = 20
	}
}

type instance struct {
	server *api.Server
	addr   string
}

var instances []*instance

// Shared HTTP client for all requests (avoids connection churn)
var httpClient = &http.Client{
	Timeout: requestTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
}

// instanceCounter for round-robin selection
var instanceCounter atomic.Uint64

func TestMain(m *testing.M) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Create multiple registry instances sharing the same database
	for i := 0; i < numInstances; i++ {
		store, err := createStorage(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create storage for instance %d: %v\n", i, err)
			os.Exit(1)
		}

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

		reg := registry.New(store, schemaRegistry, compatChecker, "BACKWARD")
		cfg := &config.Config{
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 18081 + i,
			},
		}

		server := api.NewServer(cfg, reg, logger)

		// Start server in background
		go func(port int) {
			addr := fmt.Sprintf(":%d", port)
			if err := http.ListenAndServe(addr, server); err != nil {
				fmt.Fprintf(os.Stderr, "server %s exited: %v\n", addr, err)
			}
		}(cfg.Server.Port)

		instances = append(instances, &instance{
			server: server,
			addr:   fmt.Sprintf("http://localhost:%d", cfg.Server.Port),
		})
	}

	// Wait for servers to start with polling (deadline-based, not fixed sleep)
	// Each instance gets its own deadline to be fair

	// Verify servers are ready before running tests by actually querying the database
	for _, inst := range instances {
		// Per-instance deadline
		deadline := time.Now().Add(30 * time.Second)
		ready := false
		for time.Now().Before(deadline) {
			// Test actual database connectivity by registering a test schema
			// Use __healthcheck__ prefix to identify test pollution
			testSubject := fmt.Sprintf("__healthcheck__-%d", time.Now().UnixNano())
			testSchema := map[string]interface{}{
				"schema": `{"type":"record","name":"HealthCheck","fields":[{"name":"id","type":"int"}]}`,
			}
			resp, err := doRequest("POST", inst.addr+"/subjects/"+testSubject+"/versions", testSchema)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				// Clean up the healthcheck subject (best effort, ignore failures).
				// Note: Not all storage backends may support permanent deletes,
				// so we don't fail if this cleanup doesn't work.
				delResp, _ := doRequest("DELETE", inst.addr+"/subjects/"+testSubject+"?permanent=true", nil)
				if delResp != nil {
					delResp.Body.Close()
				}
				ready = true
				fmt.Fprintf(os.Stderr, "Server %s is ready (database connectivity verified)\n", inst.addr)
				break
			}
			if resp != nil {
				respBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				fmt.Fprintf(os.Stderr, "Server %s health check: status=%d, body=%s\n",
					inst.addr, resp.StatusCode, string(respBody))
			} else if err != nil {
				fmt.Fprintf(os.Stderr, "Server %s health check: %v\n", inst.addr, err)
			}
			time.Sleep(500 * time.Millisecond)
		}
		if !ready {
			fmt.Fprintf(os.Stderr, "Server %s failed to become ready within deadline\n", inst.addr)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "All %d server instances are ready\n", len(instances))

	// Run tests
	code := m.Run()

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
			Hosts:          []string{getEnvOrDefault("CASSANDRA_HOSTS", "localhost")},
			Port:           getEnvOrDefaultInt("CASSANDRA_PORT", 9042),
			Keyspace:       getEnvOrDefault("CASSANDRA_KEYSPACE", "schemaregistry"),
			Consistency:    "ONE", // Use ONE for single-node test cluster (simpler than LOCAL_ONE)
			LocalDC:        "",    // Don't use DC-aware policy for single-node setup
			ConnectTimeout: 30 * time.Second,
			Timeout:        30 * time.Second,
			Migrate:        true,
		}
		return cassandra.NewStore(context.Background(), cfg)

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

// getRandomInstance returns an instance using round-robin selection
func getRandomInstance() *instance {
	idx := instanceCounter.Add(1) % uint64(len(instances))
	return instances[idx]
}

func doRequest(method, url string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	return httpClient.Do(req)
}

// TestConcurrentSchemaRegistration tests registering schemas from multiple instances concurrently
func TestConcurrentSchemaRegistration(t *testing.T) {
	var wg sync.WaitGroup
	var successCount, errorCount int64
	errors := make(chan error, numConcurrent*numOperations)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				inst := getRandomInstance()
				subject := fmt.Sprintf("concurrent-reg-%d-%d-%d", time.Now().UnixNano(), workerID, j)

				schema := map[string]interface{}{
					"schema": fmt.Sprintf(`{"type":"record","name":"Test%d%d","fields":[{"name":"id","type":"int"}]}`, workerID, j),
				}

				resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					errors <- fmt.Errorf("worker %d op %d: %v", workerID, j, err)
					continue
				}

				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
					resp.Body.Close()
				} else {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					atomic.AddInt64(&errorCount, 1)
					errors <- fmt.Errorf("worker %d op %d: status %d, body: %s", workerID, j, resp.StatusCode, string(body))
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	t.Logf("Concurrent registration: %d successes, %d errors", successCount, errorCount)

	// Print first 10 errors
	count := 0
	for err := range errors {
		if count < 10 {
			t.Logf("Error: %v", err)
		}
		count++
	}

	if errorCount > int64(numConcurrent*numOperations/10) {
		t.Errorf("Too many errors: %d out of %d", errorCount, numConcurrent*numOperations)
	}
}

// TestConcurrentVersionUpdates tests concurrent version creation on the same subject.
// This is the key concurrency correctness test - it verifies:
// 1. All concurrent writes succeed (with compat=NONE, all schemas are allowed)
// 2. Versions are contiguous (1..N with no gaps)
// 3. Each version resolves to exactly one schema
// 4. No duplicate versions are created
func TestConcurrentVersionUpdates(t *testing.T) {
	subject := fmt.Sprintf("concurrent-updates-%d", time.Now().UnixNano())

	// Register initial schema (version 1)
	inst := instances[0]
	initialSchema := map[string]interface{}{
		"schema": `{"type":"record","name":"Updates","fields":[{"name":"id","type":"int"}]}`,
	}

	resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", initialSchema)
	if err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}
	resp.Body.Close()

	// Set compatibility to NONE to allow any changes
	configReq := map[string]string{"compatibility": "NONE"}
	resp, err = doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	resp.Body.Close()

	var wg sync.WaitGroup
	var successCount, errorCount int64

	schemaIDs := make(chan int64, numConcurrent)
	errorMsgs := make(chan string, numConcurrent)

	// Multiple workers try to register unique schemas to the same subject
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()

			// Create a unique schema for this worker (backward compatible - adds optional field)
			schema := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"Updates","fields":[{"name":"id","type":"int"},{"name":"worker%d","type":["null","string"],"default":null}]}`, workerID),
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: request error: %v", workerID, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.ID == 0 {
					atomic.AddInt64(&errorCount, 1)
					errorMsgs <- fmt.Sprintf("worker %d: 200 but bad decode: %v", workerID, err)
					return
				}
				atomic.AddInt64(&successCount, 1)
				schemaIDs <- result.ID
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d, body: %s", workerID, resp.StatusCode, string(body))
			}
		}(i)
	}

	wg.Wait()
	close(schemaIDs)
	close(errorMsgs)

	// Log any errors
	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	t.Logf("Version updates: %d successes, %d errors", successCount, errorCount)

	// With NONE compatibility, all writes should succeed
	if errorCount > 0 {
		t.Errorf("Expected 0 errors with NONE compatibility, got %d", errorCount)
	}

	// Collect all returned schema IDs for uniqueness check
	schemaIDSet := make(map[int64]int)
	for id := range schemaIDs {
		schemaIDSet[id]++
	}

	// INVARIANT 0: Number of unique IDs should equal success count (each worker gets unique ID)
	if int64(len(schemaIDSet)) != successCount {
		t.Errorf("Expected %d unique schema IDs, got %d", successCount, len(schemaIDSet))
	}
	for id, count := range schemaIDSet {
		if count > 1 {
			t.Errorf("Schema ID %d was returned %d times (should be unique per schema)", id, count)
		}
	}

	// Verify final state - get all versions
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}
	defer resp.Body.Close()

	var versions []int
	json.NewDecoder(resp.Body).Decode(&versions)
	t.Logf("Final versions: %v", versions)

	// INVARIANT 1: Expected version count = 1 (initial) + successCount
	expectedVersionCount := 1 + int(successCount)
	if len(versions) != expectedVersionCount {
		t.Errorf("Expected %d versions, got %d", expectedVersionCount, len(versions))
	}

	// INVARIANT 2: Versions must be contiguous (1..N with no gaps)
	sort.Ints(versions)
	for i, v := range versions {
		expected := i + 1
		if v != expected {
			t.Errorf("Version gap detected: expected version %d at position %d, got %d. Versions: %v", expected, i, v, versions)
			break
		}
	}

	// INVARIANT 3: Each version resolves to exactly one non-empty schema
	for _, v := range versions {
		resp, err := doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/%d", inst.addr, subject, v), nil)
		if err != nil {
			t.Errorf("Failed to get version %d: %v", v, err)
			continue
		}
		var versionResult struct {
			Schema string `json:"schema"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&versionResult); err != nil {
			resp.Body.Close()
			t.Errorf("Failed to decode version %d: %v", v, err)
			continue
		}
		resp.Body.Close()

		if versionResult.Schema == "" {
			t.Errorf("Version %d has empty schema", v)
		}
	}

	t.Logf("All %d versions verified as contiguous and resolvable with non-empty schemas", len(versions))
}

// TestConcurrentReads tests reading schemas from multiple instances.
// Verifies content on 1-in-10 reads to catch "200 but wrong/empty data" bugs.
func TestConcurrentReads(t *testing.T) {
	subject := fmt.Sprintf("concurrent-reads-%d", time.Now().UnixNano())

	// Register a schema with a known field name for verification
	inst := instances[0]
	expectedFieldName := "data"
	schemaContent := fmt.Sprintf(`{"type":"record","name":"Reads","fields":[{"name":"%s","type":"string"}]}`, expectedFieldName)
	schema := map[string]interface{}{
		"schema": schemaContent,
	}

	resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to register schema: status %d, body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	idVal, ok := result["id"]
	if !ok || idVal == nil {
		t.Fatalf("Expected 'id' in registration response, got: %v", result)
	}
	schemaID := int(idVal.(float64))

	var wg sync.WaitGroup
	var successCount, errorCount, verifyCount, verifyFail int64

	// Multiple workers read the same schema
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				inst := getRandomInstance()

				// Alternate between different read operations
				var resp *http.Response
				var err error
				opType := j % 4

				switch opType {
				case 0:
					resp, err = doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", inst.addr, schemaID), nil)
				case 1:
					resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions/latest", nil)
				case 2:
					resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
				case 3:
					resp, err = doRequest("GET", inst.addr+"/subjects", nil)
				}

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				if resp.StatusCode == http.StatusOK {
					// Verify content on 1-in-10 reads for schema content endpoints
					if j%10 == 0 && (opType == 0 || opType == 1) {
						var readResult struct {
							Schema string `json:"schema"`
						}
						if err := json.NewDecoder(resp.Body).Decode(&readResult); err != nil || readResult.Schema == "" {
							atomic.AddInt64(&verifyFail, 1)
						} else if !strings.Contains(readResult.Schema, expectedFieldName) {
							atomic.AddInt64(&verifyFail, 1)
						} else {
							atomic.AddInt64(&verifyCount, 1)
						}
						// Drain any remaining bytes after decode (deterministic connection reuse)
						_, _ = io.Copy(io.Discard, resp.Body)
					} else if opType == 0 || opType == 1 {
						// Drain body for non-verified schema endpoints to allow connection reuse
						_, _ = io.Copy(io.Discard, resp.Body)
					}
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Concurrent reads: %d successes, %d errors, %d content verifications (%d failed)",
		successCount, errorCount, verifyCount, verifyFail)

	if errorCount > int64(numConcurrent*numOperations/20) {
		t.Errorf("Too many read errors: %d out of %d", errorCount, numConcurrent*numOperations)
	}

	if verifyFail > 0 {
		t.Errorf("Content verification failed %d times (schema content mismatch)", verifyFail)
	}
}

// TestConcurrentMixedOperations tests a mix of reads and writes
func TestConcurrentMixedOperations(t *testing.T) {
	baseSubject := fmt.Sprintf("concurrent-mixed-%d", time.Now().UnixNano())

	var wg sync.WaitGroup
	var readSuccess, writeSuccess, deleteSuccess int64
	var readError, writeError, deleteError int64

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations/3; j++ {
				inst := getRandomInstance()
				subject := fmt.Sprintf("%s-%d-%d", baseSubject, workerID, j)

				// Write
				schema := map[string]interface{}{
					"schema": fmt.Sprintf(`{"type":"record","name":"Mixed%d%d","fields":[{"name":"id","type":"int"}]}`, workerID, j),
				}
				resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
				if err != nil {
					atomic.AddInt64(&writeError, 1)
					continue
				}
				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&writeSuccess, 1)
				} else {
					atomic.AddInt64(&writeError, 1)
				}
				resp.Body.Close()

				// Read
				resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions/latest", nil)
				if err != nil {
					atomic.AddInt64(&readError, 1)
					continue
				}
				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&readSuccess, 1)
				} else {
					atomic.AddInt64(&readError, 1)
				}
				resp.Body.Close()

				// Delete (soft)
				resp, err = doRequest("DELETE", inst.addr+"/subjects/"+subject, nil)
				if err != nil {
					atomic.AddInt64(&deleteError, 1)
					continue
				}
				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&deleteSuccess, 1)
				} else {
					atomic.AddInt64(&deleteError, 1)
				}
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Mixed operations - Writes: %d/%d, Reads: %d/%d, Deletes: %d/%d",
		writeSuccess, writeSuccess+writeError,
		readSuccess, readSuccess+readError,
		deleteSuccess, deleteSuccess+deleteError)
}

// TestConcurrentCompatibilityChecks tests compatibility checking under load
func TestConcurrentCompatibilityChecks(t *testing.T) {
	subject := fmt.Sprintf("concurrent-compat-%d", time.Now().UnixNano())

	// Register initial schema
	inst := instances[0]
	initialSchema := map[string]interface{}{
		"schema": `{"type":"record","name":"Compat","fields":[{"name":"id","type":"int"}]}`,
	}

	resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", initialSchema)
	if err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}
	resp.Body.Close()

	var wg sync.WaitGroup
	var compatibleCount, incompatibleCount, errorCount int64

	// Multiple workers check compatibility
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations/2; j++ {
				inst := getRandomInstance()

				// Alternate between compatible and incompatible schemas
				var schema map[string]interface{}
				if j%2 == 0 {
					// Compatible (adds optional field)
					schema = map[string]interface{}{
						"schema": fmt.Sprintf(`{"type":"record","name":"Compat","fields":[{"name":"id","type":"int"},{"name":"field%d","type":["null","string"],"default":null}]}`, j),
					}
				} else {
					// Incompatible (removes required field)
					schema = map[string]interface{}{
						"schema": `{"type":"record","name":"Compat","fields":[{"name":"different","type":"string"}]}`,
					}
				}

				resp, err := doRequest("POST", inst.addr+"/compatibility/subjects/"+subject+"/versions/latest", schema)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				resp.Body.Close()

				if isCompat, ok := result["is_compatible"].(bool); ok {
					if isCompat {
						atomic.AddInt64(&compatibleCount, 1)
					} else {
						atomic.AddInt64(&incompatibleCount, 1)
					}
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Compatibility checks: %d compatible, %d incompatible, %d errors",
		compatibleCount, incompatibleCount, errorCount)
}

// TestConcurrentConfigUpdates tests updating config from multiple instances
func TestConcurrentConfigUpdates(t *testing.T) {
	subject := fmt.Sprintf("concurrent-config-%d", time.Now().UnixNano())

	// Register a schema first
	inst := instances[0]
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Config","fields":[{"name":"id","type":"int"}]}`,
	}
	resp, _ := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
	resp.Body.Close()

	compatLevels := []string{"NONE", "BACKWARD", "FORWARD", "FULL", "BACKWARD_TRANSITIVE", "FORWARD_TRANSITIVE", "FULL_TRANSITIVE"}

	var wg sync.WaitGroup
	var successCount, errorCount int64

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numOperations/5; j++ {
				inst := getRandomInstance()
				level := compatLevels[(workerID+j)%len(compatLevels)]

				configReq := map[string]string{"compatibility": level}
				resp, err := doRequest("PUT", inst.addr+"/config/"+subject, configReq)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Config updates: %d successes, %d errors", successCount, errorCount)
}

// TestDataConsistency verifies data written by one instance can be read by another.
// Uses polling with deadline instead of fixed sleep for reliability.
func TestDataConsistency(t *testing.T) {
	subject := fmt.Sprintf("consistency-%d", time.Now().UnixNano())

	// Write from instance 0
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Consistency","fields":[{"name":"data","type":"bytes"}]}`,
	}

	resp, err := doRequest("POST", instances[0].addr+"/subjects/"+subject+"/versions", schema)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to write schema: status %d, body: %s", resp.StatusCode, body)
	}

	var writeResult struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&writeResult); err != nil || writeResult.ID == 0 {
		t.Fatalf("Failed to decode write response: %v", err)
	}
	schemaID := writeResult.ID

	// Read from all other instances with polling
	// Each instance gets its own deadline for fairness
	for i := 1; i < len(instances); i++ {
		deadline := time.Now().Add(5 * time.Second)
		var lastErr error
		success := false

		for time.Now().Before(deadline) {
			resp, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", instances[i].addr, schemaID), nil)
			if err != nil {
				lastErr = err
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				var readResult struct {
					Schema string `json:"schema"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&readResult); err == nil && readResult.Schema != "" {
					resp.Body.Close()
					success = true
					break
				}
				resp.Body.Close()
			} else {
				resp.Body.Close()
				lastErr = fmt.Errorf("status %d", resp.StatusCode)
			}
			time.Sleep(100 * time.Millisecond)
		}

		if !success {
			t.Errorf("Instance %d failed to read schema %d within deadline: %v", i, schemaID, lastErr)
		}
	}

	t.Logf("Data consistency verified: schema %d readable from all %d instances", schemaID, len(instances))
}

// avroField represents a field in an Avro schema for JSON parsing.
type avroField struct {
	Name string `json:"name"`
}

// avroSchema represents an Avro record schema for JSON parsing.
type avroSchema struct {
	Type   string      `json:"type"`
	Name   string      `json:"name"`
	Fields []avroField `json:"fields"`
}

// extractMarkerFromSchema parses an Avro schema JSON and returns the first field name.
// This is used to deterministically extract the worker marker from schemas.
// Validates that the schema matches our expected test format (record with exactly 1 field).
func extractMarkerFromSchema(schemaJSON string) (string, error) {
	var s avroSchema
	if err := json.Unmarshal([]byte(schemaJSON), &s); err != nil {
		return "", err
	}
	if s.Type != "record" {
		return "", fmt.Errorf("unexpected type %q (expected record)", s.Type)
	}
	if len(s.Fields) != 1 {
		return "", fmt.Errorf("expected 1 field, got %d", len(s.Fields))
	}
	if s.Fields[0].Name == "" {
		return "", fmt.Errorf("field name is empty")
	}
	return s.Fields[0].Name, nil
}

// TestHotSubjectContention is the critical test for monotonic version allocation.
// It hammers a single subject with many concurrent writes and verifies strict invariants:
// 1. All writes succeed (with NONE compatibility)
// 2. Versions are contiguous (1..N with no gaps)
// 3. Each worker's schema marker appears exactly once
// 4. Schema ID -> payload mapping is correct via GET /schemas/ids/{id}
func TestHotSubjectContention(t *testing.T) {
	subject := fmt.Sprintf("hot-subject-%d", time.Now().UnixNano())
	numWriters := numConcurrent * 2 // Double the writers for more contention

	// Set compatibility to NONE upfront
	inst := instances[0]
	configReq := map[string]string{"compatibility": "NONE"}
	resp, err := doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	resp.Body.Close()

	var wg sync.WaitGroup
	var successCount, errorCount int64

	// Track which schema each worker submitted (for verification)
	type writeResult struct {
		workerID     int
		schemaID     int64
		schemaMarker string // The unique "worker_N" marker in the schema
	}
	results := make(chan writeResult, numWriters)
	errorMsgs := make(chan string, numWriters)

	// All workers POST different schemas to the same subject simultaneously
	// Use "worker_N" format to avoid false-match (e.g., "w1" matching in "w12")
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()
			marker := fmt.Sprintf("worker_%d", workerID)
			schema := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"Hot","fields":[{"name":"%s","type":"int"}]}`, marker),
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", workerID, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.ID == 0 {
					atomic.AddInt64(&errorCount, 1)
					errorMsgs <- fmt.Sprintf("worker %d: 200 but bad decode: %v", workerID, err)
					return
				}
				atomic.AddInt64(&successCount, 1)
				results <- writeResult{workerID: workerID, schemaID: result.ID, schemaMarker: marker}
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d: %s", workerID, resp.StatusCode, string(body))
			}
		}(i)
	}

	wg.Wait()
	close(results)
	close(errorMsgs)

	// Log errors
	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	t.Logf("Hot subject: %d successes, %d errors out of %d writers", successCount, errorCount, numWriters)

	// All writes should succeed with NONE compatibility
	if errorCount > 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}

	// Collect schema IDs and markers for verification, and build ID->marker map
	schemaIDSet := make(map[int64]int)        // schemaID -> count
	submittedMarkers := make(map[string]bool) // markers that were submitted
	idToMarker := make(map[int64]string)      // for verifying GET /schemas/ids/{id}
	for r := range results {
		schemaIDSet[r.schemaID]++
		submittedMarkers[r.schemaMarker] = true
		idToMarker[r.schemaID] = r.schemaMarker
	}

	// INVARIANT 1: Each schema ID should be unique (different schemas = different IDs)
	for id, count := range schemaIDSet {
		if count > 1 {
			t.Errorf("Schema ID %d was returned %d times (should be unique per schema)", id, count)
		}
	}

	// INVARIANT 1b: Number of unique IDs should equal success count
	if int64(len(schemaIDSet)) != successCount {
		t.Errorf("Expected %d unique schema IDs, got %d", successCount, len(schemaIDSet))
	}

	// Verify versions are contiguous
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}
	defer resp.Body.Close()

	var versions []int
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		t.Fatalf("Failed to decode versions: %v", err)
	}
	sort.Ints(versions)

	// INVARIANT 2: Versions must be exactly 1..N with no gaps
	if len(versions) != int(successCount) {
		t.Errorf("Expected %d versions, got %d", successCount, len(versions))
	}

	for i, v := range versions {
		expected := i + 1
		if v != expected {
			t.Errorf("Version gap: expected %d at position %d, got %d", expected, i, v)
			break
		}
	}

	// INVARIANT 3: Each version returns a valid schema with exactly one of the submitted markers
	// Use exact JSON parsing to avoid O(n²) and false-match issues
	markerCounts := make(map[string]int) // marker -> count (should all be 1)
	for _, v := range versions {
		resp, err := doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/%d", inst.addr, subject, v), nil)
		if err != nil {
			t.Errorf("Failed to get version %d: %v", v, err)
			continue
		}
		var versionResult struct {
			Schema string `json:"schema"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&versionResult); err != nil {
			resp.Body.Close()
			t.Errorf("Failed to decode version %d: %v", v, err)
			continue
		}
		resp.Body.Close()

		if versionResult.Schema == "" {
			t.Errorf("Version %d has empty schema", v)
			continue
		}

		// Extract marker using exact JSON parsing (O(1) per version, no false-match)
		marker, err := extractMarkerFromSchema(versionResult.Schema)
		if err != nil {
			t.Errorf("Version %d: failed to extract marker: %v", v, err)
			continue
		}

		if !submittedMarkers[marker] {
			t.Errorf("Version %d has unknown marker %q", v, marker)
			continue
		}

		markerCounts[marker]++
	}

	// Verify exactly-once: each submitted marker appears exactly once
	// Iterate over submittedMarkers to catch missing markers explicitly
	for marker := range submittedMarkers {
		count := markerCounts[marker]
		if count != 1 {
			t.Errorf("Marker %q count=%d (expected exactly 1)", marker, count)
		}
	}

	// INVARIANT 4: Verify schema ID -> payload mapping via GET /schemas/ids/{id}
	// This catches bugs where responses return unique IDs but store wrong payloads
	for schemaID, expectedMarker := range idToMarker {
		resp, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", inst.addr, schemaID), nil)
		if err != nil {
			t.Errorf("Failed to GET /schemas/ids/%d: %v", schemaID, err)
			continue
		}
		var idResult struct {
			Schema string `json:"schema"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&idResult); err != nil {
			resp.Body.Close()
			t.Errorf("Failed to decode schema ID %d: %v", schemaID, err)
			continue
		}
		resp.Body.Close()

		actualMarker, err := extractMarkerFromSchema(idResult.Schema)
		if err != nil {
			t.Errorf("Schema ID %d: failed to extract marker: %v", schemaID, err)
			continue
		}

		if actualMarker != expectedMarker {
			t.Errorf("Schema ID %d: expected marker %q, got %q (ID->payload mismatch)", schemaID, expectedMarker, actualMarker)
		}
	}

	t.Logf("Verified %d contiguous versions with %d unique schema IDs, %d unique markers, and ID->payload mapping",
		len(versions), len(schemaIDSet), len(markerCounts))
}

// TestSchemaIdempotency verifies that posting the same schema multiple times
// returns the same ID/version (Confluent-like deduplication behavior).
// This test documents the expected contract: identical schema content to the same
// subject returns the same schema ID and does not create duplicate versions.
func TestSchemaIdempotency(t *testing.T) {
	subject := fmt.Sprintf("idempotency-%d", time.Now().UnixNano())
	numWriters := numConcurrent

	// The exact same schema for all workers
	schemaContent := `{"type":"record","name":"Idempotent","fields":[{"name":"id","type":"int"}]}`
	schema := map[string]interface{}{
		"schema": schemaContent,
	}

	var wg sync.WaitGroup
	var successCount, errorCount int64

	schemaIDs := make(chan int64, numWriters)
	errorMsgs := make(chan string, numWriters)

	// All workers POST the exact same schema
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()
			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", workerID, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.ID == 0 {
					atomic.AddInt64(&errorCount, 1)
					errorMsgs <- fmt.Sprintf("worker %d: 200 but bad decode: %v", workerID, err)
					return
				}
				atomic.AddInt64(&successCount, 1)
				schemaIDs <- result.ID
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d: %s", workerID, resp.StatusCode, string(body))
			}
		}(i)
	}

	wg.Wait()
	close(schemaIDs)
	close(errorMsgs)

	// Log errors
	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// Collect all schema IDs
	var allIDs []int64
	for id := range schemaIDs {
		allIDs = append(allIDs, id)
	}

	if len(allIDs) == 0 {
		t.Fatal("No successful registrations")
	}

	// INVARIANT: All results should have the same schema ID (deduplication)
	expectedID := allIDs[0]
	for i, id := range allIDs {
		if id != expectedID {
			t.Errorf("Result %d has schemaID %d, expected %d (same schema should return same ID)",
				i, id, expectedID)
		}
	}

	// Verify only one version was created
	inst := instances[0]
	resp, err := doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}
	defer resp.Body.Close()

	var versions []int
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		t.Fatalf("Failed to decode versions: %v", err)
	}

	// INVARIANT: Should be exactly 1 version (deduplication)
	if len(versions) != 1 {
		t.Errorf("Expected 1 version (idempotent registration), got %d: %v", len(versions), versions)
	}

	t.Logf("Idempotency verified: %d concurrent writes all returned schema ID %d, 1 version created",
		len(allIDs), expectedID)
}

// TestSchemaIDUniqueness verifies that different schemas get different IDs.
// This tests global schema ID allocation across multiple subjects.
func TestSchemaIDUniqueness(t *testing.T) {
	var wg sync.WaitGroup
	numSchemas := numConcurrent * 2

	schemaIDs := make(chan int64, numSchemas)
	errorMsgs := make(chan string, numSchemas)
	var successCount, errorCount int64

	// Each worker registers a unique schema under its own subject
	for i := 0; i < numSchemas; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()
			subject := fmt.Sprintf("unique-id-%d-%d", time.Now().UnixNano(), workerID)
			schema := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"Unique%d","fields":[{"name":"id","type":"int"}]}`, workerID),
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", workerID, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.ID == 0 {
					atomic.AddInt64(&errorCount, 1)
					errorMsgs <- fmt.Sprintf("worker %d: 200 but bad decode: %v", workerID, err)
					return
				}
				atomic.AddInt64(&successCount, 1)
				schemaIDs <- result.ID
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d: %s", workerID, resp.StatusCode, string(body))
			}
		}(i)
	}

	wg.Wait()
	close(schemaIDs)
	close(errorMsgs)

	// Log errors
	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// Collect all IDs
	idSet := make(map[int64]int)
	for id := range schemaIDs {
		idSet[id]++
	}

	// INVARIANT: Each ID should appear only once (unique per schema)
	duplicates := 0
	for id, count := range idSet {
		if count > 1 {
			t.Errorf("Schema ID %d was assigned %d times (should be unique)", id, count)
			duplicates++
		}
	}

	if duplicates > 0 {
		t.Errorf("Found %d duplicate schema IDs", duplicates)
	}

	t.Logf("Schema ID uniqueness verified: %d unique IDs allocated, %d successes, %d errors",
		len(idSet), successCount, errorCount)
}
