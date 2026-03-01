Read the attached `implementation-specs.md` file. This is a companion document to the existing `docs/ui/ux-requirements.md` in the repo.

Do the following:

1. Commit `implementation-specs.md` to the repo as `docs/ui/implementation-specs.md` on the `main` branch.
   
   Commit message: "docs: add detailed Web UI implementation specs — page layouts, API mappings, TypeScript types, BDD features (#273)"

2. Add a reference to it at the top of `docs/ui/ux-requirements.md` by inserting the following after the opening blockquote:
   
   ```
   > **Implementation detail:** See [`implementation-specs.md`](implementation-specs.md) for page layouts (ASCII wireframes), exact API call specs, TypeScript types, Monaco Editor configuration, form validation rules, BDD Gherkin features, API client architecture, and the error message catalogue.
   ```

3. Add a comment on issue #273 summarising what was added:

   ```
   Added `docs/ui/implementation-specs.md` — companion to the UX requirements providing implementation-level detail:

   - **URL route table** — every SPA route with component, access control, and API calls on mount
   - **TypeScript API types** — full interfaces for all request/response shapes
   - **Page layouts** — ASCII wireframes for all 12 pages (login, app shell, subjects list, subject detail, schema version, register schema, schema browser, user management, API key management, global config, import, about, profile)
   - **Form validation rules** — every field, constraint, and error message
   - **Monaco Editor config** — language modes, protobuf syntax registration, editor options
   - **API call specifications** — exact request/response for every major operation
   - **API client architecture** — central `apiFetch()` wrapper, TanStack Query keys/hooks, auth context
   - **BDD feature files** — complete Gherkin for auth (login, API key, session), browsing (subjects list, schema version, navigation), and authoring (register, delete, compare)
   - **Error message catalogue** — every error the UI can display with trigger and type
   - **Schema-format-specific rendering** — Avro, Protobuf, JSON Schema display and editing notes
   - **Responsive breakpoints** — desktop/tablet/mobile layout adaptations
   - **Confirmation dialog patterns** — simple confirm vs destructive confirm with name typing
   - **Toast notification system** — success/error/warning patterns
   ```

Use the `gh` CLI for all GitHub operations. The repo is `axonops/axonops-schema-registry`.
