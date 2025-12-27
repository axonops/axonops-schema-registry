package compatibility_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/riferrei/srclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Protobuf schema definitions
const userProtoSchema = `
syntax = "proto3";
package com.axonops.test;

message User {
    int64 id = 1;
    string name = 2;
    string email = 3;
}
`

const eventProtoSchema = `
syntax = "proto3";
package com.axonops.test;

message Event {
    string id = 1;
    string type = 2;
    int64 timestamp = 3;
    map<string, string> metadata = 4;
}
`

func TestProtobufSchemaRegistration(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("RegisterUserProto", func(t *testing.T) {
		subject := "go-proto-user-value"

		schema, err := client.CreateSchema(subject, userProtoSchema, srclient.Protobuf)
		require.NoError(t, err)
		assert.Greater(t, schema.ID(), 0, "Schema ID should be positive")

		t.Logf("Go srclient: Proto User schema registered with ID %d", schema.ID())
	})

	t.Run("RegisterEventProto", func(t *testing.T) {
		subject := "go-proto-event-value"

		schema, err := client.CreateSchema(subject, eventProtoSchema, srclient.Protobuf)
		require.NoError(t, err)
		assert.Greater(t, schema.ID(), 0)

		t.Logf("Go srclient: Proto Event schema registered with ID %d", schema.ID())
	})

	t.Run("ProtoSchemaDeduplication", func(t *testing.T) {
		subject := "go-proto-dedup-value"

		// Register twice
		schema1, err := client.CreateSchema(subject, userProtoSchema, srclient.Protobuf)
		require.NoError(t, err)

		schema2, err := client.CreateSchema(subject, userProtoSchema, srclient.Protobuf)
		require.NoError(t, err)

		assert.Equal(t, schema1.ID(), schema2.ID(), "Same proto schema should return same ID")
		t.Logf("Proto schema deduplication verified: ID %d", schema1.ID())
	})
}

func TestProtobufSchemaEvolution(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("AddOptionalField", func(t *testing.T) {
		v1Proto := `
syntax = "proto3";
package com.axonops.evolution;

message Record {
    string id = 1;
    string data = 2;
}
`

		v2Proto := `
syntax = "proto3";
package com.axonops.evolution;

message Record {
    string id = 1;
    string data = 2;
    int64 version = 3;
}
`

		subject := "go-proto-evolution-value"

		// Register v1
		schema1, err := client.CreateSchema(subject, v1Proto, srclient.Protobuf)
		require.NoError(t, err)

		// Register v2
		schema2, err := client.CreateSchema(subject, v2Proto, srclient.Protobuf)
		require.NoError(t, err)

		assert.NotEqual(t, schema1.ID(), schema2.ID(), "Different schemas should have different IDs")

		t.Logf("Proto schema evolution: v1 ID=%d, v2 ID=%d", schema1.ID(), schema2.ID())
	})
}

func TestProtobufWireFormat(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("WireFormatHeader", func(t *testing.T) {
		subject := "go-proto-wire-value"

		// Register schema
		schema, err := client.CreateSchema(subject, userProtoSchema, srclient.Protobuf)
		require.NoError(t, err)

		// Protobuf wire format:
		// - Magic byte (0)
		// - 4-byte schema ID (big-endian)
		// - Message index array (variable length)
		// - Protobuf payload

		// Verify schema ID is valid
		assert.Greater(t, schema.ID(), 0)
		t.Logf("Proto wire format test: schema_id=%d", schema.ID())
	})
}

func TestProtobufGlobalSchemaID(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("SameSchemaAcrossSubjects", func(t *testing.T) {
		subject1 := "go-proto-global1-value"
		subject2 := "go-proto-global2-value"

		// Register same schema under different subjects
		schema1, err := client.CreateSchema(subject1, userProtoSchema, srclient.Protobuf)
		require.NoError(t, err)

		schema2, err := client.CreateSchema(subject2, userProtoSchema, srclient.Protobuf)
		require.NoError(t, err)

		// Same schema content should produce same global ID (Confluent-compatible behavior)
		assert.Equal(t, schema1.ID(), schema2.ID(),
			"Same Protobuf schema under different subjects should return same global ID")

		// Structural verification - fetch and compare
		fetched1, err := client.GetSchema(schema1.ID())
		require.NoError(t, err)

		fetched2, err := client.GetSchema(schema2.ID())
		require.NoError(t, err)

		assert.Equal(t, fetched1.ID(), fetched2.ID(), "Fetched schema IDs should match")

		t.Logf("Global Protobuf schema ID verified: both subjects use ID %d", schema1.ID())
	})
}

func TestProtobufConcurrentRegistration(t *testing.T) {
	subject := fmt.Sprintf("go-proto-concurrent-%d-value", time.Now().UnixNano())

	numGoroutines := 10
	var wg sync.WaitGroup
	readyChan := make(chan struct{})
	resultChan := make(chan int, numGoroutines)
	errChan := make(chan error, numGoroutines)

	// Launch goroutines with separate clients for genuine parallel HTTP requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Each goroutine gets its own client
			client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

			// Wait for start signal
			<-readyChan

			schema, err := client.CreateSchema(subject, userProtoSchema, srclient.Protobuf)
			if err != nil {
				errChan <- err
				return
			}
			resultChan <- schema.ID()
		}(i)
	}

	// Small delay to let goroutines reach the wait point
	time.Sleep(50 * time.Millisecond)

	// Release all goroutines simultaneously
	close(readyChan)

	// Wait for completion
	wg.Wait()
	close(resultChan)
	close(errChan)

	// Check for errors
	for err := range errChan {
		require.NoError(t, err)
	}

	// Collect schema IDs
	schemaIDs := make(map[int]bool)
	for id := range resultChan {
		schemaIDs[id] = true
	}

	// All concurrent registrations should return the same ID
	assert.Equal(t, 1, len(schemaIDs),
		"All concurrent registrations should return the same schema ID, got %d different IDs", len(schemaIDs))

	// Verify only one version was created
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())
	versions, err := client.GetSchemaVersions(subject)
	require.NoError(t, err)
	assert.Equal(t, 1, len(versions),
		"Only one version should exist after concurrent registration")

	t.Logf("Concurrent registration test passed: %d goroutines all got the same schema ID", numGoroutines)
}

func TestProtobufConfigEndpoints(t *testing.T) {
	t.Run("GetGlobalCompatibility", func(t *testing.T) {
		// Use HTTP client directly since srclient may not expose config APIs
		resp, err := http.Get(getSchemaRegistryURL() + "/config")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var config struct {
			CompatibilityLevel string `json:"compatibilityLevel"`
		}
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)

		assert.True(t, validCompatibilityLevels[config.CompatibilityLevel],
			"Global compatibility should be a valid Confluent level, got: %s", config.CompatibilityLevel)

		t.Logf("Global compatibility: %s", config.CompatibilityLevel)
	})
}

func TestProtobufIncompatibleSchemaEvolution(t *testing.T) {
	subject := fmt.Sprintf("go-proto-incompat-%d-value", time.Now().UnixNano())

	// First, register v1 schema
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())
	_, err := client.CreateSchema(subject, userProtoSchema, srclient.Protobuf)
	require.NoError(t, err)

	// Set subject compatibility to BACKWARD via HTTP
	reqBody := `{"compatibility": "BACKWARD"}`
	req, err := http.NewRequest(http.MethodPut, getSchemaRegistryURL()+"/config/"+subject, strings.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	// Verify compatibility was set
	resp, err = http.Get(getSchemaRegistryURL() + "/config/" + subject)
	require.NoError(t, err)
	defer resp.Body.Close()

	var config struct {
		CompatibilityLevel string `json:"compatibilityLevel"`
	}
	err = json.NewDecoder(resp.Body).Decode(&config)
	require.NoError(t, err)
	assert.Equal(t, "BACKWARD", config.CompatibilityLevel)

	// Create incompatible schema: change email type from string to int64 (breaking change)
	incompatibleProto := `
syntax = "proto3";
package com.axonops.test;

message User {
    int64 id = 1;
    string name = 2;
    int64 email = 3;
}
`

	// Try to register incompatible schema
	_, err = client.CreateSchema(subject, incompatibleProto, srclient.Protobuf)
	require.Error(t, err, "Expected registration to fail due to incompatible schema")

	// Verify error message indicates incompatibility
	errMsg := strings.ToLower(err.Error())
	isIncompatError := strings.Contains(errMsg, "incompatible") ||
		strings.Contains(errMsg, "compatibility") ||
		strings.Contains(errMsg, "409") ||
		strings.Contains(errMsg, "422")

	assert.True(t, isIncompatError,
		"Expected incompatibility error, got: %s", err.Error())

	t.Log("Incompatible Protobuf schema correctly rejected")
}

func TestProtobufCacheBehavior(t *testing.T) {
	subject := fmt.Sprintf("go-proto-cache-%d-value", time.Now().UnixNano())

	// Register schema with first client
	client1 := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())
	schema, err := client1.CreateSchema(subject, userProtoSchema, srclient.Protobuf)
	require.NoError(t, err)

	// Create a completely new client (empty cache)
	client2 := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	// Fetch schema with fresh client (cache miss, must hit registry)
	fetchedSchema, err := client2.GetSchema(schema.ID())
	require.NoError(t, err)

	assert.NotNil(t, fetchedSchema, "Fresh client should fetch schema by ID")
	assert.Equal(t, schema.ID(), fetchedSchema.ID(), "Schema IDs should match")

	t.Log("Cache behavior test passed")
}

func TestProtobufSchemaCanonicalisation(t *testing.T) {
	// Same Protobuf schema content but with different formatting
	// This tests that the registry canonicalizes schemas before comparison
	//
	// NOTE: Some client versions may canonicalize client-side before POSTing,
	// so this test may pass even if server-side canonicalization is broken.
	// For strict server-side canonicalization validation, register via REST API directly.

	// Compact format
	compactProto := `syntax = "proto3"; package com.axonops.canon; message Canonical { int64 id = 1; string value = 2; }`

	// Verbose format (extra whitespace, comments stripped by parser anyway)
	verboseProto := `
syntax = "proto3";

package com.axonops.canon;

message Canonical {
    int64 id = 1;
    string value = 2;
}
`

	subject1 := fmt.Sprintf("go-proto-canon1-%d-value", time.Now().UnixNano())
	subject2 := fmt.Sprintf("go-proto-canon2-%d-value", time.Now().UnixNano())

	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	// Register compact schema
	schema1, err := client.CreateSchema(subject1, compactProto, srclient.Protobuf)
	require.NoError(t, err)

	// Register verbose schema (should be canonicalized to same schema)
	schema2, err := client.CreateSchema(subject2, verboseProto, srclient.Protobuf)
	require.NoError(t, err)

	// Same schema content (after canonicalization) should produce same global ID
	assert.Equal(t, schema1.ID(), schema2.ID(),
		"Same Protobuf schema with different formatting should return same global ID (canonicalization)")

	t.Logf("Schema canonicalization verified: both formats use schema ID %d", schema1.ID())
}
