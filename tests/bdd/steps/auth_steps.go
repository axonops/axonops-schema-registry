//go:build bdd

package steps

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
	"github.com/golang-jwt/jwt/v5"
)

// jwtPrivateKey is the cached RSA private key for signing JWT tokens in BDD tests.
// Loaded once per test process from tests/bdd/certs/jwt/jwt-private.pem.
var (
	jwtPrivateKey     *rsa.PrivateKey
	jwtPrivateKeyOnce sync.Once
	jwtPrivateKeyErr  error
)

// loadJWTPrivateKey loads and caches the RSA private key from the test fixtures directory.
func loadJWTPrivateKey() (*rsa.PrivateKey, error) {
	jwtPrivateKeyOnce.Do(func() {
		// Find the key relative to this source file.
		_, filename, _, _ := runtime.Caller(0)
		stepsDir := filepath.Dir(filename)
		keyPath := filepath.Join(stepsDir, "..", "certs", "jwt", "jwt-private.pem")

		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			jwtPrivateKeyErr = fmt.Errorf("read JWT private key: %w", err)
			return
		}
		key, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
		if err != nil {
			jwtPrivateKeyErr = fmt.Errorf("parse JWT private key: %w", err)
			return
		}
		jwtPrivateKey = key
	})
	return jwtPrivateKey, jwtPrivateKeyErr
}

// RegisterAuthSteps registers authentication and admin-related step definitions.
func RegisterAuthSteps(ctx *godog.ScenarioContext, tc *TestContext) {
	// --- Auth credential steps ---
	ctx.Step(`^I authenticate as "([^"]*)" with password "([^"]*)"$`, func(username, password string) error {
		tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
		return nil
	})

	ctx.Step(`^I authenticate with API key "([^"]*)"$`, func(key string) error {
		key = tc.resolveVars(key)
		tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(key+":ignored"))
		return nil
	})

	ctx.Step(`^I authenticate with bearer token "([^"]*)"$`, func(token string) error {
		token = tc.resolveVars(token)
		tc.AuthHeader = "Bearer " + token
		return nil
	})

	ctx.Step(`^I obtain an OIDC token for "([^"]*)" with password "([^"]*)"$`, func(username, password string) error {
		tokenURL, ok := tc.StoredValues["_oidc_token_url"]
		if !ok {
			return fmt.Errorf("_oidc_token_url not set in StoredValues")
		}
		clientID, _ := tc.StoredValues["_oidc_client_id"]
		if clientID == nil {
			clientID = "schema-registry"
		}
		clientSecret, _ := tc.StoredValues["_oidc_client_secret"]
		if clientSecret == nil {
			clientSecret = "schema-registry-secret"
		}

		resp, err := http.PostForm(fmt.Sprintf("%v", tokenURL), url.Values{
			"grant_type":    {"password"},
			"client_id":     {fmt.Sprintf("%v", clientID)},
			"client_secret": {fmt.Sprintf("%v", clientSecret)},
			"username":      {username},
			"password":      {password},
			"scope":         {"openid"},
		})
		if err != nil {
			return fmt.Errorf("OIDC token request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read OIDC token response: %w", err)
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("OIDC token request returned %d: %s", resp.StatusCode, string(body))
		}

		var tokenResp map[string]interface{}
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return fmt.Errorf("parse OIDC token response: %w", err)
		}

		// Use id_token for OIDC authentication (contains correct aud claim).
		// Fall back to access_token if id_token is not present.
		token := ""
		if idToken, ok := tokenResp["id_token"].(string); ok && idToken != "" {
			token = idToken
		} else if accessToken, ok := tokenResp["access_token"].(string); ok && accessToken != "" {
			token = accessToken
		}
		if token == "" {
			return fmt.Errorf("no id_token or access_token in OIDC response: %s", string(body))
		}

		tc.StoredValues["_oidc_token"] = token
		tc.AuthHeader = "Bearer " + token
		return nil
	})

	// --- JWT token generation steps ---
	ctx.Step(`^I generate a JWT token with claims:$`, func(table *godog.Table) error {
		privateKey, err := loadJWTPrivateKey()
		if err != nil {
			return fmt.Errorf("load JWT private key: %w", err)
		}
		claims := jwt.MapClaims{
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(5 * time.Minute).Unix(),
		}
		for _, row := range table.Rows {
			if len(row.Cells) != 2 {
				continue
			}
			key := row.Cells[0].Value
			val := row.Cells[1].Value
			claims[key] = val
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signed, err := token.SignedString(privateKey)
		if err != nil {
			return fmt.Errorf("sign JWT: %w", err)
		}
		tc.StoredValues["_jwt_token"] = signed
		tc.AuthHeader = "Bearer " + signed
		return nil
	})

	ctx.Step(`^I generate an expired JWT token with claims:$`, func(table *godog.Table) error {
		privateKey, err := loadJWTPrivateKey()
		if err != nil {
			return fmt.Errorf("load JWT private key: %w", err)
		}
		claims := jwt.MapClaims{
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
			"exp": time.Now().Add(-1 * time.Hour).Unix(),
		}
		for _, row := range table.Rows {
			if len(row.Cells) != 2 {
				continue
			}
			key := row.Cells[0].Value
			val := row.Cells[1].Value
			claims[key] = val
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signed, err := token.SignedString(privateKey)
		if err != nil {
			return fmt.Errorf("sign expired JWT: %w", err)
		}
		tc.StoredValues["_jwt_token"] = signed
		tc.AuthHeader = "Bearer " + signed
		return nil
	})

	ctx.Step(`^I generate a JWT token signed with wrong key with claims:$`, func(table *godog.Table) error {
		// Generate an ephemeral RSA key that doesn't match the registry's public key.
		wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("generate wrong RSA key: %w", err)
		}
		claims := jwt.MapClaims{
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(5 * time.Minute).Unix(),
		}
		for _, row := range table.Rows {
			if len(row.Cells) != 2 {
				continue
			}
			key := row.Cells[0].Value
			val := row.Cells[1].Value
			claims[key] = val
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signed, err := token.SignedString(wrongKey)
		if err != nil {
			return fmt.Errorf("sign JWT with wrong key: %w", err)
		}
		tc.StoredValues["_jwt_token"] = signed
		tc.AuthHeader = "Bearer " + signed
		return nil
	})

	ctx.Step(`^I clear authentication$`, func() error {
		tc.AuthHeader = ""
		return nil
	})

	ctx.Step(`^I authenticate with stored API key "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		key := fmt.Sprintf("%v", val)
		tc.AuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(key+":ignored"))
		return nil
	})

	// --- Admin user management steps ---
	ctx.Step(`^I create a user with username "([^"]*)" password "([^"]*)" role "([^"]*)"$`, func(username, password, role string) error {
		body := map[string]interface{}{
			"username": username,
			"password": password,
			"role":     role,
		}
		return tc.POST("/admin/users", body)
	})

	ctx.Step(`^I create a user with username "([^"]*)" password "([^"]*)" role "([^"]*)" email "([^"]*)"$`, func(username, password, role, email string) error {
		body := map[string]interface{}{
			"username": username,
			"password": password,
			"role":     role,
			"email":    email,
		}
		return tc.POST("/admin/users", body)
	})

	ctx.Step(`^I list all users$`, func() error {
		return tc.GET("/admin/users")
	})

	ctx.Step(`^I get user by ID "([^"]*)"$`, func(idVar string) error {
		resolved := tc.resolveVars(idVar)
		return tc.GET("/admin/users/" + resolved)
	})

	ctx.Step(`^I get user by stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.GET(fmt.Sprintf("/admin/users/%v", val))
	})

	ctx.Step(`^I update user "([^"]*)" with:$`, func(idVar string, body *godog.DocString) error {
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		resolved := tc.resolveVars(idVar)
		return tc.PUT("/admin/users/"+resolved, req)
	})

	ctx.Step(`^I update user with stored ID "([^"]*)" with:$`, func(varName string, body *godog.DocString) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		return tc.PUT(fmt.Sprintf("/admin/users/%v", val), req)
	})

	ctx.Step(`^I delete user with stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.DELETE(fmt.Sprintf("/admin/users/%v", val))
	})

	// --- Admin API key management steps ---
	ctx.Step(`^I create an API key with name "([^"]*)" role "([^"]*)" expires_in (\d+)$`, func(name, role string, expiresIn int) error {
		body := map[string]interface{}{
			"name":       name,
			"role":       role,
			"expires_in": expiresIn,
		}
		return tc.POST("/admin/apikeys", body)
	})

	ctx.Step(`^I create an API key with name "([^"]*)" role "([^"]*)" expires_in (\d+) for_user_id (\d+)$`, func(name, role string, expiresIn, userID int) error {
		body := map[string]interface{}{
			"name":        name,
			"role":        role,
			"expires_in":  expiresIn,
			"for_user_id": userID,
		}
		return tc.POST("/admin/apikeys", body)
	})

	ctx.Step(`^I list all API keys$`, func() error {
		return tc.GET("/admin/apikeys")
	})

	ctx.Step(`^I get API key by stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.GET(fmt.Sprintf("/admin/apikeys/%v", val))
	})

	ctx.Step(`^I update API key with stored ID "([^"]*)" with:$`, func(varName string, body *godog.DocString) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		var req interface{}
		if err := json.Unmarshal([]byte(body.Content), &req); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		return tc.PUT(fmt.Sprintf("/admin/apikeys/%v", val), req)
	})

	ctx.Step(`^I delete API key with stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.DELETE(fmt.Sprintf("/admin/apikeys/%v", val))
	})

	ctx.Step(`^I revoke API key with stored ID "([^"]*)"$`, func(varName string) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		return tc.POST(fmt.Sprintf("/admin/apikeys/%v/revoke", val), nil)
	})

	ctx.Step(`^I rotate API key with stored ID "([^"]*)" expires_in (\d+)$`, func(varName string, expiresIn int) error {
		val, ok := tc.StoredValues[varName]
		if !ok {
			return fmt.Errorf("no stored value %q", varName)
		}
		body := map[string]interface{}{
			"expires_in": expiresIn,
		}
		return tc.POST(fmt.Sprintf("/admin/apikeys/%v/rotate", val), body)
	})

	ctx.Step(`^I list roles$`, func() error {
		return tc.GET("/admin/roles")
	})

	// --- Metrics steps ---
	ctx.Step(`^I get the metrics$`, func() error {
		return tc.GET("/metrics")
	})

	ctx.Step(`^the response should contain Prometheus metric "([^"]*)"$`, func(metricName string) error {
		body := string(tc.LastBody)
		if !strings.Contains(body, metricName) {
			return fmt.Errorf("metrics response does not contain %q (first 500 chars: %s)", metricName, truncate(body, 500))
		}
		return nil
	})

	// --- Response assertions for nested fields ---
	ctx.Step(`^the response users array should have length (\d+)$`, func(expected int) error {
		if tc.LastJSON == nil {
			return fmt.Errorf("no JSON object in last response")
		}
		users, ok := tc.LastJSON["users"]
		if !ok {
			return fmt.Errorf("no 'users' field in response: %s", string(tc.LastBody))
		}
		arr, ok := users.([]interface{})
		if !ok {
			return fmt.Errorf("'users' field is not an array: %T", users)
		}
		if len(arr) != expected {
			return fmt.Errorf("expected users array length %d, got %d", expected, len(arr))
		}
		return nil
	})

	ctx.Step(`^the response apikeys array should have length (\d+)$`, func(expected int) error {
		if tc.LastJSON == nil {
			return fmt.Errorf("no JSON object in last response")
		}
		keys, ok := tc.LastJSON["api_keys"]
		if !ok {
			return fmt.Errorf("no 'api_keys' field in response: %s", string(tc.LastBody))
		}
		arr, ok := keys.([]interface{})
		if !ok {
			return fmt.Errorf("'api_keys' field is not an array: %T", keys)
		}
		if len(arr) != expected {
			return fmt.Errorf("expected api_keys array length %d, got %d", expected, len(arr))
		}
		return nil
	})

	ctx.Step(`^the response roles array should have length (\d+)$`, func(expected int) error {
		if tc.LastJSON == nil {
			return fmt.Errorf("no JSON object in last response")
		}
		roles, ok := tc.LastJSON["roles"]
		if !ok {
			return fmt.Errorf("no 'roles' field in response: %s", string(tc.LastBody))
		}
		arr, ok := roles.([]interface{})
		if !ok {
			return fmt.Errorf("'roles' field is not an array: %T", roles)
		}
		if len(arr) != expected {
			return fmt.Errorf("expected roles array length %d, got %d", expected, len(arr))
		}
		return nil
	})

	ctx.Step(`^the response field "([^"]*)" should not be empty$`, func(field string) error {
		val, err := tc.JSONFieldString(field)
		if err != nil {
			return err
		}
		if val == "" {
			return fmt.Errorf("field %q is empty", field)
		}
		return nil
	})

	// --- mTLS client certificate steps ---
	ctx.Step(`^I connect with mTLS certificate "([^"]*)"$`, func(certName string) error {
		_, filename, _, _ := runtime.Caller(0)
		stepsDir := filepath.Dir(filename)
		certsDir := filepath.Join(stepsDir, "..", "certs", "mtls")
		certFile := filepath.Join(certsDir, certName+".pem")
		keyFile := filepath.Join(certsDir, certName+"-key.pem")
		caFile := filepath.Join(certsDir, "ca.pem")
		return tc.SetMTLSClient(certFile, keyFile, caFile)
	})

	ctx.Step(`^I connect without a client certificate$`, func() error {
		tc.SetTLSOnlyClient()
		return nil
	})

	ctx.Step(`^the connection should be refused$`, func() error {
		if LastError == nil {
			return fmt.Errorf("expected connection error but request succeeded with status %d", tc.LastStatusCode)
		}
		return nil
	})

	ctx.Step(`^I attempt a GET request to "([^"]*)"$`, func(path string) error {
		return tc.DoRequestAllowError("GET", path, nil)
	})

	ctx.Step(`^I attempt a POST request to "([^"]*)" with body:$`, func(path string, body *godog.DocString) error {
		var parsed interface{}
		if err := json.Unmarshal([]byte(body.Content), &parsed); err != nil {
			return fmt.Errorf("parse body: %w", err)
		}
		return tc.DoRequestAllowError("POST", path, parsed)
	})

	ctx.Step(`^I attempt a DELETE request to "([^"]*)"$`, func(path string) error {
		return tc.DoRequestAllowError("DELETE", path, nil)
	})

	ctx.Step(`^I attempt a PUT request to "([^"]*)" with body:$`, func(path string, body *godog.DocString) error {
		var parsed interface{}
		if err := json.Unmarshal([]byte(body.Content), &parsed); err != nil {
			return fmt.Errorf("parse body: %w", err)
		}
		return tc.DoRequestAllowError("PUT", path, parsed)
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
