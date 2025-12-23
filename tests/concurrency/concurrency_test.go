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
	"strconv"
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

const (
	numInstances   = 3
	numConcurrent  = 10
	numOperations  = 100
	requestTimeout = 30 * time.Second
)

type instance struct {
	server *api.Server
	addr   string
}

var instances []*instance

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
			_ = http.ListenAndServe(addr, server)
		}(cfg.Server.Port)

		instances = append(instances, &instance{
			server: server,
			addr:   fmt.Sprintf("http://localhost:%d", cfg.Server.Port),
		})
	}

	// Wait for servers to start
	time.Sleep(2 * time.Second)

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
			Hosts:               []string{getEnvOrDefault("CASSANDRA_HOSTS", "localhost")},
			Port:                getEnvOrDefaultInt("CASSANDRA_PORT", 9042),
			Keyspace:            getEnvOrDefault("CASSANDRA_KEYSPACE", "schemaregistry"),
			Consistency:         "LOCAL_ONE", // Use LOCAL_ONE for single-node test cluster
			LocalDC:             "dc1",       // Match the DC configured in the test container
			ReplicationStrategy: "SimpleStrategy",
			ReplicationFactor:   1,
			ConnectTimeout:      60 * time.Second, // Longer timeout for CI
			Timeout:             60 * time.Second,
			NumConns:            50, // Higher connection pool for concurrency tests
		}
		return cassandra.NewStore(cfg)

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

func getRandomInstance() *instance {
	return instances[time.Now().UnixNano()%int64(len(instances))]
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

	client := &http.Client{Timeout: requestTimeout}
	return client.Do(req)
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

// TestConcurrentVersionUpdates tests updating the same subject from multiple instances
func TestConcurrentVersionUpdates(t *testing.T) {
	subject := fmt.Sprintf("concurrent-updates-%d", time.Now().UnixNano())

	// Register initial schema
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
	var successCount, conflictCount int64
	versionsCreated := make(chan int, numConcurrent)

	// Multiple workers try to update the same subject
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			inst := getRandomInstance()

			// Create a unique schema for this worker
			schema := map[string]interface{}{
				"schema": fmt.Sprintf(`{"type":"record","name":"Updates","fields":[{"name":"id","type":"int"},{"name":"worker%d","type":["null","string"],"default":null}]}`, workerID),
			}

			resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)
				if id, ok := result["id"].(float64); ok {
					versionsCreated <- int(id)
				}
			} else if resp.StatusCode == http.StatusConflict {
				atomic.AddInt64(&conflictCount, 1)
			}
		}(i)
	}

	wg.Wait()
	close(versionsCreated)

	t.Logf("Version updates: %d successes, %d conflicts", successCount, conflictCount)

	// Verify final state
	resp, err = doRequest("GET", inst.addr+"/subjects/"+subject+"/versions", nil)
	if err != nil {
		t.Fatalf("Failed to get versions: %v", err)
	}
	defer resp.Body.Close()

	var versions []int
	json.NewDecoder(resp.Body).Decode(&versions)
	t.Logf("Final versions: %v", versions)
}

// TestConcurrentReads tests reading schemas from multiple instances
func TestConcurrentReads(t *testing.T) {
	subject := fmt.Sprintf("concurrent-reads-%d", time.Now().UnixNano())

	// Register a schema
	inst := instances[0]
	schema := map[string]interface{}{
		"schema": `{"type":"record","name":"Reads","fields":[{"name":"data","type":"string"}]}`,
	}

	resp, err := doRequest("POST", inst.addr+"/subjects/"+subject+"/versions", schema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	schemaID := int(result["id"].(float64))

	var wg sync.WaitGroup
	var successCount, errorCount int64

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

				switch j % 4 {
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

	t.Logf("Concurrent reads: %d successes, %d errors", successCount, errorCount)

	if errorCount > int64(numConcurrent*numOperations/20) {
		t.Errorf("Too many read errors: %d out of %d", errorCount, numConcurrent*numOperations)
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

// TestDataConsistency verifies data written by one instance can be read by another
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
	var writeResult map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&writeResult)
	resp.Body.Close()

	schemaID := int(writeResult["id"].(float64))

	// Small delay for replication
	time.Sleep(100 * time.Millisecond)

	// Read from all other instances
	for i := 1; i < len(instances); i++ {
		resp, err := doRequest("GET", fmt.Sprintf("%s/schemas/ids/%d", instances[i].addr, schemaID), nil)
		if err != nil {
			t.Errorf("Instance %d failed to read: %v", i, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Instance %d returned status %d", i, resp.StatusCode)
		}

		var readResult map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&readResult)
		resp.Body.Close()

		if readResult["schema"] == nil {
			t.Errorf("Instance %d returned empty schema", i)
		}
	}
}
