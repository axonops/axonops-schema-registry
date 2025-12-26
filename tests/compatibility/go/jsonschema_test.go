package compatibility_test

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/riferrei/srclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JSON Schema definitions
const userJSONSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "User",
	"type": "object",
	"properties": {
		"id": {"type": "integer"},
		"name": {"type": "string"},
		"email": {"type": "string", "format": "email"}
	},
	"required": ["id", "name"]
}`

const orderJSONSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "Order",
	"type": "object",
	"properties": {
		"orderId": {"type": "string"},
		"customerId": {"type": "string"},
		"amount": {"type": "number"},
		"items": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"productId": {"type": "string"},
					"quantity": {"type": "integer"}
				}
			}
		}
	},
	"required": ["orderId", "customerId", "amount"]
}`

func TestJSONSchemaRegistration(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("RegisterUserJSONSchema", func(t *testing.T) {
		subject := "go-json-user-value"

		schema, err := client.CreateSchema(subject, userJSONSchema, srclient.Json)
		require.NoError(t, err)
		assert.Greater(t, schema.ID(), 0, "Schema ID should be positive")

		t.Logf("Go srclient: JSON User schema registered with ID %d", schema.ID())
	})

	t.Run("RegisterOrderJSONSchema", func(t *testing.T) {
		subject := "go-json-order-value"

		schema, err := client.CreateSchema(subject, orderJSONSchema, srclient.Json)
		require.NoError(t, err)
		assert.Greater(t, schema.ID(), 0)

		t.Logf("Go srclient: JSON Order schema registered with ID %d", schema.ID())
	})

	t.Run("JSONSchemaDeduplication", func(t *testing.T) {
		subject := "go-json-dedup-value"

		// Register twice
		schema1, err := client.CreateSchema(subject, userJSONSchema, srclient.Json)
		require.NoError(t, err)

		schema2, err := client.CreateSchema(subject, userJSONSchema, srclient.Json)
		require.NoError(t, err)

		assert.Equal(t, schema1.ID(), schema2.ID(), "Same schema should return same ID")
		t.Logf("JSON schema deduplication verified: ID %d", schema1.ID())
	})
}

func TestJSONSchemaSerialization(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("WireFormatStructure", func(t *testing.T) {
		subject := "go-json-wire-value"

		// Register schema
		schema, err := client.CreateSchema(subject, userJSONSchema, srclient.Json)
		require.NoError(t, err)

		// Create test data
		user := map[string]interface{}{
			"id":    1,
			"name":  "Test User",
			"email": "test@example.com",
		}

		// Encode as JSON
		jsonBytes, err := json.Marshal(user)
		require.NoError(t, err)

		// Create wire format: magic byte (0) + 4-byte schema ID (big-endian) + JSON payload
		wireBytes := make([]byte, 5+len(jsonBytes))
		wireBytes[0] = 0 // Magic byte
		binary.BigEndian.PutUint32(wireBytes[1:5], uint32(schema.ID()))
		copy(wireBytes[5:], jsonBytes)

		// Verify wire format
		assert.Equal(t, byte(0), wireBytes[0], "Magic byte should be 0")
		schemaID := binary.BigEndian.Uint32(wireBytes[1:5])
		assert.Equal(t, uint32(schema.ID()), schemaID)

		t.Logf("JSON wire format: magic=0x%02x, schema_id=%d, total_len=%d", wireBytes[0], schemaID, len(wireBytes))
	})

	t.Run("SerializationRoundtrip", func(t *testing.T) {
		subject := "go-json-roundtrip-value"

		// Register schema
		schema, err := client.CreateSchema(subject, userJSONSchema, srclient.Json)
		require.NoError(t, err)

		// Original data
		type User struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}

		original := User{ID: 42, Name: "Jane Doe", Email: "jane@example.com"}

		// Serialize
		jsonBytes, err := json.Marshal(original)
		require.NoError(t, err)

		// Create wire format
		wireBytes := make([]byte, 5+len(jsonBytes))
		wireBytes[0] = 0
		binary.BigEndian.PutUint32(wireBytes[1:5], uint32(schema.ID()))
		copy(wireBytes[5:], jsonBytes)

		// Deserialize (skip wire format header)
		var decoded User
		err = json.Unmarshal(wireBytes[5:], &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.ID, decoded.ID)
		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Email, decoded.Email)

		t.Logf("JSON roundtrip verified for user: %s", decoded.Name)
	})

	t.Run("ComplexJSONSchema", func(t *testing.T) {
		subject := "go-json-complex-value"

		// Register schema
		schema, err := client.CreateSchema(subject, orderJSONSchema, srclient.Json)
		require.NoError(t, err)

		// Original data
		type Item struct {
			ProductID string `json:"productId"`
			Quantity  int    `json:"quantity"`
		}
		type Order struct {
			OrderID    string  `json:"orderId"`
			CustomerID string  `json:"customerId"`
			Amount     float64 `json:"amount"`
			Items      []Item  `json:"items"`
		}

		original := Order{
			OrderID:    "ORD-001",
			CustomerID: "CUST-123",
			Amount:     199.99,
			Items: []Item{
				{ProductID: "PROD-A", Quantity: 2},
				{ProductID: "PROD-B", Quantity: 1},
			},
		}

		// Serialize
		jsonBytes, err := json.Marshal(original)
		require.NoError(t, err)

		// Create wire format
		wireBytes := make([]byte, 5+len(jsonBytes))
		wireBytes[0] = 0
		binary.BigEndian.PutUint32(wireBytes[1:5], uint32(schema.ID()))
		copy(wireBytes[5:], jsonBytes)

		// Deserialize
		var decoded Order
		err = json.Unmarshal(wireBytes[5:], &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.OrderID, decoded.OrderID)
		assert.Equal(t, original.Amount, decoded.Amount)
		assert.Len(t, decoded.Items, 2)

		t.Logf("Complex JSON roundtrip verified: %s", decoded.OrderID)
	})
}

func TestJSONSchemaEvolution(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("AddOptionalProperty", func(t *testing.T) {
		v1Schema := `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"title": "Config",
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"value": {"type": "string"}
			},
			"required": ["name", "value"]
		}`

		v2Schema := `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"title": "Config",
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"value": {"type": "string"},
				"description": {"type": "string"}
			},
			"required": ["name", "value"]
		}`

		subject := "go-json-evolution-value"

		// Register v1
		schema1, err := client.CreateSchema(subject, v1Schema, srclient.Json)
		require.NoError(t, err)

		// Register v2
		schema2, err := client.CreateSchema(subject, v2Schema, srclient.Json)
		require.NoError(t, err)

		assert.NotEqual(t, schema1.ID(), schema2.ID(), "Different schemas should have different IDs")

		t.Logf("JSON schema evolution: v1 ID=%d, v2 ID=%d", schema1.ID(), schema2.ID())
	})
}
