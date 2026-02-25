package serde_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// JSON Schema + CEL Data Contract Tests (Go)
//
// The Confluent Go SerDe client (confluent-kafka-go) only provides Avro
// serializer/deserializer with rule execution. JSON Schema and Protobuf
// SerDe are only available in the Java client. Therefore, these tests
// validate that the AxonOps Schema Registry correctly stores and returns
// CEL ruleSet definitions for JSON Schema subjects via the REST API.
// ============================================================================

// TestCelConditionStoredWithJsonSchema registers a JSON Schema with CEL
// domain rules and verifies that the rules are stored and returned in the
// schema version response.
func TestCelConditionStoredWithJsonSchema(t *testing.T) {
	subject := uniqueSubject("cel-jsonschema-stored")
	defer deleteSubject(t, subject)

	jsonSchemaStr := `{"$schema":"http://json-schema.org/draft-07/schema#","title":"Product","type":"object","properties":{"name":{"type":"string"},"price":{"type":"number"},"sku":{"type":"string"}},"required":["name","price","sku"]}`

	body := `{
		"schemaType": "JSON",
		"schema": "` + escapeJSON(jsonSchemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "nameNotEmpty",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "message.name != ''",
					"onFailure": "ERROR"
				},
				{
					"name": "pricePositive",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "message.price > 0",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	schemaID := registerSchemaViaHTTP(t, subject, body)
	assert.Greater(t, schemaID, 0, "schema should be registered with a positive ID")

	// Fetch version response and verify rules are present
	versionResp := getSchemaVersionResponse(t, subject, 1)
	assert.Contains(t, versionResp, "ruleSet", "version response should contain ruleSet")
	assert.Contains(t, versionResp, "nameNotEmpty", "version response should contain rule 'nameNotEmpty'")
	assert.Contains(t, versionResp, "pricePositive", "version response should contain rule 'pricePositive'")
	assert.Contains(t, versionResp, "CEL", "version response should contain rule type 'CEL'")
	assert.Contains(t, versionResp, "CONDITION", "version response should contain rule kind 'CONDITION'")

	// Parse and structurally verify the ruleSet
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(versionResp), &parsed), "version response should be valid JSON")

	// Verify schemaType is JSON
	schemaType, ok := parsed["schemaType"].(string)
	require.True(t, ok, "schemaType should be a string")
	assert.Equal(t, "JSON", schemaType, "schemaType should be JSON")

	// Verify ruleSet structure
	ruleSet, ok := parsed["ruleSet"].(map[string]interface{})
	require.True(t, ok, "ruleSet should be present and be an object")

	domainRules, ok := ruleSet["domainRules"].([]interface{})
	require.True(t, ok, "domainRules should be present and be an array")
	assert.Len(t, domainRules, 2, "should have 2 domain rules")

	// Verify first rule
	rule0, ok := domainRules[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "nameNotEmpty", rule0["name"])
	assert.Equal(t, "CEL", rule0["type"])
	assert.Equal(t, "CONDITION", rule0["kind"])
	assert.Equal(t, "WRITE", rule0["mode"])

	// Verify second rule
	rule1, ok := domainRules[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "pricePositive", rule1["name"])
	assert.Equal(t, "CEL", rule1["type"])
	assert.Equal(t, "CONDITION", rule1["kind"])

	t.Logf("CEL rules stored and returned with JSON Schema: schema ID %d, 2 rules verified", schemaID)
}

// TestCelConditionValidJsonSchema registers a JSON Schema with a CEL CONDITION
// rule via HTTP and verifies the schema can be fetched by the Go client,
// confirming the registry correctly handles JSON Schema + ruleSet combinations.
func TestCelConditionValidJsonSchema(t *testing.T) {
	subject := uniqueSubject("cel-jsonschema-valid")
	defer deleteSubject(t, subject)

	jsonSchemaStr := `{"$schema":"http://json-schema.org/draft-07/schema#","title":"Event","type":"object","properties":{"eventId":{"type":"string"},"timestamp":{"type":"integer"}},"required":["eventId","timestamp"]}`

	body := `{
		"schemaType": "JSON",
		"schema": "` + escapeJSON(jsonSchemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "eventIdRequired",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "message.eventId != ''",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	schemaID := registerSchemaViaHTTP(t, subject, body)
	assert.Greater(t, schemaID, 0, "schema should be registered with a positive ID")

	// Use the Go schema registry client to fetch the schema by ID.
	// This validates that the registry serves JSON Schema correctly
	// to the Go client even though Go doesn't have a JSON Schema serializer.
	client := newClient(t)
	schemaInfo, err := client.GetBySubjectAndID(subject, schemaID)
	require.NoError(t, err, "should be able to fetch JSON Schema via Go client")
	assert.Equal(t, "JSON", schemaInfo.SchemaType, "schema type should be JSON")

	// Verify the schema string is valid JSON
	var schemaMap map[string]interface{}
	err = json.Unmarshal([]byte(schemaInfo.Schema), &schemaMap)
	require.NoError(t, err, "schema string should be valid JSON")

	// Verify the ruleSet is returned when fetching by version
	versionResp := getSchemaVersionResponse(t, subject, 1)
	assert.Contains(t, versionResp, "eventIdRequired", "rule should be stored")
	assert.Contains(t, versionResp, "CEL", "rule type should be CEL")

	t.Logf("JSON Schema with CEL rules fetched successfully via Go client: schema ID %d", schemaID)
}
