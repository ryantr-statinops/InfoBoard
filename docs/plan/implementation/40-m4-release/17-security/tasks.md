# Tasks — Security and privacy

- [ ] <a id="t17-001"></a>**T17-001 — Enforce local write Host/Origin boundaries**
  - Product: `PR-SEC-01` · Evidence: `RQ-011`
  - Depends on: `T10-003` · Code: request middleware and write routes
  - Verify: valid/invalid Host and Origin integration tests
  - Accept when: unsafe cross-origin writes are rejected without wildcard CORS.
  - Commit: `security: enforce host origin and request boundaries`
  - Evidence: [execution entry](execution.md#t17-001)

- [ ] <a id="t17-002"></a>**T17-002 — Harden URL redirect and network validation**
  - Product: `PR-SEC-01…02` · Evidence: `RQ-011`
  - Depends on: `T13-003` · Code: DNS/socket/redirect checks
  - Verify: public, loopback, private, reserved, and metadata fixtures
  - Accept when: initial and redirected destinations are bounded and safe.
  - Commit: `security: harden url redirect and network validation`
  - Evidence: [execution entry](execution.md#t17-002)

- [ ] <a id="t17-003"></a>**T17-003 — Add rendering and log redaction controls**
  - Product: `PR-SEC-01` · Evidence: `RQ-011`
  - Depends on: `T10-002`, `T13-001` · Code: templates and structured logging
  - Verify: XSS, secret, query, body, and credential-bearing URL tests
  - Accept when: untrusted content renders safely and forbidden values do not appear in logs.
  - Commit: `security: add rendering escaping and log redaction`
  - Evidence: [execution entry](execution.md#t17-003)

- [ ] <a id="t17-004"></a>**T17-004 — Run the security regression suite**
  - Product: `PR-SEC-01…02` · Evidence: `RQ-011`
  - Depends on: `T17-001`, `T17-002`, `T17-003` · Code: security test suite
  - Verify: `uv run pytest -q`
  - Accept when: limits, SSRF, escaping, Origin/Host, and redaction evidence is reviewable.
  - Commit: `test: add ingestion and rendering security cases`
  - Evidence: [execution entry](execution.md#t17-004)
