# Tasks — Dashboard and item workspace

- [ ] <a id="t12-001"></a>**T12-001 — Build the dashboard shell and item list**
  - Product: `PR-UX-01` · Evidence: `RQ-001`
  - Depends on: `T10-002`, `T11-002` · Code: target dashboard routes/templates
  - Verify: list, empty, and loading UI tests
  - Accept when: the dashboard renders a safe empty state and a persisted item list.
  - Commit: `feat: add dashboard shell and item list`
  - Evidence: [execution entry](execution.md#t12-001)

- [ ] <a id="t12-002"></a>**T12-002 — Add capture, detail, edit, and soft delete flows**
  - Product: `PR-CAP-01`, `PR-CAP-04`, `PR-ORG-03`, `PR-ORG-04`, `PR-UX-01`, `PR-UX-03` · Evidence: `RQ-001`
  - Depends on: `T12-001` · Code: item API, detail fragment, validation/error states
  - Verify: API flow and reload tests
  - Accept when: create/edit/delete behavior survives reload and deleted items leave public lists.
  - Commit: `feat: implement item workspace actions`
  - Evidence: [execution entry](execution.md#t12-002)

- [ ] <a id="t12-003"></a>**T12-003 — Add collection assignment and filters**
  - Product: `PR-ORG-01`, `PR-ORG-02`, `PR-UX-02` · Evidence: `RQ-002`
  - Depends on: `T12-002` · Code: collection controls and filtered list
  - Verify: collection attach/filter integration test
  - Accept when: many-to-many assignment and filtering preserve item state.
  - Commit: `feat: add collections and dashboard filters`
  - Evidence: [execution entry](execution.md#t12-003)

- [ ] <a id="t12-004"></a>**T12-004 — Add notes and item context**
  - Product: `PR-ORG-03` · Evidence: `RQ-003`
  - Depends on: `T12-002` · Code: notes UI and CRUD routes
  - Verify: note create/edit/delete integration test
  - Accept when: notes remain attached to the item and are visible after reload.
  - Commit: `feat: add item notes and context panel`
  - Evidence: [execution entry](execution.md#t12-004)

- [ ] <a id="t12-005"></a>**T12-005 — Verify responsive and degraded dashboard states**
  - Product: `PR-UX-03`, `PR-UX-04`, `PR-REC-04` · Evidence: `RQ-013`
  - Depends on: `T12-003`, `T12-004` · Code: UI states and unavailable/degraded capability messaging
  - Verify: browser smoke at 1440 px and 390 px plus component-unavailable scenario
  - Accept when: core dashboard remains understandable and does not claim an unavailable capability is healthy.
  - Commit: `test: cover dashboard responsive and degraded states`
  - Evidence: [execution entry](execution.md#t12-005)
