# Security and privacy hardening: implementation plan

## Approach

Close local HTTP, SSRF, rendering, resource-limit, secret, consent, archive, and diagnostic boundaries with adversarial verification. Implement in HTTP middleware, capture/provider clients, renderers/templates, configuration, security tests with bounded inputs, explicit checkpoints, stable public state, and failure injection from the first delivery slice.

## Flow

1. Establish fixtures and measurable consistency/security invariants.
2. Implement the bounded service or projection boundary.
3. Add atomic activation, fallback, degradation, or rollback behavior.
4. Verify interruption, corruption, deletion, stale data, and restart.
5. Record reproducible, redacted evidence and update rollups.

## Operational boundary

Canonical SQLite and referenced snapshot files are the only required recovery inputs. Optional and derived components never write back to canonical tables. Known-stale output is never presented as current.

## Rollout

Complete API-CANONICAL, CAP-URL, SEM-CONFIG, ORG-UI, REL-RECOVERY; deliver [tasks](tasks.md) in order; satisfy milestone acceptance before marking done.
