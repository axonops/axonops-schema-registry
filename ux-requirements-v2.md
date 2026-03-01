# AxonOps Schema Registry — Web UI Product Requirements

> **Purpose:** Define what users can do with the Web UI, how features behave, and in what order they should be delivered — so that claude-code can implement them and BDD tests can verify them.
>
> **Audience:** Implementers (claude-code), QA (BDD feature authors), product reviewers.

---

## Personas

| Persona | Role(s) | What they care about |
|---------|---------|---------------------|
| **Platform Engineer** | super_admin, admin | Configuring the registry, managing users and API keys, setting compatibility policies, importing/migrating schemas, operational visibility |
| **Developer** | developer | Registering and evolving schemas quickly with confidence, understanding compatibility rules, checking what schemas exist, creating API keys for their services |
| **Auditor / Viewer** | readonly | Inspecting schemas, understanding what's registered, tracing schema references, reviewing version history |

---

## Delivery Phases

Each phase is a shippable increment. Features within a phase can be implemented and tested independently.

| Phase | Theme | Value delivered |
|-------|-------|----------------|
| **1** | See & Sign In | Users can authenticate and browse everything in the registry |
| **2** | Author & Evolve | Users can register schemas, evolve them, and understand compatibility |
| **3** | Govern | Admins can control compatibility policies, modes, and perform migrations |
| **4** | Administer | Admins can manage users, API keys, and credentials |
| **5** | Delight | Power-user features, polish, OIDC/SAML, advanced visualisation |

---

## How to Read This Document

Each **Feature** is a self-contained unit of user value. Features are grouped by domain. Each feature includes:

- **User story** — who wants what and why
- **Capabilities** — the specific things a user can do (these map 1:1 to BDD scenarios)
- **Acceptance criteria** — observable outcomes that prove the capability works
- **Edge cases & schema-format notes** — the tricky stuff that must be handled
- **Test instrumentation** — `data-testid` attributes for Playwright targeting
- **Phase** — when this feature ships

---

# DOMAIN: Authentication & Sessions

## Feature: Sign In

**Phase 1**

### User Story

> As any user, I want to sign in to the Web UI using my existing credentials (username/password, LDAP, or API key) so that I can access the registry with the permissions assigned to my role.

### How It Works

The Go backend issues a **signed JWT session token** after validating credentials. The UI stores this token in memory (React context) and sends it as `Authorization: Bearer <token>` on all API requests. The UI never holds raw passwords beyond the initial login POST.

This design means the UI is **auth-method-agnostic** — it collects credentials, sends them to the backend, and receives a token. The backend decides how to validate (local DB, LDAP, or later OIDC/SAML).

### Backend Endpoints Required

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `GET /ui/auth/config` | GET | None | Reports which login methods are enabled |
| `POST /ui/auth/login` | POST | None | Username/password login → session token |
| `POST /ui/auth/apikey` | POST | None | API key login → session token |
| `GET /ui/auth/session` | GET | Bearer | Validates token, returns user info + role |
| `POST /ui/auth/logout` | POST | Bearer | Invalidates session |

### Session Token

- Signed JWT with configurable TTL (default 30 minutes)
- Contains: `sub` (username), `role`, `email`, `auth_method`, `exp`, `jti`
- Auto-refreshed by the frontend when approaching expiry (configurable window)
- Configuration:

```yaml
ui:
  enabled: true
  base_path: "/ui"
  session:
    secret: "${UI_SESSION_SECRET}"
    token_ttl_minutes: 30
    refresh_window_minutes: 5
```

### Capabilities

**C1.1 — Discover available login methods**
- On page load, the UI calls `GET /ui/auth/config` to learn what's enabled
- The login form adapts: shows username/password fields when `basic` is listed, shows SSO button when `oidc` is present (Phase 5), always shows API key toggle when `api_key` is listed
- Acceptance: login form renders correct elements based on server config
- `data-testid`: `login-page`, `login-username-input`, `login-password-input`, `login-submit-btn`, `login-apikey-toggle`, `login-apikey-input`, `login-sso-btn` (Phase 5), `login-error-msg`

**C1.2 — Sign in with username and password**
- User enters username + password and clicks Sign In (or presses Enter)
- Frontend POSTs to `/ui/auth/login` with `{ "username": "...", "password": "..." }`
- Backend validates against local DB first; if not found and LDAP enabled, attempts LDAP bind
- On LDAP success, maps LDAP groups to registry roles via existing `role_mapping` config
- On success: token stored in memory, user redirected to originally-requested URL (or `/ui/subjects`)
- On failure: inline error "Invalid username or password" — no indication of _which_ was wrong
- On rate limit (429): "Too many login attempts. Please wait and try again."

**C1.3 — Sign in with API key**
- User toggles "Use API Key instead" — form switches to a single API key input
- Frontend POSTs to `/ui/auth/apikey` with `{ "key": "sr_live_abc123..." }`
- Backend validates via SHA-256 hash lookup, returns session token scoped to the key's role
- Same success/failure behaviour as C1.2

**C1.4 — Session persistence within a tab**
- Token lives in React context (in-memory only — not localStorage, not cookies)
- Refreshing the page clears the session — user must re-authenticate
- This is a deliberate security choice: no persistent credential storage in the browser

**C1.5 — Session expiry and renewal**
- When the token is within `refresh_window_minutes` of expiry and the user is active, the frontend calls `GET /ui/auth/session` to obtain a fresh token
- If the token has already expired (no API call made in time), any subsequent API call returns 401 → frontend clears state and redirects to login with message "Your session has expired. Please sign in again."

**C1.6 — Sign out**
- User clicks their username → dropdown → "Sign Out"
- Frontend POSTs to `/ui/auth/logout`, clears token, redirects to login
- `data-testid`: `nav-user-menu`, `nav-signout-btn`

**C1.7 — Role-based UI rendering**
- The token contains the user's role. The UI uses this to control visibility:

| Capability | super_admin | admin | developer | readonly |
|------------|:-----------:|:-----:|:---------:|:--------:|
| Browse schemas & subjects | ✓ | ✓ | ✓ | ✓ |
| Register/evolve schemas | ✓ | ✓ | ✓ | ✗ |
| Delete schemas/subjects | ✓ | ✓ | ✗ | ✗ |
| Change compatibility/mode | ✓ | ✓ | ✗ | ✗ |
| Import schemas | ✓ | ✓ | ✗ | ✗ |
| Manage users | ✓ | ✓* | ✗ | ✗ |
| Manage all API keys | ✓ | ✓ | ✗ | ✗ |
| Manage own API keys | ✓ | ✓ | ✓ | ✓ |
| Change own password | ✓ | ✓ | ✓ | ✓ |

*admin can manage users but not super_admins.

- Actions the user cannot perform are **hidden** (not disabled/grayed out)
- If a user navigates directly to a URL they lack permission for, they see a "You don't have permission to access this page" message (not a raw 403)

### Edge Cases

- LDAP user's first login: if the user exists in LDAP but not in the local DB, the backend creates a local user record with the LDAP-mapped role on first successful bind
- API key that's expired: "This API key has expired. Please use a valid key or sign in with username and password."
- API key that's revoked: same error as invalid — no distinction exposed to the user
- Multiple tabs: each tab holds its own independent session token in memory

---

## Feature: Sign In with SSO (OIDC / SAML)

**Phase 5**

### User Story

> As a user in an organisation that uses Okta/Keycloak/Azure AD/Auth0, I want to sign in via my identity provider so that I don't need a separate password for the schema registry.

### Capabilities

**C1.8 — OIDC login flow**
- Login page shows "Sign in with {display_name}" button (text configurable, e.g., "Sign in with Okta")
- Clicking navigates to `/ui/auth/oidc/login` → Go backend redirects to IdP
- After IdP authentication, callback lands at `/ui/auth/oidc/callback`
- Backend exchanges auth code for tokens, maps claims to registry role, issues session token
- User is redirected to `/ui/` with token in URL fragment (not query string — avoids server logs)

**C1.9 — mTLS auto-login**
- If the server is configured with `client_auth: verify`, TLS handshake already authenticated the user
- `/ui/auth/config` reports `mtls_authenticated: true` with the client CN
- Login page shows "Authenticated as {CN} via client certificate" with a "Continue" button
- No credentials form needed

### Backend Endpoints (Phase 5 additions)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `GET /ui/auth/oidc/login` | GET | Redirect to OIDC provider |
| `GET /ui/auth/oidc/callback` | GET | Handle OIDC redirect, issue token |
| `GET /ui/auth/saml/login` | GET | Redirect to SAML IdP |
| `POST /ui/auth/saml/callback` | POST | Handle SAML assertion, issue token |

---

# DOMAIN: Schema Browsing & Discovery

## Feature: Browse Subjects

**Phase 1**

### User Story

> As any user, I want to see all subjects in the registry so that I can find the schemas my team owns or depends on, understand what's registered, and navigate to details.

### Capabilities

**C2.1 — View the subject list**
- Shows all subjects in a table: subject name, schema type (Avro/Protobuf/JSON Schema), latest version number, compatibility level, mode, last updated timestamp
- Compatibility column shows "BACKWARD" if overridden at subject level, or "Global: BACKWARD" if inherited — making it immediately clear which subjects have custom policies
- Sorted alphabetically by default; clickable column headers for re-sorting
- `data-testid`: `subjects-list-table`, `subjects-row-{subjectName}`

**C2.2 — Search and filter subjects**
- Text input filters subjects by name as the user types (debounced 300ms, case-insensitive substring match)
- Subjects matching the filter are shown; non-matching hidden client-side (for registries with <1000 subjects) or server-side via `subjectPrefix` (for larger registries)
- `data-testid`: `subjects-search-input`, `subjects-count-badge`

**C2.3 — View deleted subjects**
- Checkbox "Show deleted subjects" — toggles visibility of soft-deleted subjects
- Deleted subjects render with strikethrough text and a "Deleted" badge
- Clicking a deleted subject navigates to its detail page where permanent-delete or recovery options are available
- `data-testid`: `subjects-show-deleted-toggle`

**C2.4 — Paginate large subject lists**
- For registries with many subjects, paginate using `offset`/`limit` from the API
- Show page controls: previous/next, current page indicator
- `data-testid`: `subjects-pagination`, `subjects-page-prev`, `subjects-page-next`

**C2.5 — Navigate to subject detail**
- Clicking a subject name navigates to the subject detail page
- Subject name is a link styled as such

**C2.6 — Empty state**
- When no subjects exist: friendly message "No subjects registered yet. Register your first schema to get started." with a CTA button linking to schema registration (if user has permission)
- `data-testid`: `subjects-empty-state`

---

## Feature: Inspect a Subject

**Phase 1**

### User Story

> As any user, I want to drill into a subject to see all its versions, understand its configuration, and explore individual schema versions.

### Capabilities

**C3.1 — View version history**
- Shows all versions of a subject in a table: version number, global schema ID, status (active/deleted), registered timestamp
- Most recent version at the top
- Each version links to the full schema detail view
- `data-testid`: `subject-versions-table`, `subject-version-row-{version}`

**C3.2 — View subject metadata**
- Shows: subject name (as heading), schema type badge, total version count, current compatibility level (with inheritance indicator), current mode
- `data-testid`: `subject-name-heading`, `subject-schema-type-badge`, `subject-compat-badge`, `subject-mode-badge`

**C3.3 — View deleted versions**
- Toggle "Show deleted versions" to include soft-deleted versions
- Deleted versions shown with visual indicator
- `data-testid`: `subject-show-deleted-toggle`

**C3.4 — Quick-view latest schema**
- Below the version table, show a read-only preview of the latest schema version with syntax highlighting
- This saves a click for the most common task: "what does this schema look like right now?"
- `data-testid`: `subject-latest-schema-preview`

---

## Feature: Inspect a Schema Version

**Phase 1**

### User Story

> As any user, I want to examine a specific schema version in detail — see its full definition, understand its references, and compare it to other versions — so that I can understand the data contract.

### Capabilities

**C4.1 — View the full schema definition**
- Monaco Editor in read-only mode with full syntax highlighting for the schema type
- Avro: JSON syntax highlighting with awareness of Avro-specific constructs
- Protobuf: proto syntax highlighting (proto2/proto3)
- JSON Schema: JSON syntax highlighting
- `data-testid`: `version-schema-viewer`

**C4.2 — Copy and download the schema**
- "Copy to clipboard" button — copies the raw schema string
- "Download" button — downloads as `.avsc` (Avro), `.proto` (Protobuf), or `.json` (JSON Schema)
- `data-testid`: `version-copy-btn`, `version-download-btn`

**C4.3 — View schema metadata**
- Panel showing: global schema ID, version number, subject, schema type, registered timestamp, status
- `data-testid`: `version-metadata-panel`

**C4.4 — View schema references (outgoing)**
- If this schema references other schemas, list them as clickable links
- Each shows: referenced subject, referenced version, reference name
- For Avro: references are used for reusable named types across schemas
- For Protobuf: references map to `import` statements
- For JSON Schema: references map to `$ref` external URIs
- `data-testid`: `version-references-list`

**C4.5 — View "referenced by" (incoming)**
- List schemas that reference *this* schema version (via `GET /subjects/{subject}/versions/{version}/referencedby`)
- Shows which other schemas depend on this one — critical for understanding blast radius before making changes
- `data-testid`: `version-referenced-by-list`

**C4.6 — Breadcrumb navigation**
- Breadcrumb trail: Subjects → {subject name} → Version {n}
- Each segment is clickable for quick navigation up the hierarchy
- `data-testid`: `breadcrumb`

---

## Feature: Compare Schema Versions (Diff)

**Phase 2**

### User Story

> As a developer, I want to compare two versions of a schema side-by-side so that I can see exactly what changed between versions — fields added, removed, type changes, default changes — and understand the evolution history.

### Capabilities

**C5.1 — Side-by-side diff of two versions**
- On the schema version detail page, a dropdown lets the user select another version to compare against
- Uses Monaco's built-in diff editor: left pane = selected version, right pane = current version
- Added lines highlighted in green, removed in red, changed in yellow
- `data-testid`: `version-diff-select`, `version-diff-viewer`

**C5.2 — Diff-aware display for schema types**
- For Avro (JSON): diff works naturally on the JSON structure
- For Protobuf: diff works on the raw `.proto` text
- For JSON Schema: diff works on the JSON structure
- The diff should use the **normalized/canonical** form of the schema if available, so cosmetic changes (whitespace, field reordering in JSON) don't show as false diffs

**C5.3 — Summary of changes**
- Above the diff, show a human-readable summary: "3 fields added, 1 field removed, 1 type changed"
- This is a best-effort parse — works well for Avro/JSON Schema (JSON-based), less precise for Protobuf (text-based)
- `data-testid`: `version-diff-summary`

### Schema-Format Complexity Notes

**Avro diffs that matter:**
- Field added with default → backward compatible
- Field added without default → backward incompatible
- Field removed → forward incompatible
- Type widened (int → long) → check compatibility mode
- Union type added/removed → depends on direction
- `logicalType` added/changed → may break consumers
- `aliases` added → transparent to compatibility but meaningful to display
- Nested record changed → the diff must recurse into nested type definitions
- `order` attribute changed on fields → affects sort ordering but not serialization

**Protobuf diffs that matter:**
- Field added with new tag number → generally safe
- Field removed → the tag number must not be reused
- Field type changed → breaking
- `oneof` fields added/removed → must be handled carefully
- `reserved` declarations added → significant for evolution
- Package/import changes → affects generated code

**JSON Schema diffs that matter:**
- `required` array changed → changes validation strictness
- `properties` added/removed → new or lost fields
- `additionalProperties` changed → open/closed content model
- `allOf`/`anyOf`/`oneOf` composition changed → complex semantics
- `$ref` targets changed → different referenced definition

---

## Feature: Search Schemas by Global ID

**Phase 1**

### User Story

> As a developer debugging a serialization issue, I want to look up a schema by its global ID (which I found in a Kafka message header) so that I can see what schema was used to serialize that message.

### Capabilities

**C6.1 — Look up schema by global ID**
- Input field for entering a numeric schema ID
- On submit, fetches `GET /schemas/ids/{id}` and displays the schema
- Shows: the schema definition, type, and all subjects/versions that use this schema (via `GET /schemas/ids/{id}/subjects` and `GET /schemas/ids/{id}/versions`)
- `data-testid`: `schemas-id-input`, `schemas-id-search-btn`, `schemas-result-viewer`

**C6.2 — Browse all schemas**
- Filterable table/list of all schemas in the registry
- Filters: schema type (Avro/Protobuf/JSON Schema), subject prefix, latestOnly toggle, show deleted
- Uses `GET /schemas` with query parameters
- `data-testid`: `schemas-browse-table`, `schemas-type-filter`, `schemas-prefix-filter`

---

# DOMAIN: Schema Authoring & Evolution

## Feature: Register a New Schema

**Phase 2**

### User Story

> As a developer, I want to register a schema for a new subject (e.g., `orders-value`) so that producers and consumers can use it for serialization, with confidence that it's valid and the registry accepts it.

### Capabilities

**C7.1 — Create a new subject with an initial schema**
- Accessible from: the subjects list page ("Register New Schema" button) or the subject detail page ("Register New Version" button — which pre-fills the subject name)
- User provides:
  1. **Subject name** (text input — if creating new; pre-filled if adding version to existing)
  2. **Schema type** (Avro, Protobuf, JSON Schema — dropdown)
  3. **Schema definition** (Monaco Editor with full syntax support)
  4. **Schema references** (optional — see C7.5)
  5. **Normalize** (optional checkbox — sends `?normalize=true`)
- `data-testid`: `register-subject-input`, `register-type-select`, `register-schema-editor`, `register-normalize-toggle`, `register-submit-btn`

**C7.2 — Real-time schema validation as you type**
- The editor provides immediate feedback on syntax errors:
  - **Avro**: validates JSON syntax + Avro schema structure (required fields like `type`, `name`, `fields`; valid field types; union rules — no nested unions, no duplicate types; enum symbols must be non-empty; fixed size must be positive; namespace format)
  - **Protobuf**: validates proto2/proto3 syntax (message structure, field numbers, type declarations, reserved fields, oneof groups, import statements)
  - **JSON Schema**: validates against the JSON Schema meta-schema (valid `type` values, `properties` structure, `$ref` format, `required` as array of strings)
- Syntax errors shown as red squiggles inline in the editor
- A status badge shows "Valid ✓" or "N errors found ✗" below the editor
- This is **client-side validation only** — a fast feedback loop. Server-side validation happens on submit.
- `data-testid`: `register-validation-status`

**C7.3 — Check compatibility before registering**
- "Check Compatibility" button calls `POST /compatibility/subjects/{subject}/versions` with the schema
- Result displayed inline:
  - ✅ "Compatible" — green indicator with details of what was checked
  - ❌ "Incompatible" — red indicator with the specific incompatibility reason from the API (e.g., "Field 'email' was removed, which is not allowed under BACKWARD compatibility")
- This is optional — the user can skip it and register directly. But the UI should encourage checking first (button is visually prominent, positioned before the Register button)
- For a brand-new subject (no existing versions), compatibility check always passes — the UI can show "No previous versions to check against"
- `data-testid`: `register-compat-check-btn`, `register-compat-result`

**C7.4 — Submit the schema**
- "Register" button sends `POST /subjects/{subject}/versions` with the schema
- On success: toast notification "Schema registered as version {n} (ID: {id})", redirect to the new version detail page
- On failure (incompatible): error message with the incompatibility details from the API
- On failure (invalid schema): error message with parsing errors from the API
- On failure (409 conflict): "This exact schema is already registered as version {n}" — link to the existing version
- `data-testid`: `register-submit-btn`

**C7.5 — Add schema references**
- A collapsible "References" section below the editor
- User can add references: each reference requires subject name (with autocomplete from existing subjects), version (number or "latest" = `-1`), and reference name
- For Avro: references allow reusing named types defined in other schemas (e.g., a shared `Address` record used by `Customer` and `Order` schemas)
- For Protobuf: references map to `import` statements — the reference name is the import path
- For JSON Schema: references map to `$ref` URIs
- Multiple references can be added
- `data-testid`: `register-references-section`, `register-add-reference-btn`, `register-reference-subject-input`, `register-reference-version-input`, `register-reference-name-input`

### Schema-Format Authoring Notes

These are the real-world complexities the editor must handle well:

**Avro authoring scenarios:**
- Simple record with primitive fields (string, int, long, float, double, boolean, bytes, null)
- **Nullable fields** — the most common pattern: `"type": ["null", "string"]` with `"default": null`. The UI should not trip users up here — null must be first in the union for the default to work
- **Nested records** — a record containing another record inline. Named types are defined where first used and can be referenced by name later in the same schema
- **Enums** with `symbols` array and optional `default`. Enum evolution is tricky: adding symbols at the end is backward-compatible, removing or reordering is not
- **Arrays and maps** — `{"type": "array", "items": "string"}` and `{"type": "map", "values": "long"}`
- **Unions** — JSON arrays of types. Rules: no nested unions, no duplicate types (except named types with different fullnames), null typically first for optional fields
- **Logical types** — `{"type": "long", "logicalType": "timestamp-millis"}`, `{"type": "int", "logicalType": "date"}`, `{"type": "string", "logicalType": "uuid"}`, `{"type": "bytes", "logicalType": "decimal", "precision": 10, "scale": 2}`, `{"type": "fixed", "size": 12, "logicalType": "duration"}`
- **Fixed type** — `{"type": "fixed", "name": "md5", "size": 16}`
- **Aliases** on records and fields — `"aliases": ["OldName"]` — allow renaming without breaking compatibility
- **Doc strings** — `"doc": "Description of this field"` — documentation embedded in the schema
- **Field ordering** — `"order": "ascending"` | `"descending"` | `"ignore"` — affects sort order
- **Default values** — must match the type of the first element in a union; for bytes/fixed, JSON strings mapping to byte values
- **Namespace inheritance** — nested records inherit namespace from parent unless explicitly overridden

**Protobuf authoring scenarios:**
- proto2 vs proto3 syntax (`syntax = "proto2"` vs `syntax = "proto3"`)
- `message` definitions with typed fields and unique field numbers
- `enum` types (proto enums must have a 0-valued first entry in proto3)
- `oneof` groups — mutually exclusive fields
- `map` fields — `map<string, int32>`
- `repeated` fields for arrays
- `reserved` field numbers and names — critical for safe evolution
- `import` statements — these map to schema references in the registry
- `package` declarations — affect generated code namespaces
- Nested message types — messages defined inside other messages
- Well-known types (`google.protobuf.Timestamp`, `google.protobuf.Any`, etc.) — these may require references

**JSON Schema authoring scenarios:**
- `type` keyword with single or array values
- `properties` with nested schemas
- `required` array for mandatory fields
- `$ref` for referencing definitions (internal `#/definitions/` or external via schema references)
- `allOf`, `anyOf`, `oneOf` composition — powerful but complex
- `additionalProperties` — open vs closed content model (critical for compatibility)
- `enum` for string enums
- `pattern` for regex validation
- `format` for semantic validation (date-time, email, uri, etc.)
- `definitions` / `$defs` for reusable sub-schemas

### Templates (Phase 5 enhancement)

When the editor is empty, offer "Start from template" with common patterns:
- Avro: Simple record, Record with nullable fields, Record with nested types, Enum, Record with logical types
- Protobuf: proto3 message, Message with enum, Message with nested messages
- JSON Schema: Object schema, Object with $ref, Array schema

---

## Feature: Evolve an Existing Schema

**Phase 2**

### User Story

> As a developer, I want to register a new version of an existing schema with confidence that it won't break my producers or consumers, getting clear feedback about what's compatible and what's not.

### Capabilities

**C8.1 — Register a new version from the subject detail page**
- "Register New Version" button on the subject detail page
- Subject name is pre-filled and read-only
- Schema type is pre-selected to match the existing schema type
- The editor can optionally pre-populate with the latest version's schema as a starting point ("Start from latest version" link) — so the user can make incremental changes rather than writing from scratch
- `data-testid`: `subject-register-version-btn`

**C8.2 — Compatibility feedback with specific field-level detail**
- When the user clicks "Check Compatibility", the response from the API includes details about *why* a schema is incompatible
- The UI should display these details clearly, e.g.:
  - "BACKWARD incompatibility: Field 'email' with type 'string' was removed. Under BACKWARD compatibility, consumers using the old schema must be able to read data written with the new schema. Removing a field breaks old consumers that expect it."
  - "FORWARD incompatibility: Field 'phone' was added without a default value. Under FORWARD compatibility, consumers using the new schema must be able to read data written with the old schema. New fields without defaults can't be filled when reading old data."
- For complex cases (nested record changes, union type changes), the message should trace the path: "In field 'address.zip_code': type changed from 'string' to 'int'"

**C8.3 — Compatibility mode explainer**
- Next to the compatibility check result, show a brief explanation of what the current compatibility level means:
  - BACKWARD: "New schema can be used to read data written with the previous schema"
  - FORWARD: "Previous schema can be used to read data written with the new schema"
  - FULL: "Both backward and forward compatible"
  - NONE: "No compatibility checking — any schema can be registered"
  - Transitive variants: "Checked against ALL previous versions, not just the latest"
- This helps developers who don't have the compatibility matrix memorized
- `data-testid`: `compat-explainer`

### Schema Evolution Scenarios the UI Must Handle Well

These are the real-world evolution patterns that trip people up. The UI should help users succeed with these, not just accept/reject silently:

**Common Avro evolutions:**
- Adding an optional field: `"type": ["null", "string"], "default": null` → always safe under BACKWARD
- Adding a required field (no default): → breaks BACKWARD compatibility. The UI should flag this clearly: "Add a default value to make this backward-compatible"
- Removing a field: → breaks FORWARD compatibility. The UI should explain: "Consumers using the new schema won't be able to read the old field"
- Changing field type: int → long (widening) may be allowed; string → int is always breaking
- Adding enum symbols: safe if appended; reordering or removing is not
- Changing a union: adding a new type to a union is generally backward-compatible if null handling is preserved

**Common Protobuf evolutions:**
- Adding a new field with a new tag number → safe
- Removing a field → the tag number must be added to `reserved` to prevent reuse
- Renaming a field → safe in proto3 (field numbers matter, not names), but changes generated code
- Changing field type → breaking (even int32 → int64)
- Adding to a oneof → complex compatibility implications

**Common JSON Schema evolutions:**
- Adding a new property (not in `required`) → safe
- Adding a property to `required` → breaks backward compatibility
- Removing `additionalProperties: false` → loosens validation (backward-safe, forward-breaking)
- Adding `additionalProperties: false` → tightens validation (forward-safe, backward-breaking)

---

## Feature: Delete Schemas

**Phase 2**

### User Story

> As an admin, I want to delete schemas or entire subjects that are no longer needed — with safety nets to prevent accidental permanent data loss.

### Capabilities

**C9.1 — Soft-delete a schema version**
- Admin clicks "Delete" on a specific version → confirmation dialog: "This will soft-delete version {n} of '{subject}'. The schema can be recovered later."
- Calls `DELETE /subjects/{subject}/versions/{version}`
- Version is marked as deleted but remains recoverable
- `data-testid`: `version-delete-btn`, `confirm-dialog`, `confirm-dialog-confirm-btn`, `confirm-dialog-cancel-btn`

**C9.2 — Permanently delete a schema version**
- On a soft-deleted version, an additional "Permanently Delete" button appears
- Requires typing the subject name to confirm
- Calls `DELETE /subjects/{subject}/versions/{version}?permanent=true`
- Warning: "This action cannot be undone. The schema will be permanently removed."
- `data-testid`: `version-permanent-delete-btn`, `confirm-dialog-name-input`

**C9.3 — Soft-delete an entire subject**
- Admin clicks "Delete Subject" → confirmation dialog with subject name
- Calls `DELETE /subjects/{subject}`
- All versions are soft-deleted
- `data-testid`: `subject-delete-btn`

**C9.4 — Permanently delete an entire subject**
- Available only when subject is already soft-deleted
- Requires typing subject name to confirm
- Calls `DELETE /subjects/{subject}?permanent=true`
- `data-testid`: `subject-permanent-delete-btn`

### Edge Cases

- Cannot delete a schema version that is referenced by other schemas — the API returns an error. The UI should show: "This version cannot be deleted because it is referenced by: {list of subjects/versions}. Remove the references first."
- Deleting the only version of a subject effectively deletes the subject
- Soft-deleted versions are not visible by default but can be shown via toggle (see C3.3)

---

# DOMAIN: Governance & Configuration

## Feature: Manage Compatibility Levels

**Phase 3**

### User Story

> As a platform engineer, I want to set compatibility policies at the global level and override them per-subject so that different teams can evolve their schemas under appropriate rules — strict for production topics, relaxed for experimental ones.

### Capabilities

**C10.1 — View and change the global compatibility level**
- Shows current global level (from `GET /config`)
- Dropdown with all 7 options: NONE, BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE
- Each option has a one-line description in the dropdown
- "Save" button calls `PUT /config`
- `data-testid`: `config-global-compat-select`, `config-global-compat-save-btn`

**C10.2 — View all subject-level overrides**
- Table showing subjects with non-default compatibility levels
- Each row: subject name (link), compatibility level, "Reset to Global" action
- This gives admins a single view of all policy deviations
- `data-testid`: `config-overrides-table`

**C10.3 — Set subject-level compatibility**
- From the subject detail page (Configuration tab), dropdown to select compatibility level
- "Save" calls `PUT /config/{subject}`
- "Reset to Global Default" calls `DELETE /config/{subject}` — removes the override
- Shows: "Currently: FULL (overrides global: BACKWARD)" or "Currently: BACKWARD (inherited from global)"
- `data-testid`: `subject-compat-select`, `subject-compat-save-btn`, `subject-compat-reset-btn`

**C10.4 — Compatibility impact warning**
- Before changing a compatibility level to something more permissive (e.g., BACKWARD → NONE), show a warning: "Changing to NONE means any schema can be registered regardless of compatibility with previous versions. This could allow schemas that break existing consumers."
- Before changing to something more restrictive (e.g., NONE → FULL_TRANSITIVE), show: "Existing schemas may not be compatible under the new policy. This only affects future registrations."

---

## Feature: Manage Modes

**Phase 3**

### User Story

> As a platform engineer, I want to control whether the registry accepts writes, so that I can put it in read-only mode during maintenance or enable import mode for migrations.

### Capabilities

**C11.1 — View and change the global mode**
- Shows current mode from `GET /mode`
- Dropdown: READWRITE, READONLY, READONLY_OVERRIDE, IMPORT
- Each mode has a description:
  - READWRITE: "Normal operation — schemas can be registered and read"
  - READONLY: "No new schemas can be registered. Read access only."
  - READONLY_OVERRIDE: "Read-only for most subjects, but specific subjects can be set to READWRITE"
  - IMPORT: "Schemas can be imported with specific IDs — used for migration"
- `data-testid`: `mode-global-select`, `mode-global-save-btn`

**C11.2 — Subject-level mode override**
- Same pattern as compatibility: set per-subject, reset to global
- `data-testid`: `subject-mode-select`, `subject-mode-save-btn`, `subject-mode-reset-btn`

---

## Feature: Import Schemas (Migration)

**Phase 3**

### User Story

> As a platform engineer migrating from Confluent Schema Registry, I want to import schemas with their original IDs preserved so that existing producers/consumers continue to work without changes.

### Capabilities

**C12.1 — Import schemas via JSON upload**
- Monaco editor for pasting import JSON, or file upload (drag & drop)
- Uses `POST /import/schemas`
- `data-testid`: `import-editor`, `import-file-upload`, `import-submit-btn`

**C12.2 — Pre-flight mode check**
- Before allowing import, check that the mode is set to IMPORT
- If not: "The registry must be in IMPORT mode to import schemas. Current mode: READWRITE." with a button to switch mode (if user has permission)
- `data-testid`: `import-mode-warning`, `import-switch-mode-btn`

**C12.3 — Import results**
- After import, show results per schema: success (with assigned ID) or failure (with error reason)
- `data-testid`: `import-results-panel`

---

# DOMAIN: User & Access Management

## Feature: Manage Users

**Phase 4**

### User Story

> As a platform engineer, I want to create, update, and delete user accounts so that the right people have the right level of access to the schema registry.

### Capabilities

**C13.1 — List all users**
- Table: username, email, role (color-coded badge), enabled/disabled status, created timestamp
- Search/filter by username or email
- `data-testid`: `users-list-table`, `users-search-input`, `users-create-btn`

**C13.2 — Create a new user**
- Modal form: username (required, unique), email (required), password (required, with strength indicator), role (dropdown), enabled (toggle, default: true)
- On submit: `POST /admin/users`
- Success: toast "User '{username}' created" + row appears in table
- Validation: username must be unique (409 → "A user with this username already exists"), password strength requirements shown
- `data-testid`: `user-form-username-input`, `user-form-email-input`, `user-form-password-input`, `user-form-role-select`, `user-form-enabled-toggle`, `user-form-submit-btn`

**C13.3 — Edit a user**
- Click user row → edit modal. Username is read-only. Password field is optional ("leave blank to keep current").
- On submit: `PUT /admin/users/{id}`
- `data-testid`: `user-edit-btn`, `user-form-save-btn`

**C13.4 — Delete a user**
- Confirmation dialog requiring typing the username
- On confirm: `DELETE /admin/users/{id}`
- Cannot delete yourself
- `data-testid`: `user-delete-btn`

**C13.5 — Role permission visibility**
- admin users cannot see or modify super_admin users
- Role dropdown for admin users only shows: admin, developer, readonly (not super_admin)

---

## Feature: Manage API Keys

**Phase 4**

### User Story

> As a platform engineer, I want to create API keys for services and CI/CD pipelines, with appropriate roles and expiration, and rotate or revoke them when needed.

### Capabilities

**C14.1 — List all API keys**
- Table: name, key prefix, role, owner, created, expires, status (active/revoked/expired)
- Expired keys shown with visual indicator
- `data-testid`: `apikeys-list-table`, `apikeys-create-btn`

**C14.2 — Create an API key**
- Modal: name (required), role (dropdown), expiration (presets: 30d, 90d, 1y, no expiry; or custom date picker)
- On submit: `POST /admin/apikeys`
- **One-time key display:** After creation, show the full key in a prominent highlighted box with copy button. Warning: "This key will only be shown once. Copy it now and store it securely."
- The modal stays open until the user explicitly dismisses it
- `data-testid`: `apikey-form-name-input`, `apikey-form-role-select`, `apikey-form-expiry-select`, `apikey-form-submit-btn`, `apikey-created-key-display`, `apikey-copy-btn`

**C14.3 — Revoke an API key**
- Confirmation: "Revoke key '{name}'? Services using this key will immediately lose access."
- Calls `POST /admin/apikeys/{id}/revoke`
- `data-testid`: `apikey-revoke-btn`

**C14.4 — Rotate an API key**
- Confirmation: "Rotating creates a new key and immediately revokes the old one. Services using the old key will lose access."
- Calls `POST /admin/apikeys/{id}/rotate`
- Shows the new key (one-time display, same as creation)
- `data-testid`: `apikey-rotate-btn`

**C14.5 — Delete an API key**
- Permanently removes the key record
- Calls `DELETE /admin/apikeys/{id}`
- `data-testid`: `apikey-delete-btn`

---

## Feature: Self-Service Account Management

**Phase 4**

### User Story

> As any user, I want to change my own password and manage my own API keys without needing an admin to do it for me.

### Capabilities

**C15.1 — View my profile**
- Shows: username, email, role (read-only), auth method (local/LDAP/OIDC)
- `data-testid`: `profile-info`

**C15.2 — Change my password**
- Form: current password, new password (with strength indicator), confirm new password
- Calls `POST /me/password`
- Only shown for local-auth users (LDAP/OIDC users manage passwords at their IdP)
- `data-testid`: `profile-change-password-section`, `profile-current-password-input`, `profile-new-password-input`, `profile-confirm-password-input`, `profile-password-submit-btn`

**C15.3 — Manage my own API keys**
- Same UI as admin API key management but filtered to keys owned by the current user
- Any user can create API keys scoped to their own role level (a developer can't create an admin key)
- `data-testid`: `my-apikeys-table`, `my-apikeys-create-btn`

---

# DOMAIN: Navigation & System Information

## Feature: Application Shell & Navigation

**Phase 1**

### User Story

> As any user, I want a clear, consistent navigation structure so that I can efficiently move between different areas of the registry.

### Capabilities

**C16.1 — Sidebar navigation**
- Collapsible sidebar with sections:
  - **Schemas**: Subjects, Schema Browser
  - **Configuration** (admin+ only): Compatibility, Modes
  - **Administration** (admin+ only): Users, API Keys, Import
  - **Account**: My Profile, My API Keys
  - **System**: About
- Active page highlighted
- Sections hidden based on role (see C1.7)
- `data-testid`: `nav-sidebar`, `nav-subjects-link`, `nav-schemas-link`, `nav-config-link`, `nav-modes-link`, `nav-users-link`, `nav-apikeys-link`, `nav-import-link`, `nav-profile-link`, `nav-my-apikeys-link`, `nav-about-link`

**C16.2 — Top bar**
- AxonOps logo + "Schema Registry" title
- Right side: username display, sign out action
- `data-testid`: `nav-topbar`, `nav-user-display`

**C16.3 — Status bar (footer)**
- Shows: server version, storage backend type, total subjects count, total schemas count
- Updates on navigation (from data already loaded by the app)
- `data-testid`: `status-bar`, `status-version`, `status-storage-type`, `status-subject-count`, `status-schema-count`

**C16.4 — Responsive sidebar**
- Desktop: sidebar always visible, can be collapsed to icon-only
- Tablet: sidebar collapsed by default, toggleable
- Mobile: hamburger menu, sidebar as overlay

---

## Feature: Command Palette (Quick Search)

**Phase 5**

### User Story

> As a power user managing hundreds of subjects, I want to press Ctrl+K and instantly jump to any subject, schema, or page without clicking through navigation.

### Capabilities

**C17.1 — Open command palette with Ctrl+K / Cmd+K**
- Modal overlay with search input, auto-focused
- Results grouped by type: Subjects, Schemas (by ID), Pages (Config, Users, etc.)
- Arrow keys to navigate, Enter to select, Escape to close
- `data-testid`: `command-palette`, `command-palette-input`

**C17.2 — Search results from subjects and schema IDs**
- Searches subject names (substring match)
- Searches schema IDs (prefix match on numeric input)
- Shows top 10 results, updates as user types

---

## Feature: About / Cluster Information

**Phase 1**

### User Story

> As any user, I want to see what version of the registry is running, what storage backend is in use, and basic statistics so that I can verify my environment.

### Capabilities

**C18.1 — Display system information**
- Server version + commit (from `GET /v1/metadata/version`)
- Cluster ID (from `GET /v1/metadata/id`)
- Supported schema types (from `GET /schemas/types`)
- Storage backend type
- Auth methods enabled
- Total schemas / subjects counts
- `data-testid`: `about-version`, `about-cluster-id`, `about-schema-types`, `about-storage-type`

---

# CROSS-CUTTING CONCERNS

## Notifications & Feedback

- **Success toast**: green, auto-dismiss after 5s (`data-testid="toast-success"`)
- **Error toast**: red, persistent until dismissed (`data-testid="toast-error"`)
- **Warning toast**: yellow, auto-dismiss after 8s (`data-testid="toast-warning"`)
- All mutating actions (register, delete, config change) produce a toast

## Confirmation Dialogs

- All destructive actions require confirmation (`data-testid="confirm-dialog"`)
- Permanent deletes require typing the resource name (`data-testid="confirm-dialog-name-input"`)
- Cancel and Confirm buttons (`data-testid="confirm-dialog-cancel-btn"`, `data-testid="confirm-dialog-confirm-btn"`)

## Loading, Empty, and Error States

Every data view has three states:
- **Loading**: skeleton/shimmer placeholders (`data-testid="{view}-loading"`)
- **Empty**: friendly message with guidance and CTA if applicable (`data-testid="{view}-empty"`)
- **Error**: error message with retry button (`data-testid="{view}-error"`)

## Error Handling

API errors are mapped to user-friendly messages:
- `401` → redirect to login
- `403` → "You don't have permission to perform this action"
- `404` → "This resource was not found"
- `409` → "This resource already exists" (with link to existing)
- `422` → show validation errors from the API response body
- `429` → "Rate limited. Please wait and try again." (respect `Retry-After`)
- `500` → "Something went wrong. Please try again."

## Keyboard Navigation

- Tab order follows logical reading order
- Escape closes modals/drawers
- Enter submits focused forms
- Ctrl/Cmd+K opens command palette (Phase 5)

## Accessibility

- All interactive elements reachable via keyboard
- ARIA labels on icon-only buttons
- Color is never the sole indicator of state (always paired with text or icon)
- Minimum contrast ratio 4.5:1
- Form validation errors announced to screen readers
- Focus management on modal open/close

## Dark Mode

- Follow OS/system preference via Tailwind's `dark:` variants
- No manual toggle in Phase 1 (add in Phase 5 if desired)

---

# TECHNICAL FOUNDATION

## Frontend Stack

| Concern | Choice | Rationale |
|---------|--------|-----------|
| Framework | React 18+ with TypeScript | Strong ecosystem, widely understood |
| Build | Vite | Fast dev builds, clean production output for Go embed |
| Styling | Tailwind CSS | Utility-first, small bundle, dark mode support |
| Components | shadcn/ui | Accessible primitives, Tailwind-native |
| Code editor | Monaco Editor | Syntax highlighting, diff, validation for Avro/Protobuf/JSON |
| Data fetching | TanStack Query (React Query) | Caching, refetching, optimistic updates |
| Routing | React Router v6+ | Nested routes, lazy loading |
| Icons | Lucide React | Consistent, lightweight |

## Serving Model

- Go binary embeds pre-built SPA via `embed` directive
- All UI assets served under `/ui/`
- SPA makes API calls to same origin (no CORS)
- Config:

```yaml
ui:
  enabled: false     # opt-in
  base_path: "/ui"
```

## BDD Test Instrumentation

- Every interactive element carries a `data-testid` attribute
- Convention: `data-testid="{area}-{element}-{qualifier}"`
- State indicators: `*-loading`, `*-empty`, `*-error`
- Removing/renaming a `data-testid` is a breaking change requiring step definition updates

## Test File Structure

```
tests/e2e/
├── features/                    # Gherkin .feature files
│   ├── auth/
│   │   ├── login.feature
│   │   └── session.feature
│   ├── browsing/
│   │   ├── subjects-list.feature
│   │   ├── subject-detail.feature
│   │   ├── schema-version.feature
│   │   └── schema-browser.feature
│   ├── authoring/
│   │   ├── register-schema.feature
│   │   ├── evolve-schema.feature
│   │   └── delete-schema.feature
│   ├── governance/
│   │   ├── compatibility.feature
│   │   ├── modes.feature
│   │   └── import.feature
│   ├── admin/
│   │   ├── user-management.feature
│   │   ├── apikey-management.feature
│   │   └── self-service.feature
│   └── navigation/
│       └── navigation.feature
├── steps/                       # TypeScript step definitions
├── pages/                       # Page Object classes
├── support/
│   ├── world.ts                 # Cucumber World with Playwright
│   ├── hooks.ts                 # Before/After hooks
│   └── api-helpers.ts           # Direct API calls for test data seeding
└── playwright.config.ts
```

## Test Data Seeding

Tests seed data via direct API calls (not through the UI) for speed:
- `createSubject(name, schema, type)` → registers a schema under a subject
- `createUser(username, password, role)` → creates a test user
- `createApiKey(name, role)` → creates a test API key
- `setCompatibility(subject, level)` → sets compatibility level
- `setMode(subject, mode)` → sets mode
- `cleanupAll()` → cleans up all test data (called in After hooks)

## Test Environment

```yaml
# docker-compose.test.yml
services:
  schema-registry:
    image: axonops-schema-registry:test
    ports:
      - "8081:8081"
    environment:
      STORAGE_TYPE: memory
      ADMIN_PASSWORD: test-admin-password
      UI_ENABLED: "true"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/"]
      interval: 2s
      timeout: 5s
      retries: 10
```

Tests run against `http://localhost:8081/ui/` with Playwright in headless mode.

---

# PHASE SUMMARY

## Phase 1 — See & Sign In
- F: Sign In (C1.1–C1.7)
- F: Application Shell & Navigation (C16.1–C16.4)
- F: Browse Subjects (C2.1–C2.6)
- F: Inspect a Subject (C3.1–C3.4)
- F: Inspect a Schema Version (C4.1–C4.6)
- F: Search Schemas by Global ID (C6.1–C6.2)
- F: About / Cluster Info (C18.1)

## Phase 2 — Author & Evolve
- F: Register a New Schema (C7.1–C7.5)
- F: Evolve an Existing Schema (C8.1–C8.3)
- F: Compare Schema Versions (C5.1–C5.3)
- F: Delete Schemas (C9.1–C9.4)

## Phase 3 — Govern
- F: Manage Compatibility Levels (C10.1–C10.4)
- F: Manage Modes (C11.1–C11.2)
- F: Import Schemas (C12.1–C12.3)

## Phase 4 — Administer
- F: Manage Users (C13.1–C13.5)
- F: Manage API Keys (C14.1–C14.5)
- F: Self-Service Account Management (C15.1–C15.3)

## Phase 5 — Delight
- F: Sign In with SSO (C1.8–C1.9)
- F: Command Palette (C17.1–C17.2)
- Schema Templates
- Schema Evolution Timeline visualisation
- Schema Statistics Dashboard
- Bulk operations (multi-select subjects)
- Export/Download (full registry dump)
- Dark mode toggle
