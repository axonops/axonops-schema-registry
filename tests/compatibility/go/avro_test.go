package compatibility_test

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Avro schemas
const userAvroSchema = `{
	"type": "record",
	"name": "User",
	"namespace": "com.axonops.test",
	"fields": [
		{"name": "id", "type": "long"},
		{"name": "name", "type": "string"},
		{"name": "email", "type": ["null", "string"], "default": null}
	]
}`

const simpleAvroSchema = `{
	"type": "record",
	"name": "SimpleRecord",
	"fields": [
		{"name": "value", "type": "string"}
	]
}`

const paymentAvroSchema = `{
	"type": "record",
	"name": "Payment",
	"namespace": "com.axonops.test",
	"fields": [
		{"name": "id", "type": "string"},
		{"name": "amount", "type": "double"},
		{"name": "currency", "type": {"type": "enum", "name": "Currency", "symbols": ["USD", "EUR", "GBP"]}}
	]
}`

func getSchemaRegistryURL() string {
	url := os.Getenv("SCHEMA_REGISTRY_URL")
	if url == "" {
		return "http://localhost:8081"
	}
	return url
}

// validCompatibilityLevels are the known Confluent compatibility levels
var validCompatibilityLevels = map[string]bool{
	"NONE":                true,
	"BACKWARD":            true,
	"FORWARD":             true,
	"FULL":                true,
	"BACKWARD_TRANSITIVE": true,
	"FORWARD_TRANSITIVE":  true,
	"FULL_TRANSITIVE":     true,
}

func TestAvroSchemaRegistration(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("RegisterUserSchema", func(t *testing.T) {
		subject := "go-avro-user-value"

		schema, err := client.CreateSchema(subject, userAvroSchema, srclient.Avro)
		require.NoError(t, err)
		assert.Greater(t, schema.ID(), 0, "Schema ID should be positive")

		t.Logf("Go srclient: User schema registered with ID %d", schema.ID())
	})

	t.Run("RegisterSimpleSchema", func(t *testing.T) {
		subject := "go-avro-simple-value"

		schema, err := client.CreateSchema(subject, simpleAvroSchema, srclient.Avro)
		require.NoError(t, err)
		assert.Greater(t, schema.ID(), 0)

		t.Logf("Go srclient: Simple schema registered with ID %d", schema.ID())
	})

	t.Run("SchemaDeduplication", func(t *testing.T) {
		subject := "go-avro-dedup-value"

		// Register twice
		schema1, err := client.CreateSchema(subject, userAvroSchema, srclient.Avro)
		require.NoError(t, err)

		schema2, err := client.CreateSchema(subject, userAvroSchema, srclient.Avro)
		require.NoError(t, err)

		assert.Equal(t, schema1.ID(), schema2.ID(), "Same schema should return same ID")
		t.Logf("Schema deduplication verified: ID %d", schema1.ID())
	})
}

func TestAvroSerialization(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("WireFormatStructure", func(t *testing.T) {
		subject := "go-avro-wire-value"

		// Register schema
		schema, err := client.CreateSchema(subject, userAvroSchema, srclient.Avro)
		require.NoError(t, err)

		// Create Avro codec
		codec, err := goavro.NewCodec(userAvroSchema)
		require.NoError(t, err)

		// Create test data
		user := map[string]interface{}{
			"id":    int64(1),
			"name":  "Test User",
			"email": goavro.Union("string", "test@example.com"),
		}

		// Encode with Avro
		avroBytes, err := codec.BinaryFromNative(nil, user)
		require.NoError(t, err)

		// Create wire format: magic byte (0) + 4-byte schema ID (big-endian) + Avro payload
		wireBytes := make([]byte, 5+len(avroBytes))
		wireBytes[0] = 0 // Magic byte
		binary.BigEndian.PutUint32(wireBytes[1:5], uint32(schema.ID()))
		copy(wireBytes[5:], avroBytes)

		// Verify wire format
		assert.Equal(t, byte(0), wireBytes[0], "Magic byte should be 0")
		schemaID := binary.BigEndian.Uint32(wireBytes[1:5])
		assert.Equal(t, uint32(schema.ID()), schemaID)

		t.Logf("Wire format: magic=0x%02x, schema_id=%d, total_len=%d", wireBytes[0], schemaID, len(wireBytes))
	})

	t.Run("SerializationRoundtrip", func(t *testing.T) {
		subject := "go-avro-roundtrip-value"

		// Register schema
		schema, err := client.CreateSchema(subject, userAvroSchema, srclient.Avro)
		require.NoError(t, err)

		// Create Avro codec
		codec, err := goavro.NewCodec(userAvroSchema)
		require.NoError(t, err)

		// Original data
		original := map[string]interface{}{
			"id":    int64(42),
			"name":  "Jane Doe",
			"email": goavro.Union("string", "jane@example.com"),
		}

		// Serialize
		avroBytes, err := codec.BinaryFromNative(nil, original)
		require.NoError(t, err)

		// Create wire format
		wireBytes := make([]byte, 5+len(avroBytes))
		wireBytes[0] = 0
		binary.BigEndian.PutUint32(wireBytes[1:5], uint32(schema.ID()))
		copy(wireBytes[5:], avroBytes)

		// Deserialize (skip wire format header)
		decoded, _, err := codec.NativeFromBinary(wireBytes[5:])
		require.NoError(t, err)

		decodedMap := decoded.(map[string]interface{})
		assert.Equal(t, original["id"], decodedMap["id"])
		assert.Equal(t, original["name"], decodedMap["name"])

		t.Logf("Roundtrip verified for user: %v", decodedMap["name"])
	})

	t.Run("NullHandling", func(t *testing.T) {
		subject := "go-avro-null-value"

		// Register schema
		_, err := client.CreateSchema(subject, userAvroSchema, srclient.Avro)
		require.NoError(t, err)

		// Create Avro codec
		codec, err := goavro.NewCodec(userAvroSchema)
		require.NoError(t, err)

		// User with null email
		original := map[string]interface{}{
			"id":    int64(100),
			"name":  "No Email User",
			"email": nil,
		}

		// Serialize
		avroBytes, err := codec.BinaryFromNative(nil, original)
		require.NoError(t, err)

		// Deserialize
		decoded, _, err := codec.NativeFromBinary(avroBytes)
		require.NoError(t, err)

		decodedMap := decoded.(map[string]interface{})
		assert.Nil(t, decodedMap["email"])

		t.Log("Null handling verified")
	})
}

func TestAvroGlobalSchemaID(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("SameSchemaAcrossSubjects", func(t *testing.T) {
		subject1 := "go-avro-global1-value"
		subject2 := "go-avro-global2-value"

		// Register same schema under different subjects
		schema1, err := client.CreateSchema(subject1, userAvroSchema, srclient.Avro)
		require.NoError(t, err)

		schema2, err := client.CreateSchema(subject2, userAvroSchema, srclient.Avro)
		require.NoError(t, err)

		// Same schema content should produce same global ID (Confluent-compatible behavior)
		assert.Equal(t, schema1.ID(), schema2.ID(),
			"Same schema under different subjects should return same global ID")

		// Structural verification - fetch and compare
		fetched1, err := client.GetSchema(schema1.ID())
		require.NoError(t, err)

		fetched2, err := client.GetSchema(schema2.ID())
		require.NoError(t, err)

		assert.Equal(t, fetched1.ID(), fetched2.ID(), "Fetched schema IDs should match")

		t.Logf("Global schema ID verified: both subjects use ID %d", schema1.ID())
	})
}

func TestAvroSchemaEvolution(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("BackwardCompatibleSchema", func(t *testing.T) {
		v1Schema := `{
			"type": "record",
			"name": "Event",
			"namespace": "com.axonops.evolution",
			"fields": [
				{"name": "id", "type": "long"},
				{"name": "type", "type": "string"}
			]
		}`

		v2Schema := `{
			"type": "record",
			"name": "Event",
			"namespace": "com.axonops.evolution",
			"fields": [
				{"name": "id", "type": "long"},
				{"name": "type", "type": "string"},
				{"name": "timestamp", "type": "long", "default": 0}
			]
		}`

		subject := "go-avro-evolution-value"

		// Register v1
		schema1, err := client.CreateSchema(subject, v1Schema, srclient.Avro)
		require.NoError(t, err)

		// Register v2
		schema2, err := client.CreateSchema(subject, v2Schema, srclient.Avro)
		require.NoError(t, err)

		assert.NotEqual(t, schema1.ID(), schema2.ID(), "Different schemas should have different IDs")
		assert.Greater(t, schema2.ID(), schema1.ID(), "Newer schema should have higher ID")

		t.Logf("Schema evolution: v1 ID=%d, v2 ID=%d", schema1.ID(), schema2.ID())
	})
}

func TestAvroPaymentSchema(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("PaymentSerialization", func(t *testing.T) {
		subject := "go-avro-payment-value"

		// Register schema
		schema, err := client.CreateSchema(subject, paymentAvroSchema, srclient.Avro)
		require.NoError(t, err)

		// Create Avro codec
		codec, err := goavro.NewCodec(paymentAvroSchema)
		require.NoError(t, err)

		payment := map[string]interface{}{
			"id":       "pay-001",
			"amount":   99.99,
			"currency": "USD",
		}

		// Serialize
		avroBytes, err := codec.BinaryFromNative(nil, payment)
		require.NoError(t, err)

		// Create wire format
		wireBytes := make([]byte, 5+len(avroBytes))
		wireBytes[0] = 0
		binary.BigEndian.PutUint32(wireBytes[1:5], uint32(schema.ID()))
		copy(wireBytes[5:], avroBytes)

		// Deserialize
		decoded, _, err := codec.NativeFromBinary(wireBytes[5:])
		require.NoError(t, err)

		decodedMap := decoded.(map[string]interface{})
		assert.Equal(t, payment["id"], decodedMap["id"])
		assert.Equal(t, payment["amount"], decodedMap["amount"])
		assert.Equal(t, payment["currency"], decodedMap["currency"])

		t.Logf("Payment serialization verified: %v", decodedMap["id"])
	})
}

func TestAvroConcurrentRegistration(t *testing.T) {
	subject := fmt.Sprintf("go-avro-concurrent-%d-value", time.Now().UnixNano())

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

			schema, err := client.CreateSchema(subject, userAvroSchema, srclient.Avro)
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

func TestAvroConfigEndpoints(t *testing.T) {
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

func TestAvroIncompatibleSchemaEvolution(t *testing.T) {
	subject := fmt.Sprintf("go-avro-incompat-%d-value", time.Now().UnixNano())

	// First, register v1 schema
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())
	_, err := client.CreateSchema(subject, userAvroSchema, srclient.Avro)
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

	// Create incompatible schema: change email type from union to int (breaking change)
	incompatibleSchema := `{
		"type": "record",
		"name": "User",
		"namespace": "com.axonops.test",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "name", "type": "string"},
			{"name": "email", "type": "int"}
		]
	}`

	// Try to register incompatible schema
	_, err = client.CreateSchema(subject, incompatibleSchema, srclient.Avro)
	require.Error(t, err, "Expected registration to fail due to incompatible schema")

	// Verify error message indicates incompatibility
	errMsg := strings.ToLower(err.Error())
	isIncompatError := strings.Contains(errMsg, "incompatible") ||
		strings.Contains(errMsg, "compatibility") ||
		strings.Contains(errMsg, "409") ||
		strings.Contains(errMsg, "422")

	assert.True(t, isIncompatError,
		"Expected incompatibility error, got: %s", err.Error())

	t.Log("Incompatible schema correctly rejected")
}

func TestAvroCacheBehavior(t *testing.T) {
	subject := fmt.Sprintf("go-avro-cache-%d-value", time.Now().UnixNano())

	// Register schema with first client
	client1 := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())
	schema, err := client1.CreateSchema(subject, userAvroSchema, srclient.Avro)
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

func TestAvroSchemaCanonicalisation(t *testing.T) {
	// Same Avro schema content but with different formatting
	// This tests that the registry canonicalizes schemas before comparison
	//
	// NOTE: Some client versions may canonicalize client-side before POSTing,
	// so this test may pass even if server-side canonicalization is broken.
	// For strict server-side canonicalization validation, register via REST API directly.

	// Compact format (minimal whitespace)
	compactSchema := `{"type":"record","name":"Canonical","namespace":"com.axonops.canon","fields":[{"name":"id","type":"long"},{"name":"value","type":"string"}]}`

	// Verbose format (extra whitespace)
	verboseSchema := `{
		"type": "record",
		"name": "Canonical",
		"namespace": "com.axonops.canon",
		"fields": [
			{"name": "id", "type": "long"},
			{"name": "value", "type": "string"}
		]
	}`

	subject1 := fmt.Sprintf("go-avro-canon1-%d-value", time.Now().UnixNano())
	subject2 := fmt.Sprintf("go-avro-canon2-%d-value", time.Now().UnixNano())

	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	// Register compact schema
	schema1, err := client.CreateSchema(subject1, compactSchema, srclient.Avro)
	require.NoError(t, err)

	// Register verbose schema (should be canonicalized to same schema)
	schema2, err := client.CreateSchema(subject2, verboseSchema, srclient.Avro)
	require.NoError(t, err)

	// Same schema content (after canonicalization) should produce same global ID
	assert.Equal(t, schema1.ID(), schema2.ID(),
		"Same Avro schema with different formatting should return same global ID (canonicalization)")

	t.Logf("Schema canonicalization verified: both formats use schema ID %d", schema1.ID())
}
