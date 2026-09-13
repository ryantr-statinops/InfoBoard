# URL validation and normalization: tasks

This file is the sole authoritative checklist for CAP-URL. Do not check an item until its evidence anchor contains real results.

## CAP-URL-001 — Implement normalization policy version 1

- [ ] **CAP-URL-001: Implement normalization policy version 1**
- Outcome: Implement normalization policy version 1.
- Prerequisites: API-CANONICAL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: capture service, URL policy module, HTTP client adapter, security fixtures.
- Actions:
  1. Normalize scheme/host/port/path/query deterministically, remove the approved tracking set, preserve meaningful path semantics, and persist the revision.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Table-driven normalization, repeated-query, IDN, trailing-slash, and duplicate tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-url-001) are reviewed.
- Commit boundary: one to three scoped commits.

## CAP-URL-002 — Enforce SSRF and redirect boundaries

- [ ] **CAP-URL-002: Enforce SSRF and redirect boundaries**
- Outcome: Enforce SSRF and redirect boundaries.
- Prerequisites: API-CANONICAL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: capture service, URL policy module, HTTP client adapter, security fixtures.
- Actions:
  1. Validate scheme, hostname resolution, every redirect, and connected address; reject private/non-public targets and unsafe ports.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: DNS rebinding, redirect-to-private, mixed-address, encoded-host, and timeout tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-url-002) are reviewed.
- Commit boundary: one to three scoped commits.

## CAP-URL-003 — Bound fetch and parsing resources

- [ ] **CAP-URL-003: Bound fetch and parsing resources**
- Outcome: Bound fetch and parsing resources.
- Prerequisites: API-CANONICAL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: capture service, URL policy module, HTTP client adapter, security fixtures.
- Actions:
  1. Apply request, redirect, compressed/decompressed body, content-type, parser, and time limits with safe errors.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Oversize, decompression-bomb, malformed content, and cancellation tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-url-003) are reviewed.
- Commit boundary: one to three scoped commits.
