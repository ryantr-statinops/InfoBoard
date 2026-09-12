# Internal PRD — Requirement traceability

**Status:** `active`

Product requirements are the canonical product intent. This document connects
each requirement group to the five documentation layers that must be locked
before a delivery plan is written.

This is not an implementation status report. Task IDs, evidence requirements,
delivery status, and execution logs will be created later in `docs/plan/` from
this product contract.

## Cross-layer coverage

| Product requirements | Product outcome | Design | Architecture | Quality | Operations | Release class |
| --- | --- | --- | --- | --- | --- | --- |
| `PR-CAP-01…05` | Capture and persistence | [Core flows](../../design/01-core-flows-and-screens.md) | [Ingestion](../../architecture/03-ingestion-pipeline.md), [data model](../../architecture/02-data-model-and-erd.md) | [Test strategy](../../quality/01-test-strategy.md) | [Backup and restore](../../operations/01-backup-restore-and-rebuild.md) | Core |
| `PR-ORG-01…04` | Collections, notes, status, and delete | [Information architecture](../../design/00-information-architecture.md), [core flows](../../design/01-core-flows-and-screens.md) | [Data model](../../architecture/02-data-model-and-erd.md), [lifecycle](../../architecture/06-lifecycle-and-recovery.md) | [Test strategy](../../quality/01-test-strategy.md) | [Backup and restore](../../operations/01-backup-restore-and-rebuild.md) | Core |
| `PR-RET-01…06` | Filtered retrieval and analytics fallback | [Core flows](../../design/01-core-flows-and-screens.md), [UI states](../../design/02-ui-states-and-responsive.md) | [Search and analytics](../../architecture/05-search-and-analytics.md) | [Search evaluation](../../quality/02-search-and-performance-evaluation.md) | [Health and troubleshooting](../../operations/02-health-and-troubleshooting.md) | Core, with optional full-mode extensions |
| `PR-REC-01…05` | Feedback, retry, fallback, and recovery | [UI states](../../design/02-ui-states-and-responsive.md) | [Indexing](../../architecture/04-indexing-pipeline.md), [lifecycle](../../architecture/06-lifecycle-and-recovery.md) | [Quality gates](../../quality/03-mvp-quality-gates.md) | [Health](../../operations/02-health-and-troubleshooting.md), [recovery](../../operations/01-backup-restore-and-rebuild.md) | Core |
| `PR-UX-01…04` | Simple, responsive, and recoverable experience | [Design system](../../design/README.md), [UI states](../../design/02-ui-states-and-responsive.md) | [System overview](../../architecture/00-system-overview.md) | [Quality attributes](../../quality/00-quality-attributes.md) | [Local setup](../../operations/00-local-setup-and-modes.md) | Core |
| `PR-SEC-01…03` | Safe input, network, rendering, and logging boundary | [UI states](../../design/02-ui-states-and-responsive.md) | [Security boundaries](../../architecture/07-security-boundaries.md) | [Test strategy](../../quality/01-test-strategy.md) | [Health and diagnostics](../../operations/02-health-and-troubleshooting.md) | Core |
| `PR-SEC-04` | Explicit consent before cloud-provider transfer | [Advanced mode](../../design/00-information-architecture.md) | [Security boundaries](../../architecture/07-security-boundaries.md) | [Quality attributes](../../quality/00-quality-attributes.md) | [Local setup and modes](../../operations/00-local-setup-and-modes.md) | Future boundary |
| `PR-OPS-01…02` | Repeatable release, upgrade, rollback, and evidence | [Settings/maintenance](../../design/01-core-flows-and-screens.md) | [Lifecycle and recovery](../../architecture/06-lifecycle-and-recovery.md) | [Quality gates](../../quality/03-mvp-quality-gates.md) | [Upgrade and rollback](../../operations/03-upgrade-release-and-rollback.md) | Release |

## Product-to-plan handoff

The future `docs/plan/` must add, for every individual `PR-*`:

- A concrete delivery outcome and milestone.
- The architecture and design decisions it consumes.
- The quality gate and operational procedure required for acceptance.
- A task breakdown and evidence location.
- A clear distinction between core release, conditional full mode, and future
  discovery.

No product requirement is considered implemented because a target document,
diagram, mockup, checklist, adapter, or code example exists.

## Traceability rules

- `PR-*` is the only product requirement ID system.
- Product references describe intent and boundaries; they do not report runtime
  completion.
- Optional full-mode requirements must never block a core-mode release.
- `PR-SEC-04` remains a future trust boundary and is not an MVP gate.
- A requirement change must update Product, the affected Design, Architecture,
  Quality, and Operations references before entering planning.
- Implementation task IDs and execution evidence belong in `docs/plan/` only
  after these five layers are locked.
