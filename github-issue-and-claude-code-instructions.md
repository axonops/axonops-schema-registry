# GitHub Issue #273 — Body

Paste the following as the issue description:

---

## Summary

Add an optional, built-in Web UI to the schema registry, served from `/ui/` by the Go binary itself. The UI is a React SPA embedded via Go's `embed` directive. When `ui.enabled: true` in config, users get a full management console for browsing schemas, registering and evolving schemas, managing compatibility/modes, and administering users and API keys — all through the browser.

## Design Decisions

- **UI template base:** [satnaing/shadcn-admin](https://github.com/satnaing/shadcn-admin) (MIT licensed). Use this as the starting point for layout, sidebar, command palette, dark mode, and component patterns. Strip out the demo pages and replace with schema registry features.
- **Stack:** Vite + React 18+ + TypeScript + shadcn/ui + Tailwind CSS + TanStack Router + TanStack Query + Lucide icons + Monaco Editor (for schema editing/diffing)
- **Auth model:** The Go backend issues signed JWT session tokens via new `/ui/auth/*` endpoints. The SPA stores tokens in React state (memory only). See the auth architecture doc linked below.
- **BDD testing:** All interactive elements carry `data-testid` attributes. Tests use Playwright + Cucumber.js (Gherkin) + TypeScript. Multiple docker-compose test profiles cover different auth backends (basic, LDAP, OIDC, noauth).

## Documentation

The full requirements, auth architecture, and testing strategy are in three documents:

1. **[Product Requirements (ux-requirements-v2.md)](link-to-file)** — Feature-driven spec organised by domain (Auth, Browsing, Authoring, Governance, Administration). Each feature has user stories, capabilities (C1.1, C7.3, etc.), acceptance criteria, schema-format-specific notes, and `data-testid` attributes. Capabilities map 1:1 to BDD scenarios.

2. **[Auth Architecture & Testing Strategy (auth-architecture-and-testing.md)](link-to-file)** — How the session-token bridge works across all auth methods (local DB, LDAP, OIDC, API key, mTLS) with sequence diagrams. Docker-compose test profiles for each auth backend with pre-seeded users, LDIF files, Keycloak realm configs, and example Gherkin features.

## Delivery Phases

| Phase | Theme | Features |
|-------|-------|----------|
| **1** | See & Sign In | Login (basic + LDAP + API key), navigation shell, subjects list, subject detail, schema version viewer, schema browser, about page |
| **2** | Author & Evolve | Schema registration with Monaco editor, real-time validation, compatibility checking, schema references, version diff, delete (soft + permanent) |
| **3** | Govern | Global/subject compatibility config, global/subject modes, schema import |
| **4** | Administer | User CRUD, API key CRUD (create/revoke/rotate), self-service profile & password change |
| **5** | Delight | OIDC/SAML login, command palette, schema templates, evolution timeline, statistics dashboard, bulk operations, export |

## Labels

`enhancement`, `component/ui`

---

# Comment for Claude-Code Implementation Instructions

Post the following as a comment on the issue:

---

## Implementation Guide for Claude-Code

### Getting Started

1. **Read the full requirements first.** The product requirements doc and auth architecture doc define everything — features, capabilities, acceptance criteria, test IDs, edge cases, and schema-format quirks. Don't start coding without reading both.

2. **Clone and study `satnaing/shadcn-admin`** (https://github.com/satnaing/shadcn-admin) as the UI template base. It provides:
   - Collapsible sidebar with nav groups and role-based visibility
   - Top bar with user menu
   - Command palette (Ctrl+K) — we'll use this in Phase 5
   - Dark/light mode with system preference detection
   - Responsive layout (desktop sidebar, tablet collapse, mobile hamburger)
   - Login/auth pages
   - User management table pages
   - Settings pages
   - Toast notifications
   - Confirmation dialogs
   - Loading/empty/error states
   
   Strip out the demo content (tasks, chats, dashboard widgets) and replace with our schema registry features. Keep the layout shell, sidebar data structure, auth context pattern, and component primitives.

3. **Tech stack — use exactly this:**
   - React 18+ with TypeScript (strict mode)
   - Vite for building (output goes into a `ui/dist/` directory that Go will embed)
   - Tailwind CSS v4 + shadcn/ui components
   - TanStack Router for client-side routing
   - TanStack Query (React Query) for all API data fetching, caching, and mutations
   - Monaco Editor (`@monaco-editor/react`) for schema viewing, editing, and diffing
   - Lucide React for icons
   - No Next.js, no SSR — this is a pure client-side SPA

### Frontend Directory Structure

Place the UI source under `ui/` in the repository root:

```
ui/
├── src/
│   ├── api/                     # API client functions
│   │   ├── client.ts            # Base fetch wrapper with auth header injection
│   │   ├── subjects.ts          # Subject API calls
│   │   ├── schemas.ts           # Schema API calls
│   │   ├── config.ts            # Compatibility config API calls
│   │   ├── modes.ts             # Mode API calls
│   │   ├── users.ts             # User management API calls
│   │   ├── apikeys.ts           # API key management API calls
│   │   └── auth.ts              # Auth endpoints (/ui/auth/*)
│   ├── components/
│   │   ├── ui/                  # shadcn/ui primitives (button, dialog, table, etc.)
│   │   ├── layout/              # Shell, sidebar, topbar, status bar
│   │   ├── schema-editor/       # Monaco editor wrapper with format-specific config
│   │   ├── schema-diff/         # Monaco diff viewer wrapper
│   │   └── shared/              # Toasts, confirm dialogs, loading states, breadcrumbs
│   ├── features/                # Feature modules (1 per domain)
│   │   ├── auth/                # Login page, auth context, session management
│   │   ├── subjects/            # Subject list, subject detail
│   │   ├── schemas/             # Schema version detail, schema browser
│   │   ├── authoring/           # Register schema, evolve schema
│   │   ├── config/              # Compatibility settings
│   │   ├── modes/               # Mode management
│   │   ├── admin/               # User management, API key management
│   │   ├── account/             # My profile, my API keys
│   │   └── about/               # Cluster info
│   ├── hooks/                   # Shared React hooks
│   ├── lib/                     # Utilities (date formatting, role checks, etc.)
│   ├── types/                   # TypeScript types for API responses
│   ├── routes.tsx               # TanStack Router route tree
│   └── main.tsx                 # Entry point
├── index.html
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
├── package.json
└── components.json              # shadcn/ui config
```

### API Client Pattern

All API calls go through a central client that injects the auth token:

```typescript
// ui/src/api/client.ts
import { useAuthStore } from '@/features/auth/auth-store';

const BASE_URL = ''; // same origin — no prefix needed

export async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const token = useAuthStore.getState().token;
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options?.headers,
  };
  if (token) {
    (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${BASE_URL}${path}`, { ...options, headers });

  if (response.status === 401) {
    useAuthStore.getState().clearSession();
    window.location.href = '/ui/login';
    throw new Error('Unauthorized');
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new ApiError(response.status, error.message || response.statusText, error);
  }

  if (response.status === 204) return undefined as T;
  return response.json();
}
```

### Auth Implementation (Phase 1)

The auth flow requires Go backend changes. Implement these endpoints in the Go codebase:

```
GET  /ui/auth/config    → { "methods": ["basic","api_key"], "ldap_enabled": true/false }
POST /ui/auth/login     → accepts {"username","password"}, returns {"token","expires_at","user"}
POST /ui/auth/apikey    → accepts {"key"}, returns {"token","expires_at","user"}
GET  /ui/auth/session   → validates Bearer token, returns user info (used for token refresh)
POST /ui/auth/logout    → invalidates token (add jti to short deny-list until expiry)
```

These endpoints call the **existing** auth validation logic (same bcrypt comparison, same LDAP bind, same API key hash lookup) and wrap the result in a signed JWT. The existing API auth middleware must be updated to **also** accept these UI-issued JWTs alongside all existing auth methods.

Session token JWT claims: `{ sub, role, email, auth_method, iat, exp, jti }`

Signing: HMAC-SHA256 using `ui.session.secret` from config. Default TTL: 30 minutes.

### Go-Side: Embedding the SPA

Add to the Go binary:

```go
//go:embed ui/dist/*
var uiFS embed.FS

// In your router setup:
if config.UI.Enabled {
    // Serve SPA assets
    uiHandler := http.FileServer(http.FS(uiFS))
    mux.Handle("/ui/", http.StripPrefix("/ui/", uiHandler))
    
    // SPA fallback: any /ui/* path that doesn't match a static file
    // should return index.html (for client-side routing)
    // ...
    
    // Auth endpoints
    mux.HandleFunc("GET /ui/auth/config", handleAuthConfig)
    mux.HandleFunc("POST /ui/auth/login", handleLogin)
    mux.HandleFunc("POST /ui/auth/apikey", handleApiKeyLogin)
    mux.HandleFunc("GET /ui/auth/session", handleSession)
    mux.HandleFunc("POST /ui/auth/logout", handleLogout)
}
```

The Vite build must output to `ui/dist/`. The Go build must run `cd ui && npm run build` first, then `go build`. Add this to the Makefile.

### BDD Test Instrumentation — CRITICAL

**Every interactive element MUST have a `data-testid` attribute.** This is non-negotiable. The naming convention is `{area}-{element}-{qualifier}`.

Examples:
- `data-testid="login-username-input"`
- `data-testid="subjects-list-table"`
- `data-testid="subject-register-version-btn"`
- `data-testid="register-schema-editor"`
- `data-testid="toast-success"`
- `data-testid="confirm-dialog-confirm-btn"`
- `data-testid="nav-sidebar-subjects-link"`

State indicators:
- `data-testid="subjects-list-loading"` (skeleton)
- `data-testid="subjects-list-empty"` (empty state)
- `data-testid="subjects-list-error"` (error with retry)

The full list of required test IDs is in the product requirements doc under each capability. Removing or renaming a `data-testid` is a breaking change.

### BDD Test Structure

```
tests/e2e/
├── profiles/                    # Docker-compose stacks per auth config
│   ├── basic/                   # Local DB auth only
│   │   ├── docker-compose.yml
│   │   └── config.yaml
│   ├── ldap/                    # Local DB + OpenLDAP
│   │   ├── docker-compose.yml
│   │   ├── config.yaml
│   │   └── seed/bootstrap.ldif
│   ├── oidc/                    # Phase 5: Local DB + Keycloak
│   │   ├── docker-compose.yml
│   │   ├── config.yaml
│   │   └── seed/schema-registry-realm.json
│   └── noauth/                  # Auth disabled
│       ├── docker-compose.yml
│       └── config.yaml
├── features/                    # Gherkin .feature files (grouped by domain)
│   ├── auth/
│   ├── browsing/
│   ├── authoring/
│   ├── governance/
│   ├── admin/
│   └── navigation/
├── steps/                       # TypeScript step definitions
├── pages/                       # Page Object classes
├── support/
│   ├── world.ts                 # Cucumber World with Playwright browser
│   ├── hooks.ts                 # Before/After hooks (data seeding/cleanup)
│   └── api-helpers.ts           # Direct API calls for test data seeding
└── playwright.config.ts
```

Features use `@profile:` tags to control which profiles they run against (e.g., `@profile:basic @profile:ldap`). Separate GitHub Actions jobs run each profile in parallel.

Test data is seeded via direct API calls in Before hooks, not through the UI. This keeps tests fast and focused on the capability being tested.

### Phase 1 Implementation Order

Build in this order — each step is independently testable:

1. **Vite project scaffolding** — set up the project under `ui/`, install dependencies, configure Vite to output to `ui/dist/`, verify the build produces embeddable static files
2. **Go embedding** — wire up the `embed.FS`, SPA fallback handler, and `ui.enabled` config flag. Verify `/ui/` serves `index.html`
3. **Auth endpoints** — implement `/ui/auth/config`, `/ui/auth/login`, `/ui/auth/session`, `/ui/auth/logout` in Go. Write the JWT signing/validation. Update auth middleware to accept UI tokens
4. **Login page** — React login form, auth context/store, token management, redirect-after-login
5. **Application shell** — sidebar (from shadcn-admin template), top bar with user menu, role-based nav visibility, status bar footer
6. **Subjects list page** — table, search, pagination, deleted toggle, empty state
7. **Subject detail page** — version table, metadata panel, latest schema preview
8. **Schema version detail** — Monaco read-only viewer, metadata, copy/download, references, referenced-by
9. **Schema browser** — global ID lookup, browse with filters
10. **About page** — server version, cluster ID, storage type, schema types
11. **BDD test infrastructure** — Playwright + Cucumber setup, `basic` profile docker-compose, page objects, first feature files for login and subjects browsing

### What NOT To Do

- **Don't use Next.js.** This is a client-side SPA embedded in a Go binary. No SSR.
- **Don't use localStorage or cookies** for auth tokens. Memory only (Zustand or React context).
- **Don't create a separate backend or proxy.** The SPA talks to the same Go server on the same origin.
- **Don't skip `data-testid` attributes.** Every interactive element needs one.
- **Don't hard-code users or roles in the frontend.** Everything comes from the API / session token.
- **Don't implement OIDC/SAML in Phase 1.** The auth architecture supports it, but the UI login flow and Go endpoints for OIDC are Phase 5.
- **Don't use `WidthType.PERCENTAGE`** — wrong doc, ignore that. (This is a React app, not a docx.)
