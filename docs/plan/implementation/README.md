# InfoBoard — Implementation workspace

**Canonical context:** [Product](../../product/internal-prd/README.md) · [Architecture](../../architecture/README.md) · [Design](../../design/README.md) · [Quality](../../quality/README.md) · [Operations](../../operations/README.md)

This is the execution-ready implementation workspace. It is organized by milestone and epic package so that every delivery unit can be followed from requirement to plan, task, code/test verification, evidence, and acceptance.

## Reading order and status

Read in this order: [program roadmap](00-program/roadmap.md) → active milestone → epic `README.md` → `plan.md` → `tasks.md` → `references.md` → optional `examples.md`/`guides/` → `execution.md` for M1–M4 or `evidence.md` for M5+ discovery.

Every epic reports three independent values:

- `Plan status`: `draft | ready`.
- `Delivery status`: `not_started | in_progress | blocked | review | merged | verified`.
- `Baseline coverage`: `missing | partial | complete`.

See [task conventions](00-program/task-conventions.md) for the task, evidence, and commit rules.

## Milestones

- [M1 — Local dashboard](10-m1-local-dashboard/README.md)
- [M2 — Reliable ingestion](20-m2-ingestion/README.md)
- [M3 — Search and insights](30-m3-search-and-insights/README.md)
- [M4 — Hardening and release](40-m4-release/README.md)
- [M5+ — Post-MVP discovery](50-post-mvp/README.md)

## Program and governance

- [Charter](00-program/charter.md) · [Current state](00-program/current-state.md) · [Roadmap](00-program/roadmap.md)
- [Workflow](00-program/delivery-workflow.md) · [Contract index](00-program/contracts.md) · [Risks](00-program/risks.md)
- [Implementation index](00-program/implementation-index.md) · [Reference index](00-program/reference-index.md)
- [Operational runbooks](90-governance/runbooks.md) · [Decision register](90-governance/decision-register.md) · [Traceability](90-governance/requirement-traceability.md)

## Package rules

`plan.md` is implementation intent and may change when an approved decision changes. `tasks.md` is the actionable checklist. `execution.md` for M1–M4 and `evidence.md` for M5+ discovery are append-only evidence logs and must not rewrite acceptance retroactively. A feature may contain multiple commits but has one review gate. After merge, update the package evidence before starting the next package.

Product, architecture, design, quality, and operations remain canonical in their own documentation layers. This workspace records how those decisions are delivered and what evidence proves them.
