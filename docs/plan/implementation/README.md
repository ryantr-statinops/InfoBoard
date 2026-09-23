# InfoBoard implementation plans

This directory is the implementation control plane for the current InfoBoard product contract.

InfoBoard is a Chrome/Edge desktop extension for Linux, macOS, and Windows. The extension owns browser APIs, tab observation, the focused search surface, and tab activation. A local Go Native Messaging host owns the framed protocol, current-profile projection reconciliation, in-memory lexical search, deterministic ranking, bounded SQLite persistence, health, and diagnostics.

## Source of truth

The binding product contract is in [`docs/plan/refactor/README.md`](../refactor/README.md), [`architecture.md`](../refactor/architecture.md), [`requirements.md`](../refactor/requirements.md), [`runtime-protocol.md`](../refactor/runtime-protocol.md), and [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md). If an implementation plan conflicts with those documents, update the plan before implementation begins.

## Workspace contract

- [`index.md`](index.md) is the inventory, dependency DAG, requirement ownership map, logical target layout, and execution protocol.
- `phase-01` through `phase-20` are standalone implementation plans. Each has its own scope, prerequisites, deliverables, skill references, commit plan, acceptance checks, risks, and references outside `docs/`.
- Implementation targets are currently logical `to-create` paths under `extension/`, `host/`, `packaging/`, `fixtures/`, and `tests/`.
- Phase branches are created from the latest `dev`, contain one assigned phase file, and merge back to `dev`. `main` is read-only for this work.
- Do not add unrelated product areas, browser data sources, remote services, or unbounded persistence.

## Execution order

Follow the waves and dependencies in [`index.md`](index.md). The control owner creates the structural cutover and root-control commits; phase owners author only their assigned phase file. Revalidate the corpus after every merge and push every accepted commit to its configured remote branch.
