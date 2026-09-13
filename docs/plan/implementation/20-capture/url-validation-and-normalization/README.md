# URL validation and normalization

**Package:** CAP-URL  
**Status:** not_started  
**Milestone:** [20-capture](../README.md)

## Outcome

Create deterministic URL identity while enforcing public-network and bounded-fetch security at every connection and redirect.

## Navigation

- [Implementation plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [Canonical references](references.md)
- [Execution evidence](execution.md)

## Dependencies

API-CANONICAL. See the program [dependency map](../../00-program/dependency-map.md).

## Owned work

- [CAP-URL-001](tasks.md#cap-url-001) — Implement normalization policy version 1
- [CAP-URL-002](tasks.md#cap-url-002) — Enforce SSRF and redirect boundaries
- [CAP-URL-003](tasks.md#cap-url-003) — Bound fetch and parsing resources

## Non-goals

This package does not redefine upstream contracts, claim unevidenced runtime completion, or take ownership from another package.
