# Provider configuration and consent

**Package:** SEM-CONFIG  
**Status:** not_started  
**Milestone:** [50-semantic-provider](../README.md)

## Outcome

Configure and verify a self-hosted OpenAI-compatible endpoint without accepting secrets through the API or transferring user content before consent.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

API-CANONICAL, RET-KEYWORD; see [dependency map](../../00-program/dependency-map.md).

## Owned work

- [SEM-CONFIG-001](tasks.md#sem-config-001) — Implement environment-only secret and settings projection
- [SEM-CONFIG-002](tasks.md#sem-config-002) — Validate endpoint and verification boundaries
- [SEM-CONFIG-003](tasks.md#sem-config-003) — Implement explicit consent and revocation

## Non-goals

This package does not make a derived store authoritative, send content without consent, or redefine provider/search contracts.
