//go:build bdd

// Package bdd contains BDD tests using godog (Cucumber for Go).
package bdd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/cucumber/godog"

	"github.com/axonops/schema-registry-ui/internal/config"
	"github.com/axonops/schema-registry-ui/internal/server"
)

// testContext holds the state for a single scenario.
type testContext struct {
	uiServer     *httptest.Server
	srServer     *httptest.Server
	cookie       *http.Cookie
	otherCookie  *http.Cookie
	lastResp     *http.Response
	lastBody     []byte
	client       *http.Client
	cfg          *config.Config
	tempDir      string
	srReceivedAt map[string]http.Header // path -> headers received by mock SR
	srStopped    bool
}

// ---------- Mock Schema Registry ----------

func (tc *testContext) createMockSR() {
	mux := http.NewServeMux()

	mux.HandleFunc("/subjects", func(w http.ResponseWriter, r *http.Request) {
		tc.srReceivedAt["/subjects"] = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"test-topic", "user-events"})
	})

	mux.HandleFunc("/subjects/test-topic/versions", func(w http.ResponseWriter, r *http.Request) {
		tc.srReceivedAt["/subjects/test-topic/versions"] = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]int{1})
	})

	mux.HandleFunc("/schemas/ids/1", func(w http.ResponseWriter, r *http.Request) {
		tc.srReceivedAt["/schemas/ids/1"] = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"schema": `{"type":"string"}`})
	})

	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		tc.srReceivedAt["/config"] = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"compatibilityLevel": "BACKWARD"})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tc.srReceivedAt[r.URL.Path] = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	tc.srServer = httptest.NewServer(mux)
}

func (tc *testContext) createUI(apiToken string) error {
	if tc.srServer == nil {
		tc.createMockSR()
	}

	tc.cfg = &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 0},
		Registry: config.RegistryConfig{
			URL:      tc.srServer.URL,
			APIToken: apiToken,
		},
		Auth: config.AuthConfig{
			HtpasswdFile:  filepath.Join(tc.tempDir, "htpasswd"),
			SessionSecret: "bdd-test-secret-32-characters!!",
			SessionTTL:    3600,
			CookieName:    "sr_session",
			CookieSecure:  false,
		},
	}

	spaFS := &fstest.MapFS{
		"index.html": {Data: []byte("<html>BDD SPA</html>")},
	}

	srv, err := server.New(tc.cfg, spaFS)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	tc.uiServer = httptest.NewServer(srv.Handler())
	return nil
}

func (tc *testContext) reset() {
	tc.cleanup()
	tc.cookie = nil
	tc.otherCookie = nil
	tc.lastResp = nil
	tc.lastBody = nil
	tc.srStopped = false
	tc.srReceivedAt = make(map[string]http.Header)
	tc.uiServer = nil
	tc.srServer = nil

	dir, err := os.MkdirTemp("", "bdd-htpasswd-*")
	if err != nil {
		panic(err)
	}
	tc.tempDir = dir
}

func (tc *testContext) cleanup() {
	if tc.uiServer != nil {
		tc.uiServer.Close()
	}
	if tc.srServer != nil && !tc.srStopped {
		tc.srServer.Close()
	}
}

func (tc *testContext) saveResponse(resp *http.Response) error {
	tc.lastResp = resp
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	tc.lastBody = body
	return nil
}

func (tc *testContext) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, tc.uiServer.URL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if tc.cookie != nil {
		req.AddCookie(tc.cookie)
	}
	return tc.client.Do(req)
}

// ---------- Step Definitions ----------

func (tc *testContext) theUIServerIsRunning() error {
	if tc.uiServer != nil {
		return nil
	}
	return tc.createUI("")
}

func (tc *testContext) theMockSchemaRegistryIsRunning() error {
	if tc.srServer == nil {
		tc.createMockSR()
	}
	return nil
}

func (tc *testContext) theDefaultAdminUserExists() error {
	// Bootstrap creates admin:admin automatically
	return nil
}

func (tc *testContext) iLoginAsWithPassword(username, password string) error {
	payload, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(tc.uiServer.URL+"/api/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) theResponseStatusShouldBe(expected int) error {
	if tc.lastResp.StatusCode != expected {
		return fmt.Errorf("expected status %d, got %d (body: %s)", expected, tc.lastResp.StatusCode, string(tc.lastBody))
	}
	return nil
}

func (tc *testContext) theResponseShouldContainUsername(username string) error {
	var body map[string]interface{}
	if err := json.Unmarshal(tc.lastBody, &body); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	got, ok := body["username"]
	if !ok || got != username {
		return fmt.Errorf("expected username %q, got %v", username, got)
	}
	return nil
}

func (tc *testContext) aSessionCookieShouldBeSet() error {
	for _, c := range tc.lastResp.Cookies() {
		if c.Name == "sr_session" && c.Value != "" {
			return nil
		}
	}
	return fmt.Errorf("no session cookie found in response")
}

func (tc *testContext) theSessionCookieShouldBeCleared() error {
	for _, c := range tc.lastResp.Cookies() {
		if c.Name == "sr_session" && c.MaxAge == -1 {
			return nil
		}
	}
	return fmt.Errorf("session cookie was not cleared")
}

func (tc *testContext) iAmLoggedInAs(username string) error {
	password := username
	if username == "admin" {
		password = "admin"
	}
	payload, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(tc.uiServer.URL+"/api/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "sr_session" {
			tc.cookie = c
			return nil
		}
	}
	return fmt.Errorf("no session cookie after login")
}

func (tc *testContext) anotherUserIsLoggedInAs(username string) error {
	password := username + "123"
	payload, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(tc.uiServer.URL+"/api/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("login as %s failed with status %d", username, resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "sr_session" {
			tc.otherCookie = c
			return nil
		}
	}
	return fmt.Errorf("no session cookie after login for %s", username)
}

func (tc *testContext) iLogout() error {
	resp, err := tc.doRequest("POST", "/api/auth/logout", nil)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iCheckMySession() error {
	resp, err := tc.doRequest("GET", "/api/auth/session", nil)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iCheckMySessionWithoutACookie() error {
	tc.cookie = nil
	resp, err := tc.doRequest("GET", "/api/auth/session", nil)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) theOtherUserChecksTheirSession() error {
	req, _ := http.NewRequest("GET", tc.uiServer.URL+"/api/auth/session", nil)
	if tc.otherCookie != nil {
		req.AddCookie(tc.otherCookie)
	}
	resp, err := tc.client.Do(req)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iRequestTheAuthConfig() error {
	resp, err := http.Get(tc.uiServer.URL + "/api/auth/config")
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) theResponseShouldIndicateAuthIsEnabled() error {
	var body map[string]interface{}
	if err := json.Unmarshal(tc.lastBody, &body); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}
	if body["auth_enabled"] != true {
		return fmt.Errorf("expected auth_enabled=true, got %v", body["auth_enabled"])
	}
	return nil
}

// -- User Management Steps --

func (tc *testContext) userExistsWithPassword(username, password string) error {
	// Ensure we're authenticated (some scenarios don't have "I am logged in" before this step)
	if tc.cookie == nil {
		if err := tc.iAmLoggedInAs("admin"); err != nil {
			return fmt.Errorf("auto-login for user creation: %w", err)
		}
	}
	payload, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := tc.doRequest("POST", "/api/users", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		return fmt.Errorf("failed to create user %s: status %d", username, resp.StatusCode)
	}
	return nil
}

func (tc *testContext) userIsDisabled(username string) error {
	if tc.cookie == nil {
		if err := tc.iAmLoggedInAs("admin"); err != nil {
			return fmt.Errorf("auto-login for user disable: %w", err)
		}
	}
	body, _ := json.Marshal(map[string]interface{}{"enabled": false})
	resp, err := tc.doRequest("PUT", "/api/users/"+username, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)
	return nil
}

func (tc *testContext) iListAllUsers() error {
	resp, err := tc.doRequest("GET", "/api/users", nil)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iListAllUsersWithoutAuthentication() error {
	tc.cookie = nil
	resp, err := tc.doRequest("GET", "/api/users", nil)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) theUserListShouldContainNUsers(n int) error {
	var users []interface{}
	if err := json.Unmarshal(tc.lastBody, &users); err != nil {
		return fmt.Errorf("parsing user list: %w (body: %s)", err, string(tc.lastBody))
	}
	if len(users) != n {
		return fmt.Errorf("expected %d users, got %d", n, len(users))
	}
	return nil
}

func (tc *testContext) theUserListShouldInclude(username string) error {
	var users []map[string]interface{}
	if err := json.Unmarshal(tc.lastBody, &users); err != nil {
		return fmt.Errorf("parsing user list: %w", err)
	}
	for _, u := range users {
		if u["username"] == username {
			return nil
		}
	}
	return fmt.Errorf("user %q not found in list", username)
}

func (tc *testContext) iCreateUserWithPassword(username, password string) error {
	payload, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := tc.doRequest("POST", "/api/users", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iDisableUser(username string) error {
	body, _ := json.Marshal(map[string]interface{}{"enabled": false})
	resp, err := tc.doRequest("PUT", "/api/users/"+username, bytes.NewReader(body))
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iEnableUser(username string) error {
	body, _ := json.Marshal(map[string]interface{}{"enabled": true})
	resp, err := tc.doRequest("PUT", "/api/users/"+username, bytes.NewReader(body))
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iChangeThePasswordForTo(username, password string) error {
	body, _ := json.Marshal(map[string]interface{}{"password": &password})
	resp, err := tc.doRequest("PUT", "/api/users/"+username, bytes.NewReader(body))
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iDeleteUser(username string) error {
	resp, err := tc.doRequest("DELETE", "/api/users/"+username, nil)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iChangeMyPasswordFromTo(oldPassword, newPassword string) error {
	body, _ := json.Marshal(map[string]string{
		"current_password": oldPassword,
		"new_password":     newPassword,
	})
	resp, err := tc.doRequest("POST", "/api/users/me/password", bytes.NewReader(body))
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iAmNotAuthenticated() error {
	tc.cookie = nil
	return nil
}

// -- Proxy Steps --

func (tc *testContext) iRequestViaTheProxy(path string) error {
	resp, err := tc.doRequest("GET", path, nil)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) iRequestWithoutAuthentication(path string) error {
	tc.cookie = nil
	resp, err := tc.doRequest("GET", path, nil)
	if err != nil {
		return err
	}
	return tc.saveResponse(resp)
}

func (tc *testContext) theResponseShouldContain(text string) error {
	if !strings.Contains(string(tc.lastBody), text) {
		return fmt.Errorf("response body does not contain %q: %s", text, string(tc.lastBody))
	}
	return nil
}

func (tc *testContext) theSchemaRegistryIsStopped() error {
	if tc.srServer != nil {
		tc.srServer.Close()
		tc.srStopped = true
	}
	return nil
}

func (tc *testContext) theUIIsConfiguredWithAPIToken(token string) error {
	tc.cleanup()
	tc.srReceivedAt = make(map[string]http.Header)
	tc.srStopped = false
	tc.createMockSR()
	if err := tc.createUI(token); err != nil {
		return err
	}
	return tc.iAmLoggedInAs("admin")
}

func (tc *testContext) theSchemaRegistryShouldReceiveTheAuthorizationHeader() error {
	for _, headers := range tc.srReceivedAt {
		auth := headers.Get("Authorization")
		if auth != "" && strings.HasPrefix(auth, "Bearer ") {
			return nil
		}
	}
	return fmt.Errorf("no Authorization header received by Schema Registry")
}

func (tc *testContext) theSchemaRegistryShouldNotReceiveAnyCookies() error {
	for path, headers := range tc.srReceivedAt {
		if headers.Get("Cookie") != "" {
			return fmt.Errorf("SR received cookies at %s: %s", path, headers.Get("Cookie"))
		}
	}
	return nil
}

// ---------- Scenario Initialization ----------

func InitializeScenario(ctx *godog.ScenarioContext) {
	tc := &testContext{
		client:       &http.Client{},
		srReceivedAt: make(map[string]http.Header),
	}

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		tc.reset()
		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		tc.cleanup()
		if tc.tempDir != "" {
			os.RemoveAll(tc.tempDir)
		}
		return ctx, nil
	})

	// Background steps
	ctx.Step(`^the UI server is running$`, tc.theUIServerIsRunning)
	ctx.Step(`^the mock Schema Registry is running$`, tc.theMockSchemaRegistryIsRunning)
	ctx.Step(`^the default admin user exists$`, tc.theDefaultAdminUserExists)

	// Auth steps
	ctx.Step(`^I login as "([^"]*)" with password "([^"]*)"$`, tc.iLoginAsWithPassword)
	ctx.Step(`^I am logged in as "([^"]*)"$`, tc.iAmLoggedInAs)
	ctx.Step(`^another user is logged in as "([^"]*)"$`, tc.anotherUserIsLoggedInAs)
	ctx.Step(`^I logout$`, tc.iLogout)
	ctx.Step(`^I check my session$`, tc.iCheckMySession)
	ctx.Step(`^I check my session without a cookie$`, tc.iCheckMySessionWithoutACookie)
	ctx.Step(`^the other user checks their session$`, tc.theOtherUserChecksTheirSession)
	ctx.Step(`^I request the auth config$`, tc.iRequestTheAuthConfig)
	ctx.Step(`^I am not authenticated$`, tc.iAmNotAuthenticated)

	// Response assertions
	ctx.Step(`^the response status should be (\d+)$`, tc.theResponseStatusShouldBe)
	ctx.Step(`^the response should contain username "([^"]*)"$`, tc.theResponseShouldContainUsername)
	ctx.Step(`^a session cookie should be set$`, tc.aSessionCookieShouldBeSet)
	ctx.Step(`^the session cookie should be cleared$`, tc.theSessionCookieShouldBeCleared)
	ctx.Step(`^the response should indicate auth is enabled$`, tc.theResponseShouldIndicateAuthIsEnabled)
	ctx.Step(`^the response should contain "([^"]*)"$`, tc.theResponseShouldContain)

	// User management steps
	ctx.Step(`^user "([^"]*)" exists with password "([^"]*)"$`, tc.userExistsWithPassword)
	ctx.Step(`^user "([^"]*)" is disabled$`, tc.userIsDisabled)
	ctx.Step(`^I list all users$`, tc.iListAllUsers)
	ctx.Step(`^I list all users without authentication$`, tc.iListAllUsersWithoutAuthentication)
	ctx.Step(`^the user list should contain (\d+) users?$`, tc.theUserListShouldContainNUsers)
	ctx.Step(`^the user list should include "([^"]*)"$`, tc.theUserListShouldInclude)
	ctx.Step(`^I create user "([^"]*)" with password "([^"]*)"$`, tc.iCreateUserWithPassword)
	ctx.Step(`^I disable user "([^"]*)"$`, tc.iDisableUser)
	ctx.Step(`^I enable user "([^"]*)"$`, tc.iEnableUser)
	ctx.Step(`^I change the password for "([^"]*)" to "([^"]*)"$`, tc.iChangeThePasswordForTo)
	ctx.Step(`^I delete user "([^"]*)"$`, tc.iDeleteUser)
	ctx.Step(`^I change my password from "([^"]*)" to "([^"]*)"$`, tc.iChangeMyPasswordFromTo)

	// Proxy steps
	ctx.Step(`^I request "([^"]*)" via the proxy$`, tc.iRequestViaTheProxy)
	ctx.Step(`^I request "([^"]*)" without authentication$`, tc.iRequestWithoutAuthentication)
	ctx.Step(`^the Schema Registry is stopped$`, tc.theSchemaRegistryIsStopped)
	ctx.Step(`^the UI is configured with API token "([^"]*)"$`, tc.theUIIsConfiguredWithAPIToken)
	ctx.Step(`^the Schema Registry should receive the authorization header$`, tc.theSchemaRegistryShouldReceiveTheAuthorizationHeader)
	ctx.Step(`^the Schema Registry should not receive any cookies$`, tc.theSchemaRegistryShouldNotReceiveAnyCookies)
}

func TestBDD(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("BDD tests failed")
	}
}
