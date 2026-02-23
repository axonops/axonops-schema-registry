package serde_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// CSFLE Extras — Go-Only Additional Tests
//
// These tests exercise CSFLE (Client-Side Field Level Encryption) scenarios
// beyond what the Java test suite covers, providing additional coverage for
// multi-subject KEK isolation, non-PII field integrity, and schema evolution.
// ============================================================================

// TestCsfleDifferentKeksPerSubject verifies that two subjects can use
// different KEK names independently, each encrypting and decrypting
// their own data without cross-contamination.
func TestCsfleDifferentKeksPerSubject(t *testing.T) {
	skipIfNoVault(t)

	subjectA := uniqueSubject("csfle-kek-a")
	subjectB := uniqueSubject("csfle-kek-b")
	defer deleteSubject(t, subjectA)
	defer deleteSubject(t, subjectB)

	kekNameA := "test-kek-multi-a-" + subjectA
	kekNameB := "test-kek-multi-b-" + subjectB

	customerSchema := `{"type":"record","name":"Customer","namespace":"com.axonops.test.csflextra","fields":[{"name":"customerId","type":"string"},{"name":"name","type":"string"},{"name":"ssn","type":"string","confluent:tags":["PII"]}]}`

	bodyA := buildSchemaWithEncryptRule(customerSchema, kekNameA)
	bodyB := buildSchemaWithEncryptRule(customerSchema, kekNameB)

	registerSchemaViaHTTP(t, subjectA, bodyA)
	registerSchemaViaHTTP(t, subjectB, bodyB)

	client := newClient(t)
	serA := newCsfleSerializer(t, client)
	serB := newCsfleSerializer(t, client)
	deserA := newCsfleDeserializer(t, client)
	deserB := newCsfleDeserializer(t, client)

	// Serialize data for subject A.
	custA := Customer{CustomerID: "CUST-A", Name: "Alice", SSN: "111-22-3333"}
	bytesA, err := serA.Serialize(topicFromSubject(subjectA), &custA)
	require.NoError(t, err, "serialization for subject A should succeed")

	// Serialize data for subject B.
	custB := Customer{CustomerID: "CUST-B", Name: "Bob", SSN: "444-55-6666"}
	bytesB, err := serB.Serialize(topicFromSubject(subjectB), &custB)
	require.NoError(t, err, "serialization for subject B should succeed")

	// Decrypt A's data with A's deserializer.
	var resultA Customer
	err = deserA.DeserializeInto(topicFromSubject(subjectA), bytesA, &resultA)
	require.NoError(t, err, "deserialization of A's data should succeed")
	assert.Equal(t, "111-22-3333", resultA.SSN, "A's SSN should decrypt correctly")

	// Decrypt B's data with B's deserializer.
	var resultB Customer
	err = deserB.DeserializeInto(topicFromSubject(subjectB), bytesB, &resultB)
	require.NoError(t, err, "deserialization of B's data should succeed")
	assert.Equal(t, "444-55-6666", resultB.SSN, "B's SSN should decrypt correctly")

	// Verify different KEKs were created.
	kekRespA := getKEK(t, kekNameA)
	kekRespB := getKEK(t, kekNameB)
	assert.NotEmpty(t, kekRespA, "KEK A should exist in DEK Registry")
	assert.NotEmpty(t, kekRespB, "KEK B should exist in DEK Registry")
	assert.True(t, strings.Contains(kekRespA, kekNameA), "KEK A response should contain its name")
	assert.True(t, strings.Contains(kekRespB, kekNameB), "KEK B response should contain its name")
}

// TestCsfleNonPiiFieldsNotEncrypted verifies that fields without the PII
// tag remain in plaintext in the serialized bytes, while PII-tagged fields
// are encrypted and not visible as plaintext.
func TestCsfleNonPiiFieldsNotEncrypted(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-nonpii")
	defer deleteSubject(t, subject)

	kekName := "test-kek-nonpii-" + subject

	paymentSchema := `{"type":"record","name":"PaymentEvent","namespace":"com.axonops.test.csflextra","fields":[{"name":"customerId","type":"string"},{"name":"creditCardNumber","type":"string","confluent:tags":["PII"]},{"name":"amount","type":"double"},{"name":"merchantName","type":"string"}]}`

	body := buildSchemaWithEncryptRule(paymentSchema, kekName)
	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newCsfleSerializer(t, client)

	payment := PaymentEvent{
		CustomerID:       "CUST-PAY",
		CreditCardNumber: "4111-1111-1111-1111",
		Amount:           299.99,
		MerchantName:     "ACME Store",
	}
	bytes, err := ser.Serialize(topicFromSubject(subject), &payment)
	require.NoError(t, err, "serialization should succeed")

	rawStr := string(bytes)

	// Non-PII fields should appear in raw bytes (Avro encodes strings in UTF-8).
	assert.True(t, strings.Contains(rawStr, "CUST-PAY"),
		"non-PII customerId should be in raw bytes")
	assert.True(t, strings.Contains(rawStr, "ACME Store"),
		"non-PII merchantName should be in raw bytes")

	// PII field (credit card) should NOT appear as plaintext.
	assert.False(t, strings.Contains(rawStr, "4111-1111-1111-1111"),
		"PII creditCardNumber should NOT be in raw bytes as plaintext")
}

// TestCsfleWithSchemaEvolution verifies that encrypted data serialized with
// v1 of a schema can still be decrypted after evolving to v2.
func TestCsfleWithSchemaEvolution(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-evolve")
	defer deleteSubject(t, subject)

	kekName := "test-kek-evolve-" + subject

	// v1: Customer with customerId, name, ssn (PII).
	v1Schema := `{"type":"record","name":"Customer","namespace":"com.axonops.test.csflextra","fields":[{"name":"customerId","type":"string"},{"name":"name","type":"string"},{"name":"ssn","type":"string","confluent:tags":["PII"]}]}`

	v1Body := buildSchemaWithEncryptRule(v1Schema, kekName)
	registerSchemaViaHTTP(t, subject, v1Body)

	// Serialize with v1.
	client1 := newClient(t)
	ser1 := newCsfleSerializer(t, client1)

	cust := Customer{CustomerID: "CUST-EVO", Name: "Evolving User", SSN: "999-88-7777"}
	v1Bytes, err := ser1.Serialize(topicFromSubject(subject), &cust)
	require.NoError(t, err, "v1 serialization should succeed")

	// Verify v1 round-trip works.
	deser1 := newCsfleDeserializer(t, client1)
	var v1Result Customer
	err = deser1.DeserializeInto(topicFromSubject(subject), v1Bytes, &v1Result)
	require.NoError(t, err, "v1 deserialization should succeed")
	assert.Equal(t, "999-88-7777", v1Result.SSN, "v1 SSN should decrypt correctly")

	// Evolve to v2: add an optional field.
	setSubjectConfig(t, subject, `{"compatibility":"NONE"}`)

	v2Schema := `{"type":"record","name":"Customer","namespace":"com.axonops.test.csflextra","fields":[{"name":"customerId","type":"string"},{"name":"name","type":"string"},{"name":"ssn","type":"string","confluent:tags":["PII"]},{"name":"loyaltyTier","type":["null","string"],"default":null}]}`

	v2Body := buildSchemaWithEncryptRule(v2Schema, kekName)
	registerSchemaViaHTTP(t, subject, v2Body)

	// Create fresh client to pick up v2 metadata. Decrypt v1 bytes using v2.
	client2 := newClient(t)
	deser2 := newCsfleDeserializer(t, client2)

	var v2Result Customer
	err = deser2.DeserializeInto(topicFromSubject(subject), v1Bytes, &v2Result)
	require.NoError(t, err, "v1 bytes should still decrypt with v2 schema")
	assert.Equal(t, "999-88-7777", v2Result.SSN,
		"SSN should decrypt correctly even after schema evolution")
	assert.Equal(t, "CUST-EVO", v2Result.CustomerID)
	assert.Equal(t, "Evolving User", v2Result.Name)
}
