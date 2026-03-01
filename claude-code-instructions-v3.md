# Claude Code Instructions — Updating GitHub Issue #273

## Quick Start

```bash
cd /path/to/axonops-schema-registry

# 1. Copy the files into the repo root
cp /path/to/web-ui-requirements-v3.md .

# 2. Update the issue body
gh issue edit 273 --repo axonops/axonops-schema-registry --body-file web-ui-requirements-v3.md

# 3. Create labels
gh label create "web-ui" --repo axonops/axonops-schema-registry --color "1d76db" --description "Web UI feature" 2>/dev/null
gh label create "enhancement" --repo axonops/axonops-schema-registry --color "a2eeef" --description "New feature or request" 2>/dev/null
gh issue edit 273 --repo axonops/axonops-schema-registry --add-label "enhancement,web-ui"
```

## Create Phase Sub-Issues

```bash
REPO="axonops/axonops-schema-registry"

gh issue create --repo $REPO --title "Web UI Phase 0: Prerequisites (Go 1.26, Tags, Downloads, Search backend)" --body "Part of #273

- [ ] Upgrade to Go 1.26, run go fix ./...
- [ ] Enable goroutine leak profiler in CI
- [ ] Implement schema tags backend (/api/v1/subjects/{subject}/tags, /api/v1/tags)
- [ ] Implement download endpoints (/download, /export)
- [ ] Implement search endpoint (/api/v1/search)" --label "web-ui,enhancement"

gh issue create --repo $REPO --title "Web UI Phase 1: Foundation (Scaffold, Embed, Auth, Dashboard)" --body "Part of #273

- [ ] Vite + React + TypeScript + Tailwind + Shadcn/ui scaffold
- [ ] Go //go:embed + /ui/ route + SPA fallback
- [ ] --disable-ui flag + UI_ENABLED env
- [ ] /api/v1/ui/config endpoint
- [ ] Basic Auth login flow + session JWT
- [ ] CSRF protection (SameSite + custom header)
- [ ] Dashboard page
- [ ] Sidebar nav with context switcher
- [ ] Dark/light mode
- [ ] BDD harness + CI pipeline" --label "web-ui,enhancement"

gh issue create --repo $REPO --title "Web UI Phase 2: Schema Operations (Explorer, Editor, Tags)" --body "Part of #273

- [ ] Subject Explorer with tags, filters, sort, pagination
- [ ] Subject Detail with versions, tags editor, downloads
- [ ] Schema registration with Monaco editor
- [ ] Compatibility check with field-level errors
- [ ] Create new subject with tag assignment
- [ ] BDD coverage" --label "web-ui,enhancement"

gh issue create --repo $REPO --title "Web UI Phase 3: Diff, Search & Downloads" --body "Part of #273

- [ ] Schema diff (Monaco Diff Editor + compatibility annotations)
- [ ] Global search (Ctrl+K, all modes: subject, ID, tag, field)
- [ ] Schema downloads (individual, bulk ZIP, diff export)
- [ ] Compatibility Manager (global + per-subject)
- [ ] Mode Manager (admin only)
- [ ] BDD coverage" --label "web-ui,enhancement"

gh issue create --repo $REPO --title "Web UI Phase 4: Contexts & Enterprise Features" --body "Part of #273

- [ ] Context switcher + context-scoped routing
- [ ] DEK Registry / Encryption Manager
- [ ] Exporter Manager
- [ ] Data Contracts viewer/editor
- [ ] BDD coverage" --label "web-ui,enhancement"

gh issue create --repo $REPO --title "Web UI Phase 5: Auth, Admin & Audit" --body "Part of #273

- [ ] Full RBAC enforcement (4 roles)
- [ ] Additional auth: LDAP, OIDC (PKCE), API Keys, mTLS
- [ ] Server Configuration Viewer (secrets redacted)
- [ ] Admin panel
- [ ] Audit log viewer
- [ ] Embedded API Docs (ReDoc)
- [ ] BDD coverage" --label "web-ui,enhancement"

gh issue create --repo $REPO --title "Web UI Phase 6: Polish & Adoption" --body "Part of #273

- [ ] Keyboard shortcuts (Ctrl+K, Ctrl+S, j/k nav)
- [ ] Branding/theming configuration
- [ ] Onboarding wizard
- [ ] Accessibility audit (WCAG 2.1 AA)
- [ ] Responsive testing
- [ ] Full BDD regression (Chromium, Firefox, WebKit)
- [ ] Documentation + screenshots
- [ ] Performance testing" --label "web-ui,enhancement"
```

## Verify

```bash
gh issue view 273 --repo axonops/axonops-schema-registry
gh issue list --repo axonops/axonops-schema-registry --label "web-ui"
```
