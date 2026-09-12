# Tasks — Packaging, release, and upgrades

- [ ] <a id="t21-001"></a>**T21-001 — Define core/full dependency and mode contracts**
  - Product: `PR-OPS-01` · Evidence: `RQ-014`
  - Depends on: 10, 15 · Code: target dependency extras and mode validation
  - Verify: clean core install and explicit full-mode missing-dependency behavior
  - Accept when: core has no optional-store requirement and full mode is explicit.
  - Commit: `chore: define core and full dependency extras`
  - Evidence: [execution entry](execution.md#t21-001)

- [ ] <a id="t21-002"></a>**T21-002 — Add model preparation and environment validation**
  - Product: `PR-OPS-01` · Evidence: `RQ-014`
  - Depends on: `T21-001` · Code: target preparation/validation command
  - Verify: missing model, wrong Python, and locked dependency scenarios
  - Accept when: model preparation is explicit and failures are actionable.
  - Commit: `feat: add model preparation and environment validation`
  - Evidence: [execution entry](execution.md#t21-002)

- [ ] <a id="t21-003"></a>**T21-003 — Document release, upgrade, and rollback**
  - Product: `PR-OPS-01…02` · Evidence: `RQ-014`
  - Depends on: `T18-002`, `T20-005` · Code: release checklist and operator guide
  - Verify: clean-checkout and current-DB-copy walkthrough
  - Accept when: every release action has expected result, failure handling, and rollback.
  - Commit: `docs: add release upgrade and rollback instructions`
  - Evidence: [execution entry](execution.md#t21-003)

- [ ] <a id="t21-004"></a>**T21-004 — Verify release smoke and artifact contents**
  - Product: `PR-OPS-01…02` · Evidence: `RQ-014`
  - Depends on: `T21-003` · Code: release smoke workflow
  - Verify: install, migrate, health, dashboard, and rollback transcript
  - Accept when: clean core release and supported full-mode behavior are evidenced.
  - Commit: `ci: add release smoke verification`
  - Evidence: [execution entry](execution.md#t21-004)
