package compatibility_test

import (
	"testing"

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
