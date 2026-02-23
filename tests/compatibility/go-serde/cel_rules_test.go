package serde_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// CEL Data Contract Tests
//
// These tests validate CEL-based data contract rules (CONDITION and
// FIELD-level TRANSFORM/CONDITION) through the Confluent Go SerDe client
// against the AxonOps Schema Registry.
// ============================================================================

// TestCelConditionValidDataPasses verifies that a CEL CONDITION rule on WRITE
// mode allows serialization when the condition is satisfied.
func TestCelConditionValidDataPasses(t *testing.T) {
	subject := uniqueSubject("cel-cond-valid")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"Order","namespace":"com.axonops.test.cel","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "amount-positive",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "message.Amount > 0.0",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	order := Order{OrderID: "ORD-001", Amount: 100.50, Currency: "USD"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &order)
	require.NoError(t, err, "serialization should succeed for valid data")

	var result Order
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, "ORD-001", result.OrderID)
	assert.Equal(t, 100.50, result.Amount)
	assert.Equal(t, "USD", result.Currency)
}

// TestCelConditionInvalidDataRejected verifies that a CEL CONDITION rule on
// WRITE mode rejects serialization when the condition is violated.
func TestCelConditionInvalidDataRejected(t *testing.T) {
	subject := uniqueSubject("cel-cond-invalid")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"Order","namespace":"com.axonops.test.cel","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "amount-positive",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "message.Amount > 0.0",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)

	order := Order{OrderID: "ORD-BAD", Amount: -5.0, Currency: "USD"}
	_, err := ser.Serialize(topicFromSubject(subject), &order)
	require.Error(t, err, "serialization should fail for negative amount")
	assert.True(t, isRuleError(err), "error should be a rule violation, got: %v", err)
}

// TestCelConditionOnReadRejectsAtDeserialization verifies that a CEL CONDITION
// rule with mode=READ allows serialization but rejects at deserialization.
func TestCelConditionOnReadRejectsAtDeserialization(t *testing.T) {
	subject := uniqueSubject("cel-cond-read")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"OrderStatus","namespace":"com.axonops.test.cel","fields":[{"name":"orderId","type":"string"},{"name":"status","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "no-cancelled-on-read",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "READ",
					"expr": "message.Status != 'CANCELLED'",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	// Serialization should succeed because the rule is READ mode only.
	orderStatus := OrderStatus{OrderID: "ORD-CANCEL", Status: "CANCELLED"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &orderStatus)
	require.NoError(t, err, "serialization should succeed (rule is READ mode)")

	// Deserialization should fail because the rule fires on READ.
	var result OrderStatus
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.Error(t, err, "deserialization should fail for CANCELLED status")
	assert.True(t, isRuleError(err), "error should be a rule violation, got: %v", err)
}

// TestMultipleCelConditionsChained verifies that multiple CEL CONDITION rules
// are evaluated in order and all must pass.
func TestMultipleCelConditionsChained(t *testing.T) {
	subject := uniqueSubject("cel-cond-multi")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"Order","namespace":"com.axonops.test.cel","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "amount-positive",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "message.Amount > 0.0",
					"onFailure": "ERROR"
				},
				{
					"name": "currency-three-chars",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "size(message.Currency) == 3",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	// Case A: Valid amount but currency is only 2 chars -> should fail.
	badOrder := Order{OrderID: "ORD-SHORT", Amount: 100, Currency: "US"}
	_, err := ser.Serialize(topicFromSubject(subject), &badOrder)
	require.Error(t, err, "serialization should fail for 2-char currency")
	assert.True(t, isRuleError(err), "error should be a rule violation, got: %v", err)

	// Case B: Both conditions satisfied -> should succeed.
	goodOrder := Order{OrderID: "ORD-GOOD", Amount: 100, Currency: "USD"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &goodOrder)
	require.NoError(t, err, "serialization should succeed when all conditions pass")

	var result Order
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialization should succeed")
	assert.Equal(t, "ORD-GOOD", result.OrderID)
	assert.Equal(t, float64(100), result.Amount)
	assert.Equal(t, "USD", result.Currency)
}

// TestCelFieldTransformMaskSsnOnRead verifies that a CEL_FIELD TRANSFORM rule
// with PII tags masks the SSN field on deserialization.
func TestCelFieldTransformMaskSsnOnRead(t *testing.T) {
	subject := uniqueSubject("cel-field-mask")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"User","namespace":"com.axonops.test.cel","fields":[{"name":"name","type":"string"},{"name":"ssn","type":"string","confluent:tags":["PII"]}]}`

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
					"expr": "typeName == 'STRING' ; 'XXX-XX-' + value[7:11]",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	user := User{Name: "Jane Doe", SSN: "123-45-6789"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &user)
	require.NoError(t, err, "serialization should succeed")

	var result User
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, "Jane Doe", result.Name, "name should be unchanged")
	assert.Equal(t, "XXX-XX-6789", result.SSN, "SSN should be masked on read")
}

// TestCelFieldConditionRejectsEmptyPii verifies that a CEL_FIELD CONDITION
// rule rejects empty PII fields on WRITE.
func TestCelFieldConditionRejectsEmptyPii(t *testing.T) {
	subject := uniqueSubject("cel-field-cond")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"User","namespace":"com.axonops.test.cel","fields":[{"name":"name","type":"string"},{"name":"ssn","type":"string","confluent:tags":["PII"]}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "pii-not-empty",
					"kind": "CONDITION",
					"type": "CEL_FIELD",
					"mode": "WRITE",
					"tags": ["PII"],
					"expr": "typeName == 'STRING' ; value != ''",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)

	// Case A: Empty SSN -> should fail.
	emptyUser := User{Name: "Empty SSN", SSN: ""}
	_, err := ser.Serialize(topicFromSubject(subject), &emptyUser)
	require.Error(t, err, "serialization should fail for empty PII field")
	assert.True(t, isRuleError(err), "error should be a rule violation, got: %v", err)

	// Case B: Valid SSN -> should succeed.
	validUser := User{Name: "Valid SSN", SSN: "123-45-6789"}
	_, err = ser.Serialize(topicFromSubject(subject), &validUser)
	require.NoError(t, err, "serialization should succeed for non-empty PII field")
}

// TestCelFieldTransformNormalizeCountry verifies that a CEL_FIELD TRANSFORM
// rule normalizes the country field to uppercase on WRITE.
func TestCelFieldTransformNormalizeCountry(t *testing.T) {
	subject := uniqueSubject("cel-field-upper")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"Address","namespace":"com.axonops.test.cel","fields":[{"name":"street","type":"string"},{"name":"country","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "normalize-country",
					"kind": "TRANSFORM",
					"type": "CEL_FIELD",
					"mode": "WRITE",
					"expr": "name == 'country' ; value.upperAscii()",
					"onFailure": "ERROR"
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	addr := Address{Street: "123 Main St", Country: "us"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &addr)
	require.NoError(t, err, "serialization should succeed")

	var result Address
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, "123 Main St", result.Street, "street should be unchanged")
	assert.Equal(t, "US", result.Country, "country should be uppercased by WRITE transform")
}

// TestDisabledRuleSkipped verifies that a rule with "disabled":true is not
// evaluated and data passes through without enforcement.
func TestDisabledRuleSkipped(t *testing.T) {
	subject := uniqueSubject("cel-disabled")
	defer deleteSubject(t, subject)

	schemaStr := `{"type":"record","name":"Order","namespace":"com.axonops.test.cel","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"currency","type":"string"}]}`

	body := `{
		"schemaType": "AVRO",
		"schema": "` + escapeJSON(schemaStr) + `",
		"ruleSet": {
			"domainRules": [
				{
					"name": "high-amount-only",
					"kind": "CONDITION",
					"type": "CEL",
					"mode": "WRITE",
					"expr": "message.Amount > 1000.0",
					"onFailure": "ERROR",
					"disabled": true
				}
			]
		}
	}`

	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)

	// Amount is 5.0 which violates the rule, but the rule is disabled.
	order := Order{OrderID: "ORD-LOW", Amount: 5.0, Currency: "USD"}
	bytes, err := ser.Serialize(topicFromSubject(subject), &order)
	require.NoError(t, err, "serialization should succeed (rule is disabled)")

	var result Order
	err = deser.DeserializeInto(topicFromSubject(subject), bytes, &result)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, "ORD-LOW", result.OrderID)
	assert.Equal(t, 5.0, result.Amount)
	assert.Equal(t, "USD", result.Currency)
}
