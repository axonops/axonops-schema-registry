package serde_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// CEL Extras — Go-Only Additional Tests
//
// These tests exercise CEL rule scenarios beyond what the Java test suite
// covers, providing additional coverage for edge cases and combinations.
// ============================================================================

// TestCelConditionMultiFieldValidation verifies that a CEL CONDITION rule
// can validate across multiple fields in a single expression.
func TestCelConditionMultiFieldValidation(t *testing.T) {
	subject := uniqueSubject("cel-multi-field")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"Order","namespace":"com.axonops.test.celx","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "multi-field-check",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "message.amount > 0.0 && size(message.currency) == 3 && size(message.orderId) > 0",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	// Case A: empty orderId violates the multi-field check.
	_, err := ser.Serialize(topicFromSubject(subject), &Order{OrderID: "", Amount: 100, Currency: "USD"})
	require.Error(t, err, "serialization should fail for empty orderId")
	assert.True(t, isRuleError(err), "error should be a rule violation, got: %v", err)

	// Case B: All three fields valid -> should succeed.
	goodOrder := Order{OrderID: "ORD-MF", Amount: 50.0, Currency: "EUR"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &goodOrder)
	require.NoError(t, err, "serialization should succeed when all fields valid")

	var result Order
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialization should succeed")
	assert.Equal(t, "ORD-MF", result.OrderID)
	assert.Equal(t, 50.0, result.Amount)
	assert.Equal(t, "EUR", result.Currency)
}

// TestCelConditionWriteReadBothPaths verifies that a WRITEREAD mode rule
// fires on both serialization and deserialization.
func TestCelConditionWriteReadBothPaths(t *testing.T) {
	subject := uniqueSubject("cel-writeread")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"Order","namespace":"com.axonops.test.celx","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "amount-positive-both",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITEREAD",
					"expr": "message.amount > 0.0",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)

	// WRITE path: negative amount should fail at serialization.
	_, err := ser.Serialize(topicFromSubject(subject), &Order{OrderID: "WR-BAD", Amount: -10.0, Currency: "USD"})
	require.Error(t, err, "WRITE path should reject negative amount")
	assert.True(t, isRuleError(err), "error should be a rule violation, got: %v", err)

	// WRITE path: valid data should succeed and round-trip.
	deser := newRuleDeserializer(t, client)
	validOrder := Order{OrderID: "WR-OK", Amount: 42.0, Currency: "GBP"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &validOrder)
	require.NoError(t, err, "WRITE path should accept positive amount")

	var result Order
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "READ path should succeed for positive amount")
	assert.Equal(t, "WR-OK", result.OrderID)
	assert.Equal(t, 42.0, result.Amount)
}

// TestCelFieldTransformMultipleTags verifies that CEL_FIELD rules can target
// different tags with different transforms on the same schema.
func TestCelFieldTransformMultipleTags(t *testing.T) {
	subject := uniqueSubject("cel-multi-tag")
	defer deleteSubject(t, subject)

	// Schema with two differently-tagged fields.
	schemaStr := `{"type":"record","name":"UserProfile","namespace":"com.axonops.test.celx","fields":[{"name":"userId","type":"string"},{"name":"ssn","type":"string","confluent:tags":["PII"]},{"name":"email","type":"string","confluent:tags":["CONTACT"]},{"name":"creditCard","type":"string","confluent:tags":["PII"]}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "mask-pii",
					"kind": "TRANSFORM",
					"type": "CEL_FIELD",
					"mode": "READ",
					"tags": ["PII"],
					"expr": "typeName == 'STRING' ; '***MASKED***'",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	profile := UserProfile{
		UserID:     "U-123",
		SSN:        "123-45-6789",
		Email:      "user@example.com",
		CreditCard: "4111-1111-1111-1111",
	}
	bytes, err := ser.Serialize(topicFromSubject(subject), &profile)
	require.NoError(t, err, "serialization should succeed")

	var result UserProfile
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, "U-123", result.UserID, "userId should be unchanged (no tag)")
	assert.Equal(t, "***MASKED***", result.SSN, "SSN (PII tag) should be masked")
	assert.Equal(t, "user@example.com", result.Email, "email (CONTACT tag, not PII) should not be masked")
	assert.Equal(t, "***MASKED***", result.CreditCard, "creditCard (PII tag) should be masked")
}

// TestCelConditionMixedModes verifies that WRITE-mode and READ-mode rules
// on the same schema fire at the correct stage independently.
func TestCelConditionMixedModes(t *testing.T) {
	subject := uniqueSubject("cel-mixed-mode")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"OrderStatus","namespace":"com.axonops.test.celx","fields":[{"name":"orderId","type":"string"},{"name":"status","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "orderId-required-write",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "size(message.orderId) > 0",
					"onFailure": "ERROR"
				},
				{
					"name": "no-error-status-read",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "READ",
					"expr": "message.status != 'ERROR'",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	// Case A: Empty orderId → WRITE rule rejects.
	_, err := ser.Serialize(topicFromSubject(subject), &OrderStatus{OrderID: "", Status: "OK"})
	require.Error(t, err, "WRITE rule should reject empty orderId")
	assert.True(t, isRuleError(err), "error should be a rule violation, got: %v", err)

	// Case B: status=ERROR → WRITE succeeds, READ rejects.
	errStatus := OrderStatus{OrderID: "ORD-ERR", Status: "ERROR"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &errStatus)
	require.NoError(t, err, "WRITE should succeed (no WRITE rule for status)")

	var result OrderStatus
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.Error(t, err, "READ rule should reject ERROR status")
	assert.True(t, isRuleError(err), "error should be a rule violation, got: %v", err)

	// Case C: Valid data passes both WRITE and READ.
	okStatus := OrderStatus{OrderID: "ORD-OK", Status: "SHIPPED"}
	bytes, err = ser.Serialize(topicFromSubject(subject), &okStatus)
	require.NoError(t, err, "WRITE should succeed for valid data")

	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "READ should succeed for non-ERROR status")
	assert.Equal(t, "ORD-OK", result.OrderID)
	assert.Equal(t, "SHIPPED", result.Status)
}
