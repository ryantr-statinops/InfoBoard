# API compatibility transition

**Package:** RELEASE-COMPAT  
**Status:** not_started  
**Milestone:** [99-release](../README.md)

## Outcome

Ship `/api/*` as thin one-release adapters over `/api/v1/*`, publish deprecation metadata, and remove them only in the following release.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

API-CANONICAL and all API-owning packages; removal requires one released compatibility version. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [RELEASE-COMPAT-001](tasks.md#release-compat-001) — Implement behavior-preserving compatibility adapters
- [RELEASE-COMPAT-002](tasks.md#release-compat-002) — Emit deprecation and retirement guidance
- [RELEASE-COMPAT-003](tasks.md#release-compat-003) — Remove adapters only after the compatibility release

## Non-goals

No canonical behavior is waived, no evidence is fabricated, and compatibility routes are not removed before the accepted release boundary.
