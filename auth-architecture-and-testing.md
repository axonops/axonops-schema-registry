# AxonOps Schema Registry — Web UI Auth Architecture & E2E Testing Strategy

> **Addendum to:** Web UI Product Requirements v2
>
> **Purpose:** Explain exactly how the React SPA authenticates against the multi-method Go backend, and how to BDD-test every auth configuration in CI using Docker Compose.

---

## Part 1: How the UI Authentication Works

### The Problem

The Go backend already supports 6 auth methods: Basic Auth (local DB), LDAP, OIDC, JWT, API Keys, and mTLS. These work perfectly for `curl` and programmatic clients. But a React SPA running in a browser has different constraints:

- You can't hold raw passwords in JavaScript memory for 30 minutes
- OIDC and SAML require browser redirects to external identity providers — that's incompatible with a `fetch()` call
- mTLS happens at the TLS layer before the application even sees the request
- The SPA needs a single, uniform way to authenticate API calls regardless of how the user originally proved their identity

### The Solution: Session Token Bridge

The Go backend gets a small set of **new endpoints** (under `/ui/auth/`) that act as a bridge. They accept credentials via whatever method the user authenticates with, validate them using the **existing auth logic**, and issue a short-lived **signed JWT session token**. From that point on, the React SPA and the existing API middleware speak the same language — `Authorization: Bearer <token>`.

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Go Binary (port 8081)                           │
│                                                                     │
│  ┌──────────────────────┐     ┌──────────────────────────────────┐  │
│  │  UI Auth Endpoints    │     │  Existing REST API               │  │
│  │  (new, ~200 lines Go) │     │  (completely unchanged)          │  │
│  │                       │     │                                  │  │
│  │  POST /ui/auth/login  │     │  GET  /subjects                  │  │
│  │  POST /ui/auth/apikey │     │  POST /subjects/{s}/versions     │  │
│  │  GET  /ui/auth/session│     │  DELETE /subjects/{s}            │  │
│  │  POST /ui/auth/logout │     │  GET  /admin/users               │  │
│  │  GET  /ui/auth/config │     │  POST /admin/apikeys             │  │
│  │                       │     │  PUT  /config                    │  │
│  │  Phase 5 additions:   │     │  ...all existing endpoints...    │  │
│  │  GET /ui/auth/oidc/   │     │                                  │  │
│  │      login            │     │  ┌────────────────────────────┐  │  │
│  │  GET /ui/auth/oidc/   │     │  │  Auth Middleware           │  │  │
│  │      callback         │     │  │                            │  │  │
│  │                       │     │  │  Checks (in order):        │  │  │
│  └──────────┬────────────┘     │  │  1. Bearer token (JWT)  ◄──┼──┼── NEW: also accepts
│             │                  │  │  2. X-API-Key header       │  │   UI session tokens
│             │ validates        │  │  3. Basic Auth header      │  │
│             │ credentials      │  │  4. mTLS client cert       │  │
│             │ using SAME       │  │                            │  │  │
│             │ auth logic       │  │  Existing clients using    │  │  │
│             │                  │  │  curl -u user:pass or      │  │  │
│             ▼                  │  │  -H "X-API-Key: ..." keep  │  │  │
│       ┌─────────────┐         │  │  working exactly as before │  │  │
│       │ Issue signed │         │  └────────────────────────────┘  │  │
│       │ JWT session  │         │                                  │  │
│       │ token        │─────────┘                                  │  │
│       └─────────────┘                                             │  │
└─────────────────────────────────────────────────────────────────────┘
         ▲                                    ▲
         │ (1) Login once                     │ (2) All subsequent requests
         │                                    │     with Bearer token
    ┌────┴────────────────────────────────────┴────┐
    │              Browser (React SPA)              │
    │                                               │
    │  On login:                                    │
    │    POST /ui/auth/login                        │
    │    Body: { "username": "alice", "password": "..." }
    │    Response: { "token": "eyJ...", "user": {...} }
    │                                               │
    │  Token stored in React state (memory only)    │
    │                                               │
    │  Every API call after login:                  │
    │    GET /subjects                              │
    │    Header: Authorization: Bearer eyJ...       │
    │                                               │
    │    POST /subjects/orders-value/versions       │
    │    Header: Authorization: Bearer eyJ...       │
    └───────────────────────────────────────────────┘
```

### Flow: Local Database User

```
Browser                              Go Backend
  │                                     │
  │  POST /ui/auth/login                │
  │  {"username":"alice",               │
  │   "password":"s3cret"}              │
  │  ──────────────────────────────────►│
  │                                     │
  │                           ┌─────────┴──────────┐
  │                           │ Look up "alice" in  │
  │                           │ local users table   │
  │                           │ bcrypt-compare      │
  │                           │ password hash       │
  │                           │ → match!            │
  │                           │ role = "developer"  │
  │                           └─────────┬──────────┘
  │                                     │
  │                           ┌─────────┴──────────┐
  │                           │ Sign JWT:           │
  │                           │ {                   │
  │                           │   sub: "alice",     │
  │                           │   role: "developer",│
  │                           │   auth_method:      │
  │                           │     "local",        │
  │                           │   exp: <now+30min>, │
  │                           │   jti: "uuid-1234"  │
  │                           │ }                   │
  │                           └─────────┬──────────┘
  │                                     │
  │  ◄─────────────────────────────────-│
  │  200 OK                             │
  │  {                                  │
  │    "token": "eyJhbGciOi...",        │
  │    "expires_at": "2026-02-26T...",  │
  │    "user": {                        │
  │      "username": "alice",           │
  │      "email": "alice@example.com",  │
  │      "role": "developer"            │
  │    }                                │
  │  }                                  │
  │                                     │
  │  ═══════════ login complete ════════│
  │                                     │
  │  GET /subjects                      │
  │  Authorization: Bearer eyJhbGciOi..│
  │  ──────────────────────────────────►│
  │                                     │
  │                           ┌─────────┴──────────┐
  │                           │ Auth middleware:    │
  │                           │ Verify JWT sig     │
  │                           │ Check exp          │
  │                           │ Extract role       │
  │                           │ → developer: allow │
  │                           └─────────┬──────────┘
  │                                     │
  │  ◄────────────────────────────────-─│
  │  200 OK                             │
  │  ["orders-value", "payments-value"] │
```

### Flow: LDAP User

```
Browser                              Go Backend                    LDAP Server
  │                                     │                              │
  │  POST /ui/auth/login                │                              │
  │  {"username":"bob",                 │                              │
  │   "password":"ldap-pass"}           │                              │
  │  ──────────────────────────────────►│                              │
  │                                     │                              │
  │                           ┌─────────┴──────────┐                   │
  │                           │ Look up "bob" in   │                   │
  │                           │ local users table  │                   │
  │                           │ → NOT FOUND        │                   │
  │                           │                    │                   │
  │                           │ LDAP enabled?      │                   │
  │                           │ → YES              │                   │
  │                           └─────────┬──────────┘                   │
  │                                     │                              │
  │                                     │  LDAP BIND                   │
  │                                     │  dn: cn=service,dc=corp,dc=io│
  │                                     │  ────────────────────────────►│
  │                                     │  ◄──── bind success          │
  │                                     │                              │
  │                                     │  LDAP SEARCH                 │
  │                                     │  base: ou=Users,dc=corp,dc=io│
  │                                     │  filter: (sAMAccountName=bob)│
  │                                     │  ────────────────────────────►│
  │                                     │  ◄──── found: cn=bob,...     │
  │                                     │                              │
  │                                     │  LDAP BIND (user)            │
  │                                     │  dn: cn=bob,ou=Users,...     │
  │                                     │  password: ldap-pass         │
  │                                     │  ────────────────────────────►│
  │                                     │  ◄──── bind success          │
  │                                     │                              │
  │                                     │  LDAP SEARCH (groups)        │
  │                                     │  filter: (member=cn=bob,...) │
  │                                     │  ────────────────────────────►│
  │                                     │  ◄──── groups:               │
  │                                     │    cn=Developers,ou=Groups   │
  │                                     │                              │
  │                           ┌─────────┴──────────┐                   │
  │                           │ Role mapping:      │                   │
  │                           │ "cn=Developers,..."│                   │
  │                           │   → "developer"    │                   │
  │                           │                    │                   │
  │                           │ Sign JWT:          │                   │
  │                           │ {                  │                   │
  │                           │  sub: "bob",       │                   │
  │                           │  role: "developer",│                   │
  │                           │  auth_method:"ldap"│                   │
  │                           │ }                  │                   │
  │                           └─────────┬──────────┘                   │
  │                                     │                              │
  │  ◄─────────────────────────────────-│                              │
  │  200 OK { "token": "eyJ..." }       │                              │
  │                                     │                              │
  │  ═══ from here, identical to local DB ═══                          │
  │  All API calls use Bearer token.                                   │
  │  The React SPA has no idea it was LDAP.                            │
```

### Flow: API Key Login

```
Browser                              Go Backend
  │                                     │
  │  POST /ui/auth/apikey               │
  │  {"key":"sr_live_abc123..."}        │
  │  ──────────────────────────────────►│
  │                                     │
  │                           ┌─────────┴──────────┐
  │                           │ SHA-256 hash the   │
  │                           │ provided key       │
  │                           │ Look up hash in    │
  │                           │ api_keys table     │
  │                           │ → found!           │
  │                           │ name: "ci-pipeline"│
  │                           │ role: "developer"  │
  │                           │ expired? NO        │
  │                           │ revoked? NO        │
  │                           │                    │
  │                           │ Sign JWT:          │
  │                           │ {                  │
  │                           │  sub:"ci-pipeline",│
  │                           │  role: "developer",│
  │                           │  auth_method:      │
  │                           │    "api_key"       │
  │                           │ }                  │
  │                           └─────────┬──────────┘
  │                                     │
  │  ◄─────────────────────────────────-│
  │  200 OK { "token": "eyJ..." }       │
```

### Flow: OIDC (Phase 5 — Keycloak / Okta / Azure AD)

```
Browser                    Go Backend                         IdP (Keycloak)
  │                           │                                    │
  │ GET /ui/auth/config       │                                    │
  │ ─────────────────────────►│                                    │
  │ ◄── {                     │                                    │
  │   methods:["basic","oidc"]│                                    │
  │   oidc: {                 │                                    │
  │     display_name:         │                                    │
  │       "Sign in with SSO", │                                    │
  │     login_url:            │                                    │
  │       "/ui/auth/oidc/     │                                    │
  │        login"             │                                    │
  │   }                       │                                    │
  │ }                         │                                    │
  │                           │                                    │
  │ (user clicks SSO button)  │                                    │
  │                           │                                    │
  │ navigate to               │                                    │
  │ /ui/auth/oidc/login       │                                    │
  │ ─────────────────────────►│                                    │
  │                           │── build authorize URL              │
  │ ◄── 302 Redirect          │                                    │
  │   Location: https://      │                                    │
  │   keycloak.corp.io/       │                                    │
  │   realms/sr/protocol/     │                                    │
  │   openid-connect/auth     │                                    │
  │   ?client_id=schema-reg   │                                    │
  │   &redirect_uri=          │                                    │
  │    /ui/auth/oidc/callback │                                    │
  │   &response_type=code     │                                    │
  │   &state=random123        │                                    │
  │                           │                                    │
  │ (browser goes to Keycloak)│                                    │
  │ ───────────────────────────────────────────────────────────────►│
  │                           │                         user types │
  │                           │                         credentials│
  │                           │                         at Keycloak│
  │ ◄──────────────────────────────── 302 Redirect                 │
  │                           │    /ui/auth/oidc/callback           │
  │                           │    ?code=authz-code-xyz             │
  │                           │    &state=random123                 │
  │                           │                                    │
  │ GET /ui/auth/oidc/        │                                    │
  │     callback?code=xyz     │                                    │
  │ ─────────────────────────►│                                    │
  │                           │── exchange code for tokens ────────►│
  │                           │◄── {                               │
  │                           │     id_token: "eyJ...",            │
  │                           │     access_token: "eyJ..."         │
  │                           │   }                                │
  │                           │                                    │
  │                           │── decode id_token claims:          │
  │                           │   preferred_username: "carol"      │
  │                           │   groups: ["/schema-reg-devs"]     │
  │                           │                                    │
  │                           │── role mapping config:             │
  │                           │   "/schema-reg-devs" → "developer" │
  │                           │                                    │
  │                           │── sign UI session JWT:             │
  │                           │   {sub:"carol",                    │
  │                           │    role:"developer",               │
  │                           │    auth_method:"oidc"}             │
  │                           │                                    │
  │ ◄── 302 Redirect          │                                    │
  │   /ui/#token=eyJ...       │  (token in fragment, not query     │
  │                           │   string — avoids server logs)     │
  │                           │                                    │
  │ (React app reads token    │                                    │
  │  from URL fragment,       │                                    │
  │  stores in state,         │                                    │
  │  strips from URL)         │                                    │
  │                           │                                    │
  │ GET /subjects             │                                    │
  │ Authorization: Bearer eyJ.│                                    │
  │ ─────────────────────────►│                                    │
  │                           │── validate JWT (same as all others)│
```

### Key Points

1. **The React SPA never knows HOW the user authenticated.** It either collects username/password and POSTs to `/ui/auth/login`, or it navigates to `/ui/auth/oidc/login` for SSO. Either way, it ends up with a JWT in React state. All subsequent API calls are identical.

2. **The existing API is completely unchanged.** Clients using `curl -u user:pass`, `-H "X-API-Key: ..."`, or OIDC bearer tokens continue working. The only change is that the auth middleware now also accepts JWTs signed by the registry's own session secret.

3. **No cookies, no localStorage.** The token lives in React state (JavaScript memory). Closing the tab or refreshing the page clears it. This is a deliberate security choice.

4. **The `/ui/auth/login` endpoint reuses the existing auth validation logic.** It's not reimplementing authentication — it's calling the same `ValidateBasicAuth()` function (and the same LDAP bind logic) that the API middleware uses, then wrapping the result in a JWT.

---

## Part 2: BDD Testing Strategy for Authentication

### The Challenge

Authentication testing requires real infrastructure:
- **LDAP** needs a real OpenLDAP server with pre-seeded users and groups
- **OIDC** needs a real identity provider (Keycloak) with a pre-configured realm
- **mTLS** needs generated certificates
- **API Keys** and **local users** need to be seeded in the registry itself

Each configuration is a different docker-compose setup, and the BDD tests need to verify that the UI login flow works correctly in each one. Additionally, the test must verify that role mapping works end-to-end: an LDAP user in the "Developers" group should see the developer UI, an OIDC user with the "/admins" claim should see the admin UI, etc.

### Test Profiles

We define **test profiles** — each is a self-contained docker-compose stack with a specific auth configuration. Tests are tagged by profile so CI can run the right tests against the right stack.

```
tests/e2e/
├── profiles/
│   ├── basic/                          # Profile: local DB auth only
│   │   ├── docker-compose.yml
│   │   └── config.yaml                 # registry config: auth.methods = [basic, api_key]
│   │
│   ├── ldap/                           # Profile: local DB + LDAP
│   │   ├── docker-compose.yml          # includes OpenLDAP container
│   │   ├── config.yaml                 # registry config: auth.methods = [basic, api_key] + ldap.enabled = true
│   │   └── seed/
│   │       └── bootstrap.ldif          # pre-seeded LDAP users and groups
│   │
│   ├── oidc/                           # Profile: local DB + OIDC (Phase 5)
│   │   ├── docker-compose.yml          # includes Keycloak container
│   │   ├── config.yaml                 # registry config: auth.methods = [basic, api_key, oidc]
│   │   └── seed/
│   │       └── realm-export.json       # Keycloak realm with pre-configured users/clients
│   │
│   └── noauth/                         # Profile: auth disabled (anonymous access)
│       ├── docker-compose.yml
│       └── config.yaml                 # registry config: auth.enabled = false
│
├── features/
│   └── auth/
│       ├── login-basic.feature         # @profile:basic @profile:ldap
│       ├── login-ldap.feature          # @profile:ldap
│       ├── login-apikey.feature        # @profile:basic @profile:ldap
│       ├── login-oidc.feature          # @profile:oidc
│       ├── session-management.feature  # @profile:basic (tests apply to all, run against basic)
│       ├── role-based-access.feature   # @profile:basic @profile:ldap
│       └── noauth-bypass.feature       # @profile:noauth
```

### Profile: `basic` — Local DB Auth Only

This is the simplest profile and the one most tests run against.

**docker-compose.yml:**
```yaml
services:
  schema-registry:
    build:
      context: ../../..
      dockerfile: Dockerfile
    ports:
      - "8081:8081"
    environment:
      STORAGE_TYPE: memory
      ADMIN_PASSWORD: test-admin-password
      UI_ENABLED: "true"
      UI_SESSION_SECRET: test-session-secret-at-least-32-chars
    volumes:
      - ./config.yaml:/etc/axonops-schema-registry/config.yaml
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/"]
      interval: 2s
      timeout: 5s
      retries: 15
```

**config.yaml:**
```yaml
server:
  host: "0.0.0.0"
  port: 8081

storage:
  type: memory

ui:
  enabled: true
  session:
    secret: "test-session-secret-at-least-32-chars-long"
    token_ttl_minutes: 30

security:
  auth:
    enabled: true
    methods:
      - basic
      - api_key
    bootstrap:
      enabled: true
      username: admin
      password: test-admin-password
      email: admin@test.local
    api_key:
      header: "X-API-Key"
      key_prefix: "sr_"
    rbac:
      enabled: true
      default_role: readonly
      super_admins:
        - admin
```

**Test data seeding (via API in Before hooks):**
```typescript
// support/api-helpers.ts
export async function seedBasicTestUsers(baseUrl: string) {
  const adminAuth = basicAuthHeader('admin', 'test-admin-password');

  // Create users for each role
  await createUser(baseUrl, adminAuth, {
    username: 'test-developer',
    password: 'dev-password-123',
    email: 'dev@test.local',
    role: 'developer',
  });

  await createUser(baseUrl, adminAuth, {
    username: 'test-readonly',
    password: 'readonly-password-123',
    email: 'readonly@test.local',
    role: 'readonly',
  });

  await createUser(baseUrl, adminAuth, {
    username: 'test-admin',
    password: 'admin-password-123',
    email: 'admin2@test.local',
    role: 'admin',
  });

  // Create an API key for API key login tests
  const apiKeyResponse = await createApiKey(baseUrl, adminAuth, {
    name: 'test-dev-key',
    role: 'developer',
    expires_in: 86400, // 24 hours
  });

  return { apiKey: apiKeyResponse.key };
}
```

**BDD feature: login-basic.feature:**
```gherkin
@profile:basic @profile:ldap
Feature: Sign in with username and password

  Background:
    Given the schema registry is running with auth enabled
    And the following local users exist:
      | username       | role      |
      | admin          | super_admin |
      | test-developer | developer   |
      | test-readonly  | readonly    |
      | test-admin     | admin       |

  Scenario: Successful login with valid credentials
    Given I am on the login page
    Then the login form should show username and password fields
    When I enter username "test-developer"
    And I enter password "dev-password-123"
    And I click "Sign In"
    Then I should be redirected to the subjects page
    And the top bar should display username "test-developer"
    And the sidebar should show "Subjects" link
    And the sidebar should show "Schema Browser" link
    And the sidebar should NOT show "Users" link
    And the sidebar should NOT show "Compatibility" link

  Scenario: Login failure with wrong password
    Given I am on the login page
    When I enter username "test-developer"
    And I enter password "wrong-password"
    And I click "Sign In"
    Then I should see an error message "Invalid username or password"
    And I should remain on the login page
    And the password field should be cleared
    And the username field should retain "test-developer"

  Scenario: Login failure with non-existent user
    Given I am on the login page
    When I enter username "nobody"
    And I enter password "anything"
    And I click "Sign In"
    Then I should see an error message "Invalid username or password"

  Scenario: Admin user sees full navigation
    Given I am on the login page
    When I sign in as "admin" with password "test-admin-password"
    Then the sidebar should show "Users" link
    And the sidebar should show "API Keys" link
    And the sidebar should show "Compatibility" link
    And the sidebar should show "Modes" link
    And the sidebar should show "Import" link

  Scenario: Readonly user cannot see write actions
    Given I sign in as "test-readonly" with password "readonly-password-123"
    When I navigate to the subjects page
    Then I should NOT see the "Register New Schema" button
    When I navigate to subject "test-subject" detail page
    Then I should NOT see the "Register New Version" button
    And I should NOT see the "Delete Subject" button

  Scenario: Login form submits on Enter key
    Given I am on the login page
    When I enter username "test-developer"
    And I enter password "dev-password-123"
    And I press Enter
    Then I should be redirected to the subjects page

  Scenario: Login redirects to originally requested URL
    Given I am not authenticated
    When I navigate directly to "/ui/subjects/orders-value"
    Then I should be redirected to the login page
    When I sign in as "test-developer" with password "dev-password-123"
    Then I should be redirected to "/ui/subjects/orders-value"
```

**BDD feature: login-apikey.feature:**
```gherkin
@profile:basic @profile:ldap
Feature: Sign in with API key

  Background:
    Given the schema registry is running with auth enabled
    And an API key "sr_test_devkey123" exists with role "developer"

  Scenario: Switch to API key login mode
    Given I am on the login page
    When I click "Use API Key instead"
    Then the username field should be hidden
    And the password field should be hidden
    And an API key input field should be visible

  Scenario: Successful API key login
    Given I am on the login page
    When I click "Use API Key instead"
    And I enter API key "sr_test_devkey123"
    And I click "Sign In"
    Then I should be redirected to the subjects page
    And I should be authenticated with role "developer"

  Scenario: Expired API key login
    Given an expired API key "sr_test_expired" exists
    And I am on the login page
    When I click "Use API Key instead"
    And I enter API key "sr_test_expired"
    And I click "Sign In"
    Then I should see an error message containing "expired"

  Scenario: Revoked API key login
    Given a revoked API key "sr_test_revoked" exists
    And I am on the login page
    When I click "Use API Key instead"
    And I enter API key "sr_test_revoked"
    And I click "Sign In"
    Then I should see an error message "Invalid API key"

  Scenario: Switch back to username/password mode
    Given I am on the login page
    When I click "Use API Key instead"
    Then I should see the API key input
    When I click "Use username and password"
    Then the username field should be visible
    And the password field should be visible
    And the API key input should be hidden
```

**BDD feature: session-management.feature:**
```gherkin
@profile:basic
Feature: Session management

  Scenario: Session persists across page navigations
    Given I sign in as "test-developer"
    When I navigate to the subjects page
    And I navigate to the schema browser page
    And I navigate back to the subjects page
    Then I should still be authenticated as "test-developer"

  Scenario: Signing out clears the session
    Given I sign in as "test-developer"
    When I click the user menu
    And I click "Sign Out"
    Then I should be redirected to the login page
    When I navigate directly to "/ui/subjects"
    Then I should be redirected to the login page

  Scenario: Refreshing the page clears the session
    Given I sign in as "test-developer"
    And I am on the subjects page
    When I refresh the browser page
    Then I should be redirected to the login page

  Scenario: Expired token redirects to login
    Given I sign in as "test-developer"
    And I am on the subjects page
    And I wait for the session token to expire
    When I click on a subject name
    Then I should be redirected to the login page
    And I should see a message "Your session has expired"

  Scenario: Direct URL access without auth redirects to login
    Given I am not authenticated
    When I navigate directly to "/ui/admin/users"
    Then I should be redirected to the login page
```

---

### Profile: `ldap` — Local DB + LDAP

This profile spins up an OpenLDAP server alongside the registry to test LDAP authentication, group-based role mapping, and fallback behaviour.

**docker-compose.yml:**
```yaml
services:
  openldap:
    image: osixia/openldap:1.5.0
    command: --copy-service
    environment:
      LDAP_ORGANISATION: "AxonOps Test"
      LDAP_DOMAIN: "test.axonops.io"
      LDAP_BASE_DN: "dc=test,dc=axonops,dc=io"
      LDAP_ADMIN_PASSWORD: "ldap-admin-password"
      LDAP_TLS: "false"
    volumes:
      - ./seed/bootstrap.ldif:/container/service/slapd/assets/config/bootstrap/ldif/custom/50-bootstrap.ldif
    ports:
      - "389:389"
    healthcheck:
      test: ["CMD", "ldapsearch", "-x", "-H", "ldap://localhost", "-b", "dc=test,dc=axonops,dc=io", "-D", "cn=admin,dc=test,dc=axonops,dc=io", "-w", "ldap-admin-password"]
      interval: 3s
      timeout: 5s
      retries: 10

  schema-registry:
    build:
      context: ../../..
      dockerfile: Dockerfile
    ports:
      - "8081:8081"
    environment:
      STORAGE_TYPE: memory
      ADMIN_PASSWORD: test-admin-password
      UI_ENABLED: "true"
      UI_SESSION_SECRET: test-session-secret-at-least-32-chars
      LDAP_BIND_PASSWORD: "ldap-admin-password"
    volumes:
      - ./config.yaml:/etc/axonops-schema-registry/config.yaml
    depends_on:
      openldap:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/"]
      interval: 2s
      timeout: 5s
      retries: 15
```

**seed/bootstrap.ldif:**
```ldif
# Organisational Units
dn: ou=Users,dc=test,dc=axonops,dc=io
objectClass: organizationalUnit
ou: Users

dn: ou=Groups,dc=test,dc=axonops,dc=io
objectClass: organizationalUnit
ou: Groups

# ── Users ──

dn: cn=ldap-admin,ou=Users,dc=test,dc=axonops,dc=io
objectClass: inetOrgPerson
cn: ldap-admin
sn: Admin
uid: ldap-admin
mail: ldap-admin@test.axonops.io
userPassword: ldap-admin-pass

dn: cn=ldap-developer,ou=Users,dc=test,dc=axonops,dc=io
objectClass: inetOrgPerson
cn: ldap-developer
sn: Developer
uid: ldap-developer
mail: ldap-developer@test.axonops.io
userPassword: ldap-dev-pass

dn: cn=ldap-readonly,ou=Users,dc=test,dc=axonops,dc=io
objectClass: inetOrgPerson
cn: ldap-readonly
sn: Readonly
uid: ldap-readonly
mail: ldap-readonly@test.axonops.io
userPassword: ldap-readonly-pass

dn: cn=ldap-unmapped,ou=Users,dc=test,dc=axonops,dc=io
objectClass: inetOrgPerson
cn: ldap-unmapped
sn: Unmapped
uid: ldap-unmapped
mail: ldap-unmapped@test.axonops.io
userPassword: ldap-unmapped-pass

# ── Groups (role mapping targets) ──

dn: cn=SchemaRegistryAdmins,ou=Groups,dc=test,dc=axonops,dc=io
objectClass: groupOfNames
cn: SchemaRegistryAdmins
member: cn=ldap-admin,ou=Users,dc=test,dc=axonops,dc=io

dn: cn=Developers,ou=Groups,dc=test,dc=axonops,dc=io
objectClass: groupOfNames
cn: Developers
member: cn=ldap-developer,ou=Users,dc=test,dc=axonops,dc=io

dn: cn=Viewers,ou=Groups,dc=test,dc=axonops,dc=io
objectClass: groupOfNames
cn: Viewers
member: cn=ldap-readonly,ou=Users,dc=test,dc=axonops,dc=io
```

**config.yaml (relevant auth section):**
```yaml
security:
  auth:
    enabled: true
    methods:
      - basic
      - api_key
    bootstrap:
      enabled: true
      username: admin
      password: test-admin-password
      email: admin@test.local
    ldap:
      enabled: true
      url: ldap://openldap:389
      bind_dn: "cn=admin,dc=test,dc=axonops,dc=io"
      bind_password: "${LDAP_BIND_PASSWORD}"
      base_dn: "dc=test,dc=axonops,dc=io"
      user_search_base: "ou=Users,dc=test,dc=axonops,dc=io"
      user_search_filter: "(cn=%s)"
      username_attribute: cn
      email_attribute: mail
      group_search_base: "ou=Groups,dc=test,dc=axonops,dc=io"
      group_search_filter: "(member=%s)"
      group_attribute: memberOf
      role_mapping:
        "cn=SchemaRegistryAdmins,ou=Groups,dc=test,dc=axonops,dc=io": admin
        "cn=Developers,ou=Groups,dc=test,dc=axonops,dc=io": developer
        "cn=Viewers,ou=Groups,dc=test,dc=axonops,dc=io": readonly
      default_role: readonly
      connection_timeout: 10
      request_timeout: 30
```

**BDD feature: login-ldap.feature:**
```gherkin
@profile:ldap
Feature: Sign in via LDAP

  Background:
    Given the schema registry is running with LDAP authentication enabled
    And the LDAP server contains the following users:
      | username       | password        | groups                |
      | ldap-admin     | ldap-admin-pass | SchemaRegistryAdmins  |
      | ldap-developer | ldap-dev-pass   | Developers            |
      | ldap-readonly  | ldap-readonly-pass | Viewers            |
      | ldap-unmapped  | ldap-unmapped-pass | (none)              |

  Scenario: LDAP user signs in and gets correct role from group mapping
    Given I am on the login page
    When I enter username "ldap-developer"
    And I enter password "ldap-dev-pass"
    And I click "Sign In"
    Then I should be redirected to the subjects page
    And the top bar should display username "ldap-developer"
    And I should be authenticated with role "developer"
    And I should see the "Register New Schema" button
    And the sidebar should NOT show "Users" link

  Scenario: LDAP admin user gets admin navigation
    Given I am on the login page
    When I sign in as "ldap-admin" with password "ldap-admin-pass"
    Then the sidebar should show "Users" link
    And the sidebar should show "API Keys" link
    And the sidebar should show "Compatibility" link

  Scenario: LDAP readonly user gets restricted UI
    Given I sign in as "ldap-readonly" with password "ldap-readonly-pass"
    When I navigate to the subjects page
    Then I should NOT see the "Register New Schema" button
    When I navigate to subject "test-subject" detail page
    Then I should NOT see the "Register New Version" button

  Scenario: LDAP user with no group mapping gets default role
    Given I am on the login page
    When I sign in as "ldap-unmapped" with password "ldap-unmapped-pass"
    Then I should be authenticated with role "readonly"

  Scenario: LDAP user with wrong password is rejected
    Given I am on the login page
    When I enter username "ldap-developer"
    And I enter password "wrong-password"
    And I click "Sign In"
    Then I should see an error message "Invalid username or password"

  Scenario: Local user still works when LDAP is enabled
    Given I am on the login page
    When I sign in as "admin" with password "test-admin-password"
    Then I should be redirected to the subjects page
    And I should be authenticated with role "super_admin"

  Scenario: LDAP user does not see "change password" option
    Given I sign in as "ldap-developer" with password "ldap-dev-pass"
    When I navigate to my profile page
    Then I should NOT see the "Change Password" section
    And I should see a message "Password is managed by your LDAP directory"
```

---

### Profile: `oidc` — Local DB + OIDC via Keycloak (Phase 5)

**docker-compose.yml:**
```yaml
services:
  keycloak:
    image: quay.io/keycloak/keycloak:26.0
    command: start-dev --import-realm
    environment:
      KC_BOOTSTRAP_ADMIN_USERNAME: kc-admin
      KC_BOOTSTRAP_ADMIN_PASSWORD: kc-admin-password
      KC_HTTP_PORT: 8080
    volumes:
      - ./seed/schema-registry-realm.json:/opt/keycloak/data/import/schema-registry-realm.json
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD-SHELL", "exec 3<>/dev/tcp/localhost/8080 && echo -e 'GET /health/ready HTTP/1.1\\r\\nHost: localhost\\r\\n\\r\\n' >&3 && cat <&3 | grep -q '200'"]
      interval: 5s
      timeout: 10s
      retries: 30
      start_period: 30s

  schema-registry:
    build:
      context: ../../..
      dockerfile: Dockerfile
    ports:
      - "8081:8081"
    environment:
      STORAGE_TYPE: memory
      ADMIN_PASSWORD: test-admin-password
      UI_ENABLED: "true"
      UI_SESSION_SECRET: test-session-secret-at-least-32-chars
      OIDC_CLIENT_SECRET: test-client-secret
    volumes:
      - ./config.yaml:/etc/axonops-schema-registry/config.yaml
    depends_on:
      keycloak:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/"]
      interval: 2s
      timeout: 5s
      retries: 15
```

**seed/schema-registry-realm.json:**
```json
{
  "realm": "schema-registry",
  "enabled": true,
  "roles": {
    "realm": [
      { "name": "sr-admin" },
      { "name": "sr-developer" },
      { "name": "sr-readonly" }
    ]
  },
  "groups": [
    { "name": "schema-registry-admins", "realmRoles": ["sr-admin"] },
    { "name": "developers", "realmRoles": ["sr-developer"] },
    { "name": "readonly-users", "realmRoles": ["sr-readonly"] }
  ],
  "users": [
    {
      "username": "oidc-admin",
      "enabled": true,
      "emailVerified": true,
      "email": "oidc-admin@test.axonops.io",
      "credentials": [{ "type": "password", "value": "oidc-admin-pass" }],
      "groups": ["schema-registry-admins"]
    },
    {
      "username": "oidc-developer",
      "enabled": true,
      "emailVerified": true,
      "email": "oidc-dev@test.axonops.io",
      "credentials": [{ "type": "password", "value": "oidc-dev-pass" }],
      "groups": ["developers"]
    },
    {
      "username": "oidc-readonly",
      "enabled": true,
      "emailVerified": true,
      "email": "oidc-readonly@test.axonops.io",
      "credentials": [{ "type": "password", "value": "oidc-readonly-pass" }],
      "groups": ["readonly-users"]
    }
  ],
  "clients": [
    {
      "clientId": "schema-registry",
      "enabled": true,
      "clientAuthenticatorType": "client-secret",
      "secret": "test-client-secret",
      "redirectUris": ["http://localhost:8081/ui/auth/oidc/callback"],
      "webOrigins": ["http://localhost:8081"],
      "publicClient": false,
      "directAccessGrantsEnabled": true,
      "defaultClientScopes": ["openid", "profile", "email", "roles"],
      "protocolMappers": [
        {
          "name": "group-membership",
          "protocol": "openid-connect",
          "protocolMapper": "oidc-group-membership-mapper",
          "config": {
            "claim.name": "groups",
            "full.path": false,
            "id.token.claim": "true",
            "access.token.claim": "true",
            "userinfo.token.claim": "true"
          }
        }
      ]
    }
  ]
}
```

**BDD feature: login-oidc.feature:**
```gherkin
@profile:oidc
Feature: Sign in via OIDC (SSO)

  Background:
    Given the schema registry is running with OIDC authentication enabled
    And Keycloak is running with the "schema-registry" realm
    And the following OIDC users exist in Keycloak:
      | username       | password        | group                   |
      | oidc-admin     | oidc-admin-pass | schema-registry-admins  |
      | oidc-developer | oidc-dev-pass   | developers              |
      | oidc-readonly  | oidc-readonly-pass | readonly-users       |

  Scenario: Login page shows SSO button when OIDC is enabled
    Given I am on the login page
    Then I should see a "Sign in with SSO" button
    And I should still see the username and password fields

  Scenario: OIDC login flow redirects through Keycloak
    Given I am on the login page
    When I click "Sign in with SSO"
    Then I should be redirected to the Keycloak login page
    When I enter username "oidc-developer" in the Keycloak form
    And I enter password "oidc-dev-pass" in the Keycloak form
    And I click "Sign In" on the Keycloak page
    Then I should be redirected back to the schema registry UI
    And I should be authenticated as "oidc-developer"
    And I should have role "developer"

  Scenario: OIDC admin user gets correct permissions
    Given I complete the OIDC login flow as "oidc-admin" with password "oidc-admin-pass"
    Then the sidebar should show "Users" link
    And the sidebar should show "API Keys" link

  Scenario: OIDC user does not see "change password"
    Given I complete the OIDC login flow as "oidc-developer" with password "oidc-dev-pass"
    When I navigate to my profile page
    Then I should NOT see the "Change Password" section
    And I should see a message "Password is managed by your identity provider"

  Scenario: Local login still works alongside OIDC
    Given I am on the login page
    When I enter username "admin"
    And I enter password "test-admin-password"
    And I click "Sign In"
    Then I should be redirected to the subjects page
    And I should be authenticated as "admin"
```

---

### Profile: `noauth` — Auth Disabled

Tests that the UI works without authentication (for development/testing deployments).

**BDD feature: noauth-bypass.feature:**
```gherkin
@profile:noauth
Feature: UI without authentication

  Background:
    Given the schema registry is running with auth disabled

  Scenario: No login page shown
    When I navigate to "/ui/"
    Then I should be on the subjects page
    And I should NOT see a login page

  Scenario: All features are accessible
    When I navigate to "/ui/subjects"
    Then I should see the subjects list
    And I should see the "Register New Schema" button
    When I navigate to "/ui/admin/users"
    Then I should see the user management page
```

---

### Profile: `role-access` — Role-Based Access Control Tests

These are particularly important because they verify the **end-to-end** chain: LDAP group → role mapping → UI visibility → API authorisation.

**BDD feature: role-based-access.feature:**
```gherkin
@profile:basic @profile:ldap
Feature: Role-based access control in the UI

  Scenario Outline: Navigation visibility by role
    Given I sign in as a user with role "<role>"
    Then the sidebar should <subjects_visible> "Subjects" link
    And the sidebar should <schemas_visible> "Schema Browser" link
    And the sidebar should <config_visible> "Compatibility" link
    And the sidebar should <modes_visible> "Modes" link
    And the sidebar should <users_visible> "Users" link
    And the sidebar should <apikeys_visible> "API Keys" link
    And the sidebar should <import_visible> "Import" link

    Examples:
      | role        | subjects_visible | schemas_visible | config_visible | modes_visible | users_visible | apikeys_visible | import_visible |
      | super_admin | show             | show            | show           | show          | show          | show            | show           |
      | admin       | show             | show            | show           | show          | show          | show            | show           |
      | developer   | show             | show            | NOT show       | NOT show      | NOT show      | NOT show        | NOT show       |
      | readonly    | show             | show            | NOT show       | NOT show      | NOT show      | NOT show        | NOT show       |

  Scenario: Developer can register schemas but not delete
    Given I sign in as a user with role "developer"
    And subject "test-subject" exists with at least one version
    When I navigate to the subject "test-subject" detail page
    Then I should see the "Register New Version" button
    And I should NOT see the "Delete Subject" button
    And I should NOT see "Delete" buttons on version rows

  Scenario: Readonly user cannot register or delete
    Given I sign in as a user with role "readonly"
    When I navigate to the subjects page
    Then I should NOT see the "Register New Schema" button
    When I navigate to subject "test-subject" detail page
    Then I should NOT see the "Register New Version" button
    And I should NOT see the "Delete Subject" button

  Scenario: Direct URL access to admin page is denied for developer
    Given I sign in as a user with role "developer"
    When I navigate directly to "/ui/admin/users"
    Then I should see a message "You don't have permission to access this page"

  Scenario: Developer can manage own API keys only
    Given I sign in as a user with role "developer"
    When I navigate to "/ui/account/apikeys"
    Then I should see the "My API Keys" page
    And I should see the "Create API Key" button
    When I navigate to "/ui/admin/apikeys"
    Then I should see a message "You don't have permission to access this page"
```

---

### CI Execution Strategy

**GitHub Actions workflow:**

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e-basic:
    name: "E2E: Basic Auth"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20 }
      - name: Install Playwright
        run: npx playwright install --with-deps chromium
      - name: Start test environment
        run: docker compose -f tests/e2e/profiles/basic/docker-compose.yml up -d --build --wait
      - name: Run BDD tests
        run: |
          cd tests/e2e
          npx cucumber-js --tags "@profile:basic" --format json:reports/basic.json
      - name: Collect traces on failure
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-traces-basic
          path: tests/e2e/traces/
      - name: Teardown
        if: always()
        run: docker compose -f tests/e2e/profiles/basic/docker-compose.yml down -v

  e2e-ldap:
    name: "E2E: LDAP Auth"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20 }
      - name: Install Playwright
        run: npx playwright install --with-deps chromium
      - name: Start test environment
        run: docker compose -f tests/e2e/profiles/ldap/docker-compose.yml up -d --build --wait
      - name: Run BDD tests
        run: |
          cd tests/e2e
          npx cucumber-js --tags "@profile:ldap" --format json:reports/ldap.json
      - name: Collect traces on failure
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-traces-ldap
          path: tests/e2e/traces/
      - name: Teardown
        if: always()
        run: docker compose -f tests/e2e/profiles/ldap/docker-compose.yml down -v

  e2e-noauth:
    name: "E2E: No Auth"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20 }
      - name: Install Playwright
        run: npx playwright install --with-deps chromium
      - name: Start test environment
        run: docker compose -f tests/e2e/profiles/noauth/docker-compose.yml up -d --build --wait
      - name: Run BDD tests
        run: |
          cd tests/e2e
          npx cucumber-js --tags "@profile:noauth" --format json:reports/noauth.json
      - name: Teardown
        if: always()
        run: docker compose -f tests/e2e/profiles/noauth/docker-compose.yml down -v

  # Phase 5: uncomment when OIDC is implemented
  # e2e-oidc:
  #   name: "E2E: OIDC Auth"
  #   ...
```

**Key design decisions:**
- Each profile runs as a **separate CI job** so they can run in parallel
- Profile jobs are independent — failure in LDAP tests doesn't block Basic tests
- Playwright traces, screenshots, and video are captured on failure for debugging
- Docker Compose `--wait` ensures healthchecks pass before tests start
- The `@profile:` tags control which features run against which stack
- Some features carry multiple tags (e.g., `@profile:basic @profile:ldap`) meaning they run in both profiles — this verifies that basic login still works when LDAP is also enabled

---

### Testing Edge Cases Across Profiles

| Test scenario | Profile | Why it matters |
|---------------|---------|---------------|
| Local user login works when LDAP is enabled | `ldap` | Ensures LDAP doesn't break local auth |
| LDAP user with no group gets default role | `ldap` | Verifies `default_role: readonly` config |
| LDAP server down → graceful error | `ldap` (stop OpenLDAP mid-test) | Users shouldn't see a stack trace |
| OIDC callback with invalid state param | `oidc` | Prevents CSRF attacks |
| Session expires during navigation | `basic` | Token TTL enforcement works |
| API key login → then access admin page | `basic` | Role from API key is respected in UI |
| Login rate limiting (429) | `basic` | UI shows retry message, not raw error |
| `auth.enabled: false` → no login page | `noauth` | Anonymous mode works end-to-end |
| Admin creates user → new user can log in | `basic` | Full lifecycle: create user via admin UI → that user signs in |
| LDAP admin changes password in LDAP → new password works | `ldap` | Would need to ldapmodify the test server mid-test |
| Multiple browser tabs with different users | `basic` | Independent sessions, no crosstalk |
