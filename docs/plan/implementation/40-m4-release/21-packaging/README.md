# 21 — Packaging, release, and upgrades

**Plan status:** `ready` · **Delivery status:** `not_started` · **Baseline coverage:** `missing`

The epic makes core/full modes, clean installation, upgrades, release artifacts, and rollback repeatable and explicit.

**Requirements:** `PR-OPS-01…02`, `RQ-014`  
**Dependencies:** 10, 11, 18, 20

## Reading order

1. [Plan](plan.md)
2. [Tasks](tasks.md)
3. [References](references.md)
4. [Examples](examples.md)
5. [Release guide](guides/release.md)
6. [Execution evidence](execution.md)

## Package index

| Area | Location |
| --- | --- |
| Manifest | [`pyproject.toml`](../../../../../pyproject.toml), [`uv.lock`](../../../../../uv.lock) |
| Setup | [`README.md`](../../../../../README.md), [local setup](../../../../operations/00-local-setup-and-modes.md) |
| Release operations | [upgrade/release/rollback](../../../../operations/03-upgrade-release-and-rollback.md) |
