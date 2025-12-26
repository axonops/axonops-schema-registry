package compatibility_test

import (
	"encoding/binary"
	"os"
	"testing"

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
