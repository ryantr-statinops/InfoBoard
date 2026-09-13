# Packaging and MVP release

**Package:** RELEASE-PACKAGE  
**Status:** not_started  
**Milestone:** [99-release](../README.md)

## Outcome

Produce a clean local installation with reproducible configuration, data paths, upgrade/rollback behavior, and complete release evidence.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

RELEASE-TEST, REL-RECOVERY, REL-SECURITY. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [RELEASE-PACKAGE-001](tasks.md#release-package-001) — Lock dependencies and clean-install workflow
- [RELEASE-PACKAGE-002](tasks.md#release-package-002) — Verify upgrade, backup, rollback, and rebuild
- [RELEASE-PACKAGE-003](tasks.md#release-package-003) — Assemble and approve MVP release evidence

## Non-goals

No canonical behavior is waived, no evidence is fabricated, and compatibility routes are not removed before the accepted release boundary.
