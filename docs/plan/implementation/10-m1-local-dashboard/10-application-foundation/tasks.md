# Tasks — Application foundation

- [ ] <a id="t10-001"></a>**T10-001 — Introduce settings and application factory**
  - Product: `PR-UX-01` · Evidence: `RQ-001`
  - Depends on: — · Code: target settings/factory modules, `app/main.py`
  - Verify: temporary-settings core test
  - Accept when: the public entrypoint remains `app.main:app` and tests do not use the default database.
  - Commit: `refactor: introduce settings and application factory`
  - Evidence: [execution entry](execution.md#t10-001)

- [ ] <a id="t10-002"></a>**T10-002 — Split routes, services, and storage wiring**
  - Product: `PR-UX-01` · Evidence: `RQ-001`
  - Depends on: `T10-001` · Code: `app/main.py`, `app/services.py`, target route/storage modules
  - Verify: API flow and import-graph tests
  - Accept when: routes validate input and domain services own business behavior.
  - Commit: `refactor: split api routes from domain services`
  - Evidence: [execution entry](execution.md#t10-002)

- [ ] <a id="t10-003"></a>**T10-003 — Make lifespan and error handling explicit**
  - Product: `PR-UX-01` · Evidence: `RQ-013`
  - Depends on: `T10-001`, `T10-002` · Code: application lifespan and exception handlers
  - Verify: startup/shutdown and error-envelope tests
  - Accept when: initialization runs once, shutdown is graceful, and failures use the shared contract.
  - Commit: `fix: make startup shutdown and template errors explicit`
  - Evidence: [execution entry](execution.md#t10-003)

- [ ] <a id="t10-004"></a>**T10-004 — Verify application factory and lifecycle**
  - Product: `PR-UX-01` · Evidence: `RQ-001`, `RQ-013`
  - Depends on: `T10-003` · Code: tests and smoke command
  - Verify: `uv run pytest -q` and `uv run ruff check .`
  - Accept when: the relevant baseline suite passes and no task remains unverified.
  - Commit: `test: cover app factory and lifespan`
  - Evidence: [execution entry](execution.md#t10-004)
