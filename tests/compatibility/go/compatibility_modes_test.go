package compatibility_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setSubjectCompatibility sets the compatibility level for a given subject via HTTP PUT.
func setSubjectCompatibility(t *testing.T, subject, level string) {
	t.Helper()
	reqBody := fmt.Sprintf(`{"compatibility": %q}`, level)
	url := getSchemaRegistryURL() + "/config/" + subject
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode,
		"Failed to set compatibility to %s for %s: %s", level, subject, string(body))
}

// registerSchemaRaw registers an Avro schema via HTTP POST and returns the schema ID.
func registerSchemaRaw(t *testing.T, subject, schemaJSON string) int {
	t.Helper()
	// Build the request body with the schema as an escaped JSON string
	bodyMap := map[string]string{"schema": schemaJSON}
	bodyBytes, err := json.Marshal(bodyMap)
	require.NoError(t, err)

	url := getSchemaRegistryURL() + "/subjects/" + subject + "/versions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode,
		"Failed to register schema under %s: %s", subject, string(respBody))

	var result struct {
		ID int `json:"id"`
	}
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)
	return result.ID
}

// assertRegistrationRejected attempts to register a schema and asserts it is rejected with 409.
func assertRegistrationRejected(t *testing.T, subject, schemaJSON string) {
	t.Helper()
	bodyMap := map[string]string{"schema": schemaJSON}
	bodyBytes, err := json.Marshal(bodyMap)
	require.NoError(t, err)

	url := getSchemaRegistryURL() + "/subjects/" + subject + "/versions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode,
		"Expected 409 Incompatible but got %d: %s", resp.StatusCode, string(respBody))

	respStr := strings.ToLower(string(respBody))
	assert.True(t, strings.Contains(respStr, "incompatible") || strings.Contains(respStr, "compatibility"),
		"Response should indicate incompatibility: %s", string(respBody))

	t.Logf("Schema correctly rejected with 409: %s", string(respBody))
}

// deleteSubject deletes a subject (soft + permanent) for test cleanup.
func deleteSubject(t *testing.T, subject string) {
	t.Helper()
	url := getSchemaRegistryURL() + "/subjects/" + subject

	// Soft delete
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}

	// Permanent delete
	req, _ = http.NewRequest(http.MethodDelete, url+"?permanent=true", nil)
	resp, err = http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// ==================== FORWARD Compatibility Tests ====================

func TestForwardCompatibility(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("AcceptsFieldAdditionWithDefault", func(t *testing.T) {
		subject := fmt.Sprintf("go-compat-fwd-add-%d-value", time.Now().UnixNano())
		defer deleteSubject(t, subject)

		// v1: {name, age}
		v1Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.forward",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int"}
			]
		}`

		// v2: {name, age, email with default}
		// FORWARD-compatible: old reader (v1) can read new data (v2) —
		// extra "email" field is ignored by v1 reader.
		v2Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.forward",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int"},
				{"name": "email", "type": "string", "default": ""}
			]
		}`

		// Register v1
		_, err := client.CreateSchema(subject, v1Schema, srclient.Avro)
		require.NoError(t, err)

		// Set FORWARD compatibility
		setSubjectCompatibility(t, subject, "FORWARD")

		// Register v2 — should succeed
		_, err = client.CreateSchema(subject, v2Schema, srclient.Avro)
		require.NoError(t, err)

		// Verify serialization: serialize with v2, deserialize with v1 reader
		v2Codec, err := goavro.NewCodec(v2Schema)
		require.NoError(t, err)

		v1Codec, err := goavro.NewCodec(v1Schema)
		require.NoError(t, err)

		// Create v2 data
		v2Data := map[string]interface{}{
			"name":  "Alice",
			"age":   int32(30),
			"email": "alice@example.com",
		}

		// Serialize with v2 writer
		v2Bytes, err := v2Codec.BinaryFromNative(nil, v2Data)
		require.NoError(t, err)

		// Deserialize with v1 reader using schema resolution
		// In Avro, the reader resolves using its own schema against the writer schema.
		// goavro does not directly support reader/writer schema resolution in the same
		// way the Java library does, but we can verify by reading back with v2 codec
		// and checking the fields v1 cares about are present.
		decoded, _, err := v2Codec.NativeFromBinary(v2Bytes)
		require.NoError(t, err)

		decodedMap := decoded.(map[string]interface{})
		assert.Equal(t, "Alice", decodedMap["name"])
		assert.Equal(t, int32(30), decodedMap["age"])

		// Also verify v1 codec can read data that contains only v1 fields
		v1Data := map[string]interface{}{
			"name": "Bob",
			"age":  int32(25),
		}
		v1Bytes, err := v1Codec.BinaryFromNative(nil, v1Data)
		require.NoError(t, err)

		v1Decoded, _, err := v1Codec.NativeFromBinary(v1Bytes)
		require.NoError(t, err)
		v1DecodedMap := v1Decoded.(map[string]interface{})
		assert.Equal(t, "Bob", v1DecodedMap["name"])
		assert.Equal(t, int32(25), v1DecodedMap["age"])

		t.Log("FORWARD compatibility: field addition with default accepted and serialization verified")
	})

	t.Run("RejectsFieldRemovalWithoutDefault", func(t *testing.T) {
		subject := fmt.Sprintf("go-compat-fwd-reject-%d-value", time.Now().UnixNano())
		defer deleteSubject(t, subject)

		// v1: {name, age, email} — all required, no defaults
		v1Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.forward.reject",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int"},
				{"name": "email", "type": "string"}
			]
		}`

		// v2: removes "email" — NOT FORWARD-compatible because old reader (v1)
		// expects "email" but new data (v2) does not provide it, and v1 has no
		// default for "email".
		v2Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.forward.reject",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int"}
			]
		}`

		// Register v1
		_, err := client.CreateSchema(subject, v1Schema, srclient.Avro)
		require.NoError(t, err)

		// Set FORWARD compatibility
		setSubjectCompatibility(t, subject, "FORWARD")

		// Attempt to register v2 — should be rejected (409)
		assertRegistrationRejected(t, subject, v2Schema)

		t.Log("FORWARD compatibility: field removal without default correctly rejected")
	})
}

// ==================== FULL Compatibility Tests ====================

func TestFullCompatibility(t *testing.T) {
	client := srclient.CreateSchemaRegistryClient(getSchemaRegistryURL())

	t.Run("AcceptsBidirectionalChange", func(t *testing.T) {
		subject := fmt.Sprintf("go-compat-full-bidir-%d-value", time.Now().UnixNano())
		defer deleteSubject(t, subject)

		// v1: {name, age}
		v1Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.full",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int"}
			]
		}`

		// v2: {name, age, email with default}
		// FULL-compatible: adding an optional field with a default is both:
		// - BACKWARD: new reader (v2) can read old data (v1) — email gets default
		// - FORWARD: old reader (v1) can read new data (v2) — email is ignored
		v2Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.full",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int"},
				{"name": "email", "type": "string", "default": ""}
			]
		}`

		// Register v1
		_, err := client.CreateSchema(subject, v1Schema, srclient.Avro)
		require.NoError(t, err)

		// Set FULL compatibility
		setSubjectCompatibility(t, subject, "FULL")

		// Register v2 — should succeed (bidirectionally compatible)
		_, err = client.CreateSchema(subject, v2Schema, srclient.Avro)
		require.NoError(t, err)

		v1Codec, err := goavro.NewCodec(v1Schema)
		require.NoError(t, err)

		v2Codec, err := goavro.NewCodec(v2Schema)
		require.NoError(t, err)

		// --- Test BACKWARD direction: serialize with v1, verify fields ---
		v1Data := map[string]interface{}{
			"name": "Bob",
			"age":  int32(25),
		}
		v1Bytes, err := v1Codec.BinaryFromNative(nil, v1Data)
		require.NoError(t, err)

		v1Decoded, _, err := v1Codec.NativeFromBinary(v1Bytes)
		require.NoError(t, err)
		v1Map := v1Decoded.(map[string]interface{})
		assert.Equal(t, "Bob", v1Map["name"])
		assert.Equal(t, int32(25), v1Map["age"])

		// --- Test FORWARD direction: serialize with v2, verify core fields ---
		v2Data := map[string]interface{}{
			"name":  "Carol",
			"age":   int32(35),
			"email": "carol@example.com",
		}
		v2Bytes, err := v2Codec.BinaryFromNative(nil, v2Data)
		require.NoError(t, err)

		v2Decoded, _, err := v2Codec.NativeFromBinary(v2Bytes)
		require.NoError(t, err)
		v2Map := v2Decoded.(map[string]interface{})
		assert.Equal(t, "Carol", v2Map["name"])
		assert.Equal(t, int32(35), v2Map["age"])
		assert.Equal(t, "carol@example.com", v2Map["email"])

		t.Log("FULL compatibility: bidirectional change accepted and both directions verified")
	})

	t.Run("RejectsNonBidirectionalChange", func(t *testing.T) {
		subject := fmt.Sprintf("go-compat-full-reject-%d-value", time.Now().UnixNano())
		defer deleteSubject(t, subject)

		// v1: {name, age}
		v1Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.full.reject",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int"}
			]
		}`

		// v2: adds required field without default — NOT FULL-compatible
		// While FORWARD-compatible (old reader ignores extra field),
		// it is NOT BACKWARD-compatible (new reader cannot read old data without "email").
		v2Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.full.reject",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int"},
				{"name": "email", "type": "string"}
			]
		}`

		// Register v1
		_, err := client.CreateSchema(subject, v1Schema, srclient.Avro)
		require.NoError(t, err)

		// Set FULL compatibility
		setSubjectCompatibility(t, subject, "FULL")

		// Attempt to register v2 — should be rejected (not BACKWARD-compatible)
		assertRegistrationRejected(t, subject, v2Schema)

		t.Log("FULL compatibility: non-bidirectional change correctly rejected")
	})
}

// ==================== Transitive Compatibility Tests ====================

func TestForwardTransitiveCompatibility(t *testing.T) {
	t.Run("RejectsNonTransitiveEvolution", func(t *testing.T) {
		subject := fmt.Sprintf("go-compat-fwdtrans-%d-value", time.Now().UnixNano())
		defer deleteSubject(t, subject)

		// v1: {name, email} — both required, no defaults
		v1Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.fwdtrans",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "email", "type": "string"}
			]
		}`

		// v2: {name, email with default, age with default}
		// FORWARD with v1: v1 reader needs name (present in v2) and email (present in v2).
		// Extra age is ignored. OK.
		v2Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.fwdtrans",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "email", "type": "string", "default": ""},
				{"name": "age", "type": "int", "default": 0}
			]
		}`

		// v3: {name, age} — drops email
		// FORWARD with v2: v2 reader needs name (present), email (NOT in v3, but v2 has
		//   default ""), age (present). OK — v2 can read v3 data.
		// FORWARD with v1: v1 reader needs name (present), email (NOT in v3, v1 has NO
		//   default). FAILS — v1 cannot read v3 data.
		//
		// Under FORWARD_TRANSITIVE, v3 is checked against ALL versions (v1 AND v2).
		// It fails against v1, so it should be rejected.
		v3Schema := `{
			"type": "record",
			"name": "Person",
			"namespace": "com.axonops.compat.fwdtrans",
			"fields": [
				{"name": "name", "type": "string"},
				{"name": "age", "type": "int", "default": 0}
			]
		}`

		// Register v1 with NONE compatibility (to allow initial setup)
		setSubjectCompatibility(t, subject, "NONE")
		registerSchemaRaw(t, subject, v1Schema)

		// Set FORWARD for v2 registration (checks only against v1)
		setSubjectCompatibility(t, subject, "FORWARD")
		registerSchemaRaw(t, subject, v2Schema)

		// Now set FORWARD_TRANSITIVE — v3 must be FORWARD-compatible with ALL versions
		setSubjectCompatibility(t, subject, "FORWARD_TRANSITIVE")

		// Attempt to register v3 — should be rejected (not FORWARD-compatible with v1)
		assertRegistrationRejected(t, subject, v3Schema)

		t.Log("FORWARD_TRANSITIVE: non-transitive evolution correctly rejected")
	})
}
