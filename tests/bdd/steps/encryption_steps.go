//go:build bdd

package steps

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/cucumber/godog"
)

// RegisterEncryptionSteps registers all encryption-related step definitions
// for KMS-backed server-side field-level encryption BDD tests.
func RegisterEncryptionSteps(ctx *godog.ScenarioContext, tc *TestContext) {

	// Given a shared KEK "name" with KMS type "type" and key ID "keyId"
	ctx.Step(`^a shared KEK "([^"]*)" with KMS type "([^"]*)" and key ID "([^"]*)"$`, func(name, kmsType, keyID string) error {
		kmsProps := kmsPropsForType(kmsType)

		body := map[string]interface{}{
			"name":     name,
			"kmsType":  kmsType,
			"kmsKeyId": keyID,
			"kmsProps": kmsProps,
			"shared":   true,
		}
		if err := tc.POST("/dek-registry/v1/keks", body); err != nil {
			return fmt.Errorf("create shared KEK %q: %w", name, err)
		}
		if tc.LastStatusCode != 200 {
			return fmt.Errorf("expected 200 creating shared KEK %q, got %d: %s", name, tc.LastStatusCode, string(tc.LastBody))
		}
		return nil
	})

	// When I create a DEK for subject "subject" under KEK "kekName"
	ctx.Step(`^I create a DEK for subject "([^"]*)" under KEK "([^"]*)"$`, func(subject, kekName string) error {
		body := map[string]interface{}{
			"subject":   subject,
			"algorithm": "AES256_GCM",
		}
		if err := tc.POST("/dek-registry/v1/keks/"+kekName+"/deks", body); err != nil {
			return fmt.Errorf("create DEK for subject %q under KEK %q: %w", subject, kekName, err)
		}
		return nil
	})

	// When I create a DEK for subject "subject" under KEK "kekName" with algorithm "algo"
	ctx.Step(`^I create a DEK for subject "([^"]*)" under KEK "([^"]*)" with algorithm "([^"]*)"$`, func(subject, kekName, algo string) error {
		body := map[string]interface{}{
			"subject":   subject,
			"algorithm": algo,
		}
		if err := tc.POST("/dek-registry/v1/keks/"+kekName+"/deks", body); err != nil {
			return fmt.Errorf("create DEK for subject %q under KEK %q with algorithm %q: %w", subject, kekName, algo, err)
		}
		return nil
	})

	// When I create a DEK for subject "subject" under KEK "kekName" with encrypted key material "material"
	ctx.Step(`^I create a DEK for subject "([^"]*)" under KEK "([^"]*)" with encrypted key material "([^"]*)"$`, func(subject, kekName, material string) error {
		body := map[string]interface{}{
			"subject":              subject,
			"algorithm":            "AES256_GCM",
			"encryptedKeyMaterial": material,
		}
		if err := tc.POST("/dek-registry/v1/keks/"+kekName+"/deks", body); err != nil {
			return fmt.Errorf("create DEK with key material for subject %q under KEK %q: %w", subject, kekName, err)
		}
		return nil
	})

	// Then the response field "field" should be non-empty
	ctx.Step(`^the response field "([^"]*)" should be non-empty$`, func(field string) error {
		if tc.LastJSON == nil {
			return fmt.Errorf("no JSON object in last response")
		}
		val, ok := tc.LastJSON[field]
		if !ok {
			return fmt.Errorf("field %q not found in response: %s", field, string(tc.LastBody))
		}
		if val == nil {
			return fmt.Errorf("field %q is null", field)
		}
		s, ok := val.(string)
		if ok && s == "" {
			return fmt.Errorf("field %q is an empty string", field)
		}
		return nil
	})

	// Then the response field "field" should be empty or absent
	ctx.Step(`^the response field "([^"]*)" should be empty or absent$`, func(field string) error {
		if tc.LastJSON == nil {
			// No JSON object means field is absent.
			return nil
		}
		val, ok := tc.LastJSON[field]
		if !ok {
			return nil // absent
		}
		if val == nil {
			return nil // null
		}
		s, ok := val.(string)
		if ok && s == "" {
			return nil // empty string
		}
		return fmt.Errorf("field %q is present and non-empty: %v", field, val)
	})

	// Then I can unwrap the encrypted key material using KMS type "type" and key ID "keyId"
	ctx.Step(`^I can unwrap the encrypted key material using KMS type "([^"]*)" and key ID "([^"]*)"$`, func(kmsType, keyID string) error {
		if tc.LastJSON == nil {
			return fmt.Errorf("no JSON object in last response")
		}

		encryptedKeyMaterial, ok := tc.LastJSON["encryptedKeyMaterial"]
		if !ok || encryptedKeyMaterial == nil {
			return fmt.Errorf("encryptedKeyMaterial not found in response")
		}
		ciphertext, ok := encryptedKeyMaterial.(string)
		if !ok || ciphertext == "" {
			return fmt.Errorf("encryptedKeyMaterial is not a non-empty string: %v", encryptedKeyMaterial)
		}

		keyMaterial, ok := tc.LastJSON["keyMaterial"]
		if !ok || keyMaterial == nil {
			return fmt.Errorf("keyMaterial not found in response")
		}
		expectedPlaintext, ok := keyMaterial.(string)
		if !ok || expectedPlaintext == "" {
			return fmt.Errorf("keyMaterial is not a non-empty string: %v", keyMaterial)
		}

		// The encryptedKeyMaterial is base64-encoded in the API response.
		// Decode it to get the raw ciphertext (e.g., "vault:v1:...").
		rawCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
		if err != nil {
			return fmt.Errorf("base64 decode of encryptedKeyMaterial failed: %w", err)
		}

		// Decrypt via KMS Transit endpoint
		decryptedBase64, err := transitDecrypt(kmsType, keyID, string(rawCiphertext))
		if err != nil {
			return fmt.Errorf("transit decrypt failed: %w", err)
		}

		// The Transit API returns base64-encoded plaintext
		decryptedBytes, err := base64.StdEncoding.DecodeString(decryptedBase64)
		if err != nil {
			return fmt.Errorf("base64 decode of decrypted plaintext failed: %w", err)
		}

		// The keyMaterial in the response is also base64-encoded raw key bytes
		expectedBytes, err := base64.StdEncoding.DecodeString(expectedPlaintext)
		if err != nil {
			return fmt.Errorf("base64 decode of keyMaterial failed: %w", err)
		}

		if !bytes.Equal(decryptedBytes, expectedBytes) {
			return fmt.Errorf("unwrapped key material does not match: decrypted %d bytes, expected %d bytes",
				len(decryptedBytes), len(expectedBytes))
		}

		return nil
	})
}

// kmsPropsForType returns KMS provider properties populated from environment variables.
func kmsPropsForType(kmsType string) map[string]string {
	switch kmsType {
	case "hcvault":
		return map[string]string{
			"vault.address": os.Getenv("KMS_VAULT_ADDR"),
			"vault.token":   os.Getenv("KMS_VAULT_TOKEN"),
		}
	case "openbao":
		return map[string]string{
			"openbao.address": os.Getenv("KMS_BAO_ADDR"),
			"openbao.token":   os.Getenv("KMS_BAO_TOKEN"),
		}
	default:
		return map[string]string{}
	}
}

// transitDecrypt calls the KMS Transit decrypt endpoint to unwrap ciphertext.
func transitDecrypt(kmsType, keyID, ciphertext string) (string, error) {
	var addr, token string

	switch kmsType {
	case "hcvault":
		addr = os.Getenv("KMS_VAULT_ADDR")
		token = os.Getenv("KMS_VAULT_TOKEN")
	case "openbao":
		addr = os.Getenv("KMS_BAO_ADDR")
		token = os.Getenv("KMS_BAO_TOKEN")
	default:
		return "", fmt.Errorf("unsupported KMS type for transit decrypt: %s", kmsType)
	}

	if addr == "" {
		return "", fmt.Errorf("KMS address not set for type %s", kmsType)
	}
	if token == "" {
		return "", fmt.Errorf("KMS token not set for type %s", kmsType)
	}

	// Ensure addr has no trailing slash
	addr = strings.TrimRight(addr, "/")

	// POST to Transit decrypt endpoint: /v1/transit/decrypt/{keyId}
	decryptURL := fmt.Sprintf("%s/v1/transit/decrypt/%s", addr, keyID)

	reqBody := map[string]string{
		"ciphertext": ciphertext,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal decrypt request: %w", err)
	}

	req, err := http.NewRequest("POST", decryptURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create decrypt request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vault-Token", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("transit decrypt HTTP call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read decrypt response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("transit decrypt returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response: {"data": {"plaintext": "<base64>"}}
	var result struct {
		Data struct {
			Plaintext string `json:"plaintext"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse decrypt response: %w", err)
	}

	if result.Data.Plaintext == "" {
		return "", fmt.Errorf("transit decrypt returned empty plaintext")
	}

	return result.Data.Plaintext, nil
}
