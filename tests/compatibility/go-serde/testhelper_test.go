package serde_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avrov2"
	"github.com/stretchr/testify/require"

	// Rule executor side-effect imports (auto-discovered via init())
	_ "github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/rules/cel"
	_ "github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/rules/encryption"
	_ "github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/rules/encryption/hcvault"
	_ "github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/rules/jsonata"
)

const schemaRegistryContentType = "application/vnd.schemaregistry.v1+json"

// ============================================================================
// Environment Helpers
// ============================================================================

func getRegistryURL() string {
	if url := os.Getenv("SCHEMA_REGISTRY_URL"); url != "" {
		return url
	}
	return "http://localhost:8081"
}

func getVaultURL() string {
	if url := os.Getenv("VAULT_URL"); url != "" {
		return url
	}
	return "http://localhost:18200"
}

func getVaultToken() string {
	if token := os.Getenv("VAULT_TOKEN"); token != "" {
		return token
	}
	return "test-root-token"
}

func getVaultBaseURL() string {
	return strings.TrimRight(getVaultURL(), "/")
}

func skipIfNoVault(t *testing.T) {
	t.Helper()
	vaultURL := getVaultURL()
	resp, err := http.Get(vaultURL + "/v1/sys/health")
	if err != nil {
		t.Skipf("Vault not accessible at %s: %v", vaultURL, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 429 && resp.StatusCode != 472 {
		t.Skipf("Vault not healthy at %s: HTTP %d", vaultURL, resp.StatusCode)
	}
}

// ============================================================================
// Unique Names and Subject Utilities
// ============================================================================

func uniqueSubject(prefix string) string {
	return fmt.Sprintf("%s-%d-value", prefix, time.Now().UnixMilli())
}

func topicFromSubject(subject string) string {
	return strings.TrimSuffix(subject, "-value")
}

// ============================================================================
// HTTP Helpers
// ============================================================================

func httpDo(t *testing.T, method, url, body string) (*http.Response, error) {
	t.Helper()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", schemaRegistryContentType)
	}
	req.Header.Set("Accept", schemaRegistryContentType)
	return http.DefaultClient.Do(req)
}

// registerSchemaViaHTTP registers a schema with optional ruleSet/metadata via
// the REST API and returns the global schema ID.
func registerSchemaViaHTTP(t *testing.T, subject, body string) int {
	t.Helper()
	url := getRegistryURL() + "/subjects/" + subject + "/versions"
	resp, err := httpDo(t, "POST", url, body)
	require.NoError(t, err, "failed to register schema for %s", subject)
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	require.Equal(t, 200, resp.StatusCode,
		"failed to register schema for %s: HTTP %d - %s", subject, resp.StatusCode, string(respBody))

	var result struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.Unmarshal(respBody, &result), "failed to parse schema ID response")
	return result.ID
}

// getSchemaVersionResponse fetches a schema version and returns the raw JSON.
func getSchemaVersionResponse(t *testing.T, subject string, version int) string {
	t.Helper()
	url := fmt.Sprintf("%s/subjects/%s/versions/%d", getRegistryURL(), subject, version)
	resp, err := httpDo(t, "GET", url, "")
	require.NoError(t, err, "failed to get %s v%d", subject, version)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, 200, resp.StatusCode,
		"failed to get %s v%d: HTTP %d - %s", subject, version, resp.StatusCode, string(body))
	return string(body)
}

// deleteSubject permanently deletes a subject (soft then hard delete).
// Errors are ignored since this is used for cleanup.
func deleteSubject(t *testing.T, subject string) {
	t.Helper()
	url := getRegistryURL() + "/subjects/" + subject
	if resp, err := httpDo(t, "DELETE", url, ""); err == nil {
		resp.Body.Close()
	}
	if resp, err := httpDo(t, "DELETE", url+"?permanent=true", ""); err == nil {
		resp.Body.Close()
	}
}

// setSubjectConfig sets subject-level config (compatibility, defaultRuleSet, etc.).
func setSubjectConfig(t *testing.T, subject, body string) {
	t.Helper()
	url := getRegistryURL() + "/config/" + subject
	resp, err := httpDo(t, "PUT", url, body)
	require.NoError(t, err, "failed to set config for %s", subject)
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	require.Equal(t, 200, resp.StatusCode,
		"failed to set config for %s: HTTP %d - %s", subject, resp.StatusCode, string(respBody))
}

// getKEK fetches a KEK from the DEK Registry. Returns empty string if not found.
func getKEK(t *testing.T, kekName string) string {
	t.Helper()
	url := getRegistryURL() + "/dek-registry/v1/keks/" + kekName
	resp, err := httpDo(t, "GET", url, "")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// getDEK fetches a DEK from the DEK Registry. Returns empty string if not found.
func getDEK(t *testing.T, kekName, subject string) string {
	t.Helper()
	url := getRegistryURL() + "/dek-registry/v1/keks/" + kekName + "/deks/" + subject
	resp, err := httpDo(t, "GET", url, "")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// ============================================================================
// Schema Registry Client & SerDe Factories
// ============================================================================

func newClient(t *testing.T) schemaregistry.Client {
	t.Helper()
	conf := schemaregistry.NewConfig(getRegistryURL())
	client, err := schemaregistry.NewClient(conf)
	require.NoError(t, err, "failed to create schema registry client")
	return client
}

// newRuleSerializer creates an avrov2 Serializer with AutoRegisterSchemas=false
// and UseLatestVersion=true for data contract rule execution.
func newRuleSerializer(t *testing.T, client schemaregistry.Client) *avrov2.Serializer {
	t.Helper()
	conf := avrov2.NewSerializerConfig()
	conf.AutoRegisterSchemas = false
	conf.UseLatestVersion = true
	ser, err := avrov2.NewSerializer(client, serde.ValueSerde, conf)
	require.NoError(t, err, "failed to create rule-aware serializer")
	return ser
}

// newRuleDeserializer creates an avrov2 Deserializer for rule execution.
func newRuleDeserializer(t *testing.T, client schemaregistry.Client) *avrov2.Deserializer {
	t.Helper()
	conf := avrov2.NewDeserializerConfig()
	deser, err := avrov2.NewDeserializer(client, serde.ValueSerde, conf)
	require.NoError(t, err, "failed to create rule-aware deserializer")
	return deser
}

// newCsfleSerializer creates an avrov2 Serializer configured for CSFLE.
func newCsfleSerializer(t *testing.T, client schemaregistry.Client) *avrov2.Serializer {
	t.Helper()
	t.Setenv("VAULT_TOKEN", getVaultToken())
	conf := avrov2.NewSerializerConfig()
	conf.AutoRegisterSchemas = false
	conf.UseLatestVersion = true
	conf.RuleConfig = map[string]string{
		"secret.access.key": getVaultToken(),
	}
	ser, err := avrov2.NewSerializer(client, serde.ValueSerde, conf)
	require.NoError(t, err, "failed to create CSFLE serializer")
	return ser
}

// newCsfleDeserializer creates an avrov2 Deserializer configured for CSFLE.
func newCsfleDeserializer(t *testing.T, client schemaregistry.Client) *avrov2.Deserializer {
	t.Helper()
	t.Setenv("VAULT_TOKEN", getVaultToken())
	conf := avrov2.NewDeserializerConfig()
	conf.RuleConfig = map[string]string{
		"secret.access.key": getVaultToken(),
	}
	deser, err := avrov2.NewDeserializer(client, serde.ValueSerde, conf)
	require.NoError(t, err, "failed to create CSFLE deserializer")
	return deser
}

// ============================================================================
// JSON Helpers
// ============================================================================

// escapeJSON escapes a raw JSON string for embedding inside a JSON string value.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// jsonQuote compacts a raw schema string and wraps it in JSON string quotes.
func jsonQuote(raw string) string {
	compacted := strings.Join(strings.Fields(raw), " ")
	return `"` + escapeJSON(compacted) + `"`
}

// buildSchemaWithEncryptRule builds JSON for registering a schema with ENCRYPT rule.
func buildSchemaWithEncryptRule(avroSchema, kekName string) string {
	vaultBase := getVaultBaseURL()
	return fmt.Sprintf(`{
		"schemaType": "AVRO",
		"schema": %s,
		"ruleSet": {
			"domainRules": [
				{
					"name": "encrypt-pii",
					"kind": "TRANSFORM",
					"type": "ENCRYPT",
					"mode": "WRITEREAD",
					"tags": ["PII"],
					"params": {
						"encrypt.kek.name": "%s",
						"encrypt.kms.type": "hcvault",
						"encrypt.kms.key.id": "%s/transit/keys/test-key"
					},
					"onFailure": "ERROR,NONE"
				}
			]
		}
	}`, jsonQuote(avroSchema), kekName, vaultBase)
}

// isRuleError checks if an error is from a data contract rule violation.
func isRuleError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Rule") ||
		strings.Contains(msg, "rule") ||
		strings.Contains(msg, "condition") ||
		strings.Contains(msg, "Condition")
}

// ============================================================================
// Go Struct Definitions (avro tags for hamba/avro v2 field mapping)
// ============================================================================

// --- CEL Test Structs ---

type Order struct {
	OrderID  string  `avro:"orderId" json:"orderId"`
	Amount   float64 `avro:"amount" json:"amount"`
	Currency string  `avro:"currency" json:"currency"`
}

type OrderStatus struct {
	OrderID string `avro:"orderId" json:"orderId"`
	Status  string `avro:"status" json:"status"`
}

type User struct {
	Name string `avro:"name" json:"name"`
	SSN  string `avro:"ssn" json:"ssn"`
}

type Address struct {
	Street  string `avro:"street" json:"street"`
	Country string `avro:"country" json:"country"`
}

// --- Migration Test Structs ---

type OrderV1 struct {
	OrderID string `avro:"orderId" json:"orderId"`
	State   string `avro:"state" json:"state"`
}

type OrderV2 struct {
	OrderID string `avro:"orderId" json:"orderId"`
	Status  string `avro:"status" json:"status"`
}

type PaymentV1 struct {
	ID     string  `avro:"id" json:"id"`
	Amount float64 `avro:"amount" json:"amount"`
}

type PaymentV2 struct {
	ID       string  `avro:"id" json:"id"`
	Amount   float64 `avro:"amount" json:"amount"`
	Currency string  `avro:"currency" json:"currency"`
}

type PersonV1 struct {
	FirstName string `avro:"firstName" json:"firstName"`
	LastName  string `avro:"lastName" json:"lastName"`
}

type PersonV2 struct {
	FullName string `avro:"fullName" json:"fullName"`
}

// --- CSFLE Test Structs ---

type Customer struct {
	CustomerID string `avro:"customerId" json:"customerId"`
	Name       string `avro:"name" json:"name"`
	SSN        string `avro:"ssn" json:"ssn"`
}

type UserProfile struct {
	UserID     string `avro:"userId" json:"userId"`
	SSN        string `avro:"ssn" json:"ssn"`
	Email      string `avro:"email" json:"email"`
	CreditCard string `avro:"creditCard" json:"creditCard"`
}

type PaymentEvent struct {
	CustomerID       string  `avro:"customerId" json:"customerId"`
	CreditCardNumber string  `avro:"creditCardNumber" json:"creditCardNumber"`
	Amount           float64 `avro:"amount" json:"amount"`
	MerchantName     string  `avro:"merchantName" json:"merchantName"`
}

// --- Global Policy Test Structs ---

type OrderPolicy struct {
	OrderID string  `avro:"orderId" json:"orderId"`
	Amount  float64 `avro:"amount" json:"amount"`
}

type OrderPolicyV2 struct {
	OrderID string  `avro:"orderId" json:"orderId"`
	Amount  float64 `avro:"amount" json:"amount"`
	Notes   *string `avro:"notes" json:"notes"`
}

type Contact struct {
	Name  string `avro:"name" json:"name"`
	Email string `avro:"email" json:"email"`
}

// Ensure imports are used (will be used by test files).
var (
	_ = schemaregistry.NewConfig
	_ = serde.ValueSerde
	_ = avrov2.NewSerializerConfig
	_ = require.NoError
)
