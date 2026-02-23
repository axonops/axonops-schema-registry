package serde_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Avro schemas with confluent:tags for CSFLE encryption.
const customerSchema = `{"type":"record","name":"Customer","namespace":"com.axonops.test.csfle","fields":[{"name":"customerId","type":"string"},{"name":"name","type":"string"},{"name":"ssn","type":"string","confluent:tags":["PII"]}]}`

const userProfileSchema = `{"type":"record","name":"UserProfile","namespace":"com.axonops.test.csfle","fields":[{"name":"userId","type":"string"},{"name":"ssn","type":"string","confluent:tags":["PII"]},{"name":"email","type":"string","confluent:tags":["PII"]},{"name":"creditCard","type":"string","confluent:tags":["PII"]}]}`

const paymentEventSchema = `{"type":"record","name":"PaymentEvent","namespace":"com.axonops.test.csfle","fields":[{"name":"customerId","type":"string"},{"name":"creditCardNumber","type":"string","confluent:tags":["PII"]},{"name":"amount","type":"double"},{"name":"merchantName","type":"string"}]}`

// TestCsfleEncryptDecryptRoundTrip verifies that a Customer record can be
// serialized with CSFLE encryption and deserialized back with all fields intact.
func TestCsfleEncryptDecryptRoundTrip(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-roundtrip")
	kekName := "kek-roundtrip-" + subject
	defer deleteSubject(t, subject)

	body := buildSchemaWithEncryptRule(customerSchema, kekName)
	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newCsfleSerializer(t, client)
	deser := newCsfleDeserializer(t, client)

	original := Customer{
		CustomerID: "CUST-001",
		Name:       "Jane Doe",
		SSN:        "123-45-6789",
	}

	topic := topicFromSubject(subject)
	bytes, err := ser.Serialize(topic, &original)
	require.NoError(t, err, "serialization should succeed")
	require.NotEmpty(t, bytes, "serialized bytes should not be empty")

	// Note: the ENCRYPT rule modifies the original struct's PII fields in-place
	// during serialization, so we compare against literal expected values.
	var result Customer
	err = deser.DeserializeInto(topic, bytes, &result)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, "CUST-001", result.CustomerID)
	assert.Equal(t, "Jane Doe", result.Name)
	assert.Equal(t, "123-45-6789", result.SSN)
}

// TestCsfleRawBytesNoPlaintextPII verifies that PII-tagged fields are not
// present as plaintext in the serialized byte stream.
func TestCsfleRawBytesNoPlaintextPII(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-noplain")
	kekName := "kek-noplain-" + subject
	defer deleteSubject(t, subject)

	body := buildSchemaWithEncryptRule(customerSchema, kekName)
	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newCsfleSerializer(t, client)

	original := Customer{
		CustomerID: "CUST-002",
		Name:       "John Smith",
		SSN:        "123-45-6789",
	}

	topic := topicFromSubject(subject)
	bytes, err := ser.Serialize(topic, &original)
	require.NoError(t, err, "serialization should succeed")
	require.NotEmpty(t, bytes, "serialized bytes should not be empty")

	assert.False(t, strings.Contains(string(bytes), "123-45-6789"),
		"raw bytes must not contain plaintext SSN")
}

// TestCsfleMultiplePIIFields verifies that all PII-tagged fields in a schema
// with multiple sensitive fields are encrypted and round-trip correctly.
func TestCsfleMultiplePIIFields(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-multipii")
	kekName := "kek-multipii-" + subject
	defer deleteSubject(t, subject)

	body := buildSchemaWithEncryptRule(userProfileSchema, kekName)
	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newCsfleSerializer(t, client)
	deser := newCsfleDeserializer(t, client)

	original := UserProfile{
		UserID:     "USER-100",
		SSN:        "987-65-4321",
		Email:      "secret@example.com",
		CreditCard: "4111-1111-1111-1111",
	}

	topic := topicFromSubject(subject)
	bytes, err := ser.Serialize(topic, &original)
	require.NoError(t, err, "serialization should succeed")
	require.NotEmpty(t, bytes, "serialized bytes should not be empty")

	rawStr := string(bytes)
	assert.False(t, strings.Contains(rawStr, "987-65-4321"),
		"raw bytes must not contain plaintext SSN")
	assert.False(t, strings.Contains(rawStr, "secret@example.com"),
		"raw bytes must not contain plaintext email")
	assert.False(t, strings.Contains(rawStr, "4111-1111-1111-1111"),
		"raw bytes must not contain plaintext credit card")

	// Note: ENCRYPT modifies original PII fields in-place, use literal values.
	var result UserProfile
	err = deser.DeserializeInto(topic, bytes, &result)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, "USER-100", result.UserID)
	assert.Equal(t, "987-65-4321", result.SSN)
	assert.Equal(t, "secret@example.com", result.Email)
	assert.Equal(t, "4111-1111-1111-1111", result.CreditCard)
}

// TestCsfleCreditCardProtection verifies that a credit card number tagged as PII
// is encrypted in the wire format and that non-PII fields remain intact.
func TestCsfleCreditCardProtection(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-cc")
	kekName := "kek-cc-" + subject
	defer deleteSubject(t, subject)

	body := buildSchemaWithEncryptRule(paymentEventSchema, kekName)
	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newCsfleSerializer(t, client)
	deser := newCsfleDeserializer(t, client)

	original := PaymentEvent{
		CustomerID:       "CUST-PAY-001",
		CreditCardNumber: "4532-0150-1234-5678",
		Amount:           149.99,
		MerchantName:     "Coffee Shop",
	}

	topic := topicFromSubject(subject)
	bytes, err := ser.Serialize(topic, &original)
	require.NoError(t, err, "serialization should succeed")
	require.NotEmpty(t, bytes, "serialized bytes should not be empty")

	assert.False(t, strings.Contains(string(bytes), "4532-0150-1234-5678"),
		"raw bytes must not contain plaintext credit card number")

	// Note: ENCRYPT modifies original PII fields in-place, use literal values.
	var result PaymentEvent
	err = deser.DeserializeInto(topic, bytes, &result)
	require.NoError(t, err, "deserialization should succeed")

	assert.Equal(t, "4532-0150-1234-5678", result.CreditCardNumber)
	assert.Equal(t, "CUST-PAY-001", result.CustomerID)
	assert.InDelta(t, 149.99, result.Amount, 0.001)
	assert.Equal(t, "Coffee Shop", result.MerchantName)
}

// TestCsfleDekCaching verifies that a DEK cached during serialization allows
// a second deserializer (without an explicit Vault token) to decrypt the data.
func TestCsfleDekCaching(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-dekcache")
	kekName := "kek-dekcache-" + subject
	defer deleteSubject(t, subject)

	body := buildSchemaWithEncryptRule(customerSchema, kekName)
	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newCsfleSerializer(t, client)

	original := Customer{
		CustomerID: "CUST-CACHE",
		Name:       "Cache Test",
		SSN:        "555-66-7777",
	}

	topic := topicFromSubject(subject)
	bytes, err := ser.Serialize(topic, &original)
	require.NoError(t, err, "serialization should succeed")
	require.NotEmpty(t, bytes, "serialized bytes should not be empty")

	assert.False(t, strings.Contains(string(bytes), "555-66-7777"),
		"raw bytes must not contain plaintext SSN")

	// Use a rule deserializer without explicit Vault token — should still
	// work because the DEK is cached in the client-side encryption library.
	deser2 := newRuleDeserializer(t, client)

	// Note: ENCRYPT modifies original PII fields in-place, use literal values.
	var result Customer
	err = deser2.DeserializeInto(topic, bytes, &result)
	require.NoError(t, err, "deserialization with cached DEK should succeed")

	assert.Equal(t, "CUST-CACHE", result.CustomerID)
	assert.Equal(t, "Cache Test", result.Name)
	assert.Equal(t, "555-66-7777", result.SSN)
}

// TestCsfleDekAutoCreated verifies that a Data Encryption Key (DEK) is
// automatically created upon the first serialization of an encrypted schema.
func TestCsfleDekAutoCreated(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-dekauto")
	kekName := "kek-dekauto-" + subject
	defer deleteSubject(t, subject)

	body := buildSchemaWithEncryptRule(customerSchema, kekName)
	registerSchemaViaHTTP(t, subject, body)

	// Before serialization, no DEK should exist.
	dekBefore := getDEK(t, kekName, subject)
	assert.Empty(t, dekBefore, "DEK should not exist before first serialization")

	client := newClient(t)
	ser := newCsfleSerializer(t, client)

	original := Customer{
		CustomerID: "CUST-DEKAUTO",
		Name:       "DEK Auto",
		SSN:        "111-22-3333",
	}

	topic := topicFromSubject(subject)
	bytes, err := ser.Serialize(topic, &original)
	require.NoError(t, err, "serialization should succeed")
	require.NotEmpty(t, bytes, "serialized bytes should not be empty")

	// After serialization, the DEK should have been auto-created.
	dekAfter := getDEK(t, kekName, subject)
	assert.NotEmpty(t, dekAfter, "DEK should exist after first serialization")
	assert.True(t, strings.Contains(dekAfter, "encryptedKeyMaterial"),
		"DEK response should contain encryptedKeyMaterial")
}

// TestCsfleKekAutoCreated verifies that a Key Encryption Key (KEK) is
// automatically created in the registry upon the first encrypted serialization.
func TestCsfleKekAutoCreated(t *testing.T) {
	skipIfNoVault(t)

	subject := uniqueSubject("csfle-kekauto")
	kekName := "kek-kekauto-" + subject

	// Before schema registration, no KEK should exist.
	kekBefore := getKEK(t, kekName)
	assert.Empty(t, kekBefore, "KEK should not exist before schema registration")

	defer deleteSubject(t, subject)

	body := buildSchemaWithEncryptRule(customerSchema, kekName)
	registerSchemaViaHTTP(t, subject, body)

	client := newClient(t)
	ser := newCsfleSerializer(t, client)

	original := Customer{
		CustomerID: "CUST-KEKAUTO",
		Name:       "KEK Auto",
		SSN:        "444-55-6666",
	}

	topic := topicFromSubject(subject)
	bytes, err := ser.Serialize(topic, &original)
	require.NoError(t, err, "serialization should succeed")
	require.NotEmpty(t, bytes, "serialized bytes should not be empty")

	// After serialization, the KEK should have been auto-created.
	kekAfter := getKEK(t, kekName)
	assert.NotEmpty(t, kekAfter, "KEK should exist after first serialization")
	assert.True(t, strings.Contains(kekAfter, "hcvault"),
		"KEK response should reference hcvault as the KMS type")
}
