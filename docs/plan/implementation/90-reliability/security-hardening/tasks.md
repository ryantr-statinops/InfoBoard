# Security and privacy hardening: tasks

This is the sole authoritative checklist for REL-SECURITY.

## REL-SECURITY-001 — Harden local HTTP and browser mutation boundaries

- [ ] **REL-SECURITY-001: Harden local HTTP and browser mutation boundaries**
- Outcome: Harden local HTTP and browser mutation boundaries.
- Prerequisites: API-CANONICAL, CAP-URL, SEM-CONFIG, ORG-UI, REL-RECOVERY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: HTTP middleware, capture/provider clients, renderers/templates, configuration, security tests.
- Actions:
  1. Validate host/origin policy, apply CSRF protections where required, constrain binding defaults, reject unknown fields, and use safe headers.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Host poisoning, cross-origin mutation, CSRF, malformed header, and local-bind tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-security-001) are reviewed.
- Commit boundary: one to three scoped commits.

## REL-SECURITY-002 — Harden content, provider, and resource handling

- [ ] **REL-SECURITY-002: Harden content, provider, and resource handling**
- Outcome: Harden content, provider, and resource handling.
- Prerequisites: API-CANONICAL, CAP-URL, SEM-CONFIG, ORG-UI, REL-RECOVERY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: HTTP middleware, capture/provider clients, renderers/templates, configuration, security tests.
- Actions:
  1. Enforce SSRF/redirect/size/time bounds, sanitize rendering, gate consent at transfer, and redact every secret/content path.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: XSS, decompression, rebinding, consent race, log/response redaction, and resource exhaustion tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-security-002) are reviewed.
- Commit boundary: one to three scoped commits.

## REL-SECURITY-003 — Harden backup, maintenance, and filesystem boundaries

- [ ] **REL-SECURITY-003: Harden backup, maintenance, and filesystem boundaries**
- Outcome: Harden backup, maintenance, and filesystem boundaries.
- Prerequisites: API-CANONICAL, CAP-URL, SEM-CONFIG, ORG-UI, REL-RECOVERY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: HTTP middleware, capture/provider clients, renderers/templates, configuration, security tests.
- Actions:
  1. Validate target paths and archive entries, prevent unsafe overwrite/traversal, constrain permissions, and keep maintenance actions explicit.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Symlink, traversal, broad path, permission, corrupt archive, and unauthorized action tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-security-003) are reviewed.
- Commit boundary: one to three scoped commits.
