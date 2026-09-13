# ChromaDB semantic index: implementation plan

## Approach

Maintain a persistent revision-scoped vector index whose results are always revalidated against current canonical SQLite state. Implement the boundary in Chroma adapter, vector index job handler, revision activation, cleanup/rebuild tests. Use explicit adapters so provider and store failure cannot leak into canonical repositories or keyword retrieval.

## Flow

1. Validate configuration, consent, revision, and canonical eligibility.
2. Perform bounded provider or derived-store work.
3. Validate every external/derived result before use.
4. Persist durable status/checkpoints and project safe degradation.
5. Test revocation, incompatibility, outage, restart, cleanup, and rebuild.

## Security and recovery

Recheck consent immediately before transfer. Never log API keys, query text, raw snapshots, or sensitive responses. Derived records carry bookmark, snapshot, chunk, and revision identity and are disposable.

## Rollout

Complete SEM-EMBED, CAP-WORKER; deliver [tasks](tasks.md) in order; record real [evidence](execution.md); update rollups only after milestone acceptance.
