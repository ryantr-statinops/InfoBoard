# Tasks — Data model and migrations

- [ ] <a id="t11-001"></a>**T11-001 — Add the versioned SQLite migration runner**
  - Product: `PR-REC-01`, `PR-REC-02` · Evidence: `RQ-002`, `RQ-005`
  - Depends on: `T10-001` · Code: target migration runner, `app/db.py`
  - Verify: clean install migration test
  - Accept when: migrations run transactionally and record the schema version.
  - Commit: `feat: add versioned sqlite migration runner`
  - Evidence: [execution entry](execution.md#t11-001)

- [ ] <a id="t11-002"></a>**T11-002 — Normalize content versions and schema indexes**
  - Product: `PR-CAP-02`, `PR-CAP-04`, `PR-ORG-01`, `PR-ORG-02`, `PR-ORG-04`, `PR-REC-01` · Evidence: `RQ-002`, `RQ-005`
  - Depends on: `T11-001` · Code: items, item_contents, chunks, relations
  - Verify: schema/invariant tests and data-preservation fixture
  - Accept when: `(item_id, content_version)`, `chunks.position`, and uniqueness constraints are enforced.
  - Commit: `feat: normalize item content version and indexes`
  - Evidence: [execution entry](execution.md#t11-002)

- [ ] <a id="t11-003"></a>**T11-003 — Make FTS synchronization recoverable**
  - Product: `PR-REC-03` · Evidence: `RQ-006`
  - Depends on: `T11-002` · Code: FTS synchronization and rebuild path
  - Verify: transactional FTS and rebuild tests
  - Accept when: FTS failure cannot delete canonical data and rebuild restores the index.
  - Commit: `feat: make fts synchronization transactional`
  - Evidence: [execution entry](execution.md#t11-003)

- [ ] <a id="t11-004"></a>**T11-004 — Verify fresh install and upgrade migrations**
  - Product: `PR-REC-01`, `PR-REC-02` · Evidence: `RQ-002`, `RQ-005`
  - Depends on: `T11-003` · Code: migration fixtures and tests
  - Verify: fresh, upgrade, rerun, concurrent, delete/restore test matrix
  - Accept when: no rows are lost and all expected schema checks pass.
  - Commit: `test: cover fresh install and upgrade migrations`
  - Evidence: [execution entry](execution.md#t11-004)
