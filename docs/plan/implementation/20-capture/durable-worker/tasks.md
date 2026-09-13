# Durable sequential worker: tasks

This file is the sole authoritative checklist for CAP-WORKER. Do not check an item until its evidence anchor contains real results.

## CAP-WORKER-001 — Implement sequential durable claim and lease semantics

- [ ] **CAP-WORKER-001: Implement sequential durable claim and lease semantics**
- Outcome: Implement sequential durable claim and lease semantics.
- Prerequisites: APP-LIFECYCLE, DB-MIGRATION, CAP-SNAPSHOT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/worker.py`, job repositories, application lifespan, worker tests.
- Actions:
  1. Claim one eligible operation at a time with lease owner/expiry and atomic state transitions; keep capture attempts distinct from index jobs.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Competing-claim, ordering, lease expiry, and invalid-transition tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-worker-001) are reviewed.
- Commit boundary: one to three scoped commits.

## CAP-WORKER-002 — Add bounded retry and safe checkpoint recovery

- [ ] **CAP-WORKER-002: Add bounded retry and safe checkpoint recovery**
- Outcome: Add bounded retry and safe checkpoint recovery.
- Prerequisites: APP-LIFECYCLE, DB-MIGRATION, CAP-SNAPSHOT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/worker.py`, job repositories, application lifespan, worker tests.
- Actions:
  1. Use three attempts with documented backoff, preserve checkpoints, expose retryability, and support manual retry after exhaustion.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Clock-controlled backoff, exhaustion, restart, and checkpoint-resume tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-worker-002) are reviewed.
- Commit boundary: one to three scoped commits.

## CAP-WORKER-003 — Integrate worker lifecycle and graceful shutdown

- [ ] **CAP-WORKER-003: Integrate worker lifecycle and graceful shutdown**
- Outcome: Integrate worker lifecycle and graceful shutdown.
- Prerequisites: APP-LIFECYCLE, DB-MIGRATION, CAP-SNAPSHOT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/worker.py`, job repositories, application lifespan, worker tests.
- Actions:
  1. Recover expired work after migrations, stop new claims on shutdown, complete or release current work safely, and expose health.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Lifespan start/stop, crash simulation, and health-state tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-worker-003) are reviewed.
- Commit boundary: one to three scoped commits.
