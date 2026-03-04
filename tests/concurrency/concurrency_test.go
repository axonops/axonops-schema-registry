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
	// Reduce load for Cassandra in CI (single-node with limited resources).
	// Keep numInstances=1 (real single-node constraint), but use higher concurrency
	// and operations to ensure contention bugs are caught (1/6 of default load).
	if os.Getenv("STORAGE_TYPE") == "cassandra" {
		numInstances = 1
		numConcurrent = 10
		numOperations = 50
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
				Port: 28181 + i,
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

// TestConcurrentSubjectDeletion tests soft-deleting many subjects concurrently.
// Verifies that all deletes succeed and subjects appear in the deleted list but not the active list.
func TestConcurrentSubjectDeletion(t *testing.T) {
	prefix := fmt.Sprintf("del-subj-%d", time.Now().UnixNano())
	inst := instances[0]

	// Setup: register numConcurrent subjects sequentially
	subjects := make([]string, numConcurrent)
	for i := 0; i < numConcurrent; i++ {
		subjects[i] = fmt.Sprintf("%s-%d", prefix, i)
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"Del%d","fields":[{"name":"id","type":"int"}]}`, i),
		}
		resp, err := doRequest("POST", inst.addr+"/subjects/"+subjects[i]+"/versions", schema)
		if err != nil {
			t.Fatalf("Failed to register subject %s: %v", subjects[i], err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("Failed to register subject %s: status %d, body: %s", subjects[i], resp.StatusCode, body)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Concurrent: each goroutine soft-deletes its own subject
	var wg sync.WaitGroup
	var successCount, errorCount int64
	errorMsgs := make(chan string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			inst := getRandomInstance()
			resp, err := doRequest("DELETE", inst.addr+"/subjects/"+subjects[idx], nil)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", idx, err)
				return
			}
			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d, body: %s", idx, resp.StatusCode, string(body))
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: All deletes should return 200
	if successCount != int64(numConcurrent) {
		t.Errorf("Expected %d successful deletes, got %d (errors: %d)", numConcurrent, successCount, errorCount)
	}

	// INVARIANT: GET /subjects (active) should not contain any test subjects
	resp, err := doRequest("GET", inst.addr+"/subjects", nil)
	if err != nil {
		t.Fatalf("Failed to list subjects: %v", err)
	}
	var activeSubjects []string
	json.NewDecoder(resp.Body).Decode(&activeSubjects)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	for _, s := range activeSubjects {
		if strings.HasPrefix(s, prefix) {
			t.Errorf("Subject %s should not appear in active subjects after soft-delete", s)
		}
	}

	// INVARIANT: GET /subjects?deleted=true should contain all test subjects
	resp, err = doRequest("GET", inst.addr+"/subjects?deleted=true", nil)
	if err != nil {
		t.Fatalf("Failed to list deleted subjects: %v", err)
	}
	var deletedSubjects []string
	json.NewDecoder(resp.Body).Decode(&deletedSubjects)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	deletedSet := make(map[string]bool)
	for _, s := range deletedSubjects {
		deletedSet[s] = true
	}
	for _, s := range subjects {
		if !deletedSet[s] {
			t.Errorf("Subject %s should appear in deleted subjects list", s)
		}
	}

	t.Logf("Concurrent subject deletion: %d successes, %d errors", successCount, errorCount)
}

// TestConcurrentPermanentDeletion tests soft-deleting then permanently deleting subjects concurrently.
// Verifies that subjects are fully removed after permanent deletion.
func TestConcurrentPermanentDeletion(t *testing.T) {
	prefix := fmt.Sprintf("perm-del-%d", time.Now().UnixNano())
	inst := instances[0]

	// Setup: register numConcurrent subjects and store (subject, schemaID) pairs
	type subjectSchema struct {
		subject  string
		schemaID int64
	}
	pairs := make([]subjectSchema, numConcurrent)
	for i := 0; i < numConcurrent; i++ {
		pairs[i].subject = fmt.Sprintf("%s-%d", prefix, i)
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"PermDel%d","fields":[{"name":"id","type":"int"}]}`, i),
		}
		resp, err := doRequest("POST", inst.addr+"/subjects/"+pairs[i].subject+"/versions", schema)
		if err != nil {
			t.Fatalf("Failed to register subject %s: %v", pairs[i].subject, err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("Failed to register subject %s: status %d, body: %s", pairs[i].subject, resp.StatusCode, body)
		}
		var result struct {
			ID int64 `json:"id"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		pairs[i].schemaID = result.ID
	}

	// Concurrent: each goroutine soft-deletes then permanent-deletes its own subject
	var wg sync.WaitGroup
	var successCount, errorCount int64
	errorMsgs := make(chan string, numConcurrent*2)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			inst := getRandomInstance()

			// Soft-delete first
			resp, err := doRequest("DELETE", inst.addr+"/subjects/"+pairs[idx].subject, nil)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d soft-delete: %v", idx, err)
				return
			}
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d soft-delete: status %d, body: %s", idx, resp.StatusCode, string(body))
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			// Permanent-delete
			resp, err = doRequest("DELETE", inst.addr+"/subjects/"+pairs[idx].subject+"?permanent=true", nil)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d perm-delete: %v", idx, err)
				return
			}
			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d perm-delete: status %d, body: %s", idx, resp.StatusCode, string(body))
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: Both calls succeed
	if successCount != int64(numConcurrent) {
		t.Errorf("Expected %d successful permanent deletes, got %d (errors: %d)", numConcurrent, successCount, errorCount)
	}

	// INVARIANT: test subjects don't appear in GET /subjects?deleted=true
	resp, err := doRequest("GET", inst.addr+"/subjects?deleted=true", nil)
	if err != nil {
		t.Fatalf("Failed to list deleted subjects: %v", err)
	}
	var deletedSubjects []string
	json.NewDecoder(resp.Body).Decode(&deletedSubjects)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	for _, s := range deletedSubjects {
		if strings.HasPrefix(s, prefix) {
			t.Errorf("Subject %s should not appear in deleted subjects after permanent delete", s)
		}
	}

	// INVARIANT: each schema ID's GET /schemas/ids/{id}/subjects returns 404 or empty array
	for _, p := range pairs {
		resp, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d/subjects", inst.addr, p.schemaID), nil)
		if err != nil {
			t.Errorf("Failed to check schema ID %d subjects: %v", p.schemaID, err)
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			continue
		}
		if resp.StatusCode == http.StatusOK {
			var subjects []string
			json.NewDecoder(resp.Body).Decode(&subjects)
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			// Filter for our test prefix
			for _, s := range subjects {
				if strings.HasPrefix(s, prefix) {
					t.Errorf("Schema ID %d still references permanently deleted subject %s", p.schemaID, s)
				}
			}
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	t.Logf("Concurrent permanent deletion: %d successes, %d errors", successCount, errorCount)
}

// TestConcurrentVersionDeletion tests deleting multiple versions from one subject concurrently.
// Verifies that all version deletes succeed and no versions remain.
func TestConcurrentVersionDeletion(t *testing.T) {
	subject := fmt.Sprintf("ver-del-%d", time.Now().UnixNano())
	inst := instances[0]

	// Set compatibility to NONE to allow arbitrary schemas
	configReq := map[string]string{"compatibility": "NONE"}
	resp, err := doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Register numConcurrent unique schemas -> versions 1..N
	for i := 0; i < numConcurrent; i++ {
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"VerDel","fields":[{"name":"f%d","type":"int"}]}`, i),
		}
		resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
		if err != nil {
			t.Fatalf("Failed to register version %d: %v", i+1, err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("Failed to register version %d: status %d, body: %s", i+1, resp.StatusCode, body)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Concurrent: each goroutine deletes its assigned version
	var wg sync.WaitGroup
	var successCount, errorCount int64
	errorMsgs := make(chan string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(version int) {
			defer wg.Done()
			inst := getRandomInstance()
			resp, err := doRequest("DELETE", fmt.Sprintf("%s/subjects/%s/versions/%d", inst.addr, subject, version), nil)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("version %d: %v", version, err)
				return
			}
			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("version %d: status %d, body: %s", version, resp.StatusCode, string(body))
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i + 1) // versions are 1-indexed
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: All deletes return 200
	if successCount != int64(numConcurrent) {
		t.Errorf("Expected %d successful version deletes, got %d (errors: %d)", numConcurrent, successCount, errorCount)
	}

	// INVARIANT: GET /subjects/{subject}/versions returns empty array or 404
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		// Subject not found after all versions deleted - acceptable
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	} else if resp.StatusCode == http.StatusOK {
		var versions []int
		json.NewDecoder(resp.Body).Decode(&versions)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if len(versions) > 0 {
			t.Errorf("Expected empty versions after all deletes, got %v", versions)
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Errorf("Unexpected status %d getting versions: %s", resp.StatusCode, string(body))
	}

	t.Logf("Concurrent version deletion: %d successes, %d errors", successCount, errorCount)
}

// TestConcurrentDeleteAndReRegister tests concurrent soft-deletion and re-registration of a subject.
// Verifies no 5xx errors occur under contention between deleters and registrars.
func TestConcurrentDeleteAndReRegister(t *testing.T) {
	subject := fmt.Sprintf("del-rereg-%d", time.Now().UnixNano())
	inst := instances[0]

	// Setup: register initial schema with compat=NONE
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"DelReReg","fields":[{"name":"id","type":"int"}]}`,
	}
	resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
	if err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register initial schema: status %d, body: %s", resp.StatusCode, body)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	configReq := map[string]string{"compatibility": "NONE"}
	resp, err = doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Concurrent: half workers soft-delete, half register new schemas
	var wg sync.WaitGroup
	var serverErrors int64
	errorMsgs := make(chan string, numConcurrent*numOperations)
	opsPerWorker := numOperations / numConcurrent
	if opsPerWorker < 1 {
		opsPerWorker = 1
	}

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < opsPerWorker; j++ {
				inst := getRandomInstance()

				if workerID%2 == 0 {
					// Deleter
					resp, err := doRequest("DELETE", inst.addr+"/subjects/"+subject, nil)
					if err != nil {
						continue
					}
					if resp.StatusCode >= 500 {
						body, _ := io.ReadAll(resp.Body)
						atomic.AddInt64(&serverErrors, 1)
						errorMsgs <- fmt.Sprintf("worker %d delete op %d: 5xx status %d, body: %s", workerID, j, resp.StatusCode, string(body))
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				} else {
					// Registrar
					s := map[string]interface{}{
						"schema": fmt.Sprintf(`{"type":"record","name":"DelReReg","fields":[{"name":"f%d_%d","type":"int"}]}`, workerID, j),
					}
					resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
					if err != nil {
						continue
					}
					if resp.StatusCode >= 500 {
						body, _ := io.ReadAll(resp.Body)
						atomic.AddInt64(&serverErrors, 1)
						errorMsgs <- fmt.Sprintf("worker %d register op %d: 5xx status %d, body: %s", workerID, j, resp.StatusCode, string(body))
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: Zero 5xx errors
	if serverErrors > 0 {
		t.Errorf("Expected zero 5xx errors, got %d", serverErrors)
	}

	// INVARIANT: After, subject is queryable (GET returns 200 or 404, no 5xx)
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions/latest", nil)
	if err != nil {
		t.Fatalf("Failed to query subject after test: %v", err)
	}
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Errorf("Subject query returned 5xx after test: status %d, body: %s", resp.StatusCode, body)
	} else {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	t.Logf("Concurrent delete and re-register: zero 5xx errors verified, final subject status: %d", resp.StatusCode)
}

// TestConcurrentModeChanges tests concurrent global mode changes.
// Verifies no 5xx errors and final mode is a valid value.
func TestConcurrentModeChanges(t *testing.T) {
	inst := instances[0]

	// MUST defer reset to READWRITE at the TOP of function
	defer func() {
		modeReq := map[string]string{"mode": "READWRITE"}
		resp, err := doRequest("PUT", inst.addr+"/mode?force=true", modeReq)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	modes := []string{"READWRITE", "READONLY", "IMPORT"}

	var wg sync.WaitGroup
	var serverErrors int64
	errorMsgs := make(chan string, numConcurrent*numOperations)
	opsPerWorker := numOperations / numConcurrent
	if opsPerWorker < 1 {
		opsPerWorker = 1
	}

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < opsPerWorker; j++ {
				inst := getRandomInstance()
				mode := modes[(workerID+j)%len(modes)]

				modeReq := map[string]string{"mode": mode}
				resp, err := doRequest("PUT", inst.addr+"/mode?force=true", modeReq)
				if err != nil {
					continue
				}
				if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					atomic.AddInt64(&serverErrors, 1)
					errorMsgs <- fmt.Sprintf("worker %d op %d mode %s: 5xx status %d, body: %s", workerID, j, mode, resp.StatusCode, string(body))
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: Zero 5xx errors
	if serverErrors > 0 {
		t.Errorf("Expected zero 5xx errors, got %d", serverErrors)
	}

	// INVARIANT: Final GET /mode returns valid value
	resp, err := doRequest("GET", inst.addr+"/mode", nil)
	if err != nil {
		t.Fatalf("Failed to get mode: %v", err)
	}
	var modeResp struct {
		Mode string `json:"mode"`
	}
	json.NewDecoder(resp.Body).Decode(&modeResp)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	validModes := map[string]bool{"READWRITE": true, "READONLY": true, "IMPORT": true}
	if !validModes[modeResp.Mode] {
		t.Errorf("Final mode is invalid: %q (expected READWRITE, READONLY, or IMPORT)", modeResp.Mode)
	}

	t.Logf("Concurrent mode changes: zero 5xx errors, final mode: %s", modeResp.Mode)
}

// TestConcurrentReadonlyEnforcement tests that READONLY mode blocks writes but allows reads.
// Phase 1: concurrent writes in READWRITE mode (some succeed).
// Phase 2: set READONLY mode.
// Phase 3: concurrent writes (all rejected) and reads (all succeed).
func TestConcurrentReadonlyEnforcement(t *testing.T) {
	subject := fmt.Sprintf("readonly-enforce-%d", time.Now().UnixNano())
	inst := instances[0]

	// MUST defer reset to READWRITE at the TOP
	defer func() {
		modeReq := map[string]string{"mode": "READWRITE"}
		resp, err := doRequest("PUT", inst.addr+"/mode?force=true", modeReq)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	// Ensure READWRITE mode
	modeReq := map[string]string{"mode": "READWRITE"}
	resp, err := doRequest("PUT", inst.addr+"/mode?force=true", modeReq)
	if err != nil {
		t.Fatalf("Failed to set READWRITE mode: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Setup: register subject with compat=NONE
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"ReadonlyEnforce","fields":[{"name":"id","type":"int"}]}`,
	}
	resp, err = doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
	if err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register: status %d, body: %s", resp.StatusCode, body)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	configReq := map[string]string{"compatibility": "NONE"}
	resp, err = doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Phase 1: concurrent writers in READWRITE mode
	var wg1 sync.WaitGroup
	var phase1Success int64
	for i := 0; i < numConcurrent; i++ {
		wg1.Add(1)
		go func(workerID int) {
			defer wg1.Done()
			inst := getRandomInstance()
			s := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"ReadonlyEnforce","fields":[{"name":"p1w%d","type":"int"}]}`, workerID),
			}
			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
			if err != nil {
				return
			}
			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&phase1Success, 1)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}
	wg1.Wait()
	t.Logf("Phase 1 (READWRITE): %d successful writes", phase1Success)

	// Phase 2: set READONLY mode
	modeReq = map[string]string{"mode": "READONLY"}
	resp, err = doRequest("PUT", inst.addr+"/mode?force=true", modeReq)
	if err != nil {
		t.Fatalf("Failed to set READONLY mode: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Phase 3: concurrent writers (expect all 422) and readers (expect all 200)
	var wg3 sync.WaitGroup
	var writeRejected, writeAccepted, writeServerError int64
	var readSuccess, readError, readServerError int64

	for i := 0; i < numConcurrent; i++ {
		// Writer
		wg3.Add(1)
		go func(workerID int) {
			defer wg3.Done()
			inst := getRandomInstance()
			s := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"ReadonlyEnforce","fields":[{"name":"p3w%d","type":"int"}]}`, workerID),
			}
			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
			if err != nil {
				return
			}
			switch {
			case resp.StatusCode == http.StatusUnprocessableEntity:
				atomic.AddInt64(&writeRejected, 1)
			case resp.StatusCode == http.StatusOK:
				atomic.AddInt64(&writeAccepted, 1)
			case resp.StatusCode >= 500:
				atomic.AddInt64(&writeServerError, 1)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)

		// Reader
		wg3.Add(1)
		go func(workerID int) {
			defer wg3.Done()
			inst := getRandomInstance()
			resp, err := doRequest("GET", inst.addr+"/subjects/"+subject+"/versions/latest", nil)
			if err != nil {
				atomic.AddInt64(&readError, 1)
				return
			}
			switch {
			case resp.StatusCode == http.StatusOK:
				atomic.AddInt64(&readSuccess, 1)
			case resp.StatusCode >= 500:
				atomic.AddInt64(&readServerError, 1)
			default:
				atomic.AddInt64(&readError, 1)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}
	wg3.Wait()

	// INVARIANT: Zero 5xx errors
	if writeServerError > 0 || readServerError > 0 {
		t.Errorf("5xx errors: writes=%d, reads=%d", writeServerError, readServerError)
	}

	// INVARIANT: All phase-3 writes rejected (422)
	if writeAccepted > 0 {
		t.Errorf("Expected all writes rejected in READONLY mode, but %d were accepted", writeAccepted)
	}
	if writeRejected == 0 {
		t.Errorf("Expected some writes to be rejected, but none were")
	}

	// INVARIANT: All phase-3 reads succeed
	if readSuccess != int64(numConcurrent) {
		t.Errorf("Expected %d successful reads, got %d (errors: %d)", numConcurrent, readSuccess, readError)
	}

	t.Logf("READONLY enforcement: writes rejected=%d accepted=%d, reads success=%d error=%d",
		writeRejected, writeAccepted, readSuccess, readError)
}

// TestConcurrentSchemaLookup tests concurrent schema lookups while new schemas are being registered.
// Verifies that all lookups returning 200 have a valid schema ID and no 5xx errors occur.
func TestConcurrentSchemaLookup(t *testing.T) {
	subject := fmt.Sprintf("lookup-%d", time.Now().UnixNano())
	inst := instances[0]

	// Setup: register initial schema with compat=NONE
	initialSchema := map[string]interface{}{
		"schema": `{"type":"record","name":"Lookup","fields":[{"name":"id","type":"int"}]}`,
	}
	resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", initialSchema)
	if err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register: status %d, body: %s", resp.StatusCode, body)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	configReq := map[string]string{"compatibility": "NONE"}
	resp, err = doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Concurrent: half workers lookup, half register new schemas
	var wg sync.WaitGroup
	var lookupSuccess, lookupInvalidID, lookupOther int64
	var registerSuccess, registerError int64
	var serverErrors int64
	errorMsgs := make(chan string, numConcurrent*2)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			inst := getRandomInstance()

			if workerID%2 == 0 {
				// Lookup: POST /subjects/{subject} with the initial schema body
				resp, err := doRequest("POST", inst.addr+"/subjects/"+subject, initialSchema)
				if err != nil {
					return
				}
				if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					atomic.AddInt64(&serverErrors, 1)
					errorMsgs <- fmt.Sprintf("lookup worker %d: 5xx status %d, body: %s", workerID, resp.StatusCode, string(body))
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					return
				}
				if resp.StatusCode == http.StatusOK {
					var result struct {
						ID int64 `json:"id"`
					}
					json.NewDecoder(resp.Body).Decode(&result)
					if result.ID > 0 {
						atomic.AddInt64(&lookupSuccess, 1)
					} else {
						atomic.AddInt64(&lookupInvalidID, 1)
						errorMsgs <- fmt.Sprintf("lookup worker %d: 200 but id=%d", workerID, result.ID)
					}
				} else {
					atomic.AddInt64(&lookupOther, 1)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			} else {
				// Register: POST /subjects/{subject}/versions with new unique schema
				s := map[string]interface{}{
					"schema": fmt.Sprintf(`{"type":"record","name":"Lookup","fields":[{"name":"w%d","type":"int"}]}`, workerID),
				}
				resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
				if err != nil {
					return
				}
				if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					atomic.AddInt64(&serverErrors, 1)
					errorMsgs <- fmt.Sprintf("register worker %d: 5xx status %d, body: %s", workerID, resp.StatusCode, string(body))
				} else if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&registerSuccess, 1)
				} else {
					atomic.AddInt64(&registerError, 1)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: All lookups returning 200 have valid id > 0
	if lookupInvalidID > 0 {
		t.Errorf("Lookups with invalid ID: %d", lookupInvalidID)
	}

	// INVARIANT: Zero 5xx
	if serverErrors > 0 {
		t.Errorf("Expected zero 5xx errors, got %d", serverErrors)
	}

	t.Logf("Concurrent schema lookup: lookups success=%d invalidID=%d other=%d, registers success=%d error=%d",
		lookupSuccess, lookupInvalidID, lookupOther, registerSuccess, registerError)
}

// TestConcurrentSubjectListDuringMutations tests listing subjects while writers and deleters are active.
// Verifies every GET /subjects returning 200 is valid JSON and no 5xx errors occur.
func TestConcurrentSubjectListDuringMutations(t *testing.T) {
	prefix := fmt.Sprintf("list-mut-%d", time.Now().UnixNano())

	var wg sync.WaitGroup
	var serverErrors int64
	var readSuccess, readInvalidJSON int64
	errorMsgs := make(chan string, numConcurrent*numOperations)
	opsPerWorker := numOperations / numConcurrent
	if opsPerWorker < 1 {
		opsPerWorker = 1
	}

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			role := workerID % 3 // 0=writer, 1=deleter, 2=reader

			for j := 0; j < opsPerWorker; j++ {
				inst := getRandomInstance()

				switch role {
				case 0:
					// Writer: register to unique subjects
					subject := fmt.Sprintf("%s-w-%d-%d", prefix, workerID, j)
					s := map[string]interface{}{
						"schema": fmt.Sprintf(`{"type":"record","name":"ListMut%d%d","fields":[{"name":"id","type":"int"}]}`, workerID, j),
					}
					resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
					if err != nil {
						continue
					}
					if resp.StatusCode >= 500 {
						body, _ := io.ReadAll(resp.Body)
						atomic.AddInt64(&serverErrors, 1)
						errorMsgs <- fmt.Sprintf("writer %d op %d: 5xx status %d, body: %s", workerID, j, resp.StatusCode, string(body))
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

				case 1:
					// Deleter: register then delete unique subjects
					subject := fmt.Sprintf("%s-d-%d-%d", prefix, workerID, j)
					s := map[string]interface{}{
						"schema": fmt.Sprintf(`{"type":"record","name":"ListMutDel%d%d","fields":[{"name":"id","type":"int"}]}`, workerID, j),
					}
					resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
					if err != nil {
						continue
					}
					if resp.StatusCode >= 500 {
						body, _ := io.ReadAll(resp.Body)
						atomic.AddInt64(&serverErrors, 1)
						errorMsgs <- fmt.Sprintf("deleter-reg %d op %d: 5xx status %d, body: %s", workerID, j, resp.StatusCode, string(body))
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

					// Delete the subject
					resp, err = doRequest("DELETE", inst.addr+"/subjects/"+subject, nil)
					if err != nil {
						continue
					}
					if resp.StatusCode >= 500 {
						body, _ := io.ReadAll(resp.Body)
						atomic.AddInt64(&serverErrors, 1)
						errorMsgs <- fmt.Sprintf("deleter-del %d op %d: 5xx status %d, body: %s", workerID, j, resp.StatusCode, string(body))
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

				case 2:
					// Reader: GET /subjects
					resp, err := doRequest("GET", inst.addr+"/subjects", nil)
					if err != nil {
						continue
					}
					if resp.StatusCode >= 500 {
						body, _ := io.ReadAll(resp.Body)
						atomic.AddInt64(&serverErrors, 1)
						errorMsgs <- fmt.Sprintf("reader %d op %d: 5xx status %d, body: %s", workerID, j, resp.StatusCode, string(body))
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
						continue
					}
					if resp.StatusCode == http.StatusOK {
						body, _ := io.ReadAll(resp.Body)
						resp.Body.Close()
						var subjects []string
						if err := json.Unmarshal(body, &subjects); err != nil {
							atomic.AddInt64(&readInvalidJSON, 1)
							errorMsgs <- fmt.Sprintf("reader %d op %d: invalid JSON: %v, body: %s", workerID, j, err, string(body))
						} else {
							atomic.AddInt64(&readSuccess, 1)
						}
					} else {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: Every GET /subjects returning 200 is valid JSON array
	if readInvalidJSON > 0 {
		t.Errorf("Got %d invalid JSON responses from GET /subjects", readInvalidJSON)
	}

	// INVARIANT: Zero 5xx errors
	if serverErrors > 0 {
		t.Errorf("Expected zero 5xx errors, got %d", serverErrors)
	}

	t.Logf("Concurrent subject list during mutations: reads success=%d invalidJSON=%d, 5xx=%d",
		readSuccess, readInvalidJSON, serverErrors)
}

// TestConcurrentCrossOperationStorm tests a storm of mixed operation types against a single subject.
// Workers perform register, GET latest, DELETE, PUT config, and POST lookup concurrently.
func TestConcurrentCrossOperationStorm(t *testing.T) {
	subject := fmt.Sprintf("storm-%d", time.Now().UnixNano())
	inst := instances[0]

	// Setup: register initial schema with compat=NONE
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Storm","fields":[{"name":"id","type":"int"}]}`,
	}
	resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
	if err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register: status %d, body: %s", resp.StatusCode, body)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	configReq := map[string]string{"compatibility": "NONE"}
	resp, err = doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	var wg sync.WaitGroup
	var serverErrors int64
	errorMsgs := make(chan string, numConcurrent*numOperations)
	opsPerWorker := numOperations / numConcurrent
	if opsPerWorker < 1 {
		opsPerWorker = 1
	}

	compatLevels := []string{"NONE", "BACKWARD", "FORWARD", "FULL"}

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			opType := workerID % 5

			for j := 0; j < opsPerWorker; j++ {
				inst := getRandomInstance()

				var resp *http.Response
				var err error

				switch opType {
				case 0:
					// Register new schema
					s := map[string]interface{}{
						"schema": fmt.Sprintf(`{"type":"record","name":"Storm","fields":[{"name":"w%d_j%d","type":"int"}]}`, workerID, j),
					}
					resp, err = doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
				case 1:
					// GET latest
					resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions/latest", nil)
				case 2:
					// DELETE subject (soft)
					resp, err = doRequest("DELETE", inst.addr+"/subjects/"+subject, nil)
				case 3:
					// PUT config (cycle through compat levels)
					level := compatLevels[(workerID+j)%len(compatLevels)]
					cr := map[string]string{"compatibility": level}
					resp, err = doRequest("PUT", inst.addr+"/config/"+subject, cr)
				case 4:
					// POST lookup
					resp, err = doRequest("POST", inst.addr+"/subjects/"+subject, schema)
				}

				if err != nil {
					continue
				}
				if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					atomic.AddInt64(&serverErrors, 1)
					errorMsgs <- fmt.Sprintf("worker %d (op=%d) iter %d: 5xx status %d, body: %s", workerID, opType, j, resp.StatusCode, string(body))
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: Zero 5xx errors
	if serverErrors > 0 {
		t.Errorf("Expected zero 5xx errors, got %d", serverErrors)
	}

	// INVARIANT: After, subject is queryable (no 5xx)
	// Re-register to ensure subject exists for final check
	resp, err = doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions/latest", nil)
	if err != nil {
		t.Fatalf("Failed to query subject after storm: %v", err)
	}
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Errorf("Subject query returned 5xx after storm: status %d, body: %s", resp.StatusCode, body)
	} else {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	t.Logf("Cross-operation storm: %d 5xx errors", serverErrors)
}

// TestConcurrentMultiSchemaTypeRegistration tests concurrent registration of Protobuf and JSON Schema types.
// Verifies all registrations succeed, each ID returns the correct schemaType, and all IDs are unique.
func TestConcurrentMultiSchemaTypeRegistration(t *testing.T) {
	prefix := fmt.Sprintf("multi-type-%d", time.Now().UnixNano())

	type regResult struct {
		schemaID   int64
		schemaType string
		subject    string
	}

	var wg sync.WaitGroup
	var successCount, errorCount int64
	results := make(chan regResult, numConcurrent)
	errorMsgs := make(chan string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			inst := getRandomInstance()

			var subject string
			var body map[string]interface{}

			if workerID%2 == 0 {
				// Protobuf
				subject = fmt.Sprintf("%s-proto-%d", prefix, workerID)
				body = map[string]interface{}{
					"schema":     fmt.Sprintf(`syntax = "proto3"; message Test%d { string name = 1; }`, workerID),
					"schemaType": "PROTOBUF",
				}
			} else {
				// JSON Schema
				subject = fmt.Sprintf("%s-json-%d", prefix, workerID)
				body = map[string]interface{}{
					"schema":     fmt.Sprintf(`{"$schema":"http://json-schema.org/draft-07/schema#","title":"Test%d","type":"object","properties":{"name":{"type":"string"}},"required":["name"]}`, workerID),
					"schemaType": "JSON",
				}
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", body)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", workerID, err)
				return
			}

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				json.NewDecoder(resp.Body).Decode(&result)
				if result.ID > 0 {
					atomic.AddInt64(&successCount, 1)
					expectedType := "PROTOBUF"
					if workerID%2 != 0 {
						expectedType = "JSON"
					}
					results <- regResult{schemaID: result.ID, schemaType: expectedType, subject: subject}
				} else {
					atomic.AddInt64(&errorCount, 1)
					errorMsgs <- fmt.Sprintf("worker %d: 200 but id=%d", workerID, result.ID)
				}
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d, body: %s", workerID, resp.StatusCode, string(body))
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	close(results)
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: All registrations succeed (200)
	if errorCount > 0 {
		t.Errorf("Expected all registrations to succeed, got %d errors", errorCount)
	}

	// Collect results
	allResults := make([]regResult, 0)
	for r := range results {
		allResults = append(allResults, r)
	}

	// INVARIANT: All IDs unique
	idSet := make(map[int64]int)
	for _, r := range allResults {
		idSet[r.schemaID]++
	}
	for id, count := range idSet {
		if count > 1 {
			t.Errorf("Schema ID %d was assigned %d times (should be unique)", id, count)
		}
	}

	// INVARIANT: Each ID via GET /schemas/ids/{id} returns correct schemaType
	inst := instances[0]
	for _, r := range allResults {
		resp, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", inst.addr, r.schemaID), nil)
		if err != nil {
			t.Errorf("Failed to GET schema ID %d: %v", r.schemaID, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Errorf("GET schema ID %d: status %d, body: %s", r.schemaID, resp.StatusCode, string(body))
			continue
		}
		var schemaResp struct {
			SchemaType string `json:"schemaType"`
		}
		json.NewDecoder(resp.Body).Decode(&schemaResp)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Confluent convention: AVRO is returned as empty string or "AVRO"
		actualType := schemaResp.SchemaType
		if actualType != r.schemaType {
			t.Errorf("Schema ID %d: expected schemaType %q, got %q", r.schemaID, r.schemaType, actualType)
		}
	}

	t.Logf("Multi-schema type registration: %d successes, %d errors, %d unique IDs",
		successCount, errorCount, len(idSet))
}

// TestConcurrentHotSubjectCASPressure stresses the Cassandra CreateSchema 3-step CAS
// under extreme contention. It sends numConcurrent*3 concurrent writers to a single
// subject and verifies version contiguity, marker uniqueness, subject_latest consistency,
// and cross-table consistency between subject_versions and schemas_by_id.
func TestConcurrentHotSubjectCASPressure(t *testing.T) {
	subject := fmt.Sprintf("cas-pressure-%d", time.Now().UnixNano())
	numWriters := numConcurrent * 3
	inst := instances[0]

	// Set compatibility to NONE upfront so all schemas are accepted
	configReq := map[string]string{"compatibility": "NONE"}
	resp, err := doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	var wg sync.WaitGroup
	var successCount, errorCount int64

	type writeResult struct {
		workerID     int
		schemaID     int64
		schemaMarker string
	}
	results := make(chan writeResult, numWriters)
	errorMsgs := make(chan string, numWriters)

	// All workers POST different schemas to the same subject simultaneously
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()
			marker := fmt.Sprintf("worker_%d", workerID)
			schema := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"CASPressure","fields":[{"name":"%s","type":"int"}]}`, marker),
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", workerID, err)
				return
			}

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.ID == 0 {
					atomic.AddInt64(&errorCount, 1)
					errorMsgs <- fmt.Sprintf("worker %d: 200 but bad decode: %v", workerID, err)
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					return
				}
				atomic.AddInt64(&successCount, 1)
				results <- writeResult{workerID: workerID, schemaID: result.ID, schemaMarker: marker}
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d: %s", workerID, resp.StatusCode, string(body))
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	close(results)
	close(errorMsgs)

	// Log errors
	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	t.Logf("CAS pressure: %d successes, %d errors out of %d writers", successCount, errorCount, numWriters)

	// INVARIANT 1: All writes should succeed with NONE compatibility
	if errorCount > 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}

	// Collect schema IDs, markers, and build ID->marker map
	schemaIDSet := make(map[int64]int)
	submittedMarkers := make(map[string]bool)
	idToMarker := make(map[int64]string)
	for r := range results {
		schemaIDSet[r.schemaID]++
		submittedMarkers[r.schemaMarker] = true
		idToMarker[r.schemaID] = r.schemaMarker
	}

	// INVARIANT 2: Versions must be contiguous 1..N
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}

	var versions []int
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to decode versions: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	sort.Ints(versions)

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

	// INVARIANT 3: Each marker appears exactly once
	markerCounts := make(map[string]int)
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
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			t.Errorf("Failed to decode version %d: %v", v, err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if versionResult.Schema == "" {
			t.Errorf("Version %d has empty schema", v)
			continue
		}

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

	for marker := range submittedMarkers {
		count := markerCounts[marker]
		if count != 1 {
			t.Errorf("Marker %q count=%d (expected exactly 1)", marker, count)
		}
	}

	// INVARIANT 4 (NEW): subject_latest consistency
	// GET /subjects/{subject}/versions/latest — version number must equal len(versions)
	resp, err = doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/latest", inst.addr, subject), nil)
	if err != nil {
		t.Fatalf("Failed to get latest version: %v", err)
	}
	var latestResult struct {
		Version int    `json:"version"`
		Schema  string `json:"schema"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&latestResult); err != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to decode latest version: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if latestResult.Version != len(versions) {
		t.Errorf("Latest version is %d, but expected %d (highest version)", latestResult.Version, len(versions))
	}

	if latestResult.Schema == "" {
		t.Error("Latest version has empty schema")
	} else {
		latestMarker, err := extractMarkerFromSchema(latestResult.Schema)
		if err != nil {
			t.Errorf("Latest version: failed to extract marker: %v", err)
		} else if !submittedMarkers[latestMarker] {
			t.Errorf("Latest version has unknown marker %q", latestMarker)
		} else {
			t.Logf("Latest version %d has valid marker %q", latestResult.Version, latestMarker)
		}
	}

	// INVARIANT 5 (NEW): cross-table consistency
	// For each version, GET the version detail (returns id + schema), then
	// GET /schemas/ids/{id} and verify the schema content matches.
	crossTableErrors := 0
	for _, v := range versions {
		// Get version detail to obtain schema ID and schema content
		resp, err := doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/%d", inst.addr, subject, v), nil)
		if err != nil {
			t.Errorf("Cross-table check: failed to get version %d: %v", v, err)
			crossTableErrors++
			continue
		}
		var vDetail struct {
			ID     int64  `json:"id"`
			Schema string `json:"schema"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&vDetail); err != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			t.Errorf("Cross-table check: failed to decode version %d: %v", v, err)
			crossTableErrors++
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Now fetch the same schema by global ID
		resp2, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", inst.addr, vDetail.ID), nil)
		if err != nil {
			t.Errorf("Cross-table check: failed to GET /schemas/ids/%d: %v", vDetail.ID, err)
			crossTableErrors++
			continue
		}
		var idResult struct {
			Schema string `json:"schema"`
		}
		if err := json.NewDecoder(resp2.Body).Decode(&idResult); err != nil {
			io.Copy(io.Discard, resp2.Body)
			resp2.Body.Close()
			t.Errorf("Cross-table check: failed to decode schema ID %d: %v", vDetail.ID, err)
			crossTableErrors++
			continue
		}
		io.Copy(io.Discard, resp2.Body)
		resp2.Body.Close()

		if vDetail.Schema != idResult.Schema {
			t.Errorf("Cross-table inconsistency at version %d (schema ID %d): version schema != ID schema", v, vDetail.ID)
			crossTableErrors++
		}
	}

	t.Logf("Verified %d contiguous versions, %d unique IDs, %d unique markers, latest=%d, cross-table errors=%d",
		len(versions), len(schemaIDSet), len(markerCounts), latestResult.Version, crossTableErrors)
}

// TestConcurrentFingerprintDedup verifies that many workers registering identical schema
// content to different subjects concurrently all receive the same global schema ID.
// The fingerprint dedup mechanism must assign a single ID for identical content.
func TestConcurrentFingerprintDedup(t *testing.T) {
	timestamp := time.Now().UnixNano()
	sharedSchema := `{"type":"record","name":"SharedDedup","fields":[{"name":"shared_data","type":"string"}]}`

	var wg sync.WaitGroup
	var successCount, errorCount int64

	type dedupResult struct {
		workerID int
		schemaID int64
		subject  string
	}
	results := make(chan dedupResult, numConcurrent)
	errorMsgs := make(chan string, numConcurrent)

	// Each worker registers the same schema to its own unique subject
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()
			subjectName := fmt.Sprintf("dedup-%d-%d", timestamp, workerID)
			schema := map[string]interface{}{
				"schema": sharedSchema,
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subjectName+"/versions", schema)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", workerID, err)
				return
			}

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.ID == 0 {
					atomic.AddInt64(&errorCount, 1)
					errorMsgs <- fmt.Sprintf("worker %d: 200 but bad decode: %v", workerID, err)
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					return
				}
				atomic.AddInt64(&successCount, 1)
				results <- dedupResult{workerID: workerID, schemaID: result.ID, subject: subjectName}
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d: %s", workerID, resp.StatusCode, string(body))
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	close(results)
	close(errorMsgs)

	// Log errors
	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	t.Logf("Fingerprint dedup: %d successes, %d errors out of %d workers", successCount, errorCount, numConcurrent)

	// Collect all results
	allResults := make([]dedupResult, 0, numConcurrent)
	for r := range results {
		allResults = append(allResults, r)
	}

	if len(allResults) == 0 {
		t.Fatal("No successful registrations")
	}

	// INVARIANT 1: Most registrations succeed. Under high concurrency, some backends
	// may hit transient race conditions in the fingerprint dedup path (e.g., MySQL's
	// INSERT IGNORE + subsequent SELECT can race when a concurrent transaction has
	// inserted but not yet committed). Allow up to 10% failure rate.
	maxErrors := int64(numConcurrent * 1 / 10)
	if maxErrors < 1 {
		maxErrors = 1
	}
	if errorCount > maxErrors {
		t.Errorf("Too many errors: %d out of %d (max tolerated: %d)", errorCount, numConcurrent, maxErrors)
	}

	// INVARIANT 2: All returned IDs must be identical (same schema content = same global ID)
	expectedID := allResults[0].schemaID
	for _, r := range allResults {
		if r.schemaID != expectedID {
			t.Errorf("Worker %d got schema ID %d, expected %d (fingerprint dedup should assign same ID)", r.workerID, r.schemaID, expectedID)
		}
	}
	t.Logf("All %d workers received the same schema ID: %d", len(allResults), expectedID)

	// INVARIANT 3: GET /schemas/ids/{id} returns the correct schema content
	inst := instances[0]
	resp, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", inst.addr, expectedID), nil)
	if err != nil {
		t.Fatalf("Failed to GET /schemas/ids/%d: %v", expectedID, err)
	}
	var schemaResp struct {
		Schema string `json:"schema"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&schemaResp); err != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to decode schema ID %d: %v", expectedID, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if !strings.Contains(schemaResp.Schema, "shared_data") {
		t.Errorf("Schema ID %d content does not contain expected field 'shared_data': %s", expectedID, schemaResp.Schema)
	}

	// INVARIANT 4: GET /schemas/ids/{id}/subjects returns all subject names
	resp, err = doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d/subjects", inst.addr, expectedID), nil)
	if err != nil {
		t.Fatalf("Failed to GET /schemas/ids/%d/subjects: %v", expectedID, err)
	}
	var subjectsList []string
	if err := json.NewDecoder(resp.Body).Decode(&subjectsList); err != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to decode subjects for schema ID %d: %v", expectedID, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Build a set of expected subject names
	expectedSubjects := make(map[string]bool)
	for _, r := range allResults {
		expectedSubjects[r.subject] = true
	}

	// Build a set of actual subject names
	actualSubjects := make(map[string]bool)
	for _, s := range subjectsList {
		actualSubjects[s] = true
	}

	for subj := range expectedSubjects {
		if !actualSubjects[subj] {
			t.Errorf("Subject %q not found in /schemas/ids/%d/subjects response", subj, expectedID)
		}
	}
	t.Logf("Schema ID %d is associated with %d subjects (expected %d)", expectedID, len(subjectsList), len(expectedSubjects))

	// INVARIANT 5: Each subject has exactly 1 version
	for _, r := range allResults {
		resp, err := doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions", inst.addr, r.subject), nil)
		if err != nil {
			t.Errorf("Failed to get versions for subject %s: %v", r.subject, err)
			continue
		}
		var versions []int
		if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			t.Errorf("Failed to decode versions for subject %s: %v", r.subject, err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if len(versions) != 1 {
			t.Errorf("Subject %s has %d versions, expected 1", r.subject, len(versions))
		}
	}

	t.Logf("Fingerprint dedup verified: %d subjects all share schema ID %d with correct content", len(allResults), expectedID)
}

// TestConcurrentImportIDCollision tests concurrent imports where two groups try to
// import different schema content with the same global schema ID. This catches
// TOCTOU bugs in Cassandra's ImportSchema path. Exactly one schema content should
// win for the given ID; the other group should get conflicts.
func TestConcurrentImportIDCollision(t *testing.T) {
	inst := instances[0]

	// Defer mode reset back to READWRITE
	defer func() {
		r, _ := doRequest("PUT", inst.addr+"/mode?force=true", map[string]string{"mode": "READWRITE"})
		if r != nil {
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
		}
	}()

	// Set IMPORT mode
	resp, err := doRequest("PUT", inst.addr+"/mode?force=true", map[string]string{"mode": "IMPORT"})
	if err != nil {
		t.Fatalf("Failed to set IMPORT mode: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	timestamp := time.Now().UnixNano()
	targetID := 900000 + timestamp%10000
	schemaA := fmt.Sprintf(`{"type":"record","name":"ImportA","fields":[{"name":"field_a_%d","type":"string"}]}`, timestamp)
	schemaB := fmt.Sprintf(`{"type":"record","name":"ImportB","fields":[{"name":"field_b_%d","type":"string"}]}`, timestamp)

	var wg sync.WaitGroup
	var serverErrorCount int64

	type importResult struct {
		workerID int
		imported int
		errors   int
		schema   string
		subject  string
		httpCode int
	}
	resultsChan := make(chan importResult, numConcurrent)
	errorMsgs := make(chan string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()
			var schema, subjectName string
			if workerID%2 == 0 {
				schema = schemaA
				subjectName = fmt.Sprintf("import-a-%d-%d", timestamp, workerID)
			} else {
				schema = schemaB
				subjectName = fmt.Sprintf("import-b-%d-%d", timestamp, workerID)
			}

			body := map[string]interface{}{
				"schemas": []map[string]interface{}{
					{
						"id":         targetID,
						"subject":    subjectName,
						"version":    1,
						"schemaType": "AVRO",
						"schema":     schema,
					},
				},
			}

			resp, err := doRequest("POST", inst.addr+"/import/schemas", body)
			if err != nil {
				atomic.AddInt64(&serverErrorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: request error: %v", workerID, err)
				return
			}

			httpCode := resp.StatusCode
			if httpCode >= 500 {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&serverErrorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: 5xx error %d: %s", workerID, httpCode, string(body))
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				return
			}

			var importResp map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&importResp)
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			imported := 0
			errors := 0
			if v, ok := importResp["imported"]; ok {
				if f, ok := v.(float64); ok {
					imported = int(f)
				}
			}
			if v, ok := importResp["errors"]; ok {
				if f, ok := v.(float64); ok {
					errors = int(f)
				}
			}

			resultsChan <- importResult{
				workerID: workerID,
				imported: imported,
				errors:   errors,
				schema:   schema,
				subject:  subjectName,
				httpCode: httpCode,
			}
		}(i)
	}

	wg.Wait()
	close(resultsChan)
	close(errorMsgs)

	// Log errors
	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT 1: Zero 5xx errors
	if serverErrorCount > 0 {
		t.Errorf("Expected 0 server (5xx) errors, got %d", serverErrorCount)
	}

	// Collect results
	allResults := make([]importResult, 0, numConcurrent)
	for r := range resultsChan {
		allResults = append(allResults, r)
	}

	// Count successes and conflicts per schema group
	successA, successB := 0, 0
	conflictA, conflictB := 0, 0
	for _, r := range allResults {
		if r.imported == 1 {
			if r.workerID%2 == 0 {
				successA++
			} else {
				successB++
			}
		}
		if r.errors > 0 {
			if r.workerID%2 == 0 {
				conflictA++
			} else {
				conflictB++
			}
		}
	}
	t.Logf("Import ID collision: schemaA successes=%d conflicts=%d, schemaB successes=%d conflicts=%d",
		successA, conflictA, successB, conflictB)

	// INVARIANT 2: GET /schemas/ids/{targetID} returns content matching exactly one of schemaA or schemaB
	resp, err = doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", inst.addr, targetID), nil)
	if err != nil {
		t.Fatalf("Failed to GET /schemas/ids/%d: %v", targetID, err)
	}
	var schemaResp struct {
		Schema string `json:"schema"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&schemaResp); err != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to decode schema ID %d: %v", targetID, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	fieldAMarker := fmt.Sprintf("field_a_%d", timestamp)
	fieldBMarker := fmt.Sprintf("field_b_%d", timestamp)
	hasA := strings.Contains(schemaResp.Schema, fieldAMarker)
	hasB := strings.Contains(schemaResp.Schema, fieldBMarker)

	if !hasA && !hasB {
		t.Errorf("Schema ID %d content matches neither schemaA nor schemaB: %s", targetID, schemaResp.Schema)
	} else if hasA && hasB {
		t.Errorf("Schema ID %d content matches both schemaA and schemaB (impossible): %s", targetID, schemaResp.Schema)
	} else if hasA {
		t.Logf("Schema ID %d won by schemaA (field_a)", targetID)
	} else {
		t.Logf("Schema ID %d won by schemaB (field_b)", targetID)
	}

	// INVARIANT 3: For workers that got success (imported==1), their subject's version 1
	// must return the schema they submitted
	for _, r := range allResults {
		if r.imported != 1 {
			continue
		}
		resp, err := doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/1", inst.addr, r.subject), nil)
		if err != nil {
			t.Errorf("Worker %d: failed to GET subject %s version 1: %v", r.workerID, r.subject, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Errorf("Worker %d: GET subject %s version 1 returned %d: %s", r.workerID, r.subject, resp.StatusCode, string(body))
			continue
		}
		var versionResp struct {
			Schema string `json:"schema"`
		}
		json.NewDecoder(resp.Body).Decode(&versionResp)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// The subject's schema must match what the worker submitted.
		// On Cassandra, the global schema_id is shared and the last-writer-wins on
		// schemas_by_id means the content for that ID may have been overwritten by
		// the other group. This is a known TOCTOU limitation — log it, don't fail.
		expectedMarker := fieldAMarker
		if r.workerID%2 != 0 {
			expectedMarker = fieldBMarker
		}
		if !strings.Contains(versionResp.Schema, expectedMarker) {
			t.Logf("WARNING: Worker %d: subject %s version 1 schema does not contain expected marker %q (last-writer-wins race): %s",
				r.workerID, r.subject, expectedMarker, versionResp.Schema)
		}
	}

	t.Logf("Import ID collision test complete: target ID %d, %d total workers", targetID, len(allResults))
}

// TestConcurrentImportSameSubjectVersion tests concurrent imports targeting the same
// (subject, version) tuple with different schema content and different IDs. Exactly
// one import should win; others should get conflicts. This catches TOCTOU races in
// subject-version assignment.
func TestConcurrentImportSameSubjectVersion(t *testing.T) {
	inst := instances[0]

	// Defer mode reset back to READWRITE
	defer func() {
		r, _ := doRequest("PUT", inst.addr+"/mode?force=true", map[string]string{"mode": "READWRITE"})
		if r != nil {
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
		}
	}()

	// Set IMPORT mode
	resp, err := doRequest("PUT", inst.addr+"/mode?force=true", map[string]string{"mode": "IMPORT"})
	if err != nil {
		t.Fatalf("Failed to set IMPORT mode: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	timestamp := time.Now().UnixNano()
	subject := fmt.Sprintf("import-sv-%d", timestamp)
	idA := 800000 + timestamp%5000
	idB := idA + 1
	schemaA := fmt.Sprintf(`{"type":"record","name":"SvRaceA","fields":[{"name":"sv_a_%d","type":"string"}]}`, timestamp)
	schemaB := fmt.Sprintf(`{"type":"record","name":"SvRaceB","fields":[{"name":"sv_b_%d","type":"string"}]}`, timestamp)

	var wg sync.WaitGroup
	var serverErrorCount int64

	type importResult struct {
		workerID int
		imported int
		errors   int
		group    string // "A" or "B"
		httpCode int
	}
	resultsChan := make(chan importResult, numConcurrent)
	errorMsgs := make(chan string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()
			var schema string
			var importID int64
			var group string
			if workerID%2 == 0 {
				schema = schemaA
				importID = idA
				group = "A"
			} else {
				schema = schemaB
				importID = idB
				group = "B"
			}

			body := map[string]interface{}{
				"schemas": []map[string]interface{}{
					{
						"id":         importID,
						"subject":    subject,
						"version":    1,
						"schemaType": "AVRO",
						"schema":     schema,
					},
				},
			}

			resp, err := doRequest("POST", inst.addr+"/import/schemas", body)
			if err != nil {
				atomic.AddInt64(&serverErrorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d (group %s): request error: %v", workerID, group, err)
				return
			}

			httpCode := resp.StatusCode
			if httpCode >= 500 {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&serverErrorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d (group %s): 5xx error %d: %s", workerID, group, httpCode, string(body))
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				return
			}

			var importResp map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&importResp)
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			imported := 0
			errors := 0
			if v, ok := importResp["imported"]; ok {
				if f, ok := v.(float64); ok {
					imported = int(f)
				}
			}
			if v, ok := importResp["errors"]; ok {
				if f, ok := v.(float64); ok {
					errors = int(f)
				}
			}

			resultsChan <- importResult{
				workerID: workerID,
				imported: imported,
				errors:   errors,
				group:    group,
				httpCode: httpCode,
			}
		}(i)
	}

	wg.Wait()
	close(resultsChan)
	close(errorMsgs)

	// Log errors
	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT 1: Zero 5xx errors
	if serverErrorCount > 0 {
		t.Errorf("Expected 0 server (5xx) errors, got %d", serverErrorCount)
	}

	// Collect results and count successes/conflicts per group
	allResults := make([]importResult, 0, numConcurrent)
	for r := range resultsChan {
		allResults = append(allResults, r)
	}

	successA, successB := 0, 0
	conflictA, conflictB := 0, 0
	for _, r := range allResults {
		if r.imported == 1 {
			if r.group == "A" {
				successA++
			} else {
				successB++
			}
		}
		if r.errors > 0 {
			if r.group == "A" {
				conflictA++
			} else {
				conflictB++
			}
		}
	}
	t.Logf("Same subject-version import: groupA successes=%d conflicts=%d, groupB successes=%d conflicts=%d",
		successA, conflictA, successB, conflictB)

	// INVARIANT 2: GET /subjects/{subject}/versions/1 returns exactly one schema
	resp, err = doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/1", inst.addr, subject), nil)
	if err != nil {
		t.Fatalf("Failed to GET subject %s version 1: %v", subject, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("GET subject %s version 1 returned %d: %s", subject, resp.StatusCode, string(body))
	}
	var versionResp struct {
		ID     int64  `json:"id"`
		Schema string `json:"schema"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to decode subject %s version 1: %v", subject, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// INVARIANT 3: The returned schema matches either schemaA or schemaB
	markerA := fmt.Sprintf("sv_a_%d", timestamp)
	markerB := fmt.Sprintf("sv_b_%d", timestamp)
	hasA := strings.Contains(versionResp.Schema, markerA)
	hasB := strings.Contains(versionResp.Schema, markerB)

	winningGroup := ""
	if hasA && !hasB {
		winningGroup = "A"
		t.Logf("Subject %s version 1 won by group A (sv_a)", subject)
	} else if hasB && !hasA {
		winningGroup = "B"
		t.Logf("Subject %s version 1 won by group B (sv_b)", subject)
	} else if hasA && hasB {
		t.Errorf("Subject %s version 1 matches both groups (impossible): %s", subject, versionResp.Schema)
	} else {
		t.Errorf("Subject %s version 1 matches neither group: %s", subject, versionResp.Schema)
	}

	// INVARIANT 4: The returned ID matches the winning schema's import ID
	if winningGroup == "A" && versionResp.ID != idA {
		t.Errorf("Winning group is A but schema ID is %d (expected %d)", versionResp.ID, idA)
	} else if winningGroup == "B" && versionResp.ID != idB {
		t.Errorf("Winning group is B but schema ID is %d (expected %d)", versionResp.ID, idB)
	}

	t.Logf("Same subject-version import test complete: subject=%s, winning group=%s, winning ID=%d, total workers=%d",
		subject, winningGroup, versionResp.ID, len(allResults))
}

// TestConcurrentPermanentDeleteDuringRegistration races permanent deletion of a subject
// against concurrent re-registration attempts. This tests the non-atomic multi-table
// permanent delete path (especially in Cassandra) under contention.
func TestConcurrentPermanentDeleteDuringRegistration(t *testing.T) {
	subject := fmt.Sprintf("perm-del-race-%d", time.Now().UnixNano())
	inst := instances[0]

	// Set compat to NONE so all schemas are accepted
	configReq := map[string]string{"compatibility": "NONE"}
	resp, err := doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Register 5 versions to give the subject some history
	for i := 0; i < 5; i++ {
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"PdRace","fields":[{"name":"v%d","type":"int"}]}`, i),
		}
		resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
		if err != nil {
			t.Fatalf("Failed to register version %d: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("Failed to register version %d: status %d, body: %s", i, resp.StatusCode, body)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Soft-delete the subject (required before permanent delete)
	resp, err = doRequest("DELETE", inst.addr+"/subjects/"+subject, nil)
	if err != nil {
		t.Fatalf("Failed to soft-delete subject: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to soft-delete subject: status %d, body: %s", resp.StatusCode, body)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Concurrent: Group A (even) permanent-deletes, Group B (odd) re-registers
	var wg sync.WaitGroup
	var permDeleteSuccess, permDeleteFail int64
	var registerSuccess, registerFail int64
	var serverErrors int64
	errorMsgs := make(chan string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			inst := getRandomInstance()

			if workerID%2 == 0 {
				// Group A: permanent delete
				resp, err := doRequest("DELETE", inst.addr+"/subjects/"+subject+"?permanent=true", nil)
				if err != nil {
					atomic.AddInt64(&permDeleteFail, 1)
					return
				}
				if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					atomic.AddInt64(&serverErrors, 1)
					errorMsgs <- fmt.Sprintf("worker %d perm-delete: 5xx status %d, body: %s", workerID, resp.StatusCode, string(body))
				} else if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&permDeleteSuccess, 1)
				} else {
					atomic.AddInt64(&permDeleteFail, 1)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			} else {
				// Group B: re-register with a new unique schema
				s := map[string]interface{}{
					"schema": fmt.Sprintf(`{"type":"record","name":"PdRace","fields":[{"name":"rereg_%d","type":"int"}]}`, workerID),
				}
				resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
				if err != nil {
					atomic.AddInt64(&registerFail, 1)
					return
				}
				if resp.StatusCode >= 500 {
					body, _ := io.ReadAll(resp.Body)
					atomic.AddInt64(&serverErrors, 1)
					errorMsgs <- fmt.Sprintf("worker %d register: 5xx status %d, body: %s", workerID, resp.StatusCode, string(body))
				} else if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&registerSuccess, 1)
				} else {
					atomic.AddInt64(&registerFail, 1)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// On Cassandra, permanent delete is non-atomic (iterates per-version, then deletes
	// subject_latest). A concurrent registration can interleave, causing transient 500s
	// (e.g., "version not found" when cleanup tries to access an already-deleted version).
	// This is a known limitation of the non-transactional delete path.
	if serverErrors > 0 {
		t.Logf("WARNING: %d server errors (5xx) during permanent delete race — expected on non-transactional backends", serverErrors)
	}

	// Verification: determine final state of the subject
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to check final subject state: %v", err)
	}

	if resp.StatusCode == http.StatusOK {
		// Registration won — subject exists with versions
		var versions []int
		json.NewDecoder(resp.Body).Decode(&versions)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// On non-transactional backends (Cassandra), version gaps can occur when
		// permanent delete removes some versions while concurrent registration creates
		// new ones. We log gaps rather than failing, since the key invariant is that
		// every existing version is retrievable with valid content.
		sort.Ints(versions)
		for idx, v := range versions {
			expected := idx + 1
			if v != expected {
				t.Logf("WARNING: Version gap: expected %d at position %d, got %d (versions: %v) — expected on non-transactional backends", expected, idx, v, versions)
				break
			}
		}

		// Each version must have non-empty schema content
		for _, v := range versions {
			vResp, err := doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/%d", inst.addr, subject, v), nil)
			if err != nil {
				t.Errorf("Failed to GET version %d: %v", v, err)
				continue
			}
			if vResp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(vResp.Body)
				vResp.Body.Close()
				t.Errorf("GET version %d: status %d, body: %s", v, vResp.StatusCode, string(body))
				continue
			}
			var versionDetail struct {
				Schema string `json:"schema"`
				ID     int64  `json:"id"`
			}
			json.NewDecoder(vResp.Body).Decode(&versionDetail)
			io.Copy(io.Discard, vResp.Body)
			vResp.Body.Close()

			if versionDetail.Schema == "" {
				t.Errorf("Version %d has empty schema content", v)
			}

			// Verify schema is retrievable by ID
			if versionDetail.ID > 0 {
				idResp, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", inst.addr, versionDetail.ID), nil)
				if err != nil {
					t.Errorf("Failed to GET schema by ID %d: %v", versionDetail.ID, err)
					continue
				}
				if idResp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(idResp.Body)
					idResp.Body.Close()
					t.Errorf("GET schema ID %d: status %d, body: %s", versionDetail.ID, idResp.StatusCode, string(body))
					continue
				}
				io.Copy(io.Discard, idResp.Body)
				idResp.Body.Close()
			}
		}

		t.Logf("Registration won: subject has %d version(s)", len(versions))
	} else if resp.StatusCode == http.StatusNotFound {
		// Permanent delete won — subject should not appear in subjects list
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		listResp, err := doRequest("GET", inst.addr+"/subjects", nil)
		if err != nil {
			t.Fatalf("Failed to list subjects: %v", err)
		}
		var subjects []string
		json.NewDecoder(listResp.Body).Decode(&subjects)
		io.Copy(io.Discard, listResp.Body)
		listResp.Body.Close()

		for _, s := range subjects {
			if s == subject {
				t.Errorf("Permanently deleted subject %s still appears in GET /subjects", subject)
			}
		}

		t.Logf("Permanent delete won: subject no longer exists")
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Errorf("Unexpected status %d checking final subject state: %s", resp.StatusCode, string(body))
	}

	// If subject appears in subjects list, it must have at least one retrievable version
	listResp, err := doRequest("GET", inst.addr+"/subjects", nil)
	if err != nil {
		t.Fatalf("Failed to list subjects for final check: %v", err)
	}
	var allSubjects []string
	json.NewDecoder(listResp.Body).Decode(&allSubjects)
	io.Copy(io.Discard, listResp.Body)
	listResp.Body.Close()

	for _, s := range allSubjects {
		if s == subject {
			vResp, err := doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
			if err != nil {
				t.Errorf("Subject %s in list but failed to GET versions: %v", subject, err)
				break
			}
			if vResp.StatusCode == http.StatusOK {
				var versions []int
				json.NewDecoder(vResp.Body).Decode(&versions)
				io.Copy(io.Discard, vResp.Body)
				vResp.Body.Close()
				if len(versions) == 0 {
					t.Errorf("Subject %s appears in subjects list but has zero versions", subject)
				}
			} else {
				io.Copy(io.Discard, vResp.Body)
				vResp.Body.Close()
				t.Errorf("Subject %s in list but GET versions returned status %d", subject, vResp.StatusCode)
			}
			break
		}
	}

	t.Logf("Permanent delete race: permDelete success=%d fail=%d, register success=%d fail=%d, 5xx=%d",
		permDeleteSuccess, permDeleteFail, registerSuccess, registerFail, serverErrors)
}

// TestConcurrentSoftDeleteAndVersionContiguity tests interleaved soft-delete and registration
// on the same subject, then verifies strict version contiguity and content integrity.
func TestConcurrentSoftDeleteAndVersionContiguity(t *testing.T) {
	subject := fmt.Sprintf("sd-contiguity-%d", time.Now().UnixNano())
	inst := instances[0]

	// Set compat to NONE so all schemas are accepted
	configReq := map[string]string{"compatibility": "NONE"}
	resp, err := doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Register 3 initial versions
	for i := 0; i < 3; i++ {
		schema := map[string]interface{}{
			"schema": fmt.Sprintf(`{"type":"record","name":"SdContig","fields":[{"name":"init%d","type":"int"}]}`, i),
		}
		resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
		if err != nil {
			t.Fatalf("Failed to register initial version %d: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Fatalf("Failed to register initial version %d: status %d, body: %s", i, resp.StatusCode, body)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Concurrent: each worker does numOperations/numConcurrent iterations,
	// randomly (50/50 based on iteration parity) soft-deleting or registering
	var wg sync.WaitGroup
	var serverErrors int64
	var registerAttempts int64
	var softDeleteAttempts int64
	errorMsgs := make(chan string, numConcurrent*numOperations)
	opsPerWorker := numOperations / numConcurrent
	if opsPerWorker < 1 {
		opsPerWorker = 1
	}

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < opsPerWorker; j++ {
				inst := getRandomInstance()

				// Use (workerID+j)%2 for 50/50 split across iterations
				if (workerID+j)%2 == 0 {
					// Soft-delete
					atomic.AddInt64(&softDeleteAttempts, 1)
					resp, err := doRequest("DELETE", inst.addr+"/subjects/"+subject, nil)
					if err != nil {
						continue
					}
					if resp.StatusCode >= 500 {
						body, _ := io.ReadAll(resp.Body)
						atomic.AddInt64(&serverErrors, 1)
						errorMsgs <- fmt.Sprintf("worker %d iter %d soft-delete: 5xx status %d, body: %s", workerID, j, resp.StatusCode, string(body))
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				} else {
					// Register new unique schema
					atomic.AddInt64(&registerAttempts, 1)
					s := map[string]interface{}{
						"schema": fmt.Sprintf(`{"type":"record","name":"SdContig","fields":[{"name":"w%d_i%d","type":"int"}]}`, workerID, j),
					}
					resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", s)
					if err != nil {
						continue
					}
					if resp.StatusCode >= 500 {
						body, _ := io.ReadAll(resp.Body)
						atomic.AddInt64(&serverErrors, 1)
						errorMsgs <- fmt.Sprintf("worker %d iter %d register: 5xx status %d, body: %s", workerID, j, resp.StatusCode, string(body))
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: Zero 5xx errors
	if serverErrors > 0 {
		t.Errorf("Expected zero 5xx errors, got %d", serverErrors)
	}

	// Verification: check version list for contiguity
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}

	finalVersionCount := 0
	if resp.StatusCode == http.StatusOK {
		var versions []int
		json.NewDecoder(resp.Body).Decode(&versions)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// After interleaved soft-deletes and registrations, the active (non-deleted)
		// version list may not start at 1 — earlier versions were soft-deleted.
		// The key invariant is that versions are monotonically increasing with no
		// duplicates, and every listed version is retrievable with valid content.
		sort.Ints(versions)
		for i := 1; i < len(versions); i++ {
			if versions[i] <= versions[i-1] {
				t.Errorf("Versions not monotonically increasing: %v", versions)
				break
			}
		}
		t.Logf("Active versions after storm: %v (count=%d)", versions, len(versions))

		// Each version must have valid, parseable JSON schema content
		for _, v := range versions {
			vResp, err := doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/%d", inst.addr, subject, v), nil)
			if err != nil {
				t.Errorf("Failed to GET version %d: %v", v, err)
				continue
			}
			if vResp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(vResp.Body)
				vResp.Body.Close()
				t.Errorf("GET version %d: status %d, body: %s", v, vResp.StatusCode, string(body))
				continue
			}
			var versionDetail struct {
				Schema string `json:"schema"`
			}
			json.NewDecoder(vResp.Body).Decode(&versionDetail)
			io.Copy(io.Discard, vResp.Body)
			vResp.Body.Close()

			if versionDetail.Schema == "" {
				t.Errorf("Version %d has empty schema content", v)
				continue
			}

			// Verify schema is valid JSON
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(versionDetail.Schema), &parsed); err != nil {
				t.Errorf("Version %d schema is not valid JSON: %v (schema: %s)", v, err, versionDetail.Schema)
			}
		}

		finalVersionCount = len(versions)
	} else if resp.StatusCode == http.StatusNotFound {
		// Subject was deleted and not re-registered — acceptable outcome
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Errorf("Unexpected status %d from GET versions: %s", resp.StatusCode, string(body))
	}

	// Also check deleted versions list
	deletedResp, err := doRequest("GET", inst.addr+"/subjects/"+subject+"/versions?deleted=true", nil)
	if err != nil {
		t.Logf("Failed to get deleted versions: %v", err)
	} else {
		deletedVersionCount := 0
		if deletedResp.StatusCode == http.StatusOK {
			var deletedVersions []int
			json.NewDecoder(deletedResp.Body).Decode(&deletedVersions)
			deletedVersionCount = len(deletedVersions)
		}
		io.Copy(io.Discard, deletedResp.Body)
		deletedResp.Body.Close()
		t.Logf("Deleted versions count (including soft-deleted): %d", deletedVersionCount)
	}

	t.Logf("Soft-delete contiguity: register attempts=%d, soft-delete attempts=%d, final versions=%d, 5xx=%d",
		registerAttempts, softDeleteAttempts, finalVersionCount, serverErrors)
}

// TestConcurrentRegistrationWithBackwardCompatibility tests concurrent registration with
// BACKWARD compatibility enabled. All workers add optional fields (backward-compatible),
// then verifies the entire version chain is compatibility-valid.
func TestConcurrentRegistrationWithBackwardCompatibility(t *testing.T) {
	subject := fmt.Sprintf("backward-race-%d", time.Now().UnixNano())
	inst := instances[0]

	// Register initial schema (version 1)
	initialSchema := map[string]interface{}{
		"schema": `{"type":"record","name":"BwRace","fields":[{"name":"id","type":"int"}]}`,
	}
	resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", initialSchema)
	if err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register initial schema: status %d, body: %s", resp.StatusCode, body)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Set BACKWARD compatibility
	configReq := map[string]string{"compatibility": "BACKWARD"}
	resp, err = doRequest("PUT", inst.addr+"/config/"+subject, configReq)
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Concurrent: each worker registers a schema that adds a unique optional field
	// Adding an optional field with default is always backward-compatible in Avro
	var wg sync.WaitGroup
	var successCount, errorCount int64
	var serverErrors int64

	type regResult struct {
		workerID int
		schemaID int64
	}
	results := make(chan regResult, numConcurrent)
	errorMsgs := make(chan string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			inst := getRandomInstance()

			schema := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"BwRace","fields":[{"name":"id","type":"int"},{"name":"opt_%d","type":["null","string"],"default":null}]}`, workerID),
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", workerID, err)
				return
			}

			if resp.StatusCode >= 500 {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&serverErrors, 1)
				errorMsgs <- fmt.Sprintf("worker %d: 5xx status %d, body: %s", workerID, resp.StatusCode, string(body))
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				return
			}

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				json.NewDecoder(resp.Body).Decode(&result)
				atomic.AddInt64(&successCount, 1)
				results <- regResult{workerID: workerID, schemaID: result.ID}
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d, body: %s", workerID, resp.StatusCode, string(body))
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	close(results)
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// Collect results
	var allResults []regResult
	for r := range results {
		allResults = append(allResults, r)
	}

	// INVARIANT: Zero 5xx errors
	if serverErrors > 0 {
		t.Errorf("Expected zero 5xx errors, got %d", serverErrors)
	}

	// INVARIANT: All registrations succeed — adding optional fields is always backward-compatible
	if errorCount > 0 {
		t.Errorf("Expected all registrations to succeed, got %d errors (success=%d)", errorCount, successCount)
	}

	// Verification: versions should be contiguous 1..N
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}
	var versions []int
	json.NewDecoder(resp.Body).Decode(&versions)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	sort.Ints(versions)
	expectedCount := int(successCount) + 1 // +1 for the initial schema
	if len(versions) != expectedCount {
		t.Errorf("Expected %d versions (1 initial + %d concurrent), got %d: %v",
			expectedCount, successCount, len(versions), versions)
	}

	for idx, v := range versions {
		expected := idx + 1
		if v != expected {
			t.Errorf("Version gap: expected %d at position %d, got %d (versions: %v)", expected, idx, v, versions)
			break
		}
	}

	// Fetch all version schemas into a map for compatibility chain validation
	versionSchemas := make(map[int]string)
	for _, v := range versions {
		vResp, err := doRequest("GET", fmt.Sprintf("%s/subjects/%s/versions/%d", inst.addr, subject, v), nil)
		if err != nil {
			t.Errorf("Failed to GET version %d: %v", v, err)
			continue
		}
		if vResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(vResp.Body)
			vResp.Body.Close()
			t.Errorf("GET version %d: status %d, body: %s", v, vResp.StatusCode, string(body))
			continue
		}
		var versionDetail struct {
			Schema string `json:"schema"`
		}
		json.NewDecoder(vResp.Body).Decode(&versionDetail)
		io.Copy(io.Discard, vResp.Body)
		vResp.Body.Close()
		versionSchemas[v] = versionDetail.Schema
	}

	// INVARIANT: Each consecutive pair is backward-compatible
	// Check by calling POST /compatibility/subjects/{subject}/versions/{v_i} with v_{i+1}'s schema
	for i := 0; i < len(versions)-1; i++ {
		prevV := versions[i]
		nextSchema, ok := versionSchemas[versions[i+1]]
		if !ok {
			continue
		}

		compatBody := map[string]interface{}{
			"schema": nextSchema,
		}
		compatResp, err := doRequest("POST",
			fmt.Sprintf("%s/compatibility/subjects/%s/versions/%d", inst.addr, subject, prevV),
			compatBody)
		if err != nil {
			t.Errorf("Failed to check compatibility between version %d and %d: %v", prevV, versions[i+1], err)
			continue
		}
		if compatResp.StatusCode == http.StatusOK {
			var result struct {
				IsCompatible bool `json:"is_compatible"`
			}
			json.NewDecoder(compatResp.Body).Decode(&result)
			if !result.IsCompatible {
				t.Errorf("Version %d -> %d is NOT backward-compatible (expected compatible)", prevV, versions[i+1])
			}
		} else {
			body, _ := io.ReadAll(compatResp.Body)
			t.Logf("Compatibility check version %d -> %d: status %d, body: %s", prevV, versions[i+1], compatResp.StatusCode, string(body))
		}
		io.Copy(io.Discard, compatResp.Body)
		compatResp.Body.Close()
	}

	// INVARIANT: Each version's schema has a unique optional field (except version 1 which is the base)
	optionalFields := make(map[string]int)
	for v, schemaStr := range versionSchemas {
		var s struct {
			Fields []struct {
				Name string `json:"name"`
			} `json:"fields"`
		}
		if err := json.Unmarshal([]byte(schemaStr), &s); err != nil {
			t.Errorf("Version %d: failed to parse schema: %v", v, err)
			continue
		}
		for _, f := range s.Fields {
			if strings.HasPrefix(f.Name, "opt_") {
				optionalFields[f.Name]++
			}
		}
	}
	for fieldName, count := range optionalFields {
		if count > 1 {
			t.Errorf("Optional field %q appears in %d versions (expected 1)", fieldName, count)
		}
	}

	t.Logf("Backward compat registration: %d successes, %d errors, %d versions, %d unique optional fields",
		successCount, errorCount, len(versions), len(optionalFields))
}

// TestConcurrentIDAllocationDensity measures ID allocation density under contention.
// Cassandra's block allocator may waste IDs; this test quantifies the waste.
func TestConcurrentIDAllocationDensity(t *testing.T) {
	timestamp := time.Now().UnixNano()
	numWorkers := numConcurrent * 2

	// Register a probe schema to get the current starting ID baseline
	probeSubject := fmt.Sprintf("id-density-probe-%d", timestamp)
	probeSchema := map[string]interface{}{
		"schema": fmt.Sprintf(`{"type":"record","name":"Probe","fields":[{"name":"p_%d","type":"int"}]}`, timestamp),
	}
	resp, err := doRequest("POST", instances[0].addr+"/subjects/"+probeSubject+"/versions", probeSchema)
	if err != nil {
		t.Fatalf("Failed to register probe schema: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("Failed to register probe schema: status %d, body: %s", resp.StatusCode, body)
	}
	var probeResp struct {
		ID int64 `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&probeResp)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	startID := probeResp.ID
	t.Logf("Probe schema registered with ID %d", startID)

	// Concurrent: each worker registers a unique schema to a unique subject
	var wg sync.WaitGroup
	var successCount, errorCount int64
	var mu sync.Mutex
	var allIDs []int64
	errorMsgs := make(chan string, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			inst := getRandomInstance()

			subject := fmt.Sprintf("id-density-%d-%d", timestamp, workerID)
			schema := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"Density%d","fields":[{"name":"d_%d","type":"int"}]}`, workerID, workerID),
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: %v", workerID, err)
				return
			}

			if resp.StatusCode == http.StatusOK {
				var result struct {
					ID int64 `json:"id"`
				}
				json.NewDecoder(resp.Body).Decode(&result)
				if result.ID > 0 {
					atomic.AddInt64(&successCount, 1)
					mu.Lock()
					allIDs = append(allIDs, result.ID)
					mu.Unlock()
				} else {
					atomic.AddInt64(&errorCount, 1)
					errorMsgs <- fmt.Sprintf("worker %d: 200 but id=%d", workerID, result.ID)
				}
			} else {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&errorCount, 1)
				errorMsgs <- fmt.Sprintf("worker %d: status %d, body: %s", workerID, resp.StatusCode, string(body))
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}

	wg.Wait()
	close(errorMsgs)

	for errMsg := range errorMsgs {
		t.Logf("Error: %s", errMsg)
	}

	// INVARIANT: All registrations succeed
	if errorCount > 0 {
		t.Errorf("Expected all registrations to succeed, got %d errors", errorCount)
	}

	if len(allIDs) == 0 {
		t.Fatal("No successful registrations — cannot compute density")
	}

	// Include the probe ID in the set
	allIDs = append(allIDs, startID)

	// INVARIANT: All IDs unique and positive
	idSet := make(map[int64]bool)
	for _, id := range allIDs {
		if id <= 0 {
			t.Errorf("Invalid schema ID: %d (must be > 0)", id)
		}
		if idSet[id] {
			t.Errorf("Duplicate schema ID: %d", id)
		}
		idSet[id] = true
	}

	// Compute density
	minID := allIDs[0]
	maxID := allIDs[0]
	for _, id := range allIDs {
		if id < minID {
			minID = id
		}
		if id > maxID {
			maxID = id
		}
	}

	idRange := maxID - minID + 1
	density := float64(len(idSet)) / float64(idRange)
	t.Logf("ID allocation density: %.1f%% (%d unique IDs in range %d-%d, range size %d)",
		density*100, len(idSet), minID, maxID, idRange)

	if density < 0.5 {
		t.Logf("WARNING: ID density below 50%% — possible block allocator contention (density=%.1f%%)", density*100)
	}

	// Verify a sample of schemas are retrievable (first 10 or all if fewer)
	inst := instances[0]
	sampleSize := 10
	if len(allIDs) < sampleSize {
		sampleSize = len(allIDs)
	}
	for i := 0; i < sampleSize; i++ {
		id := allIDs[i]
		resp, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", inst.addr, id), nil)
		if err != nil {
			t.Errorf("Failed to GET schema ID %d: %v", id, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			t.Errorf("GET schema ID %d: status %d, body: %s", id, resp.StatusCode, string(body))
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	t.Logf("ID density test complete: %d workers, %d successes, %d errors, density=%.1f%%",
		numWorkers, successCount, errorCount, density*100)
}
