# Built-in Web UI for AxonOps Schema Registry

## Issue #273 — Comprehensive Requirements & Implementation Plan

> **Repository:** https://github.com/axonops/axonops-schema-registry
> **Wireframes:** See the attached `axonops-ui-wireframes.jsx` React artifact for interactive mockups of all key pages (Dashboard, Subject Explorer, Schema Detail with Diff, Admin Config, API Docs).

---

## Sub-Issues

| Phase | Issue |
|-------|-------|
| Phase 0: Prerequisites | #277 |
| Phase 1: Foundation | #278 |
| Phase 2: Schema Operations | #279 |
| Phase 3: Diff, Search & Downloads | #280 |
| Phase 4: Contexts & Enterprise | #281 |
| Phase 5: Auth, Admin & Audit | #282 |
| Phase 6: Polish & Adoption | #283 |

---

## 1. Overview

Add an embedded, production-grade web UI to the AxonOps Schema Registry. The UI is bundled into the Go single binary using `//go:embed`, served alongside the existing REST API, and tested via BDD with Playwright and Cucumber.

**Goal:** A rich, branded interface covering every schema registry feature — schema management, compatibility, contexts, data contracts, CSFLE encryption, exporters, auth, RBAC, audit, and administration — so users never need `curl` or external tools.

**Non-goal:** This UI is for **using and administering the schema registry**. It is NOT a monitoring/alerting dashboard. Monitoring, metrics visualisation, and alerting are handled by the AxonOps platform. The UI will not include metrics charts, Prometheus dashboards, or alerting configuration.

---

## 2. Go 1.26 Upgrade

Go 1.26 was released on February 10, 2026. **We should upgrade to Go 1.26 as part of this work**, ideally as a pre-requisite before starting the UI feature. Key benefits:

| Feature | Benefit for This Project |
|---------|------------------------|
| **Green Tea GC (default)** | 10-40% lower GC overhead. The UI adds embedded assets and more concurrent requests — Green Tea helps keep latency flat. |
| **`new(expr)` syntax** | Cleaner code in the new session/auth endpoints. `new(sessionTimeout)` instead of the `ptr()` helper pattern. |
| **`go fix` modernizers** | Run `go fix ./...` on the codebase to adopt Go 1.26 idioms before the UI work begins. Free code quality improvement. |
| **Goroutine leak profiler** | Enable `GOEXPERIMENT=goroutineleak` in CI. The UI session management and SSE connections are prime candidates for goroutine leaks. Catch them early. |
| **~30% cgo overhead reduction** | If any storage backend drivers use cgo (e.g., certain PostgreSQL or MySQL drivers), free performance gain. |
| **Self-referential generics** | Useful for generic typed API response wrappers in the new UI backend endpoints. |
| **Stack-allocated slice backing** | More slices on the stack → less GC pressure for the many small allocations in request handling. |

**Action:** Create a separate PR to upgrade `go.mod` to Go 1.26, run `go fix ./...`, update CI images, and verify all existing tests pass. Then begin UI work on Go 1.26.

---

## 3. Frontend Technology Approach

### 3.1 Recommended Stack: Embedded SPA (React/Vite + go:embed)

After evaluating Go+Templ+HTMX (the "GoTTH" stack) vs embedded SPA, **embedded SPA is recommended** because:

1. **Monaco Editor is non-negotiable** — schema editing, syntax highlighting, diff viewing, and search-in-schema are fundamentally JavaScript-heavy. With HTMX they'd need to be bolted on as islands, defeating the simplicity benefit.
2. **Proven pattern** — Traefik, MinIO, Consul, Gitea all embed SPAs in Go binaries. Well-trodden path.
3. **Ecosystem maturity** — React+TypeScript has the richest component ecosystem and tooling support.
4. **Rich interactivity** — drag-and-drop tag management, real-time schema validation, inline diff annotations, keyboard shortcuts — all first-class in React.

| Layer | Technology |
|-------|-----------|
| Framework | React 18+ / TypeScript |
| Build | Vite |
| Styling | Tailwind CSS + Shadcn/ui |
| Code editor | Monaco Editor |
| Diff | Monaco Diff Editor |
| State | TanStack Query |
| Routing | React Router v6 (hash mode) |
| Testing | Playwright + Cucumber.js |

### 3.2 Architecture

- Build frontend into `web/dist/`, embed with `//go:embed web/dist/*`
- Serve at `/ui/` with SPA fallback; redirect `/` → `/ui/` for browser requests
- API at existing paths unchanged
- `--disable-ui` flag / `UI_ENABLED=false` for headless deployments
- Hash routing (`/#/subjects`) avoids API path conflicts
- `make dev-ui` runs Go backend + Vite dev server concurrently
- `make build` produces single binary with embedded UI

---

## 4. Authentication, Sessions & Security

*(This section is detailed because it's the most architecturally complex part. The existing API has 6 auth methods, and the browser introduces a session layer on top.)*

### 4.1 The Core Problem

The REST API uses **stateless, per-request auth** (each `curl` includes credentials). A browser UI cannot work this way — storing passwords in JS memory is insecure, mTLS is browser-level, OIDC requires redirect flows.

**Solution:** A session layer. The UI authenticates once → receives a JWT session cookie → uses it for subsequent calls. The backend translates session → user identity + role.

### 4.2 Dual Auth Path

The server has TWO auth paths. Priority order:
1. `Authorization` header (Basic/Bearer) or API key header → traditional per-request auth (for `curl`, Kafka clients, CI/CD)
2. Session cookie + `X-Requested-With: SchemaRegistryUI` header → session auth (for UI)
3. Neither → 401

**No existing integration is affected.**

### 4.3 Per-Method Login Flows

| Auth Method | UI Flow |
|-------------|---------|
| **None** | No login; UI loads directly |
| **Basic Auth** | Login form → POST `/api/v1/auth/login` → JWT cookie |
| **API Keys** | Login form → POST with key → JWT cookie (key never stored client-side) |
| **LDAP/AD** | Login form → server binds LDAP → maps groups to roles → JWT cookie |
| **OIDC** | "Login with SSO" → redirect to IdP → callback → JWT cookie. PKCE mandatory. |
| **mTLS** | Transparent (browser cert at TLS layer). Identity from cert CN/SANs. |

### 4.4 Security Mitigations

**CSRF:** `SameSite=Strict` cookie + `X-Requested-With` custom header + Go 1.26 `Sec-Fetch-Site` check. No traditional CSRF tokens needed.

**XSS:** CSP headers on `/ui/*`, `HttpOnly` session cookie, React auto-escaping. Schema content displayed in Monaco (safe) but field names/descriptions in HTML must be escaped.

**OIDC:** PKCE mandatory, state parameter, nonce validation, IdP tokens stored SERVER-SIDE only. Browser never sees them.

**Session:** JWT in `HttpOnly`/`Secure`/`SameSite=Strict` cookie. Server-side session store (by `jti`) for revocation on logout. Configurable timeout.

### 4.5 RBAC in UI

| Role | Can Do | Cannot Do |
|------|--------|-----------|
| **readonly** | Browse everything | All mutations hidden (not disabled) |
| **developer** | Register schemas, create subjects, check compat, manage tags | Delete, change config/mode, admin |
| **admin** | Everything above + delete, config, mode, encryption, exporters, view server config | Manage super_admins |
| **super_admin** | Everything | — |

### 4.6 New Auth Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/auth/login` | POST | Authenticate, return JWT cookie |
| `/api/v1/auth/logout` | POST | Invalidate session |
| `/api/v1/auth/me` | GET | Current user identity/role/perms |
| `/api/v1/auth/refresh` | POST | Refresh expiring token |
| `/api/v1/auth/oidc/authorize` | GET | Start OIDC flow |
| `/api/v1/auth/oidc/callback` | GET | OIDC callback |

---

## 5. New Feature: Schema Tags

**This is a new backend + UI feature.** Tags allow users to categorise and filter schemas with custom labels.

### 5.1 Concept

- Tags are arbitrary string labels attached to **subjects** (e.g., `production`, `staging`, `deprecated`, `encrypted`, `pii`, `team-payments`)
- A subject can have 0-N tags
- Tags are searchable and filterable in the UI
- Tags are stored alongside subject metadata in the storage backend
- Tags have no effect on schema compatibility or validation — they are purely organisational

### 5.2 New API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/subjects/{subject}/tags` | GET | List tags for a subject |
| `/api/v1/subjects/{subject}/tags` | PUT | Set tags (replaces all) `{"tags": ["production", "pii"]}` |
| `/api/v1/subjects/{subject}/tags/{tag}` | PUT | Add a single tag |
| `/api/v1/subjects/{subject}/tags/{tag}` | DELETE | Remove a single tag |
| `/api/v1/tags` | GET | List all unique tags across all subjects (for autocomplete) |
| `/api/v1/tags/{tag}/subjects` | GET | List all subjects with a given tag |

### 5.3 UI for Tags

- **Subject Explorer**: Tags shown as coloured badges per subject. Filter dropdown with all available tags. Multi-select filtering.
- **Subject Detail**: Tag editor — add/remove tags inline. Autocomplete from existing tags. Colour-coded badges.
- **Search**: Search by tag. "Show all subjects tagged `production`" as a one-click filter.
- **Bulk tagging**: Select multiple subjects in explorer → apply tags in bulk (admin+).
- Tags `deprecated` and `pii` could have special visual treatment (red badge, warning icon).

### 5.4 Storage

Tags stored as a JSON array or junction table in the storage backend:
- **PostgreSQL/MySQL**: `subject_tags` table with `(subject_name, context, tag)` or JSON column on subjects
- **Cassandra**: Collection column or separate table with SAI index
- **Memory**: Map in the subject struct

---

## 6. Schema Downloads & Export

### 6.1 Individual Schema Download

- **Download button** on every schema view → downloads the schema as a file
  - Avro: `{subject}-v{version}.avsc` (JSON)
  - Protobuf: `{subject}-v{version}.proto`
  - JSON Schema: `{subject}-v{version}.json`
- **Copy to clipboard** button (JSON or proto IDL)
- **Copy Schema ID** button

### 6.2 Bulk Downloads

- **All versions of a subject** → ZIP file containing every version as individual files
  - Download button in subject detail sidebar: "⬇ All Versions (.zip)"
  - Naming: `users-value/v1.avsc`, `users-value/v2.avsc`, etc.
- **All subjects in a context** → ZIP file with one folder per subject, each containing all versions
  - Download button in Subject Explorer header: "⬇ Export All"
- **Filtered export** → download only the subjects matching current filters/tags
  - "⬇ Export Filtered" when filters are active
- **Schema diff export** → download a unified diff as a `.patch` or `.diff` file

### 6.3 New API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/subjects/{subject}/versions/{version}/download` | GET | Download schema as file with correct Content-Type and filename |
| `/api/v1/subjects/{subject}/download` | GET | Download all versions as ZIP |
| `/api/v1/export` | POST | Bulk export (accepts filter params) as ZIP |

---

## 7. Enhanced Search

### 7.1 Global Search Bar

A global search bar in the top navigation — always accessible, keyboard shortcut `Ctrl+K` / `Cmd+K` to focus.

### 7.2 Search Modes

- **Subject search**: Prefix matching on subject names (`GET /subjects?subjectPrefix=`)
- **Schema ID lookup**: Enter a numeric ID → resolves to schema content and associated subjects
- **Tag search**: `tag:production` → shows all subjects with that tag
- **Field name search**: Search for schemas containing a specific field name (e.g., "email", "customer_id"). Client-side search across fetched schemas, or new backend endpoint.
- **Type search**: Filter by Avro, Protobuf, JSON Schema
- **Cross-context**: Toggle to search across all contexts or only current
- **Combined**: `type:avro tag:production email` → Avro schemas tagged production containing "email" field

### 7.3 Search Results

- Results grouped by type: Subjects, Schemas, Tags
- Each result shows: subject name, schema type badge, version count, tags, context
- Click to navigate to subject detail
- Keyboard navigable (arrow keys, Enter to select)

### 7.4 New Backend Endpoint (Optional)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/search` | GET | Full-text search across subjects, schema content, tags. Query params: `q`, `type`, `tag`, `context`, `field` |

If backend search is deferred, client-side search can work for registries with <5000 schemas by fetching and indexing locally.

---

## 8. Embedded API Documentation

### 8.1 Concept

The registry already generates OpenAPI docs and has Swagger UI at `/docs`. The web UI should integrate this documentation for a unified experience.

### 8.2 Implementation

- **API Docs page** (`/#/api-docs`) in the UI navigation under a "Developer" section
- **ReDoc renderer**: Embed ReDoc (MIT licensed, ~200KB) to render the bundled `openapi.yml` beautifully
  - ReDoc provides a three-panel layout: nav, content, code samples
  - Better reading experience than Swagger UI for documentation browsing
- **Swagger UI link**: Button to open the existing `/docs` Swagger UI in a new tab for interactive API testing
- **Download OpenAPI spec**: Button to download the raw `openapi.yml` file
- **Version display**: Show the API version from the spec

### 8.3 UI Integration

The API docs page should feel native to the UI — same nav, same theming (dark mode support), same branding. ReDoc supports custom theme colours, so it should use the same accent colour as the rest of the UI.

Since the `openapi.yml` is already embedded in the Go binary (for the existing Swagger UI), the ReDoc page can fetch it from the same endpoint and render it client-side.

---

## 9. UI Pages — Complete Specification

### 9.1 Dashboard (`/#/`)

Overview cards: total subjects, total schemas, global compat, mode, active contexts, auth method. Recent activity feed (from audit log if enabled, or from API polling). System info sidebar: storage, auth, features, uptime. Quick-access to recently viewed subjects (localStorage). READONLY/IMPORT mode banner on all pages.

*See wireframe: Dashboard page.*

### 9.2 Context Switcher (Nav)

Dropdown in sidebar. Lists all contexts. Scopes all views. Persisted in URL hash. Context-prefixed API calls.

### 9.3 Subject Explorer (`/#/subjects`)

Searchable, sortable, filterable table. Columns: Name, Type (badge), Version, Compat, **Tags** (coloured badges), Modified. Filters: type, compat, **tags** (multi-select), deleted status. **Download buttons**: per-subject and bulk export. "Create New Subject" (developer+). Pagination/virtual scroll.

*See wireframe: Subject Explorer page.*

### 9.4 Subject Detail (`/#/subjects/{name}`)

Version sidebar with all versions. Schema viewer (Monaco, syntax highlighted). **Tabs**: Schema, Tree View, Diff, Data Contract, **History**. **Version diff selector** in sidebar: pick any two versions, click "View Diff". **Tags editor**: inline tag management. **Download options**: single version, all versions ZIP, copy to clipboard, copy schema ID. **Details panel**: compat level, schema ID, references, tags, encryption status. **Actions** (role-gated): Register New Version, Change Compat, Delete (soft/hard), Undelete.

**Schema History / Lifecycle Timeline:**
- **Timeline view** showing the full evolution of a subject's schema across all versions
- Each version displayed as a node on the timeline with: version number, schema ID, registration date, schema type
- **Change summary** between consecutive versions: fields added (green), fields removed (red), fields modified (yellow), type changes
- **Compatibility status** at each transition: whether the change was backward/forward/fully compatible
- **Data contract changes**: show when rules were added, modified, or removed between versions
- **Metadata changes**: track when tags, compatibility level, or mode were changed
- Click any version node to view its full schema; click between two nodes to see the diff
- **Visual indicators**: breaking changes highlighted, encryption rules added/removed, soft-deleted versions shown as greyed-out
- Filter timeline by date range, change type (field add/remove/modify), or compatibility impact
- Export timeline as a change log (JSON or markdown)

*See wireframe: Schema Detail page.*

### 9.5 Schema Diff (`/#/subjects/{name}/diff/{v1}/{v2}`)

Side-by-side diff (Monaco Diff Editor). Added lines green, removed lines red. **Compatibility analysis panel** below the diff showing which changes are safe/breaking under the current compat mode, with per-field explanations. Version selectors to compare any two versions. **Download diff** as `.diff` file.

*See wireframe: Schema Diff page.*

### 9.6 Schema Editor (`/#/subjects/{name}/register` or `/#/subjects/new`)

Monaco Editor with schema type selector, syntax highlighting, real-time validation, line numbers. "Check Compatibility" button with inline result (✅/❌ + field-level detail). "Register" button. Schema reference support. Data contract metadata/rules editor (collapsible). **Tag assignment** on new subject creation.

### 9.7 Schema Search (`/#/search`)

Global search bar (`Ctrl+K`). Modes: subject prefix, schema ID, tag, field name, type. Cross-context toggle. Grouped results with navigation.

### 9.8 Compatibility Manager (`/#/compatibility`)

Global view/edit (admin+). Per-subject overrides table. Reference card explaining all 7 modes.

### 9.9 Mode Manager (`/#/mode`) — Admin Only

View and toggle READWRITE/READONLY/IMPORT with confirmation dialogs. Context-specific mode.

### 9.10 DEK Registry / Field-Level Encryption (`/#/encryption`)

**Client-Side Field Level Encryption (CSFLE)** management for protecting sensitive data at the field level.

**KEK (Key Encryption Key) Management:**
- List all KEKs with name, KMS type, KMS key ID, shared status, creation date
- Create new KEK: name, KMS type (hcvault, aws-kms, azure-kms, gcp-kms), KMS key ID, shared flag, KMS properties
- KEK detail view: all metadata, list of associated DEKs, usage count
- Update KEK: doc, shared flag, KMS properties
- Delete/undelete KEK with confirmation
- Test KEK connectivity to KMS provider

**DEK (Data Encryption Key) Management:**
- List DEKs per KEK, showing subject, version, algorithm, encrypted key material (truncated)
- Create DEK: subject, algorithm (AES256_GCM, AES128_GCM, AES256_SIV), optional version
- DEK detail: full metadata, version history, creation timestamp
- Delete/undelete DEK with version support
- Version history viewer for rotated DEKs

**Dashboard Integration:**
- Lock/shield icon on subjects that use ENCRYPT rules in the Subject Explorer
- "Encrypted Fields" badge in Subject Detail showing which fields have `confluent:tags` with encryption
- KMS connection status indicator (green/red) for each configured provider

**Endpoints used:** `GET/POST /dek-registry/v1/keks`, `GET/PUT/DELETE /dek-registry/v1/keks/{name}`, `GET/POST /dek-registry/v1/keks/{name}/deks`, etc.

### 9.11 Exporter Manager (`/#/exporters`)

Exporter list with status. Detail view. CRUD. Lifecycle controls (pause/resume/reset). Admin+ only for mutations.

### 9.12 Data Contracts (`/#/data-contracts`)

**Data Contracts** enable schema-level business rules and data quality enforcement. The UI provides full management of the three-layer rule system.

**Rule Set Viewer/Editor:**
- View `ruleSet` attached to any schema version — domain rules with name, kind, type, mode, expression, tags, parameters
- Supported rule kinds: `CONDITION` (validate), `TRANSFORM` (modify), with `onFailure` action
- Supported rule types: `CEL` (whole-message conditions), `CEL_FIELD` (per-field conditions/transforms), `ENCRYPT` (field-level encryption), `JSONATA` (migration transforms)
- Rule expression editor with syntax highlighting for CEL and JSONata
- Mode selector: WRITE, READ, WRITEREAD
- Tags selector: link rules to specific field tags (e.g., `PII`, `SENSITIVE`)
- onFailure selector: ERROR, NONE, DLQ

**Three-Layer Rule Merge Visualisation:**
- **Schema-level rules** (ruleSet): rules attached directly to the schema version
- **Default rules** (defaultRuleSet): inherited from subject config, applied when schema has no rules
- **Override rules** (overrideRuleSet): always applied from subject config, cannot be overridden by schema
- Visual merge view showing which rules come from which layer, with inheritance indicators
- Colour-coded: schema rules (blue), default rules (grey), override rules (orange)

**Metadata Editor:**
- View/edit schema metadata (key-value pairs) and metadata tags
- Metadata properties: arbitrary key-value pairs attached to schema versions
- Metadata tags: string tags for schema classification (different from subject tags)

**Data Contract Dashboard:**
- List of all subjects with active data contracts (any subject with ruleSet, defaultRuleSet, or overrideRuleSet)
- Filter by rule type (CEL, ENCRYPT, JSONATA), rule kind (CONDITION, TRANSFORM), tags
- Encryption status per subject: which fields are encrypted, which KEK is used
- Quick view of rule violations (if audit logging is enabled)

**Key Scenarios:**
- Register a schema with CEL CONDITION rules that validate business logic on WRITE
- Attach CEL_FIELD TRANSFORM rules that mask PII fields on READ
- Configure ENCRYPT rules with `confluent:tags` for automatic field-level encryption
- Set up JSONata UPGRADE/DOWNGRADE migration rules between schema versions
- Define default and override rule sets at the subject config level
- View the merged rule set showing all three layers for any schema version

### 9.13 Server Configuration (`/#/admin/config`) — Admin Only

**Full running config** in structured view with **all secrets redacted** (`[REDACTED]`). Sections: Server, Storage, Auth, RBAC, Rate Limiting, Audit, Encryption, Exporters, Contexts, UI. Feature flags summary (✅/❌). Build info, Go version, uptime, goroutines, memory. Health check status for backends. Env var override highlighting.

*See wireframe: Server Configuration page.*

**New endpoints:** `/api/v1/admin/config`, `/api/v1/admin/health`, `/api/v1/admin/info`

### 9.14 API Documentation (`/#/api-docs`)

Embedded ReDoc rendering the bundled `openapi.yml`. Link to Swagger UI at `/docs`. Download OpenAPI spec button. Themed to match the UI.

*See wireframe: API Docs page.*

### 9.15 Audit Log (`/#/audit`) — If Enabled

Paginated, filterable event table. Filter by type, user, subject, context, time range. Event detail modal. Export CSV/JSON.

**New endpoints:** `/api/v1/audit/events`, `/api/v1/audit/events/{id}`

### ~~Metrics Dashboard~~ — REMOVED

**Not included.** Monitoring, metrics, and alerting are handled by the AxonOps platform per product mandate. This UI is for schema management and administration only.

---

## 10. UX Design Direction

### 10.1 Aesthetic

**Dark-first, developer-focused, information-dense.** Think: VS Code meets Linear meets Vercel Dashboard. Clean lines, monospace for data, generous but not wasteful whitespace. The dark theme should be the default and the primary design target. Light mode available but secondary.

### 10.2 Key UX Principles

- **Schema-centric**: The schema is the star. Large, syntax-highlighted code views. Diffs are first-class citizens, not afterthoughts.
- **Context-aware**: The current context is always visible. Switching is instant. The URL always reflects the full state.
- **Keyboard-first**: `Ctrl+K` search, `Ctrl+S` save, `Ctrl+Enter` register. Vim-style `j`/`k` for list navigation. Tab through everything.
- **Zero-config**: Works beautifully out of the box. Branding customisation is optional, not required.
- **Feedback-rich**: Toast notifications, inline validation, skeleton loading states, optimistic UI updates.
- **Information density**: Show data, not chrome. Minimize clicks to get to the schema content. The Subject Explorer → Subject Detail → Schema View should be at most 2 clicks.

### 10.3 Signature UX Features

- **Schema diff with compatibility annotations** — nobody else does this. Show the diff AND tell you if it's safe.
- **Global `Ctrl+K` search** — search everything from anywhere. Subjects, schemas, tags, schema IDs.
- **Inline tag management** — add/remove tags with autocomplete, right on the subject detail page.
- **One-click downloads** — download any schema, any version, all versions, bulk export. No friction.
- **Live syntax validation** — as you type in the editor, validation errors appear inline. No "submit and see".
- **Context switcher** — always visible, scopes everything instantly.

---

## 11. Branding & Theming

AxonOps logo in sidebar. Configurable via config: `ui.branding.logo_url`, `ui.branding.primary_color`, `ui.branding.app_title`. All tokens in CSS variables / Tailwind config. Light/dark mode toggle persisted in localStorage.

---

## 12. All New REST API Endpoints Summary

| Endpoint | Method | Purpose | Auth |
|----------|--------|---------|------|
| `/ui/*` | GET | Serve SPA | Config-dependent |
| `/api/v1/auth/login` | POST | Authenticate | No |
| `/api/v1/auth/logout` | POST | End session | Session |
| `/api/v1/auth/me` | GET | User identity/role | Session |
| `/api/v1/auth/refresh` | POST | Refresh token | Session |
| `/api/v1/auth/oidc/authorize` | GET | OIDC start | No |
| `/api/v1/auth/oidc/callback` | GET | OIDC callback | No |
| `/api/v1/ui/config` | GET | UI config (branding, features, auth) | No |
| `/api/v1/admin/config` | GET | Sanitised server config | Admin+ |
| `/api/v1/admin/health` | GET | Backend health | Admin+ |
| `/api/v1/admin/info` | GET | Build/runtime info | Admin+ |
| `/api/v1/audit/events` | GET | Audit events (paginated) | Admin+ |
| `/api/v1/audit/events/{id}` | GET | Single audit event | Admin+ |
| `/api/v1/subjects/{subject}/tags` | GET | List subject tags | Auth'd |
| `/api/v1/subjects/{subject}/tags` | PUT | Set subject tags | Developer+ |
| `/api/v1/subjects/{subject}/tags/{tag}` | PUT | Add tag | Developer+ |
| `/api/v1/subjects/{subject}/tags/{tag}` | DELETE | Remove tag | Developer+ |
| `/api/v1/tags` | GET | List all unique tags | Auth'd |
| `/api/v1/tags/{tag}/subjects` | GET | Subjects by tag | Auth'd |
| `/api/v1/subjects/{subject}/versions/{v}/download` | GET | Download schema file | Auth'd |
| `/api/v1/subjects/{subject}/download` | GET | Download all versions ZIP | Auth'd |
| `/api/v1/export` | POST | Bulk export (filtered) ZIP | Auth'd |
| `/api/v1/search` | GET | Full-text search | Auth'd |

---

## 13. BDD Testing

### 13.1 Stack & Structure

Playwright + Cucumber.js + TypeScript. Tests in `tests/e2e/`. Page Object Model. In-memory storage. Headless CI.

### 13.2 Feature Files (18 total)

```
tests/e2e/features/
├── dashboard.feature           # Overview cards, activity feed, system info
├── subjects.feature            # List, search, filter, create, delete
├── schema-editor.feature       # Register, validate, compat check
├── schema-viewer.feature       # View, tree view, syntax highlight
├── schema-diff.feature         # Side-by-side, compat annotations, download diff
├── schema-search.feature       # Global search, by ID, by tag, by field
├── schema-download.feature     # Individual, bulk, ZIP, clipboard
├── tags.feature                # Create, edit, delete, filter, bulk
├── compatibility.feature       # Global, per-subject, overrides, reference card
├── mode-management.feature     # READWRITE/READONLY/IMPORT toggle
├── contexts.feature            # Switcher, isolation, URL state
├── data-contracts.feature      # Metadata, rules, 3-layer merge
├── encryption.feature          # KEK/DEK CRUD, KMS status
├── exporters.feature           # CRUD, lifecycle, status
├── authentication.feature      # Basic, API key, OIDC, session, expiry
├── rbac.feature                # Role enforcement, hidden buttons
├── audit-log.feature           # View, filter, export
├── admin-config.feature        # Server config, redaction, health
└── api-docs.feature            # ReDoc renders, download spec
```

### 13.3 Key Scenarios

```gherkin
Feature: Schema Tags
  Scenario: Add tags to a subject
    Given the subject "users-value" exists
    And I am logged in as "developer"
    When I navigate to "users-value"
    And I add the tag "production"
    Then the subject should show the "production" tag badge

  Scenario: Filter subjects by tag
    Given subjects exist with tags:
      | subject         | tags              |
      | users-value     | production, pii   |
      | orders-value    | production        |
      | test-value      | staging           |
    When I filter by tag "production"
    Then I should see 2 subjects

Feature: Schema Downloads
  Scenario: Download a single schema version
    Given the subject "users-value" exists with 3 versions
    When I navigate to "users-value" version 2
    And I click "Download Schema"
    Then a file "users-value-v2.avsc" should be downloaded

  Scenario: Download all versions as ZIP
    Given the subject "users-value" exists with 3 versions
    When I click "Download All Versions"
    Then a ZIP file should be downloaded containing 3 schema files

  Scenario: Bulk export filtered subjects
    Given 10 subjects exist, 3 tagged "production"
    When I filter by tag "production"
    And I click "Export Filtered"
    Then a ZIP should contain schemas for only the 3 filtered subjects

Feature: Schema Diff with Compatibility Annotations
  Scenario: Diff shows breaking changes
    Given "users-value" v5 has field "age" (int)
    And "users-value" v7 does not have field "age"
    And compatibility is "BACKWARD"
    When I view diff between v5 and v7
    Then the removed "age" field should be highlighted red
    And the compatibility panel should show "BREAKING: removed field 'age'"

Feature: Global Search
  Scenario: Search with Ctrl+K
    When I press Ctrl+K
    Then the search bar should be focused
    When I type "users"
    Then I should see matching subjects in the dropdown

Feature: Embedded API Docs
  Scenario: View API documentation
    When I navigate to "API Docs"
    Then I should see the ReDoc-rendered API reference
    And I should see endpoint groups for Schemas, Subjects, Compatibility
    And I should be able to download the OpenAPI spec

Feature: Server Configuration
  Scenario: Secrets are redacted
    Given I am logged in as "admin"
    When I navigate to Server Configuration
    Then database password should show "[REDACTED]"
    And OIDC client secret should show "[REDACTED]"
    And the storage backend type should be visible
```

### 13.4 CI

Two jobs: no-auth and basic-auth. Traces/screenshots on failure. `make test-e2e` target.

---

## 14. Acceptance Criteria

1. All schema operations available in UI (browse, view, register, diff, compat check, download, delete)
2. **Schema tags** — create, edit, delete, filter, bulk tag
3. **Schema downloads** — individual, bulk ZIP, diff export, clipboard
4. **Global search** — by subject, ID, tag, field, type
5. **API docs** — ReDoc embedded, OpenAPI downloadable
6. Auth works for all methods (None, Basic, API Key, LDAP, OIDC, mTLS)
7. RBAC enforced (mutations hidden for unauthorized roles)
8. Multi-tenant contexts navigable with context switcher
9. Schema diff with compatibility annotations
10. DEK Registry and Exporter management functional
11. Server configuration visible to admins (secrets redacted)
12. Audit log viewer functional when enabled
13. All BDD scenarios pass in headless Playwright CI
14. Single binary (`go build` produces API + UI)
15. Branding configurable, dark/light mode
16. Built on **Go 1.26**
17. `--disable-ui` flag works
18. Documentation updated

---

## 15. Implementation Phases

### Phase 0: Prerequisites (Week 0)
- [ ] Upgrade to Go 1.26 (separate PR)
- [ ] Run `go fix ./...` to modernize codebase
- [ ] Enable goroutine leak profiler in CI
- [ ] Implement schema tags backend (`/api/v1/subjects/{subject}/tags`, `/api/v1/tags`)
- [ ] Implement download endpoints (`/download`, `/export`)
- [ ] Implement search endpoint (`/api/v1/search`)

### Phase 1: Foundation (Weeks 1-2)
- [ ] Frontend scaffolding: Vite + React + TypeScript + Tailwind + Shadcn/ui
- [ ] Go `//go:embed` + `/ui/` route + SPA fallback
- [ ] `--disable-ui` flag
- [ ] `/api/v1/ui/config` endpoint
- [ ] Auth: Basic Auth login, `/api/v1/auth/*` endpoints
- [ ] Session management (JWT + HttpOnly cookie + CSRF protection)
- [ ] Dashboard page
- [ ] Sidebar nav with context switcher
- [ ] Dark/light mode
- [ ] BDD harness + CI + first smoke test

### Phase 2: Schema Operations (Weeks 3-4)
- [ ] Subject Explorer (list, search, filter, sort, **tags**, pagination)
- [ ] Subject Detail (versions, viewer, **tags editor**, **download buttons**)
- [ ] Schema registration with Monaco editor
- [ ] Compatibility check with field-level error messages
- [ ] Create new subject form with tag assignment
- [ ] BDD coverage

### Phase 3: Diff, Search & Downloads (Weeks 5-6)
- [ ] Schema diff (Monaco Diff Editor, side-by-side, **compatibility annotations**)
- [ ] **Global search** (`Ctrl+K`, all modes)
- [ ] **Schema downloads** (individual, bulk ZIP, diff export, clipboard)
- [ ] Compatibility Manager (global + per-subject)
- [ ] Mode Manager (admin only)
- [ ] BDD coverage

### Phase 4: Contexts & Enterprise (Weeks 7-8)
- [ ] Context switcher + context-scoped routing
- [ ] DEK Registry / Encryption Manager
- [ ] Exporter Manager
- [ ] Data Contracts viewer/editor
- [ ] BDD coverage

### Phase 5: Auth, Admin & Audit (Weeks 9-10)
- [ ] Full RBAC enforcement
- [ ] Additional auth: LDAP, OIDC (PKCE), API Keys, mTLS
- [ ] **Server Configuration Viewer** (`/api/v1/admin/config` with redaction)
- [ ] Admin panel
- [ ] Audit log viewer
- [ ] **Embedded API Docs** (ReDoc)
- [ ] BDD coverage

### Phase 6: Polish (Weeks 11-12)
- [ ] Branding/theming
- [ ] **Keyboard shortcuts** (Ctrl+K, Ctrl+S, Ctrl+Enter, j/k navigation)
- [ ] Onboarding wizard (empty registry)
- [ ] Accessibility audit (WCAG 2.1 AA)
- [ ] Responsive testing
- [ ] Full BDD regression (Chromium, Firefox, WebKit)
- [ ] Documentation + screenshots
- [ ] Performance testing

---

## 16. Config Additions

```yaml
ui:
  enabled: true
  branding:
    logo_url: ""
    primary_color: "#3b82f6"
    app_title: "AxonOps Schema Registry"
  session:
    timeout: "30m"
    secret: ""  # auto-generated if empty
  security:
    csp_enabled: true
```

---

## 17. Non-Functional Requirements

- API latency: no measurable impact from UI embed
- Frontend bundle: < 2MB gzipped
- Binary size increase: < 5MB
- Browsers: latest 2 of Chrome, Firefox, Safari, Edge
- Initial load: < 2s on localhost
- WCAG 2.1 AA
- Go 1.26 minimum
