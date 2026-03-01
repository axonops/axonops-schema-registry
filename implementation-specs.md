# AxonOps Schema Registry — Web UI Implementation Specifications

> **Companion to:** `ux-requirements.md` — this document provides the implementation-level detail for each feature: page layouts, exact API calls, form field specs, TypeScript types, Monaco Editor configuration, and comprehensive BDD feature files.

---

## URL Route Table

Every route in the SPA, the component it renders, who can access it, and what API calls it makes on load.

| Route | Component | Access | API calls on mount | Phase |
|-------|-----------|--------|-------------------|-------|
| `/ui/login` | `LoginPage` | Anyone (unauthenticated) | `GET /ui/auth/config` | 1 |
| `/ui/subjects` | `SubjectsListPage` | Any authenticated | `GET /subjects?subjectPrefix=&deleted=false` | 1 |
| `/ui/subjects/:subject` | `SubjectDetailPage` | Any authenticated | `GET /subjects/{subject}/versions`, `GET /config/{subject}`, `GET /mode/{subject}` | 1 |
| `/ui/subjects/:subject/versions/:version` | `SchemaVersionPage` | Any authenticated | `GET /subjects/{subject}/versions/{version}`, `GET /subjects/{subject}/versions/{version}/referencedby` | 1 |
| `/ui/subjects/:subject/register` | `RegisterSchemaPage` | developer+ | `GET /subjects` (for reference autocomplete) | 2 |
| `/ui/schemas` | `SchemaBrowserPage` | Any authenticated | `GET /schemas?latestOnly=true&limit=50` | 1 |
| `/ui/schemas/:id` | `SchemaByIdPage` | Any authenticated | `GET /schemas/ids/{id}`, `GET /schemas/ids/{id}/subjects`, `GET /schemas/ids/{id}/versions` | 1 |
| `/ui/config` | `GlobalConfigPage` | admin+ | `GET /config`, `GET /subjects` (to show overrides) | 3 |
| `/ui/modes` | `GlobalModesPage` | admin+ | `GET /mode`, `GET /subjects` | 3 |
| `/ui/import` | `ImportPage` | admin+ | `GET /mode` (check import mode) | 3 |
| `/ui/admin/users` | `UserManagementPage` | admin+ | `GET /admin/users` | 4 |
| `/ui/admin/users/:id` | `UserDetailPage` | admin+ | `GET /admin/users/{id}` | 4 |
| `/ui/admin/apikeys` | `ApiKeyManagementPage` | admin+ | `GET /admin/apikeys` | 4 |
| `/ui/account` | `MyProfilePage` | Any authenticated | `GET /me` | 4 |
| `/ui/account/apikeys` | `MyApiKeysPage` | Any authenticated | `GET /admin/apikeys` (filtered to own) | 4 |
| `/ui/about` | `AboutPage` | Any authenticated | `GET /v1/metadata/version`, `GET /v1/metadata/id`, `GET /schemas/types` | 1 |

**Route guards:** The router wraps protected routes in an `AuthGuard` component that checks for a valid token. If no token, redirect to `/ui/login?redirect={originalPath}`. Role-based guards use a `RoleGuard` component that checks `token.role` and renders a 403 page if insufficient.

---

## TypeScript API Response Types

Define these in `ui/src/types/api.ts`. Every API call in the client layer must be typed.

```typescript
// ── Auth ──

export interface AuthConfig {
  methods: ('basic' | 'api_key' | 'oidc')[];
  ldap_enabled: boolean;
  oidc?: {
    display_name: string;
    login_url: string;
  };
}

export interface AuthResponse {
  token: string;
  expires_at: string;       // ISO 8601
  user: AuthUser;
}

export interface AuthUser {
  username: string;
  email: string;
  role: 'super_admin' | 'admin' | 'developer' | 'readonly';
  auth_method: 'local' | 'ldap' | 'api_key' | 'oidc';
}

// ── Subjects ──

export interface Subject {
  subject: string;
  // derived from latest version:
  schema_type?: 'AVRO' | 'PROTOBUF' | 'JSON';
  latest_version?: number;
  compatibility_level?: CompatibilityLevel;
  compatibility_inherited?: boolean; // true if using global default
  mode?: Mode;
  mode_inherited?: boolean;
}

// GET /subjects returns string[]
// We enrich this client-side by calling config/mode per subject
// OR we use GET /schemas?latestOnly=true&subjectPrefix= to get more detail

export interface SubjectVersion {
  subject: string;
  id: number;               // global schema ID
  version: number;
  schema: string;            // the raw schema string
  schemaType: 'AVRO' | 'PROTOBUF' | 'JSON';
  references: SchemaReference[];
}

export interface SchemaReference {
  name: string;              // reference name (import path for proto, type name for avro)
  subject: string;           // referenced subject
  version: number;           // referenced version (-1 for latest)
}

// ── Schemas ──

export interface Schema {
  id: number;
  schema: string;
  schemaType: 'AVRO' | 'PROTOBUF' | 'JSON';
  references: SchemaReference[];
  subject?: string;          // present when fetched via /schemas/ids/{id}/subjects
  version?: number;
}

export interface SchemaSubjectVersion {
  subject: string;
  version: number;
}

// ── Config ──

export type CompatibilityLevel =
  | 'NONE'
  | 'BACKWARD'
  | 'BACKWARD_TRANSITIVE'
  | 'FORWARD'
  | 'FORWARD_TRANSITIVE'
  | 'FULL'
  | 'FULL_TRANSITIVE';

export interface CompatibilityConfig {
  compatibilityLevel: CompatibilityLevel;
}

export interface CompatibilityCheckResult {
  is_compatible: boolean;
  messages?: string[];       // incompatibility reasons from the server
}

// ── Mode ──

export type Mode = 'READWRITE' | 'READONLY' | 'READONLY_OVERRIDE' | 'IMPORT';

export interface ModeConfig {
  mode: Mode;
}

// ── Users ──

export interface User {
  id: number;
  username: string;
  email: string;
  role: 'super_admin' | 'admin' | 'developer' | 'readonly';
  enabled: boolean;
  created_at: string;        // ISO 8601
  updated_at: string;
}

export interface CreateUserRequest {
  username: string;          // required, 3-50 chars, alphanumeric + hyphens + underscores
  password: string;          // required, min 8 chars, must contain upper, lower, digit
  email: string;             // required, valid email format
  role: 'admin' | 'developer' | 'readonly';   // super_admin only assignable by super_admin
  enabled?: boolean;         // default true
}

export interface UpdateUserRequest {
  password?: string;         // optional, if blank keep current
  email?: string;
  role?: 'admin' | 'developer' | 'readonly';
  enabled?: boolean;
}

// ── API Keys ──

export interface ApiKey {
  id: number;
  key_prefix: string;       // e.g., "sr_live_abc"
  name: string;
  role: 'admin' | 'developer' | 'readonly';
  username: string;          // owner
  created_at: string;
  expires_at: string | null; // null = no expiry
  is_active: boolean;
  revoked_at: string | null;
}

export interface CreateApiKeyRequest {
  name: string;              // required, 1-100 chars
  role: 'admin' | 'developer' | 'readonly';
  expires_in?: number;       // seconds, optional
}

export interface CreateApiKeyResponse extends ApiKey {
  key: string;               // FULL key — only returned once at creation
}

export interface RotateApiKeyResponse extends ApiKey {
  key: string;               // new full key
}

// ── Metadata ──

export interface ServerVersion {
  version: string;
  commit: string;
}

export interface ClusterId {
  id: string;
}

// ── Import ──

export interface ImportSchemaRequest {
  schema: string;
  schemaType: 'AVRO' | 'PROTOBUF' | 'JSON';
  id?: number;               // preserved ID for migration
  version?: number;
  subject?: string;
  references?: SchemaReference[];
}

// ── Errors ──

export interface ApiError {
  error_code: number;
  message: string;
}
```

---

## Page Layouts

ASCII wireframes showing the spatial arrangement of every page. These are the blueprint for claude-code.

### Login Page (`/ui/login`)

```
┌──────────────────────────────────────────────────┐
│                                                  │
│                                                  │
│           ┌──────────────────────┐               │
│           │   [AxonOps Logo]     │               │
│           │   Schema Registry    │               │
│           │                      │               │
│           │  ┌────────────────┐  │               │
│           │  │ Username       │  │ data-testid="login-username-input"
│           │  └────────────────┘  │               │
│           │  ┌────────────────┐  │               │
│           │  │ Password       │  │ data-testid="login-password-input"
│           │  └────────────────┘  │               │
│           │                      │               │
│           │  [═══ Sign In ═══]   │ data-testid="login-submit-btn"
│           │                      │               │
│           │  ─── or ───          │               │
│           │  Use API Key instead │ data-testid="login-apikey-toggle"
│           │                      │               │
│           │  (Phase 5:)          │               │
│           │  [Sign in with SSO]  │ data-testid="login-sso-btn"
│           │                      │               │
│           │  ┌────────────────┐  │               │
│           │  │ Error message  │  │ data-testid="login-error-msg"
│           │  └────────────────┘  │               │
│           └──────────────────────┘               │
│                                                  │
└──────────────────────────────────────────────────┘
```

When "Use API Key instead" is toggled:
- Username and password fields are hidden
- A single "API Key" input appears in their place (`data-testid="login-apikey-input"`)
- Toggle text changes to "Use username and password" (`data-testid="login-password-toggle"`)

### Application Shell (all authenticated pages)

```
┌────────────────────────────────────────────────────────────────────┐
│ [≡] AxonOps Schema Registry                   alice (developer) ▼ │ topbar
│                                                  └─ Sign Out      │ data-testid="nav-topbar"
├──────────┬─────────────────────────────────────────────────────────┤
│          │                                                         │
│ SCHEMAS  │  ┌─ Breadcrumb: Subjects > orders-value > v3 ────────┐ │
│ ○ Subjects│  │                                                   │ │
│ ○ Browser │  │  Page content area                                │ │
│          │  │                                                     │ │
│ CONFIG   │  │  (varies per page — see individual layouts below)  │ │
│ ○ Compat │  │                                                     │ │
│ ○ Modes  │  │                                                     │ │
│          │  │                                                     │ │
│ ADMIN    │  │                                                     │ │
│ ○ Users  │  │                                                     │ │
│ ○ API Keys│ │                                                     │ │
│ ○ Import │  │                                                     │ │
│          │  │                                                     │ │
│ ACCOUNT  │  │                                                     │ │
│ ○ Profile│  │                                                     │ │
│ ○ My Keys│  │                                                     │ │
│          │  │                                                     │ │
│ SYSTEM   │  │                                                     │ │
│ ○ About  │  └─────────────────────────────────────────────────────┘ │
│          │                                                         │
├──────────┴─────────────────────────────────────────────────────────┤
│ v0.1.0 · memory · 12 subjects · 47 schemas                        │ status bar
└────────────────────────────────────────────────────────────────────┘

Sidebar sections visible per role:
  - SCHEMAS:  all roles
  - CONFIG:   admin, super_admin only
  - ADMIN:    admin, super_admin only
  - ACCOUNT:  all roles
  - SYSTEM:   all roles
```

### Subjects List (`/ui/subjects`)

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  Subjects                                    [+ Register New]   │ data-testid="subjects-register-btn"
│                                              (developer+ only)  │
│  ┌──────────────────────────┐  ☐ Show deleted                  │ data-testid="subjects-show-deleted-toggle"
│  │ 🔍 Search subjects...   │  12 subjects                      │ data-testid="subjects-search-input"
│  └──────────────────────────┘                                   │   data-testid="subjects-count-badge"
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Subject            │ Type │ Latest │ Compatibility │ Mode   ││ data-testid="subjects-list-table"
│  ├─────────────────────────────────────────────────────────────┤│
│  │ orders-value       │ AVRO │ v5     │ BACKWARD      │ RW    ││ data-testid="subjects-row-orders-value"
│  │ payments-value     │ AVRO │ v3     │ Global: BACK  │ RW    ││
│  │ users-value        │ PROTO│ v2     │ FULL          │ RW    ││
│  │ events-value       │ JSON │ v1     │ Global: BACK  │ RW    ││
│  │ ~~deleted-topic~~  │ AVRO │ v4     │ NONE     [DEL]│ RO    ││ (strikethrough + badge if deleted)
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  ◀ Page 1 of 3 ▶                                                │ data-testid="subjects-pagination"
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Empty state (when no subjects):
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│              📦                                                  │
│         No subjects registered yet.                              │ data-testid="subjects-list-empty"
│    Register your first schema to get started.                    │
│              [+ Register New Schema]                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Subject Detail (`/ui/subjects/:subject`)

```
┌─────────────────────────────────────────────────────────────────┐
│  ← Subjects                                                     │ breadcrumb
│                                                                 │
│  orders-value                                      [AVRO]       │ data-testid="subject-name-heading"
│  Compatibility: BACKWARD · Mode: READWRITE                      │   data-testid="subject-schema-type-badge"
│                                                                 │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ [Versions]  [Configuration]  [Mode]                        │ │ tabs
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ── Versions tab (default) ──────────────────────────────────── │
│                                                                 │
│  [+ Register New Version]  ☐ Show deleted    [Delete Subject]   │
│  data-testid="subject-register-version-btn"                     │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Version │ Schema ID │ Status │ Registered          │      │  │ data-testid="subject-versions-table"
│  ├───────────────────────────────────────────────────────────┤  │
│  │ v5      │ 47        │ Active │ 2026-02-25 14:30:00 │ [⋮] │  │
│  │ v4      │ 38        │ Active │ 2026-02-20 10:15:00 │ [⋮] │  │
│  │ v3      │ 31        │ Active │ 2026-02-15 09:00:00 │ [⋮] │  │
│  │ v2      │ 22        │ Active │ 2026-02-10 16:45:00 │ [⋮] │  │
│  │ v1      │ 12        │ Active │ 2026-02-05 11:20:00 │ [⋮] │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ── Latest schema preview ──────────────────────────────────── │
│  ┌───────────────────────────────────────────────────────────┐  │ data-testid="subject-latest-schema-preview"
│  │ {                                                         │  │
│  │   "type": "record",                                      │  │ Monaco Editor (read-only)
│  │   "name": "Order",                                       │  │
│  │   "namespace": "com.example.orders",                     │  │
│  │   "fields": [                                            │  │
│  │     {"name": "id", "type": "string"},                    │  │
│  │     {"name": "amount", "type": "double"},                │  │
│  │     {"name": "currency", "type": "string"},              │  │
│  │     {"name": "customer_id", "type": ["null", "string"],  │  │
│  │      "default": null}                                     │  │
│  │   ]                                                       │  │
│  │ }                                                         │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ── Configuration tab ──────────────────────────────────────── │
│                                                                 │
│  Current: BACKWARD (overrides global: FULL)                     │
│  ┌─────────────────────────┐                                    │
│  │ BACKWARD             ▼  │  [Save]  [Reset to Global]        │
│  └─────────────────────────┘                                    │
│  data-testid="subject-compat-select"                            │
│                                                                 │
│  ── Mode tab ───────────────────────────────────────────────── │
│                                                                 │
│  Current: READWRITE (inherited from global)                     │
│  ┌─────────────────────────┐                                    │
│  │ READWRITE             ▼  │  [Save]  [Reset to Global]       │
│  └─────────────────────────┘                                    │
│  data-testid="subject-mode-select"                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Schema Version Detail (`/ui/subjects/:subject/versions/:version`)

```
┌─────────────────────────────────────────────────────────────────┐
│  ← Subjects > orders-value > Version 5                          │ breadcrumb
│                                                                 │
│  ┌─ Metadata ─────────────────────────────────────────────────┐ │ data-testid="version-metadata-panel"
│  │ Global ID: 47 · Version: 5 · Type: AVRO · Status: Active  │ │
│  │ Registered: 2026-02-25 14:30:00 UTC                        │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─ Schema ─────────────────────────── [Copy] [Download .avsc]┐ │ data-testid="version-schema-viewer"
│  │                                                             │ │   data-testid="version-copy-btn"
│  │  {                                                          │ │   data-testid="version-download-btn"
│  │    "type": "record",                                        │ │
│  │    "name": "Order",                                         │ │ Monaco Editor (read-only)
│  │    ...                                                      │ │ Full syntax highlighting
│  │  }                                                          │ │
│  │                                                             │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─ References (outgoing) ────────────────────────────────────┐ │ data-testid="version-references-list"
│  │ This schema references:                                     │ │
│  │  • Address (→ common-types-value v2)                        │ │ (clickable links)
│  │  • Currency (→ enums-value v1)                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─ Referenced By (incoming) ─────────────────────────────────┐ │ data-testid="version-referenced-by-list"
│  │ Schemas that reference this version:                        │ │
│  │  • invoice-value v3 (ID: 51)                               │ │
│  │  • shipment-value v2 (ID: 49)                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─ Compare Versions ─────────────────────────────────────────┐ │
│  │ Compare with: [v4 ▼]   [Compare]                           │ │ data-testid="version-diff-select"
│  │                                                             │ │
│  │ (when Compare clicked, shows Monaco diff viewer below)      │ │ data-testid="version-diff-viewer"
│  │ ┌─────────────────────┬─────────────────────┐              │ │
│  │ │ Version 4           │ Version 5            │              │ │
│  │ │  "fields": [        │  "fields": [         │              │ │
│  │ │    ...               │    ...                │              │ │
│  │ │-   (removed line)   │+   (added line)      │              │ │
│  │ │    ...               │    ...                │              │ │
│  │ └─────────────────────┴─────────────────────┘              │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  [Delete Version] (admin+ only, soft delete)                    │ data-testid="version-delete-btn"
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Register Schema (`/ui/subjects/:subject/register` or modal)

```
┌─────────────────────────────────────────────────────────────────┐
│  Register New Schema Version                               [✕]  │
│                                                                 │
│  Subject:  ┌────────────────────────┐                           │ data-testid="register-subject-input"
│            │ orders-value           │  (pre-filled if from       │ (read-only if adding version)
│            └────────────────────────┘   subject detail)          │
│                                                                 │
│  Type:     ┌────────────────────────┐                           │ data-testid="register-type-select"
│            │ AVRO                ▼  │                            │
│            └────────────────────────┘                            │
│                                                                 │
│  ☐ Normalize schema                                             │ data-testid="register-normalize-toggle"
│                                                                 │
│  Schema:                              [Start from latest v5]    │
│  ┌─────────────────────────────────────────────────────────────┐│ data-testid="register-schema-editor"
│  │                                                              ││
│  │  (Monaco Editor — full editing mode)                         ││ Language mode set by Type selector
│  │                                                              ││ Syntax highlighting + validation
│  │  {                                                           ││
│  │    "type": "record",                                         ││
│  │    "name": "Order",                                          ││
│  │    "fields": [                                               ││
│  │      {"name": "id", "type": "string"},                      ││
│  │      {"name": "amount", "type": "double"}                   ││
│  │    ]                                                         ││
│  │  }                                                           ││
│  │                                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│  Validation: ✅ Valid Avro schema               │               │ data-testid="register-validation-status"
│                                                                 │
│  ▶ References (0)                                               │ data-testid="register-references-section"
│  ┌─────────────────────────────────────────────────────────────┐│ (collapsible)
│  │ Subject            │ Version │ Reference Name │ [Remove]    ││
│  │ common-types-value │ latest  │ Address        │   [✕]       ││
│  │                              [+ Add Reference]              ││ data-testid="register-add-reference-btn"
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  ┌────────────────────────┐  ┌────────────────────────┐        │
│  │  Check Compatibility   │  │      Register          │        │
│  └────────────────────────┘  └────────────────────────┘        │
│  data-testid=                 data-testid=                      │
│    "register-compat-check-btn"  "register-submit-btn"           │
│                                                                 │
│  ┌─ Compatibility Result ─────────────────────────────────────┐ │ data-testid="register-compat-result"
│  │ ✅ Compatible with version 5 under BACKWARD compatibility  │ │ (shown after check)
│  │                                                             │ │
│  │ OR:                                                         │ │
│  │ ❌ Incompatible: Field 'email' was removed. Under BACKWARD │ │
│  │    compatibility, consumers using the old schema must be    │ │
│  │    able to read data written with the new schema. Removing  │ │
│  │    a required field breaks old consumers.                   │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### User Management (`/ui/admin/users`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Users                                          [+ Create User] │ data-testid="users-create-btn"
│                                                                 │
│  ┌──────────────────────────┐                                   │
│  │ 🔍 Search users...      │  4 users                           │ data-testid="users-search-input"
│  └──────────────────────────┘                                   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Username   │ Email          │ Role      │ Status  │ Actions ││ data-testid="users-list-table"
│  ├─────────────────────────────────────────────────────────────┤│
│  │ admin      │ admin@co.io    │ ■ S.Admin │ Enabled │ [⋮]    ││
│  │ alice      │ alice@co.io    │ ● Admin   │ Enabled │ [⋮]    ││
│  │ bob        │ bob@co.io      │ ◆ Dev     │ Enabled │ [⋮]    ││
│  │ carol      │ carol@co.io    │ ○ Read    │ Disabled│ [⋮]    ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Create User Dialog:
┌──────────────────────────────────────┐
│  Create User                    [✕]  │
│                                      │
│  Username *  ┌──────────────────┐    │ data-testid="user-form-username-input"
│              │                  │    │ validation: 3-50 chars, [a-zA-Z0-9_-]
│              └──────────────────┘    │
│  Email *     ┌──────────────────┐    │ data-testid="user-form-email-input"
│              │                  │    │ validation: valid email format
│              └──────────────────┘    │
│  Password *  ┌──────────────────┐    │ data-testid="user-form-password-input"
│              │                  │    │ validation: min 8 chars
│              └──────────────────┘    │
│              ███░░░ Medium           │ (strength indicator)
│                                      │
│  Role *      ┌──────────────────┐    │ data-testid="user-form-role-select"
│              │ developer      ▼ │    │ options based on current user's role
│              └──────────────────┘    │
│  Enabled     [■] On                 │ data-testid="user-form-enabled-toggle"
│                                      │
│         [Cancel]    [Create User]    │ data-testid="user-form-submit-btn"
│                                      │
└──────────────────────────────────────┘
```

---

## Form Validation Rules

Every form field, its type, validation, and error message.

### Login Form

| Field | Type | Required | Validation | Error message |
|-------|------|----------|------------|---------------|
| Username | text | Yes (in password mode) | Non-empty | "Username is required" |
| Password | password | Yes (in password mode) | Non-empty | "Password is required" |
| API Key | text | Yes (in API key mode) | Non-empty, starts with `sr_` | "API key is required" / "Invalid API key format" |

### Create User Form

| Field | Type | Required | Validation | Error message |
|-------|------|----------|------------|---------------|
| Username | text | Yes | 3-50 chars, `[a-zA-Z0-9_-]` only | "Username must be 3-50 characters, letters, numbers, hyphens, underscores only" |
| Email | email | Yes | Valid email regex | "Please enter a valid email address" |
| Password | password | Yes | Min 8 chars | "Password must be at least 8 characters" |
| Role | select | Yes | One of: admin, developer, readonly | — |
| Enabled | toggle | No | Boolean | — |

### Create API Key Form

| Field | Type | Required | Validation | Error message |
|-------|------|----------|------------|---------------|
| Name | text | Yes | 1-100 chars | "Name is required" / "Name must be under 100 characters" |
| Role | select | Yes | One of: admin, developer, readonly (constrained by user's own role) | — |
| Expiration | select | Yes | Preset or custom date | — |
| Custom date | date picker | Conditional | Must be in the future | "Expiration date must be in the future" |

### Register Schema Form

| Field | Type | Required | Validation | Error message |
|-------|------|----------|------------|---------------|
| Subject | text | Yes (for new) | 1-255 chars, no whitespace | "Subject name is required" |
| Schema type | select | Yes | AVRO, PROTOBUF, JSON | — |
| Schema | Monaco editor | Yes | Non-empty, valid syntax for selected type | "Schema is required" / syntax errors shown inline |
| Normalize | checkbox | No | Boolean | — |
| Reference subject | autocomplete | Per-reference | Must match existing subject | "Subject not found" |
| Reference version | number/select | Per-reference | Positive integer or -1 (latest) | "Invalid version" |
| Reference name | text | Per-reference | Non-empty | "Reference name is required" |

### Change Password Form

| Field | Type | Required | Validation | Error message |
|-------|------|----------|------------|---------------|
| Current password | password | Yes | Non-empty | "Current password is required" |
| New password | password | Yes | Min 8 chars, different from current | "Password must be at least 8 characters" |
| Confirm password | password | Yes | Must match new password | "Passwords do not match" |

---

## Monaco Editor Configuration

How to configure Monaco for each schema type.

```typescript
// ui/src/components/schema-editor/monaco-config.ts

export function getMonacoLanguage(schemaType: 'AVRO' | 'PROTOBUF' | 'JSON'): string {
  switch (schemaType) {
    case 'AVRO':    return 'json';      // Avro schemas are JSON
    case 'JSON':    return 'json';      // JSON Schema is JSON
    case 'PROTOBUF': return 'protobuf'; // Needs custom language registration
  }
}

export function getMonacoOptions(readonly: boolean): monaco.editor.IStandaloneEditorConstructionOptions {
  return {
    readOnly: readonly,
    minimap: { enabled: false },
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    wrappingIndent: 'indent',
    automaticLayout: true,
    tabSize: 2,
    formatOnPaste: true,
    formatOnType: true,
    renderWhitespace: 'selection',
    fontSize: 13,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, Monaco, monospace",
    theme: 'vs-dark',          // or 'vs' for light mode — bind to system theme
    scrollbar: {
      verticalScrollbarSize: 8,
      horizontalScrollbarSize: 8,
    },
  };
}

export function getFileExtension(schemaType: 'AVRO' | 'PROTOBUF' | 'JSON'): string {
  switch (schemaType) {
    case 'AVRO':     return '.avsc';
    case 'JSON':     return '.json';
    case 'PROTOBUF': return '.proto';
  }
}

export function getDownloadFilename(subject: string, version: number, schemaType: 'AVRO' | 'PROTOBUF' | 'JSON'): string {
  return `${subject}-v${version}${getFileExtension(schemaType)}`;
}
```

**Protobuf language registration:** Monaco doesn't have built-in proto support. Register a custom language:

```typescript
// Register protobuf syntax highlighting
monaco.languages.register({ id: 'protobuf' });
monaco.languages.setMonarchTokensProvider('protobuf', {
  keywords: [
    'syntax', 'package', 'import', 'option', 'message', 'enum', 'service',
    'rpc', 'returns', 'oneof', 'map', 'reserved', 'repeated', 'optional',
    'required', 'extend', 'extensions', 'to', 'max', 'true', 'false',
    'public', 'weak', 'stream',
  ],
  typeKeywords: [
    'double', 'float', 'int32', 'int64', 'uint32', 'uint64', 'sint32',
    'sint64', 'fixed32', 'fixed64', 'sfixed32', 'sfixed64', 'bool',
    'string', 'bytes',
  ],
  tokenizer: {
    root: [
      [/\/\/.*$/, 'comment'],
      [/\/\*/, 'comment', '@comment'],
      [/"([^"\\]|\\.)*$/, 'string.invalid'],
      [/"/, 'string', '@string'],
      [/[a-zA-Z_]\w*/, {
        cases: {
          '@keywords': 'keyword',
          '@typeKeywords': 'type',
          '@default': 'identifier'
        }
      }],
      [/[{}()\[\]]/, '@brackets'],
      [/[0-9]+/, 'number'],
      [/[;,.]/, 'delimiter'],
      [/=/, 'operator'],
    ],
    comment: [
      [/[^/*]+/, 'comment'],
      [/\*\//, 'comment', '@pop'],
      [/[/*]/, 'comment'],
    ],
    string: [
      [/[^\\"]+/, 'string'],
      [/\\./, 'string.escape'],
      [/"/, 'string', '@pop'],
    ],
  },
});
```

---

## API Call Specifications Per Feature

Exact API calls for each capability, with request/response examples.

### C2.1 — View the subject list

```
GET /subjects
Response: ["orders-value", "payments-value", "users-value"]

// For each subject, to get schema type and latest version:
GET /subjects/{subject}/versions/latest
Response: {
  "subject": "orders-value",
  "id": 47,
  "version": 5,
  "schema": "{...}",
  "schemaType": "AVRO",
  "references": []
}

// For compatibility level:
GET /config/{subject}
Response: { "compatibilityLevel": "BACKWARD" }
// 404 means inherited from global

// For mode:
GET /mode/{subject}
Response: { "mode": "READWRITE" }
// 404 means inherited from global

// For global defaults (called once):
GET /config
Response: { "compatibilityLevel": "BACKWARD" }

GET /mode
Response: { "mode": "READWRITE" }
```

**Performance note:** For a registry with many subjects, making N+1 calls is expensive. The UI should:
1. Call `GET /subjects` to get the list
2. Call `GET /schemas?latestOnly=true` to get all latest schemas in one call (includes type info)
3. Call `GET /config` once for global default
4. Call `GET /mode` once for global default
5. Lazy-load per-subject config/mode only when the user navigates to a subject

### C7.4 — Submit the schema

```
POST /subjects/{subject}/versions
Content-Type: application/vnd.schemaregistry.v1+json

Request body:
{
  "schema": "{\"type\":\"record\",\"name\":\"Order\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}",
  "schemaType": "AVRO",
  "references": [
    {
      "name": "com.example.Address",
      "subject": "common-types-value",
      "version": 2
    }
  ]
}

// NOTE: the "schema" field is a JSON-encoded STRING, not a nested object.
// For Avro: JSON.stringify(parsedSchema)
// For Protobuf: the raw .proto text as a string
// For JSON Schema: JSON.stringify(parsedSchema)

Success response:
{ "id": 48 }    // the global schema ID

Error responses:
409: { "error_code": 409, "message": "Schema already registered with id 47" }
422: { "error_code": 42201, "message": "Invalid Avro schema: ..." }
422: { "error_code": 409, "message": "Schema being registered is incompatible with an earlier schema..." }
```

### C7.3 — Check compatibility

```
POST /compatibility/subjects/{subject}/versions?verbose=true
Content-Type: application/vnd.schemaregistry.v1+json

Request body:
{
  "schema": "{...}",
  "schemaType": "AVRO",
  "references": []
}

Compatible response:
{ "is_compatible": true }

Incompatible response:
{
  "is_compatible": false,
  "messages": [
    "Incompatibility{type:READER_FIELD_MISSING_DEFAULT_VALUE, location:/fields/2, message:..., reader:..., writer:...}"
  ]
}
```

### C9.1 — Soft-delete a version

```
DELETE /subjects/{subject}/versions/{version}

Response: 5   // the deleted version number

// Permanent delete:
DELETE /subjects/{subject}/versions/{version}?permanent=true

Response: 5
```

### C13.2 — Create a user

```
POST /admin/users
Authorization: Bearer <admin-token>
Content-Type: application/json

Request body:
{
  "username": "developer1",
  "password": "SecurePass123!",
  "email": "dev1@example.com",
  "role": "developer",
  "enabled": true
}

Success response:
{
  "id": 5,
  "username": "developer1",
  "email": "dev1@example.com",
  "role": "developer",
  "enabled": true,
  "created_at": "2026-02-26T10:30:00Z",
  "updated_at": "2026-02-26T10:30:00Z"
}

Error responses:
409: { "error_code": 409, "message": "User 'developer1' already exists" }
400: { "error_code": 400, "message": "Invalid role: must be one of admin, developer, readonly" }
```

### C14.2 — Create an API key

```
POST /admin/apikeys
Authorization: Bearer <admin-token>
Content-Type: application/json

Request body:
{
  "name": "ci-pipeline",
  "role": "developer",
  "expires_in": 2592000
}

Success response:
{
  "id": 123,
  "key": "sr_live_abc123...",         // ⚠️ ONLY SHOWN ONCE
  "key_prefix": "sr_live_abc",
  "name": "ci-pipeline",
  "role": "developer",
  "username": "admin",
  "expires_at": "2026-03-28T10:30:00Z"
}
```

---

## Comprehensive BDD Features for Phase 1

### `features/browsing/subjects-list.feature`

```gherkin
@profile:basic @profile:ldap
Feature: Browse subjects

  Background:
    Given I am signed in as "test-developer" with role "developer"
    And the following subjects exist:
      | subject          | type  | versions | compatibility | mode      |
      | orders-value     | AVRO  | 5        | BACKWARD      | READWRITE |
      | payments-value   | AVRO  | 3        | (global)      | READWRITE |
      | users-proto      | PROTO | 2        | FULL          | READWRITE |
      | events-json      | JSON  | 1        | (global)      | READWRITE |

  Scenario: Subject list displays all subjects with metadata
    When I navigate to the subjects page
    Then I should see 4 subjects in the table
    And the subject "orders-value" should show type "AVRO"
    And the subject "orders-value" should show latest version "5"
    And the subject "orders-value" should show compatibility "BACKWARD"
    And the subject "users-proto" should show type "PROTOBUF"

  Scenario: Inherited vs overridden compatibility is distinguishable
    When I navigate to the subjects page
    Then the subject "orders-value" compatibility should show "BACKWARD"
    And the subject "payments-value" compatibility should show "Global: BACKWARD"

  Scenario: Search filters subjects by name
    When I navigate to the subjects page
    And I type "order" in the search input
    Then I should see 1 subject in the table
    And I should see "orders-value" in the results
    And I should NOT see "payments-value" in the results

  Scenario: Search is case-insensitive
    When I navigate to the subjects page
    And I type "ORDERS" in the search input
    Then I should see 1 subject in the table
    And I should see "orders-value" in the results

  Scenario: Search with no results shows empty state
    When I navigate to the subjects page
    And I type "nonexistent" in the search input
    Then I should see 0 subjects in the table
    And I should see the message "No subjects match your search"

  Scenario: Clearing search shows all subjects
    When I navigate to the subjects page
    And I type "order" in the search input
    Then I should see 1 subject in the table
    When I clear the search input
    Then I should see 4 subjects in the table

  Scenario: Deleted subjects are hidden by default
    Given the subject "old-topic" exists and is soft-deleted
    When I navigate to the subjects page
    Then I should see 4 subjects in the table
    And I should NOT see "old-topic" in the results

  Scenario: Showing deleted subjects includes them with visual indicator
    Given the subject "old-topic" exists and is soft-deleted
    When I navigate to the subjects page
    And I toggle "Show deleted subjects"
    Then I should see 5 subjects in the table
    And the subject "old-topic" should have a "Deleted" badge

  Scenario: Clicking a subject name navigates to detail
    When I navigate to the subjects page
    And I click on subject "orders-value"
    Then I should be on the subject detail page for "orders-value"

  Scenario: Developer sees Register button
    When I navigate to the subjects page
    Then I should see the "Register New Schema" button

  Scenario: Empty registry shows helpful empty state
    Given no subjects exist
    When I navigate to the subjects page
    Then I should see the empty state message "No subjects registered yet"
    And I should see a "Register your first schema" call to action

  Scenario: Subject count badge updates with filter
    When I navigate to the subjects page
    Then the subject count badge should show "4"
    When I type "order" in the search input
    Then the subject count badge should show "1"
```

### `features/browsing/schema-version.feature`

```gherkin
@profile:basic
Feature: Inspect a schema version

  Background:
    Given I am signed in as "test-developer" with role "developer"
    And subject "orders-value" exists with the following Avro schema at version 3:
      """
      {
        "type": "record",
        "name": "Order",
        "namespace": "com.example",
        "fields": [
          {"name": "id", "type": "string"},
          {"name": "amount", "type": "double"},
          {"name": "currency", "type": "string"}
        ]
      }
      """

  Scenario: Schema is displayed with syntax highlighting
    When I navigate to version 3 of subject "orders-value"
    Then the schema viewer should contain "Order"
    And the schema viewer should be in read-only mode

  Scenario: Metadata panel shows correct information
    When I navigate to version 3 of subject "orders-value"
    Then the metadata panel should show version "3"
    And the metadata panel should show type "AVRO"
    And the metadata panel should show status "Active"
    And the metadata panel should show a registration timestamp

  Scenario: Copy to clipboard copies the raw schema
    When I navigate to version 3 of subject "orders-value"
    And I click the "Copy" button
    Then the clipboard should contain the raw schema JSON

  Scenario: Download produces correctly named file
    When I navigate to version 3 of subject "orders-value"
    And I click the "Download" button
    Then a file "orders-value-v3.avsc" should be downloaded

  Scenario: Breadcrumb navigation works
    When I navigate to version 3 of subject "orders-value"
    Then the breadcrumb should show "Subjects > orders-value > Version 3"
    When I click "orders-value" in the breadcrumb
    Then I should be on the subject detail page for "orders-value"
    When I click "Subjects" in the breadcrumb
    Then I should be on the subjects list page

  Scenario: Schema references are displayed as links
    Given version 3 of "orders-value" references "common-types-value" version 2 as "Address"
    When I navigate to version 3 of subject "orders-value"
    Then the references section should list "Address → common-types-value v2"
    When I click on the reference "common-types-value v2"
    Then I should be on the version detail page for "common-types-value" version 2

  Scenario: Referenced-by section shows dependent schemas
    Given version 1 of "invoice-value" references "orders-value" version 3
    When I navigate to version 3 of subject "orders-value"
    Then the "Referenced By" section should list "invoice-value"
```

### `features/navigation/navigation.feature`

```gherkin
@profile:basic
Feature: Application navigation

  Scenario: Sidebar shows correct sections for developer
    Given I am signed in as a user with role "developer"
    Then the sidebar should contain section "SCHEMAS" with items:
      | Subjects       |
      | Schema Browser |
    And the sidebar should contain section "ACCOUNT" with items:
      | My Profile |
      | My API Keys |
    And the sidebar should contain section "SYSTEM" with items:
      | About |
    And the sidebar should NOT contain section "CONFIGURATION"
    And the sidebar should NOT contain section "ADMINISTRATION"

  Scenario: Sidebar shows all sections for admin
    Given I am signed in as a user with role "admin"
    Then the sidebar should contain section "CONFIGURATION" with items:
      | Compatibility |
      | Modes         |
    And the sidebar should contain section "ADMINISTRATION" with items:
      | Users    |
      | API Keys |
      | Import   |

  Scenario: Status bar shows registry info
    Given I am signed in as "test-developer"
    Then the status bar should display the server version
    And the status bar should display the storage type
    And the status bar should display the subject count
    And the status bar should display the schema count

  Scenario: Active page is highlighted in sidebar
    Given I am signed in as "test-developer"
    When I navigate to the subjects page
    Then "Subjects" should be highlighted in the sidebar
    When I navigate to the schema browser page
    Then "Schema Browser" should be highlighted in the sidebar
    And "Subjects" should NOT be highlighted in the sidebar

  Scenario: User menu shows username and sign out
    Given I am signed in as "test-developer"
    Then the top bar should display "test-developer"
    When I click the user menu
    Then I should see a "Sign Out" option

  Scenario: Sidebar collapses on smaller screens
    Given I am signed in as "test-developer"
    And the viewport is 768px wide
    Then the sidebar should be collapsed
    When I click the sidebar toggle
    Then the sidebar should be expanded
```

---

### Schema Browser (`/ui/schemas`)

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  Schema Browser                                                 │
│                                                                 │
│  Lookup by Global ID:                                           │
│  ┌──────────────────────────┐  [Lookup]                        │ data-testid="schemas-id-input"
│  │ Enter schema ID...       │                                   │   data-testid="schemas-id-lookup-btn"
│  └──────────────────────────┘                                   │
│                                                                 │
│  ── Latest Schemas ─────────────────────────────────────────── │
│                                                                 │
│  ┌──────────────────────────┐                                   │
│  │ 🔍 Filter by subject... │                                   │ data-testid="schemas-filter-input"
│  └──────────────────────────┘                                   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ ID  │ Subject          │ Version │ Type │ References       ││ data-testid="schemas-list-table"
│  ├─────────────────────────────────────────────────────────────┤│
│  │ 47  │ orders-value     │ v5      │ AVRO │ 2 refs           ││
│  │ 38  │ payments-value   │ v3      │ AVRO │ —                ││
│  │ 31  │ users-proto      │ v2      │ PROTO│ 1 ref            ││
│  │ 22  │ events-json      │ v1      │ JSON │ —                ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  ◀ 1 2 3 ▶                                                     │ data-testid="schemas-pagination"
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Schema By ID result page (`/ui/schemas/:id`):
┌─────────────────────────────────────────────────────────────────┐
│  ← Schema Browser                                              │
│                                                                 │
│  Schema ID: 47                                    [Copy] [DL]  │
│  Type: AVRO                                                     │
│                                                                 │
│  ┌─ Used in subjects ────────────────────────────────────────┐  │ data-testid="schema-subjects-list"
│  │ • orders-value v5                                         │  │ (clickable links)
│  │ • orders-value-staging v3                                 │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Schema content ──────────────────────────────────────────┐  │ data-testid="schema-viewer"
│  │ { "type": "record", ... }                                 │  │ Monaco (read-only)
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### API Key Management (`/ui/admin/apikeys`)

```
┌─────────────────────────────────────────────────────────────────┐
│  API Keys                                   [+ Create API Key]  │ data-testid="apikeys-create-btn"
│                                                                 │
│  ┌──────────────────────────┐                                   │
│  │ 🔍 Search keys...       │  6 keys                            │ data-testid="apikeys-search-input"
│  └──────────────────────────┘                                   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Prefix      │ Name         │ Role │ Owner │ Expires  │ [⋮] ││ data-testid="apikeys-list-table"
│  ├─────────────────────────────────────────────────────────────┤│
│  │ sr_live_abc │ ci-pipeline  │ Dev  │ admin │ 30 days  │ [⋮] ││
│  │ sr_live_def │ monitoring   │ Read │ admin │ Never    │ [⋮] ││
│  │ sr_live_ghi │ staging-ci   │ Dev  │ alice │ Expired! │ [⋮] ││ (expired = red badge)
│  │ sr_live_jkl │ old-key      │ Dev  │ bob   │ Revoked  │ [⋮] ││ (revoked = grey + strikethrough)
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  Actions menu (⋮):                                              │
│    • Rotate key (generates new key, invalidates old)            │ data-testid="apikey-rotate-btn"
│    • Revoke key (permanently disables)                          │ data-testid="apikey-revoke-btn"
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Create API Key dialog:
┌──────────────────────────────────────┐
│  Create API Key                 [✕]  │
│                                      │
│  Name *      ┌──────────────────┐    │ data-testid="apikey-form-name-input"
│              │                  │    │
│              └──────────────────┘    │
│  Role *      ┌──────────────────┐    │ data-testid="apikey-form-role-select"
│              │ developer      ▼ │    │
│              └──────────────────┘    │
│  Expires     ┌──────────────────┐    │ data-testid="apikey-form-expiry-select"
│              │ 30 days        ▼ │    │ Options: 7d, 30d, 90d, 1y, Never, Custom
│              └──────────────────┘    │
│                                      │
│         [Cancel]    [Create Key]     │ data-testid="apikey-form-submit-btn"
└──────────────────────────────────────┘

After creation — key reveal (SHOWN ONCE ONLY):
┌──────────────────────────────────────┐
│  ✅ API Key Created                  │
│                                      │
│  ⚠️ Copy this key now. It will NOT  │
│  be shown again.                     │
│                                      │
│  ┌──────────────────────────────┐    │ data-testid="apikey-created-value"
│  │ sr_live_abc123def456ghi789   │    │ (monospace, selectable)
│  └──────────────────────────────┘    │
│  [📋 Copy to clipboard]             │ data-testid="apikey-created-copy-btn"
│                                      │
│              [Done]                  │
└──────────────────────────────────────┘
```

### Global Compatibility Config (`/ui/config`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Compatibility Configuration                                    │
│                                                                 │
│  ┌─ Global Default ──────────────────────────────────────────┐  │
│  │                                                            │  │
│  │  Current:  BACKWARD                                        │  │
│  │                                                            │  │
│  │  Change to: ┌────────────────┐  [Save]                    │  │ data-testid="config-global-compat-select"
│  │             │ BACKWARD     ▼ │                             │  │   data-testid="config-global-compat-save-btn"
│  │             └────────────────┘                             │  │
│  │                                                            │  │
│  │  Options: NONE, BACKWARD, BACKWARD_TRANSITIVE,            │  │
│  │           FORWARD, FORWARD_TRANSITIVE, FULL,               │  │
│  │           FULL_TRANSITIVE                                  │  │
│  │                                                            │  │
│  │  ℹ️ Changing the global default affects all subjects      │  │
│  │  that do not have a subject-level override.                │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Subject Overrides ───────────────────────────────────────┐  │
│  │                                                            │  │ data-testid="config-overrides-table"
│  │  Subject          │ Override Level        │ Action         │  │
│  │  orders-value     │ BACKWARD              │ [Reset]        │  │
│  │  users-proto      │ FULL                  │ [Reset]        │  │
│  │                                                            │  │
│  │  2 of 12 subjects have overrides                           │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Import Schemas (`/ui/import`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Import Schemas                                                 │
│                                                                 │
│  ┌─ Prerequisites ───────────────────────────────────────────┐  │
│  │ ⚠️ Registry mode must be IMPORT to use this feature.     │  │
│  │ Current mode: READWRITE                                    │  │
│  │ [Switch to IMPORT mode]                                    │  │ data-testid="import-switch-mode-btn"
│  └────────────────────────────────────────────────────────────┘  │
│  (above warning hides when mode is already IMPORT)              │
│                                                                 │
│  ┌─ Import Method ───────────────────────────────────────────┐  │
│  │                                                            │  │
│  │  (•) Single Schema    ( ) Bulk JSON File                  │  │ data-testid="import-method-radio"
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ── Single Schema ──────────────────────────────────────────── │
│                                                                 │
│  Subject:   ┌────────────────────────┐                         │ data-testid="import-subject-input"
│             │                        │                          │
│             └────────────────────────┘                          │
│  Schema ID: ┌────────────┐  Version: ┌────────────┐           │ data-testid="import-id-input"
│             │            │           │            │             │   data-testid="import-version-input"
│             └────────────┘           └────────────┘             │
│  Type:      ┌────────────────────────┐                         │ data-testid="import-type-select"
│             │ AVRO                ▼  │                          │
│             └────────────────────────┘                          │
│  Schema:                                                        │
│  ┌─────────────────────────────────────────────────────────────┐│ data-testid="import-schema-editor"
│  │ (Monaco editor)                                              ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  [Import Schema]                                                │ data-testid="import-submit-btn"
│                                                                 │
│  ── Bulk JSON File ─────────────────────────────────────────── │
│                                                                 │
│  Drop a JSON file or click to browse:                           │
│  ┌─────────────────────────────────────────────────────────────┐│ data-testid="import-file-dropzone"
│  │           📁 Drop JSON file here                             ││
│  │           or click to browse                                 ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                 │
│  Expected format: array of objects with schema, schemaType,     │
│  subject, id, version, and optional references fields.          │
│                                                                 │
│  Preview: 47 schemas to import                                  │ data-testid="import-preview-count"
│  [Import All]                                                   │ data-testid="import-bulk-submit-btn"
│                                                                 │
│  ┌─ Progress ────────────────────────────────────────────────┐  │ data-testid="import-progress"
│  │ ██████████░░░░ 32/47 imported                              │  │
│  │ ✅ orders-value v1 (ID: 12)                                │  │
│  │ ✅ orders-value v2 (ID: 22)                                │  │
│  │ ❌ payments-value v1 — Subject already exists              │  │
│  │ ...                                                        │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### About Page (`/ui/about`)

```
┌─────────────────────────────────────────────────────────────────┐
│  About                                                          │
│                                                                 │
│  ┌─ AxonOps Schema Registry ─────────────────────────────────┐  │
│  │                                                            │  │ data-testid="about-info-panel"
│  │  Version:        0.1.0                                     │  │ data-testid="about-version"
│  │  Commit:         abc1234                                   │  │ data-testid="about-commit"
│  │  Cluster ID:     sr-cluster-xyz                            │  │ data-testid="about-cluster-id"
│  │  Storage:        memory                                    │  │ data-testid="about-storage"
│  │  Schema Types:   AVRO, PROTOBUF, JSON                     │  │ data-testid="about-schema-types"
│  │                                                            │  │
│  │  GitHub: github.com/axonops/axonops-schema-registry       │  │ (clickable link)
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Statistics ──────────────────────────────────────────────┐  │
│  │                                                            │  │ data-testid="about-stats-panel"
│  │   ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐         │  │
│  │   │   12   │  │   47   │  │    3   │  │    8   │         │  │
│  │   │Subjects│  │Schemas │  │ Types  │  │ Users  │         │  │
│  │   └────────┘  └────────┘  └────────┘  └────────┘         │  │
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### My Profile (`/ui/account`)

```
┌─────────────────────────────────────────────────────────────────┐
│  My Profile                                                     │
│                                                                 │
│  ┌─ Account Information ─────────────────────────────────────┐  │ data-testid="profile-info-panel"
│  │                                                            │  │
│  │  Username:   alice                                         │  │ (read-only)
│  │  Email:      ┌────────────────────┐  [Save]               │  │ data-testid="profile-email-input"
│  │              │ alice@company.io   │                         │  │   data-testid="profile-email-save-btn"
│  │              └────────────────────┘                         │  │
│  │  Role:       developer                                     │  │ (read-only)
│  │  Auth:       local                                         │  │ (read-only)
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌─ Change Password ─────────────────────────────────────────┐  │ data-testid="profile-password-section"
│  │  (hidden for LDAP/OIDC users — they manage passwords      │  │
│  │   through their identity provider)                         │  │
│  │                                                            │  │
│  │  Current password: ┌──────────────────┐                    │  │ data-testid="profile-current-password-input"
│  │                    └──────────────────┘                     │  │
│  │  New password:     ┌──────────────────┐                    │  │ data-testid="profile-new-password-input"
│  │                    └──────────────────┘                     │  │
│  │  Confirm:          ┌──────────────────┐                    │  │ data-testid="profile-confirm-password-input"
│  │                    └──────────────────┘                     │  │
│  │                                                            │  │
│  │                  [Change Password]                          │  │ data-testid="profile-change-password-btn"
│  └────────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## API Client Architecture

Central API client that all components use. Handles auth headers, error mapping, and token refresh.

```typescript
// ui/src/api/client.ts

import { ApiError } from '../types/api';

let accessToken: string | null = null;
let onAuthFailure: (() => void) | null = null;

export function setToken(token: string | null) {
  accessToken = token;
}

export function getToken(): string | null {
  return accessToken;
}

export function setOnAuthFailure(handler: () => void) {
  onAuthFailure = handler;
}

/**
 * Central fetch wrapper for all API calls.
 * - Injects Authorization header
 * - Handles 401 → redirect to login
 * - Parses JSON responses
 * - Throws typed ApiError on non-2xx
 */
export async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    'Accept': 'application/vnd.schemaregistry.v1+json, application/json',
    ...(options.headers as Record<string, string> || {}),
  };

  if (accessToken) {
    headers['Authorization'] = `Bearer ${accessToken}`;
  }

  // Set Content-Type for non-GET requests if body is present
  if (options.body && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/vnd.schemaregistry.v1+json';
  }

  const response = await fetch(path, {
    ...options,
    headers,
  });

  // Handle 401 — token expired or invalid
  if (response.status === 401) {
    setToken(null);
    onAuthFailure?.();
    throw new ApiClientError(401, 'Session expired. Please sign in again.');
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  const body = await response.json();

  if (!response.ok) {
    throw new ApiClientError(
      response.status,
      body.message || `Request failed with status ${response.status}`,
      body.error_code
    );
  }

  return body as T;
}

export class ApiClientError extends Error {
  constructor(
    public readonly status: number,
    message: string,
    public readonly errorCode?: number,
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}
```

### TanStack Query Configuration

```typescript
// ui/src/api/queries.ts

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch, ApiClientError } from './client';
import type {
  SubjectVersion,
  CompatibilityConfig,
  CompatibilityCheckResult,
  ModeConfig,
  Schema,
  SchemaSubjectVersion,
} from '../types/api';

// ── Query Keys ──
// Structured keys enable granular cache invalidation

export const queryKeys = {
  subjects: {
    all: ['subjects'] as const,
    list: (opts?: { deleted?: boolean; prefix?: string }) =>
      ['subjects', 'list', opts] as const,
    detail: (subject: string) => ['subjects', subject] as const,
    versions: (subject: string) => ['subjects', subject, 'versions'] as const,
    version: (subject: string, version: number) =>
      ['subjects', subject, 'versions', version] as const,
    config: (subject: string) => ['subjects', subject, 'config'] as const,
    mode: (subject: string) => ['subjects', subject, 'mode'] as const,
  },
  schemas: {
    all: ['schemas'] as const,
    byId: (id: number) => ['schemas', id] as const,
    subjects: (id: number) => ['schemas', id, 'subjects'] as const,
  },
  config: {
    global: ['config', 'global'] as const,
  },
  mode: {
    global: ['mode', 'global'] as const,
  },
  users: {
    all: ['users'] as const,
    detail: (id: number) => ['users', id] as const,
  },
  apikeys: {
    all: ['apikeys'] as const,
  },
  metadata: {
    version: ['metadata', 'version'] as const,
    clusterId: ['metadata', 'clusterId'] as const,
    schemaTypes: ['metadata', 'schemaTypes'] as const,
  },
} as const;

// ── Subject queries ──

export function useSubjects(opts?: { deleted?: boolean; prefix?: string }) {
  return useQuery({
    queryKey: queryKeys.subjects.list(opts),
    queryFn: () => apiFetch<string[]>(
      `/subjects${opts?.deleted ? '?deleted=true' : ''}`
    ),
  });
}

export function useSubjectVersions(subject: string) {
  return useQuery({
    queryKey: queryKeys.subjects.versions(subject),
    queryFn: () => apiFetch<number[]>(`/subjects/${encodeURIComponent(subject)}/versions`),
    enabled: !!subject,
  });
}

export function useSubjectVersion(subject: string, version: number | 'latest') {
  return useQuery({
    queryKey: queryKeys.subjects.version(subject, version as number),
    queryFn: () => apiFetch<SubjectVersion>(
      `/subjects/${encodeURIComponent(subject)}/versions/${version}`
    ),
    enabled: !!subject && version !== undefined,
  });
}

export function useSubjectConfig(subject: string) {
  return useQuery({
    queryKey: queryKeys.subjects.config(subject),
    queryFn: async () => {
      try {
        return await apiFetch<CompatibilityConfig>(
          `/config/${encodeURIComponent(subject)}`
        );
      } catch (e) {
        if (e instanceof ApiClientError && e.status === 404) {
          return null;  // inherits global
        }
        throw e;
      }
    },
    enabled: !!subject,
  });
}

// ── Schema mutations ──

export function useRegisterSchema(subject: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: { schema: string; schemaType: string; references?: any[] }) =>
      apiFetch<{ id: number }>(
        `/subjects/${encodeURIComponent(subject)}/versions`,
        { method: 'POST', body: JSON.stringify(body) }
      ),
    onSuccess: () => {
      // Invalidate caches that show version lists
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.versions(subject) });
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.list() });
      queryClient.invalidateQueries({ queryKey: queryKeys.schemas.all });
    },
  });
}

export function useCheckCompatibility(subject: string) {
  return useMutation({
    mutationFn: (body: { schema: string; schemaType: string; references?: any[] }) =>
      apiFetch<CompatibilityCheckResult>(
        `/compatibility/subjects/${encodeURIComponent(subject)}/versions?verbose=true`,
        { method: 'POST', body: JSON.stringify(body) }
      ),
  });
}

export function useDeleteVersion(subject: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ version, permanent }: { version: number; permanent?: boolean }) =>
      apiFetch<number>(
        `/subjects/${encodeURIComponent(subject)}/versions/${version}${permanent ? '?permanent=true' : ''}`,
        { method: 'DELETE' }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.subjects.versions(subject) });
    },
  });
}
```

### Auth Context

```typescript
// ui/src/context/auth-context.tsx

import { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react';
import type { AuthUser, AuthResponse, AuthConfig } from '../types/api';
import { apiFetch, setToken, setOnAuthFailure } from '../api/client';

interface AuthContextType {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  authConfig: AuthConfig | null;
  login: (username: string, password: string) => Promise<void>;
  loginWithApiKey: (apiKey: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const refreshTimerRef = useRef<number>();

  // Schedule token refresh before expiry
  const scheduleRefresh = useCallback((expiresAt: string) => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);

    const expiryMs = new Date(expiresAt).getTime();
    const refreshMs = expiryMs - Date.now() - 60_000; // refresh 1 min before expiry

    if (refreshMs > 0) {
      refreshTimerRef.current = window.setTimeout(async () => {
        try {
          const res = await apiFetch<AuthResponse>('/ui/auth/session', { method: 'GET' });
          setToken(res.token);
          setUser(res.user);
          scheduleRefresh(res.expires_at);
        } catch {
          // Token refresh failed — force re-login
          setToken(null);
          setUser(null);
        }
      }, refreshMs);
    }
  }, []);

  // On mount: fetch auth config and validate existing session
  useEffect(() => {
    async function init() {
      try {
        const config = await apiFetch<AuthConfig>('/ui/auth/config');
        setAuthConfig(config);

        // Try to validate existing session (cookie-based refresh if supported)
        const session = await apiFetch<AuthResponse>('/ui/auth/session');
        setToken(session.token);
        setUser(session.user);
        scheduleRefresh(session.expires_at);
      } catch {
        // No valid session — user needs to login
      } finally {
        setIsLoading(false);
      }
    }
    init();

    // Set up 401 handler
    setOnAuthFailure(() => {
      setUser(null);
      // Router will redirect to /ui/login via AuthGuard
    });
  }, [scheduleRefresh]);

  const login = useCallback(async (username: string, password: string) => {
    const res = await apiFetch<AuthResponse>('/ui/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    setToken(res.token);
    setUser(res.user);
    scheduleRefresh(res.expires_at);
  }, [scheduleRefresh]);

  const loginWithApiKey = useCallback(async (apiKey: string) => {
    const res = await apiFetch<AuthResponse>('/ui/auth/apikey', {
      method: 'POST',
      body: JSON.stringify({ key: apiKey }),
    });
    setToken(res.token);
    setUser(res.user);
    scheduleRefresh(res.expires_at);
  }, [scheduleRefresh]);

  const logout = useCallback(async () => {
    try {
      await apiFetch('/ui/auth/logout', { method: 'POST' });
    } catch {
      // Best-effort server-side invalidation
    }
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    setToken(null);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{
      user,
      isAuthenticated: !!user,
      isLoading,
      authConfig,
      login,
      loginWithApiKey,
      logout,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
```

---

## Additional API Call Specifications

### List all versions of a subject

```
GET /subjects/{subject}/versions
Response: [1, 2, 3, 4, 5]

GET /subjects/{subject}/versions?deleted=true
Response: [1, 2, 3, 4, 5, 6]   // includes soft-deleted
```

### Fetch a specific version

```
GET /subjects/{subject}/versions/{version}
Response:
{
  "subject": "orders-value",
  "id": 47,
  "version": 5,
  "schema": "{\"type\":\"record\",\"name\":\"Order\",...}",
  "schemaType": "AVRO",
  "references": [
    { "name": "com.example.Address", "subject": "common-types-value", "version": 2 }
  ]
}
```

### Fetch schema by global ID

```
GET /schemas/ids/{id}
Response:
{
  "schema": "{...}",
  "schemaType": "AVRO",
  "references": []
}

GET /schemas/ids/{id}/subjects
Response: [
  { "subject": "orders-value", "version": 5 },
  { "subject": "orders-staging-value", "version": 3 }
]
```

### Set subject-level compatibility

```
PUT /config/{subject}
Content-Type: application/vnd.schemaregistry.v1+json

Request: { "compatibility": "FULL" }
Response: { "compatibility": "FULL" }
```

### Reset subject-level compatibility (inherit global)

```
DELETE /config/{subject}
Response: { "compatibility": "BACKWARD" }    // returns the global default
```

### Set subject-level mode

```
PUT /mode/{subject}
Content-Type: application/vnd.schemaregistry.v1+json

Request: { "mode": "READONLY" }
Response: { "mode": "READONLY" }
```

### Set global mode

```
PUT /mode
Content-Type: application/vnd.schemaregistry.v1+json

Request: { "mode": "IMPORT" }
Response: { "mode": "IMPORT" }
```

### Soft-delete an entire subject

```
DELETE /subjects/{subject}
Response: [1, 2, 3, 4, 5]    // all deleted version numbers

DELETE /subjects/{subject}?permanent=true
Response: [1, 2, 3, 4, 5]
```

### List users

```
GET /admin/users
Authorization: Bearer <admin-token>

Response: [
  {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "role": "super_admin",
    "enabled": true,
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  },
  ...
]
```

### Update a user

```
PUT /admin/users/{id}
Authorization: Bearer <admin-token>
Content-Type: application/json

Request: {
  "email": "newemail@example.com",
  "role": "admin",
  "enabled": true
}

Response: { "id": 2, "username": "alice", ... }
```

### Delete a user

```
DELETE /admin/users/{id}
Authorization: Bearer <admin-token>

Response: 204 No Content
```

### List API keys

```
GET /admin/apikeys
Authorization: Bearer <admin-token>

Response: [
  {
    "id": 123,
    "key_prefix": "sr_live_abc",
    "name": "ci-pipeline",
    "role": "developer",
    "username": "admin",
    "created_at": "2026-02-01T10:00:00Z",
    "expires_at": "2026-03-01T10:00:00Z",
    "is_active": true,
    "revoked_at": null
  },
  ...
]
```

### Revoke an API key

```
POST /admin/apikeys/{id}/revoke
Authorization: Bearer <admin-token>

Response: { "id": 123, "is_active": false, "revoked_at": "2026-02-26T15:00:00Z", ... }
```

### Rotate an API key

```
POST /admin/apikeys/{id}/rotate
Authorization: Bearer <admin-token>

Response: {
  "id": 123,
  "key": "sr_live_newkey789...",      // NEW key — only shown once
  "key_prefix": "sr_live_new",
  ...
}
```

### Referenced-by lookup

```
GET /subjects/{subject}/versions/{version}/referencedby
Response: [51, 49]    // global schema IDs that reference this version
```

---

## Comprehensive BDD Features — Auth

### `features/auth/login.feature`

```gherkin
@profile:basic
Feature: Sign in with username and password

  Background:
    Given the schema registry is running with basic auth enabled
    And the following users exist:
      | username      | password        | role       | enabled |
      | test-admin    | AdminPass123!   | admin      | true    |
      | test-dev      | DevPass123!     | developer  | true    |
      | test-reader   | ReadPass123!    | readonly   | true    |
      | disabled-user | DisabledPass1!  | developer  | false   |

  Scenario: Successful login redirects to subjects page
    Given I am on the login page
    When I enter "test-admin" in the username field
    And I enter "AdminPass123!" in the password field
    And I click the sign in button
    Then I should be redirected to the subjects page
    And the top bar should display "test-admin"

  Scenario: Wrong password shows error
    Given I am on the login page
    When I enter "test-admin" in the username field
    And I enter "WrongPassword" in the password field
    And I click the sign in button
    Then I should see the error "Invalid username or password"
    And I should remain on the login page

  Scenario: Non-existent username shows error
    Given I am on the login page
    When I enter "nonexistent" in the username field
    And I enter "SomePass123!" in the password field
    And I click the sign in button
    Then I should see the error "Invalid username or password"

  Scenario: Disabled user cannot login
    Given I am on the login page
    When I enter "disabled-user" in the username field
    And I enter "DisabledPass1!" in the password field
    And I click the sign in button
    Then I should see the error "Invalid username or password"

  Scenario: Empty fields show validation errors
    Given I am on the login page
    When I click the sign in button
    Then I should see the error "Username is required"

  Scenario: Login preserves redirect destination
    Given I directly navigate to "/ui/subjects/orders-value"
    Then I should be redirected to the login page
    When I sign in as "test-dev" with password "DevPass123!"
    Then I should be redirected to "/ui/subjects/orders-value"

  Scenario: Already authenticated user is redirected from login
    Given I am signed in as "test-dev"
    When I navigate to the login page
    Then I should be redirected to the subjects page
```

### `features/auth/apikey-login.feature`

```gherkin
@profile:basic
Feature: Sign in with API key

  Background:
    Given the schema registry is running with basic auth enabled
    And the following API keys exist:
      | name         | role      | key                        | active |
      | ci-key       | developer | sr_live_testkey123abc      | true   |
      | revoked-key  | developer | sr_live_revokedkey456      | false  |

  Scenario: Toggle to API key login mode
    Given I am on the login page
    When I click "Use API Key instead"
    Then I should see the API key input field
    And I should NOT see the username field
    And I should NOT see the password field

  Scenario: Successful API key login
    Given I am on the login page
    And I switch to API key mode
    When I enter "sr_live_testkey123abc" in the API key field
    And I click the sign in button
    Then I should be redirected to the subjects page

  Scenario: Invalid API key shows error
    Given I am on the login page
    And I switch to API key mode
    When I enter "sr_live_invalidkey999" in the API key field
    And I click the sign in button
    Then I should see the error "Invalid API key"

  Scenario: Revoked API key shows error
    Given I am on the login page
    And I switch to API key mode
    When I enter "sr_live_revokedkey456" in the API key field
    And I click the sign in button
    Then I should see the error "Invalid API key"

  Scenario: Toggle back to password mode
    Given I am on the login page
    And I switch to API key mode
    When I click "Use username and password"
    Then I should see the username field
    And I should see the password field
    And I should NOT see the API key input field
```

### `features/auth/session.feature`

```gherkin
@profile:basic
Feature: Session management

  Background:
    Given the schema registry is running with sessions configured to expire after 5 minutes

  Scenario: Sign out clears session
    Given I am signed in as "test-admin"
    When I click the user menu
    And I click "Sign Out"
    Then I should be redirected to the login page
    When I navigate to the subjects page
    Then I should be redirected to the login page

  Scenario: Expired session redirects to login
    Given I am signed in as "test-admin"
    And I wait for the session to expire
    When I navigate to the subjects page
    Then I should be redirected to the login page
    And I should see the message "Your session has expired. Please sign in again."

  Scenario: Session displays correct user info
    Given I am signed in as "test-dev" with role "developer"
    Then the top bar should display username "test-dev"
    And the auth context should have role "developer"
```

## Comprehensive BDD Features — Phase 2

### `features/authoring/register-schema.feature`

```gherkin
@profile:basic
Feature: Register a new schema

  Background:
    Given I am signed in as "test-developer" with role "developer"
    And no subjects exist

  Scenario: Register the first Avro schema under a new subject
    When I navigate to "/ui/subjects"
    And I click "Register New Schema"
    And I enter "orders-value" as the subject name
    And I select "AVRO" as the schema type
    And I enter the following schema in the editor:
      """
      {
        "type": "record",
        "name": "Order",
        "namespace": "com.example",
        "fields": [
          {"name": "id", "type": "string"},
          {"name": "amount", "type": "double"}
        ]
      }
      """
    And I click "Register"
    Then I should see a success toast "Schema registered as version 1"
    And I should be navigated to the subject detail page for "orders-value"
    And the versions table should show version 1

  Scenario: Register with invalid Avro schema shows validation error
    When I navigate to the register schema page
    And I enter "bad-schema" as the subject name
    And I select "AVRO" as the schema type
    And I enter the following schema in the editor:
      """
      { "type": "record", "name": }
      """
    Then the validation status should show an error
    And the "Register" button should be disabled

  Scenario: Check compatibility before registering
    Given subject "orders-value" exists with an Avro schema containing fields "id" and "amount"
    When I navigate to register a new version of "orders-value"
    And I add a new field "currency" with type "string" and a default value
    And I click "Check Compatibility"
    Then I should see "Compatible" in the compatibility result

  Scenario: Incompatible schema shows detailed error
    Given subject "orders-value" exists with an Avro schema containing fields "id" and "amount"
    And the compatibility level for "orders-value" is "BACKWARD"
    When I navigate to register a new version of "orders-value"
    And I remove the "amount" field from the schema
    And I click "Check Compatibility"
    Then I should see "Incompatible" in the compatibility result
    And the result should include a message about the removed field

  Scenario: Start from latest version pre-fills the editor
    Given subject "orders-value" exists at version 3
    When I navigate to register a new version of "orders-value"
    And I click "Start from latest"
    Then the editor should contain the schema from version 3

  Scenario: Register a Protobuf schema
    When I navigate to the register schema page
    And I enter "users-proto" as the subject name
    And I select "PROTOBUF" as the schema type
    And I enter the following schema in the editor:
      """
      syntax = "proto3";
      package com.example;

      message User {
        string id = 1;
        string name = 2;
        string email = 3;
      }
      """
    And I click "Register"
    Then I should see a success toast "Schema registered as version 1"

  Scenario: Register a JSON Schema
    When I navigate to the register schema page
    And I enter "events-json" as the subject name
    And I select "JSON" as the schema type
    And I enter the following schema in the editor:
      """
      {
        "type": "object",
        "properties": {
          "event_id": {"type": "string"},
          "timestamp": {"type": "string", "format": "date-time"}
        },
        "required": ["event_id", "timestamp"]
      }
      """
    And I click "Register"
    Then I should see a success toast "Schema registered as version 1"

  Scenario: Register a schema with references
    Given subject "common-types-value" exists with an Avro schema for "Address"
    When I navigate to the register schema page
    And I enter "orders-value" as the subject name
    And I select "AVRO" as the schema type
    And I add a reference to "common-types-value" version 1 named "com.example.Address"
    And I enter a schema that uses the "Address" type
    And I click "Register"
    Then I should see a success toast "Schema registered as version 1"

  Scenario: Readonly user cannot register schemas
    Given I am signed in as "test-reader" with role "readonly"
    When I navigate to the subjects page
    Then I should NOT see the "Register New Schema" button
    When I directly navigate to "/ui/subjects/test/register"
    Then I should see a "Permission denied" message

  Scenario: Duplicate schema registration returns existing ID
    Given subject "orders-value" exists with a specific schema at version 1
    When I register the exact same schema to "orders-value"
    Then I should see a toast indicating the schema already exists with its ID
```

### `features/authoring/delete-schema.feature`

```gherkin
@profile:basic
Feature: Delete schemas

  Background:
    Given I am signed in as "test-admin" with role "admin"
    And subject "orders-value" exists with 3 versions

  Scenario: Soft-delete a specific version
    When I navigate to version 2 of subject "orders-value"
    And I click "Delete Version"
    Then I should see a confirmation dialog
    When I click "Confirm" in the dialog
    Then I should see a success toast "Version 2 of 'orders-value' deleted"
    And version 2 should be marked as deleted

  Scenario: Soft-delete an entire subject
    When I navigate to the subject detail page for "orders-value"
    And I click "Delete Subject"
    Then I should see a confirmation dialog asking me to type the subject name
    When I type "orders-value" in the confirmation input
    And I click "Confirm" in the dialog
    Then I should see a success toast "Subject 'orders-value' deleted"
    And I should be redirected to the subjects list
    And "orders-value" should not be visible by default

  Scenario: Cancel delete does nothing
    When I navigate to version 2 of subject "orders-value"
    And I click "Delete Version"
    And I click "Cancel" in the confirmation dialog
    Then the dialog should close
    And version 2 should still be active

  Scenario: Permanent delete requires explicit confirmation
    Given version 2 of "orders-value" has been soft-deleted
    When I navigate to version 2 of subject "orders-value" with deleted=true
    And I click "Permanently Delete"
    Then I should see a confirmation dialog warning that this is irreversible
    When I type "orders-value" in the confirmation input
    And I click "Permanently Delete" in the dialog
    Then I should see a success toast "Version 2 permanently deleted"

  Scenario: Developer cannot permanently delete
    Given I am signed in as "test-developer" with role "developer"
    And version 2 of "orders-value" has been soft-deleted
    When I navigate to version 2 of subject "orders-value"
    Then I should NOT see a "Permanently Delete" button

  Scenario: Cannot delete version that is referenced by other schemas
    Given version 1 of "common-types-value" is referenced by "orders-value" v2
    When I navigate to version 1 of subject "common-types-value"
    And I click "Delete Version"
    And I confirm the deletion
    Then I should see an error "Cannot delete: referenced by orders-value"
```

### `features/authoring/compare-versions.feature`

```gherkin
@profile:basic
Feature: Compare schema versions

  Background:
    Given I am signed in as "test-developer" with role "developer"
    And subject "orders-value" exists with the following versions:
      | version | fields                          |
      | 1       | id, amount                      |
      | 2       | id, amount, currency            |
      | 3       | id, amount, currency, customer  |

  Scenario: Side-by-side diff shows added fields
    When I navigate to version 3 of subject "orders-value"
    And I select version 2 from the "Compare with" dropdown
    And I click "Compare"
    Then I should see a diff viewer with two panels
    And the right panel should highlight the "customer" field as added
    And the left panel should show version 2

  Scenario: Diff shows removed fields
    When I navigate to version 2 of subject "orders-value"
    And I select version 3 from the "Compare with" dropdown
    And I click "Compare"
    Then the left panel should highlight the "customer" field as removed

  Scenario: Comparing identical versions shows no changes
    When I navigate to version 2 of subject "orders-value"
    And I select version 2 from the "Compare with" dropdown
    And I click "Compare"
    Then I should see a message "No differences between versions"
```

---

## Schema-Format-Specific Rendering

How the UI should handle format-specific concerns when displaying and editing schemas.

### Avro Specifics
- **Display:** Pretty-print JSON with 2-space indentation. Highlight `"type"`, `"name"`, `"namespace"`, `"fields"`, `"doc"` as keywords
- **Nullable fields:** Display `["null", "string"]` unions clearly — consider showing a "nullable" badge next to the field name in a future tree view
- **Logical types:** Recognize and display human-readable labels: `"logicalType": "timestamp-millis"` → show "timestamp (ms)" in tree view
- **Editor validation:** Parse as JSON. Additionally validate that `type` is one of: null, boolean, int, long, float, double, bytes, string, record, enum, array, map, union, fixed
- **Download extension:** `.avsc`

### Protobuf Specifics
- **Display:** Use the custom protobuf language mode registered with Monaco (see Monaco config section)
- **Reserved fields:** Highlight `reserved` blocks visually — these are critical for safe evolution and should be prominent
- **Imports:** `import` statements map to schema references. Show them as clickable links in the references panel
- **Syntax:** Detect `proto2` vs `proto3` from the `syntax` line and validate accordingly (proto2 requires `required`/`optional`, proto3 does not)
- **Editor validation:** Basic syntax validation (balanced braces, valid keywords). Server does full validation
- **Download extension:** `.proto`

### JSON Schema Specifics
- **Display:** Pretty-print JSON with 2-space indentation
- **`$ref` links:** Internal `$ref` values (e.g., `"$ref": "#/$defs/Address"`) should be navigable within the editor. External `$ref` to other subjects should render as links in the references panel
- **`additionalProperties`:** Show a visual indicator (open/closed lock icon) to indicate whether the schema allows extra fields — this is critical for understanding compatibility behavior
- **Composition:** `allOf`, `anyOf`, `oneOf` — these should be visible in the tree view (Phase 5)
- **Download extension:** `.json`

---

## Confirmation Dialog Patterns

Two levels of confirmation used throughout the UI:

### Simple Confirm (for reversible actions)

```
┌──────────────────────────────────────┐
│  Delete version 2?             [✕]  │ data-testid="confirm-dialog"
│                                      │
│  This will soft-delete version 2     │
│  of "orders-value". You can          │
│  restore it later.                   │
│                                      │
│       [Cancel]    [Delete]           │ data-testid="confirm-dialog-cancel-btn"
│                                      │   data-testid="confirm-dialog-confirm-btn"
└──────────────────────────────────────┘
```

### Destructive Confirm (for irreversible actions)

Requires typing the resource name to proceed.

```
┌──────────────────────────────────────┐
│  ⚠️ Permanently delete?        [✕] │ data-testid="confirm-dialog"
│                                      │
│  This will PERMANENTLY delete        │
│  subject "orders-value" and all      │
│  its versions. This cannot be        │
│  undone.                             │
│                                      │
│  Type "orders-value" to confirm:     │
│  ┌──────────────────────────────┐    │ data-testid="confirm-dialog-name-input"
│  │                              │    │
│  └──────────────────────────────┘    │
│                                      │
│  [Cancel]  [Permanently Delete]      │ Confirm button disabled until
│                                      │ input matches exactly
└──────────────────────────────────────┘
```

---

## Toast Notification System

Toasts appear in the bottom-right corner. Stack when multiple are active.

```typescript
// ui/src/components/toast.tsx — use shadcn/ui's Sonner integration

import { toast } from 'sonner';

// Success — auto-dismiss 5 seconds
toast.success('Schema registered as version 5 (ID: 47)');

// Error — manual dismiss only
toast.error('Something went wrong. Please try again.');

// Warning — auto-dismiss 8 seconds
toast.warning('This exact schema is already registered as version 3');

// data-testid attributes:
// toast-success, toast-error, toast-warning
```

---

## Responsive Breakpoints

| Breakpoint | Width | Sidebar | Table columns | Editor height |
|------------|-------|---------|---------------|---------------|
| Desktop | ≥1280px | Expanded (240px) | All columns visible | 500px |
| Tablet | 768–1279px | Collapsed (icons only, expand on hover) | Hide least important columns | 400px |
| Mobile | <768px | Hidden (hamburger toggle) | Card layout instead of table | 300px |

Mobile-specific adaptations:
- Tables become card lists (each row = a card)
- Monaco editor goes full-width
- Modals become full-screen drawers (slide up from bottom)
- Sidebar uses overlay mode (slide in from left)

---

## Error Message Catalogue

Every error message the UI can display, where it appears, and what triggers it.

| Context | Trigger | Message | Type | Dismissal |
|---------|---------|---------|------|-----------|
| Login | Wrong credentials (401) | "Invalid username or password" | Inline | — |
| Login | Rate limited (429) | "Too many login attempts. Please wait and try again." | Inline | — |
| Login | Expired API key | "This API key has expired" | Inline | — |
| Login | Server error (500) | "Unable to connect to the registry. Please try again." | Inline | — |
| Session | Token expired | "Your session has expired. Please sign in again." | Banner on login page | — |
| Navigation | 403 on direct URL | "You don't have permission to access this page" | Full page | — |
| Any page | 404 on resource | "This resource was not found" | Full page | — |
| Any page | 500 on API call | "Something went wrong. Please try again." | Toast (error) | Manual |
| Any page | Network error | "Unable to reach the server. Check your connection." | Toast (error) | Manual |
| Register | Invalid schema syntax | (inline in editor as red squiggles + status badge) | Inline | — |
| Register | Schema incompatible (422) | Server message, e.g., "Field 'email' was removed..." | Inline result | — |
| Register | Duplicate schema (409) | "This exact schema is already registered as version {n}" | Toast (warning) | Auto 8s |
| Register | Schema registered OK | "Schema registered as version {n} (ID: {id})" | Toast (success) | Auto 5s |
| Delete | Soft delete OK | "Version {n} of '{subject}' deleted" | Toast (success) | Auto 5s |
| Delete | Cannot delete (has references) | "Cannot delete: referenced by {list}" | Toast (error) | Manual |
| Delete | Permanent delete OK | "Version {n} permanently deleted" | Toast (success) | Auto 5s |
| Config | Compatibility saved | "Compatibility level updated to {level}" | Toast (success) | Auto 5s |
| Config | Mode saved | "Mode updated to {mode}" | Toast (success) | Auto 5s |
| Users | User created | "User '{username}' created" | Toast (success) | Auto 5s |
| Users | Duplicate username (409) | "A user with this username already exists" | Inline on form | — |
| Users | User deleted | "User '{username}' deleted" | Toast (success) | Auto 5s |
| API Keys | Key created | "API key created. Copy it now — it won't be shown again." | Inline in modal | — |
| API Keys | Key revoked | "API key '{name}' revoked" | Toast (success) | Auto 5s |
| API Keys | Key rotated | "API key '{name}' rotated. Copy the new key." | Inline in modal | — |
| Password | Changed OK | "Password changed successfully" | Toast (success) | Auto 5s |
| Password | Wrong current password | "Current password is incorrect" | Inline on form | — |
| Import | Mode not IMPORT | "The registry must be in IMPORT mode to import schemas" | Inline warning | — |
| Import | Import succeeded | "Successfully imported {n} schemas" | Toast (success) | Auto 5s |
| Import | Partial failure | "{n} schemas imported, {m} failed" | Toast (warning) | Manual |
