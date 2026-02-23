package serde_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Global Policy Tests
//
// These tests verify that subject-level defaultRuleSet, overrideRuleSet,
// and rule inheritance across schema versions work correctly with the
// Confluent Go SerDe client.
// ============================================================================

const (
	orderPolicySchema = `{"type":"record","name":"Order","namespace":"com.axonops.test.policy","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"}]}`

	orderPolicyV2Schema = `{"type":"record","name":"Order","namespace":"com.axonops.test.policy","fields":[{"name":"orderId","type":"string"},{"name":"amount","type":"double"},{"name":"notes","type":["null","string"],"default":null}]}`

	contactSchema = `{"type":"record","name":"Contact","namespace":"com.axonops.test.policy","fields":[{"name":"name","type":"string"},{"name":"email","type":"string","confluent:tags":["PII"]}]}`
)

// TestDefaultRuleSetApplied verifies that a defaultRuleSet configured at the
// subject level is inherited by schemas registered without their own ruleSet.
func TestDefaultRuleSetApplied(t *testing.T) {
	subject := uniqueSubject("default-rule")
	defer deleteSubject(t, subject)

	// Set subject config with a defaultRuleSet that enforces amount > 0.
	setSubjectConfig(t, subject, `{"compatibility":"NONE","defaultRuleSet":{"domainRules":[{"name":"amount-positive","kind":"CONDITION","type":"CEL","mode":"WRITE","expr":"message.Amount > 0.0","onFailure":"ERROR"}]}}`)

	// Register the schema WITHOUT any ruleSet — it should inherit the default.
	registerSchemaViaHTTP(t, subject, `{"schema": `+jsonQuote(orderPolicySchema)+`}`)

	// Verify the inherited rule appears in the version response.
	versionResp := getSchemaVersionResponse(t, subject, 1)
	assert.True(t, strings.Contains(versionResp, "amount-positive"),
		"expected version response to contain inherited rule 'amount-positive', got: %s", versionResp)

	// Create client, serializer.
	client := newClient(t)
	ser := newRuleSerializer(t, client)
	topic := topicFromSubject(subject)

	// Negative amount should be rejected by the inherited default rule.
	_, err := ser.Serialize(topic, &OrderPolicy{OrderID: "ORD-BAD", Amount: -1.0})
	require.Error(t, err, "expected serialization to fail for negative amount")
	assert.True(t, isRuleError(err), "expected a rule error, got: %v", err)

	// Positive amount should succeed.
	bytes, err := ser.Serialize(topic, &OrderPolicy{OrderID: "ORD-GOOD", Amount: 100.0})
	require.NoError(t, err, "expected serialization to succeed for positive amount")
	assert.NotEmpty(t, bytes, "serialized bytes should not be empty")
}

// TestOverrideRuleSetEnforced verifies that an overrideRuleSet configured at
// the subject level takes precedence, and that schema-level rules are still
// enforced alongside the override.
func TestOverrideRuleSetEnforced(t *testing.T) {
	subject := uniqueSubject("override-rule")
	defer deleteSubject(t, subject)

	// Set subject config with an overrideRuleSet that enforces a strict range.
	setSubjectConfig(t, subject, `{"compatibility":"NONE","overrideRuleSet":{"domainRules":[{"name":"amount-range-override","kind":"CONDITION","type":"CEL","mode":"WRITE","expr":"message.Amount > 0.0 && message.Amount < 10000.0","onFailure":"ERROR"}]}}`)

	// Register the schema WITH its own permissive rule (orderId non-empty).
	body := `{"schema": ` + jsonQuote(orderPolicySchema) + `, "ruleSet": {"domainRules":[{"name":"orderId-required","kind":"CONDITION","type":"CEL","mode":"WRITE","expr":"size(message.OrderID) > 0","onFailure":"ERROR"}]}}`
	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	topic := topicFromSubject(subject)

	// Amount 50000 exceeds the override range — should fail.
	_, err := ser.Serialize(topic, &OrderPolicy{OrderID: "ORD-OVER", Amount: 50000.0})
	require.Error(t, err, "expected serialization to fail for amount exceeding override range")
	assert.True(t, isRuleError(err), "expected a rule error for override violation, got: %v", err)

	// Empty orderId violates the schema-level rule — should fail.
	_, err = ser.Serialize(topic, &OrderPolicy{OrderID: "", Amount: 100.0})
	require.Error(t, err, "expected serialization to fail for empty orderId")
	assert.True(t, isRuleError(err), "expected a rule error for empty orderId, got: %v", err)

	// Valid order within range and with non-empty orderId — should succeed.
	bytes, err := ser.Serialize(topic, &OrderPolicy{OrderID: "ORD-VALID", Amount: 100.0})
	require.NoError(t, err, "expected serialization to succeed for valid order")
	assert.NotEmpty(t, bytes, "serialized bytes should not be empty")
}

// TestRuleInheritanceFromV1 verifies that rules defined on v1 of a schema are
// inherited by v2 when v2 is registered without its own ruleSet.
func TestRuleInheritanceFromV1(t *testing.T) {
	subject := uniqueSubject("rule-inherit")
	defer deleteSubject(t, subject)

	// Set compatibility to NONE so we can evolve freely.
	setSubjectConfig(t, subject, `{"compatibility":"NONE"}`)

	// Register v1 with an explicit rule: amount must be positive.
	v1Body := `{"schema": ` + jsonQuote(orderPolicySchema) + `, "ruleSet": {"domainRules":[{"name":"amount-positive","kind":"CONDITION","type":"CEL","mode":"WRITE","expr":"message.Amount > 0.0","onFailure":"ERROR"}]}}`
	registerSchemaViaHTTP(t, subject, v1Body)

	// Register v2 WITHOUT any ruleSet — should inherit from v1.
	v2Body := `{"schema": ` + jsonQuote(orderPolicyV2Schema) + `}`
	registerSchemaViaHTTP(t, subject, v2Body)

	// Verify the inherited rule appears in the v2 response.
	v2Resp := getSchemaVersionResponse(t, subject, 2)
	assert.True(t, strings.Contains(v2Resp, "amount-positive"),
		"expected v2 response to contain inherited rule 'amount-positive', got: %s", v2Resp)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	topic := topicFromSubject(subject)

	// Negative amount should be rejected by the inherited rule.
	_, err := ser.Serialize(topic, &OrderPolicyV2{OrderID: "V2-BAD", Amount: -1.0})
	require.Error(t, err, "expected serialization to fail for negative amount on v2")
	assert.True(t, isRuleError(err), "expected a rule error, got: %v", err)

	// Positive amount with notes should succeed.
	notes := "priority order"
	bytes, err := ser.Serialize(topic, &OrderPolicyV2{OrderID: "V2-GOOD", Amount: 50.0, Notes: &notes})
	require.NoError(t, err, "expected serialization to succeed for valid v2 order")
	assert.NotEmpty(t, bytes, "serialized bytes should not be empty")
}

// TestPiiMaskingViaTagPropagation verifies that a CEL_FIELD TRANSFORM rule
// using PII tags correctly redacts tagged fields on READ while leaving
// non-tagged fields intact.
func TestPiiMaskingViaTagPropagation(t *testing.T) {
	subject := uniqueSubject("pii-mask")
	defer deleteSubject(t, subject)

	// Set compatibility to NONE.
	setSubjectConfig(t, subject, `{"compatibility":"NONE"}`)

	// Register Contact schema with a CEL_FIELD rule that masks PII-tagged fields.
	body := `{"schema": ` + jsonQuote(contactSchema) + `, "ruleSet": {"domainRules":[{"name":"mask-pii","kind":"TRANSFORM","type":"CEL_FIELD","mode":"READ","tags":["PII"],"expr":"typeName == 'STRING' ; 'REDACTED'","onFailure":"ERROR"}]}}`
	registerSchemaViaHTTP(t, subject, body)

	// Verify the rule and PII tag appear in the version response.
	versionResp := getSchemaVersionResponse(t, subject, 1)
	assert.True(t, strings.Contains(versionResp, "mask-pii"),
		"expected version response to contain rule 'mask-pii', got: %s", versionResp)
	assert.True(t, strings.Contains(versionResp, "PII"),
		"expected version response to contain tag 'PII', got: %s", versionResp)

	client := newClient(t)
	ser := newRuleSerializer(t, client)
	deser := newRuleDeserializer(t, client)
	topic := topicFromSubject(subject)

	// Serialize a contact with real data.
	original := Contact{Name: "Alice Smith", Email: "user@example.com"}
	bytes, err := ser.Serialize(topic, &original)
	require.NoError(t, err, "expected serialization to succeed")
	assert.NotEmpty(t, bytes, "serialized bytes should not be empty")

	// Deserialize — the PII-tagged email field should be redacted.
	var result Contact
	err = deser.DeserializeInto(topic, bytes, &result)
	require.NoError(t, err, "expected deserialization to succeed")

	assert.Equal(t, "Alice Smith", result.Name,
		"name should not be redacted (no PII tag)")
	assert.Equal(t, "REDACTED", result.Email,
		"email should be redacted via PII masking rule")
}
