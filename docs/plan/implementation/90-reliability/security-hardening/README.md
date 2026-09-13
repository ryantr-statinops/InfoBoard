# Security and privacy hardening

**Package:** REL-SECURITY  
**Status:** not_started  
**Milestone:** [90-reliability](../README.md)

## Outcome

Close local HTTP, SSRF, rendering, resource-limit, secret, consent, archive, and diagnostic boundaries with adversarial verification.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

API-CANONICAL, CAP-URL, SEM-CONFIG, ORG-UI, REL-RECOVERY. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [REL-SECURITY-001](tasks.md#rel-security-001) — Harden local HTTP and browser mutation boundaries
- [REL-SECURITY-002](tasks.md#rel-security-002) — Harden content, provider, and resource handling
- [REL-SECURITY-003](tasks.md#rel-security-003) — Harden backup, maintenance, and filesystem boundaries

## Non-goals

This package does not move authority out of SQLite, hide known staleness, or claim success without reproducible evidence.
