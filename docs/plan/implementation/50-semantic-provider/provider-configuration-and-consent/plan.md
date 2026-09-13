# Provider configuration and consent: implementation plan

## Approach

Configure and verify a self-hosted OpenAI-compatible endpoint without accepting secrets through the API or transferring user content before consent. Implement the boundary in `app/semantic.py`, settings/provider services, `/api/v1/settings/semantic`, security and contract tests. Use explicit adapters so provider and store failure cannot leak into canonical repositories or keyword retrieval.

## Flow

1. Validate configuration, consent, revision, and canonical eligibility.
2. Perform bounded provider or derived-store work.
3. Validate every external/derived result before use.
4. Persist durable status/checkpoints and project safe degradation.
5. Test revocation, incompatibility, outage, restart, cleanup, and rebuild.

## Security and recovery

Recheck consent immediately before transfer. Never log API keys, query text, raw snapshots, or sensitive responses. Derived records carry bookmark, snapshot, chunk, and revision identity and are disposable.

## Rollout

Complete API-CANONICAL, RET-KEYWORD; deliver [tasks](tasks.md) in order; record real [evidence](execution.md); update rollups only after milestone acceptance.
